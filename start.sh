#!/bin/sh
# ARTEX 守护进程启动脚本（Linux / macOS / Docker ENTRYPOINT）
#
# 用法：
#   ./start.sh                       前台运行（按 Ctrl-C 停止）
#   nohup ./start.sh >artex.log 2>&1 &   后台常驻
#   ./start.sh -addr :9000           其余参数原样传给 artex
#
# 本脚本只做一件事：运行 artex，进程结束后根据退出码决定是否重新拉起。
#
#   0      用户正常停止      → 结束循环
#   75     程序请求重启      → 立即重新运行（在界面点击“一键更新”或“回滚”）
#   其他   异常退出          → 退避后重新运行（1→2→4…最多 60 秒）
#
# 下载、SHA256 校验和验证、版本替换刻意不放在这里。那段逻辑要在 sh 和 bat 里各写一遍，
# 偏偏又是最不能出错的部分。一旦替换成无法执行的二进制，本脚本会忠实地
# 反复拉起它，用户只能直接登入机器手动恢复。所以验证与替换全部放在 Go（selfupdate 包）中，
# 由 artex 在启动时自行处理，脚本保持简单。
set -u

cd "$(dirname "$0")" || exit 1

BIN=./artex
[ -x "$BIN" ] || { echo "[artex] 找不到可执行文件：$BIN" >&2; exit 1; }

RESTART_CODE=75
MAX_DELAY=60

child=0
stopping=0

# 把停止信号转发给 artex 本体。
#
# 在 Docker 中这是必需的。docker stop 只把 SIGTERM 发给 PID 1（即本脚本），
# 不会发给子进程。不转发的话 artex 收不到信号，无法优雅退出，
# 10 秒后会被 SIGKILL 强制结束，正在运行的任务会被中途打断。
forward() {
	stopping=1
	if [ "$child" -ne 0 ]; then
		kill -TERM "$child" 2>/dev/null || true
	fi
}
trap forward INT TERM

delay=1
while :; do
	"$BIN" "$@" &
	child=$!

	# 信号会中断 wait，使其返回大于 128 的值。此时子进程仍在进行优雅退出，
	# 需要再 wait 一次才能拿到真正的退出码。
	wait "$child"
	code=$?
	if [ "$code" -gt 128 ]; then
		wait "$child"
		code=$?
	fi
	child=0

	if [ "$stopping" -eq 1 ]; then
		echo "[artex] 已停止"
		exit 0
	fi

	case "$code" in
		0)
			echo "[artex] 正常退出"
			exit 0
			;;
		"$RESTART_CODE")
			# 更新/回滚已就绪。再次运行时 artex 会在启动时完成版本替换（参见 selfupdate.Bootstrap）。
			echo "[artex] 请求重启（应用新版本）…"
			delay=1
			;;
		*)
			echo "[artex] 异常退出（code=$code），${delay}s 后重启" >&2
			sleep "$delay"
			delay=$((delay * 2))
			[ "$delay" -gt "$MAX_DELAY" ] && delay=$MAX_DELAY
			;;
	esac
done
