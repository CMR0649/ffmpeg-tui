// Package ui 实现 ffmpeg-tui 的界面层：顶部标签栏、内容区与底部按键提示。
package ui

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
	"github.com/rivo/tview"
)

// Version 是当前版本号。
const Version = "beta0.2"

// tabNames 定义顶部标签页（顺序即显示顺序）。
var tabNames = []string{"文件", "视频", "音频", "任务", "设置"}

// App 组装整个 TUI 界面。
type App struct {
	tviewApp  *tview.Application
	pages     *tview.Pages // 标签页内容
	rootPages *tview.Pages // 根页面：main（主界面）+ dialog（模态对话框）
	tabBar    *tview.TextView
	tabRanges []struct{ start, end int } // 各标签在标签栏中的横向范围（鼠标点击用）
	current   int

	// 文件页状态
	filesList          *tview.List
	fileContainerLabel *tview.TextView
	outputContainer    string
	fileBarFocusables  []tview.Primitive // 文件页 Tab 循环：文件列表 / 添加文件 / 设置输出容器
	fileBarButtons     []tview.Primitive // 文件页底部横向按钮组：[添加文件] [设置输出容器]（左右键切换）

	// 对话框状态
	dialogOpen       bool
	dialogFocusables []tview.Primitive // 对话框内可聚焦组件（Tab 循环）
	dialogFocusIndex int
	dialogButtons    []tview.Primitive // 对话框内横向按钮组：[确定] [取消]（左右键切换）
}

// NewApp 创建并初始化应用。
func NewApp() *App {
	// 统一使用单线边框：tview 默认在获得焦点时切换为双线边框，
	// 与界面设计（┌─┐ 样式）不符，这里固定为单线。
	tview.Borders.HorizontalFocus = tview.BoxDrawingsLightHorizontal
	tview.Borders.VerticalFocus = tview.BoxDrawingsLightVertical
	tview.Borders.TopLeftFocus = tview.BoxDrawingsLightDownAndRight
	tview.Borders.TopRightFocus = tview.BoxDrawingsLightDownAndLeft
	tview.Borders.BottomLeftFocus = tview.BoxDrawingsLightUpAndRight
	tview.Borders.BottomRightFocus = tview.BoxDrawingsLightUpAndLeft

	a := &App{
		tviewApp:        tview.NewApplication(),
		outputContainer: "mp4",
	}

	// 标签页内容。
	a.pages = tview.NewPages()
	for i, name := range tabNames {
		var content tview.Primitive
		if i == 0 {
			content = a.buildFilesPage() // 文件页（含底部操作栏）
		} else {
			content = buildPage(i)
		}
		a.pages.AddPage(name, content, true, i == 0)
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
	a.tabBar = tabBar

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
	footer.SetText(" A/D：切换    方向键：选择    Delete：移除 ")
	footer.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	footer.SetTextStyle(textStyle)

	mainRoot := tview.NewFlex().SetDirection(tview.FlexRow)
	mainRoot.AddItem(topBar, 1, 0, false)
	mainRoot.AddItem(a.pages, 0, 1, true)
	mainRoot.AddItem(footer, 1, 0, false)

	// 根页面：主界面 + 模态对话框层。
	a.rootPages = tview.NewPages()
	a.rootPages.AddPage("main", mainRoot, true, true)

	a.tviewApp.SetRoot(a.rootPages, true)
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

// fileBarFocusNext / fileBarFocusPrev 在文件页组件（文件列表/添加文件/设置输出容器）间循环焦点。
func (a *App) fileBarFocusNext() {
	n := len(a.fileBarFocusables)
	if n == 0 {
		return
	}
	a.tviewApp.SetFocus(a.fileBarFocusables[(a.fileBarCurrentIndex()+1)%n])
}

func (a *App) fileBarFocusPrev() {
	n := len(a.fileBarFocusables)
	if n == 0 {
		return
	}
	a.tviewApp.SetFocus(a.fileBarFocusables[(a.fileBarCurrentIndex()-1+n)%n])
}

// fileBarCurrentIndex 返回当前焦点在文件页组件列表中的下标（不在列表中时视为 0）。
func (a *App) fileBarCurrentIndex() int {
	cur := a.tviewApp.GetFocus()
	for i, p := range a.fileBarFocusables {
		if p == cur {
			return i
		}
	}
	return 0
}

// moveHorizontalFocus 处理左/右方向键：当焦点位于横向排列的按钮组
// （文件页底部 [添加文件] [设置输出容器]，或对话框 [确定] [取消]）时，
// 在同组按钮间循环切换焦点。返回 true 表示事件已消费。
func (a *App) moveHorizontalFocus(left bool) bool {
	var group []tview.Primitive
	if a.dialogOpen {
		group = a.dialogButtons
	} else if a.current == 0 {
		group = a.fileBarButtons
	}
	if len(group) == 0 {
		return false
	}
	cur := a.tviewApp.GetFocus()
	idx := -1
	for i, p := range group {
		if p == cur {
			idx = i
			break
		}
	}
	if idx < 0 {
		return false // 焦点不在按钮组中（如列表），左右键放行给组件
	}
	if left {
		idx = (idx - 1 + len(group)) % len(group)
	} else {
		idx = (idx + 1) % len(group)
	}
	a.tviewApp.SetFocus(group[idx])
	return true
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
// 对话框打开时：Esc 关闭对话框，其余按键交给对话框组件。
func (a *App) handleKeys(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyCtrlC:
		a.tviewApp.Stop()
		return nil
	case tcell.KeyTab:
		if a.dialogOpen {
			a.dialogFocusNext()
			return nil
		}
		if a.current == 0 { // 文件页：Tab 在 文件列表/添加文件/设置输出容器 间循环
			a.fileBarFocusNext()
			return nil
		}
	case tcell.KeyBacktab:
		if a.dialogOpen {
			a.dialogFocusPrev()
			return nil
		}
		if a.current == 0 { // 文件页：Shift-Tab 反向循环
			a.fileBarFocusPrev()
			return nil
		}
	case tcell.KeyLeft, tcell.KeyRight:
		// 左右方向键在横向排列的按钮组（文件页底部 / 对话框 [确定][取消]）间切换焦点。
		if a.moveHorizontalFocus(event.Key() == tcell.KeyLeft) {
			return nil
		}
	case tcell.KeyDelete:
		// 文件界面：Delete 移除当前选中的文件。
		if !a.dialogOpen && a.current == 0 && a.filesList != nil && a.filesList.GetItemCount() > 0 {
			a.filesList.RemoveItem(a.filesList.GetCurrentItem())
			return nil
		}
	case tcell.KeyEscape:
		if a.dialogOpen {
			a.closeDialog()
			return nil
		}
		a.tviewApp.Stop()
		return nil
	case tcell.KeyRune:
		if a.dialogOpen {
			return event // 对话框打开时，A/D 不切换标签，按键交给对话框
		}
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
	if a.dialogOpen { // 对话框打开时不切换标签
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
