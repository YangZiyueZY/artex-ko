/**
 * i18n locale 键一致性辅助模块。
 *
 * 以 zh.json 为基准（baseline），比较各 locale 的“扁平化键集合”，找出所有缺失（missing）
 * 与多余（extra）。CLI（check-i18n-keys.ts）与 Vitest 测试共享这套纯逻辑，
 * 因此检查规则只在一处定义。
 */

export interface KeyDiff {
  missing: string[]; // 基准中存在、目标中不存在的键
  extra: string[]; // 基准中不存在、目标中存在的键
}

/** 把嵌套 JSON 对象扁平化为 "a.b.c" 形式的键集合。数组·原始值按叶子处理。 */
export function flattenKeys(tree: unknown, prefix = ""): Set<string> {
  const out = new Set<string>();
  if (tree === null || typeof tree !== "object" || Array.isArray(tree)) {
    if (prefix) out.add(prefix);
    return out;
  }
  for (const [key, value] of Object.entries(tree as Record<string, unknown>)) {
    const path = prefix ? `${prefix}.${key}` : key;
    for (const leaf of flattenKeys(value, path)) out.add(leaf);
  }
  return out;
}

/** 返回目标相对基准的缺失·多余键（已排序数组）。 */
export function diffKeys(baseline: Set<string>, target: Set<string>): KeyDiff {
  const missing = [...baseline].filter((k) => !target.has(k)).sort();
  const extra = [...target].filter((k) => !baseline.has(k)).sort();
  return { missing, extra };
}

/** 缺失与多余都为空时返回 true。 */
export function isInSync(diff: KeyDiff): boolean {
  return diff.missing.length === 0 && diff.extra.length === 0;
}
