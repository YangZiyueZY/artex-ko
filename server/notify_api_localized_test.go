package server

import (
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/Autumn-27/artex/db"
)

// 回归防护测试：notify_api.go 的通知配置 API 响应文案与测试消息必须保持简体中文。
// 中文判定复用 F3a 提供的 assertChineseError（含汉字、不含韩文）。

// TestNotifyErrorConstantsLocalized 断言十个响应常量与三个测试消息常量全部为简体中文；
// 只要有一个被改回韩文，本测试即失败。
func TestNotifyErrorConstantsLocalized(t *testing.T) {
	cases := map[string]string{
		"notifyErrBadJSON":         notifyErrBadJSON,
		"notifyErrKindInvalidFmt":  fmt.Sprintf(notifyErrKindInvalidFmt, "email / webhook"),
		"notifyErrNameMissing":     notifyErrNameMissing,
		"notifyErrNameEmpty":       notifyErrNameEmpty,
		"notifyErrModeInvalid":     notifyErrModeInvalid,
		"notifyErrRateNegative":    notifyErrRateNegative,
		"notifyErrChannelID":       notifyErrChannelID,
		"notifyErrKindUnregFmt":    fmt.Sprintf(notifyErrKindUnregFmt, "bogus"),
		"notifyErrDeliveryID":      notifyErrDeliveryID,
		"notifyErrChannelNotFound": notifyErrChannelNotFound,
		"notifyTestName":           notifyTestName,
		"notifyTestClass":          notifyTestClass,
		"notifyTestSummary":        notifyTestSummary,
	}
	for label, msg := range cases {
		assertChineseError(t, label, msg)
	}
}

// TestNotifyChannelLookupErrLocalized 用真实代码检查「找不到渠道 → 404」路径。
// notifyChannelLookupErr 是不需要 Server/DB、只用 w 与 err 就能跑的包级函数，因此把哨兵
// 错误原样传入即可在没有 DB 的情况下确认 404 响应正文里装的是中文常量。
func TestNotifyChannelLookupErrLocalized(t *testing.T) {
	rec := httptest.NewRecorder()
	notifyChannelLookupErr(rec, db.ErrNotificationChannelNotFound)
	if rec.Code != 404 {
		t.Fatalf("状态码 = %d, 期望 = 404 (正文 %q)", rec.Code, rec.Body.String())
	}
	got := decodeErrorField(t, rec.Body.Bytes())
	if got != notifyErrChannelNotFound {
		t.Fatalf("响应文案 = %q, 期望 = %q", got, notifyErrChannelNotFound)
	}
	assertChineseError(t, "channel_not_found", got)
}

// TestNotifyTestMessageLocalized 以纯函数方式组装用于检查渠道连通性的测试消息，
// 校验真正发给用户的正文（标题、分类、摘要）是简体中文，不需要 DB 与网络。
func TestNotifyTestMessageLocalized(t *testing.T) {
	const base = "https://artex.example.test"
	msg := notifyTestMessage(base)
	if len(msg.Items) != 1 {
		t.Fatalf("测试消息条目数 = %d, 期望 = 1", len(msg.Items))
	}
	item := msg.Items[0]
	assertChineseError(t, "test.name", item.Name)
	assertChineseError(t, "test.class", item.VulnClass)
	assertChineseError(t, "test.summary", item.Summary)
	if msg.HomeURL != base || item.DetailURL != base {
		t.Fatalf("链接保留失败：HomeURL=%q DetailURL=%q (期望 %q)", msg.HomeURL, item.DetailURL, base)
	}
}
