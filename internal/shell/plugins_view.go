//go:build nogui && echo

package shell

import (
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/yetone/magpie/internal/plugin"
)

// The Plugins page, as magpie draws it (internal/gui/assets/plugins.js): a
// head of three tabs, a search field and the 「国内镜像」 switch, over one of
// three bodies — the built-ins' switches (plugins_builtin.go), the market
// (plugins_discover.go), and what was installed from it (plugins_installed.go).
//
// This file is the head and the parts the three share: the words the page
// translates, the cards' look, and the notice a write leaves.

// plugNum is a count as the page's words carry it.
func plugNum(n int) string { return strconv.Itoa(n) }

// plugSubNames are a plugin's subscriptions' names, as its row lists them.
func (a *App) plugSubNames(e PluginEntryRow) []string {
	out := make([]string, 0, len(e.Providers))
	for _, s := range e.Providers {
		out = append(out, s.Name)
	}
	return out
}

// plugMarketData is what the market answered, kept across frames: the
// plugins magpie suggests and what npm says of each. plugins_market.go asks
// it once, when Discover first shows.
type plugMarketData struct {
	Listings []plugin.Listing
	NPM      map[string]plugin.NPM
	At       time.Time
}

// plugCheckResult is what the page's Check for updates found, by spec, so a
// row can say an update is out before npm's hourly cache does.
type plugCheckResult struct {
	At time.Time
	By map[string]plugin.VersionCheck
}

// plugDetail is the plugin whose own page is open over the list: what its
// card or row knew, and where it came from.
type plugDetail struct {
	Spec      string
	Name      string
	Icon      string
	Kind      string
	Community bool
	Replaces  string
	NPM       *plugin.NPM
	Tag       *plugin.Tagged
	Providers []string
	Have      bool
}

// pluginsView is the page.
func (a *App) pluginsView(c *ui.Context) {
	t := c.Theme()
	if a.plugTab > PlugTabInstalled {
		a.plugTab = PlugTabDiscover
	}
	// the market downloads, so it is asked as its tab shows, as magpie's
	// page asks it — not on the first paint, and never by a tester, which
	// has no window and no network
	if a.win != nil && a.plugTab != PlugTabBuiltin {
		a.plugMarketKick()
	}
	// the host's own answer on subscriptions two plugins sign in to, asked
	// once when the page shows (magpie asks it with its state)
	if a.win != nil {
		a.plugHostKick()
	}

	ui.Column(c).Fill().Padding(8, 24, 0, 24).Gap(12).Children(func() {
		a.pluginsHead(c, t)
		if a.plugNotice != "" {
			a.plugNoticeLine(c, t)
		}
		switch a.plugTab {
		case PlugTabBuiltin:
			a.pluginsBuiltin(c, t)
		case PlugTabInstalled:
			a.pluginsInstalled(c, t)
		default:
			a.pluginsDiscover(c, t)
		}
	})
}

// pluginsHead is the tabs, the mirror switch and the search field; the last
// two are left out of Built-in, which has nothing to search or fetch
// (plugins.js's searchShown).
func (a *App) pluginsHead(c *ui.Context, t *ui.Theme) {
	ui.Row(c).AlignItems(ui.Center).Gap(10).Children(func() {
		a.plugTabs(c, t)
		ui.Spacer(c)
		if a.plugTab != PlugTabBuiltin {
			a.plugMirror(c, t)
			a.plugFindField(c, t)
		}
	})
}

