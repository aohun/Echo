//go:build nogui && echo

package shell

// The tray panel: the small popover the menu bar icon opens, as magpie's
// own panel is. Four tabs, magpie's four: the agents this machine has, the
// allowances of every subscription, the usage of the period, and the routing
// groups. The readings are the pages' own (ReadQuotas, groupQuotas, the
// usage page's summary, the routing page's groups), so the panel and the
// page cannot say different things about the same accounts. The panel
// closes as the user clicks elsewhere, presses Escape or clicks the icon
// again; 打开 Echo brings the main window up.

import (
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/yetone/magpie/internal/provider"
	"github.com/yetone/magpie/internal/usage"
	"github.com/yetone/magpie/internal/usageapi"
)

// The panel's four tabs, magpie's four.
const (
	panelTabAgents  = "agents"
	panelTabQuota   = "quota"
	panelTabUsage   = "usage"
	panelTabRouting = "routing"
)

// panelAgentRows is how many agents the panel lists before it says how many
// more there are: magpie's panel lists a few and keeps the rest one click
// away.
const panelAgentRows = 5

// togglePanel opens the panel by the tray icon, or closes it.
func (a *App) togglePanel() {
	if a.trayPanel != nil {
		a.trayPanel.Close()
		return
	}
	a.panelTab = panelTabQuota
	a.panelAllAgents = false
	// what the cache holds shows at once; the vendors are asked behind it
	a.panelQuotas = PeekQuotas()
	a.panelReadAt = time.Now()
	a.panelRefreshing = false
	a.trayPanel = mygo.NewFlyout(mygo.FlyoutOptions{
		Tray: a.tray,
		// Below the icon of the macOS menu bar; a Windows taskbar's icon
		// has it above, a Linux one in the work area's middle.
		Placement: mygo.PlacementBottom,
		Gap:       4,
		Width:     400,
		Height:    540,
		Focusable: true,
		Shadow:    true,
		Popover:   true,
		Content:   ui.View(a.panelView),
	})
	a.trayPanel.OnClosed(func() {
		a.trayPanel = nil
		if a.panelStop != nil {
			close(a.panelStop)
			a.panelStop = nil
		}
	})
	// while the panel is up, read again now and then, as magpie's own
	// panel's page does: a popover left open outlives the vendors' caches
	a.panelStop = make(chan struct{})
	stop := a.panelStop
	tick := time.NewTicker(time.Minute)
	go func() {
		defer tick.Stop()
		for {
			select {
			case <-stop:
				return
			case <-tick.C:
				a.refreshPanelQuotas(false)
			}
		}
	}()
	a.refreshPanelQuotas(false)
}

// closePanel takes the panel down, when it is up.
func (a *App) closePanel() {
	if a.trayPanel != nil {
		a.trayPanel.Close()
	}
}

// updatePanel runs fn against the panel's window, when the panel is up.
func (a *App) updatePanel(fn func()) {
	if w := a.trayPanel; w != nil {
		w.Update(fn)
	}
}

// refreshPanelQuotas reads the allowance cards again, behind the panel.
func (a *App) refreshPanelQuotas(force bool) {
	go func() {
		qs := ReadQuotas(force)
		a.updatePanel(func() {
			a.panelQuotas = qs
			a.panelReadAt = time.Now()
		})
	}()
}

// refreshPanelAll reads every subscription's card again from its vendor,
// one after another as the Usage page's per-card refresh is, the panel
// showing each answer as it lands. The button that asks for it turns
// itself back on when the last card is in.
func (a *App) refreshPanelAll() {
	for _, g := range groupQuotas(a.panelQuotas) {
		qs := RefreshQuotaCard(g.Provider, "")
		a.updatePanel(func() {
			a.panelQuotas = qs
			a.panelReadAt = time.Now()
		})
	}
	qs := ReadQuotas(true)
	a.updatePanel(func() {
		a.panelQuotas = qs
		a.panelReadAt = time.Now()
		a.panelRefreshing = false
	})
}

