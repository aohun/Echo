//go:build nogui && echo

package shell

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/yetone/magpie/internal/plugin"
	"github.com/yetone/magpie/internal/provider"
)

// The page's writings and its asks of the network: the market (the plugins
// magpie suggests, what npm says of each, and the repositories tagged on
// GitHub), the npm search, a check for updates, and the writes themselves —
// add, remove, update, upgrade, on or off, options — which run magpie's own
// functions (internal/gui/plugins.go's /api/plugins/{op}) off the main
// thread, as magpie's page runs them off its own.
//
// The functions the page calls are variables so that a test can stand in
// for the network, as internal/gui's are (its addPlugin, removePlugin, …).

var (
	plugMarketFetch   = plugin.Market
	plugInfoFetch     = plugin.Info
	plugTaggedFetch   = plugin.TaggedRepos
	plugSearchNPM     = plugin.Search
	plugReadmeFetch   = plugin.Readme
	plugGitHeadsFetch = plugin.GitHeads

	plugAddPlugin     = provider.AddPlugin
	plugRemovePlugin  = provider.RemovePlugin
	plugSetPluginOff  = provider.SetPluginOff
	plugUpdatePlugins = plugin.Update
	plugUpgradePlugin = plugin.Upgrade
	plugCheckUpdates  = plugin.CheckNow
	plugSetOptions    = plugin.SetOptions
	plugPreferPlugin  = plugin.Prefer
	plugMoveProvider  = provider.Move
	plugHandOver      = provider.HandOver
)

// plugHostKick asks the plugin host which plugins sign in to what, once,
// when the page first shows: two plugins signing in to one subscription is
// the host's own answer (plugin.Clashes over what it loaded), and it is
// asked the way magpie's page asks it — off the page, so the list draws
// without waiting. A host that isn't up and can't be (no Bun) is left
// alone: there is then nothing installed to sign in anywhere.
//
// The host's state is read inside the goroutine, never on the main
// thread: plugin.Running takes the host's lock, and the host's start
// holds it across the whole start — a page drawn while the host comes up
// would freeze the window for its length.
func (a *App) plugHostKick() {
	if a.plugHostDone || a.plugHostBusy {
		return
	}
	a.plugHostBusy = true
	a.invalidate()
	go func() {
		if !plugin.Running() && !plugin.HasBun() {
			a.plugHostState(func() {
				a.plugHostBusy = false
				a.plugHostDone = true
			})
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		loaded, err := plugin.Plugins(ctx)
		a.plugHostState(func() {
			a.plugHostBusy = false
			if err != nil {
				return
			}
			a.plugHostDone = true
			a.plugClashes = a.plugClashRows(loaded)
			a.invalidate()
		})
	}()
}

// plugHostState runs fn on the main thread, the goroutine's way back into
// the page's state.
func (a *App) plugHostState(fn func()) {
	if a.win == nil {
		fn()
		return
	}
	a.win.Update(fn)
}

// plugClashRows is the host's answer as the rows say it: each clash's
// subscription named as its provider names it, and the other plugin named
// as its row does (magpie's nameOfSpec).
func (a *App) plugClashRows(loaded []plugin.Loaded) map[string][]PluginClashRow {
	idName := map[string]string{}
	for _, p := range plugin.Cached() {
		idName[p.ID] = p.Name
	}
	specName := map[string]string{}
	for _, e := range a.data.Plugins.Installed {
		specName[e.Spec] = a.plugShownName(e)
	}
	out := map[string][]PluginClashRow{}
	for spec, cs := range plugin.Clashes(loaded) {
		for _, cl := range cs {
			row := PluginClashRow{Clash: cl, Name: idName[cl.ID]}
			if row.Name == "" {
				row.Name = cl.ID
			}
			others := cl.With
			if cl.By != spec {
				others = []string{cl.By}
			}
			names := make([]string, 0, len(others))
			for _, o := range others {
				if n := specName[o]; n != "" {
					names = append(names, n)
				} else {
					names = append(names, plugin.Name(o))
				}
			}
			row.Others = strings.Join(names, ", ")
			out[spec] = append(out[spec], row)
		}
	}
	return out
}

// plugMarketKick asks the market once, when Discover (or a search on any
// tab) first shows. It is the page's one download: the plugins magpie
// suggests, then what npm says of every listing and every installed npm
// plugin. The repositories tagged on GitHub are asked apart
// (plugTaggedKick), because GitHub answers last and a section that isn't
// there would read as there being none.
func (a *App) plugMarketKick() {
	if a.plugMarket != nil || a.plugMarketBusy {
		return
	}
	a.plugMarketBusy = true
	a.invalidate()
	installed := make([]string, 0, 8)
	for _, e := range a.data.Plugins.Installed {
		if !plugin.IsPath(e.Spec) && !plugin.IsGit(e.Spec) {
			installed = append(installed, plugin.Name(e.Spec))
		}
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		listings := plugMarketFetch(ctx)
		names := make([]string, 0, len(listings)+len(installed))
		for _, l := range listings {
			names = append(names, l.Package)
		}
		names = append(names, installed...)
		npm := plugInfoFetch(ctx, names)
		run := func() {
			a.plugMarketBusy = false
			a.plugMarketErr = ""
			a.plugMarket = &plugMarketData{
				Listings: listings,
				NPM:      npm,
				At:       time.Now(),
			}
		}
		if a.win == nil {
			run()
			return
		}
		a.win.Update(run)
	}()
	a.plugTaggedKick()
}

// plugTaggedKick asks GitHub for the repositories tagged magpie-plugin, and
// each installed git plugin's commit, once: the Discover tab's GitHub
// section stands in its place with a word while it waits, and says so with a
// way to ask again when it couldn't (magpie's ghWaiting and ghFailed).
func (a *App) plugTaggedKick() {
	if a.plugTaggedBusy || a.plugTaggedDone {
		return
	}
	a.plugTaggedBusy = true
	a.plugTaggedErr = ""
	a.invalidate()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		repos := plugTaggedFetch(ctx)
		heads := plugGitHeadsFetch(ctx)
		run := func() {
			a.plugTaggedBusy = false
			a.plugTagged, a.plugHeads, a.plugTaggedDone = repos, heads, true
		}
		if a.win == nil {
			run()
			return
		}
		a.win.Update(run)
	}()
}

