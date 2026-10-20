//go:build nogui && echo

package shell

// The Usage page: magpie's own Usage page, drawn in native UI.
//
// Its four segments are magpie's: the Overview (the period's spend, the
// allowances, and what each agent, model, account and gateway key spent),
// the Requests (every call the gateway and the agents' session files tell
// of, a page at a time, with its trend and its heatmap), the Sessions (the
// agents' own conversations, what each cost and when they were at work),
// and the Context (what each agent's prompts hold, as the gateway read
// them).
//
// The reads are magpie's too (usageapi, usage, sessions, gui.ContextOf);
// this file and the three beside it are the drawing. A segment reads when
// it is shown or a filter changed, never in a frame: the ledger over a long
// history takes seconds, and a frame must not wait on it.

import (
	"fmt"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/yetone/magpie/internal/agent"
	"github.com/yetone/magpie/internal/usage"
	"github.com/yetone/magpie/internal/usageapi"
)

// The head's segments, in magpie's order.
var usTabs = []struct {
	id    string
	label string
}{
	{UsTabOverview, "总览"},
	{UsTabRequests, "请求"},
	{UsTabSessions, "会话"},
	{UsTabContext, "上下文"},
}

// The periods the head offers, and the usage.Period each names.
var usPeriods = []struct {
	id    string
	label string
}{
	{string(usage.Today), "今天"},
	{string(usage.Week), "7 天"},
	{string(usage.Month), "30 天"},
	{string(usage.All), "全部"},
}

// The Sessions segment's ranges, with how many days each reaches back (0 is
// every session the agent files hold).
var usSessRanges = []struct {
	id    string
	label string
	days  int
}{
	{"today", "今天", 1},
	{"7d", "7 天", 7},
	{"30d", "30 天", 30},
	{"90d", "90 天", 90},
	{"all", "全部", 0},
}

// The Context segment's ranges.
var usCtxRanges = []struct {
	id   string
	days int
}{
	{"1", 1},
	{"7", 7},
	{"30", 30},
}

// usCtxLabels are those ranges as the strip shows them.
var usCtxLabels = []string{"今天", "7 天", "30 天"}

// usageQuery is the page's state as the one value its reads are keyed by.
func (a *App) usageQuery() UsageQuery {
	return UsageQuery{
		Tab:        a.usTab,
		Period:     a.usPeriod,
		Day:        a.usSelectedDay,
		Agent:      a.usAgentFilter,
		Provider:   a.usProviderFilter,
		Purpose:    a.usPurposeFilter,
		Account:    a.usAccountFilter,
		CallerKey:  a.usKeyFilter,
		Model:      a.usModelFilter,
		Status:     a.usStatusFilter,
		Query:      a.usSearchQuery,
		Via:        a.usViaFilter,
		Offset:     a.usLedOffset,
		Limit:      50,
		SessRange:  a.usSessRange,
		SessAgent:  a.usSessAgent,
		SessModel:  a.usSessModel,
		SessFolder: a.usSessFolder,
		SessQuery:  a.usSessQuery,
		CtxDays:    a.usCtxDays,
	}
}

// usQueryKey is the query as a comparable string: a filter that changed
// makes another key, and the page reads again for it.
func usQueryKey(q UsageQuery) string {
	return strings.Join([]string{
		q.Tab, string(q.Period), q.Day, q.Agent, q.Provider, q.Purpose,
		q.Account, q.CallerKey, q.Model, q.Status, q.Query, q.Via,
		fmt.Sprint(q.Offset), fmt.Sprint(q.Limit), q.SessRange, q.SessAgent,
		q.SessModel, q.SessFolder, q.SessQuery, fmt.Sprint(q.CtxDays),
	}, "\x00")
}

// usEnsure reads the page's data when what it holds was read for another
// query, off the main thread, and hands it back to the view when it
// arrives. A read in flight for the same query is left to finish; one for
// another query is superseded, and its answer dropped.
func (a *App) usEnsure() {
	q := a.usageQuery()
	key := usQueryKey(q)
	if a.usFresh == key || a.usPending == key {
		return
	}
	a.usPending = key
	gen := a.usGen + 1
	a.usGen = gen
	a.usBusy = true
	if a.win != nil {
		a.win.Invalidate()
	}
	go func() {
		d := ReadUsageData(q)
		apply := func() {
			if a.usGen != gen {
				return // a later read is wanted instead
			}
			a.data.Usage = d
			a.usFresh = key
			a.usPending = ""
			a.usBusy = false
		}
		if a.win == nil {
			apply()
			return
		}
		a.win.Update(apply)
	}()
}

