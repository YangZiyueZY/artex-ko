# ARTEX 指标（机器可读）

中文 · [English](README.en.md) · [한국어](README.ko.md)

> [`artex_indicators.csv`](artex_indicators.csv) 把 ARTEX 实际发出的独有指纹（入侵指标，IoC）汇总到一个文件里。它采用机器可读的 CSV 形式，可直接放进威胁情报平台·SIEM 查询表·主机分诊检查表。所有取值都是在本仓库源码中核实过的字符串，每一行都同时记录来源文件与建立在它之上的检测规则。背景说明见[防御·检测指南](../../docs/defense-zh.md)第 2 节“防御者可观测的指纹”。只能用于**防御·检测目的**，保护你自己拥有或已获得书面许可的系统。

一份机器可读的清单，收录 ARTEX 自身发出的独有指纹，面向需要原子指标而不是检测逻辑的防御者：把 [`artex_indicators.csv`](artex_indicators.csv) 直接放进威胁情报平台、SIEM 查询表或主机分诊检查表即可。每个取值都是在本仓库源码中核实过的字符串 —— 与[检测规则](../README.md)和防御指南（[中文版](../../docs/defense-zh.md)、[英文版](../../docs/defense-en.md)第 2 节）所依赖的同一套依据 —— 并且每一行都记录它来自哪里、建立在它之上的规则（如果有）。

## 列说明

- **`id`** —— 该指标的稳定 slug。
- **`type`** —— 指标种类：`http.user-agent`、`string`（要在日志/文件中狩猎的字面量）、`port`、`ip-dst|port`，或 `other`（不属于以上任何一类的主机工件，例如数据库 schema 对象名）。它们映射到等价的 MISP/STIX 属性类型。
- **`value`** —— 指标本身，逐字保留，包括非 ASCII 的护栏标记。
- **`perspective`** —— `target`（可在**指向**被 ARTEX 探测系统的流量中观测）或 `forensic`（可在 ARTEX 运行过、或被用作经转的主机**上**观测）。防御指南刻意区分两者；混用会得出错误结论。
- **`source`** —— 发出该值的仓库内相对源文件，多个用 `;` 分隔。这就是依据所在：如果上游再同步改动了发送方，这里的指标必须随之变化。
- **`rule`** —— 建立在确切取值之上的检测规则，多个用 `;` 分隔；对于直接分诊、而不作为（噪声很大的）规则发布的主机取证指标，该列为空。
- **`description`** —— 一行说明，包含适用时的如实提示。

## MISP 事件导出

