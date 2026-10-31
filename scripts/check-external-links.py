#!/usr/bin/env python3
"""检查受跟踪的 Markdown 文档所指向的外部链接（http·https）是否仍然有效。

姊妹脚本 `check-doc-links.py` 只检查仓库内部链接·锚点，对外部 URL
按设计跳过（因为依赖网络、容易 flaky，所以不放进合并门禁）。本脚本
负责那项“另行检查”。README 顶部·防御指南·detections README 引导
访客和防御者“去这里看看”的外部链接（事件报告入口 boho.or.kr·
privacy.go.kr·pipc.go.kr·fsec.or.kr，CISA KEV，OWASP·SigmaHQ·Suricata·MITRE·MISP，
原始演示 artex-demo.vercel.app，GitHub 徽章等）一旦变质、迁移、关闭，
就会悄悄保持损坏状态，本脚本定期·手动地把它们揪出来。

选择检查 URL 的规则
- 遍历所有受跟踪的 `.md`，但跳过 fenced code block（``` 或 ~~~）和行内代码跨度
  （`` `...` ``）中的 URL。其中的 URL 是命令示例、配置值、引用的外部
  资产（例如旁支问题验证文档中的 `id.redhaze.top`），不是“读者会跟随的引用链接”。
- 从 Markdown 链接·图片（`[text](url)`·`![alt](url)`）、自动链接（`<url>`）以及剩余
  正文中的裸 URL 中收集 http·https 地址。
- 排除保留·占位主机：localhost、私有/回环 IP（127.·10.·192.168.·
  169.254.·172.16~31.·0.0.0.0·::1），RFC 2606/6761 保留域（example.com/org/net/edu·
  `*.example.*`·`.test`·`.invalid`·`.local`·`.tld`），以及没有点、不是 FQDN 的名称（例如 `target`）。

如何确认存活（把 MAINTAINING.md 8.2 的教训写进代码）
国内公共·安全机构网站会拒绝 HEAD 请求和默认 User-Agent，或多次
重定向，因此简单检查会把正常链接误判为损坏。所以本脚本
**用浏览器 User-Agent、以 GET、跟随重定向**确认。状态分为三种。
- OK：最终状态为 2xx·3xx。链接有效。
- RESTRICTED：401·403·405·429。主机仍存活，只是确认方式被服务器访问策略（机器人拦截、
  方法拒绝、限速）挡住，并非损坏链接。会报告但不计为失败。
- DOWN：404·410·5xx（重试后仍然）·DNS/连接/超时/SSL 错误（重试后仍然）。很可能
  确实坏了。

网络错误·5xx·429 会稍作延迟后重试，以区分临时抖动和真实故障。
只使用标准库。本脚本仅对仓库**公开文档已经指向的**引用 URL 发送
GET 以确认存活，不扫描·探测任何目标。

已知例外（allowlist）
`scripts/external-links-allowlist.txt` 中列出的 URL 若结果为 DOWN，会单独
归类为 "ALLOWED"，不计入 strict 退出码。把我们不拥有、无法修复、由上游原文保留
文件（例如上游 CHANGELOG.zh.md）继承的死链透明地记录下来，使周期性 strict 检查
不会因那一个而永久变红，而**只在出现新损坏链接时**变红。

执行
- 默认（报告·退出码 0）：`python3 -I scripts/check-external-links.py`
- 非合并门禁的周期/发布检查（有新损坏链接时变红）：`--strict`（allowlist 之外
  只要有一个 DOWN 就以退出码 1 结束）。RESTRICTED·ALLOWED 在 strict 下也不计为失败。
- 不联网、只预览提取集合：`--list`（不发起调用，输出检查目标 URL 及出处）。
"""
import argparse
import http.client
import os
import re
import socket
import ssl
import subprocess
import sys
import time
import urllib.error
import urllib.request
from collections import defaultdict
from urllib.parse import urlsplit

