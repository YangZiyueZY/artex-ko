#!/usr/bin/env bash
# 开发模式：同时运行后端（127.0.0.1:8787）、流量代理（:8788）与前端 next dev（:5173）。
# 前端的 /api 会反向代理到后端。按 Ctrl-C 一起结束。
#
# 单文件二进制（内嵌前端）方式请参阅 README 的“单文件二进制”一节，不使用本脚本。
set -euo pipefail
cd "$(dirname "$0")"

# 退出时结束本进程组的所有子进程（后端 + 前端）。
cleanup() { kill 0 2>/dev/null || true; }
trap cleanup EXIT INT TERM

# 后端（普通 go run，不内嵌前端）。并发 work agent 数量在“系统设置”中配置。
go run ./cmd/artex -addr 127.0.0.1:8787 -proxy 127.0.0.1:8788 &

# 前端热重载（Vite/Next dev server，/api 反向代理到 :8787）。
( cd web && npm run dev ) &

echo "[dev] 后端 :8787 / 代理 :8788 / 前端 http://localhost:5173  （按 Ctrl-C 退出）"
wait
