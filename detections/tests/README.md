# ARTEX 检测规则测试

中文 · [English](README.en.md) · [한국어](README.ko.md)

> 本目录是可复现的回归测试，用来证明 [`../`](../) 中的检测规则确实会触发 —— 同样重要的是，在正常流量上保持安静。仓库里不提交二进制抓包：抓包在**每次运行时确定性地合成**，运行后即删除，因此测试以可读源码而不是不透明夹具的形式交付，也不会让仓库变大。所有测试只能用于**防御·检测目的**，保护你自己拥有或已获得书面许可的系统。

跑不起来的检测只是声明；这些测试把规则文件与防御指南中的声明，变成审查者可以从源码重新运行的东西。

## 一次跑完所有套件 —— [`run-all.sh`](run-all.sh)

[`run-all.sh`](run-all.sh) 用一条命令、按与 CI 相同的顺序运行下面全部八个套件，不必手工调用八个 `run.sh`。即使前面某个套件失败，每个套件也会跑完；脚本最后为每个套件打印一行 PASS/FAIL 摘要，只要有套件失败就以非零码退出。

在套件之前，它先运行 harness 自检([`check-harness-sync.sh`](check-harness-sync.sh)):若本套件清单、[CI](../../.github/workflows/detections.yml) 中的逐套件步骤、以及磁盘上的套件目录三者所指的套件或顺序不一致，该次运行即失败。这是八个套件自身看不到的盲区：只接入三者之一(新增了 CI 步骤却没有 `run-all.sh` 条目，或有目录从未加入任何一方)的套件，会让每个套件测试都通过，而本地绿色的 `run-all.sh` 却悄悄不再代表 CI 绿色。该检查是门禁而非第九个套件：它不出现在下面的摘要里，八个检测套件仍然是八个。

另外两个门禁以同样方式运行 —— 位于套件循环之前、不进入摘要、从不计为套件：[主机分诊工具](../triage/)的 `--self-test`([`triage-selftest.sh`](triage-selftest.sh))与防御模板一致性检查([`check-defense-template.sh`](check-defense-template.sh))。后者重跑防御指南就其中内联 Sigma 关联模板所陈述的事实(`sigma check` 通过、能在 splunk/eql/loki 上转换、lucene 与 kusto 拒绝该关联、韩文版与英文版规则正文逐字节一致)；该模板以散文形式存在于 [`../sigma/`](../sigma/) 规则树之外，否则没有套件能发现它相对上游 sigma-cli 悄悄过期。

```sh
detections/tests/run-all.sh
```

预期输出（节选）：

```
===== detection suites summary =====
  PASS  sigma
  PASS  sigma_match
  PASS  sigma_lint
  PASS  sigma_backends
  PASS  suricata
  PASS  attack
  PASS  indicators
  PASS  misp
RESULT: PASS
```

由于任何失败都会非零退出，它可以直接放进 pre-commit 钩子。仓库根的 [`.pre-commit-config.yaml`](../../.pre-commit-config.yaml) 里有现成例子：用 `pip install pre-commit && pre-commit install` 安装后，运行器会在触及检测规则或它们所固定的上游源文件的提交上触发 —— 与 CI 同范围。各套件支持的镜像与版本覆盖变量(`PYTHON_IMAGE`、`SIGMA_CLI_VERSION`、`SIGMAHQ_VALIDATORS_VERSION`、`SURICATA_IMAGE`)会被该运行器继承，所以导出其中任何一个都一次性作用于所有套件。

## Suricata —— [`suricata/`](suricata/)

[`suricata/run.sh`](suricata/run.sh) 端到端验证 [`../suricata/artex.rules`](../suricata/artex.rules) 中的网络规则，并断言五个性质：

