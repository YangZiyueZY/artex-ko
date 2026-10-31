package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// 回归防护测试：conversations.go 的对话（聊天）API 错误响应必须保持简体中文。
// 中文判定复用 F3a 提供的 assertChineseError（含汉字、不含韩文）。

// TestConversationErrorConstantsLocalized 断言十二个响应常量全部为简体中文。
// 处理器要先过 s.pg(w)(DB)，在没有 DB 的机器上跑不到底，因此直接断言常量本身
// （auth.go 先例）。含 %d 的格式字符串用真实实参填充后再检查最终文案。
func TestConversationErrorConstantsLocalized(t *testing.T) {
	cases := []struct {
		name string
		msg  string
	}{
		{"request_too_large", convErrRequestTooLarge},
		{"agent_key_empty", convErrAgentKeyEmpty},
		{"agent_key_too_long", fmt.Sprintf(convErrAgentKeyTooLong, maxConversationAgentKeyRunes)},
		{"agent_not_found", convErrAgentNotFound},
		{"llm_profile", convErrLLMProfile},
		{"title_too_long", fmt.Sprintf(convErrTitleTooLong, maxConversationTitleRunes)},
		{"title_or_pinned", convErrTitleOrPinned},
		{"title_empty", convErrTitleEmpty},
		{"bad_conv_id", convErrBadConvID},
		{"ids_count", fmt.Sprintf(convErrIDsCount, maxConversationDeleteBatch)},
		{"message_empty", convErrMessageEmpty},
		{"busy", convErrBusy},
	}
	for _, c := range cases {
		assertChineseError(t, c.name, c.msg)
	}
}

// TestDecodeConversationRequestTooLargeLocalized 检查请求正文超限路径，一直校验到真实
// HTTP 响应正文。decodeConversationRequest 是不经过 Server(DB) 的包级函数，因此没有
// DB 也能跑完，并确认常量确实被装进响应（请求正文超限 →
// 413 + 中文文案）。
func TestDecodeConversationRequestTooLargeLocalized(t *testing.T) {
	// 超过 64KB 上限的合法 JSON 正文。MaxBytesReader 会在读取过程中返回超限。
	body := `{"title":"` + strings.Repeat("a", maxConversationRequestBytes+1024) + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/conversations", strings.NewReader(body))
	rec := httptest.NewRecorder()
	var dst struct {
		Title string `json:"title"`
	}
	if decodeConversationRequest(rec, req, &dst) {
		t.Fatal("正文已超出上限，但 decodeConversationRequest 仍返回了 true")
	}
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("状态码 = %d, 期望 = %d", rec.Code, http.StatusRequestEntityTooLarge)
	}
	var resp struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("响应 JSON 解析失败：%v (正文=%q)", err, rec.Body.String())
	}
	if resp.Error != convErrRequestTooLarge {
		t.Fatalf("响应 error = %q, 期望 = %q", resp.Error, convErrRequestTooLarge)
	}
	assertChineseError(t, "decode.too_large.response", resp.Error)
}

// TestConversationDefaultTitlesLocalized 断言对话默认标题的两个常量（F8）都是简体中文
// 且彼此可区分。convDefaultTitle 既是创建时的默认值，又是自动标题分支的哨兵，一旦把它
// 改回韩文，用户就会在对话列表与删除弹窗里看到韩文标题。
func TestConversationDefaultTitlesLocalized(t *testing.T) {
	assertChineseError(t, "default_title", convDefaultTitle)
	assertChineseError(t, "attachment_title", convAttachmentTitle)
	if convDefaultTitle == convAttachmentTitle {
		t.Fatal("默认标题和附件默认标题不能相同")
	}
}

// TestIsDefaultConversationTitle 钉住自动标题分支的判定：创建默认值（convDefaultTitle）
// 与空标题属于自动标题对象，用户自己起的标题则不是。创建默认值与哨兵共用同一个常量，
// 这样可防止两者错位导致自动标题不再生效的回归。
func TestIsDefaultConversationTitle(t *testing.T) {
	if !isDefaultConversationTitle("") {
		t.Fatal("空标题应当属于自动标题的覆盖对象")
	}
	if !isDefaultConversationTitle(convDefaultTitle) {
		t.Fatalf("生成默认值 %q 应当属于自动标题的覆盖对象", convDefaultTitle)
	}
	if isDefaultConversationTitle("用户起的标题") {
		t.Fatal("用户起的标题不应属于自动标题的覆盖对象")
	}
}

// TestConversationRetestReasonsLocalized 断言复验终结原因的三个常量（F9）是简体中文。
// 这些值会写入 finding_retests.error 字段，并作为复验面板的 item.error 展示，一旦改回
// 韩文，用户就会在面板里看到韩文原因。
func TestConversationRetestReasonsLocalized(t *testing.T) {
	assertChineseError(t, "retest_failed_to_start", convRetestFailedToStart)
	assertChineseError(t, "retest_status_read_failed", convRetestStatusReadFailed)
	assertChineseError(t, "retest_stopped_or_closed", convRetestStoppedOrClosed)
}

// TestTranscriptErrorSummaryLocalized 钉住活动转写的错误包装（F9）。聊天轮次
// （无标签）与任务主智能体（标签“主智能体”）两种调用都产出相同的“(…错误: …)”形式，
// 内部 err 原文保持不变，且包装里不得出现韩文与全角符号。
func TestTranscriptErrorSummaryLocalized(t *testing.T) {
	cases := []struct {
		name  string
		label string
		errm  string
		want  string
	}{
		{"chat_turn", "", "connection reset", "(错误: connection reset)"},
		{"main_agent", "主智能体", "connection reset", "(主智能体 错误: connection reset)"},
	}
	for _, c := range cases {
		got := transcriptErrorSummary(c.label, c.errm)
		if got != c.want {
			t.Fatalf("%s: transcriptErrorSummary = %q, 期望 = %q", c.name, got, c.want)
		}
		if !strings.Contains(got, c.errm) {
			t.Fatalf("%s: err 原文未被保留：%q", c.name, got)
		}
		// 确认括号与冒号仍是 ASCII（全角符号会破坏转写的渲染与检索）。
		if strings.ContainsAny(got, "（）：") {
			t.Fatalf("%s: 仍残留全角符号：%q", c.name, got)
		}
		// 包装标签必须含汉字且不含韩文（err 原文不是检查对象，保持 ASCII）。
		// 没有标签的聊天轮次也必须包含「错误」这类中文。
		wrapper := strings.ReplaceAll(got, c.errm, "")
		assertChineseError(t, c.name+".wrapper", wrapper)
	}
}
