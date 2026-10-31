package server

import (
	"testing"
	"time"

	"github.com/Autumn-27/artex/db"
)

// 第 ⑤ 层（原地重跑）的默认值：5 次、3s 起指数、单次封顶 60s。
func TestResolveModelErrorRetryDefaults(t *testing.T) {
	retries, base, cap := resolveModelErrorRetry(db.LLMRetryPolicy{})
	if retries != 5 || base != 3*time.Second || cap != time.Minute {
		t.Fatalf("retry policy = %d/%v/%v, want 5/3s/60s", retries, base, cap)
	}
}

// 用户配了间隔(>0)就改用固定间隔；次数 0=内置默认、负数=不重跑。
func TestResolveModelErrorRetryOverrides(t *testing.T) {
	retries, base, cap := resolveModelErrorRetry(db.LLMRetryPolicy{
		Intent: db.RetryRule{Attempts: 2, IntervalMS: 1500},
	})
	if retries != 2 || base != 1500*time.Millisecond || cap != 1500*time.Millisecond {
		t.Fatalf("pinned ladder = %d/%v/%v, want 2/1.5s/1.5s", retries, base, cap)
	}
	if retries, _, _ := resolveModelErrorRetry(db.LLMRetryPolicy{Intent: db.RetryRule{Attempts: -1}}); retries != 0 {
		t.Fatalf("retries=%d, want 0 (disabled)", retries)
	}
}

// 第 ⑥ 层（延迟重排）的默认梯度：1min→2min→4min→8min，封顶 10min。
func TestResolveModelErrorRequeueDefaults(t *testing.T) {
	attempts, backoff := resolveModelErrorRequeue(db.LLMRetryPolicy{})
	if attempts != 4 {
		t.Fatalf("attempts=%d, want 4", attempts)
	}
	want := []time.Duration{time.Minute, 2 * time.Minute, 4 * time.Minute, 8 * time.Minute, 10 * time.Minute}
	for i, w := range want {
		if got := backoff(i + 1); got != w {
			t.Fatalf("backoff(%d)=%v, want %v", i+1, got, w)
		}
	}
}

// 重排层同样支持固定间隔与 -1 关闭。
func TestResolveModelErrorRequeueOverrides(t *testing.T) {
	attempts, backoff := resolveModelErrorRequeue(db.LLMRetryPolicy{
		Requeue: db.RetryRule{Attempts: 2, IntervalMS: 30_000},
	})
	if attempts != 2 || backoff(1) != 30*time.Second || backoff(9) != 30*time.Second {
		t.Fatalf("pinned requeue = %d/%v, want 2/30s fixed", attempts, backoff(1))
	}
	if attempts, backoff := resolveModelErrorRequeue(db.LLMRetryPolicy{Requeue: db.RetryRule{Attempts: -1}}); attempts != 0 || backoff(1) != 0 {
		t.Fatalf("disabled requeue = %d/%v, want 0", attempts, backoff(1))
	}
}

func TestExpBackoffLadder(t *testing.T) {
	base, cap := 3*time.Second, time.Minute
	want := map[int]time.Duration{
		0: 3 * time.Second, // n<1 按第 1 次
		1: 3 * time.Second,
		2: 6 * time.Second,
		3: 12 * time.Second,
		4: 24 * time.Second,
		5: 48 * time.Second,
		6: time.Minute, // 96s 封顶 60s
		9: time.Minute,
	}
	for n, w := range want {
		if got := expBackoff(n, base, cap); got != w {
			t.Errorf("expBackoff(%d)=%v, want %v", n, got, w)
		}
	}
}

// 领取栅栏：到期前禁止领取，到期后放行；非正延迟视为立即到期。
func TestClaimHoldGatesAndExpires(t *testing.T) {
	e := &Engine{}
	if !e.claimAllowed(1) {
		t.Fatal("no hold must be claimable")
	}
	e.holdIntentClaim(1, "t1", time.Minute)
	if e.claimAllowed(1) {
		t.Fatal("an active hold must block claiming")
	}
	e.holdIntentClaim(1, "t1", -time.Second)
	if !e.claimAllowed(1) {
		t.Fatal("an expired hold must allow claiming")
	}
}

// 任务 teardown 只清掉属于该任务的状态，别动别的任务。
func TestForgetTaskModelRetry(t *testing.T) {
	e := &Engine{}
	st := e.modelRetryEntry(1, "t1")
	st.requeues = 2
	e.modelRetryEntry(2, "t2")
	e.holdIntentClaim(1, "t1", time.Minute)
	e.holdIntentClaim(2, "t2", time.Minute)

	e.forgetTaskModelRetry("t1")
	if _, ok := e.retryState.Load(int64(1)); ok {
		t.Fatal("t1 retry state must be cleared")
	}
	if _, ok := e.claimHold.Load(int64(1)); ok {
		t.Fatal("t1 claim hold must be cleared")
	}
	if _, ok := e.retryState.Load(int64(2)); !ok {
		t.Fatal("t2 retry state must survive")
	}
	if _, ok := e.claimHold.Load(int64(2)); !ok {
		t.Fatal("t2 claim hold must survive")
	}
}

// 活动流里的等待时长必须是好读的中文。
func TestHumanWait(t *testing.T) {
	cases := map[time.Duration]string{
		time.Minute:            "1 分钟",
		2 * time.Minute:        "2 分钟",
		90 * time.Second:       "1 分 30 秒",
		5 * time.Second:        "5 秒",
		10 * time.Minute:       "10 分钟",
	}
	for d, want := range cases {
		if got := humanWait(d); got != want {
			t.Errorf("humanWait(%v)=%q, want %q", d, got, want)
		}
	}
}
