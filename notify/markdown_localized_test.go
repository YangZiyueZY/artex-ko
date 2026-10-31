package notify

import (
	"strings"
	"testing"
)

// hanFreeItems 构造不含 CJK 的条目（ASCII 标题·类型·概要）。
// 数据全 ASCII 时，输出里出现中文只可能来自骨架标签，断言才能精准锁定标签回退。
func hanFreeItems(n int) []Item {
	items := make([]Item, 0, n)
	for i := 0; i < n; i++ {
		items = append(items, Item{
			FindingID: int64(i + 1),
			Name:      "login-flaw",
			VulnClass: "SQLi",
			Severity:  "high",
			Summary:   "SQL injection via q param",
			Assets:    []string{"a.example.com"},
		})
	}
	return items
}

// TestMarkdownTitleLocalized 检查 markdownTitle——它被 markdown 系渠道（dingtalk·wecom）
// 与共用标题的 feishu·html·telegram·webhook 一起使用——是简体中文。
// 任何一个回退，这五个渠道的标题就会全部语言混杂。
func TestMarkdownTitleLocalized(t *testing.T) {
	// 多条（汇总）：「漏洞摘要 · 共 N 条」
	got := markdownTitle(Message{Batch: true, Items: hanFreeItems(3)})
	assertChinese(t, "markdownTitle(batch)", got)
	if !strings.Contains(got, "漏洞摘要") || !strings.Contains(got, "共 3 条") {
		t.Errorf("多条标题应为「漏洞摘要 · 共 3 条」的形式，收到的值为 %q", got)
	}
	// 无条目：「漏洞通知」
	empty := markdownTitle(Message{})
	assertChinese(t, "markdownTitle(empty)", empty)
	if empty != "漏洞通知" {
		t.Errorf("空消息标题应为「漏洞通知」，收到的值为 %q", empty)
	}
}

// TestMarkdownBatchIntroLocalized 检查汇总消息头部（时间窗·条数·超限提示）是简体中文。
// 条目数据不含 CJK，所以输出里若出现韩文，说明头部文案回退了。
func TestMarkdownBatchIntroLocalized(t *testing.T) {
	items := hanFreeItems(3)

	// 有时间窗：「最近 N 分钟新增漏洞 N 条」
	win := markdownBatchIntro(Message{Batch: true, WindowMinutes: 30}, items, 3)
	if hasHangul(win) {
		t.Errorf("时间窗标题中残留韩文：%q", win)
	}
	if !strings.Contains(win, "最近 30 分钟") || !strings.Contains(win, "新增漏洞 3 条") {
		t.Errorf("时间窗标题应为「最近 30 分钟新增漏洞 3 条」的形式，收到的值为 %q", win)
	}

	// 无时间窗：「新增漏洞 N 条」（不出现分钟标注）
	noWin := markdownBatchIntro(Message{Batch: true}, items, 3)
	if hasHangul(noWin) {
		t.Errorf("标题中残留韩文：%q", noWin)
	}
	if !strings.Contains(noWin, "新增漏洞 3 条") || strings.Contains(noWin, "分钟") {
		t.Errorf("无时间窗的标题应为「新增漏洞 3 条」的形式（不出现「分钟」），收到的值为 %q", noWin)
	}

	// 只装下一部分时：（本条消息仅显示前 N 条，其余 N 条将在下一条消息中继续发送）
	trunc := markdownBatchIntro(Message{Batch: true, WindowMinutes: 30}, items, 5)
	if hasHangul(trunc) {
		t.Errorf("超限提示中残留韩文：%q", trunc)
	}
	for _, want := range []string{"前 3 条", "其余 2 条", "下一条消息中"} {
		if !strings.Contains(trunc, want) {
			t.Errorf("超限提示中应包含 %q，收到的值为 %q", want, trunc)
		}
	}
}

// TestWriteItemLabelsLocalized 检查单条详情渲染（markdown 三渠道共享路径）的
// 字段标签（状态变更·类型·资产·概要）与详情链接是简体中文。
func TestWriteItemLabelsLocalized(t *testing.T) {
	it := Item{
		Name:       "login-flaw",
		VulnClass:  "SQLi",
		Severity:   "high",
		Summary:    "SQL injection via q param",
		Assets:     []string{"a.example.com"},
		DetailURL:  "https://platform.example/finding/1",
		FromStatus: "pending",
		ToStatus:   "fixed",
	}
	var b strings.Builder
	writeItem(&b, it, "", true)
	got := b.String()

	assertChinese(t, "writeItem(single)", got)
	for _, want := range []string{
		"**状态变更**: 待处理 → 已修复",
		"**类型**: SQLi",
		"**资产**: a.example.com",
		"**概要**: SQL injection via q param",
		"[查看详情](https://platform.example/finding/1)",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("单条渲染结果中应包含 %q：\n%s", want, got)
		}
	}
}

// TestMarkdownBodyFooterLocalized 检查多条正文末尾的平台入口链接是简体中文
// （仅在 HomeURL 存在时追加）。
func TestMarkdownBodyFooterLocalized(t *testing.T) {
	m := Message{Batch: true, HomeURL: "https://platform.example", Items: hanFreeItems(2)}
	body, kept := markdownBody(m, 0)
	if kept != 2 {
		t.Fatalf("不限额度 (0) 时必须容纳全部 2 条，收到的值为 %d", kept)
	}
	if !strings.Contains(body, "[在平台查看全部](https://platform.example)") {
		t.Errorf("正文末尾应有「在平台查看全部」链接：\n%s", body)
	}
}