// panelView is the panel's whole interface.
func (a *App) panelView(c *ui.Context) {
	t := c.Theme()
	c.Root().Background(ui.Transparent)
	box := ui.Column(c).Fill()
	switch {
	case c.Vibrancy():
		// Over the popover's material on macOS, magpie's panel paints its
		// own colour on it: the background at 93% (--panel-tint in its
		// app.css), so the panel reads as one surface and the material
		// shows through a little — never a bare transparent panel.
		box.Background(t.Surface.Alpha(0.93)).Radius(12)
	case runtime.GOOS == "windows": // which rounds and outlines the flyout
		box.Background(t.Surface).Radius(12).Border(1, t.Border)
	default:
		box.Background(t.Surface).Radius(12).Border(1, t.Border)
	}

	box.Children(func() {
		a.panelHeader(c, t)
		a.panelTabs(c, t)

		// the tab's content
		switch a.panelTab {
		case panelTabQuota:
			a.panelQuotaTab(c, t)
		case panelTabUsage:
			a.panelUsageTab(c, t)
		case panelTabRouting:
			a.panelRoutingTab(c, t)
		default:
			a.panelAgentTab(c, t)
		}

		a.panelFooter(c, t)
	})
}

// panelHeader is magpie's panel's first line: the app's mark and its name,
// and the three buttons that act on the panel.
//
// The first two act on the tab they stand on, as magpie's do: its refresh
// reads what the page shown reads (its refreshKind), and its "open" brings
// the window up on that page.
func (a *App) panelHeader(c *ui.Context, t *ui.Theme) {
	ui.Row(c).AlignItems(ui.Center).Gap(8).Padding(10, 12, 4, 12).FillWidth().Children(func() {
		ui.Image(c, appIconPicture()).Size(20, 20).Radius(5)
		ui.Text(c, a.text.S.AppName).FontSize(13).FontWeight(600).TextColor(t.Text).Shrink(0).SingleLine()
		ui.Spacer(c)

		// read again: what the tab shows
		ref := ui.Row(c).Center().Width(24).Height(24).Radius(6).Shrink(0).
			Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).
			Label(a.panelRefreshTip()).Tooltip(a.panelRefreshTip())
		if ref.Hovered() || a.panelRefreshing {
			ref.Background(a.hoverBg(t))
		}
		if ref.Clicked() {
			a.panelRefresh(c)
		}
		ref.Children(func() {
			ui.Icon(c, refreshIcon()).Size(13, 13).TextColor(t.TextMuted)
		})

		// open as a window, on the page this tab stands for
		open := ui.Row(c).Center().Width(24).Height(24).Radius(6).Shrink(0).
			Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).
			Label(a.panelOpenTip()).Tooltip(a.panelOpenTip())
		if open.Hovered() {
			open.Background(a.hoverBg(t))
		}
		if open.Clicked() {
			a.panelOpen()
		}
		open.Children(func() {
			ui.Icon(c, openWindowIcon()).Size(13, 13).TextColor(t.TextMuted)
		})

		// the settings
		prefs := ui.Row(c).Center().Width(24).Height(24).Radius(6).Shrink(0).
			Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).
			Label("设置").Tooltip("设置")
		if prefs.Hovered() {
			prefs.Background(a.hoverBg(t))
		}
		if prefs.Clicked() {
			a.closePanel()
			a.Show(PageSettings)
		}
		prefs.Children(func() {
			ui.Icon(c, gearIcon()).Size(14, 14).TextColor(t.TextMuted)
		})
	})
}