// plugTabs is the three tabs, a segmented pill as magpie's .segs and
// .lib-tabs draw them: Built-in, Discover, and Installed with how many
// there are once there are any.
func (a *App) plugTabs(c *ui.Context, t *ui.Theme) {
	bg := ui.RGBA(0, 0, 0, 0.05)
	sel := ui.Hex("#ffffff")
	if t.Dark {
		bg = ui.RGBA(255, 255, 255, 0.07)
		sel = ui.Hex("#34343a")
	}

	p := &a.text.S.Plug
	updates := a.data.Plugins.Updates
	installed := p.TabInstalled
	if n := len(a.data.Plugins.Installed); n > 0 && updates == 0 {
		installed += " · " + plugNum(n)
	}
	labels := []string{p.TabBuiltin, p.TabDiscover, installed}

	box := ui.Row(c).AlignItems(ui.Center).Gap(2).Padding(2).Radius(7).
		Background(bg).Shrink(0)
	box.Children(func() {
		for i, label := range labels {
			tab := PluginTab(i)
			on := a.plugTab == tab
			b := ui.Row(c).AlignItems(ui.Center).Height(26).Padding(0, 11).
				Radius(5).Shrink(0).Cursor(ui.CursorPointer).
				Focusable().Role(ui.RoleButton).Label(label)
			if on {
				b.Background(sel)
			} else if b.Hovered() || b.Focused() {
				b.Background(a.skHoverBg(t))
			}
			if b.Clicked() && !on {
				a.plugTab = tab
				a.plugNotice = ""
				c.Invalidate()
			}
			if tab == PlugTabInstalled && updates > 0 {
				b.Tooltip(p.UpdatesTip)
			}
			b.Children(func() {
				col, weight := t.TextMuted, 500
				if on {
					col, weight = t.Text, 600
				}
				ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
					ui.Text(c, label).FontSize(12.5).FontWeight(weight).
						TextColor(col).SingleLine()
					if tab == PlugTabInstalled && updates > 0 {
						badgeText := plugNum(updates) + " 待更新"
						if a.lang != LangZH {
							badgeText = plugNum(updates) + " updates"
							if updates == 1 {
								badgeText = "1 update"
							}
						}
						ui.Row(c).Center().Height(16).Padding(0, 5).Radius(8).
							Background(t.Accent).Shrink(0).Children(func() {
							ui.Text(c, badgeText).FontSize(10).FontWeight(600).
								TextColor(t.AccentText).SingleLine()
						})
					}
				})
			})
		}
	})
}

// plugMirror is 「国内镜像」: whether the market, npm and Bun are asked of
// mirrors in China first (settings.ChinaMirror). It is drawn only once the
// installed list has come, which says how the setting stands, as magpie's
// button is.
func (a *App) plugMirror(c *ui.Context, t *ui.Theme) {
	p := &a.text.S.Plug
	on := a.data.Plugins.Mirror
	row := ui.Row(c).AlignItems(ui.Center).Gap(7).Shrink(0).
		Cursor(ui.CursorPointer).Role(ui.RoleButton).Focusable().
		Label(p.Mirror).Tooltip(p.MirrorTip)
	if a.plugMirrorBusy {
		row.Opacity(0.6)
	}
	if row.Clicked() && !a.plugMirrorBusy {
		a.plugMirrorBusy = true
		go func() {
			err := plugMirrorSeam(!on)
			a.win.Update(func() {
				a.plugMirrorBusy = false
				if err != nil {
					a.plugNotice, a.plugNoticeErr = err.Error(), true
				} else {
					a.plugNotice, a.plugNoticeErr = "", false
					a.data.Plugins.Mirror = !on
				}
				a.win.Invalidate()
			})
		}()
	}
	row.Children(func() {
		col := t.TextMuted
		if on {
			col = t.Text
		}
		ui.Text(c, p.Mirror).FontSize(12).TextColor(col).SingleLine()
		v := on
		sw := ui.Switch(c, &v).Shrink(0)
		if sw.Changed() && !a.plugMirrorBusy {
			a.plugMirrorBusy = true
			go func() {
				err := plugMirrorSeam(v)
				a.win.Update(func() {
					a.plugMirrorBusy = false
					if err != nil {
						a.plugNotice, a.plugNoticeErr = err.Error(), true
					} else {
						a.plugNotice, a.plugNoticeErr = "", false
						a.data.Plugins.Mirror = v
					}
					a.win.Invalidate()
				})
			}()
		}
	})
}

