package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/sahilm/fuzzy"
)

// maxPathExpand 单个路径表达式最多展开出的路径数量
const maxPathExpand = 500

// searchFlags ${s...} 的搜索标志
type searchFlags struct {
	caseSensitive bool // .cs
	recursive     bool // .r
	fuzzyMatch    bool // .*
}

// pathExprToken 一个 ${...} 表达式
type pathExprToken struct {
	name    string // file / file.d / s / s.cs / s.r / s.*
	pattern string // ${s...}(pattern) 的模式
	length  int    // 表达式在字符串中占用的字节数
}

// parsePathToken 解析 s 开头的 ${...} 表达式（s 必须以 "${" 开头）
func parsePathToken(s string) (pathExprToken, error) {
	if !strings.HasPrefix(s, "${") {
		return pathExprToken{}, fmt.Errorf("不是 ${...} 表达式")
	}
	end := strings.IndexByte(s, '}')
	if end < 0 {
		return pathExprToken{}, fmt.Errorf("表达式缺少 }")
	}
	name := strings.TrimSpace(s[2:end])
	tok := pathExprToken{name: name, length: end + 1}
	rest := s[end+1:]
	if _, ok := parseSearchFlags(name); ok && strings.HasPrefix(rest, "(") {
		pattern, n, ok := scanParenPattern(rest)
		if !ok {
			return pathExprToken{}, fmt.Errorf("表达式 ${%s} 缺少 } 后面的 (模式)", name)
		}
		tok.pattern = pattern
		tok.length += n
	}
	return tok, nil
}

// scanParenPattern 从 s[0]=='(' 扫描到配对的 ')'，返回括号内文本与占用长度；
// 支持嵌套括号与 \ 转义（便于书写正则）
func scanParenPattern(s string) (string, int, bool) {
	if !strings.HasPrefix(s, "(") {
		return "", 0, false
	}
	depth := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '\\':
			i++
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return s[1:i], i + 1, true
			}
		}
	}
	return "", 0, false
}

// parseSearchFlags 解析 s / s.cs / s.r / s.* / s.cs.r.* 形式的名字
func parseSearchFlags(name string) (searchFlags, bool) {
	parts := strings.Split(name, ".")
	if len(parts) == 0 || parts[0] != "s" {
		return searchFlags{}, false
	}
	var f searchFlags
	for _, p := range parts[1:] {
		switch p {
		case "":
			// 忽略空段
		case "cs":
			f.caseSensitive = true
		case "r":
			f.recursive = true
		case "*":
			f.fuzzyMatch = true
		default:
			return searchFlags{}, false
		}
	}
	return f, true
}

// hasGlobMeta 报告字符串是否含通配符
func hasGlobMeta(s string) bool {
	return strings.ContainsAny(s, "*?[")
}

// pathMatcher 返回按名称匹配的函数（正则 / 模糊 / 包含）
func pathMatcher(pattern string, f searchFlags) (func(string) bool, error) {
	// /re/ 形式表示正则（带 .cs 时区分大小写）
	if len(pattern) >= 2 && strings.HasPrefix(pattern, "/") && strings.HasSuffix(pattern, "/") {
		expr := pattern[1 : len(pattern)-1]
		if !f.caseSensitive {
			expr = "(?i)" + expr
		}
		re, err := regexp.Compile(expr)
		if err != nil {
			return nil, fmt.Errorf("正则无效：%v", err)
		}
		return re.MatchString, nil
	}
	if f.fuzzyMatch {
		// 模糊匹配由 github.com/sahilm/fuzzy 提供（不区分大小写）
		return func(name string) bool {
			return len(fuzzy.Find(pattern, []string{name})) > 0
		}, nil
	}
	if f.caseSensitive {
		return func(name string) bool { return strings.Contains(name, pattern) }, nil
	}
	lower := strings.ToLower(pattern)
	return func(name string) bool { return strings.Contains(strings.ToLower(name), lower) }, nil
}

// searchFiles 在 base 目录下按名称搜索文件，返回匹配的文件路径；
// base 为空时用当前目录；f.recursive 时递归子目录
func searchFiles(base, pattern string, f searchFlags) ([]string, error) {
	match, err := pathMatcher(pattern, f)
	if err != nil {
		return nil, err
	}
	dir := base
	if dir == "" {
		dir = "."
	}
	var out []string
	add := func(path, name string) {
		if len(out) >= maxPathExpand {
			return
		}
		if match(name) {
			out = append(out, path)
		}
	}
	if f.recursive {
		err := filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
			if err != nil || d == nil || d.IsDir() {
				return nil
			}
			add(p, d.Name())
			if len(out) >= maxPathExpand {
				return filepath.SkipAll
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	} else {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil, err
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			add(filepath.Join(dir, e.Name()), e.Name())
		}
	}
	sort.Strings(out)
	return out, nil
}

