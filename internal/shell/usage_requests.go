//go:build nogui && echo

package shell

// The Usage page's Requests segment: every call the gateway and the
// agents' session files tell of — its totals, its trend by provider,
// agent or model, the last 53 weeks a day each, and the calls themselves
// a page at a time, with the filters magpie's own page offers.

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/yetone/magpie/internal/usage"
	"github.com/yetone/magpie/internal/usageapi"
)

// usChoice is one of a strip's choices: the id the query takes and the
// word it shows.
type usChoice struct {
	id    string
	label string
}

// usLedMetrics are the trend's measures, as magpie's chart offers them.
var usLedMetrics = []usChoice{
	{"tokens", "Token"},
	{"cost", "费用"},
	{"calls", "请求数"},
	{"speed", "速度"},
}

// usLedSplits are the ways the trend tells its points apart.
var usLedSplits = []usChoice{
	{"model", "模型"},
	{"provider", "供应商"},
	{"agent", "Agent"},
}

// usHeatMetrics are what a heatmap day is coloured by.
var usHeatMetrics = []usChoice{
	{"tokens", "Token"},
	{"calls", "请求数"},
	{"cost", "费用"},
}

// usCol is one column of the requests table.
type usCol struct {
	title string
	width float32
}

// usSeriesColors are the trend's palette: a provider, agent or model keeps
// the colour its rank gives it, so a chart read twice reads the same.
var usSeriesColors = []ui.Color{
	ui.Hex("#007aff"), ui.Hex("#34c759"), ui.Hex("#ff9500"), ui.Hex("#af52de"),
	ui.Hex("#5ac8fa"), ui.Hex("#ff3b30"), ui.Hex("#ffcc00"), ui.Hex("#8e8e93"),
}

func (a *App) usageRequests(c *ui.Context, t *ui.Theme) {
	d := a.data.Usage
	if d.Query.Tab != UsTabRequests {
		a.usSkeleton(c, t)
		return
	}
	l := d.Ledger
	a.usReqFilters(c, t, l)
	if l.Total == 0 {
		filtered := a.usFiltered()
		if filtered {
			a.usEmpty(c, t, "没有符合这些筛选条件的请求。", "清掉筛选条件，或换一段时间。")
		} else {
			a.usEmpty(c, t, a.usNoCallsIn(a.usPeriod), "经过网关的调用，以及 Agent 自己会话文件里的调用，都会列在这里。")
		}
		return
	}
	a.usReqKPIs(c, t, l)
	a.usReqTrend(c, t, l)
	a.usReqHeat(c, t, d.Heatmap)
	a.usReqVias(c, t, l)
	a.usReqTable(c, t, l)
	a.usNote(c, t, "每一行的 Token 是供应商上报的用量；费用按有效模型价格估算（含自定义价格）。导出 CSV 会保存所有分页。")
}

// usFiltered says whether any filter but the period is on.
func (a *App) usFiltered() bool {
	return a.usSelectedDay != "" || a.usAgentFilter != "" || a.usProviderFilter != "" ||
		a.usPurposeFilter != "" || a.usAccountFilter != "" || a.usKeyFilter != "" ||
		a.usModelFilter != "" || a.usViaFilter != "" || a.usStatusFilter != "" ||
		strings.TrimSpace(a.usSearchQuery) != ""
}

