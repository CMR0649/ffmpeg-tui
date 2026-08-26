#!/bin/sh
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
