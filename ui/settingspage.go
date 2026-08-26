package ui

import (
	"strings"

	"github.com/rivo/tview"
)

// outputNamingOptions 输出选项（单选）。
var outputNamingOptions = []string{
	"添加时间（默认）",
	"添加指定后缀",
	"不添加后缀",
}

// buildSettingsPage 构建「设置」页：输出选项 / 导出配置 / 加载配置 / 指定默认配置。
func (a *App) buildSettingsPage() tview.Primitive {
	list := tview.NewList()
	list.SetBorder(true)
	list.SetTitle(" 设置 ")
	a.settingsList = list
	a.refreshSettingsPage()
	return list
}

// refreshSettingsPage 按当前配置重建设置页选项列表。
func (a *App) refreshSettingsPage() {
	l := a.settingsList
	l.Clear()
	l.AddItem("输出选项", a.outputNamingLabel(), 0, func() { a.editOutputNaming() })
	l.AddItem("指定后缀", a.suffixLabel(), 0, func() { a.editSuffix() })
	l.AddItem("导出配置（JSON）", "保存当前配置为 JSON 文件", 0, func() { a.exportConfig() })
	l.AddItem("加载配置", "从 JSON 文件加载配置", 0, func() { a.loadConfig() })
	l.AddItem("指定默认配置", "保存当前配置为启动时加载的默认配置", 0, func() { a.saveAsDefault() })
}

// editOutputNaming 输出选项（添加时间 / 指定后缀 / 不添加后缀）。
func (a *App) editOutputNaming() {
	a.showOptionDialog("输出选项", outputNamingOptions, func(i int) {
		switch i {
		case 0:
			a.cfg.OutputNaming = "timestamp"
		case 1:
			a.cfg.OutputNaming = "suffix"
			if a.cfg.Suffix == "" {
				a.promptSuffix()
			}
		case 2:
			a.cfg.OutputNaming = "none"
		}
		a.refreshSettingsPage()
	})
}

// outputNamingLabel 输出选项项的当前值显示。
func (a *App) outputNamingLabel() string {
	switch a.cfg.OutputNaming {
	case "suffix":
		return "添加指定后缀：" + a.cfg.Suffix
	case "none":
		return "不添加后缀"
	default:
		return "添加时间（默认）[文件名]_YYYY-M-D-hhmmss"
	}
}

// editSuffix 指定后缀输入。
func (a *App) editSuffix() {
	a.promptSuffix()
}

// promptSuffix 弹出后缀输入对话框。
func (a *App) promptSuffix() {
	a.showInputDialog("指定后缀", a.cfg.Suffix, func(text string) {
		a.cfg.Suffix = strings.TrimSpace(text)
		a.refreshSettingsPage()
	})
}

// suffixLabel 指定后缀项的当前值显示。
func (a *App) suffixLabel() string {
	if a.cfg.Suffix == "" {
		return "—"
	}
	return a.cfg.Suffix
}

// exportConfig 导出当前配置为 JSON 文件。
func (a *App) exportConfig() {
	a.showInputDialog("导出配置（JSON 文件路径）", "", func(path string) {
		path = strings.TrimSpace(path)
		if path == "" {
			return
		}
		if err := a.cfg.SaveJSON(path); err != nil {
			a.showMessageDialog("导出失败", "无法保存配置文件：\n"+err.Error())
			return
		}
		a.showMessageDialog("导出成功", "配置已导出到：\n"+path)
	})
}

// loadConfig 从 JSON 文件加载配置。
func (a *App) loadConfig() {
	a.showInputDialog("加载配置（JSON 文件路径）", "", func(path string) {
		path = strings.TrimSpace(path)
		if path == "" {
			return
		}
		if err := a.cfg.LoadJSON(path); err != nil {
			a.showMessageDialog("加载失败", "无法加载配置文件：\n"+err.Error())
			return
		}
		a.refreshAllPages()
		a.showMessageDialog("加载成功", "已加载配置：\n"+path)
	})
}

// saveAsDefault 把当前配置保存为默认配置（启动时自动加载）。
func (a *App) saveAsDefault() {
	if err := a.cfg.SaveJSON(defaultConfigPath()); err != nil {
		a.showMessageDialog("保存失败", "无法保存默认配置：\n"+err.Error())
		return
	}
	a.showMessageDialog("已保存", "当前配置已保存为默认配置：\n"+defaultConfigPath())
}
