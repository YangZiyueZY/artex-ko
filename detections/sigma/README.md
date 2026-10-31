# ARTEX 检测规则（Sigma / 主机·日志·SIEM）

中文 · [English](README.en.md) · [한국어](README.ko.md)

> 本目录把[防御·检测指南](../../docs/defense-zh.md)第 4 节“检测规则”中的伪规则，整理成可直接部署的 [Sigma](https://sigmahq.io) 规则。网络层由 [`../suricata/`](../suricata/) 负责。所有规则只能用于**防御·检测目的**，保护你自己拥有或已获得书面许可的系统。完整检测集合的总览见 **[../README.md](../README.md)**。

这是 ARTEX 检测集合中面向主机、日志与 SIEM 的一层。这些 [Sigma](https://sigmahq.io) 规则把防御指南（[中文版](../../docs/defense-zh.md)、[英文版](../../docs/defense-en.md)、[韩文版](../../docs/defense-ko.md)第 4 节）中的伪规则固化成与厂商无关的格式，再转换到你自己的 SIEM 或 EDR 查询语言。每个指标都以本仓库源码中核实过的字符串或行为为依据，而非推测。网络层在 [`../suricata/`](../suricata/) 下；完整检测集合 —— ATT&CK 覆盖层、指标 CSV / MISP 导出、主机分诊脚本 —— 的索引见 [`../README.md`](../README.md)。

## 原子规则

一条规则，一个可观测事实。可以单条转换，也可以作为整棵规则树的一部分转换。

- **[`artex_enrich_user_agent.yml`](artex_enrich_user_agent.yml)** —— *ARTEX 资产补强探测 User-Agent*。来自资产补强的入站 `artex-enrich/1.0` User-Agent(`enrich/enrich.go`)。目标侧，辅助指标。`level: high`。
- **[`artex_selfupdate_egress.yml`](artex_selfupdate_egress.yml)** —— *ARTEX 自动更新出站 User-Agent*。来自自动更新流程的出站 `artex-selfupdate` User-Agent(`selfupdate/github.go`)。主机/取证出站指标。`level: medium`。
- **[`artex_guard_audit_framing.yml`](artex_guard_audit_framing.yml)** —— *ARTEX 平台护栏审计日志措辞*。工具调用被拦截时写入审计日志的平台护栏管控标记(`guard/guard.go`)。主机/取证指标。`level: high`。
- **[`artex_recording_proxy_ca.yml`](artex_recording_proxy_ca.yml)** —— *ARTEX 记录代理 MITM CA 证书工件*。记录代理在 `_ca/mitmproxy-ca-cert.pem` 布局下创建 MITM CA 文件(`traffic/traffic.go`)。主机/取证工件；该文件名与独立 mitmproxy 共用，因此属于狩猎线索。`level: medium`。
- **[`destructive_command_hunting.yml`](destructive_command_hunting.yml)** —— *破坏性命令执行（ARTEX 护栏名单狩猎）*。镜像 ARTEX 护栏内置拦截名单的破坏性 shell/DB 命令(`db/db.go` 种子)。通用狩猎线索，**不是** ARTEX 签名。`level: medium`。

## 关联规则（基于行为） —— [`correlation/`](correlation/)

静态字符串可以改，行为更难藏。这些 Sigma **关联**规则编码了防御指南中基于行为的那一层（4.1–4.2 节与 4.4 节）。每条都按 `id` 引用上面的原子规则，所以**要转换整棵 `sigma/` 树，而不是单条关联文件**，否则引用无法解析（[Sigma 测试](../tests/sigma/)断言的正是这种依赖关系）。

- **[`correlation/artex_enrich_scan_velocity.yml`](correlation/artex_enrich_scan_velocity.yml)** —— *补强扫描速率*。短窗口内同一来源大量 `artex-enrich/1.0` 探测（补强以并发 4 运行、没有速率限制）—— 单请求规则漏掉的速度维度。`event_count`,`level: high`。
- **[`correlation/artex_enrich_fanout.yml`](correlation/artex_enrich_fanout.yml)** —— *补强扇出*。同一来源携带补强 User-Agent 打到大量**不同**主机：机器级速度地横扫资产清单，其中“不同主机数”而不是请求量才是关键。`value_count`,`level: high`。
- **[`correlation/artex_guard_block_burst.yml`](correlation/artex_guard_block_burst.yml)** —— *护栏拦截突发*。同一主机上反复出现平台护栏管控标记 —— 是活跃运行的 ARTEX 触发了自己的护栏，而非某份文档只是引用了该标记。`event_count`,`level: high`。
- **[`correlation/artex_guard_marker_then_destructive.yml`](correlation/artex_guard_marker_then_destructive.yml)** —— *护栏标记伴随破坏性命令*。护栏标记与破坏性命令在窗口内于同一主机共现（防御指南 4.2 节，多阶段）：把 ARTEX 专属标记与本来通用的破坏性命令信号结合，从而提高特异性。`temporal`,`level: high`。

阈值与窗口都是保守默认值 —— 请按你的基线调优。纯 Web 的多阶段情形（枚举 → 试探 → 认证）仍需环境专属的基础规则，因为该模式无法归约为某个 ARTEX 独有 User-Agent；可作为起点的通用行为基模板在[防御指南 4.2 节](../../docs/defense-zh.md)中，它刻意不在本受测规则树里，因为无法以 ARTEX 源码为依据。

## 范围与如实说明 —— 部署前必读

- **静态指标可以被改掉。** 运营者可以设置不同的 User-Agent 或清掉 CA 文件，因此某个原子指标没有出现**不代表**安全。经得起时间的是 `correlation/` 规则所依据的行为 —— 同一来源把侦察 → 枚举 → 试探 → 认证/注入尝试串成一条链，随响应调整，毫不停歇地跑。
- **破坏性命令规则属于通用狩猎。** 它镜像 ARTEX 的护栏拦截名单，但同样的命令正当管理员也会执行。命中只当线索，先给你的环境配置允许名单，不要单凭它把事件归属到 ARTEX。
- **端口与 schema 属于主机取证，不是 Sigma。** 服务器默认 `:8787`、记录代理 `127.0.0.1:8788`(`cmd/artex/main.go`)以及 PostgreSQL 探索图谱 schema，最好在可疑主机上检查，因此它们出现在[指标 CSV](../indicators/)与[主机分诊脚本](../triage/)里，而不是作为噪声很大的规则发布。
- **`logsource` 与字段名是通用的。** 规则使用通用的 `category`/`product` 日志源与字段名(`cs-user-agent`、`CommandLine`、`TargetFilename`)。转换时请用处理管线(`-p`)把它们映射到你产品的 schema；详见下面的后端说明。

## 验证与转换

用 [sigma-cli](https://github.com/SigmaHQ/sigma-cli)(pySigma)验证。在仓库根目录执行：

```sh
python3 -m venv .venv && . .venv/bin/activate
pip install sigma-cli

# 结构 + 最佳实践校验(期望:0 错误、0 问题)
sigma check detections/sigma/

# 加载本规则集记录的基线,跑完整 SigmaHQ 规范集(期望:0 问题)
pip install pySigma-validators-sigmahq
sigma check --validation-config detections/tests/sigma_lint/validators.yml detections/sigma/

# 转换整棵树,以便关联规则能解析它们按 id 引用的原子规则
sigma plugin install splunk
sigma convert -t splunk --without-pipeline detections/sigma/
```

各后端对关联的支持不同，所以 `-t` 的选择很关键：Splunk、Elasticsearch EQL、Grafana Loki 能转换整棵树，而 Elasticsearch Lucene、OpenSearch 与 Microsoft `kusto` 后端只能转换五条原子规则（窗口需在产品内自行表达）。实测的各后端支持矩阵、`--without-pipeline` / `-p` 的字段映射说明见 [`../README.md`](../README.md)，并由 [`../tests/sigma_backends/`](../tests/sigma_backends/) 复现。

## 测试

[`../tests/`](../tests/) 下有四个覆盖本规则的可复现测试套件，每个都只需要 Docker:[`sigma/`](../tests/sigma/)（校验、整树编译，以及关联规则单独转换必然失败）、[`sigma_match/`](../tests/sigma_match/)（规则在恶意样本上确实触发、在正常样本上保持安静）、[`sigma_backends/`](../tests/sigma_backends/)（五个后端的可移植性）、[`sigma_lint/`](../tests/sigma_lint/)（完整 SigmaHQ 校验器基线，0 问题）。参见 [`../tests/README.md`](../tests/README.md)。

## 贡献

欢迎检测贡献。新规则应让每个指标都以可观测事实为依据，在 `description` 中写明局限，干净地通过 SigmaHQ 校验器基线(`sigma check --validation-config ../tests/sigma_lint/validators.yml .`)，并避免任何读起来像攻击指引的内容。参见 [`../../CONTRIBUTING.md`](../../CONTRIBUTING.md) 与 [`../suricata/`](../suricata/) / [`../README.md`](../README.md) 中的网络层。
