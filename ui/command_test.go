package ui

import (
	"strings"
	"testing"
)

// TestFilterChainString 表驱动校验滤镜串生成与转义
// （样例已在 ffmpeg 7.1.5 实测通过）
func TestFilterChainString(t *testing.T) {
	cases := []struct {
		name  string
		chain []FilterInstance
		want  string
	}{
		{
			name:  "字幕文件名含逗号分号",
			chain: []FilterInstance{{Name: "subtitles", Params: []FilterParam{{Key: "filename", Value: "/tmp/weird,name;test.srt"}}}},
			want:  `subtitles=/tmp/weird\,name\;test.srt`,
		},
		{
			name:  "drawtext 文本含单引号冒号逗号分号",
			chain: []FilterInstance{{Name: "drawtext", Params: []FilterParam{{Key: "text", Value: "it's a: test, ok;"}}}},
			want:  `drawtext=text=it\'s a\: test\, ok\;`,
		},
		{
			name: "多滤镜顺序",
			chain: []FilterInstance{
				{Name: "crop", Params: []FilterParam{{Key: "w", Value: "160"}, {Key: "h", Value: "120"}}},
				{Name: "drawtext", Params: []FilterParam{{Key: "text", Value: "hi"}}},
			},
			want: `crop=w=160:h=120,drawtext=text=hi`,
		},
		{
			name:  "无参数裸名",
			chain: []FilterInstance{{Name: "hflip"}},
			want:  "hflip",
		},
		{
			name:  "空链",
			chain: nil,
			want:  "",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := filterChainString(c.chain); got != c.want {
				t.Errorf("filterChainString()=%q want %q", got, c.want)
			}
		})
	}
}

// TestEscapeFilterValue 校验各特殊字符的转义
func TestEscapeFilterValue(t *testing.T) {
	cases := []struct{ in, want string }{
		{`a\b`, `a\\b`},
		{`a'b`, `a\'b`},
		{`a:b`, `a\:b`},
		{`a,b`, `a\,b`},
		{`a;b`, `a\;b`},
		{`a[b]c`, `a\[b\]c`},
		{"plain", "plain"},
	}
	for _, c := range cases {
		if got := escapeFilterValue(c.in); got != c.want {
			t.Errorf("escapeFilterValue(%q)=%q want %q", c.in, got, c.want)
		}
	}
}

// TestBuildCommandFilters 校验 -vf/-af 的插入位置与编码器条件
func TestBuildCommandFilters(t *testing.T) {
	a := &App{cfg: &Config{
		VideoEncoder: "libx264",
		AudioEncoder: "aac",
		VideoFPS:     "30",
		CustomParams: "-x264-params key=val",
		VideoFilters: []FilterInstance{{Name: "crop", Params: []FilterParam{{Key: "w", Value: "100"}}}},
		AudioFilters: []FilterInstance{{Name: "loudnorm"}},
	}}
	got := strings.Join(a.buildCommand("in.mp4", "out.mkv", true, true), " ")
	// -vf 在 -r 之后、自定义参数之前
	if !strings.Contains(got, "-r 30 -vf crop=w=100 -x264-params") {
		t.Errorf("-vf position: %s", got)
	}
	// -af 在音频编码参数之后
	if !strings.Contains(got, "-c:a aac -af loudnorm") {
		t.Errorf("-af position: %s", got)
	}

	// 复制流（编码器为空）时不输出滤镜
	a.cfg.VideoEncoder = ""
	a.cfg.AudioEncoder = ""
	got = strings.Join(a.buildCommand("in.mp4", "out.mkv", true, true), " ")
	if strings.Contains(got, "-vf") || strings.Contains(got, "-af") {
		t.Errorf("copy stream must not emit filters: %s", got)
	}

	// 禁用编码器时也不输出
	a.cfg.VideoEncoder = encoderDisabled
	a.cfg.AudioEncoder = encoderDisabled
	got = strings.Join(a.buildCommand("in.mp4", "out.mkv", true, true), " ")
	if strings.Contains(got, "-vf") || strings.Contains(got, "-af") {
		t.Errorf("disabled encoder must not emit filters: %s", got)
	}
}
