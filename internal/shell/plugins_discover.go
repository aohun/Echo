//go:build nogui && echo

package shell

import (
	"fmt"
	"sort"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/yetone/magpie/internal/plugin"
	"github.com/yetone/magpie/internal/update"
)

// Plugins › Discover: the market of plugins magpie suggests (its community's,
// a card each), the repositories their authors tagged magpie-plugin on
// GitHub — nobody reviewed those — the npm search the head's field makes, and
// the way to install one the market doesn't list. magpie's page draws it in
// internal/gui/assets/plugins.js (drawBody).

// plugCard is one card of a section: what a plugin is, where it comes from,
// and whether it is installed.
type plugCard struct {
	Package   string
	Name      string
	Icon      string
	Summary   string
	Community bool
	Replaces  string
	Kind      string // "middleware", "agent", "tools", "page", or "" for a provider
	NPM       *plugin.NPM
	Tag       *plugin.Tagged
	// Providers are the subscriptions it signs in to, as the market names
	// them.
	Providers []string
}

// pluginsDiscover is the tab's body.
func (a *App) pluginsDiscover(c *ui.Context, t *ui.Theme) {
	m := a.plugMarket
	if m == nil {
		if a.plugMarketErr != "" {
			ui.Text(c, plugText(a.text.S.Plug.CouldntLoad, "error", a.plugMarketErr)).
				FontSize(12.5).TextColor(t.Danger).Wrap()
			return
		}
		ui.Column(c).FillWidth().Gap(14).Children(func() {
			a.plugIntro(c, t)
			ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
				ui.Spinner(c).Size(14, 14)
				ui.Text(c, a.text.S.Plug.Checking).FontSize(12).TextColor(t.TextMuted)
			})
		})
		return
	}

	f := strings.ToLower(strings.TrimSpace(a.plugQuery))
	listed := make(map[string]bool, len(m.Listings))
	for _, l := range m.Listings {
		listed[l.Package] = true
	}
	var ls []plugCard
	for _, l := range m.Listings {
		if plugListingMatches(a, l, f) {
			ls = append(ls, a.plugCardOfListing(l))
		}
	}
	var gh []plugin.Tagged
	for _, g := range a.plugTagged {
		if g.Package != "" && listed[g.Package] {
			continue
		}
		if !plugTagMatches(g, f) {
			continue
		}
		gh = append(gh, g)
	}

	ui.Column(c).FillWidth().Gap(18).Children(func() {
		if f == "" {
			a.plugIntro(c, t)
			a.plugManualForm(c, t)
			// magpie's community's alone: others' plugins are found by a
			// search
			sects := []struct{ kind, title, hint string }{
				{"", a.text.S.Plug.SectSubscriptions, a.text.S.Plug.SectSubscriptionsHint},
				{"middleware", a.text.S.Plug.SectMiddleware, a.text.S.Plug.SectMiddlewareHint},
				{"agent", a.text.S.Plug.SectAgents, a.text.S.Plug.SectAgentsHint},
				{"tools", a.text.S.Plug.SectTools, a.text.S.Plug.SectToolsHint},
				{"page", a.text.S.Plug.SectPages, a.text.S.Plug.SectPagesHint},
			}
			for _, s := range sects {
				var cards []plugCard
				for _, l := range ls {
					if l.Community && plugKindOf(l.Kind) == s.kind {
						cards = append(cards, l)
					}
				}
				if len(cards) > 0 {
					a.plugSection(c, t, s.title, s.hint, cards)
				}
			}
			if len(gh) > 0 {
				a.plugGhSection(c, t, gh)
			} else if a.plugTaggedErr != "" {
				a.plugGhFailed(c, t)
			} else if !a.plugTaggedDone {
				a.plugGhWaiting(c, t)
			}
			return
		}
		if len(ls) > 0 {
			a.plugSection(c, t, a.text.S.Plug.Suggested, "", ls)
		}
		if len(gh) > 0 {
			a.plugGhSection(c, t, gh)
		} else if a.plugTaggedErr == "" && !a.plugTaggedDone {
			a.plugGhWaiting(c, t)
		}
		a.plugNpmSection(c, t, listed, len(ls) > 0)
	})
}

