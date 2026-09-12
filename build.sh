#!/bin/sh
set -eu
cd "$(dirname "$0")"
OUT="bin"
mkdir -p "$OUT"
go mod tidy
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o "$OUT/ffmpeg-tui_linux" .
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o "$OUT/ffmpeg-tui_win64.exe" .
GOOS=windows GOARCH=386 CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o "$OUT/ffmpeg-tui_win32.exe" .
ls -lh "$OUT"
