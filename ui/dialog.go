package ui

import (
	"errors"
	"fmt"
	"math"
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

// dialogFocusNext / dialogFocusPrev 在对话框内循环移动焦点（Tab / Shift-Tab）。
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

// closeDialog 关闭当前对话框，并把焦点还给当前标签页内容。
// 若正处在选项搜索中，则重建选项对话框（取消搜索）。
func (a *App) closeDialog() {
	if !a.dialogOpen {
		return
	}
	a.dialogOpen = false
	a.rootPages.RemovePage("dialog")
	if a.searching {
		a.searching = false
		a.buildOptionDialog()
		return
	}
	a.tviewApp.SetFocus(a.pages.GetPage(tabNames[a.current]))
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

// showOptionDialog 显示选项选择对话框；confirm(index) 在确定后调用。
// 打开后按 "/" 打开搜索框过滤选项。
func (a *App) showOptionDialog(title string, options []string, confirm func(int)) {
	a.optDialog = &optionDialog{title: title, options: options, confirm: confirm}
	a.buildOptionDialog()
}

// buildOptionDialog 按当前过滤条件重建选项对话框。
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
	for _, idx := range d.filtered {
		idx := idx
		list.AddItem(d.options[idx], "", 0, func() {
			a.closeDialog()
			d.confirm(idx)
		})
	}
	if len(d.filtered) == 0 {
		list.AddItem("（无匹配项）", "", 0, nil)
	}
	d.list = list

	ok := tview.NewButton("[确定]")
	cancel := tview.NewButton("[取消]")
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

// startOptionSearch 打开搜索框过滤选项对话框；取消时回到原对话框。
func (a *App) startOptionSearch() {
	d := a.optDialog
	if d == nil {
		return
	}
	a.searching = true
	a.showInputDialog("搜索", d.filter, func(text string) {
		a.searching = false
		d.filter = strings.TrimSpace(text)
		a.buildOptionDialog()
	})
}

// showInputDialog 显示输入对话框；confirm(text) 在确定后调用。
func (a *App) showInputDialog(title, initial string, confirm func(string)) {
	input := tview.NewInputField()
	input.SetText(initial)
	input.SetFieldWidth(34)
	input.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			a.closeDialog()
			confirm(input.GetText())
		}
	})

	ok := tview.NewButton("[确定]")
	cancel := tview.NewButton("[取消]")
	ok.SetSelectedFunc(func() {
		a.closeDialog()
		confirm(input.GetText())
	})
	cancel.SetSelectedFunc(func() { a.closeDialog() })

	a.dialogButtons = []tview.Primitive{ok, cancel}
	a.showDialog(a.buildDialogBox(title, input, 50, 8, ok, cancel), []tview.Primitive{input, ok, cancel}, input)
}

// showMessageDialog 显示提示对话框（单一 [确定] 按钮）。
func (a *App) showMessageDialog(title, text string) {
	tv := tview.NewTextView()
	tv.SetText(text)
	tv.SetTextAlign(tview.AlignLeft)
	tv.SetDynamicColors(true)
	tv.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	tv.SetTextStyle(tcell.StyleDefault.
		Foreground(tview.Styles.PrimaryTextColor).
		Background(tview.Styles.PrimitiveBackgroundColor))

	ok := tview.NewButton("[确定]")
	ok.SetSelectedFunc(func() { a.closeDialog() })

	a.dialogButtons = []tview.Primitive{ok}
	a.showDialog(a.buildDialogBox(title, tv, 60, 8, ok), []tview.Primitive{ok}, ok)
}

// ---------- 文件页操作 ----------

