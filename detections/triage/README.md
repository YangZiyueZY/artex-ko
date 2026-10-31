# ARTEX 主机分诊（triage）

中文 · [English](README.en.md) · [한국어](README.ko.md)

> [`artex_host_triage.py`](artex_host_triage.py) 是一个**在单台可疑主机上直接运行**的只读分诊脚本。它不面向使用 SIEM(Sigma)·网络传感器（Suricata）·威胁情报平台（指标 CSV·MISP）的防御者，而面向另一类响应者：站在可疑主机的 shell 前、没有 SIEM 的人。脚本依据仓库源码检查监听端口·记录代理工件·日志标记·PostgreSQL schema，并给每条发现附上同样的局限说明（端口可以改动，CA 文件名与独立 mitmproxy 共用等）。请只在你自己拥有或已获得书面许可的主机上使用。

这是一个只读分诊助手，**在单台可疑主机上**运行，用本机状态回答“这台机器上跑过 ARTEX 吗”。本目录其余部分服务于运行 SIEM([Sigma](../sigma/))、网络传感器（[Suricata](../suricata/)）或威胁情报平台（[指标](../indicators/)）的防御者；这个脚本服务于另一种响应者：站在主机 shell 前、没有 SIEM，需要用本机上的东西快速给出可辩护答案的人。

它把本目录其余部分发布的同一批指纹落到实处，**并额外覆盖指标清单刻意不带 Sigma 规则的三个主机/DB 指标**(`server-listen-port`、`recording-proxy-endpoint`、`postgres-exploration-schema`)——它们无法从日志或网络观测，只能在主机本身上检查。

## 检查内容

每一项检查都以本仓库源码中核实过的字符串或路径为依据，每条发现都带有与对应 Sigma 规则或指标行相同的如实提示。

- **监听端口** —— `:8787`（管理 UI）与 `127.0.0.1:8788`（记录代理），即 [`cmd/artex/main.go`](../../cmd/artex/main.go) 中 `--addr` / `--proxy` 标志的默认值。活动主机上从 `ss`/`netstat`/`lsof` 解析，或从你用 `--ports-from` 传入的文件读取。
- **记录代理工件** —— 记录器首次启动时写出的 MITM CA,`<data-dir>/traffic/_ca/mitmproxy-ca-cert.pem`，以及同级的 `_index/index.sqlite` 与 `_blobs/`([`traffic/traffic.go`](../../traffic/traffic.go)；数据目录默认为二进制旁的 `data/`)。该 CA 是实施中间人流量记录器的信任锚（ATT&CK T1557）。
- **日志标记** —— 补强探测器 UA `artex-enrich/1.0`([`enrich/enrich.go`](../../enrich/enrich.go))、自动更新出站 UA `artex-selfupdate`([`selfupdate/github.go`](../../selfupdate/github.go))，以及平台护栏审计标记([`guard/guard.go`](../../guard/guard.go)；原样保留，包括非 ASCII 的边框文字，以便 grep 命中)在你指向的日志文件里。压缩为 `.gz`/`.bz2`/`.xz` 的轮转日志也会解压后一并扫描，从而覆盖主机的日志历史；没有标准库解码器的格式(`.zst`/`.lz4`)会报告为**已跳过**，而不是被悄悄当成干净 —— 请先解压或手工 `grep`。
- **PostgreSQL 探索图谱 schema** —— ARTEX 存储中的双图谱表(`exploration_nodes`/`_edges`/`_anchors`，连同 `assets`/`companies`/`activity` 与 `agent_prompts` 种子)([`db/schema.sql`](../../db/schema.sql))。有 `psql` 时可对 DSN 执行；否则脚本会打印出确切的只读查询供你手工运行。
- **进程环境注入** —— 某个运行中的进程的环境里同时带有代理变量(`HTTP_PROXY` / `HTTPS_PROXY` / `ALL_PROXY`)**与**指向 `mitmproxy-ca-cert.pem` 的工具链 CA 信任变量(`SSL_CERT_FILE` / `CURL_CA_BUNDLE` / `REQUESTS_CA_BUNDLE` / `GIT_SSL_CAINFO` / `NODE_EXTRA_CA_CERTS`)。ARTEX 正是把这些变量注入它拉起的每个 worker 工具([`agent/worker.go`](../../agent/worker.go) 的 `proxyEnv`，由 [`agent/proxyenv_test.go`](../../agent/proxyenv_test.go) 断言)。这些**变量名硬编码在源码里**（只有值可配置），因此即使运营者改名二进制或换端口，这一线索依然成立——比单纯的监听端口更强的信号。活动 Linux 主机上从 `/proc` 读取，或用 `--proc-from` 传入捕获的转储。代理与 mitmproxy CA 同时存在报高风险；只有 mitmproxy CA，或只有 ARTEX 默认代理端点(`127.0.0.1:8788`)，报中风险；企业代理而没有 mitmproxy CA 则刻意不报。

