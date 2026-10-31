package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// 回归防护测试：task_metadata.go 的任务元数据修改 API 错误响应必须保持简体中文。
// 中文判定复用 F3a 提供的 assertChineseError（含汉字、不含韩文）。

// TestTaskMetadataErrorConstantsLocalized 断言四个响应常量全部为简体中文；
// 只要有一个被改回韩文，本测试即失败。
func TestTaskMetadataErrorConstantsLocalized(t *testing.T) {
	cases := map[string]string{
		"errTaskMetaRequestTooLarge": errTaskMetaRequestTooLarge,
		"errTaskMetaNoFields":        errTaskMetaNoFields,
		"errTaskMetaNameEmpty":       errTaskMetaNameEmpty,
		"errTaskMetaNameTooLongFmt":  fmt.Sprintf(errTaskMetaNameTooLongFmt, maxTaskNameRunes),
	}
	for label, msg := range cases {
		assertChineseError(t, label, msg)
	}
}

// TestUpdateTaskMetadataResponsesLocalized 检查四条输入校验路径，一直校验到真实 HTTP
// 响应正文。updateTaskMetadata 的这四条路径（正文超限、缺少字段、名称为空、名称超长）
// 只读 s.m.Task(查表)与请求正文，在调用 s.m.UpdateTaskMetadata(DB) 之前就返回，因此
// 只要在 tasks 映射里放一个任务就能在没有 DB 的情况下跑到底，并确认常量确实进入响应。
func TestUpdateTaskMetadataResponsesLocalized(t *testing.T) {
	s := &Server{m: &Manager{tasks: map[string]*Task{"t1": {ID: "t1"}}}}

	call := func(body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPatch, "/api/tasks/t1/metadata", strings.NewReader(body))
		req.SetPathValue("id", "t1")
		rec := httptest.NewRecorder()
		s.updateTaskMetadata(rec, req)
		return rec
	}
	errBody := func(t *testing.T, rec *httptest.ResponseRecorder) string {
		t.Helper()
		var resp struct {
			Error string `json:"error"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("响应 JSON 解析失败：%v (正文 %q)", err, rec.Body.String())
		}
		return resp.Error
	}

	cases := []struct {
		name string
		body string
		code int
		want string
	}{
			// 超过 16KB 上限的合法 JSON 正文。MaxBytesReader 会在读取过程中返回超限。
		{"request_too_large", `{"name":"` + strings.Repeat("a", maxTaskMetadataRequestSize+1024) + `"}`, http.StatusRequestEntityTooLarge, errTaskMetaRequestTooLarge},
			// name 与 pinned 都不传，就没有要修改的字段。
		{"no_fields", `{}`, http.StatusBadRequest, errTaskMetaNoFields},
		{"name_empty", `{"name":"  "}`, http.StatusBadRequest, errTaskMetaNameEmpty},
		{"name_too_long", `{"name":"` + strings.Repeat("测", maxTaskNameRunes+1) + `"}`, http.StatusBadRequest, fmt.Sprintf(errTaskMetaNameTooLongFmt, maxTaskNameRunes)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec := call(c.body)
			if rec.Code != c.code {
				t.Fatalf("状态码 = %d, 期望 = %d (正文 %q)", rec.Code, c.code, rec.Body.String())
			}
			if got := errBody(t, rec); got != c.want {
				t.Fatalf("响应文案 = %q, 期望 = %q", got, c.want)
			}
			assertChineseError(t, c.name, errBody(t, rec))
		})
	}
}