// usReload reads the page again for the filters it shows, as magpie's
// Refresh does: the allowances are read from their vendors too.
func (a *App) usReload(asked bool) {
	if asked {
		quotasCacheMu.Lock()
		quotasLastRead = time.Time{}
		quotasCacheMu.Unlock()
	}
	a.usFresh = ""
	a.usEnsure()
}

// LoadUsageNow reads the page's data on the calling thread, which is what a
// test does: with no window there is no frame to wait for.
func (a *App) LoadUsageNow() {
	d := ReadUsageData(a.usageQuery())
	a.data.Usage = d
	a.usFresh = usQueryKey(d.Query)
	a.usPending = ""
	a.usBusy = false
}

// usageView draws the page: the head every segment shares, then the
// segment itself.
func (a *App) usageView(c *ui.Context) {
	if a.usTab == "" {
		a.usTab = UsTabOverview
	}
	a.usEnsure()
	t := c.Theme()
	ui.Column(c).FillWidth().Children(func() {
		a.usageHead(c, t)
		if a.usNotice != "" {
			a.usageNoticeBar(c, t)
		}
		switch a.usTab {
		case UsTabRequests:
			a.usageRequests(c, t)
		case UsTabSessions:
			a.usageSessions(c, t)
		case UsTabContext:
			a.usageContext(c, t)
		default:
			a.usageOverview(c, t)
		}
	})
}