// panelRefresh reads again what the tab shows, as magpie's refresh button
// does on the page it stands on: the agents on this computer, a
// subscription's card, the usage, or the routing groups.
func (a *App) panelRefresh(c *ui.Context) {
	switch a.panelTab {
	case panelTabQuota:
		// every subscription's card again, from its vendor
		if a.panelRefreshing {
			return
		}
		a.panelRefreshing = true
		c.Invalidate()
		go a.refreshPanelAll()
	case panelTabUsage:
		go func() {
			u := ReadUsageData(UsageQuery{Tab: UsTabOverview, Period: usage.Month, Limit: 50, SessRange: "30d", CtxDays: 7})
			a.updatePanel(func() { a.data.Usage = u })
		}()
	case panelTabRouting:
		go func() {
			r := readRoutingData()
			a.updatePanel(func() { a.data.Routing = r })
		}()
	default:
		// the agents on this computer again: one just installed or
		// removed, or its settings changed
		go func() {
			agents, installs, profiles := readAgentPageData()
			a.updatePanel(func() {
				a.data.Agents = agents
				a.data.AgentInstalls = installs
				a.data.AgentProfiles = profiles
			})
		}()
	}
}

// panelRefreshTip says what the refresh button reads on the tab it stands
// on, as magpie's syncTitle does.
func (a *App) panelRefreshTip() string {
	switch a.panelTab {
	case panelTabQuota:
		return "重新读取各订阅的额度"
	case panelTabUsage:
		return "重新读取用量"
	case panelTabRouting:
		return "重新读取路由组与模型列表"
	default:
		return "重新查找这台机器上的 Agent：装了新的、删掉了，或配置变了"
	}
}

// panelOpen brings the window up on the page the tab stands for.
func (a *App) panelOpen() {
	switch a.panelTab {
	case panelTabQuota:
		a.usTab = UsTabOverview
		a.Show(PageUsage)
	case panelTabUsage:
		a.usTab = UsTabRequests
		a.Show(PageUsage)
	case panelTabRouting:
		a.Show(PageRouting)
	default:
		a.Show(PageAgents)
	}
}

// panelOpenTip says where the open button goes.
func (a *App) panelOpenTip() string {
	switch a.panelTab {
	case panelTabQuota:
		return "打开主窗口的用量页（额度）"
	case panelTabUsage:
		return "打开主窗口的用量页（请求）"
	case panelTabRouting:
		return "打开主窗口的路由页"
	default:
		return "打开主窗口的 Agent 页"
	}
}

// panelTabs is the panel's tab strip: the allowances, the one the tray
// panel shows. The other readings — the agents, the usage, the routing —
// are the main window's business, and their tabs are hidden here.
func (a *App) panelTabs(c *ui.Context, t *ui.Theme) {
	ui.Row(c).AlignItems(ui.Center).Gap(2).Padding(0, 12, 8, 12).FillWidth().Children(func() {
		pill := ui.Row(c).AlignItems(ui.Center).Center().Gap(3).Grow(1).Height(26).Radius(7).
			Background(t.Background).Border(1, t.Border).
			Shadow(0, 1, 2, 0, ui.RGBA(0, 0, 0, 0.06)).
			Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).
			Label(a.text.S.PanelQuota)
		if pill.Clicked() {
			a.panelTab = panelTabQuota
			a.refreshPanelQuotas(false)
			c.Invalidate()
			a.invalidate()
		}
		pill.Children(func() {
			ui.Text(c, a.text.S.PanelQuota).FontSize(12).FontWeight(550).TextColor(t.Text).SingleLine()
		})
	})
}

// panelAgents are the agents the panel lists: the ones on this machine.
// The ones Echo knows but has not found are the Agents page's business, not
// the menu bar's — magpie's panel counts and lists the agents it found too.
func (a *App) panelAgents() []AgentRow {
	out := make([]AgentRow, 0, len(a.data.Agents))
	for _, ag := range a.data.Agents {
		if ag.Found {
			out = append(out, ag)
		}
	}
	return out
}

