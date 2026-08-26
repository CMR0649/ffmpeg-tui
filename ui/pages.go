package ui

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// buildPage 根据标签序号返回对应的内容页（与 tabNames 顺序一致）。
// 序号 0（文件页）由 App.buildFilesPage 构建，这里处理其余页面。
func (a *App) buildPage(index int) tview.Primitive {
	switch index {
	case 1:
		return a.buildVideoPage()
	case 2:
		return a.buildAudioPage()
	case 3:
		return a.buildTasksPage()
	default:
		return a.buildSettingsPage()
	}
}

func (a *App) buildFilesPage() tview.Primitive {
	a.filesList = tview.NewList()

	// 输出目录（选项）：默认与输入文件相同
	a.outputDirBtn = tview.NewButton("")
	a.updateOutputDirButton()
	a.outputDirBtn.SetSelectedFunc(func() { a.editOutputDir() })

	addBtn := tview.NewButton("[添加文件]")
	addBtn.SetSelectedFunc(func() { a.onAddFile() })
	setBtn := tview.NewButton("[设置输出容器]")
	setBtn.SetSelectedFunc(func() { a.onSetContainer() })

	// 文件页 Tab 焦点循环：文件列表 → 输出目录 → [添加文件] → [设置输出容器]
	a.fileBarFocusables = []tview.Primitive{a.filesList, a.outputDirBtn, addBtn, setBtn}
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
	page.SetTitle(" 文件 — 输入文件 ")
	page.AddItem(a.outputDirBtn, 1, 0, false) // 输出目录行
	page.AddItem(a.filesList, 0, 1, true)
	page.AddItem(bar, 1, 0, false)
	return page
}

// updateOutputDirButton 刷新输出目录按钮文本。
func (a *App) updateOutputDirButton() {
	if a.outputDirBtn == nil {
		return
	}
	label := a.cfg.OutputDir
	if label == "" || label == "$file" {
		label = "与输入文件相同"
	}
	a.outputDirBtn.SetLabel("输出目录：" + label)
}

// editOutputDir 编辑输出目录：空或 $file 表示与输入文件相同；
// 输入文件在不同目录时分别输出到与之对应的目录。
func (a *App) editOutputDir() {
	a.showInputDialog("输出目录（$file = 输入文件所在目录）", a.cfg.OutputDir, func(text string) {
		a.cfg.OutputDir = strings.TrimSpace(text)
		a.updateOutputDirButton()
	})
}
