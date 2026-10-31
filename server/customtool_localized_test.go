package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Autumn-27/artex/db"
)

// 回归防护测试：customtool.go 的自定义工具 CRUD 与试跑端点经 writeErr 返回的校验错误
// 响应必须保持简体中文。中文判定复用 F3a 的 assertChineseError（含汉字、不含韩文），
// 响应正文提取复用 task_categories 测试里的 decodeErrorField（同 package server）。
// 智能体读取的 actool.Errorf 工具结果与工具 schema 的 description 属于大脑边界，
// 不在翻译范围内，本测试也不会改动它们。

// TestCustomToolErrorConstantsLocalized 断言经 writeErr 暴露的八个常量全部为简体中文。
// 其中两个（errCustomToolKeyExists、errCustomToolEditCustomOnly）要处理器先调用
// pg.GetTool(DB) 才能到达，下面的真实 HTTP 测试打不到，因此用常量断言钉住，
// 并已通过调用图追踪确认“哪个分支使用哪个常量”
// （与 goals_api、task_control 的 DB 路径校验密度一致）。
func TestCustomToolErrorConstantsLocalized(t *testing.T) {
	for _, c := range []struct{ name, msg string }{
		{"keyFormat", errCustomToolKeyFormat},
		{"kindInvalid", errCustomToolKindInvalid},
		{"httpSchemaRequired", errCustomToolHTTPSchemaRequired},
		{"keyExists", errCustomToolKeyExists},
		{"editCustomOnly", errCustomToolEditCustomOnly},
		{"badBody", errCustomToolBadBody},
		{"shellNoExec", errCustomToolShellNoExec},
		{"unknownKindPrefix", errCustomToolUnknownKindPrefix},
	} {
		assertChineseError(t, "customtool."+c.name, c.msg)
	}
}

// newCustomToolServer 构造一个能过 pg 闸门（s.pg）、但没有真实数据库连接的 Server。
// &db.DB{} 内嵌的 *sql.DB 是 nil，但指针本身非 nil，因此 pg() 不会写 503 而是直接放行。
// 下面的校验分支全部在 pg.GetTool 之类的真实 DB 调用“之前”返回
// （已按代码路径确认），因此不会解引用 nil DB。
func newCustomToolServer() *Server {
	return &Server{m: &Manager{pg: &db.DB{}}}
}

// TestCustomToolCreateValidationLocalized 用真实 HTTP 跑 pgCreateCustomTool 的三条输入
// 校验路径，确认响应正文是简体中文。三条路径都在 pg.GetTool（第 68 行）
// 之前返回。
func TestCustomToolCreateValidationLocalized(t *testing.T) {
	cases := []struct {
		name, body, want string
	}{
		{"bad-key", `{"key":"BadKey","kind":"command"}`, errCustomToolKeyFormat},
		{"bad-kind", `{"key":"goodkey","kind":"bogus"}`, errCustomToolKindInvalid},
		{"http-no-schema", `{"key":"goodkey","kind":"http"}`, errCustomToolHTTPSchemaRequired},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := newCustomToolServer()
			req := httptest.NewRequest(http.MethodPost, "/api/tools/custom", strings.NewReader(c.body))
			rec := httptest.NewRecorder()
			s.pgCreateCustomTool(rec, req)
			if rec.Code != 400 && rec.Code != 409 {
				t.Fatalf("期望验证错误的状态码，实际返回 %d (正文 %q)", rec.Code, rec.Body.String())
			}
			got := decodeErrorField(t, rec.Body.Bytes())
			if got != c.want {
				t.Fatalf("响应正文不一致：got %q want %q", got, c.want)
			}
			assertChineseError(t, "customtool.create."+c.name, got)
		})
	}
}

// TestCustomToolTestRunValidationLocalized 用真实 HTTP 跑 pgTestCustomTool 的三条输入
// 校验路径，确认响应正文是简体中文。三条路径都不涉及 DB 调用。
func TestCustomToolTestRunValidationLocalized(t *testing.T) {
	cases := []struct {
		name, body, want string
	}{
		{"bad-body", `not-json`, errCustomToolBadBody},
		{"shell", `{"kind":"shell"}`, errCustomToolShellNoExec},
		{"unknown-kind", `{"kind":"bogus"}`, errCustomToolUnknownKindPrefix + "bogus"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := newCustomToolServer()
			req := httptest.NewRequest(http.MethodPost, "/api/tools/custom/test", strings.NewReader(c.body))
			rec := httptest.NewRecorder()
			s.pgTestCustomTool(rec, req)
			if rec.Code != 400 {
				t.Fatalf("期望状态码 400，实际返回 %d (正文 %q)", rec.Code, rec.Body.String())
			}
			got := decodeErrorField(t, rec.Body.Bytes())
			if got != c.want {
				t.Fatalf("响应正文不一致：got %q want %q", got, c.want)
			}
			// unknown-kind 后面会拼上 req.Kind("bogus") 的拉丁尾巴，因此只检查是否含汉字。
			assertChineseError(t, "customtool.test."+c.name, errCustomToolUnknownKindPrefix)
		})
	}
}