// panelAgentTab is the agents, as magpie's panel lists them: a card of
// rows, each with the agent's mark, its name and the model it runs on, and
// the rest of them under a line that says how many there are.
func (a *App) panelAgentTab(c *ui.Context, t *ui.Theme) {
	agents := a.panelAgents()
	if len(agents) == 0 {
		a.panelEmpty(c, t, "没有找到编码 Agent。", "装好一个，或从资源库安装。")
		return
	}
	rest := 0
	if !a.panelAllAgents && len(agents) > panelAgentRows {
		rest = len(agents) - panelAgentRows
		agents = agents[:panelAgentRows]
	}

	ui.Scroll(c).Grow(1).Shrink(1).FillWidth().Padding(0, 12, 8, 12).Children(func() {
		card := ui.Column(c).FillWidth().Radius(10).
			Background(t.Background).Border(1, t.Border)
		card.Children(func() {
			for i, ag := range agents {
				a.panelAgentRow(c, t, ag, i == 0)
			}
			if rest > 0 || a.panelAllAgents {
				a.panelAgentMore(c, t, rest)
			}
		})
	})
}

// panelAgentRow is one agent in the panel: its mark, its name, the model it
// runs on, and the effort's bars — magpie's panel row.
func (a *App) panelAgentRow(c *ui.Context, t *ui.Theme, ag AgentRow, first bool) {
	row := ui.Row(c).AlignItems(ui.Center).Gap(10).Height(36).Padding(0, 12).FillWidth().
		Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).
		Label(ag.Name)
	if !first {
		row.BorderWidth(0.5, 0, 0, 0).BorderColor(a.panelLine(t))
	}
	if row.Hovered() {
		row.Background(a.hoverBg(t))
	}
	if row.Clicked() {
		a.closePanel()
		a.Show(PageAgents)
	}
	row.Children(func() {
		// the mark, with the drift badge on its corner, as magpie's
		mark := ui.Box(c).Width(18).Height(18).Shrink(0)
		mark.Children(func() {
			ui.Image(c, iconPictureFor(ag.Icon)).Size(18, 18).Radius(4)
			if ag.Drift {
				badge := ui.Box(c).Absolute().Right(-4).Bottom(-4).
					Width(11).Height(11).Radius(6).
					Background(driftColor(t)).Border(2, t.Background)
				badge.Children(func() {
					ui.Icon(c, reapplyIcon()).Size(7, 7).TextColor(t.Background)
				})
			}
		})

		ui.Text(c, ag.Name).FontSize(12.5).FontWeight(600).TextColor(t.Text).
			Grow(1).Shrink(1).SingleLine()

		// the model it runs on, in words, as magpie's panel says it: the
		// agent's own pick, or the models its menu lists
		model := ag.CurrentName
		if ag.ModelList {
			model = agentModelsSaid(ag.ModelNames, ag.ModelCount)
		}
		ui.Text(c, model).FontSize(12).TextColor(t.TextMuted).
			Shrink(1).SingleLine().MaxWidth(200).Tooltip(model)

		a.panelEffortBars(c, t, ag)

		ui.Text(c, "›").FontSize(13).TextColor(t.TextMuted).Shrink(0).SingleLine()
	})
}

// panelEffortBars is the agent's effort at a glance: three bars, as many
// lit as the level is high, magpie's way of saying it without words.
func (a *App) panelEffortBars(c *ui.Context, t *ui.Theme, ag AgentRow) {
	lit := panelEffortLit(ag.Effort)
	if lit == 0 {
		return
	}
	bars := ui.Row(c).AlignItems(ui.End).Gap(1.5).Shrink(0)
	bars.Children(func() {
		for i, h := range []float32{4, 7, 10} {
			ink := t.TextMuted
			if i < lit {
				ink = t.Text
			}
			ui.Row(c).Width(2.5).Height(h).Radius(1.25).Background(ink)
		}
	})
}

// panelEffortLit is how many of the three bars an effort level lights: the
// levels magpie knows, in their order, as a share of three.
func panelEffortLit(effort string) int {
	switch effort {
	case "minimal", "low":
		return 1
	case "medium", "high":
		return 2
	case "xhigh", "max":
		return 3
	}
	return 0
}

