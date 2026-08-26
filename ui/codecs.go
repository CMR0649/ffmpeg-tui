package ui

// 编码器与相关选项数据表，均以 FFmpeg 官方文档
// （https://ffmpeg.org/documentation.html）为准。

// VideoEncoderInfo 描述一种视频编码（第一级选择）及其具体编码器。
type VideoEncoderInfo struct {
	Label      string   // 显示名，如 H.264
	Encoders   []string // 具体编码器，如 libx264
	Presets    []string // preset 选项（视编码器而定，空表示无）
	HasQuality bool     // 是否支持设置质量值（如 -crf）
}

// x264Presets libx264 / libx265 的 -preset 选项（FFmpeg 文档）。
var x264Presets = []string{
	"ultrafast", "superfast", "veryfast", "faster", "fast",
	"medium", "slow", "slower", "veryslow", "placebo",
}

// videoEncoders 视频编码表。
var videoEncoders = []VideoEncoderInfo{
	{Label: "复制流（默认）"},
	{Label: "H.264", Encoders: []string{"libx264"}, Presets: x264Presets, HasQuality: true},
	{Label: "H.265 / HEVC", Encoders: []string{"libx265"}, Presets: x264Presets, HasQuality: true},
	{Label: "VP9", Encoders: []string{"libvpx-vp9"}, HasQuality: true},
	{Label: "AV1", Encoders: []string{"libaom-av1", "libsvtav1"}, HasQuality: true},
	{Label: "VP8", Encoders: []string{"libvpx"}, HasQuality: true},
	{Label: "MPEG-4", Encoders: []string{"mpeg4"}},
	{Label: "MPEG-2", Encoders: []string{"mpeg2video"}},
	{Label: "ProRes", Encoders: []string{"prores_ks"}},
}

// videoDecoderNames 常见视频解码器（可以为空）。
var videoDecoderNames = []string{"空", "h264", "hevc", "mpeg4", "mpeg2video", "vp8", "vp9", "av1"}

// qualityModes 控制方式：中文 + 缩写。
// 对应 FFmpeg 文档：-crf（恒定质量）、-b:v（可变/固定码率）、-maxrate/-minrate/-bufsize。
var qualityModes = []string{"恒定质量 CRF", "可变码率 VBR", "固定码率 CBR"}

// AudioEncoderInfo 描述一种音频编码器。
type AudioEncoderInfo struct {
	Label string // 显示名
	Name  string // FFmpeg 编码器名；空表示复制流
	Wav   bool   // 是否 WAV/PCM（位深度与编码器名称同步）
}

// audioEncoders 音频编码器表。
var audioEncoders = []AudioEncoderInfo{
	{Label: "复制流（默认）"},
	{Label: "AAC", Name: "aac"},
	{Label: "MP3", Name: "libmp3lame"},
	{Label: "FLAC", Name: "flac"},
	{Label: "Opus", Name: "libopus"},
	{Label: "Vorbis", Name: "libvorbis"},
	{Label: "WAV (PCM)", Name: "pcm_s16le", Wav: true},
}

// sampleRates 采样率选项（Hz）。
var sampleRates = []string{"原始", "8000", "11025", "16000", "22050", "32000", "44100", "48000", "88200", "96000", "192000"}

// bitDepths 位深度选项（WAV/PCM 编码器，名称与编码器同步）。
var bitDepths = []struct {
	Label string
	Name  string
}{
	{"16-bit", "pcm_s16le"},
	{"24-bit", "pcm_s24le"},
	{"32-bit", "pcm_s32le"},
}