// plugIntro is the page's own words about what a plugin is, and where to read
// about writing one, matching magpie's modern header design.
func (a *App) plugIntro(c *ui.Context, t *ui.Theme) {
	p := &a.text.S.Plug
	card := ui.Row(c).AlignItems(ui.Center).Gap(18).FillWidth().Padding(16, 20).
		Radius(14).Border(1, plugLineC(t))
	if t.Dark {
		card.Gradient(ui.Hex("#222226"), ui.RGBA(35, 33, 48, 1), 120)
	} else {
		card.Gradient(ui.Hex("#ffffff"), ui.RGBA(243, 244, 255, 1), 120)
	}
	card.Children(func() {
		ui.Column(c).Grow(1).Shrink(1).Gap(6).Children(func() {
			ui.Text(c, p.IntroTitle).FontSize(15).FontWeight(650).TextColor(t.Text).
				Wrap()
			ui.Text(c, p.IntroBody).FontSize(12).TextColor(t.TextMuted).Wrap()
			ui.Row(c).AlignItems(ui.Center).Gap(8).Wrap().Children(func() {
				ui.Text(c, p.IntroMore).FontSize(12).TextColor(t.TextMuted).
					Grow(1).Shrink(1).Wrap()
				link := ui.Row(c).AlignItems(ui.Center).Gap(4).Padding(3, 8).Radius(6).
					Background(ui.RGBA(0, 0, 0, 0.05)).
					Cursor(ui.CursorPointer).Focusable().Role(ui.RoleLink).
					Label(p.WriteAPlugin).Shrink(0)
				if t.Dark {
					link.Background(ui.RGBA(255, 255, 255, 0.08))
				}
				if link.Hovered() {
					link.Background(a.skHoverBg(t))
				}
				if link.Clicked() {
					// the docs exist in en, zh and ja; the language drawn
					// picks the path, as magpie's own link does
					docs := "https://usemagpie.ai/docs/plugins"
					if a.lang == LangZH {
						docs = "https://usemagpie.ai/docs/zh/plugins"
					}
					a.openBrowser(docs)
				}
				link.Children(func() {
					ui.Text(c, p.WriteAPlugin).FontSize(11.5).FontWeight(500).
						TextColor(t.Text).SingleLine()
					ui.Icon(c, plugIcon("out")).Size(11, 11).TextColor(t.TextMuted)
				})
			})
			ui.Row(c).AlignItems(ui.Center).Gap(6).Margin(2, 0, 0, 0).Children(func() {
				ui.Icon(c, plugIcon("shield")).Size(13, 13).TextColor(ui.Hex("#10b981")).Shrink(0)
				ui.Text(c, p.Trust).FontSize(11.5).TextColor(t.TextMuted).
					Grow(1).Shrink(1).Wrap()
			})
		})
		// Artwork: 2-row cluster with real icons like in magpie
		ui.Column(c).AlignItems(ui.End).Gap(8).Shrink(0).Children(func() {
			ui.Row(c).Gap(8).Children(func() {
				for _, ic := range []string{"zed", "opencode", "gemini-color"} {
					a.plugArtTile(c, t, ic, 4)
				}
			})
			ui.Row(c).Gap(8).Children(func() {
				for _, ic := range []string{"devin", "kiro-color"} {
					a.plugArtTile(c, t, ic, 0)
				}
			})
		})
	})
}

// plugArtTile is one rounded logo square in the intro artwork cluster.
func (a *App) plugArtTile(c *ui.Context, t *ui.Theme, ic string, offsetY float32) {
	bg := ui.Hex("#ffffff")
	line := ui.RGBA(0, 0, 0, 0.08)
	if t.Dark {
		bg = ui.RGBA(42, 42, 46, 1)
		line = ui.RGBA(255, 255, 255, 0.12)
	}
	tile := ui.Row(c).Center().Width(34).Height(34).Radius(10).
		Background(bg).Border(1, line).Shrink(0)
	if offsetY != 0 {
		tile.Margin(offsetY, 0, 0, 0)
	}
	if !t.Dark {
		tile.Shadow(0, 2, 6, 0, ui.RGBA(0, 0, 0, 0.08))
	}
	pic := iconPictureFor(ic)
	tile.Children(func() {
		if pic != nil && pic != genericIcon() {
			ui.Image(c, pic).Size(20, 20).Radius(2)
		}
	})
}

// plugSection is one row of cards under its heading.
func (a *App) plugSection(c *ui.Context, t *ui.Theme, title, hint string, cards []plugCard) {
	ui.Column(c).FillWidth().Gap(9).Children(func() {
		a.plugKicker(c, t, title, hint)
		ui.Row(c).Wrap().Gap(10).FillWidth().Children(func() {
			for _, cd := range cards {
				a.plugCard(c, t, cd)
			}
		})
	})
}

// plugCardBgCardWidth and plugCardHeight are a card's size: as wide as
// magpie's grid column, as tall as its content.
const plugCardWidth = 300

