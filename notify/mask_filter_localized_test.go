package notify

import (
	"errors"
	"strings"
	"testing"
)

// F4 ⑥：把 notify/mask.go·filter.go 的用户可见文案（保存/更新通知渠道配置时
// server/notify_api.go 以 writeErr(400) 原样返回的校验错误）锁成简体中文。
// channel.go 没有用户可见文案（全是注释），不在范围内。
// hasHan / hasHangul / assertChinese 定义在 notify_localized_test.go。

// TestFilterValidateLocalized 检查 min_severity 填错时的校验错误。
// 该错误经保存·更新路径（server/notify_api.go:264·369）以 400 暴露给用户。
func TestFilterValidateLocalized(t *testing.T) {
	err := Filter{MinSeverity: "hgih"}.Validate()
	if err == nil {
		t.Fatal("最小严重度无效却没有报错")
	}
	assertChinese(t, "Filter.Validate", err.Error())
	// low/medium/high/critical 是配置枚举，必须原样保留。
	for _, tok := range []string{"low", "medium", "high", "critical"} {
		if !strings.Contains(err.Error(), tok) {
			t.Errorf("严重度 token %q 丢失了：%q", tok, err.Error())
		}
	}
	// 合法值与空值都应通过。
	if err := (Filter{MinSeverity: "high"}).Validate(); err != nil {
		t.Errorf("high 是有效值却报错了：%v", err)
	}
	if err := (Filter{}).Validate(); err != nil {
		t.Errorf("空的最小严重度是有效值却报错了：%v", err)
	}
}

// TestPrepareConfigUpdateUnknownKindLocalized 检查用未注册渠道类型更新配置时的错误
// （server/notify_api.go:326 处以 400 返回）。
func TestPrepareConfigUpdateUnknownKindLocalized(t *testing.T) {
	_, err := PrepareConfigUpdate("definitely-not-a-channel", map[string]any{}, map[string]any{})
	if err == nil {
		t.Fatal("未注册的通道类型却没有报错")
	}
	assertChinese(t, "PrepareConfigUpdate unknown kind", err.Error())
}

// TestDestinationChangedErrorLocalized 走真实路径触发 ErrDestinationChangedWithoutCredentials：
// 只把 webhook 的目标地址（url）换成新值，而对凭据（headers）不做任何表态的 PATCH。
// 服务端会把它作为 400 返回。
func TestDestinationChangedErrorLocalized(t *testing.T) {
	_, err := PrepareConfigUpdate("webhook",
		map[string]any{
			"url":     "https://old.example.com/hook",
			"headers": map[string]any{"Authorization": "Bearer real-token"},
		},
		map[string]any{"url": "https://attacker.example.com/hook"},
	)
	if err == nil {
		t.Fatal("更改了目标地址却未表态凭据，也没有报错")
	}
	var de *ErrDestinationChangedWithoutCredentials
	if !errors.As(err, &de) {
		t.Fatalf("不是预期的错误类型：%T", err)
	}
	assertChinese(t, "ErrDestinationChangedWithoutCredentials", err.Error())
	// 变更的目标键与缺失的凭据键名是配置字段名，必须原样出现在消息里。
	if !strings.Contains(err.Error(), "url") || !strings.Contains(err.Error(), "headers") {
		t.Errorf("缺少字段键名：%q", err.Error())
	}
}

// TestRejectMaskedInContainersLocalized 走真实路径触发「结构体内部的掩码哨兵值被拒」：
// 在 headers（既是对象又是凭据字段）里塞入掩码哨兵值的 PATCH。
func TestRejectMaskedInContainersLocalized(t *testing.T) {
	_, err := PrepareConfigUpdate("webhook",
		map[string]any{},
		map[string]any{"headers": map[string]any{"Authorization": MaskedPrefix + ":…abc123"}},
	)
	if err == nil {
		t.Fatal("在结构体内部夹带了掩码哨兵却没有报错")
	}
	assertChinese(t, "rejectMaskedInContainers", err.Error())
	// 哨兵字面量（__masked__）必须出现在消息里，运营者才知道是哪个值有问题。
	if !strings.Contains(err.Error(), MaskedPrefix) {
		t.Errorf("消息中没有掩码哨兵：%q", err.Error())
	}
}
