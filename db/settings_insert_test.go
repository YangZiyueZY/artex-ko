package db

import (
	"fmt"
	"testing"
	"time"
)

// InsertSettingIfAbsent 是 auth.password_hash 的安全网：调用方的 GetSetting 检查可能
// 因数据库错误而失效，也可能被并发请求抢先（bcrypt 要花几十毫秒）。所以「只初始化
// 一次」的保证必须落在主键约束上，而不是 if 判断里。
func TestInsertSettingIfAbsentDoesNotOverwrite(t *testing.T) {
	d, err := Open(testDSN(t))
	if err != nil {
		t.Skipf("postgres unavailable (%v) — skipping", err)
	}
	defer d.Close()

	// 用测试专用键，不碰开发数据库里真正的 auth.password_hash。
	key := fmt.Sprintf("test.insert_if_absent.%d", time.Now().UnixNano())
	defer func() { _, _ = d.Exec(`DELETE FROM settings WHERE key=$1`, key) }()

	inserted, err := d.InsertSettingIfAbsent(key, "first")
	if err != nil {
		t.Fatal(err)
	}
	if !inserted {
		t.Fatal("首次写入应为 inserted=true")
	}

	inserted, err = d.InsertSettingIfAbsent(key, "second")
	if err != nil {
		t.Fatal(err)
	}
	if inserted {
		t.Fatal("key 已存在时 inserted 应为 false")
	}

	got, ok, err := d.GetSetting(key)
	if err != nil || !ok {
		t.Fatalf("GetSetting: ok=%v err=%v", ok, err)
	}
	if got != "first" {
		t.Fatalf("值被覆盖成了 %q。应保持 %q", got, "first")
	}
}
