//go:build nogui && echo

package shell

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/yetone/magpie/internal/gateway"
	"github.com/yetone/magpie/internal/plugin"
	"github.com/yetone/magpie/internal/provider"
	"github.com/yetone/magpie/internal/update"
)

// Plugins › Installed: what was added from the market, a row each, as
// magpie's page draws them (plugins.js's installedRow): its name and version
// (a git plugin's commit), a badge when an update is out, what it is, the
// subscriptions it signs in to, how its gateway middleware is doing, and the
// ways to sign in, update, switch off or remove it. Under it, what the
// plugins run on and the ways to look for updates.

// pluginsInstalled is the tab's body.
func (a *App) pluginsInstalled(c *ui.Context, t *ui.Theme) {
	es := a.plugEntriesSorted()
	if len(es) == 0 {
		a.plugNone(c, t)
		return
	}
	outdated := a.plugOutdated(es)
	ui.Column(c).FillWidth().Gap(10).Children(func() {
		// what the sidebar's dot is about, said where it is acted on:
		// which plugins have a newer version, and one press to update them
		// all (magpie's pm-updates bar)
		if len(outdated) > 0 {
			a.plugUpdatesBar(c, t, outdated)
		}
		// by name, A→Z or Z→A, as magpie's row-head sort picks (sort.js)
		if len(es) > 1 {
			ui.Row(c).AlignItems(ui.Center).FillWidth().Children(func() {
				ui.Spacer(c)
				if seg := ui.Segmented(c, &a.plugSort, "A→Z", "Z→A").Shrink(0); seg.Changed() {
					c.Invalidate()
				}
			})
		}
		list := ui.Column(c).FillWidth().Border(1, plugLineC(t)).Radius(12).
			Background(plugCardBg(t))
		list.Children(func() {
			for i, e := range es {
				a.plugEntryRow(c, t, e, i > 0)
			}
		})
		a.plugFoot(c, t, es)
	})
}

// plugOutdated is the installed plugins with a newer version out: npm's, a
// git one's repository at a commit beyond the one installed, or what a check
// found just now.
func (a *App) plugOutdated(es []PluginEntryRow) []PluginEntryRow {
	out := make([]PluginEntryRow, 0, len(es))
	for _, e := range es {
		if e.Off {
			continue
		}
		latest := a.plugLatestFor(e)
		head := a.plugHeadFor(e)
		if (latest != "" && e.Version != "" && update.Newer(latest, e.Version)) ||
			(e.Commit != "" && head != "" && head != e.Commit) {
			out = append(out, e)
		}
	}
	return out
}

// plugUpdatesBar is the bar above Installed: which plugins have a newer
// version, and Update all.
func (a *App) plugUpdatesBar(c *ui.Context, t *ui.Theme, out []PluginEntryRow) {
	p := &a.text.S.Plug
	names := make([]string, 0, len(out))
	for _, e := range out {
		names = append(names, a.plugShownName(e))
	}
	line := plugText(p.UpdateOne, "name", strings.Join(names, ", "))
	if len(out) > 1 {
		line = plugText(p.UpdateMany, "n", plugNum(len(out)), "names", strings.Join(names, ", "))
	}
	bg := ui.RGBA(0, 0, 0, 0.04)
	if t.Dark {
		bg = ui.RGBA(255, 255, 255, 0.06)
	}
	ui.Row(c).AlignItems(ui.Center).Gap(9).FillWidth().Padding(8, 12).
		Radius(10).Background(bg).Children(func() {
		ui.Box(c).Width(7).Height(7).Radius(4).Background(t.Accent).Shrink(0)
		ui.Text(c, line).FontSize(12).TextColor(t.Text).Grow(1).Shrink(1).
			MaxLines(2).Wrap()
		label := p.UpdateAll
		if a.plugBusy == "update" {
			label = p.Updating
		} else {
			label = plugText(p.UpdateAll, "n", plugNum(len(out)))
		}
		if b := a.plugAct(c, t, label, "up", a.plugBusy == "" && !a.plugChecking); b.Clicked() {
			a.plugUpdateAll()
		}
	})
}

// plugNone is what the tab says with nothing installed.
func (a *App) plugNone(c *ui.Context, t *ui.Theme) {
	p := &a.text.S.Plug
	ui.Row(c).Center().FillWidth().Padding(40, 16).Children(func() {
		ui.Column(c).AlignItems(ui.Center).Gap(8).Children(func() {
			a.plugTile(c, t, "", "Echo", "", true)
			ui.Text(c, p.NoPlugins).FontSize(13.5).FontWeight(600).TextColor(t.Text)
			ui.Text(c, p.NoPluginsHint).FontSize(12).TextColor(t.TextMuted).
				Wrap().MaxWidth(340)
			if b := a.plugAct(c, t, p.DiscoverPlugins, "get", true); b.Clicked() {
				a.plugTab = PlugTabDiscover
				a.plugMarketKick()
				c.Invalidate()
			}
		})
	})
}

// plugEntriesSorted is the installed list in the order the reader picked.
func (a *App) plugEntriesSorted() []PluginEntryRow {
	es := make([]PluginEntryRow, len(a.data.Plugins.Installed))
	copy(es, a.data.Plugins.Installed)
	sort.SliceStable(es, func(i, j int) bool {
		x, y := strings.ToLower(a.plugShownName(es[i])), strings.ToLower(a.plugShownName(es[j]))
		if a.plugSort == 1 {
			return x > y
		}
		return x < y
	})
	return es
}

