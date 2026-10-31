#!/usr/bin/env bash
# ARTEX 更新脚本：① Docker 更新（拉取新镜像并重建）  ② 本地编译更新（重新构建二进制）
# 与 install.sh 配对：install 负责首次安装，update 负责升级到新版本。
# 无需手动执行 DB 迁移：artex 每次启动都会幂等地重跑 schema.sql（含 ADD COLUMN/CREATE
# INDEX IF NOT EXISTS），即“重启即迁移”。数据（pgdata 卷、./data、./skills）不受影响。
set -euo pipefail
cd "$(cd "$(dirname "$0")" && pwd)"

info(){ printf '\033[36m[*]\033[0m %s\n' "$*"; }
ok(){   printf '\033[32m[+]\033[0m %s\n' "$*"; }
warn(){ printf '\033[33m[!]\033[0m %s\n' "$*"; }
die(){  printf '\033[31m[x]\033[0m %s\n' "$*" >&2; exit 1; }
ask(){  local p="$1" d="${2:-}" a; read -rp "$p${d:+ [$d]}: " a; echo "${a:-$d}"; }

# ── 可选：把仓库同步到最新代码（compose、脚本、本地编译源码都由此更新） ───────
sync_repo(){
  [ -d .git ] && command -v git >/dev/null 2>&1 || { warn "不是 git 工作副本，跳过 git pull"; return; }
  [ "$(ask '要拉取最新代码吗（git pull --ff-only）？（y/n）' y)" = y ] || return
  if ! git pull --ff-only; then
    warn "git pull 未能以快进方式完成（存在本地改动或分支分叉）。请先手动处理后再重试；本次继续使用当前代码"
  fi
}

# ── ① Docker 更新 ───────────────────────────────
update_docker(){
  command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1 \
    || die "找不到 docker / docker compose。请先用 ./install.sh 完成安装与部署"
  [ -f .env ] || die "找不到 .env。请先用 ./install.sh 完成首次部署"

  # 可选：升级到指定的版本 tag（留空则沿用 .env 的 ARTEX_TAG，默认为 latest）
  local tag; tag="$(ask '目标镜像 tag（直接回车则使用 .env / latest）' '')"
  if [ -n "$tag" ]; then
    if grep -q '^ARTEX_TAG=' .env; then
      sed -i.bak "s|^ARTEX_TAG=.*|ARTEX_TAG=${tag}|" .env && rm -f .env.bak
    else
      printf '\nARTEX_TAG=%s\n' "$tag" >> .env
    fi
    ok "已把 ARTEX_TAG 设置为 ${tag}"
  fi

  # 只动 artex。postgres 固定为 16-alpine，不必跟着升级（拉取只是浪费带宽，
  # 而且大版本变更还有兼容风险）。artex 声明了 depends_on postgres，因此带服务名
  # up 时，pg 未启动会自动拉起，已在运行则保持原样、不重新配置。
  info "拉取新镜像（仅 artex）…"
  if ! docker compose pull artex; then
    warn "拉取镜像失败（请确认 .env 的 ARTEX_IMAGE 与指定 tag 是已发布的镜像和标签、网络可用）"
    die "镜像更新失败。也可以选择“2) 本地更新（用 go 重新编译）”"
  fi
  info "重建并启动（artex 重启时会自动迁移 schema）…"
  docker compose up -d artex
  ok "更新完成 → http://localhost:8787"
  info "查看日志：docker compose logs -f artex"
  info "清理旧镜像（可选）：docker image prune -f"
}

# ── ② 本地编译更新 ──────────────────────────────
update_local(){
  command -v go >/dev/null 2>&1 || die "找不到 Go（需要 >=1.26）：https://go.dev/dl/"
  [ -f config.json ] || warn "找不到 config.json。如果是首次部署，请使用 ./install.sh"
  ok "Go: $(go version)"

  if command -v npm >/dev/null 2>&1; then
    info "重新构建前端静态产物…"
    ( cd web && npm ci && npm run build:static )
    rm -rf server/webui/dist && cp -r web/out server/webui/dist
    info "重新编译内嵌前端的单文件二进制…"
    CGO_ENABLED=0 go build -tags embedui -trimpath -o artex ./cmd/artex
  else
    warn "找不到 npm。只编译不内嵌前端的后端（前端需要用 npm run dev 单独运行）"
    CGO_ENABLED=0 go build -o artex ./cmd/artex
  fi
  ok "编译完成 → ./artex"
  warn "要使改动生效，请重启运行中的 artex 进程（重启时会自动迁移 schema）"
}

echo "=============================="
echo "  ARTEX 更新"
echo "  1) Docker 更新（拉取新镜像并重建）"
echo "  2) 本地更新（用 go 重新编译）"
echo "=============================="
case "$(ask '选择' 1)" in
  1) sync_repo; update_docker ;;
  2) sync_repo; update_local ;;
  *) die "选择无效" ;;
esac
