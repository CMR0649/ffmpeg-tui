package ui

import (
	"fmt"
	"strings"

	"github.com/rivo/tview"
)

// outputNamingOptions 输出选项（单选）——按当前语言生成
func (a *App) outputNamingOptions() []string {
	return []string{a.s.NamingTimestamp, a.s.NamingSuffix, a.s.NamingNone}
}

// buildSettingsPage 构建「设置」页：输出选项 / FFmpeg 路径 / 语言 /
// 配置管理 / 关于（最后一个选项）
func (a *App) buildSettingsPage() tview.Primitive {
	list := tview.NewList()
	list.SetBorder(true)
	list.SetTitle(a.s.SettingsTitle)
	a.settingsList = list
	a.refreshSettingsPage()
	return list
}

// refreshSettingsPage 按当前配置与语言重建设置页选项列表
func (a *App) refreshSettingsPage() {
	l := a.settingsList
	l.Clear()
	l.AddItem(a.s.OutputOption, a.outputNamingLabel(), 0, func() { a.editOutputNaming() })
	l.AddItem(a.s.Suffix, a.suffixLabel(), 0, func() { a.editSuffix() })
	l.AddItem(a.s.FFmpegPath, a.ffmpegPathLabel(), 0, func() { a.editFFmpegPath() })
	l.AddItem(a.s.Language, a.languageLabel(), 0, func() { a.editLanguage() })
	l.AddItem(a.s.ExportCfg, "JSON", 0, func() { a.exportConfig() })
	l.AddItem(a.s.LoadCfg, "JSON", 0, func() { a.loadConfig() })
	l.AddItem(a.s.SetDefaultCfg, "", 0, func() { a.saveAsDefault() })
	l.AddItem(a.s.About, "", 0, func() { a.showAboutDialog() }) // 最后一个选项
}

// editOutputNaming 输出选项（添加时间 / 指定后缀 / 不添加后缀）
func (a *App) editOutputNaming() {
	a.showOptionDialog(a.s.OutputOption, a.outputNamingOptions(), func(i int) {
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

// outputNamingLabel 输出选项项的当前值显示
func (a *App) outputNamingLabel() string {
	switch a.cfg.OutputNaming {
	case "suffix":
		return strings.Replace(a.s.NamingSuffix, "%s", a.cfg.Suffix, 1)
	case "none":
		return a.s.NamingNone
	default:
		return a.s.NamingTimestamp
	}
}

// editSuffix 指定后缀输入
func (a *App) editSuffix() {
	a.promptSuffix()
}

// promptSuffix 弹出后缀输入对话框
func (a *App) promptSuffix() {
	a.showInputDialog(a.s.Suffix, a.cfg.Suffix, func(text string) {
		a.cfg.Suffix = strings.TrimSpace(text)
		a.refreshSettingsPage()
	})
}

// suffixLabel 指定后缀项的当前值显示
func (a *App) suffixLabel() string {
	if a.cfg.Suffix == "" {
		return "—"
	}
	return a.cfg.Suffix
}

// editFFmpegPath 指定 FFmpeg 可执行文件路径（留空 = 系统 PATH）
func (a *App) editFFmpegPath() {
	a.showInputDialog(a.s.FFmpegPath, a.cfg.FFmpegPath, func(text string) {
		a.cfg.FFmpegPath = strings.TrimSpace(text)
		// 路径变化：清空编码器/格式缓存并按新路径重新加载
		resetCodecCaches()
		loadCodecLists(a.ffmpegBin())
		loadFormats(a.ffmpegBin())
		a.refreshSettingsPage()
	})
}

// ffmpegPathLabel FFmpeg 路径项的当前值显示
func (a *App) ffmpegPathLabel() string {
	if a.cfg.FFmpegPath == "" {
		return a.s.FFmpegPathDefault
	}
	return a.cfg.FFmpegPath
}

// editLanguage 切换界面语言（中文 / English）
func (a *App) editLanguage() {
	a.showOptionDialog(a.s.Language, []string{a.s.LangZh, a.s.LangEn}, func(i int) {
		if i == 0 {
			a.setLanguage("zh")
		} else {
			a.setLanguage("en")
		}
	})
}

// languageLabel 语言项的当前值显示
func (a *App) languageLabel() string {
	return langStrings(a.lang).LangZh + " / " + langStrings(a.lang).LangEn
}

// refreshAllUI 完整重建界面：选项页/任务页重建（标题随之更新）、
// 文件页标题与按钮/容器标签刷新、标签栏与底部提示刷新。
// 加载配置或切换语言后调用，确保所有选项与文本即时生效。
func (a *App) refreshAllUI() {
	for _, i := range []int{1, 2, 3, 4, 5} {
		a.pages.RemovePage(tabKeys[i])
		a.pages.AddPage(tabKeys[i], a.buildPage(i), true, false)
	}
	a.refreshTasks()
	if a.filesPage != nil {
		a.filesPage.SetTitle(" " + a.s.FilesTitle + " ")
	}
	if a.filesAddBtn != nil {
		a.filesAddBtn.SetLabel(a.s.AddFile)
	}
	if a.filesSetBtn != nil {
		a.filesSetBtn.SetLabel(a.s.SetContainer)
	}
	a.updateOutputDirButton()
	if a.fileContainerLabel != nil {
		a.fileContainerLabel.SetText(fmt.Sprintf(a.s.ContainerLabel, a.cfg.OutputContainer))
	}
	a.renderTabBar()
	a.renderFooter()
}

// setLanguage 切换语言并刷新所有界面文本
func (a *App) setLanguage(lang string) {
	a.cfg.Lang = lang
	a.lang = lang
	a.s = langStrings(lang)
	a.refreshAllUI()
}

// exportConfig 导出当前配置为 JSON 文件
func (a *App) exportConfig() {
	a.showInputDialog(a.s.ExportCfg, "", func(path string) {
		path = strings.TrimSpace(path)
		if path == "" {
			return
		}
		if err := a.cfg.SaveJSON(path); err != nil {
			a.showMessageDialog(a.s.Hint, fmt.Sprintf(a.s.SaveCfgFailed, err.Error()))
			return
		}
		a.showMessageDialog(a.s.Hint, fmt.Sprintf(a.s.Exported, path))
	})
}

// loadConfig 从 JSON 文件加载配置
func (a *App) loadConfig() {
	a.showInputDialog(a.s.LoadCfg, "", func(path string) {
		path = strings.TrimSpace(path)
		if path == "" {
			return
		}
		if err := a.cfg.LoadJSON(path); err != nil {
			a.showMessageDialog(a.s.Hint, fmt.Sprintf(a.s.LoadCfgFailed, err.Error()))
			return
		}
		// 配置可能包含语言设置，重新应用语言并完整重建界面
		a.lang = effectiveLang(a.cfg.Lang)
		a.s = langStrings(a.lang)
		a.refreshAllUI()
		a.showMessageDialog(a.s.Hint, fmt.Sprintf(a.s.Loaded, path))
	})
}

// saveAsDefault 把当前配置保存为默认配置（启动时自动加载）
func (a *App) saveAsDefault() {
	if err := a.cfg.SaveJSON(defaultConfigPath()); err != nil {
		a.showMessageDialog(a.s.Hint, fmt.Sprintf(a.s.SaveDefaultFailed, err.Error()))
		return
	}
	a.showMessageDialog(a.s.Hint, fmt.Sprintf(a.s.SavedDefault, defaultConfigPath()))
}