# [text](url) 正文链接（非图片）和 ![alt](url) 图片。路径截到空格·右括号为止。
MD_LINK = re.compile(r"(?<!\!)\[[^\]]*\]\(([^)\s]+)")
MD_IMAGE = re.compile(r"!\[[^\]]*\]\(([^)\s]+)")
# <https://...> 自动链接。
AUTOLINK = re.compile(r"<(https?://[^>\s]+)>")
# 正文中的裸 URL。末尾标点会在后面剥掉。
BARE_URL = re.compile(r"https?://[^\s)>\]\"'`]+")
# 行内代码跨度 `...`（单反引号）。提取前用空格抹掉。
INLINE_CODE = re.compile(r"`[^`]*`")
# 正文 URL 末尾常见的标点。
TRAILING_PUNCT = ".,;:!?\"'»)]}>"

# IPv4 私有/回环/链路本地网段（172.16~31. 另有模式）。
PRIVATE_IPV4 = re.compile(r"^(127\.|10\.|192\.168\.|169\.254\.|0\.0\.0\.0$)")
PRIVATE_IPV4_172 = re.compile(r"^172\.(1[6-9]|2\d|3[01])\.")

BROWSER_UA = (
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) "
    "AppleWebKit/537.36 (KHTML, like Gecko) Chrome/140.0 Safari/537.36"
)

# 主机存活但确认方式被服务器策略挡住的状态码（并非损坏）。
RESTRICTED_CODES = {401, 403, 405, 429}

ALLOWLIST_PATH = os.path.join("scripts", "external-links-allowlist.txt")


def load_allowlist(root: str) -> set:
    """读取我们无法修复的已知 DOWN URL 集合（没有则为空集合）。

    每行一个 URL。`#` 之后是注释，空行·纯注释行会忽略。
    """
    path = os.path.join(root, ALLOWLIST_PATH)
    allow = set()
    if not os.path.exists(path):
        return allow
    with open(path, encoding="utf-8") as fh:
        for line in fh:
            line = line.split("#", 1)[0].strip()
            if line:
                allow.add(line)
    return allow


def strip_code(line: str) -> str:
    """删除 fence 之外一行中的行内代码跨度（其中的 URL 不是引用）。"""
    return INLINE_CODE.sub(" ", line)


def is_checkable(url: str) -> bool:
    """过滤掉保留·占位主机，只保留真正值得检查的外部 URL。"""
    parts = urlsplit(url)
    if parts.scheme not in ("http", "https"):
        return False
    host = parts.hostname
    if not host:
        return False
    host = host.lower()
    if host == "localhost" or host.endswith(".localhost"):
        return False
    if host == "::1":
        return False
    if PRIVATE_IPV4.match(host) or PRIVATE_IPV4_172.match(host):
        return False
    if host in ("example.com", "example.org", "example.net", "example.edu"):
        return False
    if host.endswith((".example.com", ".example.org", ".example.net", ".example")):
        return False
    if host.endswith((".test", ".invalid", ".local", ".localdomain", ".tld")):
        return False
    if "." not in host:  # 不是 FQDN 的裸名称（例如 target）
        return False
    return True


def extract(root: str, md_files: list) -> dict:
    """从受跟踪的 .md 生成“待检查外部 URL → [(文件, 行号), ...]”字典。"""
    found = defaultdict(list)
    for md in md_files:
        with open(os.path.join(root, md), encoding="utf-8", errors="ignore") as fh:
            lines = fh.readlines()
        in_fence = False
        for lineno, raw in enumerate(lines, 1):
            stripped = raw.lstrip()
            if stripped.startswith("```") or stripped.startswith("~~~"):
                in_fence = not in_fence
                continue
            if in_fence:
                continue
            line = strip_code(raw)
            candidates = []
            for pattern in (MD_LINK, MD_IMAGE, AUTOLINK):
                candidates.extend(m.group(1) for m in pattern.finditer(line))
            for m in BARE_URL.finditer(line):
                candidates.append(m.group(0).rstrip(TRAILING_PUNCT))
            for url in candidates:
                url = url.strip().rstrip(TRAILING_PUNCT)
                if is_checkable(url):
                    where = (md, lineno)
                    if where not in found[url]:
                        found[url].append(where)
    return found


