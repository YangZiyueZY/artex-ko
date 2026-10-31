# 自主 AI 攻击的防御与检测指南

> 本文档用于帮助**防御方**理解 ARTEX 这类**自主 AI 渗透智能体**是如何运作的，并据此培养**检测与拦截**这类攻击的能力。这不是攻击方法指南。所有内容只能用于保护你自己拥有、或已获得书面明确授权的系统。未经授权探测或攻击他人的信息网络，其行为本身即构成犯罪（参见[上级 README 的安全与滥用警告](../README.md)）。
>
> 其他语言：**[English](defense-en.md)** · **[한국어](defense-ko.md)**。

自主 AI 攻击工具把过去由一个人贴身手动执行的渗透测试，变成了 LLM 智能体**自行拆分目标、执行真实工具、累积发现**的 24 小时自动化过程。防御者面对的对手，由此从“一名熟练攻击者”变成“一支不知疲倦、不会停歇的智能体集群”。本指南整理的正是这一变化对检测与响应提出了什么要求。

---

## 1. 自主 AI 攻击与传统扫描器有何不同

传统的漏洞扫描器（例如基于固定签名的工具）按预定顺序抛出事先定好的检查项，跑完就结束。ARTEX 这类自主智能体的结构不同。正如[系统架构](../README.md)所述，下列要素结合在一起来**在无人介入的情况下跑完多阶段攻击链**。

- **分工明确的多智能体。** 拆分目标的 `goals`、判定下一步方向并产出意图（intent）的 `planner`、把单条意图落到真实工具上执行的多个 `worker`、以及人来介入的 `mainagent` 各司其职。planner 是唯一的意图生产者，worker 们并行执行这些意图。
- **在双图谱上累积状态。** 分别积累“有什么”（资产图谱）与“测到了哪里”（探索图谱），再用锚点把两者关联起来。因此攻击会**逐步深入**，从不同角度重访同一资产，并在先前的观察之上搭建下一步。
- **事件驱动的闭环。** 图每次变化都会唤醒 planner 去分派下一条意图，worker 的写入又触发下一轮循环。这个循环一直持续到目标被证明完成为止。
- **串行攻击链的稳定推进。** planner 把“发现注入点 → 获取凭据 → 横向移动 → 权限提升”这类依赖顺序只写入共享 todolist 一次，并且只给前置条件已满足的后续步骤分派意图。因此在无状态会话环境里攻击链也不会错位、能够跑完全程。
- **LLM 按上下文改写载荷。** 执行工具与载荷不是写死在代码里的常量，而是 LLM 读取上下文后生成的值，所以每个请求的形态都会略有差异。

### 为什么更难检测

- **固定签名难以命中。** 载荷会随上下文变化，精确匹配已知恶意字符串的规则（WAF 签名）很容易被绕开。
- **可以放慢，也可以像人一样分段推进。** 智能体会在轮次之间休息，并根据发现改变方向，因此只看“短时间内请求量大”的速率型检测可能漏掉它。
- **侦察与渗透连成一条流。** 过去人看完侦察结果、几天后再手动攻击的那道间隙消失了，从首次接触到数据外传的时间大幅缩短。

### 但为什么仍然能检测：行为难以隐藏

静态指纹（User-Agent、特定载荷字符串）只要运营者愿意就能改掉。但**自主智能体的行为方式**是攻击的本质，很难改变。下面第 2~4 节正是基于这一行为视角。

- 同一来源把**多个性质不同的请求阶段**（侦察 → 枚举 → 认证尝试 → 利用）以**连贯的会话串成一条链**。
- 在人类会犯困、会停手的时间段里仍然**毫无断点地持续推进**探索。
- 从失败响应中取得线索，**系统地改写下一个请求**，是适应性推进而非随机模糊测试。

---

## 2. 防御者可观测的指纹（IoC·签名）

把指纹分成两个视角。**（甲）目标（被攻击方）视角**是可以在指向你系统的流量里看到的；**（乙）运营者·取证视角**是在 ARTEX 真实运行过（或被用作中继）的主机上可以看到的。两者不要混用：目标侧可见的静态指纹很有限，行为指纹才是关键。

### （甲）目标视角：指向我方系统的流量

