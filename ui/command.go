package ui

import (
	"path/filepath"
	"strings"
	"time"
)

// encoderDisabled 编码器「禁用」的存储值
const encoderDisabled = "none"

// buildCommand 根据当前配置与输入文件的流类型生成 ffmpeg 转码命令参数
// 选项映射以 FFmpeg 官方文档为准：-c:v/-c:a、-preset、码率控制参数（-crf 等）、
// -b:v/-maxrate/-minrate、-s、-r、-b:a、-ar、-sample_fmt
// 编码器为「禁用」时不输出对应流
func (a *App) buildCommand(input, output string, hasVideo, hasAudio bool) []string {
	args := []string{"-y"}
	if a.cfg.VideoDecoder != "" {
		args = append(args, "-c:v", a.cfg.VideoDecoder) // 输入流解码器
	}
	args = append(args, "-i", input)

	videoDisabled := a.cfg.VideoEncoder == encoderDisabled
	audioDisabled := a.cfg.AudioEncoder == encoderDisabled

	// 视频
	if hasVideo && !videoDisabled {
		if a.cfg.VideoEncoder != "" {
			args = append(args, "-c:v", a.cfg.VideoEncoder)
			if a.cfg.VideoPreset != "" {
				args = append(args, "-preset", a.cfg.VideoPreset)
			}
			args = append(args, qualityArgs(a.cfg.QualityMode, a.cfg.QualityValue)...)
			if a.cfg.VideoBitrate != "" {
				args = append(args, "-b:v", a.cfg.VideoBitrate+"k")
			}
			if a.cfg.VideoMaxrate != "" {
				args = append(args, "-maxrate", a.cfg.VideoMaxrate+"k")
			}
			if a.cfg.VideoMinrate != "" {
				args = append(args, "-minrate", a.cfg.VideoMinrate+"k")
			}
			if a.cfg.VideoWidth != "" && a.cfg.VideoHeight != "" {
				args = append(args, "-s", a.cfg.VideoWidth+"x"+a.cfg.VideoHeight)
			}
			if a.cfg.VideoFPS != "" {
				args = append(args, "-r", a.cfg.VideoFPS)
			}
		} else {
			args = append(args, "-c:v", "copy") // 默认复制流
		}
		// 自定义参数：追加到生成的视频参数末尾
		if a.cfg.CustomParams != "" {
			args = append(args, strings.Fields(a.cfg.CustomParams)...)
		}
	}

	// 音频
	if hasAudio && !audioDisabled {
		if a.cfg.AudioEncoder != "" {
			args = append(args, "-c:a", a.cfg.AudioEncoder)
			if a.cfg.AudioBitrate != "" {
				args = append(args, "-b:a", a.cfg.AudioBitrate+"k")
			}
			if a.cfg.SampleRate != "" {
				args = append(args, "-ar", a.cfg.SampleRate)
			}
			if a.cfg.BitDepth != "" {
				// 音频位深：-sample_fmt s[位深]，如 -sample_fmt s16
				args = append(args, "-sample_fmt", "s"+a.cfg.BitDepth)
			}
		} else {
			args = append(args, "-c:a", "copy") // 默认复制流
		}
	}

	if a.cfg.OutputContainer != "" {
		args = append(args, "-f", a.cfg.OutputContainer) // 显式指定输出容器格式
	}
	args = append(args, output)
	return args
}

// qualityArgs 码率控制方式对应的参数：质量值紧跟其参数（-qp_i -qp_p 为两个参数）
func qualityArgs(mode, value string) []string {
	if value == "" {
		return nil
	}
	switch qualityModeKey(mode) {
	case "qp":
		return []string{"-qp", value}
	case "cq":
		return []string{"-cq", value}
	case "qp_i_p":
		return []string{"-qp_i", value, "-qp_p", value}
	case "global_quality":
		return []string{"-global_quality", value}
	default:
		return []string{"-crf", value}
	}
}

// containerExt 输出容器格式名 → 文件扩展名。
// 仅列出格式名与常用后缀不一致的项，其余直接用格式名。
func containerExt(format string) string {
	switch format {
	case "matroska":
		return "mkv"
	case "mpegts":
		return "ts"
	case "mpeg", "mpegvideo", "mpeg1video", "mpeg2video":
		return "mpg"
	case "asf":
		return "wmv"
	case "adts":
		return "aac"
	case "hls":
		return "m3u8"
	case "dash":
		return "mpd"
	}
	return format
}

// outputPath 根据输出目录（$file 规则）与输出命名规则生成输出路径
// 输入文件在不同目录时分别输出到与之对应的目录
func (a *App) outputPath(input string) string {
	dir := ResolveOutputDir(a.cfg.OutputDir, input)
	base := strings.TrimSuffix(filepath.Base(input), filepath.Ext(input))
	ext := ""
	if a.cfg.OutputContainer != "" {
		ext = "." + containerExt(a.cfg.OutputContainer)
	}
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

// qualityModeKey 归一化码率控制方式：兼容旧配置的 vbr/cbr 与中文值
func qualityModeKey(mode string) string {
	switch mode {
	case "qp":
		return "qp"
	case "cq":
		return "cq"
	case "qp_i_p":
		return "qp_i_p"
	case "global_quality":
		return "global_quality"
	}
	return "crf"
}
