package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Autumn-27/artex/config"
)

// TestOutputLanguageCacheFollowsSettings 钉住缓存的输出语言（每次 writeErr 与标签都读它）
// 会被 PUT /api/settings 刷新：切换后下一条响应即刻生效，不必每条消息都回查 settings 表。
func TestOutputLanguageCacheFollowsSettings(t *testing.T) {
	t.Setenv("ARTEX_LANG", "")
	m, err := NewManager(t.TempDir(), "")
	if err != nil {
		t.Skipf("postgres unavailable (%v) — skipping", err)
	}
	defer m.Close()
	prev, hadPrev, _ := m.pg.GetSetting(settingOutputLanguage)
	t.Cleanup(func() { config.CurrentLanguage = nil })
	// defer (not t.Cleanup) so the restore runs before the deferred m.Close.
	defer func() {
		if hadPrev {
			_ = m.pg.SetSetting(settingOutputLanguage, prev)
		} else {
			_, _ = m.pg.Exec(`DELETE FROM settings WHERE key=$1`, settingOutputLanguage)
		}
	}()
	_, _ = m.pg.Exec(`DELETE FROM settings WHERE key=$1`, settingOutputLanguage)

	s := New(context.Background(), m, t.TempDir(), t.TempDir(), t.TempDir())
	token, err := signJWT(s.jwtKey)
	if err != nil {
		t.Fatal(err)
	}
	do := func(method, path string, body any) *httptest.ResponseRecorder {
		var buf bytes.Buffer
		if body != nil {
			_ = json.NewEncoder(&buf).Encode(body)
		}
		req := httptest.NewRequest(method, path, &buf)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		s.Handler().ServeHTTP(rec, req)
		return rec
	}
	errOf := func(rec *httptest.ResponseRecorder) string {
		var v struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(rec.Body.Bytes(), &v)
		return v.Error
	}

	if got := s.outputLanguage(); got != "zh" {
		t.Fatalf("默认输出语言 = %q，期望 zh", got)
	}
	// zh 默认：目录命中中文源文案本身（writeErr 走 trMsg，zh 直接回键）
	if got := errOf(do(http.MethodPut, "/api/settings", map[string]any{"output_language": "fr"})); got != errOutputLanguageInvalid {
		t.Fatalf("zh 默认应返回中文源文案 %q，实际 %q", errOutputLanguageInvalid, got)
	}

	if rec := do(http.MethodPut, "/api/settings", map[string]any{"output_language": "ko"}); rec.Code != http.StatusOK {
		t.Fatalf("set ko status=%d body=%s", rec.Code, rec.Body.String())
	}
	// 切到 ko：同一条错误必须换成韩语译文——既不再是中文键，也不能含汉字
	if got := errOf(do(http.MethodPut, "/api/settings", map[string]any{"output_language": "fr"})); got == "" || got == errOutputLanguageInvalid {
		t.Fatalf("切到 ko 后应返回韩语译文（不应等于中文源 %q），实际 %q", errOutputLanguageInvalid, got)
	} else if !containsHangul(got) {
		t.Fatalf("切到 ko 后的译文应含韩文，实际 %q", got)
	}
	var settings struct {
		OutputLanguage string `json:"output_language"`
	}
	_ = json.Unmarshal(do(http.MethodGet, "/api/settings", nil).Body.Bytes(), &settings)
	if settings.OutputLanguage != "ko" {
		t.Fatalf("GET /api/settings output_language = %q，期望 ko", settings.OutputLanguage)
	}
}

// containsHangul 判断字符串里是否含韩文音节（本文件仅用于“切到 ko 后应拿到韩语译文”的断言）。
func containsHangul(s string) bool {
	for _, r := range s {
		if r >= 0xAC00 && r <= 0xD7A3 {
			return true
		}
	}
	return false
}
