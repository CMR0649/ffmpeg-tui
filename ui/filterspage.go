package ui

import (
	"fmt"
	"path/filepath"
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
	a.filterVideoList.SetFocusFunc(func() { a.filterActive = filterChainVideo })

	a.filterAudioList = tview.NewList()
	a.filterAudioList.SetBorder(true)
	a.selectOnSecondClick(a.filterAudioList)
	a.filterAudioList.SetFocusFunc(func() { a.filterActive = filterChainAudio })

	addVideoBtn := tview.NewButton(tview.Escape(a.s.FilterAddVideo))
	addVideoBtn.SetSelectedFunc(func() { a.addFilter(filterChainVideo) })
	addAudioBtn := tview.NewButton(tview.Escape(a.s.FilterAddAudio))
	addAudioBtn.SetSelectedFunc(func() { a.addFilter(filterChainAudio) })
	upBtn := tview.NewButton(tview.Escape(a.s.FilterMoveUp))
	upBtn.SetSelectedFunc(func() { a.moveSelectedFilter(-1) })
	downBtn := tview.NewButton(tview.Escape(a.s.FilterMoveDown))
	downBtn.SetSelectedFunc(func() { a.moveSelectedFilter(1) })

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

// filterItemText 列表项文本：滤镜名 + 显式参数
func filterItemText(f FilterInstance) string {
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

// ---------- 参数表单对话框 ----------

// filterFormState 滤镜参数表单对话框的状态
type filterFormState struct {
	filter  string
	which   int
	idx     int // -1 表示新增
	params  []filterParamInfo
	values  map[string]string // 显式设置的参数
	page    int
	perPage int
	form    *tview.Form
	preview *tview.TextView
	pageLbl *tview.TextView
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

// openFilterForm 打开滤镜参数表单（idx=-1 为新增）
func (a *App) openFilterForm(which int, name string, idx int) {
	info := probeFilter(a.ffmpegBin(), name)
	if len(info.Params) == 0 && idx < 0 {
		// 无参数滤镜（如 hflip）：直接添加
		a.appendFilter(which, FilterInstance{Name: name})
		return
	}
	d := &filterFormState{
		filter: name,
		which:  which,
		idx:    idx,
		params: info.Params,
		values: map[string]string{},
	}
	if idx >= 0 {
		chain := a.filterChain(which)
		if idx < len(chain) {
			for _, p := range chain[idx].Params {
				d.values[p.Key] = p.Value
			}
		}
	}
	_, sh := a.screenSize()
	// 弹窗高度 = 字段数 + 8（页码 1 + 预览 2 + 边框与按钮 5）；
	// 高度上限为屏高-6，故每页字段数取屏高-14，保证上下各留 3 行
	d.perPage = sh - 14
	if d.perPage < 2 {
		d.perPage = 2
	}

	d.form = tview.NewForm()
	d.form.SetBorder(false)
	d.preview = tview.NewTextView()
	d.preview.SetWrap(true)
	d.preview.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	d.preview.SetTextStyle(tcell.StyleDefault.
		Foreground(tview.Styles.PrimaryTextColor).
		Background(tview.Styles.PrimitiveBackgroundColor))
	d.pageLbl = tview.NewTextView()
	d.pageLbl.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	d.pageLbl.SetTextStyle(tcell.StyleDefault.
		Foreground(tview.Styles.PrimaryTextColor).
		Background(tview.Styles.PrimitiveBackgroundColor))

	a.filterForm = d
	a.rebuildFilterForm()

	ok := tview.NewButton(tview.Escape("[" + a.s.OK + "]"))
	ok.SetSelectedFunc(func() { a.confirmFilterForm() })
	cancel := tview.NewButton(tview.Escape("[" + a.s.Cancel + "]"))
	cancel.SetSelectedFunc(func() { a.closeFilterForm() })

	body := tview.NewFlex().SetDirection(tview.FlexRow)
	body.AddItem(d.pageLbl, 1, 0, false)
	body.AddItem(d.form, 0, 1, true)
	body.AddItem(d.preview, 2, 0, false)

	h := len(d.params)
	if h > d.perPage {
		h = d.perPage
	}
	h += 8
	a.dialogButtons = []tview.Primitive{ok, cancel}
	a.showDialog(a.buildDialogBox(a.s.FilterParamsTitle+" "+name, body, 70, h, ok, cancel),
		[]tview.Primitive{d.form, ok, cancel}, d.form)
}

// rebuildFilterForm 按当前页重建表单控件并刷新预览
func (a *App) rebuildFilterForm() {
	d := a.filterForm
	if d == nil {
		return
	}
	d.form.Clear(true)
	start := d.page * d.perPage
	end := start + d.perPage
	if end > len(d.params) {
		end = len(d.params)
	}
	for i := start; i < end; i++ {
		a.addFilterFormItem(d, d.params[i])
	}
	a.updateFilterFormPageLabel()
	a.updateFilterPreview()
}

// addFilterFormItem 按参数类型添加控件
func (a *App) addFilterFormItem(d *filterFormState, p filterParamInfo) {
	label := a.filterParamLabel(p)
	if d := filterDefaultLabel(p); d != "" {
		label += " = " + d
	}
	switch {
	case p.Type == "boolean":
		cb := tview.NewCheckbox().SetLabel(label)
		cb.SetChecked(d.values[p.Key] != "")
		cb.SetChangedFunc(func(checked bool) {
			if checked {
				d.values[p.Key] = "1"
			} else {
				delete(d.values, p.Key)
			}
			a.updateFilterPreview()
		})
		d.form.AddFormItem(cb)
	case len(p.Enum) > 0:
		names := make([]string, len(p.Enum))
		sel := 0
		for i, e := range p.Enum {
			names[i] = e.Name
			if e.Name == d.values[p.Key] || (d.values[p.Key] == "" && e.Name == p.Default) {
				sel = i
			}
		}
		dd := tview.NewDropDown().SetLabel(label).SetOptions(names, nil)
		dd.SetCurrentOption(sel)
		dd.SetSelectedFunc(func(text string, _ int) {
			if text == p.Default || text == "" {
				delete(d.values, p.Key)
			} else {
				d.values[p.Key] = text
			}
			a.updateFilterPreview()
		})
		d.form.AddFormItem(dd)
	default:
		text := d.values[p.Key]
		if text == "" {
			text = p.Default
		}
		in := tview.NewInputField().SetLabel(label).SetText(text).SetFieldWidth(0)
		in.SetChangedFunc(func(v string) {
			v = strings.TrimSpace(v)
			if v == "" || v == p.Default {
				delete(d.values, p.Key)
			} else {
				d.values[p.Key] = v
			}
			a.updateFilterPreview()
		})
		if filterFilePathParam(d.filter, p.Key) {
			in.SetLabel(label + a.s.FilterPickHint)
			in.SetDoneFunc(func(key tcell.Key) {
				if key == tcell.KeyEnter {
					a.pickFilterFilePath(in)
				}
			})
		}
		d.form.AddFormItem(in)
	}
}

// pickFilterFilePath 用系统文件选择器填写文件路径参数（回车触发）
func (a *App) pickFilterFilePath(in *tview.InputField) {
	var paths []string
	a.tviewApp.Suspend(func() { paths, _ = runSystemFilePicker() })
	if len(paths) > 0 {
		in.SetText(filepath.ToSlash(paths[0]))
	}
}

// updateFilterFormPageLabel 刷新页码提示
func (a *App) updateFilterFormPageLabel() {
	d := a.filterForm
	if d == nil || d.pageLbl == nil {
		return
	}
	pages := (len(d.params) + d.perPage - 1) / d.perPage
	if pages < 1 {
		pages = 1
	}
	if pages <= 1 {
		d.pageLbl.SetText("")
		return
	}
	d.pageLbl.SetText(fmt.Sprintf("%s  %d/%d  (PgUp/PgDn)", a.s.FilterParamsTitle, d.page+1, pages))
}

// updateFilterPreview 刷新底部实时预览串
func (a *App) updateFilterPreview() {
	d := a.filterForm
	if d == nil || d.preview == nil {
		return
	}
	inst := FilterInstance{Name: d.filter}
	for _, p := range d.params {
		if v, ok := d.values[p.Key]; ok {
			inst.Params = append(inst.Params, FilterParam{Key: p.Key, Value: v})
		}
	}
	d.preview.SetText(a.s.FilterPreview + filterChainString([]FilterInstance{inst}))
}

// filterFormPrevPage 上一页参数
func (a *App) filterFormPrevPage() {
	if a.filterForm == nil || a.filterForm.page <= 0 {
		return
	}
	a.filterForm.page--
	a.rebuildFilterForm()
}

// filterFormNextPage 下一页参数
func (a *App) filterFormNextPage() {
	d := a.filterForm
	if d == nil {
		return
	}
	pages := (len(d.params) + d.perPage - 1) / d.perPage
	if d.page+1 >= pages {
		return
	}
	d.page++
	a.rebuildFilterForm()
}

// filterFormFocused 报告焦点是否在参数表单上（含其内部字段）
func (a *App) filterFormFocused() bool {
	d := a.filterForm
	if d == nil || d.form == nil {
		return false
	}
	focused := tview.Primitive(a.tviewApp.GetFocus())
	if focused == nil {
		return false
	}
	if focused == tview.Primitive(d.form) {
		return true
	}
	for i := 0; i < d.form.GetFormItemCount(); i++ {
		if tview.Primitive(d.form.GetFormItem(i)) == focused {
			return true
		}
	}
	return false
}

// confirmFilterForm 校验并应用表单
func (a *App) confirmFilterForm() {
	d := a.filterForm
	if d == nil {
		return
	}
	// 校验数值范围
	for _, p := range d.params {
		v, ok := d.values[p.Key]
		if !ok || p.Type == "boolean" || len(p.Enum) > 0 {
			continue
		}
		if isNumericType(p.Type) {
			f, err := parseFloatValue(v)
			if err != nil {
				a.showMessageDialog(a.s.Hint, fmt.Sprintf(a.s.FilterValueInvalid, p.Key, v))
				return
			}
			if p.HasRange && (f < p.Min || f > p.Max) {
				a.showMessageDialog(a.s.Hint, fmt.Sprintf(a.s.FilterValueInvalid, p.Key, v))
				return
			}
		}
	}
	inst := FilterInstance{Name: d.filter}
	for _, p := range d.params {
		if v, ok := d.values[p.Key]; ok {
			inst.Params = append(inst.Params, FilterParam{Key: p.Key, Value: v})
		}
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

// closeFilterForm 关闭参数表单对话框
func (a *App) closeFilterForm() {
	a.filterForm = nil
	a.closeDialog()
}

// isNumericType 报告参数类型是否数值
func isNumericType(t string) bool {
	switch t {
	case "int", "int64", "double", "float", "rational":
		return true
	}
	return false
}

// parseFloatValue 解析数值参数（支持整数、小数与分数）
func parseFloatValue(v string) (float64, error) {
	v = strings.TrimSpace(v)
	if i := strings.Index(v, "/"); i > 0 {
		a, err := parseFloatValue(v[:i])
		if err != nil {
			return 0, err
		}
		b, err := parseFloatValue(v[i+1:])
		if err != nil || b == 0 {
			return 0, fmt.Errorf("invalid rational")
		}
		return a / b, nil
	}
	var f float64
	_, err := fmt.Sscanf(v, "%g", &f)
	if err != nil {
		return 0, err
	}
	return f, nil
}

// filterFormItemCount 当前页表单字段数
func (a *App) filterFormItemCount() int {
	if a.filterForm == nil || a.filterForm.form == nil {
		return 0
	}
	return a.filterForm.form.GetFormItemCount()
}

// filterFormFocusedIndex 当前聚焦的表单字段下标
func (a *App) filterFormFocusedIndex() int {
	if a.filterForm == nil || a.filterForm.form == nil {
		return 0
	}
	item, _ := a.filterForm.form.GetFocusedItemIndex()
	return item
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