// plugTaggedAgain asks GitHub again, as the section's Try again does.
func (a *App) plugTaggedAgain() {
	a.plugTagged, a.plugTaggedDone = nil, false
	a.plugTaggedKick()
}

// plugSearchKick asks npm for what was just typed, once typing pauses —
// magpie's own 350ms. One letter asks nothing: it would match nearly
// everything.
func (a *App) plugSearchKick() {
	if a.plugSearchTimer != nil {
		a.plugSearchTimer.Stop()
		a.plugSearchTimer = nil
	}
	q := strings.TrimSpace(a.plugQuery)
	if len(q) < 2 {
		a.plugSearch, a.plugSearchErr, a.plugSearchQ = nil, "", ""
		a.plugSearchBusy = false
		return
	}
	a.plugSearchQ = q
	a.plugSearchBusy = true
	a.plugSearchTimer = time.AfterFunc(350*time.Millisecond, func() {
		if app := Current(); app != nil {
			if app.win == nil {
				app.plugSearchNow(q)
				return
			}
			app.win.Update(func() { app.plugSearchNow(q) })
		}
	})
}

// plugSearchNow asks npm for one query.
func (a *App) plugSearchNow(q string) {
	if strings.TrimSpace(q) != a.plugSearchQ {
		return
	}
	a.plugSearchBusy = true
	a.invalidate()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		hits, err := plugSearchNPM(ctx, q)
		run := func() {
			a.plugSearchBusy = false
			if err != nil {
				a.plugSearchErr = err.Error()
				return
			}
			a.plugSearchErr = ""
			a.plugSearch = hits
		}
		if a.win == nil {
			run()
			return
		}
		a.win.Update(run)
	}()
}

