package server

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
)

// 回归防护测试：chatupload.go 的聊天附件上传 API 错误响应必须保持简体中文。
// 中文判定复用 F3a 的 assertChineseError（含汉字、不含韩文），响应正文提取复用
// task_categories 测试里的 decodeErrorField（同 package server）。

// TestChatUploadErrorConstantsLocalized 断言七个响应常量全部为简体中文。
// 其中三条是后面要拼接 err.Error() 的前缀常量，结尾会带 ": "，
// 但只要含汉字且不含韩文就能通过判定。
func TestChatUploadErrorConstantsLocalized(t *testing.T) {
	cases := map[string]string{
		"scope_invalid": errChatUploadScopeInvalid,
		"bad_id":        errChatUploadBadID,
		"task_deleting": errChatUploadTaskDeleting,
		"mkdir":         errChatUploadMkdir,
		"parse":         errChatUploadParse,
		"no_file":       errChatUploadNoFile,
		"save_failed":   errChatUploadSaveFailed,
	}
	for label, msg := range cases {
		assertChineseError(t, label, msg)
	}
}

// TestChatUploadHandlerResponsesLocalized 检查不碰 DB 就能走完的 chatUpload 路径，
// 一直校验到真实 HTTP 响应正文。scope/id 校验在触及 Manager 之前就返回；
// no-file 用 scope=session（非任务），因此无需引擎与 DB，只用临时目录即可跑完；
// task-deleting 与 goals 测试一样，用删除闸门（engine.deleting）造出 409。
func TestChatUploadHandlerResponsesLocalized(t *testing.T) {
	// 只有字段、没有 "file" 的 multipart 正文 — ParseMultipartForm 会通过，但文件数为 0。
	noFileBody := func() (*bytes.Buffer, string) {
		var buf bytes.Buffer
		mw := multipart.NewWriter(&buf)
		_ = mw.WriteField("other", "x")
		_ = mw.Close()
		return &buf, mw.FormDataContentType()
	}

	t.Run("scope-invalid", func(t *testing.T) {
		s := &Server{}
		req := httptest.NewRequest(http.MethodPost, "/api/chat/upload?scope=bogus&id=x", nil)
		rec := httptest.NewRecorder()
		s.chatUpload(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("状态码 = %d, 期望 = 400 (正文 %q)", rec.Code, rec.Body.String())
		}
		if got := decodeErrorField(t, rec.Body.Bytes()); got != errChatUploadScopeInvalid {
			t.Fatalf("响应文案 = %q, 期望 = %q", got, errChatUploadScopeInvalid)
		}
		assertChineseError(t, "scope-invalid", errChatUploadScopeInvalid)
	})

	t.Run("bad-id", func(t *testing.T) {
		s := &Server{}
		// scope=session（非任务），因此在触及 Manager 之前就在校验 id 时返回。
		req := httptest.NewRequest(http.MethodPost, "/api/chat/upload?scope=session&id=../evil", nil)
		rec := httptest.NewRecorder()
		s.chatUpload(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("状态码 = %d, 期望 = 400 (正文 %q)", rec.Code, rec.Body.String())
		}
		if got := decodeErrorField(t, rec.Body.Bytes()); got != errChatUploadBadID {
			t.Fatalf("响应文案 = %q, 期望 = %q", got, errChatUploadBadID)
		}
		assertChineseError(t, "bad-id", errChatUploadBadID)
	})

	t.Run("no-file", func(t *testing.T) {
		s := &Server{m: &Manager{dir: t.TempDir()}}
		body, ctype := noFileBody()
		req := httptest.NewRequest(http.MethodPost, "/api/chat/upload?scope=session&id=sess1", body)
		req.Header.Set("Content-Type", ctype)
		rec := httptest.NewRecorder()
		s.chatUpload(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("状态码 = %d, 期望 = 400 (正文 %q)", rec.Code, rec.Body.String())
		}
		if got := decodeErrorField(t, rec.Body.Bytes()); got != errChatUploadNoFile {
			t.Fatalf("响应文案 = %q, 期望 = %q", got, errChatUploadNoFile)
		}
		assertChineseError(t, "no-file", errChatUploadNoFile)
	})

	t.Run("task-deleting", func(t *testing.T) {
		s := &Server{m: &Manager{tasks: map[string]*Task{"t1": {ID: "t1"}}}, engine: &Engine{}}
		s.engine.deleting.Store("t1", true)
		req := httptest.NewRequest(http.MethodPost, "/api/chat/upload?scope=task&id=t1", nil)
		rec := httptest.NewRecorder()
		s.chatUpload(rec, req)
		if rec.Code != http.StatusConflict {
			t.Fatalf("状态码 = %d, 期望 = 409 (正文 %q)", rec.Code, rec.Body.String())
		}
		if got := decodeErrorField(t, rec.Body.Bytes()); got != errChatUploadTaskDeleting {
			t.Fatalf("响应文案 = %q, 期望 = %q", got, errChatUploadTaskDeleting)
		}
		assertChineseError(t, "task-deleting", errChatUploadTaskDeleting)
	})
}