- **资产补强（enrich）查询的 User-Agent `artex-enrich/1.0`。** ARTEX 在自动补强资产（确认 DNS·HTTP）时，用这个 User-Agent 直接向目标发 `GET`(`enrich/enrich.go`)。这条路径与 LLM 无关，由 ARTEX 自行产生，特征是**不跟随重定向、关闭 keep-alive、只读响应开头以提取 `<title>`**。默认并发数为 4。因此，如果多个资产同时收到带 `artex-enrich/1.0` UA 的**短连接·单次 GET·读完标题就断**的查询，强烈提示这是 ARTEX 系补强流量。但运营者可以改 UA，所以**它没有出现并不代表安全**。
- **真正的攻击流量沿用工具自身的指纹。** worker 通过真实工具（用 Bash 调起的外部工具·HTTP）发请求。ARTEX 只是往子进程环境变量里注入 `HTTP_PROXY` 与代理 CA 路径，把这些流量导过自己的记录代理，**并不会给攻击流量强加 ARTEX 专属 User-Agent**。所以目标看到的 User-Agent 与请求头，是**当时执行的那个工具的默认值**（各类命令行工具的默认 UA）。运营者若未自定义，就会留下常见自动化工具的指纹；若自定义过，也可能伪装成正常浏览器。因此**不要依赖单一 UA 匹配，要与行为检测结合使用**。
- **worker 内置的 WebFetch 工具会留下 `norma/0.4` User-Agent。** 与上面用 Bash 调起的外部工具不同，ARTEX 通过 norma SDK(`github.com/Autumn-27/norma`)直接执行的 HTTP 查询工具（WebFetch）会在攻击阶段带上该 SDK 的默认 User-Agent `norma/0.4`。在明文 HTTP 传输时（或在 TLS 终结点）可以在网络上观测到，[Suricata 规则 sid 1000003](../detections/suricata/README.md)会命中这个前缀(`norma/`)。不过这个 UA 并非 ARTEX 独有，其他使用 norma SDK 的工具也会用它，因此和 `artex-enrich/1.0` 一样，它只是辅助线索而非独立证据；用 Bash 调起的工具各用自己的 UA，所以**它没有出现并不代表安全**。
- **没有内置速率限制。** ARTEX 自身没有针对目标流量的限速，请求速率由 LLM 驱动的外部工具决定。作为替代，每个任务的 worker 默认并发 3 个，同一目标上可能有**多条意图同时进行**。也就是说，它既可能表现为“缓慢的单会话”，也可能表现为“多路并发推进”，用单一固定阈值很难抓住。
- **行为签名（最重要）。** 下面这些模式的**同时出现**指向自主智能体。
  - 同一来源（或少量轮换来源）把**侦察 → 目录·端点枚举 → 参数试探 → 认证·注入尝试**在**很短间隔内连成链**。
  - 针对同一端点连续发送、并**依据响应码与响应体长度系统性改写**的请求（是适应，不是随机模糊测试）。
  - 超出人类工作时间、**长时间无断点**持续推进的会话。
  - 遭遇失败（401/403/429）后仍不停手，继续尝试**绕行改写**的执拗。

### （乙）运营者·取证视角：ARTEX 运行过的主机

入侵调查中，需要在中继或经转主机上查找 ARTEX 安装·运行的痕迹时参考本节。

- **默认监听端口 `:8787`。** ARTEX 服务器的默认 HTTP 监听地址(`cmd/artex/main.go`，可用 `--addr` 修改)。如果内网主机在该端口上开着管理 UI（仪表盘·任务·资产图谱），就是怀疑存在 ARTEX 实例的依据。
- **记录型 MITM 代理 `127.0.0.1:8788`。** 拦截并全程记录 worker 的 Bash·HTTP 执行的本地代理默认地址(`cmd/artex/main.go` 中 `--proxy` 的默认值，仅监听回环)。它会自建 CA 来解密并记录 TLS(`mitmproxy-ca-cert.pem`)，因此主机上是否存在**由 ARTEX 安装的可信 CA 证书**、以及子进程是否被注入了 `HTTP_PROXY` 与代理 CA 路径环境变量，都是线索。这组注入会施加到 ARTEX 拉起的所有 worker 工具上，而且变量名硬编码在源码里(`agent/worker.go`)，所以**运行中的进程是否同时带有代理变量与 mitmproxy CA 信任变量**，比单个监听端口更具特异性。[主机分诊脚本](../detections/triage/README.md)会在 `/proc`（或取证镜像里捕获的环境变量转储）中核对这一组合。
- **self-update 出站 `artex-selfupdate`。** 自动更新查询 GitHub Release 时使用的 User-Agent(`selfupdate/`)。如果出站日志里出现以该 UA 访问代码托管站的请求，就说明这台主机上运行过 ARTEX 二进制。
- **PostgreSQL 上的双图谱。** 含有 `exploration_nodes`·`assets`·`companies`·`activity` 等表以及 `agent_prompts` 种子的数据库，是 ARTEX 数据存储的特征。
- **基于数据库的正则审批规则与审计日志。** 评估工具调用的 intercept 规则存放在数据库里并按优先级做正则求值(`intercept/`)。被拦截的调用会以 `【ARTEX 平台管控·非目标防御】` 开头的管控措辞记入审计日志(`GET /api/audit`)，因此在被入侵主机的审计记录中看到该字符串可以佐证 ARTEX 运行过。
- **破坏性命令狩猎指标。** ARTEX 自身护栏拦截的命令模式，反过来就是自主智能体**可能会尝试**的命令集合。请把下列内容作为主机命令审计中的狩猎指标：`rm -rf`、`mkfs`、`dd of=/dev/`、`shred`/`wipe`、SQL 的 `DROP DATABASE`/`DROP TABLE`/`TRUNCATE`、MongoDB 的 `drop`/`dropDatabase`、Redis 的 `FLUSHALL`/`FLUSHDB`、`curl`/`wget` 的 `-X DELETE`，以及 `curl … | nc …` 一类数据外传管道。只是最后这类外传管道并不是破坏数据的命令，而是把数据往外送的外泄信号，性质不同。与前面的破坏性模式不同，ARTEX 护栏对这一条默认保持关闭（因为在正常的调试用反向 shell 或数据传输管道上误报很频繁），用于部署的破坏性命令狩猎规则([`destructive_command_hunting.yml`](../detections/sigma/destructive_command_hunting.yml))也只限定在破坏范围内、不含该模式，因此不要把外传管道直接用于拦截，而应作为独立狩猎指标并按自身环境调优。

