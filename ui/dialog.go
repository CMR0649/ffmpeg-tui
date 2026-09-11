package ui

import (
	"encoding/base64"
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
	a.dialogOpen = false
	a.rootPages.RemovePage("dialog")
	a.tviewApp.SetFocus(a.pages.GetPage(tabKeys[a.current]))
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
	grid := tview.NewGrid()
	grid.SetColumns(0, width, 0)
	grid.SetRows(0, height, 0)
	grid.AddItem(box, 1, 1, 1, 1, 0, 0, true)
	grid.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	return grid
}

type optionDialog struct {
	title    string
	options  []string
	confirm  func(int)
	filter   string
	filtered []int
	list     *tview.List
}

// showOptionDialog 显示选项选择对话框；confirm(index) 在确定后调用
// 打开后按 "/" 打开搜索框过滤选项
func (a *App) showOptionDialog(title string, options []string, confirm func(int)) {
	a.optDialog = &optionDialog{title: title, options: options, confirm: confirm}
	a.buildOptionDialog()
}

// buildOptionDialog 按当前过滤条件重建选项对话框
func (a *App) buildOptionDialog() {
	d := a.optDialog
	if d == nil {
		return
	}
	// 过滤
	d.filtered = d.filtered[:0]
	needle := strings.ToLower(d.filter)
	for i, opt := range d.options {
		if needle == "" || strings.Contains(strings.ToLower(opt), needle) {
			d.filtered = append(d.filtered, i)
		}
	}

	list := tview.NewList()
	a.selectOnSecondClick(list)
	for _, idx := range d.filtered {
		idx := idx
		list.AddItem(d.options[idx], "", 0, func() {
			a.closeDialog()
			d.confirm(idx)
		})
	}
	if len(d.filtered) == 0 {
		list.AddItem(a.noMatchLabel(), "", 0, nil)
	}
	d.list = list

	ok := tview.NewButton(tview.Escape("[" + a.s.OK + "]"))
	cancel := tview.NewButton(tview.Escape("[" + a.s.Cancel + "]"))
	ok.SetSelectedFunc(func() {
		ci := list.GetCurrentItem()
		a.closeDialog()
		if ci >= 0 && ci < len(d.filtered) {
			d.confirm(d.filtered[ci])
		}
	})
	cancel.SetSelectedFunc(func() { a.closeDialog() })

	height := len(d.filtered) + 6
	if height < 10 {
		height = 10
	}
	if height > 18 {
		height = 18
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
	list := tview.NewList()
	a.selectOnSecondClick(list)

	// applyFilter 按输入内容实时过滤并重建结果列表
	applyFilter := func(text string) {
		d.filter = text
		d.filtered = d.filtered[:0]
		needle := strings.ToLower(text)
		for i, opt := range d.options {
			if needle == "" || strings.Contains(strings.ToLower(opt), needle) {
				d.filtered = append(d.filtered, i)
			}
		}
		list.Clear()
		for _, idx := range d.filtered {
			idx := idx
			list.AddItem(d.options[idx], "", 0, func() {
				a.closeDialog()
				d.confirm(idx)
			})
		}
		if len(d.filtered) == 0 {
			list.AddItem(a.noMatchLabel(), "", 0, nil)
		}
	}
	input.SetChangedFunc(applyFilter)
	input.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			// 输入框 Enter：直接应用当前高亮的结果
			ci := list.GetCurrentItem()
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
		ci := list.GetCurrentItem()
		a.closeDialog()
		if ci >= 0 && ci < len(d.filtered) {
			d.confirm(d.filtered[ci])
		}
	})
	cancel.SetSelectedFunc(func() { a.closeDialog() })

	a.dialogButtons = []tview.Primitive{ok, cancel}
	a.showDialog(a.buildDialogBox(d.title, body, 62, 16, ok, cancel),
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
	a.showDialog(a.buildDialogBox(title, input, 62, 8, ok, cancel), []tview.Primitive{input, ok, cancel}, input)
}

