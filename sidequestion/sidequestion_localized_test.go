package sidequestion

import (
	"errors"
	"fmt"
	"testing"
	"unicode"
)

// assertChinese 在 msg 为空、含韩文（残留没汉化干净的文案），或完全没有中文汉字时失败。
// ASCII 字段名与「1–4000」这类区间写法都没问题；只有韩文才说明没汉化。
func assertChinese(t *testing.T, label, msg string) {
	t.Helper()
	if msg == "" {
		t.Fatalf("%s: 消息为空", label)
	}
	han := false
	for _, r := range msg {
		if unicode.Is(unicode.Hangul, r) {
			t.Fatalf("%s：仍残留韩文：%q", label, msg)
		}
		if unicode.Is(unicode.Han, r) {
			han = true
		}
	}
	if !han {
		t.Fatalf("%s：没有中文汉字：%q", label, msg)
	}
}

// TestSideQuestionOutputsLocalized 守住 F21：本包产出的每一条面向用户的旁路提问
// （곁질문）错误与答案文本常量都必须是简体中文。它们会经 /api/.../side-questions
// 进入聊天的旁路提问面板，所以这里回退成别的语言就是用户可见的回归，必须让构建失败。
func TestSideQuestionOutputsLocalized(t *testing.T) {
	assertChinese(t, "ErrContextBudget", ErrContextBudget.Error())
	assertChinese(t, "errSideModelInterrupted", errSideModelInterrupted.Error())
	assertChinese(t, "errSideNoAnswer", errSideNoAnswer.Error())
	assertChinese(t, "msgSideToolUnavailable", msgSideToolUnavailable)
	assertChinese(t, "errSideSummaryCallCap", errSideSummaryCallCap.Error())
	assertChinese(t, "sideSummaryFailedPrefix", sideSummaryFailedPrefix)
	assertChinese(t, "errSideSummaryIncomplete", errSideSummaryIncomplete.Error())
	assertChinese(t, "errSideSummaryOverBudget", errSideSummaryOverBudget.Error())
	assertChinese(t, "errSideHistoryCursor", errSideHistoryCursor.Error())
	assertChinese(t, "errSideCompactionStalled", errSideCompactionStalled.Error())
}

// TestSideQuestionBudgetSentinelPreserved：汉化消息不能破坏那些用 errors.Is
// 判断 ErrContextBudget 哨兵的调用方（见 context_test.go，恢复路径放弃后依赖它）。
func TestSideQuestionBudgetSentinelPreserved(t *testing.T) {
	if !errors.Is(fmt.Errorf("prepare: %w", ErrContextBudget), ErrContextBudget) {
		t.Fatal("ErrContextBudget 哨兵识别 (errors.Is) 被破坏")
	}
}

// TestSideQuestionBrainPreserved：智能体大脑提示词（回答指令与摘要指令）必须保持
// 原本那套经过基准验证的中文。汉化方针是「不翻译大脑正文，只统一输出语言」，
// 所以这两条提示词一旦被改动，基准行为就会漂移。
func TestSideQuestionBrainPreserved(t *testing.T) {
	for _, c := range []struct{ label, text string }{
		{"instruction", instruction},
		{"summaryInstruction", summaryInstruction},
	} {
		han := false
		for _, r := range c.text {
			if unicode.Is(unicode.Han, r) {
				han = true
				break
			}
		}
		if !han {
			t.Errorf("%s: 大脑提示词不再是中文（存在基准漂移风险）", c.label)
		}
	}
}
