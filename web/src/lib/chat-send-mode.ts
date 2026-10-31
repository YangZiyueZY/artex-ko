"use client";

import * as React from "react";

import { useTranslations } from "next-intl";

import { getLocalStorageValue, setLocalStorageValue } from "@/lib/local-storage.client";

// 会话输入框的发送/换行按键组合。属纯前端配置，只保存在 localStorage，
// 不保存到服务器，也不与账号同步。因此更换浏览器后需要重新设置。
// 参考 issue #39：0.3.2 将 Ctrl+Enter 发送改为 Enter 发送，
// 这里把旧的按键组合重新作为选项提供。
export type ChatSendMode = "enter" | "ctrl-enter";

export const CHAT_SEND_MODE_KEY = "artex_chat_send_mode";
export const DEFAULT_CHAT_SEND_MODE: ChatSendMode = "enter";

// 设置界面发送方式下拉框使用的选项。标签随 locale 变化，因此不放在模块
// 常量中，而是通过 hook 返回（在纯模块里硬编码韩语会让 zh locale 显示为韩语）。
// value 是稳定标识符，可直接用于 localStorage 键与行为判定。
export function useChatSendModeOptions(): { value: ChatSendMode; label: string }[] {
  const t = useTranslations("chatSendMode.option");
  return [
    { value: "enter", label: t("enter") },
    { value: "ctrl-enter", label: t("ctrlEnter") },
  ];
}

function parseMode(raw: string | null): ChatSendMode {
  return raw === "ctrl-enter" || raw === "enter" ? raw : DEFAULT_CHAT_SEND_MODE;
}

// 同一标签页内的订阅者集合。localStorage 的 storage 事件只在“其他”标签页触发，
// 因此在当前标签页修改设置后，必须通过 emit 通知同一标签页的输入框。否则
// 需要刷新页面才能生效。
const listeners = new Set<() => void>();

function subscribe(listener: () => void) {
  listeners.add(listener);
  window.addEventListener("storage", listener);
  return () => {
    listeners.delete(listener);
    window.removeEventListener("storage", listener);
  };
}

// 返回字符串字面量，因此 Object.is 按值比较，useSyncExternalStore 不会陷入
// 无限循环。
function getSnapshot(): ChatSendMode {
  return parseMode(getLocalStorageValue(CHAT_SEND_MODE_KEY));
}

// 服务端没有 localStorage，所以先渲染默认值，hydrate 后再由 getSnapshot
// 修正取值。
function getServerSnapshot(): ChatSendMode {
  return DEFAULT_CHAT_SEND_MODE;
}

export function useChatSendMode(): ChatSendMode {
  return React.useSyncExternalStore(subscribe, getSnapshot, getServerSnapshot);
}

export function setChatSendMode(mode: ChatSendMode) {
  setLocalStorageValue(CHAT_SEND_MODE_KEY, mode);
  for (const listener of listeners) listener();
}

// shouldSubmitOnKey 判断一次按键是否对应发送。
// isComposing / keyCode 229 表示韩语·中文等输入法正在组合字符，
// 必须直接放行。否则用 Enter 确认字符时会误发送。
// enter 模式只排除 Shift，与 0.3.2 的行为逐字保持一致（未修改设置的
// 用户操作手感不变）。ctrl-enter 模式同时接受 Ctrl 与 Cmd（macOS）。
export function shouldSubmitOnKey(e: React.KeyboardEvent, mode: ChatSendMode): boolean {
  if (e.key !== "Enter") return false;
  if (e.nativeEvent.isComposing || e.nativeEvent.keyCode === 229) return false;
  if (mode === "ctrl-enter") return e.ctrlKey || e.metaKey;
  return !e.shiftKey;
}
