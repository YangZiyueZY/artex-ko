# 变更日志

中文 · [English](CHANGELOG.en.md) · [韩语原文](CHANGELOG.ko.md)

本文记录 ARTEX 全球版（本 fork）在上游仓库基础上追加的变更。格式参考
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/)。

上游 ARTEX 项目各版本的发布历史（0.3.x 及更早）与贡献者名单，已按原版中文原样保留在
[`CHANGELOG.zh.md`](CHANGELOG.zh.md) 中，便于与上游变更对照。每项变更的详细内容与依据可在
仓库提交历史中查看。

## [Unreleased]

### 本地化（i18n）

- **新增了西班牙语文档。**以英语 README 为基准，把 `README.es.md` 翻译成自然的西班牙语，并在五份 README（英语、韩语、中文、西班牙语、日语）的语言导航中加入西班牙语链接。代码、命令、payload、URL、标识符保留原文，只翻译说明文本。
- **在多语言栈全链路加入日语（ja）。**在应用的支持语言中加入日语，可在设置界面切换。UI 消息（`web/messages/ja.json`）、日期区域设置（`ja-JP`）、智能体输出语言、设置与服务器消息目录（`config/msg_i18n.go`、`server/errmsg_catalog.go`）均已翻译为日语；智能体的内部推理提示词与其他语言遵循同一原则保留原文。文档方面新增 `README.ja.md`，并更新五份 README 的语言导航与支持语言说明，使其包含日语。

### 修复

- 任务报告加载失败时与空报告区分开，并提供重试按钮。切换任务时，上一次请求的迟到响应不会覆盖新的报告。

## [0.3.15-global.1] · 2026-10-14 · ARTEX-global 转型发布

把韩语版（`0.3.15-ko.1`）转型为面向全球防御者与研究者的多语言版 ARTEX-global 的发布。以上游 ARTEX 0.3.15 为基础，未改动智能体能力。

### 身份 · 多语言（ARTEX-global 转型）

- **把 ARTEX 韩语版转型为 ARTEX 全球版。**把定位从特定地区版改为面向全球防御者、研究者的多语言版。不改智能体功能，只涉及多语言化与文档范围。
  - **默认语言从韩语改为英语。**把面向用户的输出默认值设为英语（设置层级 `config.DefaultLanguage`、Web 端 `DEFAULT_LOCALE`、`config.example.json`、`.env.example` 注释），韩语、中文、西班牙语在设置界面切换。智能体包独立测试的兜底值（`agent.defaultOutputLang`）与韩语 golden 测试原样保留以防回归。
  - **把身份表述改写为全球版。**README（韩语、英语、中文）、CONTRIBUTING、SECURITY、维护者手册、Issue/PR 模板中的「韩语版」表述改为「全球版」，并同步修改英语 README 的「为什么是韩语版」一节及其锚点（`#why-a-korean-edition` → `#why-a-global-edition`）。仓库 slug 与镜像路径在本版本下方的「仓库改名」条目中一并改为 `artex-global`。

### 首屏与截图改为英语优先

- **把 GitHub 默认 README 设为英语。**把英语文档作为 `README.md`（仓库首屏），韩语正文移到 `README.ko.md`，并重新安排根目录文档指向上级 README 的链接与锚点，使英语 = `README.md`、韩语 = `README.ko.md`。检测规则目录内的 README 链接保持原样。
- **把预览截图替换为英语版。**以默认英语重新捕获 mock UI，放入 `screenshots/en/`（dashboard、findings、chat），并移除旧的韩语截图（`screenshots/ko/`）。
- **让界面日期跟随应用语言。**把 18 处硬编码的 `ko-KR` 区域设置改为 `web/src/i18n/datetime.ts` 的 `getDateLocale()`（显示语言 → BCP47），默认英语下日期以英语显示。
- **把演示数据改为英语。**将 mock 数据（`web/src/lib/mock/{data,handler}.ts`）中约 220 处面向用户的韩语文本改为英语，命令、payload、URL、标识符保留原文。

### 仓库改名（artex-ko → artex-global）

