package ui

import (
	"os/exec"
	"regexp"
	"sort"
	"strings"
)

// 编码器/解码器列表直接通过 ffmpeg 命令获取（不依赖外部文件）：
//   ffmpeg -encoders   → 编码器列表
//   ffmpeg -decoders   → 解码器列表
// 编码器详情通过 `ffmpeg -h encoder=名称` 获取。

var (
	videoEncoders []string // 视频编码器（动态）
	audioEncoders []string // 音频编码器（动态）
	videoDecoders []string // 视频解码器（动态）
	audioDecoders []string // 音频解码器（动态）
	outputFormats []string // 可作输出容器的格式（ffmpeg -formats 中带 E 标记）

	encoderInfoCache = map[string]*EncoderInfo{}
)

// EncoderInfo 通过 `ffmpeg -h encoder=名称` 获取的编码器详情。
type EncoderInfo struct {
	Name         string
	Description  string
	PixelFormats []string
	Options      []string // AVOptions 选项名
}

// HasOption 报告编码器是否支持指定选项（如 preset / crf / qscale）。
func (e *EncoderInfo) HasOption(name string) bool {
	for _, o := range e.Options {
		if o == name {
			return true
		}
	}
	return false
}

// loadCodecLists 加载编码器/解码器列表（幂等，直接使用 ffmpeg 命令）。
func loadCodecLists() {
	if len(videoEncoders) > 0 || len(audioEncoders) > 0 {
		return
	}
	videoEncoders, audioEncoders = codecListFromCmd("ffmpeg", "-encoders")
	videoDecoders, audioDecoders = codecListFromCmd("ffmpeg", "-decoders")
}

// loadFormats 加载可作输出容器的格式列表（幂等）：
// `ffmpeg -formats` 输出中，标记第二列为 E（Muxing supported）的格式。
func loadFormats() {
	if len(outputFormats) > 0 {
		return
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return
	}
	out, err := exec.Command("ffmpeg", "-formats").Output()
	if err != nil {
		return
	}
	outputFormats = parseFormats(string(out))
}

// parseFormats 解析 ffmpeg -formats 输出，返回带 E（muxing）标记的格式名。
// 数据行第一列是标记（如 "DE"、"D"、"E"、" d"），第二列是格式名；
// 表头说明行（含 "." 或 "="）被跳过。
func parseFormats(text string) []string {
	seen := map[string]bool{}
	var names []string
	for _, line := range strings.Split(text, "\n") {
		f := strings.Fields(line)
		if len(f) < 2 {
			continue
		}
		flags := f[0]
		if strings.ContainsAny(flags, ".=") {
			continue // 表头/说明行
		}
		if !strings.Contains(flags, "E") {
			continue // 只有 E 标记（可封装/输出）的格式才能用作输出容器
		}
		name := strings.Split(f[1], ",")[0]
		if seen[name] {
			continue
		}
		seen[name] = true
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// codecLineRe 匹配 " V....D libx264  H.264 ..." 行。
var codecLineRe = regexp.MustCompile(`^\s([VA])\S*\s+(\S+)\s*(.*)$`)

// parseCodecList 解析编码器/解码器列表文本，返回视频与音频名称列表（排序、去重）。
func parseCodecList(text string) (video, audio []string) {
	seen := map[string]bool{}
	for _, line := range strings.Split(text, "\n") {
		m := codecLineRe.FindStringSubmatch(line)
		if m == nil || m[2] == "=" {
			continue // 跳过表头说明行（如 "V..... = Video"）
		}
		typ, name := m[1], m[2]
		if seen[name] {
			continue
		}
		seen[name] = true
		if typ == "V" {
			video = append(video, name)
		} else {
			audio = append(audio, name)
		}
	}
	sort.Strings(video)
	sort.Strings(audio)
	return
}

// codecListFromCmd 运行 ffmpeg -encoders / -decoders 获取列表。
func codecListFromCmd(bin string, args ...string) (video, audio []string) {
	if _, err := exec.LookPath(bin); err != nil {
		return nil, nil
	}
	out, err := exec.Command(bin, args...).Output()
	if err != nil {
		return nil, nil
	}
	return parseCodecList(string(out))
}

// probeEncoder 运行 `ffmpeg -h encoder=名称` 获取编码器详情（带缓存）。
func probeEncoder(name string) *EncoderInfo {
	if info, ok := encoderInfoCache[name]; ok {
		return info
	}
	info := &EncoderInfo{Name: name}
	if out, err := exec.Command("ffmpeg", "-h", "encoder="+name).Output(); err == nil {
		parseEncoderHelp(string(out), info)
	}
	encoderInfoCache[name] = info
	return info
}

// parseEncoderHelp 解析 `ffmpeg -h encoder=名称` 输出。
func parseEncoderHelp(out string, info *EncoderInfo) {
	inOptions := false
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimRight(line, "\r")
		switch {
		case strings.HasPrefix(line, "Encoder "):
			rest := strings.TrimPrefix(line, "Encoder ")
			if i := strings.Index(rest, " ["); i >= 0 {
				info.Name = rest[:i]
				info.Description = strings.TrimSuffix(rest[i+2:], "]:")
			} else {
				info.Name = strings.TrimSuffix(rest, ":")
			}
		case strings.HasPrefix(line, "    Supported pixel formats:"):
			info.PixelFormats = strings.Fields(strings.TrimPrefix(line, "    Supported pixel formats:"))
		case strings.HasSuffix(line, " AVOptions:"):
			inOptions = true
		default:
			if inOptions {
				t := strings.TrimSpace(line)
				if strings.HasPrefix(t, "-") {
					if f := strings.Fields(t); len(f) >= 1 {
						info.Options = append(info.Options, strings.TrimPrefix(f[0], "-"))
					}
				}
			}
		}
	}
}
