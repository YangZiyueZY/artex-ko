package agent

import (
	"strings"
	"testing"
	"unicode"
)

// A2：收尾/结算提示词（wrap-up / settlement prompt）中文化守卫。
//
// 这些常量在 run 或任务因步骤/时长预算被终止时由 settlement 阶段注入，直接决定
// 展示给用户的最终摘要，因此 (1) 必须是简体中文（含汉字）、(2) 不得残留韩文
// （Hangul）、(3) 工具名与「只用一句话纯文本」这类指令含义必须原样保留。
//
// DB 里 wrapup_prompt / task_timeout_wrapup_prompt 默认为空串（db/db.go 的 builtin
// 智能体 INSERT 不填这两列），为空即回落到这里的常量，也就是说这些常量是收尾文句
// 的唯一来源。
func TestWrapupPromptsLocalizedToChinese(t *testing.T) {
	all := map[string]string{
		"settleWrapUpPrompt":        settleWrapUpPrompt,
		"plannerWrapUpDefault":      plannerWrapUpDefault,
		"mainAgentWrapUpDefault":    mainAgentWrapUpDefault,
		"genericWrapUpDefault":      genericWrapUpDefault,
		"workerTaskTimeoutDefault":  workerTaskTimeoutDefault,
		"plannerTaskTimeoutDefault": plannerTaskTimeoutDefault,
	}

	hasScript := func(s string, table *unicode.RangeTable) bool {
		for _, r := range s {
			if unicode.Is(table, r) {
				return true
			}
		}
		return false
	}

	for name, p := range all {
		if !hasScript(p, unicode.Han) {
			t.Errorf("%s：没有中文汉字，未被中文化：%q", name, p)
		}
		// 中文化完成后不应残留韩文：Hangul 与汉字分属不同 Unicode 区块，可以精确判定。
		if hasScript(p, unicode.Hangul) {
			t.Errorf("%s：仍残留韩文，翻译未完成：%q", name, p)
		}
	}

	// 工具名是标识符，不参与翻译，必须原样保留。
	// worker 系（per-run·task-timeout）要用 record_fact 落结论、report_finding 落漏洞，
	// 最后只用一句话纯文本向用户总结。
	mustContain := func(name, p string, subs ...string) {
		for _, s := range subs {
			if !strings.Contains(p, s) {
				t.Errorf("%s：应保留的指令含义 %q 缺失", name, s)
			}
		}
	}
	mustContain("settleWrapUpPrompt", settleWrapUpPrompt,
		"insert_assets", "record_fact", "report_finding", "一句话", "纯文本")
	mustContain("workerTaskTimeoutDefault", workerTaskTimeoutDefault,
		"insert_assets", "record_fact", "report_finding", "一句话", "纯文本")
	mustContain("genericWrapUpDefault", genericWrapUpDefault, "一句话", "纯文本")
	mustContain("mainAgentWrapUpDefault", mainAgentWrapUpDefault, "一句话", "纯文本")
	// planner 不出总结句（只做判定然后结束），但保留意图·目标·待办工具。
	mustContain("plannerWrapUpDefault", plannerWrapUpDefault, "add_intent", "prove_goal", "TodoWrite")
	mustContain("plannerTaskTimeoutDefault", plannerTaskTimeoutDefault, "prove_goal")

	// 下面这组把 worker / mainagent / planner 三类角色的语义差异钉死，防止有人把
	// 默认值统一替换成同一段中文文案后测试仍然是绿的。
	mustNotContain := func(name, p string, subs ...string) {
		for _, s := range subs {
			if strings.Contains(p, s) {
				t.Errorf("%s：不应出现 %q", name, s)
			}
		}
	}
	// 会产出用户可见总结的角色：必须点名用简体中文书写。
	for _, name := range []string{"settleWrapUpPrompt", "mainAgentWrapUpDefault", "genericWrapUpDefault", "workerTaskTimeoutDefault"} {
		mustContain(name, all[name], "简体中文")
	}
	// planner 系两个默认值都明确「不输出总结文本」，因此不得出现「一句话」总结指令。
	mustContain("plannerWrapUpDefault", plannerWrapUpDefault, "不输出总结文本", "任务本身并未结束")
	mustContain("plannerTaskTimeoutDefault", plannerTaskTimeoutDefault, "不输出总结文本", "不要再产生任何新意图")
	mustNotContain("plannerWrapUpDefault", plannerWrapUpDefault, "一句话")
	mustNotContain("plannerTaskTimeoutDefault", plannerTaskTimeoutDefault, "一句话", "add_intent")
	// worker 的 per-run 谈「本次执行」的预算，task-timeout 谈「整个任务」的时长上限。
	mustContain("settleWrapUpPrompt", settleWrapUpPrompt, "本次执行")
	mustContain("workerTaskTimeoutDefault", workerTaskTimeoutDefault, "整个任务已到达时长上限")
	mustNotContain("workerTaskTimeoutDefault", workerTaskTimeoutDefault, "本次规划轮次")
	// mainagent 只谈本次交互，不涉及任务级超时（它没有 task-timeout 文案）。
	mustContain("mainAgentWrapUpDefault", mainAgentWrapUpDefault, "步骤预算即将耗尽")
	mustNotContain("mainAgentWrapUpDefault", mainAgentWrapUpDefault, "整个任务已到达时长上限")
}

