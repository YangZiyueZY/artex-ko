/**
 * Script: extract-i18n.ts
 *
 * ARTEX 中文化 —— UI 字符串首次提取器。
 *
 * 用 TypeScript AST 解析 `src/` 下的所有 `.ts`/`.tsx`，只提取面向用户展示的
 * 硬编码中文字符串。提取对象分三类。
 *   1) 字符串字面量            例) title: "仪表盘"
 *   2) 模板字面量              例) `已删除 ${n} 个对话` → "已删除 {var0} 个对话"
 *   3) JSX 文本节点            例) <span>故障转移</span>
 * 注释（`//`、`/* *\/`）不是 AST 节点，因此会自动排除。前置·后置注释中的
 * 中文不是翻译对象（BRIEF：注释翻译优先级最低）。
 *
 * 产出物（web/messages/）：
 *   - zh.json          按命名空间嵌套的 { 键: "原文中文" }（保留原文用于对照上游）
 *   - ko.json          同样的骨架，值为 ""（B3 阶段填入韩语）
 *   - zh.sources.json  每个键的来源（文件:行）·类型·占位符 —— 供 B2 接线、B3 翻译、漂移追踪使用
 *
 * 重新提取是非破坏性的。只从代码中刷新提取生成的顶层命名空间（基于文件路径：
 * app·components·lib 等），其余“精选命名空间”
 *（手工翻译好的 nav·search·header 等）从既有文件中原样保留。规则：
 * 提取命名空间是清单，不要手工修改；运行时翻译只放在精选命名空间里。
 * 这样重新提取才不会覆盖已有翻译。
 *
 * 命名空间从文件路径得到。例) src/app/(main)/system/llm/page.tsx
 *   → app.main.system.llm.page  （去掉路由组括号、去掉 _ 文件夹的下划线）
 * 键是原文 sha1 的前 8 位。即使顺序或文件变化，相同文案仍得到相同的键，
 * 重跑时 diff 更小（便于对照上游更新）。
 *
 * 执行：npm run extract:i18n
 */

import * as ts from "typescript";

import crypto from "node:crypto";
import fs from "node:fs";
import path from "node:path";

const SRC_DIR = path.resolve(__dirname, "..");
const MESSAGES_DIR = path.resolve(__dirname, "../../messages");
const REPO_WEB_DIR = path.resolve(__dirname, "../..");

/** 是否包含至少 1 个汉字（Han）。判断 UI 翻译对象的唯一标准。 */
const HAN = /\p{Script=Han}/u;

/** 提取时排除的目录（解析工具自身·构建产物）。 */
const SKIP_DIRS = new Set(["scripts", "node_modules", ".next"]);

type Kind = "string" | "template" | "jsx";

interface Message {
  ns: string;
  key: string;
  text: string;
  kind: Kind;
  placeholders: string[]; // 模板中 ${...} 的原始表达式（仅供参考）
  occurrences: string[]; // "相对路径:行"
}

/** 以 src 为基准的相对路径 → 用点连接的命名空间。 */
function toNamespace(absFile: string): string {
  const rel = path.relative(SRC_DIR, absFile).replace(/\.[tj]sx?$/, "");
  return rel
    .split(path.sep)
    .map((seg) =>
      seg
        .replace(/^\((.*)\)$/, "$1") // (main) → main  （路由组）
        .replace(/^_/, "") // _components → components
        .replace(/[^A-Za-z0-9가-힣]+/g, "-")
        .replace(/^-+|-+$/g, ""),
    )
    .filter(Boolean)
    .join(".");
}

function hashKey(text: string): string {
  return crypto.createHash("sha1").update(text).digest("hex").slice(0, 8);
}

/** 递归遍历目录，收集 .ts/.tsx 文件。 */
function collectFiles(dir: string, out: string[] = []): string[] {
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    if (entry.isDirectory()) {
      if (SKIP_DIRS.has(entry.name)) continue;
      collectFiles(path.join(dir, entry.name), out);
    } else if (/\.(ts|tsx)$/.test(entry.name)) {
      out.push(path.join(dir, entry.name));
    }
  }
  return out;
}

