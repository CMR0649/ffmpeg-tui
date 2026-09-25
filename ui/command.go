package ui

import (
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// encoderDisabled 编码器「禁用」的存储值
const encoderDisabled = "none"

// buildCommand 根据当前配置与输入文件的流类型生成 ffmpeg 转码命令参数
// 选项映射以 FFmpeg 官方文档为准：-map、-c:v/-c:a、-preset、-crf、-q:v、
// -b:v/-maxrate/-minrate/-bufsize、-s、-r、-b:a、-ar、-sample_fmt
// 输入文件参数后始终加 -map 0:v:0? / -map 0:a:0?（? 表示流不存在时忽略）；
// 编码器为「禁用」时不映射对应流
func (a *App) buildCommand(input, output string, hasVideo, hasAudio bool) []string {
	args := []string{"-y"}
	if a.cfg.VideoDecoder != "" {
		args = append(args, "-c:v", a.cfg.VideoDecoder) // 输入流解码器
	}
	args = append(args, "-i", input)

	videoDisabled := a.cfg.VideoEncoder == encoderDisabled
	audioDisabled := a.cfg.AudioEncoder == encoderDisabled
	if !videoDisabled {
		args = append(args, "-map", "0:v:0?")
	}
	if !audioDisabled {
		args = append(args, "-map", "0:a:0?")
	}

	// 视频
	if hasVideo && !videoDisabled {
		if a.cfg.VideoEncoder != "" {
			args = append(args, "-c:v", a.cfg.VideoEncoder)
			if a.cfg.VideoPreset != "" {
				args = append(args, "-preset", a.cfg.VideoPreset)
			}
			switch qualityModeKey(a.cfg.QualityMode) {
			case "crf":
				if a.cfg.QualityValue != "" {
					args = append(args, "-crf", a.cfg.QualityValue)
				}
			case "vbr":
				// 质量值仅在恒定质量（CRF）模式可用，VBR 由比特率控制。
			case "cbr":
				if b := a.cfg.VideoBitrate; b != "" {
					args = append(args, "-b:v", b+"k", "-minrate", b+"k", "-maxrate", b+"k", "-bufsize", cbrBufsize(b)+"k")
				}
			}
			if a.cfg.VideoBitrate != "" && qualityModeKey(a.cfg.QualityMode) != "cbr" {
				args = append(args, "-b:v", a.cfg.VideoBitrate+"k")
			}
			if a.cfg.VideoMaxrate != "" && qualityModeKey(a.cfg.QualityMode) != "cbr" {
				args = append(args, "-maxrate", a.cfg.VideoMaxrate+"k")
			}
			if a.cfg.VideoMinrate != "" && qualityModeKey(a.cfg.QualityMode) != "cbr" {
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

// cbrBufsize CBR 模式的 -bufsize（取基础比特率的 2 倍）
func cbrBufsize(bitrate string) string {
	n, err := strconv.Atoi(bitrate)
	if err != nil || n <= 0 {
		return "0"
	}
	return strconv.Itoa(n * 2)
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

// qualityModeKey 归一化控制方式：兼容新键（crf/vbr/cbr）与旧配置的中文值
func qualityModeKey(mode string) string {
	switch mode {
	case "crf", "恒定质量 CRF":
		return "crf"
	case "vbr", "可变码率 VBR":
		return "vbr"
	case "cbr", "固定码率 CBR":
		return "cbr"
	}
	return "crf"
}
