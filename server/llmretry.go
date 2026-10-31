package server

import (
	"time"

	"github.com/Autumn-27/artex/agent"
	"github.com/Autumn-27/artex/db"
)

// 重试策略的服务端解析，见 docs/LLM重试设计.md。六层里：
//   - 建连 / 空响应 / 同 provider 安全窗口 是「跟着端点走」的，每个 LLM 配置可以覆盖
//     全局默认（profile 的某项留空就继承全局，全局也没配就用内置默认）；
//   - 熔断 / 意图重跑 / 延迟重排 是进程级的，只有全局一份。
//
// 「模型报错不中断任务」靠最后两层 + 错误码分级（server/llmerror.go）：先原地指数退避
// 重跑，仍失败且属于外部可恢复错误（网络抖动/限流/5xx/额度耗尽）时退回 open 延迟重排；
// 鉴权、参数、模型名、上下文超长这类确定性错误不重试，直接交回规划者/用户。
//
// 全局策略读一次 DB 一行 settings，调用点都在低频路径（构建 provider、work 收尾、
// 保存配置），不值得再加一层缓存；熔断参数是例外——它在失败路径上每次都要读，所以
// 由 applyRetryPolicy 推给 Registry 保存。

// retryPolicy reads the global policy; a nil DB yields the zero policy (all
// layers on their built-in defaults).
func (s *Server) retryPolicy() db.LLMRetryPolicy {
	if s.m == nil || s.m.pg == nil {
		return db.LLMRetryPolicy{}
	}
	return s.m.pg.LLMRetryPolicy()
}

// resolveRetry layers one profile's override on top of the global policy and
// converts the result into the form agent.Config carries. Rules combine field by
// field, so a profile that only pins an interval still inherits the global count.
func resolveRetry(o db.RetryOverride, pol db.LLMRetryPolicy) agent.RetryConfig {
	connect := o.Connect.Or(pol.Connect)
	empty := o.Empty.Or(pol.Empty)
	stream := o.Stream.Or(pol.Stream)
	return agent.RetryConfig{
		// 次数在这里保持「0=默认 / 负=关闭」的原始语义:SDK 的 MaxRetries /
		// EmptyResponseRetries 与之完全同构,交给它自己解析即可。
		ConnectAttempts: connect.Attempts, ConnectInterval: connect.Interval(),
		EmptyAttempts: empty.Attempts, EmptyInterval: empty.Interval(),
		StreamAttempts: stream.Attempts, StreamInterval: stream.Interval(),
	}
}

// applyProfileRetry fills cfg.Retry for a profile read from the DB.
func (s *Server) applyProfileRetry(cfg *agent.Config, p *db.LLMProfile) {
	if p == nil {
		return
	}
	cfg.Retry = resolveRetry(p.Retry, s.retryPolicy())
}

// 熔断(轮询冷却)的默认值,与 llmpool 内置的一致 —— 这里只在「用户配了值」时才覆盖。

// 第 ⑤ 层(整条意图原地重跑)与第 ⑥ 层(延迟重排)的内置默认。
// 「模型报错不中断任务」就落在这两层：先秒级指数退避重跑几次；仍失败且属于
// 外部可恢复错误时，退回 open 并加领取栅栏，几分钟后自动重试，任务照常跑。
const (
	// 原地重跑：默认 5 次，退避 base×2^n（3s→6s→12s→24s→48s），单次封顶 60s。
	modelErrorRetries      = 5
	modelErrorRetryBase    = 3 * time.Second
	modelErrorRetryCap     = 60 * time.Second
	// 延迟重排：默认 4 次，1min→2min→4min→8min，单次封顶 10min。
	modelErrorRequeueCount = 4
	modelErrorRequeueBase  = time.Minute
	modelErrorRequeueCap   = 10 * time.Minute
)

// applyRetryPolicy pushes the process-wide layers of the policy into the objects
// that consume them on a hot path: the circuit-breaker registry. Called at
// startup and whenever the policy is saved.
func (s *Server) applyRetryPolicy() {
	pol := s.retryPolicy()
	if s.llmHealth != nil {
		s.llmHealth.SetPolicy(pol.Breaker.Attempts, pol.Breaker.Interval())
	}
}

