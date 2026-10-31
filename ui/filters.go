package ui

import (
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// ---------- 数据模型 ----------

// FilterParam 滤镜的一个显式设置的参数
type FilterParam struct {
	Key   string `json:"key"`
	Value string `json:"value"` // 统一存字符串
}

// FilterInstance 链中的一个滤镜实例
type FilterInstance struct {
	Name   string        `json:"name"`
	Params []FilterParam `json:"params"` // 仅保存显式设置的参数（有序）
}

// filterChainString 把滤镜链拼为逗号分隔的简单滤镜图串：
// 首个参数用「名称=参数」连接，其余参数用「:」连接；
// 位置型参数（如 subtitles 的 filename）省略 key
func filterChainString(chain []FilterInstance) string {
	parts := make([]string, 0, len(chain))
	for _, f := range chain {
		s := f.Name
		for i, p := range f.Params {
			sep := ":"
			if i == 0 {
				sep = "="
			}
			key := ""
			if !positionalFilterParam(f.Name, p.Key) {
				key = p.Key + "="
			}
			s += sep + key + escapeFilterValue(p.Value)
		}
		parts = append(parts, s)
	}
	return strings.Join(parts, ",")
}

// positionalFilterParam 报告滤镜参数是否以位置形式（不带 key）输出。
// ffmpeg 中 subtitles 的 filename 常作位置参数。
func positionalFilterParam(filter, key string) bool {
	return filter == "subtitles" && key == "filename"
}

// escapeFilterValue 按 ffmpeg 滤镜图两级转义；仅对 string 型参数值调用，
// 数字/枚举值无需转义（argv 直传 exec.Command，无 shell 引号问题）
func escapeFilterValue(s string) string {
	for _, r := range [][2]string{
		{`\`, `\\`}, {`'`, `\'`}, {`:`, `\:`},
		{`,`, `\,`}, {`;`, `\;`}, {`[`, `\[`}, {`]`, `\]`},
	} {
		s = strings.ReplaceAll(s, r[0], r[1])
	}
	return s
}

// copyFilters 深拷贝滤镜链（预设与配置之间互不影响）
func copyFilters(chain []FilterInstance) []FilterInstance {
	if chain == nil {
		return nil
	}
	out := make([]FilterInstance, len(chain))
	for i, f := range chain {
		out[i] = FilterInstance{Name: f.Name}
		if len(f.Params) > 0 {
			out[i].Params = make([]FilterParam, len(f.Params))
			copy(out[i].Params, f.Params)
		}
	}
	return out
}

// ---------- 动态解析（仿 encoders.go 缓存模式） ----------

// filterDef 一个可用滤镜的定义
type filterDef struct {
	Name string
	Desc string
	IO   string // 输入输出规格，如 V->V / A->A
}

// filterParamInfo 滤镜一个参数的解析结果
type filterParamInfo struct {
	Key      string
	Type     string // boolean/int/double/float/string
	Desc     string
	Default  string
	HasRange bool
	Min, Max float64
	Enum     []filterEnum // 枚举取值（int 类型带子行时）
}

// filterEnum 枚举取值
type filterEnum struct {
	Name string // 枚举名，如 send_field
	Num  string // 整数值
	Desc string
}

// filterInfo 一个滤镜的解析结果
type filterInfo struct {
	Name   string
	Desc   string
	Params []filterParamInfo
}

var (
	filterVideoDefs []filterDef
	filterAudioDefs []filterDef
	filterDefsReady bool
	filterInfoCache = map[string]*filterInfo{}
	filterInfos     = map[string]*filterInfo{}
)

var (
	reFilterDefault = regexp.MustCompile(`\(default ([^()]*)\)`)
	reFilterFromTo  = regexp.MustCompile(`\(from ([-\d.eE+]+) to ([-\d.eE+]+)\)`)
)

// resetFilterCaches 清空滤镜列表与详情缓存
func resetFilterCaches() {
	filterVideoDefs, filterAudioDefs = nil, nil
	filterDefsReady = false
	filterInfoCache = map[string]*filterInfo{}
	filterInfos = map[string]*filterInfo{}
}

// loadFilterLists 用 `ffmpeg -filters` 加载视频/音频滤镜列表（缓存）
func loadFilterLists(bin string) {
	if filterDefsReady {
		return
	}
	if _, err := exec.LookPath(bin); err != nil {
		return
	}
	out, err := exec.Command(bin, "-hide_banner", "-filters").Output()
	if err != nil {
		return
	}
	filterVideoDefs, filterAudioDefs = parseFilterList(string(out))
	filterDefsReady = true
}

// parseFilterList 解析 `ffmpeg -filters`：只保留 IO 精确为 V->V / A->A 的滤镜
func parseFilterList(text string) (video, audio []filterDef) {
	for _, line := range strings.Split(text, "\n") {
		f := strings.Fields(line)
		if len(f) < 3 {
			continue
		}
		io := f[2]
		if !strings.Contains(io, "->") {
			continue
		}
		desc := ""
		if len(f) > 3 {
			desc = strings.Join(f[3:], " ")
		}
		switch io {
		case "V->V":
			video = append(video, filterDef{Name: f[1], Desc: desc, IO: io})
		case "A->A":
			audio = append(audio, filterDef{Name: f[1], Desc: desc, IO: io})
		}
	}
	sort.Slice(video, func(i, j int) bool { return video[i].Name < video[j].Name })
	sort.Slice(audio, func(i, j int) bool { return audio[i].Name < audio[j].Name })
	return video, audio
}

// probeFilter 运行 `ffmpeg -h filter=名称` 获取滤镜参数，结果按名称缓存
func probeFilter(bin, name string) *filterInfo {
	if info, ok := filterInfoCache[name]; ok {
		return info
	}
	info := &filterInfo{Name: name}
	if out, err := exec.Command(bin, "-hide_banner", "-h", "filter="+name).Output(); err == nil {
		info = parseFilterHelp(string(out), name)
	}
	filterInfoCache[name] = info
	filterInfos[name] = info
	return info
}

// parseFilterHelp 解析 `ffmpeg -h filter=名称` 输出
func parseFilterHelp(out, name string) *filterInfo {
	info := &filterInfo{Name: name}
	inOpts := false
	optIndent := -1
	sawTitle := false
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimRight(line, "\r")
		trimmed := strings.TrimSpace(line)
		if !sawTitle && strings.HasPrefix(line, "Filter ") {
			sawTitle = true
			continue
		}
		if !inOpts {
			// 标题后的缩进行是滤镜说明
			if sawTitle && trimmed != "" && strings.HasPrefix(line, " ") && !strings.HasSuffix(trimmed, ":") {
				if info.Desc == "" {
					info.Desc = trimmed
				}
			}
			if strings.HasSuffix(line, " AVOptions:") {
				inOpts = true
				if i := strings.Index(line, " AVOptions:"); i > 0 {
					info.Name = strings.TrimSpace(line[:i])
				}
			}
			continue
		}
		if trimmed == "" {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " "))
		f := strings.Fields(trimmed)
		if len(f) >= 2 && strings.HasPrefix(f[1], "<") {
			if optIndent < 0 {
				optIndent = indent
			}
			info.Params = append(info.Params, parseFilterParam(f, trimmed))
			continue
		}
		// 缩进更深的子行：枚举取值
		if optIndent >= 0 && indent > optIndent && len(info.Params) > 0 && len(f) >= 2 && isIntegerField(f[1]) {
			last := &info.Params[len(info.Params)-1]
			last.Enum = append(last.Enum, filterEnum{Name: f[0], Num: f[1], Desc: enumDescription(f)})
		}
	}
	info.Params = mergeFilterAliases(info.Params)
	return info
}

