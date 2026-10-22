//go:build nogui && echo

package shell

import (
	"context"
	"strings"
	"time"

	"github.com/yetone/magpie/internal/agent"
	"github.com/yetone/magpie/internal/middleware"
	"github.com/yetone/magpie/internal/plugin"
	"github.com/yetone/magpie/internal/pluginrt"
	"github.com/yetone/magpie/internal/provider"
	"github.com/yetone/magpie/internal/settings"
)

// The Plugins page is magpie's own (internal/gui/assets/plugins.js): three
// tabs over one page. Built-in lists the parts of magpie the runtime can
// turn off (internal/pluginrt, plugins/), each with a switch; Discover is
// the market of OpenCode provider plugins — subscriptions and middleware
// someone else wrote — and what npm says of them; Installed is what was
// added, with the ways to sign in, update, switch off or remove it.
//
// This file is the page's read half: the values the view draws, taken from
// magpie's domain packages once, off the main thread, so a frame never
// touches the filesystem or the network.

// PluginTab is one of the page's three tabs, in magpie's order.
type PluginTab int

const (
	PlugTabBuiltin PluginTab = iota
	PlugTabDiscover
	PlugTabInstalled
)

// PluginsData is the Plugins page as the view draws it.
type PluginsData struct {
	// Builtins are magpie's built-in plugins, each in the state the
	// runtime has it now (builtins.go).
	Builtins []BuiltinRow
	// Installed are the plugins added from the market, as their rows say
	// them (plugins.js's installedRow).
	Installed []PluginEntryRow
	// Movable are the built-ins with accounts a plugin could run instead,
	// which a card and a row offer to move (provider.MoveCandidates).
	Movable []provider.MoveCandidate
	// Broken is a plugins.yml that can't be read: every built-in then
	// stays as it was, and the page says so instead of offering switches
	// that would not be saved.
	Broken string
	// Bun is whether the runtime plugins download is here, and BunVer
	// the version in use: the page says what they run on.
	Bun    bool
	BunVer string
	// Mirror is the 「国内镜像」 switch (settings.ChinaMirror).
	Mirror bool
	// On is each built-in's active state, by id, which the navigation
	// reads to leave out a page whose plugin is off.
	On map[string]bool
	// Updates is how many plugins have a newer version waiting, and
	// UpdateNames what they are called: the sidebar's entry says "n
	// updates" in its count's place, as magpie's own does
	// (renderPluginDot), and Installed names them above its list.
	Updates     int
	UpdateNames []string
	// OffPages are the window's parts of the built-ins that are off, by
	// the part's key: a page of the navigation (nav.page), a settings
	// section, or a part of a page — magpie's featureHides, which leaves
	// what a switched-off plugin was out of the window at once.
	OffPages map[string]bool
}

// BuiltinRow is one of magpie's built-in plugins as Plugins › Built-in
// draws it: its name and what it is (English, translated by the page), the
// switch's state, and what keeps it from being turned off.
type BuiltinRow struct {
	ID, Name, Sub, One, Many string
	// On is whether it runs now — active, not merely chosen: a plugin
	// waiting for a service or failed is shown off, as magpie shows it.
	On bool
	// Chosen is the user's own say in plugins.yml, before what is in use.
	Chosen bool
	Source pluginrt.Source
	State  pluginrt.State
	// Why says what kept it from Active: the service it waits for, or the
	// error it failed with.
	Why string
	// InUse counts what the user has made with it; one in use stays on.
	InUse int
	// Parts are the window's parts it is, by slot (nav.page, part, …).
	Parts []pluginrt.Part
	// Schema and Config are its settings, if it has any: the page draws
	// their form under its row, and each change is saved at once.
	Schema *pluginrt.Schema
	Config map[string]any
}

// PluginClashRow is a subscription two installed plugins sign in to, as the
// row says it: the provider's name, which plugin runs it (By), and the other
// plugin's name. magpie's pluginClashJSON, over plugin.Clash.
type PluginClashRow struct {
	plugin.Clash
	Name   string // the provider's own name
	Others string // the other plugin's name
}