// plugEntryRow is one installed plugin's row.
func (a *App) plugEntryRow(c *ui.Context, t *ui.Theme, e PluginEntryRow, divider bool) {
	p := &a.text.S.Plug
	name := a.plugShownName(e)
	latest := a.plugLatestFor(e)
	head := a.plugHeadFor(e)
	ck, checked := a.plugCheckFor(e.Spec)
	gitBehind := e.Commit != "" && head != "" && head != e.Commit
	out := !e.Off && ((latest != "" && e.Version != "" && update.Newer(latest, e.Version)) || gitBehind)
	asking := a.plugAskRemove == e.Spec || a.plugAskOff == e.Spec || a.plugAskMove == e.Spec

	row := ui.Row(c).AlignItems(ui.Start).FillWidth().MinHeight(54).
		Padding(8, 10, 8, 12).Gap(11).
		Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton)
	if divider {
		row.BorderWidth(1, 0, 0, 0).BorderColor(plugLineC(t))
	}
	if row.Clicked() && !asking {
		a.plugOpenDetail(e.Spec, name, e)
	}

	row.Children(func() {
		ic := a.plugIconForEntry(e)
		if e.Off {
			ui.Row(c).Opacity(0.5).Shrink(0).Children(func() {
				a.plugTile(c, t, ic, name, e.Kind(), false)
			})
		} else {
			a.plugTile(c, t, ic, name, e.Kind(), false)
		}

		who := ui.Column(c).Grow(1).Shrink(1).Gap(2)
		if e.Off {
			who.Opacity(0.5)
		}
		who.Children(func() {
			a.plugEntryNameLine(c, t, e, name, latest, gitBehind, ck, checked)
			a.plugEntrySub(c, t, e, name, asking)
			if !asking {
				// a subscription another installed plugin signs in to as
				// well: one of them serves it, and the row says which
				for _, cl := range a.plugClashes[e.Spec] {
					a.plugClashLine(c, t, e, cl)
				}
			}
			if !asking && !e.Off {
				// the built-in subscriptions moved onto it, as its own lines
				if e.Middleware != nil && e.Middleware.Error != "" {
					ui.Text(c, plugText(p.MwNotLoaded, "error", e.Middleware.Error)).
						FontSize(11.5).TextColor(t.Danger).Wrap()
				} else if e.Middleware != nil {
					a.plugMwLine(c, t, e)
				}
				if e.Gateway != nil {
					a.plugGatewayLine(c, t, e)
				}
				if e.IsAgent {
					a.plugAgentLine(c, t, e)
				}
				if len(e.Tools) > 0 {
					a.plugToolsLine(c, t, e)
				}
				if e.HasPage {
					a.plugPageLine(c, t, e)
				}
				if len(e.Panels) > 0 {
					a.plugPanelsLine(c, t, e)
				}
				if len(e.Actions) > 0 {
					a.plugActionsLine(c, t, e)
				}
				if (e.Middleware != nil || e.IsGateway) && a.plugOptions == e.Spec {
					a.plugOptionsEditor(c, t, e)
				}
			}
		})

		a.plugEntryButtons(c, t, e, name, latest, out, asking)
	})
}

// plugEntryNameLine is a row's first line: the name, its version or commit,
// and the badge for what a check found. Each value says over itself what it
// is (magpie's own titles).
func (a *App) plugEntryNameLine(c *ui.Context, t *ui.Theme, e PluginEntryRow, name, latest string, gitBehind bool, ck plugin.VersionCheck, checked bool) {
	p := &a.text.S.Plug
	ui.Row(c).AlignItems(ui.Center).Gap(6).Wrap().Children(func() {
		ui.Text(c, name).FontSize(13).FontWeight(500).TextColor(t.Text).
			Shrink(0).SingleLine()
		if e.Version != "" {
			ver := ui.Text(c, "v"+e.Version).Font("monospace").FontSize(10.5).
				TextColor(t.TextMuted).Shrink(0)
			if latest != "" && latest != e.Version {
				ver.Tooltip(plugText(p.VersionTip, "have", e.Version, "v", latest))
			} else if latest != "" {
				ver.Tooltip(plugText(p.LatestTip, "v", latest))
			}
		}
		if plugin.IsGit(e.Spec) && e.Commit != "" {
			ui.Text(c, e.Commit).Font("monospace").FontSize(10.5).
				TextColor(t.TextMuted).Shrink(0).
				Tooltip(plugText(p.GitCommitTip, "repo", e.Spec))
		}
		switch {
		case gitBehind:
			a.plugChipTip(c, t, p.NewCommits, "up",
				plugText(p.GitHaveTip, "have", e.Commit, "repo", e.Spec, "head", latest))
		case latest != "" && e.Version != "" && update.Newer(latest, e.Version):
			a.plugChipTip(c, t, plugText(p.OutVersion, "v", latest), "up",
				plugText(p.VersionTip, "have", e.Version, "v", latest))
		case checked && ck.Status == "unknown":
			// asked just now, and it didn't say: the chip says why over
			// itself
			a.plugChipTip(c, t, p.CouldntCheck, "warn", a.plugWhyText(ck))
		case e.AutoUpdated != nil && e.AutoUpdated.To == e.Version:
			// what it did, and when, of the reader's own hands-off update
			a.plugChipTip(c, t, p.AutoUpdated, "",
				plugText(p.AutoTip, "from", e.AutoUpdated.From, "to", e.AutoUpdated.To,
					"date", plugShortDate(a.lang, e.AutoUpdated.At)))
		case (ck.Status == "current" || ck.Status == "update") && e.Version != "":
			tip := plugText(p.LatestTip, "v", e.Version)
			if plugin.IsGit(e.Spec) {
				head := ck.Head
				if head == "" {
					head = e.Commit
				}
				tip = plugText(p.GitHeadTip, "head", head, "repo", e.Spec)
			}
			if !a.plugCheckedAt.IsZero() {
				tip += "\n" + plugText(p.CheckedTip, "time", plugAt(a.lang, a.plugCheckedAt))
			}
			a.plugChipTip(c, t, p.UpToDate, "", tip)
		}
		// the built-in subscriptions moved onto it: taking it away or off
		// stops them, as there is no built-in to go back to, so the row
		// says it carries them
		if moved := a.plugMovedNames(e); len(moved) > 0 && !e.Off {
			a.plugChipTip(c, t, plugText(p.Serves, "names", strings.Join(moved, ", ")),
				"moved", plugText(p.MovedTip, "names", strings.Join(moved, ", ")))
		}
		// what it is
		if e.Middleware != nil || e.IsMiddleware {
			a.plugKindChip(c, t, "middleware")
		}
		if e.IsGateway {
			a.plugKindChip(c, t, "gateway")
		}
		if e.IsAgent {
			a.plugKindChip(c, t, "agent")
		}
		toolsOnly := len(e.Tools) > 0 && len(e.Providers) == 0
		if len(e.Tools) > 0 {
			a.plugKindChip(c, t, "tools")
		}
		if e.HasPage {
			a.plugKindChip(c, t, "page")
		}
		if len(e.Panels) > 0 {
			a.plugKindChip(c, t, "pn")
		}
		if len(e.Providers) > 0 || !(e.GatewayOnly || e.MiddlewareOnly || e.InMagpieOnly || toolsOnly) {
			a.plugKindChip(c, t, "")
		}
	})
}

