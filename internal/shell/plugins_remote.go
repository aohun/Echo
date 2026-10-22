//go:build nogui && echo

package shell

// The Plugins page's remote read (docs/08 M4): what is installed, what
// npm said of it last, the movable built-ins and the mirror switch from
// a magpie web instance's /api/plugins, and the updates waiting from its
// /api/plugins/updates — the very documents magpie's own page drives
// (internal/gui/plugins.go), decoded straight in.
//
// The page's writes ride the same seam (docs/08 M4): add, remove, off,
// upgrade, update, options, the check for updates and the mirror switch
// ask the instance first — its /api/plugins/{op}, /api/plugins/check and
// /api/plugins/mirror — and a move asks its /api/provider/move; any
// remote failure falls back to the embedded write. An add, upgrade or
// update on the instance runs the instance's own hand-over, so the
// embedded one is skipped in remote mode (the call sites pass handOver
// only when the write stays here).
//
// What stays embedded even in remote mode:
//   - Built-ins: the Built-in tab is Echo's own runtime
//     (internal/pluginrt) — this process's parts of the shell; no
//     magpie endpoint serves them, so their cut keeps the embedded read
//     and the tab draws the same either way.
//   - A row's subscriptions: the document carries only their names
//     (providers), not which are signed in nor the ids the sign-in goes
//     by — the row draws without its subscription line, and the sign-in
//     (the instance's /api/plugin-signin) stays with the embedded rows
//     until that cut is mapped.
//   - A git plugin's commit and head, a row's gateway request hook and
//     its state, its agent, its tools, and the panels, actions and page
//     its package adds: none ride the document — the host's diagnostics
//     are the serving process's, and a plugin's own page is magpie
//     browser-mode's feature; the shell never hosts plugin HTML.
//   - The Discover tab's network reads (the market, npm's answers, the
//     search, a package's page) and the folder picker: the shell asks
//     npm and GitHub itself and opens its own folder dialog, as it does
//     embedded; only the installed half of the page goes remote.
//   - Prefer (plugin.Prefer): the subscription a row hands to another of
//     its plugins has no endpoint in magpie's surface, so it stays an
//     embedded write.

import (
	"context"
	"time"

	"github.com/yetone/magpie/internal/magpieclient"
	"github.com/yetone/magpie/internal/middleware"
	"github.com/yetone/magpie/internal/plugin"
	"github.com/yetone/magpie/internal/provider"
)

// pluginEntryRemote is one installed plugin as /api/plugins answers it:
// magpie's pluginEntryJSON, its Entry carried as it is. Only the cuts
// the row draws are named; the rest is left out.
type pluginEntryRemote struct {
	plugin.Entry
	Error     string   `json:"error"`
	Providers []string `json:"providers"`
	Version   string   `json:"version"`
	Latest    string   `json:"latest"`
	Moved     []string `json:"moved"`
	// AutoUpdated is the update magpie made to it by itself lately.
	AutoUpdated *plugin.Updated `json:"autoUpdated"`
	// Middleware is the gateway middleware it has, and how it is doing.
	Middleware *middleware.State `json:"middleware"`
	// OptionsExample is what its package suggests for its options.
	OptionsExample map[string]any `json:"optionsExample"`
	IsMiddleware   bool           `json:"isMiddleware"`
	MiddlewareOnly bool           `json:"middlewareOnly"`
}

func readPluginsRemote(c *magpieclient.Client) (PluginsData, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	var doc struct {
		Plugins []pluginEntryRemote      `json:"plugins"`
		Bun     bool                     `json:"bun"`
		BunVer  string                   `json:"bunVersion"`
		Error   string                   `json:"error"`
		Movable []provider.MoveCandidate `json:"movable"`
		Mirror  bool                     `json:"mirror"`
	}
	if err := c.Do(ctx, "GET", "/api/plugins", nil, &doc); err != nil {
		return PluginsData{}, false
	}
	var upd plugin.Updates
	if err := c.Do(ctx, "GET", "/api/plugins/updates", nil, &upd); err != nil {
		return PluginsData{}, false
	}
	d := PluginsData{
		Installed: make([]PluginEntryRow, 0, len(doc.Plugins)),
		Bun:       doc.Bun,
		BunVer:    doc.BunVer,
		Mirror:    doc.Mirror,
		Movable:   doc.Movable,
		On:        map[string]bool{},
		OffPages:  map[string]bool{},
	}
	for _, e := range doc.Plugins {
		// the row's name and package are the spec's, as the embedded
		// read's are; the document's package adds nothing
		row := PluginEntryRow{
			Spec:           e.Spec,
			Package:        plugin.Name(e.Spec),
			Name:           plugin.Name(e.Spec),
			Version:        e.Version,
			Latest:         e.Latest,
			Off:            e.Off,
			Error:          e.Error,
			Options:        e.Options,
			OptionsExample: e.OptionsExample,
			Moved:          e.Moved,
			AutoUpdated:    e.AutoUpdated,
			Middleware:     e.Middleware,
			IsMiddleware:   e.IsMiddleware,
			MiddlewareOnly: e.MiddlewareOnly,
		}
		if row.Moved == nil {
			row.Moved = []string{}
		}
		d.Installed = append(d.Installed, row)
	}
	d.Updates = len(upd.Waiting)
	for _, w := range upd.Waiting {
		d.UpdateNames = append(d.UpdateNames, w.Package)
	}
	// the built-ins are this shell's own runtime; their cut keeps the
	// embedded read (see the header)
	builtins, broken := pluginBuiltins()
	d.Builtins, d.Broken = builtins, broken
	d.On, d.OffPages = pluginsBuiltinCut(builtins)
	return d, true
}

