/**
 * Script: check-i18n-keys.ts
 *
 * locale 消息文件（web/messages/{ko,en,zh,es,ja}.json）的键一致性守卫。
 * 以 zh.json 为基准，检查其余 locale 的键集合是否完全一致（缺失·多余都查），
 * 不一致则以非 0 退出。执行：npm run check:i18n
 */

import { diffKeys, flattenKeys, isInSync } from "./i18n-keys";
import fs from "node:fs";
import path from "node:path";

const MESSAGES_DIR = path.resolve(__dirname, "../../messages");
const BASELINE = "zh";
const LOCALES = ["en", "ko", "es", "ja"] as const;

function readLocale(locale: string): Set<string> {
  const file = path.join(MESSAGES_DIR, `${locale}.json`);
  const tree = JSON.parse(fs.readFileSync(file, "utf8")) as unknown;
  return flattenKeys(tree);
}

function preview(keys: string[]): string {
  const shown = keys.slice(0, 20).join(", ");
  return keys.length > 20 ? `${shown} …` : shown;
}

const baseline = readLocale(BASELINE);
let failed = false;

console.log("─".repeat(64));
console.log(`i18n 键一致性检查（基准：${BASELINE}.json，${baseline.size} 个键）`);
console.log("─".repeat(64));

for (const locale of LOCALES) {
  const diff = diffKeys(baseline, readLocale(locale));
  if (isInSync(diff)) {
    console.log(`[OK] ${locale}.json — 一致`);
    continue;
  }
  failed = true;
  console.log(`[FAIL] ${locale}.json — 不一致`);
  if (diff.missing.length) console.log(`   缺失 ${diff.missing.length}：${preview(diff.missing)}`);
  if (diff.extra.length) console.log(`   多余 ${diff.extra.length}：${preview(diff.extra)}`);
}

console.log("─".repeat(64));
if (failed) {
  console.error("i18n 键不一致。请补齐缺失/多余键后重新运行。");
  process.exit(1);
}
console.log("所有 locale 的键集合一致。");
