package llmpool

import (
	"context"
	"errors"
	"strings"
	"testing"
	"unicode"

	"github.com/Autumn-27/norma/llm"
)

// hasHangul 探测 s 里是否含有韩文字符。
func hasHangul(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Hangul, r) {
			return true
		}
	}
	return false
}

// hasHanzi 探测 s 里是否含有 CJK 汉字。
func hasHanzi(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}

// assertSurfacedExhaustion 检查链路耗尽时回传给调用方的错误：它仍必须包装
// ErrExhausted 哨兵（调用方依赖 errors.Is）、读起来是简体中文（不含韩文），并以
// 本地化后的哨兵文案开头。failProv 的正文是 ASCII（"anthropic: status 402: nope"），
// 所以整条回传串都应该由中文骨架 + ASCII 明细组成。
func assertSurfacedExhaustion(t *testing.T, label string, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("%s: 链路耗尽却返回了 nil 错误", label)
	}
	if !errors.Is(err, ErrExhausted) {
		t.Fatalf("%s: 回传错误不再包装 ErrExhausted：%v", label, err)
	}
	msg := err.Error()
	if hasHangul(msg) {
		t.Errorf("%s: 回传错误里残留韩文：%q", label, msg)
	}
	if !hasHanzi(msg) {
		t.Errorf("%s: 回传错误里没有中文汉字：%q", label, msg)
	}
	if !strings.HasPrefix(msg, ErrExhausted.Error()) {
		t.Errorf("%s: 回传错误没有以本地化后的哨兵文案开头：%q", label, msg)
	}
}

// 链路耗尽的错误是面向用户的：整条 LLM 链路都失败的 Worker 会经 agent/capture.go
// 把它作为「结果」活动记录，展示在执行对话里。所以它的文案必须是简体中文，
// 同时它的身份（errors.Is）必须保留给按哨兵分支的调用方。
func TestExhaustedErrorLocalized(t *testing.T) {
	assertChineseErrText(t, "ErrExhausted", ErrExhausted.Error())

	// 两条回传路径都跑一遍，这样将来改动任一 wrap 点都会被逮到。
	streamChain := New([]*Member{
		member(1, "a", 10, failProv("a", 402)),
		member(2, "b", 5, failProv("b", 402)),
	}, NewRegistry(nil, nil))
	_, serr := drain(streamChain.Stream(context.Background(), llm.CompletionRequest{}))
	assertSurfacedExhaustion(t, "Stream", serr)

	completeChain := New([]*Member{
		member(1, "a", 10, failProv("a", 402)),
		member(2, "b", 5, failProv("b", 402)),
	}, NewRegistry(nil, nil))
	_, _, _, cerr := completeChain.Complete(context.Background(), llm.CompletionRequest{})
	assertSurfacedExhaustion(t, "Complete", cerr)
}

// assertChineseErrText 要求 s 含中文汉字且不含韩文。
func assertChineseErrText(t *testing.T, label, s string) {
	t.Helper()
	if !hasHanzi(s) {
		t.Errorf("%s：没有中文汉字：%q", label, s)
	}
	if hasHangul(s) {
		t.Errorf("%s：仍残留韩文：%q", label, s)
	}
}
