package ui

import (
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// buildCommand 根据当前配置生成 ffmpeg 转码命令参数。
// 选项映射以 FFmpeg 官方文档为准：-c:v/-c:a、-preset、-crf、-q:v、
// -b:v/-maxrate/-minrate/-bufsize、-s、-r、-b:a、-ar。
func (a *App) buildCommand(input, output string) []string {
	args := []string{"-y"}
	if a.cfg.VideoDecoder != "" {
		args = append(args, "-c:v", a.cfg.VideoDecoder) // 输入流解码器
	}
	args = append(args, "-i", input)

	// 视频
	if a.cfg.VideoEncoder != "" {
		args = append(args, "-c:v", a.cfg.VideoEncoder)
		if a.cfg.VideoPreset != "" {
			args = append(args, "-preset", a.cfg.VideoPreset)
		}
		switch a.cfg.QualityMode {
		case "恒定质量 CRF":
			if a.cfg.QualityValue != "" {
				args = append(args, "-crf", a.cfg.QualityValue)
			}
		case "可变码率 VBR":
			if a.cfg.QualityValue != "" {
				args = append(args, "-q:v", a.cfg.QualityValue)
			}
		case "固定码率 CBR":
			if b := a.cfg.VideoBitrate; b != "" {
				args = append(args, "-b:v", b+"k", "-minrate", b+"k", "-maxrate", b+"k", "-bufsize", cbrBufsize(b)+"k")
			}
		}
		if a.cfg.VideoBitrate != "" && a.cfg.QualityMode != "固定码率 CBR" {
			args = append(args, "-b:v", a.cfg.VideoBitrate+"k")
		}
		if a.cfg.VideoMaxrate != "" && a.cfg.QualityMode != "固定码率 CBR" {
			args = append(args, "-maxrate", a.cfg.VideoMaxrate+"k")
		}
		if a.cfg.VideoMinrate != "" && a.cfg.QualityMode != "固定码率 CBR" {
			args = append(args, "-minrate", a.cfg.VideoMinrate+"k")
		}
		if a.cfg.VideoWidth != "" && a.cfg.VideoHeight != "" {
			args = append(args, "-s", a.cfg.VideoWidth+"x"+a.cfg.VideoHeight)
		}
		if a.cfg.VideoFPS != "" {
			args = append(args, "-r", a.cfg.VideoFPS)
		}
	} else {
		args = append(args, "-c:v", "copy")
	}

	// 音频
	if a.cfg.AudioEncoder != "" {
		args = append(args, "-c:a", a.cfg.AudioEncoder)
		if a.cfg.AudioBitrate != "" {
			args = append(args, "-b:a", a.cfg.AudioBitrate+"k")
		}
		if a.cfg.SampleRate != "" {
			args = append(args, "-ar", a.cfg.SampleRate)
		}
	} else {
		args = append(args, "-c:a", "copy")
	}

	args = append(args, output)
	return args
}

// cbrBufsize CBR 模式的 -bufsize（取基础比特率的 2 倍）。
func cbrBufsize(bitrate string) string {
	n, err := strconv.Atoi(bitrate)
	if err != nil || n <= 0 {
		return "0"
	}
	return strconv.Itoa(n * 2)
}

// outputPath 根据输出目录（$file 规则）与输出命名规则生成输出路径。
// 输入文件在不同目录时分别输出到与之对应的目录。
func (a *App) outputPath(input string) string {
	dir := ResolveOutputDir(a.cfg.OutputDir, input)
	base := strings.TrimSuffix(filepath.Base(input), filepath.Ext(input))
	ext := "." + a.outputContainer
	name := base + ext
	switch a.cfg.OutputNaming {
	case "timestamp":
		name = base + "_" + TimestampFormat("YYYY-M-D-hhmmss", time.Now()) + ext
	case "suffix":
		if a.cfg.Suffix != "" {
			name = base + a.cfg.Suffix + ext
		}
	}
	return filepath.Join(dir, name)
}