// usageHead is the row above every segment: the segments, the period (or
// the range the segment keeps), Refresh, and what the period cost.
func (a *App) usageHead(c *ui.Context, t *ui.Theme) {
	// The segment and period strips point at indexes, as MyGo's Segmented
	// takes them: the fields are set from the state the page holds, and a
	// click writes a new one back.
	for i, s := range usTabs {
		if s.id == a.usTab {
			a.usTabIdx = i
		}
	}
	for i, p := range usPeriods {
		if p.id == string(a.usPeriod) {
			a.usPeriodIdx = i
		}
	}
	for i, r := range usSessRanges {
		if r.id == a.usSessRange {
			a.usSessRangeIdx = i
		}
	}
	for i, r := range usCtxRanges {
		if r.id == fmt.Sprint(a.usCtxDays) {
			a.usCtxDaysIdx = i
		}
	}

	lineC := ui.RGBA(0, 0, 0, 0.08)
	if t.Dark {
		lineC = ui.RGBA(255, 255, 255, 0.08)
	}

	ui.Column(c).FillWidth().BorderWidth(0, 0, 0.5, 0).BorderColor(lineC).Children(func() {
		ui.Row(c).AlignItems(ui.Center).Gap(10).Padding(12, 20, 10, 20).Children(func() {
			labels := make([]string, len(usTabs))
			for i, s := range usTabs {
				labels[i] = s.label
			}
			if ui.Segmented(c, &a.usTabIdx, labels...).Changed() {
				if a.usTabIdx >= 0 && a.usTabIdx < len(usTabs) {
					a.usTab = usTabs[a.usTabIdx].id
					a.usLedOffset = 0
					a.usSelectedDay = ""
					a.usFresh = ""
					c.Invalidate()
				}
			}

			switch a.usTab {
			case UsTabOverview, UsTabRequests:
				pl := make([]string, len(usPeriods))
				for i, p := range usPeriods {
					pl[i] = p.label
				}
				if ui.Segmented(c, &a.usPeriodIdx, pl...).Changed() {
					if a.usPeriodIdx >= 0 && a.usPeriodIdx < len(usPeriods) {
						a.usPeriod = usage.Period(usPeriods[a.usPeriodIdx].id)
						a.usLedOffset = 0
						a.usSelectedDay = ""
						a.usFresh = ""
						c.Invalidate()
					}
				}
			case UsTabSessions:
				rl := make([]string, len(usSessRanges))
				for i, r := range usSessRanges {
					rl[i] = r.label
				}
				if ui.Segmented(c, &a.usSessRangeIdx, rl...).Changed() {
					if a.usSessRangeIdx >= 0 && a.usSessRangeIdx < len(usSessRanges) {
						a.usSessRange = usSessRanges[a.usSessRangeIdx].id
						a.usSessListed = 200
						a.usFresh = ""
						c.Invalidate()
					}
				}
			case UsTabContext:
				rl := make([]string, len(usCtxRanges))
				for i := range usCtxRanges {
					rl[i] = usCtxLabels[i]
				}
				if ui.Segmented(c, &a.usCtxDaysIdx, rl...).Changed() {
					if a.usCtxDaysIdx >= 0 && a.usCtxDaysIdx < len(usCtxRanges) {
						a.usCtxDays = usCtxRanges[a.usCtxDaysIdx].days
						a.usFresh = ""
						c.Invalidate()
					}
				}
			}

			ui.Spacer(c)

			if a.usBusy {
				ui.Spinner(c).Size(12, 12)
			}

			// Refresh: the segment read again, the allowances asked of
			// their vendors too, as magpie's own head does
			refresh := ui.Row(c).AlignItems(ui.Center).Gap(5).Padding(4, 9).Radius(6).
				Background(a.fieldBg(t)).Border(1, t.Border).
				Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).Label("刷新").
				Tooltip("立即重新读取这一页")
			if refresh.Hovered() {
				refresh.Background(a.hoverBg(t))
			}
			if refresh.Clicked() {
				a.usReload(true)
			}
			refresh.Children(func() {
				ui.Icon(c, refreshIcon()).Size(12, 12).TextColor(t.TextMuted)
				ui.Text(c, "刷新").FontSize(11.5).TextColor(t.TextMuted).SingleLine()
			})

			// What the period cost, and how many calls had no known
			// price: a total never reads as if those cost nothing.
			cost, unpriced := a.usHeadCost()
			if cost > 0 || unpriced > 0 {
				ui.Row(c).AlignItems(ui.Center).Gap(5).Shrink(0).Children(func() {
					if cost > 0 {
						ui.Text(c, "≈"+FormatCost(cost)).FontSize(12.5).FontWeight(600).TextColor(t.Text).SingleLine()
					}
					note := "按有效价格估算"
					noteC := t.TextMuted
					if unpriced > 0 {
						note = fmt.Sprintf("%s · %d 次调用无价格", note, unpriced)
						noteC = ui.Hex("#b45309")
					}
					ui.Text(c, note).FontSize(11).TextColor(noteC).SingleLine()
				})
			}
		})
	})
}

// usHeadCost is the head's sum: the period's cost and its unpriced calls,
// from whichever segment is shown.
func (a *App) usHeadCost() (float64, int) {
	d := a.data.Usage
	switch a.usTab {
	case UsTabRequests:
		return d.Ledger.Cost, d.Ledger.Unpriced
	case UsTabSessions:
		var cost float64
		unpriced := map[string]bool{}
		for _, s := range d.SessionsList {
			cost += s.Cost
			if !s.Priced {
				for _, m := range s.Models {
					unpriced[m] = true
				}
			}
		}
		return cost, len(unpriced)
	}
	return d.Overview.Cost, d.Overview.Unpriced
}

// usageNoticeBar is the page's last write, said over it and gone when it
// has been read.
func (a *App) usageNoticeBar(c *ui.Context, t *ui.Theme) {
	bg := ui.RGBA(52, 199, 89, 0.12)
	borderC := ui.RGBA(52, 199, 89, 0.3)
	textC := ui.Hex("#1f7a3a")
	if t.Dark {
		textC = ui.Hex("#7ee2a0")
	}
	if a.usNoticeErr {
		bg = ui.RGBA(255, 59, 48, 0.10)
		borderC = ui.RGBA(255, 59, 48, 0.28)
		textC = ui.Hex("#c0392b")
		if t.Dark {
			textC = ui.Hex("#ff8a80")
		}
	}
	ui.Row(c).AlignItems(ui.Center).Gap(8).Padding(7, 14).Radius(8).Margin(10, 20, 0, 20).
		Background(bg).Border(1, borderC).FillWidth().Children(func() {
		ui.Text(c, a.usNotice).FontSize(11.5).TextColor(textC).Grow(1).Shrink(1).SingleLine()
		closeBtn := ui.Text(c, "✕").FontSize(11).TextColor(textC).Cursor(ui.CursorPointer)
		if closeBtn.Clicked() {
			a.usNotice = ""
			c.Invalidate()
		}
	})
}