// PluginSubRow is one subscription a plugin signs in to, as its row says
// it: the provider's name and whether an account is signed in.
type PluginSubRow struct {
	ID       string // magpie's, which the add sheet and sign-in go by
	Pid      string // OpenCode's, which the plugin knows it by
	Name     string
	Icon     string
	Methods  []plugin.Method
	SignedIn bool
}

// PluginEntryRow is one plugin installed from the market, as its row in
// Installed draws it.
type PluginEntryRow struct {
	// Spec is how it was added (an npm name, github:owner/repo, or a
	// folder); Package is the name its package.json gives it, when it is
	// a git one; Name is what the market calls it, else the package's or
	// the folder's.
	Spec    string
	Package string
	Name    string
	// Version is what is installed; Latest is npm's newest, as npm said
	// it last; Commit and Head are a git plugin's installed commit and
	// the one its repository is at now.
	Version, Latest string
	Commit, Head    string
	// Off is a plugin the user turned off.
	Off bool
	// Error is why it didn't load, when it didn't.
	Error string
	// Providers are the subscriptions it signs in to; Named says whether
	// that is the plugins' own answer or a list kept from before they
	// were last asked (an empty list only means "none" in the first case).
	Providers []PluginSubRow
	Named     bool
	// Moved are the built-in subscriptions moved onto it, which stay
	// moved when it is turned off or removed: the built-ins are retired,
	// so there is nothing to go back to.
	Moved []string
	// Middleware is the gateway middleware it has, and how it is doing.
	Middleware *middleware.State
	// Gateway is the asynchronous request hook it runs, and how it is
	// doing; IsGateway and GatewayOnly say whether its package has one,
	// and whether that is all it has.
	Gateway     *plugin.GatewayState
	IsGateway   bool
	GatewayOnly bool
	// IsMiddleware and MiddlewareOnly say whether its package has gateway
	// middleware, and whether that is all it has; Options are what it runs
	// with, OptionsExample what its package suggests.
	IsMiddleware   bool
	MiddlewareOnly bool
	Options        map[string]any
	OptionsExample map[string]any
	// IsAgent and HasPage say whether its package adds an agent or a page;
	// InMagpieOnly is whether all it has runs in Echo itself.
	IsAgent bool
	// AgentName is the agent it adds, and AgentErr why that agent didn't
	// load, when it didn't.
	AgentName string
	AgentErr  string
	HasPage   bool
	PageTitle string
	PageReads []string
	// PageReplaces is the built-in page whose place its page takes while
	// the plugin is on (magpie.page.replaces), or "".
	PageReplaces string
	// Panels are the slots of Echo's own pages it adds panels to
	// (magpie.panels), PanelReads what they may read of Echo's, and
	// PanelTitles their own titles, one per line, for the tooltip.
	Panels      []string
	PanelReads  []string
	PanelTitles string
	// Actions are the menus its page adds items to (magpie.actions), and
	// ActionTitles their own titles, one per line.
	Actions      []string
	ActionTitles string
	InMagpieOnly bool
	// Tools are the tools it gives, by the names magpie noted last.
	Tools []plugin.Tool
	// AutoUpdated is an update magpie made to it by itself lately.
	AutoUpdated *plugin.Updated
}

