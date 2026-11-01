package ui

import (
	"os"
	"reflect"
	"testing"
)

// TestParseRepoLangFile 解析仓库自带的 lang/zh_CN.lang 并校验完整性
func TestParseRepoLangFile(t *testing.T) {
	data, err := os.ReadFile("../lang/zh_CN.lang")
	if err != nil {
		t.Fatalf("read lang file: %v", err)
	}
	p := parseLangFile(string(data))
	if p.ID != "zh_CN" {
		t.Errorf("id=%q", p.ID)
	}
	if p.Name != "简体中文" {
		t.Errorf("name=%q", p.Name)
	}
	// 覆盖全部文案字段
	v := reflect.ValueOf(zh).Elem()
	typ := v.Type()
	var missing []string
	for i := 0; i < v.NumField(); i++ {
		name := typ.Field(i).Name
		if _, ok := p.Values[name]; !ok {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		t.Errorf("missing %d fields: %v", len(missing), missing)
	}
	// 未知键应当被忽略（不报错）
	if _, ok := p.Values["NoSuchKey"]; ok {
		t.Error("unexpected unknown key")
	}
	// 滤镜强制替换块
	if len(p.Replace) == 0 {
		t.Error("empty replacement block")
	}
	if got := p.Replace["hflip Horizontally flip the input video."]; got != "hflip 水平翻转输入视频。" {
		t.Errorf("replace hflip=%q", got)
	}
}

// TestApplyLangPack 校验语言包覆盖内置文案
func TestApplyLangPack(t *testing.T) {
	pack := &langPack{Values: map[string]string{"TabFiles": "文件X", "OK": "好的"}}
	s := applyLangPack(langStrings("en"), pack)
	if s.TabFiles != "文件X" || s.OK != "好的" {
		t.Errorf("apply failed: %q %q", s.TabFiles, s.OK)
	}
	if s.Cancel != "Cancel" {
		t.Errorf("未覆盖字段应保留原值: %q", s.Cancel)
	}
}

// TestParseLangSyntax 校验双引号/单引号/分号一行多条/注释
func TestParseLangSyntax(t *testing.T) {
	text := `
# 注释
[lang=ja_JP]
[name=日本語]
{
    A = "x"; B = 'y'
    C = "带;分号"
    # 注释行
}
{{
    "a b" = "c d"
}}
`
	p := parseLangFile(text)
	if p.ID != "ja_JP" || p.Name != "日本語" {
		t.Errorf("header: %q/%q", p.ID, p.Name)
	}
	if p.Values["A"] != "x" || p.Values["B"] != "y" {
		t.Errorf("values A/B: %q %q", p.Values["A"], p.Values["B"])
	}
	if p.Values["C"] != "带;分号" {
		t.Errorf("quoted semicolon: %q", p.Values["C"])
	}
	if p.Replace["a b"] != "c d" {
		t.Errorf("replace: %q", p.Replace["a b"])
	}
}

// TestParseLangRawText 校验 /-原文-\ 形式的原样文本：
// 内部引号、括号、等号、分号、以及 / \ 混排都按字面保留
func TestParseLangRawText(t *testing.T) {
	text := `
[lang=xx_YY]
[name=/-原样 名称-\]
{
    A = /-含 "引号" 与 [方括号] 与 {花括号} 的文本-\
    B = /-含分号; 的文本-\ ; C = "普通"
    D = /-a=b-\ ; E = a/b
    F = /-it's a 'test'-\
    G = /-{a/\b}[c]-\
}
{{
    /-"ass Render ASS subtitles"-\ = /-ass 翻译 "带引号"-\
}}
`
	p := parseLangFile(text)
	if p.Name != "原样 名称" {
		t.Errorf("name=%q", p.Name)
	}
	cases := map[string]string{
		"A": `含 "引号" 与 [方括号] 与 {花括号} 的文本`,
		"B": "含分号; 的文本",
		"C": "普通",
		"D": "a=b",
		"E": "a/b",
		"F": "it's a 'test'",
		"G": `{a/\b}[c]`,
	}
	for k, want := range cases {
		if got := p.Values[k]; got != want {
			t.Errorf("Values[%s]=%q want %q", k, got, want)
		}
	}
	if got := p.Replace[`"ass Render ASS subtitles"`]; got != `ass 翻译 "带引号"` {
		t.Errorf("Replace=%q", got)
	}
}

// TestParseQuotedKeysWithQuotes 校验键内含引号的三种写法：
// 未转义引号、转义引号、以及用 /-…-/ 包裹引号
func TestParseQuotedKeysWithQuotes(t *testing.T) {
	text := `
{{
    "deinterlace_qsv Quick Sync Video /-"deinterlacing"-\" = "去隔行"
    "scale_qsv Quick Sync Video \"scaling and format conversion\"" = "缩放"
    "vpp_qsv Quick Sync Video "VPP"" = "VPP 加速"
}}
`
	p := parseLangFile(text)
	cases := map[string]string{
		`deinterlace_qsv Quick Sync Video "deinterlacing"`:           "去隔行",
		`scale_qsv Quick Sync Video "scaling and format conversion"`: "缩放",
		`vpp_qsv Quick Sync Video "VPP"`:                             "VPP 加速",
	}
	for k, want := range cases {
		if got := p.Replace[k]; got != want {
			t.Errorf("Replace[%q]=%q want %q", k, got, want)
		}
	}
}

// TestTranslateListText 校验列表文本替换（含名称与说明间空格差异）
func TestTranslateListText(t *testing.T) {
	pack := parseLangFile(`
lang = "zh_CN"
{{
    "vpp_qsv Quick Sync Video "VPP"" = "vpp_qsv 快速同步视频 "VPP""
    /-detelecine Apply an inverse telecine pattern.-\ = "detelecine 应用反转电视电影模式。"
}}
`)
	a := &App{langReplace: pack.Replace}
	a.langNormReplace = map[string]string{}
	for k, v := range pack.Replace {
		a.langNormReplace[normalizeListText(k)] = v
	}
	// 列表里的名称与说明之间是两个空格
	if got := a.translateListText(`vpp_qsv  Quick Sync Video "VPP"`); got != `vpp_qsv 快速同步视频 "VPP"` {
		t.Errorf("vpp_qsv: %q", got)
	}
	if got := a.translateListText("detelecine  Apply an inverse telecine pattern."); got != "detelecine 应用反转电视电影模式。" {
		t.Errorf("detelecine: %q", got)
	}
	// 未配置的文本原样返回
	if got := a.translateListText("foo  Bar"); got != "foo  Bar" {
		t.Errorf("untranslated: %q", got)
	}
}