// usSetNotice says what the page's last write did, and takes the words
// away after a while as magpie's status line does.
func (a *App) usSetNotice(msg string, isErr bool) {
	a.usNotice = msg
	a.usNoticeErr = isErr
	if a.usNoticeTimer != nil {
		a.usNoticeTimer.Stop()
	}
	a.usNoticeTimer = time.AfterFunc(6*time.Second, func() {
		if a.win != nil {
			a.win.Update(func() {
				a.usNotice = ""
				a.win.Invalidate()
			})
			return
		}
		a.usNotice = ""
	})
}

// ---- small pieces the segments share ----

// usSectionHead is a list's heading: its name, whether it is folded, and
// what it is of.
func (a *App) usSectionHead(c *ui.Context, t *ui.Theme, id, title, note string, foldable bool) {
	ui.Row(c).AlignItems(ui.Center).Gap(6).Padding(14, 20, 4, 20).Children(func() {
		if foldable {
			folded := a.usFolds[id]
			btn := ui.Row(c).AlignItems(ui.Center).Gap(4).Cursor(ui.CursorPointer).
				Focusable().Role(ui.RoleButton).Label(title)
			if btn.Clicked() {
				a.usFolds[id] = !folded
				c.Invalidate()
			}
			btn.Children(func() {
				chev := "▾"
				if folded {
					chev = "▸"
				}
				ui.Text(c, chev).FontSize(10).TextColor(t.TextMuted).SingleLine()
				ui.Text(c, title).FontSize(12.5).FontWeight(600).TextColor(t.Text).SingleLine()
			})
		} else {
			ui.Text(c, title).FontSize(12.5).FontWeight(600).TextColor(t.Text).SingleLine()
		}
		if note != "" {
			ui.Text(c, note).FontSize(11).TextColor(t.TextMuted).SingleLine()
		}
	})
}

// usNote is the line under a segment saying what its numbers are of.
func (a *App) usNote(c *ui.Context, t *ui.Theme, text string) {
	if text == "" {
		return
	}
	ui.Text(c, text).FontSize(10.5).TextColor(t.TextMuted).
		Padding(8, 20, 10, 20).FillWidth()
}

// usEmpty is what a segment says when it has nothing to draw.
func (a *App) usEmpty(c *ui.Context, t *ui.Theme, title, sub string) {
	ui.Column(c).FillWidth().Padding(40, 20).AlignItems(ui.Center).Gap(6).Children(func() {
		ui.Text(c, title).FontSize(13.5).FontWeight(600).TextColor(t.Text).SingleLine()
		if sub != "" {
			ui.Text(c, sub).FontSize(11.5).TextColor(t.TextMuted).SingleLine()
		}
	})
}

// usKPI is one tile of a segment's strip: the figure, what it is, and a
// line under it that says what it is made of.
func (a *App) usKPI(c *ui.Context, t *ui.Theme, value, label, sub string, subColor ui.Color) {
	ui.Column(c).Grow(1).Shrink(1).Basis(0).Padding(10, 12).Radius(9).
		Background(a.fieldBg(t)).Border(1, t.Border).Gap(2).Children(func() {
		ui.Text(c, value).FontSize(19).FontWeight(700).TextColor(t.Text).SingleLine()
		ui.Text(c, label).FontSize(11).TextColor(t.TextMuted).SingleLine()
		if sub != "" {
			ui.Text(c, sub).FontSize(10).TextColor(subColor).SingleLine()
		}
	})
}

