# ARTEX 检测规则（Suricata / 网络层）

中文 · [English](README.en.md) · [한국어](README.ko.md)

> 本目录把[防御·检测指南](../../docs/defense-zh.md)第 2·4 节的网络观测指纹，整理成可直接部署的 [Suricata](https://suricata.io) 规则。日志·主机层由 [`../sigma/`](../sigma/)(Sigma)负责。所有规则只能用于**防御·检测目的**，保护你自己拥有或已获得书面许可的系统。

这是 [Sigma 规则](../sigma/) 的网络层配套。这些 [Suricata](https://suricata.io) 签名覆盖线上可观测的两个 ARTEX 工件，每个指标都以本仓库源码中核实过的字符串或行为为依据，而不是推测。主机·日志·SIEM 层在 [`../sigma/`](../sigma/) 下；防御指南（[中文版](../../docs/defense-zh.md)、[英文版](../../docs/defense-en.md)）解释了整体图景。

## 规则 —— [`artex.rules`](artex.rules)

- **sid 1000001** —— `ARTEX enrichment prober User-Agent`。入站 HTTP `GET`，其 User-Agent 以 `artex-enrich/` 开头 —— 即资产补强探测器(`enrich/enrich.go:233`)。这是单请求层面的存在性指标。`classtype: attempted-recon`。
- **sid 1000002** —— `ARTEX enrichment prober high-rate enumeration`。同一 User-Agent 越过 `detection_filter` 速率（**每来源 300 秒内 30 次请求**），即单请求规则会漏掉的机器级速度。与 Sigma 关联规则 `artex_enrich_scan_velocity` 对应。`classtype: attempted-recon`。
- **sid 1000003** —— `ARTEX worker WebFetch User-Agent`。入站 HTTP 请求，其 User-Agent 以 `norma/` 开头 —— norma SDK 的 WebFetch 工具(`github.com/Autumn-27/norma/tool/webfetch.go`)。该 UA 在所有 norma 版本（v0.1.0–v0.4.3，已核实）中都是硬编码的，并经由不修改请求头的记录代理(`traffic/traffic.go`)到达目标。与补强探测器不同，它在**攻击阶段**（主动漏洞试探）触发。`classtype: attempted-recon`。

## 范围与如实说明 —— 部署前必读

- **两个 ARTEX User-Agent 能在网络上观测到。** 补强探测器在侦察阶段发送 `artex-enrich/1.0`(`enrich/enrich.go:233`);norma SDK 的 WebFetch 工具在攻击阶段发送 `norma/0.4`(`github.com/Autumn-27/norma/tool/webfetch.go`)。记录代理(`traffic/traffic.go`)不修改请求头，因此两个 UA 都会原样抵达目标。其他 worker 工具(Bash 子进程如 `curl`、`nmap`)使用各自的 User-Agent —— 这些请用通用扫描器签名与 [`../sigma/`](../sigma/) 下的行为基 SIEM 规则来检测。
- **User-Agent 只在明文下可见。** 它出现在明文 HTTP，或被 TLS 终结代理/WAF 检视的位置。端到端 TLS 会把它加密，所以请部署在真正能看到 HTTP 请求缓冲区的地方。
- **静态 User-Agent 可以被运营者改掉**，因此它没有出现**不代表**安全。经得起时间的是行为 —— 速率与广度 —— 这也是 sid 1000002（以及 Sigma 关联层）以速度为依据的原因，也是纯 Web 多阶段检测需要环境专属基础规则的原因。
- **刻意省略的内容。** 自动更新 User-Agent `artex-selfupdate` 经 HTTPS 发往 GitHub，网络侧不可观测（仅凭 TLS SNI 太常见，不宜告警）。审计管控标记是运营侧日志工件，不是面向目标的流量 —— 请用 [`../sigma/artex_guard_audit_framing.yml`](../sigma/artex_guard_audit_framing.yml) 检测。服务器端口 `:8787` 与记录代理 `127.0.0.1:8788`(`cmd/artex/main.go`)属于主机取证(`ss`/`netstat`)，不是网络签名。
- **有一个指标依托于固定的上游依赖，而不是本仓库源码。** 开头一句说每个指标都在本仓库源码中核实过，sid 1000003 是唯一的例外。它的 `norma/` User-Agent 硬编码在 norma SDK(`github.com/Autumn-27/norma/tool/webfetch.go`)里，ARTEX 在 `go.mod` 中固定为 `v0.4.3` 而非把该库 vendor 进本仓库，因此该字符串的依据是这个固定的上游依赖（在 v0.1.0–v0.4.3 中硬编码，已核实），不是这里的某个文件。这也是指标来源测试会重读 ARTEX 自己的发送方、却不重读它的原因。[指标清单](../indicators/README.md)写明了同样的提示，这里也再说明一次，好让上文“已在本仓库源码中核实”的说法在不留隐含例外的前提下成立。

## 验证与测试

已用 Suricata 8 验证。加载测试不需要任何流量，并且总会运行：

```sh
# 语法 + 引擎加载测试(期望输出:"Configuration provided was successfully loaded")
docker run --rm -v "$PWD/detections/suricata":/r -w /r jasonish/suricata:latest \
  suricata -T -S artex.rules -l /tmp --init-errors-fatal
```

`--init-errors-fatal` 会让“能解析但初始化失败”的规则也成为硬错误，因此加载测试不可能在签名被悄悄丢弃的情况下通过。

要确认规则真的会触发，可用 [`../tests/suricata/`](../tests/suricata/) 里可复现的回归测试。它先跑同一个加载检查，再用 scapy 合成确定性抓包，对其运行 `suricata -r`，并断言告警数量 —— 只需要 Docker:

```sh
detections/tests/suricata/run.sh
```

它断言 sid 1000001 对每次探测恰好触发一次（35 条流的抓包中 35 次），sid 1000002 越过“300 秒内 30 次”的速率（Suricata 8.0.7 上为 **5** 条告警，流 31–35），以及同一抓包换成正常浏览器 User-Agent 后产生 **0** 条告警 —— 从而确认签名的特异性。参见 [`../tests/README.md`](../tests/README.md)。若想对自己的流量做检查，可对本地服务器做一次回环 `curl -A 'artex-enrich/1.0'` 抓包，然后读取告警：

```sh
suricata -r enrich.pcap -S artex.rules -l out && \
  grep -c '"signature_id":1000001' out/eve.json    # 存在性:每次探测一条
```

## 贡献

欢迎提交检测贡献。新规则应让每个指标都以可观测事实为依据，在注释中写明局限，干净地通过 `suricata -T`，并避免任何读起来像攻击指引的内容。参见 [`../../CONTRIBUTING.md`](../../CONTRIBUTING.md) 与 [`../sigma/`](../sigma/) / [`../README.md`](../README.md) 里的 Sigma 层。