- **把仓库名改为 `artex-global`。**GitHub slug 改为 `jiwoochris/artex-global`（旧 URL 由 GitHub 重定向保留），容器镜像路径改为 `ghcr.io/jiwoochris/artex-global`。仓库内 126 处（37 个文件）统一替换：徽章、clone URL、`docker-compose.yml`、`release.yml`、`install.sh`、文档、版本表述、检测规则 author、MISP 事件名等。上游 Go 模块（`github.com/Autumn-27/artex`）与历史版本标签（`0.3.15-ko.1`）保持原样。

### 本地化（i18n）

- **按语言显示内置智能体的名称与说明。**内置智能体（目标分解、计划制定、主智能体、执行、Auto、渗透测试、漏洞复测）由 DB 种子（`db/db.go` 的 `builtinAgents`、`server/finding_retests.go` 的复测智能体）写入，此前只显示中文。现在用智能体 `key` 到 `web/messages/` 中查找，按当前语言显示其名称、说明与提示词变量标签（找不到时回退 DB 值）。用户自建智能体直接显示 DB 值。该改动作用于智能体列表、编辑抽屉、对话界面；对话中产生的审批、拦截记录也不再显示内部 key（如 `pentest`），而是显示智能体名称。

### 上游跟进

- **单独统计模型判定（兜底审批）的 token 用量（上游 0.3.15，#191）。**这是 `0.3.15-ko.1` 遗漏的上游变更。判定调用以 `worker=judge` 聚合，并新增 `GET /api/intercept/judge/usage`。在「系统 → 命令拦截」的模型判定开关下方显示调用次数、输入、输出、缓存读写累计值与最近 30 天的每日柱状图。

### 安全

- **补齐 #22 认证加固的缺口。**读取密码哈希失败时，在 token 校验与启动首次执行处理中仍被折叠为「没有密码」，导致空哈希被缓存。现在读取失败时 token 校验直接拒绝，启动时跳过环境变量初始化与设置 token 发放。`ARTEX_ADMIN_PASSWORD` 初始化也改为用 `InsertSettingIfAbsent` 只写一次，并检查 72 字节上限。`/setup` 界面在无法确认状态时不再显示初始化表单，而是提示重试（上游 #197 的界面改动）。

## [0.3.15-ko.1] · 2026-10-12 · 韩语版首个发布

以上游 ARTEX 0.3.15 为基础的韩语版首个发布。

### 分发

- **发布韩语版 Docker 镜像。**上游镜像 `autumn27/artex` 已从 Docker Hub 下架，导致 `docker compose up -d` 快速启动失败。现在每个发布标签都会向 `ghcr.io/jiwoochris/artex-global` 推送多架构（amd64、arm64）镜像，`docker-compose.yml` 拉取该镜像。

### 安全

- **阻止首启管理员账号被远程抢占。**尚无密码的实例会免认证接受 `POST /api/auth/init`，谁先摸到端口就能设定管理员密码并取得 JWT（非公开举报：Dangel）。现在必须与启动时只在服务器控制台输出的设置 token（`setup_token`）匹配才能初始化，错误尝试会触发次数限制。为无人值守部署新增 `ARTEX_SETUP_TOKEN`、`ARTEX_ADMIN_PASSWORD` 环境变量，并在 `/setup` 界面加入 token 输入框。
- **默认绑定改为 127.0.0.1。**`artex -addr` 默认值从 `:8787` 改为 `127.0.0.1:8787`，`docker-compose.yml` 的 8787 端口映射改为 `${ARTEX_BIND:-127.0.0.1}`。需要从其他设备访问的部署必须显式指定 `-addr :8787` 或在 `.env` 中设置 `ARTEX_BIND=0.0.0.0`。无密码绑定非回环地址时，启动日志会打印告警。
- **修改密码后所有既有 token 立即失效。**JWT 签名密钥由密码哈希派生，无论通过界面修改、`reset-password.sh` 还是直接改 DB，旧会话都会断开。升级后也需要重新登录一次。修改密码的响应中会带上新的 token，因此当前浏览器会话可以继续。
- **把 CORS 全放行改为允许列表。**不再返回 `Access-Control-Allow-Origin: *`，只对 `ARTEX_CORS_ORIGINS` 中列出的来源（默认 `next dev` 的 `http://localhost:5173`）返回该头。同时新增服务器侧最小密码长度（8 位）校验与初始化、登录失败日志（含客户端地址）。
- **移植上游 ARTEX 0.3.15 的认证加固。**读取密码相关设置失败时，不再包装成 `initialized:false`，而是以 503 关闭（`authStatus`、`authInit`、`authLogin`、修改密码）。首次设置密码用 `INSERT ... ON CONFLICT DO NOTHING`（[`db.InsertSettingIfAbsent`](db/settings.go)）原子化，即使读取检查抖动，主键约束也会保护既有值。超过 72 字节的密码不再返回 500 而是 400 拒绝；并设置数据库连接池上限（32），减少 `too many clients` 错误动摇 fail-open 判定的路径。移植对齐上游提交 `a951e4a`（0.3.15），本 fork 只加入与 PR #20 不冲突的部分。

