package ui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Config 保存全部可配置选项（视频/音频/文件输出/设置）。
// 选项名与取值以 FFmpeg 官方文档（https://ffmpeg.org/documentation.html）为准。
type Config struct {
	// 视频
	VideoDecoder string `json:"video_decoder"` // 可为空
	VideoEncoder string `json:"video_encoder"` // 如 libx264；空表示复制流
	VideoPreset  string `json:"video_preset"`  // 视编码器而定，如 libx264 的 slow
	QualityMode  string `json:"quality_mode"`  // 恒定质量 CRF / 可变码率 VBR / 固定码率 CBR
	QualityValue string `json:"quality_value"` // 质量值（如 CRF 23）
	VideoBitrate string `json:"video_bitrate"` // 基础比特率，kbps
	VideoMaxrate string `json:"video_maxrate"` // 最高比特率，kbps
	VideoMinrate string `json:"video_minrate"` // 最低比特率，kbps
	VideoWidth   string `json:"video_width"`   // 分辨率宽度
	VideoHeight  string `json:"video_height"`  // 分辨率高度
	VideoFPS     string `json:"video_fps"`     // 帧率

	// 音频
	AudioEncoder string `json:"audio_encoder"` // 如 aac；空表示复制流
	AudioBitrate string `json:"audio_bitrate"` // 比特率，kbps
	SampleRate   string `json:"sample_rate"`   // 采样率，Hz
	BitDepth     string `json:"bit_depth"`     // 位深度（WAV/PCM 编码器，与编码器名称同步）

	// 文件 / 输出
	OutputDir    string `json:"output_dir"`    // 空或 $file = 输入文件所在目录
	OutputNaming string `json:"output_naming"` // timestamp / suffix / none
	Suffix       string `json:"suffix"`        // 指定后缀
}

// DefaultConfig 返回默认配置。
func DefaultConfig() *Config {
	return &Config{
		QualityMode:  "恒定质量 CRF",
		QualityValue: "23",
		OutputNaming: "timestamp",
	}
}

// defaultConfigPath 返回默认配置文件路径（用户配置目录）。
func defaultConfigPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	return filepath.Join(dir, "ffmpeg-tui", "config.json")
}

// fileExists 报告文件是否存在。
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// SaveJSON 把配置保存为 JSON 文件。
func (c *Config) SaveJSON(path string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return os.WriteFile(path, data, 0o644)
}

// LoadJSON 从 JSON 文件加载配置。
func (c *Config) LoadJSON(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, c)
}

// TimestampFormat 按给定格式格式化时间戳：
// 字母 Y/M/D/h/m/s（年/月/日/时/分/秒）的连续个数表示补零位数。
// 例如 "YYYY-M-D-hhmmss" → "2026-8-26-143000"。
func TimestampFormat(format string, t time.Time) string {
	var sb strings.Builder
	for i := 0; i < len(format); i++ {
		c := format[i]
		if c != 'Y' && c != 'M' && c != 'D' && c != 'h' && c != 'm' && c != 's' {
			sb.WriteByte(c)
			continue
		}
		j := i
		for j < len(format) && format[j] == c {
			j++
		}
		n := j - i
		var v int
		switch c {
		case 'Y':
			v = t.Year()
		case 'M':
			v = int(t.Month())
		case 'D':
			v = t.Day()
		case 'h':
			v = t.Hour()
		case 'm':
			v = t.Minute()
		case 's':
			v = t.Second()
		}
		s := strconv.Itoa(v)
		if n > len(s) {
			s = strings.Repeat("0", n-len(s)) + s
		}
		sb.WriteString(s)
		i = j - 1
	}
	return sb.String()
}

// ResolveOutputDir 解析输出目录：设置值为空或 $file 时返回输入文件所在目录；
// 设置值含 $file 时替换为输入文件所在目录；否则返回设置值本身。
// 输入文件在不同目录时，各文件输出到与之对应的目录。
func ResolveOutputDir(setting, inputPath string) string {
	dir := filepath.Dir(inputPath)
	if setting == "" || setting == "$file" {
		return dir
	}
	return strings.ReplaceAll(setting, "$file", dir)
}