def probe(url: str, timeout: float, retries: int, delay: float):
    """用浏览器 UA 对 URL 发送 GET，返回（分类, 详情）。

    分类为 "OK" · "RESTRICTED" · "DOWN"。网络错误·5xx·429 会重试。
    """
    req = urllib.request.Request(
        url,
        method="GET",
        headers={
            "User-Agent": BROWSER_UA,
            "Accept": "*/*",
            "Accept-Language": "ko,en;q=0.8",
        },
    )
    last = ""
    for attempt in range(retries + 1):
        try:
            with urllib.request.urlopen(req, timeout=timeout) as resp:
                code = resp.getcode()
                if code in RESTRICTED_CODES:
                    return "RESTRICTED", f"HTTP {code}"
                return "OK", f"HTTP {code}"
        except urllib.error.HTTPError as exc:
            code = exc.code
            if code in RESTRICTED_CODES:
                return "RESTRICTED", f"HTTP {code}"
            if code >= 500 or code == 408:
                last = f"HTTP {code}"  # 可能是暂时性的，重试
            else:
                return "DOWN", f"HTTP {code}"  # 404·410 等为确定结果，无需重试
        except (
            urllib.error.URLError,
            http.client.HTTPException,
            ssl.SSLError,
            socket.timeout,
            ConnectionError,
            TimeoutError,
            OSError,
        ) as exc:
            reason = getattr(exc, "reason", exc)
            last = f"{type(exc).__name__}: {reason}"
        if attempt < retries:
            time.sleep(delay * (attempt + 1))
    return "DOWN", last


def main() -> int:
    ap = argparse.ArgumentParser(description="受跟踪 Markdown 的外部链接存活检查")
    ap.add_argument("--strict", action="store_true",
                    help="只要有 DOWN 就返回退出码 1（用于周期/发布检查）")
    ap.add_argument("--list", action="store_true",
                    help="不发起网络调用，只输出检查目标 URL 和出处")
    ap.add_argument("--timeout", type=float, default=15.0, help="请求超时（秒）")
    ap.add_argument("--retries", type=int, default=2, help="网络错误·5xx·429 的重试次数")
    ap.add_argument("--delay", type=float, default=0.5, help="调用之间·重试之间的默认延迟（秒）")
    args = ap.parse_args()

    root = subprocess.check_output(
        ["git", "rev-parse", "--show-toplevel"], text=True
    ).strip()
    tracked = subprocess.check_output(["git", "ls-files"], cwd=root, text=True).splitlines()
    md_files = [f for f in tracked if f.endswith(".md")]

    allow = load_allowlist(root)
    found = extract(root, md_files)
    urls = sorted(found)
    print(f"从受跟踪 Markdown {len(md_files)} 个中收集到待检查外部 URL {len(urls)} 个")

    if args.list:
        for url in urls:
            where = ", ".join(f"{f}:{ln}" for f, ln in found[url])
            tag = " [allowlist]" if url in allow else ""
            print(f"  {url}{tag}  ({where})")
        return 0

    results = {"OK": [], "RESTRICTED": [], "ALLOWED": [], "DOWN": []}
    for i, url in enumerate(urls):
        if i:
            time.sleep(args.delay)  # 调用之间稍作延迟（规避沙箱限流）
        verdict, detail = probe(url, args.timeout, args.retries, args.delay)
        if verdict == "DOWN" and url in allow:
            verdict = "ALLOWED"  # 无法修复的已知上游继承 DOWN
        results[verdict].append((url, detail))
        print(f"  [{verdict:10}] {url} — {detail}")

    print(
        f"\n摘要：OK {len(results['OK'])} · RESTRICTED {len(results['RESTRICTED'])} · "
        f"ALLOWED {len(results['ALLOWED'])} · DOWN {len(results['DOWN'])}"
    )
    if results["RESTRICTED"]:
        print("RESTRICTED（主机存活·确认方式被挡，并非损坏链接）：")
        for url, detail in results["RESTRICTED"]:
            print(f"  {url} — {detail}")
    if results["ALLOWED"]:
        print("ALLOWED（allowlist 中记录的已知 DOWN —— 我们无法修复，排除在 strict 之外）：")
        for url, detail in results["ALLOWED"]:
            where = ", ".join(f"{f}:{ln}" for f, ln in found[url])
            print(f"  {url} — {detail}  （引用：{where}）")
    if results["DOWN"]:
        print("DOWN（很可能已损坏 —— 需要确认）：")
        for url, detail in results["DOWN"]:
            where = ", ".join(f"{f}:{ln}" for f, ln in found[url])
            print(f"  {url} — {detail}  （引用：{where}）")

    if args.strict and results["DOWN"]:
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
