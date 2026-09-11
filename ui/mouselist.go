package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// selectOnSecondClick 使列表需要点击两次：第一次点击仅把光标移到该项（选中），
// 再次点击同一项才触发该项动作
func (a *App) selectOnSecondClick(l *tview.List) {
	l.SetMouseCapture(func(action tview.MouseAction, event *tcell.EventMouse) (tview.MouseAction, *tcell.EventMouse) {
		if action == tview.MouseLeftClick {
			x, y := event.Position()
			if idx := listIndexAtPoint(l, x, y); idx >= 0 && idx != l.GetCurrentItem() {
				a.tviewApp.SetFocus(l)
				l.SetCurrentItem(idx)
				return tview.MouseConsumed, nil
			}
		}
		return action, event
	})
}

// listIndexAtPoint 返回鼠标位置对应的列表项下标（-1 表示不在任何项上）；
// 列表默认显示次要文本，每项占两行
func listIndexAtPoint(l *tview.List, x, y int) int {
	rectX, rectY, width, height := l.GetInnerRect()
	if x < rectX || x >= rectX+width || y < rectY || y >= rectY+height {
		return -1
	}
	offset, _ := l.GetOffset()
	idx := (y-rectY)/2 + offset
	if idx < 0 || idx >= l.GetItemCount() {
		return -1
	}
	return idx
}