// plugCheck asks npm now — not what it said within the hour — for each
// installed plugin's newest version. Nothing is installed: the rows then say
// what it found (an update, up to date, or why it couldn't be asked), and
// Update installs it as before.
func (a *App) plugCheck() {
	if a.plugChecking || a.plugBusy != "" {
		return
	}
	a.plugChecking = true
	a.plugNotice = ""
	a.invalidate()
	through := make(chan struct{})
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		c := plugCheckUpdates(ctx)
		run := func() {
			a.plugChecking = false
			a.plugChecks = &plugCheckResult{At: c.At, By: map[string]plugin.VersionCheck{}}
			for _, p := range c.Plugins {
				a.plugChecks.By[p.Spec] = p
			}
			a.plugNotice, a.plugNoticeErr = a.plugCheckWord(c.Plugins), false
			a.Reload()
			close(through)
		}
		if a.win == nil {
			run()
			return
		}
		a.win.Update(run)
	}()
	if a.win == nil {
		<-through
	}
}

// plugCheckWord is what a check found, in the page's words: how many have an
// update, or that the ones checked are all up to date — with some not
// checked, "every plugin" isn't known to be up to date (#1942).
func (a *App) plugCheckWord(ps []plugin.VersionCheck) string {
	p := &a.text.S.Plug
	asked, ups, unknown := 0, 0, 0
	var firstUnknown plugin.VersionCheck
	for _, c := range ps {
		if c.Status != "update" && c.Status != "current" && c.Status != "unknown" {
			continue
		}
		asked++
		if c.Status == "update" && !c.Off {
			ups++
		}
		if c.Status == "unknown" {
			if unknown == 0 {
				firstUnknown = c
			}
			unknown++
		}
	}
	if asked > 0 && unknown == asked {
		return plugText(p.CheckFailed, "error", a.plugWhyText(firstUnknown))
	}
	msg := p.CheckAllUp
	switch {
	case ups == 1:
		msg = p.CheckOne
	case ups > 1:
		msg = plugText(p.CheckMany, "n", plugNum(ups))
	case unknown > 0:
		msg = p.CheckNone
	}
	if unknown > 0 {
		msg += " · " + plugText(p.CheckUnknown, "n", plugNum(unknown), "error", a.plugWhyText(firstUnknown))
	}
	return msg
}

// plugWhyText is why one plugin couldn't be checked, in the page's words:
// a git one names its repository, an npm one npm's own answer.
func (a *App) plugWhyText(c plugin.VersionCheck) string {
	p := &a.text.S.Plug
	if plugin.IsGit(c.Spec) {
		answer := c.Error
		if c.Limited != nil {
			answer = a.plugLimitedText(c.Limited)
		}
		return plugText(p.GitNoAnswer, "repo", c.Spec, "error", answer)
	}
	switch c.Why {
	case "offline":
		return p.NPMOffline
	case "limited":
		return p.NPMLimited
	case "missing":
		return p.NPMMissing
	}
	return plugText(p.NPMError, "error", c.Error)
}

// plugLimitedText is GitHub's rate limit used up, in the page's words: when
// it lifts, and how a token raises it — as magpie's own skills page says it
// (#1942).
func (a *App) plugLimitedText(l *plugin.GitHubLimit) string {
	p := &a.text.S.Plug
	at := ""
	if l.Until != "" {
		if when, err := time.Parse(time.RFC3339, l.Until); err == nil && !when.IsZero() {
			at = plugClock(a.lang, when)
		}
	}
	switch {
	case l.Token && at != "":
		return plugText(p.LimitTokenUntil, "time", at)
	case l.Token:
		return p.LimitToken
	case at != "":
		return plugText(p.LimitAnonUntil, "time", at)
	}
	return p.LimitAnon
}

