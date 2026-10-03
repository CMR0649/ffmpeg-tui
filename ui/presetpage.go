package ui

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// buildPresetsPage 构建「预设」页：预设列表 + 底部操作栏
// （[保存预设] [添加自定义命令] [打开预设文件夹]）
func (a *App) buildPresetsPage() tview.Primitive {
	a.presetList = tview.NewList()
	a.selectOnSecondClick(a.presetList)

	saveBtn := tview.NewButton(tview.Escape(a.s.SavePreset))
	saveBtn.SetSelectedFunc(func() { a.showSavePresetDialog() })
	cmdBtn := tview.NewButton(tview.Escape(a.s.AddCustomCmd))
	cmdBtn.SetSelectedFunc(func() { a.showAddCustomCommandDialog() })
	openBtn := tview.NewButton(tview.Escape(a.s.OpenPresetDir))
	openBtn.SetSelectedFunc(func() { a.openPresetDir() })

	a.presetBarButtons = []tview.Primitive{saveBtn, cmdBtn, openBtn}
	a.presetBarFocusables = []tview.Primitive{a.presetList, saveBtn, cmdBtn, openBtn}

	bar := tview.NewFlex()
	bar.AddItem(saveBtn, 0, 1, false)
	bar.AddItem(nil, 2, 0, false)
	bar.AddItem(cmdBtn, 0, 1, false)
	bar.AddItem(nil, 2, 0, false)
	bar.AddItem(openBtn, 0, 1, false)
	bar.AddItem(nil, 0, 1, false)

	page := tview.NewFlex().SetDirection(tview.FlexRow)
	page.SetBorder(true)
	page.SetTitle(a.s.PresetsTitle)
	page.AddItem(a.presetList, 0, 1, true)
	page.AddItem(bar, 1, 0, false)

	a.refreshPresets()
	return page
}

// refreshPresets 从所有预设文件夹读取并重建预设列表（按显示名排序）。
func (a *App) refreshPresets() {
	a.presetList.Clear()
	a.presetEntries = nil
	entries := listPresetEntries()
	a.presetEntries = entries
	for _, e := range entries {
		e := e
		a.presetList.AddItem(tview.Escape(e.Label), tview.Escape(a.presetSummary(e.Dir, e.Name)), 0, func() { a.loadPreset(e) })
	}
}

// presetSummary 预设的简要描述（自定义命令 / 编码器 + 容器）。
func (a *App) presetSummary(dir, name string) string {
	p, err := loadPresetFileAt(dir, name)
	if err != nil {
		return ""
	}
	if p.CustomCommand != "" {
		return p.CustomCommand
	}
	var parts []string
	if p.VideoEncoder != "" {
		parts = append(parts, p.VideoEncoder)
	}
	if p.AudioEncoder != "" {
		parts = append(parts, p.AudioEncoder)
	}
	if p.OutputContainer != "" {
		parts = append(parts, p.OutputContainer)
	}
	return strings.Join(parts, " · ")
}

// loadPreset 应用选中的预设，并提示已加载。
func (a *App) loadPreset(e presetEntry) {
	p, err := loadPresetFileAt(e.Dir, e.Name)
	if err != nil {
		a.showMessageDialog(a.s.Hint, fmt.Sprintf(a.s.LoadPresetFailed, err.Error()))
		return
	}
	p.applyTo(a.cfg)
	a.customCmd = p.CustomCommand
	a.refreshAllUI()
	a.showMessageDialog(a.s.Hint, fmt.Sprintf(a.s.LoadedPreset, e.Name))
}

// deleteSelectedPreset 删除选中的预设。
func (a *App) deleteSelectedPreset() {
	if a.presetList.GetItemCount() == 0 {
		return
	}
	idx := a.presetList.GetCurrentItem()
	if idx < 0 || idx >= len(a.presetEntries) {
		return
	}
	e := a.presetEntries[idx]
	if err := deletePresetAt(e.Dir, e.Name); err != nil {
		a.showMessageDialog(a.s.Hint, fmt.Sprintf(a.s.DeletePresetFailed, err.Error()))
		return
	}
	a.refreshPresets()
}

