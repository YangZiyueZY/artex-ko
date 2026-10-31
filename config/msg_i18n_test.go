package config

import (
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestStartupMessageSwitchesLanguage 钉住 DB 配置启动错误与来源标签跟随 ARTEX_LANG：
// 默认简体中文（zh 是源语言），配置 en/ko/es/ja 时取各自译文。运营者需要据此操作的
// 标识符一律保持原样。
func TestStartupMessageSwitchesLanguage(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ARTEX_PG_DSN", "")
	t.Setenv("ARTEX_CONFIG", filepath.Join(dir, "nope.json"))

	// 默认（未设 ARTEX_LANG）→ 简体中文（本项目的源语言）。
	t.Setenv("ARTEX_LANG", "")
	def := mustDSNErr(t)
	assertChinese(t, "默认启动错误", def)
	for _, id := range []string{"ARTEX_PG_DSN", "database", "dsn", "host/user/dbname"} {
		if !containsStr(def, id) {
			t.Errorf("默认启动错误必须保留标识符 %q：%q", id, def)
		}
	}

	// 显式韩语 → 韩文译文。
	t.Setenv("ARTEX_LANG", "ko")
	ko := mustDSNErr(t)
	assertKoreanText(t, "ko startup error", ko)

	// 英语 → 无韩文、无汉字，并保留标识符。
	t.Setenv("ARTEX_LANG", "en")
	en := mustDSNErr(t)
	if hasHangul(en) {
		t.Errorf("en 启动错误里残留韩文：%q", en)
	}
	if hasHan(en) {
		t.Errorf("en 启动错误里残留中文汉字：%q", en)
	}
	for _, id := range []string{"ARTEX_PG_DSN", "database", "dsn", "host/user/dbname"} {
		if !containsStr(en, id) {
			t.Errorf("en 启动错误必须保留标识符 %q：%q", id, en)
		}
	}

	// 中文 / 西班牙语 / 日语 → 非空，并且与韩语译文不同（确保真的按语言切换了）。
	for _, lang := range []string{"zh", "es", "ja"} {
		t.Setenv("ARTEX_LANG", lang)
		if got := mustDSNErr(t); got == "" {
			t.Errorf("%s 启动错误不应为空", lang)
		} else if got == ko {
			t.Errorf("%s 启动错误与韩语译文相同，语言切换未生效：%q", lang, got)
		}
	}
}

func mustDSNErr(t *testing.T) string {
	t.Helper()
	_, _, err := PostgresDSN()
	if err == nil {
		t.Fatal("缺少配置时应当报错")
	}
	return err.Error()
}

func containsStr(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// TestCatalogFormatVerbsMatch 钉住每条译文都按同样的顺序保留中文模板里的格式动词。
// 调用方只在中文模板外面套一层（log.Printf(T(zh), args...)），所以动词丢了或换了顺序
// 就会在运营者控制台上打出 %!s(MISSING) 或把参数调换。
func TestCatalogFormatVerbsMatch(t *testing.T) {
	verbs := regexp.MustCompile(`%[-+# 0-9.]*[a-zA-Z%]`)
	for name, cat := range map[string]map[string]map[string]string{"cfgCatalog": cfgCatalog, "logCatalog": logCatalog} {
		for zh, tr := range cat {
			want := strings.Join(verbs.FindAllString(zh, -1), " ")
			for lang, v := range tr {
				if got := strings.Join(verbs.FindAllString(v, -1), " "); got != want {
					t.Errorf("%s[%q][%s] 格式动词 = %q，期望 %q", name, zh, lang, got, want)
				}
			}
		}
	}
}