// plugCardOfListing reads a market listing as a card.
func (a *App) plugCardOfListing(l plugin.Listing) plugCard {
	cd := plugCard{
		Package:   l.Package,
		Name:      l.Name,
		Icon:      l.Icon,
		Summary:   a.plugSummary(l.Summary),
		Community: l.Community,
		Replaces:  l.Replaces,
		Kind:      plugKindOf(l.Kind),
		Providers: l.Providers,
	}
	if a.plugMarket != nil {
		if n, ok := a.plugMarket.NPM[l.Package]; ok {
			nn := n
			cd.NPM = &nn
		}
	}
	// a repository tagged on GitHub, listed by the market too: the card is
	// the market's, with GitHub's stars on it
	for i := range a.plugTagged {
		g := &a.plugTagged[i]
		if g.Package == l.Package || plugGitRepo(l.Package) == strings.ToLower(g.Repo) {
			cd.Tag = g
			break
		}
	}
	return cd
}

// plugCardOfTag reads a repository tagged on GitHub as a card, as magpie's
// ghListing does: its owner is its publisher, its version its package.json's.
func plugCardOfTag(g plugin.Tagged) plugCard {
	cd := plugCard{
		Package: g.Spec,
		Name:    g.Repo,
		Icon:    g.Icon,
		Kind:    plugKindOf(g.Kind),
		Tag:     &g,
	}
	if i := strings.LastIndex(g.Repo, "/"); i >= 0 && i+1 < len(g.Repo) {
		cd.Name = g.Repo[i+1:]
	}
	if g.Description != "" {
		cd.Summary = g.Description
	}
	n := plugin.NPM{Version: g.Version, Publisher: g.Owner, Repository: g.URL, License: g.License}
	cd.NPM = &n
	return cd
}

// plugSummary is a listing's own words in the language drawn, else English.
func (a *App) plugSummary(m map[string]string) string {
	if s := m[pluginLangKey(a.lang)]; s != "" {
		return s
	}
	return m["en"]
}

// pluginLangKey is the language the market's summaries are keyed by: the
// page's own language, and English for the languages the market doesn't
// carry (magpie's langKey).
func pluginLangKey(lang Lang) string {
	switch lang {
	case LangZH:
		return "zh"
	}
	return "en"
}

// plugCard draws one card: its picture and name, who wrote it, its summary,
// what it is, and the one button it shows.
func (a *App) plugCard(c *ui.Context, t *ui.Theme, cd plugCard) {
	p := &a.text.S.Plug
	e := a.plugEntryOf(cd.Package, cd.Tag)
	card := ui.Column(c).Width(plugCardWidth).Shrink(0).Gap(7).Padding(12, 12).
		Radius(11).Border(1, plugLineC(t)).Background(plugCardBg(t)).
		Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).Label(cd.Name)
	if e != nil {
		card.BorderColor(t.Accent)
	}
	if card.Hovered() {
		card.BorderColor(t.Accent)
	}
	if card.Clicked() {
		a.plugOpenDetailCard(cd)
	}
	card.Children(func() {
		ui.Row(c).AlignItems(ui.Start).Gap(10).FillWidth().Children(func() {
			a.plugTile(c, t, a.plugIconForCard(cd), cd.Name, cd.Kind, false)
			ui.Column(c).Grow(1).Shrink(1).Gap(2).Children(func() {
				ui.Text(c, cd.Name).FontSize(13).FontWeight(600).TextColor(t.Text).
					Wrap().MaxLines(2)
				ui.Text(c, a.plugBy(cd)).FontSize(11).TextColor(t.TextMuted).
					SingleLine()
			})
			a.plugCardAction(c, t, cd, e)
		})
		if cd.Summary != "" {
			ui.Text(c, cd.Summary).FontSize(11.5).TextColor(t.TextMuted).
				MaxLines(3).Wrap()
		}
		ui.Row(c).AlignItems(ui.Center).Gap(8).Wrap().FillWidth().Children(func() {
			a.plugKindChip(c, t, cd.Kind)
			if cd.Tag != nil {
				a.plugUnofficial(c, t)
				ui.Row(c).AlignItems(ui.Center).Gap(4).Shrink(0).
					Tooltip(plugText(p.StarsTip, "n", plugCount(cd.Tag.Stars))).
					Children(func() {
						ui.Icon(c, plugIcon("star")).Size(11, 11).TextColor(t.TextMuted)
						ui.Text(c, plugCount(cd.Tag.Stars)).FontSize(11).
							TextColor(t.TextMuted).SingleLine()
					})
			}
			if cd.NPM != nil && cd.NPM.Weekly > 0 {
				ui.Row(c).AlignItems(ui.Center).Gap(4).Shrink(0).
					Tooltip(plugText(p.DownloadsTip, "n", plugCount(cd.NPM.Weekly))).
					Children(func() {
						ui.Icon(c, downloadIcon()).Size(11, 11).TextColor(t.TextMuted)
						ui.Text(c, plugText(p.PerWeek, "n", plugCount(cd.NPM.Weekly))).
							FontSize(11).TextColor(t.TextMuted).SingleLine()
					})
			}
			if cd.NPM != nil && cd.NPM.Version != "" {
				ver := "v" + cd.NPM.Version
				if e != nil && e.Version != "" && cd.NPM.Version != e.Version {
					ver = "v" + e.Version + " → v" + cd.NPM.Version
				}
				ui.Text(c, ver).Font("monospace").FontSize(10.5).
					TextColor(t.TextMuted).SingleLine()
			}
			if e != nil && e.Error != "" && !e.Off {
				a.plugChip(c, t, p.DidntLoadChip, "bad")
			} else if e != nil && e.Off {
				a.plugChip(c, t, p.Off, "")
			}
			if cd.Replaces != "" {
				a.plugChipTip(c, t, p.Replaces, "", p.ReplacesTip)
			}
		})
	})
}

