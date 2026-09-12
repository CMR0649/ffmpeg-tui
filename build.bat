@echo off
setlocal
cd /d "%~dp0"
set OUT=bin
if not exist "%OUT%" mkdir "%OUT%"
go mod tidy
if errorlevel 1 exit /b 1
set GOOS=windows
set GOARCH=amd64
set CGO_ENABLED=0
go build -trimpath -ldflags "-s -w" -o "%OUT%\ffmpeg-tui_win64.exe" .
if errorlevel 1 exit /b 1
set GOOS=windows
set GOARCH=386
set CGO_ENABLED=0
go build -trimpath -ldflags "-s -w" -o "%OUT%\ffmpeg-tui_win32.exe" .
if errorlevel 1 exit /b 1
set GOOS=linux
set GOARCH=amd64
set CGO_ENABLED=0
go build -trimpath -ldflags "-s -w" -o "%OUT%\ffmpeg-tui_linux" .
if errorlevel 1 exit /b 1
dir /b "%OUT%"
endlocal