// usReqFilters is the row of filters: the status segments, one picker an
// agent, provider, account and key, the search, and the export.
func (a *App) usReqFilters(c *ui.Context, t *ui.Theme, l usageapi.Ledger) {
	// the status strip, as magpie's: every request, the failed ones, and
	// those answered after a try that failed first
	for i, s := range []string{"", "failed", "retried"} {
		if s == a.usStatusFilter {
			a.usStatusIdx = i
		}
	}
	ui.Row(c).AlignItems(ui.Center).Gap(8).Padding(10, 20, 6, 20).FillWidth().Children(func() {
		if ui.Segmented(c, &a.usStatusIdx, "全部", "失败", "重试后成功").Changed() {
			a.usStatusFilter = []string{"", "failed", "retried"}[a.usStatusIdx]
			a.usLedOffset = 0
			a.usFresh = ""
			c.Invalidate()
		}

		a.usPick(c, t, "全部 Agents", a.usAgentFilter, usAgentPick(l.Agents), a.usAgentFilter != "", func(v string) {
			a.usAgentFilter = v
			a.usLedOffset = 0
			a.usFresh = ""
		})
		a.usPick(c, t, "全部供应商", a.usProviderFilter, usNamedPick(l.Providers, "全部供应商"), a.usProviderFilter != "", func(v string) {
			a.usProviderFilter = v
			a.usLedOffset = 0
			a.usFresh = ""
		})
		if len(l.Accounts) > 0 {
			var opts []usPickOption
			opts = append(opts, usPickOption{"", "全部账号", ""})
			for _, acc := range l.Accounts {
				opts = append(opts, usPickOption{acc.ID, a.usAccountName(acc.ID), strings.Join(acc.Providers, " · ")})
			}
			a.usPick(c, t, "全部账号", a.usAccountFilter, opts, a.usAccountFilter != "", func(v string) {
				a.usAccountFilter = v
				a.usLedOffset = 0
				a.usFresh = ""
			})
		}
		if len(l.CallerKeys) > 0 {
			var opts []usPickOption
			opts = append(opts, usPickOption{"", "全部网关密钥", ""})
			for _, k := range l.CallerKeys {
				opts = append(opts, usPickOption{k.ID, k.Name, ""})
			}
			a.usPick(c, t, "全部网关密钥", a.usKeyFilter, opts, a.usKeyFilter != "", func(v string) {
				a.usKeyFilter = v
				a.usLedOffset = 0
				a.usFresh = ""
			})
		}
		if len(l.Purposes) > 0 {
			var opts []usPickOption
			opts = append(opts, usPickOption{"", "全部用途", ""})
			for _, p := range l.Purposes {
				opts = append(opts, usPickOption{p, usPurposeName(p), ""})
			}
			a.usPick(c, t, "全部用途", a.usPurposeFilter, opts, a.usPurposeFilter != "", func(v string) {
				a.usPurposeFilter = v
				a.usLedOffset = 0
				a.usFresh = ""
			})
		}

		// a day picked on the heatmap is a filter of its own, cleared here
		if a.usSelectedDay != "" {
			chip := ui.Row(c).AlignItems(ui.Center).Gap(5).Padding(2, 7).Radius(5).
				Background(ui.RGBA(0, 122, 255, 0.10)).Cursor(ui.CursorPointer).
				Focusable().Role(ui.RoleButton).Label("清除日期筛选")
			if chip.Clicked() {
				a.usSelectedDay = ""
				a.usLedOffset = 0
				a.usFresh = ""
				c.Invalidate()
			}
			chip.Children(func() {
				ui.Text(c, a.usSelectedDay).FontSize(10.5).FontWeight(600).TextColor(ui.Hex("#007aff")).SingleLine()
				ui.Text(c, "×").FontSize(10).TextColor(ui.Hex("#007aff")).SingleLine()
			})
		}

		ui.Spacer(c)

		search := ui.SearchField(c, &a.usSearchQuery).Width(210).
			Placeholder("搜索模型、供应商、会话").Label("搜索请求")
		if search.Changed() {
			a.usSearchTyped()
			c.Invalidate()
		}

		export := ui.Row(c).AlignItems(ui.Center).Padding(4, 9).Radius(6).
			Background(a.fieldBg(t)).Border(1, t.Border).Cursor(ui.CursorPointer).
			Focusable().Role(ui.RoleButton).Label("导出 CSV").
			Tooltip("把列表里的每一次请求（所有分页）保存成 Downloads 里的一个 CSV 文件")
		if export.Hovered() {
			export.Background(a.hoverBg(t))
		}
		if export.Clicked() && !a.usExporting {
			a.usExport()
		}
		export.Children(func() {
			if a.usExporting {
				ui.Spinner(c).Size(10, 10)
			}
			ui.Text(c, "导出 CSV").FontSize(11).TextColor(t.TextMuted).SingleLine()
		})
	})
}

// usPickOption is one choice of a filter menu: the value the query takes,
// its name, and a note under it.
type usPickOption struct {
	value string
	name  string
	note  string
}

// usAgentPick is the Agents picker's options.
func usAgentPick(agents []usageapi.Named) []usPickOption {
	opts := []usPickOption{{"", "全部 Agents", ""}}
	for _, a := range agents {
		opts = append(opts, usPickOption{a.ID, a.Name, ""})
	}
	return opts
}

// usNamedPick is a picker over named ids, its first choice every one.
func usNamedPick(named []usageapi.Named, all string) []usPickOption {
	opts := []usPickOption{{"", all, ""}}
	for _, n := range named {
		opts = append(opts, usPickOption{n.ID, n.Name, ""})
	}
	return opts
}

// usPick is a filter as a menu button: its name is the choice now, and
// the menu offers every one and a way back to all.
func (a *App) usPick(c *ui.Context, t *ui.Theme, all, cur string, opts []usPickOption, on bool, pick func(string)) {
	label := all
	for _, o := range opts {
		if o.value != "" && o.value == cur {
			label = o.name
		}
	}
	btn := ui.MenuButton(c, label, func(m *ui.Menu) {
		for _, o := range opts {
			o := o
			item := m.Item(o.name)
			if o.value == cur {
				item.Checked(true)
			}
			if item.Chosen() {
				pick(o.value)
			}
		}
	}).Padding(4, 9).Radius(6).Border(1, t.Border)
	if on {
		btn.Background(ui.RGBA(0, 122, 255, 0.08))
	} else {
		btn.Background(a.fieldBg(t))
	}
}

