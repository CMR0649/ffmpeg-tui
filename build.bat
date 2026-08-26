@echo off
REM ffmpeg-tui 构建脚本（Windows 环境）
REM 生成：
REM   bin\ffmpeg-tui.exe  —— Windows 可执行文件
REM   bin\ffmpeg-tui      —— Linux 可执行文件（交叉编译）
REM 前置：已安装 Go 工具链
setlocal
cd /d "%~dp0"

set OUT=bin
if not exist "%OUT%" mkdir "%OUT%"

echo ==^> 准备依赖 (go mod tidy)
go mod tidy
if errorlevel 1 exit /b 1

echo ==^> 构建 Windows 可执行文件
set GOOS=windows
set GOARCH=amd64
set CGO_ENABLED=0
go build -trimpath -ldflags "-s -w" -o "%OUT%\ffmpeg-tui.exe" .
if errorlevel 1 exit /b 1

echo ==^> 构建 Linux 可执行文件（交叉编译）
set GOOS=linux
set GOARCH=amd64
set CGO_ENABLED=0
go build -trimpath -ldflags "-s -w" -o "%OUT%\ffmpeg-tui" .
if errorlevel 1 exit /b 1

echo ==^> 完成
dir /b "%OUT%"
endlocal
