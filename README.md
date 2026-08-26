# ffmpeg-tui

一个用于 FFmpeg 的终端用户界面（TUI）。当前版本：**beta0.1**（界面演示 Demo）。

## 功能（beta0.1）

- 标签页式界面：`[文件]` `[视频]` `[音频]` `[任务]` `[设置]`
- 当前选中的标签以 **白色背景** 高亮，作为光标指示
- 底部按键提示：`A/D：切换    方向键：选择`
- 支持鼠标点击标签栏切换页面（终端需支持鼠标事件）
- 本版本仅演示界面与交互，**不执行真实转码**

## 技术选型

使用 **Go** 编写，理由：

- 编译为单一静态可执行文件，**无需安装任何运行时**（满足跨平台分发要求）
- 原生支持 Windows / Linux 交叉编译（纯 Go，无 cgo）
- 终端 UI 基于 [tview](https://github.com/rivo/tview)（纯 Go，跨平台处理终端原始模式、颜色、鼠标）

## 构建

两种方式任选其一：

### 方式一：本地构建（需安装 Go 工具链）

- Linux/macOS：`./build.sh`
- Windows：`build.bat`

### 方式二：容器构建（推荐，本仓库约定）

见 `../ffmpeg-tui-container/`，通过 podman（rootless）在容器中构建，
生成物输出到 `bin/` 目录。

两个脚本都会生成：

| 平台 | 产物 |
| --- | --- |
| Linux | `bin/ffmpeg-tui` |
| Windows | `bin/ffmpeg-tui.exe` |

## 运行

```bash
# Linux
./bin/ffmpeg-tui

# Windows（cmd 或 PowerShell）
bin\ffmpeg-tui.exe
```

## 按键

| 按键 | 功能 |
| --- | --- |
| `A` / `D` | 切换标签页 |
| `↑` / `↓` | 在当前页面的列表中选择项目 |
| `Q` / `Esc` / `Ctrl+C` | 退出 |

## 目录结构

```
ffmpeg-tui/
├── main.go          # 入口
├── ui/
│   ├── app.go       # 界面组装：标签栏 / 内容区 / 底部提示 / 按键与鼠标
│   └── pages.go     # 五个标签页的内容
├── build.sh         # Linux 构建脚本（含交叉编译 Windows）
├── build.bat        # Windows 构建脚本（含交叉编译 Linux）
└── README.md
```

## 路线图

- [ ] 真实文件选择与 FFmpeg 命令生成
- [ ] 任务队列与进度显示（解析 ffmpeg 进度输出）
- [ ] 选项修改与预设管理
