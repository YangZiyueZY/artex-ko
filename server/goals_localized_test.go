package server

import "testing"

// 回归防护测试：goals.go 中任务恢复（admitPausedTask）前置条件校验的错误文案必须保持
// 简体中文。中文判定复用 F3a 提供的 assertChineseError（含汉字、不含韩文）。返回这两条
// 文案的 admitTaskWhen(requirePaused=true) 路径要在 s.concMu 加锁后经过 s.m.Task()、
// s.engine.IsDeleting()、beginTaskOperation() 引擎闸门，在没有 DB 与引擎的机器上跑不到底，
// 因此直接钉住常量本身（与 finding_retests·finding_traffic 的闸门后路径同一做法）。
// 若有人把这些字面量改回韩文，本测试即失败。
//
// 调用图判定（见 goals.go 常量块注释）：这两条文案只会出现在单条（server.go:1194 →
// writeErr 409）与批量（task_control.go:298 → items[].error）任务控制响应中，属于纯用户
// 可见。编排器路径（orchestration.go:343）固定 action="pause"，触及不到恢复校验，因此不是大脑输入。
func TestGoalResumeErrorsLocalized(t *testing.T) {
	for _, c := range []struct {
		label, msg string
	}{
		{"resume_terminal", errGoalResumeTerminal},
		{"resume_not_paused", errGoalResumeNotPaused},
	} {
		assertChineseError(t, c.label, c.msg)
	}
}
