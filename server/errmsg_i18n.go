package server

import "github.com/Autumn-27/artex/config"

// 用户可见的后端消息目录：以中文源文本为键，映射到 en/es/ja/ko 译文。本仓库的
// writeErr 响应文案与部分 writeJSON 错误字段以中文常量/字面量书写（F3b/F34）。
// 中文（zh）是键本身，因此不存放；查询失败或 zh 时返回键（即中文）。
//
// 键从源代码的中文常量/字面量按字节原样提取生成（scripts 辅助）。不是手工键入，
// 因此 writeErr 收到的运行期值与键不会错位而悄悄漏译。
//
// 格式字符串常量（%d、%s、%w 等）格式化后与键不同，因此调用方在传给 fmt.Sprintf/
// fmt.Errorf 之前，先用 trMsg(模板) 替换模板本身。

// trMsg 把中文源消息替换为当前输出语言。目录里没有、或当前语言是中文时返回键（即中文）。
// config.ActiveLanguage 反映运行期设置（设置表），用户选定的语言会立即生效。
func trMsg(ko string) string {
	if m, ok := errCatalog[ko]; ok {
		if v, ok2 := m[config.ActiveLanguage()]; ok2 && v != "" {
			return v
		}
	}
	return ko
}

// logT translates an operator-facing console log template (config.T) so server
// log.Printf sites can wrap a template without each file importing config.
func logT(ko string) string { return config.T(ko) }

// loadOutputLanguage 从唯一事实来源解析用户可见输出语言：运行期设置（设置表）优先于
// config.Language()（环境变量 ARTEX_LANG > 配置文件 > zh）。只在（重新）填充缓存时调用；
// 它会做一次 DB 查询和配置文件读取。
func (s *Server) loadOutputLanguage() string {
	if s.m != nil && s.m.pg != nil {
		if v, ok, _ := s.m.pg.GetSetting(settingOutputLanguage); ok {
			if nv := config.NormalizeLanguage(v); nv != "" {
				return nv
			}
		}
	}
	return config.Language()
}

// outputLanguage returns the cached output language, resolving it on first use.
func (s *Server) outputLanguage() string {
	if v, ok := s.outputLang.Load().(string); ok && v != "" {
		return v
	}
	v := s.loadOutputLanguage()
	s.outputLang.Store(v)
	return v
}
