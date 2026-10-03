package ui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
)

// Preset 一个预设：只保存编码配置与输出容器。
type Preset struct {
	// 视频
	VideoDecoder string `json:"video_decoder"`
	VideoEncoder string `json:"video_encoder"`
	VideoPreset  string `json:"video_preset"`
	QualityMode  string `json:"quality_mode"`
	QualityValue string `json:"quality_value"`
	VideoBitrate string `json:"video_bitrate"`
	VideoMaxrate string `json:"video_maxrate"`
	VideoMinrate string `json:"video_minrate"`
	VideoWidth   string `json:"video_width"`
	VideoHeight  string `json:"video_height"`
	VideoFPS     string `json:"video_fps"`
	CustomParams string `json:"custom_params"`

	// 音频
	AudioEncoder string `json:"audio_encoder"`
	AudioBitrate string `json:"audio_bitrate"`
	SampleRate   string `json:"sample_rate"`
	BitDepth     string `json:"bit_depth"`

	// 输出容器
	OutputContainer string `json:"output_container"`

	// 自定义命令（有此字段时按该命令执行，占位符见 resolveCustomCommand）
	CustomCommand string `json:"custom_command"`
}

// fromConfig 从配置填充预设。
func (p *Preset) fromConfig(c *Config) {
	p.VideoDecoder = c.VideoDecoder
	p.VideoEncoder = c.VideoEncoder
	p.VideoPreset = c.VideoPreset
	p.QualityMode = c.QualityMode
	p.QualityValue = c.QualityValue
	p.VideoBitrate = c.VideoBitrate
	p.VideoMaxrate = c.VideoMaxrate
	p.VideoMinrate = c.VideoMinrate
	p.VideoWidth = c.VideoWidth
	p.VideoHeight = c.VideoHeight
	p.VideoFPS = c.VideoFPS
	p.CustomParams = c.CustomParams
	p.AudioEncoder = c.AudioEncoder
	p.AudioBitrate = c.AudioBitrate
	p.SampleRate = c.SampleRate
	p.BitDepth = c.BitDepth
	p.OutputContainer = c.OutputContainer
}

// applyTo 把预设应用到配置。
func (p *Preset) applyTo(c *Config) {
	c.VideoDecoder = p.VideoDecoder
	c.VideoEncoder = p.VideoEncoder
	c.VideoPreset = p.VideoPreset
	c.QualityMode = p.QualityMode
	c.QualityValue = p.QualityValue
	c.VideoBitrate = p.VideoBitrate
	c.VideoMaxrate = p.VideoMaxrate
	c.VideoMinrate = p.VideoMinrate
	c.VideoWidth = p.VideoWidth
	c.VideoHeight = p.VideoHeight
	c.VideoFPS = p.VideoFPS
	c.CustomParams = p.CustomParams
	c.AudioEncoder = p.AudioEncoder
	c.AudioBitrate = p.AudioBitrate
	c.SampleRate = p.SampleRate
	c.BitDepth = p.BitDepth
	c.OutputContainer = p.OutputContainer
}

// presetDir 预设保存目录：Windows 为 .\preset，其余平台为用户配置目录下的 preset。
func presetDir() string {
	if runtime.GOOS == "windows" {
		return filepath.Join(".", "preset")
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	return filepath.Join(dir, "ffmpeg-tui", "preset")
}

// presetDirs 预设读取目录列表（按优先级）：当前目录的 preset 在前，
// 随后是用户配置目录下的 preset（Windows 仅当前目录）。
func presetDirs() []string {
	dirs := []string{filepath.Join(".", "preset")}
	if runtime.GOOS == "windows" {
		return dirs
	}
	if d, err := os.UserConfigDir(); err == nil {
		dirs = append(dirs, filepath.Join(d, "ffmpeg-tui", "preset"))
	}
	return dirs
}

// presetDirLabel 预设目录的短标签（用于重名时的 [目录名]/ 前缀）
func presetDirLabel(dir string) string {
	base := filepath.Base(dir)
	parent := filepath.Base(filepath.Dir(dir))
	if parent == "." || parent == string(filepath.Separator) || parent == "" {
		return base
	}
	return parent + "/" + base
}

// presetEntry 一个预设条目：所在目录、名称与显示文本
type presetEntry struct {
	Dir   string
	Name  string
	Label string
}

// listPresetEntries 列出所有预设目录中的预设；同名预设同时显示，
// 并用 [目录名]/ 前缀区分。
func listPresetEntries() []presetEntry {
	dirs := presetDirs()
	namesByDir := make([][]string, len(dirs))
	count := map[string]int{}
	for i, dir := range dirs {
		names, err := listPresetNamesIn(dir)
		if err != nil {
			names = nil
		}
		namesByDir[i] = names
		for _, n := range names {
			count[n]++
		}
	}
	var entries []presetEntry
	for i, dir := range dirs {
		label := presetDirLabel(dir)
		for _, n := range namesByDir[i] {
			disp := n
			if count[n] > 1 {
				disp = "[" + label + "]/" + n
			}
			entries = append(entries, presetEntry{Dir: dir, Name: n, Label: disp})
		}
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Label < entries[j].Label })
	return entries
}

// listPresetNamesIn 返回指定目录中的预设名称列表（按名称排序）。
func listPresetNamesIn(dir string) ([]string, error) {
	items, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range items {
		if e.IsDir() {
			continue
		}
		base := e.Name()
		if !strings.HasSuffix(base, ".json") {
			continue
		}
		names = append(names, unescapeFilename(strings.TrimSuffix(base, ".json")))
	}
	sort.Strings(names)
	return names, nil
}

// filenameInvalidChar 报告字符是否不能出现在文件名中（含各平台非法字符与 %）。
func filenameInvalidChar(r rune) bool {
	if r <= 0x1f || r == 0x7f {
		return true
	}
	switch r {
	case '/', '\\', ':', '*', '?', '"', '<', '>', '|', '%':
		return true
	}
	return false
}

// escapeFilename 把预设名称转义为安全的文件名：非法字符用 %XX 十六进制表示。
func escapeFilename(name string) string {
	var sb strings.Builder
	for _, r := range name {
		if filenameInvalidChar(r) {
			sb.WriteString(fmt.Sprintf("%%%02X", r))
		} else {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

// unescapeFilename 把转义后的文件名还原为预设名称。
func unescapeFilename(name string) string {
	var sb strings.Builder
	for i := 0; i < len(name); i++ {
		if name[i] == '%' && i+2 < len(name) {
			if v, err := strconv.ParseUint(name[i+1:i+3], 16, 32); err == nil {
				sb.WriteRune(rune(v))
				i += 2
				continue
			}
		}
		sb.WriteByte(name[i])
	}
	return sb.String()
}

// savePreset 把预设保存为 <名称>.json（名称为文件名，已转义）。
func savePreset(name string, p *Preset) error {
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	dir := presetDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, escapeFilename(name)+".json"), data, 0o644)
}

// loadPresetFileAt 从指定目录按名称读取预设。
func loadPresetFileAt(dir, name string) (*Preset, error) {
	data, err := os.ReadFile(filepath.Join(dir, escapeFilename(name)+".json"))
	if err != nil {
		return nil, err
	}
	var p Preset
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// deletePresetAt 从指定目录按名称删除预设。
func deletePresetAt(dir, name string) error {
	return os.Remove(filepath.Join(dir, escapeFilename(name)+".json"))
}
