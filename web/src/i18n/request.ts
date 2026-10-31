import { getRequestConfig } from "next-intl/server";

import { type Locale, resolveLocale } from "./config";

// 按 locale 拆分消息加载器。使用显式 import 映射，避免打包器把 messages/ 整个目录
// 当作 context 模块拉进来（排除仅本地存在的 zh.sources.json）。
const loaders: Record<Locale, () => Promise<{ default: Record<string, unknown> }>> = {
  en: () => import("../../messages/en.json"),
  ko: () => import("../../messages/ko.json"),
  zh: () => import("../../messages/zh.json"),
  es: () => import("../../messages/es.json"),
  ja: () => import("../../messages/ja.json"),
};

// next-intl 请求配置。本项目不使用 i18n 路径路由（分段·中间件），因此在这里
// 直接决定 locale。不引用 requestLocale，所以不会触发动态渲染。此设置只决定
// 服务端组件与预渲染的默认语言，运行时的用户切换由客户端 IntlRuntimeProvider
// 负责（messages/en·ko·zh·es·ja）。
export default getRequestConfig(async () => {
  const locale = resolveLocale();
  const messages = (await loaders[locale]()).default;
  return { locale, messages };
});
