//go:build nogui && echo

package shell

// The Usage page's Sessions segment: the agents' own conversations, read
// from the files they leave on this computer — what each cost, when they
// were at work, which folders and models they used, what they called their
// tools and skills for, and the sessions themselves, a page at a time.
//
// It is magpie's own segment: the totals, the activity chart with its six
// measures, the projects and models, the hour of the day a session was at
// work, the shape of a session, the tools and the skills, and the list.
// The reading is sessions.StatsFor and its Overview; this file is the
// drawing.

import (
	"fmt"
	"math"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/yetone/magpie/internal/sessions"
)

// usSessMetrics are the activity chart's measures, as magpie offers them.
var usSessMetrics = []usChoice{
	{"tokens", "Token"},
	{"output", "输出"},
	{"messages", "消息"},
	{"sessions", "会话"},
	{"cost", "费用"},
	{"active", "活跃"},
}

// usSessTops are the ways the top sessions are ranked.
var usSessTops = []usChoice{
	{"tokens", "Token"},
	{"cost", "费用"},
	{"active", "活跃"},
}

// usSessShapes are the ways a session's shape is told.
var usSessShapes = []usChoice{
	{"messages", "消息"},
	{"minutes", "时长"},
	{"autonomy", "工具调用"},
}

// usToolCats are the kinds of tool, coloured as magpie colours them.
var usToolCats = map[string]ui.Color{
	"Bash":  ui.Hex("#e0823d"),
	"Edit":  ui.Hex("#4f8cf0"),
	"Read":  ui.Hex("#2fb087"),
	"Write": ui.Hex("#9b6cf0"),
	"Grep":  ui.Hex("#e0608f"),
	"Glob":  ui.Hex("#d4a72c"),
	"Task":  ui.Hex("#35a9c7"),
	"Tool":  ui.Hex("#7c83f2"),
	"Other": ui.Hex("#9aa0a8"),
}

// usToolColor is a kind's colour, the grey for the rest.
func usToolColor(cat string) ui.Color {
	if c, ok := usToolCats[cat]; ok {
		return c
	}
	return usToolCats["Other"]
}

func (a *App) usageSessions(c *ui.Context, t *ui.Theme) {
	d := a.data.Usage
	if d.Query.Tab != UsTabSessions {
		a.usSkeleton(c, t)
		return
	}
	st, ov := d.SessionsStats, d.SessionsOverview
	rows := d.SessionsList

	a.usSessFilters(c, t, st, rows)

	if len(rows) == 0 && ov.Count == 0 {
		if a.usSessFiltered() {
			a.usEmpty(c, t, "没有符合这些筛选条件的会话。", "清掉筛选条件，或换一段时间。")
		} else {
			a.usEmpty(c, t, "这段时间还没有会话。",
				"Claude Code、Codex、OpenCode、Pi 等 Agent 在电脑上留下的会话会列在这里，连同每个会话的花费。")
		}
		return
	}

	a.usSessKPIs(c, t, st, ov)
	a.usSessChart(c, t, st, ov)
	a.usSessGrid(c, t, st, ov, rows)
	a.usSessTopCard(c, t, ov)
	a.usSessShapeCard(c, t, ov)
	a.usSessTools(c, t, ov)
	a.usSessSkills(c, t, ov)
	a.usSessList(c, t, rows)
	a.usNote(c, t, usSessNote(len(rows), a.usSessListed))
}

// usSessFiltered says whether any filter but the range is on.
func (a *App) usSessFiltered() bool {
	return (a.usSessAgent != "" && a.usSessAgent != "all") || a.usSessModel != "" ||
		a.usSessFolder != "" || strings.TrimSpace(a.usSessQuery) != ""
}