> 小结：**目标侧防御要挂在行为指纹上，静态 UA(`artex-enrich/1.0`·`norma/0.4` 等)只作辅助线索**。运营者·取证指纹(`:8787`·`127.0.0.1:8788`·`artex-selfupdate`·DB schema·审计日志措辞)在**调查被入侵的经转主机时**有效。

### 为什么封禁 IP 是脆弱的初级防御

安全事件曝光后，社交媒体与论坛上常出现“共享攻击 IP 列表，在防火墙上封禁”的帖子。与分享者的善意相反，**把来源不明的非官方 IP 列表直接写进封禁规则并不值得推荐**，在自主 AI 攻击场景下尤其如此。

- **难以验证来源。** 个人发布的非官方列表无法确认是谁、依据什么收集，混入了与事件无关的 IP 也没有筛除手段。
- **很快过时。** 自主智能体会经由 VPN、云实例、被窃取的中继服务器频繁更换出口 IP（见上文（甲）节的“少量轮换来源”），昨天观测到的攻击 IP 今天很可能已被丢弃，即使封了列表，攻击者换个 IP 就回来了。
- **误封风险很大。** 列表里若混有共享网段、CDN、正常云服务的 IP，一旦封禁就会连带切断正当客户流量乃至内部服务。再叠加自动封禁（见第 6 节），误封造成的损失会扩散得更快。

这并不意味着封 IP 毫无用处。只有**从应急处置机构或可信威胁情报渠道取得官方 IoC，先评估有效期限与误封风险再落地**时，它才有意义。但逐条封 IP 只是追赶不断轮换的出口地址的临时措施，真正经得起时间的是难以改变的**行为**（第 2~4 节的行为基检测）与**攻击面收敛**（第 3·5 节的加固：清理暴露资产·打补丁·MFA）。本指南的前提——“指纹可以改，行为难藏”——在这里同样成立。

---

## 3. 攻击者瞄准的入口与加固

自主智能体盯上的是和人类攻击者**相同的弱点**，只是更快、更执拗地反复尝试。以下是从防御视角出发的优先加固点。

### 3.1 外部暴露的攻击面与已知（n-day）漏洞

自主智能体最先、也最稳定地瞄准的入口，并不是巧妙的 0-day，而是**暴露在外部却尚未打补丁的已知漏洞**。边界设备（VPN·防火墙）、对外开放的管理与运营控制台、应用服务器·中间件·框架（例如被广泛滥用的 WebLogic·Struts 系），以及**并非主业务、而是为合作·招商·员工接入的辅助系统**，都是典型靶标。自主智能体用资产图谱自动枚举暴露面，随后针对成百上千个资产，同时压上已有公开 PoC 的 n-day,**在补丁落地之前**完成利用。找出暴露弱点并套入利用代码的这一速度，正是它与人类攻击者形成不对称的地方。

