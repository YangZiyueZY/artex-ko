package intercept

import (
	"fmt"
	"strings"
	"testing"
	"unicode"
)

// hasHangul 探测字符串里是否含有韩文音节。
func hasHangul(s string) bool {
	for _, r := range s {
		if r >= 0xAC00 && r <= 0xD7A3 {
			return true
		}
	}
	return false
}

// hasHan 探测字符串里是否含有 CJK 汉字。
func hasHan(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}

// assertChinese 断言文案是简体中文：含汉字且不含韩文。
// 语言目录里的 "ko" 条目是给韩语用户看的译文（有意保留），所以只有在显式测回落时
// 才允许韩文；默认输出语言路径必须命中中文。
func assertChinese(t *testing.T, label, s string) {
	t.Helper()
	if !hasHan(s) {
		t.Errorf("%s：没有中文汉字：%q", label, s)
	}
	if hasHangul(s) {
		t.Errorf("%s：仍残留韩文：%q", label, s)
	}
}

// TestInterceptMessagesLocalized 把面向用户的判定/审批文案钉成简体中文。
// 这些文案会出现在审批记录原因、活动流，以及「已被处理」请求的 409 响应里。
func TestInterceptMessagesLocalized(t *testing.T) {
	for _, c := range []struct{ name, s string }{
		{"msgReviewContextIncomplete", msgReviewContextIncomplete},
		{"msgModelApprovalFailed", msgModelApprovalFailed},
		{"msgModelOutputUnparsable", msgModelOutputUnparsable},
		{"reasonWorkCanceled", reasonWorkCanceled},
		{"reasonWorkCanceledPreExec", reasonWorkCanceledPreExec},
		{"reasonApprovalTimeout", reasonApprovalTimeout},
		{"reasonManualDeny", reasonManualDeny},
		{"reasonManualAllow", reasonManualAllow},
		{"ErrAlreadyDecided", ErrAlreadyDecided.Error()},
	} {
		assertChinese(t, c.name, c.s)
	}
}

// TestJudgeActionLabelLocalized 检查判定动作的三个标签是中文
// （允许 / 拦截 / 询问，见术语表），并且未知 action 原样透传。
func TestJudgeActionLabelLocalized(t *testing.T) {
	want := map[string]string{
		"allow": "允许",
		"deny":  "拦截",
		"ask":   "询问",
	}
	for action, label := range want {
		got := judgeActionLabel(action)
		assertChinese(t, "judgeActionLabel("+action+")", got)
		if got != label {
			t.Errorf("judgeActionLabel(%s) = %q，期望 %q", action, got, label)
		}
	}
	if got := judgeActionLabel("weird"); got != "weird" {
		t.Errorf("judgeActionLabel(weird) = %q，期望原样透传", got)
	}
}

// TestDefaultMessageLocalized 检查规则派生的拦截/询问文案是中文，并且仍带上规则名；
// allow 保持空串。
func TestDefaultMessageLocalized(t *testing.T) {
	for _, action := range []string{"deny", "ask"} {
		msg := defaultMessage(action, "R1")
		assertChinese(t, "defaultMessage("+action+")", msg)
		if !strings.Contains(msg, "R1") {
			t.Errorf("defaultMessage(%s) 丢失了规则名：%q", action, msg)
		}
	}
	if got := defaultMessage("allow", "R1"); got != "" {
		t.Errorf("defaultMessage(allow) = %q，期望空串", got)
	}
}

// TestToolApprovalSummaryLocalized 检查活动摘要的中文形态，并且仍能被 transcript.tsx
// 解析：它用 /\(#(\d+)\)/ 取 pending id，用 /工具\s+(\S+)\s+审批请求/ 取工具名
// （同时保留了汉化前旧韩文摘要的兼容正则）。
func TestToolApprovalSummaryLocalized(t *testing.T) {
	s := fmt.Sprintf(msgToolApprovalRequestFmt, "Bash", 42)
	assertChinese(t, "msgToolApprovalRequestFmt", s)
	if !strings.Contains(s, "(#42)") {
		t.Errorf("摘要丢失了 (#N) 标记（transcript.tsx 的 pending_id 正则）：%q", s)
	}
	if !strings.Contains(s, "工具 Bash 审批请求") {
		t.Errorf("摘要丢失了「工具 X 审批请求」的形式（transcript.tsx 的 toolName 正则）：%q", s)
	}
}
