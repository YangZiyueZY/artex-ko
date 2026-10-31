# 上游同步与防止翻译漂移（维护者指南）

中文 · [English](MAINTAINING.en.md) · [韩语原文](MAINTAINING.ko.md)

本文整理了**维护者**在跟进原版仓库 [Autumn-27/ARTEX](https://github.com/Autumn-27/ARTEX)
（目前无法访问，2026-10 确认）变更的同时，维持多语言本地化的流程。贡献范围、法律责任与
本地化方针在 [CONTRIBUTING.md](CONTRIBUTING.md) 中，面向用户的说明在 [README](README.md)
中，因此本文只聚焦于**如何实际执行这些方针**。

本地化的核心目标可以用一句话概括：完整保留原版的**判断性能**，只把用户可见的产出换成
所设置的语言。每次上游更新时这条边界都容易被冲淡，因此用下面的流程与检查防止翻译漂移。

---

## 1. 本地化结构一览

本仓库 **fork** 了上游 ARTEX，并在其历史之上叠加了本地化提交。上游 `main` 的所有提交都
包含在本仓库的历史中，本地化提交加在其上。因此跟进上游变更，就变成「查看与上游 `main`
的差异，区分要保留与要翻译的部分再行跟进」这项工作。

产出分为三类。

- **保留原文的资产**（见第 2 节）。翻译它们会破坏性能或与上游的对照。
- **在代码中固定的输出语言强制**。`agent/prompt.go` 的 `langDirective()` 会在各角色的
  system 提示词末尾追加「用户可见输出要使用设置的语言」的指示。
- **本地化的用户可见字符串**。UI 字符串放在 `web/messages/*.json` 中，服务器的用户响应
  文案放在各 Go 文件的具名常量中。

---

## 2. 保留原文的资产（禁止翻译）

下列资产不翻译，保持原文（中文或英语）。当上游变更触及这些资产时，**不做翻译、原样跟进**。

- **智能体内部推理提示词（大脑正文）。**位于 `agent/promptcatalog.go` 与 DB 种子
  `agent_prompts` 中的行为准则正文。必须以原文（中文）保持经过基准测试的行为，否则翻译会
  让判断产生漂移。
- **兼作展示与智能体输入的字符串。**部分文案既显示在活动时间线上，又作为规划器、报告器
  的输入上下文被回灌（任务中断原因、拦截阻断消息、流量证据 helper 等）。同一条记录兼具两种
  用途，因此保留原文。判断依据按事项记录在 `work/DECISIONS-FOR-JIWOO.md` 的「大脑边界
  记录」中。
- **上游原版中文文档与字符串。**中文主文档 [`README.md`](README.md) 已并入上游的中文 README
  原文（原 `README.zh.md` 仅作为指向 `README.md` 的旧链接占位），UI 字符串原文保留在
  `web/messages/zh.json` 中，便于与上游变更对照；英语、韩语、西班牙语、日语的译文分别填入
  `web/messages/{en,ko,es,ja}.json`。
- **命令、payload、代码、URL、标识符、日志原文。**它们是分析所需的原始内容，不做翻译。
  Go 代码注释优先级最低，可在上游对照结束之后再处理。

---

## 3. 上游跟踪基准

上游 remote 需要按下述方式配置。不存在时请添加。

```bash
git remote add upstream https://github.com/Autumn-27/ARTEX
git remote -v   # 确认能看到 upstream
```

当前已跟进完成的上游基准提交如下。

- **基准 = `d003372`**（上游 `main`，2026-10-03，合并 PR #189 `fix/sse-same-origin`）。

这个值表示「截至该提交的上游变更已全部并入本仓库」。每次跟进新的上游变更时，按第 7 节的
方法更新该基准。

---

## 4. 跟进上游变更的流程

### 4.1 拉取上游并查看差异

```bash
git fetch upstream
git rev-list --count d003372..upstream/main        # 未跟进的上游提交数
git log --oneline d003372..upstream/main           # 未跟进的提交列表
```

`git fetch` 只更新上游的远程跟踪分支，不会影响工作树与 `HEAD`。未跟进的提交为 0，说明
与上游已同步，无需继续处理。

### 4.2 对变更文件分类

查看未跟进的提交改动了哪些文件，按第 2 节的保留资产与翻译对象分类。

```bash
git log --name-status --oneline d003372..upstream/main
```

分类标准如下。

- 如果改动了 `agent/promptcatalog.go`、`agent_prompts` 种子，以及第 2 节中兼作展示与
  输入的字符串 → **不做翻译、原样跟进**。
- 如果改动了 Go 后端逻辑（`db/`、`llmrec/`、`server/` 等） → 逻辑原样跟进，同时检查
  是否**新增了用户响应文案**（`writeErr` 等）并翻译为本地化具名常量。
- 如果 UI（`web/src/**`）有改动、**新增了界面字符串** → 不要硬编码，而是在
  `web/messages/zh.json`（原文）与其他语言文件（译文）中用相同的键添加。
- 如果**检测规则所固定的上游指标**（`enrich/enrich.go` 的 prober User-Agent、
  `selfupdate/` 的自更新 User-Agent、`guard/guard.go` 的审计标记、`db/db.go` 的破坏性
  命令 deny 清单、`cmd/artex/main.go` 的默认监听与记录代理端口）发生变化 → 把
  `detections/` 的 Sigma、Suricata 规则与 ATT&CK 覆盖层，以及
  `detections/indicators/artex_indicators.csv` 的值也改成新值。这些指标不是翻译对象，而是
  **检测的依据**，上游一旦改值，规则就会悄悄过时。5.4 的指标一致性测试会自动抓住这种错位。

### 4.3 跟进

按功能单位合并或选择性跟进后，把 4.2 中筛出的新字符串翻译为目标语言。合并过程中很容易
出现 `ko.json`、`zh.json` 键错位或原文混进用户可见位置的情况，因此跟进后必须运行第 5 节
的检查。

> **示例（2026-10-05 时点的未跟进提交）。** `git fetch upstream` 后，上游 `main` 已前进到
> `b55ceb1`，相对基准 `d003372` 有 2 个提交（`86729b6` 模型兜底审批 token 计量功能 +
> 合并提交 `b55ceb1`）尚未跟进。这些提交改动了 `db/llm_usage.go`、`llmrec/llmrec.go`、
> `server/intercept.go`、`server/server.go` 等 Go 逻辑，以及
> `web/src/app/(main)/system/intercept/page.tsx`、`web/src/lib/api.ts`、
> `web/src/lib/mock/handler.ts`、`web/src/lib/types.ts`。因此维护者应原样跟进 Go 逻辑，
> 只把 intercept 设置页面新增的界面字符串抽成 `ko.json`、`zh.json` 的键并翻译。（写本文时
> 这两个提交尚未跟进，因此基准仍记为 `d003372`。）

---

## 5. 翻译对称与漂移检查

跟进上游或做翻译之后，确认下面三项。

### 5.1 ko ↔ zh 消息对称与用户可见 CJK

`ko.json` 与 `zh.json` 的键必须完全一致，且 `ko.json` 的值中不得残留中文汉字。下面的
脚本会输出三个数值。

```bash
python3 - <<'PY'
import json, re
ko = json.load(open('web/messages/ko.json'))
zh = json.load(open('web/messages/zh.json'))
def flatten(d, p=''):
    out = {}
    if isinstance(d, dict):
        for k, v in d.items(): out.update(flatten(v, p + '/' + k))
    elif isinstance(d, list):
        for i, v in enumerate(d): out.update(flatten(v, p + '/' + str(i)))
    else: out[p] = d
    return out
fk, fz = flatten(ko), flatten(zh)
han = re.compile(r'[㐀-鿿]')
print('ko leaf keys :', len(fk))
print('zh leaf keys :', len(fz))
print('key symdiff  :', len(set(fk) ^ set(fz)))        # 必须为 0
print('ko vals w/CJK:', sum(1 for v in fk.values() if isinstance(v, str) and han.search(v)))  # 必须为 0
PY
```

基准值（2026-10-05）：`ko leaf keys = 2950`、`zh leaf keys = 2950`、`key symdiff = 0`、
`ko vals w/CJK = 0`。键的数量可能随上游跟进而增加，但 ko 与 zh 必须始终相同，
`key symdiff` 与 `ko vals w/CJK` 必须始终为 0。

### 5.2 确认大脑资产保留原文

大脑正文保持中文原文，因此下面的检查中**汉字行数跌到 0** 反而是大脑被误翻译而污染的信号。

```bash
python3 -c "import re; han=re.compile(r'[㐀-鿿]'); t=open('agent/promptcatalog.go').read(); print('promptcatalog.go CJK lines =', sum(1 for l in t.splitlines() if han.search(l)))"
```

基准值（2026-10-05）：`promptcatalog.go CJK lines = 70`。该数值大幅减少时，请确认大脑正文
是否被翻译了。

### 5.3 确认原文没有漏进构建产物

把 UI 静态导出后，如果某个语言的预渲染 HTML 中出现了未翻译的中文，就说明翻译遗漏。

```bash
cd web && npm ci && NEXT_EXPORT=1 npm run build   # 生成 out/
# 确认 out/**/*.html 中可见文本里没有残留的中文汉字（中文为主语言，其他语言界面不应出现中文）
```

### 5.4 确认检测指标仍与上游源码一致

`detections/` 的规则以���游实际输出的字符串（prober User-Agent、自更新 User-Agent、审计
标记、破坏性命令 deny 清单）为依据。上游重新同步改变这些值时，翻译检查会全部通过，但已
发布的规则会悄悄停止匹配。下面的测试会双向确认每个指标是否仍同时存在于上游源码与规则
两侧，因此请在重新同步后一并运行。

```bash
detections/tests/indicators/run.sh   # 用 Docker 隔离运行，RESULT: PASS 即一致
```

失败时会输出哪个指标错位、以及方向（是上游源码变了，还是规则变了），请按 4.2 最后一条分类
标准把规则、覆盖层改成新值。该测试在仓库 CI
（[`.github/workflows/detections.yml`](.github/workflows/detections.yml)）中，也会在规则树或
上述上游源码文件发生变化的每次推送、PR 时自动运行，把重新同步的漂移拦在合并门禁上。

新增指标时如果**固定了新的上游源码文件**（例如加入 `cmd/artex/main.go` 的端口指标时），
必须把该文件也加入上述工作流的 `push`、`pull_request` `paths` 过滤器。漏掉的话，只改该
源码的 PR 就无法触发指标测试，漂移会悄悄通过合并门禁。这种同步本身也由指标测试自动检查
（第五项检查 `CI triggers this test when any pinned source changes`）：测试读取的所有非
`detections/` 源码如果没有列举在两侧 `paths` 块中，测试就会失败，因此源码固定与 CI 触发
条件不会在错位状态下被合并。

### 5.5 升级检测测试工具版本固定时

检测测试以固定版本运行 `sigma-cli`、SigmaHQ 验证器插件（`pySigma-validators-sigmahq`）与
Suricata 镜像（默认值在各 `run.sh` 中，可用环境变量覆盖）。升级这些固定版本时，可能出现
并非上游源码、而是**工具侧的漂移**。尤其是 SigmaHQ 验证器每次升级都会新增惯例检查，因此
`detections/tests/sigma_lint/run.sh` 可能把新问题标红。此时要么让规则符合新惯例，要么在
[`detections/tests/sigma_lint/validators.yml`](detections/tests/sigma_lint/validators.yml)
的排除列表中加入理由（当惯例不适用于单独规则集时）。后端插件改变支持范围时，
`sigma_backends` 测试会给出同样的信号。

---

## 6. 用构建与测试做最后验证

跟进、翻译之后，按 [CONTRIBUTING.md 的开发环境](CONTRIBUTING.md#开发环境)中的步骤验证后端
与前端。本地没有 Go 时，可以用 Docker 做同样的运行。

```bash
docker run --rm -v "$PWD":/src -w /src \
  -v artexko-gomod:/go/pkg/mod -v artexko-gocache:/root/.cache/go-build \
  golang:1.26 sh -c 'go build ./... && go vet ./... && go test ./... -count=1'
```

翻译用户可见文案时，请一并编写断言该文案的回归测试（`*_localized_test.go`），以便日后
上游变更再次带回原文时由测试抓住。翻译验证必须使用有能力的（前沿级）模型。低端、小型
模型可能让输出退回原文，因此不能只凭其输出判断翻译是否生效。

---

## 7. 更新基准记录

跟进上游变更并完成验证后，请把本文第 3 节的**基准提交值更新为新的上游提交**，并把该变更
包含在同一提交或后续提交中。这样下一位维护者就能在本文一处确认「跟进到了哪里」。

提交信息遵循 [CONTRIBUTING.md 的提交信息规则](CONTRIBUTING.md#提交信息)。例如上游同步
提交可以这样写。

```
chore(upstream): 跟进上游 d003372..b55ceb1（intercept token 计量） + 新增 UI 字符串翻译
```

---

## 8. 审查与验证守则（常见陷阱）

这里记录维护者在检查上游跟进、翻译、文档补充时反复踩到的两个陷阱。两者都属于「检查方法
本身有误，把正常的东西误判为损坏」的情况，因此固化为守则以避免不必要的回滚。

### 8.1 确认仓库 CI 状态时要指定仓库

本仓库是上游 ARTEX 的 fork，本地 `git remote` 中同时登记了 `origin`
（jiwoochris/artex-global）与 `upstream`（Autumn-27/ARTEX）（见第 3 节）。在这种状态下，
给 `gh` 命令不指定仓库，`gh` 会**默认选择上游仓库**，显示没有我们工作流的上游运行结果。
于是可能看到上游 CI 为绿就**误以为我们的 CI 通过了**，或把我们自己的工作流
（`ci.yml`、`detections.yml`）误判为「HTTP 404 … not found」。

所以确认 CI 时始终指定仓库。

```bash
gh run list -R jiwoochris/artex-global --workflow ci.yml --limit 5
gh run list -R jiwoochris/artex-global --workflow detections.yml --limit 5
```

设置一次之后，可以省略 `-R` 而默认查看我们的仓库。不过该设置属于**本地 gh 配置**，不会
提交到仓库，因此在新机器或新的 checkout 中需要重新指定。

```bash
gh repo set-default jiwoochris/artex-global
gh repo set-default --view   # 确认能看到 jiwoochris/artex-global
```

### 8.2 文档中的外部链接要像浏览器一样用 GET 确认

防御指南（[`docs/defense-ko.md`](docs/defense-ko.md)、[`defense-en.md`](docs/defense-en.md)）
第 7 节收录了国内官方渠道（boho.or.kr、fsec.or.kr、pipc.go.kr）的链接。确认这些链接是否
有效时，只用 `curl -I`（HEAD 请求）或默认 User-Agent 会**把正常的链接误判为损坏**。国内
公共、安全机构站点会因下列三种原因拒绝简单确认。

- **拒绝 HEAD 请求。**例如 fsec.or.kr 对 `curl -I`（HEAD）返回 400。
- **屏蔽默认 `curl` User-Agent。**fsec.or.kr 与 pipc.go.kr 对以默认 UA 发送的 GET 请求也
  返回 400（用浏览器 UA 发送则为 200）。
- **重定向到其他地址。**pipc.go.kr 会从 `www.pipc.go.kr` 两次重定向到 `pipc.go.kr/np/`，
  不跟随重定向就会错过最终状态。

因此确认链接时要**使用浏览器 User-Agent、用 GET、跟随重定向**。

```bash
UA='Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/140.0 Safari/537.36'
for u in https://www.boho.or.kr https://www.fsec.or.kr https://www.pipc.go.kr; do
  curl -sS -L -A "$UA" -o /dev/null -w "$u -> %{http_code} %{url_effective}\n" "$u"
done
```

最终状态码为 200 时链接有效。如果状态码是 400、403，请先怀疑并非链接损坏，而是**确认方式
被服务器的访问策略阻挡**，逐个排除 HEAD、默认 UA、未跟随重定向等因素后重新确认。
（以 2026-10-06 的确认结果为准，三个链接在上述方法下均为 200，pipc.go.kr 是在两次重定向
之后返回 200。）

这一手工流程已被 `scripts/check-external-links.py` 自动化。它从所有受跟踪的 `.md` 中收集
代码围栏、行内代码之外的外部链接（排除保留、占位主机），按上述方式用浏览器 UA、GET、跟随
重定向确认状态，并对网络错误、5xx、429 重试，以区分瞬时报错与真实故障。结果分为四类：
OK（2xx、3xx）、RESTRICTED（401、403、405、429，主机存活但确认方式被阻挡）、ALLOWED
（`scripts/external-links-allowlist.txt` 中记录、我们无法修复的上游继承死链）、DOWN
（404、410、5xx、连接错误，损坏可能性高）。

- 不联网、只预览检查对象：`python3 -I scripts/check-external-links.py --list`
- 发布、周期检查（出现新损坏链接时以非零退出）：`python3 -I scripts/check-external-links.py --strict`

外部链接存活具有 flaky 特性，因此**不纳入合并门禁**。取而代之，非阻塞工作流
[`external-links`](.github/workflows/external-links.yml) 在每周一与手动执行时运行
`--strict`，把 allowlist 之外新出现的 DOWN 标红。上游原文存档文件继承的死链（例如
`CHANGELOG.zh.md` 中署名的、已消失的贡献者账号）我们无法修复，因此在 allowlist 中连同
理由一起记录，并从 strict 检查中排除。

---

## 9. 发布流水线

推送版本标签（`v*`）会触发 [`.github/workflows/release.yml`](.github/workflows/release.yml)，
为 10 个平台构建运行版本，并在满足条件时构建多架构 Docker 镜像。本节整理流水线依赖哪些
前提、产出什么，以及如何在不发布的情况下试跑。推送标签会在公开仓库生成 GitHub Release，
所以打发布标签要在发布决定之后进行。

### 9.1 如何打发布标签

推送以 `v` 开头的标签即会触发工作流。

```bash
git tag v0.3.15-zh.1
git push origin v0.3.15-zh.1
```

只想验证流水线、暂不发布时，在 Actions 页面手动运行 `release` 工作流
（`workflow_dispatch`）：可填版本号（留空则用 `dev.<运行号>`）、可勾选是否顺带推送镜像。
手动运行不创建 GitHub Release，产物只保留在 workflow artifact 里。

### 9.2 流水线做什么

工作流分为五个作业。

- **frontend。**把前端静态导出一次（`web/out`），并把该产物作为 `web-dist` 制品上传。
  下面的 binaries 作业会为每个目标重新下载并复用该产物。
- **binaries。**用矩阵交叉编译 10 个目标，并为每个目标打包成 zip：
  - linux/amd64、linux/arm64、**linux/386（32 位 x86）**、**linux/arm/v7、linux/arm/v6（32 位 ARM）**
  - windows/amd64、windows/arm64、**windows/386**
  - darwin/amd64、darwin/arm64（Go 不支持 32 位 macOS）

  每个 zip 都会校验内容（二进制、启动脚本、`config.example.json`、`skills/`）；linux/amd64
  直接跑 `artex -h` 冒烟测试，linux/386 在 runner 支持 IA32 模拟时也跑一次（否则只告警），
  其余目标用 `file` 核对 ELF/PE/Mach-O 架构。
- **checksums。**汇总所有 zip 生成 `SHA256SUMS`，作为 `release-assets` 制品上传（手动运行也能下载）。
- **release。**仅在标签推送时运行：创建 GitHub Release 并附上 zip 与 `SHA256SUMS`。
- **docker。**接收 binaries 交叉编译出的 linux 二进制，构建多架构（amd64、arm64）镜像，并以
  发布标签与 `latest` 推送到 GitHub Container Registry 的 `ghcr.io/<owner>/<repo>`（镜像名取
  当前仓库并转小写，可用仓库变量 `ARTEX_IMAGE` 覆盖）。登录使用工作流默认的 `GITHUB_TOKEN`
  （作业权限 `packages: write`），因此没有额外的 secret。

### 9.3 构建前提是否与仓库结构相符

流水线在下列三个前提之上运行，本文在本地直接复现 binaries 作业，确认这些前提目前是否都与
仓库结构相符。

- **前端内嵌。**binaries 作业把 frontend 作业上传的 `web-dist`（即 `web/out` 的内容）下载
  到 `server/webui/dist`，`server/webui_embed.go` 的 `//go:embed all:webui/dist` 会把该位置
  内嵌进二进制。因此 binaries 作业不重新构建前端，而是以 `ARTEX_SKIP_FRONTEND=1` 调用
  [`build.sh`](build.sh)。
- **二进制与打包路径。**`build.sh --target <os>/<arch>[/变体]` 生成
  `dist/artex-<os>-<arch>/artex` 二进制与 `dist/` 下的 zip 包；32 位 ARM 把变体写进架构名
  （`--target linux/arm/v7` → `dist/artex-linux-armv7/artex`、`artex-<版本>-linux-armv7.zip`），
  并同时通过 `-X` 把变体写进二进制，供页面一键更新拼出正确的包名。zip 中除二进制外还包含
  启动脚本（Linux、macOS 为 `start.sh`，Windows 为 `start.bat`）、`skills/`、
  `config.example.json`、`README.md`。
- **Docker 镜像的二进制复制。**binaries 作业把 linux 二进制单独作为 `bin-linux-<arch>`
  制品上传，docker 作业把它下载到 `dist/<arch>/artex`。
  [`Dockerfile`](Dockerfile) 的 `COPY dist/${TARGETARCH}/artex` 会以多架构构建时 buildx 按
  平台填好的 `TARGETARCH` 取该路径。[`.dockerignore`](.dockerignore) 没有排除 `dist/`，
  因此二进制会包含在构建上下文中。

### 9.4 Docker 镜像命名空间

工作流把镜像发布到当前仓库对应的 GHCR 命名空间（`ghcr.io/<owner>/<repo>`，全小写），
也就是本仓库的 `ghcr.io/yangziyuez/artex-ko`；`docker-compose.yml` 通过
`${ARTEX_IMAGE:-ghcr.io/yangziyuez/artex-ko}` 拉取同一个镜像，可在 `.env` 里改 `ARTEX_IMAGE`
指向别的命名空间。上游 `autumn27/artex` 已从 Docker Hub 下架，不使用。GHCR 包首次发布时
通常被创建为私有，所以首次发布后请在包设置中确认是否为公开（public）。若为私有，
`docker compose pull` 会因认证错误失败。

### 9.5 不打标签、在本地预先验证

想在不打公开发布标签的情况下只确认流水线前提，可以在本地复现 binaries 作业。本地没有 Go
时，可以用 Docker 做同样的运行。

```bash
# 1) 前端静态导出（对应 release.yml 的 frontend 作业）
cd web && npm ci && npm run build:static && cd ..
# 2) 放到 binaries 作业接收制品的位置
rm -rf server/webui/dist && mkdir -p server/webui/dist && cp -a web/out/. server/webui/dist/
# 3) 只构建一个目标，环境与 binaries 作业相同
docker run --rm -v "$PWD":/app -w /app \
  -e ARTEX_SKIP_FRONTEND=1 -e ARTEX_SKIP_NPM_CI=1 \
  -e ARTEX_COMPRESS=0 -e ARTEX_PACKAGE=1 -e ARTEX_PACKAGE_DIR=dist \
  -e ARTEX_BUILD_VERSION=v0.0.0-local \
  golang:1.26 bash -c 'apt-get update && apt-get install -y zip && ./build.sh --target linux/amd64'
# 3b) 32 位 / ARM 目标同理（Go 交叉编译，不需要额外工具链）
#     ./build.sh --target linux/386 ; ./build.sh --target linux/arm/v7 ; ./build.sh --target windows/386
# 4) 确认产物：dist/artex-linux-amd64/artex · dist/artex-linux-armv7/artex · dist/*.zip · dist/SHA256SUMS
```

`dist/artex-linux-amd64/artex` 是静态链接的 ELF，加上 `-h` 会输出用法后以退出码 0 结束。
这正是 binaries 作业快速验证所确认的行为。构建产物（`dist/`、`server/webui/dist/`）不提交
进仓库（由 `.gitignore` 排除）。