// plugBy is who wrote a card's plugin: magpie's community, its npm
// publisher, or its repository's owner.
func (a *App) plugBy(cd plugCard) string {
	if cd.Community {
		return a.text.S.Plug.Community
	}
	if cd.NPM != nil && cd.NPM.Publisher != "" {
		return cd.NPM.Publisher + " · " + cd.Package
	}
	return cd.Package
}

// plugUnofficial is the mark a repository tagged on GitHub carries.
func (a *App) plugUnofficial(c *ui.Context, t *ui.Theme) {
	p := &a.text.S.Plug
	chip := ui.Row(c).AlignItems(ui.Center).Gap(4).Shrink(0).
		Tooltip(plugText(p.UnofficialTip, "topic", plugin.Topic))
	chip.Children(func() {
		ui.Box(c).Width(5).Height(5).Radius(3).Background(t.Warning).Shrink(0)
		ui.Text(c, p.Unofficial).FontSize(10.5).TextColor(t.Warning).SingleLine()
	})
}

// plugCardAction is the one button a card shows: what to do with the plugin
// next, as magpie's actionFor.
func (a *App) plugCardAction(c *ui.Context, t *ui.Theme, cd plugCard, e *PluginEntryRow) {
	p := &a.text.S.Plug
	pkg := cd.Package
	switch {
	case a.plugBusy == "add:"+pkg || a.plugBusy == "add:"+plugSpecOf(e) ||
		a.plugBusy == "upgrade:"+pkg || a.plugBusy == "move:"+pkg:
		label := p.Installing
		switch {
		case strings.HasPrefix(a.plugBusy, "move:"):
			label = p.Moving
		case strings.HasPrefix(a.plugBusy, "upgrade:"):
			label = p.Updating
		case !a.data.Plugins.Bun:
			label = p.GettingBun
		}
		busy := ui.Row(c).AlignItems(ui.Center).Gap(5).Shrink(0)
		// the first plugin needs Bun, downloaded once: the button says so
		if !a.data.Plugins.Bun {
			busy.Tooltip(plugText(p.BunTip, "v", a.data.Plugins.BunVer))
		}
		busy.Children(func() {
			ui.Spinner(c).Size(11, 11)
			ui.Text(c, label).FontSize(12).TextColor(t.TextMuted).SingleLine()
		})
		return
	case a.plugMarketBusy:
		ui.Spinner(c).Size(12, 12)
		return
	case a.plugBusy != "":
		ui.Spinner(c).Size(12, 12)
		return
	}

	if e == nil {
		if cd.NPM == nil || cd.NPM.Version == "" && cd.Tag == nil && cd.NPM.Weekly == 0 && cd.NPM.Description == "" {
			b := a.plugAct(c, t, p.ComingSoon, "", false)
			_ = b
			return
		}
		if mv := a.plugMovers(pkg); len(mv) > 0 {
			// whose accounts the move would bring, then what it does
			b := a.plugAct(c, t, p.Move, "get", true)
			b.Tooltip(a.plugMoveMine(mv[0]) + " — " +
				plugText(p.MoveTip, "name", mv[0].Name))
			if b.Clicked() {
				a.plugMove(mv[0].ID)
			}
			return
		}
		spec := cd.Package
		if cd.Tag != nil && cd.Tag.Spec != "" {
			spec = cd.Tag.Spec
		}
		if b := a.plugAct(c, t, p.Install, "get", true); b.Clicked() {
			a.plugAdd(spec)
		}
		return
	}
	if e.Off {
		if b := a.plugAct(c, t, p.SwitchOn, "", true); b.Clicked() {
			a.plugSetOff(e.Spec, false)
		}
		return
	}
	// signed in to nothing yet, with a built-in's accounts to bring
	cands := a.plugMovers(e.Spec)
	signedIn := false
	unsigned := false
	var firstUnsigned PluginSubRow
	for _, s := range e.Providers {
		if s.SignedIn {
			signedIn = true
		} else if !unsigned {
			firstUnsigned, unsigned = s, true
		}
	}
	if !signedIn && len(cands) > 0 {
		// installed already: the title names it rather than whose accounts
		b := a.plugAct(c, t, p.Move, "get", true)
		b.Tooltip(plugText(p.MoveHere, "name", cands[0].Name) + " — " +
			plugText(p.MoveTip, "name", cands[0].Name))
		if b.Clicked() {
			a.plugMove(cands[0].ID)
		}
		return
	}
	if unsigned && !signedIn {
		b := a.plugAct(c, t, p.SignIn, "go", true)
		if len(e.Providers) > 1 {
			b.Tooltip(plugText(p.SignInTo, "name", firstUnsigned.Name))
		}
		if b.Clicked() {
			a.plugSignInOpen(firstUnsigned)
		}
		return
	}
	// an update out: the one button says so in place of Signed in or
	// Installed
	latest := a.plugLatestFor(*e)
	head := a.plugHeadFor(*e)
	gitBehind := e.Commit != "" && head != "" && !plugSameCommit(e.Commit, head)
	if (latest != "" && e.Version != "" && plugNewer(latest, e.Version)) || gitBehind {
		b := a.plugAct(c, t, p.Update, "up", true)
		switch {
		case plugin.IsGit(e.Spec):
			b.Tooltip(plugText(p.GitFetchTip, "repo", e.Spec) + "\n" +
				plugText(p.GitHaveTip, "have", e.Commit, "repo", e.Spec, "head", head))
		case latest != "":
			tip := plugText(p.UpdateTo, "v", "v"+latest)
			if plugCommunityAuto(e.Spec) {
				tip += "\n" + p.AutoHourTip
			}
			b.Tooltip(tip)
		}
		if b.Clicked() {
			a.plugUpgrade(e.Spec, latest)
		}
		return
	}
	if signedIn {
		b := a.plugAct(c, t, p.SignedIn, "done", true)
		for _, s := range e.Providers {
			if s.SignedIn {
				b.Tooltip(plugText(p.OpenInProviders, "name", s.Name))
				break
			}
		}
		if b.Clicked() {
			a.page = PageProviders
			c.Invalidate()
		}
		return
	}
	b := a.plugAct(c, t, p.Installed, "", true)
	_ = b
}

