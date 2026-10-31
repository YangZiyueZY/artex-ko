package server

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/Autumn-27/artex/db"
)

// 回归防护测试：engine.go 的任务控制与意图介入错误中，“用户可见”文案必须保持简体中文。
// 中文判定复用 F3a 提供的 assertChineseError（含汉字、不含韩文）。
//
// 调用图判定（见 engine.go 常量块注释）：
//   - ControlWork 的四个错误（没有正在运行的 work、已有控制在进行中、等待收尾被取消、
//     等待收尾超时）→ applyIntentControl(task_control.go) → controlIntent(server.go:1236) →
//     writeErr 409。纯用户可见，不会触及智能体工具（actool）路径。
//   - runDetachedIntent 的两个错误（Worker 未就绪、恢复 CAS 冲突）→
//     sendWorkerMessage(intent_intervention.go:147) → writeErr。纯用户可见。
//   - 反之，SteerWork、KillWork（steer_work、kill_work 工具的 actool.Errorf）与
//     transitionIntentState（内部状态转移日志）里的中文是大脑输入或日志，予以保留，
//     不是本测试的对象。

// TestControlWorkNoRunningWorkErrorLocalized 在没有 DB 与引擎的情况下直接驱动
// ControlWork，验证没有运行中的 work 时确实返回中文错误。run==nil 分支只读 e.work 映射，
// 因此 NewEngine(nil) 就能跑到底。由于用 %w 包裹了 errWorkControlConflict 哨兵，
// 这里同时确认 errors.Is 关系未被破坏；若有人把这些字面量改回韩文，本测试即失败。
func TestControlWorkNoRunningWorkErrorLocalized(t *testing.T) {
	e := NewEngine(nil)
	err := e.ControlWork(context.Background(), 42, "pause")
	if err == nil {
		t.Fatal("没有正在执行的 work，但 ControlWork 返回了 nil")
	}
	if !errors.Is(err, errWorkControlConflict) {
		t.Fatalf("errors.Is(err, errWorkControlConflict) = false, err=%v", err)
	}
	assertChineseError(t, "control_work_no_running", err.Error())
}

// TestEngineUserFacingErrorsLocalized 钉住其余用户可见格式字符串——它们位于引擎/DB 闸门
// 或协程时序之后，难以端到端驱动（与 finding_retests、finding_traffic 的闸门后路径
// 同一做法）。格式字符串先用代表性实参填充再判定。
func TestEngineUserFacingErrorsLocalized(t *testing.T) {
	for _, c := range []struct {
		label, msg string
	}{
		{"work_control_busy", fmt.Errorf(errWorkControlBusyFmt, errWorkControlConflict, int64(42), "pause").Error()},
		{"work_control_wait", fmt.Errorf(errWorkControlWaitFmt, int64(42), "pause", context.DeadlineExceeded).Error()},
		{"detached_worker_not_ready", errDetachedWorkerNotReady},
		{"detached_state_conflict", fmt.Errorf(errIntentCtrlStateConflictFmt, db.ErrIntentStateConflict).Error()},
	} {
		assertChineseError(t, c.label, c.msg)
	}
}
