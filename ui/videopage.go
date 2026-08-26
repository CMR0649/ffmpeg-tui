package ui

import (
	"fmt"
	"strings"

	"github.com/rivo/tview"
)

// buildVideoPage 构建「视频」页：编码 / 质量 / 画面 三组选项。
// 编码器列表从 ffmpeg 命令动态加载，编码器详情通过 `ffmpeg -h encoder=名称`
// 获取，选项以 FFmpeg 文档为准。
func (a *App) buildVideoPage() tview.Primitive {
	list := tview.NewList()
	list.SetBorder(true)
	list.SetTitle(a.s.VideoTitle)
	a.videoList = list
	a.refreshVideoPage()
	return list
}

// refreshVideoPage 按当前配置与语言重建视频页选项列表。
func (a *App) refreshVideoPage() {
	l := a.videoList
	l.Clear()

	// 编码
	l.AddItem(a.s.Encoder, a.videoEncoderLabel(), 0, func() { a.editVideoEncoder() })
	l.AddItem(a.s.Decoder, a.videoDecoderLabel(), 0, func() { a.editVideoDecoder() })
	l.AddItem(a.s.Preset, a.videoPresetLabel(), 0, func() { a.editVideoPreset() })

	// 质量
	l.AddItem(a.s.QualityMode, a.qualityModeLabel(), 0, func() { a.editQualityMode() })
	l.AddItem(a.s.Quality, a.qualityValueLabel(), 0, func() { a.editQualityValue() })
	l.AddItem(a.s.BitrateBase, a.bitrateLabel("VideoBitrate"), 0, func() { a.editVideoBitrate(a.s.BitrateBase, "VideoBitrate") })
	l.AddItem(a.s.BitrateMax, a.bitrateLabel("VideoMaxrate"), 0, func() { a.editVideoBitrate(a.s.BitrateMax, "VideoMaxrate") })
	l.AddItem(a.s.BitrateMin, a.bitrateLabel("VideoMinrate"), 0, func() { a.editVideoBitrate(a.s.BitrateMin, "VideoMinrate") })

	// 画面
	l.AddItem(a.s.Resolution, a.resolutionLabel(), 0, func() { a.editResolution() })
	l.AddItem(a.s.FPS, a.fpsLabel(), 0, func() { a.editFPS() })
}

// ---------- 编码 ----------

// editVideoEncoder 从动态加载的视频编码器列表中选择编码器（默认复制流）。
func (a *App) editVideoEncoder() {
	loadCodecLists(a.ffmpegBin())
	labels := make([]string, 0, len(videoEncoders)+1)
	labels = append(labels, a.copyStreamLabel())
	labels = append(labels, videoEncoders...)
	a.showOptionDialog(a.s.Encoder, labels, func(i int) {
		if i == 0 {
			a.cfg.VideoEncoder = ""
			a.cfg.VideoPreset = ""
			a.refreshVideoPage()
			return
		}
		name := videoEncoders[i-1]
		a.cfg.VideoEncoder = name
		info := probeEncoder(a.ffmpegBin(), name)
		if info.HasOption("preset") {
			a.cfg.VideoPreset = "medium"
		} else {
			a.cfg.VideoPreset = ""
		}
		a.refreshVideoPage()
	})
}

// copyStreamLabel 「复制流（默认）」按当前语言。
func (a *App) copyStreamLabel() string {
	if a.lang == "en" {
		return "Copy stream (default)"
	}
	return "复制流（默认）"
}

// videoEncoderLabel 编码器项的当前值显示（含 ffmpeg -h 获取的描述）。
func (a *App) videoEncoderLabel() string {
	if a.cfg.VideoEncoder == "" {
		return a.copyStreamLabel()
	}
	info := probeEncoder(a.ffmpegBin(), a.cfg.VideoEncoder)
	if info.Description != "" {
		return a.cfg.VideoEncoder + "（" + info.Description + "）"
	}
	return a.cfg.VideoEncoder
}

