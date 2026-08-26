#!/bin/sh
set -eu
cd "$(dirname "$0")"
OUT="bin"
mkdir -p "$OUT"
go mod tidy
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o "$OUT/ffmpeg-tui" .
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o "$OUT/ffmpeg-tui.exe" .
ls -lh "$OUT"
