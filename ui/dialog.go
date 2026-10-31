package ui

import (
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
	"github.com/rivo/tview"
)

// ---------- 对话框核心 ----------

func (a *App) showDialog(content tview.Primitive, focusables []tview.Primitive, initial tview.Primitive) {
	a.rootPages.RemovePage("dialog")
	a.rootPages.AddPage("dialog", content, true, true)
	a.dialogOpen = true
	a.dialogFocusables = focusables
	a.dialogFocusIndex = 0
	for i, p := range focusables {
		if p == initial {
			a.dialogFocusIndex = i
			break
		}
	}
	a.tviewApp.SetFocus(initial)
}

// dialogFocusNext / dialogFocusPrev 在对话框内循环移动焦点（Tab / Shift-Tab）
func (a *App) dialogFocusNext() {
	n := len(a.dialogFocusables)
	if n == 0 {
		return
	}
	a.dialogFocusIndex = (a.dialogFocusIndex + 1) % n
	a.tviewApp.SetFocus(a.dialogFocusables[a.dialogFocusIndex])
}

func (a *App) dialogFocusPrev() {
	n := len(a.dialogFocusables)
	if n == 0 {
		return
	}
	a.dialogFocusIndex = (a.dialogFocusIndex - 1 + n) % n
	a.tviewApp.SetFocus(a.dialogFocusables[a.dialogFocusIndex])
}

// closeDialog 关闭当前对话框，并把焦点还给当前标签页内容
// 若正处在选项搜索中，则重建选项对话框（取消搜索）
func (a *App) closeDialog() {
	if !a.dialogOpen {
		return
	}
	a.filterForm = nil // 关闭任何对话框时丢弃参数表单状态
	a.dialogOpen = false
	a.rootPages.RemovePage("dialog")
	a.tviewApp.SetFocus(a.pages.GetPage(tabKeys[a.current]))
}

// screenSize 返回界面可用尺寸（宽、高），未初始化时给默认值
func (a *App) screenSize() (int, int) {
	_, _, w, h := a.rootPages.GetRect()
	if w <= 0 || h <= 0 {
		return 80, 24
	}
	return w, h
}

// clampDialogSize 把弹窗宽高限制在界面内：宽度最多留 2 列余量，
// 高度最多为界面高度减 6，保证弹窗上方与下方各至少空出 3 行
func clampDialogSize(sw, sh, width, height int) (int, int) {
	if width > sw-2 {
		width = sw - 2
	}
	if maxH := sh - 6; height > maxH {
		height = maxH
	}
	if width < 10 {
		width = 10
	}
	if height < 4 {
		height = 4
	}
	return width, height
}

// centeredGrid 把 box 居中放在界面内（宽高经 clampDialogSize 限制）
func (a *App) centeredGrid(box tview.Primitive, width, height int) *tview.Grid {
	sw, sh := a.screenSize()
	width, height = clampDialogSize(sw, sh, width, height)
	grid := tview.NewGrid()
	grid.SetColumns(0, width, 0)
	grid.SetRows(0, height, 0)
	grid.AddItem(box, 1, 1, 1, 1, 0, 0, true)
	grid.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	return grid
}

// dialogTextHeight 返回文本在指定宽度内占用的行数（含换行与自动折行），
// 按「空格断行 + 超长单词按字符折行」估算，结果不小于实际行数，避免文字被裁掉
func dialogTextHeight(text string, width int) int {
	if width < 1 {
		width = 1
	}
	total := 0
	for _, line := range strings.Split(text, "\n") {
		total += wrappedLineCount(line, width)
	}
	if total < 1 {
		total = 1
	}
	return total
}

// wrappedLineCount 估算一段（无换行符的）文本折行后的行数
func wrappedLineCount(line string, width int) int {
	if width < 1 {
		width = 1
	}
	if runewidth.StringWidth(line) <= width {
		return 1
	}
	lines := 1
	cur := -1 // 当前行已用宽度（-1 表示行首，无前导空格）
	for _, word := range strings.Split(line, " ") {
		w := runewidth.StringWidth(word)
		if cur < 0 {
			for w > width {
				lines++
				w -= width
			}
			cur = w
			continue
		}
		if cur+1+w <= width {
			cur += 1 + w
			continue
		}
		lines++
		for w > width {
			lines++
			w -= width
		}
		cur = w
	}
	return lines
}

