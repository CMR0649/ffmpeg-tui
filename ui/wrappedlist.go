package ui

import (
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
	"github.com/rivo/tview"
)

// wrappedList 可换行的选项列表：每项按宽度自动折行，折行后的各行紧贴显示，
// 选中项整行高亮；支持上下键、PgUp/PgDn、Home/End、回车确认与鼠标点击。
// 用于编码器/滤镜等「名称 + 长说明」的选项弹窗，避免长文本被横向截断。
type wrappedList struct {
	*tview.Box

	items      []string
	selected   int             // 当前选中项下标
	topLine    int             // 首个可见行（跨项累计的行号）
	onSelected func(index int) // 回车或再次点击选中项时调用
	lines      []listLine      // 绘制后的扁平行（供鼠标定位）
	width      int             // 上次绘制时使用的宽度
}

// listLine 列表中的一行：item < 0 表示项与项之间的空行
type listLine struct {
	item int
	text string
}

// newWrappedList 创建一个可换行的选项列表
func newWrappedList() *wrappedList {
	return &wrappedList{Box: tview.NewBox()}
}

// SetItems 设置列表项并回到第一项
func (w *wrappedList) SetItems(items []string) {
	w.items = items
	w.selected = 0
	w.topLine = 0
	w.lines = nil
}

// GetSelected 返回当前选中项下标
func (w *wrappedList) GetSelected() int { return w.selected }

// SetSelected 设置选中项（越界自动收敛）
func (w *wrappedList) SetSelected(i int) {
	if i < 0 {
		i = 0
	}
	if i >= len(w.items) {
		i = len(w.items) - 1
	}
	w.selected = i
}

// SetSelectedFunc 设置确认回调（回车或再次点击选中项时触发）
func (w *wrappedList) SetSelectedFunc(fn func(index int)) { w.onSelected = fn }

// activate 触发确认回调
func (w *wrappedList) activate() {
	if w.onSelected != nil && w.selected >= 0 && w.selected < len(w.items) {
		w.onSelected(w.selected)
	}
}

// Draw 绘制列表：先按当前宽度折行，再按滚动位置输出可见行
func (w *wrappedList) Draw(screen tcell.Screen) {
	w.Box.DrawForSubclass(screen, w)
	x, y, width, height := w.GetInnerRect()
	if width <= 0 || height <= 0 {
		return
	}
	w.width = width
	lines := w.buildLines(width)
	w.lines = lines
	total := len(lines)

	// 选中项所在行范围
	selStart, selEnd := -1, -1
	if len(w.items) > 0 {
		if w.selected < 0 {
			w.selected = 0
		}
		if w.selected >= len(w.items) {
			w.selected = len(w.items) - 1
		}
		for i, ln := range lines {
			if ln.item == w.selected {
				if selStart < 0 {
					selStart = i
				}
				selEnd = i + 1
			}
		}
	}
	if selStart < 0 {
		selStart, selEnd = 0, 0
	}
	// 滚动，保证选中项完整可见
	if selStart < w.topLine {
		w.topLine = selStart
	}
	if selEnd > w.topLine+height {
		w.topLine = selEnd - height
	}
	if maxTop := total - height; w.topLine > maxTop {
		w.topLine = maxTop
	}
	if w.topLine < 0 {
		w.topLine = 0
	}

	normal := tcell.StyleDefault.
		Foreground(tview.Styles.PrimaryTextColor).
		Background(tview.Styles.PrimitiveBackgroundColor)
	chosen := tcell.StyleDefault.
		Foreground(tview.Styles.PrimitiveBackgroundColor).
		Background(tview.Styles.PrimaryTextColor)

	for i, ln := range lines {
		if i < w.topLine || i >= w.topLine+height {
			continue
		}
		style := normal
		if ln.item == w.selected && w.HasFocus() {
			style = chosen
		}
		drawListLine(screen, ln.text, x, y+i-w.topLine, width, style)
	}
}

// buildLines 把各项折行后展开为扁平行：项内各行紧贴，项与项之间插入一个空行
func (w *wrappedList) buildLines(width int) []listLine {
	out := make([]listLine, 0, len(w.items)*2)
	for i, it := range w.items {
		if i > 0 {
			out = append(out, listLine{item: -1})
		}
		for _, text := range wrapText(it, width) {
			out = append(out, listLine{item: i, text: text})
		}
	}
	return out
}

