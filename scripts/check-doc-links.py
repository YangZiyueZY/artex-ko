#!/usr/bin/env python3
"""检查仓库内 Markdown 文档的内部链接、图片和锚点引用完整性。

遍历所有受跟踪的 `.md` 文档，确认两件事。

1. 指向仓库内其他文件的相对路径链接（`[text](path)`）和图片
   引用（`![alt](path)`·`<img src="path">`）是否指向确实存在的文件。
2. 文档锚点链接（`[text](#heading)`·`[text](other.md#heading)`）是否指向目标 `.md`
   文档中确实存在的标题。锚点 slug 用标准库复现 GitHub 的规则
   （转小写 → 只保留 `\\p{Word}`·连字符·空格后删除其余 → 空格转连字符，
   同一 slug 冲突时按出现顺序加 `-1`·`-2` 后缀）生成。

只要有一个损坏的引用就以退出码 1 结束，因此可直接用于 CI 合并门禁
（`.github/workflows/docs.yml`）和贡献者的本地验证。

不检查的对象：
- 外部 URL（`http://`·`https://`·`mailto:`·`tel:`·`data:`）：依赖网络，
  flaky，因此这个确定性门禁不处理。外部链接状态另行
  检查。
- 指向非 `.md` 文件的链接中 `#` 之后的部分（例如源码行号）：不是标题
  锚点，因此只确认文件存在，不检查锚点。
- 图片引用中的 `#` 片段：图片没有标题锚点的意义。
- fenced code block（用 ``` 或 ~~~ 包裹的块）中的示例链接·标题样注释：
  不是真实引用·标题，而是代码示例，因此跳过。

只使用标准库，不访问网络。运行：`python3 -I scripts/check-doc-links.py`
"""
import os
import re
import subprocess
import sys
import unicodedata
from collections import defaultdict

# [text](path) 中非图片（`![...]`）的链接。路径截到空格为止（不含标题 "..."）。
MD_LINK = re.compile(r"(?<!\!)\[[^\]]*\]\(([^)\s]+)")
# ![alt](path) 图片链接。
MD_IMAGE = re.compile(r"!\[[^\]]*\]\(([^)\s]+)")
# <img ... src="path" ...> HTML 图片。
HTML_IMAGE = re.compile(r"<img[^>]*\bsrc=[\"']([^\"']+)[\"']", re.IGNORECASE)

# 网络·非文件 scheme。锚点（`#`）是本文档内引用，因此不列入这里。
EXTERNAL_PREFIXES = ("http://", "https://", "mailto:", "tel:", "data:")

ATX_HEADING = re.compile(r"^(#{1,6})\s+(.*?)\s*#*\s*$")
# 去掉标题文本中的内联 Markdown，只保留纯文本。
MD_INLINE_LINK = re.compile(r"\[([^\]]*)\]\([^)]*\)")


def is_external(target: str) -> bool:
    return target.startswith(EXTERNAL_PREFIXES)


def _is_word_char(ch: str) -> bool:
    """判断字符是否为 GitHub 的锚点 slug 规则保留的 `\\p{Word}` 字符。

    Ruby 正则 `\\p{Word}` = Letter(L*) + Mark(M*) + Decimal_Number(Nd)
    + Connector_Punctuation(Pc) + Join_Control(U+200C·U+200D)。韩文·汉字·
    下划线·emoji 的 variation selector(Mn) 等都会原样保留。
    """
    if ch == "_":
        return True
    if ch in ("‌", "‍"):
        return True
    category = unicodedata.category(ch)
    if category[0] in ("L", "M"):
        return True
    return category in ("Nd", "Pc")


def slugify(text: str) -> str:
    """把标题文本转换为 GitHub 的锚点 slug（不含重复后缀处理）。"""
    text = text.strip().lower()
    out = []
    for ch in text:
        if ch == " ":
            out.append("-")
        elif ch == "-" or _is_word_char(ch):
            out.append(ch)
    return "".join(out)