### 本地化（i18n）

- **新增英语、中文、西班牙语切换。**默认语言仍为韩语，在设置界面的「显示语言」中选择后，Web UI（`web/messages/{ko,en,zh,es}.json`）与智能体撰写的报告、摘要、对话回复，以及服务器错误消息、状态标签会一起切换。运维控制台的日志与服务器默认值由 `ARTEX_LANG`（或配置文件中的 `language`）决定。智能体的行为准则正文本轮仍保留原文。
- **强制面向用户的输出使用韩语。**为保留基准性能，被基准测试过的智能体行为准则正文（大脑）原样保留，只通过代码固定片段（`langDirective`）指示面向用户的产出（漏洞报告、事实摘要、最终摘要、聊天回复）用韩语撰写。命令、payload、代码、日志原文保留原样。
- **把 Web UI 翻译为韩语。**在 Next App Router 中引入 `next-intl`，并把字符串拆分到 `web/messages/ko.json` 与 `web/messages/zh.json`。原版中文保留在 `zh.json` 中，便于与上游更新对照。仪表盘、漏洞、对话、通知发送、拦截、LLM 设置等界面字符串均已本地化。
- **把服务器 API 面向用户的错误与响应改为韩语。**替换了返回浏览器的 HTTP 错误与响应文案。但会被回灌给智能体大脑作为输入的文案为防基准漂移保留原文，其判断依据记录在仓库工作文档中。
- **整理本地化文档。**制作了韩语 `README.md` 并配套英语 `README.en.md`，原版中文以 `README.zh.md` 保留。

### 防御与检测资料

