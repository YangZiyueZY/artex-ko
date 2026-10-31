package server

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/Autumn-27/artex/db"
	"github.com/Autumn-27/norma/harness"
)

// 模型错误码重试的「延迟重排」段（第 ⑤ 层原地重跑之上的兜底，参数见 server/llmretry.go）。
//
// 原地重跑只覆盖秒级抖动；供应商连续 5xx、限流冷却、额度重置要等几分钟到几十分钟。
// 这些情况下把意图判成 blocked 等于中断任务（用户只能手动重跑），所以这里改成：
// 把意图退回 open，并加一段「领取栅栏」（栅栏到期前 claimNext 跳过它），到期后由
// worker 正常重新领取。planner/worker 循环全程不停，用户看到的是「正在自动重试」。
//
// 预算：每条意图最多重排 modelErrorRequeueCount 次（可由全局策略的 requeue 层覆盖），
// 用尽仍失败才落 blocked 交给规划者/用户。计数存在进程内，重启后清零重来。

// modelRetryState 记录一条意图已经用掉几次延迟重排。
type modelRetryState struct {
	mu       sync.Mutex
	taskID   string
	requeues int
}

// claimHoldEntry 是「在 until 之前不要重新领取这条意图」的栅栏。
type claimHoldEntry struct {
	taskID string
	until  time.Time
}

// modelRetryEntry 取出（或创建）某条意图的重排状态。
func (e *Engine) modelRetryEntry(intentID int64, taskID string) *modelRetryState {
	if v, ok := e.retryState.Load(intentID); ok {
		return v.(*modelRetryState)
	}
	v, _ := e.retryState.LoadOrStore(intentID, &modelRetryState{taskID: taskID})
	return v.(*modelRetryState)
}

// clearModelRetryState 清掉一条意图的重排计数与领取栅栏（成功、判死或被控制时调用）。
func (e *Engine) clearModelRetryState(intentID int64) {
	e.retryState.Delete(intentID)
	e.claimHold.Delete(intentID)
}

// forgetTaskModelRetry 在任务 teardown 时清掉属于该任务的全部重排状态与栅栏。
func (e *Engine) forgetTaskModelRetry(taskID string) {
	e.retryState.Range(func(k, v any) bool {
		if st, ok := v.(*modelRetryState); ok && st.taskID == taskID {
			e.retryState.Delete(k)
		}
		return true
	})
	e.claimHold.Range(func(k, v any) bool {
		if h, ok := v.(claimHoldEntry); ok && h.taskID == taskID {
			e.claimHold.Delete(k)
		}
		return true
	})
}

// holdIntentClaim 给一条刚退回 open 的意图加领取栅栏。
func (e *Engine) holdIntentClaim(intentID int64, taskID string, d time.Duration) {
	if d <= 0 {
		e.claimHold.Delete(intentID)
		return
	}
	e.claimHold.Store(intentID, claimHoldEntry{taskID: taskID, until: time.Now().Add(d)})
}

// claimAllowed 判断一条意图当前能否被领取：栅栏到期即放行并清掉栅栏。
func (e *Engine) claimAllowed(intentID int64) bool {
	v, ok := e.claimHold.Load(intentID)
	if !ok {
		return true
	}
	h := v.(claimHoldEntry)
	if time.Now().Before(h.until) {
		return false
	}
	e.claimHold.Delete(intentID)
	return true
}

// maybeRequeueModelError 在意图即将被判死之前做最后一件事：如果这次失败属于
// 「外部可恢复」（网络抖动/限流/供应商 5xx/额度耗尽），且预算还没用完、任务没有
// 被暂停/删除/收尾，就把意图退回 open 并加延迟栅栏，返回 true 表示本次收场已被
// 接管（不要落 blocked）。
func (e *Engine) maybeRequeueModelError(t *Task, intent *db.Node, name string, err error, reason harness.TerminalReason, killed bool, action string) bool {
	if err == nil || reason != harness.ReasonModelError || killed || action != "" {
		return false
	}
	if e.IsPaused(t.ID) || e.IsDeleting(t.ID) || e.isSettling(t.ID) {
		return false
	}
	if isTerminalStatus(e.m.TaskStatus(t.ID)) {
		return false
	}
	info := classifyLLMError(err)
	if !info.Requeue {
		return false
	}
	attempts, delay := e.modelErrorRequeuePolicy()
	if attempts <= 0 || delay == nil {
		return false
	}
	state := e.modelRetryEntry(intent.ID, t.ID)
	state.mu.Lock()
	if state.requeues >= attempts {
		state.mu.Unlock()
		return false
	}
	state.requeues++
	used := state.requeues
	state.mu.Unlock()

	wait := delay(used)
	if wait <= 0 {
		return false
	}
	if err := transitionIntentState(t.Store, intent.ID, "running", "open"); err != nil {
		log.Printf("[worker %s] task %s 意图 #%d 延迟重排状态回退失败: %v", name, t.ID, intent.ID, err)
		return false
	}
	e.holdIntentClaim(intent.ID, t.ID, wait)

	iid := intent.ID
	summary := fmt.Sprintf(modelErrorRequeueSummaryFmt, info.Summary, humanWait(wait), used, attempts)
	e.emitActivity(t, db.Activity{
		NodeID:  &iid,
		Worker:  "system",
		Kind:    "text",
		IsError: true,
		Summary: summary,
		Detail:  fmt.Sprintf(modelErrorRequeueDetailFmt, info.Summary, truncErr(err.Error(), 1200), humanWait(wait), used, attempts),
	})
	log.Printf("[worker %s] task %s 意图 #%d 模型错误可恢复(%s)，%s 后自动重试(%d/%d)：%v",
		name, t.ID, intent.ID, info.Summary, wait, used, attempts, err)
	return true
}

// 写进活动流（用户可见）的重排文案。
const (
	modelErrorRequeueSummaryFmt = "模型调用失败（%s），%s后自动重试（第 %d/%d 次）；任务保持运行，可继续等待或手动暂停"
	modelErrorRequeueDetailFmt  = "错误类别：%s\n\n原始错误：\n%s\n\n处理：%s后自动重试（第 %d/%d 次）。重试不会重复执行已完成的工具调用；若模型配置本身有误（密钥、模型名、余额），请到「系统配置 → 模型」修正，或手动暂停任务。"
)

// humanWait 把等待时长写成用户好读的中文（整分钟 / 分+秒 / 秒）。
func humanWait(d time.Duration) string {
	switch {
	case d >= time.Minute && d%time.Minute == 0:
		return fmt.Sprintf("%d 分钟", int(d/time.Minute))
	case d > time.Minute:
		return fmt.Sprintf("%d 分 %d 秒", int(d/time.Minute), int((d%time.Minute)/time.Second))
	default:
		return fmt.Sprintf("%d 秒", int(d/time.Second))
	}
}

// truncErr 截断过长的错误文本，避免活动详情被整页 HTML 错误页撑爆。
func truncErr(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "…"
}