// usShareBar is a row's share of a total, a bar in its track. The fill's
// width is the track's own, not a percentage: a track of a fixed width
// lays a percentage out against what it is given, and a row that sizes
// itself to its children gives it nothing.
func (a *App) usShareBar(c *ui.Context, t *ui.Theme, share float64, color ui.Color, width float32) {
	track := ui.Row(c).Width(width).Height(5).Radius(3).Background(a.usTrackBg(t)).Shrink(0)
	track.Children(func() {
		p := share
		if p > 1 {
			p = 1
		}
		if p < 0.015 && p > 0 {
			p = 0.015 // a share too small to see still shows it is there
		}
		ui.Row(c).Width(width * float32(p)).Height(5).Radius(3).Background(color)
	})
}

// usTrackBg is the empty part of a bar.
func (a *App) usTrackBg(t *ui.Theme) ui.Color {
	if t.Dark {
		return ui.RGBA(255, 255, 255, 0.08)
	}
	return ui.RGBA(0, 0, 0, 0.06)
}

// usHoverBg is the fill a row takes under the pointer.
func (a *App) hoverBg(t *ui.Theme) ui.Color {
	if t.Dark {
		return ui.RGBA(255, 255, 255, 0.06)
	}
	return ui.RGBA(0, 0, 0, 0.045)
}

// usPill is a small tag: the segment's filters, a source badge, a tag.
func (a *App) usPill(c *ui.Context, text string, fg, bg ui.Color) {
	ui.Row(c).AlignItems(ui.Center).Padding(1.5, 6).Radius(4).Background(bg).Shrink(0).Children(func() {
		ui.Text(c, text).FontSize(10).FontWeight(600).TextColor(fg).SingleLine()
	})
}

// usMutedPill is usPill in the page's quiet colours.
func (a *App) usMutedPill(c *ui.Context, t *ui.Theme, text string) {
	bg := ui.RGBA(0, 0, 0, 0.05)
	if t.Dark {
		bg = ui.RGBA(255, 255, 255, 0.07)
	}
	a.usPill(c, text, t.TextMuted, bg)
}

// usAgentName is an agent's name in the reader's language, from the id
// the ledger records.
func usAgentName(id string) string {
	for _, a := range agent.Clients() {
		if a.ID == id {
			return a.Name
		}
	}
	if id == "" {
		return "未知"
	}
	return id
}

// usAgentIcon is an agent's logo, or the generic mark.
func usAgentIcon(id string) string {
	for _, a := range agent.Clients() {
		if a.ID == id {
			return a.Icon
		}
	}
	return "generic"
}

// usModelLabel shortens a model id for a narrow column, keeping the tail
// that tells the versions apart.
func usModelLabel(m string) string {
	if m == "" {
		return "—"
	}
	return m
}

// usSourceBadge says where a request row came from: the gateway, an
// agent's own session file, or an app that reported it.
func (a *App) usSourceBadge(c *ui.Context, t *ui.Theme, row usageapi.Row) {
	switch {
	case row.Source == "log":
		a.usPill(c, "本地会话", ui.Hex("#5b5bd6"), ui.RGBA(91, 91, 214, 0.10))
	case row.Source == usage.SourceReported:
		a.usPill(c, "应用上报", ui.Hex("#0f766e"), ui.RGBA(15, 118, 110, 0.10))
	case row.Access == "subscription":
		a.usPill(c, "订阅", ui.Hex("#0f766e"), ui.RGBA(15, 118, 110, 0.10))
	case row.Access == "api":
		a.usPill(c, "API", ui.Hex("#8a5a00"), ui.RGBA(138, 90, 0, 0.10))
	}
}

// usStatusTone is how a failed call is coloured, and what its dot says.
func usStatusTone(t *ui.Theme, failed bool) ui.Color {
	if failed {
		return ui.Hex("#ff3b30")
	}
	if t.Dark {
		return ui.Hex("#30d158")
	}
	return ui.Hex("#34c759")
}

// usPageNumbers is "{a}–{b} of {n}" for the pager.
func usPageNumbers(offset, shown, total int) string {
	if total == 0 {
		return "0"
	}
	return fmt.Sprintf("%s–%s / %s", FormatCount(offset+1), FormatCount(offset+shown), FormatCount(total))
}