// showMessageDialog 显示提示对话框（单一 [确定] 按钮）
func (a *App) showMessageDialog(title, text string) {
	tv := tview.NewTextView()
	tv.SetText(text)
	tv.SetTextAlign(tview.AlignLeft)
	tv.SetDynamicColors(true)
	tv.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	tv.SetTextStyle(tcell.StyleDefault.
		Foreground(tview.Styles.PrimaryTextColor).
		Background(tview.Styles.PrimitiveBackgroundColor))

	ok := tview.NewButton(tview.Escape("[" + a.s.OK + "]"))
	ok.SetSelectedFunc(func() { a.closeDialog() })

	a.dialogButtons = []tview.Primitive{ok}
	a.showDialog(a.buildDialogBox(title, tv, 60, 8, ok), []tview.Primitive{ok}, ok)
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

// addFilePath 把文件路径加入文件列表，并异步调用 ffprobe 读取文件信息
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
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		ps := `Add-Type -AssemblyName System.Windows.Forms; $f = New-Object System.Windows.Forms.OpenFileDialog; $f.Multiselect = $true; $f.Filter = '所有文件 (*.*)|*.*'; if ($f.ShowDialog() -eq [System.Windows.Forms.DialogResult]::OK) { $f.FileNames -join [Environment]::NewLine }`
		cmd = exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", ps)
	default:
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
	var paths []string
	for _, line := range strings.Split(string(out), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			paths = append(paths, line)
		}
	}
	return paths, nil
}

// ---------- ffprobe 文件信息 ----------

// probeFile 调用 ffprobe（Windows 为 ffprobe.exe）读取媒体文件信息，
// 返回一行格式化摘要；ffprobe 不可用或解析失败时返回相应提示
func (a *App) probeFile(path string) string {
	name := a.ffprobeBin()
	exe, err := exec.LookPath(name)
	if err != nil {
		return fmt.Sprintf(a.s.NoFFprobe, name)
	}
	out, err := exec.Command(exe,
		"-v", "error",
		"-show_entries", "format=duration,size,format_name",
		"-show_entries", "stream=codec_name,codec_type,width,height,avg_frame_rate,sample_rate,channels",
		"-of", "default",
		path,
	).Output()
	if err != nil {
		return fmt.Sprintf(a.s.ReadInfoFailed, err.Error())
	}
	return a.formatProbeInfo(string(out))
}

// ffprobeBin 返回 ffprobe 可执行文件（跟随 FFmpeg 路径设置）
func (a *App) ffprobeBin() string {
	return ffprobeBin(a.cfg)
}

// ffmpegBin 返回 ffmpeg 可执行文件（跟随 FFmpeg 路径设置）
func (a *App) ffmpegBin() string {
	return ffmpegBin(a.cfg)
}

// parseFailedText 返回文件信息解析失败的提示
func (a *App) parseFailedText() string {
	return a.s.ParseInfoFailed
}

// formatProbeInfo 解析 ffprobe 的 key=value 输出（含 [STREAM]/[FORMAT] 段落），
// 生成如 "H.264 · 1920×1080 · 29.97 fps · AAC 48000Hz · 2ch · 00:12:08 · 1.2 GB" 的摘要
func (a *App) formatProbeInfo(raw string) string {
	var format map[string]string
	var streams []map[string]string
	var cur map[string]string
	inStream := false
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		switch line {
		case "[STREAM]":
			inStream = true
			cur = map[string]string{}
		case "[/STREAM]":
			inStream = false
			streams = append(streams, cur)
			cur = nil
		case "[FORMAT]":
			inStream = false
			format = map[string]string{}
		case "[/FORMAT]":
			inStream = false
		default:
			if inStream && cur != nil {
				if i := strings.Index(line, "="); i > 0 {
					cur[strings.TrimSpace(line[:i])] = strings.TrimSpace(line[i+1:])
				}
			} else if format != nil {
				if i := strings.Index(line, "="); i > 0 {
					format[strings.TrimSpace(line[:i])] = strings.TrimSpace(line[i+1:])
				}
			}
		}
	}

	var video, audio map[string]string
	for _, s := range streams {
		switch s["codec_type"] {
		case "video":
			if video == nil {
				video = s
			}
		case "audio":
			if audio == nil {
				audio = s
			}
		}
	}

	var parts []string
	if video != nil {
		parts = append(parts, strings.ToUpper(video["codec_name"]))
		if video["width"] != "" && video["height"] != "" {
			parts = append(parts, video["width"]+"×"+video["height"])
		}
		if fps := parseFPS(video["avg_frame_rate"]); fps > 0 {
			parts = append(parts, formatFPS(fps))
		}
	}
	if audio != nil {
		p := strings.ToUpper(audio["codec_name"])
		if audio["sample_rate"] != "" && audio["sample_rate"] != "0" {
			p += " " + audio["sample_rate"] + "Hz"
		}
		if audio["channels"] != "" && audio["channels"] != "0" {
			p += " · " + audio["channels"] + "ch"
		}
		parts = append(parts, p)
	}
	if dur := parseDuration(format["duration"]); dur > 0 {
		parts = append(parts, formatDuration(dur))
	}
	if size := parseSize(format["size"]); size > 0 {
		parts = append(parts, formatSize(size))
	}
	if len(parts) == 0 {
		return a.parseFailedText()
	}
	return strings.Join(parts, " · ")
}

