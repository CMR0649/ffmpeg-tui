package ui

import (
	"os"
	"strings"
)

// Strings 保存界面文本（中英文），用于语言切换。
type Strings struct {
	// 标签页
	TabFiles, TabVideo, TabAudio, TabTasks, TabSettings string
	// 底部按键提示
	FooterSwitch, FooterSelect, FooterDelete string
	// 通用
	OK, Cancel, Search, Hint string
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
	NoFFprobe       string
	ReadInfoFailed  string
	ParseInfoFailed string
	// 视频页
	VideoTitle, Encoder, Decoder, Preset, QualityMode, Quality, BitrateBase, BitrateMax, BitrateMin, Resolution, FPS string
	// 质量模式（键 → 显示）
	QualityCRF, QualityVBR, QualityCBR string
	QualityUnsupported                 string
	// 音频页
	AudioTitle, AudioEncoder, AudioBitrate, SampleRate, BitDepth string
	BitDepthNotSet                                               string
	WavSync                                                      string
	// 设置页
	SettingsTitle, OutputOption, Suffix, ExportCfg, LoadCfg, SetDefaultCfg, FFmpegPath, Language, About string
	NamingTimestamp, NamingSuffix, NamingNone                                                           string
	FFmpegPathDefault                                                                                   string
	LangZh, LangEn                                                                                      string
	// 任务页
	TasksTitle, TaskAdd, TaskStart, TaskClear      string
	TaskWaiting, TaskRunning, TaskDone, TaskFailed string
	NoFilesToTask                                  string
	// 输出容器
	ContainerTitle  string
	ContainerFailed string
	// 对话框提示
	PresetUnsupported     string
	QualityUnsupportedMsg string
	// 关于
	AboutTitle, AboutLine1, AboutLine2, AboutLine3, AboutGitHub string
	// 其他
	FFmpegNotFound                                                                  string
	SaveCfgFailed, Exported, LoadCfgFailed, Loaded, SaveDefaultFailed, SavedDefault string
}

var zh = &Strings{
	TabFiles: "文件", TabVideo: "视频", TabAudio: "音频", TabTasks: "任务", TabSettings: "设置",
	FooterSwitch: "A/D：切换", FooterSelect: "方向键：选择", FooterDelete: "Delete：移除",
	OK: "确定", Cancel: "取消", Search: "搜索", Hint: "提示",
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
	NoFFprobe:       "未找到 %s，无法读取文件信息",
	ReadInfoFailed:  "无法读取文件信息：%s",
	ParseInfoFailed: "未能解析文件信息",
	VideoTitle:      " 视频 — 编码选项 ",
	Encoder:         "编码器", Decoder: "解码器", Preset: "预设",
	QualityMode: "控制方式", Quality: "质量",
	BitrateBase: "基础比特率", BitrateMax: "最高比特率", BitrateMin: "最低比特率",
	Resolution: "分辨率", FPS: "帧率",
	QualityCRF: "恒定质量 CRF", QualityVBR: "可变码率 VBR", QualityCBR: "固定码率 CBR",
	QualityUnsupported: "编码器不支持",
	AudioTitle:         " 音频 — 编码选项 ",
	AudioEncoder:       "编码器", AudioBitrate: "比特率", SampleRate: "采样率", BitDepth: "位深度",
	BitDepthNotSet: "未设置",
	WavSync:        "位深度",
	SettingsTitle:  " 设置 ",
	OutputOption:   "输出选项", Suffix: "指定后缀",
	ExportCfg: "导出配置（JSON）", LoadCfg: "加载配置", SetDefaultCfg: "保存为默认配置",
	FFmpegPath: "FFmpeg 路径", Language: "语言", About: "关于",
	NamingTimestamp:   "添加时间（默认）",
	NamingSuffix:      "添加指定后缀：%s",
	NamingNone:        "不添加后缀",
	FFmpegPathDefault: "系统 PATH",
	LangZh:            "中文", LangEn: "English",
	TasksTitle: " 任务 ",
	TaskAdd:    "[添加任务]", TaskStart: "[开始]", TaskClear: "[清空]",
	TaskWaiting: "等待中", TaskRunning: "转码中", TaskDone: "已完成", TaskFailed: "失败",
	NoFilesToTask:         "请先在文件页添加输入文件。",
	ContainerTitle:        "输出容器",
	ContainerFailed:       "无法获取容器格式列表",
	PresetUnsupported:     "当前编码器不提供 preset 选项",
	QualityUnsupportedMsg: "当前编码器不支持设置质量值。",
	AboutTitle:            "关于",
	AboutLine1:            "FFmpeg-TUI",
	AboutLine2:            "%s",
	AboutLine3:            "by CMR0649",
	AboutGitHub:           "GitHub",
	FFmpegNotFound:        "未找到 %s，无法读取文件信息",
	SaveCfgFailed:         "无法保存配置文件：\n%s",
	Exported:              "配置已导出到：\n%s",
	LoadCfgFailed:         "无法加载配置文件：\n%s",
	Loaded:                "已加载配置：\n%s",
	SaveDefaultFailed:     "无法保存默认配置：\n%s",
	SavedDefault:          "当前配置已保存为默认配置：\n%s",
}

