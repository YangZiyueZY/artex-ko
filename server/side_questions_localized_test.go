package server

import "testing"

// TestSideQuestionAPIErrorsLocalized 守护 F21 汉化：由 HTTP 包装层（writeErr 响应）与
// 交换状态（e.Error）暴露的每条用户可见旁路提问消息都必须是简体中文（含汉字、不含韩文）。
// 这些消息与 sidequestion 包的报错一起渲染到对话的旁路提问面板，因此两层同时汉化，
// 以免面板里语言混杂。复用 assertChineseError（F3a）。
func TestSideQuestionAPIErrorsLocalized(t *testing.T) {
	for _, c := range []struct{ label, msg string }{
		{"ctx_not_saved", sideErrCtxNotSaved},
		{"model_config_changed", sideErrModelConfigChanged},
		{"service_unavailable", sideErrServiceUnavailable},
		{"task_archived", sideErrTaskArchived},
		{"worker_deleted", sideErrWorkerDeleted},
		{"bad_question", sideErrBadQuestion},
		{"task_archiving", sideErrTaskArchiving},
		{"request_id_reused", sideErrRequestIDReused},
		{"no_snapshot", sideErrNoSnapshot},
		{"concurrency_limit", sideErrConcurrencyLimit},
		{"answer_stopped", sideErrAnswerStopped},
		{"answer_timeout", sideErrAnswerTimeout},
	} {
		assertChineseError(t, c.label, c.msg)
	}
}
