package ui

import (
	"os"
	"path/filepath"
	"testing"
)

// TestStoragePaths 校验配置/预设/语言文件的默认目录统一为 <home>/Documents/ffmpeg-tui
func TestStoragePaths(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	base := filepath.Join(home, "Documents", "ffmpeg-tui")
	if got := baseDir(); got != base {
		t.Errorf("baseDir=%q want %q", got, base)
	}
	if got, want := defaultConfigPath(), filepath.Join(base, "config.json"); got != want {
		t.Errorf("defaultConfigPath=%q want %q", got, want)
	}
	if got, want := presetDir(), filepath.Join(base, "preset"); got != want {
		t.Errorf("presetDir=%q want %q", got, want)
	}
	if got, want := langDir(), filepath.Join(base, "lang"); got != want {
		t.Errorf("langDir=%q want %q", got, want)
	}
	// 语言文件与预设一致：当前目录的 lang 也参与读取
	dirs := langDirs()
	if len(dirs) != 2 || dirs[0] != filepath.Join(".", "lang") || dirs[1] != langDir() {
		t.Errorf("langDirs=%v", dirs)
	}
}

// TestPresetAndLangDirsPortable 校验便携模式：当前目录有 config.json 时写入当前目录
func TestPresetAndLangDirsPortable(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := t.TempDir()
	wd, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(wd)

	if portableMode() {
		t.Errorf("当前目录没有 config.json 时不应是便携模式")
	}
	// 创建当前目录的 config.json 后进入便携模式
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !portableMode() {
		t.Errorf("存在 config.json 时应是便携模式")
	}
	if got, want := configSavePath(), filepath.Join(".", "config.json"); got != want {
		t.Errorf("configSavePath=%q want %q", got, want)
	}
	if got, want := presetSaveDir(), filepath.Join(".", "preset"); got != want {
		t.Errorf("presetSaveDir=%q want %q", got, want)
	}
}
