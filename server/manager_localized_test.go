package server

import "testing"

// 回归防护测试：manager.go 中“工作智能体数量”配置校验的错误文案必须保持简体中文。
// 中文判定复用 F3a 提供的 assertChineseError（含汉字、不含韩文）。SetWorkers(n<=0) 分支在
// 触及 m.pg.SetSetting 之前就返回，因此在没有 DB 的机器上也能用空 Manager 真实跑一遍。
// 若有人把这些字面量改回韩文，本测试即失败。
//
// 调用图判定（见 errWorkersPositive 常量注释）：该文案只在设置 PUT 处理器
// （server.go:3512 → writeErr 400）中出现，属于纯用户可见，没有智能体工具（actool）路径，
// 因此不是大脑输入。同文件 384 行的启动修复错误属于运维启动日志，不在 F3b 范围内
// （保留原文），也不是本测试的对象。
func TestSetWorkersErrorLocalized(t *testing.T) {
	err := (&Manager{}).SetWorkers(0)
	if err == nil {
		t.Fatal("Worker 智能体数量为 0 时必须被拒绝")
	}
	assertChineseError(t, "set_workers_nonpositive", err.Error())
}
