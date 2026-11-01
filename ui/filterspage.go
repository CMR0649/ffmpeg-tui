package ui

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/mattn/go-runewidth"
	"github.com/rivo/tview"
)

// 滤镜链类型
const (
	filterChainVideo = iota
	filterChainAudio
)

// buildFiltersPage 构建「滤镜」页：视频链 + 音频链 + 底部操作栏
func (a *App) buildFiltersPage() tview.Primitive {
	a.filterVideoList = tview.NewList()
	a.filterVideoList.SetBorder(true)
	a.selectOnSecondClick(a.filterVideoList)
	a.filterVideoList.SetFocusFunc(func() { a.updateFilterListStyles() })

	a.filterAudioList = tview.NewList()
	a.filterAudioList.SetBorder(true)
	a.selectOnSecondClick(a.filterAudioList)
	a.filterAudioList.SetFocusFunc(func() { a.updateFilterListStyles() })

	addVideoBtn := tview.NewButton(tview.Escape(a.s.FilterAddVideo))
	addVideoBtn.SetSelectedFunc(func() { a.addFilter(filterChainVideo) })
	addAudioBtn := tview.NewButton(tview.Escape(a.s.FilterAddAudio))
	addAudioBtn.SetSelectedFunc(func() { a.addFilter(filterChainAudio) })
	upBtn := tview.NewButton(tview.Escape(a.s.FilterMoveUp))
	upBtn.SetSelectedFunc(func() { a.moveSelectedFilter(-1) })
	downBtn := tview.NewButton(tview.Escape(a.s.FilterMoveDown))
	downBtn.SetSelectedFunc(func() { a.moveSelectedFilter(1) })

	for _, b := range []*tview.Button{addVideoBtn, addAudioBtn, upBtn, downBtn} {
		b := b
		b.SetFocusFunc(func() { a.updateFilterListStyles() })
	}
	a.filterBarButtons = []tview.Primitive{addVideoBtn, addAudioBtn, upBtn, downBtn}
	a.filterBarFocusables = []tview.Primitive{
		a.filterVideoList, a.filterAudioList, addVideoBtn, addAudioBtn, upBtn, downBtn,
	}

	a.filterCountLabel = tview.NewTextView()
	a.filterCountLabel.SetTextAlign(tview.AlignRight)
	a.filterCountLabel.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	a.filterCountLabel.SetTextStyle(tcell.StyleDefault.
		Foreground(tview.Styles.PrimaryTextColor).
		Background(tview.Styles.PrimitiveBackgroundColor))

	// 按钮宽度按标签长度分配权重，避免长英文标签被裁切
	weight := func(label string) int { return runewidth.StringWidth(label) + 4 }
	bar := tview.NewFlex()
	bar.AddItem(addVideoBtn, 0, weight(a.s.FilterAddVideo), false)
	bar.AddItem(nil, 2, 0, false)
	bar.AddItem(addAudioBtn, 0, weight(a.s.FilterAddAudio), false)
	bar.AddItem(nil, 2, 0, false)
	bar.AddItem(upBtn, 0, weight(a.s.FilterMoveUp), false)
	bar.AddItem(nil, 2, 0, false)
	bar.AddItem(downBtn, 0, weight(a.s.FilterMoveDown), false)
	bar.AddItem(a.filterCountLabel, 20, 0, false)

	page := tview.NewFlex().SetDirection(tview.FlexRow)
	page.SetBorder(true)
	page.SetTitle(a.s.FiltersTitle)
	page.AddItem(a.filterVideoList, 0, 1, true)
	page.AddItem(a.filterAudioList, 0, 1, false)
	page.AddItem(bar, 1, 0, false)
	a.filtersPage = page

	a.refreshFiltersPage()
	a.updateFilterListStyles()
	return page
}

// filterChain 返回指定链
func (a *App) filterChain(which int) []FilterInstance {
	if which == filterChainAudio {
		return a.cfg.AudioFilters
	}
	return a.cfg.VideoFilters
}

// setFilterChain 写回指定链
func (a *App) setFilterChain(which int, chain []FilterInstance) {
	if which == filterChainAudio {
		a.cfg.AudioFilters = chain
	} else {
		a.cfg.VideoFilters = chain
	}
}

