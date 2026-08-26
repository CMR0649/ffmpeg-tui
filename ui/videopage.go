package ui

import (
	"fmt"
	"strings"

	"github.com/rivo/tview"
)

// buildVideoPage 构建「视频」页：编码 / 质量 / 画面 三组选项。
// 选项与取值以 FFmpeg 官方文档为准。
func (a *App) buildVideoPage() tview.Primitive {
	list := tview.NewList()
	list.SetBorder(true)
	list.SetTitle(" 视频 — 编码选项 ")
	a.videoList = list
	a.refreshVideoPage()
	return list
}

// refreshVideoPage 按当前配置重建视频页选项列表。
func (a *App) refreshVideoPage() {
	l := a.videoList
	l.Clear()

	// 编码
	l.AddItem("编码器", a.videoEncoderLabel(), 0, func() { a.editVideoEncoder() })
	l.AddItem("解码器", a.videoDecoderLabel(), 0, func() { a.editVideoDecoder() })
	l.AddItem("预设", a.videoPresetLabel(), 0, func() { a.editVideoPreset() })

	// 质量
	l.AddItem("控制方式", a.qualityModeLabel(), 0, func() { a.editQualityMode() })
	l.AddItem("质量", a.qualityValueLabel(), 0, func() { a.editQualityValue() })
	l.AddItem("基础比特率", a.bitrateLabel("VideoBitrate"), 0, func() { a.editVideoBitrate("基础比特率", "VideoBitrate") })
	l.AddItem("最高比特率", a.bitrateLabel("VideoMaxrate"), 0, func() { a.editVideoBitrate("最高比特率", "VideoMaxrate") })
	l.AddItem("最低比特率", a.bitrateLabel("VideoMinrate"), 0, func() { a.editVideoBitrate("最低比特率", "VideoMinrate") })

	// 画面
	l.AddItem("分辨率", a.resolutionLabel(), 0, func() { a.editResolution() })
	l.AddItem("帧率", a.fpsLabel(), 0, func() { a.editFPS() })
}

// ---------- 编码 ----------

// editVideoEncoder 编码器两级选择：先选编码（如 H.264），再选具体编码器（如 libx264）。
func (a *App) editVideoEncoder() {
	labels := make([]string, len(videoEncoders))
	for i, e := range videoEncoders {
		labels[i] = e.Label
	}
	a.showOptionDialog("编码器（默认复制流）", labels, func(i int) {
		enc := videoEncoders[i]
		if len(enc.Encoders) == 0 {
			a.cfg.VideoEncoder = ""
			a.cfg.VideoPreset = ""
			a.refreshVideoPage()
			return
		}
		if len(enc.Encoders) == 1 {
			a.cfg.VideoEncoder = enc.Encoders[0]
			a.cfg.VideoPreset = defaultPreset(enc)
			a.refreshVideoPage()
			return
		}
		a.showOptionDialog(enc.Label, enc.Encoders, func(j int) {
			a.cfg.VideoEncoder = enc.Encoders[j]
			a.cfg.VideoPreset = defaultPreset(enc)
			a.refreshVideoPage()
		})
	})
}

// defaultPreset 返回编码的默认预设（x264/x265 为 medium，其余为空）。
func defaultPreset(enc VideoEncoderInfo) string {
	if len(enc.Presets) > 0 {
		return "medium"
	}
	return ""
}

// currentVideoEncoder 返回当前编码器对应的 VideoEncoderInfo（未匹配时返回 nil）。
func (a *App) currentVideoEncoder() *VideoEncoderInfo {
	for i := range videoEncoders {
		for _, name := range videoEncoders[i].Encoders {
			if name == a.cfg.VideoEncoder {
				return &videoEncoders[i]
			}
		}
	}
	return nil
}

// videoEncoderLabel 编码器项的当前值显示。
func (a *App) videoEncoderLabel() string {
	if a.cfg.VideoEncoder == "" {
		return "复制流（默认）"
	}
	for _, e := range videoEncoders {
		for _, name := range e.Encoders {
			if name == a.cfg.VideoEncoder {
				return name + "（" + e.Label + "）"
			}
		}
	}
	return a.cfg.VideoEncoder
}