// messageDialogHeight 提示类弹窗高度：文本行数 + 按钮与留白（3）+ 边框（2）
func (a *App) messageDialogHeight(text string, width int) int {
	return dialogTextHeight(text, width-2) + 5
}

func (a *App) buildDialogBox(title string, body tview.Primitive, width, height int, buttons ...*tview.Button) *tview.Grid {
	btnRow := tview.NewFlex()
	btnRow.AddItem(nil, 0, 1, false)
	for i, b := range buttons {
		btnRow.AddItem(b, 0, 1, false)
		if i < len(buttons)-1 {
			btnRow.AddItem(nil, 2, 0, false)
		}
	}
	btnRow.AddItem(nil, 0, 1, false)

	box := tview.NewFlex().SetDirection(tview.FlexRow)
	box.AddItem(body, 0, 1, true)
	box.AddItem(nil, 1, 0, false)
	box.AddItem(btnRow, 1, 0, false)
	box.AddItem(nil, 1, 0, false)
	box.SetBorder(true)
	box.SetTitle("┤ " + title + " ├")
	box.SetTitleAlign(tview.AlignLeft)
	return a.centeredGrid(box, width, height)
}

type optionDialog struct {
	title    string
	options  []string
	confirm  func(int)
	filter   string
	filtered []int
	list     *wrappedList
}

// showOptionDialog 显示选项选择对话框；confirm(index) 在确定后调用
// 打开后按 "/" 打开搜索框过滤选项
func (a *App) showOptionDialog(title string, options []string, confirm func(int)) {
	a.optDialog = &optionDialog{title: title, options: options, confirm: confirm}
	a.buildOptionDialog()
}

// newOptionList 创建选项对话框的可换行列表（确认后应用选中项）
func (a *App) newOptionList(d *optionDialog) *wrappedList {
	list := newWrappedList()
	list.SetItems(a.optionListTexts(d))
	list.SetSelectedFunc(func(ci int) {
		a.closeDialog()
		if ci >= 0 && ci < len(d.filtered) {
			d.confirm(d.filtered[ci])
		}
	})
	return list
}

// optionListTexts 返回当前过滤结果的显示文本（无结果时显示提示项）
func (a *App) optionListTexts(d *optionDialog) []string {
	if len(d.filtered) == 0 {
		return []string{a.noMatchLabel()}
	}
	texts := make([]string, 0, len(d.filtered))
	for _, idx := range d.filtered {
		// 语言包的强制替换表可改写列表中的文本（如滤镜说明）
		texts = append(texts, a.translateListText(d.options[idx]))
	}
	return texts
}

// filterOptions 按 d.filter 计算过滤结果下标
func (a *App) filterOptions(d *optionDialog) {
	d.filtered = d.filtered[:0]
	needle := strings.ToLower(d.filter)
	for i, opt := range d.options {
		if needle == "" || strings.Contains(strings.ToLower(opt), needle) {
			d.filtered = append(d.filtered, i)
		}
	}
}

// optionListHeight 估算选项列表折行后占用的行数（按弹窗内宽折行）
func (a *App) optionListHeight(d *optionDialog, width int) int {
	total := 0
	for i, t := range a.optionListTexts(d) {
		if i > 0 {
			total++ // 项与项之间空一行
		}
		total += len(wrapText(t, width-2))
	}
	if total < 1 {
		total = 1
	}
	return total
}

// buildOptionDialog 按当前过滤条件重建选项对话框
func (a *App) buildOptionDialog() {
	d := a.optDialog
	if d == nil {
		return
	}
	// 过滤
	a.filterOptions(d)

	list := a.newOptionList(d)

	ok := tview.NewButton(tview.Escape("[" + a.s.OK + "]"))
	cancel := tview.NewButton(tview.Escape("[" + a.s.Cancel + "]"))
	ok.SetSelectedFunc(func() {
		ci := list.GetSelected()
		a.closeDialog()
		if ci >= 0 && ci < len(d.filtered) {
			d.confirm(d.filtered[ci])
		}
	})
	cancel.SetSelectedFunc(func() { a.closeDialog() })

	// 高度按折行后的实际行数估算；超界由 centeredGrid 裁剪（上下各留 3 行）
	height := a.optionListHeight(d, 56) + 5
	if height < 7 {
		height = 7
	}
	a.dialogButtons = []tview.Primitive{ok, cancel}
	a.showDialog(a.buildDialogBox(d.title, list, 56, height, ok, cancel), []tview.Primitive{list, ok, cancel}, list)
}