// searchPreset 打开预设搜索（按 "/" 直接进入搜索框 + 实时过滤列表）。
func (a *App) searchPreset() {
	entries := listPresetEntries()
	if len(entries) == 0 {
		return
	}
	labels := make([]string, len(entries))
	for i, e := range entries {
		labels[i] = tview.Escape(e.Label)
	}
	a.optDialog = &optionDialog{
		title:   a.s.SearchPreset,
		options: labels,
		confirm: func(i int) { a.loadPreset(entries[i]) },
	}
	a.startOptionSearch()
}

// showSavePresetDialog 保存预设弹窗。
func (a *App) showSavePresetDialog() {
	label := tview.NewTextView()
	label.SetText(a.s.SavePresetName + "：")
	label.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	label.SetTextStyle(tcell.StyleDefault.
		Foreground(tview.Styles.PrimaryTextColor).
		Background(tview.Styles.PrimitiveBackgroundColor))

	input := tview.NewInputField()
	input.SetFieldWidth(0)
	input.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			a.savePresetFromInput(input)
		}
	})

	row := tview.NewFlex()
	row.AddItem(label, 0, 1, false)
	row.AddItem(input, 0, 4, false)

	body := tview.NewFlex().SetDirection(tview.FlexRow)
	body.AddItem(nil, 1, 0, false)
	body.AddItem(row, 1, 0, false)

	cancel := tview.NewButton(tview.Escape("[" + a.s.Cancel + "]"))
	ok := tview.NewButton(tview.Escape("[" + a.s.OK + "]"))
	cancel.SetSelectedFunc(func() { a.closeDialog() })
	ok.SetSelectedFunc(func() { a.savePresetFromInput(input) })

	btnRow := tview.NewFlex()
	btnRow.AddItem(nil, 0, 1, false) // 把按钮推到右侧
	btnRow.AddItem(cancel, 0, 1, false)
	btnRow.AddItem(nil, 2, 0, false)
	btnRow.AddItem(ok, 0, 1, false)

	box := tview.NewFlex().SetDirection(tview.FlexRow)
	box.AddItem(body, 0, 1, true)
	box.AddItem(nil, 1, 0, false)
	box.AddItem(btnRow, 1, 0, false)
	box.AddItem(nil, 1, 0, false)
	box.SetBorder(true)
	box.SetTitle("┤ " + a.s.SavePresetTitle + " ├")
	box.SetTitleAlign(tview.AlignLeft)

	grid := tview.NewGrid()
	grid.SetColumns(0, 54, 0)
	grid.SetRows(0, 7, 0)
	grid.AddItem(box, 1, 1, 1, 1, 0, 0, true)
	grid.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)

	a.dialogButtons = []tview.Primitive{cancel, ok}
	a.showDialog(grid, []tview.Primitive{input, cancel, ok}, input)
}

// savePresetFromInput 读取名称并保存预设。
func (a *App) savePresetFromInput(input *tview.InputField) {
	name := strings.TrimSpace(input.GetText())
	a.closeDialog()
	if name == "" {
		return
	}
	p := &Preset{}
	p.fromConfig(a.cfg)
	if err := savePreset(name, p); err != nil {
		a.showMessageDialog(a.s.Hint, fmt.Sprintf(a.s.SavePresetFailed, err.Error()))
		return
	}
	a.refreshPresets()
}

// showAddCustomCommandDialog 添加自定义命令：先输入命令，再输入预设名称。
func (a *App) showAddCustomCommandDialog() {
	a.showInputDialog(a.s.CustomCmdInput, "", func(cmd string) {
		cmd = strings.TrimSpace(cmd)
		if cmd == "" {
			return
		}
		a.showInputDialog(a.s.SavePresetName, "", func(name string) {
			name = strings.TrimSpace(name)
			if name == "" {
				return
			}
			p := &Preset{CustomCommand: cmd}
			if err := savePreset(name, p); err != nil {
				a.showMessageDialog(a.s.Hint, fmt.Sprintf(a.s.SavePresetFailed, err.Error()))
				return
			}
			a.refreshPresets()
		})
	})
}

// openPresetDir 用系统默认文件管理器打开预设文件夹。
func (a *App) openPresetDir() {
	_ = openURL(presetDir())
}