// plugSpecOf is an entry's spec, or the package name when it has none: what
// a card's write goes by.
func plugSpecOf(e *PluginEntryRow) string {
	if e == nil {
		return ""
	}
	return e.Spec
}

// plugEntryOf is the installed plugin a card is about, by package name or by
// the repository it came from, as magpie's entryOf.
func (a *App) plugEntryOf(pkg string, tag *plugin.Tagged) *PluginEntryRow {
	for i := range a.data.Plugins.Installed {
		e := &a.data.Plugins.Installed[i]
		if plugin.Name(e.Spec) == pkg {
			return e
		}
		if tag != nil {
			if e.Package != "" && plugin.IsGit(e.Spec) && e.Package == tag.Package {
				return e
			}
			if plugGitRepo(e.Spec) == strings.ToLower(tag.Repo) {
				return e
			}
		}
	}
	return nil
}

// plugGitRepo is owner/repo of a git spec, lower case, or "".
func plugGitRepo(spec string) string {
	s := strings.TrimSuffix(spec, ".git")
	s = strings.TrimPrefix(s, "github:")
	s = strings.TrimPrefix(s, "https://github.com/")
	s = strings.TrimPrefix(s, "http://github.com/")
	s = strings.TrimPrefix(s, "https://www.github.com/")
	s = strings.TrimPrefix(s, "http://www.github.com/")
	for _, pre := range []string{"gitlab:", "bitbucket:"} {
		if strings.HasPrefix(spec, pre) {
			return ""
		}
	}
	parts := strings.SplitN(strings.TrimPrefix(s, "/"), "/", 3)
	if len(parts) < 2 {
		return ""
	}
	return strings.ToLower(parts[0] + "/" + parts[1])
}

// plugSameCommit is whether two commits name the same one, one of them
// abbreviated.
func plugSameCommit(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	if len(a) > len(b) {
		return strings.HasPrefix(a, b)
	}
	return strings.HasPrefix(b, a)
}

// plugNewer is update.Newer: whether latest names a newer version than have.
func plugNewer(latest, have string) bool { return update.Newer(latest, have) }