/** 把模板字面量重组为“字面量部分 + {varN}”。返回 null 表示字面量部分不含汉字。 */
function reconstructTemplate(node: ts.TemplateExpression): { text: string; placeholders: string[] } | null {
  let text = node.head.text;
  const placeholders: string[] = [];
  node.templateSpans.forEach((span, i) => {
    placeholders.push(span.expression.getText());
    text += `{var${i}}${span.literal.text}`;
  });
  // 若汉字只出现在 ${} 内、固定文案中没有，那么这些汉字会作为内部节点另行捕获。
  if (!HAN.test(node.head.text) && !node.templateSpans.some((s) => HAN.test(s.literal.text))) {
    return null;
  }
  return { text, placeholders };
}

/** JSX 文本规范化：去掉首尾空白，并把内部连续空白/换行压成一个空格（与浏览器渲染规则一致）。 */
function normalizeJsxText(raw: string): string {
  return raw.replace(/\s+/g, " ").trim();
}

const messages = new Map<string, Message>(); // "ns\u0000key" → Message
const scannedFiles = collectFiles(SRC_DIR).sort();
const capturedLines = new Map<string, Set<number>>(); // 文件 → 提取节点命中的行号集合
let fileWithHan = 0;

function record(ns: string, text: string, kind: Kind, placeholders: string[], relFile: string, line: number) {
  const key = hashKey(text);
  const id = `${ns}\u0000${key}`;
  let msg = messages.get(id);
  if (!msg) {
    msg = { ns, key, text, kind, placeholders, occurrences: [] };
    messages.set(id, msg);
  }
  const occ = `${relFile}:${line}`;
  if (!msg.occurrences.includes(occ)) msg.occurrences.push(occ);
}

for (const absFile of scannedFiles) {
  const source = fs.readFileSync(absFile, "utf8");
  if (!HAN.test(source)) continue;
  fileWithHan++;

  const relFile = path.relative(REPO_WEB_DIR, absFile);
  const ns = toNamespace(absFile);
  const scriptKind = absFile.endsWith(".tsx") ? ts.ScriptKind.TSX : ts.ScriptKind.TS;
  const sf = ts.createSourceFile(absFile, source, ts.ScriptTarget.Latest, true, scriptKind);
  const lineOf = (pos: number) => sf.getLineAndCharacterOfPosition(pos).line + 1;
  const markLine = (pos: number) => {
    const set = capturedLines.get(relFile) ?? new Set<number>();
    set.add(lineOf(pos));
    capturedLines.set(relFile, set);
  };

  const visit = (node: ts.Node): void => {
    if (ts.isStringLiteralLike(node) && !ts.isTemplateExpression(node.parent)) {
      // 字符串字面量 + 无替换的模板。（含替换的模板的 head/middle 不会命中这里）
      if (HAN.test(node.text)) {
        record(ns, node.text, "string", [], relFile, lineOf(node.getStart(sf)));
        markLine(node.getStart(sf));
      }
    } else if (ts.isTemplateExpression(node)) {
      const r = reconstructTemplate(node);
      if (r) {
        record(ns, r.text, "template", r.placeholders, relFile, lineOf(node.getStart(sf)));
        markLine(node.getStart(sf));
      }
    } else if (ts.isJsxText(node)) {
      const text = normalizeJsxText(node.text);
      if (text && HAN.test(text)) {
        record(ns, text, "jsx", [], relFile, lineOf(node.getStart(sf)));
        markLine(node.getStart(sf));
      }
    }
    ts.forEachChild(node, visit);
  };
  visit(sf);
}

// --- 组装嵌套 JSON ---------------------------------------------------------
type Tree = { [k: string]: Tree | string };

function setNested(root: Tree, nsPath: string, key: string, value: string) {
  const segs = nsPath.split(".");
  let node: Tree = root;
  for (const seg of segs) {
    if (typeof node[seg] !== "object") node[seg] = {};
    node = node[seg] as Tree;
  }
  node[key] = value;
}

/** 递归排序键，最小化重跑时的 diff。 */
function sortTree(t: Tree): Tree {
  const out: Tree = {};
  for (const k of Object.keys(t).sort()) {
    const v = t[k];
    out[k] = typeof v === "string" ? v : sortTree(v);
  }
  return out;
}

const zhTree: Tree = {};
const koTree: Tree = {};
const sources: Record<string, { text: string; kind: Kind; placeholders: string[]; occurrences: string[] }> = {};

