package main

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"
	"unicode"

	"github.com/Autumn-27/artex/agent"
)

func TestShutdownContextPreservesNamedCause(t *testing.T) {
	signalCtx, signalCancel := context.WithCancel(context.Background())
	ctx, shutdown := shutdownContext(signalCtx)
	defer shutdown(nil)
	signalCancel()
	<-ctx.Done()
	if code, _, _, ok := agent.AbortReason(ctx); !ok || code != "shutdown" {
		t.Fatalf("code=%q ok=%v，期望 shutdown", code, ok)
	}
}

// TestPrintBannerLocalized 检查打到 stdout 的启动横幅是简体中文、且不残留韩文。
// 横幅是用户运行 ARTEX 时看到的第一屏内容（backlog F11）。run() 里的
// "[config] ..." 日志行刻意不在范围内——那是日志，汉化方针把它排在最低优先级（Z2）。
func TestPrintBannerLocalized(t *testing.T) {
	out := captureStdout(t, func() { printBanner(":8787") })
	t.Logf("横幅示例：\n%s", out)

	hasHangul := strings.ContainsFunc(out, func(r rune) bool {
		return unicode.Is(unicode.Hangul, r)
	})
	if hasHangul {
		t.Fatalf("横幅中残留韩文：%q", out)
	}

	for _, want := range []string{"自主渗透测试系统", "版本", "监听", ":8787"} {
		if !strings.Contains(out, want) {
			t.Errorf("横幅中缺少 %q：%q", want, out)
		}
	}

	hasHan := strings.ContainsFunc(out, func(r rune) bool {
		return unicode.Is(unicode.Han, r)
	})
	if !hasHan {
		t.Errorf("横幅中没有中文汉字：%q", out)
	}
}

// captureStdout 在 fn 执行期间重定向 os.Stdout，并返回 fn 写入的内容。
// printBanner 用的是 fmt.Print*，在调用时才解析 os.Stdout，所以在这里替换即可捕获横幅。
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = w

	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()

	fn()
	_ = w.Close()
	os.Stdout = orig
	return <-done
}
