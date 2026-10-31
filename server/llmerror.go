package server

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// 模型错误分级：把一次模型调用失败按「外部能否自愈」分类。
//
// 背景：供应商抖动（网络断开、限流、5xx 过载）是常态，一次报错就把意图判成
// blocked 会让整个任务停摆，用户只能手动重跑。分级后由引擎分两段兜底：
//   - Immediate：同一条意图、同一个配置上立刻重跑是否安全且有希望（传输层抖动）；
//   - Requeue：隔一段时间自动重排（返回 open + 领取栅栏）是否有意义（等外部恢复，
//     包括额度耗尽这种要等额度重置的情况）。
//
// 不可恢复的错误（鉴权失败、参数/模型名错误、上下文超长、流已交付输出后的失败）
// 仍然按原样交给规划者/用户处理，不做无意义的循环重试。
type llmErrorClass int

const (
	llmErrorUnknown llmErrorClass = iota
	llmErrorTransient             // 网络抖动 / 超时 / 限流 / 供应商 5xx：等一会儿多半会好
	llmErrorQuota                 // 额度或余额耗尽：等额度恢复或换配置
	llmErrorContextTooLong        // 上下文超限：重发没用，交给压缩/收窄
	llmErrorAuth                  // 鉴权失败：配置问题，重试不会好
	llmErrorBadRequest            // 参数、模型名、端点等确定性拒绝
)

// llmErrorInfo 是一次模型失败的结构化结论，供日志、活动流与重试决策共用。
type llmErrorInfo struct {
	Class      llmErrorClass
	StatusCode int    // 从错误文本里解析出的 HTTP 状态码；0 = 没解析到
	Immediate  bool   // 引擎可原地重跑这条意图
	Requeue    bool   // 引擎可把它退回 open 并延迟重排
	Summary    string // 面向用户/日志的中文短说明
}

// llmStatusPattern 匹配 SDK(以及 OpenAI 兼容网关)错误文本里的状态码：
// "anthropic: status 529: ..." / "openai: code=insufficient_quota status=429 ..."。
var llmStatusPattern = regexp.MustCompile(`(?i)\bstatus[\s:=]+(\d{3})\b`)

// classifyLLMError 是整条重试链唯一的分级入口。nil 返回零值（不重试）。
func classifyLLMError(err error) llmErrorInfo {
	if err == nil {
		return llmErrorInfo{}
	}
	// 任务路由层的包装：链条耗尽 = 所有配置都不可用（多为额度耗尽），值得等外部恢复；
	// 其余包装错误意味着输出可能已经交付，立即重放不安全，只能判死或等人工。
	if isTaskLLMChainExhausted(err) {
		return llmErrorInfo{Class: llmErrorQuota, Requeue: true, Summary: "所有模型配置都没有可用额度"}
	}
	if isTaskLLMRuntimeError(err) {
		return llmErrorInfo{Class: llmErrorUnknown, Summary: "模型流已交付输出后失败，未自动重放"}
	}
	msg := err.Error()
	lower := strings.ToLower(msg)
	status := llmStatusOf(msg)
	if isQuotaExhaustedError(err) {
		return llmErrorInfo{Class: llmErrorQuota, StatusCode: status, Requeue: true, Summary: "模型额度或余额耗尽"}
	}
	if isContextLengthMessage(lower) {
		return llmErrorInfo{Class: llmErrorContextTooLong, StatusCode: status, Summary: "上下文长度超过模型上限"}
	}
	info := llmErrorInfo{StatusCode: status}
	switch {
	case status == 402:
		info.Class, info.Requeue, info.Summary = llmErrorQuota, true, "模型账户余额不足(HTTP 402)"
	case status == 401 || status == 403:
		info.Class, info.Summary = llmErrorAuth, fmt.Sprintf("模型鉴权失败(HTTP %d)，请检查密钥或权限", status)
	case status == 400:
		info.Class, info.Summary = llmErrorBadRequest, "模型服务拒绝该请求(HTTP 400)，请检查模型名与参数"
	case status == 404 || status == 405 || status == 406 || status == 410 || status == 422:
		info.Class, info.Summary = llmErrorBadRequest, fmt.Sprintf("模型端点或参数不受支持(HTTP %d)", status)
	case status == 413:
		info.Class, info.Summary = llmErrorContextTooLong, "请求体超过模型服务上限(HTTP 413)"
	case status == 408 || status == 409 || status == 425:
		info.Class, info.Immediate, info.Requeue = llmErrorTransient, true, true
		info.Summary = fmt.Sprintf("模型请求超时或冲突(HTTP %d)", status)
	case status == 429:
		info.Class, info.Immediate, info.Requeue = llmErrorTransient, true, true
		info.Summary = "模型服务限流(HTTP 429)"
	case status >= 500 && status <= 599:
		info.Class, info.Immediate, info.Requeue = llmErrorTransient, true, true
		info.Summary = fmt.Sprintf("模型服务暂时不可用(HTTP %d)", status)
	case status > 0:
		info.Class, info.Summary = llmErrorBadRequest, fmt.Sprintf("模型服务返回 HTTP %d", status)
	case isNetworkMessage(lower):
		info.Class, info.Immediate, info.Requeue = llmErrorTransient, true, true
		info.Summary = "模型连接中断或网络抖动"
	default:
		// 无法识别：保留历史行为（原地重跑，有界），但不做延迟重排——
		// 未知错误可能来自代码/协议变更，无限等下去没有意义。
		info.Class, info.Immediate, info.Summary = llmErrorUnknown, true, "模型调用失败（无法识别的错误）"
	}
	return info
}

// streamRetryable 给「同 provider 安全窗口」那一层用：除确定性拒绝外都可重放，
// 与历史行为一致（无法识别的错误按可重试处理，让上层先行兜底）。
func (i llmErrorInfo) streamRetryable() bool {
	switch i.Class {
	case llmErrorQuota, llmErrorContextTooLong, llmErrorAuth, llmErrorBadRequest:
		return false
	}
	return true
}

// llmStatusOf 从错误文本里取出第一个 HTTP 状态码，取不到返回 0。
func llmStatusOf(msg string) int {
	if m := llmStatusPattern.FindStringSubmatch(msg); len(m) == 2 {
		if n, err := strconv.Atoi(m[1]); err == nil {
			return n
		}
	}
	return 0
}

// isContextLengthMessage 识别上下文超限（不同供应商/网关的措辞）。
func isContextLengthMessage(lower string) bool {
	for _, k := range []string{
		"context length", "context_length", "maximum context", "prompt is too long",
		"too long", "context window", "reduce the length", "input is too long",
	} {
		if strings.Contains(lower, k) {
			return true
		}
	}
	return false
}

// isNetworkMessage 识别没有状态码的传输层失败与流内错误事件。
func isNetworkMessage(lower string) bool {
	for _, k := range []string{
		"connection reset", "connection refused", "broken pipe", "i/o timeout",
		"tls handshake", "no such host", "server misbehaving", "network is unreachable",
		"unexpected eof", "unexpected end of file", "eof", "timeout", "temporary failure",
		"stream error", "overloaded_error", "overloaded", "api_error", "server_error",
		"rate limit", "too many requests", "service unavailable", "bad gateway",
	} {
		if strings.Contains(lower, k) {
			return true
		}
	}
	return false
}