func (a *App) onAddFile() {
	a.showOptionDialog("添加文件", []string{"输入路径", "选择文件"}, func(choice int) {
		switch choice {
		case 0:
			a.showInputDialog("输入路径", "", func(text string) {
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

// pickFileFromSystem 调用系统文件选择器添加文件；无可用的选择器时给出提示。
func (a *App) pickFileFromSystem() {
	var path string
	var err error
	a.tviewApp.Suspend(func() {
		path, err = runSystemFilePicker()
	})
	if err != nil {
		a.showMessageDialog("提示", "无法调用系统文件选择器："+err.Error()+"\n请改用「输入路径」方式添加文件。")
		return
	}
	if path != "" {
		a.addFilePath(path)
	}
}

// addFilePath 把文件路径加入文件列表，并异步调用 ffprobe 读取文件信息
func (a *App) addFilePath(path string) {
	idx := a.filesList.GetItemCount()
	a.filesList.AddItem(filepath.Base(path), "正在读取文件信息…", 0, nil)
	a.files = append(a.files, path)
	a.tviewApp.SetFocus(a.filesList)
	go func() {
		info := probeFile(path)
		a.tviewApp.QueueUpdateDraw(func() {
			a.filesList.SetItemText(idx, filepath.Base(path), info)
		})
	}()
}

func (a *App) onSetContainer() {
	loadFormats()
	if len(outputFormats) == 0 {
		a.showMessageDialog("输出容器", "无法获取容器格式列表（ffmpeg -formats）。")
		return
	}
	labels := make([]string, len(outputFormats))
	copy(labels, outputFormats)
	a.showOptionDialog("输出容器", labels, func(i int) {
		a.outputContainer = outputFormats[i]
		a.updateFileBar()
	})
}

// updateFileBar 刷新底部操作栏右侧的容器名显示。
func (a *App) updateFileBar() {
	a.fileContainerLabel.SetText(fmt.Sprintf("输出容器：%s", a.outputContainer))
}

// ---------- 系统文件选择器 ----------

// runSystemFilePicker 调用系统文件选择器，返回选中的文件路径。
// 用户取消时返回空字符串；找不到可用的选择器时返回错误。
// Linux: zenity / qarma / kdialog / Xdialog；Windows: PowerShell OpenFileDialog。
func runSystemFilePicker() (string, error) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		ps := `Add-Type -AssemblyName System.Windows.Forms; $f = New-Object System.Windows.Forms.OpenFileDialog; $f.Filter = '所有文件 (*.*)|*.*'; if ($f.ShowDialog() -eq [System.Windows.Forms.DialogResult]::OK) { $f.FileName }`
		cmd = exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", ps)
	default:
		for _, tool := range []struct {
			name string
			args []string
		}{
			{"zenity", []string{"--file-selection", "--title=选择文件"}},
			{"qarma", []string{"--file-selection", "--title=选择文件"}},
			{"kdialog", []string{"--getopenfilename", ".", "所有文件 (*)"}},
			{"Xdialog", []string{"--fselect", ".", "20", "60"}},
		} {
			if p, err := exec.LookPath(tool.name); err == nil {
				cmd = exec.Command(p, tool.args...)
				break
			}
		}
	}
	if cmd == nil {
		return "", errors.New("未找到可用的系统文件选择器（zenity/kdialog 等）")
	}
	out, err := cmd.Output()
	if err != nil {
		// 非零退出（如用户取消）视为取消。
		return "", nil
	}
	return strings.TrimSpace(string(out)), nil
}

// ---------- ffprobe 文件信息 ----------

// probeFile 调用 ffprobe（Windows 为 ffprobe.exe）读取媒体文件信息，
// 返回一行格式化摘要；ffprobe 不可用或解析失败时返回相应提示。
func probeFile(path string) string {
	name := "ffprobe"
	if runtime.GOOS == "windows" {
		name = "ffprobe.exe"
	}
	exe, err := exec.LookPath(name)
	if err != nil {
		return "未找到 " + name + "，无法读取文件信息"
	}
	out, err := exec.Command(exe,
		"-v", "error",
		"-show_entries", "format=duration,size,format_name",
		"-show_entries", "stream=codec_name,codec_type,width,height,avg_frame_rate,sample_rate,channels",
		"-of", "default",
		path,
	).Output()
	if err != nil {
		return "无法读取文件信息：" + err.Error()
	}
	return formatProbeInfo(string(out))
}

// formatProbeInfo 解析 ffprobe 的 key=value 输出（含 [STREAM]/[FORMAT] 段落），
// 生成如 "H.264 · 1920×1080 · 29.97 fps · AAC 48000Hz · 2ch · 00:12:08 · 1.2 GB" 的摘要。
func formatProbeInfo(raw string) string {
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
		return "未能解析文件信息"
	}
	return strings.Join(parts, " · ")
}

// formatFPS 格式化帧率：整数显示为 "25 fps"，否则保留两位小数（如 "29.97 fps"）。
func formatFPS(fps float64) string {
	if fps == math.Trunc(fps) {
		return fmt.Sprintf("%.0f fps", fps)
	}
	return fmt.Sprintf("%.2f fps", fps)
}

// parseFPS 解析 ffprobe 的帧率（如 "30000/1001" 或 "30"），无法解析时返回 0。
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

// parseDuration 解析时长（秒），无效时返回 0。
func parseDuration(d string) float64 {
	f, err := strconv.ParseFloat(d, 64)
	if err != nil || f <= 0 {
		return 0
	}
	return f
}

// formatDuration 把秒数格式化为 00:12:08 / 00:45。
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

// parseSize 解析文件大小（字节），无效时返回 0。
func parseSize(size string) int64 {
	n, err := strconv.ParseInt(size, 10, 64)
	if err != nil || n <= 0 {
		return 0
	}
	return n
}

// formatSize 把字节数格式化为 MB / GB。
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
