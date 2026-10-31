package ui

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
)

// langPack 一个语言包（来自 <配置目录>/lang/*.lang）
type langPack struct {
	ID      string            // 语言标识符，如 zh_CN
	Name    string            // 语言名（显示在语言列表中）
	Values  map[string]string // 文案键（Strings 字段名）→ 文本
	Replace map[string]string // 列表文本 → 替换文本（{{ }} 块）
	Path    string            // 来源文件路径
}

// 语言包缓存
var (
	langPacks       []*langPack
	langPacksLoaded bool
)

// loadLangPacks 读取 langDir 下的 *.lang 文件（带缓存）
func loadLangPacks() []*langPack {
	if langPacksLoaded {
		return langPacks
	}
	langPacksLoaded = true
	langPacks = nil
	seen := map[string]bool{}
	// 与预设一致：当前目录的 lang 优先，其次默认目录；同标识符只取先读到的
	for _, dir := range langDirs() {
		files, err := filepath.Glob(filepath.Join(dir, "*.lang"))
		if err != nil {
			continue
		}
		sort.Strings(files)
		for _, f := range files {
			data, err := os.ReadFile(f)
			if err != nil {
				continue
			}
			p := parseLangFile(string(data))
			if p.ID == "" || seen[strings.ToLower(p.ID)] {
				continue
			}
			seen[strings.ToLower(p.ID)] = true
			if p.Name == "" {
				p.Name = p.ID
			}
			p.Path = f
			langPacks = append(langPacks, p)
		}
	}
	return langPacks
}

// reloadLangPacks 丢弃缓存并重新读取语言文件
func reloadLangPacks() {
	langPacksLoaded = false
	langPacks = nil
	loadLangPacks()
}

// findLangPack 按标识符查找语言包，找不到返回 nil
func findLangPack(id string) *langPack {
	if id == "" {
		return nil
	}
	for _, p := range loadLangPacks() {
		if strings.EqualFold(p.ID, id) {
			return p
		}
	}
	return nil
}

// parseLangFile 解析 *.lang 文本：支持 [lang=xx]、[name=xx] 头部、
// { } 文案块与 {{ }} 强制替换块；# 开头为注释，; 可在一行内分隔多条
func parseLangFile(text string) *langPack {
	pack := &langPack{Values: map[string]string{}, Replace: map[string]string{}}
	const (
		blockNone = iota
		blockValues
		blockReplace
	)
	block := blockNone
	for _, raw := range strings.Split(text, "\n") {
		line := strings.TrimSpace(strings.TrimRight(raw, "\r"))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		switch line {
		case "{{":
			block = blockReplace
			continue
		case "{":
			block = blockValues
			continue
		case "}}", "}":
			block = blockNone
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			body := strings.TrimSpace(line[1 : len(line)-1])
			k, v, ok := strings.Cut(body, "=")
			if !ok {
				continue
			}
			switch strings.ToLower(strings.TrimSpace(k)) {
			case "lang":
				pack.ID = unquoteLangValue(v)
			case "name":
				pack.Name = unquoteLangValue(v)
			}
			continue
		}
		for _, stmt := range splitLangStatements(line) {
			key, val, ok := splitLangAssign(stmt)
			if !ok || key == "" {
				continue
			}
			if block == blockReplace {
				pack.Replace[key] = val
			} else {
				pack.Values[key] = val
			}
		}
	}
	return pack
}

// splitLangStatements 按引号外的分号把一行拆成多条语句
func splitLangStatements(line string) []string {
	var out []string
	var cur strings.Builder
	var quote byte
	for i := 0; i < len(line); i++ {
		c := line[i]
		if quote != 0 {
			cur.WriteByte(c)
			if c == '\\' && i+1 < len(line) {
				i++
				cur.WriteByte(line[i])
				continue
			}
			if c == quote {
				quote = 0
			}
			continue
		}
		switch c {
		case '"', '\'':
			quote = c
			cur.WriteByte(c)
		case ';':
			if s := strings.TrimSpace(cur.String()); s != "" {
				out = append(out, s)
			}
			cur.Reset()
		default:
			cur.WriteByte(c)
		}
	}
	if s := strings.TrimSpace(cur.String()); s != "" {
		out = append(out, s)
	}
	return out
}