def heading_slugs(root: str, md: str) -> set:
    """根据文件中所有 ATX 标题生成 GitHub 锚点 slug 集合。

    同一 slug 冲突时像 GitHub 一样按出现顺序添加 `-1`·`-2` 后缀。
    """
    slugs = set()
    counts = defaultdict(int)
    in_fence = False
    with open(os.path.join(root, md), encoding="utf-8", errors="ignore") as fh:
        for line in fh:
            stripped = line.lstrip()
            if stripped.startswith("```") or stripped.startswith("~~~"):
                in_fence = not in_fence
                continue
            if in_fence:
                continue
            match = ATX_HEADING.match(line.rstrip("\n"))
            if not match:
                continue
            raw = match.group(2)
            raw = MD_INLINE_LINK.sub(r"\1", raw)
            raw = raw.replace("`", "").replace("*", "").replace("_", "")
            base = slugify(raw)
            seen = counts[base]
            counts[base] += 1
            slugs.add(base if seen == 0 else f"{base}-{seen}")
    return slugs


def main() -> int:
    root = subprocess.check_output(
        ["git", "rev-parse", "--show-toplevel"], text=True
    ).strip()
    tracked = subprocess.check_output(
        ["git", "ls-files"], cwd=root, text=True
    ).splitlines()
    md_files = [f for f in tracked if f.endswith(".md")]

    slug_cache: dict = {}

    def slugs_of(md_path: str) -> set:
        if md_path not in slug_cache:
            slug_cache[md_path] = heading_slugs(root, md_path)
        return slug_cache[md_path]

    checked_files = 0
    checked_anchors = 0
    broken_files = []
    broken_anchors = []
    for md in md_files:
        with open(os.path.join(root, md), encoding="utf-8", errors="ignore") as fh:
            lines = fh.readlines()
        base_dir = os.path.dirname(md)
        in_fence = False
        for lineno, line in enumerate(lines, 1):
            stripped = line.lstrip()
            if stripped.startswith("```") or stripped.startswith("~~~"):
                in_fence = not in_fence
                continue
            if in_fence:
                continue
            for pattern in (MD_LINK, MD_IMAGE, HTML_IMAGE):
                for match in pattern.finditer(line):
                    target = match.group(1).strip()
                    if is_external(target):
                        continue

                    if target.startswith("#"):
                        file_part, anchor = "", target[1:]
                    elif "#" in target:
                        file_part, anchor = target.split("#", 1)
                    else:
                        file_part, anchor = target, ""
                    file_part = file_part.split("?", 1)[0]
                    anchor = anchor.strip()

                    # 链接指向的 .md 文档（锚点检查对象）。链接其他文件时就是该文件，
                    # 没有文件部分、只有 `#锚点` 时就是本文档。
                    target_md = None
                    if file_part:
                        checked_files += 1
                        resolved = os.path.normpath(
                            os.path.join(root, base_dir, file_part)
                        )
                        if not os.path.exists(resolved):
                            broken_files.append((md, lineno, target))
                            continue
                        if file_part.endswith(".md"):
                            target_md = os.path.normpath(
                                os.path.join(base_dir, file_part)
                            )
                    else:
                        target_md = md

                    # 锚点只对正文链接（MD_LINK）有意义。图片引用的
                    # `#` 片段会跳过。
                    if anchor and target_md is not None and pattern is MD_LINK:
                        checked_anchors += 1
                        if anchor not in slugs_of(target_md):
                            broken_anchors.append((md, lineno, target, target_md))

    print(
        f"受跟踪 Markdown {len(md_files)} 个 · 文件引用 {checked_files} 个 · "
        f"锚点引用 {checked_anchors} 个已检查"
    )
    if broken_files:
        print(f"损坏的文件引用 {len(broken_files)} 个 —— 指向的文件不在仓库中：")
        for md, lineno, target in broken_files:
            print(f"  {md}:{lineno} -> {target}")
    if broken_anchors:
        print(f"损坏的锚点引用 {len(broken_anchors)} 个 —— 目标文档中没有该标题：")
        for md, lineno, target, target_md in broken_anchors:
            print(f"  {md}:{lineno} -> {target}  （目标：{target_md}）")
    if broken_files or broken_anchors:
        return 1
    print("损坏引用 0 —— 所有内部链接·图片·锚点都指向真实存在的目标。")
    return 0


if __name__ == "__main__":
    sys.exit(main())
