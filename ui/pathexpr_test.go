package ui

import (
	"os"
	"path/filepath"
	"testing"
)

// mkTestTree 建一个临时目录树用于搜索测试
func mkTestTree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	files := []string{
		"a.srt", "B.srt", "abc.txt", "xyz.srt",
		filepath.Join("sub", "inner.srt"),
	}
	for _, f := range files {
		p := filepath.Join(root, f)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// TestResolveFileTokens 校验 ${file} / ${file.d}
func TestResolveFileTokens(t *testing.T) {
	got, err := resolvePathExpr("${file}", "/in/x.mkv")
	if err != nil || len(got) != 1 || got[0] != "/in/x.mkv" {
		t.Fatalf("${file}=%v err=%v", got, err)
	}
	got, err = resolvePathExpr("${file.d}/out", "/in/x.mkv")
	if err != nil || len(got) != 1 || got[0] != "/in/out" {
		t.Fatalf("${file.d}=%v err=%v", got, err)
	}
	// 没有输入文件时报错
	if _, err := resolvePathExpr("${file}", ""); err == nil {
		t.Error("没有输入文件时应报错")
	}
}

// TestResolveSearchContains 校验 ${s}(text) 包含匹配（非递归）
func TestResolveSearchContains(t *testing.T) {
	root := mkTestTree(t)
	got, err := resolvePathExpr(root+"/${s}(.srt)", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("期望 3 个 .srt（不含子目录），得到 %v", got)
	}
	for _, p := range got {
		if filepath.Dir(p) != root {
			t.Errorf("不应递归：%s", p)
		}
	}
	// 区分大小写
	got, err = resolvePathExpr(root+"/${s.cs}(B)", "")
	if err != nil || len(got) != 1 || filepath.Base(got[0]) != "B.srt" {
		t.Fatalf("${s.cs}(B)=%v err=%v", got, err)
	}
	// 递归
	got, err = resolvePathExpr(root+"/${s.r}(.srt)", "")
	if err != nil || len(got) != 4 {
		t.Fatalf("${s.r}(.srt)=%v err=%v", got, err)
	}
}

// TestResolveSearchRegex 校验 ${s}(/re/) 正则匹配
func TestResolveSearchRegex(t *testing.T) {
	root := mkTestTree(t)
	got, err := resolvePathExpr(root+"/${s}(/^a.*\\.srt$/)", "")
	if err != nil || len(got) != 1 || filepath.Base(got[0]) != "a.srt" {
		t.Fatalf("正则匹配=%v err=%v", got, err)
	}
	// 正则 + 区分大小写
	if _, err := resolvePathExpr(root+"/${s.cs}(/^b/)", ""); err == nil {
		t.Error("区分大小写的正则不应匹配 B.srt")
	}
	if got, err := resolvePathExpr(root+"/${s}(/^b/)", ""); err != nil || len(got) != 1 {
		t.Errorf("忽略大小写的正则应匹配：%v %v", got, err)
	}
}

// TestResolveSearchFuzzy 校验 ${s.*}(text) 模糊匹配
func TestResolveSearchFuzzy(t *testing.T) {
	root := mkTestTree(t)
	// "ib.srt" 作为子序列可命中 B.srt / inner.srt 等
	got, err := resolvePathExpr(root+"/${s.r.*}(inner)", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || filepath.Base(got[0]) != "inner.srt" {
		t.Fatalf("模糊匹配=%v", got)
	}
}

// TestResolveGlob 校验通配符展开
func TestResolveGlob(t *testing.T) {
	root := mkTestTree(t)
	got, err := resolvePathExpr(root+"/*.srt", "")
	if err != nil || len(got) != 3 {
		t.Fatalf("glob=%v err=%v", got, err)
	}
	// 无匹配时原样返回
	got, err = resolvePathExpr(root+"/*.nomatch", "")
	if err != nil || len(got) != 1 {
		t.Fatalf("无匹配=%v err=%v", got, err)
	}
}

// TestResolveUnknownExpr 校验未知表达式报错
func TestResolveUnknownExpr(t *testing.T) {
	if _, err := resolvePathExpr("${zzz}", ""); err == nil {
		t.Error("未知表达式应报错")
	}
}

// TestResolveFilterText 校验滤镜参数文本里的表达式替换
func TestResolveFilterText(t *testing.T) {
	root := mkTestTree(t)
	got := resolveFilterText("f=${file.d}/a.srt", "/in/x.mkv")
	if got != "f=/in/a.srt" {
		t.Errorf("resolveFilterText=%q", got)
	}
	got = resolveFilterText("f="+root+"/${s}(.srt)", "")
	if got != "f="+filepath.Join(root, "B.srt") {
		t.Errorf("search in filter text=%q", got)
	}
	// 无表达式原样返回
	if got := resolveFilterText("I=-16", ""); got != "I=-16" {
		t.Errorf("plain=%q", got)
	}
}

// TestResolveOutputDir 校验输出目录解析（$file 兼容、${file.d}、通配符、普通值）
func TestResolveOutputDir(t *testing.T) {
	cases := []struct{ setting, input, want string }{
		{"", "/in/x.mkv", "/in"},
		{"$file", "/in/x.mkv", "/in"},
		{"${file.d}/out", "/in/x.mkv", "/in/out"},
		{"/abs/out", "/in/x.mkv", "/abs/out"},
		{"rel/out", "/in/x.mkv", "rel/out"},
	}
	for _, c := range cases {
		if got := ResolveOutputDir(c.setting, c.input); got != c.want {
			t.Errorf("ResolveOutputDir(%q,%q)=%q want %q", c.setting, c.input, got, c.want)
		}
	}
}

// TestFilterChainResolvesExpr 校验命令生成时按输入文件解析滤镜参数里的表达式
func TestFilterChainResolvesExpr(t *testing.T) {
	chain := []FilterInstance{{Name: "subtitles", Manual: "f=${file.d}/sub.srt"}}
	got := filterChainStringInput(chain, "/in/x.mkv")
	if got != "subtitles=f=/in/sub.srt" {
		t.Errorf("chain=%q", got)
	}
	// 预览（无输入文件）保留原样
	if got := filterChainString(chain); got != "subtitles=f=${file.d}/sub.srt" {
		t.Errorf("preview=%q", got)
	}
}
