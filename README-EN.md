# ffmpeg-tui
[中文](https://github.com/CMR0649/ffmpeg-tui/blob/main/README.md)

> [!NOTE]
> This article is translated using AI

A TUI for FFmpeg

## Build
Install Go (>=1.24.0)  
Run build.sh (Windows users should run build.bat)

## Run
In general, just double-click the executable file. If it doesn't run, execute it in the terminal.
> [!TIP]
> Windows users are advised to use [Windows Terminal](https://apps.microsoft.com/detail/9n0dx20hk701)

## Key Bindings

| Key | Function |
| --- | --- |
| `A` / `D` | Switch tabs |
| `↑` / `↓` | Select an item in the list on the current page |
| `←` / `→` | Switch focus between horizontally arranged options (buttons at the bottom of the file page, dialog `[OK]`/`[Cancel]`) |
| `Tab` / `Shift-Tab` | Cycle focus on the current page |
| `Delete` | File page: remove the currently selected file |
| `Enter` | Activate button / confirm dialog (confirm within input fields) |
| `Esc` | Close dialog; exit if no dialog is open |
| `Q` / `Esc` | Exit |
