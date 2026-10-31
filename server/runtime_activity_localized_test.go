package server

import (
	"fmt"
	"strings"
	"testing"
)

// 回归防护测试：守护 engine.go、engine_timeout.go、goals.go 中“纯显示”用途的运行期活动
// 摘要常量为简体中文。这些摘要会出现在看板转写（activity transcript）里，但不带 node_id
// （以任务为单位），因此不会被任何智能体上下文回读——回读路径（planner.workerOutput 与
// get_worker_output 工具）只挑 intent 范围（node_id）内的 'result'/'text' 活动。所以汉化
// 它们不触碰大脑输入（BRIEF 边界 #1）。反之，会被回读进大脑的摘要（种子意图摘要
// `完成任务目标…`、默认描述 `未命名任务` 等）保留原文，不属于本测试对象。中文判定复用
// assertChineseError（含汉字、不含韩文，intercept_archive_localized_test.go）。[[G132]]
func TestRuntimeActivitySummariesLocalized(t *testing.T) {
	// 不带格式参数的固定摘要：必须原样是中文。
	plain := map[string]string{
		"goalless_task_done": goallessTaskDoneSummary,
		"goal_breakdown_r0":  goalBreakdownRound0Summary,
		"queued_no_llm":      queuedNoLLMSummary,
		"queued_fifo":        queuedFIFOSummary,
	}
	for label, msg := range plain {
		assertChineseError(t, label, msg)
	}

	// %d 格式字符串：确认格式参数仍在、格式化结果也是中文，且数字真正被体现出来。
	fmtCases := map[string]string{
		"planner_round":       plannerRoundSummaryFmt,
		"timeout_final_round": timeoutFinalRoundSummaryFmt,
		"queued_conc_limit":   queuedConcurrencyLimitSummaryFmt,
	}
	for label, f := range fmtCases {
		if !strings.Contains(f, "%d") {
			t.Fatalf("%s: 格式参数 %%d 丢失了：%q", label, f)
		}
		got := fmt.Sprintf(f, 7)
		assertChineseError(t, label, got)
		if !strings.Contains(got, "7") {
			t.Fatalf("%s: 格式参数没有体现在结果中：%q", label, got)
		}
	}
}
