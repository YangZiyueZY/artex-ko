package server

import "testing"

// 回归防护测试：triggers.go 的触发器配置校验错误文案必须保持简体中文。
// 中文判定复用 F3a 提供的 assertChineseError（含汉字、不含韩文）。

// TestTriggerErrorConstantsLocalized 断言三个错误常量全部为简体中文。
func TestTriggerErrorConstantsLocalized(t *testing.T) {
	cases := []struct {
		name string
		msg  string
	}{
		{"no_condition", errTriggerNoCondition},
		{"tool_set_empty", errTriggerToolSetEmpty},
		{"custom_only", errTriggerCustomOnly},
	}
	for _, c := range cases {
		assertChineseError(t, c.name, c.msg)
	}
}

// TestValidateTriggerLocalized 直接调用纯函数 validateTrigger，确认“缺少条件”“未选择工具”
// 路径返回中文常量、合法输入返回空字符串。该返回值会被 pgCreateTrigger·pgUpdateTrigger
// 通过 writeErr(400, msg) 原样暴露给用户。
func TestValidateTriggerLocalized(t *testing.T) {
	if msg := validateTrigger(&triggerReq{}); msg != errTriggerNoCondition {
		t.Fatalf("缺少条件文案 = %q, 期望 = %q", msg, errTriggerNoCondition)
	}
	if msg := validateTrigger(&triggerReq{OnToolCall: true}); msg != errTriggerToolSetEmpty {
		t.Fatalf("未选择工具文案 = %q, 期望 = %q", msg, errTriggerToolSetEmpty)
	}
	if msg := validateTrigger(&triggerReq{OnFinding: true}); msg != "" {
		t.Fatalf("有效输入文案 = %q, 期望 = 空字符串", msg)
	}
	if msg := validateTrigger(&triggerReq{OnToolCall: true, ToolNames: []string{"nmap"}}); msg != "" {
		t.Fatalf("有效工具调用文案 = %q, 期望 = 空字符串", msg)
	}
}
