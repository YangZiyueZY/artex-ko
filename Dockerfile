# syntax=docker/dockerfile:1
#
# 仅运行镜像（不在镜像内编译）：只安装常用工具，放入**预先编译好的 Linux 单文件二进制**。
# 二进制由 CI 的 binaries 作业交叉编译（纯 Go，无需 QEMU），按目标架构放在
# 构建上下文的 dist/<TARGETARCH>/artex。这样多架构构建时 arm64 只需模拟
# apt 层，不再模拟 Next/Go 编译，因此快得多。
#
# 在本地手动构建镜像时，请先自行准备好二进制：
#   cd web && npm run build:static && cd ..
#   mkdir -p server/webui && cp -r web/out server/webui/dist
#   CGO_ENABLED=0 GOARCH=amd64 go build -tags embedui -o dist/amd64/artex ./cmd/artex
#   docker build -t artex:local .
FROM python:3.12-slim-bookworm
ARG TARGETARCH
# 常用工具：在 ripgrep / curl / vim 之外，再加上侦察中常用的工具集（可按需增删）。
# Node 从 NodeSource 安装 20.x：bookworm 自带的 apt nodejs 是 18，而 Playwright 要求 >=20。
RUN apt-get update && apt-get install -y --no-install-recommends \
      ca-certificates ripgrep curl wget vim git jq unzip \
      dnsutils iputils-ping netcat-openbsd inetutils-telnet whois nmap \
    && curl -fsSL https://deb.nodesource.com/setup_20.x | bash - \
    && apt-get install -y --no-install-recommends nodejs \
    && rm -rf /var/lib/apt/lists/*
# 预先全局安装 Playwright MCP 和 CLI（避免运行时用 npx 从网络下载）。
# @playwright/mcp：browser MCP 可直接用 `npx @playwright/mcp` 运行（已全局安装，无需 -y/@latest）。
# @playwright/cli：提供 playwright-cli，安装后用 --help 确认能否执行。
# 随后安装 playwright（提供浏览器管理），并用 --with-deps 预先装好 chromium 及其系统依赖。
# 这样容器内的 MCP/CLI 首次启动即可使用，也不会从网络下载浏览器。
RUN npm install -g @playwright/mcp@latest @playwright/cli@latest playwright@latest \
    && playwright-cli --help \
    && playwright install --with-deps chromium \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /app
# 预先编译好的对应架构二进制（dist/amd64/artex 或 dist/arm64/artex）
COPY dist/${TARGETARCH}/artex /app/artex
# 守护启动脚本：进程结束后根据退出码决定是否重新拉起，界面中的一键更新通过该脚本完成替换。
# 该脚本同时负责把 SIGTERM 转发给 artex：docker stop 只把信号发给 PID 1，
# 不转发的话 artex 收不到信号，无法优雅退出，10 秒后会被 SIGKILL 强制结束。
COPY start.sh /app/start.sh
RUN chmod +x /app/artex /app/start.sh
COPY skills/ /app/skills/
# data/（SQLite + jwt.key）持久化位置
VOLUME ["/app/data"]
EXPOSE 8787 8788
ENTRYPOINT ["/app/start.sh"]
CMD ["-addr", ":8787", "-proxy", ":8788"]