// usPurposeName is a purpose as the page names it: the catalog's own word
// for the kinds it groups, and the key for one it doesn't know.
func usPurposeName(p string) string {
	switch p {
	case "", "unmarked":
		return "未标注用途"
	}
	if kind, ok := strings.CutPrefix(p, "kind:"); ok {
		for k, v := range usage.PurposeKinds() {
			if k == kind {
				return v.Name
			}
		}
		return kind
	}
	return p
}

// usSearchTyped asks for the page again once typing pauses, as magpie's
// own search waits a quarter second.
func (a *App) usSearchTyped() {
	a.usModelFilter = ""
	if a.usSearchTimer != nil {
		a.usSearchTimer.Stop()
	}
	a.usSearchTimer = time.AfterFunc(280*time.Millisecond, func() {
		if a.win == nil {
			a.usLedOffset = 0
			a.usFresh = ""
			a.usEnsure()
			return
		}
		a.win.Update(func() {
			a.usLedOffset = 0
			a.usFresh = ""
			a.win.Invalidate()
		})
	})
}

// usExport writes every row the filters keep to a CSV in Downloads.
func (a *App) usExport() {
	a.usExporting = true
	f := usageFilterOf(a.usageQuery())
	period := a.usPeriod
	go func() {
		path, rows, err := ExportUsageCSV(period, f)
		apply := func() {
			a.usExporting = false
			switch {
			case err != nil:
				a.usSetNotice("导出失败："+err.Error(), true)
			default:
				a.usSetNotice(fmt.Sprintf("已保存 %d 条请求到 %s", rows, path), false)
			}
		}
		if a.win == nil {
			apply()
			return
		}
		a.win.Update(apply)
	}()
}

// ---- totals ----

func (a *App) usReqKPIs(c *ui.Context, t *ui.Theme, l usageapi.Ledger) {
	total := l.Input + l.Output + l.CacheRead + l.CacheWrite
	hit := cacheHitRate(l.Totals)
	reqSub := "没有失败"
	reqColor := t.TextMuted
	if l.Errors > 0 {
		reqSub = fmt.Sprintf("%d 失败 · 成功率 %.1f%%", l.Errors, 100*(1-float64(l.Errors)/float64(max(1, l.Calls))))
		reqColor = ui.Hex("#c0392b")
	}
	speed := outputSpeed(l.Totals)
	speedText := "—"
	speedSub := "没有计时的流式回复"
	if speed > 0 {
		speedText = fmt.Sprintf("%.0f tok/s", speed)
		speedSub = fmt.Sprintf("首个 Token %s · %d 条计时", FormatDuration(ttftOf(l.Totals)), l.Timed)
	}

	ui.Row(c).FillWidth().Gap(8).Padding(8, 20, 10, 20).Children(func() {
		a.usKPI(c, t, FormatCount(total), "Token", fmt.Sprintf("%s 入 · %s 出 · %s 缓存", FormatCount(l.Input), FormatCount(l.Output), FormatCount(l.CacheRead+l.CacheWrite)), t.TextMuted)
		a.usKPI(c, t, FormatCount(l.Calls), "次请求", reqSub, reqColor)
		a.usKPI(c, t, "≈"+FormatCost(l.Cost), "费用", fmt.Sprintf("%d 次调用无价格", l.Unpriced), t.TextMuted)
		a.usKPI(c, t, fmt.Sprintf("%.1f%%", hit*100), "缓存命中率", "提示词里从缓存读到的比例", t.TextMuted)
		a.usKPI(c, t, speedText, "输出速度", speedSub, t.TextMuted)
	})
}

// ---- the trend and its ranking ----

