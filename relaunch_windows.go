//go:build windows

package main

import "syscall"

// windowsHasConsole 检测 Windows 进程是否拥有控制台窗口。
// 双击控制台程序时 Windows 会分配新控制台；无控制台（如从 GUI 上下文
// 启动）时返回 false，交由 relaunchInTerminal 重新启动。
func windowsHasConsole() bool {
	k32 := syscall.NewLazyDLL("kernel32.dll")
	getConsoleWindow := k32.NewProc("GetConsoleWindow")
	h, _, _ := getConsoleWindow.Call()
	return h != 0
}
