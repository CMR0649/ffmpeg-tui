//go:build !windows

package main

// windowsHasConsole 在非 Windows 平台不使用。
func windowsHasConsole() bool { return true }
