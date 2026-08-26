package ui

import (
	"os/exec"
	"regexp"
	"sort"
	"strings"
)

// 编码器/解码器列表直接通过 ffmpeg 命令获取
//   ffmpeg -encoders   → 编码器列表
//   ffmpeg -decoders   → 解码器列表
// 编码器详情通过 `ffmpeg -h encoder=名称` 获取。

var (
	videoEncoders []string // 视频编码器
	audioEncoders []string // 音频编码器
	videoDecoders []string // 视频解码器
	audioDecoders []string // 音频解码器
	outputFormats []string // 可作输出容器的格式

	encoderInfoCache = map[string]*EncoderInfo{}
)

// EncoderInfo 通过 `ffmpeg -h encoder=名称` 获取的编码器详情。
type EncoderInfo struct {
	Name         string
	Description  string
	PixelFormats []string
	Options      []string            // AVOptions 选项名
	OptionValues map[string][]string // 选项的枚举取值（如 -preset 的 slow/medium/…）
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

// loadFormats 加载可作输出容器的格式列表
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

// parseFormats 解析 ffmpeg -formats 输出
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
			continue
		}
		if !strings.Contains(flags, "E") {
			continue
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

// parseCodecList 解析编码器/解码器列表文本，返回视频与音频名称列表
func parseCodecList(text string) (video, audio []string) {
	seen := map[string]bool{}
	for _, line := range strings.Split(text, "\n") {
		m := codecLineRe.FindStringSubmatch(line)
		if m == nil || m[2] == "=" {
			continue
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
// AVOptions 段中：选项行 "  -preset  <int>  …"，其后的缩进行
// "     slow  0  …" 是该选项的枚举取值（如 av1_nvenc 的 preset）。
func parseEncoderHelp(out string, info *EncoderInfo) {
	inOptions := false
	lastOption := ""
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
			if !inOptions {
				continue
			}
			indent := len(line) - len(strings.TrimLeft(line, " "))
			t := strings.TrimSpace(line)
			if strings.HasPrefix(t, "-") {
				if f := strings.Fields(t); len(f) >= 1 {
					lastOption = strings.TrimPrefix(f[0], "-")
					info.Options = append(info.Options, lastOption)
				}
				continue
			}
			if lastOption != "" && indent >= 4 {
				// 选项的枚举取值行："     slow  0  E..V....... hq 2 passes"
				if f := strings.Fields(t); len(f) >= 2 {
					if info.OptionValues == nil {
						info.OptionValues = map[string][]string{}
					}
					info.OptionValues[lastOption] = append(info.OptionValues[lastOption], f[0])
				}
			}
		}
	}
}
