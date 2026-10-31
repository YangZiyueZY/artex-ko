package agent

// 面向用户的输出语言（展示语言）选择。本仓库把英文·韩文·中文·西班牙文·日文五种语言中的
// 一种强制为智能体呈现给用户的自然语言。智能体的「大脑」（经过基准测试的中文推理正文，
// 段[A]）与语言无关，保持原样；这里只决定用户阅读的表层语言。

// OutputLanguage 是由服务端接线的钩子：返回服务端从配置（config 的 ARTEX_LANG、
// 运行期 settings KV）中选出的输出语言代码（"en"/"ko"/"zh"/"es"/"ja"）。为 nil 或
// 空值时回落到下面的默认值。注入方式与 PromptOverride 相同。
var OutputLanguage func() string

// 服务端未接该钩子时（例如只跑 agent 包单测）的默认输出语言。
// 默认值为简体中文（仅作为 agent 包单测的兜底；运行时由服务端按设置层的默认值接线
// OutputLanguage 钩子）。英文·韩文·西班牙文·日文通过设置切换。
const defaultOutputLang = "zh"

// SupportedOutputLangs 是可切换的输出语言代码列表。
var SupportedOutputLangs = []string{"en", "ko", "zh", "es", "ja"}

// normalizeLang 把不在支持列表内的语言代码收窄为默认值。
func normalizeLang(code string) string {
	for _, c := range SupportedOutputLangs {
		if c == code {
			return code
		}
	}
	return defaultOutputLang
}

// resolveOutputLang 返回当前生效的输出语言代码：先看服务端接线的钩子，
// 没有就用默认值。
func resolveOutputLang() string {
	if OutputLanguage != nil {
		if code := OutputLanguage(); code != "" {
			return normalizeLang(code)
		}
	}
	return defaultOutputLang
}

// langPhrase 是（用中文写的）指令里指名「用这种语言书写」时使用的写法。
// 括号中同时给出该语言的自称，避免模型搞不清到底是哪种语言。
var langPhrase = map[string]string{
	"ko": "韩语（한국어）",
	"en": "英语（English）",
	"zh": "中文（简体中文）",
	"es": "西班牙语（Español）",
	"ja": "日语（日本語）",
}

// langBare 是指令内反复指名时使用的简短语言名。
var langBare = map[string]string{
	"ko": "韩语",
	"en": "英语",
	"zh": "中文",
	"es": "西班牙语",
	"ja": "日语",
}