同一批指标还以可直接导入的 [MISP](https://www.misp-project.org/) 事件形式提供：[`artex_indicators.misp.json`](artex_indicators.misp.json)。运行 MISP 实例（或能摄入 MISP 格式的威胁情报平台）的防御者可以直接导入这些指纹，不必手工映射 CSV 列。之后借助 MISP 自带的转换器，导出到 STIX 2.1 也只有一步之遥，因此本仓库不必再造一个会丢信息的第二格式。

- **类型映射。** 每个 CSV `type` 转成等价的 MISP 属性类型：`http.user-agent` → `user-agent`；护栏标记 `string` → `pattern-in-file`（分类 *Artifacts dropped*）;`port` → `port`;`ip-dst|port` → `ip-dst|port`(复合值采用 MISP 的 `ip|port` 形式，因此 `127.0.0.1:8788` 存为 `127.0.0.1|8788`)；探索图谱 schema 指纹 `other` → `other`（分类 *Other*）。
- **`to_ids` 如实跟随 `rule` 列。** 有检测规则建立在之上的行属于可处置指标，标记为 `to_ids: true`。没有规则的主机取证行 —— 默认监听端口、回环代理端点、探索图谱 schema 指纹 —— 属于分诊线索而非处置型 IoC，因此 `to_ids: false` 并设 `disable_correlation: true`(一个常见端口、`127.0.0.1` 或通用表名不该污染 MISP 的关联)。这与 CSV 的 `rule` 列以及下面的提示所表达的区分完全一致。
- **导入方式。** 用 [pymisp](https://github.com/MISP/PyMISP) 执行 `MISPEvent().load_file("artex_indicators.misp.json")`，或使用 *Add event → Populate from … → MISP format* 界面，或调用 REST API。事件为未发布状态并带 `tlp:clear` 标签；导入时请按你的实例需要设置分发与发布状态。

## 如何如实阅读本清单

- **这些是可变的指纹，不是安全的证明。** 运营者可以换 User-Agent 或改默认端口，因此这里任何取值**没有出现**都**不代表** ARTEX 不存在。经得起时间的是行为 —— 见关联规则以及防御指南第 1、2 节与 4.1–4.2 节。
- **通用狩猎线索被刻意排除。** 破坏性 shell/DB 命令(`rm -rf`、`DROP DATABASE` 等)**不是** ARTEX 指纹 —— 正当管理员也会执行。它们是狩猎线索而非可导入的指标，因此放在 [`destructive_command_hunting.yml`](../sigma/destructive_command_hunting.yml) 与防御指南里，不放进本清单。把它们作为处置型指标导入会造成误报。
- **norma WebFetch User-Agent 是线上签名，不是原子指标。** worker 的页面抓取工具在攻击阶段发送 `norma/0.4`,Suricata sid 1000003 对 `norma/` 前缀触发，但该字符串是 norma SDK 自身硬编码的 User-Agent(`github.com/Autumn-27/norma/tool/webfetch.go`)，所有基于 norma 的工具共用，并非 ARTEX 独有指纹。若把它作为处置型指标导入，就会对所有 norma-SDK 流量告警 —— 与破坏性命令落入的误报陷阱相同 —— 因此它被刻意排除在本清单外，只作为网络规则发布([`../suricata/README.md`](../suricata/README.md),sid 1000003)。另外它依据的是固定依赖(`github.com/Autumn-27/norma`)，而不是本仓库自身源码，所以下面的来源一致性测试无法像重读 ARTEX 自己的发送方那样重读它。
- **主机取证端口用于分诊，不用于拦截。** `:8787` 与 `127.0.0.1:8788` 描述的是可能运行 ARTEX 的主机；用 `ss`/`netstat` 检查，不要盲目上防火墙。
- **探索图谱 schema 指纹用于检视数据库，不是网络/文件 IoC。** `exploration_nodes` 表是 ARTEX 保存在 PostgreSQL 中探索图谱的核心。不要凭单次命中下结论；要确认同级表(`exploration_edges`、`exploration_anchors`、`assets`、`companies`、`activity`)与 `agent_prompts` 种子在同一个数据库里。运营者可以重命名或删除表，所以没有出现并不代表安全。
- **没有 `rule` 的主机/DB 行也有执行者。** 三个直接分诊、而不作为 Sigma 规则发布的指标 —— 监听端口、记录代理端点、该 schema 指纹 —— 都由[主机分诊脚本](../triage/)在可疑主机上检查，因此只有 shell 访问、没有 SIEM 的响应者不必手工跑 `ss`/`netstat`/`psql`。

## 验证

清单由[指标来源一致性测试](../tests/indicators/run.sh)覆盖：它重读本 CSV，逐行断言取值仍存在于所引源文件、仍固定在被引规则中，并断言测试所依据的每个指标都出现在清单里。若某行偏离来源，或已知指纹被从清单中删除，测试即失败。运行方式：

```sh
detections/tests/indicators/run.sh
```

MISP 事件由独立的 [MISP 导出一致性测试](../tests/misp/run.sh)覆盖：它在 pymisp 下加载事件（从而保证每个属性类型都是 MISP 服务器真正接受的类型），并断言它与本 CSV 逐行同步 —— 取值相同、类型/分类符合意图、`to_ids` 标志与 `rule` 列一致。事件与 CSV 一同手工维护 —— 它还带有 CSV 里没有的逐属性整理注释、稳定 UUID 与事件级标签，所以不存在会用以丢失信息的默认值覆盖它们的生成器。当你新增、删除或重设某行 CSV 的类型时，请在同一次提交里同步修改 [`artex_indicators.misp.json`](artex_indicators.misp.json)(为新属性提供新的 `uuid` 与作为依据的 `comment`)；在两者一致之前该测试会一直失败，因此这种更新不会被悄悄忘记。运行方式：

```sh
detections/tests/misp/run.sh
```
