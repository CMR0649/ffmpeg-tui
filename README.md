# ffmpeg-tui

一个用于 FFmpeg 的 TUI

> [!NOTE]
> 如果你是Windows用户，我更推荐你使用[FFmpegFreeUI](https://github.com/Lake1059/FFmpegFreeUI)

## 构建
安装go（>=1.24.0）  
运行build.sh（Windows需运行build.bat）

## 运行
一般来说，双击可执行文件即可，如果无法运行，请在终端中执行
> [!NOTE]
> Windows用户建议使用[Windows Terminal](https://apps.microsoft.com/detail/9n0dx20hk701)

## 按键

| 按键 | 功能 |
| --- | --- |
| `A` / `D` | 切换标签页 |
| `↑` / `↓` | 在当前页面的列表中选择项目 |
| `←` / `→` | 在横向排列的选项间切换焦点（文件页底部按钮、对话框 `[确定]`/`[取消]`） |
| `Tab` / `Shift-Tab` | 文件页：列表 ↔ 输出目录 ↔ `[添加文件]` ↔ `[设置输出容器]`；对话框内：切换焦点 |
| `Delete` | 文件页：移除当前选中的文件|
| `Enter` | 激活按钮 / 确认对话框（输入框内确认） |
| `Esc` | 关闭对话框；无对话框时退出 |
| `Q` / `Ctrl+C` | 退出 |