- **有效** —— 整个规则文件能在 `suricata -T --init-errors-fatal` 下加载，因此解析或初始化失败的规则，即使下面的抓包没有触发它也会被抓到。普通 `suricata -r` 会跳过这类规则并仍然以 0 退出，所以这个加载检查相当于 Sigma 套件的 `sigma check` 有效性断言。
- **存在性（enrich）** —— sid `1000001` 对每次补强探测恰好触发一次。
- **速率** —— sid `1000002` 在越过 `detection_filter`（每来源 300 秒内 30 次请求）后触发。
- **存在性（WebFetch）** —— sid `1000003` 对每次 norma WebFetch 请求恰好触发一次，而同一份抓包上 enrich 的 sid 保持安静 —— 两个网络签名彼此区分，而不只是各自存在。
- **特异性** —— 同一份抓包只把 User-Agent 换成正常浏览器，就产生**零**条 ARTEX 告警。

[`suricata/gen_pcap.py`](suricata/gen_pcap.py) 用 [scapy](https://scapy.net) 生成抓包：从一个固定来源发出 N 条独立的明文 HTTP 请求/响应流，各带指定 User-Agent，时间戳为固定基准、间隔一秒。它只写文件 —— 从不发包，也不碰网络。

### 运行

只需要 Docker;scapy 与 Suricata 都在容器里运行。

```sh
detections/tests/suricata/run.sh
```

预期输出（节选）：

```
  PASS  ruleset loads with zero parse/init errors (suricata -T)
  PASS  sid 1000001 presence: one alert per probe  (got 35, want eq 35)
  PASS  sid 1000002 velocity: fires past 30-in-300s  (got 5, want ge 1)
  PASS  sid 1000003 presence: one alert per WebFetch request  (got 8, want eq 8)
  PASS  enrich sids stay silent on norma traffic (specificity)  (got 0, want eq 0)
  PASS  benign browser UA produces no ARTEX alerts  (got 0, want eq 0)
RESULT: PASS
```

任何断言失败时脚本都会非零退出，因此可直接放进 CI 或 pre-commit 钩子。若你在内部做了镜像，可用 `SURICATA_IMAGE` / `PYTHON_IMAGE` 覆盖镜像。

### 为什么速率告警数是下限而不是精确匹配

`run.sh` 对存在性计数(`1000001 == 35`、`1000003 == 8`)与正常计数(`== 0`)做精确断言，因为这些与引擎版本无关：每个匹配请求一条告警，换 User-Agent 则不匹配。而速率规则的计数取决于具体 Suricata 版本在边界上如何判定 `detection_filter` 阈值，因此测试只断言 `>= 1`，并单独记录参考值。在 **Suricata 8.0.7** 上，参考运行在 sid `1000002` 上产生 **5** 条告警（流 31–35，越过 300 秒内 30 次的阈值之后）。

## Sigma —— [`sigma/`](sigma/)

[`sigma/run.sh`](sigma/run.sh) 用 [sigma-cli](https://github.com/SigmaHQ/sigma-cli)(pySigma)对 [`../sigma/`](../sigma/) 下的 Sigma 规则做结构校验与编译校验，并断言五个性质：

- **有效** —— `sigma check` 对整棵树报告 0 错误、0 条件错误、0 问题。
- **可编译** —— `sigma convert -t splunk` 能把整棵树无错地转换成后端查询语言。
- **指标保留** —— 每个原子指标字符串(`artex-enrich/1.0`、`artex-selfupdate`、护栏标记，以及记录代理 CA 文件名 `mitmproxy-ca-cert.pem`)仍存在于编译后的查询中，因此规则不会悄悄丢掉它赖以成立的字符串。
- **关联可编译** —— [`../sigma/correlation/`](../sigma/correlation/) 中的行为规则会输出其 `event_count` / `value_count` 聚合，而不是被丢弃。
- **关联确实承载逻辑** —— *单独*转换一条关联规则会失败，因为它按 `id` 引用其原子基础规则；该引用是被强制的，不是装饰。这是上面 Suricata 特异性断言在 Sigma 一侧的对应物。

这就是 [`../README.md`](../README.md) 中记录的结构 + 编译校验，只是可执行、带断言。配套的 [`sigma_match/`](sigma_match/) 套件则为原子规则与关联规则补上*匹配*的另一半 —— 具代表性的恶意事件（或时间线）会触发每条规则，而正常事件不会 —— 于是 Sigma 规则既有可复现的校验测试，也有可复现的匹配测试，与 Suricata 规则一致。（早先担心手写匹配器太弱会削弱规则，这一顾虑已通过把解析全部交给 pySigma 解决；信任模型见下一节。）

### 运行

只需要 Docker;sigma-cli 与 splunk 后端都在容器里运行，不向仓库写入任何东西。

```sh
detections/tests/sigma/run.sh
```

预期输出（节选）：

```
  PASS  sigma check: 0 errors, 0 condition errors, 0 issues
  PASS  whole tree converts to splunk (exit 0)
  PASS  indicator present: artex-enrich/1.0
  PASS  correlation rule fails to convert alone — it requires its atomic base rule
RESULT: PASS
```

任何断言失败时脚本都会非零退出，因此可直接放进 CI 或 pre-commit 钩子。sigma-cli 固定为参考版本(`3.1.0`)；若在内部镜像，可用 `SIGMA_CLI_VERSION` 覆盖，或用 `PYTHON_IMAGE` 覆盖镜像。

## Sigma 实时事件匹配 —— [`sigma_match/`](sigma_match/)

[`sigma_match/run.sh`](sigma_match/run.sh) 证明 [`../sigma/`](../sigma/) 下的 Sigma 规则 —— 包括原子规则与 [`../sigma/correlation/`](../sigma/correlation/) 下的关联规则 —— 在匹配事件（或时间线）上确实*触发*，而在正常事件上保持安静 —— 也就是 Suricata 套件为网络规则提供的“跑不起来的检测只是声明”这一保证，在这里扩展到主机/日志层规则。它断言六个性质，原子规则与关联规则各三个：

- **规则/样本配对** —— 每条原子规则都有一个 [`events/<name>.json`](sigma_match/events/) 样本文件，且每个样本文件都能映射回一条规则，因此新增规则却没带样本会在这里失败，而不会变成未测试状态。
- **真阳性** —— 每条规则都能匹配它全部恶意样本事件。
- **真阴性** —— 每条规则都不匹配它任何正常样本事件。例如 `.mitmproxy/` 下单独存在的 `mitmproxy-ca-cert.pem` **不会**触发记录代理规则，因为该规则的 `|all` 还要求 ARTEX 写出的 `_ca/` 目录 —— 正是匹配测试证明了这种区分能力。
- **关联规则/时间线配对** —— 每条关联规则都有一个 [`events/correlation/<name>.json`](sigma_match/events/correlation/) 时间线文件，且每条时间线都能映射回一条规则。时间线中的每个事件都带一个以相对秒表示的 `ts` 字段。
- **关联真阳性** —— 每条规则在正向时间线上触发：同一分组内、窗口范围内达到阈值。例如同一来源(`c-ip`)的请求在 10 分钟内扇出到 20 台不同主机，会触发补强扇出规则。
- **关联真阴性** —— 在未达阈值、达到阈值但事件分散到窗口之外、事件被拆到不同分组、或 temporal 规则缺少某个环节时，规则保持安静。特别地，高数量但低广度的突发**不会**触发扇出规则：关键信号是广度而不是数量，而正是匹配测试证明了这种区分能力。

信任模型是让 pySigma —— 而不是手写代码 —— 解析每条规则：原子规则解析为条件树(`|contains` → 通配值，`|all` → AND,`1 of selection_*` → OR)，关联规则解析为其聚合规格（类型、group-by、timespan、阈值，以及对原子基础规则的已解析引用）。[`check.py`](sigma_match/check.py) 只遍历这棵树与规格，用完全相同的原子匹配器决定哪些事件喂给被引用的规则，因此权威的 Sigma 逻辑仍留在 pySigma 中；遇到任何它不显式支持的构造，它会抛错而不是放过（失败即关闭）。范围与限制写在脚本头部：关联窗口采用标准滑动窗口解释(以每个匹配事件为锚点的 `timespan` 秒窗口)，真实 SIEM 的窗口可能不同；匹配**不区分大小写**(这是 `sigma/` 套件所针对的 splunk 后端默认行为，破坏性规则自身的误报说明也以此为前提)；关键字匹配是全文本子串查找。它是针对规则字段/取值/条件/聚合逻辑的回归测试，不能替代在你自己的 SIEM 中验证，因为后者的字段归一化可能不同。

### 运行

只需要 Docker;pySigma 在容器里运行，不向仓库写入任何东西。

```sh
detections/tests/sigma_match/run.sh
```

预期输出（节选）：

```
  PASS  rule/sample pairing: 5 atomic rules, 5 event files, no orphans
  PASS  artex_enrich_user_agent: 1/1 positive events matched
  PASS  artex_recording_proxy_ca: 2/2 benign events correctly not matched
  PASS  rule/timeline pairing: 4 correlation rules, 4 timeline files, no orphans
  PASS  artex_enrich_fanout: fired — 20 distinct hosts from one source within the 10-minute window
  PASS  artex_enrich_fanout: quiet — high volume, low breadth: 25 requests from one source but only 4 distinct hosts
RESULT: PASS
```

任何断言失败时脚本都会非零退出，因此可直接放进 CI 或 pre-commit 钩子。pySigma 固定为参考版本(`2.0.0`)；若在内部镜像，可用 `PYSIGMA_VERSION` 覆盖，或用 `PYTHON_IMAGE` 覆盖镜像。

## Sigma 后端可移植性 —— [`sigma_backends/`](sigma_backends/)

[`sigma_backends/run.sh`](sigma_backends/run.sh) 证明规则不只限于 Sigma 测试所验证的单个 Splunk 示例，并让 [`../README.md`](../README.md) 中的各后端支持矩阵保持如实。Sigma 关联转换与后端相关，因此 README 需要告诉防御者哪些 `-t` 目标能接整棵树、哪些只能接原子规则 —— 而这样的说法只有能重跑才可信。它断言两个性质，都是正向断言，所以只有真的回归才会失败：

- **关联可移植** —— 整棵树（原子 + 关联）能在 Splunk、Elasticsearch `eql` 目标与 Grafana `loki` 上转换，且补强指标保留在每个查询中。这说明关联规则并不是只能在 Splunk 上用。
- **仅原子规则的回落方案可用** —— 五条原子规则仍能在 `lucene` 与 Microsoft `kusto` 后端上转换（它们在固定版本下不支持 Sigma 关联转换），因此使用这些后端的防御者可以部署原子规则，并在产品内原生表达关联窗口。

它刻意不断言反向命题“后端 X 做不了关联”：那会把后端的*改进*变成红色构建。如实的限制写在 README 里，并由本测试的命令复现。[`sigma_backends/check.sh`](sigma_backends/check.sh) 是容器内那一半；它安装固定版本的 sigma-cli 与四个后端，并以只读方式挂载规则树。

### 运行

只需要 Docker;sigma-cli 与各后端都在容器里运行，不向仓库写入任何东西。

```sh
detections/tests/sigma_backends/run.sh
```

预期输出（节选）：

```
  PASS  whole tree (atomic + correlation) converts on 'eql', enrich indicator survives
  PASS  five atomic rules convert on 'kusto', enrich indicator survives
RESULT: PASS
```

任何断言失败时脚本都会非零退出。sigma-cli 已固定(`3.1.0`，可用 `SIGMA_CLI_VERSION` 覆盖)；后端插件会安装当时最新兼容版本，因此这个套件对上游后端发布最敏感 —— 插件一旦放弃支持，构建就会变红，而这正是要同步更新固定版本与 README 矩阵的信号。

## SigmaHQ 规范 lint —— [`sigma_lint/`](sigma_lint/)

[`sigma_lint/run.sh`](sigma_lint/run.sh) 让 README 与 `CONTRIBUTING.md` 中“干净通过 `sigma check`”的承诺覆盖 SigmaHQ 的规范，而不只是 pySigma 的核心检查。普通 `sigma check` 不会加载 `pySigma-validators-sigmahq` 插件，所以标题大小写、字段名分类法、logsource 分类法与参考链接约定都得不到检查。本套件安装该插件，并针对 [`sigma_lint/validators.yml`](sigma_lint/validators.yml) 中记录的基线运行完整集合。它断言两个性质：

- **记录的基线是干净的** —— 带 `validators.yml` 的 `sigma check` 报告 0 错误、0 问题。
- **完整集合确实在运行，且只剩记录的排除项** —— 不加任何排除运行全部 SigmaHQ 校验器时仍会报告问题，且每一个都属于 `validators.yml` 刻意禁用的四项（别无其他）。这是防空转守卫：若插件加载失败，全量运行什么都不会报告，第一个性质就会以错误的原因通过，因此要求已知排除项必须出现。

这四项排除对应 SigmaHQ 单仓归档方案(logsource 前缀与 `correlation_` 文件名)与分类法(通用 `application` logsource 与无 product 的 `process_creation`)，以及分支引用与永久链接的参考约定 —— 它们都不适合一个引用自身活文档的小型独立规则集。每项排除都在 `validators.yml` 中就地写明理由。由于*其余*每项 SigmaHQ 检查都会强制执行，规则一旦引入新的规范问题 —— 标题大小写错、字段名不在分类表内 —— 构建就会变红。`pySigma-validators-sigmahq` 已固定(`0.21.0`，可用 `SIGMAHQ_VALIDATORS_VERSION` 覆盖)；升级它可能暴露新的规范，这就是更新规则或记录基线的信号。

### 运行

只需要 Docker;sigma-cli 与校验器插件都在容器里运行，不向仓库写入任何东西。

```sh
detections/tests/sigma_lint/run.sh
```

预期输出（节选）：

```
  PASS  sigma check with the documented baseline: 0 errors, 0 issues
  PASS  every reported issue is one of the four documented exclusions
RESULT: PASS
```

## ATT&CK 层 —— [`attack/`](attack/)

[`attack/run.sh`](attack/run.sh) 检查 [`../attack/artex_navigator_layer.json`](../attack/artex_navigator_layer.json) 中的 [ATT&CK 覆盖层](../attack/)是否与它声称覆盖的规则保持一致。脱离规则集的覆盖层比没有更糟，因此它把“这些规则覆盖这些 ATT&CK 技术”变成审查者可以从源码重跑的东西。它断言：

- **层文件有效** —— 文件能按 JSON 解析，并带有 ATT&CK Navigator v4.x 要求的字段，每条目都有格式正确的技术 ID 与合法 ATT&CK 战术。
- **双向匹配** —— 打分的技术*正好*是 Sigma 规则上的 `attack.*` 技术标签：规则里有而层里缺、层里有而规则里缺，都不允许。战术同样如此匹配。
- **有依据** —— 每个打分技术的注释都指到一个存在的规则文件，因此层不可能引用已被改名或删除的规则。

这是一致性检查，不是触发测试：它不需要任何检测后端，只需要 Python 标准库，所以与 Sigma、Suricata 测试不同，它不依赖版本相关的计数。[`attack/check.py`](attack/check.py) 是容器内那一半；它以只读方式读取 detections 树，不写任何东西。

### 运行

只需要 Docker；检查在 Python 容器里运行，不向仓库写入任何东西。

```sh
detections/tests/attack/run.sh
```

预期输出（节选）：

```
  PASS  scored techniques match the rule set exactly (8: T1059, T1105, T1485, T1489, T1557, T1561.002, T1592, T1595)
  PASS  scored tactics match the rule set exactly (collection, command-and-control, credential-access, execution, impact, reconnaissance)
RESULT: PASS
```

任何断言失败时脚本都会非零退出，因此可直接放进 CI 或 pre-commit 钩子。若在内部镜像，可用 `PYTHON_IMAGE` 覆盖镜像。

## 指标来源一致性 —— [`indicators/`](indicators/)

[`indicators/run.sh`](indicators/run.sh) 证明上面三个测试都没覆盖的一件事：每条规则固定的指标，仍是 ARTEX 自身源码实际发出的字符串。Sigma 测试证明指标在规则→查询的*编译*中保留；ATT&CK 测试证明层与规则标签一致；Suricata 测试证明网络规则在合成抓包上*触发*。它们都没有回头看指标所声称的来源文件。它们都漏掉的腐坏，是上游再同步把探测器 User-Agent 升到 `artex-enrich/2.0` 或改写了护栏标记：每条规则仍能编译、层仍能匹配、pcap 测试仍能触发 —— 而部署出去的规则却悄悄不再匹配真实的 ARTEX 流量。它对每个指标做双向断言：

- **来源仍会发出它** —— 该值仍存在于产生它的上游源文件中(`enrich/enrich.go` 的 `artex-enrich/1.0`、`selfupdate/` 的 `artex-selfupdate`、`guard/guard.go` 的护栏标记)。值不见了就意味着出现了规则尚未跟进的上游变更。
- **规则仍固定它** —— 该值仍存在于建立在其上的规则中，因此规则编辑无法悄悄把指标移离来源。Suricata 规则按它的 `startswith` 前缀检查，与它实际匹配线上的方式一致。
- **拦截名单对应关系** —— 破坏性命令 token(`rm -rf`、`mkfs`、`DROP DATABASE`、`FLUSHALL`)同时出现在 ARTEX 的护栏拦截名单(`db/db.go`)与镜像它的狩猎规则中。这些是通用狩猎线索而非独有指纹，因此测试只断言规则实际声称的对应关系。
- **发布清单仍有依据** —— 机器可读的指标清单 [`detections/indicators/artex_indicators.csv`](../indicators/artex_indicators.csv)，也就是防御者导入的工件，会被逐行重读：每个值都必须仍存在于它所引用的源文件，并仍固定在它所引用的规则中，而且测试所依据的每个指纹都必须出现在清单里。因此发布的 CSV 无法在任一方向上悄悄偏离它所声称的来源。
- **两个门禁都覆盖每个固定源文件** —— 测试读取的每个上游源文件都由运行它的两个门禁覆盖：CI 工作流的 `push` 与 `pull_request` 路径过滤器([`.github/workflows/detections.yml`](../../.github/workflows/detections.yml))，以及本地 pre-commit 钩子的 `files` 正则([`.pre-commit-config.yaml`](../../.pre-commit-config.yaml))。所需集合由指标自身推导而来，因此当新增一个固定源文件(如 `cmd/artex/main.go` 的端口此前那样)却没把它接进*两个*门禁时，这里就会失败 —— 否则只改动该源文件的变更会在缺少它的那个门禁上跳过测试：在 CI 上它绿灯通过合并门禁，在钩子上它永远在本地抓不到，尽管钩子承诺“与 CI 同范围”。

这使 [`../README.md`](../README.md) 的承诺 —— “这里每个指标都以本仓库源码中核实过的字符串为依据，而非推测” —— 以及 CONTRIBUTING 的第一条贡献约定，变成审查者可以重跑的守卫。与 ATT&CK 测试一样，它不需要检测后端，只需要 Python 标准库；[`indicators/check.py`](indicators/check.py) 以只读方式读取规则树、发布的指标清单、被固定的源文件包，以及触发它的两个门禁（CI 工作流与 pre-commit 配置），不写任何东西。

### 运行

只需要 Docker；检查在 Python 容器里运行，不向仓库写入任何东西。

```sh
detections/tests/indicators/run.sh
```

预期输出（节选）：

```
  PASS  enrichment prober User-Agent: 'artex-enrich/1.0' emitted by enrich/enrich.go
  PASS  detections/sigma/artex_enrich_user_agent.yml pins 'artex-enrich/1.0'
  PASS  'FLUSHALL' present in both db/db.go and detections/sigma/destructive_command_hunting.yml
  PASS  enrich-user-agent: 'artex-enrich/1.0' grounded in enrich/enrich.go
  PASS  tested fingerprint 'artex-enrich/1.0' is published in the list
  PASS  .github/workflows/detections.yml push paths covers cmd/artex/main.go
  PASS  .pre-commit-config.yaml files covers cmd/artex/main.go
RESULT: PASS
```

任何断言失败时脚本都会非零退出，因此可直接放进 CI 或 pre-commit 钩子。若在内部镜像，可用 `PYTHON_IMAGE` 覆盖镜像。

## MISP 导出一致性 —— [`misp/`](misp/)

[`misp/run.sh`](misp/run.sh) 覆盖指标的第二种发布形式 —— 可直接导入的 MISP 事件 [`detections/indicators/artex_indicators.misp.json`](../indicators/artex_indicators.misp.json)。上面的指标测试让 CSV 以源码为依据；这个测试让 MISP 事件 —— 防御者真正载入威胁情报平台的工件 —— 不偏离该 CSV。它断言：

- **它确实是 MISP** —— 事件能在 [pymisp](https://github.com/MISP/PyMISP) 下加载，而 pymisp 的对象模型会拒绝任何 `type` 不是真实 MISP 类型的属性。看起来合理但类型无效的内容会在这里失败，因此“有效的 MISP”由 MISP 服务器自己使用的库来证明，而不是靠声称。
- **与 CSV 逐行同步** —— 每个 CSV 行都恰好映射到一个 MISP 属性，类型与分类符合意图(`http.user-agent` → `user-agent`；护栏标记 `string` → `pattern-in-file`;`port` → `port`;`ip-dst|port` → `ip-dst|port`，复合值采用 `ip|port` 形式；探索图谱 schema 的 `other` → `other`)，并且没有 MISP 属性缺少对应的 CSV 行。事件与 CSV 一同手工维护，因此新增、删除或重设某行 CSV 的类型却没在同一次提交里同步更新 `artex_indicators.misp.json` 就会失败。
- **`to_ids` 与 `rule` 列一致** —— 有规则支撑的指标为 `to_ids: true`；没有规则的主机取证行为 `to_ids: false` 且 `disable_correlation: true`。把标志翻到与 CSV 相悖的一侧就会失败，因此 MISP 事件无法悄悄夸大或隐瞒哪些指纹可处置。
- **护栏标记逐字节保留**，并且 `detections/**` 在 CI 路径过滤器内，因此对 CSV 或事件的改动都会触发本套件。

与上面那些纯标准库测试不同，本套件在它的容器内安装固定版本的 `pymisp`（宿主机上不安装任何东西）;[`misp/check.py`](misp/check.py) 以只读方式读取 CSV、MISP 事件与 CI 工作流，不写任何东西。

### 运行

只需要 Docker;pymisp 安装在容器内，不向仓库写入任何东西。

```sh
detections/tests/misp/run.sh
```

任何断言失败时脚本都会非零退出。若在内部镜像，可用 `PYTHON_IMAGE` 覆盖镜像，用 `PYMISP_VERSION` 覆盖固定的库版本。

## 贡献

新检测规则若能配一条证明它会触发的测试，就更有说服力。测试应确定性地合成自己的输入，对与引擎版本无关的性质做精确断言（较软的性质则用下限加记录参考值），并避免任何读起来像攻击指引的内容。参见 [`../../CONTRIBUTING.md`](../../CONTRIBUTING.md) 与 [`../README.md`](../README.md) 中的规则索引。

八个套件都在 CI 中运行(见 [`../../.github/workflows/detections.yml`](../../.github/workflows/detections.yml))，在每次触及 `detections/` 的 push 或 pull request 上执行 —— 其中指标测试还会在它所固定的上游源文件(`enrich/`、`selfupdate/`、`guard/`、`db/`、`cmd/artex/main.go`)变化时运行 —— 因此丢掉指标、偏离 ATT&CK 层、在已记录的后端上不再可转换、破坏 SigmaHQ 规范、与源码脱钩、让 MISP 事件偏离 CSV，或固定了新源文件却没接入工作流的规则改动，都会在合并之前让构建变红。