// drawListLine 以指定样式输出一行，先把整行填成空格以保证选中行整行高亮
func drawListLine(screen tcell.Screen, text string, x, y, width int, style tcell.Style) {
	for col := 0; col < width; col++ {
		screen.SetContent(x+col, y, ' ', nil, style)
	}
	col := 0
	for _, r := range text {
		rw := runewidth.RuneWidth(r)
		if rw <= 0 {
			continue
		}
		if col+rw > width {
			break
		}
		screen.SetContent(x+col, y, r, nil, style)
		col += rw
	}
}

// InputHandler 处理上下键、翻页、Home/End 与回车
func (w *wrappedList) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	return w.WrapInputHandler(func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
		switch event.Key() {
		case tcell.KeyUp:
			w.move(-1)
		case tcell.KeyDown:
			w.move(1)
		case tcell.KeyPgUp:
			w.movePage(-1)
		case tcell.KeyPgDn:
			w.movePage(1)
		case tcell.KeyHome:
			w.SetSelected(0)
		case tcell.KeyEnd:
			w.SetSelected(len(w.items) - 1)
		case tcell.KeyEnter:
			w.activate()
		}
	})
}

// move 上下移动选中项
func (w *wrappedList) move(delta int) {
	w.SetSelected(w.selected + delta)
}

// movePage 按可视高度翻页
func (w *wrappedList) movePage(dir int) {
	_, _, _, height := w.GetInnerRect()
	if height < 1 {
		height = 1
	}
	w.SetSelected(w.selected + dir*height/2)
}

// MouseHandler 处理鼠标点击：点击非选中项先选中，点击选中项则确认
func (w *wrappedList) MouseHandler() func(action tview.MouseAction, event *tcell.EventMouse, setFocus func(p tview.Primitive)) (bool, tview.Primitive) {
	return w.WrapMouseHandler(func(action tview.MouseAction, event *tcell.EventMouse, setFocus func(p tview.Primitive)) (bool, tview.Primitive) {
		// 滚轮：上下移动选中项（全局鼠标捕获通常已处理，此处兜底）
		switch action {
		case tview.MouseScrollUp:
			w.SetSelected(w.selected - 1)
			return true, nil
		case tview.MouseScrollDown:
			w.SetSelected(w.selected + 1)
			return true, nil
		}
		if action != tview.MouseLeftClick {
			return false, nil
		}
		x, y := event.Position()
		ix, iy, width, height := w.GetInnerRect()
		if x < ix || x >= ix+width || y < iy || y >= iy+height {
			return false, nil
		}
		setFocus(w)
		if idx := w.itemAtLine(w.topLine + y - iy); idx >= 0 {
			if idx == w.selected {
				w.activate()
			} else {
				w.SetSelected(idx)
			}
		}
		return true, nil
	})
}

// itemAtLine 返回扁平行号对应的项下标（项间空行归到其上一项，-1 表示无）
func (w *wrappedList) itemAtLine(line int) int {
	if line < 0 || line >= len(w.lines) {
		return -1
	}
	if it := w.lines[line].item; it >= 0 {
		return it
	}
	for i := line - 1; i >= 0; i-- {
		if w.lines[i].item >= 0 {
			return w.lines[i].item
		}
	}
	return -1
}

// wrapText 把文本按宽度折成多行：优先在空格处断行，超长单词与中文按字符折行
func wrapText(text string, width int) []string {
	if width < 1 {
		width = 1
	}
	if text == "" {
		return []string{""}
	}
	var lines []string
	var cur []rune
	curWidth := 0
	for _, r := range text {
		rw := runewidth.RuneWidth(r)
		if rw <= 0 {
			rw = 1
		}
		if curWidth+rw <= width {
			cur = append(cur, r)
			curWidth += rw
			continue
		}
		// 需要折行：尝试在最后一个空格处断开，避免截断单词
		if idx := lastSpaceIndex(cur); idx > 0 {
			lines = append(lines, strings.TrimRight(string(cur[:idx]), " "))
			cur = append([]rune{}, cur[idx+1:]...)
			curWidth = runewidth.StringWidth(string(cur))
		} else {
			lines = append(lines, string(cur))
			cur = nil
			curWidth = 0
		}
		if curWidth+rw > width { // 单个字符宽于整行
			lines = append(lines, string(cur))
			cur = nil
			curWidth = 0
		}
		cur = append(cur, r)
		curWidth += rw
	}
	lines = append(lines, string(cur))
	return lines
}

// lastSpaceIndex 返回 runes 中最后一个空格的下标（无则 -1）
func lastSpaceIndex(runes []rune) int {
	for i := len(runes) - 1; i >= 0; i-- {
		if runes[i] == ' ' {
			return i
		}
	}
	return -1
}