func (a *App) usReqTrend(c *ui.Context, t *ui.Theme, l usageapi.Ledger) {
	// the two switches, as magpie's chart has them
	if ui.Segmented(c, &a.usLedMetric, usLedLabels(usLedMetrics)...).Changed() {
		c.Invalidate()
	}
	// the split sits at the right of the same row
	ui.Row(c).AlignItems(ui.Center).Gap(8).Padding(0, 20, 2, 20).Children(func() {
		ui.Text(c, "拆分").FontSize(10.5).TextColor(t.TextMuted).SingleLine()
		if ui.Segmented(c, &a.usSplit, usLedLabels(usLedSplits)...).Changed() {
			c.Invalidate()
		}
		ui.Spacer(c)
		ui.Text(c, usBucketWord(l.Bucket)).FontSize(10).TextColor(t.TextMuted).SingleLine()
	})

	split := usLedSplits[clampIdx(a.usSplit, len(usLedSplits))].id
	metric := usLedMetrics[clampIdx(a.usLedMetric, len(usLedMetrics))].id
	dim := split
	shares := l.By[dim]

	// the ranks are the legend and a way in: a click lists only that one
	click := func(id string) {
		switch dim {
		case "provider":
			if a.usProviderFilter == id {
				a.usProviderFilter = ""
			} else {
				a.usProviderFilter = id
			}
		case "agent":
			if a.usAgentFilter == id {
				a.usAgentFilter = ""
			} else {
				a.usAgentFilter = id
			}
		}
		a.usLedOffset = 0
		a.usFresh = ""
	}

	ui.Row(c).AlignItems(ui.Stretch).Gap(10).Padding(0, 20, 8, 20).FillWidth().Children(func() {
		ui.Column(c).Grow(1).Shrink(1).Children(func() {
			a.usTrendChart(c, t, l, dim, metric)
		})
		ui.Column(c).Width(230).Shrink(0).Gap(3).Children(func() {
			ui.Text(c, "按 "+usLedLabel(usLedSplits, dim)).FontSize(10.5).TextColor(t.TextMuted).SingleLine()
			if len(shares) == 0 {
				ui.Text(c, "这段时间没有可拆分的调用。").FontSize(10.5).TextColor(t.TextMuted).SingleLine()
				return
			}
			for i, s := range shares {
				s := s
				if i >= 8 {
					break
				}
				row := ui.Row(c).AlignItems(ui.Center).Gap(6).Padding(2, 4).Radius(5).FillWidth()
				if row.Hovered() {
					row.Background(a.hoverBg(t))
				}
				row.Children(func() {
					ui.Row(c).Width(7).Height(7).Radius(2).Background(usSeriesColors[i%len(usSeriesColors)])
					ui.Text(c, s.Name).FontSize(10.5).TextColor(t.Text).Grow(1).Shrink(1).SingleLine()
					ui.Text(c, usMetricText(metric, s.Totals)).FontSize(10).TextColor(t.TextMuted).SingleLine()
				})
				if row.Clicked() {
					click(s.ID)
					c.Invalidate()
				}
			}
		})
	})
}

// usTrendChart is the trend: a column a bucket, each column stacked by the
// dimension picked, in the ranks' colours.
func (a *App) usTrendChart(c *ui.Context, t *ui.Theme, l usageapi.Ledger, dim, metric string) {
	pts := l.Series
	if len(pts) == 0 {
		ui.Text(c, "这段时间没有可画的请求。").FontSize(10.5).TextColor(t.TextMuted).SingleLine()
		return
	}
	// the ranks' colours, by the order the shares came in
	colorOf := map[string]ui.Color{}
	for i, s := range l.By[dim] {
		colorOf[s.ID] = usSeriesColors[i%len(usSeriesColors)]
	}
	peak := 0.0
	vals := make([][]float64, len(pts))
	ids := make([][]string, len(pts))
	for i, p := range pts {
		by := p.By[dim]
		sum := 0.0
		for id, part := range by {
			v := usMetricValue(metric, part)
			if v <= 0 {
				continue
			}
			vals[i] = append(vals[i], v)
			ids[i] = append(ids[i], id)
			sum += v
		}
		// the column is as tall as its parts together: a bucket's own
		// total is the same figure only when every call of it is told
		// apart, and the tail is summed in one part the page doesn't name
		if sum <= 0 {
			sum = usMetricSum(metric, p)
		}
		if sum > peak {
			peak = sum
		}
	}
	if peak <= 0 {
		peak = 1
	}
	// the ids sorted by the colour they were given, so a stack reads the
	// same from one bucket to the next
	for i := range ids {
		order := make([]int, len(ids[i]))
		for j := range order {
			order[j] = j
		}
		sortByRank(order, func(j int) int { return usRankOf(l.By[dim], ids[i][j]) })
		nv := make([]float64, len(order))
		ni := make([]string, len(order))
		for j, k := range order {
			nv[j], ni[j] = vals[i][k], ids[i][k]
		}
		vals[i], ids[i] = nv, ni
	}

	const chartH = 120.0
	every := 1
	if n := len(pts); n > 8 {
		every = int(math.Ceil(float64(n) / 7))
	}
	ui.Column(c).FillWidth().Gap(4).Children(func() {
		ui.Row(c).AlignItems(ui.End).Gap(1).Height(float32(chartH)).FillWidth().Children(func() {
			for i, p := range pts {
				p := p
				cell := ui.Column(c).Grow(1).Shrink(1).Basis(0).Height(float32(chartH))
				cell.Tooltip(usTrendTip(p, metric))
				if cell.Hovered() {
					cell.Background(ui.RGBA(0, 122, 255, 0.05))
				}
				cell.Children(func() {
					ui.Spacer(c)
					for j, v := range vals[i] {
						h := float32(chartH) * float32(v/peak)
						if h < 0.6 {
							continue
						}
						col := colorOf[ids[i][j]]
						if col == (ui.Color{}) {
							col = usSeriesColors[len(usSeriesColors)-1]
						}
						ui.Row(c).FillWidth().Height(h).Background(col)
					}
				})
			}
		})
		ui.Row(c).FillWidth().Gap(1).Children(func() {
			for i, p := range pts {
				label := ""
				if i%every == 0 || (i == len(pts)-1 && (len(pts)-1)%every >= every/2) {
					label = p.Label
				}
				ui.Row(c).Grow(1).Basis(0).Children(func() {
					if label != "" {
						ui.Text(c, label).FontSize(9).TextColor(t.TextMuted).SingleLine()
					}
				})
			}
		})
	})
}

