package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// relaunchEnv 标记进程已由终端模拟器重新启动（防止递归启动）。
const relaunchEnv = "FFMPEG_TUI_IN_TERMINAL"

// hasTerminal 报告当前进程是否已连接到终端（可运行 TUI）。
func hasTerminal() bool {
	if os.Getenv(relaunchEnv) == "1" {
		return true
	}
	switch runtime.GOOS {
	case "windows":
		return windowsHasConsole()
	default:
		f, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
		if err != nil {
			return false
		}
		f.Close()
		return true
	}
}

// relaunchInTerminal 尝试在终端模拟器中重新启动自身（用于双击可执行文件
// 而无终端上下文的情况）；成功返回 true。
func relaunchInTerminal() bool {
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	env := append(os.Environ(), relaunchEnv+"=1")
	switch runtime.GOOS {
	case "windows":
		// 用 PowerShell 在新建控制台窗口中启动自身。
		ps := "Start-Process -FilePath '" + strings.ReplaceAll(exe, "'", "''") +
			"' -WorkingDirectory '" + strings.ReplaceAll(filepath.Dir(exe), "'", "''") + "'"
		cmd := exec.Command("powershell", "-NoProfile", "-Command", ps)
		cmd.Env = env
		return cmd.Start() == nil
	default:
		// Linux 等：依次尝试常见终端模拟器，统一使用 "-e <程序>" 形式
		// （Ghostty/gnome-terminal/konsole/xfce4-terminal/xterm 均支持；
		// 注意不能用 "--"：Ghostty 会把 "--" 当作未知配置键解析，报
		// "unknown field" / "invalid field" 错误）。
		for _, t := range []string{
			"ghostty",
			"gnome-terminal",
			"x-terminal-emulator",
			"konsole",
			"xfce4-terminal",
			"xterm",
		} {
			if p, err := exec.LookPath(t); err == nil {
				cmd := exec.Command(p, "-e", exe)
				cmd.Env = env
				if err := cmd.Start(); err == nil {
					return true
				}
			}
		}
	}
	return false
}
