package ui

import (
	"bufio"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/rivo/tview"
)

// Task 一个转码任务。
type Task struct {
	Input    string
	Output   string
	Status   string // 等待中 / 转码中 / 已完成 / 失败
	Progress float64
	duration float64
	index    int
}

// timeRe 匹配 ffmpeg 进度输出中的 time=HH:MM:SS.xx。
var timeRe = regexp.MustCompile(`time=(\d+):(\d+):(\d+\.?\d*)`)

// addTasksFromFiles 把文件页的文件列表生成为转码任务。
func (a *App) addTasksFromFiles() {
	if len(a.files) == 0 {
		a.showMessageDialog("提示", "请先在文件页添加输入文件。")
		return
	}
	a.tasks = nil
	for _, input := range a.files {
		t := &Task{
			Input:    input,
			Output:   a.outputPath(input),
			Status:   "等待中",
			duration: probeDuration(input),
		}
		a.tasks = append(a.tasks, t)
	}
	a.refreshTasks()
}

// refreshTasks 重建任务列表。
func (a *App) refreshTasks() {
	a.taskList.Clear()
	for _, t := range a.tasks {
		t.index = a.taskList.GetItemCount()
		a.taskList.AddItem(taskMainText(t), taskSecondaryText(t), 0, nil)
	}
}

func taskMainText(t *Task) string {
	return filepath.Base(t.Input) + " → " + filepath.Base(t.Output)
}

func taskSecondaryText(t *Task) string {
	switch t.Status {
	case "已完成":
		return "已完成 · 100%"
	case "转码中":
		return fmt.Sprintf("转码中 · %.0f%%", t.Progress)
	case "失败":
		return "失败"
	default:
		return "等待中"
	}
}

// updateTask 更新单个任务在列表中的显示。
func (a *App) updateTask(t *Task) {
	a.tviewApp.QueueUpdateDraw(func() {
		a.taskList.SetItemText(t.index, taskMainText(t), taskSecondaryText(t))
	})
}

// startTasks 串行执行所有等待中的任务。
func (a *App) startTasks() {
	go func() {
		for _, t := range a.tasks {
			if t.Status != "等待中" {
				continue
			}
			t.Status = "转码中"
			a.updateTask(t)
			err := a.runTask(t)
			if err != nil {
				t.Status = "失败"
				t.Progress = 0
			} else {
				t.Status = "已完成"
				t.Progress = 100
			}
			a.updateTask(t)
		}
	}()
}

// runTask 执行单个转码任务并解析进度。
func (a *App) runTask(t *Task) error {
	cmd := exec.Command("ffmpeg", a.buildCommand(t.Input, t.Output)...)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	scanner := bufio.NewScanner(stderr)
	for scanner.Scan() {
		if m := timeRe.FindStringSubmatch(scanner.Text()); m != nil && t.duration > 0 {
			h, _ := strconv.Atoi(m[1])
			min, _ := strconv.Atoi(m[2])
			sec, _ := strconv.ParseFloat(m[3], 64)
			elapsed := float64(h*3600+min*60) + sec
			t.Progress = elapsed / t.duration * 100
			if t.Progress > 100 {
				t.Progress = 100
			}
			a.updateTask(t)
		}
	}
	if err := cmd.Wait(); err != nil {
		return err
	}
	return nil
}

// clearFinishedTasks 清空已完成与失败的任务。
func (a *App) clearFinishedTasks() {
	kept := make([]*Task, 0, len(a.tasks))
	for _, t := range a.tasks {
		if t.Status == "等待中" || t.Status == "转码中" {
			kept = append(kept, t)
		}
	}
	a.tasks = kept
	a.refreshTasks()
}

// probeDuration 用 ffprobe 获取媒体时长（秒）。
func probeDuration(path string) float64 {
	name := "ffprobe"
	if runtime.GOOS == "windows" {
		name = "ffprobe.exe"
	}
	out, err := exec.Command(name, "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1", path).Output()
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

// buildTasksPage 构建「任务」页：任务列表 + 底部操作栏（[添加任务] [开始] [清空]）。
func (a *App) buildTasksPage() tview.Primitive {
	a.taskList = tview.NewList()

	addBtn := tview.NewButton("[添加任务]")
	addBtn.SetSelectedFunc(func() { a.addTasksFromFiles() })
	startBtn := tview.NewButton("[开始]")
	startBtn.SetSelectedFunc(func() { a.startTasks() })
	clearBtn := tview.NewButton("[清空]")
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
	page.SetTitle(" 任务 ")
	page.AddItem(a.taskList, 0, 1, true)
	page.AddItem(bar, 1, 0, false)
	return page
}