// startOptionSearch 打开搜索框过滤选项对话框；取消时回到原对话框
// startOptionSearch 打开搜索弹窗：标题沿用原弹窗，输入框 + 实时结果列表，
// 结果随输入实时刷新，选中后直接应用（不再重新打开原弹窗）
func (a *App) startOptionSearch() {
	d := a.optDialog
	if d == nil {
		return
	}
	input := tview.NewInputField()
	input.SetFieldWidth(0) // 填满弹窗内宽
	list := a.newOptionList(d)

	// applyFilter 按输入内容实时过滤并重建结果列表
	applyFilter := func(text string) {
		d.filter = text
		a.filterOptions(d)
		list.SetItems(a.optionListTexts(d))
	}
	input.SetChangedFunc(applyFilter)
	input.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			// 输入框 Enter：直接应用当前高亮的结果
			ci := list.GetSelected()
			a.closeDialog()
			if ci >= 0 && ci < len(d.filtered) {
				d.confirm(d.filtered[ci])
			}
		}
	})
	applyFilter(d.filter)

	body := tview.NewFlex().SetDirection(tview.FlexRow)
	body.AddItem(input, 1, 0, true)
	body.AddItem(nil, 1, 0, false) // 输入框与结果列表之间的间隔
	body.AddItem(list, 0, 1, true)

	ok := tview.NewButton(tview.Escape("[" + a.s.OK + "]"))
	cancel := tview.NewButton(tview.Escape("[" + a.s.Cancel + "]"))
	ok.SetSelectedFunc(func() {
		ci := list.GetSelected()
		a.closeDialog()
		if ci >= 0 && ci < len(d.filtered) {
			d.confirm(d.filtered[ci])
		}
	})
	cancel.SetSelectedFunc(func() { a.closeDialog() })

	a.dialogButtons = []tview.Primitive{ok, cancel}
	// 输入框 1 行 + 间隔 1 行 + 折行后的结果行数 + 按钮与留白 3 + 边框 2
	height := a.optionListHeight(d, 62) + 7
	if height < 9 {
		height = 9
	}
	a.showDialog(a.buildDialogBox(d.title, body, 62, height, ok, cancel),
		[]tview.Primitive{input, list, ok, cancel}, input)
}

// showInputDialog 显示输入对话框；confirm(text) 在确定后调用
func (a *App) showInputDialog(title, initial string, confirm func(string)) {
	input := tview.NewInputField()
	input.SetText(initial)
	input.SetFieldWidth(0) // 0 = 填满弹窗内宽
	input.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			a.closeDialog()
			confirm(input.GetText())
		}
	})

	ok := tview.NewButton(tview.Escape("[" + a.s.OK + "]"))
	cancel := tview.NewButton(tview.Escape("[" + a.s.Cancel + "]"))
	ok.SetSelectedFunc(func() {
		a.closeDialog()
		confirm(input.GetText())
	})
	cancel.SetSelectedFunc(func() { a.closeDialog() })

	a.dialogButtons = []tview.Primitive{ok, cancel}
	// 输入框 1 行 + 按钮与留白 3 + 边框 2
	a.showDialog(a.buildDialogBox(title, input, 62, 6, ok, cancel), []tview.Primitive{input, ok, cancel}, input)
}

// showMessageDialog 显示提示对话框（单一 [确定] 按钮）
func (a *App) showMessageDialog(title, text string) {
	tv := tview.NewTextView()
	tv.SetText(text)
	tv.SetTextAlign(tview.AlignLeft)
	// 内容过长时自动换行（按词换行，中文按字换行），换行后两行紧贴
	tv.SetWrap(true)
	tv.SetWordWrap(true)
	tv.SetDynamicColors(true)
	tv.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	tv.SetTextStyle(tcell.StyleDefault.
		Foreground(tview.Styles.PrimaryTextColor).
		Background(tview.Styles.PrimitiveBackgroundColor))

	ok := tview.NewButton(tview.Escape("[" + a.s.OK + "]"))
	ok.SetSelectedFunc(func() { a.closeDialog() })

	a.dialogButtons = []tview.Primitive{ok}
	// 高度随文本行数（含折行）调整
	a.showDialog(a.buildDialogBox(title, tv, 60, a.messageDialogHeight(text, 60), ok), []tview.Primitive{ok}, ok)
}

