package server

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"unicode"

	"github.com/Autumn-27/artex/config"
)

// TestLogTTemplatesHaveTranslations 固定本包每个 logT("…") 模板在 config 的日志目录里
// 都有 en/es/ja/ko 译文。目录以中文源模板为键，config.T 在查不到时回落为键本身（中文），
// 因此只要键因为一个字符（空格、转义引号）漂移，其他语言的运营者就会看到中文控制台日志，
// 而没有任何测试发现。同时校验源模板本身是中文（含汉字、不含韩文）。
func TestLogTTemplatesHaveTranslations(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	var keys []string
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, f, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", f, err)
		}
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || len(call.Args) != 1 {
				return true
			}
			if id, ok := call.Fun.(*ast.Ident); !ok || id.Name != "logT" {
				return true
			}
			// 直接写字面量时它就是模板本身；参数是计算表达式（如 map 查询）时，
			// 逐个检查它包含的每个字符串字面量。
			found := 0
			ast.Inspect(call.Args[0], func(n ast.Node) bool {
				lit, ok := n.(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					return true
				}
				s, err := strconv.Unquote(lit.Value)
				if err != nil {
					t.Fatalf("%s: %v", fset.Position(lit.Pos()), err)
				}
				if strings.IndexFunc(s, func(r rune) bool { return unicode.Is(unicode.Hangul, r) }) >= 0 {
					t.Errorf("%s: logT 模板仍含韩文，未汉化：%q", fset.Position(lit.Pos()), s)
					return true
				}
				if strings.IndexFunc(s, func(r rune) bool { return unicode.Is(unicode.Han, r) }) >= 0 {
					keys = append(keys, s)
					found++
				}
				return true
			})
			if found == 0 {
				t.Errorf("%s: logT 参数里没有中文模板字面量", fset.Position(call.Pos()))
			}
			return true
		})
	}
	if len(keys) == 0 {
		t.Fatal("没有扫描到任何 logT 调用点；扫描逻辑已失效")
	}
	for _, lang := range []string{"en", "es", "ja", "ko"} {
		t.Setenv("ARTEX_LANG", lang)
		for _, k := range keys {
			if config.T(k) == k {
				t.Errorf("logT 模板缺少 %s 译文：%q", lang, k)
			}
		}
	}
}

// TestErrCatalogFormatVerbsMatch 固定翻译后的 API 错误模板保持中文模板的格式动词顺序；
// trMsg 在 fmt.Sprintf/Errorf 之前替换模板，动词错位会把响应拼坏。
func TestErrCatalogFormatVerbsMatch(t *testing.T) {
	verbs := regexp.MustCompile(`%[-+# 0-9.]*[a-zA-Z%]`)
	for zh, tr := range errCatalog {
		want := strings.Join(verbs.FindAllString(zh, -1), " ")
		for lang, v := range tr {
			if got := strings.Join(verbs.FindAllString(v, -1), " "); got != want {
				t.Errorf("errCatalog[%q][%s] verbs = %q, want %q", zh, lang, got, want)
			}
		}
	}
}

// TestAuthErrorsTranslated 固定 auth 端点返回的每个 authErr* 消息都有 en/es/ja/ko 译文，
// 这样非中文用户登录失败或首次初始化失败时，能看到所选语言的错误。同时校验源消息本身
// 已是简体中文（含汉字、不含韩文），防止回退。
func TestAuthErrorsTranslated(t *testing.T) {
	t.Cleanup(func() { config.CurrentLanguage = nil })
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "auth.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	var msgs []string
	for _, decl := range file.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.CONST {
			continue
		}
		for _, spec := range gd.Specs {
			vs := spec.(*ast.ValueSpec)
			for i, name := range vs.Names {
				if !strings.HasPrefix(name.Name, "authErr") || i >= len(vs.Values) {
					continue
				}
				if lit, ok := vs.Values[i].(*ast.BasicLit); ok && lit.Kind == token.STRING {
					s, _ := strconv.Unquote(lit.Value)
					msgs = append(msgs, s)
				}
			}
		}
	}
	if len(msgs) == 0 {
		t.Fatal("没有扫描到任何 authErr 常量；扫描逻辑已失效")
	}
	for _, m := range msgs {
		if strings.IndexFunc(m, func(r rune) bool { return unicode.Is(unicode.Hangul, r) }) >= 0 {
			t.Errorf("auth 错误源消息仍含韩文，未汉化：%q", m)
		}
		if strings.IndexFunc(m, func(r rune) bool { return unicode.Is(unicode.Han, r) }) < 0 {
			t.Errorf("auth 错误源消息不是中文：%q", m)
		}
	}
	for _, lang := range []string{"en", "es", "ja", "ko"} {
		config.CurrentLanguage = func() string { return lang }
		for _, m := range msgs {
			if trMsg(m) == m {
				t.Errorf("auth 错误缺少 %s 译文：%q", lang, m)
			}
		}
	}
}