// panelAgentMore is the line under the list that says how many agents are
// not in it, and shows them: magpie's "show the rest".
func (a *App) panelAgentMore(c *ui.Context, t *ui.Theme, rest int) {
	btn := ui.Row(c).AlignItems(ui.Center).Center().Gap(5).FillWidth().Height(34).
		BorderWidth(0.5, 0, 0, 0).BorderColor(a.panelLine(t)).
		Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).
		Label(a.panelMoreWord(rest))
	if btn.Hovered() {
		btn.Background(a.hoverBg(t))
	}
	if btn.Clicked() {
		a.panelAllAgents = !a.panelAllAgents
		c.Invalidate()
		a.invalidate()
	}
	btn.Children(func() {
		ui.Text(c, a.panelMoreWord(rest)).FontSize(12).TextColor(t.TextMuted).SingleLine()
		chev := chevDown()
		if a.panelAllAgents {
			chev = chevRight()
		}
		ui.Icon(c, chev).Size(9, 9).TextColor(t.TextMuted)
	})
}

// panelMoreWord is the line's words: how many agents are not listed, or
// that they are all listed now.
func (a *App) panelMoreWord(rest int) string {
	if a.panelAllAgents {
		return "收起"
	}
	return fmt.Sprintf("显示其余 %d 个", rest)
}

// panelQuotaTab is the allowances, as magpie's panel draws them: a group a
// provider, and under it a card an account — its windows as rings.
func (a *App) panelQuotaTab(c *ui.Context, t *ui.Theme) {
	if len(a.panelQuotas) == 0 {
		a.panelEmpty(c, t, "没有可显示的订阅。", "在供应商页登录一个订阅后，这里显示它的额度。")
		return
	}
	ui.Scroll(c).Grow(1).Shrink(1).FillWidth().Padding(2, 10).Children(func() {
		ui.Column(c).FillWidth().Gap(10).Children(func() {
			for _, g := range groupQuotas(a.panelQuotas) {
				a.panelQuotaGroup(c, t, g)
			}
			// the reading's own age, not the panel's: each card says when
			// its vendor read it, and the oldest of those is how old what
			// this tab shows is — a kept card is not passed off as fresh
			at := a.panelReadAt
			oldest := time.Time{}
			for _, q := range a.panelQuotas {
				if q.ReadAt != nil && !q.ReadAt.IsZero() && (oldest.IsZero() || q.ReadAt.Before(oldest)) {
					oldest = *q.ReadAt
				}
			}
			if !oldest.IsZero() {
				at = oldest
			}
			if !at.IsZero() {
				ui.Text(c, "更新于 "+usAgo(at)).FontSize(9.5).TextColor(t.TextMuted).
					Padding(0, 4).SingleLine()
			}
		})
	})
}

// panelQuotaGroup is one provider's group: its name on a line of its own,
// and the accounts under it — magpie's group.
func (a *App) panelQuotaGroup(c *ui.Context, t *ui.Theme, g quotaGroup) {
	ui.Column(c).FillWidth().Gap(6).Children(func() {
		ui.Row(c).AlignItems(ui.Center).Gap(7).Children(func() {
			ui.Image(c, iconPictureFor(g.Icon)).Size(14, 14).Radius(3)
			ui.Text(c, strings.ToUpper(g.Name)).FontSize(10.5).FontWeight(600).
				TextColor(t.TextMuted).Grow(1).Shrink(1).SingleLine()
			if g.Plan != "" {
				ui.Text(c, g.Plan).FontSize(11).TextColor(t.TextMuted).Shrink(0).SingleLine()
			}
		})
		for _, q := range g.Quotas {
			a.panelQuotaAccount(c, t, q)
		}
	})
}

