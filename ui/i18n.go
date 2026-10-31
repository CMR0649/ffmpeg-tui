package ui

import (
	"os"
	"runtime"
	"strings"
)

// Strings 保存界面文本（中英文），用于语言切换
type Strings struct {
	// 标签页
	TabFiles, TabVideo, TabAudio, TabFilters, TabTasks, TabPresets, TabSettings string
	// 底部按键提示（按页面）
	FooterFiles, FooterVideoAudio, FooterFilters, FooterPresets, FooterSettings, FooterTasks string
	// 通用
	OK, Cancel, Hint string
	// 滤镜页
	FiltersTitle, FilterVideoChain, FilterAudioChain     string
	FilterAddVideo, FilterAddAudio                       string
	FilterMoveUp, FilterMoveDown                         string
	FilterEmpty, FilterCount                             string
	FilterCopyWarn, FilterNeedEncode                     string
	FilterParamsTitle, FilterPreview, FilterValueInvalid string
	FilterListFailed, FilterPickHint                     string
	// 文件页
	FilesTitle      string
	OutputDir       string
	OutputDirTitle  string
	OutputDirSame   string
	AddFile         string
	SetContainer    string
	ContainerLabel  string
	InputPath       string
	PickFile        string
	ReadingInfo     string
	NoFFmpeg        string
	ParseInfoFailed string
	// 视频页
	VideoTitle, Encoder, Decoder, Preset, QualityMode, Quality, BitrateBase, BitrateMax, BitrateMin, Resolution, FPS string
	CustomParams                                                                                                     string
	// 质量模式（键 → 显示）
	// 音频页
	AudioTitle, AudioEncoder, AudioBitrate, SampleRate, BitDepth string
	Unspecified                                                  string
	Disable                                                      string
	// 设置页
	SettingsTitle, OutputOption, Suffix, ExportCfg, LoadCfg, SetDefaultCfg, FFmpegPath, Language, About string
	RestoreDefaults                                                                                     string
	RestoreDefaultsConfirm                                                                              string
	ParallelTasks                                                                                       string
	NamingTimestamp, NamingSuffix, NamingNone                                                           string
	FFmpegPathDefault                                                                                   string
	LangZh, LangEn                                                                                      string
	// 任务页
	TasksTitle, TaskAdd, TaskStart, TaskClear      string
	TaskWaiting, TaskRunning, TaskDone, TaskFailed string
	NoFilesToTask                                  string
	TaskShowCmd, CmdLabel, CmdReturnHint           string
	TaskSavedTo, TaskLogSavedTo                    string
	// 任务进度（frame= 行解析）
	ProgBitrate, ProgRate, ProgETA string
	// 预设页
	PresetsTitle                                           string
	SavePreset, SavePresetTitle                            string
	OpenPresetDir, SavePresetName                          string
	AddCustomCmd, CustomCmdInput                           string
	LoadedPreset, CustomCmdActive                          string
	SearchPreset                                           string
	SavePresetFailed, LoadPresetFailed, DeletePresetFailed string
	// 输出容器
	ContainerTitle  string
	ContainerFailed string
	// 对话框提示
	PresetUnsupported string
	// 关于
	AboutTitle, AboutLine1, AboutLine2, AboutLine3, AboutGitHub string
	// 其他
	SaveCfgFailed, Exported, LoadCfgFailed, Loaded, SaveDefaultFailed, SavedDefault string
}

