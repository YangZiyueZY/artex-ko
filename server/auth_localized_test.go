package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// 回归防护测试：守护认证响应文案的汉化（backlog F3b）。用两条无需 DB 也能跑通的
// 路径校验：① requireAuth 中间件不经过 pg，因此检查到真实 HTTP 响应正文；
// ② 其余处理器的文案是具名常量，直接检查常量本身。
// 中文判定复用 F3a 提供的 assertChineseError 助手（含汉字、不含韩文）。

// TestRequireAuthMessagesLocalized 用真实 HTTP 处理器确认：token 缺失或错误时
// requireAuth 返回的 401 响应正文是简体中文且没有韩文残留。该中间件不使用
// 数据库，因此在没有 DB 的环境里也能跑到底。
func TestRequireAuthMessagesLocalized(t *testing.T) {
	s := &Server{jwtKey: []byte(strings.Repeat("k", 32))}
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := s.requireAuth(next)

	cases := []struct {
		name   string
		bearer string
		want   string
	}{
		{"无令牌", "", authErrUnauthorized},
		{"令牌无效", "Bearer not-a-valid-token", authErrTokenInvalid},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/api/tasks", nil)
			if c.bearer != "" {
				r.Header.Set("Authorization", c.bearer)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, r)

			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("状态码 = %d，期望 401", rec.Code)
			}
			body := rec.Body.String()
			if !strings.Contains(body, c.want) {
				t.Errorf("响应正文中缺少 %q: %s", c.want, body)
			}
			assertChineseError(t, c.name, body)
		})
	}
}

// TestAuthErrorConstantsLocalized 断言认证处理器使用的全部用户可见文案常量都
// 含汉字且不含韩文，从而无需 DB 也能覆盖那些无法直接用 requireAuth 打到的
// （= 需要经过 pg 的）处理器文案。
func TestAuthErrorConstantsLocalized(t *testing.T) {
	consts := map[string]string{
		"authErrUnauthorized":          authErrUnauthorized,
		"authErrTokenInvalid":          authErrTokenInvalid,
		"authErrPasswordAlreadySet":    authErrPasswordAlreadySet,
		"authErrPasswordEmpty":         authErrPasswordEmpty,
		"authErrNewPasswordEmpty":      authErrNewPasswordEmpty,
		"authErrPasswordHash":          authErrPasswordHash,
		"authErrSaveFailedPrefix":      authErrSaveFailedPrefix,
		"authErrTokenGen":              authErrTokenGen,
		"authErrBadRequest":            authErrBadRequest,
		"authErrPasswordNotInit":       authErrPasswordNotInit,
		"authErrCurrentPasswordWrong":  authErrCurrentPasswordWrong,
		"authErrBadCredential":         authErrBadCredential,
		"authErrPasswordTooShort":      authErrPasswordTooShort,
		"authErrSetupTokenInvalid":     authErrSetupTokenInvalid,
		"authErrTooManyAttempts":       authErrTooManyAttempts,
		"authErrPasswordTooLong":       authErrPasswordTooLong,
		"authErrDataSourceUnavailable": authErrDataSourceUnavailable,
	}
	for name, msg := range consts {
		assertChineseError(t, name, msg)
	}
}