// plugListingMatches is whether the page's search words match a listing:
// its name, package, summary, the subscriptions it signs in to or its
// publisher.
func plugListingMatches(a *App, l plugin.Listing, f string) bool {
	if f == "" {
		return true
	}
	hay := strings.ToLower(strings.Join([]string{
		l.Name, l.Package, a.plugSummary(l.Summary), strings.Join(l.Providers, " "),
	}, " "))
	if l.Package != "" {
		if n, ok := a.plugMarket.NPM[l.Package]; ok {
			hay += " " + strings.ToLower(n.Publisher) + " " + strings.ToLower(n.Description)
		}
	}
	return strings.Contains(hay, f)
}

// plugTagMatches is whether the page's search words match a repository: its
// name, owner, package or description.
func plugTagMatches(g plugin.Tagged, f string) bool {
	if f == "" {
		return true
	}
	hay := strings.ToLower(strings.Join([]string{g.Repo, g.Owner, g.Package, g.Description, g.Spec}, " "))
	return strings.Contains(hay, f)
}

// plugGhSection is the repositories tagged on GitHub: its own strips for the
// kinds there are and for what is installed, and a field that narrows them.
func (a *App) plugGhSection(c *ui.Context, t *ui.Theme, tags []plugin.Tagged) {
	p := &a.text.S.Plug
	ui.Column(c).FillWidth().Gap(9).Children(func() {
		a.plugKicker(c, t, p.GhTitle, plugText(p.GhHint, "topic", plugin.Topic))
		if len(tags) > 6 {
			a.plugGhBar(c, t, tags)
		}
		shown := a.plugGhShown(tags)
		if len(shown) == 0 && len(tags) > 6 {
			ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
				ui.Text(c, p.NoGhMatch).FontSize(12).TextColor(t.TextMuted).Shrink(0)
				if b := a.plugTextButton(c, t, p.ShowAll, true, true); b.Clicked() {
					a.plugGhKind, a.plugGhHave, a.plugGhQuery = "", "", ""
					a.plugGhKindIdx, a.plugGhHaveIdx = 0, 0
					c.Invalidate()
				}
			})
			return
		}
		ui.Row(c).Wrap().Gap(10).FillWidth().Children(func() {
			for _, g := range shown {
				a.plugCard(c, t, plugCardOfTag(g))
			}
		})
	})
}

// plugGhWaiting is the GitHub section while GitHub hasn't answered: its head
// and a word that it is looking, so a section that isn't there yet doesn't
// read as there being no such plugins (magpie's ghWaiting).
func (a *App) plugGhWaiting(c *ui.Context, t *ui.Theme) {
	p := &a.text.S.Plug
	ui.Column(c).FillWidth().Gap(9).Children(func() {
		a.plugKicker(c, t, p.GhTitle, plugText(p.GhHint, "topic", plugin.Topic))
		ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
			ui.Spinner(c).Size(13, 13)
			ui.Text(c, p.LookingOnGitHub).FontSize(12).TextColor(t.TextMuted)
		})
	})
}

// plugGhFailed is the same section when GitHub couldn't be asked at all,
// with another go (magpie's ghFailed).
func (a *App) plugGhFailed(c *ui.Context, t *ui.Theme) {
	p := &a.text.S.Plug
	ui.Column(c).FillWidth().Gap(9).Children(func() {
		a.plugKicker(c, t, p.GhTitle, plugText(p.GhHint, "topic", plugin.Topic))
		ui.Row(c).AlignItems(ui.Center).Gap(10).Wrap().Children(func() {
			ui.Text(c, plugText(p.CouldntAskGitHub, "error", a.plugTaggedErr)).
				FontSize(12).TextColor(t.TextMuted).Shrink(1)
			if b := a.plugTextButton(c, t, p.TryAgain, true, true); b.Clicked() {
				a.plugTaggedAgain()
				c.Invalidate()
			}
		})
	})
}