// usMetricValue is one part's figure for the metric picked.
func usMetricValue(metric string, p usage.Part) float64 {
	switch metric {
	case "cost":
		return p.Cost
	case "calls":
		return float64(p.Calls)
	case "speed":
		if p.DecodeMs <= 0 {
			return 0
		}
		return float64(p.DecodeOut) / (float64(p.DecodeMs) / 1000)
	}
	return float64(p.Tokens)
}

// usMetricSum is a whole point's figure for the metric picked.
func usMetricSum(metric string, p usage.SeriesPoint) float64 {
	switch metric {
	case "cost":
		return p.Cost
	case "calls":
		return float64(p.Calls)
	case "speed":
		return outputSpeed(p.Totals)
	}
	return float64(p.Tokens())
}

// usMetricText is a figure as the rank list writes it.
func usMetricText(metric string, tot usage.Totals) string {
	switch metric {
	case "cost":
		if tot.Cost <= 0 {
			return "—"
		}
		return "≈" + FormatCost(tot.Cost)
	case "calls":
		return FormatCount(tot.Calls)
	case "speed":
		if s := outputSpeed(tot); s > 0 {
			return fmt.Sprintf("%.0f tok/s", s)
		}
		return "—"
	}
	return FormatCount(tot.Input + tot.Output + tot.CacheRead + tot.CacheWrite)
}

// usTrendTip is one bucket of the trend in words.
func usTrendTip(p usage.SeriesPoint, metric string) string {
	when := p.Label
	if !p.Time.IsZero() {
		when = p.Time.Local().Format("1月2日")
	}
	return fmt.Sprintf("%s · %s · %d 次调用", when, usMetricText(metric, p.Totals), p.Calls)
}

// usBucketWord names the bucket the trend sums by.
func usBucketWord(bucket string) string {
	switch bucket {
	case "hour":
		return "按小时"
	case "week":
		return "按周"
	}
	return "按天"
}

// usRankOf is where an id sits in a dimension's shares, for the order of
// a stack.
func usRankOf(shares []usageapi.Share, id string) int {
	for i, s := range shares {
		if s.ID == id {
			return i
		}
	}
	return len(shares)
}

// sortByRank sorts indexes by their rank.
func sortByRank(order []int, rank func(int) int) {
	for i := 1; i < len(order); i++ {
		for j := i; j > 0 && rank(order[j]) < rank(order[j-1]); j-- {
			order[j], order[j-1] = order[j-1], order[j]
		}
	}
}

// ---- the heatmap ----

