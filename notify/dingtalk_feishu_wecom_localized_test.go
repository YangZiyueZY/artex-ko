package notify

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// asciiStatusChangeItem 构造一个不含 CJK 字符（纯 ASCII）的状态变更条目。
// 数据全 ASCII 时，输出里出现的任何中文都只可能来自骨架标签，断言才能精准锁定标签回退。
func asciiStatusChangeItem() Item {
	return Item{
		FindingID:  7,
		Name:       "login-flaw",
		VulnClass:  "SQLi",
		Severity:   "high",
		Summary:    "SQL injection via q param",
		Assets:     []string{"a.example.com"},
		DetailURL:  "https://platform.example/finding/7",
		FromStatus: "pending",
		ToStatus:   "fixed",
	}
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("卡片序列化失败：%v", err)
	}
	return string(b)
}

// TestFeishuItemLinesLocalized 检查飞书卡片单条正文的字段标签（状态变更·类型·
// 资产·概要）是简体中文，且与 markdown.go（F4②）逐字一致。feishu.go 用的是这些
// 标签的**自己那份副本**（不是三渠道共享的 markdown 函数），所以必须单独钉住这颗针，
// 才能在共享函数之外抓到「只有飞书正文回退成别的语言」的回归。
func TestFeishuItemLinesLocalized(t *testing.T) {
	got := feishuItemLines(asciiStatusChangeItem())
	assertChinese(t, "feishuItemLines", got)
	if hasHangul(got) {
		t.Errorf("飞书单条正文中残留韩文：\n%s", got)
	}
	for _, want := range []string{
		"**状态变更**: 待处理 → 已修复",
		"**类型**: SQLi",
		"**资产**: a.example.com",
		"**概要**: SQL injection via q param",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("飞书单条正文中应包含 %q：\n%s", want, got)
		}
	}
}

// TestFeishuCardButtonsLocalized 检查飞书卡片按钮标签（查看详情·在平台查看全部）
// 是简体中文，并且整张卡片不残留韩文（数据为 ASCII）。
func TestFeishuCardButtonsLocalized(t *testing.T) {
	// 单条：「查看详情」按钮（有 DetailURL 时）
	single, _ := feishuCard(Message{Items: []Item{asciiStatusChangeItem()}})
	singleJSON := mustJSON(t, single)
	if hasHangul(singleJSON) {
		t.Errorf("单条飞书卡片中残留韩文：\n%s", singleJSON)
	}
	if !strings.Contains(singleJSON, "查看详情") {
		t.Errorf("单条飞书卡片中应有「查看详情」按钮：\n%s", singleJSON)
	}

	// 多条：「在平台查看全部」按钮（有 HomeURL 时）
	batch, _ := feishuCard(Message{Batch: true, HomeURL: "https://platform.example", Items: hanFreeItems(2)})
	batchJSON := mustJSON(t, batch)
	if hasHangul(batchJSON) {
		t.Errorf("多条飞书卡片中残留韩文：\n%s", batchJSON)
	}
	if !strings.Contains(batchJSON, "在平台查看全部") {
		t.Errorf("多条飞书卡片中应有「在平台查看全部」按钮：\n%s", batchJSON)
	}
	if !strings.Contains(batchJSON, "漏洞摘要 · 共 2 条") {
		t.Errorf("多条飞书卡片标题应为「漏洞摘要 · 共 2 条」：\n%s", batchJSON)
	}
}

// TestDingTalkActionCardButtonLocalized 通过真实 Send 路径（假接收端）检查钉钉
// ActionCard 的「查看详情」按钮（singleTitle）是简体中文。
func TestDingTalkActionCardButtonLocalized(t *testing.T) {
	var singleTitle string
	srv := capturePost(t, `{"errcode":0,"errmsg":"ok"}`, func(_ *testing.T, body map[string]any, _ *http.Request) {
		card, _ := body["actionCard"].(map[string]any)
		singleTitle, _ = card["singleTitle"].(string)
	})
	m := Message{Items: []Item{asciiStatusChangeItem()}}
	if _, err := (dingTalkChannel{}).Send(context.Background(), map[string]any{"webhook": srv.URL}, m); err != nil {
		t.Fatalf("投递失败：%v", err)
	}
	if singleTitle != "查看详情" {
		t.Errorf("钉钉 ActionCard 的 singleTitle 应为「查看详情」，收到的值为 %q", singleTitle)
	}
}