// filterListFor 返回指定链对应的列表
func (a *App) filterListFor(which int) *tview.List {
	if which == filterChainAudio {
		return a.filterAudioList
	}
	return a.filterVideoList
}

// focusedFilterChain 返回当前焦点所在（或最近使用）的链
func (a *App) focusedFilterChain() int {
	if a.tviewApp.GetFocus() == a.filterAudioList {
		return filterChainAudio
	}
	if a.tviewApp.GetFocus() == a.filterVideoList {
		return filterChainVideo
	}
	return a.filterActive
}

// focusedFilterList 返回当前应操作的链列表（供 pageList 使用）
func (a *App) focusedFilterList() *tview.List {
	return a.filterListFor(a.focusedFilterChain())
}

// updateFilterListStyles 设置两条链的选中样式：
// 焦点所在链表为白底蓝字；焦点不在任一列表时，最近操作的链表为白底黑字；
// 另一条链不显示选中，保证同一时刻只有一处可见选中
func (a *App) updateFilterListStyles() {
	if a.filterVideoList == nil || a.filterAudioList == nil {
		return
	}
	focused := a.tviewApp.GetFocus()
	if focused == a.filterVideoList {
		a.filterActive = filterChainVideo
	}
	if focused == a.filterAudioList {
		a.filterActive = filterChainAudio
	}
	a.applyFilterListStyle(a.filterVideoList, filterChainVideo, focused == a.filterVideoList)
	a.applyFilterListStyle(a.filterAudioList, filterChainAudio, focused == a.filterAudioList)
}

// 链列表选中样式
const (
	filterSelNone    = iota // 不显示选中
	filterSelActive         // 选中但列表无焦点（白底黑字）
	filterSelFocused        // 选中且列表有焦点（白底蓝字）
)

// filterSelectionStyle 返回某条链应使用的选中样式：
// 有焦点 → 白底蓝字；无焦点但属于最近操作的链 → 白底黑字；其余不显示选中
func (a *App) filterSelectionStyle(which int, focused bool) int {
	switch {
	case focused:
		return filterSelFocused
	case which == a.filterActive:
		return filterSelActive
	}
	return filterSelNone
}

// applyFilterListStyle 应用单条链的选中样式
func (a *App) applyFilterListStyle(l *tview.List, which int, focused bool) {
	switch a.filterSelectionStyle(which, focused) {
	case filterSelFocused:
		l.SetSelectedTextColor(tcell.ColorBlue)
		l.SetSelectedBackgroundColor(tcell.ColorWhite)
	case filterSelActive:
		l.SetSelectedTextColor(tcell.ColorBlack)
		l.SetSelectedBackgroundColor(tcell.ColorWhite)
	default:
		l.SetSelectedTextColor(tview.Styles.PrimaryTextColor)
		l.SetSelectedBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	}
}

// filterChainTitle 链标题：复制流且链非空时加警告前缀
func (a *App) filterChainTitle(which int) string {
	title, chain, enc := a.s.FilterVideoChain, a.cfg.VideoFilters, a.cfg.VideoEncoder
	if which == filterChainAudio {
		title, chain, enc = a.s.FilterAudioChain, a.cfg.AudioFilters, a.cfg.AudioEncoder
	}
	if len(chain) > 0 && enc == "" {
		return a.s.FilterCopyWarn + title
	}
	return title
}

// refreshFiltersPage 重建两条链的列表、标题与计数
func (a *App) refreshFiltersPage() {
	a.refreshFilterList(filterChainVideo)
	a.refreshFilterList(filterChainAudio)
	if a.filterVideoList != nil {
		a.filterVideoList.SetTitle(a.filterChainTitle(filterChainVideo))
	}
	if a.filterAudioList != nil {
		a.filterAudioList.SetTitle(a.filterChainTitle(filterChainAudio))
	}
	if a.filterCountLabel != nil {
		a.filterCountLabel.SetText(fmt.Sprintf(a.s.FilterCount, len(a.cfg.VideoFilters), len(a.cfg.AudioFilters)))
	}
}