// editVideoDecoder 解码器（可以为空）。
func (a *App) editVideoDecoder() {
	a.showOptionDialog("解码器（可以为空）", videoDecoderNames, func(i int) {
		if videoDecoderNames[i] == "空" {
			a.cfg.VideoDecoder = ""
		} else {
			a.cfg.VideoDecoder = videoDecoderNames[i]
		}
		a.refreshVideoPage()
	})
}

// videoDecoderLabel 解码器项的当前值显示。
func (a *App) videoDecoderLabel() string {
	if a.cfg.VideoDecoder == "" {
		return "空"
	}
	return a.cfg.VideoDecoder
}

// editVideoPreset 预设（视编码器而定）。
func (a *App) editVideoPreset() {
	enc := a.currentVideoEncoder()
	if enc == nil || len(enc.Presets) == 0 {
		a.showMessageDialog("预设（视编码器而定）", "当前编码器不提供 preset 选项（视编码器而定）。")
		return
	}
	a.showOptionDialog("预设（视编码器而定）", enc.Presets, func(i int) {
		a.cfg.VideoPreset = enc.Presets[i]
		a.refreshVideoPage()
	})
}

// videoPresetLabel 预设项的当前值显示。
func (a *App) videoPresetLabel() string {
	if a.cfg.VideoPreset == "" {
		return "—"
	}
	return a.cfg.VideoPreset
}

// ---------- 质量 ----------

// editQualityMode 控制方式（中文 + 缩写）。
func (a *App) editQualityMode() {
	a.showOptionDialog("控制方式", qualityModes, func(i int) {
		a.cfg.QualityMode = qualityModes[i]
		a.refreshVideoPage()
	})
}

// qualityModeLabel 控制方式项的当前值显示。
func (a *App) qualityModeLabel() string {
	if a.cfg.QualityMode == "" {
		return "恒定质量 CRF"
	}
	return a.cfg.QualityMode
}

// videoEncoderHasQuality 报告当前编码器是否支持设置质量值。
func (a *App) videoEncoderHasQuality() bool {
	enc := a.currentVideoEncoder()
	return enc != nil && enc.HasQuality
}

// editQualityValue 质量值：编码器不支持时显示「编码器不支持」。
func (a *App) editQualityValue() {
	if !a.videoEncoderHasQuality() {
		a.showMessageDialog("质量", "当前编码器不支持设置质量值。")
		return
	}
	a.showInputDialog("质量值（如 CRF 0-51）", a.cfg.QualityValue, func(text string) {
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
		return "编码器不支持"
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
	a.showInputDialog("宽度（像素）", a.cfg.VideoWidth, func(w string) {
		a.cfg.VideoWidth = strings.TrimSpace(w)
		a.showInputDialog("高度（像素）", a.cfg.VideoHeight, func(h string) {
			a.cfg.VideoHeight = strings.TrimSpace(h)
			a.refreshVideoPage()
		})
	})
}

// resolutionLabel 分辨率项的当前值显示。
func (a *App) resolutionLabel() string {
	if a.cfg.VideoWidth == "" && a.cfg.VideoHeight == "" {
		return "原始"
	}
	return fmt.Sprintf("%s×%s", a.cfg.VideoWidth, a.cfg.VideoHeight)
}

// editFPS 帧率输入。
func (a *App) editFPS() {
	a.showInputDialog("帧率（如 30、29.97、30000/1001）", a.cfg.VideoFPS, func(text string) {
		a.cfg.VideoFPS = strings.TrimSpace(text)
		a.refreshVideoPage()
	})
}

// fpsLabel 帧率项的当前值显示。
func (a *App) fpsLabel() string {
	if a.cfg.VideoFPS == "" {
		return "原始"
	}
	return a.cfg.VideoFPS
}