// plugFindField is the head's search field: the market's plugins, the npm
// search, and — while it has words — the page is Discover, as magpie's is
// (typing on Installed moves there).
func (a *App) plugFindField(c *ui.Context, t *ui.Theme) {
	p := &a.text.S.Plug
	bg := ui.RGBA(0, 0, 0, 0.055)
	if t.Dark {
		bg = ui.RGBA(255, 255, 255, 0.075)
	}
	box := ui.Row(c).AlignItems(ui.Center).Gap(6).Padding(0, 10).
		Height(28).Width(280).Radius(8).Background(bg).Shrink(0)

	var field ui.Element
	box.Children(func() {
		ui.Icon(c, searchIcon()).Size(12, 12).TextColor(t.TextMuted).Shrink(0)
		field = ui.TextInputBase(c, &a.plugQuery).Grow(1).Shrink(1).
			Placeholder(p.SearchPlaceholder).Label(p.SearchLabel)
		if a.plugQuery != "" {
			if b := a.skIconButton(c, t, closeIcon(), a.text.S.LibClear, false); b.Clicked() {
				a.plugQuery = ""
				a.plugSearch = nil
				a.plugSearchErr = ""
				c.Invalidate()
			}
		}
	})
	if field.Changed() {
		if a.plugTab != PlugTabDiscover {
			a.plugTab = PlugTabDiscover
			a.plugMarketKick()
		}
		a.plugSearchKick()
	}
}

// plugNoticeLine is what the last write said, above the body.
func (a *App) plugNoticeLine(c *ui.Context, t *ui.Theme) {
	col := t.TextMuted
	if a.plugNoticeErr {
		col = t.Danger
	}
	ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
		ui.Text(c, a.plugNotice).FontSize(11.5).TextColor(col).
			Grow(1).Shrink(1).Wrap()
		if b := a.skIconButton(c, t, closeIcon(), a.text.S.LibClose, false); b.Clicked() {
			a.plugNotice = ""
			c.Invalidate()
		}
	})
}

// plugText fills {name} slots, as magpie's t(str, vars) does.
func plugText(s string, kv ...string) string {
	for i := 0; i+1 < len(kv); i += 2 {
		s = strings.ReplaceAll(s, "{"+kv[i]+"}", kv[i+1])
	}
	return s
}

// plugWord translates one of the built-ins' English strings — a manifest's
// name, what it is, or what of it is in use — in the language drawn.
func (a *App) plugWord(s string) string {
	if a.lang == LangZH {
		if w, ok := pluginWords[s]; ok {
			return w
		}
	}
	return s
}

// plugName is a built-in's name.
func (a *App) plugName(b BuiltinRow) string { return a.plugWord(b.Name) }

// plugUse is what of a built-in is in use, in the language drawn: its One
// for 1, its Many for any other number.
func (a *App) plugUse(b BuiltinRow, n int) string {
	one, many := a.plugWord(b.One), a.plugWord(b.Many)
	if n == 1 {
		return plugText(one, "n", plugNum(n))
	}
	return plugText(many, "n", plugNum(n))
}

// plugTile is a plugin's picture: draws the plugin's real icon when available,
// falling back to its kind glyph or letter initial.
func (a *App) plugTile(c *ui.Context, t *ui.Theme, icon, name, kind string, big bool) {
	size, radius, font := float32(36), float32(10), float32(15)
	imgSize := float32(22)
	if big {
		size, radius, font = 44, 12, 19
		imgSize = 28
	}
	bg := ui.Hex("#ffffff")
	line := ui.RGBA(0, 0, 0, 0.08)
	if t.Dark {
		bg = ui.Hex("#2a2a2e")
		line = ui.RGBA(255, 255, 255, 0.1)
	}

	tile := ui.Row(c).Center().Width(size).Height(size).Radius(radius).
		Background(bg).Border(1, line).Shrink(0)
	if !t.Dark {
		tile.Shadow(0, 1, 3, 0, ui.RGBA(0, 0, 0, 0.06))
	}

	var pic ui.ImageSource
	if icon != "" {
		pic = iconPictureFor(icon)
	}
	tile.Children(func() {
		if pic != nil && pic != genericIcon() {
			ui.Image(c, pic).Size(imgSize, imgSize).Radius(2)
			return
		}
		if kind != "" && kind != "provider" {
			ui.Icon(c, plugKindIcon(kind)).Size(imgSize-4, imgSize-4).TextColor(t.TextMuted)
			return
		}
		initial := "?"
		if name != "" {
			initial = strings.ToUpper(string([]rune(name)[0]))
		}
		col := t.TextMuted
		if kind == "" || kind == "provider" {
			col = t.Accent
		}
		ui.Text(c, initial).FontSize(font).FontWeight(600).
			TextColor(col).SingleLine()
	})
}

