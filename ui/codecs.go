package ui

// 选项相关常量表。编码器/解码器列表为动态加载（见 encoders.go），
// 选项取值以 FFmpeg 官方文档（https://ffmpeg.org/documentation.html）为准。

// x264Presets libx264 / libx265 等编码器的 -preset 选项（FFmpeg 文档）。
var x264Presets = []string{
	"ultrafast", "superfast", "veryfast", "faster", "fast",
	"medium", "slow", "slower", "veryslow", "placebo",
}

// qualityModes 控制方式：中文 + 缩写。
// 对应 FFmpeg 文档：-crf（恒定质量）、-b:v / -maxrate / -minrate / -bufsize。
var qualityModes = []string{"恒定质量 CRF", "可变码率 VBR", "固定码率 CBR"}

// videoDecoderNames 常用视频解码器快捷列表（可以为本机 ffmpeg 支持的全部解码器，
// 详见 encoders.go 动态加载；此处保留一个常用子集作为无解码器列表时的兜底）。
var videoDecoderNames = []string{"空", "h264", "hevc", "mpeg4", "mpeg2video", "vp8", "vp9", "av1"}

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
