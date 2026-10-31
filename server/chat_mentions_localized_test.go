package server

import (
	"fmt"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// TestChatMentionErrorsLocalized 守护 chat_mentions.go 的汉化：每条用户可见的
// @引用 错误响应都必须是简体中文（含汉字、不含韩文）。线上 token 标签
// （chatMentionPattern·chatMentionKinds）、Agent 输入快照的标题，以及喂给该快照的
// 截断标记按设计保留原文语言，因此这里刻意不检查它们。
func TestChatMentionErrorsLocalized(t *testing.T) {
	// Fixed user-facing literals.
	for label, msg := range map[string]string{
		"badID":       errChatMentionBadID,
		"tooMany":     errChatMentionTooMany,
		"badSearch":   errChatMentionBadSearch,
		"dataUnavail": errChatMentionDataUnavail,
		"tooLarge":    errChatMentionTooLarge,
	} {
		assertChineseError(t, label, msg)
	}

	// parseChatMentions — a non-positive id inside a valid wire token.
	if _, err := parseChatMentions("@[漏洞#0]"); err == nil {
		t.Fatal("错误的引用 ID 不应通过校验")
	} else {
		assertChineseError(t, "parse.badID", err.Error())
	}

	// parseChatMentions — exceeding the per-message mention cap (11 distinct ids).
	var b strings.Builder
	for i := 1; i <= 11; i++ {
		fmt.Fprintf(&b, "@[漏洞#%d] ", i)
	}
	if _, err := parseChatMentions(b.String()); err == nil {
		t.Fatal("引用数量超限不应通过校验")
	} else {
		assertChineseError(t, "parse.tooMany", err.Error())
	}

	// composeChatMentionMessage — a mention is present but the database is nil.
	if _, err := composeChatMentionMessage(nil, "@[漏洞#1]"); err == nil {
		t.Fatal("没有 DB 时引用解析不应通过校验")
	} else {
		assertChineseError(t, "compose.dataUnavail", err.Error())
	}

	// searchChatMentions — invalid kind and over-long query are rejected before
	// the database handle is touched, so &Server{} is enough to reach the branch.
	for _, q := range []string{"kind=unsupported", "q=" + url.QueryEscape(strings.Repeat("字", 201))} {
		w := httptest.NewRecorder()
		(&Server{}).searchChatMentions(w, httptest.NewRequest("GET", "/api/chat/mentions?"+q, nil))
		if w.Code != 400 {
			t.Fatalf("期望校验失败(400)，实际返回 %d (%s)", w.Code, q)
		}
		assertChineseError(t, "search."+q, decodeErrorField(t, w.Body.Bytes()))
	}

	// 显示标签映射与 UI 的引用类型保持一致；not-found 模板对已知类型必须解析出
	// 完整的中文消息。
	for _, kind := range []string{"finding", "asset", "company", "endpoint", "ip", "app", "root_domain", "subdomain", "service"} {
		if chatMentionKindLabel[kind] == "" {
			t.Fatalf("缺少类型标签：%s", kind)
		}
	}
	assertChineseError(t, "notFound", fmt.Sprintf(errChatMentionNotFoundFmt, chatMentionKindLabel["finding"], 7))
}