for (const msg of messages.values()) {
  setNested(zhTree, msg.ns, msg.key, msg.text);
  setNested(koTree, msg.ns, msg.key, "");
  sources[`${msg.ns}.${msg.key}`] = {
    text: msg.text,
    kind: msg.kind,
    placeholders: msg.placeholders,
    occurrences: msg.occurrences.sort(),
  };
}

const sortedSources: typeof sources = {};
for (const k of Object.keys(sources).sort()) sortedSources[k] = sources[k];

fs.mkdirSync(MESSAGES_DIR, { recursive: true });
const write = (name: string, data: unknown) =>
  fs.writeFileSync(path.join(MESSAGES_DIR, name), `${JSON.stringify(data, null, 2)}\n`, "utf8");

/** 读取既有消息文件。不存在或损坏时返回空树。 */
function readTree(name: string): Tree {
  try {
    return JSON.parse(fs.readFileSync(path.join(MESSAGES_DIR, name), "utf8")) as Tree;
  } catch {
    return {};
  }
}

/**
 * 非破坏性合并。提取新生成的树（fresh）中的顶层命名空间以代码为准，
 * 直接采用；只存在于既有文件（existing）中的顶层命名空间（=手工翻译的精选
 * 命名空间）予以保留。提取命名空间内过时的（上游已删除的）键会自然
 * 消失（便于对照上游）。
 */
function mergeCurated(fresh: Tree, existing: Tree): Tree {
  const extractedTop = new Set(Object.keys(fresh));
  const out: Tree = { ...fresh };
  for (const [ns, subtree] of Object.entries(existing)) {
    if (!extractedTop.has(ns)) out[ns] = subtree;
  }
  return out;
}

write("zh.json", sortTree(mergeCurated(zhTree, readTree("zh.json"))));
write("ko.json", sortTree(mergeCurated(koTree, readTree("ko.json"))));
write("zh.sources.json", sortedSources);

// --- 汇总报告 ------------------------------------------------------------
const byKind: Record<Kind, number> = { string: 0, template: 0, jsx: 0 };
const byNs = new Map<string, number>();
let totalOccurrences = 0;
for (const msg of messages.values()) {
  byKind[msg.kind]++;
  byNs.set(msg.ns, (byNs.get(msg.ns) ?? 0) + 1);
  totalOccurrences += msg.occurrences.length;
}

// 覆盖率估算：含汉字的源码行中，被提取节点命中的行所占比例。
// 其余基本上都是注释（=非翻译对象）。
let hanLines = 0;
let capturedHanLines = 0;
for (const absFile of scannedFiles) {
  const relFile = path.relative(REPO_WEB_DIR, absFile);
  const lines = fs.readFileSync(absFile, "utf8").split(/\r?\n/);
  const capSet = capturedLines.get(relFile) ?? new Set<number>();
  lines.forEach((ln, i) => {
    if (HAN.test(ln)) {
      hanLines++;
      if (capSet.has(i + 1)) capturedHanLines++;
    }
  });
}

const topNs = [...byNs.entries()].sort((a, b) => b[1] - a[1]).slice(0, 15);

console.log("─".repeat(64));
console.log("ARTEX i18n 提取完成");
console.log("─".repeat(64));
console.log(`扫描文件            : ${scannedFiles.length}（src/ 下的 .ts/.tsx，不含 scripts）`);
console.log(`含汉字的文件         : ${fileWithHan}`);
console.log(`唯一消息（键）       : ${messages.size}`);
console.log(`  ├─ 字符串字面量     : ${byKind.string}`);
console.log(`  ├─ 模板字面量       : ${byKind.template}`);
console.log(`  └─ JSX 文本         : ${byKind.jsx}`);
console.log(`总出现位置           : ${totalOccurrences}（含重复使用）`);
console.log(`命名空间数量         : ${byNs.size}`);
console.log("");
console.log(`含汉字的源码行       : ${hanLines}`);
console.log(`  └─ 提取捕获的行     : ${capturedHanLines}（${((capturedHanLines / hanLines) * 100).toFixed(1)}%）`);
console.log(`     其余 ${hanLines - capturedHanLines} 行大多是代码注释（非翻译对象）`);
console.log("");
console.log("消息最多的命名空间前 15 名：");
for (const [ns, n] of topNs) console.log(`  ${String(n).padStart(4)}  ${ns}`);
console.log("");
console.log(`输出：web/messages/{zh.json, ko.json, zh.sources.json}`);
console.log("─".repeat(64));