// panelQuotaAccount is one account: who it is and when its windows reset on
// the left, and its windows as rings on the right — magpie's card.
func (a *App) panelQuotaAccount(c *ui.Context, t *ui.Theme, q provider.SubscriptionQuota) {
	if q.User == "" && q.Error == "" && len(q.Windows) == 0 {
		return
	}
	ui.Row(c).AlignItems(ui.Center).Gap(12).FillWidth().Padding(8, 10).Radius(10).
		Background(t.Background).Border(1, t.Border).Children(func() {
		// the left: who, and when what is left is lost
		ui.Column(c).Grow(1).Shrink(1).Gap(2).Children(func() {
			name := q.User
			if name == "" {
				name = q.Provider
			}
			ui.Text(c, name).FontSize(12.5).TextColor(t.Text).Grow(1).Shrink(1).SingleLine()
			if q.Error != "" {
				ui.Text(c, q.Error).FontSize(11).TextColor(t.TextMuted).Grow(1).Shrink(1).Wrap()
				return
			}
			if when := panelQuotaReset(q); when != "" {
				ui.Text(c, when).FontSize(11).TextColor(t.TextMuted).Grow(1).Shrink(1).SingleLine()
			}
			if until := panelQuotaUntil(q); until != "" {
				ui.Text(c, until).FontSize(11).TextColor(t.Warning).Grow(1).Shrink(1).SingleLine()
			}
		})

		// the right: the windows as rings
		rows := quotaRowsOf(q)
		if len(rows) > 0 {
			ui.Row(c).AlignItems(ui.Start).Gap(10).Shrink(0).Children(func() {
				for _, r := range rows {
					a.panelQuotaRing(c, t, r)
				}
			})
		}
	})
}

// panelQuotaRing is one window: its share as a ring, and what it is under
// it.
func (a *App) panelQuotaRing(c *ui.Context, t *ui.Theme, r quotaRow) {
	w := r.w
	col := ui.Column(c).AlignItems(ui.Center).Gap(3).Shrink(0).Tooltip(r.tool)
	col.Children(func() {
		quotaDial(c, t, float32(w.Used))
		family, span := quotaRingWords(w)
		if family != "" {
			ui.Text(c, family).FontSize(9.5).TextColor(t.TextMuted).
				Width(46).TextAlign(ui.Center).SingleLine()
		}
		ui.Text(c, span).FontSize(9.5).TextColor(t.TextMuted).
			Width(46).TextAlign(ui.Center).SingleLine()
	})
}

// panelQuotaReset is when the account's next window resets.
func panelQuotaReset(q provider.SubscriptionQuota) string {
	var soonest *time.Time
	for _, w := range q.Windows {
		if w.ResetsAt == nil {
			continue
		}
		if soonest == nil || w.ResetsAt.Before(*soonest) {
			soonest = w.ResetsAt
		}
	}
	if soonest == nil {
		return ""
	}
	return "重置于 " + usResetIn(*soonest)
}

// panelQuotaUntil is when what is left of the account is lost, which is
// not the same as when its windows reset.
func panelQuotaUntil(q provider.SubscriptionQuota) string {
	var until *time.Time
	for _, w := range q.Windows {
		if w.RunsOutAt == nil {
			continue
		}
		if until == nil || w.RunsOutAt.Before(*until) {
			until = w.RunsOutAt
		}
	}
	if until == nil {
		return ""
	}
	return "截至 " + until.Format("1月2日 15:04")
}

