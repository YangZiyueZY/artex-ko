package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// hasHan 探测 s 里是否含 CJK 汉字——汉化后这是「源语言」本身。
func hasHan(s string) bool {
	for _, r := range s {
		if r >= 0x4e00 && r <= 0x9fff {
			return true
		}
	}
	return false
}

// hasHangul 探测 s 里是否含韩文。韩文只应出现在语言目录的 "ko" 译文里。
func hasHangul(s string) bool {
	for _, r := range s {
		if r >= 0xac00 && r <= 0xd7a3 {
			return true
		}
	}
	return false
}

// assertChinese 断言文案是简体中文：含汉字、不含韩文。
func assertChinese(t *testing.T, label, s string) {
	t.Helper()
	if hasHangul(s) {
		t.Errorf("%s：仍残留韩文：%q", label, s)
	}
	if !hasHan(s) {
		t.Errorf("%s：没有中文汉字：%q", label, s)
	}
}

// assertKoreanText 断言文案确实回落到了语言目录里的韩语译文（专门测 ko 分支的
// 用例才用它；韩文只允许作为面向韩语用户的译文存在）。
func assertKoreanText(t *testing.T, label, s string) {
	t.Helper()
	if !hasHangul(s) {
		t.Errorf("%s：期望韩语译文，但结果里没有韩文：%q", label, s)
	}
}

// TestPostgresDSNErrorLocalized 钉住安装期启动错误在默认语言下是简体中文。
// 当 ARTEX_PG_DSN 与配置文件都没给出数据库时，PostgresDSN 返回的错误会原样传播
// （db.DSN → server.NewManager → cmd/artex/main.go 里的 log.Fatalf("open stores: %v", err)）。
// 配置缺失时运营者第一眼看到的就是它，所以必须读得通、是中文。
func TestPostgresDSNErrorLocalized(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ARTEX_PG_DSN", "")
	t.Setenv("ARTEX_CONFIG", filepath.Join(dir, "nope.json"))
	// 清空 ARTEX_LANG，走默认语言（zh），防止外部环境变量漏进来。
	t.Setenv("ARTEX_LANG", "")

	_, _, err := PostgresDSN()
	if err == nil {
		t.Fatal("缺少配置时应当报错")
	}
	assertChinese(t, "启动错误", err.Error())

	// 运营者需要据此操作的标识符保持原样（不翻译）。
	for _, want := range []string{"ARTEX_PG_DSN", "database", "dsn", "host/user/dbname"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("启动错误应原样保留标识符 %q：%q", want, err.Error())
		}
	}

	// 显式设为 ko 时必须拿到语言目录里的韩语译文（这是有意保留的韩文）。
	t.Setenv("ARTEX_LANG", "ko")
	_, _, koErr := PostgresDSN()
	if koErr == nil {
		t.Fatal("缺少配置时应当报错（ko）")
	}
	assertKoreanText(t, "ko 启动错误", koErr.Error())
	if koErr.Error() == err.Error() {
		t.Errorf("ko 译文没有与中文键区分开：%q", koErr.Error())
	}

	// 记录传播后运营者实际看到的样子（这条路径上没有任何 %w 包装）。
	t.Logf("open stores: %v", err)
}

// TestPostgresDSNSourceLocalized 钉住三种来源标签（打在 server/manager.go:361）
// 在默认语言下是简体中文。它们与启动错误同在一个函数里，必须一起汉化，
// 否则同一个文件里会两种语言混着来。
func TestPostgresDSNSourceLocalized(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	t.Setenv("ARTEX_LANG", "")

	// 环境变量来源
	t.Setenv("ARTEX_CONFIG", filepath.Join(dir, "nope.json"))
	t.Setenv("ARTEX_PG_DSN", "postgres://envwins/x")
	if _, source, err := PostgresDSN(); err != nil {
		t.Fatalf("环境变量来源：%v", err)
	} else {
		assertChinese(t, "env source", source)
	}

	// 配置文件 dsn 来源
	t.Setenv("ARTEX_PG_DSN", "")
	if err := os.WriteFile(cfgPath, []byte(`{"database":{"dsn":"postgres://full/dsn"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ARTEX_CONFIG", cfgPath)
	if _, source, err := PostgresDSN(); err != nil {
		t.Fatalf("dsn 来源：%v", err)
	} else {
		assertChinese(t, "dsn source", source)
	}

	// 配置文件字段来源
	if err := os.WriteFile(cfgPath, []byte(`{"database":{"host":"10.1.2.3","dbname":"d","user":"u"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, source, err := PostgresDSN(); err != nil {
		t.Fatalf("字段来源：%v", err)
	} else {
		assertChinese(t, "fields source", source)
	}

	// 切到 ko 时三种来源标签都应换成韩文译文（防语言表漏项）。
	t.Setenv("ARTEX_LANG", "ko")
	if _, source, err := PostgresDSN(); err != nil {
		t.Fatalf("ko 字段来源：%v", err)
	} else {
		assertKoreanText(t, "ko fields source", source)
	}
}
