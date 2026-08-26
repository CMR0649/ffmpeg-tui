// Package ui 实现 ffmpeg-tui 的界面层：顶部标签栏、内容区与底部按键提示。
package ui

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
	"github.com/rivo/tview"
)

// Version 是当前演示版本号。
const Version = "beta0.1"

// tabNames 定义顶部标签页（顺序即显示顺序）。
var tabNames = []string{"文件", "视频", "音频", "任务", "设置"}

// App 组装整个 TUI 界面。
type App struct {
	tviewApp  *tview.Application
	pages     *tview.Pages
	tabBar    *tview.TextView
	tabRanges []struct{ start, end int } // 各标签在标签栏中的横向范围（鼠标点击用）
	current   int
}

// NewApp 创建并初始化应用。
func NewApp() *App {
	pages := tview.NewPages()
	for i, name := range tabNames {
		pages.AddPage(name, buildPage(i), true, i == 0)
	}

	// 文本区显式样式：白字黑底。
	// 若不设置，tview 会用 ColorDefault 背景填充文本区，浅色主题终端下
	// 整行会跟随终端默认背景（如白色），导致标签栏显示异常。
	textStyle := tcell.StyleDefault.
		Foreground(tview.Styles.PrimaryTextColor).
		Background(tview.Styles.PrimitiveBackgroundColor)

	tabBar := tview.NewTextView()
	tabBar.SetDynamicColors(true)
	tabBar.SetWordWrap(false)
	tabBar.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	tabBar.SetTextStyle(textStyle)

	version := tview.NewTextView()
	version.SetDynamicColors(true)
	version.SetTextAlign(tview.AlignRight)
	version.SetText(fmt.Sprintf(" [yellow]v%s[-]", Version))
	version.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	version.SetTextStyle(textStyle)

	topBar := tview.NewFlex()
	topBar.AddItem(tabBar, 0, 1, false)
	topBar.AddItem(version, len("v"+Version)+3, 0, false)

	footer := tview.NewTextView()
	footer.SetTextAlign(tview.AlignCenter)
	footer.SetText(" A/D：切换    方向键：选择 ")
	footer.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	footer.SetTextStyle(textStyle)

	root := tview.NewFlex().SetDirection(tview.FlexRow)
	root.AddItem(topBar, 1, 0, false)
	root.AddItem(pages, 0, 1, true)
	root.AddItem(footer, 1, 0, false)

	a := &App{
		tviewApp: tview.NewApplication(),
		pages:    pages,
		tabBar:   tabBar,
	}
	a.tviewApp.SetRoot(root, true)
	a.tviewApp.EnableMouse(true)
	a.tviewApp.SetInputCapture(a.handleKeys)
	a.tabBar.SetMouseCapture(a.handleTabBarClick)
	a.renderTabBar()
	return a
}

// Run 启动事件循环。
func (a *App) Run() {
	if err := a.tviewApp.Run(); err != nil {
		panic(err)
	}
}

// renderTabBar 重绘标签栏：当前标签以白色背景高亮（黑色文字），其余为默认样式。
func (a *App) renderTabBar() {
	var sb strings.Builder
	x := 0
	a.tabRanges = a.tabRanges[:0]
	for i, name := range tabNames {
		// 转义方括号，避免被当作颜色标签（中文标签名不会命中，转义仅为稳妥）。
		label := tview.Escape("[" + name + "]")
		width := runewidth.StringWidth(label)
		if i == a.current {
			sb.WriteString("[black:white]  ")
			sb.WriteString(label)
			// 注意：tview 中 [-] 只重置前景色，背景色会延续导致整行变白，
			// 必须用 [-:-:-] 同时重置前景/背景/属性。
			sb.WriteString("  [-:-:-]")
			width += 4
		} else {
			// 未选中标签显式使用白字黑底，不依赖重置标签，保证各种终端一致。
			sb.WriteString("[white:black]   ")
			sb.WriteString(label)
			sb.WriteString("   ")
			width += 6
		}
		a.tabRanges = append(a.tabRanges, struct{ start, end int }{x, x + width})
		x += width
	}
	a.tabBar.SetText(sb.String())
}

// switchTab 切换到第 i 个标签（越界自动循环）。
func (a *App) switchTab(i int) {
	n := len(tabNames)
	if n == 0 {
		return
	}
	a.current = ((i % n) + n) % n
	a.pages.SwitchToPage(tabNames[a.current])
	a.renderTabBar()
	a.tviewApp.SetFocus(a.pages.GetPage(tabNames[a.current]))
}

// handleKeys 处理全局按键：A/D 切换标签页，Q/Esc/Ctrl+C 退出。
// 方向键由当前页面的列表组件自行处理（用于选择项目）。
func (a *App) handleKeys(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyCtrlC, tcell.KeyEscape:
		a.tviewApp.Stop()
		return nil
	case tcell.KeyRune:
		switch event.Rune() {
		case 'a', 'A':
			a.switchTab(a.current - 1)
			return nil
		case 'd', 'D':
			a.switchTab(a.current + 1)
			return nil
		case 'q', 'Q':
			a.tviewApp.Stop()
			return nil
		}
	}
	return event
}

// handleTabBarClick 支持鼠标点击标签栏切换标签。
func (a *App) handleTabBarClick(action tview.MouseAction, event *tcell.EventMouse) (tview.MouseAction, *tcell.EventMouse) {
	if action != tview.MouseLeftClick {
		return action, event
	}
	x, y := event.Position()
	if y != 0 { // 标签栏位于屏幕第一行
		return action, event
	}
	for i, r := range a.tabRanges {
		if x >= r.start && x < r.end {
			a.switchTab(i)
			break
		}
	}
	return 0, nil
}
