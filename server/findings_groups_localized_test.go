package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// 回归防护测试：findings_groups.go 的用户可见文案必须保持简体中文。
// 中文判定复用 F3a 提供的 assertChineseError（含汉字、不含韩文）。

// TestDeepenFindingBodyTooLargeLocalized 用真实 HTTP 确认正文超限（413）响应是中文。
// 该路径在 MaxBytesReader 解码阶段就返回，不会触及 s.m.pg(DB)，因此空 Server 也能跑到底。
func TestDeepenFindingBodyTooLargeLocalized(t *testing.T) {
	s := &Server{}
	body := `{"description":"` + strings.Repeat("a", 33<<10) + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/exploration/findings/1/deepen", strings.NewReader(body))
	req.SetPathValue("id", "1")
	rec := httptest.NewRecorder()
	s.deepenFinding(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("状态码 = %d, 期望 = %d (正文 %q)", rec.Code, http.StatusRequestEntityTooLarge, rec.Body.String())
	}
	var resp struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("响应 JSON 解析失败：%v (正文 %q)", err, rec.Body.String())
	}
	const want = "请求正文过大"
	if resp.Error != want {
		t.Fatalf("响应文案 = %q, 期望 = %q", resp.Error, want)
	}
	assertChineseError(t, "body_too_large", resp.Error)
}

// TestFindingFollowUpAuditSummaryLocalized 断言后序意图的活动摘要常量为简体中文。
// 保存该摘要的 AddFindingFollowUpIntent 需要 DB 事务，在没有 DB 的机器上跑不到底，
// 因此直接断言常量本身。该文案是与意图 payload（用户输入的 description）相互独立的、
// 仅用于活动时间线展示的摘要。
func TestFindingFollowUpAuditSummaryLocalized(t *testing.T) {
	assertChineseError(t, "follow_up_summary", auditFindingFollowUpSummary)
}
