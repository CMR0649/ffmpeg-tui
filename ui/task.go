package ui

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rivo/tview"
	"golang.org/x/term"
)

// Task 一个转码任务
type Task struct {
	Input    string
	Output   string
	Status   string // 等待中 / 转码中 / 已完成 / 失败
	Progress float64
	ErrMsg   string // 失败原因
	duration float64
	hasVideo bool
	hasAudio bool
	index    int
	// frame= 行解析出的实时数据
	frameCount, fps, bitrate, speed string
}

// frameRe 匹配 ffmpeg 进度输出中以 frame= 开头的行，
// 提取 frame(已处理帧)/fps(帧处理速率)/bitrate/speed
var frameRe = regexp.MustCompile(`frame=\s*(\S+)\s+fps=\s*(\S+).*?bitrate=\s*(\S+).*?speed=\s*(\S+)`)

// scanProgress 进度输出以 \r 分隔（ffmpeg 在同一行用回车覆盖），
// 既按 \r 也按 \n 分行
func scanProgress(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.IndexAny(data, "\r\n"); i >= 0 {
		return i + 1, data[:i], nil
	}
	if atEOF {
		return len(data), data, nil
	}
	return 0, nil, nil
}

// addTasksFromFiles 把文件页的文件列表生成为转码任务
func (a *App) addTasksFromFiles() {
	if len(a.files) == 0 {
		a.showMessageDialog(a.s.Hint, a.s.NoFilesToTask)
		return
	}
	a.tasks = nil
	for _, input := range a.files {
		hasVideo, hasAudio := a.probeStreams(input)
		t := &Task{
			Input:    input,
			Output:   a.outputPath(input),
			Status:   "waiting",
			duration: a.probeDuration(input),
			hasVideo: hasVideo,
			hasAudio: hasAudio,
		}
		a.tasks = append(a.tasks, t)
	}
	a.refreshTasks()
}

// refreshTasks 重建任务列表
func (a *App) refreshTasks() {
	a.taskList.Clear()
	for _, t := range a.tasks {
		t := t
		t.index = a.taskList.GetItemCount()
		a.taskList.AddItem(taskMainText(t), a.taskSecondaryText(t), 0, func() { a.openTaskTarget(t) })
	}
}

func taskMainText(t *Task) string {
	return filepath.Base(t.Input) + " → " + filepath.Base(t.Output)
}

func (a *App) taskSecondaryText(t *Task) string {
	switch t.Status {
	case "done":
		return a.s.TaskDone + " · " + a.s.TaskSavedTo + t.Output
	case "running":
		return a.taskProgressText(t)
	case "failed":
		return a.s.TaskFailed + " · " + a.s.TaskLogSavedTo + ffmpegLogPath()
	default:
		return a.s.TaskWaiting
	}
}

// taskProgressText 转码中的实时数据显示：仅接受以 frame= 开头的行，
// 展示 frame/fps/bitrate/speed；中文用对应汉字段名，英文用原文
func (a *App) taskProgressText(t *Task) string {
	if t.frameCount == "" && t.fps == "" && t.bitrate == "" && t.speed == "" {
		return a.s.TaskRunning
	}
	var parts []string
	if t.frameCount != "" {
		parts = append(parts, a.s.FrameCount+"="+t.frameCount)
	}
	if t.fps != "" {
		parts = append(parts, a.s.FrameFPS+"="+t.fps)
	}
	if t.bitrate != "" {
		parts = append(parts, a.s.FrameBitrate+"="+t.bitrate)
	}
	if t.speed != "" {
		parts = append(parts, a.s.FrameSpeed+"="+t.speed)
	}
	return strings.Join(parts, "  ")
}

// updateTask 更新单个任务在列表中的显示
func (a *App) updateTask(t *Task) {
	a.tviewApp.QueueUpdateDraw(func() {
		a.taskList.SetItemText(t.index, taskMainText(t), a.taskSecondaryText(t))
	})
}

// startTasks 并行执行所有等待中的任务（并行数为配置值）
func (a *App) startTasks() {
	n := a.cfg.ParallelTasks
	if n < 1 {
		n = 1
	}
	go func() {
		sem := make(chan struct{}, n)
		var wg sync.WaitGroup
		for _, t := range a.tasks {
			if t.Status != "waiting" {
				continue
			}
			wg.Add(1)
			sem <- struct{}{}
			go func(t *Task) {
				defer wg.Done()
				defer func() { <-sem }()
				t.Status = "running"
				a.updateTask(t)
				err := a.runTask(t)
				if err != nil {
					t.Status = "failed"
					t.Progress = 0
					t.ErrMsg = err.Error()
				} else {
					t.Status = "done"
					t.Progress = 100
				}
				a.updateTask(t)
			}(t)
		}
		wg.Wait()
	}()
}

