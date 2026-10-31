package ui

import (
	"strings"
	"testing"

	"github.com/mattn/go-runewidth"
)

func TestDialogTextHeight(t *testing.T) {
	cases := []struct {
		text    string
		w, want int
	}{
		{"abc", 10, 1},
		{"a\nb", 10, 2},
		{"", 10, 1},
		{strings.Repeat("x", 25), 10, 3},
		{"hello world foo bar", 10, 3},
		{strings.Repeat("中", 10), 10, 2},
		{"已加载配置：\n/tmp/a/b/c.json", 20, 2},
	}
	for _, c := range cases {
		if got := dialogTextHeight(c.text, c.w); got != c.want {
			t.Errorf("dialogTextHeight(%q,%d)=%d want %d", c.text, c.w, got, c.want)
		}
	}
	a := &App{}
	if a.messageDialogHeight("a", 60) != 6 {
		t.Errorf("1 line=%d", a.messageDialogHeight("a", 60))
	}
	if a.messageDialogHeight("a\nb\nc", 60) != 8 {
		t.Errorf("3 lines=%d", a.messageDialogHeight("a\nb\nc", 60))
	}
}
func TestClampDialogSize(t *testing.T) {
	// 25 行终端：最高 19 行，上下各留 3 行
	w, h := clampDialogSize(120, 25, 56, 999)
	if w != 56 || h != 19 {
		t.Errorf("got %dx%d want 56x19", w, h)
	}
	// 宽不超过 屏宽-2
	w, h = clampDialogSize(40, 25, 999, 10)
	if w != 38 || h != 10 {
		t.Errorf("got %dx%d want 38x10", w, h)
	}
	// 最小尺寸
	w, h = clampDialogSize(10, 6, 5, 2)
	if w != 10 || h != 4 {
		t.Errorf("got %dx%d want 10x4", w, h)
	}
	// 正常尺寸不受影响
	w, h = clampDialogSize(120, 25, 60, 8)
	if w != 60 || h != 8 {
		t.Errorf("got %dx%d want 60x8", w, h)
	}
}

// TestWrapText 校验选项列表的折行：每行不超宽、折行后各行紧贴（无空行）
func TestWrapText(t *testing.T) {
	cases := []struct {
		text  string
		width int
		want  []string
	}{
		{"short", 20, []string{"short"}},
		{"", 20, []string{""}},
		{"alphaextract  Extract an alpha channel as a grayscale image component.", 40,
			[]string{"alphaextract  Extract an alpha channel", "as a grayscale image component."}},
		{"中文测试文本", 4, []string{"中文", "测试", "文本"}},
		{"abcdefghij", 4, []string{"abcd", "efgh", "ij"}},
	}
	for _, c := range cases {
		got := wrapText(c.text, c.width)
		if len(got) != len(c.want) {
			t.Errorf("wrapText(%q,%d)=%q want %q", c.text, c.width, got, c.want)
			continue
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("wrapText(%q,%d)[%d]=%q want %q", c.text, c.width, i, got[i], c.want[i])
			}
		}
		for _, line := range got {
			if w := runewidthWidth(line); w > c.width {
				t.Errorf("line %q width %d > %d", line, w, c.width)
			}
			if strings.TrimSpace(line) == "" && c.text != "" {
				t.Errorf("unexpected empty line in %q", got)
			}
		}
	}
}

// TestWrappedListItemAtLine 校验鼠标行号到项下标的映射（项间空行归到上一项）
func TestWrappedListItemAtLine(t *testing.T) {
	w := newWrappedList()
	w.SetItems([]string{"one", "two"})
	// 展开后：0=one, 1=空行, 2=two, 3=two-cont
	w.lines = []listLine{{item: 0, text: "one"}, {item: -1}, {item: 1, text: "two"}, {item: 1, text: "two-cont"}}
	cases := map[int]int{0: 0, 1: 0, 2: 1, 3: 1, 4: -1, -1: -1}
	for line, want := range cases {
		if got := w.itemAtLine(line); got != want {
			t.Errorf("itemAtLine(%d)=%d want %d", line, got, want)
		}
	}
}

// TestWrappedListBuildLines 校验项内紧贴、项间空一行
func TestWrappedListBuildLines(t *testing.T) {
	w := newWrappedList()
	w.SetItems([]string{"alphaextract  Extract an alpha channel as a grayscale image component.", "short"})
	lines := w.buildLines(40)
	var got []string
	for _, ln := range lines {
		if ln.item < 0 {
			got = append(got, "<blank>")
		} else {
			got = append(got, ln.text)
		}
	}
	want := []string{"alphaextract  Extract an alpha channel", "as a grayscale image component.", "<blank>", "short"}
	if len(got) != len(want) {
		t.Fatalf("buildLines=%q want %q", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("line %d=%q want %q", i, got[i], want[i])
		}
	}
}

// runewidthWidth 返回字符串显示宽度
func runewidthWidth(s string) int {
	w := 0
	for _, r := range s {
		w += runewidth.RuneWidth(r)
	}
	return w
}
