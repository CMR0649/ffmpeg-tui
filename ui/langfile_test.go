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
