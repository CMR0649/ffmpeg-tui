package ui

import "github.com/rivo/tview"

// buildPage 根据标签序号返回对应的内容页（与 tabNames 顺序一致）。
func buildPage(index int) tview.Primitive {
	switch index {
	case 0:
		return buildFilesPage()
	case 1:
		return buildVideoPage()
	case 2:
		return buildAudioPage()
	case 3:
		return buildTasksPage()
	default:
		return buildSettingsPage()
	}
}

// buildFilesPage 构建「文件」页：输入文件列表（beta0.1 为示例数据）。
func buildFilesPage() tview.Primitive {
	list := tview.NewList()
	list.SetBorder(true)
	list.SetTitle(" 文件 — 输入文件（示例数据） ")
	list.AddItem("sample.mp4", "H.264 · 720p · 00:03:24 · 48.2 MB", 0, nil)
	list.AddItem("video.mkv", "HEVC · 1080p · 00:12:08 · 1.2 GB", 0, nil)
	list.AddItem("clip.avi", "MPEG-4 · 480p · 00:00:45 · 21.5 MB", 0, nil)
	list.AddItem("record.wav", "PCM 16bit · 44.1kHz · 00:01:30 · 15.2 MB", 0, nil)
	list.AddItem("—", "beta0.1：仅演示界面，暂不支持真实文件操作", 0, nil)
	return list
}

// buildVideoPage 构建「视频」页：视频编码选项（beta0.1 为示例数据）。
func buildVideoPage() tview.Primitive {
	list := tview.NewList()
	list.SetBorder(true)
	list.SetTitle(" 视频 — 编码选项（示例数据） ")
	list.AddItem("编码器", "当前：libx264", 0, nil)
	list.AddItem("码率", "当前：4000 kbps", 0, nil)
	list.AddItem("分辨率", "当前：1920×1080", 0, nil)
	list.AddItem("帧率", "当前：30 fps", 0, nil)
	list.AddItem("像素格式", "当前：yuv420p", 0, nil)
	list.AddItem("—", "beta0.1：仅演示界面，暂不支持修改选项", 0, nil)
	return list
}

// buildAudioPage 构建「音频」页：音频编码选项（beta0.1 为示例数据）。
func buildAudioPage() tview.Primitive {
	list := tview.NewList()
	list.SetBorder(true)
	list.SetTitle(" 音频 — 编码选项（示例数据） ")
	list.AddItem("编码器", "当前：aac", 0, nil)
	list.AddItem("码率", "当前：192 kbps", 0, nil)
	list.AddItem("采样率", "当前：48000 Hz", 0, nil)
	list.AddItem("声道数", "当前：2（立体声）", 0, nil)
	list.AddItem("音量增益", "当前：0 dB", 0, nil)
	list.AddItem("—", "beta0.1：仅演示界面，暂不支持修改选项", 0, nil)
	return list
}

// buildTasksPage 构建「任务」页：转码队列（beta0.1 为示例数据）。
func buildTasksPage() tview.Primitive {
	list := tview.NewList()
	list.SetBorder(true)
	list.SetTitle(" 任务 — 转码队列（示例数据） ")
	list.AddItem("转码 sample.mp4 → output.mp4", "状态：排队中 · 进度 0%", 0, nil)
	list.AddItem("转码 video.mkv → output.mkv", "状态：等待 · 进度 0%", 0, nil)
	list.AddItem("—", "beta0.1：仅演示界面，暂不支持真实任务", 0, nil)
	return list
}

// buildSettingsPage 构建「设置」页：全局选项（beta0.1 为示例数据）。
func buildSettingsPage() tview.Primitive {
	list := tview.NewList()
	list.SetBorder(true)
	list.SetTitle(" 设置 — 全局选项（示例数据） ")
	list.AddItem("输出目录", "当前：./output", 0, nil)
	list.AddItem("覆盖模式", "当前：询问", 0, nil)
	list.AddItem("线程数", "当前：自动", 0, nil)
	list.AddItem("硬件加速", "当前：关闭", 0, nil)
	list.AddItem("日志级别", "当前：info", 0, nil)
	list.AddItem("—", "beta0.1：仅演示界面，暂不支持修改选项", 0, nil)
	return list
}