// TestChinaPlatformValidateLocalized 检查钉钉·飞书·企业微信的配置校验错误是简体中文，
// 并保留「Webhook」这个技术词（它会通过通知设置保存·测试 API 直接展示给用户）。
func TestChinaPlatformValidateLocalized(t *testing.T) {
	for _, kind := range []string{KindDingTalk, KindFeishu, KindWeCom} {
		ch, ok := Get(kind)
		if !ok {
			t.Fatalf("%s 渠道未注册", kind)
		}

		// 空配置：「没有 Webhook 地址」
		missing := ch.Validate(map[string]any{})
		if missing == nil {
			t.Fatalf("%s：空配置应校验失败", kind)
		}
		assertChinese(t, kind+" missing", missing.Error())
		if !strings.Contains(missing.Error(), "Webhook") || !strings.Contains(missing.Error(), "没有") {
			t.Errorf("%s：缺失错误应为「没有 Webhook 地址」，收到的值为 %q", kind, missing)
		}

		// 非法地址（ftp）：「Webhook 地址无效：……」
		bad := ch.Validate(map[string]any{"webhook": "ftp://x"})
		if bad == nil {
			t.Fatalf("%s：ftp 地址应校验失败", kind)
		}
		if hasHangul(bad.Error()) {
			t.Errorf("%s：无效地址错误中残留韩文：%q", kind, bad)
		}
		if !strings.Contains(bad.Error(), "Webhook 地址无效") {
			t.Errorf("%s：无效地址错误应以「Webhook 地址无效」开头，收到的值为 %q", kind, bad)
		}
	}
}

// TestChinaPlatformSendErrorsLocalized 检查钉钉·飞书·企业微信的发送失败原因以简体中文
// 写入投递历史（last_error）。平台名与 UI 渠道标签（DingTalk/Feishu/WeCom）一致。
// 平台返回的 errmsg 是 ASCII，所以错误里若出现韩文，说明骨架文案还没汉化干净。
func TestChinaPlatformSendErrorsLocalized(t *testing.T) {
	// 业务错误码（HTTP 200 + body 里的 errcode/code != 0）路径。
	bizCases := []struct {
		kind     string
		resp     string
		wantSubs []string
	}{
		{KindDingTalk, `{"errcode":310000,"errmsg":"keyword not matched"}`, []string{"DingTalk 发生错误", "310000"}},
		{KindFeishu, `{"code":19021,"msg":"sign error"}`, []string{"Feishu 返回错误", "19021"}},
		{KindWeCom, `{"errcode":45009,"errmsg":"freq out of limit"}`, []string{"WeCom 请求被限制", "45009"}},
		{KindWeCom, `{"errcode":93000,"errmsg":"invalid webhook"}`, []string{"WeCom 发生错误", "93000"}},
	}
	for _, tc := range bizCases {
		srv := capturePost(t, tc.resp, nil)
		ch, ok := Get(tc.kind)
		if !ok {
			t.Fatalf("%s 渠道未注册", tc.kind)
		}
		_, err := ch.Send(context.Background(), map[string]any{"webhook": srv.URL}, singleMsg())
		if err == nil {
			t.Fatalf("%s：%s 响应应为错误", tc.kind, tc.resp)
		}
		if hasHangul(err.Error()) {
			t.Errorf("%s：发送错误中残留韩文：%q", tc.kind, err)
		}
		for _, sub := range tc.wantSubs {
			if !strings.Contains(err.Error(), sub) {
				t.Errorf("%s：发送错误中应包含 %q，收到的值为 %q", tc.kind, sub, err)
			}
		}
	}

	// 响应不是 JSON 时的解析失败路径：「<平台> 无法解析响应」。
	parseCases := []struct {
		kind     string
		platform string
	}{
		{KindDingTalk, "DingTalk"},
		{KindFeishu, "Feishu"},
		{KindWeCom, "WeCom"},
	}
	for _, pc := range parseCases {
		srv := capturePost(t, `not-json`, nil)
		ch, ok := Get(pc.kind)
		if !ok {
			t.Fatalf("%s 渠道未注册", pc.kind)
		}
		_, err := ch.Send(context.Background(), map[string]any{"webhook": srv.URL}, singleMsg())
		if err == nil {
			t.Fatalf("%s：非 JSON 响应应为错误", pc.kind)
		}
		want := pc.platform + " 无法解析响应"
		if !strings.Contains(err.Error(), want) {
			t.Errorf("%s：解析失败错误中应包含 %q，收到的值为 %q", pc.kind, want, err)
		}
	}
}