var zh = &Strings{
	TabFiles: "文件", TabVideo: "视频", TabAudio: "音频", TabFilters: "滤镜", TabTasks: "任务", TabPresets: "预设", TabSettings: "设置",
	FooterFiles:      "A/D：切换   Tab：焦点   方向键：选择   回车：确定   Delete：移除   Esc/Q：退出",
	FooterVideoAudio: "A/D：切换   Tab：焦点   方向键：选择   回车：确定   /：搜索   Esc/Q：退出",
	FooterFilters:    "A/D：切换   Tab：焦点   方向键：选择   回车：确定   +/-：排序   Delete：删除   /：搜索   Esc/Q：退出",
	FooterPresets:    "A/D：切换   Tab：焦点   方向键：选择   回车：确定   /：搜索   Delete：删除   Esc：退出",
	FooterSettings:   "A/D：切换   Tab：焦点   方向键：选择   回车：确定   Esc/Q：退出",
	FooterTasks:      "A/D：切换   Tab：焦点   方向键：选择   回车：确定   Delete：移除/终止   Esc/Q：退出",
	FiltersTitle:     " 滤镜 ",
	FilterVideoChain: " 视频滤镜（从上到下依次应用） ",
	FilterAudioChain: " 音频滤镜（从上到下依次应用） ",
	FilterAddVideo:   "[添加视频滤镜]", FilterAddAudio: "[添加音频滤镜]",
	FilterMoveUp: "[上移]", FilterMoveDown: "[下移]",
	FilterEmpty:       "（空，按回车或 / 添加）",
	FilterCount:       "视频 %d 个 · 音频 %d 个",
	FilterCopyWarn:    "[yellow]⚠ 复制流时滤镜不生效[-] ",
	FilterNeedEncode:  "滤镜链非空但当前编码器为复制流，滤镜不会生效。是否切换为默认编码（libx264 / aac）？",
	FilterParamsTitle: "滤镜参数", FilterPreview: "预览：",
	FilterValueInvalid: "参数 %s 的值无效：%s",
	FilterListFailed:   "无法获取滤镜列表", FilterPickHint: "（回车选择文件）",
	OK: "确定", Cancel: "取消", Hint: "提示",
	FilesTitle:      "文件 — 输入文件",
	OutputDir:       "输出目录",
	OutputDirTitle:  "输出目录（$file 为输入文件目录）",
	OutputDirSame:   "与输入文件相同",
	AddFile:         "[添加文件]",
	SetContainer:    "[设置输出容器]",
	ContainerLabel:  "输出容器：%s",
	InputPath:       "输入路径",
	PickFile:        "选择文件",
	ReadingInfo:     "正在读取文件信息…",
	NoFFmpeg:        "未找到 %s，无法读取文件信息",
	ParseInfoFailed: "未能解析文件信息",
	VideoTitle:      " 视频 — 编码选项 ",
	Encoder:         "编码器", Decoder: "解码器", Preset: "预设",
	QualityMode: "控制方式", Quality: "质量",
	BitrateBase: "基础比特率", BitrateMax: "最高比特率", BitrateMin: "最低比特率",
	Resolution: "分辨率", FPS: "帧率",
	CustomParams: "自定义参数",
	AudioTitle:   " 音频 — 编码选项 ",
	AudioEncoder: "编码器", AudioBitrate: "比特率", SampleRate: "采样率", BitDepth: "位深度",
	Unspecified:     "未指定",
	Disable:         "禁用",
	SettingsTitle:   " 设置 ",
	ParallelTasks:   "并行任务数",
	RestoreDefaults: "恢复默认配置", RestoreDefaultsConfirm: "确定恢复默认配置吗？",
	OutputOption: "输出选项", Suffix: "指定后缀",
	ExportCfg: "导出配置", LoadCfg: "加载配置", SetDefaultCfg: "保存配置",
	FFmpegPath: "FFmpeg 路径", Language: "语言", About: "关于",
	NamingTimestamp:   "添加时间（默认）",
	NamingSuffix:      "添加指定后缀：%s",
	NamingNone:        "不添加后缀",
	FFmpegPathDefault: "系统 PATH",
	LangZh:            "中文", LangEn: "English",
	TasksTitle: " 任务 ",
	TaskAdd:    "[添加任务]", TaskStart: "[开始]", TaskClear: "[清空]",
	TaskWaiting: "等待中", TaskRunning: "转码中", TaskDone: "已完成", TaskFailed: "失败",
	NoFilesToTask: "请先在文件页添加输入文件",
	TaskShowCmd:   "[显示命令]", CmdLabel: "命令:", CmdReturnHint: "按下回车或空格返回...", TaskSavedTo: "保存到", TaskLogSavedTo: "日志保存至",
	ProgBitrate: "比特率: ", ProgRate: "速率: ", ProgETA: "预计需要",
	PresetsTitle: " 预设 ",
	SavePreset:   "[保存预设]", SavePresetTitle: "保存预设",
	OpenPresetDir: "[打开预设文件夹]", SavePresetName: "预设名称",
	AddCustomCmd: "[添加自定义命令]", CustomCmdInput: "自定义命令",
	LoadedPreset: "已加载预设%s", CustomCmdActive: "已加载自定义命令，是否禁用",
	SearchPreset:       "搜索预设",
	SavePresetFailed:   "无法保存预设：\n%s",
	LoadPresetFailed:   "无法加载预设：\n%s",
	DeletePresetFailed: "无法删除预设：\n%s",
	ContainerTitle:     "输出容器",
	ContainerFailed:    "无法获取容器格式列表",
	PresetUnsupported:  "当前编码器不提供 preset 选项",
	AboutTitle:         "关于",
	AboutLine1:         "FFmpeg-TUI",
	AboutLine2:         "%s",
	AboutLine3:         "本软件是免费的自由软件",
	AboutGitHub:        "GitHub",
	SaveCfgFailed:      "无法保存配置文件：\n%s",
	Exported:           "配置已导出到：\n%s",
	LoadCfgFailed:      "无法加载配置文件：\n%s",
	Loaded:             "已加载配置：\n%s",
	SaveDefaultFailed:  "无法保存配置：\n%s",
	SavedDefault:       "当前配置已保存为：\n%s",
}