// refreshFilterList 重建某条链的列表（空链显示占位项）
func (a *App) refreshFilterList(which int) {
	l := a.filterListFor(which)
	if l == nil {
		return
	}
	l.Clear()
	chain := a.filterChain(which)
	if len(chain) == 0 {
		l.AddItem(a.s.FilterEmpty, "", 0, func() { a.addFilter(which) })
		return
	}
	for i := range chain {
		i := i
		l.AddItem(tview.Escape(filterItemText(chain[i])), "", 0, func() { a.editFilter(which, i) })
	}
}

// filterItemText 列表项文本：滤镜名 + 手动参数或显式参数
func filterItemText(f FilterInstance) string {
	if f.Manual != "" {
		return f.Name + "  " + f.Manual
	}
	if len(f.Params) == 0 {
		return f.Name
	}
	ps := make([]string, 0, len(f.Params))
	for _, p := range f.Params {
		ps = append(ps, p.Key+"="+p.Value)
	}
	return f.Name + "  " + strings.Join(ps, "  ")
}

// addFilter 选择并添加一个滤镜到指定链
func (a *App) addFilter(which int) {
	loadFilterLists(a.ffmpegBin())
	defs := filterVideoDefs
	title := a.s.FilterAddVideo
	if which == filterChainAudio {
		defs, title = filterAudioDefs, a.s.FilterAddAudio
	}
	if len(defs) == 0 {
		a.showMessageDialog(a.s.Hint, a.s.FilterListFailed)
		return
	}
	labels := make([]string, len(defs))
	for i, d := range defs {
		labels[i] = d.Name + "  " + d.Desc
	}
	a.showOptionDialog(title, labels, func(i int) {
		a.openFilterForm(which, defs[i].Name, -1)
	})
}

// addFilterFocused 对当前链打开滤镜选择器（"/" 快捷键）
func (a *App) addFilterFocused() {
	a.addFilter(a.focusedFilterChain())
}

// editFilter 重新编辑链中第 idx 个滤镜的参数
func (a *App) editFilter(which, idx int) {
	chain := a.filterChain(which)
	if idx < 0 || idx >= len(chain) {
		return
	}
	a.openFilterForm(which, chain[idx].Name, idx)
}

// deleteSelectedFilter 删除焦点所在链的选中滤镜；始终消费按键
func (a *App) deleteSelectedFilter() bool {
	which := a.focusedFilterChain()
	chain := a.filterChain(which)
	l := a.filterListFor(which)
	if l == nil || len(chain) == 0 {
		return true
	}
	idx := l.GetCurrentItem()
	if idx < 0 || idx >= len(chain) {
		return true
	}
	nchain := append([]FilterInstance{}, chain[:idx]...)
	nchain = append(nchain, chain[idx+1:]...)
	a.setFilterChain(which, nchain)
	a.refreshFiltersPage()
	return true
}

// moveSelectedFilter 上移（delta=-1）/下移（delta=1）选中的滤镜
func (a *App) moveSelectedFilter(delta int) {
	which := a.focusedFilterChain()
	chain := a.filterChain(which)
	l := a.filterListFor(which)
	if l == nil || len(chain) < 2 {
		return
	}
	idx := l.GetCurrentItem()
	j := idx + delta
	if idx < 0 || idx >= len(chain) || j < 0 || j >= len(chain) {
		return
	}
	chain[idx], chain[j] = chain[j], chain[idx]
	a.refreshFiltersPage()
	if id := a.filterListFor(which).GetCurrentItem(); id != j {
		a.filterListFor(which).SetCurrentItem(j)
	}
}

// ---------- 滤镜参数对话框 ----------

// filterFormState 滤镜参数对话框状态
type filterFormState struct {
	filter  string
	which   int
	idx     int               // -1 表示新增
	params  []filterParamInfo // 只读参数提示
	manual  string            // 滤镜参数（唯一来源）
	field   *tview.InputField // 滤镜参数输入框
	hints   *tview.TextView
	preview *tview.TextView
	ok      *tview.Button
	cancel  *tview.Button
	hintTop int // 参数提示的滚动行
}

