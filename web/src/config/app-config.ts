import { DEFAULT_LOCALE, type Locale, resolveLocale } from "@/i18n/config";

import packageJson from "../../package.json";

const currentYear = new Date().getFullYear();

// 浏览器标签页标题与搜索引擎 meta 描述。它们作为 layout.tsx 的静态 metadata，
// 渲染到所有页面的 <title>·<meta name="description"> 中，因此按构建时的默认 locale
// 分支五种语言。标题使用冒号而非长破折号，
// 保持“产品名：描述”的形式。该 meta 是构建时的静态值，不会响应运行时的语言
// 切换（仅涉及标签页标题一行，故可接受）。
const META_BY_LOCALE: Record<Locale, { title: string; description: string }> = {
  en: {
    title: "ARTEX: Autonomous Penetration Testing Console",
    description: "An LLM-driven system console for autonomous penetration testing",
  },
  ko: {
    title: "ARTEX: 자율 침투 테스트 콘솔",
    description: "LLM 기반으로 자율 침투 테스트를 수행하는 시스템 콘솔",
  },
  zh: {
    title: "ARTEX — 自主渗透测试控制台",
    description: "LLM 驱动的自主渗透测试系统控制台",
  },
  es: {
    title: "ARTEX: Consola de Pruebas de Penetración Autónomas",
    description: "Una consola de sistema impulsada por LLM para pruebas de penetración autónomas",
  },
  ja: {
    title: "ARTEX: 自律型ペネトレーションテストコンソール",
    description: "LLM を活用して自律的にペネトレーションテストを行うシステムコンソール",
  },
};

export const APP_CONFIG = {
  name: "ARTEX",
  version: packageJson.version,
  copyright: `© ${currentYear}, ARTEX.`,
  meta: META_BY_LOCALE[resolveLocale()] ?? META_BY_LOCALE[DEFAULT_LOCALE],
};
