package report

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
	"unicode"

	"github.com/Autumn-27/artex/db"
)

// containsHangul 探测字符串里是否混有韩文（한국어）音节。汉化后所有骨架文案都应是
// 简体中文，出现韩文就说明有文案没换干净或回退了。
func containsHangul(s string) (rune, bool) {
	for _, r := range s {
		if unicode.Is(unicode.Hangul, r) {
			return r, true
		}
	}
	return 0, false
}

func sampleFindingNode(t *testing.T) *db.Node {
	t.Helper()
	payload, err := json.Marshal(map[string]any{
		"vulnclass": "SQL Injection",
		"name":      "登录表单 SQL 注入",
		"severity":  "high",
		"summary":   "已确认登录参数存在基于错误的 SQL 注入。",
		"evidence":  map[string]string{"poc": "' OR '1'='1"},
	})
	if err != nil {
		t.Fatalf("序列化 payload 失败：%v", err)
	}
	return &db.Node{ID: 1, Kind: "finding", Payload: payload}
}

func sampleDBFinding() *db.DBFinding {
	return &db.DBFinding{
		ID:              123,
		VulnClass:       "SQL Injection",
		Name:            "登录表单 SQL 注入",
		Severity:        "high",
		Summary:         "已确认登录参数存在基于错误的 SQL 注入。",
		Evidence:        "' OR '1'='1",
		Status:          "confirmed",
		Report:          "详细分析正文。",
		CreatedAt:       time.Date(2026, 10, 4, 2, 30, 0, 0, time.UTC),
		EvidenceVersion: 2,
		TaskDescription: "演示目标渗透测试",
		TrafficBindings: []db.FindingTrafficBinding{
			{
				ID:   9,
				Role: "request",
				Note: "注入 payload 的发送点",
				Snapshot: db.TrafficEvidenceSnapshot{
					Method: "POST",
					URL:    "https://sandbox.local/login",
					Status: 200,
				},
			},
		},
	}
}

// TestReportMarkdownLocalized 验证探索图报告（report.Markdown）的骨架是简体中文，
// 并且完全不残留韩文。
func TestReportMarkdownLocalized(t *testing.T) {
	out := Markdown(Input{
		Title:       "演示任务",
		Goal:        "沙箱全面检查",
		GeneratedAt: time.Date(2026, 10, 4, 2, 30, 0, 0, time.UTC),
		AssetCounts: map[string]int{"host": 2, "url": 5},
		Findings:    []*db.Node{sampleFindingNode(t)},
	})
	if r, ok := containsHangul(out); ok {
		t.Fatalf("报告骨架中残留韩文 %q：\n%s", string(r), out)
	}
	for _, want := range []string{
		"# 渗透测试报告: 演示任务",
		"- **任务目标**: 沙箱全面检查",
		"- **生成时间**:",
		"## 概要",
		"- 已确认漏洞: **1** 条",
		"## 漏洞",
		"**PoC / 证据：**",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("报告骨架中缺少 %q", want)
		}
	}
}

// TestReportMarkdownEmptyLocalized 验证没有漏洞时的提示语是简体中文。
func TestReportMarkdownEmptyLocalized(t *testing.T) {
	out := Markdown(Input{Title: "空任务", GeneratedAt: time.Now()})
	if r, ok := containsHangul(out); ok {
		t.Fatalf("空报告中残留韩文 %q：\n%s", string(r), out)
	}
	if !strings.Contains(out, "_本次没有已确认的漏洞。_") {
		t.Errorf("空报告的提示语不是中文：\n%s", out)
	}
}

// TestFindingsMarkdownLocalized 验证漏洞摘要/详情 Markdown 导出的骨架与严重程度标签
// 是简体中文，且不残留韩文。
func TestFindingsMarkdownLocalized(t *testing.T) {
	f := sampleDBFinding()
	out := FindingsMarkdown([]*db.DBFinding{f}, time.Date(2026, 10, 4, 2, 30, 0, 0, time.UTC))
	if r, ok := containsHangul(out); ok {
		t.Fatalf("漏洞摘要报告中残留韩文 %q：\n%s", string(r), out)
	}
	for _, want := range []string{
		"# 漏洞摘要报告",
		"- **生成时间**:",
		"- **漏洞总数**: 1 条",
		"## 概要",
		"| 严重程度 | 数量 |",
		"| 严重 |", "| 高 |", "| 中 |", "| 低 |", // 严重程度标签（与 UI status.severity 一致）
		"## 漏洞详情",
		"- **类型**: SQL Injection",
		"- **状态**: confirmed",
		"- **所属任务**: 演示目标渗透测试",
		"- **发现时间**:",
		"**证据：**",
		"**详细报告：**",
		"## 相关流量证据",
		"证据版本：2，绑定数量：1。",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("摘要报告中缺少 %q", want)
		}
	}

	single := SingleFindingMarkdown(f, time.Date(2026, 10, 4, 2, 30, 0, 0, time.UTC))
	if r, ok := containsHangul(single); ok {
		t.Fatalf("单条报告中残留韩文 %q：\n%s", string(r), single)
	}
	for _, want := range []string{
		"- **严重程度**: high",
		"## 概要",
		"## 证据",
		"## 详细报告",
		"[请求消息](evidence/123/9/request.http)",
		"[响应消息](evidence/123/9/response.http)",
	} {
		if !strings.Contains(single, want) {
			t.Errorf("单条报告中缺少 %q", want)
		}
	}
}

// TestFindingsCSVLocalized 验证 CSV 导出表头是简体中文且不残留韩文。
// （只需 BOM 之后的表头行是中文即可，数据值保持原样。）
func TestFindingsCSVLocalized(t *testing.T) {
	raw := FindingsCSV([]*db.DBFinding{sampleDBFinding()})
	out := strings.TrimPrefix(string(raw), "\xEF\xBB\xBF")
	header := strings.SplitN(out, "\n", 2)[0]
	if r, ok := containsHangul(header); ok {
		t.Fatalf("CSV 表头中残留韩文 %q：%s", string(r), header)
	}
	for _, col := range []string{"ID", "名称", "类型", "严重程度", "状态", "所属任务", "发现时间", "概要", "流量证据数量", "流量证据 ID"} {
		if !strings.Contains(header, col) {
			t.Errorf("CSV 表头中缺少 %q 列：%s", col, header)
		}
	}
}