- **新增防御、检测指南。**整理了自主 AI 攻击与既有扫描器的差异、防御者可观测的指纹（IoC）、入口点与加固、检测规则、事件响应：韩语指南（[`docs/defense-ko.md`](docs/defense-ko.md)）与同内容的英语版（[`docs/defense-en.md`](docs/defense-en.md)）。
- **提供可部署的检测规则。**把指南中的指纹转成可直接使用的规则。主机与日志层以 [Sigma](https://sigmahq.io) 原子与关联规则承载（[`detections/sigma/`](detections/sigma/)），网络层以针对 enrich prober 与 norma SDK WebFetch 的 User-Agent 的 [Suricata](https://suricata.io) 规则承载（[`detections/suricata/`](detections/suricata/)）。
- **可视化 ATT&CK 覆盖度。**把规则打标的技术整理为 MITRE ATT&CK Navigator 层（[`detections/attack/`](detections/attack/)）。
- **以标准格式提供机器可读的失陷指标（IoC）。**把 ARTEX 输出的独有指纹汇总为一个 CSV 文件（[`detections/indicators/artex_indicators.csv`](detections/indicators/artex_indicators.csv)），并以可直接导入威胁情报平台的 MISP 事件（[`detections/indicators/artex_indicators.misp.json`](detections/indicators/artex_indicators.misp.json)）提供同一批指标。规则支持的指标标注 `to_ids`，主机取证端口则标注为分类线索。
- **附带可复现的检测测试。**新增八种实际运行规则加以证明的测试（Sigma 结构与编译校验、Sigma 实时事件匹配、后端可移植性、SigmaHQ 惯例 lint、Suricata 加载与触发、ATT&CK 层一致性、指标与来源一致性、MISP 导出与 CSV 同步），以及一次跑完的批量 runner 与 pre-commit 示例，并接入 CI 合并门禁。Sigma 实时事件匹配不仅确认规则可编译，还确认原子与关联规则都对恶意样本实际触发、对正常样本保持静默。

### 仓库治理

- **加入安全与滥用警示及国内法提示。**在 README 最上方加入使用范围、《信息通信网法》《个人信息保护法》提示与禁止滥用的警告。
- **用韩语 UI 截图替换界面预览。**
- **整理维护者手册与贡献指南。**制作了防止上游同步与翻译漂移的手册（[`MAINTAINING.md`](MAINTAINING.md)）与检测规则贡献约定（[`CONTRIBUTING.md`](CONTRIBUTING.md)）。手册中还整理了发布流水线的构建前提，以及不打标签、在本地验证这些前提的流程。
- **新增推送与 PR 合并门禁 CI。**上游仓库只在标签发布时运行 CI，本 fork 在每次推送与 PR 上运行 Go 构建、静态分析（`go vet`）、单元测试（[`ci.yml`](.github/workflows/ci.yml)）、Web UI 静态构建（[`web.yml`](.github/workflows/web.yml)）、文档站内链接与图片引用完整性检查（[`docs.yml`](.github/workflows/docs.yml)），以便在合并前抓住本地化过程引入的回归。需要数据库的集成测试也按包隔离 PostgreSQL 服务一并验证。文档链接检查由不依赖外部网络的确定性脚本（[`scripts/check-doc-links.py`](scripts/check-doc-links.py)）执行，避免多语言文档之间的大量相对链接与界面预览图片在损坏状态下被合并。文档锚点（`#标题`）链接也按与 GitHub 相同的 slug 规则与标题比对，从而抓住因标题文字变化而悄然断掉的目录与交叉引用链接。检测规则套件由上述「防御与检测资料」一节说明的合并门禁负责。
- **定期巡检外部链接存活情况。**防御指南指向的事件举报窗口、标准参考等外部链接依赖远端服务器状态、具有 flaky 特性，因此不放进合并门禁，而由非阻塞工作流（[`external-links`](.github/workflows/external-links.yml)）在每周一与手动执行时，用浏览器 User-Agent、GET、跟随重定向的方式运行检查（[`scripts/check-external-links.py`](scripts/check-external-links.py)）。主机存活但只是确认方式被阻挡（屏蔽爬虫、限速）的情况，以及我们无法修复的上游继承死链（allowlist）都不计为失败，因此只有我们文档中精选的外部链接新出现损坏时才会标红。
- **补齐贡献与治理基础设施。**新增 Bug、功能、翻译 Issue 模板（[`.github/ISSUE_TEMPLATE/`](.github/ISSUE_TEMPLATE/)）与 Pull Request 模板（[`PULL_REQUEST_TEMPLATE.md`](.github/PULL_REQUEST_TEMPLATE.md)）、安全漏洞举报政策（[`SECURITY.md`](SECURITY.md)）、行为准则（[`CODE_OF_CONDUCT.md`](CODE_OF_CONDUCT.md)），让外部贡献者以一致的格式提交 Issue、PR 与安全举报。
- **为海外贡献者补全英语文档层。**本仓库曾以韩语为主要语言，为了让不读韩语的贡献者、安全研究者与防御者获得相同信息，核心文档同时提供英语版。除英语 `README.en.md` 与防御指南（[`docs/defense-en.md`](docs/defense-en.md)）之外，还补齐了变更日志（[`CHANGELOG.en.md`](CHANGELOG.en.md)）、安全举报政策（[`SECURITY.en.md`](SECURITY.en.md)）、行为准则（[`CODE_OF_CONDUCT.en.md`](CODE_OF_CONDUCT.en.md)）、贡献指南（[`CONTRIBUTING.en.md`](CONTRIBUTING.en.md)）、维护者手册（[`MAINTAINING.en.md`](MAINTAINING.en.md)）、流量证据设计文档（[`docs/finding-traffic-evidence-en.md`](docs/finding-traffic-evidence-en.md)）以及 Bug、功能、翻译 Issue 模板的英语版。各语言版本在文件头部互相链接，从任一语言都能跳到另一语言。（Pull Request 模板目前只提供韩语版。）

---

上游 ARTEX 项目各版本的发布历史与贡献者名单，可在 [`CHANGELOG.zh.md`](CHANGELOG.zh.md) 中按原文查看。
