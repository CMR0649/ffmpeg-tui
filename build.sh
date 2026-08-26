#!/bin/sh
# ffmpeg-tui 构建脚本（Linux 环境，POSIX sh 兼容，可在容器内运行）
# 生成：
#   bin/ffmpeg-tui      —— Linux 可执行文件
#   bin/ffmpeg-tui.exe  —— Windows 可执行文件（交叉编译）
# 前置：已安装 Go 工具链（推荐在构建容器中执行，见 ../ffmpeg-tui-container/build.sh）
set -eu
cd "$(dirname "$0")"

OUT="bin"
mkdir -p "$OUT"

echo "==> 准备依赖 (go mod tidy)"
go mod tidy

echo "==> 构建 Linux 可执行文件"
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o "$OUT/ffmpeg-tui" .

echo "==> 构建 Windows 可执行文件（交叉编译）"
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o "$OUT/ffmpeg-tui.exe" .

echo "==> 完成"
ls -lh "$OUT"