// pluginEntries is the installed list as it is read for the page.
func pluginEntries() []PluginEntryRow {
	l := plugin.Load()
	if len(l.Plugins) == 0 {
		return nil
	}
	// each plugin's subscriptions, by the names it gave last (Cached keeps
	// them without starting the host or asking anyone)
	subs := map[string][]PluginSubRow{}
	for _, p := range plugin.Cached() {
		subs[p.Spec] = append(subs[p.Spec], PluginSubRow{
			ID:       provider.PluginID(p.ID),
			Pid:      p.ID,
			Name:     p.Name,
			Icon:     p.Icon,
			Methods:  p.Methods,
			SignedIn: p.SignedIn,
		})
	}
	// npm's newest, as it said last: asking it again is a check for updates
	names := make([]string, 0, len(l.Plugins))
	for _, e := range l.Plugins {
		if plugin.IsPath(e.Spec) || plugin.IsGit(e.Spec) {
			continue
		}
		names = append(names, plugin.Name(e.Spec))
	}
	known := plugin.InfoCached(names)
	mws := middleware.States()
	// only an existing plugin host is asked, so a page read never starts
	// one just to show its diagnostics (plugin.GatewayStates)
	gws := plugin.GatewayStates(context.Background())
	answered := plugin.Answered()
	toolNames := plugin.ToolNames()
	var agentErrs map[string]string
	for _, e := range l.Plugins {
		if f, _ := plugin.Agent(plugin.Target(e.Spec)); f != "" && !e.Off {
			agentErrs = agent.PluginErrors()
			break
		}
	}

	out := make([]PluginEntryRow, 0, len(l.Plugins))
	for _, e := range l.Plugins {
		row := PluginEntryRow{
			Spec:      e.Spec,
			Package:   plugin.Name(e.Spec),
			Name:      plugin.Name(e.Spec),
			Version:   plugin.Installed(e.Spec),
			Off:       e.Off,
			Options:   e.Options,
			Providers: subs[e.Spec],
			Named:     answered,
			Moved:     provider.MovedOnto(e.Spec),
		}
		if row.Providers == nil {
			row.Providers = []PluginSubRow{}
		}
		if row.Moved == nil {
			row.Moved = []string{}
		}
		if !plugin.IsPath(e.Spec) && !plugin.IsGit(e.Spec) {
			row.Latest = known[plugin.Name(e.Spec)].Version
		}
		if plugin.IsGit(e.Spec) {
			row.Package = plugin.Name(e.Spec)
			row.Commit = plugin.ShortCommit(plugin.GitCommit(e.Spec))
			row.Head = plugin.ShortCommit(plugin.GitHeadCached(e.Spec))
		}
		if u, ok := plugin.LastUpdated(plugin.Name(e.Spec), time.Now().Add(-72*time.Hour)); ok && !plugin.IsPath(e.Spec) {
			row.AutoUpdated = &u
		}
		if m, ok := mws[e.Spec]; ok {
			row.Middleware = &m
		}
		if file, only := plugin.Middleware(plugin.Target(e.Spec)); file != "" {
			row.IsMiddleware, row.MiddlewareOnly = true, only
			row.OptionsExample = plugin.OptionsExample(plugin.Target(e.Spec))
		}
		if file := plugin.Gateway(plugin.Target(e.Spec)); file != "" {
			row.IsGateway = true
			row.GatewayOnly = plugin.GatewayOnly(plugin.Target(e.Spec))
			row.OptionsExample = plugin.OptionsExample(plugin.Target(e.Spec))
			if !e.Off {
				st := plugin.GatewayState{LastOutcome: "idle"}
				if found, ok := gws[e.Spec]; ok {
					st = found
				}
				row.Gateway = &st
			}
		}
		if file, _ := plugin.Agent(plugin.Target(e.Spec)); file != "" {
			row.IsAgent = true
			row.InMagpieOnly = plugin.InMagpieOnly(plugin.Target(e.Spec))
			if !e.Off {
				row.AgentErr = agentErrs[e.Spec]
				for _, ag := range agent.All() {
					if ag.Plugin == e.Spec {
						row.AgentName = ag.Name
						break
					}
				}
			}
		}
		if w, _ := plugin.Window(plugin.Target(e.Spec)); w != nil {
			if w.Page != nil {
				row.HasPage = true
				row.PageTitle = w.Page.Title
				// the reads its page and its panels share
				row.PageReads = w.Reads
				row.PageReplaces = w.Page.Replaces
			}
			row.InMagpieOnly = plugin.InMagpieOnly(plugin.Target(e.Spec))
			for _, pn := range w.Panels {
				row.Panels = append(row.Panels, pn.Slot)
				if t := strings.TrimSpace(pn.Title); t != "" {
					if row.PanelTitles != "" {
						row.PanelTitles += "\n"
					}
					row.PanelTitles += t
				}
			}
			if len(w.Panels) > 0 {
				row.PanelReads = w.Reads
			}
			for _, ac := range w.Actions {
				row.Actions = append(row.Actions, ac.Menu)
				if t := strings.TrimSpace(ac.Title); t != "" {
					if row.ActionTitles != "" {
						row.ActionTitles += "\n"
					}
					row.ActionTitles += t
				}
			}
		}
		// a plugin of tools alone signs in to nothing: its row says the
		// names it noted last, which the host needn't run to give
		for _, n := range toolNames[e.Spec] {
			row.Tools = append(row.Tools, plugin.Tool{Name: n})
		}
		out = append(out, row)
	}
	return out
}