// runTask 执行单个转码任务并解析进度；失败时写入 ffmpeg 日志并返回错误摘要
func (a *App) runTask(t *Task) error {
	argv := a.resolvedArgs(t)
	cmd := exec.Command(argv[0], argv[1:]...)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	lastErr := ""
	var output strings.Builder
	scanner := bufio.NewScanner(stderr)
	scanner.Split(scanProgress)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "frame=") {
			// 进度行：解析实时数据，但不写入日志
			if m := frameRe.FindStringSubmatch(line); m != nil {
				t.frameCount, t.fps, t.bitrate, t.speed = m[1], m[2], m[3], m[4]
				a.updateTask(t)
			}
		} else {
			output.WriteString(line)
			output.WriteString("\n")
		}
		if strings.Contains(line, "Error") || strings.Contains(line, "error") {
			if s := strings.TrimSpace(line); len(s) > 60 {
				lastErr = s[:60]
			} else {
				lastErr = s
			}
		}
	}
	if err := cmd.Wait(); err != nil {
		a.writeFFmpegLog(argv, output.String())
		if lastErr == "" {
			return err
		}
		return fmt.Errorf("%s", lastErr)
	}
	return nil
}

// ffmpegLogPath ffmpeg 日志路径：Windows 为 .\ffmpeg.log，其余平台为缓存目录
func ffmpegLogPath() string {
	if runtime.GOOS == "windows" {
		return filepath.Join(".", "ffmpeg.log")
	}
	dir, err := os.UserCacheDir()
	if err != nil {
		dir = "."
	}
	return filepath.Join(dir, "ffmpeg-tui", "ffmpeg.log")
}

// writeFFmpegLog 以追加方式把失败任务的执行时间、命令与 ffmpeg 输出写入日志
func (a *App) writeFFmpegLog(argv []string, output string) {
	path := ffmpegLogPath()
	if dir := filepath.Dir(path); dir != "." {
		_ = os.MkdirAll(dir, 0o755)
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	var sb strings.Builder
	sb.WriteString(time.Now().Format("2006-01-02 15:04:05"))
	sb.WriteString("\n")
	sb.WriteString(strings.Join(argv, " "))
	sb.WriteString("\n\n")
	sb.WriteString(output)
	sb.WriteString("\n------\n")
	_, _ = f.WriteString(sb.String())
}

// openTaskTarget 打开任务的输出目录（已完成）或日志目录（失败）
func (a *App) openTaskTarget(t *Task) {
	switch t.Status {
	case "done":
		_ = openURL(filepath.Dir(t.Output))
	case "failed":
		_ = openURL(filepath.Dir(ffmpegLogPath()))
	}
}

// customCmdRe 匹配自定义命令中的 output.*/output.<后缀> 与 input 占位符
var customCmdRe = regexp.MustCompile(`output\.(\*|[A-Za-z0-9]+)|input`)

// resolveCustomCommand 替换自定义命令中的占位符：
// output.* → 输出文件；input → 输入文件；ffmpeg → 设置指定的 ffmpeg（若有）
func (a *App) resolveCustomCommand(tpl, input, output string) string {
	s := customCmdRe.ReplaceAllStringFunc(tpl, func(m string) string {
		if strings.HasPrefix(m, "output") {
			return output
		}
		return input
	})
	if a.cfg.FFmpegPath != "" {
		s = strings.ReplaceAll(s, "ffmpeg", a.ffmpegBin())
	}
	return s
}

// splitCommandLine 按空白拆分命令行，支持单/双引号
func splitCommandLine(s string) []string {
	var args []string
	var cur strings.Builder
	var quote rune
	inToken := false
	for _, r := range s {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				cur.WriteRune(r)
			}
		case r == '\'' || r == '"':
			quote = r
			inToken = true
		case r == ' ' || r == '\t' || r == '\n':
			if inToken {
				args = append(args, cur.String())
				cur.Reset()
				inToken = false
			}
		default:
			cur.WriteRune(r)
			inToken = true
		}
	}
	if inToken {
		args = append(args, cur.String())
	}
	return args
}

// resolvedArgs 返回任务要执行的完整 argv（自定义命令优先）
func (a *App) resolvedArgs(t *Task) []string {
	if a.customCmd != "" {
		return splitCommandLine(a.resolveCustomCommand(a.customCmd, t.Input, t.Output))
	}
	return append([]string{a.ffmpegBin()},
		a.buildCommand(t.Input, t.Output, t.hasVideo, t.hasAudio)...)
}

// taskCommandString 生成任务命令（输入以 input 代替，输出以 output.后缀 代替）；
// 自定义命令直接显示其模板
func (a *App) taskCommandString(t *Task) string {
	if a.customCmd != "" {
		return a.customCmd
	}
	out := "output" + filepath.Ext(t.Output)
	args := a.buildCommand("input", out, t.hasVideo, t.hasAudio)
	return a.ffmpegBin() + " " + strings.Join(args, " ")
}