// resolvePathExpr 展开路径表达式，返回全部匹配结果；
// 不含表达式时只做通配符展开（无匹配则原样返回）
func resolvePathExpr(expr, inputFile string) ([]string, error) {
	if !strings.Contains(expr, "${") {
		if hasGlobMeta(expr) {
			if m, err := filepath.Glob(expr); err == nil && len(m) > 0 {
				sort.Strings(m)
				return m, nil
			}
		}
		return []string{expr}, nil
	}
	var out []string
	if err := expandPathInto(&out, expr, inputFile); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("没有匹配的文件：%s", expr)
	}
	return out, nil
}

// expandPathInto 递归展开表达式
func expandPathInto(out *[]string, tmpl, inputFile string) error {
	i := strings.Index(tmpl, "${")
	if i < 0 {
		*out = append(*out, tmpl)
		if len(*out) > maxPathExpand {
			return fmt.Errorf("匹配结果过多（超过 %d 个）", maxPathExpand)
		}
		return nil
	}
	prefix := tmpl[:i]
	tok, err := parsePathToken(tmpl[i:])
	if err != nil {
		return err
	}
	after := tmpl[i+tok.length:]

	var values []string
	switch tok.name {
	case "file":
		if inputFile == "" {
			return fmt.Errorf("${file} 需要输入文件")
		}
		values = []string{inputFile}
	case "file.d":
		if inputFile == "" {
			return fmt.Errorf("${file.d} 需要输入文件")
		}
		values = []string{filepath.Dir(inputFile)}
	default:
		flags, ok := parseSearchFlags(tok.name)
		if !ok {
			return fmt.Errorf("未知表达式 ${%s}", tok.name)
		}
		matches, err := searchFiles(prefix, tok.pattern, flags)
		if err != nil {
			return err
		}
		if len(matches) == 0 {
			return fmt.Errorf("没有匹配的文件：${%s}(%s)", tok.name, tok.pattern)
		}
		// 搜索结果已包含目录前缀，直接拼接后半段
		for _, m := range matches {
			if err := expandPathInto(out, m+after, inputFile); err != nil {
				return err
			}
		}
		return nil
	}
	for _, v := range values {
		if err := expandPathInto(out, prefix+v+after, inputFile); err != nil {
			return err
		}
	}
	return nil
}

// resolvePathExprOne 展开为单个路径（多匹配取第一个）
func resolvePathExprOne(expr, inputFile string) (string, error) {
	m, err := resolvePathExpr(expr, inputFile)
	if err != nil {
		return "", err
	}
	if len(m) == 0 {
		return "", fmt.Errorf("没有匹配的文件：%s", expr)
	}
	return m[0], nil
}

// resolveFilterText 替换滤镜参数文本中的路径表达式（${file}、${file.d}、${s...}(...)）；
// 展开失败时原样返回，避免影响命令生成
func resolveFilterText(s, inputFile string) string {
	if !strings.Contains(s, "${") {
		return s
	}
	for i := 0; i < 20; i++ { // 限制替换轮次，避免异常输入导致死循环
		idx := strings.Index(s, "${")
		if idx < 0 {
			break
		}
		tok, err := parsePathToken(s[idx:])
		if err != nil {
			return s
		}
		if flags, ok := parseSearchFlags(tok.name); ok {
			// 搜索：表达式前面连续的路径字符作为搜索目录
			start := idx
			for start > 0 && !strings.ContainsRune(" :,;=\"'|", rune(s[start-1])) {
				start--
			}
			matches, err := searchFiles(s[start:idx], tok.pattern, flags)
			if err != nil || len(matches) == 0 {
				return s
			}
			s = s[:start] + matches[0] + s[idx+tok.length:]
			continue
		}
		if tok.name != "file" && tok.name != "file.d" {
			return s
		}
		val, err := resolvePathExprOne(s[idx:idx+tok.length], inputFile)
		if err != nil {
			return s
		}
		s = s[:idx] + val + s[idx+tok.length:]
	}
	return s
}