// plugEntrySub is a row's second line: off, why it didn't load, the
// subscriptions it signs in to, or what it signs in to nothing of.
func (a *App) plugEntrySub(c *ui.Context, t *ui.Theme, e PluginEntryRow, name string, asking bool) {
	p := &a.text.S.Plug
	switch {
	case asking:
		// the ask is said in the row, not a dialog, so the reader sees
		// which subscriptions it is about (magpie's asking)
		text := plugText(p.AskOff, "names", strings.Join(a.plugMovedNames(e), ", "))
		switch {
		case a.plugAskRemove == e.Spec:
			text = plugText(p.AskRemove, "names", strings.Join(a.plugMovedNames(e), ", "))
		case a.plugAskMove == e.Spec:
			text = plugText(p.RunAgain, "names", a.plugMoverNames(e.Spec))
		}
		ui.Text(c, text).FontSize(11.5).TextColor(t.Text).Wrap()
	case e.Off:
		ui.Text(c, p.Off).FontSize(11.5).TextColor(t.TextMuted)
	case e.Error != "":
		ui.Text(c, plugText(p.DidntLoad, "error", e.Error)).FontSize(11.5).
			TextColor(t.Danger).Wrap()
	case len(e.Providers) > 0:
		cands := a.plugMovers(e.Spec)
		some := false
		for _, s := range e.Providers {
			if s.SignedIn {
				some = true
			}
		}
		ui.Row(c).AlignItems(ui.Center).Gap(2).Wrap().Children(func() {
			for i, s := range e.Providers {
				if i > 0 {
					ui.Text(c, "·").FontSize(11.5).TextColor(t.TextMuted).Shrink(0)
				}
				on := s.SignedIn
				builtin := !s.SignedIn && plugHasMover(cands, s.Pid)
				col := t.TextMuted
				if on {
					col = t.Success
				}
				label := plugText(p.SubNotSignedIn, "name", s.Name)
				if on {
					label = plugText(p.SubSignedIn, "name", s.Name)
				} else if builtin {
					label = plugText(p.SubBuiltin, "name", s.Name)
				}
				chip := ui.Row(c).AlignItems(ui.Center).Gap(5).Shrink(0)
				// whose it is when it isn't signed in here: its built-in
				// runs it with the reader's accounts, or it belongs to
				// another of the plugin's own subscriptions
				if builtin {
					chip.Tooltip(plugText(p.SubBuiltinTip, "name", s.Name))
				} else if !s.SignedIn && some {
					other := ""
					for _, y := range e.Providers {
						if y.SignedIn {
							other = y.Name
							break
						}
					}
					chip.Tooltip(plugText(p.SubOwn, "name", s.Name, "other", other))
				}
				chip.Children(func() {
					ui.Box(c).Width(6).Height(6).Radius(3).Background(col).Shrink(0)
					ui.Text(c, label).FontSize(11.5).TextColor(t.TextMuted).SingleLine()
				})
			}
		})
	case (e.GatewayOnly || e.MiddlewareOnly || e.InMagpieOnly || (len(e.Tools) > 0 && len(e.Providers) == 0) || e.HasPage) && len(e.Providers) == 0:
		// only middleware, a request hook, an agent, tools or a page: its
		// own line says what it does
	case len(a.plugClashes[e.Spec]) > 0 && len(e.Providers) == 0:
		// the clash lines below say whose it is
	case e.Named:
		ui.Text(c, p.SignsInNothing).FontSize(11.5).TextColor(t.TextMuted).Wrap()
	}
}

// plugClashLine is a subscription two plugins sign in to: the one that
// serves it says so, and the other says which plugin has it (magpie's
// clashLine).
func (a *App) plugClashLine(c *ui.Context, t *ui.Theme, e PluginEntryRow, cl PluginClashRow) {
	p := &a.text.S.Plug
	serves := cl.By == e.Spec
	text, tip := p.ClashServed, p.ClashServedTip
	col := t.Warning
	if serves {
		text, tip = p.ClashServes, p.ClashServesTip
		col = t.TextMuted
	}
	ui.Row(c).AlignItems(ui.Center).Gap(5).Tooltip(plugText(tip, "name", cl.Name)).
		Children(func() {
			ui.Box(c).Width(6).Height(6).Radius(3).Background(col).Shrink(0)
			ui.Text(c, plugText(text, "name", cl.Name, "other", cl.Others)).
				FontSize(11.5).TextColor(col).Wrap()
		})
}

