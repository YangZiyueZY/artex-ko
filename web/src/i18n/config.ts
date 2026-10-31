// 支持的 locale 与默认值。本项目以中文为源语言、默认使用中文，并保留英语、韩语、
// 西班牙语、日语四种语言供用户设置切换。
export const LOCALES = ["en", "ko", "zh", "es", "ja"] as const;
export type Locale = (typeof LOCALES)[number];

// 默认显示语言为中文。英语、韩语、西班牙语、日语通过设置切换。构建时可用
// NEXT_PUBLIC_LOCALE 覆盖，运行时用户在设置界面选择的值（Cookie·localStorage）拥有最终决定权。
export const DEFAULT_LOCALE: Locale = "zh";

// 语言选择器中展示的各 locale 的自有语言名称。无论以哪种语言查看，母语名称都应以
// 该语言呈现，因此不作为翻译对象，而使用固定写法。
export const LOCALE_LABELS: Record<Locale, string> = {
  en: "English",
  ko: "한국어",
  zh: "中文",
  es: "Español",
  ja: "日本語",
};

// 保存用户所选显示语言的键。界面读取 localStorage 的值。Cookie 中也写入同样的值，
// 但目前没有读取它的地方（留待服务端需要判断语言时使用）。
export const LOCALE_COOKIE = "artex_locale";
export const LOCALE_STORAGE_KEY = "artex_locale";

// 用于将任意字符串收窄为受支持 locale 的守卫。
export function isLocale(raw: unknown): raw is Locale {
  return typeof raw === "string" && (LOCALES as readonly string[]).includes(raw);
}

// 在构建时决定当前 locale。为兼容静态导出（next.config 的 output: "export"），
// 不使用 cookies()·headers() 等动态 API，只读取环境变量。
// NEXT_PUBLIC_LOCALE 为空或不在支持列表内时回退到默认值。该值只用于服务端
// 预渲染的初始 HTML，实际用户显示语言会由客户端通过 Cookie 覆盖。
export function resolveLocale(): Locale {
  const raw = process.env.NEXT_PUBLIC_LOCALE;
  return isLocale(raw) ? raw : DEFAULT_LOCALE;
}
