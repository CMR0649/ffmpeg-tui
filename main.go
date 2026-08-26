// ffmpeg-tui —— 一个用于 FFmpeg 的终端用户界面（TUI）。
// 双击可执行文件时若无终端上下文，会自动尝试在终端模拟器中重新启动自身。
package main

import (
	"fmt"
	"os"

	"ffmpeg-tui/ui"
)

func main() {
	// 没有终端（如文件管理器中双击可执行文件）时：
	// 优先尝试在终端模拟器中重新启动，失败则给出提示。
	if !hasTerminal() {
		if relaunchInTerminal() {
			return // 已在终端模拟器中重新启动，退出本进程
		}
		exe, _ := os.Executable()
		fmt.Println("ffmpeg-tui 需要在终端中运行。")
		fmt.Println("请在终端中执行以下命令：")
		fmt.Println()
		fmt.Println("  " + exe)
		fmt.Println()
		fmt.Println("按回车键退出…")
		_, _ = fmt.Scanln()
		return
	}
	ui.NewApp().Run()
}
