package notify

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// 本文件锁住 F4 ④⑤（telegram·webhook·共用 HTTP 传输层）的中文化，防止回退：
// telegram.go 的消息标签·汇总标题，webhook.go 的配置校验错误，http.go 的传输错误，
// 以及五个渠道共用的 validateHTTPURL——任何一处回退都会被这里逮住。
// 助手 hasHan·hasHangul·assertChinese 定义在 notify_localized_test.go（同包）。
//
// 数据（标题·类型·资产·概要）全部用 ASCII，这样「输出里没有韩文」这条断言只盯
// 骨架标签的回退，而不受内容影响。

// TestTelegramItemLabelsLocalized 打开单条 Telegram 消息的所有标签分支，确认骨架
// 是简体中文且不残留韩文。
func TestTelegramItemLabelsLocalized(t *testing.T) {
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
	out, kept := telegramHTML(m)
	if kept != 1 {
		t.Fatalf("单条消息应上报 1，收到的值为 %d", kept)
	}
	if hasHangul(out) {
		t.Errorf("Telegram 正文中残留韩文：\n%s", out)
	}
	for _, want := range []string{"状态变更", "类型", "资产", "概要", "查看详情"} {
		if !strings.Contains(out, want) {
			t.Errorf("Telegram 正文中缺少 %q 标签：\n%s", want, out)
		}
	}
}

// TestTelegramBatchTitleLocalized 检查汇总（digest）标题的超限·时间窗分支都渲染成
// 简体中文，且与 markdownTitle 逐字一致，避免渠道之间各说各话。
func TestTelegramBatchTitleLocalized(t *testing.T) {
	items := []Item{{Name: "a", Severity: "high"}, {Name: "b", Severity: "low"}}

	// 超限 + 时间窗：摘要·总条数·前 N 条·其余·下一条消息·最近 N 分钟。
	over := telegramBatchTitle(Message{WindowMinutes: 30}, items, 5)
	assertChinese(t, "telegramBatchTitle(超限)", over)
	for _, want := range []string{"漏洞摘要 · 共 5 条", "仅显示前 2 条", "其余 3 条", "下一条消息", "最近 30 分钟"} {
		if !strings.Contains(over, want) {
			t.Errorf("超限标题中缺少 %q：%q", want, over)
		}
	}

	// 未超限且无时间窗：不应出现「下一条消息」提示。
	full := telegramBatchTitle(Message{}, items, 2)
	assertChinese(t, "telegramBatchTitle(全量)", full)
	if !strings.Contains(full, "漏洞摘要 · 共 2 条") {
		t.Errorf("全量标题中应包含“漏洞摘要 · 共 2 条”：%q", full)
	}
	if strings.Contains(full, "下一条消息") {
		t.Errorf("未超出上限却出现了“下一条消息”提示：%q", full)
	}
}

// TestTelegramValidateLocalized 检查必填字段缺失错误是简体中文，并说明缺的是哪个字段。
// 技术术语 Bot Token·Chat ID 原样保留。
func TestTelegramValidateLocalized(t *testing.T) {
	if err := (telegramChannel{}).Validate(map[string]any{}); err == nil {
		t.Fatal("缺少 bot_token 应验证失败")
	} else {
		assertChinese(t, "telegram Validate(bot_token)", err.Error())
		if !strings.Contains(err.Error(), "Bot Token") {
			t.Errorf("应提示缺少 Bot Token：%q", err.Error())
		}
	}
	if err := (telegramChannel{}).Validate(map[string]any{"bot_token": "t"}); err == nil {
		t.Fatal("缺少 chat_id 应验证失败")
	} else {
		assertChinese(t, "telegram Validate(chat_id)", err.Error())
		if !strings.Contains(err.Error(), "Chat ID") {
			t.Errorf("应提示缺少 Chat ID：%q", err.Error())
		}
	}
}

// TestWebhookValidateLocalized 检查通用 Webhook 配置校验的三种错误都是简体中文，
// 并且每种都说清了问题的原因。
func TestWebhookValidateLocalized(t *testing.T) {
	if err := (webhookChannel{}).Validate(map[string]any{}); err == nil {
		t.Fatal("缺少 url 应验证失败")
	} else {
		assertChinese(t, "webhook Validate(url)", err.Error())
		if !strings.Contains(err.Error(), "目标 URL") {
			t.Errorf("应提示缺少目标 URL：%q", err.Error())
		}
	}
	// 不支持的方法：中文 + 暴露允许列表。
	if err := (webhookChannel{}).Validate(map[string]any{"url": "https://example.com/hook", "method": "DELETE"}); err == nil {
		t.Fatal("DELETE 应验证失败")
	} else {
		assertChinese(t, "webhook Validate(method)", err.Error())
		if !strings.Contains(err.Error(), "GET/POST/PUT/PATCH") {
			t.Errorf("应提示允许的方法列表：%q", err.Error())
		}
	}
	// 模板语法错误：中文（被包装的原因是 Go 模板错误，不含韩文）。
	if err := (webhookChannel{}).Validate(map[string]any{"url": "https://example.com/hook", "body_template": "{{"}); err == nil {
		t.Fatal("损坏的模板应验证失败")
	} else {
		assertChinese(t, "webhook Validate(template)", err.Error())
		if !strings.Contains(err.Error(), "模板") {
			t.Errorf("应提示这是模板错误：%q", err.Error())
		}
	}
}