// showConfirmDialog 显示带 [确定] [取消] 的确认对话框；确定后调用 onOK
func (a *App) showConfirmDialog(text string, onOK func()) {
	tv := tview.NewTextView()
	tv.SetText(text)
	tv.SetTextAlign(tview.AlignLeft)
	// 内容过长时自动换行（按词换行，中文按字换行），换行后两行紧贴
	tv.SetWrap(true)
	tv.SetWordWrap(true)
	tv.SetDynamicColors(true)
	tv.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	tv.SetTextStyle(tcell.StyleDefault.
		Foreground(tview.Styles.PrimaryTextColor).
		Background(tview.Styles.PrimitiveBackgroundColor))

	ok := tview.NewButton(tview.Escape("[" + a.s.OK + "]"))
	cancel := tview.NewButton(tview.Escape("[" + a.s.Cancel + "]"))
	ok.SetSelectedFunc(func() {
		a.closeDialog()
		if onOK != nil {
			onOK()
		}
	})
	cancel.SetSelectedFunc(func() { a.closeDialog() })

	a.dialogButtons = []tview.Primitive{ok, cancel}
	a.showDialog(a.buildDialogBox(a.s.Hint, tv, 56, a.messageDialogHeight(text, 56), ok, cancel),
		[]tview.Primitive{tv, ok, cancel}, ok)
}

// ---------- 文件页操作 ----------

func (a *App) onAddFile() {
	a.showOptionDialog(a.s.AddFile, []string{a.s.InputPath, a.s.PickFile}, func(choice int) {
		switch choice {
		case 0:
			a.showInputDialog(a.s.InputPath, "", func(text string) {
				text = strings.TrimSpace(text)
				if text != "" {
					a.addFilePath(text)
				}
			})
		case 1:
			a.pickFileFromSystem()
		}
	})
}

// pickFileFromSystem 调用系统文件选择器添加文件；无可用的选择器时给出提示
func (a *App) pickFileFromSystem() {
	var paths []string
	var err error
	a.tviewApp.Suspend(func() {
		paths, err = runSystemFilePicker()
	})
	if err != nil {
		a.showMessageDialog(a.s.Hint, a.filePickerFailMsg(err.Error()))
		return
	}
	for _, p := range paths {
		if p != "" {
			a.addFilePath(p)
		}
	}
}

// addFilePath 把文件路径加入文件列表，并异步读取文件信息
func (a *App) addFilePath(path string) {
	idx := a.filesList.GetItemCount()
	a.filesList.AddItem(filepath.Base(path), a.s.ReadingInfo, 0, nil)
	a.files = append(a.files, path)
	a.tviewApp.SetFocus(a.filesList)
	go func() {
		info := a.probeFile(path)
		a.tviewApp.QueueUpdateDraw(func() {
			a.filesList.SetItemText(idx, filepath.Base(path), info)
		})
	}()
}

func (a *App) onSetContainer() {
	// 已加载自定义命令时先提示是否禁用，再打开容器选择
	a.promptDisableCustomCmd(a.openContainerDialog)
}

// openContainerDialog 打开输出容器选择对话框
func (a *App) openContainerDialog() {
	loadFormats(a.ffmpegBin())
	if len(outputFormats) == 0 {
		a.showMessageDialog(a.s.ContainerTitle, a.s.ContainerFailed)
		return
	}
	labels := make([]string, len(outputFormats))
	copy(labels, outputFormats)
	a.showOptionDialog(a.s.ContainerTitle, labels, func(i int) {
		a.cfg.OutputContainer = outputFormats[i]
		a.updateFileBar()
	})
}

// updateFileBar 刷新底部操作栏右侧的容器名显示
func (a *App) updateFileBar() {
	a.fileContainerLabel.SetText(fmt.Sprintf(a.s.ContainerLabel, a.cfg.OutputContainer))
}

// ---------- 系统文件选择器 ----------