- **压缩攻击面。** 持续维护对外暴露的资产·管理控制台·辅助系统的清单，把并非必须对外的部分挪到内网、VPN 或允许名单之后。
- **快速修补已知漏洞。** 边界设备·Web 服务器·应用服务器·中间件的公开漏洞（n-day）是自主智能体的首选目标，因此要先补已有公开 PoC 流出的组件，把“漏洞公开”与“补丁落地”之间的间隔压到最短。先处理哪些，可以把汇总真实滥用观测的 [CISA KEV（已知被利用漏洞）目录](https://www.cisa.gov/known-exploited-vulnerabilities-catalog)作为优先级输入，并与 7.1 的国内建议（KISA 的 Boho Nara）交叉核对。与其把某个 CVE 编号写死在文档里，不如以这类持续更新的官方清单为准，这样即使自主智能体把目标换成新出现的 n-day，你也不会落后。
- **收紧不得不暴露的接口。** 对必须对外开放的管理·运营接口，叠加来源限制（IP 允许名单）·MFA·VPN，从根上阻止未认证枚举。
- **按与主业务同等的标准管理辅助系统。** 合作·招商·员工用辅助系统也要保持与主业务一致的补丁与监控水平。自主智能体真正撬开的入口往往不是主业务，而是这类辅助路径。认证流程本身的加固见下面的 3.2。

从检测角度看，如果外部来源针对某个已知漏洞的具体路径（URL·参数）在很短间隔内连续发请求，就是 n-day 扫描的信号。这类信号与第 2 节的行为指纹、第 4 节的同源多阶段关联规则结合起来时最容易显现。

### 3.2 辅助认证与身份验证流程

附加服务·合作·招商渠道这类**走与主业务不同路径接入的认证与身份验证流程**，校验往往更宽松，因此会成为绕行目标。自主智能体会自动枚举这类路径，读取响应差异来系统性地寻找绕过条件。

- 把身份验证·认证各步骤的强度**统一到与主业务一致**，并全量检查可被绕过的辅助路径。
- 认证状态的迁移（未登录 → 已登录，普通 → 提权）必须在**服务端重新校验**，不要直接采信客户端送来的信任标记（cookie·头部·参数）。
- 检查身份验证令牌·一次性码的**有效期、可复用性与可猜测性**。

### 3.3 API 认证与授权（IDOR·权限提升）

- 对每一次对象访问强制**服务端归属与权限校验**（阻止只改标识符就能打开他人资源的 IDOR）。
- 枚举水平与垂直提权路径。自主智能体会机械地递增递减标识符并大规模尝试，因此**单条手动测试容易漏掉的缺口**会很快被它找出来。

### 3.4 凭证填充（credential stuffing）

用泄露的账号口令列表做撞库，会被自主智能体以**速度和分布式**放大。

- 在登录·身份验证端点上加**自适应限流**（基于 IP·账号·设备·行为）。
- 对敏感操作强制**多因素认证（MFA）**。即使撞库把密码猜对了，二次认证仍能挡住。
- 用**凭据泄露检测**（比对已知泄露名单、异常登录地域与速率）提前拦截。
- 对登录失败与成功的**分布变化**（突然出现的低速·大范围尝试）告警。

### 3.5 会话·令牌·密钥管理

- 会话令牌的**范围·有效期·刷新**都要最小化，每次敏感迁移都重新认证。
- API key 与内部令牌**不得出现在响应、日志与错误信息里**（自主智能体会主动从错误响应中收集线索）。

---

## 4. 检测规则与日志模式（实战）

下面以不绑定具体产品的**伪规则**形式书写，请移植成你自己的 WAF·IPS·SIEM 语法。上述规则中基于静态指纹的部分，已提供可直接部署的 [Sigma 规则(`detections/sigma/`)](../detections/README.md)；核心的行为与关联检测（4.1·4.2）虽然不能完全归约为单条规则，但其中已在 ARTEX 代码里核实过的行为指标，也已提供可直接部署的 [Sigma 关联规则(`detections/sigma/correlation/`)](../detections/README.md)，包括补强查询速率、补强查询目标主机数、护栏拦截突发、护栏标记与破坏性命令在同一主机同时出现。只有那类纯 Web 的多阶段关联（枚举 → 试探 → 认证），其流量无法归约为某个 ARTEX 独有 User-Agent，因此需要环境专属的基础规则。这一关联已经放在下面的 4.2 中，以可直接部署的通用行为基 Sigma 基础模板形式给出，请按你自己的 SIEM 与基线调优后作为起点。网络层在明文 HTTP 传输时（或在 TLS 终结点）可观测到的 ARTEX User-Agent 共有两个，两者都已提供 [Suricata 规则(`detections/suricata/`)](../detections/suricata/README.md):一个是 enrich 探测器的 `artex-enrich/1.0`（存在性签名 sid 1000001 与高速枚举变体 sid 1000002），另一个是 norma SDK 的 WebFetch 工具在攻击阶段发出的 `norma/0.4`(sid 1000003)。

### 4.1 WAF·IPS（基于行为）

- 当同一来源把**性质不同的多组请求**（静态资源请求占比低，枚举·参数试探·认证尝试占比高）以**一个会话**串联起来时，提高其分值。
- 对**随响应码与响应体长度作出反应而改写自身**的连续请求（熵值高、但并非随机的适应模式）加大权重。
- 对 `artex-enrich/1.0` 这类已知自动化 UA 直接打上**高风险**标签，但不把“没有该 UA”解读为安全。

### 4.2 SIEM 关联规则

- **同源多阶段关联**:同一 IP/ASN/会话里，(a) 目录·端点枚举、（b） 参数试探、（c） 认证/注入尝试在**同一个短窗口内全部**被观测到，即产生“疑似自主攻击”告警。
- **时段异常**:偏离服务正常流量分布、**长时间无断点**持续推进的单会话。
- **失败后仍持续**:收到 403/429 之后仍不停手、继续**绕行改写**的来源。

下面给出可直接部署的**同源多阶段关联**基础模板。攻击流量里没有 ARTEX 独有 User-Agent，所以这个模板与 `detections/sigma/` 下基于 ARTEX 源码的规则不同，是**通用行为基规则**:它不看某个攻击工具的指纹，只看“一个来源在短窗口内同时做了枚举·试探·认证”这一行为。三个子规则与一条 temporal 关联规则（只有当同一客户端在时间窗内同时满足三者才触发）写在同一个文件里。
```yaml
# ── 通用行为基模板（并非 ARTEX 专属签名） ──
# 防御指南 4.2 节的同源多阶段 Web 攻击模式（枚举 → 试探 → 认证）。
# ARTEX 的攻击流量里没有 ARTEX 指纹，因此与 detections/sigma/ 下的规则不同，这里不是
# 用 ARTEX 源码固定依据的规则，而是通用行为基的起点。字段名（SigmaHQ 的 Web 服务器分类）
# 与阈值·时间窗务必按你自己的日志与基线调整。
# 自足：三个子规则 + 一条 temporal 关联规则，只有同一客户端在窗口内同时满足三者才触发。
title: Web Endpoint Enumeration Burst From One Source
id: f03c360c-dc33-4a8a-afa8-821b1ff5c4e3
status: experimental
description: |
    Stage 1 of the same-source multi-stage pattern in the ARTEX defense guide section 4.2: a
    burst of endpoint or directory enumeration from a single client, seen as a high rate of 404
    and 400 responses in a short window. This is generic behaviour, not an ARTEX-specific
    signature; tune the count and window to your own baseline. On its own this leg is low signal
    and earns weight only inside the correlation below.
references:
    - https://github.com/jiwoochris/artex-global/blob/main/docs/defense-ko.md
    - https://github.com/jiwoochris/artex-global/blob/main/docs/defense-en.md
author: artex-global defense guide (generic template)
date: 2026-10-07
tags:
    - attack.reconnaissance
    - attack.t1595
logsource:
    category: webserver
detection:
    enum_misses:
        sc-status:
            - 404
            - 400
    condition: enum_misses
falsepositives:
    - Broken links, authorised vulnerability scanners, or misconfigured clients that generate
      many 404 responses.
level: low
---
title: Web Parameter Or Path Injection Probe From One Source
id: f9296e55-6b7a-4030-b5f7-5f7b146233be
status: experimental
description: |
    Stage 2 of the same-source multi-stage pattern: parameter or path probing, matched here as
    query strings carrying common injection or traversal markers. This leg is unavoidably
    signature-like and noisy on its own, so it is scored low and earns weight only inside the
    correlation below. Extend the marker list to your own probe corpus and WAF categories; it is
    a coarse proxy for the broader "adapts requests to responses" behaviour the guide describes.
references:
    - https://github.com/jiwoochris/artex-global/blob/main/docs/defense-ko.md
    - https://github.com/jiwoochris/artex-global/blob/main/docs/defense-en.md
author: artex-global defense guide (generic template)
date: 2026-10-07
tags:
    - attack.initial-access
    - attack.t1190
logsource:
    category: webserver
detection:
    probe_markers:
        cs-uri-query|contains:
            - '../'
            - "' or "
            - ' union select '
            - '<script'
            - '; drop '
    condition: probe_markers
falsepositives:
    - Legitimate request payloads that resemble probe markers; tune the marker list to your
      application.
level: low
---
title: Authentication Or Identity-Verification Attempt From One Source
id: 2614cacb-7455-46ad-9af2-7b9633f12d7b
status: experimental
description: |
    Stage 3 of the same-source multi-stage pattern: requests to login, authentication, or
    identity-verification endpoints, or 401 and 403 responses. Map the paths and your own
    authentication-event fields to your application; auxiliary, affiliate, and broker channels
    often expose weaker identity-verification endpoints than the main service and belong here
    too. This leg is broad by design and is only meaningful inside the correlation below.
references:
    - https://github.com/jiwoochris/artex-global/blob/main/docs/defense-ko.md
    - https://github.com/jiwoochris/artex-global/blob/main/docs/defense-en.md
author: artex-global defense guide (generic template)
date: 2026-10-07
tags:
    - attack.credential-access
    - attack.t1110
logsource:
    category: webserver
detection:
    auth_path:
        cs-uri-stem|contains:
            - '/login'
            - '/auth'
            - '/verify'
            - '/otp'
    auth_deny:
        sc-status:
            - 401
            - 403
    condition: auth_path or auth_deny
falsepositives:
    - Ordinary users signing in; this leg is broad and only meaningful inside the correlation.
level: low
---
title: Same-Source Multi-Stage Web Attack (Enumeration, Probe, Auth)
id: 9b7c7b86-702f-42b8-be99-3e60a188ec5b
status: experimental
description: |
    The behaviour-based core of ARTEX defense guide section 4.2 as a deployable template: one
    client runs endpoint enumeration, parameter or path probing, and an authentication or
    identity-verification attempt within the same short window. This is the pattern an autonomous
    agent drives at machine speed and keeps driving past 403 and 429 responses. It is UA-free and
    carries no ARTEX fingerprint, so it is a GENERIC behavioural rule, not one of the
    ARTEX-source-grounded rules under detections/sigma/. Normalise the client field (c-ip, or a
    session identifier if you have one) and tune the window to your baseline. If three legs are
    too strict and miss cases, relax to any two of the three.
references:
    - https://github.com/jiwoochris/artex-global/blob/main/docs/defense-ko.md
    - https://github.com/jiwoochris/artex-global/blob/main/docs/defense-en.md
author: artex-global defense guide (generic template)
date: 2026-10-07
tags:
    - attack.initial-access
    - attack.t1190
correlation:
    type: temporal
    rules:
        - f03c360c-dc33-4a8a-afa8-821b1ff5c4e3
        - f9296e55-6b7a-4030-b5f7-5f7b146233be
        - 2614cacb-7455-46ad-9af2-7b9633f12d7b
    group-by:
        - c-ip
    timespan: 10m
falsepositives:
    - An authorised vulnerability scan or QA run from a single source; allow-list its address.
level: high
```

使用这个模板时要注意以下几点。

- 这个块用的是与检测包同一套工具验证过的。按 SigmaHQ 全套规范运行 `sigma check`，以 0 错误、0 问题通过，并且 `sigma convert -t splunk` 能生成查询(三个子规则在 10 分钟窗口内按 `c-ip` 分组，三者全部满足即触发)。只是它无法用 ARTEX 源码固定依据，因此没有放进 `detections/` 中受测试的规则树，以维护该树“只收录在源码中核实过的东西，而不是推测”的原则。这些事实不是只测一次的数值：[`detections/tests/check-defense-template.sh`](../detections/tests/check-defense-template.sh) 会在本指南每次改动时于 CI 中重新运行(`sigma check` 通过、splunk·eql·loki 能转换、lucene·kusto 拒绝关联、韩文版与英文版的规则正文逐字节一致)，以防上述说法因上游 `sigma-cli` 变化而悄悄过期。
- 该模板是关联（correlation）规则，`sigma convert` 能否整体导出模板、还是只能导出三个子规则，取决于后端是否支持 Sigma 关联转换。用同一固定版本(`sigma-cli` 3.1.0)实测：整个模板在 Splunk(`-t splunk`)·Elasticsearch EQL(`-t eql`)·Grafana Loki(`-t loki`)上可以转换。而在 Microsoft `kusto` 后端（Sentinel·Defender）与 Elasticsearch Lucene(`-t lucene`)上关联规则不会转换(报 `Backend does not support correlation rules`)，这时只转换三个子规则，并在产品侧自行表达“10 分钟窗口·同一 `c-ip`”的关联(例如 Sentinel 的定时分析规则 `summarize ... by bin(TimeGenerated, 10m), <客户端>`)。这与检测包记录的移植性一致，实测支持矩阵见 [Sigma 后端移植性](../detections/README.md)。
- 第 2 阶段（试探）依赖注入·路径穿越标记列表，是较粗的信号，单独使用误报很多。所以三个子规则的 `level` 都设得较低，只有在同源三者同时出现的关联规则里才升为高强度告警。
- 客户端按 `c-ip` 分组。如果位于代理或 CDN 之后，请改用从 `X-Forwarded-For` 还原的真实客户端地址，或会话标识符。如果要求三个阶段全部满足过于严格、出现漏报，可放宽为满足三者中的两个即触发。

### 4.3 认证日志

- 按账号·IP 统计的**登录失败率突变**、跨越大量账号的**低速分布式尝试**（撞库的典型特征），以及从失败到成功的**异常转换速度**。
- 对身份验证·一次性码端点的**枚举式访问**。

### 4.4 出站（egress）与取证

- 内网主机以 `artex-selfupdate` UA 访问代码托管站的请求。
- 内网有进程绑定 `:8787`（管理 UI）·`127.0.0.1:8788`（记录代理）。
- 以 `artex-enrich/1.0` UA 在短时间内对大量外部资产做 DNS·HTTP 补强查询的模式。

---

## 5. 加固清单

以下条目供防御团队直接照单检查。

- [ ] 登录·身份验证·敏感 API 已应用**自适应限流**（IP·账号·设备·行为）。
- [ ] 敏感操作强制 **MFA**。
- [ ] 所有对象访问都有**服务端归属与权限校验**（阻止 IDOR）。
- [ ] 认证状态迁移在**服务端重新校验**，不直接采信客户端信任标记。
- [ ] **辅助·合作·招商渠道**的身份验证强度已与主业务统一。
- [ ] 运营着泄露凭据的**侵入检测与比对**。
- [ ] WAF 以**行为基模式**运行，不只依赖固定签名。
- [ ] **不直接套用流传的非官方 IP 封禁列表**，而是从可信渠道取得官方 IoC，评估有效期限与误封风险后再落地。
- [ ] SIEM 中已加入**同源多阶段关联规则**。
- [ ] 认证·访问·出站**日志保存足够长的期限**（自主攻击很快，事后追溯非常依赖这些材料）。
- [ ] 维护对外暴露的资产·管理控制台·辅助系统的**清单以压缩攻击面**，并对边界设备·应用服务器·中间件的**已知（n-day）漏洞快速打补丁**。
- [ ] 用**网络分段**缩小横向移动与权限提升的爆炸半径。
- [ ] 密钥（键·令牌）**不出现在响应、日志与错误信息里**。
- [ ] 已准备好**自动拦截·隔离**处置（只等人批准就跟不上自主攻击的速度）。

---

## 6. 事件响应概要

自主 AI 攻击的特征是**速度**。一个人要花一年完成的渗透与外传，智能体可以在短得多的时间里跑完。响应设计也要以这一速度为前设。

- **把自动拦截前置。** 让可疑来源隔离·会话失效·速率骤降这类措施可以在人工批准之前**自动触发**。若全部绑在人工审批环里，就跟不上攻击速度。
- **提前确定要保留哪些日志。** 认证日志、访问日志（在允许范围内包含请求体）、出站日志、DNS 查询。自主攻击会很快累积痕迹，事后要复原攻击链就得靠这些材料。
- **按资产为单位追踪入侵范围。** 攻击会沿资产图谱扩散，所以要把从最初入口资产到横向移动、权限提升的**整条路径**复原出来，才能阻止再次入侵。

### 6.1 可疑主机与流量分诊（triage）流程

下面按顺序整理怀疑与 ARTEX 相关时最先要确认的事项。与第 2 节把指纹分成两个视角一样，分诊也分成**（甲）我的服务是否已成为 ARTEX 的目标**与**（乙）某台主机上是否运行过 ARTEX**。任何一步都不要凭单个命中下结论，要看多个指标与行为信号是否一起出现。即使静态指标全都没有，只要出现行为信号就继续调查。

**（甲）目标侧：我的服务是否已被 ARTEX 盯上**

1. 在访问·认证日志中查询补强查询 User-Agent `artex-enrich/1.0`。确认是否有不跟随重定向的单次 `GET` 查询在很短间隔内同时打到多个资产（第 2 节（甲））。运营者可以改 User-Agent，所以即使没有命中也要继续下一步。
2. 找出同一来源（或少量轮换来源）发出的**多阶段连锁**:从侦察到端点枚举、参数试探、认证·注入尝试在很短时间内接连出现，并随响应码·响应长度作出调整，401·403·429 之后仍不停手地尝试绕行改写。这类行为信号比静态 User-Agent 留存得更久（第 2 节（甲）“行为签名”）。
3. 如果运行着 SIEM，就把这一行为挂到 [Sigma 关联规则](../detections/README.md)（补强查询速率·目标数·护栏拦截突发·护栏标记与破坏命令同时出现）上，并按上面“把自动拦截前置”的原则，把命中的来源列为隔离·会话失效对象。

**（乙）主机取证：某台主机上是否运行过 ARTEX**

在可疑主机上确认下列各点。指标的依据见第 2 节（乙）与机器可读的[指标清单](../detections/indicators/artex_indicators.csv)。下面五项只读检查中，前四项（监听端口·出站日志·审计日志·状态与记录存储）可由[主机分诊脚本](../detections/triage/)(`detections/triage/artex_host_triage.py`)一次代跑。第五项命令审计不代跑：破坏性命令并非 ARTEX 专属指纹，正当管理员也会使用，属于狩猎线索而非自动指标，因此需要在主机命令历史中自行比对。在没有 SIEM、仅有 shell 访问时先跑脚本，每个命中都按下面的说明只当线索处理。

1. **监听端口。** 在主机上直接确认默认服务器端口 `:8787` 与回环记录代理 `127.0.0.1:8788` 是否打开。
   ```sh
   ss -ltnp | grep -E ':8787|:8788'   # 没有 ss 时改用 netstat -ltnp
   ```
   两个端口都可以被 `--addr`·`--proxy` 标志改掉，所以即使这条查询为空，也要一并查看全部监听端口与是否有内部管理 UI 在运行。
2. **出站日志。** 查发出站日志中是否有以自动更新 User-Agent `artex-selfupdate` 访问代码托管站（GitHub Release）的请求(`selfupdate/`)。这提示该主机上运行过 ARTEX 二进制。
3. **审计日志。** 审计记录里若有护栏管控标记 `【ARTEX 平台管控·非目标防御】`，即可佐证 ARTEX 运行过(`guard/guard.go`)。每次被拦截的工具调用都会以这段措辞留下记录。
4. **状态与记录存储。** ARTEX 在 PostgreSQL 中保存探索图谱(`exploration_nodes`·`assets`·`companies`·`activity` 表与 `agent_prompts` 种子)，并把状态与记录写在可执行文件旁的数据目录(`cmd/artex/main.go` 中 `--data` 的默认值)。该目录下一层有存放各任务产出的 `tasks/`、存放对话记录的 `transcripts/` 子目录，而记录代理的产出单独汇总在其中的 `traffic/` 子目录(`server/manager.go` 会在数据目录下打开 `traffic/` 作为记录代理存储)。因此可信 CA 证书在 `traffic/_ca/mitmproxy-ca-cert.pem`，流量索引在 `traffic/_index/index.sqlite`，记录下来的请求·响应体在 `traffic/_blobs/`。这三者与 `tasks/`·`transcripts/` 同时出现，则记录代理确实运行过的旁证就更强。
5. **命令审计。** 把破坏性命令狩猎指标(第 2 节（乙）末尾的 `rm -rf`·`DROP DATABASE`·`FLUSHALL`·外传管道等)与主机命令历史比对。正当管理员也会用同样的命令，因此只当线索处理。

静态指标（端口·User-Agent·标记）都可能被运营者改掉或清除。因此**没有出现并不代表安全**，把（甲）的行为信号与（乙）的主机痕迹合起来判断，才是自主 AI 攻击分诊的关键。

---

## 7. 官方渠道：IoC、安全建议与申报义务

> 本节沿用上游韩国本地化版本的内容，其中的机构与法律均为韩国口径，请在对照本国（地区）规定时替换为你所在辖区的官方渠道与法定义务。

相关防御团队应从官方渠道获取 IoC 与安全建议，事件发生后还要履行法定的申报义务。请以下列官方来源为一手依据，而不是流传的非官方列表。

### 7.1 获取 IoC 与安全建议的渠道

- **KISA（韩国互联网振兴院）的 [Boho Nara·KrCERT/CC](https://www.boho.or.kr)** 提供安全建议、漏洞公告与入侵事件响应信息，并通过网络威胁信息分析共享体系（C-TAS，仅限加入机构之间共享，与公开门户不同需另行申请）在机构间共享威胁信息。
- **[金融安全院（FSI）](https://www.fsec.or.kr)** 跨金融业界共享入侵与威胁信息（金融领域 ISAC）。若属金融行业，建议同时关注该渠道。
- **[个人信息保护委员会](https://www.pipc.go.kr)** 公开个人信息泄露申报标准与保护措施相关的告示与指南。

第 5 节加固清单中“从可信渠道取得官方 IoC”指的正是这些渠道。即使是官方 IoC，也要先评估有效期限与误封风险再落地，这一原则与第 2 节“为什么封禁 IP 是脆弱的初级防御”所述一致。

### 7.2 韩国法律上的申报义务

自主攻击扩散得很快，请把法定申报环节预先写进第 6 节的事件响应流程里。下面只是要点，准确的适用对象、期限与要件请以各机构的最新告示为准。

- **个人信息泄露**:依据《个人信息保护法》第 34 条，应在知悉泄露时起 72 小时内向个人信息保护委员会或 KISA 申报，并通知信息主体（信息主体 1 千人以上、敏感信息或唯一识别信息的泄露、外部非法访问导致的泄露等属于申报要件）。
- **入侵事件**:依据《信息通信网利用促进及信息保护等相关法律》，信息通信服务提供者应在知悉入侵事件后 24 小时内向科学技术信息通信部与 KISA(KrCERT/CC)申报。
- **金融公司**:按金融领域监管规定，可能还需向金融监督院或金融安全院等主管机构另行报告，请一并核对相应规定。

实际申报时，入侵事件可通过 [Boho Nara](https://www.boho.or.kr) 或免区号 118（KISA 网络信访中心）提交，个人信息泄露可通过个人信息保护委员会的[个人信息门户](https://www.privacy.go.kr)提交。申报期限很短，请把负责人与联络渠道预先写进第 6 节的事件响应流程。

---

## 参考

- 原始项目：[Autumn-27/ARTEX](https://github.com/Autumn-27/ARTEX)（AGPL-3.0 · 目前无法访问，2026-10 确认）。本文档是该全球版仓库的防御资料。
- 上级 README 中的[安全与滥用警告、使用范围与法律提示](../README.md)。
- 本版本是面向中文读者的本地化版本。与个人信息相关的情形，同时适用上述上位法域的相关法规；无论如何，未经授权的检查在绝大多数司法辖区都构成犯罪，务必先取得书面授权与商定的范围再开始。
- 一般 Web 安全加固的标准参考：[OWASP Top 10](https://owasp.org/www-project-top-ten/)、[OWASP ASVS（应用安全验证标准）](https://owasp.org/www-project-application-security-verification-standard/)、[OWASP API Security Top 10](https://api-security.owasp.org/)。

> 本指南会持续补充，以帮助提升防御与检测能力。欢迎通过仓库 issue 提交检测规则与加固项建议。