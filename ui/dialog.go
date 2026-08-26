package ui

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// ---------- 对话框核心 ----------

// showDialog 在根页面上覆盖显示一个模态对话框。
// focusables 是对话框内可聚焦组件（Tab/Shift-Tab 循环），initial 为初始焦点。
func (a *App) showDialog(content tview.Primitive, focusables []tview.Primitive, initial tview.Primitive) {
	a.rootPages.RemovePage("dialog")
	a.rootPages.AddPage("dialog", content, true, true)
	a.dialogOpen = true
	a.dialogFocusables = focusables
	a.dialogFocusIndex = 0
	for i, p := range focusables {
		if p == initial {
			a.dialogFocusIndex = i
			break
		}
	}
	a.tviewApp.SetFocus(initial)
}

// dialogFocusNext / dialogFocusPrev 在对话框内循环移动焦点（Tab / Shift-Tab）。
func (a *App) dialogFocusNext() {
	n := len(a.dialogFocusables)
	if n == 0 {
		return
	}
	a.dialogFocusIndex = (a.dialogFocusIndex + 1) % n
	a.tviewApp.SetFocus(a.dialogFocusables[a.dialogFocusIndex])
}

func (a *App) dialogFocusPrev() {
	n := len(a.dialogFocusables)
	if n == 0 {
		return
	}
	a.dialogFocusIndex = (a.dialogFocusIndex - 1 + n) % n
	a.tviewApp.SetFocus(a.dialogFocusables[a.dialogFocusIndex])
}

// closeDialog 关闭当前对话框，并把焦点还给当前标签页内容。
func (a *App) closeDialog() {
	if !a.dialogOpen {
		return
	}
	a.dialogOpen = false
	a.rootPages.RemovePage("dialog")
	a.tviewApp.SetFocus(a.pages.GetPage(tabNames[a.current]))
}

// buildDialogBox 构建居中显示的对话框：
//
//	┌┤ 标题 ├──────────────┐
//	│  (body)              │
//	│  [按钮1]  [按钮2]     │
//	└──────────────────────┘
//
// 返回已居中的 Grid（左右边框由 Box 绘制，保证对齐）。
func (a *App) buildDialogBox(title string, body tview.Primitive, width, height int, buttons ...*tview.Button) *tview.Grid {
	btnRow := tview.NewFlex()
	btnRow.AddItem(nil, 0, 1, false)
	for i, b := range buttons {
		btnRow.AddItem(b, 0, 1, false)
		if i < len(buttons)-1 {
			btnRow.AddItem(nil, 2, 0, false)
		}
	}
	btnRow.AddItem(nil, 0, 1, false)

	box := tview.NewFlex().SetDirection(tview.FlexRow)
	box.AddItem(body, 0, 1, true)
	box.AddItem(nil, 1, 0, false)
	box.AddItem(btnRow, 1, 0, false)
	box.AddItem(nil, 1, 0, false)
	box.SetBorder(true)
	box.SetTitle("┤ " + title + " ├")
	box.SetTitleAlign(tview.AlignLeft) // 标题紧贴左上角：┌┤ 标题 ├──

	// 用权重为 0 的外围行列把对话框居中。
	grid := tview.NewGrid()
	grid.SetColumns(0, width, 0)
	grid.SetRows(0, height, 0)
	grid.AddItem(box, 1, 1, 1, 1, 0, 0, true)
	grid.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	return grid
}

// showOptionDialog 显示选项选择对话框；confirm(index) 在确定后调用。
func (a *App) showOptionDialog(title string, options []string, confirm func(int)) {
	list := tview.NewList()
	for i, opt := range options {
		opt, i := opt, i
		list.AddItem(opt, "", 0, func() {
			a.closeDialog()
			confirm(i)
		})
	}

	ok := tview.NewButton("[确定]")
	cancel := tview.NewButton("[取消]")
	ok.SetSelectedFunc(func() {
		idx := list.GetCurrentItem()
		a.closeDialog()
		confirm(idx)
	})
	cancel.SetSelectedFunc(func() { a.closeDialog() })

	height := len(options) + 6
	if height < 8 {
		height = 8
	}
	a.showDialog(a.buildDialogBox(title, list, 36, height, ok, cancel), []tview.Primitive{list, ok, cancel}, list)
}