// runSystemFilePicker 调用系统文件选择器，返回选中的文件路径列表（可多选）。
// 用户取消时返回 nil；找不到可用的选择器时返回错误。
// Linux: zenity / qarma（--multiple，换行分隔）、kdialog（--getopenfilenames）、
//
//	Xdialog（单选）；Windows: PowerShell OpenFileDialog（Multiselect）。
func runSystemFilePicker() ([]string, error) {
	if runtime.GOOS == "windows" {
		return runWindowsFilePicker()
	}
	var cmd *exec.Cmd
	for _, tool := range []struct {
		name string
		args []string
	}{
		// --separator 传实际换行符，多选路径以换行分隔。
		{"zenity", []string{"--file-selection", "--multiple", "--separator=\n", "--title=选择文件"}},
		{"qarma", []string{"--file-selection", "--multiple", "--separator=\n", "--title=选择文件"}},
		{"kdialog", []string{"--getopenfilenames", ".", "所有文件 (*)"}},
		{"Xdialog", []string{"--fselect", ".", "20", "60"}},
	} {
		if p, err := exec.LookPath(tool.name); err == nil {
			cmd = exec.Command(p, tool.args...)
			break
		}
	}
	if cmd == nil {
		return nil, errors.New("未找到可用的系统文件选择器（zenity/kdialog 等）")
	}
	out, err := cmd.Output()
	if err != nil {
		// 非零退出（如用户取消）视为取消
		return nil, nil
	}
	// 多选路径以换行分隔。
	return splitLines(string(out)), nil
}

// runWindowsFilePicker 用 PowerShell 的 OpenFileDialog 选择文件。
// 结果由 PowerShell 以 UTF-8 写入临时文件后再读取，
// 避免控制台输出编码（UTF-16/GBK）导致中文路径乱码。
func runWindowsFilePicker() ([]string, error) {
	tmp, err := os.CreateTemp("", "ffmpeg-tui-pick-*.txt")
	if err != nil {
		return nil, err
	}
	tmpPath := tmp.Name()
	_ = tmp.Close()
	defer func() { _ = os.Remove(tmpPath) }()

	ps := `Add-Type -AssemblyName System.Windows.Forms; ` +
		`$f = New-Object System.Windows.Forms.OpenFileDialog; ` +
		`$f.Multiselect = $true; ` +
		`$f.Filter = '所有文件 (*.*)|*.*'; ` +
		`if ($f.ShowDialog() -eq [System.Windows.Forms.DialogResult]::OK) { ` +
		`[System.IO.File]::WriteAllText($env:FFMPEG_TUI_PICKFILE, ($f.FileNames -join [char]10), (New-Object System.Text.UTF8Encoding $false)) }`
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", ps)
	cmd.Env = append(os.Environ(), "FFMPEG_TUI_PICKFILE="+tmpPath)
	if err := cmd.Run(); err != nil {
		return nil, nil // 取消或启动失败
	}
	data, err := os.ReadFile(tmpPath)
	if err != nil || len(data) == 0 {
		return nil, nil
	}
	return splitLines(string(data)), nil
}

// splitLines 按换行拆分文本并去掉空行
func splitLines(s string) []string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	var out []string
	for _, line := range strings.Split(s, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			out = append(out, line)
		}
	}
	return out
}

// ---------- 文件信息（ffmpeg -i） ----------

// probeFile 用 `ffmpeg -i` 读取媒体文件信息，返回一行格式化摘要
func (a *App) probeFile(path string) string {
	name := a.ffmpegBin()
	if _, err := exec.LookPath(name); err != nil {
		return fmt.Sprintf(a.s.NoFFmpeg, name)
	}
	info := a.ffmpegInfo(path)
	var size int64
	if fi, err := os.Stat(path); err == nil {
		size = fi.Size()
	}
	return a.formatMediaInfo(info, size)
}

