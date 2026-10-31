package server

import "testing"

// 回归防护测试：finding_retests.go 的用户可见 HTTP 错误响应文案必须保持简体中文。
// 中文判定复用 F3a 提供的 assertChineseError（含汉字、不含韩文）。返回这四条文案的
// startFindingRetest 处理器开头要过 s.pg(w)(DB) 闸门，在没有 DB 的机器上跑不到底，
// 因此直接钉住常量本身（与 finding_traffic·goals_api·notify_api 的 DB 闸门路径同一做法）。
// 若有人把这些字面量改回韩文，本测试即失败。
//
// 工具说明（get_finding_retest_context·record_finding_retest_result）、参数说明、
// actool.Errorf（未关联会话的提示），以及 seedFindingRetester 写入 DB 的种子智能体名称、
// 画像与 note，都属于智能体读取的大脑输入或种子数据，刻意保留中文原文，不在本测试的
// 断言范围内（见 finding_retests.go 常量块注释）。
func TestFindingRetestErrorsLocalized(t *testing.T) {
	for _, c := range []struct {
		label, msg string
	}{
		{"notes_too_long", errFindingRetestNotesTooLong},
		{"agent_missing", errFindingRetestAgentMissing},
		{"tool_required", errFindingRetestToolRequired},
		{"service_stopping", errFindingRetestServiceStopping},
	} {
		assertChineseError(t, c.label, c.msg)
	}
}