// editVideoDecoder 解码器（可以为空），列表动态加载。
func (a *App) editVideoDecoder() {
	loadCodecLists(a.ffmpegBin())
	names := videoDecoderNames
	if len(videoDecoders) > 0 {
		names = make([]string, 0, len(videoDecoders)+1)
		names = append(names, a.autoSelectLabel())
		names = append(names, videoDecoders...)
	}
	a.showOptionDialog(a.s.Decoder, names, func(i int) {
		if names[i] == a.autoSelectLabel() {
			a.cfg.VideoDecoder = ""
		} else {
			a.cfg.VideoDecoder = names[i]
		}
		a.refreshVideoPage()
	})
}

// autoSelectLabel 「自动选择」按当前语言。
func (a *App) autoSelectLabel() string {
	if a.lang == "en" {
		return "auto"
	}
	return "自动选择"
}

// videoDecoderLabel 解码器项的当前值显示。
func (a *App) videoDecoderLabel() string {
	if a.cfg.VideoDecoder == "" {
		return a.autoSelectLabel()
	}
	return a.cfg.VideoDecoder
}

// editVideoPreset 预设（视编码器而定）：优先使用 `ffmpeg -h encoder=名称`
// 返回的 -preset 选项枚举值，无枚举值时回退到通用 x264 风格预设列表。
func (a *App) editVideoPreset() {
	if !a.videoEncoderHasPreset() {
		a.showMessageDialog(a.s.Preset, a.s.PresetUnsupported)
		return
	}
	values := presetValuesFor(a.ffmpegBin(), a.cfg.VideoEncoder)
	if len(values) == 0 {
		values = x264Presets
	}
	a.showOptionDialog(a.s.Preset, values, func(i int) {
		a.cfg.VideoPreset = values[i]
		a.refreshVideoPage()
	})
}

// presetValuesFor 返回编码器 -preset 选项的枚举取值（来自 ffmpeg -h encoder=）。
func presetValuesFor(bin, encoder string) []string {
	if encoder == "" {
		return nil
	}
	return probeEncoder(bin, encoder).OptionValues["preset"]
}

// videoEncoderHasPreset 报告当前编码器是否支持 -preset 选项。
func (a *App) videoEncoderHasPreset() bool {
	if a.cfg.VideoEncoder == "" {
		return false
	}
	return probeEncoder(a.ffmpegBin(), a.cfg.VideoEncoder).HasOption("preset")
}

// videoPresetLabel 预设项的当前值显示。
func (a *App) videoPresetLabel() string {
	if a.cfg.VideoPreset == "" {
		return "—"
	}
	return a.cfg.VideoPreset
}

// ---------- 质量 ----------

// qualityModeOptions 控制方式选项（中文+缩写，按当前语言），返回
// 显示文本与存储键。
func (a *App) qualityModeOptions() ([]string, []string) {
	return []string{a.s.QualityCRF, a.s.QualityVBR, a.s.QualityCBR},
		[]string{"crf", "vbr", "cbr"}
}

// editQualityMode 控制方式（中文 + 缩写）。
func (a *App) editQualityMode() {
	labels, keys := a.qualityModeOptions()
	a.showOptionDialog(a.s.QualityMode, labels, func(i int) {
		a.cfg.QualityMode = keys[i]
		a.refreshVideoPage()
	})
}

// qualityModeLabel 控制方式项的当前值显示（兼容旧配置存中文值）。
func (a *App) qualityModeLabel() string {
	labels, _ := a.qualityModeOptions()
	switch a.cfg.QualityMode {
	case "crf", "恒定质量 CRF":
		return labels[0]
	case "vbr", "可变码率 VBR":
		return labels[1]
	case "cbr", "固定码率 CBR":
		return labels[2]
	}
	return labels[0]
}

