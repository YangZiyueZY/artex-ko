package server

import "testing"

// 回归防护测试：finding_traffic.go 的用户可见 HTTP 错误响应文案必须保持简体中文。
// 中文判定复用 F3a 提供的 assertChineseError（含汉字、不含韩文）。返回这四条文案的路径
// （findingTrafficAccess·bindFindingTraffic·editFindingTraffic）都在 s.m.pg.GetFinding(DB)
// 闸门之后，在没有 DB 的机器上跑不到底，因此直接钉住常量本身
// （与 goals_api·notify_api 的 DB 闸门路径同一做法）。若有人把这些字面量改回韩文，
// 本测试即失败。
//
// readEvidencePreview 的 errors.New（offset/length 校验）、二进制正文占位符，以及
// roTool("get_finding_traffic") 的工具说明，是会被回读进智能体大脑的输入，
// 刻意保留中文原文，不在本测试的断言范围内。
func TestFindingTrafficErrorsLocalized(t *testing.T) {
	for _, c := range []struct {
		label, msg string
	}{
		{"inherited_readonly", errFindingTrafficInheritedReadonly},
		{"select_required", errFindingTrafficSelectRequired},
		{"version_required", errFindingTrafficVersionRequired},
		{"binding_ids_required", errFindingTrafficBindingIDsRequired},
	} {
		assertChineseError(t, c.label, c.msg)
	}
}
