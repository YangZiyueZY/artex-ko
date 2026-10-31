package server

import (
	"net/url"
	"testing"
	"unicode"
)

// assertChineseError 在 msg 为空、仍残留韩文（= 未被汉化的旧文案），或完全不含汉字
// （= 从未汉化）时失败。ASCII 字段名与枚举值允许出现；以是否含汉字（unicode.Han）
// 判定“已汉化”。
func assertChineseError(t *testing.T, label, msg string) {
	t.Helper()
	if msg == "" {
		t.Fatalf("%s: 消息为空", label)
	}
	hasHan := false
	for _, r := range msg {
		if unicode.Is(unicode.Hangul, r) {
			t.Fatalf("%s：仍残留韩文：%q", label, msg)
		}
		if unicode.Is(unicode.Han, r) {
			hasHan = true
		}
	}
	if !hasHan {
		t.Fatalf("%s：没有中文汉字：%q", label, msg)
	}
}

// TestInterceptArchiveErrorsLocalized 守护第一组（F3）汉化：intercept.go 与
// task_archives.go 里纯校验函数产生的每条用户可见错误响应都必须是简体中文
// （含汉字、不含韩文）。若有人把这些字面量改回韩文或其他语言，本测试立即失败。
func TestInterceptArchiveErrorsLocalized(t *testing.T) {
	// intercept.go: interceptFilterParams — query-string validation.
	if _, err := interceptFilterParams(url.Values{"status": {"bogus"}}); err == nil {
		t.Fatal("status 的校验不应通过")
	} else {
		assertChineseError(t, "filter.status", err.Error())
	}
	if _, err := interceptFilterParams(url.Values{"decision_source": {"bogus"}}); err == nil {
		t.Fatal("decision_source 的校验不应通过")
	} else {
		assertChineseError(t, "filter.decision_source", err.Error())
	}

	// intercept.go: validateInterceptRuleReq — each branch isolated by making the
	// earlier fields valid.
	ruleCases := []struct {
		label string
		req   interceptRuleReq
	}{
		{"rule.name", interceptRuleReq{Name: ""}},
		{"rule.match_target", interceptRuleReq{Name: "r", MatchTarget: "bad"}},
		{"rule.match_type", interceptRuleReq{Name: "r", MatchTarget: "tool_name", MatchType: "bad"}},
		{"rule.pattern_empty", interceptRuleReq{Name: "r", MatchTarget: "tool_name", MatchType: "string", Pattern: ""}},
		{"rule.action", interceptRuleReq{Name: "r", MatchTarget: "tool_name", MatchType: "string", Pattern: "p", Action: "bad"}},
		{"rule.regex", interceptRuleReq{Name: "r", MatchTarget: "tool_name", MatchType: "regex", Pattern: "(", Action: "allow"}},
	}
	for _, tc := range ruleCases {
		err := validateInterceptRuleReq(tc.req)
		if err == nil {
			t.Fatalf("%s: 校验不应通过", tc.label)
		}
		assertChineseError(t, tc.label, err.Error())
	}

	// task_archives.go: normalizeArchiveIDs — batch id validation.
	if _, err := normalizeArchiveIDs(nil); err == nil {
		t.Fatal("空的 archive_ids 校验不应通过")
	} else {
		assertChineseError(t, "archive_ids.empty", err.Error())
	}
	tooMany := make([]int64, 101)
	for i := range tooMany {
		tooMany[i] = int64(i + 1)
	}
	if _, err := normalizeArchiveIDs(tooMany); err == nil {
		t.Fatal("101 个 archive_ids 的校验不应通过")
	} else {
		assertChineseError(t, "archive_ids.too_many", err.Error())
	}
	if _, err := normalizeArchiveIDs([]int64{0}); err == nil {
		t.Fatal("archive id 为 0 的校验不应通过")
	} else {
		assertChineseError(t, "archive_ids.nonpositive", err.Error())
	}

	// task_archives.go: validateArchivePath — package path validation.
	if err := validateArchivePath("/tmp/artex-data", ""); err == nil {
		t.Fatal("空的归档路径校验不应通过")
	} else {
		assertChineseError(t, "archive_path.empty", err.Error())
	}
	if err := validateArchivePath("/tmp/artex-data", "/etc/passwd"); err == nil {
		t.Fatal("管理目录之外的路径校验不应通过")
	} else {
		assertChineseError(t, "archive_path.outside", err.Error())
	}
}