// usSessFilters is the row of filters: the agents as segments, the model
// and folder pickers, and the search.
func (a *App) usSessFilters(c *ui.Context, t *ui.Theme, st sessions.Stats, rows []sessions.Summary) {
	agents := a.data.Usage.SessionsAgents
	if a.usSessAgent != "" && a.usSessAgent != "all" {
		found := false
		for _, id := range agents {
			if id == a.usSessAgent {
				found = true
			}
		}
		if !found {
			a.usSessAgent = "all"
		}
	}
	idx := 0
	for i, id := range agents {
		if id == a.usSessAgent {
			idx = i + 1
		}
	}
	a.usSessAgentIdx = idx

	ui.Row(c).AlignItems(ui.Center).Gap(8).Padding(10, 20, 6, 20).FillWidth().Children(func() {
		// the agents, when there is more than one to tell apart
		if len(agents) > 1 {
			labels := []string{"全部"}
			for _, id := range agents {
				labels = append(labels, usAgentName(id))
			}
			if ui.Segmented(c, &a.usSessAgentIdx, labels...).Changed() {
				if a.usSessAgentIdx <= 0 || a.usSessAgentIdx > len(agents) {
					a.usSessAgent = "all"
				} else {
					a.usSessAgent = agents[a.usSessAgentIdx-1]
				}
				a.usSessListed = 200
				a.usFresh = ""
				c.Invalidate()
			}
		}

		// the models and folders the range has, as the pickers offer them
		models := usSessSums(rows, "model")
		var modelOpts []usPickOption
		modelOpts = append(modelOpts, usPickOption{"", "全部模型", ""})
		for _, m := range models {
			modelOpts = append(modelOpts, usPickOption{m.id, m.id, FormatCount(m.tokens) + " tokens"})
		}
		a.usPick(c, t, "全部模型", a.usSessModel, modelOpts, a.usSessModel != "", func(v string) {
			a.usSessModel = v
			a.usSessListed = 200
			a.usFresh = ""
		})

		folders := usSessSums(rows, "cwd")
		var folderOpts []usPickOption
		folderOpts = append(folderOpts, usPickOption{"", "全部项目", ""})
		for _, f := range folders {
			folderOpts = append(folderOpts, usPickOption{f.id, filepath.Base(f.id), f.id})
		}
		a.usPick(c, t, "全部项目", a.usSessFolder, folderOpts, a.usSessFolder != "", func(v string) {
			a.usSessFolder = v
			a.usSessListed = 200
			a.usFresh = ""
		})

		ui.Spacer(c)

		search := ui.SearchField(c, &a.usSessQuery).Width(210).
			Placeholder("搜索会话、项目、模型").Label("搜索会话")
		if search.Changed() {
			if a.usSearchTimer != nil {
				a.usSearchTimer.Stop()
			}
			a.usSearchTimer = time.AfterFunc(280*time.Millisecond, func() {
				apply := func() {
					a.usSessListed = 200
					a.usFresh = ""
					if a.win != nil {
						a.win.Invalidate()
					}
				}
				if a.win == nil {
					apply()
					return
				}
				a.win.Update(apply)
			})
			c.Invalidate()
		}
	})
}

// usSessSum is one folder's or model's tokens, the most first.
type usSessSum struct {
	id     string
	tokens int
	cost   float64
}

// usSessSums adds up the sessions by one of their fields, the most tokens
// first.
func usSessSums(rows []sessions.Summary, key string) []usSessSum {
	at := map[string]*usSessSum{}
	var order []string
	for _, s := range rows {
		var id string
		switch key {
		case "cwd":
			id = s.Cwd
		case "model":
			if len(s.Models) > 0 {
				id = s.Models[0]
			}
		}
		if id == "" {
			continue
		}
		x, ok := at[id]
		if !ok {
			x = &usSessSum{id: id}
			at[id] = x
			order = append(order, id)
		}
		x.tokens += s.Input + s.Output
		x.cost += s.Cost
	}
	out := make([]usSessSum, 0, len(order))
	for _, id := range order {
		out = append(out, *at[id])
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].tokens > out[j].tokens })
	return out
}

// ---- the totals ----

func (a *App) usSessKPIs(c *ui.Context, t *ui.Theme, st sessions.Stats, ov sessions.Overview) {
	q := a.usageQuery()
	pts := usSessPoints(st, ov, q)
	var tokens, out, cacheRead, cacheWrite, cost float64
	days := map[string]bool{}
	unpriced := map[string]bool{}
	for _, p := range pts {
		tokens += float64(p.Tokens)
		out += float64(p.Output)
		cacheRead += float64(p.CacheRead)
		cacheWrite += float64(p.CacheWrite)
		cost += p.Cost
		if p.Tokens > 0 || p.Cost > 0 {
			days[p.Date] = true
		}
		for m := range p.Unpriced {
			unpriced[m] = true
		}
	}
	var active int64
	activeKnown := q.SessModel == ""
	for _, p := range pts {
		if activeKnown {
			active += p.Active
		}
	}

	countSub := ""
	if ov.Count > 0 {
		countSub = fmt.Sprintf("中位 %s · p90 %s", FormatCount(ov.Median), FormatCount(ov.P90))
	}
	costText := "—"
	costSub := "按有效模型价格估算"
	if cost > 0 {
		costText = "≈" + FormatCost(cost)
	}
	if len(unpriced) > 0 {
		costSub = fmt.Sprintf("%d 个模型无价格", len(unpriced))
	}
	hitSub := ""
	if cacheRead > 0 && tokens > 0 {
		hitSub = fmt.Sprintf("写入 %s", FormatCount(int(cacheWrite)))
	}
	activeText := "—"
	activeSub := "按模型不保留"
	if activeKnown {
		activeText = FormatSeconds(active)
		activeSub = fmt.Sprintf("在 %d 天里", len(days))
	}

	// the folders, and how much of it the first took
	folders := usSessSums(a.data.Usage.SessionsList, "cwd")
	leadSub := ""
	if len(folders) > 0 {
		leadSub = filepath.Base(folders[0].id)
		if len(folders) > 1 && tokens > 0 {
			leadSub = fmt.Sprintf("%.0f%% 在 %s", 100*float64(folders[0].tokens)/tokens, filepath.Base(folders[0].id))
		}
	}

	ui.Row(c).FillWidth().Gap(8).Padding(8, 20, 10, 20).Children(func() {
		a.usKPI(c, t, FormatCount(ov.Count), "个会话", countSub, t.TextMuted)
		a.usKPI(c, t, FormatCount(int(tokens)), "Token", fmt.Sprintf("%s 入 · %s 出", FormatCount(int(tokens-out)), FormatCount(int(out))), t.TextMuted)
		a.usKPI(c, t, costText, "费用", costSub, t.TextMuted)
		a.usKPI(c, t, FormatCount(int(cacheRead)), "缓存读取", hitSub, t.TextMuted)
		a.usKPI(c, t, activeText, "活跃时长", activeSub, t.TextMuted)
		a.usKPI(c, t, fmt.Sprint(len(folders)), "个项目", leadSub, t.TextMuted)
	})
}

