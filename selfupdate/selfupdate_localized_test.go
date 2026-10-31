package selfupdate

// 本文件钉住自更新路径上面向用户的进度·错误文案是简体中文（回归测试 F20）。
// update.go 的 updateHub 会把 Stage 的进度回调消息作为 SSE progress.message 下发，
// 并把 Stage/FetchLatest/Rollback 返回的错误原样送进 progress.error、writeErr 正文与
// boot_notice，最终由前端（update-card）展示；所以这些文案一旦回退成别的语言，
// 用户界面就会语言混杂。日志（log.Printf）与注释不在本测试范围，这里只断言
// 「含中文汉字且不含韩文」。

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func hasHangul(s string) bool {
	for _, r := range s {
		if r >= 0xAC00 && r <= 0xD7A3 {
			return true
		}
	}
	return false
}

func hasHanzi(s string) bool {
	for _, r := range s {
		if r >= 0x4E00 && r <= 0x9FFF {
			return true
		}
	}
	return false
}

// assertChineseError 断言确实出错了，且错误文案含中文汉字、不含韩文。
// 用 %w 包裹的下层 stdlib 错误里的英文是允许的（只禁韩文）。
func assertChineseError(t *testing.T, label string, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("%s: 期望出错，但结果为 nil", label)
	}
	msg := err.Error()
	if !hasHanzi(msg) {
		t.Errorf("%s：没有中文汉字：%q", label, msg)
	}
	if hasHangul(msg) {
		t.Errorf("%s：仍残留韩文：%q", label, msg)
	}
}

// rtFunc 用来给 http.Client 注入假响应，不会真的访问 GitHub。
type rtFunc func(*http.Request) (*http.Response, error)

func (f rtFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func stubClient(status int, body string) *http.Client {
	return &http.Client{Transport: rtFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: status,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     make(http.Header),
		}, nil
	})}
}

// TestCheckURLErrorsLocalized：既有的 TestCheckURLRejectsNonGitHub 只看是否拒绝，
// 不看文案语言。这里检查两条拒绝路径的错误都是简体中文。
func TestCheckURLErrorsLocalized(t *testing.T) {
	nonHTTPS := checkURL(mustParse(t, "http://api.github.com/x"))
	assertChineseError(t, "non-https", nonHTTPS)
	if !strings.Contains(nonHTTPS.Error(), "HTTPS") {
		t.Errorf("non-https 错误中缺少 HTTPS 字样: %q", nonHTTPS.Error())
	}

	nonGitHub := checkURL(mustParse(t, "https://evil.example.com/x"))
	assertChineseError(t, "non-github-host", nonGitHub)
	if !strings.Contains(nonGitHub.Error(), "GitHub") {
		t.Errorf("non-github 错误中缺少 GitHub 字样: %q", nonGitHub.Error())
	}
}

// TestFetchLatestErrorsLocalized：版本检查路径的五种错误（请求限流·未发布·其他
// 状态·解析失败·tag 缺失）都是简体中文。用假 transport，不访问网络。
func TestFetchLatestErrorsLocalized(t *testing.T) {
	ctx := context.Background()

	_, errRate := FetchLatest(ctx, stubClient(http.StatusForbidden, ""))
	assertChineseError(t, "rate-limited", errRate)
	if !strings.Contains(errRate.Error(), "60") {
		t.Errorf("请求限制错误中缺少 60 字样: %q", errRate.Error())
	}

	_, errNF := FetchLatest(ctx, stubClient(http.StatusNotFound, ""))
	assertChineseError(t, "not-found", errNF)

	_, errStatus := FetchLatest(ctx, stubClient(http.StatusInternalServerError, ""))
	assertChineseError(t, "bad-status", errStatus)
	if !strings.Contains(errStatus.Error(), "GitHub") {
		t.Errorf("状态错误中缺少 GitHub 字样: %q", errStatus.Error())
	}

	_, errJSON := FetchLatest(ctx, stubClient(http.StatusOK, "{not json"))
	assertChineseError(t, "bad-json", errJSON)

	_, errTag := FetchLatest(ctx, stubClient(http.StatusOK, "{}"))
	assertChineseError(t, "missing-tag", errTag)
	if !strings.Contains(errTag.Error(), "tag") {
		t.Errorf("缺少 tag 的错误中缺少 tag 字样: %q", errTag.Error())
	}
}

// TestStageMissingAssetLocalized：发布版本里没有当前平台的包时，Stage 会在下载前
// 就以中文错误停下（FindAsset 失败 → 不发起网络访问）。视运行环境不同，checkWritable
// 可能先拦下来，但那个错误也是中文，所以这里只断言语言。
func TestStageMissingAssetLocalized(t *testing.T) {
	rel := &Release{TagName: "v9.9.9"} // Assets 为空
	err := Stage(context.Background(), &http.Client{}, rel, "0.0.1", nil)
	assertChineseError(t, "stage-missing-asset", err)
}