// parseFilterParam 解析一个选项行
func parseFilterParam(f []string, line string) filterParamInfo {
	p := filterParamInfo{Key: f[0], Type: strings.Trim(f[1], "<>")}
	if m := reFilterDefault.FindStringSubmatch(line); m != nil {
		p.Default = strings.Trim(m[1], `"'`)
	}
	if m := reFilterFromTo.FindStringSubmatch(line); m != nil {
		if lo, err := strconv.ParseFloat(m[1], 64); err == nil {
			if hi, err := strconv.ParseFloat(m[2], 64); err == nil {
				p.HasRange, p.Min, p.Max = true, lo, hi
			}
		}
	}
	if idx := flagsFieldIndex(f); idx >= 0 && idx+1 < len(f) {
		d := strings.Join(f[idx+1:], " ")
		d = reFilterDefault.ReplaceAllString(d, "")
		d = reFilterFromTo.ReplaceAllString(d, "")
		p.Desc = strings.TrimSpace(d)
	}
	return p
}

// flagsFieldIndex 返回字段中标志列（如 ..FV.......）的下标，找不到返回 -1
func flagsFieldIndex(fields []string) int {
	for i := 1; i < len(fields); i++ {
		if isFlagsField(fields[i]) {
			return i
		}
	}
	return -1
}