// plugMoverNames are the built-in subscriptions this plugin could carry, as
// the ask after switching it on names them.
func (a *App) plugMoverNames(spec string) string {
	cands := a.plugMovers(spec)
	names := make([]string, 0, len(cands))
	for _, c := range cands {
		names = append(names, c.Name)
	}
	return strings.Join(names, ", ")
}

// plugMwLine is a middleware's line: its hooks, how many calls and how long
// each took, and the calls that failed or were dropped (magpie's mwLine).
func (a *App) plugMwLine(c *ui.Context, t *ui.Theme, e PluginEntryRow) {
	p := &a.text.S.Plug
	m := e.Middleware
	if m == nil {
		return
	}
	var parts []string
	if len(m.Hooks) > 0 {
		parts = append(parts, plugText(p.MwHooks, "hooks", strings.Join(m.Hooks, ", ")))
	}
	if m.Calls > 0 {
		parts = append(parts, plugText(p.MwCalls, "n", fmt.Sprint(m.Calls), "us", fmt.Sprintf("%.0f", m.AvgMicro)))
	}
	if len(m.Strategies) > 0 {
		parts = append(parts, plugText(p.MwRoutings, "routings", strings.Join(m.Strategies, ", ")))
		if m.Orders > 0 {
			parts = append(parts, plugText(p.MwAsked, "n", fmt.Sprint(m.Orders)))
		}
	}
	if m.Usage {
		parts = append(parts, plugText(p.MwUsage, "n", fmt.Sprint(m.Rows)))
	}
	if len(parts) == 0 && m.Failures == 0 && m.Dropped == 0 {
		return
	}
	ui.Row(c).AlignItems(ui.Center).Gap(4).Wrap().Children(func() {
		if len(parts) > 0 {
			txt := ui.Text(c, strings.Join(parts, " · ")).FontSize(11.5).
				TextColor(t.TextMuted).Wrap()
			// what it is handed is narrower than what the gateway has
			if len(m.Events) > 0 {
				txt.Tooltip(plugText(p.MwEventsTip, "events", strings.Join(m.Events, ", ")))
			}
		}
		if m.Failures > 0 {
			// a failed call leaves the request as it was; the last one says
			// why, when it is known
			tip := p.MwFailTip
			if m.Last != "" {
				tip += "\n" + plugText(p.MwLastTip, "error", m.Last)
			}
			ui.Text(c, plugText(p.MwFailed, "n", fmt.Sprint(m.Failures))).
				FontSize(11.5).TextColor(t.Warning).Tooltip(tip).Shrink(0)
		}
		if m.Dropped > 0 {
			ui.Text(c, plugText(p.MwDropped, "n", fmt.Sprint(m.Dropped))).
				FontSize(11.5).TextColor(t.Warning).Tooltip(p.MwDroppedTip).Shrink(0)
		}
	})
}

// plugGatewayLine is a request hook's line: what it last did, how many calls
// and how long each took, how much it injected, and the calls that failed
// (magpie's gatewayLine). Its diagnostics last one plugin host's lifetime.
func (a *App) plugGatewayLine(c *ui.Context, t *ui.Theme, e PluginEntryRow) {
	p := &a.text.S.Plug
	g := e.Gateway
	if g == nil {
		return
	}
	text := plugText(p.GwLine) + ": " + plugGwOutcome(p, g.LastOutcome)
	if g.Calls > 0 {
		text += " · " + plugText(p.GwCalls, "n", fmt.Sprint(g.Calls), "ms", fmt.Sprintf("%.1f", g.AvgMillis))
	}
	if g.LastBytes > 0 {
		text += " · " + plugText(p.GwBytes, "n", fmt.Sprint(g.LastBytes))
	}
	ui.Row(c).AlignItems(ui.Center).Gap(4).Wrap().Children(func() {
		ui.Text(c, text).FontSize(11.5).TextColor(t.TextMuted).Wrap().Tooltip(p.GwTip)
		if g.Failures > 0 {
			ui.Text(c, plugText(p.MwFailed, "n", fmt.Sprint(g.Failures))).
				FontSize(11.5).TextColor(t.Warning).Shrink(0).Tooltip(p.GwTip)
		}
	})
}

// plugGwOutcome is a request hook's last outcome in the page's own words
// (magpie's own labels; anything unknown is the waiting state).
func plugGwOutcome(p *PluginStrings, code string) string {
	switch code {
	case "ready":
		return p.GwReady
	case "empty":
		return p.GwEmpty
	case "unbound":
		return p.GwUnbound
	case "unauthorized":
		return p.GwUnauthorized
	case "unavailable":
		return p.GwUnavailable
	case "invalid":
		return p.GwInvalid
	case "skipped":
		return p.GwSkipped
	case "cancelled":
		return p.GwCancelled
	case "timeout":
		return p.GwTimeout
	case "error":
		return p.GwFailed
	}
	return p.GwIdle
}

// plugAgentLine is the agent a plugin adds, as its row says it.
func (a *App) plugAgentLine(c *ui.Context, t *ui.Theme, e PluginEntryRow) {
	p := &a.text.S.Plug
	if e.AgentErr != "" {
		ui.Text(c, plugText(p.AgentFailed, "error", e.AgentErr)).FontSize(11.5).
			TextColor(t.Danger).Wrap()
		return
	}
	if e.AgentName == "" {
		return
	}
	ui.Text(c, plugText(p.AgentAdds, "name", e.AgentName)).FontSize(11.5).
		TextColor(t.TextMuted).Wrap()
}