// showInputDialog 显示输入对话框；confirm(text) 在确定后调用。
func (a *App) showInputDialog(title, initial string, confirm func(string)) {
	input := tview.NewInputField()
	input.SetText(initial)
	input.SetFieldWidth(34)
	input.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			a.closeDialog()
			confirm(input.GetText())
		}
	})

	ok := tview.NewButton("[确定]")
	cancel := tview.NewButton("[取消]")
	ok.SetSelectedFunc(func() {
		a.closeDialog()
		confirm(input.GetText())
	})
	cancel.SetSelectedFunc(func() { a.closeDialog() })

	a.showDialog(a.buildDialogBox(title, input, 50, 8, ok, cancel), []tview.Primitive{input, ok, cancel}, input)
}

// showMessageDialog 显示提示对话框（单一 [确定] 按钮）。
func (a *App) showMessageDialog(title, text string) {
	tv := tview.NewTextView()
	tv.SetText(text)
	tv.SetTextAlign(tview.AlignLeft)
	tv.SetDynamicColors(true)
	tv.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	tv.SetTextStyle(tcell.StyleDefault.
		Foreground(tview.Styles.PrimaryTextColor).
		Background(tview.Styles.PrimitiveBackgroundColor))

	ok := tview.NewButton("[确定]")
	ok.SetSelectedFunc(func() { a.closeDialog() })

	a.showDialog(a.buildDialogBox(title, tv, 60, 8, ok), []tview.Primitive{ok}, ok)
}

// ---------- 文件页操作 ----------

// onAddFile 处理「添加文件」：提供 输入路径 与 选择文件（系统文件选择器）两种方式。
func (a *App) onAddFile() {
	a.showOptionDialog("添加文件", []string{"输入路径", "选择文件"}, func(choice int) {
		switch choice {
		case 0: // 输入路径
			a.showInputDialog("输入路径", "", func(text string) {
				text = strings.TrimSpace(text)
				if text != "" {
					a.addFilePath(text)
				}
			})
		case 1: // 系统文件选择器
			a.pickFileFromSystem()
		}
	})
}

// pickFileFromSystem 调用系统文件选择器添加文件；无可用的选择器时给出提示。
func (a *App) pickFileFromSystem() {
	var path string
	var err error
	a.tviewApp.Suspend(func() {
		path, err = runSystemFilePicker()
	})
	if err != nil {
		a.showMessageDialog("提示", "无法调用系统文件选择器："+err.Error()+"\n请改用「输入路径」方式添加文件。")
		return
	}
	if path != "" {
		a.addFilePath(path)
	}
}

// addFilePath 把文件路径加入文件列表。
func (a *App) addFilePath(path string) {
	a.filesList.AddItem(filepath.Base(path), path, 0, nil)
	a.tviewApp.SetFocus(a.filesList)
}

// onSetContainer 处理「设置输出容器」。
func (a *App) onSetContainer() {
	containers := []string{"mp4", "mkv", "avi", "mov", "webm", "flv", "ts", "wmv"}
	a.showOptionDialog("设置输出容器", containers, func(i int) {
		a.outputContainer = containers[i]
		a.updateFileBar()
	})
}

// updateFileBar 刷新底部操作栏右侧的容器名显示。
func (a *App) updateFileBar() {
	a.fileContainerLabel.SetText(fmt.Sprintf("输出容器：%s", a.outputContainer))
}

// ---------- 系统文件选择器 ----------

// runSystemFilePicker 调用系统文件选择器，返回选中的文件路径。
// 用户取消时返回空字符串；找不到可用的选择器时返回错误。
// Linux: zenity / qarma / kdialog / Xdialog；Windows: PowerShell OpenFileDialog。
func runSystemFilePicker() (string, error) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		ps := `Add-Type -AssemblyName System.Windows.Forms; $f = New-Object System.Windows.Forms.OpenFileDialog; $f.Filter = '所有文件 (*.*)|*.*'; if ($f.ShowDialog() -eq [System.Windows.Forms.DialogResult]::OK) { $f.FileName }`
		cmd = exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", ps)
	default:
		for _, tool := range []struct {
			name string
			args []string
		}{
			{"zenity", []string{"--file-selection", "--title=选择文件"}},
			{"qarma", []string{"--file-selection", "--title=选择文件"}},
			{"kdialog", []string{"--getopenfilename", ".", "所有文件 (*)"}},
			{"Xdialog", []string{"--fselect", ".", "20", "60"}},
		} {
			if p, err := exec.LookPath(tool.name); err == nil {
				cmd = exec.Command(p, tool.args...)
				break
			}
		}
	}
	if cmd == nil {
		return "", errors.New("未找到可用的系统文件选择器（zenity/kdialog 等）")
	}
	out, err := cmd.Output()
	if err != nil {
		// 非零退出（如用户取消）视为取消。
		return "", nil
	}
	return strings.TrimSpace(string(out)), nil
}
