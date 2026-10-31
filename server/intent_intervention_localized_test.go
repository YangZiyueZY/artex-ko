package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// 回归防护测试：intent_intervention.go 的 Worker 介入 API 错误响应必须保持简体中文。
// 中文判定复用 F3a 提供的 assertChineseError（含汉字、不含韩文）。

// TestIntentInterventionErrorConstantsLocalized 断言十一个响应常量全部为简体中文。
// 生命周期与意图状态路径需要 s.engine 与 Store 配置，在没有 DB 的机器上跑不到底，
// 因此这些文案直接断言常量本身（conversations.go 先例）。四条输入校验路径由下面的
// HTTP 测试检查到响应正文。
func TestIntentInterventionErrorConstantsLocalized(t *testing.T) {
	cases := []struct {
		name string
		msg  string
	}{
		{"request_too_large", errIntentRequestTooLarge},
		{"message_empty", errIntentMessageEmpty},
		{"message_too_long", errIntentMessageTooLong},
		{"bad_request_id", errIntentBadRequestID},
		{"task_deleting", errIntentTaskDeleting},
		{"task_paused", errIntentTaskPaused},
		{"task_queued", errIntentTaskQueued},
		{"task_terminal", errIntentTaskTerminal},
		{"task_settling", errIntentTaskSettling},
		{"inherited_readonly", errIntentInheritedReadonly},
		{"not_paused", errIntentNotPaused},
	}
	for _, c := range cases {
		assertChineseError(t, c.name, c.msg)
	}
}

// TestSendWorkerMessageInputValidationLocalized 检查四条输入校验路径，一直校验到真实
// HTTP 响应正文。sendWorkerMessage 的这四条路径只读 s.m.Task(查表)与请求正文，不经过
// s.engine、Store 与 DB，因此只要在 tasks 映射里放一个任务就能在没有 DB 的情况下跑到底，
// 并确认常量确实被装进响应。
func TestSendWorkerMessageInputValidationLocalized(t *testing.T) {
	s := &Server{m: &Manager{tasks: map[string]*Task{"t1": {ID: "t1"}}}}

	call := func(body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/tasks/t1/intents/1/message", strings.NewReader(body))
		req.SetPathValue("id", "t1")
		req.SetPathValue("iid", "1")
		rec := httptest.NewRecorder()
		s.sendWorkerMessage(rec, req)
		return rec
	}
	errBody := func(t *testing.T, rec *httptest.ResponseRecorder) string {
		t.Helper()
		var resp struct {
			Error string `json:"error"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("响应 JSON 解析失败：%v (正文 %q)", err, rec.Body.String())
		}
		return resp.Error
	}

	cases := []struct {
		name string
		body string
		code int
		want string
	}{
			// 超过 64KB 上限的合法 JSON 正文。MaxBytesReader 会在读取过程中返回超限。
		{"request_too_large", `{"message":"` + strings.Repeat("a", maxWorkerMessageBytes+1024) + `"}`, http.StatusRequestEntityTooLarge, errIntentRequestTooLarge},
		{"message_empty", `{"message":"  "}`, http.StatusBadRequest, errIntentMessageEmpty},
		{"message_too_long", `{"message":"` + strings.Repeat("测", 4001) + `"}`, http.StatusBadRequest, errIntentMessageTooLong},
		{"bad_request_id", `{"message":"你好","request_id":"包含 空格"}`, http.StatusBadRequest, errIntentBadRequestID},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec := call(c.body)
			if rec.Code != c.code {
				t.Fatalf("状态码 = %d, 期望 = %d (正文 %q)", rec.Code, c.code, rec.Body.String())
			}
			if got := errBody(t, rec); got != c.want {
				t.Fatalf("响应文案 = %q, 期望 = %q", got, c.want)
			}
			assertChineseError(t, c.name, errBody(t, rec))
		})
	}
}
