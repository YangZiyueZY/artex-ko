package intercept

import "github.com/Autumn-27/artex/config"

// 拦截判定动作标签（allow/deny/ask）的多语言表。用于判定消息，跟随输出语言
// （config.ActiveLanguage）。未接线/不支持的语言回落到 zh。枚举键（allow/deny/ask）
// 与源标记（"[模型]"）是稳定标识符，不翻译。
var judgeActionLabels = map[string]map[string]string{
	"en": {"allow": "allow", "deny": "deny", "ask": "ask"},
	"ko": {"allow": "허용", "deny": "차단", "ask": "확인 요청"},
	"zh": {"allow": "允许", "deny": "拦截", "ask": "询问"},
	"es": {"allow": "permitir", "deny": "bloquear", "ask": "preguntar"},
	"ja": {"allow": "許可", "deny": "ブロック", "ask": "確認"},
}

func judgeActionLang() string {
	lang := config.ActiveLanguage()
	if _, ok := judgeActionLabels[lang]; ok {
		return lang
	}
	return "zh"
}