var en = &Strings{
	TabFiles: "Files", TabVideo: "Video", TabAudio: "Audio", TabFilters: "Filters", TabTasks: "Tasks", TabPresets: "Presets", TabSettings: "Settings",
	FooterFiles:      "A/D: switch   Tab: focus   Arrows: select   Enter: confirm   Delete: remove   Esc/Q: quit",
	FooterVideoAudio: "A/D: switch   Tab: focus   Arrows: select   Enter: confirm   /: search   Esc/Q: quit",
	FooterFilters:    "A/D: switch   Tab: focus   Arrows: select   Enter: confirm   +/-: reorder   Delete: remove   /: search   Esc/Q: quit",
	FooterPresets:    "A/D: switch   Tab: focus   Arrows: select   Enter: confirm   /: search   Delete: delete   Esc: quit",
	FooterSettings:   "A/D: switch   Tab: focus   Arrows: select   Enter: confirm   Esc/Q: quit",
	FooterTasks:      "A/D: switch   Tab: focus   Arrows: select   Enter: confirm   Delete: remove/terminate   Esc/Q: quit",
	FiltersTitle:     " Filters ",
	FilterVideoChain: " Video filters (top to bottom) ",
	FilterAudioChain: " Audio filters (top to bottom) ",
	FilterAddVideo:   "[Add video filter]", FilterAddAudio: "[Add audio filter]",
	FilterMoveUp: "[Up]", FilterMoveDown: "[Down]",
	FilterEmpty:       "(empty, press Enter or / to add)",
	FilterCount:       "video %d · audio %d",
	FilterCopyWarn:    "[yellow]⚠ filters have no effect with stream copy[-] ",
	FilterNeedEncode:  "The filter chain is not empty but the encoder is set to stream copy, so filters will not apply. Switch to the default encoders (libx264 / aac)?",
	FilterParamsTitle: "Filter parameters", FilterPreview: "Preview: ",
	FilterValueInvalid: "Invalid value for %s: %s",
	FilterListFailed:   "Cannot get filter list", FilterPickHint: " (Enter to pick a file)",
	OK: "OK", Cancel: "Cancel", Hint: "Notice",
	FilesTitle:      "Files — Input",
	OutputDir:       "Output directory",
	OutputDirTitle:  "Output directory ($file = input file directory)",
	OutputDirSame:   "Same as input",
	AddFile:         "[Add file]",
	SetContainer:    "[Output container]",
	ContainerLabel:  "Container: %s",
	InputPath:       "Enter path",
	PickFile:        "Pick file",
	ReadingInfo:     "Reading file info…",
	NoFFmpeg:        "%s not found, cannot read file info",
	ParseInfoFailed: "Failed to parse file info",
	VideoTitle:      " Video — Encoding Options ",
	Encoder:         "Encoder", Decoder: "Decoder", Preset: "Preset",
	QualityMode: "Rate control", Quality: "Quality",
	BitrateBase: "Base bitrate", BitrateMax: "Max bitrate", BitrateMin: "Min bitrate",
	Resolution: "Resolution", FPS: "Frame rate",
	CustomParams: "Custom parameters",
	AudioTitle:   " Audio — Encoding Options ",
	AudioEncoder: "Encoder", AudioBitrate: "Bitrate", SampleRate: "Sample rate", BitDepth: "Bit depth",
	Unspecified:     "Unspecified",
	Disable:         "Disable",
	SettingsTitle:   " Settings ",
	ParallelTasks:   "Parallel tasks",
	RestoreDefaults: "Restore defaults", RestoreDefaultsConfirm: "Restore the default configuration?",
	OutputOption: "Output naming", Suffix: "Suffix",
	ExportCfg: "Export config", LoadCfg: "Load config", SetDefaultCfg: "Save config",
	FFmpegPath: "FFmpeg path", Language: "Language", About: "About",
	NamingTimestamp:   "Add timestamp (default)",
	NamingSuffix:      "Add suffix: %s",
	NamingNone:        "No suffix",
	FFmpegPathDefault: "System PATH",
	LangZh:            "中文", LangEn: "English",
	TasksTitle: " Tasks ",
	TaskAdd:    "[Add tasks]", TaskStart: "[Start]", TaskClear: "[Clear]",
	TaskWaiting: "Waiting", TaskRunning: "Running", TaskDone: "Done", TaskFailed: "Failed",
	NoFilesToTask: "Please add input files on the Files page first.",
	TaskShowCmd:   "[Show command]", CmdLabel: "Command:", CmdReturnHint: "Press Enter or Space to return...", TaskSavedTo: "Saved to ", TaskLogSavedTo: "Log saved to ",
	ProgBitrate: "Bitrate: ", ProgRate: "Rate: ", ProgETA: "ETA ",
	PresetsTitle: " Presets ",
	SavePreset:   "[Save preset]", SavePresetTitle: "Save preset",
	OpenPresetDir: "[Open preset folder]", SavePresetName: "Preset name",
	AddCustomCmd: "[Add custom command]", CustomCmdInput: "Custom command",
	LoadedPreset: "Preset loaded: %s", CustomCmdActive: "A custom command is loaded. Disable it?",
	SearchPreset:       "Search preset",
	SavePresetFailed:   "Cannot save preset:\n%s",
	LoadPresetFailed:   "Cannot load preset:\n%s",
	DeletePresetFailed: "Cannot delete preset:\n%s",
	ContainerTitle:     "Output container",
	ContainerFailed:    "Cannot get container formats",
	PresetUnsupported:  "This encoder has no preset option",
	AboutTitle:         "About",
	AboutLine1:         "FFmpeg-TUI",
	AboutLine2:         "%s",
	AboutLine3:         "This software is free software.",
	AboutGitHub:        "GitHub",
	SaveCfgFailed:      "Cannot save config file:\n%s",
	Exported:           "Config exported to:\n%s",
	LoadCfgFailed:      "Cannot load config file:\n%s",
	Loaded:             "Config loaded from:\n%s",
	SaveDefaultFailed:  "Cannot save config:\n%s",
	SavedDefault:       "Saved current config:\n%s",
}