// formatMediaInfo 生成如 "H264 · 1280x720 · 30 fps · AAC 44100Hz · 2ch · 00:12 · 1.2 MB" 的摘要
func (a *App) formatMediaInfo(info *mediaInfo, size int64) string {
	var parts []string
	if info.HasVideo {
		parts = append(parts, strings.ToUpper(info.VideoCodec))
		if info.Width > 0 && info.Height > 0 {
			parts = append(parts, fmt.Sprintf("%dx%d", info.Width, info.Height))
		}
		if info.FPS > 0 {
			parts = append(parts, formatFPS(info.FPS))
		}
	}
	if info.HasAudio {
		p := strings.ToUpper(info.AudioCodec)
		if info.AudioRate > 0 {
			p += " " + strconv.Itoa(info.AudioRate) + "Hz"
		}
		if info.AudioChannels > 0 {
			p += " · " + strconv.Itoa(info.AudioChannels) + "ch"
		}
		parts = append(parts, p)
	}
	if info.Duration > 0 {
		parts = append(parts, formatDuration(info.Duration))
	}
	if size > 0 {
		parts = append(parts, formatSize(size))
	}
	if len(parts) == 0 {
		return a.s.ParseInfoFailed
	}
	return strings.Join(parts, " · ")
}

// ffmpegBin 返回 ffmpeg 可执行文件（跟随 FFmpeg 路径设置）
func (a *App) ffmpegBin() string {
	return ffmpegBin(a.cfg)
}

// formatFPS 格式化帧率：整数显示为 "25 fps"，非整数最多保留三位小数（如 23.976 fps）
func formatFPS(fps float64) string {
	if fps == math.Trunc(fps) {
		return fmt.Sprintf("%.0f fps", fps)
	}
	s := strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.3f", fps), "0"), ".")
	return s + " fps"
}

// formatDuration 把秒数格式化为 00:12:08 / 00:45
func formatDuration(seconds float64) string {
	s := int(seconds)
	h := s / 3600
	m := (s % 3600) / 60
	sec := s % 60
	if h > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", h, m, sec)
	}
	return fmt.Sprintf("%02d:%02d", m, sec)
}

// formatSize 把字节数格式化为 MB / GB
func formatSize(bytes int64) string {
	const (
		mb = 1 << 20
		gb = 1 << 30
	)
	switch {
	case bytes >= gb:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(gb))
	case bytes >= mb:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(mb))
	default:
		return fmt.Sprintf("%d KB", bytes/1024)
	}
}

// showAboutDialog 显示关于弹窗：FFmpeg-TUI / 版本 / by CMR0649 /
// GitHub 链接（蓝色，点击或选中确认后打开项目主页）
func (a *App) showAboutDialog() {
	tv := tview.NewTextView()
	tv.SetDynamicColors(true)
	tv.SetTextAlign(tview.AlignCenter)
	tv.SetText(a.s.AboutLine1 + "\n\n" +
		"[yellow]" + fmt.Sprintf(a.s.AboutLine2, Version) + "[-]\n\n" +
		a.s.AboutLine3 + "\n\n")
	tv.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	tv.SetTextStyle(tcell.StyleDefault.
		Foreground(tview.Styles.PrimaryTextColor).
		Background(tview.Styles.PrimitiveBackgroundColor))

	github := tview.NewButton(a.s.AboutGitHub) // 标签直接为 GitHub（不带方括号，避免被解析为颜色标签）
	// 白字蓝底（普通与聚焦状态一致），符合需求
	github.SetStyle(tcell.StyleDefault.Background(tcell.ColorBlue).Foreground(tcell.ColorWhite))
	github.SetActivatedStyle(tcell.StyleDefault.Background(tcell.ColorBlue).Foreground(tcell.ColorWhite))
	github.SetSelectedFunc(func() {
		a.closeDialog()
		_ = openURL("https://github.com/CMR0649/ffmpeg-tui")
	})

	ok := tview.NewButton(tview.Escape("[" + a.s.OK + "]"))
	ok.SetSelectedFunc(func() { a.closeDialog() })

	a.dialogButtons = []tview.Primitive{github, ok}
	a.showDialog(a.buildDialogBox(a.s.AboutTitle, tv, 50, 12, github, ok),
		[]tview.Primitive{tv, github, ok}, github)
}

// openURL 用系统默认方式打开链接
func openURL(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}

// noMatchLabel 「无匹配项」按当前语言
func (a *App) noMatchLabel() string {
	if a.lang == "en" {
		return "(no match)"
	}
	return "（无匹配项）"
}

// filePickerFailMsg 系统文件选择器不可用时的提示
func (a *App) filePickerFailMsg(err string) string {
	if a.lang == "en" {
		return "Cannot open system file picker: " + err + "\nPlease use \"Enter path\" instead."
	}
	return "无法调用系统文件选择器：" + err + "\n请改用「输入路径」方式添加文件"
}
