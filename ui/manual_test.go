package ui

import (
	"strings"
	"testing"
)

// TestFilterManualArgs 校验手动参数的拼接规则
func TestFilterManualArgs(t *testing.T) {
	cases := map[string]string{
		"f=/a.ass":     "=f=/a.ass",
		"=f=/a.ass":    "=f=/a.ass",
		":f=/a.ass":    ":f=/a.ass",
		"/tmp/a b.ass": "=/tmp/a b.ass",
		"":             "",
	}
	for in, want := range cases {
		if got := manualFilterArgs(in); got != want {
			t.Errorf("manualFilterArgs(%q)=%q want %q", in, got, want)
		}
	}
}

// TestFilterChainManual 校验手动参数优先于表单参数
func TestFilterChainManual(t *testing.T) {
	chain := []FilterInstance{
		{Name: "ass", Manual: "f=/tmp/x.ass"},
		{Name: "crop", Params: []FilterParam{{Key: "w", Value: "100"}}},
	}
	if got, want := filterChainString(chain), "ass=f=/tmp/x.ass,crop=w=100"; got != want {
		t.Errorf("filterChainString=%q want %q", got, want)
	}
	both := []FilterInstance{{Name: "loudnorm", Manual: "I=-16", Params: []FilterParam{{Key: "I", Value: "-20"}}}}
	if got := filterChainString(both); got != "loudnorm=I=-16" {
		t.Errorf("manual only: %q", got)
	}
}

// TestFilterItemTextManual 校验链列表项显示手动参数
func TestFilterItemTextManual(t *testing.T) {
	if got := filterItemText(FilterInstance{Name: "ass", Manual: "f=/tmp/x.ass"}); got != "ass  f=/tmp/x.ass" {
		t.Errorf("item text=%q", got)
	}
	if got := filterItemText(FilterInstance{Name: "crop", Params: []FilterParam{{Key: "w", Value: "1"}}}); got != "crop  w=1" {
		t.Errorf("item text=%q", got)
	}
}

// TestFilterManualCopy 校验深拷贝保留手动参数
func TestFilterManualCopy(t *testing.T) {
	src := []FilterInstance{{Name: "ass", Manual: "f=x"}}
	dst := copyFilters(src)
	if len(dst) != 1 || dst[0].Manual != "f=x" {
		t.Errorf("copyFilters=%+v", dst)
	}
	src[0].Manual = "changed"
	if dst[0].Manual != "f=x" {
		t.Errorf("copy should be independent")
	}
}

// TestBuildCommandManualFilter 校验手动参数进入 -vf
func TestBuildCommandManualFilter(t *testing.T) {
	a := &App{cfg: &Config{
		VideoEncoder: "libx264",
		AudioEncoder: "aac",
		VideoFilters: []FilterInstance{{Name: "ass", Manual: "f=/tmp/x.ass"}},
	}}
	got := strings.Join(a.buildCommand("in.mp4", "out.mkv", true, false), " ")
	if !strings.Contains(got, "-vf ass=f=/tmp/x.ass") {
		t.Errorf("cmd=%s", got)
	}
}

// TestFilterSelectionStyle 校验两条链的选中样式：同一时刻只有一条链显示选中
func TestFilterSelectionStyle(t *testing.T) {
	a := &App{}
	a.filterActive = filterChainVideo
	if got := a.filterSelectionStyle(filterChainVideo, true); got != filterSelFocused {
		t.Errorf("video focused=%d", got)
	}
	if got := a.filterSelectionStyle(filterChainAudio, false); got != filterSelNone {
		t.Errorf("audio should be none, got %d", got)
	}
	if got := a.filterSelectionStyle(filterChainVideo, false); got != filterSelActive {
		t.Errorf("video active=%d", got)
	}
	a.filterActive = filterChainAudio
	if got := a.filterSelectionStyle(filterChainAudio, true); got != filterSelFocused {
		t.Errorf("audio focused=%d", got)
	}
	if got := a.filterSelectionStyle(filterChainVideo, false); got != filterSelNone {
		t.Errorf("video should be none, got %d", got)
	}
}