// detectLang 检测界面语言：环境变量优先，其次 Linux 的 /etc/locale.conf；
// 只要匹配 zh（zh_CN、zh_TW 等）即为中文，其余一律英文
func detectLang() string {
	for _, name := range []string{"LANGUAGE", "LC_ALL", "LC_MESSAGES", "LANG"} {
		if v := os.Getenv(name); v != "" {
			return langFromLocale(v)
		}
	}
	if runtime.GOOS != "windows" {
		if v := localeConfLang(); v != "" {
			return langFromLocale(v)
		}
	}
	return "en"
}

// langFromLocale 按 locale 值返回语言：匹配 zh 为中文，否则英文
func langFromLocale(v string) string {
	if langMatchesZh(v) {
		return "zh"
	}
	return "en"
}

// langMatchesZh 报告 locale 值是否为中文：zh、zh_CN、zh_TW、zh-Hans 等；
// LANGUAGE 可能是冒号分隔的列表，任一项匹配 zh 即视为中文
func langMatchesZh(v string) bool {
	for _, part := range strings.Split(v, ":") {
		p := strings.ToLower(strings.TrimSpace(part))
		if strings.HasPrefix(p, "zh") {
			return true
		}
	}
	return false
}

// localeConfLang 读取 /etc/locale.conf 中的语言设置（Linux）
func localeConfLang() string {
	data, err := os.ReadFile("/etc/locale.conf")
	if err != nil {
		return ""
	}
	vals := map[string]string{}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		vals[strings.TrimSpace(k)] = strings.Trim(strings.TrimSpace(val), `"'`)
	}
	for _, k := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if v := vals[k]; v != "" {
			return v
		}
	}
	return ""
}

// langStrings 返回内置字符串表：zh* 用中文，其余一律英文
func langStrings(lang string) *Strings {
	if langMatchesZh(lang) {
		return zh
	}
	return en
}

// effectiveLang 计算当前语言：配置显式设置优先（可为 lang 包标识符），
// 否则按环境与 locale 检测（默认英文）
func effectiveLang(cfgLang string) string {
	if cfgLang != "" {
		return cfgLang
	}
	return detectLang()
}