// per-run 与 task-timeout 的含义必须不同（尤其是 planner）：per-run 是「只结束本轮」，
// task-timeout 是「整个任务到点结束」。常量映射若被打乱、两者混同就会在这里报错。
func TestWrapupDefaultsRouting(t *testing.T) {
	if WrapupDefault("worker") != settleWrapUpPrompt {
		t.Error("worker 的 per-run 默认值不是 settleWrapUpPrompt")
	}
	if WrapupDefault("planner") != plannerWrapUpDefault {
		t.Error("planner 的 per-run 默认值不是 plannerWrapUpDefault")
	}
	if WrapupDefault("mainagent") != mainAgentWrapUpDefault {
		t.Error("mainagent 的 per-run 默认值不是 mainAgentWrapUpDefault")
	}
	// 未注册的 key（自定义智能体）回落到 generic。
	if WrapupDefault("unknown-agent") != genericWrapUpDefault {
		t.Error("未注册的 key 没有回落到 genericWrapUpDefault")
	}
	// task-timeout 只给 worker/planner 配置，其余为空串（调用方回落到 per-run）。
	if TaskTimeoutWrapupDefault("worker") != workerTaskTimeoutDefault {
		t.Error("worker 的 task-timeout 默认值不是 workerTaskTimeoutDefault")
	}
	if TaskTimeoutWrapupDefault("planner") != plannerTaskTimeoutDefault {
		t.Error("planner 的 task-timeout 默认值不是 plannerTaskTimeoutDefault")
	}
	if TaskTimeoutWrapupDefault("mainagent") != "" {
		t.Error("mainagent 不应有 task-timeout 文案（应为空字符串）")
	}
	// per-run 与 task-timeout 文案相同，就意味着语义区分已经消失。
	if workerTaskTimeoutDefault == settleWrapUpPrompt {
		t.Error("worker 的 per-run 与 task-timeout 文案相同")
	}
	if plannerTaskTimeoutDefault == plannerWrapUpDefault {
		t.Error("planner 的 per-run 与 task-timeout 文案相同")
	}
}

// TestWrapupLangTailAcrossLanguages 固定 withWrapupLang 的语言切换覆盖：收尾提示词不像
// 系统提示词那样自带尾巴，必须靠这里追加 langDirective，否则用户看到的最后一句总结就不
// 受所选展示语言约束。内置收尾正文本身是简体中文操作指令，因此【所有】可切换语言（含
// ko）都要追加对应尾巴——少了这一条，选韩语/英语的用户会收到中文总结。
func TestWrapupLangTailAcrossLanguages(t *testing.T) {
	t.Cleanup(func() { OutputLanguage = nil })

	const body = "收尾正文。"
	for _, code := range SupportedOutputLangs {
		lang := code
		OutputLanguage = func() string { return lang }
		got := withWrapupLang(body)

		if want := body + langDirective(lang); got != want {
			t.Errorf("%s：withWrapupLang 没有追加所选语言的尾巴，got %q want %q", lang, got, want)
		}
		if !strings.Contains(got, "输出语言规约") || !strings.Contains(got, langMarker[lang]) {
			t.Errorf("%s：收尾提示词缺少点名 %q 的输出语言尾巴：%q", lang, langMarker[lang], got)
		}
		// 正文是中文指令，展示语言非中文时必须带上「不得把中文输出给用户」的反漂移条款；
		// 展示语言就是中文时该条款必须缺席。
		hasNoZhClause := strings.Contains(got, "也绝不能把中文输出给用户")
		if lang == "zh" && hasNoZhClause {
			t.Errorf("zh：展示语言为中文时不应禁止中文输出：%q", got)
		}
		if lang != "zh" && !hasNoZhClause {
			t.Errorf("%s：收尾提示词缺少禁止中文泄漏给用户的条款：%q", lang, got)
		}
	}

	// 钩子未接线（agent 包单测的默认路径）时走 defaultOutputLang，尾巴必须点名简体中文。
	OutputLanguage = nil
	gotDefault := withWrapupLang(body)
	if !strings.Contains(gotDefault, "输出语言规约") || !strings.Contains(gotDefault, langBare[defaultOutputLang]) {
		t.Errorf("默认语言 %s 的收尾提示词缺少中文输出尾巴：%q", defaultOutputLang, gotDefault)
	}
	// 空文案（如 mainagent 的 task-timeout）原样返回，不追加尾巴。
	if got := withWrapupLang(""); got != "" {
		t.Errorf("空提示词不应被追加尾巴：%q", got)
	}
}