// usReqHeat is the last 53 weeks a day each, as a calendar of
// contributions lays them out: Monday on top, a cell a day, as dark as the
// day was busy among the others. A click lists that day's requests.
func (a *App) usReqHeat(c *ui.Context, t *ui.Theme, h usage.Heatmap) {
	if h.From == "" || len(h.Days) == 0 {
		return
	}
	days := map[string]usage.Day{}
	for _, d := range h.Days {
		days[d.Date] = d
	}
	from, err := time.Parse(time.DateOnly, h.From)
	if err != nil {
		return
	}
	to, err := time.Parse(time.DateOnly, h.To)
	if err != nil {
		return
	}

	ui.Row(c).AlignItems(ui.Center).Gap(8).Padding(10, 20, 4, 20).Children(func() {
		ui.Text(c, "用量热力图").FontSize(12).FontWeight(600).TextColor(t.Text).SingleLine()
		ui.Text(c, "最近 53 周的每日用量").FontSize(10.5).TextColor(t.TextMuted).SingleLine()
		ui.Spacer(c)
		if ui.Segmented(c, &a.usHeatMetric, usLedLabels(usHeatMetrics)...).Changed() {
			c.Invalidate()
		}
	})

	metric := usHeatMetrics[clampIdx(a.usHeatMetric, len(usHeatMetrics))].id
	peak := 0.0
	for _, d := range h.Days {
		if v := usHeatValue(metric, d); v > peak {
			peak = v
		}
	}
	if peak <= 0 {
		peak = 1
	}

	// the weeks, Monday first: the grid is the days from From to To, a
	// column a week
	weeks := int(math.Ceil(to.Sub(from).Hours()/24/7)) + 1
	ui.ScrollHorizontal(c).FillWidth().Padding(2, 20, 6, 20).Children(func() {
		ui.Row(c).AlignItems(ui.Start).Gap(2).Children(func() {
			for w := 0; w < weeks; w++ {
				ui.Column(c).Gap(2).Children(func() {
					for d := 0; d < 7; d++ {
						day := from.AddDate(0, 0, w*7+d)
						if day.After(to) {
							ui.Row(c).Width(11).Height(11)
							continue
						}
						key := day.Format(time.DateOnly)
						v := usHeatValue(metric, days[key])
						cell := ui.Row(c).Width(11).Height(11).Radius(2).
							Background(usHeatColor(v / peak))
						if key == a.usSelectedDay {
							cell.Border(1, ui.Hex("#007aff"))
						}
						cell.Tooltip(usHeatTip(key, days[key], metric))
						if cell.Clicked() {
							if a.usSelectedDay == key {
								a.usSelectedDay = ""
							} else {
								a.usSelectedDay = key
							}
							a.usLedOffset = 0
							a.usFresh = ""
							c.Invalidate()
						}
					}
				})
			}
		})
	})

	ui.Row(c).AlignItems(ui.Center).Gap(5).Padding(0, 20, 10, 20).Children(func() {
		ui.Text(c, "少").FontSize(9).TextColor(t.TextMuted).SingleLine()
		for i := 0; i < 5; i++ {
			ui.Row(c).Width(9).Height(9).Radius(2).Background(usHeatColor(float64(i) / 4))
		}
		ui.Text(c, "多").FontSize(9).TextColor(t.TextMuted).SingleLine()
		if a.usSelectedDay != "" {
			ui.Text(c, "· 已选 "+a.usSelectedDay+"，再点一次取消").FontSize(9.5).TextColor(ui.Hex("#007aff")).SingleLine()
		}
	})
}

// usHeatValue is a day's figure for the heatmap's metric.
func usHeatValue(metric string, d usage.Day) float64 {
	switch metric {
	case "calls":
		return float64(d.Calls)
	case "cost":
		return d.Cost
	}
	return float64(d.Tokens)
}

// usHeatColor is a cell's colour by how busy the day was among the others:
// the theme's accent at five levels, the quietest still visible.
func usHeatColor(share float64) ui.Color {
	if share <= 0 {
		return ui.RGBA(0, 0, 0, 0.05)
	}
	levels := []float64{0.08, 0.22, 0.42, 0.66, 1}
	pick := levels[0]
	for _, l := range levels {
		if share >= l {
			pick = l
		}
	}
	return ui.RGBA(0, 122, 255, float32(0.15+0.75*pick))
}

// usHeatTip is one day of the heatmap in words.
func usHeatTip(date string, d usage.Day, metric string) string {
	if d.Calls == 0 {
		return date + " · 没有调用"
	}
	return fmt.Sprintf("%s · %d 次调用 · %s tokens · ≈%s", date, d.Calls, FormatCount(d.Tokens), FormatCost(d.Cost))
}

// ---- the sources ----

// usReqVias is the strip of sources: every request, those the gateway
// carried, those read from the agents' own files, and those an app
// reported. Each is a filter.
func (a *App) usReqVias(c *ui.Context, t *ui.Theme, l usageapi.Ledger) {
	all := l.Through.Calls + l.Direct.Calls + l.Reported.Calls
	if all == 0 {
		return
	}
	cell := func(label, value string, calls int, pick string) {
		// the cell of no source picked is on while the filter is clear,
		// which is what "every request" means
		on := a.usViaFilter == pick
		b := ui.Column(c).Grow(1).Basis(0).Padding(7, 10).Radius(8).
			Background(a.fieldBg(t)).Border(1, t.Border).Gap(1).
			Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).Label(label)
		if on {
			b.Border(1, t.Accent).Background(ui.RGBA(0, 122, 255, 0.07))
		}
		if b.Clicked() {
			if a.usViaFilter == pick {
				a.usViaFilter = ""
			} else {
				a.usViaFilter = pick
			}
			a.usLedOffset = 0
			a.usFresh = ""
			c.Invalidate()
		}
		b.Children(func() {
			ui.Text(c, label).FontSize(10).TextColor(t.TextMuted).SingleLine()
			ui.Text(c, value).FontSize(14).FontWeight(650).TextColor(t.Text).SingleLine()
			ui.Text(c, fmt.Sprintf("%d 次请求", calls)).FontSize(9.5).TextColor(t.TextMuted).SingleLine()
		})
	}

	ui.Row(c).FillWidth().Gap(8).Padding(0, 20, 8, 20).Children(func() {
		cell("全部请求", FormatCount(all), all, "")
		cell("经过网关", FormatCount(l.Through.Calls), l.Through.Calls, usage.SourceGateway)
		cell("不经过网关", FormatCount(l.Direct.Calls), l.Direct.Calls, usage.SourceSession)
		if l.Reported.Calls > 0 || a.usViaFilter == usage.SourceReported {
			cell("应用上报", FormatCount(l.Reported.Calls), l.Reported.Calls, usage.SourceReported)
		}
	})
}