// showTaskCommand 在终端中以 CLI 形式显示命令：有选中任务时按该任务生成，
// 否则按当前配置生成（假定输入含视频与音频流）
func (a *App) showTaskCommand() {
	cmd := a.currentCommandString()
	if a.taskList != nil {
		if idx := a.taskList.GetCurrentItem(); idx >= 0 && idx < len(a.tasks) {
			cmd = a.taskCommandString(a.tasks[idx])
		}
	}
	showCommandInTerminal(a.tviewApp, a.s.CmdLabel, cmd, a.s.CmdReturnHint)
}

// showCommandInTerminal 暂停 TUI，在终端中输出命令，按键后返回 TUI
func showCommandInTerminal(app *tview.Application, label, cmd, hint string) {
	app.Suspend(func() {
		fmt.Fprintln(os.Stdout, label)
		fmt.Fprintln(os.Stdout, cmd)
		fmt.Fprintln(os.Stdout, hint)
		waitAnyKey()
	})
}

// waitAnyKey 等待一次按键（回车/空格/任意键）
func waitAnyKey() {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
		return
	}
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
		return
	}
	defer func() { _ = term.Restore(fd, oldState) }()
	buf := make([]byte, 1)
	_, _ = os.Stdin.Read(buf)
}

// currentCommandString 按当前配置生成命令（输入 input，输出 output.后缀）；
// 已加载自定义命令时显示其模板
func (a *App) currentCommandString() string {
	if a.customCmd != "" {
		return a.customCmd
	}
	out := "output"
	if a.cfg.OutputContainer != "" {
		out += "." + containerExt(a.cfg.OutputContainer)
	}
	args := a.buildCommand("input", out, true, true)
	return a.ffmpegBin() + " " + strings.Join(args, " ")
}

// probeStreams 用 ffprobe 检测输入文件的视频/音频流
func (a *App) probeStreams(path string) (hasVideo, hasAudio bool) {
	out, err := exec.Command(a.ffprobeBin(), "-v", "error", "-show_entries", "stream=codec_type", "-of", "default=noprint_wrappers=1", path).Output()
	if err != nil {
		return false, false
	}
	for _, line := range strings.Split(string(out), "\n") {
		switch strings.TrimSpace(line) {
		case "codec_type=video":
			hasVideo = true
		case "codec_type=audio":
			hasAudio = true
		}
	}
	return hasVideo, hasAudio
}

// clearFinishedTasks 清空已完成与失败的任务
func (a *App) clearFinishedTasks() {
	kept := make([]*Task, 0, len(a.tasks))
	for _, t := range a.tasks {
		if t.Status == "waiting" || t.Status == "running" {
			kept = append(kept, t)
		}
	}
	a.tasks = kept
	a.refreshTasks()
}

// probeDuration 用 ffprobe 获取媒体时长（秒）
func (a *App) probeDuration(path string) float64 {
	out, err := exec.Command(a.ffprobeBin(), "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1", path).Output()
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "duration=") {
			f, _ := strconv.ParseFloat(strings.TrimPrefix(line, "duration="), 64)
			return f
		}
	}
	return 0
}

// buildTasksPage 构建「任务」页：任务列表 + 底部操作栏
// （[添加任务] [开始] [清空] [复制命令]）
func (a *App) buildTasksPage() tview.Primitive {
	a.taskList = tview.NewList()
	a.selectOnSecondClick(a.taskList)

	addBtn := tview.NewButton(tview.Escape(a.s.TaskAdd))
	addBtn.SetSelectedFunc(func() { a.addTasksFromFiles() })
	startBtn := tview.NewButton(tview.Escape(a.s.TaskStart))
	startBtn.SetSelectedFunc(func() { a.startTasks() })
	clearBtn := tview.NewButton(tview.Escape(a.s.TaskClear))
	clearBtn.SetSelectedFunc(func() { a.clearFinishedTasks() })
	showBtn := tview.NewButton(tview.Escape(a.s.TaskShowCmd))
	showBtn.SetSelectedFunc(func() { a.showTaskCommand() })

	a.taskBarButtons = []tview.Primitive{addBtn, startBtn, clearBtn, showBtn}
	a.taskBarFocusables = []tview.Primitive{a.taskList, addBtn, startBtn, clearBtn, showBtn}

	bar := tview.NewFlex()
	bar.AddItem(addBtn, 0, 1, false)
	bar.AddItem(nil, 2, 0, false)
	bar.AddItem(startBtn, 0, 1, false)
	bar.AddItem(nil, 2, 0, false)
	bar.AddItem(clearBtn, 0, 1, false)
	bar.AddItem(nil, 2, 0, false)
	bar.AddItem(showBtn, 0, 1, false)
	bar.AddItem(nil, 0, 1, false)

	page := tview.NewFlex().SetDirection(tview.FlexRow)
	page.SetBorder(true)
	page.SetTitle(a.s.TasksTitle)
	page.AddItem(a.taskList, 0, 1, true)
	page.AddItem(bar, 1, 0, false)
	return page
}