// plugToolsLine is the tools a plugin gives, which Echo's plugin tools MCP
// server offers every agent: each by the name an agent sees, what it does
// over the line itself, and the server's address to copy into an agent
// (magpie's toolsLine).
func (a *App) plugToolsLine(c *ui.Context, t *ui.Theme, e PluginEntryRow) {
	p := &a.text.S.Plug
	names := make([]string, 0, len(e.Tools))
	said := make([]string, 0, len(e.Tools))
	for _, tl := range e.Tools {
		names = append(names, tl.Name)
		if tl.Description != "" {
			said = append(said, tl.Name+" — "+tl.Description)
		} else {
			said = append(said, tl.Name)
		}
	}
	url := a.plugMCPAddress()
	ui.Row(c).AlignItems(ui.Center).Gap(6).FillWidth().Children(func() {
		ui.Text(c, plugText(p.ToolsLine, "names", strings.Join(names, ", "))).
			FontSize(11.5).TextColor(t.TextMuted).Grow(1).Shrink(1).Wrap().
			Tooltip(strings.Join(said, "\n"))
		// the address an agent is pointed at, fixed at the row's right
		b := a.plugTextButton(c, t, p.MCPAddress, true, false)
		b.Tooltip(plugText(p.MCPCopyTip, "url", url))
		if b.Clicked() {
			a.copyToClipboard(url)
		}
	})
}

// plugMoveMine is whose accounts a Move would bring, in magpie's own words:
// one account, or several.
func (a *App) plugMoveMine(c provider.MoveCandidate) string {
	p := &a.text.S.Plug
	word := p.MoveMineMany
	if c.Accounts == 1 {
		word = p.MoveMineOne
	}
	return plugText(word, "name", c.Name, "n", plugNum(c.Accounts))
}

// plugMCPAddress is where the plugin tools MCP server answers: the
// gateway's own address and its tools path, as magpie's page takes both.
func (a *App) plugMCPAddress() string {
	return strings.TrimRight(a.data.Gateway.URL, "/") + gateway.ToolsMCPPath
}

// plugPageLine is what a plugin's own page may read of Echo's, or the
// built-in page whose place it takes.
func (a *App) plugPageLine(c *ui.Context, t *ui.Theme, e PluginEntryRow) {
	p := &a.text.S.Plug
	var line string
	switch {
	case e.PageReplaces != "":
		// its page takes a built-in page's place: said with that page's
		// own name, in the window's language
		page := a.plugPageName(e.PageReplaces)
		if len(e.PageReads) > 0 {
			line = plugText(p.PageTakesPlace, "page", page, "reads", a.plugReadsText(e.PageReads))
		} else {
			line = plugText(p.PageTakesPlaceNothing, "page", page)
		}
	case len(e.PageReads) > 0:
		line = plugText(p.PageReads, "reads", a.plugReadsText(e.PageReads))
	default:
		line = p.PageReadsNothing
	}
	ui.Text(c, line).FontSize(11.5).TextColor(t.TextMuted).Wrap()
}

// plugPanelsLine is the panels a plugin adds to Echo's own pages: where
// they go, and what they may read.
func (a *App) plugPanelsLine(c *ui.Context, t *ui.Theme, e PluginEntryRow) {
	p := &a.text.S.Plug
	slots, seen := make([]string, 0, len(e.Panels)), map[string]bool{}
	for _, s := range e.Panels {
		if seen[s] {
			continue
		}
		seen[s] = true
		slots = append(slots, a.plugSlotName(s))
	}
	where := strings.Join(slots, ", ")
	line := plugText(p.PanelsNothing, "where", where)
	if len(e.PanelReads) > 0 {
		line = plugText(p.PanelsAdds, "where", where, "reads", a.plugReadsText(e.PanelReads))
	}
	txt := ui.Text(c, line).FontSize(11.5).TextColor(t.TextMuted).Wrap()
	if e.PanelTitles != "" {
		txt.Tooltip(e.PanelTitles)
	}
}

// plugActionsLine is the items a plugin's page adds to Echo's own menus.
func (a *App) plugActionsLine(c *ui.Context, t *ui.Theme, e PluginEntryRow) {
	p := &a.text.S.Plug
	menus, seen := make([]string, 0, len(e.Actions)), map[string]bool{}
	for _, m := range e.Actions {
		if seen[m] {
			continue
		}
		seen[m] = true
		menus = append(menus, a.plugMenuName(m))
	}
	txt := ui.Text(c, plugText(p.ActionsAdds, "where", strings.Join(menus, ", "))).
		FontSize(11.5).TextColor(t.TextMuted).Wrap()
	if e.ActionTitles != "" {
		txt.Tooltip(e.ActionTitles)
	}
}

// plugReadsText is what a page or panels may read, in the page's own words.
func (a *App) plugReadsText(reads []string) string {
	p := &a.text.S.Plug
	out := make([]string, 0, len(reads))
	for _, r := range reads {
		switch r {
		case "usage:summary":
			out = append(out, p.ReadUsageSummary)
		case "providers:list":
			out = append(out, p.ReadProvidersList)
		default:
			out = append(out, r)
		}
	}
	return strings.Join(out, ", ")
}

// plugPageName is a built-in page a plugin's page can take the place of,
// in the window's language.
func (a *App) plugPageName(id string) string {
	s := &a.text.S
	switch id {
	case "sessions":
		return s.Sessions
	case "library":
		return s.Library
	case "routing":
		return s.Routing
	case "usage":
		return s.Usage
	}
	return id
}