// ---- the activity chart ----

// usSessPoint is one day of the range, as the chart draws it.
type usSessPoint struct {
	Date       string
	Time       time.Time
	Tokens     int
	Input      int
	Output     int
	CacheRead  int
	CacheWrite int
	Cost       float64
	Active     int64
	Messages   int
	Sessions   int
	Unpriced   map[string]bool
}

// usSessPoints is the range a day at a time, under the agent, model and
// folder the filters name.
func usSessPoints(st sessions.Stats, ov sessions.Overview, q UsageQuery) []usSessPoint {
	if st.From == "" || st.To == "" {
		return nil
	}
	from, err := time.ParseInLocation(time.DateOnly, st.From, time.Local)
	if err != nil {
		return nil
	}
	to, err := time.ParseInLocation(time.DateOnly, st.To, time.Local)
	if err != nil {
		return nil
	}
	n := int(math.Round(to.Sub(from).Hours()/24)) + 1
	if n <= 0 {
		return nil
	}
	byDate := map[string]sessions.Day{}
	for _, d := range st.Days {
		byDate[d.Date] = d
	}

	agent := q.SessAgent
	if agent == "all" {
		agent = ""
	}
	out := make([]usSessPoint, n)
	for i := 0; i < n; i++ {
		day := from.AddDate(0, 0, i)
		key := day.Format(time.DateOnly)
		p := usSessPoint{Date: key, Time: day, Unpriced: map[string]bool{}}
		if i < len(ov.Days) {
			p.Sessions = ov.Days[i]
		}
		if i < len(ov.Messages) {
			p.Messages = ov.Messages[i]
		}
		d := byDate[key]
		for _, u := range d.Usage {
			if agent != "" && u.Agent != agent {
				continue
			}
			if q.SessModel != "" && u.Model != q.SessModel {
				continue
			}
			if q.SessFolder != "" && u.Cwd != q.SessFolder {
				continue
			}
			p.Tokens += u.Input + u.Output
			p.Input += u.Input
			p.Output += u.Output
			p.CacheRead += u.CacheRead
			p.CacheWrite += u.CacheWrite
			p.Cost += u.Cost
			if !u.Priced && u.Input+u.Output > 0 {
				p.Unpriced[u.Model] = true
			}
		}
		if q.SessModel == "" {
			for _, act := range d.Active {
				if agent != "" && act.Agent != agent {
					continue
				}
				if q.SessFolder != "" && act.Cwd != q.SessFolder {
					continue
				}
				p.Active += act.Seconds
			}
		}
		out[i] = p
	}
	return out
}

// usSessValue is one day's figure for the measure picked.
func usSessValue(metric string, p usSessPoint) float64 {
	switch metric {
	case "output":
		return float64(p.Output)
	case "messages":
		return float64(p.Messages)
	case "sessions":
		return float64(p.Sessions)
	case "cost":
		return p.Cost
	case "active":
		return float64(p.Active)
	}
	return float64(p.Tokens)
}

// usSessText writes a figure of the measure picked.
func usSessText(metric string, v float64) string {
	switch metric {
	case "cost":
		if v <= 0 {
			return "—"
		}
		return "≈" + FormatCost(v)
	case "active":
		return FormatSeconds(int64(v))
	}
	return FormatCount(int(v))
}