// plugIconByName resolves well-known plugin names or package names to an icon name.
func plugIconByName(name, spec string) string {
	n := strings.ToLower(name + " " + spec)
	switch {
	case strings.Contains(n, "kimi"):
		return "kimi"
	case strings.Contains(n, "devin"):
		return "devin"
	case strings.Contains(n, "zed"):
		return "zed"
	case strings.Contains(n, "zcode") || strings.Contains(n, "z.ai") || strings.Contains(n, "glm"):
		return "zcode"
	case strings.Contains(n, "cursor"):
		return "cursor"
	case strings.Contains(n, "qoder"):
		return "qoder"
	case strings.Contains(n, "factory"):
		return "factory"
	case strings.Contains(n, "grok") || strings.Contains(n, "xai") || strings.Contains(n, "x.ai"):
		return "xai"
	case strings.Contains(n, "command"):
		return "commandcode"
	case strings.Contains(n, "mimo"):
		return "mimocode"
	case strings.Contains(n, "kiro"):
		return "kiro-color"
	case strings.Contains(n, "workbuddy") || strings.Contains(n, "codebuddy"):
		return "workbuddy-color"
	case strings.Contains(n, "copilot"):
		return "githubcopilot"
	case strings.Contains(n, "claude") || strings.Contains(n, "anthropic"):
		return "claude-color"
	case strings.Contains(n, "gemini") || strings.Contains(n, "google"):
		return "gemini-color"
	case strings.Contains(n, "deepseek"):
		return "deepseek-color"
	case strings.Contains(n, "moonshot"):
		return "moonshot"
	case strings.Contains(n, "zen free") || strings.Contains(n, "zen-free") || strings.Contains(n, "opencode"):
		return "opencode"
	}
	return ""
}

// plugIconForCard finds the best icon for a market card or tagged repo.
func (a *App) plugIconForCard(cd plugCard) string {
	if cd.Icon != "" {
		return cd.Icon
	}
	if cd.Tag != nil && cd.Tag.Icon != "" {
		return cd.Tag.Icon
	}
	if cd.NPM != nil && cd.NPM.Icon != "" {
		return cd.NPM.Icon
	}
	firstSub := ""
	if len(cd.Providers) > 0 {
		firstSub = cd.Providers[0]
	}
	if ic := plugin.Icon(cd.Package, firstSub); ic != "" {
		return ic
	}
	return plugIconByName(cd.Name, cd.Package)
}

// plugIconForEntry finds the best icon for an installed plugin entry.
func (a *App) plugIconForEntry(e PluginEntryRow) string {
	if len(e.Providers) > 0 && e.Providers[0].Icon != "" {
		return e.Providers[0].Icon
	}
	firstSub := ""
	if len(e.Providers) > 0 {
		firstSub = e.Providers[0].Pid
		if firstSub == "" {
			firstSub = e.Providers[0].ID
		}
	}
	if ic := plugin.Icon(e.Spec, firstSub); ic != "" {
		return ic
	}
	return plugIconByName(e.Name, e.Spec)
}

// plugKind is what kind of plugin a listing or an entry is: "" for a
// provider, else "middleware", "agent", "tools" or "page".
func plugKindOf(kind string) string {
	if kind == "" || kind == "provider" {
		return ""
	}
	return kind
}

// plugKindChip is the small word for what a plugin is, with its glyph.
func (a *App) plugKindChip(c *ui.Context, t *ui.Theme, kind string) {
	p := &a.text.S.Plug
	label, col := "", t.TextMuted
	switch plugKindOf(kind) {
	case "":
		label = p.KindProvider
		col = t.Accent
	case "middleware":
		label = p.KindMiddleware
		col = t.Success
	case "gateway":
		// the same glyph and colour as middleware, as magpie's own chip
		// says its own word
		label = p.KindHook
		col = t.Success
	case "agent":
		label = p.KindAgent
		col = t.Warning
	case "tools":
		label = p.KindTools
	case "page":
		label = p.KindPage
	case "pn":
		label = p.KindPanels
	}
	if label == "" {
		return
	}
	ui.Row(c).AlignItems(ui.Center).Gap(4).Shrink(0).Children(func() {
		ui.Icon(c, plugKindIcon(kind)).Size(11, 11).TextColor(col)
		ui.Text(c, label).FontSize(10.5).TextColor(col).SingleLine()
	})
}

