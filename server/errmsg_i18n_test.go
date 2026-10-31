package server

import (
	"testing"
	"unicode"

	"github.com/Autumn-27/artex/config"
)

// TestTrMsgSwitchesLanguage 固定后端输出语言切换：以中文源消息（目录键）查询时，
// zh（含未接线、含未知语言码回落）原样返回键本身，en/es/ja/ko 返回各自译文。
// 这是 web i18n 目录在后端的对应物——证明 writeErr 响应跟随配置的语言，而不是只有中文。
func TestTrMsgSwitchesLanguage(t *testing.T) {
	t.Cleanup(func() { config.CurrentLanguage = nil })

	hasHangul := func(s string) bool {
		for _, r := range s {
			if unicode.Is(unicode.Hangul, r) {
				return true
			}
		}
		return false
	}
	hasHan := func(s string) bool {
		for _, r := range s {
			if unicode.Is(unicode.Han, r) {
				return true
			}
		}
		return false
	}

	// 一个有代表性的目录键（服务端错误常量的实际值，现为简体中文）。
	const zhMsg = errAssetIDRequired
	if !hasHan(zhMsg) || hasHangul(zhMsg) {
		t.Fatalf("前置条件：源消息必须是中文且不含韩文，实际 %q", zhMsg)
	}

	// 未接线 → ActiveLanguage 兜底 zh → 原样返回键（中文）。
	config.CurrentLanguage = nil
	if got := trMsg(zhMsg); got != zhMsg {
		t.Fatalf("未接线：trMsg 必须原样返回中文键，实际 %q", got)
	}

	// zh → 原样返回键本身。
	config.CurrentLanguage = func() string { return "zh" }
	if got := trMsg(zhMsg); got != zhMsg {
		t.Fatalf("zh：trMsg 必须返回中文源文本，实际 %q", got)
	}

	// en/es/ja/ko → 非空且不等于键；en/es 既不能留韩文也不能留汉字，
	// ko 必须是韩文（不能残留汉字），ja 允许含汉字但不得含韩文。
	for _, lang := range []string{"en", "es", "ja", "ko"} {
		config.CurrentLanguage = func() string { return lang }
		got := trMsg(zhMsg)
		if got == "" || got == zhMsg {
			t.Fatalf("%s：trMsg 必须把消息译出中文源文本，实际 %q", lang, got)
		}
		if hasHangul(got) && lang != "ko" {
			t.Fatalf("%s：译文不应混入韩文，实际 %q", lang, got)
		}
		if (lang == "en" || lang == "es") && hasHan(got) {
			t.Fatalf("%s：译文不应残留汉字，实际 %q", lang, got)
		}
		if lang == "ko" && !hasHangul(got) {
			t.Fatalf("ko：译文必须是韩文，实际 %q", got)
		}
	}

	// 未知语言码 → config.NormalizeLanguage 拒绝 → ActiveLanguage 兜底 zh → 返回键。
	config.CurrentLanguage = func() string { return "fr" }
	if got := trMsg(zhMsg); got != zhMsg {
		t.Fatalf("未知语言：trMsg 必须回落到中文键，实际 %q", got)
	}

	// 目录里没有的字符串（下层透传的错误）原样返回。
	config.CurrentLanguage = func() string { return "en" }
	const notInCatalog = "some-lower-layer-error: connection refused"
	if got := trMsg(notInCatalog); got != notInCatalog {
		t.Fatalf("目录外字符串必须原样透传，实际 %q", got)
	}
}