// ---- the calls ----

// usReqTable is the calls themselves: the newest first, a page at a time,
// each row a click away from what was said in it.
func (a *App) usReqTable(c *ui.Context, t *ui.Theme, l usageapi.Ledger) {
	// the columns, each a fixed width so a page reads down its rows
	cols := []usCol{
		{"时间", 96}, {"Agent", 150}, {"请求模型", 170}, {"实际模型", 150},
		{"输入", 66}, {"输出", 66}, {"缓存写", 62}, {"缓存读", 66},
		{"费用", 74}, {"耗时", 62}, {"状态", 130},
	}
	width := float32(0)
	for _, col := range cols {
		width += col.width
	}
	widths := usColWidths(cols)

	head := ui.Row(c).AlignItems(ui.Center).Gap(0).Padding(4, 20).FillWidth()
	head.Children(func() {
		for _, col := range cols {
			ui.Text(c, col.title).FontSize(10).FontWeight(600).TextColor(t.TextMuted).
				Width(col.width).Shrink(0).SingleLine()
		}
	})

	ui.ScrollHorizontal(c).FillWidth().Children(func() {
		ui.Column(c).Width(width + 40).Gap(0).Children(func() {
			for _, r := range l.Rows {
				r := r
				a.usReqRow(c, t, r, widths)
			}
		})
	})

	// a page at a time, the newest first
	pageSize := a.usageQuery().Limit
	if l.Total > pageSize || l.Offset > 0 {
		ui.Row(c).AlignItems(ui.Center).Gap(10).Padding(10, 20).FillWidth().Children(func() {
			ui.Spacer(c)
			prev := ui.Button(c, "较新")
			if l.Offset <= 0 {
				prev.Opacity(0.4)
			} else if prev.Clicked() {
				a.usLedOffset = max(0, l.Offset-pageSize)
				a.usFresh = ""
				c.Invalidate()
			}
			ui.Text(c, usPageNumbers(l.Offset, len(l.Rows), l.Total)).FontSize(10.5).TextColor(t.TextMuted).SingleLine()
			next := ui.Button(c, "较早")
			if l.Offset+len(l.Rows) >= l.Total {
				next.Opacity(0.4)
			} else if next.Clicked() {
				a.usLedOffset = l.Offset + pageSize
				a.usFresh = ""
				c.Invalidate()
			}
		})
	}
}

// usColWidths turns the table's columns into their widths, in order.
func usColWidths(cols []usCol) []float32 {
	out := make([]float32, len(cols))
	for i, col := range cols {
		out[i] = col.width
	}
	return out
}