// appendFilter 把滤镜追加到链尾并刷新页面
func (a *App) appendFilter(which int, inst FilterInstance) {
	chain := append(a.filterChain(which), inst)
	a.setFilterChain(which, chain)
	a.refreshFiltersPage()
	l := a.filterListFor(which)
	if l != nil {
		l.SetCurrentItem(len(chain) - 1)
	}
}

// openFilterForm 打开滤镜参数对话框（idx=-1 为新增）
func (a *App) openFilterForm(which int, name string, idx int) {
	info := probeFilter(a.ffmpegBin(), name)
	d := &filterFormState{filter: name, which: which, idx: idx, params: info.Params}
	if idx >= 0 {
		chain := a.filterChain(which)
		if idx < len(chain) {
			d.manual = chain[idx].Manual
			// 旧配置里按参数保存的滤镜：转换成滤镜参数文本
			if d.manual == "" && len(chain[idx].Params) > 0 {
				d.manual = paramsToManual(chain[idx])
			}
		}
	}
	a.showFilterFormDialog(d)
}

// showFilterFormDialog 显示（或重新显示）滤镜参数对话框：
// 上方是唯一的「滤镜参数」输入框，中间是只读参数提示，底部实时预览
func (a *App) showFilterFormDialog(d *filterFormState) {
	if d == nil {
		return
	}
	d.field = tview.NewInputField().
		SetLabel(a.s.FilterParams).
		SetText(d.manual)
	d.field.SetFieldWidth(0)
	d.field.SetChangedFunc(func(v string) {
		d.manual = strings.TrimSpace(v)
		a.updateFilterPreview()
	})
	d.field.SetDoneFunc(func(k tcell.Key) {
		if k == tcell.KeyEnter {
			a.confirmFilterForm()
		}
	})

	d.hints = tview.NewTextView()
	d.hints.SetScrollable(true)
	d.hints.SetWrap(false)
	d.hints.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	d.hints.SetTextStyle(tcell.StyleDefault.
		Foreground(tview.Styles.SecondaryTextColor).
		Background(tview.Styles.PrimitiveBackgroundColor))
	d.hints.SetText(a.filterHintsText(d))
	d.hints.ScrollTo(d.hintTop, 0)

	d.preview = tview.NewTextView()
	d.preview.SetWrap(true)
	d.preview.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	d.preview.SetTextStyle(tcell.StyleDefault.
		Foreground(tview.Styles.PrimaryTextColor).
		Background(tview.Styles.PrimitiveBackgroundColor))
	a.updateFilterPreview()

	d.ok = tview.NewButton(tview.Escape("[" + a.s.OK + "]"))
	d.ok.SetSelectedFunc(func() { a.confirmFilterForm() })
	d.cancel = tview.NewButton(tview.Escape("[" + a.s.Cancel + "]"))
	d.cancel.SetSelectedFunc(func() { a.closeFilterForm() })

	a.filterForm = d
	body := tview.NewFlex().SetDirection(tview.FlexRow)
	body.AddItem(d.field, 1, 0, true)
	body.AddItem(d.hints, 0, 1, false)
	body.AddItem(d.preview, 2, 0, false)

	// 高度：输入框 1 + 提示若干行 + 预览 2 + 边框与按钮 5；上限为屏高-6
	_, sh := a.screenSize()
	hintRows := sh - 6 - 8
	if hintRows < 3 {
		hintRows = 3
	}
	if n := len(d.params) + 1; n < hintRows {
		hintRows = n
	}
	a.dialogButtons = []tview.Primitive{d.ok, d.cancel}
	a.showDialog(a.buildDialogBox(a.s.FilterParamsTitle+" "+d.filter, body, 70, hintRows+8, d.ok, d.cancel),
		[]tview.Primitive{d.field, d.ok, d.cancel}, d.field)
}

// openFilterFormState 用已有状态重新显示滤镜参数对话框
func (a *App) openFilterFormState(d *filterFormState) {
	if d == nil {
		return
	}
	a.showFilterFormDialog(d)
}

// filterHintsText 生成只读的参数提示文本
func (a *App) filterHintsText(d *filterFormState) string {
	var b strings.Builder
	b.WriteString(a.s.FilterParamsHint)
	if len(d.params) == 0 {
		b.WriteString("\n" + a.s.FilterNoParams)
		return b.String()
	}
	for _, p := range d.params {
		line := "  " + a.filterParamLabel(p)
		if def := filterDefaultLabel(p); def != "" {
			line += " = " + def
		}
		b.WriteString("\n" + line)
	}
	return b.String()
}

