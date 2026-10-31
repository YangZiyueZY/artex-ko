package server

import (
	"context"
	"testing"

	"github.com/Autumn-27/artex/db"
)

// 回归防护测试：mcpdiscover.go 的 connectMCP 返回的传输配置校验错误文案必须保持简体中文。
// 四条错误路径（stdio 缺少命令、http 缺少 URL、sse 缺少 URL、未知传输方式）都在真正
// dial 服务器之前返回，因此无需 DB 与网络即可直接调用 connectMCP 校验返回文案。
// 中文判定复用 F3a 提供的 assertChineseError（含汉字、不含韩文）。
// 传输方式枚举（stdio/http/sse）与 URL 是线上标识符，保持 ASCII，不会被判定拦住。
// 若有人把这些文案改回韩文，常量钉住与真实路径会一起失败。
func TestConnectMCPErrorsLocalized(t *testing.T) {
	// 常量钉住
	for _, c := range []struct{ label, msg string }{
		{"stdio_no_command", errMCPStdioNoCommand},
		{"http_no_url", errMCPHTTPNoURL},
		{"sse_no_url", errMCPSSENoURL},
		{"unknown_transport", errMCPUnknownTransportFmt},
	} {
		assertChineseError(t, c.label, c.msg)
	}

	// 真实路径：四个拒绝分支都在 dial 之前返回。
	for _, c := range []struct {
		label string
		srv   *db.MCPServer
	}{
		{"stdio_missing_command", &db.MCPServer{Transport: "stdio", Command: ""}},
		{"http_missing_url", &db.MCPServer{Transport: "http", URL: ""}},
		{"sse_missing_url", &db.MCPServer{Transport: "sse", URL: ""}},
		{"unknown_transport_path", &db.MCPServer{Transport: "websocket"}},
	} {
		_, err := connectMCP(context.Background(), c.srv)
		if err == nil {
			t.Fatalf("%s: 未报错(已通过校验)", c.label)
		}
		assertChineseError(t, c.label, err.Error())
	}
}