// splitLangAssign 把 "键 = 值" 语句拆成键值；键可为带引号的整段文本
func splitLangAssign(stmt string) (key, value string, ok bool) {
	stmt = strings.TrimSpace(stmt)
	if stmt == "" {
		return "", "", false
	}
	if stmt[0] == '"' || stmt[0] == '\'' {
		q := stmt[0]
		end := -1
		for i := 1; i < len(stmt); i++ {
			if stmt[i] == '\\' {
				i++
				continue
			}
			if stmt[i] == q {
				end = i
				break
			}
		}
		if end < 0 {
			return "", "", false
		}
		rest := strings.TrimSpace(stmt[end+1:])
		if !strings.HasPrefix(rest, "=") {
			return "", "", false
		}
		return unquoteLangValue(stmt[:end+1]), unquoteLangValue(strings.TrimSpace(rest[1:])), true
	}
	k, v, ok := strings.Cut(stmt, "=")
	if !ok {
		return "", "", false
	}
	return unquoteLangValue(k), unquoteLangValue(v), true
}

// unquoteLangValue 去掉成对的单/双引号并处理 \\ \" \' \n \t 转义
func unquoteLangValue(s string) string {
	s = strings.TrimSpace(s)
	if len(s) < 2 || (s[0] != '"' && s[0] != '\'') || s[len(s)-1] != s[0] {
		return s
	}
	inner := s[1 : len(s)-1]
	var b strings.Builder
	for i := 0; i < len(inner); i++ {
		c := inner[i]
		if c == '\\' && i+1 < len(inner) {
			i++
			switch inner[i] {
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			default:
				b.WriteByte(inner[i])
			}
			continue
		}
		b.WriteByte(c)
	}
	return b.String()
}

// applyLangPack 用语言包覆盖内置文案表（键为 Strings 的字段名，未提供的保留原值）
func applyLangPack(base *Strings, pack *langPack) *Strings {
	out := *base
	v := reflect.ValueOf(&out).Elem()
	for k, val := range pack.Values {
		f := v.FieldByName(k)
		if f.IsValid() && f.Kind() == reflect.String && f.CanSet() {
			f.SetString(val)
		}
	}
	return &out
}

// resolveStrings 返回语言标识对应的文案表：内置 zh/en，或 lang 包（以英文为底覆盖）
func resolveStrings(lang string) *Strings {
	if pack := findLangPack(lang); pack != nil {
		return applyLangPack(langStrings("en"), pack)
	}
	return langStrings(lang)
}

// translateListText 按语言包的强制替换表翻译列表文本：
// 先精确匹配，再按「空白归一」匹配（忽略列表中名称与说明之间的多余空格）
func (a *App) translateListText(s string) string {
	if a.langReplace == nil {
		return s
	}
	if v, ok := a.langReplace[s]; ok {
		return v
	}
	if v, ok := a.langNormReplace[normalizeListText(s)]; ok {
		return v
	}
	return s
}

// normalizeListText 把连续空白压成一个空格并去掉首尾空白
func normalizeListText(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// applyLanguage 按当前配置计算文案表与强制替换表
func (a *App) applyLanguage() {
	a.lang = effectiveLang(a.cfg.Lang)
	a.s = resolveStrings(a.lang)
	a.langReplace = nil
	a.langNormReplace = nil
	if pack := findLangPack(a.lang); pack != nil {
		a.langReplace = pack.Replace
		if len(pack.Replace) > 0 {
			a.langNormReplace = make(map[string]string, len(pack.Replace))
			for k, v := range pack.Replace {
				a.langNormReplace[normalizeListText(k)] = v
			}
		}
	}
}