// videoEncoderHasQuality 报告当前编码器是否支持设置质量值（-crf / -qscale / -q:v / -cq）。
func (a *App) videoEncoderHasQuality() bool {
	if a.cfg.VideoEncoder == "" {
		return false
	}
	info := probeEncoder(a.ffmpegBin(), a.cfg.VideoEncoder)
	return info.HasOption("crf") || info.HasOption("qscale") || info.HasOption("q:v") || info.HasOption("cq")
}

// editQualityValue 质量值：编码器不支持时提示。
func (a *App) editQualityValue() {
	if !a.videoEncoderHasQuality() {
		a.showMessageDialog(a.s.Quality, a.s.QualityUnsupportedMsg)
		return
	}
	a.showInputDialog(a.s.Quality, a.cfg.QualityValue, func(text string) {
		text = strings.TrimSpace(text)
		if text == "" {
			return
		}
		a.cfg.QualityValue = text
		a.refreshVideoPage()
	})
}

// qualityValueLabel 质量项的当前值显示。
func (a *App) qualityValueLabel() string {
	if !a.videoEncoderHasQuality() {
		return a.s.QualityUnsupported
	}
	if a.cfg.QualityValue == "" {
		return "—"
	}
	return a.cfg.QualityValue
}

// editVideoBitrate 比特率输入（kbps）。
func (a *App) editVideoBitrate(title, field string) {
	a.showInputDialog(title+"（kbps）", a.bitrateValue(field), func(text string) {
		text = strings.TrimSpace(text)
		switch field {
		case "VideoBitrate":
			a.cfg.VideoBitrate = text
		case "VideoMaxrate":
			a.cfg.VideoMaxrate = text
		case "VideoMinrate":
			a.cfg.VideoMinrate = text
		}
		a.refreshVideoPage()
	})
}

// bitrateValue 返回比特率字段的原始值。
func (a *App) bitrateValue(field string) string {
	switch field {
	case "VideoBitrate":
		return a.cfg.VideoBitrate
	case "VideoMaxrate":
		return a.cfg.VideoMaxrate
	case "VideoMinrate":
		return a.cfg.VideoMinrate
	}
	return ""
}

// bitrateLabel 比特率项的当前值显示（kbps）。
func (a *App) bitrateLabel(field string) string {
	v := a.bitrateValue(field)
	if v == "" {
		return "—"
	}
	return v + " kbps"
}

// ---------- 画面 ----------

// editResolution 分辨率：宽度与高度分成两个输入框。
func (a *App) editResolution() {
	a.showInputDialog(a.s.Resolution+" W", a.cfg.VideoWidth, func(w string) {
		a.cfg.VideoWidth = strings.TrimSpace(w)
		a.showInputDialog(a.s.Resolution+" H", a.cfg.VideoHeight, func(h string) {
			a.cfg.VideoHeight = strings.TrimSpace(h)
			a.refreshVideoPage()
		})
	})
}

// resolutionLabel 分辨率项的当前值显示。
func (a *App) resolutionLabel() string {
	if a.cfg.VideoWidth == "" && a.cfg.VideoHeight == "" {
		return a.originalLabel()
	}
	return fmt.Sprintf("%s×%s", a.cfg.VideoWidth, a.cfg.VideoHeight)
}

// editFPS 帧率输入。
func (a *App) editFPS() {
	a.showInputDialog(a.s.FPS, a.cfg.VideoFPS, func(text string) {
		a.cfg.VideoFPS = strings.TrimSpace(text)
		a.refreshVideoPage()
	})
}

// fpsLabel 帧率项的当前值显示。
func (a *App) fpsLabel() string {
	if a.cfg.VideoFPS == "" {
		return a.originalLabel()
	}
	return a.cfg.VideoFPS
}

// originalLabel 「原始」按当前语言。
func (a *App) originalLabel() string {
	if a.lang == "en" {
		return "original"
	}
	return "原始"
}
