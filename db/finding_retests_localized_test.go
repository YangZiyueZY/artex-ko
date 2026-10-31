package db

import (
	"testing"
	"unicode"
)

// assertRetestReasonChinese 断言复验原因常量是简体中文（含汉字、不含韩文，F9）。
// 这些常量会被拼接进 FinishFindingRetest·RecoverFindingRetests 的 SQL 字面量里，
// 所以钉住常量也就同时保护了面板上展示给用户的文案。
func assertRetestReasonChinese(t *testing.T, name, s string) {
	t.Helper()
	if s == "" {
		t.Fatalf("%s：空字符串", name)
	}
	hasHan := false
	for _, r := range s {
		if unicode.Is(unicode.Hangul, r) {
			t.Fatalf("%s：仍残留韩文：%q", name, s)
		}
		if unicode.Is(unicode.Han, r) {
			hasHan = true
		}
	}
	if !hasHan {
		t.Fatalf("%s：没有中文汉字：%q", name, s)
	}
}

// TestFindingRetestReasonsLocalized 断言写入 finding_retests.error 列、并作为
// 复验面板（finding-retest-panel）item.error 展示的两条终态原因是简体中文。
// server/conversations.go 的兄弟原因（convRetest*）共用同一列与同一面板，
// 二者只要有一个不是中文，同一面板里就会语言混杂。
func TestFindingRetestReasonsLocalized(t *testing.T) {
	assertRetestReasonChinese(t, "retestNoConclusionReason", retestNoConclusionReason)
	assertRetestReasonChinese(t, "retestServiceRestartReason", retestServiceRestartReason)
}