// usSessChart is the range's activity: a column a day, as the Overview's
// timeline is drawn, with the measure a click away.
func (a *App) usSessChart(c *ui.Context, t *ui.Theme, st sessions.Stats, ov sessions.Overview) {
	pts := usSessPoints(st, ov, a.usageQuery())
	if len(pts) < 2 {
		return
	}
	metric := usSessMetrics[clampIdx(a.usSessMetric, len(usSessMetrics))].id

	ui.Row(c).AlignItems(ui.Center).Gap(8).Padding(6, 20, 0, 20).Children(func() {
		ui.Text(c, "这段时间的活跃度").FontSize(12.5).FontWeight(600).TextColor(t.Text).SingleLine()
		ui.Spacer(c)
		if ui.Segmented(c, &a.usSessMetric, usLedLabels(usSessMetrics)...).Changed() {
			c.Invalidate()
		}
	})

	peak := 0.0
	for _, p := range pts {
		if v := usSessValue(metric, p); v > peak {
			peak = v
		}
	}
	if peak <= 0 {
		peak = 1
	}

	const chartH = 110.0
	every := 1
	if n := len(pts); n > 8 {
		every = int(math.Ceil(float64(n) / 7))
	}
	ui.Column(c).FillWidth().Gap(4).Padding(0, 20, 8, 20).Children(func() {
		ui.Row(c).AlignItems(ui.End).Gap(2).Height(float32(chartH)).FillWidth().Children(func() {
			for _, p := range pts {
				p := p
				cell := ui.Column(c).Grow(1).Shrink(1).Basis(0).Height(float32(chartH))
				cell.Tooltip(usSessTip(p, metric))
				if cell.Hovered() {
					cell.Background(ui.RGBA(0, 122, 255, 0.05))
				}
				cell.Children(func() {
					ui.Spacer(c)
					if metric == "tokens" {
						// output on top of input, as the Overview draws it
						if h := float32(chartH) * float32(p.Output) / float32(peak); h > 0.5 {
							ui.Row(c).FillWidth().Height(h).Background(ui.Hex("#34c759"))
						}
						if h := float32(chartH) * float32(p.Input) / float32(peak); h > 0.5 {
							ui.Row(c).FillWidth().Height(h).Background(ui.Hex("#007aff"))
						}
						return
					}
					if h := float32(chartH) * float32(usSessValue(metric, p)/peak); h > 0.5 {
						ui.Row(c).FillWidth().Height(h).Background(usSessBarColor(metric))
					}
				})
			}
		})
		ui.Row(c).FillWidth().Gap(2).Children(func() {
			for i, p := range pts {
				label := ""
				if i%every == 0 || i == len(pts)-1 && (len(pts)-1)%every >= every/2 {
					label = p.Time.Format("1月2日")
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

// usSessBarColor is the chart's fill for a measure that isn't tokens.
func usSessBarColor(metric string) ui.Color {
	switch metric {
	case "output":
		return ui.Hex("#34c759")
	case "messages":
		return ui.Hex("#af52de")
	case "sessions":
		return ui.Hex("#5ac8fa")
	case "cost":
		return ui.Hex("#ff9500")
	case "active":
		return ui.Hex("#ff3b30")
	}
	return ui.Hex("#007aff")
}

// usSessTip is one day of the chart in words.
func usSessTip(p usSessPoint, metric string) string {
	when := p.Time.Format("1月2日")
	var parts []string
	if p.Tokens > 0 {
		parts = append(parts, FormatCount(p.Tokens)+" tokens")
	}
	if p.Cost > 0 {
		parts = append(parts, "≈"+FormatCost(p.Cost))
	}
	if p.Messages > 0 {
		parts = append(parts, fmt.Sprintf("%d 条消息", p.Messages))
	}
	if p.Sessions > 0 {
		parts = append(parts, fmt.Sprintf("%d 个会话", p.Sessions))
	}
	if p.Active > 0 {
		parts = append(parts, FormatSeconds(p.Active)+" 活跃")
	}
	if len(parts) == 0 {
		return when + " · 没有活动"
	}
	return when + " · " + strings.Join(parts, " · ")
}

// ---- projects, models and the hours ----

// usSessGrid is the two columns under the chart: the projects and the
// models, each a bar list a click turns into a filter.
func (a *App) usSessGrid(c *ui.Context, t *ui.Theme, st sessions.Stats, ov sessions.Overview, rows []sessions.Summary) {
	ui.Row(c).AlignItems(ui.Stretch).Gap(10).Padding(2, 20, 8, 20).FillWidth().Children(func() {
		ui.Column(c).Grow(1).Shrink(1).Gap(4).Children(func() {
			a.usSessBars(c, t, "项目", usSessSums(rows, "cwd"), a.usSessFolder, func(v string) {
				if a.usSessFolder == v {
					v = ""
				}
				a.usSessFolder = v
				a.usSessListed = 200
				a.usFresh = ""
			}, func(v string) string { return filepath.Base(v) })
		})
		ui.Column(c).Grow(1).Shrink(1).Gap(4).Children(func() {
			a.usSessBars(c, t, "模型", usSessSums(rows, "model"), a.usSessModel, func(v string) {
				if a.usSessModel == v {
					v = ""
				}
				a.usSessModel = v
				a.usSessListed = 200
				a.usFresh = ""
			}, func(v string) string { return v })
		})
	})

	// the hour of the day a session was at work, in this computer's zone
	a.usSessHours(c, t, st)
}

// usSessBars is a bar list of folders or models: each its tokens, its
// share, and a click to filter by it.
func (a *App) usSessBars(c *ui.Context, t *ui.Theme, title string, sums []usSessSum, cur string, pick func(string), name func(string) string) {
	a.usSectionHead(c, t, "sess-"+title, title, fmt.Sprintf("%d 个", len(sums)), false)
	if len(sums) == 0 {
		ui.Text(c, "这段时间没有可拆分的会话。").FontSize(10.5).TextColor(t.TextMuted).Padding(2, 20).SingleLine()
		return
	}
	total := 1
	for _, s := range sums {
		if s.tokens > total {
			total = s.tokens
		}
	}
	shown := min(len(sums), 8)
	for i := 0; i < shown; i++ {
		s := sums[i]
		on := s.id == cur
		row := ui.Row(c).AlignItems(ui.Center).Gap(8).Padding(3, 20).FillWidth().
			Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).Label(name(s.id))
		if row.Hovered() || on {
			row.Background(a.hoverBg(t))
		}
		if row.Clicked() {
			pick(s.id)
			c.Invalidate()
		}
		row.Children(func() {
			ui.Text(c, name(s.id)).FontSize(11).TextColor(t.Text).Width(120).Shrink(0).SingleLine().Tooltip(s.id)
			a.usShareBar(c, t, float64(s.tokens)/float64(max(1, total)), ui.Hex("#007aff"), 90)
			ui.Text(c, FormatCount(s.tokens)).FontSize(10).TextColor(t.TextMuted).Width(64).Shrink(0).SingleLine()
			cost := "—"
			if s.cost > 0 {
				cost = "≈" + FormatCost(s.cost)
			}
			ui.Text(c, cost).FontSize(10).TextColor(t.TextMuted).Shrink(1).SingleLine()
		})
	}
}

// usSessHours lays the range's active time out by day of the week and hour
// of the day.
func (a *App) usSessHours(c *ui.Context, t *ui.Theme, st sessions.Stats) {
	q := a.usageQuery()
	if q.SessModel != "" {
		a.usSectionHead(c, t, "sess-hours", "按小时", "按模型不保留", false)
		ui.Text(c, "活跃时长不按模型区分。").FontSize(10.5).TextColor(t.TextMuted).Padding(2, 20).SingleLine()
		return
	}
	agent := q.SessAgent
	if agent == "all" {
		agent = ""
	}
	var grid [7][24]int64
	var total int64
	for _, d := range st.Days {
		day, err := time.ParseInLocation(time.DateOnly, d.Date, time.Local)
		if err != nil {
			continue
		}
		wd := (int(day.Weekday()) + 6) % 7 // Monday first
		for _, act := range d.Active {
			if agent != "" && act.Agent != agent {
				continue
			}
			if q.SessFolder != "" && act.Cwd != q.SessFolder {
				continue
			}
			for h, sec := range act.Hours {
				if h < 0 || h >= 24 {
					continue
				}
				grid[wd][h] += sec
				total += sec
			}
		}
	}
	a.usSectionHead(c, t, "sess-hours", "按小时", "", false)
	if total == 0 {
		ui.Text(c, "这段时间没有记录到活跃时长。").FontSize(10.5).TextColor(t.TextMuted).Padding(2, 20).SingleLine()
		return
	}

	peak := int64(0)
	bestWD, bestH := 0, 0
	for wd := 0; wd < 7; wd++ {
		for h := 0; h < 24; h++ {
			if grid[wd][h] > peak {
				peak = grid[wd][h]
				bestWD, bestH = wd, h
			}
		}
	}
	weekdays := []string{"周一", "周二", "周三", "周四", "周五", "周六", "周日"}

	ui.Column(c).FillWidth().Gap(2).Padding(0, 20, 4, 20).Children(func() {
		for wd := 0; wd < 7; wd++ {
			wd := wd
			ui.Row(c).AlignItems(ui.Center).Gap(2).Children(func() {
				ui.Text(c, weekdays[wd]).FontSize(9.5).TextColor(t.TextMuted).Width(34).Shrink(0).SingleLine()
				for h := 0; h < 24; h++ {
					h := h
					v := grid[wd][h]
					cell := ui.Row(c).Width(11).Height(11).Radius(2).
						Background(usHeatColor(float64(v) / float64(max(1, peak))))
					cell.Tooltip(fmt.Sprintf("%s %02d:00 · %s", weekdays[wd], h, usSessDur(v)))
					if cell.Hovered() {
						cell.Border(1, t.Border)
					}
				}
			})
		}
		ui.Row(c).AlignItems(ui.Center).Gap(2).Children(func() {
			ui.Text(c, "").Width(34).Shrink(0)
			for h := 0; h < 24; h++ {
				label := ""
				if h%6 == 0 {
					label = fmt.Sprint(h)
				}
				ui.Text(c, label).FontSize(9).TextColor(t.TextMuted).Width(11).Shrink(0).SingleLine()
			}
		})
	})

	ui.Row(c).AlignItems(ui.Center).Gap(5).Padding(0, 20, 8, 20).Children(func() {
		ui.Text(c, fmt.Sprintf("最忙的是 %s %02d:00", weekdays[bestWD], bestH)).FontSize(10).TextColor(t.TextMuted).SingleLine()
		ui.Spacer(c)
		ui.Text(c, "少").FontSize(9).TextColor(t.TextMuted).SingleLine()
		for i := 0; i < 5; i++ {
			ui.Row(c).Width(9).Height(9).Radius(2).Background(usHeatColor(float64(i) / 4))
		}
		ui.Text(c, "多").FontSize(9).TextColor(t.TextMuted).SingleLine()
	})
}

// usSessDur writes a span of seconds, a dash when there is none.
func usSessDur(sec int64) string {
	if sec <= 0 {
		return "没有"
	}
	return FormatSeconds(sec)
}

// ---- top sessions, shape, tools and skills ----

// usSessTop lists the sessions that spent the most, under the filters, and
// a click opens one to its conversation.
func (a *App) usSessTopCard(c *ui.Context, t *ui.Theme, ov sessions.Overview) {
	ui.Row(c).AlignItems(ui.Center).Gap(8).Padding(10, 20, 2, 20).Children(func() {
		ui.Text(c, "花费最多的会话").FontSize(12.5).FontWeight(600).TextColor(t.Text).SingleLine()
		ui.Spacer(c)
		if ui.Segmented(c, &a.usSessTop, usLedLabels(usSessTops)...).Changed() {
			c.Invalidate()
		}
	})

	by := usSessTops[clampIdx(a.usSessTop, len(usSessTops))].id
	list := ov.Top[by]
	if len(list) == 0 {
		ui.Text(c, "这段时间没有可排的会话。").FontSize(10.5).TextColor(t.TextMuted).Padding(2, 20).SingleLine()
		return
	}
	for _, s := range list {
		s := s
		row := ui.Row(c).AlignItems(ui.Center).Gap(8).Padding(4, 20).FillWidth().
			Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).Label(usSessTitle(s))
		if row.Hovered() {
			row.Background(a.hoverBg(t))
		}
		if row.Clicked() {
			a.usOpenSession(s)
			c.Invalidate()
		}
		row.Children(func() {
			ui.Image(c, iconPictureFor(usAgentIcon(s.Agent))).Size(16, 16).Radius(3)
			ui.Column(c).Grow(1).Shrink(1).Gap(1).Children(func() {
				ui.Text(c, usSessTitle(s)).FontSize(11.5).TextColor(t.Text).SingleLine()
				ui.Text(c, usSessSub(s)).FontSize(9.5).TextColor(t.TextMuted).SingleLine().Tooltip(s.Cwd)
			})
			ui.Text(c, usSessTopValue(by, s)).FontSize(11.5).FontWeight(600).TextColor(t.Text).Shrink(0).SingleLine()
		})
	}
}

// usSessTopValue is a top session's figure for the measure picked.
func usSessTopValue(by string, s sessions.Summary) string {
	switch by {
	case "cost":
		if s.Cost <= 0 {
			return "—"
		}
		return "≈" + FormatCost(s.Cost)
	case "active":
		return usSessDur(s.Active)
	}
	return FormatCount(s.Input + s.Output)
}

// usSessShape counts the range's sessions by how they ran: by messages, by
// minutes at work, or by tool calls a prompt.
func (a *App) usSessShapeCard(c *ui.Context, t *ui.Theme, ov sessions.Overview) {
	by := usSessShapes[clampIdx(a.usSessShape, len(usSessShapes))].id
	sh := ov.Shape[by]

	ui.Row(c).AlignItems(ui.Center).Gap(8).Padding(10, 20, 2, 20).Children(func() {
		ui.Text(c, "会话的形状").FontSize(12.5).FontWeight(600).TextColor(t.Text).SingleLine()
		if ui.Segmented(c, &a.usSessShape, usLedLabels(usSessShapes)...).Changed() {
			c.Invalidate()
		}
		ui.Spacer(c)
		if sh.Total > 0 {
			ui.Text(c, fmt.Sprintf("%d 个会话", sh.Total)).FontSize(10).TextColor(t.TextMuted).SingleLine()
		}
	})
	if sh.Total == 0 {
		ui.Text(c, "这段时间没有可统计的会话。").FontSize(10.5).TextColor(t.TextMuted).Padding(2, 20).SingleLine()
		return
	}

	peak := 1
	for _, n := range sh.Counts {
		if n > peak {
			peak = n
		}
	}
	label := func(i int) string {
		lo := sh.Edges[i]
		hi := math.MaxInt
		if i+1 < len(sh.Edges) {
			hi = sh.Edges[i+1]
		}
		if by == "autonomy" && lo == 0 {
			return "<1"
		}
		if hi == math.MaxInt {
			return fmt.Sprintf("%d+", lo)
		}
		if hi-1 == lo {
			return fmt.Sprint(lo)
		}
		return fmt.Sprintf("%d–%d", lo, hi-1)
	}
	what := map[string]string{
		"messages": "个消息",
		"minutes":  "分钟",
		"autonomy": "次工具调用/提示",
	}[by]

	const barH = 70.0
	ui.Row(c).AlignItems(ui.End).Gap(8).Padding(4, 20, 4, 20).FillWidth().Children(func() {
		for i, n := range sh.Counts {
			i, n := i, n
			ui.Column(c).Grow(1).Basis(0).Gap(3).Children(func() {
				ui.Text(c, FormatCount(n)).FontSize(9.5).TextColor(t.TextMuted).SingleLine()
				h := float32(0)
				if n > 0 {
					h = float32(barH-16) * float32(n) / float32(peak)
					if h < 3 {
						h = 3
					}
				}
				bar := ui.Row(c).FillWidth().Height(h).Radius(2)
				if n > 0 {
					bar.Background(ui.Hex("#007aff"))
				} else {
					bar.Background(a.usTrackBg(t))
				}
				bar.Tooltip(fmt.Sprintf("%s %s · %d 个会话（%.0f%%）",
					label(i), what, n, 100*float64(n)/float64(max(1, sh.Total))))
				ui.Text(c, label(i)).FontSize(9).TextColor(t.TextMuted).SingleLine()
			})
		}
	})
	axis := map[string]string{
		"messages": "横轴：一个会话的消息数（提示与回复）· 纵轴：会话数",
		"minutes":  "横轴：一个会话工作的分钟数 · 纵轴：会话数",
		"autonomy": "横轴：每个提示里 Agent 自己发起的工具调用 · 纵轴：会话数",
	}[by]
	a.usNote(c, t, axis)
}

// usSessTools is what the sessions called their tools for: the most
// called, and the kinds of them.
func (a *App) usSessTools(c *ui.Context, t *ui.Theme, ov sessions.Overview) {
	tu := ov.Tools
	ui.Row(c).AlignItems(ui.Center).Gap(8).Padding(10, 20, 2, 20).Children(func() {
		ui.Text(c, "工具使用").FontSize(12.5).FontWeight(600).TextColor(t.Text).SingleLine()
		ui.Spacer(c)
		if tu.Calls > 0 {
			ui.Text(c, fmt.Sprintf("%s 次调用 · %d 个会话", FormatCount(tu.Calls), tu.Sessions)).
				FontSize(10).TextColor(t.TextMuted).SingleLine()
		}
	})
	if tu.Calls == 0 {
		ui.Text(c, "这段时间没有工具调用。").FontSize(10.5).TextColor(t.TextMuted).Padding(2, 20).SingleLine()
		return
	}
	peak := 1
	for _, s := range tu.Top {
		if s.Calls > peak {
			peak = s.Calls
		}
	}
	shown := min(len(tu.Top), 10)
	for i := 0; i < shown; i++ {
		s := tu.Top[i]
		ui.Row(c).AlignItems(ui.Center).Gap(8).Padding(2, 20).FillWidth().Children(func() {
			ui.Row(c).Width(8).Height(8).Radius(4).Background(usToolColor(s.Category)).Shrink(0)
			ui.Text(c, s.Name).FontSize(11).TextColor(t.Text).Width(150).Shrink(0).SingleLine().Tooltip(s.Name)
			a.usShareBar(c, t, float64(s.Calls)/float64(peak), usToolColor(s.Category), 110)
			ui.Text(c, FormatCount(s.Calls)).FontSize(10).TextColor(t.TextMuted).Width(60).Shrink(0).SingleLine()
			ui.Text(c, fmt.Sprintf("%d 个会话", s.Sessions)).FontSize(10).TextColor(t.TextMuted).Shrink(1).SingleLine()
		})
	}
	if len(tu.Categories) > 0 {
		ui.Row(c).AlignItems(ui.Center).Gap(6).Padding(4, 20, 6, 20).Children(func() {
			for _, cat := range tu.Categories {
				cat := cat
				name := cat.Name
				if name == "Other" {
					name = "其他"
				}
				a.usPill(c, fmt.Sprintf("%s %s", name, FormatCount(cat.Calls)), usToolColor(cat.Name),
					ui.RGBA(0, 0, 0, 0.05))
			}
		})
	}
}

// usSessSkills is the skills the sessions called up.
func (a *App) usSessSkills(c *ui.Context, t *ui.Theme, ov sessions.Overview) {
	sk := ov.Skills
	if sk.Calls == 0 {
		return
	}
	ui.Row(c).AlignItems(ui.Center).Gap(8).Padding(10, 20, 2, 20).Children(func() {
		ui.Text(c, "技能").FontSize(12.5).FontWeight(600).TextColor(t.Text).SingleLine()
		ui.Spacer(c)
		ui.Text(c, fmt.Sprintf("%s 次调用 · %d 个技能", FormatCount(sk.Calls), sk.Count)).
			FontSize(10).TextColor(t.TextMuted).SingleLine()
	})
	peak := 1
	for _, s := range sk.Top {
		if s.Calls > peak {
			peak = s.Calls
		}
	}
	shown := min(len(sk.Top), 10)
	for i := 0; i < shown; i++ {
		s := sk.Top[i]
		ui.Row(c).AlignItems(ui.Center).Gap(8).Padding(2, 20).FillWidth().Children(func() {
			ui.Text(c, s.Name).FontSize(11).TextColor(t.Text).Width(170).Shrink(0).SingleLine().Tooltip(s.Name)
			a.usShareBar(c, t, float64(s.Calls)/float64(peak), ui.Hex("#af52de"), 110)
			ui.Text(c, FormatCount(s.Calls)).FontSize(10).TextColor(t.TextMuted).Width(60).Shrink(0).SingleLine()
			ui.Text(c, fmt.Sprintf("%d 个会话", s.Sessions)).FontSize(10).TextColor(t.TextMuted).Shrink(1).SingleLine()
		})
	}
}

// ---- the list ----

// usSessList is the sessions at work in the range, under every filter, a
// page at a time.
func (a *App) usSessList(c *ui.Context, t *ui.Theme, rows []sessions.Summary) {
	ui.Row(c).AlignItems(ui.Center).Gap(8).Padding(12, 20, 4, 20).Children(func() {
		ui.Text(c, "会话").FontSize(12.5).FontWeight(600).TextColor(t.Text).SingleLine()
		ui.Text(c, fmt.Sprintf("%d 个", len(rows))).FontSize(11).TextColor(t.TextMuted).SingleLine()
	})
	if len(rows) == 0 {
		ui.Text(c, "没有符合筛选条件的会话。").FontSize(10.5).TextColor(t.TextMuted).Padding(2, 20).SingleLine()
		return
	}
	shown := min(len(rows), max(1, a.usSessListed))
	for i := 0; i < shown; i++ {
		a.usSessRow(c, t, rows[i])
	}
	if len(rows) > shown {
		more := ui.Button(c, fmt.Sprintf("再看 %d 个", min(200, len(rows)-shown)))
		more.Padding(4, 20)
		if more.Clicked() {
			a.usSessListed = shown + 200
			c.Invalidate()
		}
	}
}

// usSessRow is one session: who, where, what it spent, and a click to its
// conversation.
func (a *App) usSessRow(c *ui.Context, t *ui.Theme, s sessions.Summary) {
	row := ui.Row(c).AlignItems(ui.Center).Gap(9).Padding(5, 20).FillWidth().
		Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).Label(usSessTitle(s))
	if row.Hovered() {
		row.Background(a.hoverBg(t))
	}
	if row.Clicked() {
		a.usOpenSession(s)
		c.Invalidate()
	}
	row.Children(func() {
		ui.Image(c, iconPictureFor(usAgentIcon(s.Agent))).Size(18, 18).Radius(4)
		ui.Column(c).Grow(1).Shrink(1).Gap(1).Children(func() {
			ui.Text(c, usSessTitle(s)).FontSize(11.5).TextColor(t.Text).SingleLine()
			ui.Text(c, usSessSub(s)).FontSize(9.5).TextColor(t.TextMuted).SingleLine().Tooltip(s.Cwd)
		})
		ui.Column(c).Width(120).Shrink(0).Gap(1).Children(func() {
			ui.Text(c, FormatCount(s.Input+s.Output)).FontSize(11).FontWeight(600).TextColor(t.Text).SingleLine()
			ui.Text(c, fmt.Sprintf("%s 入 · %s 出", FormatCount(s.Input), FormatCount(s.Output))).
				FontSize(9).TextColor(t.TextMuted).SingleLine()
		})
		ui.Text(c, usSessCost(s)).FontSize(11).TextColor(t.TextMuted).Width(78).Shrink(0).SingleLine()
		ui.Text(c, usSessDur(s.Active)).FontSize(10).TextColor(t.TextMuted).Width(64).Shrink(0).SingleLine()
	})
}

// usSessTitle is a session's name: the one it was given, or what it was
// about.
func usSessTitle(s sessions.Summary) string {
	switch {
	case s.Title != "":
		return s.Title
	case s.Gateway:
		return "（未记录提示词）"
	}
	return "（没有提示词）"
}

// usSessSub is a session's second line: its folder, agent, model and when
// it was last at work.
func usSessSub(s sessions.Summary) string {
	var parts []string
	if s.Cwd != "" {
		parts = append(parts, filepath.Base(s.Cwd))
	} else if s.Gateway {
		parts = append(parts, "网关会话")
	}
	parts = append(parts, usAgentName(s.Agent))
	if len(s.Models) > 0 {
		parts = append(parts, s.Models[0])
	}
	if !s.Last.IsZero() {
		parts = append(parts, usAgo(s.Last))
	}
	return strings.Join(parts, " · ")
}

// usSessCost is a session's cost, a dash when no model of it is priced.
func usSessCost(s sessions.Summary) string {
	if !s.Priced || s.Cost <= 0 {
		return "—"
	}
	return "≈" + FormatCost(s.Cost)
}

// usSessNote is the line under the list: what its totals are of, and how
// many of them are shown.
func usSessNote(total, shown int) string {
	parts := []string{"合计包含原生会话与带会话 ID 的网关请求"}
	if shown < total {
		parts = append(parts, fmt.Sprintf("显示 %d / %d", shown, total))
	} else {
		parts = append(parts, fmt.Sprintf("%d 个会话", total))
	}
	return strings.Join(parts, " · ")
}
