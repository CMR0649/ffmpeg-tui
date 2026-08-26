package ui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// buildPage 根据标签序号返回对应的内容页（与 tabNames 顺序一致）。
// 序号 0（文件页）由 App.buildFilesPage 构建，这里处理其余页面。
func buildPage(index int) tview.Primitive {
	switch index {
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

// buildFilesPage 构建「文件」页：
//
//	文件列表（初始为空）+ 底部操作栏（[添加文件] [设置输出容器] …… 输出容器：xxx）。
//
// 底部操作栏位于页面边框之内、按键说明之上。
func (a *App) buildFilesPage() tview.Primitive {
	a.filesList = tview.NewList()

	// 底部操作栏：[添加文件] [设置输出容器] …… 输出容器：xxx
	addBtn := tview.NewButton("[添加文件]")
	addBtn.SetSelectedFunc(func() { a.onAddFile() })
	setBtn := tview.NewButton("[设置输出容器]")
	setBtn.SetSelectedFunc(func() { a.onSetContainer() })

	// 文件页 Tab 焦点循环：文件列表 → [添加文件] → [设置输出容器]
	a.fileBarFocusables = []tview.Primitive{a.filesList, addBtn, setBtn}
	// 底部横向按钮组（左右方向键切换）
	a.fileBarButtons = []tview.Primitive{addBtn, setBtn}

	a.fileContainerLabel = tview.NewTextView()
	a.fileContainerLabel.SetTextAlign(tview.AlignRight)
	a.fileContainerLabel.SetText(fmt.Sprintf("输出容器：%s", a.outputContainer))
	a.fileContainerLabel.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	a.fileContainerLabel.SetTextStyle(tcell.StyleDefault.
		Foreground(tview.Styles.PrimaryTextColor).
		Background(tview.Styles.PrimitiveBackgroundColor))

	bar := tview.NewFlex()
	bar.AddItem(addBtn, 0, 1, false)
	bar.AddItem(nil, 2, 0, false)
	bar.AddItem(setBtn, 0, 1, false)
	bar.AddItem(nil, 0, 1, false) // 弹性空白，把容器名推到右侧
	bar.AddItem(a.fileContainerLabel, 0, 1, false)

	page := tview.NewFlex().SetDirection(tview.FlexRow)
	page.SetBorder(true)
	page.SetTitle(" 文件 — 输入文件（示例数据） ")
	page.AddItem(a.filesList, 0, 1, true)
	page.AddItem(bar, 1, 0, false)
	return page
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