// TestValidateHTTPURLLocalized 检查五个渠道（telegram·webhook·dingtalk·feishu·wecom）
// 共用的 URL 校验助手的错误是简体中文。这个助手若回退成别的语言，就会和调用方的
// 中文前缀（「API 地址无效：……」）混成半中半外。
func TestValidateHTTPURLLocalized(t *testing.T) {
	// 不支持的协议。
	if err := validateHTTPURL("ftp://example.com/x"); err == nil {
		t.Fatal("应拒绝 ftp 协议")
	} else {
		assertChinese(t, "validateHTTPURL(scheme)", err.Error())
		if !strings.Contains(err.Error(), "http") {
			t.Errorf("应提示支持的协议（http/https）：%q", err.Error())
		}
	}
	// 没有主机名。
	if err := validateHTTPURL("http://"); err == nil {
		t.Fatal("无主机的地址应被拒绝")
	} else {
		assertChinese(t, "validateHTTPURL(host)", err.Error())
	}
}

// TestHTTPLocalTargetBlockedLocalized 检查本地/链路本地地址拦截错误是简体中文，
// 并且提示了绕过方式（环境变量）。
func TestHTTPLocalTargetBlockedLocalized(t *testing.T) {
	t.Setenv(AllowLocalTargetsEnv, "") // 显式关闭，走拦截路径
	err := blockInternalDial("tcp", "127.0.0.1:25", nil)
	if err == nil {
		t.Fatal("本地地址默认应被拦截")
	}
	assertChinese(t, "blockInternalDial", err.Error())
	if !strings.Contains(err.Error(), AllowLocalTargetsEnv) {
		t.Errorf("拦截错误应提示绕过方式（%s）：%q", AllowLocalTargetsEnv, err.Error())
	}
}

// TestHTTPRedactHelpersLocalized 检查地址脱敏（redact）占位符与传输错误的
// 「未知错误」分支都是简体中文。
func TestHTTPRedactHelpersLocalized(t *testing.T) {
	if got := redactRequestTarget(""); !strings.Contains(got, "无法解析") {
		t.Errorf("无法解析的地址应为中文占位符，收到的值为 %q", got)
	}
	// *url.Error 的 Err 为 nil 的分支。
	got := redactTransportError(&url.Error{Op: "Get", URL: "http://api.example", Err: nil})
	if hasHangul(got) {
		t.Errorf("传输错误脱敏结果中残留韩文：%q", got)
	}
	if !strings.Contains(got, "未知错误") {
		t.Errorf("Err 为 nil 时应为“未知错误”，收到的值为 %q", got)
	}
}

// TestHTTPStatusErrorsLocalized 通过真实的 doJSON 往返检查状态码分类（拒绝/服务器
// 错误/限流）的用户可见错误是简体中文。发往 127.0.0.1 httptest 默认被拦截，
// 所以只在这个测试里显式放开。
func TestHTTPStatusErrorsLocalized(t *testing.T) {
	t.Setenv(AllowLocalTargetsEnv, "1")
	cases := []struct {
		code int
		want string
	}{
		{http.StatusForbidden, "拒绝请求"},
		{http.StatusInternalServerError, "发生错误"},
		{http.StatusTooManyRequests, "限制请求或"},
	}
	for _, tc := range cases {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(tc.code)
			_, _ = w.Write([]byte("body"))
		}))
		_, err := doJSON(context.Background(), http.MethodPost, srv.URL, nil, map[string]any{"a": 1})
		srv.Close()
		if err == nil {
			t.Fatalf("HTTP %d 应返回错误", tc.code)
		}
		assertChinese(t, "doJSON(HTTP status)", err.Error())
		if !strings.Contains(err.Error(), tc.want) {
			t.Errorf("HTTP %d 错误中应包含 %q，收到的值为 %q", tc.code, tc.want, err.Error())
		}
	}
}