// plugSlotName is where in Echo's own pages a plugin's panel goes, in the
// window's language.
func (a *App) plugSlotName(slot string) string {
	p := &a.text.S.Plug
	switch slot {
	case "agents":
		return p.SlotAgents
	case "usage":
		return p.SlotUsage
	case "settings":
		return p.SlotSettings
	case "provider":
		return p.SlotProvider
	}
	return slot
}

// plugMenuName is one of Echo's menus a plugin's action goes in, in the
// window's language.
func (a *App) plugMenuName(menu string) string {
	p := &a.text.S.Plug
	switch menu {
	case "provider":
		return p.MenuProviders
	case "agent":
		return p.MenuAgents
	}
	return menu
}

// plugOptionsEditor is a middleware's options as JSON, in its row: what it
// runs with, or what its package suggests when it has none.
func (a *App) plugOptionsEditor(c *ui.Context, t *ui.Theme, e PluginEntryRow) {
	p := &a.text.S.Plug
	ui.Column(c).FillWidth().Gap(6).Padding(0, 0, 4, 0).Children(func() {
		ta := ui.TextArea(c, &a.plugOptsText).Font("monospace").FontSize(12).
			FillWidth().Height(120)
		_ = ta
		if a.plugOptsErr != "" {
			ui.Text(c, a.plugOptsErr).FontSize(11.5).TextColor(t.Danger).Wrap()
		}
		ui.Row(c).AlignItems(ui.Center).Gap(6).FillWidth().Children(func() {
			note := p.OptionsNoneTip
			if e.Options != nil {
				note = p.OptionsApplied
			}
			ui.Text(c, note).FontSize(11.5).TextColor(t.TextMuted).
				Grow(1).Shrink(1).SingleLine()
			if e.Options != nil {
				b := a.plugTextButton(c, t, p.Clear, true, true)
				b.Tooltip(p.ClearTip)
				if b.Clicked() {
					a.plugClearOptions(e.Spec)
				}
			}
			if b := a.plugTextButton(c, t, p.Cancel, true, false); b.Clicked() {
				a.plugOptions, a.plugOptsErr = "", ""
				c.Invalidate()
			}
			if b := a.plugTextButton(c, t, p.Save, true, true); b.Clicked() {
				a.plugSaveOptionsText(e.Spec)
			}
		})
	})
}

// plugMovedNames are the built-in subscriptions moved onto a plugin, by
// their names.
func (a *App) plugMovedNames(e PluginEntryRow) []string {
	out := make([]string, 0, len(e.Moved))
	for _, id := range e.Moved {
		out = append(out, a.plugMoverName(id))
	}
	return out
}

// plugMoverName names a built-in subscription, in the language drawn.
func (a *App) plugMoverName(id string) string {
	for _, c := range a.data.Plugins.Movable {
		if c.ID == id {
			return c.Name
		}
	}
	if a.lang == LangZH {
		switch id {
		case "anthropic":
			return "Anthropic"
		}
	}
	return id
}

// plugMovers are the built-in subscriptions this plugin could run instead.
func (a *App) plugMovers(spec string) []provider.MoveCandidate {
	pkg := plugin.Name(spec)
	out := make([]provider.MoveCandidate, 0, 2)
	for _, c := range a.data.Plugins.Movable {
		if c.Package == pkg {
			out = append(out, c)
		}
	}
	return out
}

// plugHasMover is whether a built-in with that OpenCode id is among the
// candidates: its built-in runs it, with the accounts already signed in.
func plugHasMover(cands []provider.MoveCandidate, pid string) bool {
	for _, c := range cands {
		if c.ID == pid {
			return true
		}
	}
	return false
}

