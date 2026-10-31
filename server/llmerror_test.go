package server

import (
	"errors"
	"testing"

	"github.com/Autumn-27/norma/harness"
)

// 模型错误分级是「模型报错不中断任务」的决策入口：可恢复的走重试/延迟重排，
// 确定性的立刻交给规划者/用户。这里把状态码、网络文本、额度、上下文超长的
// 归类逐条钉住。
func TestClassifyLLMErrorByStatus(t *testing.T) {
	cases := []struct {
		name                string
		err                 error
		immediate, requeue  bool
		status              int
	}{
		{"500", errors.New("openai: status 500: internal server error"), true, true, 500},
		{"502", errors.New("openai: status 502: bad gateway"), true, true, 502},
		{"503", errors.New("anthropic: status 503: service unavailable"), true, true, 503},
		{"529", errors.New("anthropic: status 529: overloaded_error"), true, true, 529},
		{"429", errors.New("openai: status 429: rate limit reached"), true, true, 429},
		{"408", errors.New("openai: status 408: request timeout"), true, true, 408},
		{"401", errors.New("openai: status 401: invalid api key"), false, false, 401},
		{"403", errors.New("anthropic: status 403: forbidden"), false, false, 403},
		{"400", errors.New("openai: status 400: bad request"), false, false, 400},
		{"404", errors.New("openai: status 404: model not found"), false, false, 404},
		{"422", errors.New("openai: status 422: invalid parameter"), false, false, 422},
		{"402", errors.New("openai: status 402: payment required"), false, true, 402},
	}
	for _, c := range cases {
		info := classifyLLMError(c.err)
		if info.Immediate != c.immediate || info.Requeue != c.requeue || info.StatusCode != c.status {
			t.Errorf("%s: info=%+v, want immediate=%v requeue=%v status=%d",
				c.name, info, c.immediate, c.requeue, c.status)
		}
	}
}

// 没有状态码的传输层失败与流内 error 事件都要按「可恢复」处理，这是供应商抖动
// 最常见的样子；额度耗尽/上下文超长则必须排除，否则重试只会白烧。
func TestClassifyLLMErrorByMessage(t *testing.T) {
	recoverable := []string{
		"dial tcp 1.2.3.4:443: i/o timeout",
		"read tcp 10.0.0.2:51234: connection reset by peer",
		"write tcp 10.0.0.2:51234: broken pipe",
		"net/http: TLS handshake timeout",
		"lookup api.example.com: no such host",
		"unexpected EOF",
		"anthropic: stream error: overloaded_error",
	}
	for _, msg := range recoverable {
		info := classifyLLMError(errors.New(msg))
		if !info.Immediate || !info.Requeue {
			t.Errorf("%q: info=%+v, want immediate+requeue", msg, info)
		}
	}

	quota := []string{
		"openai: code=insufficient_quota status=429: You exceeded your current quota",
		"billing hard limit has been reached",
	}
	for _, msg := range quota {
		info := classifyLLMError(errors.New(msg))
		if info.Class != llmErrorQuota || info.Immediate || !info.Requeue {
			t.Errorf("%q: info=%+v, want quota(no immediate, requeue)", msg, info)
		}
	}

	tooLong := []string{
		"anthropic: status 400: prompt is too long: 210000 tokens > 200000 maximum",
		"openai: context_length_exceeded: this model's maximum context length is 128000 tokens",
		"openai: status 413: request entity too large",
	}
	for _, msg := range tooLong {
		info := classifyLLMError(errors.New(msg))
		if info.Class != llmErrorContextTooLong || info.Immediate || info.Requeue {
			t.Errorf("%q: info=%+v, want contextTooLong(no retry)", msg, info)
		}
	}

	// 无法识别的错误：保留历史上的「先原地重跑」行为（有界），但不做延迟重排。
	info := classifyLLMError(errors.New("something totally unexpected"))
	if !info.Immediate || info.Requeue || info.Class != llmErrorUnknown {
		t.Fatalf("unknown error info=%+v, want immediate only", info)
	}
	if got := classifyLLMError(nil); got.Immediate || got.Requeue {
		t.Fatalf("nil error info=%+v, want zero", got)
	}
}

// 任务路由层的两种包装错误语义不同：链条耗尽值得等额度恢复；已交付输出后的失败
// 既不能立即重放（会重复输出/重复执行工具），也不该自动重排。
func TestClassifyTaskLLMErrors(t *testing.T) {
	exhausted := &taskLLMError{taskID: "7", chainExhausted: true, cause: errors.New("all selected profiles are quota exhausted")}
	info := classifyLLMError(exhausted)
	if info.Class != llmErrorQuota || info.Immediate || !info.Requeue {
		t.Fatalf("chain exhausted info=%+v, want quota/requeue", info)
	}

	partial := &taskLLMError{taskID: "7", chainExhausted: false, cause: errors.New("insufficient_quota after partial output")}
	info = classifyLLMError(partial)
	if info.Immediate || info.Requeue {
		t.Fatalf("partial-stream error info=%+v, want no replay/no requeue", info)
	}
	if retryableWorkerModelError(harness.ReasonModelError, partial) {
		t.Fatal("worker must not replay an intent after the task router handled a mid-stream error")
	}
	// 普通瞬时错误仍保留原地重跑。
	if !retryableWorkerModelError(harness.ReasonModelError, errors.New("temporary network error")) {
		t.Fatal("ordinary transient model errors should be replayed")
	}
	if retryableWorkerModelError(harness.ReasonCompleted, errors.New("status 503")) {
		t.Fatal("non model_error terminal reasons must never replay")
	}
}

// 同 provider 安全窗口那一层复用分级器后，行为必须与改造前一致：
// 4xx 确定性拒绝/额度/上下文超长不重放，其余（含无法识别）按可重放处理。
func TestStreamRetryableMatchesPreviousBehavior(t *testing.T) {
	notRetryable := []string{
		"anthropic: status 400: invalid request",
		"anthropic: status 401: unauthorized",
		"anthropic: status 403: forbidden",
		"anthropic: status 404: not found",
		"anthropic: status 422: unprocessable",
		"openai: code=insufficient_quota status=429",
		"anthropic: prompt is too long",
		"openai: status 413: payload too large",
	}
	for _, msg := range notRetryable {
		if isRetryableStreamError(errors.New(msg)) {
			t.Errorf("%q must not be retried on the same provider", msg)
		}
	}
	retryable := []string{
		"openai: status 429: slow down",
		"openai: status 500: oops",
		"openai: status 503: unavailable",
		"read tcp: connection reset by peer",
		"anthropic: stream error: overloaded_error",
		"mystery failure",
	}
	for _, msg := range retryable {
		if !isRetryableStreamError(errors.New(msg)) {
			t.Errorf("%q should be retried on the same provider", msg)
		}
	}
	if isRetryableStreamError(nil) {
		t.Fatal("nil error must not be retryable")
	}
}