// panelUsageTab is the usage of the period, as the Usage page counts it:
// the totals, and the models that spent the most of them.
func (a *App) panelUsageTab(c *ui.Context, t *ui.Theme) {
	u := a.data.Usage.Overview
	if u.Calls == 0 && len(u.Models) == 0 {
		a.panelEmpty(c, t, "这段时间没有用量。", "网关记下的调用会在这里出现。")
		return
	}
	ui.Scroll(c).Grow(1).Shrink(1).FillWidth().Padding(0, 12, 8, 12).Children(func() {
		ui.Column(c).FillWidth().Gap(8).Children(func() {
			card := ui.Column(c).FillWidth().Radius(10).
				Background(t.Background).Border(1, t.Border).Padding(10, 12).Gap(6)
			card.Children(func() {
				ui.Row(c).AlignItems(ui.Center).Children(func() {
					ui.Text(c, fmt.Sprintf("%s 次调用", FormatCount(u.Calls))).
						FontSize(13).FontWeight(600).TextColor(t.Text).Shrink(0).SingleLine()
					ui.Spacer(c)
					if u.Errors > 0 {
						ui.Text(c, fmt.Sprintf("%s 次出错", FormatCount(u.Errors))).
							FontSize(11.5).TextColor(driftColor(t)).Shrink(0).SingleLine()
					}
				})
				ui.Text(c, fmt.Sprintf("输入 %s · 输出 %s · $%.2f",
					FormatTokens(u.Input), FormatTokens(u.Output), u.Cost)).
					FontSize(11).TextColor(t.TextMuted).SingleLine()
			})

			// the models that spent the most
			if len(u.Models) > 0 {
				card := ui.Column(c).FillWidth().Radius(10).
					Background(t.Background).Border(1, t.Border).Padding(4, 0)
				card.Children(func() {
					for i, g := range u.Models {
						if i >= 6 {
							break
						}
						a.panelUsageModel(c, t, g, i == 0)
					}
				})
			}
		})
	})
}

// panelUsageModel is one model in the usage tab: its name, its share of
// the period's calls as a ring, and how many calls it was.
func (a *App) panelUsageModel(c *ui.Context, t *ui.Theme, g usageapi.Group, first bool) {
	row := ui.Row(c).AlignItems(ui.Center).Gap(8).Height(30).Padding(0, 12).FillWidth()
	if !first {
		row.BorderWidth(0.5, 0, 0, 0).BorderColor(a.panelLine(t))
	}
	share := 0.0
	if total := a.data.Usage.Overview.Calls; total > 0 {
		share = float64(g.Calls) / float64(total)
	}
	row.Children(func() {
		ui.Text(c, g.Name).FontSize(12).TextColor(t.Text).Grow(1).Shrink(1).SingleLine()
		shareRing(c, t, float32(share*100), 24, t.Text)
		ui.Text(c, FormatCount(g.Calls)).FontSize(11.5).TextColor(t.TextMuted).Shrink(0).SingleLine()
	})
}

// panelRoutingTab is the routing groups, as the Routing page lists them:
// each group's name, what it picks, and how many members it has.
func (a *App) panelRoutingTab(c *ui.Context, t *ui.Theme) {
	groups := a.data.Routing.Groups
	if len(groups) == 0 {
		a.panelEmpty(c, t, "还没有路由组。", "在路由页建一个，Agent 才能自己挑模型。")
		return
	}
	ui.Scroll(c).Grow(1).Shrink(1).FillWidth().Padding(0, 12, 8, 12).Children(func() {
		ui.Column(c).FillWidth().Gap(8).Children(func() {
			totals := a.data.Routing.Totals
			if totals.Requests > 0 {
				card := ui.Column(c).FillWidth().Radius(10).
					Background(t.Background).Border(1, t.Border).Padding(10, 12).Gap(4)
				card.Children(func() {
					ui.Text(c, fmt.Sprintf("%s 次请求", FormatCount(totals.Requests))).
						FontSize(13).FontWeight(600).TextColor(t.Text).SingleLine()
					note := ""
					if totals.Rerouted > 0 {
						note = fmt.Sprintf("改道 %d 次", totals.Rerouted)
					}
					if totals.Errors > 0 {
						if note != "" {
							note += " · "
						}
						note += fmt.Sprintf("%d 次出错", totals.Errors)
					}
					if note != "" {
						ui.Text(c, note).FontSize(11).TextColor(t.TextMuted).SingleLine()
					}
				})
			}

			card := ui.Column(c).FillWidth().Radius(10).
				Background(t.Background).Border(1, t.Border).Padding(4, 0)
			card.Children(func() {
				for i, g := range groups {
					a.panelRoutingGroup(c, t, g, i == 0)
				}
			})
		})
	})
}

