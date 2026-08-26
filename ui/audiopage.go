package ui

import (
	"fmt"
	"strings"

	"github.com/rivo/tview"
)

// buildAudioPage 构建「音频」页：编码器 / 比特率 / 采样率 / 位深度。
// 编码器列表从 ffmpeg-encoders.txt（或 ffmpeg 命令）动态加载。
func (a *App) buildAudioPage() tview.Primitive {
	list := tview.NewList()
	list.SetBorder(true)
	list.SetTitle(" 音频 — 编码选项 ")
	a.audioList = list
	a.refreshAudioPage()
	return list
}

// refreshAudioPage 按当前配置重建音频页选项列表。
func (a *App) refreshAudioPage() {
	l := a.audioList
	l.Clear()
	l.AddItem("编码器", a.audioEncoderLabel(), 0, func() { a.editAudioEncoder() })
	l.AddItem("比特率", a.audioBitrateLabel(), 0, func() { a.editAudioBitrate() })
	l.AddItem("采样率", a.sampleRateLabel(), 0, func() { a.editSampleRate() })
	l.AddItem("位深度", a.bitDepthLabel(), 0, func() { a.editBitDepth() })
}

// editAudioEncoder 编码器（默认复制流），列表动态加载。
func (a *App) editAudioEncoder() {
	loadCodecLists()
	labels := make([]string, 0, len(audioEncoders)+1)
	labels = append(labels, "复制流（默认）")
	labels = append(labels, audioEncoders...)
	a.showOptionDialog("编码器（默认复制流）", labels, func(i int) {
		if i == 0 {
			a.cfg.AudioEncoder = ""
			a.refreshAudioPage()
			return
		}
		a.cfg.AudioEncoder = audioEncoders[i-1]
		if strings.HasPrefix(a.cfg.AudioEncoder, "pcm_") {
			a.cfg.BitDepth = "16" // WAV/PCM 默认 16-bit，与编码器名称同步
		}
		a.refreshAudioPage()
	})
}

// audioEncoderLabel 编码器项的当前值显示。
func (a *App) audioEncoderLabel() string {
	if a.cfg.AudioEncoder == "" {
		return "复制流（默认）"
	}
	if strings.HasPrefix(a.cfg.AudioEncoder, "pcm_") && a.cfg.BitDepth != "" {
		return a.cfg.AudioEncoder + "（WAV " + a.cfg.BitDepth + "-bit）"
	}
	return a.cfg.AudioEncoder
}

// audioIsWav 报告当前音频编码器是否为 WAV/PCM（位深度与编码器名称同步）。
func (a *App) audioIsWav() bool {
	return strings.HasPrefix(a.cfg.AudioEncoder, "pcm_")
}

// editAudioBitrate 比特率输入（kbps）。
func (a *App) editAudioBitrate() {
	a.showInputDialog("比特率（kbps）", a.cfg.AudioBitrate, func(text string) {
		a.cfg.AudioBitrate = strings.TrimSpace(text)
		a.refreshAudioPage()
	})
}

// audioBitrateLabel 比特率项的当前值显示。
func (a *App) audioBitrateLabel() string {
	if a.cfg.AudioBitrate == "" {
		return "—"
	}
	return a.cfg.AudioBitrate + " kbps"
}

// editSampleRate 采样率选择。
func (a *App) editSampleRate() {
	a.showOptionDialog("采样率（Hz）", sampleRates, func(i int) {
		if sampleRates[i] == "原始" {
			a.cfg.SampleRate = ""
		} else {
			a.cfg.SampleRate = sampleRates[i]
		}
		a.refreshAudioPage()
	})
}

// sampleRateLabel 采样率项的当前值显示。
func (a *App) sampleRateLabel() string {
	if a.cfg.SampleRate == "" {
		return "原始"
	}
	return a.cfg.SampleRate + " Hz"
}

// editBitDepth 位深度：仅 WAV/PCM 编码器有效，且与编码器名称同步。
func (a *App) editBitDepth() {
	if !a.audioIsWav() {
		a.showMessageDialog("位深度（与编码器名称同步）", "位深度仅对 WAV/PCM 编码器有效（与编码器名称同步）。")
		return
	}
	labels := make([]string, len(bitDepths))
	for i, b := range bitDepths {
		labels[i] = b.Label
	}
	a.showOptionDialog("位深度（与编码器名称同步）", labels, func(i int) {
		a.cfg.BitDepth = strings.TrimSuffix(bitDepths[i].Label, "-bit")
		a.cfg.AudioEncoder = bitDepths[i].Name // 与编码器名称同步
		a.refreshAudioPage()
	})
}

// bitDepthLabel 位深度项的当前值显示。
func (a *App) bitDepthLabel() string {
	if !a.audioIsWav() {
		return "编码器不支持"
	}
	if a.cfg.BitDepth == "" {
		return "16-bit"
	}
	return fmt.Sprintf("%s-bit", a.cfg.BitDepth)
}