// plugChip is a small filled mark: a version, a count, a warning.
func (a *App) plugChip(c *ui.Context, t *ui.Theme, label string, tone string) {
	a.plugChipTip(c, t, label, tone, "")
}

// plugChipTip is a chip that says over itself what it means, when it has
// something to say (magpie's own chip titles).
func (a *App) plugChipTip(c *ui.Context, t *ui.Theme, label string, tone string, tip string) {
	bg, col := ui.RGBA(0, 0, 0, 0.055), t.TextMuted
	if t.Dark {
		bg = ui.RGBA(255, 255, 255, 0.08)
	}
	switch tone {
	case "up":
		bg, col = t.Accent, t.AccentText
	case "bad":
		bg, col = ui.RGBA(220, 70, 70, 0.14), t.Danger
	case "done":
		bg, col = ui.RGBA(60, 170, 90, 0.14), t.Success
	case "warn":
		bg, col = ui.RGBA(210, 150, 40, 0.16), t.Warning
	}
	chip := ui.Row(c).AlignItems(ui.Center).Height(17).Padding(0, 6).Radius(5).
		Background(bg).Shrink(0)
	if tip != "" {
		chip.Tooltip(tip)
	}
	chip.Children(func() {
		ui.Text(c, label).FontSize(10.5).FontWeight(500).
			TextColor(col).SingleLine()
	})
}

// plugAct is a plugin's one button, as magpie's .pm-act draws it: 26 high,
// rounded 13, filled by what it does.
func (a *App) plugAct(c *ui.Context, t *ui.Theme, label, tone string, enabled bool) ui.Element {
	bg, col := ui.RGBA(0, 0, 0, 0.06), t.Text
	if t.Dark {
		bg = ui.RGBA(255, 255, 255, 0.09)
	}
	switch tone {
	case "get", "up":
		bg, col = t.Accent, t.AccentText
	case "go":
		bg, col = t.Accent, t.AccentText
	case "done":
		bg, col = ui.RGBA(0, 0, 0, 0), t.Success
	}
	b := ui.Row(c).Center().Height(26).Padding(0, 12).Radius(13).
		Background(bg).Shrink(0).Cursor(ui.CursorPointer).
		Focusable().Role(ui.RoleButton).Label(label)
	if !enabled {
		b.Disabled(true).Opacity(0.55)
	}
	if b.Hovered() && enabled && tone != "done" {
		b.Opacity(0.88)
	}
	b.Children(func() {
		ui.Text(c, label).FontSize(12).FontWeight(600).
			TextColor(col).SingleLine()
	})
	return b
}

// plugCardBg is the face of the page's cards and rows.
func plugCardBg(t *ui.Theme) ui.Color {
	if t.Dark {
		return ui.Hex("#232327")
	}
	return ui.Hex("#ffffff")
}

// plugLineC is a card's or a row's hairline.
func plugLineC(t *ui.Theme) ui.Color {
	if t.Dark {
		return ui.RGBA(255, 255, 255, 0.09)
	}
	return ui.RGBA(0, 0, 0, 0.09)
}

// plugKicker is a section's small heading and its hint, as .pm-sechead.
func (a *App) plugKicker(c *ui.Context, t *ui.Theme, title, hint string) {
	ui.Row(c).AlignItems(ui.Center).Gap(8).Wrap().Children(func() {
		ui.Text(c, title).FontSize(12.5).FontWeight(650).
			TextColor(t.Text).Shrink(0)
		if hint != "" {
			ui.Text(c, hint).FontSize(11.5).TextColor(t.TextMuted).
				Shrink(1).SingleLine()
		}
	})
}
