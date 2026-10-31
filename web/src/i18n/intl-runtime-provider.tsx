"use client";

import type { ReactNode } from "react";
import * as React from "react";

import { NextIntlClientProvider } from "next-intl";

import enMessages from "../../messages/en.json";
import esMessages from "../../messages/es.json";
import jaMessages from "../../messages/ja.json";
import koMessages from "../../messages/ko.json";
import zhMessages from "../../messages/zh.json";
import type { Locale } from "./config";
import { useActiveLocale } from "./locale-store";

// 静态打包全部五种语言的消息。这样用户在设置界面切换语言时，无需重新构建、刷新页面
// 或异步加载，所有文案都会就地在所选语言下重新渲染（也与静态导出 output: "export" 兼容）。
const MESSAGES: Record<Locale, Record<string, unknown>> = {
  en: enMessages as Record<string, unknown>,
  ko: koMessages as Record<string, unknown>,
  zh: zhMessages as Record<string, unknown>,
  es: esMessages as Record<string, unknown>,
  ja: jaMessages as Record<string, unknown>,
};

// 运行环境（浏览器·预渲染服务器）的时区。
const TIME_ZONE = Intl.DateTimeFormat().resolvedOptions().timeZone;

// 负责运行时语言切换的 Provider。它替代根布局的 NextIntlClientProvider，
// 按用户选择的当前 locale（Cookie·localStorage）挑选并供给 messages。
export function IntlRuntimeProvider({ children }: Readonly<{ children: ReactNode }>) {
  const locale = useActiveLocale();

  // 语言变化时同时调整 <html lang>。用客户端选择值覆盖根布局由服务端写入的
  // 初始 lang，使屏幕阅读器·浏览器翻译按正确语言工作。
  React.useEffect(() => {
    document.documentElement.lang = locale;
  }, [locale]);

  // 与根布局的 Provider 不同，这里不继承服务端请求配置，因此直接传入 timeZone。
  // 若省略，next-intl 会在预渲染期间报 ENVIRONMENT_FALLBACK 错误。
  return (
    <NextIntlClientProvider locale={locale} messages={MESSAGES[locale]} timeZone={TIME_ZONE}>
      {children}
    </NextIntlClientProvider>
  );
}
