package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Autumn-27/artex/db"
)

// 本测试是随上游 ARTEX（0.3.15 系列）的 GET /api/intercept/judge/usage 一起移植到
// 本 fork 的。它验证：回退判定（judge）消耗的 token 累计与按日序列在真实 DB 中是否
// 聚合正确，以及没有 DB 时返回的是空序列而不是 500。
//
// 未配置数据库时与其他集成测试一样跳过（见 server/testmain_test.go）。
//
// These tests cover the endpoint restored from upstream ARTEX: judge token totals
// and the per-day series, plus the no-database graceful path.
const judgeFixtureModel = "judge-usage-test-fixture"

// judgeUsageRig opens the shared test database and registers fixture cleanup.
func judgeUsageRig(t *testing.T) (*Server, *db.DB) {
	t.Helper()
	dsn, _, err := db.DSN()
	if err != nil {
		t.Skipf("no database config (%v) — skipping", err)
	}
	d, err := db.Open(dsn)
	if err != nil {
		t.Skipf("postgres unavailable (%v) — skipping", err)
	}
	t.Cleanup(func() { _ = d.Close() })
		// llm_usage 表不由 schema.sql 创建，而是 Go 侧的 EnsureLLMUsageTable()。
		// 这里与 db 包既有测试（finding_traffic_archive_test.go 等）保持一致的建表方式。
	if err := d.EnsureLLMUsageTable(); err != nil {
		t.Fatalf("EnsureLLMUsageTable 失败：%v", err)
	}
	cleanup := func() {
		_, _ = d.Exec(`DELETE FROM llm_usage WHERE model = $1`, judgeFixtureModel)
	}
	cleanup()
	t.Cleanup(cleanup)
	return &Server{m: &Manager{pg: d}}, d
}

// callJudgeUsage invokes the handler directly (auth is covered by auth tests).
func callJudgeUsage(t *testing.T, s *Server, query string) (int, db.JudgeUsage) {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/intercept/judge/usage"+query, nil)
	s.interceptJudgeUsage(rec, req)
	var out db.JudgeUsage
	if rec.Code == http.StatusOK {
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatalf("响应 JSON 解析失败：%v (body=%s)", err, rec.Body.String())
		}
	}
	return rec.Code, out
}

// insertUsage writes one ledger row. tsShiftDays > 0 backdates it for the window test.
func insertUsage(t *testing.T, d *db.DB, worker string, in, out, cacheRead, cacheWrite int, tsShiftDays int) {
	t.Helper()
	_, err := d.Exec(`
INSERT INTO llm_usage(worker, model, profile_name, input_tokens, output_tokens, cache_read, cache_write, status, ts)
VALUES ($1,$2,'fixture',$3,$4,$5,$6,'ok', now() - ($7 * interval '1 day'))`,
		worker, judgeFixtureModel, in, out, cacheRead, cacheWrite, tsShiftDays)
	if err != nil {
		t.Fatalf("fixture 插入失败：%v", err)
	}
}

// TestInterceptJudgeUsageAggregates：累计只统计 worker='judge' 的行，按日序列里要出现
// 今天的桶，且非 judge 的行不得影响任何数值。
func TestInterceptJudgeUsageAggregates(t *testing.T) {
	s, d := judgeUsageRig(t)

	base, err := d.JudgeUsageStats(30)
	if err != nil {
		t.Fatalf("baseline JudgeUsageStats 失败：%v", err)
	}

	insertUsage(t, d, "judge", 100, 20, 7, 3, 0)
	insertUsage(t, d, "judge", 50, 10, 0, 0, 0)
		insertUsage(t, d, "worker", 999, 999, 999, 999, 0) // 不应被计入聚合

	code, got := callJudgeUsage(t, s, "")
	if code != http.StatusOK {
		t.Fatalf("status=%d want 200 (body=%s)", code, "")
	}
	if got.Calls-base.Calls != 2 {
		t.Errorf("calls delta=%d want 2", got.Calls-base.Calls)
	}
	if got.InputTokens-base.InputTokens != 150 {
		t.Errorf("input delta=%d want 150 (可能混入了非 judge 的行)", got.InputTokens-base.InputTokens)
	}
	if got.OutputTokens-base.OutputTokens != 30 {
		t.Errorf("output delta=%d want 30", got.OutputTokens-base.OutputTokens)
	}
	if got.CacheReadTokens-base.CacheReadTokens != 7 {
		t.Errorf("cache_read delta=%d want 7", got.CacheReadTokens-base.CacheReadTokens)
	}
	if got.CacheWriteTokens-base.CacheWriteTokens != 3 {
		t.Errorf("cache_write delta=%d want 3", got.CacheWriteTokens-base.CacheWriteTokens)
	}

	today := time.Now().UTC().Format("2006-01-02")
	var found bool
	for _, day := range got.Daily {
		if day.Date == today {
			found = true
			if day.Calls < 2 {
				t.Errorf("今日桶 calls=%d want >=2", day.Calls)
			}
			if day.InputTokens < 150 {
				t.Errorf("今日桶 input=%d want >=150", day.InputTokens)
			}
		}
	}
	if !found {
		t.Errorf("按日序列中没有今天(%s)的桶：%+v", today, got.Daily)
	}
}

// TestInterceptJudgeUsageDaysWindow：days 只收窄按日序列，累计仍是全时段。
func TestInterceptJudgeUsageDaysWindow(t *testing.T) {
	s, d := judgeUsageRig(t)

		insertUsage(t, d, "judge", 11, 1, 0, 0, 3) // 窗口外（3 天前）
		insertUsage(t, d, "judge", 22, 2, 0, 0, 0) // 窗口内

	_, recent := callJudgeUsage(t, s, "?days=1")
	cutoff := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")
	for _, day := range recent.Daily {
		if day.Date < cutoff {
			t.Errorf("days=1 却返回了窗口外的桶(%s) (cutoff=%s)", day.Date, cutoff)
		}
	}
	if recent.Calls < 2 {
		t.Errorf("累计 calls=%d want >=2 (应统计全部时间段，与窗口无关)", recent.Calls)
	}

		_, all := callJudgeUsage(t, s, "?days=0") // 0 会被纠正为默认 30 天
	if len(all.Daily) > 31 {
		t.Errorf("days=0 校正后的桶数量异常偏多：%d", len(all.Daily))
	}
}

// TestInterceptJudgeUsageWithoutDatabase：即使未连接数据库，也要返回 200 + 空序列，
// 以免设置页面崩坏（判定功能本身处于关闭状态）。
func TestInterceptJudgeUsageWithoutDatabase(t *testing.T) {
	s := &Server{m: &Manager{}}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/intercept/judge/usage", nil)
	s.interceptJudgeUsage(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d want 200", rec.Code)
	}
	var out db.JudgeUsage
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("响应 JSON 解析失败：%v", err)
	}
	if out.Calls != 0 || len(out.Daily) != 0 {
		t.Errorf("没有 DB 时应当是空序列：%+v", out)
	}
}
