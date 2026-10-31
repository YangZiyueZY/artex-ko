#!/usr/bin/env bash
set -euo pipefail

# Render 会随机分配 PORT 环境变量，因此指定该端口让应用启动
# 保持默认执行方式以及技能/数据路径
: "${PORT:?必须设置 PORT 环境变量}"
echo "[*] Starting ARTEX Server on port :$PORT"
exec ./artex -addr ":$PORT"
