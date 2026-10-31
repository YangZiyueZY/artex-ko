import { DEFAULT_LOCALE, isLocale, LOCALE_STORAGE_KEY, type Locale, resolveLocale } from "@/i18n/config";
import { getLocalStorageValue } from "@/lib/local-storage.client";

// 用于日期·时间格式化的 BCP47 区域设置标签。它跟随界面显示语言（locale-store），
// 因此用这个函数替代硬编码的 "ko-KR"。服务端渲染（无 window）时使用构建默认 locale，
// 客户端则反映用户选择的值。也可以在模块加载时只读取一次的 Intl 格式化器里安全使用。
const DATE_LOCALE: Record<Locale, string> = {
  en: "en-US",
  ko: "ko-KR",
  zh: "zh-CN",
  es: "es-ES",
  ja: "ja-JP",
};

export function getDateLocale(): string {
  if (typeof window === "undefined") return DATE_LOCALE[DEFAULT_LOCALE];
  const raw = getLocalStorageValue(LOCALE_STORAGE_KEY);
  const loc: Locale = isLocale(raw) ? raw : resolveLocale();
  return DATE_LOCALE[loc];
}