// pluginBuiltins reads every built-in plugin as the runtime has it now,
// bringing the runtime to plugins.yml first — the read magpie's own page
// makes (builtins.go's builtinsState), so the switches show what would be
// saved and what runs.
func pluginBuiltins() (rows []BuiltinRow, broken string) {
	cs, err := pluginrt.Refresh()
	if err != nil {
		broken = err.Error()
	}
	byID := make(map[string]pluginrt.Composed, len(cs))
	for _, c := range cs {
		byID[c.ID] = c
	}
	statuses := map[string]pluginrt.Status{}
	if rt := pluginrt.Running(); rt != nil {
		for _, s := range rt.Statuses() {
			statuses[s.ID] = s
		}
	}
	ms := pluginrt.Registered()
	rows = make([]BuiltinRow, 0, len(ms))
	for _, m := range ms {
		c := byID[m.ID]
		s := statuses[m.ID]
		rows = append(rows, BuiltinRow{
			ID: m.ID, Name: m.Name, Sub: m.Sub, One: m.One, Many: m.Many,
			On:     s.State == pluginrt.Active,
			Chosen: c.Chosen,
			Source: c.Source,
			State:  s.State,
			Why:    s.Why,
			InUse:  c.InUse,
			Parts:  m.Parts,
			Schema: m.Schema(),
			Config: c.Config,
		})
	}
	return rows, broken
}

// readPluginsData reads the page's local half: the built-ins and their
// runtime state, and what is installed. The market's half — the plugins
// magpie suggests, and what npm says of them — is asked separately, off the
// page's first paint (plugins_market.go), because it downloads.
func readPluginsData() PluginsData {
	// The experiment's seam (docs/08 M4): the page's installed half from
	// a magpie web instance when the environment points at one; the
	// built-ins stay embedded either way (plugins_remote.go's header).
	if c := remoteGateway(); c != nil {
		if d, ok := readPluginsRemote(c); ok {
			return d
		}
		// The instance went away; fall through to the embedded read.
	}
	builtins, broken := pluginBuiltins()
	d := PluginsData{
		Builtins: builtins,
		Broken:   broken,
		Bun:      plugin.HasBun(),
		BunVer:   plugin.BunInUse(),
		Mirror:   settings.Load().ChinaMirror,
		On:       make(map[string]bool, len(builtins)),
		Movable:  provider.MoveCandidates(),
		OffPages: map[string]bool{},
	}
	for _, b := range builtins {
		d.On[b.ID] = b.On
	}
	for _, b := range builtins {
		if b.On {
			continue
		}
		for _, pt := range b.Parts {
			if pt.Slot == "nav.page" {
				d.OffPages[pt.Key] = true
			}
		}
	}
	d.Installed = pluginEntries()
	// the updates waiting for the reader, which put the count on the
	// sidebar's entry and the bar above Installed (magpie's renderPluginDot
	// over /api/plugins/updates)
	pend := plugin.PendingUpdates()
	d.Updates = len(pend.Waiting)
	for _, w := range pend.Waiting {
		d.UpdateNames = append(d.UpdateNames, w.Package)
	}
	return d
}

// setChinaMirror turns the 「国内镜像」 switch on or off; a plugin list
// fetched from GitHub's slow address, or not at all, is asked again — the
// same write magpie's own page makes (internal/gui/plugins.go).
func setChinaMirror(on bool) error {
	s := settings.Load()
	if s.ChinaMirror == on {
		return nil
	}
	s.ChinaMirror = on
	if err := settings.Save(s); err != nil {
		return err
	}
	if on {
		plugin.RefreshMarket()
	}
	return nil
}
