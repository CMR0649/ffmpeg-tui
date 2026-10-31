package ui

import (
	"fmt"
	"math"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// mediaInfo 通过 `ffmpeg -i 文件` 解析出的媒体信息
type mediaInfo struct {
	Duration      float64 // 时长（秒）
	FPS           float64 // 视频帧率
	Width, Height int
	HasVideo      bool
	HasAudio      bool
	VideoCodec    string
	AudioCodec    string
	AudioRate     int
	AudioChannels int
}

var (
	// Duration: 00:00:12.00, start: 0.000000, bitrate: 1282 kb/s
	reDuration = regexp.MustCompile(`Duration:\s*(\d+):(\d+):(\d+(?:\.\d+)?)`)
	// Stream #0:0[0x1](und): Video: h264 (High), ...
	reVideoStream = regexp.MustCompile(`Stream #\S+\s*Video:\s*([A-Za-z0-9_]+)`)
	reAudioStream = regexp.MustCompile(`Stream #\S+\s*Audio:\s*([A-Za-z0-9_]+)`)
	reResolution  = regexp.MustCompile(`(\d{2,5})x(\d{2,5})`)
	reFPSField    = regexp.MustCompile(`([\d.]+)\s*fps`)
	reHz          = regexp.MustCompile(`(\d+)\s*Hz`)
	reChannels    = regexp.MustCompile(`(mono|stereo|(\d+)\s*channels)`)
)

// ffmpegInfo 运行 `ffmpeg -i 文件` 并解析媒体信息。
// 该命令会以非零状态退出（“At least one output file must be specified”），
// 但退出前会把媒体信息打印到 stderr。
func (a *App) ffmpegInfo(path string) *mediaInfo {
	cmd := exec.Command(a.ffmpegBin(), "-hide_banner", "-i", path)
	out, _ := cmd.CombinedOutput()
	return parseFFmpegInfo(string(out))
}

// parseFFmpegInfo 解析 `ffmpeg -i` 的输出
func parseFFmpegInfo(raw string) *mediaInfo {
	info := &mediaInfo{}
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimRight(line, "\r")
		if m := reDuration.FindStringSubmatch(line); m != nil {
			h, _ := strconv.Atoi(m[1])
			mi, _ := strconv.Atoi(m[2])
			sec, _ := strconv.ParseFloat(m[3], 64)
			info.Duration = float64(h*3600+mi*60) + sec
			continue
		}
		if m := reVideoStream.FindStringSubmatch(line); m != nil {
			if info.HasVideo {
				continue
			}
			info.HasVideo = true
			info.VideoCodec = m[1]
			rest := line[strings.Index(line, "Video:"):]
			if r := reResolution.FindStringSubmatch(rest); r != nil {
				info.Width, _ = strconv.Atoi(r[1])
				info.Height, _ = strconv.Atoi(r[2])
			}
			if f := reFPSField.FindStringSubmatch(rest); f != nil {
				if v, err := strconv.ParseFloat(f[1], 64); err == nil {
					info.FPS = correctFPS(v)
				}
			}
			continue
		}
		if m := reAudioStream.FindStringSubmatch(line); m != nil {
			if info.HasAudio {
				continue
			}
			info.HasAudio = true
			info.AudioCodec = m[1]
			rest := line[strings.Index(line, "Audio:"):]
			if r := reHz.FindStringSubmatch(rest); r != nil {
				info.AudioRate, _ = strconv.Atoi(r[1])
			}
			if c := reChannels.FindStringSubmatch(rest); c != nil {
				switch {
				case c[1] == "mono":
					info.AudioChannels = 1
				case c[1] == "stereo":
					info.AudioChannels = 2
				case c[2] != "":
					info.AudioChannels, _ = strconv.Atoi(c[2])
				}
			}
		}
	}
	return info
}

// correctFPS 修正 ffmpeg 显示帧率时两位小数舍入造成的失真：
// 24000/1001 会显示为 23.98，修正为 23.976
func correctFPS(fps float64) float64 {
	if math.Abs(fps-23.98) < 0.005 {
		return 23.976
	}
	return fps
}

// formatHMS 格式化时长：≥1 小时为 H:MM:SS（小时不补位），否则为 MM:SS
func formatHMS(seconds float64) string {
	s := int(seconds + 0.5)
	if s < 0 {
		s = 0
	}
	h := s / 3600
	m := (s % 3600) / 60
	sec := s % 60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, sec)
	}
	return fmt.Sprintf("%02d:%02d", m, sec)
}