命中只是**分诊线索，不是归属结论**；而所有发现都不存在，也**不代表**健康证明：运营者可以改名二进制、挪走数据目录或更改端口。

## 平台支持

脚本是纯 Python 3（只用标准库），因此凡是有 Python 3 的地方都能跑 —— 已在 Linux（CI 自测）与 macOS 上验证。两项检查与操作系统相关，两项都会优雅降级而不是报错：

- **活动端口扫描** —— 依次尝试 `ss`、`netstat`、`lsof`，使用第一个有输出的。Linux 上是 `ss`/`netstat`；在 macOS/BSD 上 `ss` 不存在、`netstat` 不接受 Linux 的 `-ltnp` 参数（退出且输出为空），于是落到 `lsof -nP -iTCP -sTCP:LISTEN`，用同样的方式解析。可用 `--ports-from` 读取已保存的列表以代替实时扫描。
- **活动进程环境扫描** —— 读取 `/proc`，因此只在 Linux 上运行。没有 `/proc` 的主机（macOS/BSD）会报告为**已跳过**，而不是干净；请在 Linux 主机上捕获转储并用 `--proc-from` 传入（见“用法”）。

其余检查 —— 记录代理工件、日志标记、PostgreSQL schema —— 读取文件系统、日志文件以及（仅在你传入 DSN 时）`psql`，因此与操作系统无关。

## 用法

```sh
# 端到端检查一台主机
detections/triage/artex_host_triage.py \
    --data-dir /opt/artex/data \
    --log /var/log/syslog --log-dir /var/log/artex \
    --pg-dsn "$ARTEX_PG_DSN"

# 机器可读的发现,并在有任何命中时以非零码退出
detections/triage/artex_host_triage.py --data-dir /opt/artex/data --json --exit-code

# 离线/取证镜像:读取已捕获的进程环境转储
#   在主机上用以下命令制作转储:
#   for p in /proc/[0-9]*; do echo "# $p"; tr '\0' '\n' < "$p/environ"; echo; done > proc_env_dump.txt
detections/triage/artex_host_triage.py --proc-from proc_env_dump.txt

# 可复现的夹具测试(不触碰任何主机状态)
detections/triage/artex_host_triage.py --self-test
```

脚本只用 Python 3 标准库：无需安装、无需联网，除 `--self-test` 自己的临时目录外不写任何东西。它读取主机状态（开放端口、数据目录、日志文件，以及仅在你传入 DSN 时的数据库）并打印发现结果。默认退出码为 `0`（分诊，不是门禁）；传入 `--exit-code` 时，有任何指标命中则以 `1` 退出。

## 如何保持如实

`--self-test` 会构造一台合成主机 —— 带预埋 CA、索引与 blob 存储的数据目录；包含每个标记的日志；端口列表；以及捕获的进程环境转储 —— 并断言每项检查都会在其上命中，随后断言干净主机、正常日志与企业代理进程产生**零**发现（无误报）。它已接入 [`detections` CI 工作流](../../.github/workflows/detections.yml)，并由 [`detections/tests/run-all.sh`](../tests/run-all.sh) 重跑，因此任何破坏检查的改动，或让指标偏离它所 grep 的源码字符串的改动，都会在合并门禁处失败。跑不起来的检测只是声明。
