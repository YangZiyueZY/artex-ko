package notify

import (
	"strings"
	"testing"
)

// 本文件锁住 F4 ③（邮件正文/HTML 模板）的中文化，防止回退。
// html.go 的标签与 email.go 的校验错误一旦回退成别的语言，这里就会报错。
// 助手 hasHan·hasHangul·assertChinese 定义在 notify_localized_test.go（同包）。
//
// 数据（标题·类型·资产·概要）全部用 ASCII，这样「输出里没有韩文」这条断言只会
// 逮住骨架标签的回退，而不是内容。严重度·状态标签由 F4 ① 统一渲染成中文。

// TestHTMLItemLabelsLocalized 打开单条邮件正文的所有标签分支，确认骨架是简体中文
// 且不残留韩文。
func TestHTMLItemLabelsLocalized(t *testing.T) {
	m := Message{
		HomeURL: "https://example.com/panel",
		Items: []Item{{
			Name:       "sqli-login", // Title() = Name
			VulnClass:  "injection",  // 与 Title() 不同，才会渲染类型行
			Severity:   "high",       // SeverityLabel → 高危
			Summary:    "login form is injectable",
			Assets:     []string{"host-a.example.com"},
			DetailURL:  "https://example.com/f/1",
			FromStatus: "pending", // IsStatusChange()=true → 渲染状态变更行
			ToStatus:   "fixed",
		}},
	}
	out := htmlBody(m, 0)
	if hasHangul(out) {
		t.Errorf("邮件正文中残留韩文：\n%s", out)
	}
	for _, want := range []string{
		"状态变更", "类型", "资产", "概要", "查看详情", "在平台查看全部",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("邮件正文中缺少 %q 标签：\n%s", want, out)
		}
	}
}

// TestHTMLBatchIntroLocalized 检查汇总（digest）标题行的两个分支（有时间窗/无时间窗）
// 都渲染成简体中文。
func TestHTMLBatchIntroLocalized(t *testing.T) {
	items := []Item{{Name: "a", Severity: "high"}, {Name: "b", Severity: "low"}}

	withWindow := htmlBatchIntro(Message{Batch: true, WindowMinutes: 30, Items: items})
	assertChinese(t, "htmlBatchIntro(时间窗)", withWindow)
	for _, want := range []string{"最近 30 分钟", "新增漏洞", "2 条"} {
		if !strings.Contains(withWindow, want) {
			t.Errorf("时间窗标题中缺少 %q：%q", want, withWindow)
		}
	}

	noWindow := htmlBatchIntro(Message{Batch: true, WindowMinutes: 0, Items: items})
	assertChinese(t, "htmlBatchIntro(无时间窗)", noWindow)
	if !strings.Contains(noWindow, "新增漏洞 2 条") {
		t.Errorf("无时间窗的标题应包含 %q：%q", "新增漏洞 2 条", noWindow)
	}
	if strings.Contains(noWindow, "分钟") {
		t.Errorf("没有时间窗却出现了「分钟」：%q", noWindow)
	}
}

// TestEmailValidateLocalized 检查 SMTP 设置校验的四种错误都是简体中文，并且每种
// 都说清了是哪个字段有问题。这些消息会直接展示给配置通知渠道的用户。
func TestEmailValidateLocalized(t *testing.T) {
	cases := []struct {
		name   string
		cfg    map[string]any
		substr string
	}{
		{"缺少服务器地址", map[string]any{"port": float64(25), "from": "a@b.c", "to": []any{"d@e.f"}}, "SMTP"},
		{"端口超出范围", map[string]any{"host": "h"}, "端口"},
		{"缺少发件人", map[string]any{"host": "h", "port": float64(25), "to": []any{"d@e.f"}}, "发件人"},
		{"缺少收件人", map[string]any{"host": "h", "port": float64(25), "from": "a@b.c"}, "收件人"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := (emailChannel{}).Validate(tc.cfg)
			if err == nil {
				t.Fatalf("校验应失败：%v", tc.cfg)
			}
			assertChinese(t, "Validate("+tc.name+")", err.Error())
			if !strings.Contains(err.Error(), tc.substr) {
				t.Errorf("错误信息中应包含 %q，收到的值为 %q", tc.substr, err.Error())
			}
		})
	}
}