// plugGhBar is the section's strips and its field: the kinds there are, each
// with how many the field leaves of it, and what is installed, on both sides.
func (a *App) plugGhBar(c *ui.Context, t *ui.Theme, tags []plugin.Tagged) {
	p := &a.text.S.Plug
	found := make([]plugin.Tagged, 0, len(tags))
	for _, g := range tags {
		if plugGhMatch(a, g) {
			found = append(found, g)
		}
	}
	kinds := []string{}
	labels := []string{}
	for _, k := range []string{"", "middleware", "agent", "tools", "page"} {
		if k != "" {
			any := false
			for _, g := range tags {
				if plugKindOf(g.Kind) == k {
					any = true
					break
				}
			}
			if !any {
				continue
			}
		}
		kinds = append(kinds, k)
		labels = append(labels, plugText(p.All, "", "")+plugKindLabel(a, k)+" · "+fmt.Sprint(plugGhCount(a, found, k, a.plugGhHave)))
	}
	// the kind strip, when more than one kind is there
	showKinds := len(kinds) > 1
	if showKinds {
		if a.plugGhKindIdx >= len(labels) {
			a.plugGhKindIdx = 0
		}
		ui.Row(c).AlignItems(ui.Center).Gap(8).Wrap().Children(func() {
			if seg := ui.Segmented(c, &a.plugGhKindIdx, labels...).Shrink(0); seg.Changed() {
				a.plugGhKind = kinds[a.plugGhKindIdx]
				c.Invalidate()
			}
		})
	} else {
		a.plugGhKind = ""
	}
	// what is installed, on both sides of it
	have, none := false, false
	for _, g := range tags {
		if a.plugEntryOf(g.Spec, &g) != nil {
			have = true
		} else {
			none = true
		}
	}
	if have && none {
		hlabels := []string{
			p.All + " · " + fmt.Sprint(plugGhCount(a, found, a.plugGhKind, "")),
			p.Installed + " · " + fmt.Sprint(plugGhCount(a, found, a.plugGhKind, "have")),
			p.NotInstalled + " · " + fmt.Sprint(plugGhCount(a, found, a.plugGhKind, "none")),
		}
		if seg := ui.Segmented(c, &a.plugGhHaveIdx, hlabels...).Shrink(0); seg.Changed() {
			a.plugGhHave = []string{"", "have", "none"}[a.plugGhHaveIdx]
			c.Invalidate()
		}
	} else {
		a.plugGhHave, a.plugGhHaveIdx = "", 0
	}
	// the field that narrows them
	a.plugFindSmall(c, t, &a.plugGhQuery, p.GhFilter, p.GhFilterLabel)
}

// plugGhShown is what the picks and the field leave.
func (a *App) plugGhShown(tags []plugin.Tagged) []plugin.Tagged {
	out := make([]plugin.Tagged, 0, len(tags))
	for _, g := range tags {
		if a.plugGhKind != "" && plugKindOf(g.Kind) != a.plugGhKind {
			continue
		}
		if a.plugGhHave != "" {
			inst := a.plugEntryOf(g.Spec, &g) != nil
			if (a.plugGhHave == "have") != inst {
				continue
			}
		}
		if !plugGhMatch(a, g) {
			continue
		}
		out = append(out, g)
	}
	return out
}

// plugGhMatch is whether the section's field matches a repository.
func plugGhMatch(a *App, g plugin.Tagged) bool {
	f := strings.ToLower(strings.TrimSpace(a.plugGhQuery))
	if f == "" {
		return true
	}
	hay := strings.ToLower(strings.Join([]string{g.Repo, g.Owner, g.Package, g.Description, g.Spec}, " "))
	return strings.Contains(hay, f)
}

// plugGhCount is how many a pick in a strip would leave, the other strip's
// pick and the field counted in: magpie's n(k, h).
func plugGhCount(a *App, found []plugin.Tagged, kind, have string) int {
	n := 0
	for _, g := range found {
		if kind != "" && plugKindOf(g.Kind) != kind {
			continue
		}
		if have != "" {
			inst := a.plugEntryOf(g.Spec, &g) != nil
			if (have == "have") != inst {
				continue
			}
		}
		n++
	}
	return n
}

// plugKindLabel is a kind's name for the section's strips.
func plugKindLabel(a *App, kind string) string {
	p := &a.text.S.Plug
	switch plugKindOf(kind) {
	case "middleware":
		return p.KindMiddleware
	case "agent":
		return p.KindAgent
	case "tools":
		return p.KindTools
	case "page":
		return p.KindPage
	}
	return p.KindProvider
}

// plugFindSmall is a section's own narrow search field, as the head's is.
func (a *App) plugFindSmall(c *ui.Context, t *ui.Theme, value *string, placeholder, label string) {
	bg := ui.RGBA(0, 0, 0, 0.055)
	if t.Dark {
		bg = ui.RGBA(255, 255, 255, 0.075)
	}
	box := ui.Row(c).AlignItems(ui.Center).Gap(6).Padding(0, 10).
		Height(28).Width(240).Radius(8).Background(bg).Shrink(0)
	box.Children(func() {
		ui.Icon(c, searchIcon()).Size(12, 12).TextColor(t.TextMuted).Shrink(0)
		ui.TextInputBase(c, value).Grow(1).Shrink(1).
			Placeholder(placeholder).Label(label)
	})
}

