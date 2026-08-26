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
const Version = "beta1.1"

// tabNames 定义顶部标签页（顺序即显示顺序）。
var tabNames = []string{"文件", "视频", "音频", "任务", "设置"}

// App 组装整个 TUI 界面。
type App struct {
	tviewApp  *tview.Application
	pages     *tview.Pages // 标签页内容
	rootPages *tview.Pages // 根页面：main（主界面）+ dialog（模态对话框）
	tabBar    *tview.TextView
	footer    *tview.TextView
	tabRanges []struct{ start, end int } // 各标签在标签栏中的横向范围（鼠标点击用）
	current   int

	// 文件页状态
	filesList          *tview.List
	fileContainerLabel *tview.TextView
	outputContainer    string
	fileBarFocusables  []tview.Primitive // 文件页 Tab 循环：文件列表 / 输出目录 / 添加文件 / 设置输出容器
	fileBarButtons     []tview.Primitive // 文件页底部横向按钮组：[添加文件] [设置输出容器]（左右键切换）
	outputDirBtn       *tview.Button     // 输出目录选项行
	files              []string          // 文件页中的文件路径列表（生成任务用）

	// 任务页状态
	tasks             []*Task
	taskList          *tview.List
	taskBarFocusables []tview.Primitive // 任务页 Tab 循环：任务列表 / 添加任务 / 开始 / 清空
	taskBarButtons    []tview.Primitive // 任务页底部横向按钮组（左右键切换）

	// 配置与各选项页
	cfg          *Config
	videoList    *tview.List
	audioList    *tview.List
	settingsList *tview.List

	// 对话框状态
	dialogOpen       bool
	dialogFocusables []tview.Primitive // 对话框内可聚焦组件（Tab 循环）
	dialogFocusIndex int
	dialogButtons    []tview.Primitive // 对话框内横向按钮组：[确定] [取消]（左右键切换）
	optDialog        *optionDialog     // 选项对话框状态（支持 "/" 搜索过滤）
	searching        bool              // 是否正在搜索选项对话框
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
		cfg:             DefaultConfig(),
	}

	// 加载编码器/解码器列表（ffmpeg-encoders.txt / ffmpeg-decoders.txt 或 ffmpeg 命令）。
	loadCodecLists()

	// 加载默认配置（若存在，来自「指定默认配置」）。
	if p := defaultConfigPath(); fileExists(p) {
		_ = a.cfg.LoadJSON(p)
	}

	// 标签页内容。
	a.pages = tview.NewPages()
	for i, name := range tabNames {
		var content tview.Primitive
		if i == 0 {
			content = a.buildFilesPage() // 文件页（含输出目录与底部操作栏）
		} else {
			content = a.buildPage(i)
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
	footer.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	footer.SetTextStyle(textStyle)
	a.footer = footer

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
	a.tviewApp.SetMouseCapture(a.handleMouse)
	a.tabBar.SetMouseCapture(a.handleTabBarClick)
	a.renderTabBar()
	a.renderFooter()
	return a
}

// handleMouse 全局鼠标处理：滚轮在列表上滚动选择选项。
// 标签栏（y=0）的滚轮交给 handleTabBarClick 切换标签。
func (a *App) handleMouse(event *tcell.EventMouse, action tview.MouseAction) (*tcell.EventMouse, tview.MouseAction) {
	if action != tview.MouseScrollUp && action != tview.MouseScrollDown {
		return event, action
	}
	_, y := event.Position()
	if y == 0 {
		return event, action // 标签栏滚轮
	}
	if cur, ok := a.tviewApp.GetFocus().(*tview.List); ok {
		idx := cur.GetCurrentItem()
		if action == tview.MouseScrollUp {
			idx--
		} else {
			idx++
		}
		if idx < 0 {
			idx = 0
		}
		if idx >= cur.GetItemCount() {
			idx = cur.GetItemCount() - 1
		}
		cur.SetCurrentItem(idx)
		return nil, 0
	}
	return event, action
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

// focusOnDialogList 报告当前焦点是否在对话框的选项列表上（选项对话框）。
func (a *App) focusOnDialogList() bool {
	if !a.dialogOpen {
		return false
	}
	_, ok := a.tviewApp.GetFocus().(*tview.List)
	return ok
}

// moveHorizontalFocus 处理左/右方向键：当焦点位于横向排列的按钮组
// （页面底部按钮组，或对话框 [确定] [取消]）时，在同组按钮间循环切换焦点。
// 返回 true 表示事件已消费。
func (a *App) moveHorizontalFocus(left bool) bool {
	var group []tview.Primitive
	if a.dialogOpen {
		group = a.dialogButtons
	} else {
		group = a.pageButtons()
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
	a.renderFooter()
	a.tviewApp.SetFocus(a.pages.GetPage(tabNames[a.current]))
}

// renderFooter 刷新底部按键提示：Delete：移除 仅在文件页显示。
func (a *App) renderFooter() {
	if a.current == 0 {
		a.footer.SetText(" A/D：切换    方向键：选择    Delete：移除 ")
	} else {
		a.footer.SetText(" A/D：切换    方向键：选择 ")
	}
}

// refreshAllPages 加载配置后刷新所有选项页。
func (a *App) refreshAllPages() {
	if a.videoList != nil {
		a.refreshVideoPage()
	}
	if a.audioList != nil {
		a.refreshAudioPage()
	}
	if a.settingsList != nil {
		a.refreshSettingsPage()
	}
	a.updateOutputDirButton()
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
		if n := a.pageFocusables(); len(n) > 0 {
			a.pageFocusNext()
			return nil
		}
	case tcell.KeyBacktab:
		if a.dialogOpen {
			a.dialogFocusPrev()
			return nil
		}
		if len(a.pageFocusables()) > 0 {
			a.pageFocusPrev()
			return nil
		}
	case tcell.KeyLeft, tcell.KeyRight:
		// 左/右方向键在横向排列的选项中切换焦点：
		// 对话框内 [确定]/[取消]、页面底部按钮组；
		// 焦点在页面列表时，→ 进入第一个按钮、← 进入最后一个按钮。
		left := event.Key() == tcell.KeyLeft
		if a.dialogOpen {
			if a.moveHorizontalFocus(left) {
				return nil
			}
			// 焦点在对话框选项列表时，左右键进入 [确定]/[取消] 按钮组
			// （→ 到 [确定]，← 到 [取消]；输入框内左右键仍为光标移动）。
			if a.focusOnDialogList() {
				if left {
					a.tviewApp.SetFocus(a.dialogButtons[len(a.dialogButtons)-1])
				} else {
					a.tviewApp.SetFocus(a.dialogButtons[0])
				}
				return nil
			}
			return event
		}
		if btns := a.pageButtons(); len(btns) > 0 {
			if a.moveHorizontalFocus(left) {
				return nil
			}
			if a.focusOnPageList() {
				if left {
					a.tviewApp.SetFocus(btns[len(btns)-1])
				} else {
					a.tviewApp.SetFocus(btns[0])
				}
				return nil
			}
		}
	case tcell.KeyUp, tcell.KeyDown:
		up := event.Key() == tcell.KeyUp
		if a.dialogOpen {
			// 输入对话框：↑/↓ 在输入框与 [确定]/[取消] 之间切换焦点；
			// 选项对话框（焦点在列表）时上下键仍用于选择选项。
			if a.dialogInputOrButtonFocused() {
				if up {
					a.dialogFocusPrev()
				} else {
					a.dialogFocusNext()
				}
				return nil
			}
			return event
		}
		// 页面：焦点在底部按钮时，↑/↓ 返回页面列表。
		if len(a.pageButtons()) > 0 && a.focusOnPageButtons() {
			a.tviewApp.SetFocus(a.pageList())
			return nil
		}
	case tcell.KeyDelete:
		// 文件界面：Delete 移除当前选中的文件（并同步文件列表数据）。
		if !a.dialogOpen && a.current == 0 && a.filesList != nil && a.filesList.GetItemCount() > 0 {
			idx := a.filesList.GetCurrentItem()
			a.filesList.RemoveItem(idx)
			if idx >= 0 && idx < len(a.files) {
				a.files = append(a.files[:idx], a.files[idx+1:]...)
			}
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
		// "/" 在选项对话框中打开搜索框过滤选项（如编码器/解码器列表）。
		if event.Rune() == '/' && a.dialogOpen && a.optDialog != nil && a.focusOnDialogList() {
			a.startOptionSearch()
			return nil
		}
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

// pageFocusables 当前页面的 Tab 焦点循环列表（文件页 / 任务页）。
func (a *App) pageFocusables() []tview.Primitive {
	switch a.current {
	case 0:
		return a.fileBarFocusables
	case 3:
		return a.taskBarFocusables
	}
	return nil
}

// pageButtons 当前页面的底部横向按钮组。
func (a *App) pageButtons() []tview.Primitive {
	switch a.current {
	case 0:
		return a.fileBarButtons
	case 3:
		return a.taskBarButtons
	}
	return nil
}

// pageList 当前页面的主列表。
func (a *App) pageList() tview.Primitive {
	switch a.current {
	case 0:
		return a.filesList
	case 3:
		return a.taskList
	}
	return nil
}

// pageFocusNext / pageFocusPrev 在当前页面的 Tab 焦点循环中移动。
func (a *App) pageFocusNext() {
	if p := a.pageList(); p == nil {
		return
	}
	items := a.pageFocusables()
	if len(items) == 0 {
		return
	}
	a.tviewApp.SetFocus(items[(a.pageCurrentIndex()+1)%len(items)])
}

func (a *App) pageFocusPrev() {
	items := a.pageFocusables()
	if len(items) == 0 {
		return
	}
	a.tviewApp.SetFocus(items[(a.pageCurrentIndex()-1+len(items))%len(items)])
}

// pageCurrentIndex 当前焦点在当前页面焦点循环中的下标。
func (a *App) pageCurrentIndex() int {
	cur := a.tviewApp.GetFocus()
	for i, p := range a.pageFocusables() {
		if p == cur {
			return i
		}
	}
	return 0
}

// focusOnPageList 报告当前焦点是否在当前页面的主列表上。
func (a *App) focusOnPageList() bool {
	cur := a.tviewApp.GetFocus()
	for _, p := range a.pageFocusables() {
		if p == cur {
			return p == a.pageList()
		}
	}
	return false
}

// focusOnPageButtons 报告当前焦点是否在当前页面底部按钮组上。
func (a *App) focusOnPageButtons() bool {
	cur := a.tviewApp.GetFocus()
	for _, p := range a.pageButtons() {
		if p == cur {
			return true
		}
	}
	return false
}

// dialogInputOrButtonFocused 报告当前焦点是否在输入框或对话框按钮上
// （输入对话框场景，用于 ↑/↓ 焦点切换）。
func (a *App) dialogInputOrButtonFocused() bool {
	cur := a.tviewApp.GetFocus()
	if _, ok := cur.(*tview.InputField); ok {
		return true
	}
	for _, p := range a.dialogButtons {
		if p == cur {
			return true
		}
	}
	return false
}

// handleTabBarClick 支持鼠标点击标签栏切换标签，以及滚轮在标签间切换。
// 在按下（MouseLeftDown）时即切换：tview 的 MouseLeftClick 要求按下与释放
// 位于同一单元格，真实鼠标点击的微小位移会导致单击事件不产生（表现为
// 需要双击才切换），因此这里直接响应按下事件。
func (a *App) handleTabBarClick(action tview.MouseAction, event *tcell.EventMouse) (tview.MouseAction, *tcell.EventMouse) {
	if a.dialogOpen { // 对话框打开时不切换标签
		return action, event
	}
	x, y := event.Position()
	switch action {
	case tview.MouseLeftDown:
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
	case tview.MouseScrollUp, tview.MouseScrollDown:
		if y != 0 {
			return action, event
		}
		if action == tview.MouseScrollUp {
			a.switchTab(a.current - 1)
		} else {
			a.switchTab(a.current + 1)
		}
		return 0, nil
	}
	return action, event
}