// plugRun runs one write, keeping the page's other buttons off it while it
// runs: magpie's page does the same with its busy set. handOver is for the
// writes after which a deprecated built-in's accounts move to their plugin;
// say is what the page says when it worked.
func (a *App) plugRun(key string, handOver bool, say string, fn func(ctx context.Context) error) {
	if a.plugBusy != "" {
		return
	}
	a.plugBusy = key
	a.plugNotice = ""
	a.invalidate()
	done := make(chan struct{})
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		err := fn(ctx)
		if err == nil && handOver {
			plugHandOver(ctx, false)
		}
		run := func() {
			a.plugBusy = ""
			if err != nil {
				a.plugNotice, a.plugNoticeErr = err.Error(), true
			} else {
				a.plugNotice, a.plugNoticeErr = say, false
			}
			a.Reload()
			close(done)
		}
		if a.win == nil {
			run()
			return
		}
		a.win.Update(run)
	}()
	// a tester has no window to post to, so the write is waited for here
	if a.win == nil {
		<-done
	}
}

// plugRunAsk is plugRun for a write whose notice depends on what it found:
// done says it, on the main thread, once fn is through.
func (a *App) plugRunAsk(key string, fn func(ctx context.Context) error, done func(err error)) {
	if a.plugBusy != "" {
		return
	}
	a.plugBusy = key
	a.plugNotice = ""
	a.invalidate()
	through := make(chan struct{})
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		err := fn(ctx)
		run := func() {
			a.plugBusy = ""
			done(err)
			a.Reload()
			close(through)
		}
		if a.win == nil {
			run()
			return
		}
		a.win.Update(run)
	}()
	if a.win == nil {
		<-through
	}
}

// plugAdd installs a plugin — an npm package, a git repository or a folder —
// and says so; a subscription among what it brings is signed in to from its
// row.
func (a *App) plugAdd(spec string) {
	p := &a.text.S.Plug
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return
	}
	say := plugText(p.InstallDone, "name", plugin.Name(spec))
	if !plugin.IsGit(spec) && !plugin.IsPath(spec) {
		say = plugText(p.InstallDoneSignIn, "name", plugin.Name(spec))
	}
	a.plugManual = ""
	a.plugRun("add:"+spec, true, say, func(ctx context.Context) error {
		_, err := plugAddPlugin(ctx, spec)
		return err
	})
}

// plugRemove removes a plugin. The built-in subscriptions moved onto it
// stay moved: the built-ins are retired, so there is nothing to go back
// to, and its accounts are kept with the plugin's sign-ins (magpie's
// remove keeps them; nothing moves back).
func (a *App) plugRemove(spec string) {
	p := &a.text.S.Plug
	a.plugRunAsk("remove:"+spec, func(ctx context.Context) error {
		return plugRemovePlugin(ctx, spec)
	}, func(err error) {
		if err != nil {
			a.plugNotice, a.plugNoticeErr = err.Error(), true
			return
		}
		a.plugNotice, a.plugNoticeErr = plugText(p.Removed, "name", plugin.Name(spec)), false
	})
}

// plugEntryFor is the installed row of a plugin, by its spec.
func (a *App) plugEntryFor(spec string) (PluginEntryRow, bool) {
	for _, e := range a.data.Plugins.Installed {
		if e.Spec == spec {
			return e, true
		}
	}
	return PluginEntryRow{}, false
}

// plugSetOff turns a plugin off or on through the provider side. A
// subscription it carries stays moved when it goes off — the built-ins
// are retired, so nothing goes back — and switching it on again offers
// the built-in a move onto it (magpie's move-again ask).
func (a *App) plugSetOff(spec string, off bool) {
	a.plugRunAsk("off:"+spec, func(ctx context.Context) error {
		return plugSetPluginOff(ctx, spec, off)
	}, func(err error) {
		if err != nil {
			a.plugNotice, a.plugNoticeErr = err.Error(), true
			return
		}
		if off {
			// nothing is said: a carried subscription stays moved, and
			// words about going back would be a lie
			return
		}
		// switched on again with a built-in it could carry: its row asks
		// (magpie's move-again ask)
		if len(a.plugMovers(spec)) > 0 {
			a.plugAskMove = spec
		}
	})
}