// isIntegerField 报告字段是否为整数
func isIntegerField(s string) bool {
	if s == "" {
		return false
	}
	if s[0] != '-' && (s[0] < '0' || s[0] > '9') {
		return false
	}
	_, err := strconv.Atoi(s)
	return err == nil
}

// ---------- curated 覆盖层 ----------

// filterParamAlias 参数别名：值为空表示隐藏该别名参数（由主参数承担）
var filterParamAlias = map[string]bool{
	"out_w": true, "out_h": true, "out_x": true, "out_y": true,
}

// mergeFilterAliases 合并别名参数：只保留主参数（如 crop 只显示 w/h/x/y）
func mergeFilterAliases(params []filterParamInfo) []filterParamInfo {
	if len(params) == 0 {
		return params
	}
	keys := map[string]bool{}
	for _, p := range params {
		keys[p.Key] = true
	}
	out := make([]filterParamInfo, 0, len(params))
	for _, p := range params {
		if filterParamAlias[p.Key] {
			continue
		}
		out = append(out, p)
	}
	return out
}

// filterFilePathParam 报告是否为文件路径型参数（提供输入路径/选择文件）
func filterFilePathParam(filter, key string) bool {
	switch filter + "." + key {
	case "subtitles.filename", "ass.filename", "drawtext.fontfile", "movie.filename", "amovie.filename":
		return true
	}
	return false
}

// filterParamDescZh 常用滤镜参数的中文说明（仅中文界面使用，不替代动态解析）
var filterParamDescZh = map[string]string{
	"crop.w":                "输出宽度",
	"crop.h":                "输出高度",
	"crop.x":                "左上角横坐标",
	"crop.y":                "左上角纵坐标",
	"scale.w":               "输出宽度",
	"scale.h":               "输出高度",
	"fps.fps":               "目标帧率",
	"drawtext.text":         "绘制文本",
	"drawtext.fontsize":     "字号",
	"drawtext.fontcolor":    "字体颜色",
	"drawtext.fontfile":     "字体文件",
	"drawtext.x":            "文本横坐标",
	"drawtext.y":            "文本纵坐标",
	"subtitles.filename":    "字幕文件",
	"subtitles.force_style": "强制字幕样式",
	"volume.volume":         "音量倍数",
	"atempo.tempo":          "变速倍数",
	"transpose.dir":         "旋转方向",
	"loudnorm.I":            "目标响度",
}

// filterParamDesc 返回参数说明：中文界面优先用 curated 中文说明
func (a *App) filterParamDesc(filter string, p filterParamInfo) string {
	if a.lang == "zh" {
		if d := filterParamDescZh[filter+"."+p.Key]; d != "" {
			return d
		}
		if d := filterParamDescZh[filter+"."+strings.TrimPrefix(p.Key, "out_")]; d != "" && filterParamAlias[p.Key] {
			return d
		}
	}
	return p.Desc
}

// filterParamLabel 参数在表单中的标签：key + 说明 + 范围
func (a *App) filterParamLabel(filter string, p filterParamInfo) string {
	label := p.Key
	if d := a.filterParamDesc(filter, p); d != "" {
		label += "（" + d + "）"
	}
	if p.HasRange {
		label += " [" + trimFloat(p.Min) + "~" + trimFloat(p.Max) + "]"
	}
	return label
}

// trimFloat 去掉浮点数多余的零
func trimFloat(v float64) string {
	s := strconv.FormatFloat(v, 'f', -1, 64)
	return s
}

// filterDefaultLabel 参数默认值展示（空则不显示）
func filterDefaultLabel(p filterParamInfo) string {
	if p.Default == "" {
		return ""
	}
	return p.Default
}
