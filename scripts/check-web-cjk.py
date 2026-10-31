#!/usr/bin/env python3
"""检查静态导出的 Web UI（web/out）的 HTML 里是否泄漏了韩文（한글）。

本仓库是 ARTEX 的简体中文发行版：中文是源语言，界面默认中文。上游分支用这个脚本
反过来守护“用户可见页面里没有中文”；汉化之后门禁方向随之翻转 —— 现在要守护的是
用户可见 HTML 里没有韩文残留（既防止贡献者硬编码韩文，也防止漏译回归）。

检查方式
--------
遍历 web/out 下所有 .html 文件的原始内容（不只是渲染后的标记，还包含内联脚本与
React Server Components 的 flight 数据），只要出现韩文字符就退出码 1。之所以看
原始内容，是因为用户可见文本会随 flight 数据（JSON 形态字符串）一起下发。

例外（有意保留）
---------------
语言切换器要把每种语言用其本族文字自称，因此 한국어（韩语）会出现在界面上。
这类刻意保留写进 ALLOWED_HANGUL，并注明理由；除此之外任何韩文都应被拦截。

不在检查范围内（诚实的局限）
---------------------------
本门禁只看静态导出结果，因此看不到 MOCK 模式（NEXT_PUBLIC_MOCK=1）运行时在
浏览器里渲染的内容：web/src/lib/mock/data.ts、handler.ts 与各页面的硬编码样例
会在 hydration 之后经 useEffect 进入 React 状态，只存在于运行时 JS bundle。
这条表面靠代码评审守护：渲染相关文件（web/src 的 .tsx、客户端数据）里残留的
韩文必须是下列之一——代码注释、被 t.has(...) 挡住实际不用的 i18n 兜底、线上协议
token、仅在 locale 为 ko 时显示的分支字面量，或语言自称名；否则就应译成中文。

范围：韩文音节（U+AC00–U+D7A3）、兼容字母、Hangul Jamo（现代 U+1100–U+11FF、
旧形 U+3130–U+318F、扩展 A U+A960–U+A97F、扩展 B U+D7B0–U+D7FF）。
只用标准库，不访问网络。
执行：python3 -I scripts/check-web-cjk.py [out 目录]
（默认取仓库根下的 web/out；CI 在 web 工作目录下用 -I ../scripts/check-web-cjk.py 调用。）
"""
import os
import re
import subprocess
import sys

# 韩文（Hangul）字符块。中文汉字·假名不在其中，因此不会误伤中文界面。
HANGUL = re.compile(
    r"[\u1100-\u11ff\u3130-\u318f\uac00-\ud7a3\ua960-\ua97f\ud7b0-\ud7ff]"
)

# 有意保留在界面上的韩文：语言切换器里各语言用本族文字自称（LOCALE_LABELS.ko = "한국어"）。
# 新增例外必须同时写明字符与理由，否则门禁保持严格。
ALLOWED_HANGUL: set = {"한", "국", "어"}


def find_out_dir() -> str:
    """确定要检查的 out 目录。

    给了路径参数就用它，否则用 git 仓库根目录下的 web/out。
    """
    if len(sys.argv) > 1:
        return os.path.abspath(sys.argv[1])
    root = subprocess.check_output(
        ["git", "rev-parse", "--show-toplevel"], text=True
    ).strip()
    return os.path.join(root, "web", "out")


def main() -> int:
    out_dir = find_out_dir()
    if not os.path.isdir(out_dir):
        print(
            f"错误：要检查的目录不存在 —— {out_dir}\n"
            "请先用 npm run build:static 生成静态导出，然后再运行。"
        )
        return 2

    checked = 0
    offenders = []  # （相对路径, 韩文字符数, 预览）
    for dirpath, _dirnames, filenames in os.walk(out_dir):
        for name in filenames:
            if not name.endswith(".html"):
                continue
            checked += 1
            path = os.path.join(dirpath, name)
            with open(path, encoding="utf-8", errors="ignore") as fh:
                text = fh.read()
            found = [ch for ch in HANGUL.findall(text) if ch not in ALLOWED_HANGUL]
            if found:
                rel = os.path.relpath(path, out_dir)
                kinds = "".join(sorted(set(found)))
                offenders.append((rel, len(found), kinds))

    print(f"已检查 HTML {checked} 个（目标：{out_dir}）")
    if offenders:
        print(f"韩文残留 {len(offenders)} 个文件 —— 用户可见 HTML 中存在韩文：")
        for rel, count, kinds in offenders:
            preview = kinds if len(kinds) <= 30 else kinds[:30] + "…"
            print(f"  out/{rel}：韩文 {count} 个 · 种类 {preview}")
        print(
            "\n请译为简体中文；若确实是必须保留的语言自称名等有意韩文，"
            "请在 scripts/check-web-cjk.py 的 ALLOWED_HANGUL 中连同理由一起添加。"
        )
        return 1
    if checked == 0:
        print("警告：没有检查到任何 HTML —— 请确认构建是否正常完成。")
        return 2
    print("韩文残留 0 —— 用户可见 HTML 中没有韩文（语言自称名除外）。")
    return 0


if __name__ == "__main__":
    sys.exit(main())