// plugNpmSection is the npm search's own section: what the head's field
// found, npm asked by the page itself.
func (a *App) plugNpmSection(c *ui.Context, t *ui.Theme, listed map[string]bool, anyAbove bool) {
	p := &a.text.S.Plug
	known := map[string]bool{}
	for k := range listed {
		known[k] = true
	}
	for _, g := range a.plugTagged {
		if g.Package != "" {
			known[g.Package] = true
		}
	}
	ui.Column(c).FillWidth().Gap(9).Children(func() {
		a.plugKicker(c, t, p.OnNPM, p.OnNPMHint)
		switch {
		case a.plugSearchBusy:
			ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
				ui.Spinner(c).Size(14, 14)
				ui.Text(c, a.text.S.Plug.Checking).FontSize(12).TextColor(t.TextMuted)
			})
		case a.plugSearchErr != "":
			ui.Text(c, a.plugSearchErr).FontSize(12).TextColor(t.Danger).Wrap()
		case len(strings.TrimSpace(a.plugQuery)) < 2:
			ui.Text(c, p.SearchNeedsTwo).FontSize(12).TextColor(t.TextMuted).Wrap()
		default:
			var hits []plugin.Hit
			for _, h := range a.plugSearch {
				if !known[h.Package] {
					hits = append(hits, h)
				}
			}
			if len(hits) == 0 {
				note := p.NothingElseNpm
				if !anyAbove {
					note = plugText(p.NoPluginCalled, "q", strings.TrimSpace(a.plugQuery))
				}
				ui.Text(c, note).FontSize(12).TextColor(t.TextMuted).Wrap()
				return
			}
			sort.SliceStable(hits, func(i, j int) bool { return hits[i].Package < hits[j].Package })
			ui.Row(c).Wrap().Gap(10).FillWidth().Children(func() {
				for _, h := range hits {
					n := h.NPM
					cd := plugCard{
						Package: h.Package,
						Name:    h.Package,
						Icon:    n.Icon,
						Summary: n.Description,
						NPM:     &n,
					}
					a.plugCard(c, t, cd)
				}
			})
		}
	})
}

// plugManualForm is the form for a plugin the market doesn't list: an npm
// package, a repository, or a folder on this computer.
func (a *App) plugManualForm(c *ui.Context, t *ui.Theme) {
	p := &a.text.S.Plug
	busy := a.plugBusy != ""
	ui.Row(c).AlignItems(ui.Center).Gap(8).FillWidth().Padding(10, 12).
		Radius(11).Border(1, plugLineC(t)).Background(plugCardBg(t)).Children(func() {
		ui.Text(c, p.HaveOne).FontSize(12).TextColor(t.TextMuted).Shrink(0)
		field := ui.TextInput(c, &a.plugManual).Grow(1).Shrink(1).
			Placeholder(p.ManualPlaceholder).Label(p.HaveOne)
		if busy {
			field.Disabled(true)
		}
		if b := a.plugIconAct(c, t, folderIcon(), p.ChooseFolderTip, !busy); b.Clicked() {
			a.plugChooseFolder()
		}
		label := p.Install
		if strings.HasPrefix(a.plugBusy, "add:") {
			label = p.Installing
		}
		// enabled while nothing is being installed, as magpie's is: an empty
		// field's press does nothing
		if b := a.plugAct(c, t, label, "get", !busy); b.Clicked() {
			a.plugAdd(a.plugManual)
		}
	})
}

// plugIconAct is a square icon button, for the folder picker.
func (a *App) plugIconAct(c *ui.Context, t *ui.Theme, icon *ui.SVG, tip string, enabled bool) ui.Element {
	b := ui.Row(c).Center().Width(26).Height(26).Radius(7).Shrink(0).
		Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).Tooltip(tip).
		Label(a.text.S.Plug.ChooseFolder)
	if !enabled {
		b.Disabled(true).Opacity(0.5)
	}
	if b.Hovered() && enabled {
		b.Background(a.skHoverBg(t))
	}
	b.Children(func() {
		ui.Icon(c, icon).Size(13, 13).TextColor(t.TextMuted)
	})
	return b
}

// plugCount is a number as the page says it: 12.3k, 4.5M.
func plugCount(n int) string {
	switch {
	case n >= 1000000:
		if n >= 10000000 {
			return fmt.Sprintf("%.0fM", float64(n)/1e6)
		}
		return fmt.Sprintf("%.1fM", float64(n)/1e6)
	case n >= 1000:
		if n >= 10000 {
			return fmt.Sprintf("%.0fk", float64(n)/1e3)
		}
		return fmt.Sprintf("%.1fk", float64(n)/1e3)
	}
	return fmt.Sprint(n)
}

// plugChooseFolder opens the system's folder picker, as magpie's own
// `plugins/choose` does, and what it answers goes in the field.
func (a *App) plugChooseFolder() {
	if a.win == nil {
		return
	}
	go func() {
		dirs, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
			Parent:    a.win,
			Title:     a.text.S.Plug.ManualPlaceholder,
			Directory: true,
			Multiple:  false,
		})
		run := func() {
			if err == nil && len(dirs) > 0 {
				a.plugManual = dirs[0]
			}
			a.invalidate()
		}
		a.win.Update(run)
	}()
}
