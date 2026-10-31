# ARTEX 检测规则

中文 · [English](README.en.md) · [한국어](README.ko.md)

> 本目录把[防御·检测指南](../docs/defense-zh.md)第 4 节“检测规则”中的伪规则，落实为可部署的 [Sigma](https://sigmahq.io) 规则，转成与厂商无关的格式后即可转换成你自己的 SIEM 或 EDR 查询语言。这里的每个指标都依据本仓库源码中核实过的字符串或行为，不是推测。所有规则只能用于**防御·检测目的**，保护你自己拥有或已获得书面许可的系统。

这是整套 ARTEX 检测集合的索引：把防御指南（[中文版](../docs/defense-zh.md) · [英文版](../docs/defense-en.md) · [韩文版](../docs/defense-ko.md)）第 4 节的伪规则固化成的 [Sigma](https://sigmahq.io) 规则、网络层 [Suricata](https://suricata.io) 规则、[ATT&CK 覆盖层](attack/)、机器可读的[指标清单](indicators/)与 MISP 导出，以及用于没有 SIEM 场景的[主机分诊脚本](triage/)。

## 原子规则

- **`sigma/artex_enrich_user_agent.yml`** —— 来自 ARTEX 资产补强的入站 `artex-enrich/1.0` User-Agent(`enrich/enrich.go`)。目标侧，辅助指标。`level: high`。
- **`sigma/artex_selfupdate_egress.yml`** —— 来自自动更新流程的出站 `artex-selfupdate` User-Agent(`selfupdate/github.go`)。主机/取证出站指标。`level: medium`。
- **`sigma/artex_guard_audit_framing.yml`** —— 工具调用被拦截时写入审计日志的平台护栏管控标记(`guard/guard.go`)。主机/取证指标。`level: high`。
- **`sigma/destructive_command_hunting.yml`** —— 镜像 ARTEX 护栏内置拦截名单的破坏性 shell/DB 命令(`db/db.go` 种子)。通用狩猎线索，不是 ARTEX 签名。`level: medium`。
- **`sigma/artex_recording_proxy_ca.yml`** —— 在 `_ca/mitmproxy-ca-cert.pem` 布局下创建记录代理 MITM CA 文件(`traffic/traffic.go`)。主机/取证工件；该文件名与独立 mitmproxy 共用，因此属于狩猎线索。`level: medium`。

## 关联规则（基于行为）

静态字符串可以改，行为更难藏。位于 [`sigma/correlation/`](sigma/correlation/) 的这些 Sigma **关联**规则编码了防御指南中基于行为的那一层（4.1–4.2 节与 4.4 节）。每条都按 `id` 引用上面的原子规则，因此要把**整棵 `sigma/` 树**转换，而不是单条关联文件，引用才能解析（见下文）。

- **`sigma/correlation/artex_enrich_scan_velocity.yml`** —— 短窗口内同一来源大量 `artex-enrich/1.0` 探测（补强以并发 4 运行、没有速率限制）。这是单请求规则漏掉的速度维度。`event_count`,`level: high`。
- **`sigma/correlation/artex_enrich_fanout.yml`** —— 同一来源携带补强 User-Agent 打到大量**不同**主机：机器级速度地横扫资产清单，关键是广度（而不只是数量）。`value_count`,`level: high`。
- **`sigma/correlation/artex_guard_block_burst.yml`** —— 同一主机上反复出现平台护栏管控标记，即活跃运行的 ARTEX 触发了自己的护栏，而非某份文档只是引用了该标记。`event_count`,`level: high`。
- **`sigma/correlation/artex_guard_marker_then_destructive.yml`** —— 护栏标记与破坏性命令在窗口内于同一主机共现（防御指南 4.2 节，多阶段）。把 ARTEX 专属标记与本来通用的破坏性命令信号结合，提高了特异性。`temporal`,`level: high`。

阈值与窗口都是保守默认值 —— 请按你的基线调优。4.2 节中纯 Web 的多阶段情形（枚举 → 试探 → 认证）仍需环境专属的基础规则，因为该模式无法归约为某个 ARTEX 独有 User-Agent。可作为起点的通用行为基 Sigma 模板见[防御指南 4.2 节](../docs/defense-zh.md)；它刻意不在本受测规则树中，因为无法以 ARTEX 源码为依据。

## 网络规则（Suricata）

Sigma 覆盖主机与日志遥测。线上可观测的两个 ARTEX User-Agent 都在 [`suricata/`](suricata/) 中以 [Suricata](https://suricata.io) 规则发布：补强探测器的 `artex-enrich/1.0`(`enrich/enrich.go`)带有存在性签名与高速枚举变体（sid 1000001–1000002）；norma SDK WebFetch 工具在攻击阶段的 `norma/0.4`(`github.com/Autumn-27/norma/tool/webfetch.go`)带有存在性签名（sid 1000003）。其他 worker 工具(Bash 执行的 `curl`、`nmap`)使用各自的 User-Agent，不带 ARTEX 独有指纹，因此网络层刻意只涵盖这两个 UA；范围、TLS 前提以及如何用 `suricata -T` 与参考 pcap 验证，见 [`suricata/README.md`](suricata/README.md)。

## ATT&CK 覆盖

这些规则标注的技术被汇总成 [MITRE ATT&CK](https://attack.mitre.org/) Navigator 层：[`attack/artex_navigator_layer.json`](attack/) —— 六个战术（侦察、命令与控制、执行、影响、凭据访问、收集）下的八个技术，每个都以规则的 `attack.*` 标签为依据，并按检测强度打分（ARTEX 专属签名 / 通用狩猎线索）。可在 [ATT&CK Navigator](https://mitre-attack.github.io/attack-navigator/) 中打开，查看每条规则覆盖哪种 ARTEX 行为；打分方式、技术与规则的对应表，以及如实说明的范围（覆盖不等于完备），见 [`attack/README.md`](attack/README.md)。[一致性测试](tests/attack/run.sh)可防止覆盖层脱离规则集漂移。

## 指标清单（机器可读）

对于想要原子指标而不是检测逻辑的防御者，[`indicators/artex_indicators.csv`](indicators/) 把 ARTEX 发出的独有指纹收进一个 CSV，可直接放进威胁情报平台、SIEM 查询表或主机分诊检查表 —— 包括补强与自动更新 User-Agent、护栏审计标记、服务器/代理默认端点、记录代理 CA 证书，以及 PostgreSQL 探索图谱 schema 指纹 —— 每行都记录它所依据的源文件以及（如果有）基于它的规则。同一批指标还以可直接导入的 [MISP](https://www.misp-project.org/) 事件形式提供([`indicators/artex_indicators.misp.json`](indicators/))，这样运行 MISP（或从 MISP 导出到 STIX）的防御者不必手工映射 CSV 列 —— 有规则支撑的指纹标记 `to_ids`，主机取证端口与 schema 指纹则不标。通用狩猎线索（破坏性命令）以及 norma SDK 共用的 `norma/0.4` WebFetch User-Agent（Suricata sid 1000003 承载的线上签名，并非 ARTEX 独有字符串）被刻意排除在可导入清单之外以避免误报；列说明、MISP 类型映射、如实提示以及防止 CSV 与 MISP 事件漂移的一致性测试见 [`indicators/README.md`](indicators/README.md)。

## 主机分诊

上述规则服务于拥有 SIEM、网络传感器或威胁情报平台的防御者。对另一种响应者 —— 站在单台可疑主机 shell 前、没有 SIEM 的人 —— [`triage/artex_host_triage.py`](triage/) 是一个只读脚本，用本机状态回答“这台机器上跑过 ARTEX 吗”。它落实同一批指纹，**外加 CSV 刻意不带 Sigma 规则的三个主机/DB 指标**（服务器监听端口、记录代理端点、PostgreSQL 探索图谱 schema）:它们无法从日志或网络观测，只能在主机上检查。它还会标记记录器的子进程环境注入 —— 运行中的进程同时带有代理变量与 mitmproxy CA 信任变量(`agent/worker.go`)—— 从 `/proc` 或 `--proc-from` 转储中读取。每条发现都是分诊线索，带有与其指标行相同的提示。见 [`triage/README.md`](triage/README.md)；内置的 `--self-test` 作为合并门禁运行（见下文）。

## 测试

规则附带 [`tests/`](tests/) 下可复现的测试，每个都只需要 Docker:

- **Suricata**([`tests/suricata/run.sh`](tests/suricata/run.sh))先校验整个规则文件能在 `suricata -T --init-errors-fatal` 下加载（解析失败的规则即使没有抓包触发也会被抓到），然后用 scapy 合成确定性抓包，对其跑 `suricata -r`，并断言存在性规则对每次探测触发一次、速率规则越过阈值、正常 User-Agent 的抓包产生零告警。仓库中不提交二进制抓包 —— 测试每次运行都会重新生成。
- **Sigma**([`tests/sigma/run.sh`](tests/sigma/run.sh))把下面的 `sigma check` 与 `sigma convert` 校验作为可执行测试：断言 0 错误、整棵树能编译成后端查询、每个原子指标字符串都保留在该查询中，以及关联规则单独转换会失败 —— 从而证明它确实依赖它所引用的原子规则。
- **Sigma 实时事件匹配**([`tests/sigma_match/run.sh`](tests/sigma_match/run.sh))把上面的 Sigma 套件从“有效/可编译”扩展到“真的会触发”，涵盖原子与关联两类规则：对每条原子规则，断言具代表性的恶意样本事件匹配、正常事件不匹配(例如 `.mitmproxy/` 下独立存在的 CA 文件不会触发记录代理规则，因为该规则的 `|all` 还要求 `_ca/` 目录)；对每条关联规则，断言正向时间线触发（窗口内在同一分组内达到阈值）、反向场景保持安静（低于阈值、超出窗口、分组被拆开或缺少某一环节）。解析全部交给 pySigma；测试只遍历编译后的条件树与聚合规格，用同一套原子匹配器决定哪些事件喂给被引用的规则。它把“跑不起来的检测只是声明”带到 Sigma 一侧，与 Suricata 已有的做法一致。
- **ATT&CK 层**([`tests/attack/run.sh`](tests/attack/run.sh))检查 ATT&CK 覆盖层与规则保持一致：其打分的技术与战术必须正好等于规则集上的 `attack.*` 标签，且每个技术都要指到一个存在的规则文件。新增规则而不更新覆盖层（或反之）会让测试失败。
- **指标来源一致性**([`tests/indicators/run.sh`](tests/indicators/run.sh))检查每条规则固定的指标仍是上游源码实际发出的字符串 —— `enrich/enrich.go` 中的 `artex-enrich/1.0`、`selfupdate/` 中的 `artex-selfupdate`、`guard/guard.go` 中的护栏标记、`db/db.go` 中的破坏性 token —— 并且仍固定在该规则中。它抓住的是上面三个测试抓不到的漂移：上游再同步改了 User-Agent 或标记，而每条规则仍然能编译、能触发。同一测试还会重读机器可读的 [`indicators/artex_indicators.csv`](indicators/artex_indicators.csv)，断言每一行发布数据仍以其来源与规则为依据，因此防御者导入的工件也不会漂移。最后，它断言它读取的每个上游源文件都出现在 CI 工作流的 `push` 与 `pull_request` 路径过滤器中，这样只改动新固定源文件的 PR 就无法跳过测试、让漂移通过合并门禁。这使得上文“依据本仓库源码中核实过的字符串，而非推测”从承诺变成守卫。
- **MISP 导出一致性**([`tests/misp/run.sh`](tests/misp/run.sh))证明 MISP 事件([`indicators/artex_indicators.misp.json`](indicators/artex_indicators.misp.json))是一份有效的 MISP 文档 —— 它能在 [pymisp](https://github.com/MISP/PyMISP) 下加载，而 pymisp 的对象模型会拒绝任何并非真实 MISP 类型的属性类型，所以这个工件是“真的能导入”而不只是“看起来像 MISP” —— 并且它与上面的 CSV 逐行同步：取值相同、每个指标的目标 MISP 类型/分类正确，`to_ids`/`disable_correlation` 标志也如实反映 CSV 的取向(有规则支撑即可处置，故 `to_ids` 打开；主机取证端口只是分诊线索，故 `to_ids` 关闭且禁用关联)。该事件与 CSV 一同手工维护 —— 它带有 CSV 没有的整理注释、UUID 与标签 —— 因此当你新增、删除或重设某行 CSV 的类型时，要在同一次提交里更新 MISP 事件，在两者一致之前该测试会一直失败。
- **Sigma 后端可移植性**([`tests/sigma_backends/run.sh`](tests/sigma_backends/run.sh))证明规则不只限于单个 Splunk 示例：整棵树（原子 + 关联）能在 Splunk、Elasticsearch `eql` 目标与 Grafana Loki 上编译，五条原子规则也仍能在不支持 Sigma 关联的后端(Elasticsearch `lucene`、Microsoft `kusto`)上编译。它以可重跑的检查为下文[验证与转换](#验证与转换)中的各后端支持矩阵提供了依据。
- **SigmaHQ 规范 lint**([`tests/sigma_lint/run.sh`](tests/sigma_lint/run.sh))针对 [`tests/sigma_lint/validators.yml`](tests/sigma_lint/validators.yml) 中记录的基线，运行完整的 SigmaHQ 校验器集(即 `pySigma-validators-sigmahq` 插件，普通 `sigma check` 不会加载它)，并断言 0 个问题。它还会检查完整集合确实运行过、并且只剩四个刻意排除并已记录在案的检查，这样某条规则一旦引入新的规范问题（标题大小写错、字段不在分类表内）就会让构建失败。

除这八个规则套件外，同一个 CI 工作流与 [`tests/run-all.sh`](tests/run-all.sh) 里还跑三个非套件门禁：一个 harness 同步检查(`run-all.sh`、CI 与各套件目录必须以相同顺序列出相同的套件);[主机分诊工具](triage/)的 `--self-test`([`tests/triage-selftest.sh`](tests/triage-selftest.sh))，它构造一台合成主机并断言每项分诊检查都会命中，而干净主机产生零发现；以及防御模板一致性检查([`tests/check-defense-template.sh`](tests/check-defense-template.sh))，它重跑防御指南内联 Sigma 关联模板所陈述的事实 —— `sigma check` 通过、能在 splunk/eql/loki 上转换、lucene 与 kusto 拒绝该关联、韩文版与英文版规则正文逐字节一致 —— 因为这个模板以散文形式存在于本规则树之外，否则没有任何套件能发现它相对上游 sigma-cli 悄悄过期。

任何断言失败时，每个脚本都以非零码退出。参见 [`tests/README.md`](tests/README.md)。

## 如何如实阅读这些规则

- **静态指标可以被改掉。** 运营者可以设置不同的 User-Agent，因此 `artex-enrich/1.0` 或 `artex-selfupdate` 没有出现并**不**意味着安全。经得起时间的信号是**行为** —— 单一来源把侦察 → 枚举 → 试探 → 认证/注入尝试串成一条链，随响应调整，毫不停歇地运行。这一层在防御指南（第 1、2 节与 4.1–4.2 节）中描述；上面的 `sigma/correlation/` 规则以可部署的关联形式发布它（速率、扇出、护栏拦截突发，以及护栏标记伴随破坏性命令的多阶段），而纯 Web 的多阶段情形仍需按你的环境编写基础规则。
- **破坏性命令规则属于通用狩猎。** 它镜像 ARTEX 的护栏拦截名单，但同样的命令正当管理员也会执行。命中只当线索，先给你的环境配置允许名单，不要单凭它把事件归属到 ARTEX。
- **端口指标属于主机取证，不是 Sigma。** ARTEX 服务器默认的 `:8787` 与记录代理 `127.0.0.1:8788`(`cmd/artex/main.go`)最好在可疑主机上用 `ss`/`netstat` 检查，因此它们写在防御指南里并列入[指标 CSV](indicators/)用于分诊，而不是作为噪声很大的网络规则发布。[主机分诊脚本](triage/)正是为有 shell 访问、但没有 SIEM 的响应者执行这些主机本地检查（端口、记录代理工件、日志标记与 PostgreSQL schema）。

## 验证与转换

这些规则用 [sigma-cli](https://github.com/SigmaHQ/sigma-cli)(pySigma)验证。复现方式：

```sh
python3 -m venv .venv && . .venv/bin/activate
pip install sigma-cli

# 结构 + 最佳实践校验(期望:0 错误、0 问题)
sigma check detections/sigma/

# 加载本独立规则集记录的基线,跑完整 SigmaHQ 规范集(期望:0 问题)。
# 上面的普通 `sigma check` 不会加载这些校验器。
pip install pySigma-validators-sigmahq
sigma check --validation-config detections/tests/sigma_lint/validators.yml detections/sigma/

# 转换到目标查询语言,例如 Splunk
sigma plugin install splunk
sigma convert -t splunk --without-pipeline detections/sigma/artex_enrich_user_agent.yml

# 转换整棵树,以便关联规则能解析它们按 id 引用的原子规则
sigma convert -t splunk --without-pipeline detections/sigma/
```

基线启用了除四项之外的所有 SigmaHQ 规范，这四项编码的是 SigmaHQ 单仓归档方案与分类法，不适用于独立规则集；每项排除及其理由都记录在 [`tests/sigma_lint/validators.yml`](tests/sigma_lint/validators.yml) 中，并由上面的 lint 测试强制。

### Sigma 后端可移植性

`sigma/correlation/` 下的规则按 `id` 引用它们的原子基础规则，因此只能在支持 Sigma 关联转换的后端上转换。各后端的支持情况不同，所以 `-t` 的选择很关键。下表以固定参考版本(`sigma-cli` 3.1.0，及当时兼容的最新后端)实测，并由 [`tests/sigma_backends/run.sh`](tests/sigma_backends/run.sh) 复现：

- **能转换整棵树（原子 + 关联）：** Splunk(`-t splunk`)、Elasticsearch EQL(`-t eql`)、Grafana Loki(`-t loki`)。直接转换 `detections/sigma/` 就能同时得到关联查询。
- **只能转换原子规则（尚不支持关联）：** Elasticsearch Lucene(`-t lucene`)、OpenSearch(`-t opensearch_lucene`)，以及面向 Sentinel 与 Defender XDR 的 Microsoft `kusto` 后端(`-t kusto`)。在这些后端上，请转换五条原子规则，并在产品中原生表达关联窗口(例如 Sentinel 的定时分析规则 `summarize ... by bin(TimeGenerated, 30m)`)。若直接传入整个目录，转换会以“Backend does not support correlation rules.”中止。

```sh
# 只转换原子规则,例如面向 Microsoft Sentinel / Defender(kusto 后端)
sigma plugin install kusto
sigma convert -t kusto --without-pipeline \
  detections/sigma/artex_enrich_user_agent.yml \
  detections/sigma/artex_selfupdate_egress.yml \
  detections/sigma/artex_guard_audit_framing.yml \
  detections/sigma/artex_recording_proxy_ca.yml \
  detections/sigma/destructive_command_hunting.yml
```

固定版本下的已知边界：Elasticsearch ES|QL 目标(`-t esql`)会拒绝护栏标记规则(`String value expressions are not supported`)，因此在那里改为转换其余三条原子规则；IBM QRadar 插件(`ibm-qradar-aql`)与所固定的 pySigma 不兼容，需要 `--force-install`，因此未纳入测试。可用 `sigma list targets` 查看你环境中已安装的后端。

上面的示例使用 `--without-pipeline`，输出的是规则正文中的通用字段名(`cs-user-agent`、`cs-host`、`CommandLine`)。要匹配你产品的 schema，请去掉该标志并用 `-p` 应用处理管线(见 `sigma list pipelines`)。注意产品管线只映射字段名，可能还需要规则通用 `logsource` 未指定的目标表 —— 例如 `-p sentinel_asim` 会以“Unable to determine table name”中止，直到你为数据设置 `query_table`；因此部署前请把字段与目标表都映射到你的环境。

## 贡献

欢迎检测与加固方面的贡献。新规则应让每个指标都以可观测事实为依据，在 `description` 中写明局限，干净地通过 SigmaHQ 校验器基线(`sigma check --validation-config tests/sigma_lint/validators.yml`)，并避免任何读起来像攻击指引的内容。参见 [`../CONTRIBUTING.md`](../CONTRIBUTING.md)。