// modelErrorRetryPolicy resolves the intent-level replay knobs (layer ⑤): how many
// times a model_error work is re-run, plus the backoff ladder between re-runs.
// 用户配的间隔(>0)会把指数退避改成固定间隔；次数 0=内置默认、负数=不重跑。
func (e *Engine) modelErrorRetryPolicy() (retries int, base, cap time.Duration) {
	var pol db.LLMRetryPolicy
	if e != nil && e.m != nil && e.m.pg != nil {
		pol = e.m.pg.LLMRetryPolicy()
	}
	return resolveModelErrorRetry(pol)
}

// resolveModelErrorRetry 是 modelErrorRetryPolicy 的纯函数内核（无 DB 便于单测）。
func resolveModelErrorRetry(pol db.LLMRetryPolicy) (retries int, base, cap time.Duration) {
	retries, base, cap = modelErrorRetries, modelErrorRetryBase, modelErrorRetryCap
	rule := pol.Intent
	if rule.Attempts != 0 {
		retries = max(rule.Attempts, 0)
	}
	if d := rule.Interval(); d > 0 {
		base, cap = d, d // 固定间隔
	}
	return retries, base, cap
}

// modelErrorRetryBackoff 是第 attempt 次(1 起)重跑前的退避。以变量暴露，便于测试置零。
var modelErrorRetryBackoff = func(attempt int, base, cap time.Duration) time.Duration {
	return expBackoff(attempt, base, cap)
}

// modelErrorRequeuePolicy resolves layer ⑥: how many delayed re-queues one model_error
// intent may consume, and how long each hold lasts. 次数 0=内置默认、负数=关闭这层；
// 间隔 0=指数梯度、>0=固定间隔。
func (e *Engine) modelErrorRequeuePolicy() (attempts int, backoff func(int) time.Duration) {
	var pol db.LLMRetryPolicy
	if e != nil && e.m != nil && e.m.pg != nil {
		pol = e.m.pg.LLMRetryPolicy()
	}
	return resolveModelErrorRequeue(pol)
}

// resolveModelErrorRequeue 是 modelErrorRequeuePolicy 的纯函数内核（无 DB 便于单测）。
func resolveModelErrorRequeue(pol db.LLMRetryPolicy) (attempts int, backoff func(int) time.Duration) {
	attempts, base, cap := modelErrorRequeueCount, modelErrorRequeueBase, modelErrorRequeueCap
	rule := pol.Requeue
	if rule.Attempts != 0 {
		attempts = max(rule.Attempts, 0)
	}
	if d := rule.Interval(); d > 0 {
		base, cap = d, d
	}
	if attempts <= 0 {
		return 0, func(int) time.Duration { return 0 }
	}
	return attempts, func(n int) time.Duration { return expBackoff(n, base, cap) }
}

// expBackoff 是 base×2^(n-1) 的退避梯度(n 从 1 起)，封顶 cap。
func expBackoff(n int, base, cap time.Duration) time.Duration {
	if n < 1 {
		n = 1
	}
	d := base
	for i := 1; i < n && d < cap; i++ {
		d *= 2
	}
	return min(d, cap)
}

// emptyTurnNudgeLimit resolves how many empty-turn continuations one work may
// inject (see steerHooks.Stop). It deliberately reuses layer ②'s knob —— 「空响应
// 重试次数」:两者是同一件事的两种手段。SDK 那层管「一个内容块都没有」，手段是把
// 同一个请求原样重发;这里管「只有思考、既无正文也无工具」，手段是追加一条指令让
// 模型带着已有思考接着做(原样重发对这种由上下文形状决定的空转没有意义)。判空口径
// 不同是因为 SDK 以「有没有 yield 过事件」为准，而思考增量本身就是事件——但用户配
// 「空响应重试几次」时想表达的是「模型没产出实质内容就再来一次」，两层共用一个次数
// 才对得上这个心智。
//
// 读全局策略而不是某个 profile 的覆盖:一个 run 中途可能因故障转移换 profile，而这
// 是整条意图的总量闸，不该跟着换端点而变。语义与 SDK 的 emptyRetries() 同构:
// 0 = 默认 defaultEmptyTurnNudges;-1(负) = 关闭空转续跑;>0 = 用该值。
func (e *Engine) emptyTurnNudgeLimit() int {
	if e == nil || e.m == nil || e.m.pg == nil {
		return defaultEmptyTurnNudges
	}
	switch n := e.m.pg.LLMRetryPolicy().Empty.Attempts; {
	case n == 0:
		return defaultEmptyTurnNudges
	case n < 0:
		return 0
	default:
		return n
	}
}
