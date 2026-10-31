package notify

import (
	"strings"
	"testing"
)

// hasHan 探测字符串里是否含有 CJK 统一汉字。汉化后它用来抓「标签被回退成非中文」
// 这类回归（配合 hasHangul 一起判断语言）。
func hasHan(s string) bool {
	for _, r := range s {
		if r >= 0x4E00 && r <= 0x9FFF {
			return true
		}
	}
	return false
}

// hasHangul 探测字符串里是否含有韩文音节。
func hasHangul(s string) bool {
	for _, r := range s {
		if r >= 0xAC00 && r <= 0xD7A3 {
			return true
		}
	}
	return false
}

// assertChinese 断言文案是简体中文：含汉字、且不含韩文。
// 语言目录里的 "ko" 条目是给韩语用户看的译文，属于允许保留的韩文（见 work/RULES-go.md），
// 所以只有在显式把输出语言切到 ko 时才允许出现韩文。
func assertChinese(t *testing.T, where, got string) {
	t.Helper()
	if hasHangul(got) {
		t.Errorf("%s：仍残留韩文：%q", where, got)
	}
	if !hasHan(got) {
		t.Errorf("%s：没有中文汉字：%q", where, got)
	}
}

// withLang 在测试期间把 notify.Lang 接到指定语言，结束后恢复原值。
func withLang(t *testing.T, lang string) {
	t.Helper()
	prev := Lang
	Lang = func() string { return lang }
	t.Cleanup(func() { Lang = prev })
}

// TestSeverityLabelLocalized 检查全部严重度枚举都渲染成中文标签。
// 标签被所有通知渠道（telegram·email·html·markdown·webhook·feishu·dingtalk·wecom）共用，
// 只要有一个回退成别的语言，所有渠道的消息就会语言混杂。
func TestSeverityLabelLocalized(t *testing.T) {
	// 默认输出语言（未接线时为 zh）的标签，与 UI status.severity 一致。
	want := map[string]string{
		"critical": "🔴 严重",
		"high":     "🟠 高危",
		"medium":   "🟡 中危",
		"low":      "🔵 低危",
	}
	for sev, label := range want {
		got := SeverityLabel(sev)
		assertChinese(t, "SeverityLabel("+sev+")", got)
		if got != label {
			t.Errorf("SeverityLabel(%q)=%q，期望 %q", sev, got, label)
		}
	}
	// 未知严重度原样返回（不臆造不存在的标签）。
	if got := SeverityLabel("made_up"); got != "made_up" {
		t.Errorf("未知严重度应保持原文，收到的值为 %q", got)
	}
	// 显式切到 ko 时必须回落到语言目录的韩文条目（这些韩文是有意保留的译文）。
	withLang(t, "ko")
	if got := SeverityLabel("high"); got != "🟠 높음" {
		t.Errorf("ko 严重度标签=%q，期望 %q", got, "🟠 높음")
	}
}

// TestStatusLabelLocalized 检查九种处置状态全部渲染成中文标签。
// 必须与 UI status.finding 命名空间（B4a）同写法，否则状态变更通知和界面会对不上。
func TestStatusLabelLocalized(t *testing.T) {
	want := map[string]string{
		"pending":        "待处理",
		"in_progress":    "处理中",
		"confirmed":      "已确认",
		"resolved":       "已处理",
		"fixed":          "已修复",
		"false_positive": "误报",
		"ignored":        "忽略",
		"duplicate":      "重复",
		"risk_accepted":  "接受风险",
	}
	for status, label := range want {
		got := StatusLabel(status)
		assertChinese(t, "StatusLabel("+status+")", got)
		if got != label {
			t.Errorf("StatusLabel(%q)=%q，期望 %q", status, got, label)
		}
	}
	// 未知状态原样返回（不臆造不存在的标签）。
	if got := StatusLabel("weird_status"); got != "weird_status" {
		t.Errorf("未知状态应保持原文，收到的值为 %q", got)
	}
	// 显式切到 ko 时回落到韩文条目。
	withLang(t, "ko")
	if got := StatusLabel("pending"); got != "처리 대기" {
		t.Errorf("ko 状态标签=%q，期望 %q", got, "처리 대기")
	}
}

// TestItemTitlePlaceholderLocalized 检查名称与类型都为空的条目，其替代标题是中文占位符
// （未命名漏洞）。绝不输出空标题。
func TestItemTitlePlaceholderLocalized(t *testing.T) {
	got := Item{}.Title()
	assertChinese(t, "Item{}.Title()", got)
	if !strings.Contains(got, "未命名漏洞") {
		t.Errorf("空条目标题应为「（未命名漏洞）」，收到的值为 %q", got)
	}
	// 有名称用名称，只有类型时用类型（确认回退逻辑未回退）。
	if n := (Item{Name: "登录 SQLi"}).Title(); n != "登录 SQLi" {
		t.Errorf("名称优先逻辑被破坏，收到的值为 %q", n)
	}
	if v := (Item{VulnClass: "XSS"}).Title(); v != "XSS" {
		t.Errorf("类型回退逻辑被破坏，收到的值为 %q", v)
	}
}

// TestAssetLineLocalized 检查资产列举串的分隔符与总数写法（ASCII 逗号 + 中文「等 N 个」）。
func TestAssetLineLocalized(t *testing.T) {
	// 未超上限：全部列出，ASCII 逗号分隔。
	if got := assetLine([]string{"a.example.com", "b.example.com"}, 3); got != "a.example.com, b.example.com" {
		t.Errorf("分隔符必须是 ASCII 逗号，收到的值为 %q", got)
	}
	// 超上限：前 limit 个 + 中文「等 N 个」（N 是总数），且不得残留韩文。
	got := assetLine([]string{"a", "b", "c", "d", "e"}, 2)
	if hasHangul(got) {
		t.Errorf("资产列表仍残留韩文：%q", got)
	}
	if !strings.Contains(got, "等 5 个") {
		t.Errorf("总数 5 应标为「等 5 个」，收到的值为 %q", got)
	}
}