// panelRoutingGroup is one routing group in the panel: its name, what it
// picks, and how many models are in it.
func (a *App) panelRoutingGroup(c *ui.Context, t *ui.Theme, g RoutingGroupRow, first bool) {
	row := ui.Row(c).AlignItems(ui.Center).Gap(8).Height(30).Padding(0, 12).FillWidth()
	if !first {
		row.BorderWidth(0.5, 0, 0, 0).BorderColor(a.panelLine(t))
	}
	name := g.Name
	if name == "" {
		name = g.ID
	}
	row.Children(func() {
		ui.Icon(c, routingFanIcon()).Size(13, 13).TextColor(t.TextMuted).Shrink(0)
		ui.Text(c, name).FontSize(12).TextColor(t.Text).Grow(1).Shrink(1).SingleLine()
		if g.Disabled {
			ui.Text(c, "已停用").FontSize(11).TextColor(t.TextMuted).Shrink(0).SingleLine()
		} else {
			ui.Text(c, fmt.Sprintf("%d 个模型", len(g.Members))).
				FontSize(11.5).TextColor(t.TextMuted).Shrink(0).SingleLine()
		}
	})
}

// panelEmpty is a tab with nothing to show: the words, and what does it.
func (a *App) panelEmpty(c *ui.Context, t *ui.Theme, title, note string) {
	ui.Column(c).Grow(1).Shrink(1).FillWidth().AlignItems(ui.Center).Gap(4).Padding(24, 20).Children(func() {
		ui.Text(c, title).FontSize(12.5).FontWeight(600).TextColor(t.Text).SingleLine()
		ui.Text(c, note).FontSize(11).TextColor(t.TextMuted).FillWidth().TextAlign(ui.Center).LineHeight(1.4)
	})
}

// panelFooter is the bar under it all: the profiles' count, then the two
// ends — magpie's foot.
func (a *App) panelFooter(c *ui.Context, t *ui.Theme) {
	a.skSheetRule(c, t)
	ui.Row(c).AlignItems(ui.Center).Gap(10).Padding(8, 14).FillWidth().Children(func() {
		profiles := a.panelProfilesWord()
		if profiles != "" {
			pf := ui.Text(c, profiles).FontSize(11).TextColor(t.TextMuted).
				Cursor(ui.CursorPointer).Shrink(0).SingleLine()
			if pf.Clicked() {
				a.closePanel()
				a.Show(PageAgents)
			}
		}
		ui.Spacer(c)
		// the two ends, as magpie's foot has them: words, not buttons
		openBtn := ui.Row(c).AlignItems(ui.Center).Padding(3, 8).Radius(5).
			Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).
			Label("打开 Echo")
		if openBtn.Hovered() {
			openBtn.Background(a.hoverBg(t))
		}
		if openBtn.Clicked() {
			a.closePanel()
			a.Show("")
		}
		openBtn.Children(func() {
			ui.Text(c, "打开 Echo").FontSize(11.5).TextColor(t.Text).SingleLine()
		})
		quitBtn := ui.Row(c).AlignItems(ui.Center).Padding(3, 8).Radius(5).
			Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).
			Label("退出")
		if quitBtn.Hovered() {
			quitBtn.Background(a.hoverBg(t))
		}
		if quitBtn.Clicked() {
			a.Quit()
		}
		quitBtn.Children(func() {
			ui.Text(c, "退出").FontSize(11.5).TextColor(t.Text).SingleLine()
		})
	})
}

// panelProfilesWord is the foot's word for the saved profiles (方案), or
// nothing when there are none.
func (a *App) panelProfilesWord() string {
	if n := len(a.data.AgentProfiles); n > 0 {
		return fmt.Sprintf("方案 %d", n)
	}
	return ""
}

// panelLine is the hairline between the panel's rows: magpie's --line, the
// theme's own border, a shade lighter.
func (a *App) panelLine(t *ui.Theme) ui.Color {
	if t.Dark {
		return ui.RGBA(255, 255, 255, 0.08)
	}
	return ui.RGBA(0, 0, 0, 0.07)
}