// formatFPS 格式化帧率：整数显示为 "25 fps"，否则保留两位小数（如 "29.97 fps"）
func formatFPS(fps float64) string {
	if fps == math.Trunc(fps) {
		return fmt.Sprintf("%.0f fps", fps)
	}
	return fmt.Sprintf("%.2f fps", fps)
}

// parseFPS 解析 ffprobe 的帧率（如 "30000/1001" 或 "30"），无法解析时返回 0
func parseFPS(rate string) float64 {
	if rate == "" || rate == "0/0" || rate == "N/A" {
		return 0
	}
	if p := strings.Split(rate, "/"); len(p) == 2 {
		n, err1 := strconv.ParseFloat(p[0], 64)
		d, err2 := strconv.ParseFloat(p[1], 64)
		if err1 == nil && err2 == nil && d > 0 {
			return n / d
		}
	}
	f, err := strconv.ParseFloat(rate, 64)
	if err != nil || f <= 0 {
		return 0
	}
	return f
}

// parseDuration 解析时长（秒），无效时返回 0
func parseDuration(d string) float64 {
	f, err := strconv.ParseFloat(d, 64)
	if err != nil || f <= 0 {
		return 0
	}
	return f
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

// parseSize 解析文件大小（字节），无效时返回 0
func parseSize(size string) int64 {
	n, err := strconv.ParseInt(size, 10, 64)
	if err != nil || n <= 0 {
		return 0
	}
	return n
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

// copyToClipboard 把文本复制到系统剪贴板：
// 先直接向终端发送 OSC 52（BEL 结尾，不依赖 tcell terminfo），
// 再用 tcell 屏幕发送一次（ST 结尾），两种终止符与两条通道都覆盖；
// 最后尝试外部剪贴板工具
func (a *App) copyToClipboard(text string) {
	writeOSC52(text)
	if a.screen != nil {
		a.screen.SetClipboard([]byte(text))
	}
	copyViaTool(text)
}

// writeOSC52 直接把 OSC 52 序列写入终端
func writeOSC52(text string) {
	if runtime.GOOS == "windows" {
		return
	}
	seq := "\x1b]52;c;" + base64.StdEncoding.EncodeToString([]byte(text)) + "\x07"
	if f, err := os.OpenFile("/dev/tty", os.O_WRONLY, 0); err == nil {
		_, _ = f.WriteString(seq)
		_ = f.Close()
		return
	}
	_, _ = os.Stdout.WriteString(seq)
}

// copyViaTool 尝试用系统剪贴板工具复制
func copyViaTool(text string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("clip")
	case "darwin":
		cmd = exec.Command("pbcopy")
	default:
		if p, err := exec.LookPath("wl-copy"); err == nil {
			cmd = exec.Command(p)
		} else if p, err := exec.LookPath("xclip"); err == nil {
			cmd = exec.Command(p, "-selection", "clipboard")
		} else if p, err := exec.LookPath("xsel"); err == nil {
			cmd = exec.Command(p, "-b", "-i")
		}
	}
	if cmd == nil {
		return
	}
	cmd.Stdin = strings.NewReader(text)
	_ = cmd.Run()
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