// usReqRow is one call: when, who, what it asked for and what answered,
// its tokens, its cost, how long it took and how it ended.
func (a *App) usReqRow(c *ui.Context, t *ui.Theme, r usageapi.Row, widths []float32) {
	failed := r.Failed() || r.Status >= 400
	row := ui.Row(c).AlignItems(ui.Center).Gap(0).Padding(4, 20).FillWidth().
		Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).
		Label(usReqRowLabel(r))
	if row.Hovered() {
		row.Background(a.hoverBg(t))
	}
	if failed {
		row.Background(ui.RGBA(255, 59, 48, 0.045))
	}
	if row.Clicked() {
		a.usInspectRequest(r)
		c.Invalidate()
	}
	row.Children(func() {
		// the time, with the day when it isn't today
		when := r.Time.Local()
		stamp := when.Format("15:04:05")
		if when.YearDay() != time.Now().YearDay() || when.Year() != time.Now().Year() {
			stamp = when.Format("1月2日 15:04")
		}
		ui.Text(c, stamp).FontSize(10.5).TextColor(t.TextMuted).Width(widths[0]).Shrink(0).SingleLine().
			Tooltip(when.Format("2006-01-02 15:04:05"))

		// who: the agent's name, and where the call came from
		ui.Row(c).AlignItems(ui.Center).Gap(5).Width(widths[1]).Shrink(0).Children(func() {
			ui.Image(c, iconPictureFor(r.Icon)).Size(15, 15).Radius(3)
			ui.Text(c, usAgentName(r.Agent)).FontSize(10.5).TextColor(t.Text).Shrink(1).SingleLine()
			a.usSourceBadge(c, t, r)
		})

		asked := r.Requested
		if asked == "" {
			asked = r.Model
		}
		ui.Text(c, usModelLabel(asked)).FontSize(10.5).TextColor(t.TextMuted).
			Width(widths[2]).Shrink(0).SingleLine().Tooltip(asked)

		served := r.Served
		if served == "" {
			served = r.Model
		}
		servedC := t.Text
		if r.Swapped {
			servedC = ui.Hex("#b45309")
		}
		ui.Text(c, usModelLabel(served)).FontSize(10.5).TextColor(servedC).
			Width(widths[3]).Shrink(0).SingleLine().Tooltip(served)

		ui.Text(c, FormatCount(r.Input)).FontSize(10.5).TextColor(t.TextMuted).Width(widths[4]).Shrink(0).SingleLine()
		ui.Text(c, FormatCount(r.Output)).FontSize(10.5).TextColor(t.TextMuted).Width(widths[5]).Shrink(0).SingleLine().
			Tooltip(usReasoningNote(r))
		ui.Text(c, FormatCount(r.CacheWrite)).FontSize(10.5).TextColor(t.TextMuted).Width(widths[6]).Shrink(0).SingleLine()
		ui.Text(c, FormatCount(r.CacheRead)).FontSize(10.5).TextColor(t.TextMuted).Width(widths[7]).Shrink(0).SingleLine()

		cost := "—"
		costC := t.TextMuted
		if r.Priced {
			cost = "≈" + FormatCost(r.Cost)
			costC = t.Text
		}
		ui.Text(c, cost).FontSize(10.5).TextColor(costC).Width(widths[8]).Shrink(0).SingleLine()

		took := "—"
		if r.Millis > 0 {
			took = FormatDuration(r.Millis)
			if r.Source == "log" {
				took = "≈" + took
			}
		}
		ui.Text(c, took).FontSize(10.5).TextColor(t.TextMuted).Width(widths[9]).Shrink(0).SingleLine().
			Tooltip(usTimingNote(r))

		// the status: the gateway's own, or the session file's word
		ui.Row(c).AlignItems(ui.Center).Gap(4).Width(widths[10]).Shrink(0).Children(func() {
			ui.Row(c).Width(6).Height(6).Radius(3).Background(usStatusTone(t, failed))
			word := fmt.Sprint(r.Status)
			if r.Status == 0 {
				if r.Source == "log" {
					if failed {
						word = "失败"
					} else {
						word = "成功"
					}
				} else {
					word = "—"
				}
			}
			if failed && r.ErrType != "" {
				word += " · " + r.ErrType
			}
			if r.Tries > 1 && !failed {
				word += fmt.Sprintf(" · %d 次尝试", r.Tries)
			}
			ui.Text(c, word).FontSize(10).TextColor(usStatusTone(t, failed)).Shrink(1).SingleLine().
				Tooltip(r.Error)
		})
	})
}

// usReqRowLabel names a row for assistive technology.
func usReqRowLabel(r usageapi.Row) string {
	model := r.Served
	if model == "" {
		model = r.Model
	}
	return fmt.Sprintf("%s %s %s", r.Time.Format("15:04"), usAgentName(r.Agent), model)
}

// usReasoningNote is a row's reasoning tokens, inside its output.
func usReasoningNote(r usageapi.Row) string {
	if r.Reasoning <= 0 {
		return ""
	}
	return fmt.Sprintf("其中 %s 推理，含在输出里", FormatCount(r.Reasoning))
}

// usTimingNote is a row's timing: the whole, the first token, and how fast
// it wrote after.
func usTimingNote(r usageapi.Row) string {
	var parts []string
	if r.Millis > 0 {
		parts = append(parts, fmt.Sprintf("耗时 %d ms", r.Millis))
	}
	if r.TTFT > 0 {
		parts = append(parts, "首个 Token "+FormatDuration(r.TTFT))
	}
	if s := usRowSpeed(r); s > 0 {
		parts = append(parts, fmt.Sprintf("%.0f tok/s", s))
	}
	if r.RequestID != "" {
		parts = append(parts, "请求 "+r.RequestID)
	}
	if r.Endpoint != "" {
		parts = append(parts, "端点 "+r.Endpoint)
	}
	if r.Session != "" {
		parts = append(parts, "会话 "+r.Session)
	}
	return strings.Join(parts, " · ")
}

// usRowSpeed is a reply's speed: its answer tokens after the first, a
// second, when the record timed them.
func usRowSpeed(r usageapi.Row) float64 {
	n, ms := r.Decode()
	if ms <= 0 || n <= 0 {
		return 0
	}
	return float64(n) / (float64(ms) / 1000)
}

// usLedLabels is a set of choices as Segmented wants them.
func usLedLabels(items []usChoice) []string {
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = it.label
	}
	return out
}

// usLedLabel is one choice's label, by id.
func usLedLabel(items []usChoice, id string) string {
	for _, it := range items {
		if it.id == id {
			return it.label
		}
	}
	return id
}

// clampIdx keeps an index inside a list.
func clampIdx(i, n int) int {
	if i < 0 {
		return 0
	}
	if i >= n {
		return max(0, n-1)
	}
	return i
}