var en = &Strings{
	TabFiles: "Files", TabVideo: "Video", TabAudio: "Audio", TabTasks: "Tasks", TabSettings: "Settings",
	FooterSwitch: "A/D: switch", FooterSelect: "Arrows: select", FooterDelete: "Delete: remove",
	OK: "OK", Cancel: "Cancel", Search: "Search", Hint: "Notice",
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
	NoFFprobe:       "%s not found, cannot read file info",
	ReadInfoFailed:  "Cannot read file info: %s",
	ParseInfoFailed: "Failed to parse file info",
	VideoTitle:      " Video — Encoding Options ",
	Encoder:         "Encoder", Decoder: "Decoder", Preset: "Preset",
	QualityMode: "Rate control", Quality: "Quality",
	BitrateBase: "Base bitrate", BitrateMax: "Max bitrate", BitrateMin: "Min bitrate",
	Resolution: "Resolution", FPS: "Frame rate",
	QualityCRF: "Constant Quality CRF", QualityVBR: "Variable Bitrate VBR", QualityCBR: "Constant Bitrate CBR",
	QualityUnsupported: "Not supported by encoder",
	AudioTitle:         " Audio — Encoding Options ",
	AudioEncoder:       "Encoder", AudioBitrate: "Bitrate", SampleRate: "Sample rate", BitDepth: "Bit depth",
	BitDepthNotSet: "Not set",
	WavSync:        "Bit depth",
	SettingsTitle:  " Settings ",
	OutputOption:   "Output naming", Suffix: "Suffix",
	ExportCfg: "Export config (JSON)", LoadCfg: "Load config", SetDefaultCfg: "Save as default config",
	FFmpegPath: "FFmpeg path", Language: "Language", About: "About",
	NamingTimestamp:   "Add timestamp (default)",
	NamingSuffix:      "Add suffix: %s",
	NamingNone:        "No suffix",
	FFmpegPathDefault: "System PATH",
	LangZh:            "中文", LangEn: "English",
	TasksTitle: " Tasks ",
	TaskAdd:    "[Add tasks]", TaskStart: "[Start]", TaskClear: "[Clear]",
	TaskWaiting: "Waiting", TaskRunning: "Running", TaskDone: "Done", TaskFailed: "Failed",
	NoFilesToTask:         "Please add input files on the Files page first.",
	ContainerTitle:        "Output container",
	ContainerFailed:       "Cannot get container formats",
	PresetUnsupported:     "This encoder has no preset option",
	QualityUnsupportedMsg: "This encoder does not support a quality value.",
	AboutTitle:            "About",
	AboutLine1:            "FFmpeg-TUI",
	AboutLine2:            "%s",
	AboutLine3:            "by CMR0649",
	AboutGitHub:           "GitHub",
	FFmpegNotFound:        "%s not found, cannot read file info",
	SaveCfgFailed:         "Cannot save config file:\n%s",
	Exported:              "Config exported to:\n%s",
	LoadCfgFailed:         "Cannot load config file:\n%s",
	Loaded:                "Config loaded from:\n%s",
	SaveDefaultFailed:     "Cannot save default config:\n%s",
	SavedDefault:          "Saved current config as default:\n%s",
}

// detectLang 按环境变量检测语言（优先级与 GNU gettext 一致）：
// LANGUAGE > LC_ALL > LC_MESSAGES > LANG；返回 "zh" 或 "en"，
// 无法判断时返回 ""。
func detectLang() string {
	for _, name := range []string{"LANGUAGE", "LC_ALL", "LC_MESSAGES", "LANG"} {
		v := os.Getenv(name)
		if v == "" {
			continue
		}
		// LANGUAGE 可以是冒号分隔的列表，取第一个。
		lv := strings.ToLower(strings.Split(v, ":")[0])
		if strings.HasPrefix(lv, "zh") || strings.HasPrefix(lv, "cmn") {
			return "zh"
		}
		if strings.HasPrefix(lv, "en") {
			return "en"
		}
	}
	return ""
}

// langStrings 返回指定语言的字符串表。
func langStrings(lang string) *Strings {
	if lang == "en" {
		return en
	}
	return zh
}

// effectiveLang 计算当前语言：配置显式设置优先，否则按环境变量，最后默认中文。
func effectiveLang(cfgLang string) string {
	if cfgLang == "zh" || cfgLang == "en" {
		return cfgLang
	}
	if d := detectLang(); d != "" {
		return d
	}
	return "zh"
}
