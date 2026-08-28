package ui

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/rivo/tview"
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
	frames, bitrate, speed string
}

// frameRe 匹配 ffmpeg 进度输出中以 frame= 开头的行，提取 fps/bitrate/speed
var frameRe = regexp.MustCompile(`frame=\s*\S+\s+fps=\s*(\S+).*?bitrate=\s*(\S+).*?speed=\s*(\S+)`)

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
		t.index = a.taskList.GetItemCount()
		a.taskList.AddItem(taskMainText(t), a.taskSecondaryText(t), 0, nil)
	}
}

func taskMainText(t *Task) string {
	return filepath.Base(t.Input) + " → " + filepath.Base(t.Output)
}

func (a *App) taskSecondaryText(t *Task) string {
	switch t.Status {
	case "done":
		return a.s.TaskDone + " · 100%"
	case "running":
		return a.taskProgressText(t)
	case "failed":
		if t.ErrMsg != "" {
			return a.s.TaskFailed + "：" + t.ErrMsg
		}
		return a.s.TaskFailed
	default:
		return a.s.TaskWaiting
	}
}

// taskProgressText 转码中的实时数据显示：仅接受以 frame= 开头的行，
// 展示 fps/bitrate/speed；中文用对应汉字段名，英文用原文
func (a *App) taskProgressText(t *Task) string {
	if t.frames == "" && t.bitrate == "" && t.speed == "" {
		return a.s.TaskRunning
	}
	var parts []string
	if t.frames != "" {
		parts = append(parts, a.s.FrameFPS+"="+t.frames)
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

// runTask 执行单个转码任务并解析进度；失败时返回 ffmpeg 的错误摘要
func (a *App) runTask(t *Task) error {
	cmd := exec.Command(a.ffmpegBin(), a.buildCommand(t.Input, t.Output, t.hasVideo, t.hasAudio)...)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	lastErr := ""
	scanner := bufio.NewScanner(stderr)
	scanner.Split(scanProgress)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "frame=") {
			if m := frameRe.FindStringSubmatch(line); m != nil {
				t.frames, t.bitrate, t.speed = m[1], m[2], m[3]
				a.updateTask(t)
			}
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
		if lastErr == "" {
			return err
		}
		return fmt.Errorf("%s", lastErr)
	}
	return nil
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

// buildTasksPage 构建「任务」页：任务列表 + 底部操作栏（[添加任务] [开始] [清空]）
func (a *App) buildTasksPage() tview.Primitive {
	a.taskList = tview.NewList()

	addBtn := tview.NewButton(tview.Escape(a.s.TaskAdd))
	addBtn.SetSelectedFunc(func() { a.addTasksFromFiles() })
	startBtn := tview.NewButton(tview.Escape(a.s.TaskStart))
	startBtn.SetSelectedFunc(func() { a.startTasks() })
	clearBtn := tview.NewButton(tview.Escape(a.s.TaskClear))
	clearBtn.SetSelectedFunc(func() { a.clearFinishedTasks() })

	a.taskBarButtons = []tview.Primitive{addBtn, startBtn, clearBtn}
	a.taskBarFocusables = []tview.Primitive{a.taskList, addBtn, startBtn, clearBtn}

	bar := tview.NewFlex()
	bar.AddItem(addBtn, 0, 1, false)
	bar.AddItem(nil, 2, 0, false)
	bar.AddItem(startBtn, 0, 1, false)
	bar.AddItem(nil, 2, 0, false)
	bar.AddItem(clearBtn, 0, 1, false)
	bar.AddItem(nil, 0, 1, false)

	page := tview.NewFlex().SetDirection(tview.FlexRow)
	page.SetBorder(true)
	page.SetTitle(a.s.TasksTitle)
	page.AddItem(a.taskList, 0, 1, true)
	page.AddItem(bar, 1, 0, false)
	return page
}