// pluginsBuiltinCut is the built-ins cut the navigation reads: which are
// on, by id, and the window's parts of the ones that are off.
func pluginsBuiltinCut(rows []BuiltinRow) (on map[string]bool, offPages map[string]bool) {
	on = make(map[string]bool, len(rows))
	offPages = map[string]bool{}
	for _, b := range rows {
		on[b.ID] = b.On
		if b.On {
			continue
		}
		for _, pt := range b.Parts {
			if pt.Slot == "nav.page" {
				offPages[pt.Key] = true
			}
		}
	}
	return on, offPages
}

// plugAddSeam installs spec: the instance's add when the environment
// points at a magpie web instance — which runs its own hand-over after —
// else this process's.
func plugAddSeam(ctx context.Context, spec string) error {
	if c := remoteGateway(); c != nil {
		if err := c.Do(ctx, "POST", "/api/plugins/add", map[string]string{"spec": spec}, nil); err == nil {
			return nil
		}
	}
	_, err := plugAddPlugin(ctx, spec)
	return err
}

// plugRemoveSeam removes spec, as the row's confirmed remove does.
func plugRemoveSeam(ctx context.Context, spec string) error {
	if c := remoteGateway(); c != nil {
		if err := c.Do(ctx, "POST", "/api/plugins/remove", map[string]string{"spec": spec}, nil); err == nil {
			return nil
		}
	}
	return plugRemovePlugin(ctx, spec)
}

// plugSetOffSeam turns spec off or on, as its row's switch does.
func plugSetOffSeam(ctx context.Context, spec string, off bool) error {
	if c := remoteGateway(); c != nil {
		if err := c.Do(ctx, "POST", "/api/plugins/off", map[string]any{"spec": spec, "off": off}, nil); err == nil {
			return nil
		}
	}
	return plugSetPluginOff(ctx, spec, off)
}

// plugUpgradeSeam installs spec's newest version, as its row's Update
// does; the instance is asked by the package's name, as its own
// upgrade resolves the spec again.
func plugUpgradeSeam(ctx context.Context, name string) error {
	if c := remoteGateway(); c != nil {
		if err := c.Do(ctx, "POST", "/api/plugins/upgrade", map[string]string{"spec": name}, nil); err == nil {
			return nil
		}
	}
	return plugUpgradePlugin(ctx, name)
}

// plugUpdateSeam updates every plugin with an update waiting, as the
// bar's Update all does.
func plugUpdateSeam(ctx context.Context) error {
	if c := remoteGateway(); c != nil {
		if err := c.Do(ctx, "POST", "/api/plugins/update", nil, nil); err == nil {
			return nil
		}
	}
	return plugUpdatePlugins(ctx)
}

// plugSetOptionsSeam saves a middleware's options, or clears them with
// nil, as the options editor's save does.
func plugSetOptionsSeam(ctx context.Context, spec string, opts map[string]any) error {
	if c := remoteGateway(); c != nil {
		if err := c.Do(ctx, "POST", "/api/plugins/options", map[string]any{"spec": spec, "options": opts}, nil); err == nil {
			return nil
		}
	}
	return plugSetOptions(spec, opts)
}

// plugMoveSeam moves a built-in subscription onto its plugin, as the
// market card's Move and the row's do — the instance's
// /api/provider/move, the one move endpoint magpie's surface has.
func plugMoveSeam(ctx context.Context, id string) error {
	if c := remoteGateway(); c != nil {
		if err := c.Do(ctx, "POST", "/api/provider/move", map[string]string{"id": id}, nil); err == nil {
			return nil
		}
	}
	return plugMoveProvider(ctx, id)
}

// plugCheckSeam asks npm now for each installed plugin's newest version:
// the instance's check, whose answer carries the checks and its own
// refreshed state (which Echo's page takes on its next read); else this
// process's, which asks npm itself.
func plugCheckSeam(ctx context.Context) plugin.VersionChecks {
	if c := remoteGateway(); c != nil {
		var ck plugin.VersionChecks
		if err := c.Do(ctx, "POST", "/api/plugins/check", nil, &ck); err == nil {
			return ck
		}
	}
	return plugCheckUpdates(ctx)
}

// plugMirrorSeam turns the 「国内镜像」 switch on or off, as the page's
// switch does; the embedded write is this process's settings, the
// remote one the instance's.
func plugMirrorSeam(on bool) error {
	if c := remoteGateway(); c != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := c.Do(ctx, "POST", "/api/plugins/mirror", map[string]bool{"on": on}, nil); err == nil {
			return nil
		}
	}
	return setChinaMirror(on)
}