// plugEntryButtons are a row's ways: Update, Move here, Use for…, Sign in,
// Options, Switch off or on, and Remove — or the ask's yes and no, while one
// is open (magpie's updateButton, moveButton and the row's own buttons).
func (a *App) plugEntryButtons(c *ui.Context, t *ui.Theme, e PluginEntryRow, name, latest string, out, asking bool) {
	p := &a.text.S.Plug
	if asking {
		mv := a.plugAskMove == e.Spec
		rm := a.plugAskRemove == e.Spec
		ui.Row(c).AlignItems(ui.Center).Gap(4).Shrink(0).Children(func() {
			no := p.Cancel
			if mv {
				no = p.NotNow
			}
			if b := a.plugTextButton(c, t, no, true, false); b.Clicked() {
				a.plugAskRemove, a.plugAskOff, a.plugAskMove = "", "", ""
				c.Invalidate()
			}
			label := p.SwitchOff
			if mv {
				label = p.Move
			} else if rm {
				label = p.Remove
			}
			if b := a.plugTextButton(c, t, label, a.plugBusy == "", true); b.Clicked() {
				spec := e.Spec
				a.plugAskRemove, a.plugAskOff, a.plugAskMove = "", "", ""
				switch {
				case mv:
					if cands := a.plugMovers(spec); len(cands) > 0 {
						a.plugMove(cands[0].ID)
					}
				case rm:
					a.plugRemove(spec)
				default:
					a.plugSetOff(spec, true)
				}
			}
		})
		return
	}

	cands := a.plugMovers(e.Spec)
	unsigned := false
	var firstUnsigned PluginSubRow
	anySigned := false
	for _, s := range e.Providers {
		if s.SignedIn {
			anySigned = true
			continue
		}
		if !unsigned {
			firstUnsigned = s
		}
		unsigned = true
	}
	here := !e.Off && e.Error == "" && !anySigned && len(cands) > 0
	busy := a.plugBusy != ""

	ui.Row(c).AlignItems(ui.Center).Gap(6).Shrink(0).Children(func() {
		if out {
			label := p.Update
			if a.plugBusy == "upgrade:"+e.Spec {
				label = p.Updating
			}
			b := a.plugAct(c, t, label, "up", !busy)
			// what the press fetches: npm's newer version, or the
			// repository's newest commit again
			switch {
			case plugin.IsGit(e.Spec):
				b.Tooltip(plugText(p.GitFetchTip, "repo", e.Spec) + "\n" +
					plugText(p.GitHaveTip, "have", e.Commit, "repo", e.Spec, "head", latest))
			case latest != "":
				tip := plugText(p.UpdateTo, "v", "v"+latest)
				// the community's own updates by itself, hourly
				if plugCommunityAuto(e.Spec) {
					tip += "\n" + p.AutoHourTip
				}
				b.Tooltip(tip)
			}
			if b.Clicked() {
				a.plugUpgrade(e.Spec, latest)
			}
		}
		if here && !busy {
			label := plugText(p.MoveHere, "name", cands[0].Name)
			if a.plugBusy == "move:"+cands[0].ID {
				label = p.Moving
			}
			b := a.plugTextButton(c, t, label, true, true)
			b.Tooltip(plugText(p.MoveTip, "name", cands[0].Name))
			if b.Clicked() {
				a.plugMove(cands[0].ID)
			}
		}
		// a subscription another installed plugin signs in to as well: this
		// one can serve it instead, both staying installed (magpie's
		// clashes)
		for _, cl := range a.plugClashes[e.Spec] {
			if cl.By == e.Spec {
				continue
			}
			label := plugText(p.UseFor, "name", cl.Name)
			if a.plugBusy == "prefer:"+e.Spec {
				label = p.Switching
			}
			b := a.plugTextButton(c, t, label, !busy, false)
			b.Tooltip(plugText(p.PreferTip, "name", cl.Name, "other", cl.Others))
			if b.Clicked() {
				a.plugPrefer(e.Spec, cl.ID, cl.Name)
			}
		}
		if unsigned && !e.Off && e.Error == "" {
			label := p.SignIn
			if anySigned {
				label = plugText(p.SignInTo, "name", firstUnsigned.Name)
			}
			if b := a.plugTextButton(c, t, label, !busy, !here && !anySigned); b.Clicked() {
				a.plugSignInOpen(firstUnsigned)
			}
		}
		if (e.IsGateway || (e.Middleware != nil && e.Middleware.Error == "")) && !e.Off && a.plugOptions != e.Spec {
			b := a.plugTextButton(c, t, p.Options, !busy, false)
			// what it runs with, or what its package suggests
			if e.Options != nil {
				b.Tooltip(p.OptionsTip)
			} else {
				b.Tooltip(p.OptionsNoneTip)
			}
			if b.Clicked() {
				a.plugOptionsOpen(e)
			}
		}
		onoff := p.SwitchOff
		if e.Off {
			onoff = p.SwitchOn
		}
		if b := a.plugTextButton(c, t, onoff, !busy, false); b.Clicked() {
			if !e.Off && len(e.Moved) > 0 {
				a.plugAskOff = e.Spec
				c.Invalidate()
			} else {
				a.plugSetOff(e.Spec, !e.Off)
			}
		}
		rb := a.plugTextButton(c, t, p.Remove, !busy, false)
		if moved := a.plugMovedNames(e); len(moved) > 0 {
			rb.Tooltip(plugText(p.RemoveMovedTip, "names", strings.Join(moved, ", ")))
		} else {
			rb.Tooltip(p.RemoveTip)
		}
		if rb.Clicked() {
			if len(e.Moved) > 0 {
				a.plugAskRemove = e.Spec
				c.Invalidate()
			} else {
				a.plugDetail = nil
				a.plugAskRemoveSheet(e.Spec)
			}
		}
	})
}

// plugCommunityAuto is whether Echo updates a plugin by itself: the
// community's own, unpinned (magpie's own check in its row).
func plugCommunityAuto(spec string) bool {
	const community = "@magpie-community/"
	if !strings.HasPrefix(spec, community) {
		return false
	}
	// a version pinned by hand is left to the reader
	rest := spec[len(community):]
	return !strings.Contains(rest, "@")
}

// plugFoot is what the plugins run on and the way to look for updates now,
// with when it was last looked (magpie's pm-foot).
func (a *App) plugFoot(c *ui.Context, t *ui.Theme, es []PluginEntryRow) {
	p := &a.text.S.Plug
	anyAsked := false
	for _, e := range es {
		if !plugin.IsPath(e.Spec) || plugin.IsGit(e.Spec) {
			anyAsked = true
		}
	}
	ui.Row(c).AlignItems(ui.Center).Gap(8).FillWidth().Padding(0, 4).Children(func() {
		line := ""
		if a.data.Plugins.Bun {
			line = plugText(p.OnBun, "v", a.data.Plugins.BunVer)
		}
		ui.Text(c, line).FontSize(11).TextColor(t.TextMuted).Shrink(0).SingleLine()
		ui.Spacer(c)
		if anyAsked {
			label := p.CheckForUpdates
			if a.plugChecking {
				label = p.Checking
			}
			b := a.plugTextButton(c, t, label, !a.plugChecking && a.plugBusy == "", false)
			tip := p.CheckTip
			if !a.plugCheckedAt.IsZero() {
				tip += "\n" + plugText(p.CheckedTip, "time", plugAt(a.lang, a.plugCheckedAt))
			}
			b.Tooltip(tip)
			if b.Clicked() {
				a.plugCheck()
			}
		}
	})
}