// paramsToManual 把按参数保存的滤镜转成滤镜参数文本（兼容旧配置）
func paramsToManual(f FilterInstance) string {
	s := filterChainString([]FilterInstance{{Name: f.Name, Params: f.Params}})
	if rest, ok := strings.CutPrefix(s, f.Name+"="); ok {
		return rest
	}
	return ""
}

// updateFilterPreview 刷新底部实时预览串
func (a *App) updateFilterPreview() {
	d := a.filterForm
	if d == nil || d.preview == nil {
		return
	}
	inst := FilterInstance{Name: d.filter}
	if m := strings.TrimSpace(d.manual); m != "" {
		inst.Manual = m
	}
	d.preview.SetText(a.s.FilterPreview + filterChainString([]FilterInstance{inst}))
}

// filterFormFocused 报告焦点是否在「滤镜参数」输入框上
func (a *App) filterFormFocused() bool {
	d := a.filterForm
	return d != nil && d.field != nil && tview.Primitive(a.tviewApp.GetFocus()) == tview.Primitive(d.field)
}

// confirmFilterForm 应用滤镜参数
func (a *App) confirmFilterForm() {
	d := a.filterForm
	if d == nil {
		return
	}
	inst := FilterInstance{Name: d.filter}
	if m := strings.TrimSpace(d.manual); m != "" {
		inst.Manual = m
	}
	chain := a.filterChain(d.which)
	if d.idx >= 0 && d.idx < len(chain) {
		chain[d.idx] = inst
	} else {
		chain = append(chain, inst)
	}
	a.setFilterChain(d.which, chain)
	a.closeFilterForm()
	a.refreshFiltersPage()
}

// closeFilterForm 关闭滤镜参数对话框
func (a *App) closeFilterForm() {
	a.filterForm = nil
	a.closeDialog()
}

// filterFormAtEnd 报告焦点是否已在滤镜参数输入框（Tab 即可离开）
func (a *App) filterFormAtEnd() bool { return true }

// filterFormAtStart 报告焦点是否已在滤镜参数输入框（Shift-Tab 即可离开）
func (a *App) filterFormAtStart() bool { return true }

// scrollFilterHints 滚动参数提示
func (a *App) scrollFilterHints(delta int) bool {
	d := a.filterForm
	if d == nil || d.hints == nil {
		return false
	}
	_, _, _, h := d.hints.GetInnerRect()
	step := h
	if step < 1 {
		step = 1
	}
	d.hintTop += delta * step
	if d.hintTop < 0 {
		d.hintTop = 0
	}
	d.hints.ScrollTo(d.hintTop, 0)
	return true
}

// ---------- 复制流与滤镜的冲突 ----------

// filterCopyConflict 报告是否存在「滤镜链非空 + 编码器为复制流」
func (a *App) filterCopyConflict() bool {
	return (len(a.cfg.VideoFilters) > 0 && a.cfg.VideoEncoder == "") ||
		(len(a.cfg.AudioFilters) > 0 && a.cfg.AudioEncoder == "")
}

// withFilterEncoding 无冲突时直接执行 fn；有冲突时先确认：
// 确定 → 切换默认编码并刷新视频/音频页后执行 fn；取消 → 不执行
func (a *App) withFilterEncoding(fn func()) {
	if !a.filterCopyConflict() {
		fn()
		return
	}
	a.showConfirmDialog(a.s.FilterNeedEncode, func() {
		def := DefaultConfig()
		if len(a.cfg.VideoFilters) > 0 && a.cfg.VideoEncoder == "" {
			a.cfg.VideoEncoder = def.VideoEncoder
		}
		if len(a.cfg.AudioFilters) > 0 && a.cfg.AudioEncoder == "" {
			a.cfg.AudioEncoder = def.AudioEncoder
		}
		if a.videoList != nil {
			a.refreshVideoPage()
		}
		if a.audioList != nil {
			a.refreshAudioPage()
		}
		a.refreshFiltersPage()
		fn()
	})
}
