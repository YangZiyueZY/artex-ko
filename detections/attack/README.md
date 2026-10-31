# ARTEX ATT&CK 覆盖

中文 · [English](README.en.md) · [한국어](README.ko.md)

> 本目录把 [`../`](../) 中 ARTEX 检测规则（Sigma·Suricata）所覆盖的攻击技术，整理成 [MITRE ATT&CK](https://attack.mitre.org/) 的战术·技术**覆盖层**。每个技术都只取自规则文件的 `attack.*` 标签，其依据是在仓库源码中核实过的字符串或行为，没有凭推测加入的条目。可直接上传到 [ATT&CK Navigator](https://mitre-attack.github.io/attack-navigator/)，一眼看出哪种 ARTEX 行为由哪条规则覆盖。本层只能用于**防御·检测目的**，保护你自己拥有或已获得书面许可的系统。

这是一个 [MITRE ATT&CK](https://attack.mitre.org/) Navigator 层，把本仓库的检测规则映射到它们所标注的 ATT&CK(Enterprise)技术。它按 [Sigma 规则](../sigma/) 上的 `attack.*` 标签手工整理——每个技术都以某条规则的指标为依据，而该指标是在本仓库源码中核实过的字符串或行为——并由[一致性测试](../tests/attack/run.sh)防止覆盖层与规则彼此漂移。

- **`artex_navigator_layer.json`** —— 覆盖层本身，ATT&CK Navigator v4.5 格式。

## 分值的含义

这里的“覆盖”指“本仓库提供了一条标注该技术的检测”，不等于“该技术已被完整覆盖”。分值刻意如实反映检测强度：

- **100 —— ARTEX 专属签名或行为。** ARTEX 独有的静态指标(`artex-enrich/1.0` / `artex-selfupdate` User-Agent、护栏审计标记)，或建立在其上的行为规则（补强查询速率·扇出、护栏拦截突发）。
- **50–65 —— 通用狩猎线索。** 参照 ARTEX 护栏拦截名单镜像出来的破坏性命令狩猎。同样的命令正当管理员也会执行，因此这些规则在正常活动上也会触发；命中只能当线索，不能当归属结论。65 表示通过关联规则把命令与 ARTEX 护栏标记配对、从而提高了特异性的情形。

## 覆盖的技术

六个战术下的八个技术，各自对应标注它的规则：

- **侦察 —— T1595（主动扫描）、T1592（收集受害者主机信息）。** [`sigma/artex_enrich_user_agent.yml`](../sigma/artex_enrich_user_agent.yml)、[`sigma/correlation/artex_enrich_scan_velocity.yml`](../sigma/correlation/artex_enrich_scan_velocity.yml)、[`sigma/correlation/artex_enrich_fanout.yml`](../sigma/correlation/artex_enrich_fanout.yml)，以及 [Suricata 规则](../suricata/artex.rules)(sid 1000001 / 1000002)。
- **命令与控制 —— T1105（入站工具传输）。** [`sigma/artex_selfupdate_egress.yml`](../sigma/artex_selfupdate_egress.yml)。
- **执行 —— T1059（命令与脚本解释器）。** [`sigma/artex_guard_audit_framing.yml`](../sigma/artex_guard_audit_framing.yml)、[`sigma/correlation/artex_guard_block_burst.yml`](../sigma/correlation/artex_guard_block_burst.yml)、[`sigma/correlation/artex_guard_marker_then_destructive.yml`](../sigma/correlation/artex_guard_marker_then_destructive.yml)。
- **影响 —— T1485（数据破坏）、T1561.002（磁盘擦除：磁盘结构擦除）、T1489（服务停止）。** [`sigma/destructive_command_hunting.yml`](../sigma/destructive_command_hunting.yml)，其中 T1485 还由 [`sigma/correlation/artex_guard_marker_then_destructive.yml`](../sigma/correlation/artex_guard_marker_then_destructive.yml) 加强。
- **凭据访问 / 收集 —— T1557（中间人攻击）。** [`sigma/artex_recording_proxy_ca.yml`](../sigma/artex_recording_proxy_ca.yml) —— ARTEX 内置流量记录器为解密并记录 worker 工具流量而安装的 MITM 根 CA 工件(`traffic/traffic.go`)。属于主机·取证狩猎线索。

## 如何使用

1. 打开 [ATT&CK Navigator](https://mitre-attack.github.io/attack-navigator/)。
2. 选择 **Open Existing Layer → Upload from local**，选中 `artex_navigator_layer.json`（或直接指向本仓库的 raw 文件 URL）。
3. 带分值的技术会按检测强度着色，每个技术的注释都写明对应的规则文件与其背后的防御指南章节。

## 范围与如实说明

- **覆盖不等于完备。** 这里给某技术打分，只表示有规则标注了它，并不表示该技术的每种变体都能检出。线上流量里只有两个 ARTEX 独有的 User-Agent 可见 —— 侦察阶段的补强探测器(`artex-enrich/1.0`)与攻击阶段的 norma SDK WebFetch 工具(`norma/0.4`) —— 其余攻击流量沿用工具默认指纹，因此真正经得起时间的是行为层面的检测(见防御指南 [中文版](../../docs/defense-zh.md) 的第 1–2 节与 4.1–4.2 节；英文·韩文版本见 [`../../docs/defense-en.md`](../../docs/defense-en.md) 与 [`../../docs/defense-ko.md`](../../docs/defense-ko.md))。纯 Web 的多阶段情形仍需按你的环境编写基础规则。
- **静态指标可以被改掉。** 运营者可以设置不同的 User-Agent，因此某个被标注的指标没有出现并不意味着安全。这与规则文件里的同一句提示一致。

## 验证与贡献

运行[一致性测试](../tests/attack/run.sh) —— 只需要 Docker —— 它断言覆盖层打分的技术与战术正好等于规则上的 `attack.*` 标签，并且每个技术都落到一个真实存在的规则文件：

```sh
detections/tests/attack/run.sh
```

新增或改标签规则时，请同步更新本覆盖层 —— 若规则里有而覆盖层没有，或覆盖层里有而规则里没有，测试就会失败。参见 [`../README.md`](../README.md) 与 [`../../CONTRIBUTING.md`](../../CONTRIBUTING.md)。