// plugUpgrade installs one plugin's newest version.
func (a *App) plugUpgrade(spec, latest string) {
	p := &a.text.S.Plug
	name := plugin.Name(spec)
	say := plugText(p.IsUpToDate, "name", name)
	if v := strings.TrimPrefix(strings.TrimSpace(latest), "v"); v != "" {
		say = plugText(p.UpdatedTo, "name", name, "v", v)
	}
	a.plugRun("upgrade:"+spec, true, say, func(ctx context.Context) error {
		return plugUpgradePlugin(ctx, name)
	})
}

// plugUpdateAll updates every plugin with an update waiting, as magpie's
// Update all does (plugin.Update).
func (a *App) plugUpdateAll() {
	a.plugRun("update", true, a.text.S.Plug.UpdatedAll, func(ctx context.Context) error {
		return plugUpdatePlugins(ctx)
	})
}

// plugSaveOptions saves a middleware's options, or clears them with nil.
func (a *App) plugSaveOptions(spec string, opts map[string]any) {
	p := &a.text.S.Plug
	name := plugin.Name(spec)
	say := plugText(p.OptionsSaved, "name", name)
	if opts == nil {
		say = plugText(p.OptionsCleared, "name", name)
	}
	a.plugOptions, a.plugOptsErr = "", ""
	a.plugRun("options:"+spec, false, say, func(ctx context.Context) error {
		return plugSetOptions(spec, opts)
	})
}

// plugMove moves a built-in subscription onto a plugin, as the market card's
// Move and the row's do. Its notice says what the plugin now carries — the
// accounts it moved with and the models it serves — and one that failed says
// whether every account needs signing in again (magpie's move()).
func (a *App) plugMove(id string) {
	p := &a.text.S.Plug
	// the accounts it carries, counted before the move
	accounts := 0
	for _, c := range provider.MoveCandidates() {
		if c.ID == id {
			accounts = c.Accounts
		}
	}
	name := a.plugMoverName(id)
	a.plugRunAsk("move:"+id, func(ctx context.Context) error {
		return plugMoveProvider(ctx, id)
	}, func(err error) {
		switch {
		case err == nil:
			models := 0
			for _, pr := range provider.All() {
				if pr.ID == id {
					models = len(pr.Exposed())
					break
				}
			}
			a.plugNotice = plugText(p.MoveDone, "name", name,
				"accounts", plugHowMany(p.AccOne, p.AccMany, accounts),
				"models", plugHowMany(p.ModelOne, p.ModelMany, models))
		case plugMoveWhy(err) == "lapsed":
			// nothing here to sign in from: say where
			a.plugNotice, a.plugNoticeErr = plugText(p.MoveLapsed, "name", name), true
		default:
			a.plugNotice, a.plugNoticeErr = plugText(p.MoveFailed, "error", err.Error()), true
		}
	})
}

// plugMoveWhy is the code a move failed with, when the domain gives one
// (provider.WhyOf): "lapsed" is every account needing to sign in again.
func plugMoveWhy(err error) string {
	if why := provider.WhyOf(err); why != nil {
		return why.Code
	}
	return ""
}

// plugHowMany says a number the way magpie says it, in the two forms it
// keeps for them ("{n} account", "{n} accounts").
func plugHowMany(one, many string, n int) string {
	if n == 1 {
		return plugText(one, "n", "1")
	}
	return plugText(many, "n", strconv.Itoa(n))
}

// plugPrefer makes this plugin the one that serves a subscription two of
// them sign in to, the other keeping its own (magpie's Use for,
// plugin.Prefer).
func (a *App) plugPrefer(spec, id, name string) {
	// the plugin as its row names it, not its package
	shown := plugin.Name(spec)
	if e, ok := a.plugEntryFor(spec); ok {
		shown = a.plugShownName(e)
	}
	a.plugRun("prefer:"+spec, false,
		plugText(a.text.S.Plug.PreferDone, "name", name, "plugin", shown),
		func(ctx context.Context) error {
			return plugPreferPlugin(plugin.Name(spec), id)
		})
}
