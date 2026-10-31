# 贡献指南（Contributing）

中文 · [English](CONTRIBUTING.en.md) · [韩语原文](CONTRIBUTING.ko.md)

感谢您关注 ARTEX 全球版（`artex-global`）。本文用简体中文整理了开始贡献之前需要了解的
范围、方针与流程。提交贡献前，请务必先阅读[使用范围与法律责任](#使用范围与法律责任)与
[本地化方针](#本地化方针)。

- 要报告 Bug 或提出功能建议 → 使用 [Issue 模板](https://github.com/jiwoochris/artex-global/issues/new/choose)。
- 发现翻译、本地化错误 → 使用「翻译与本地化错误」Issue 模板。
- 发现安全漏洞 → **不要提交公开 Issue**，请按 [SECURITY.md](SECURITY.md) 的流程处理。
- 所有参与者都必须遵守[行为准则（CODE_OF_CONDUCT.md）](CODE_OF_CONDUCT.md)。

---

## 使用范围与法律责任

ARTEX 是一款由 LLM 多智能体**自主**执行渗透测试的攻击安全工具。贡献者也和用户一样受
同样的范围限制。

- 验证代码时，只对**自己拥有或已获得书面明确授权**的目标，或**本地隔离环境**（例如用
  Docker 启动的、故意存在漏洞且属于本人的 OWASP Juice Shop、DVWA 等）运行本工具。
- 对于超出授权范围的真实、生产、远程系统执行扫描、探测、利用的代码，或助长此类使用的
  改动，我们一律不接受。
- 在韩国，未经授权侵入他人信息通信网或造成故障，违反《信息通信网利用促进及信息保护等
  相关法律》；所收集、暴露的个人信息适用《个人信息保护法》。详细提示见 [README](README.md#请先阅读授权用途与国内法提示)。

贡献所提交的代码、文档会被如何使用，其法律责任由执行它的用户本人承担。本仓库按
「原样（AS IS）」提供。

---

## 本地化方针

本仓库存在的意义，是在**完整保留原版 [Autumn-27/ARTEX](https://github.com/Autumn-27/ARTEX)
（目前无法访问，2026-10 确认）判断性能的同时，只把用户可见的产出换成所设置的语言**。
偏离这一方针的翻译贡献可能降低性能，因此不予接受。

- **不要翻译智能体的内部推理提示词（行为准则正文）。**必须以原文（中文）保持经过基准
  测试的行为。这段正文位于 `agent/promptcatalog.go` 与 DB 种子（`agent_prompts`）中。
  翻译会让智能体的判断产生漂移。
- **只把用户可见的产出强制为所设置的语言。**包括探测结果（`report_finding`）、事实摘要
  （`record_fact`）、最终报告、对话回复。该强制通过 `agent/prompt.go` 的
  `langDirective()` 这一代码固定尾巴，附加在各个角色的 system 提示词末尾。要改变输出
  语言，请修改该函数。
- **命令、payload、代码、URL、日志原文不翻译。**它们是分析所需的原始内容，需原样保留。
- **保留上游原文，便于对照。**上游 ARTEX 的原文是中文，本仓库也以中文为主文档语言；为便于
  与上游（upstream）仓库 diff，上游的中文 README 内容已并入中文主文档 [`README.md`](README.md)（原
  `README.zh.md` 保留为指向它的旧链接占位），UI 字符串原文保留在 `web/messages/zh.json` 中。
  英语、韩语、西班牙语、日语等其他语言的译文分别维护在对应的 `README.*.md` 与
  `web/messages/*.json` 中。
- 新增用户可见的字符串时不要硬编码，而应作为消息文件的键添加。
- **输出语言强制是提示词引导，而不是硬性限制。**`langDirective()` 只是**指示**把输出
  语言设为指定语言，并不强制固定。因此语言忠实度会随模型能力、角色与上下文而变化。验证
  本地化改动时请使用有能力的模型（前沿级）。低端、小型模型可能让报告、摘要退回原文
  （中文），所以不要只凭低端模型的输出判断翻译是否生效。使用 OpenAI 系列模型时的
  `max_tokens` 配置陷阱整理在 [README 的「模型选择与输出语言」一节](README.md#模型选择与输出语言)中。
- **跟进上游（upstream）变更的流程见维护者指南。**在上游 ARTEX 更新时，区分保留资产与
  翻译对象再行跟进，并检查翻译对称与漂移的 runbook 整理在 [MAINTAINING.md](MAINTAINING.md)。

---

## 开发环境

本项目由 **Go 后端**（在单一二进制中内嵌前端）与 **Next.js 前端**构成。

主要功能的设计意图整理在 `docs/` 的设计文档中。处理把漏洞与流量证据关联起来的功能
（报告智能体的自动绑定、`report_finding` 的 `traffic_refs` 等）时，请先阅读
[漏洞与流量证据关联设计文档](docs/finding-traffic-evidence-zh.md)（同目录下另有
`finding-traffic-evidence-en.md`、`finding-traffic-evidence-ko.md` 变体）。

### 所需版本

- Go 1.26 或以上（以 `go.mod` 为准）
- Node.js 22 或以上（以发布工作流为准）
- Docker 与 Docker Compose（用于本地运行、验证）

### 后端（Go）

本地已安装 Go 时，在仓库根目录执行下列命令。

```bash
go build ./...
go vet ./agent/
go test ./agent/
```

本地没有 Go 时，可以用 Docker 做同样的验证。把模块与构建缓存放在命名卷（named volume）
中，重复执行会更快。

```bash
docker run --rm -v "$PWD":/src -w /src \
  -v artexko-gomod:/go/pkg/mod -v artexko-gocache:/root/.cache/go-build \
  golang:1.26 sh -c 'go build ./... && go vet ./agent/ && go test ./agent/'
```

#### DB 集成测试（需要 postgres）

上面的 `go test ./agent/` 会**静默跳过**需要连接 PostgreSQL 才运行的 DB 集成测试。
`agent`、`config`、`db`、`evidence`、`llmrec`、`server` 六个包中含有必须有真实
数据库才运行的测试，如果既没有环境变量 `ARTEX_PG_DSN`、配置文件中也没有 `database`
项，这些测试就会以 `--- SKIP` 跳过，包以 `ok` 结束。所以修改这六个包之后，在不提供
DSN 的情况下验证，可能出现**本地通过（ok），而 PR 的 `go-db` 作业失败**的情况。

要在本地运行这些测试，请启动 PostgreSQL 并传入 `ARTEX_PG_DSN`。下面是与 CI 相同的
`postgres:16-alpine`，在隔离网络中启动的示例，并复用上面同样的命名卷。

```bash
# 1) 启动隔离网络与空的 postgres（镜像、账号与 CI 相同）。
docker network create artexko-db 2>/dev/null || true
docker run -d --name artexko-pg --network artexko-db \
  -e POSTGRES_USER=artex -e POSTGRES_PASSWORD=artex -e POSTGRES_DB=artex \
  postgres:16-alpine
until docker exec artexko-pg pg_isready -U artex -d artex >/dev/null 2>&1; do sleep 1; done

# 2) 传入 DSN，运行 DB 集成包（DSN 中的 host 是容器名）。
#    只想运行改过的包时，把 ./agent/ 换成 config、db、evidence、llmrec、server。
docker run --rm --network artexko-db -v "$PWD":/src -w /src \
  -v artexko-gomod:/go/pkg/mod -v artexko-gocache:/root/.cache/go-build \
  -e ARTEX_PG_DSN='postgres://artex:artex@artexko-pg:5432/artex?sslmode=disable' \
  golang:1.26 sh -c 'go test ./agent/ -count=1'

# 3) 清理。
docker rm -f artexko-pg && docker network rm artexko-db
```

CI 的 `go-db` 作业会在合并前，把这六个包**各自用独立 postgres 隔离**强制跑一遍
（`.github/workflows/ci.yml`）。修改了 DB 集成包时，建议在提交 PR 前用上面的方法自行
确认对应包。

### 前端（web）

```bash
cd web
npm ci
npm run dev          # 开发服务器
npm run build        # 生产构建
npm run build:static # 静态导出构建（含合并门禁与 TypeScript 类型检查）
npm run check        # Biome 检查与格式化检查（仅供参考，因既有债务尚未纳入合并门禁）
npm run check:fix    # 自动修复
```

本地没有 Node（所需版本 20+）或安装的原生依赖与平台不匹配时，可以用 Docker 做同样的验证。
把 `node_modules` 放在命名卷中，就不会与宿主平台专用的二进制（如 `@parcel/watcher`、
Biome）混在一起，重复执行也更快。本仓库对 web 的验证在 CI 中也采用这种方式。

```bash
docker run --rm -v "$PWD":/src -w /src/web \
  -v artexko-web-nm:/src/web/node_modules \
  node:22 bash -c 'npm ci && npm run build:static'
```

提交前的格式与检查由 Biome 管理。`lint-staged` 会针对已暂存的文件自动运行
`biome check --write`。

### 整体运行（Docker Compose）

```bash
cp .env.example .env     # 设置 POSTGRES_PASSWORD
docker compose up -d     # 启动 artex + postgres → http://localhost:8787
```

---

## 贡献流程

1. 先**开 Issue。**较大的改动最好在动手前通过 Issue 对齐方向。小修正（错别字、链接、
   明显的 Bug）可以直接提交 PR。
2. **Fork** 仓库并创建主题分支。分支名把改动性质放在前面，如 `feat/...`、`fix/...`、
   `docs/...`、`i18n/...`。
3. 编写改动并**自行运行相应范围的验证。**Go 改动要让上面的 `build`、`vet`、`test`
   通过。web 改动要让 `npm run build:static`（同时执行合并门禁与 TypeScript 类型检查）
   通过。`npm run check`（Biome）因上游带来的既有 lint 债务尚未纳入合并门禁，在
   `web.yml` 中也只作为参考步骤运行，因此不必让它全部通过；只要确认**自己的改动没有
   新增错误**即可（提交时 `lint-staged` 会自动对暂存的文件运行 `biome check --write`）。
   如果改了文档（`.md`），请用 `python3 -I scripts/check-doc-links.py` 确认仓库内的
   链接、图片引用与文档锚点（`#标题`）链接没有断。锚点会按 GitHub 相同规则从标题生成
   slug 进行比对，所以若改了标题文字却没有同时修好指向它的锚点链接，这里就会报错
   （CI 的 `docs` 工作流用同样的检查作为合并门禁）。这项检查也作为 `docs` 钩子写进
   仓库根目录的 [`.pre-commit-config.yaml`](.pre-commit-config.yaml)，执行过
   `pre-commit install` 后提交时会自动运行（只用 Python 标准库、不访问网络，无需 Docker
   即可完成）。
4. **提交 PR。**标题与说明遵循 [PR 模板](.github/PULL_REQUEST_TEMPLATE.md)，写清改了什么、
   为什么改、如何验证。改了 UI 请附截图。
5. 如果是对用户可见的改动（功能、本地化、文档、检测规则等），请在
   [变更日志（CHANGELOG.md）](CHANGELOG.md) 的 `[Unreleased]` 一节增加一行。内部重构或
   仅测试的改动可以省略。

### 提交信息

遵循现有提交历史的惯例。格式为 `type(scope): 描述`，描述使用简体中文书写。

- `type`：`feat` · `fix` · `docs` · `chore` · `refactor` · `test` · `i18n` 等
- `scope`：改动所在的领域（`agent` · `web` · `server` 等），可省略

示例如下。

```
feat(agent): 强制用户可见输出使用中文 (langDirective)
docs: 编写中文 README，原文保留在 README.zh.md
i18n(web): 仪表盘导航标签中文化
```

---

## 贡献检测规则与检测测试

本仓库在 [`detections/`](detections/) 中一并维护用于**防御、检测** ARTEX 这类自主 AI
攻击的规则。其中包括可部署的 [Sigma](https://sigmahq.io) 规则
（[`detections/sigma/`](detections/sigma/)）、面向网络的 [Suricata](https://suricata.io)
规则（[`detections/suricata/`](detections/suricata/)）、[MITRE ATT&CK](https://attack.mitre.org/)
覆盖层（[`detections/attack/`](detections/attack/)），以及可复现地证明这些规则确实会触发的
测试（[`detections/tests/`](detections/tests/)）。新增或修改检测规则时，请遵守下面的约定。
八个测试套件会从机制上强制其中很大一部分，所以只改规则而不更新测试与覆盖层，测试就会失败。

- **把每个指标锚定在可观测的事实上。**规则使用的字符串、User-Agent、行为阈值必须能在本
  仓库源码中实际确认，不能凭推测编写。请在规则内注明作为依据的源码文件（例如
  `artex-enrich/1.0` 指标可在 `enrich/enrich.go` 中确认）。指标一致性测试
  （[`detections/tests/indicators/`](detections/tests/indicators/)）会检查每个指标是否仍
  同时存在于上游源码与规则两侧，因此如果上游重新同步导致源码字符串改变，只要不同时修改
  规则，测试就会失败。修改机器可读的指标清单
  （[`detections/indicators/artex_indicators.csv`](detections/indicators/artex_indicators.csv)）
  时，也要一并更新原样包含这些指标的 MISP 事件
  （[`detections/indicators/artex_indicators.misp.json`](detections/indicators/artex_indicators.misp.json)）。
  MISP 导出测试（[`detections/tests/misp/`](detections/tests/misp/)）会强制两个文件逐行
  一致，并确认该事件是能被 pymisp 加载的有效 MISP 文档。
- **诚实写明局限。**Sigma 规则写在 `description` 中，Suricata 规则写在注释里，说明该规则
  抓不到的情况与误报可能。如果只是通用的狩猎线索（例如破坏性命令）而非 ARTEX 独有特征，
  请明确说明，以免仅凭一次命中就把攻击者断定为 ARTEX。
- **通过静态验证。**Sigma 规则必须以 SigmaHQ 验证器标准取得 0 个问题
  （`sigma check --validation-config detections/tests/sigma_lint/validators.yml`）。默认的
  `sigma check` 只运行 pySigma 核心验证器，因此标题写法、字段与日志源分类、参考链接等
  SigmaHQ 惯例只能靠这份标准筛出。有四个例外是不适用于单独规则集的 SigmaHQ 单体仓库
  惯例，其理由已写在
  [`detections/tests/sigma_lint/validators.yml`](detections/tests/sigma_lint/validators.yml)。
  Suricata 规则必须能用 `suricata -T` 干净加载。
- **同时提交可复现的测试。**用 [`detections/tests/`](detections/tests/) 下的测试证明规则
  会触发（或结构有效）。输入不要放入二进制文件，而是每次确定性地生成；与引擎版本无关的
  性质（存在触发、没有误报）要精确断言，随版本波动的数值则断言其下界并另行记录基准值。
  新增或修改 Sigma 关联规则时，后端可移植性测试
  （[`detections/tests/sigma_backends/`](detections/tests/sigma_backends/)）会确认该规则能在
  多个后端转换，请保持与 [`detections/README.md`](detections/README.md) 的后端支持说明
  一致。
- **同时更新 ATT&CK 覆盖层。**给规则添加或修改 `attack.*` 标签时，也要同步更新
  [`detections/attack/artex_navigator_layer.json`](detections/attack/artex_navigator_layer.json)
  中的技术与评分。一致性测试会强制规则与覆盖层双向一致，因此覆盖层里没有的规则标签、或
  规则里没有的覆盖层技术，都会导致失败。
- **不要写入读起来像攻击指导的内容。**本仓库的检测资料只保持防御、检测的定位。不接可以
  帮助攻击的叙述，例如漏洞利用的实施方法或规避检测的技巧。

八个测试套件只要有 Docker 就能直接运行，且不把生成物提交进仓库。每个脚本只要有一个断言
失败就会以非零代码结束，因此可以直接接入 CI 或 pre-commit 钩子。

```bash
detections/tests/sigma/run.sh           # Sigma：sigma check + 后端转换 + 指标保留
detections/tests/sigma_match/run.sh     # Sigma：原子规则对恶意样本触发、对正常样本静默
detections/tests/sigma_lint/run.sh      # Sigma：SigmaHQ 惯例完整验证器 + 已记录的标准
detections/tests/sigma_backends/run.sh  # Sigma 可移植性：关联规则能否在多个后端转换
detections/tests/suricata/run.sh        # Suricata：合成 pcap → suricata -r → 断言告警数量
detections/tests/attack/run.sh          # ATT&CK：覆盖层 ↔ 规则双向一致
detections/tests/indicators/run.sh      # 指标：规则中的固定指标 ↔ 上游源码双向一致
detections/tests/misp/run.sh            # MISP：指标 CSV ↔ MISP 事件同步 + pymisp 有效性
```

要一次运行八个，请用 [`detections/tests/run-all.sh`](detections/tests/run-all.sh)。它按与
CI 相同的顺序依次执行八个，即使前面的套件失败也把其余跑完，最后输出各套件的 PASS/FAIL
摘要，只要有一个失败就以非零代码结束。把这个 runner 直接作为 pre-commit 钩子的配置示例
在仓库根目录的 [`.pre-commit-config.yaml`](.pre-commit-config.yaml) 中。用
`pip install pre-commit && pre-commit install` 安装后，只有在检测规则或这些规则所固定的
上游源码发生变化的提交上（与 CI 相同范围）才会运行 runner，从而在推送前抓住规则与测试
不一致的问题。同一个配置文件里还包含检查文档内部链接、图片与锚点的 `docs` 钩子
（上面贡献流程第 3 步的 `check-doc-links.py`）。

这八个测试由仓库 CI（[`.github/workflows/detections.yml`](.github/workflows/detections.yml)）
在 `detections/` 下有改动的每次推送、PR 时运行。指标一致性测试在指标所指向的上游源码
文件（`enrich/`、`selfupdate/`、`guard/`、`db/`、`cmd/artex/main.go`）变化时也会运行，
从而一并抓住上游重新同步改变 User-Agent、标记或默认端口、让规则悄悄过时的情况。因此，只改
规则而不更新测试与覆盖层、破坏 SigmaHQ 惯例、或与源码矛盾的规则，都会在合并前于 CI 中
暴露出来。

规则索引与各规则的依据、局限整理在 [`detections/README.md`](detections/README.md)，测试的
断言项与运行方法整理在
[`detections/tests/README.md`](detections/tests/README.md)。

---

## 许可证

本项目以 **GNU Affero General Public License v3.0（AGPL-3.0）** 发布。提交贡献即视为您同意
该贡献也**以 AGPL-3.0 公开**。特别是，如果您修改本项目并通过网络（例如作为在线服务）向
用户提供，就必须向该用户公开与之对应的完整源代码。全部条款见 [LICENSE](LICENSE) 文件。
