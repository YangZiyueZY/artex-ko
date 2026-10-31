package server

import "testing"

// TestReporterSeedLabelsLocalized 是回归防护测试：守护被种子的 reporter 智能体的纯显示
// 标签（reporterAgentName·reporterAgentDescription）保持简体中文。这两个字符串只经
// `agentDTO`（server_mgmt.go）渲染到 system/agents 界面，既不进入任何智能体 system 提示词
// （renderSystem 只按 key→段[A] agent.ReporterDefaultPrompt 组装），也不进入基于模型的
// 智能体选择（reporter 触发器基于 report_finding 工具调用，是确定性的 OnToolCall），
// 属于纯 UI 标签，因此汉化不触碰 BRIEF 边界 #1（大脑不翻译）。相反，reporter 的大脑正文
// （agent.ReporterDefaultPrompt）与触发器注入消息（reporterToolCallMessage）是模型输入，
// 保留中文原文，不在本测试范围内。中文判定复用 assertChineseError（含汉字、不含韩文，
// intercept_archive_localized_test.go）。[[F35]] [[G133]]
func TestReporterSeedLabelsLocalized(t *testing.T) {
	assertChineseError(t, "reporter_name", reporterAgentName)
	assertChineseError(t, "reporter_description", reporterAgentDescription)
}
