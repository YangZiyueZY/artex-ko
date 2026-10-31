package db

import (
	"testing"
	"unicode"
)

// TestManualTaskScopeSummaryLocalized 钉住手动任务范围来源标签。
// 它会写入 task_asset_links.source_summary（AddTaskScope / SetTaskAssetSource），
// 并原样渲染在任务详情的 sessions/assets 页签上，所以一旦回退成别的语言，
// 同一面板里就会出现语言混杂的来源标签。既有的 task_assets_test.go 已经通过符号
// 断言存下来的值等于这个常量，这里再钉住常量本身，就连用户可见文案一起保护住了。
func TestManualTaskScopeSummaryLocalized(t *testing.T) {
	if manualTaskScopeSummary == "" {
		t.Fatal("manualTaskScopeSummary：空字符串")
	}
	hasHan := false
	for _, r := range manualTaskScopeSummary {
		if unicode.Is(unicode.Hangul, r) {
			t.Fatalf("manualTaskScopeSummary：仍残留韩文：%q", manualTaskScopeSummary)
		}
		if unicode.Is(unicode.Han, r) {
			hasHan = true
		}
	}
	if !hasHan {
		t.Fatalf("manualTaskScopeSummary：没有中文汉字：%q", manualTaskScopeSummary)
	}
}
