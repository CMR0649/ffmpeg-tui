package ui

import (
	"fmt"
	"strings"

	"github.com/rivo/tview"
)

// buildAudioPage 构建「音频」页：编码器 / 比特率 / 采样率 / 位深度
// 编码器列表从 ffmpeg 命令动态加载
func (a *App) buildAudioPage() tview.Primitive {
	list := tview.NewList()
	list.SetBorder(true)
	list.SetTitle(a.s.AudioTitle)
	a.audioList = list
	a.refreshAudioPage()
	return list
}

// refreshAudioPage 按当前配置与语言重建音频页选项列表
func (a *App) refreshAudioPage() {
	l := a.audioList
	l.Clear()
	l.AddItem(a.s.AudioEncoder, a.audioEncoderLabel(), 0, func() { a.editAudioEncoder() })
	l.AddItem(a.s.AudioBitrate, a.audioBitrateLabel(), 0, func() { a.editAudioBitrate() })
	l.AddItem(a.s.SampleRate, a.sampleRateLabel(), 0, func() { a.editSampleRate() })
	l.AddItem(a.s.BitDepth, a.bitDepthLabel(), 0, func() { a.editBitDepth() })
}

// editAudioEncoder 编码器（默认复制流），列表动态加载
func (a *App) editAudioEncoder() {
	loadCodecLists(a.ffmpegBin())
	labels := make([]string, 0, len(audioEncoders)+1)
	labels = append(labels, a.copyStreamLabel())
	labels = append(labels, audioEncoders...)
	a.showOptionDialog(a.s.AudioEncoder, labels, func(i int) {
		if i == 0 {
			a.cfg.AudioEncoder = ""
			a.refreshAudioPage()
			return
		}
		a.cfg.AudioEncoder = audioEncoders[i-1]
		if strings.HasPrefix(a.cfg.AudioEncoder, "pcm_") && a.cfg.BitDepth == "" {
			a.cfg.BitDepth = "16" // WAV/PCM 默认 16-bit，与编码器名称同步
		}
		a.refreshAudioPage()
	})
}

// audioEncoderLabel 编码器项的当前值显示
func (a *App) audioEncoderLabel() string {
	if a.cfg.AudioEncoder == "" {
		return a.copyStreamLabel()
	}
	if strings.HasPrefix(a.cfg.AudioEncoder, "pcm_") && a.cfg.BitDepth != "" {
		return a.cfg.AudioEncoder + "（WAV " + a.cfg.BitDepth + "-bit）"
	}
	return a.cfg.AudioEncoder
}

// audioIsWav 报告当前音频编码器是否为 WAV/PCM（位深度与编码器名称同步）
func (a *App) audioIsWav() bool {
	return strings.HasPrefix(a.cfg.AudioEncoder, "pcm_")
}

// editAudioBitrate 比特率输入（kbps）
func (a *App) editAudioBitrate() {
	a.showInputDialog(a.s.AudioBitrate+"（kbps）", a.cfg.AudioBitrate, func(text string) {
		a.cfg.AudioBitrate = strings.TrimSpace(text)
		a.refreshAudioPage()
	})
}

// audioBitrateLabel 比特率项的当前值显示
func (a *App) audioBitrateLabel() string {
	if a.cfg.AudioBitrate == "" {
		return "—"
	}
	return a.cfg.AudioBitrate + " kbps"
}

// editSampleRate 采样率选择
func (a *App) editSampleRate() {
	labels := make([]string, 0, len(sampleRates)+1)
	labels = append(labels, a.originalLabel())
	labels = append(labels, sampleRates...)
	a.showOptionDialog(a.s.SampleRate, labels, func(i int) {
		if i == 0 {
			a.cfg.SampleRate = ""
		} else {
			a.cfg.SampleRate = sampleRates[i-1]
		}
		a.refreshAudioPage()
	})
}

// sampleRateLabel 采样率项的当前值显示
func (a *App) sampleRateLabel() string {
	if a.cfg.SampleRate == "" {
		return a.originalLabel()
	}
	return a.cfg.SampleRate + " Hz"
}

// editBitDepth 位深度：对所有音频编码器可用（通过 -sample_fmt s[位深]），
// 例如 `ffmpeg -i in.mp3 -sample_fmt s16 out.flac`；
// WAV/PCM 编码器选择位深时同时同步编码器名称（16→pcm_s16le 等）
func (a *App) editBitDepth() {
	labels := make([]string, len(bitDepths))
	for i, b := range bitDepths {
		labels[i] = b.Label
	}
	a.showOptionDialog(a.s.BitDepth, labels, func(i int) {
		a.cfg.BitDepth = strings.TrimSuffix(bitDepths[i].Label, "-bit")
		if a.audioIsWav() {
			a.cfg.AudioEncoder = bitDepths[i].Name // WAV/PCM：与编码器名称同步
		}
		a.refreshAudioPage()
	})
}

// bitDepthLabel 位深度项的当前值显示
func (a *App) bitDepthLabel() string {
	if a.cfg.BitDepth == "" {
		return a.s.BitDepthNotSet
	}
	return fmt.Sprintf("%s-bit", a.cfg.BitDepth)
}