// plugAt is the hour and minute in the language drawn, as magpie's page
// says a check's time.
func plugAt(lang Lang, at time.Time) string {
	if at.IsZero() {
		return ""
	}
	return plugClock(lang, at)
}

// plugClock is a time of day in the language drawn: 24-hour for Chinese,
// 12-hour for English, as the reader's own locale says it.
func plugClock(lang Lang, at time.Time) string {
	if lang == LangZH {
		return at.Format("15:04")
	}
	return at.Format("3:04 PM")
}

// plugShortDate is a day without its year, as the auto-update chip's own
// words say it.
func plugShortDate(lang Lang, at time.Time) string {
	if lang == LangZH {
		return at.Format("1月2日")
	}
	return at.Format("Jan 2")
}

// Kind is what kind of plugin an entry is, for its tile and its chips.
func (e PluginEntryRow) Kind() string {
	if e.Middleware != nil || e.IsMiddleware {
		return "middleware"
	}
	if e.IsAgent {
		return "agent"
	}
	if len(e.Tools) > 0 && len(e.Providers) == 0 {
		return "tools"
	}
	if e.HasPage {
		return "page"
	}
	return ""
}

// plugShownName is an installed plugin's name as its row shows it: the
// market's, else the package's or the folder's.
func (a *App) plugShownName(e PluginEntryRow) string {
	pkg := plugin.Name(e.Spec)
	if a.plugMarket != nil {
		for _, l := range a.plugMarket.Listings {
			if l.Package == pkg {
				return l.Name
			}
		}
		if e.Name != "" {
			return e.Name
		}
	}
	if plugin.IsGit(e.Spec) && e.Package != "" {
		return e.Package
	}
	return e.Name
}

// plugLatestFor is npm's newest version the page knows: the check's answer
// while it stands, else what npm said last when the data was read, else the
// market's.
func (a *App) plugLatestFor(e PluginEntryRow) string {
	if a.plugChecks != nil {
		if c, ok := a.plugChecks.By[e.Spec]; ok && c.Latest != "" {
			return c.Latest
		}
	}
	if a.plugMarket != nil {
		if n, ok := a.plugMarket.NPM[plugin.Name(e.Spec)]; ok && n.Version != "" {
			return n.Version
		}
	}
	return e.Latest
}

// plugHeadFor is the commit an installed git plugin's repository is at now:
// the check's answer while it stands, else what GitHub said when the market
// was asked, else the last one kept.
func (a *App) plugHeadFor(e PluginEntryRow) string {
	if a.plugChecks != nil {
		if c, ok := a.plugChecks.By[e.Spec]; ok && c.Head != "" {
			return plugin.ShortCommit(c.Head)
		}
	}
	if a.plugMarket != nil {
		if h, ok := a.plugHeads[e.Spec]; ok && h != "" {
			return plugin.ShortCommit(h)
		}
	}
	return e.Head
}

// plugCheckFor is what the last check said of one plugin.
func (a *App) plugCheckFor(spec string) (plugin.VersionCheck, bool) {
	if a.plugChecks == nil {
		return plugin.VersionCheck{}, false
	}
	c, ok := a.plugChecks.By[spec]
	return c, ok
}

// plugTextButton is one of a row's plain buttons: a word that acts, quiet
// until hovered, and a primary one drawn in the accent.
func (a *App) plugTextButton(c *ui.Context, t *ui.Theme, label string, enabled, primary bool) ui.Element {
	col := t.Text
	if primary {
		col = t.Accent
	}
	b := ui.Row(c).Center().Height(24).Padding(0, 8).Radius(6).Shrink(0).
		Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).Label(label)
	if !enabled {
		b.Disabled(true).Opacity(0.5)
	}
	if b.Hovered() && enabled {
		b.Background(a.skHoverBg(t))
	}
	b.Children(func() {
		ui.Text(c, label).FontSize(12).FontWeight(500).TextColor(col).SingleLine()
	})
	return b
}

// plugOptionsOpen opens a middleware's options editor with what it runs
// with, or what its package suggests when it has none.
func (a *App) plugOptionsOpen(e PluginEntryRow) {
	opts := e.Options
	if opts == nil {
		opts = e.OptionsExample
	}
	if opts == nil {
		opts = map[string]any{}
	}
	text, err := json.MarshalIndent(opts, "", "  ")
	if err != nil {
		text = []byte("{}")
	}
	a.plugOptions, a.plugOptsText, a.plugOptsErr = e.Spec, string(text), ""
	a.invalidate()
}

// plugSaveOptionsText saves what was typed: a JSON object, or it says what
// was wrong and keeps the editor open.
func (a *App) plugSaveOptionsText(spec string) {
	p := &a.text.S.Plug
	var v any
	text := strings.TrimSpace(a.plugOptsText)
	if text == "" {
		text = "{}"
	}
	if err := json.Unmarshal([]byte(text), &v); err != nil {
		a.plugOptsErr = plugText(p.OptionsNotJSON, "error", err.Error())
		a.invalidate()
		return
	}
	obj, ok := v.(map[string]any)
	if !ok {
		a.plugOptsErr = p.OptionsObject
		a.invalidate()
		return
	}
	a.plugOptsErr = ""
	a.plugSaveOptions(spec, obj)
}

// plugClearOptions asks before a middleware's saved options go, as magpie's
// page does, then clears them.
func (a *App) plugClearOptions(spec string) {
	a.plugClearAsk = spec
	a.invalidate()
}

// plugAskRemoveSheet opens the ask before a plugin is removed.
func (a *App) plugAskRemoveSheet(spec string) {
	a.plugRemoveAsk = spec
	a.invalidate()
}
