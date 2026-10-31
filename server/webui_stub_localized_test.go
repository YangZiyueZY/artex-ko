//go:build !embedui

package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestWebUIStubErrorLocalized 守护未内嵌前端桩的 HTTP 响应：默认（!embedui）构建
// 必须返回中文 404 正文，而不是旧的英文/韩文文案。它是 server/ 里唯一绕过
// `writeErr(` 批量汉化的用户可见 HTTP 错误响应（用的是 http.Error）。若有人把这段
// 字面量改回韩文，本测试即失败。
func TestWebUIStubErrorLocalized(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	(&Server{}).webuiHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("状态码不是 404：%d", rec.Code)
	}
	body := strings.TrimSpace(rec.Body.String())
	assertChineseError(t, "webui.stub", body)
	// 命令引用必须原样保留（不属于翻译范围）。
	for _, want := range []string{"next dev", "-tags embedui"} {
		if !strings.Contains(body, want) {
			t.Fatalf("响应中没有命令引用 %q：%q", want, body)
		}
	}
}
