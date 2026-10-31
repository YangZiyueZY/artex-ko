@echo off
rem 将控制台切换为 UTF-8；否则本文件中的中文在 GBK 终端会乱码。
chcp 65001 >nul 2>&1
rem ARTEX 守护进程启动脚本（Windows）
rem
rem 用法：
rem   start.bat                  前台运行（按 Ctrl-C 停止）
rem   start.bat -addr :9000      其余参数原样传给 artex
rem
rem 本脚本只做一件事：运行 artex.exe，进程结束后根据退出码决定是否重新拉起。
rem
rem   0      用户正常停止   -> 结束循环
rem   75     程序请求重启   -> 立即重新运行（在界面点击“一键更新”或“回滚”）
rem   其他   异常退出       -> 退避后重新运行（1->2->4…最多 60 秒）
rem
rem 下载、SHA256 校验和验证、版本替换不在这里做，全部由 artex 在启动时自行处理
rem （selfupdate 包）。脚本保持简单。详细说明请参考 start.sh 顶部的描述。

setlocal enabledelayedexpansion
cd /d "%~dp0"

set "BIN=artex.exe"
if not exist "%BIN%" (
	echo [artex] 找不到可执行文件：%BIN% 1>&2
	exit /b 1
)

set "RESTART_CODE=75"
set "MAX_DELAY=60"
set /a delay=1

:loop
"%BIN%" %*
set "code=!ERRORLEVEL!"

if "!code!"=="0" (
	echo [artex] 正常退出
	exit /b 0
)

if "!code!"=="%RESTART_CODE%" (
	rem 更新/回滚已就绪。再次运行时 artex 会在启动时完成版本替换。
	echo [artex] 请求重启：应用新版本…
	set /a delay=1
	goto loop
)

echo [artex] 异常退出 ^(code=!code!^)，!delay!s 后重启 1>&2
rem timeout 在重定向的控制台中会失败，因此用 ping 代替（延迟 N 秒需要 N+1 次）。
set /a pings=!delay!+1
ping -n !pings! 127.0.0.1 >nul 2>&1
set /a delay=!delay!*2
if !delay! gtr %MAX_DELAY% set /a delay=%MAX_DELAY%
goto loop
