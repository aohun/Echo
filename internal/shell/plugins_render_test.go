//go:build nogui && echo

package shell

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/yetone/magpie/internal/plugin"
	"github.com/yetone/magpie/internal/provider"
	"github.com/yetone/magpie/internal/settings"
)

// The Plugins page's own checks: that each of its three tabs draws, that the
// market's cards and the GitHub section's waiting and failing states draw,
// that a plugin's own sheet draws its README as markdown, and that the page's
// words are complete in every language — all without a network: the market
// seams (plugins_market.go) are given answers here, as internal/gui's are.

// TestPluginsPageRenders draws the page's three tabs over the test home,
// where the built-ins are the real ones (plugins/) and nothing is installed.
func TestPluginsPageRenders(t *testing.T) {
	app := NewForTest()
	app.LoadNow()
	app.page = PagePlugins

	tt := ui.NewTester(app.ViewForTest(), 1180, 780)
	tt.Frame()

	p := app.text.S.Plug
	// Built-in: the tab shown first, its intro, and a row for each part
	for _, want := range []string{p.TabBuiltin, p.TabDiscover, p.BuiltinIntro} {
		if !tt.HasText(want) {
			t.Errorf("the built-in tab did not draw %q", want)
		}
	}
	if len(app.data.Plugins.Builtins) == 0 {
		t.Fatal("no built-in plugin was read: the tab drew nothing")
	}
	first := app.data.Plugins.Builtins[0]
	if !tt.HasText(app.plugName(first)) {
		t.Errorf("the built-in %q did not draw", app.plugName(first))
	}

	// Installed: nothing was added, and the tab says so
	app.plugTab = PlugTabInstalled
	tt.Frame()
	if !tt.HasText(p.NoPlugins) {
		t.Errorf("Installed with nothing added did not say so; texts were %v", tt.Texts())
	}

	// Discover: the market's listings, each a card (magpie's own take the
	// sections by kind), and GitHub still being asked for (its section
	// stands in its place while it waits)
	app.plugTab = PlugTabDiscover
	app.plugMarket = &plugMarketData{
		Listings: []plugin.Listing{{
			Package:   "@someone/planner",
			Name:      "Planner",
			Community: true,
			Summary:   map[string]string{"en": "Plans before it answers"},
		}},
		NPM: map[string]plugin.NPM{
			"@someone/planner": {Version: "2.0.0", Publisher: "someone", Weekly: 1200},
		},
	}
	app.plugTagged, app.plugTaggedDone = nil, false
	tt.Frame()
	for _, want := range []string{p.IntroTitle, p.SectSubscriptions, "Planner", p.LookingOnGitHub} {
		if !tt.HasText(want) {
			t.Errorf("Discover did not draw %q; texts were %v", want, tt.Texts())
		}
	}

	// a search: npm's own hits are cards of their own under the market's
	app.plugQuery = "plan"
	app.plugSearch = []plugin.Hit{{
		Package: "@other/thing",
		NPM:     plugin.NPM{Version: "1.2.3", Description: "Something else"},
	}}
	tt.Frame()
	if !tt.HasText(p.OnNPM) || !tt.HasText("@other/thing") {
		t.Errorf("the npm search did not draw its own hit; texts were %v", tt.Texts())
	}
	app.plugQuery, app.plugSearch = "", nil

	// GitHub answered: its card joins, with the topic it was found by
	app.plugTagged = []plugin.Tagged{{
		Spec: "github:someone/tagger", Repo: "someone/tagger", Owner: "someone",
		Description: "Tags things", Stars: 12, Package: "@someone/tagger",
	}}
	app.plugTaggedDone = true
	tt.Frame()
	if !tt.HasText("someone/tagger") {
		t.Errorf("the tagged repository did not draw; texts were %v", tt.Texts())
	}

	// GitHub couldn't be asked: the section says why and offers another go
	app.plugTagged, app.plugTaggedDone = nil, false
	app.plugTaggedErr = "GET https://api.github.com: connection refused"
	tt.Frame()
	want := plugText(p.CouldntAskGitHub, "error", app.plugTaggedErr)
	if !tt.HasText(want) {
		t.Errorf("the failed GitHub section did not say why; texts were %v", tt.Texts())
	}
	if !tt.HasText(p.TryAgain) {
		t.Errorf("the failed GitHub section offered no retry; texts were %v", tt.Texts())
	}
}

// TestPluginsInstalledRenders draws the Installed tab over a seeded plugin
// list, and the bar for the updates waiting, which the sidebar's count is
// about. The seed is plugins.json and the package under the test home
// testenv makes, so no real plugin and no network is touched.
func TestPluginsInstalledRenders(t *testing.T) {
	seedPlugin(t, "@someone/planner", "1.0.0")

	app := NewForTest()
	app.LoadNow()
	app.page = PagePlugins
	app.plugTab = PlugTabInstalled
	app.plugMarket = &plugMarketData{
		NPM: map[string]plugin.NPM{
			"@someone/planner": {Version: "2.0.0"},
		},
	}

	if len(app.data.Plugins.Installed) == 0 {
		t.Fatal("the seeded plugin was not read")
	}
	tt := ui.NewTester(app.ViewForTest(), 1180, 780)
	tt.Frame()

	p := app.text.S.Plug
	if !tt.HasText("@someone/planner") {
		t.Errorf("the installed plugin did not draw; texts were %v", tt.Texts())
	}
	// the update waiting is named above the list, with one press for all
	bar := plugText(p.UpdateOne, "name", "@someone/planner")
	if !tt.HasText(bar) {
		t.Errorf("the updates bar did not draw %q; texts were %v", bar, tt.Texts())
	}

	// the sidebar's entry says the same, in its count's place
	if !tt.HasText(p.UpdatesOne) {
		t.Errorf("the navigation did not say an update waits; texts were %v", tt.Texts())
	}
}

// seedPlugin writes one installed plugin the way an install leaves it:
// plugins.json with its spec (with options, which the row's editor reads),
// its package under node_modules with the version installed, and the update
// npm's last check found waiting — which the navigation's count reads
// (plugin.PendingUpdates over plugin-updates.json).
func seedPlugin(t *testing.T, spec, version string) {
	t.Helper()
	if err := os.MkdirAll(settings.Dir(), 0o700); err != nil {
		t.Fatal(err)
	}
	list := plugin.List{Plugins: []plugin.Entry{
		{Spec: spec, Options: map[string]any{"mode": "fast"}},
	}}
	b, err := json.Marshal(list)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(plugin.ListPath(), b, 0o600); err != nil {
		t.Fatal(err)
	}
	at := plugin.Target(spec)
	if err := os.MkdirAll(at, 0o700); err != nil {
		t.Fatal(err)
	}
	pkg := map[string]any{"name": spec, "version": version, "main": "index.js"}
	b, err = json.Marshal(pkg)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(at, "package.json"), b, 0o600); err != nil {
		t.Fatal(err)
	}
	updates := plugin.Updates{Waiting: []plugin.Waiting{{
		Spec: spec, Package: plugin.Name(spec), Version: version, Latest: "2.0.0",
	}}}
	b, err = json.Marshal(updates)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(settings.Dir(), "plugin-updates.json"), b, 0o600); err != nil {
		t.Fatal(err)
	}
}

// TestPluginsDetailSheet draws a plugin's own sheet: its facts, what it is,
// and its README as markdown — the rendering the page's own detail does
// (plugins.js's markdown), with no network: the README is the test's.
func TestPluginsDetailSheet(t *testing.T) {
	app := NewForTest()
	app.LoadNow()
	app.page = PagePlugins

	readme := strings.Join([]string{
		"# Planner",
		"",
		"Plans **before** it answers, and keeps `state` in memory.",
		"",
		"- [x] streaming",
		"- [ ] caching",
		"",
		"```js",
		"export const middleware = () => {}",
		"```",
		"",
		"| Option | Default |",
		"| --- | --- |",
		"| mode | fast |",
		"",
		"> [!NOTE]",
		"> Reads nothing of yours.",
		"",
		"---",
		"",
		"<a href=\"https://example.com/docs\">Docs</a>",
	}, "\n")

	app.plugDetail = &plugDetail{
		Spec:     "@someone/planner",
		Name:     "Planner",
		Kind:     "middleware",
		Have:     false,
		Tag:      &plugin.Tagged{Repo: "someone/planner", Owner: "someone", Stars: 42, Spec: "github:someone/planner"},
		NPM:      &plugin.NPM{Version: "2.0.0", Weekly: 1200, License: "MIT", Publisher: "someone", Description: "Plans first"},
		Replaces: "",
	}
	app.plugDetailText = readme
	app.plugDetailUpd = "Jan 2, 2026"

	tt := ui.NewTester(app.ViewForTest(), 1180, 780)
	tt.Frame()

	p := app.text.S.Plug
	for _, want := range []string{
		"Planner",
		"Plans before",                       // the bold run, on its own
		"state",                              // the inline code
		"export const middleware = () => {}", // the code block
		"streaming",                          // a list item
		"mode",                               // a table cell
		"fast",
		"NOTE: Reads nothing of yours.", // the quote's alert
		"Docs",                          // an HTML link's words
		p.Version,
		p.Stars,
		p.Close,
	} {
		if !tt.HasText(want) {
			t.Errorf("the detail sheet did not draw %q; texts were %v", want, tt.Texts())
		}
	}
}

// TestPluginsMarkdownBlocks: the README's blocks are cut where magpie's own
// renderer cuts them, so the sheet draws what the file says.
func TestPluginsMarkdownBlocks(t *testing.T) {
	src := strings.Join([]string{
		"<!-- a comment, gone -->",
		"[![badge](https://img.example/b.svg)](https://ci.example)",
		"# Title",
		"",
		"One",
		"two",
		"",
		"## Sub *line*",
		"",
		"~~~",
		"code ~~~ inner",
		"~~~",
		"",
		"1. first",
		"2. second",
		"",
		"> quoted",
		"> again",
		"",
		"| a | b |",
		"| --- | --- |",
		"| 1 | 2 |",
		"",
		"- [x] done",
		"- [ ] later",
	}, "\n")

	got := plugParseMarkdown(src)
	var kinds []mdKind
	for _, b := range got {
		kinds = append(kinds, b.kind)
	}
	want := []mdKind{mdHeading, mdParagraph, mdHeading, mdCode, mdList, mdQuote, mdTable, mdList}
	if !slices.Equal(kinds, want) {
		t.Fatalf("blocks are %v, want %v", kinds, want)
	}
	if got[0].text != "Title" || got[0].slug != "title" {
		t.Errorf("the heading is %q (%q)", got[0].text, got[0].slug)
	}
	if got[1].text != "One two" {
		t.Errorf("the paragraph's lines are not joined: %q", got[1].text)
	}
	if got[3].text != "code ~~~ inner" {
		t.Errorf("the code block is %q", got[3].text)
	}
	if len(got[4].items) != 2 || got[4].items[1].text != "second" {
		t.Errorf("the ordered list is %+v", got[4].items)
	}
	if got[5].text != "quoted again" {
		t.Errorf("the quote is %q", got[5].text)
	}
	if len(got[6].rows) != 2 || got[6].rows[1][0] != "1" {
		t.Errorf("the table is %+v", got[6].rows)
	}
	if len(got[7].items) != 2 || got[7].items[0].text != "✓ done" || got[7].items[1].text != "☐ later" {
		t.Errorf("the task list is %+v", got[7].items)
	}
}

// TestPluginsClashAndPrefer: a subscription two installed plugins sign in to
// draws the host's answer on the row that doesn't serve it, and its own
// button hands the subscription to it (magpie's clashLine and Use for).
func TestPluginsClashAndPrefer(t *testing.T) {
	spec := "@someone/planner"
	seedPlugin(t, spec, "1.0.0")

	app := NewForTest()
	app.LoadNow()
	app.page, app.plugTab = PagePlugins, PlugTabInstalled
	app.plugClashes = map[string][]PluginClashRow{spec: {{
		Clash:  plugin.Clash{ID: "acme", By: "@other/runner", With: []string{spec}},
		Name:   "Acme",
		Others: "@other/runner",
	}}}

	tt := ui.NewTester(app.ViewForTest(), 1180, 780)
	tt.Frame()
	p := app.text.S.Plug
	if !tt.HasText(plugText(p.ClashServed, "name", "Acme", "other", "@other/runner")) {
		t.Errorf("the clash was not said; texts were %v", tt.Texts())
	}
	use := plugText(p.UseFor, "name", "Acme")
	if !tt.HasText(use) {
		t.Fatalf("the row offered no way to take it over; texts were %v", tt.Texts())
	}
	// the press is the domain's own write, with the plugin the press names
	got := ""
	old := plugPreferPlugin
	plugPreferPlugin = func(name, id string) error {
		got = name + " <- " + id
		return nil
	}
	t.Cleanup(func() { plugPreferPlugin = old })
	if err := tt.Click(use); err != nil {
		t.Fatal(err)
	}
	if want := plugin.Name(spec) + " <- acme"; got != want {
		t.Errorf("Use for wrote %q, want %q", got, want)
	}
}

// TestPluginsMoveAsk: switching a plugin on again whose built-in could run on
// it asks about the move, and its Move runs the domain's own move (magpie's
// move-again ask).
func TestPluginsMoveAsk(t *testing.T) {
	spec := "@someone/planner"
	seedPlugin(t, spec, "1.0.0")

	app := NewForTest()
	app.LoadNow()
	app.page, app.plugTab = PagePlugins, PlugTabInstalled
	if len(app.data.Plugins.Installed) == 0 {
		t.Fatal("the seeded plugin was not read")
	}
	// it is switched off, and a built-in could run on it
	app.data.Plugins.Installed[0].Off = true
	app.data.Plugins.Movable = []provider.MoveCandidate{{
		ID: "acme", Name: "Acme", Package: plugin.Name(spec), Accounts: 2,
	}}

	writes := []string{}
	oldOff, oldMove := plugSetPluginOff, plugMoveProvider
	plugSetPluginOff = func(ctx context.Context, name string, off bool) error {
		writes = append(writes, name+fmt.Sprintf(" off=%v", off))
		return nil
	}
	plugMoveProvider = func(ctx context.Context, id string) error {
		writes = append(writes, "move "+id)
		return nil
	}
	t.Cleanup(func() { plugSetPluginOff, plugMoveProvider = oldOff, oldMove })

	tt := ui.NewTester(app.ViewForTest(), 1180, 780)
	tt.Frame()
	p := app.text.S.Plug
	if !tt.HasText(p.SwitchOn) {
		t.Fatalf("the switched-off plugin drew no way on; texts were %v", tt.Texts())
	}
	if err := tt.Click(p.SwitchOn); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	ask := plugText(p.RunAgain, "names", "Acme")
	if !tt.HasText(ask) {
		t.Fatalf("switching it on did not ask about the move; texts were %v", tt.Texts())
	}
	if err := tt.Click(p.Move); err != nil {
		t.Fatal(err)
	}
	want := []string{spec + " off=false", "move acme"}
	if !slices.Equal(writes, want) {
		t.Errorf("the writes were %v, want %v", writes, want)
	}
}

// TestPluginsSwitchOffKeepsTheMovedOnes: switching off a plugin the
// built-ins were moved onto keeps them moved — the built-ins are retired,
// there is no way back — so nothing about going back is said (magpie's own
// change to its page, with MoveBack gone), while the ask still names them.
func TestPluginsSwitchOffKeepsTheMovedOnes(t *testing.T) {
	spec := "@someone/planner"
	seedPlugin(t, spec, "1.0.0")

	app := NewForTest()
	app.LoadNow()
	app.page, app.plugTab = PagePlugins, PlugTabInstalled
	e, ok := app.plugEntryFor(spec)
	if !ok {
		t.Fatal("the seeded plugin was not read")
	}
	// a built-in's accounts are on it; switching it off leaves them there
	e.Moved = []string{"acme"}
	app.data.Plugins.Movable = []provider.MoveCandidate{{
		ID: "acme", Name: "Acme", Package: plugin.Name(spec), Accounts: 2,
	}}
	for i := range app.data.Plugins.Installed {
		if app.data.Plugins.Installed[i].Spec == spec {
			app.data.Plugins.Installed[i].Moved = []string{"acme"}
		}
	}

	offSpec, offWas := "", false
	oldOff := plugSetPluginOff
	plugSetPluginOff = func(ctx context.Context, name string, off bool) error {
		offSpec, offWas = name, off
		return nil
	}
	t.Cleanup(func() { plugSetPluginOff = oldOff })

	tt := ui.NewTester(app.ViewForTest(), 1180, 780)
	tt.Frame()
	p := app.text.S.Plug
	// the row asks first, since the subscription it carries stops with it
	if err := tt.Click(p.SwitchOff); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !tt.HasText(plugText(p.AskOff, "names", "Acme")) {
		t.Fatalf("switching it off asked nothing; texts were %v", tt.Texts())
	}
	// the ask's own yes is the press that switches it off
	if err := tt.Click(p.SwitchOff); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if offSpec != spec || !offWas {
		t.Errorf("the write was %q off=%v, want %q off=true", offSpec, offWas, spec)
	}
	if app.plugNotice != "" {
		t.Errorf("switching it off said %q, want nothing said", app.plugNotice)
	}
}

// TestPluginsRequestHookRow: a plugin whose package declares an asynchronous
// request hook (package.json magpie.hooks) says so with its own chip and its
// own line, as magpie's page does, and no host is asked for its diagnostics:
// with none running the line waits for a request.
func TestPluginsRequestHookRow(t *testing.T) {
	spec := "@someone/hooks"
	seedPlugin(t, spec, "1.0.0")
	at := plugin.Target(spec)
	pkg := map[string]any{
		"name":    spec,
		"version": "1.0.0",
		"magpie":  map[string]any{"hooks": "hooks.js"},
	}
	b, err := json.Marshal(pkg)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(at, "package.json"), b, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(at, "hooks.js"), []byte("export const onRequest = async () => ({})\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	app := NewForTest()
	app.LoadNow()
	app.page, app.plugTab = PagePlugins, PlugTabInstalled
	e, ok := app.plugEntryFor(spec)
	if !ok {
		t.Fatal("the seeded plugin was not read")
	}
	if !e.IsGateway || !e.GatewayOnly {
		t.Fatalf("the request hook was not read: isGateway=%v gatewayOnly=%v", e.IsGateway, e.GatewayOnly)
	}
	if e.Gateway == nil || e.Gateway.LastOutcome != "idle" {
		t.Fatalf("its diagnostics were not kept: %+v", e.Gateway)
	}

	tt := ui.NewTester(app.ViewForTest(), 1180, 780)
	tt.Frame()
	p := app.text.S.Plug
	if !tt.HasText(p.KindHook) {
		t.Errorf("the row did not say it is a request hook; texts were %v", tt.Texts())
	}
	if want := p.GwLine + ": " + p.GwIdle; !tt.HasText(want) {
		t.Errorf("the row did not say what the hook last did; texts were %v", tt.Texts())
	}
	// a hook only is no provider: its row shows no provider chip — the
	// navigation's own entry for the Providers page is the only place that
	// word is drawn
	providers := 0
	for _, s := range tt.Texts() {
		if s == p.KindProvider {
			providers++
		}
	}
	if providers != 1 {
		t.Errorf("the provider chip was drawn for a request hook only (%d times); texts were %v", providers, tt.Texts())
	}
}

// TestPluginsPanelsAndActionsRow: a plugin whose page takes a built-in
// page's place, whose panels go into one of Echo's own pages and whose
// items go into a menu says all three in its row, as magpie's page does.
// The frames themselves are HTML the native shell does not draw: the row
// names them.
func TestPluginsPanelsAndActionsRow(t *testing.T) {
	spec := "@someone/panel"
	seedPlugin(t, spec, "1.0.0")
	at := plugin.Target(spec)
	pkg := map[string]any{
		"name":    spec,
		"version": "1.0.0",
		"magpie": map[string]any{
			"page": map[string]any{
				"title":    "Notes",
				"file":     "ui/notes.html",
				"replaces": "usage",
			},
			"panels": []map[string]any{
				{"slot": "provider", "file": "ui/spend.html", "title": "Spend"},
			},
			"actions": []map[string]any{
				{"id": "note", "menu": "provider", "title": "Write a note"},
			},
			"permissions": []string{"usage:summary"},
		},
	}
	b, err := json.Marshal(pkg)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(at, "package.json"), b, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(at, "ui"), 0o700); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"notes.html", "spend.html"} {
		if err := os.WriteFile(filepath.Join(at, "ui", f), []byte("<!doctype html>\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	app := NewForTest()
	app.LoadNow()
	app.page, app.plugTab = PagePlugins, PlugTabInstalled
	e, ok := app.plugEntryFor(spec)
	if !ok {
		t.Fatal("the seeded plugin was not read")
	}
	if !e.HasPage || e.PageReplaces != "usage" {
		t.Fatalf("its page was not read: hasPage=%v replaces=%q", e.HasPage, e.PageReplaces)
	}
	if len(e.Panels) != 1 || e.Panels[0] != "provider" || len(e.Actions) != 1 || e.Actions[0] != "provider" {
		t.Fatalf("its panels and actions were not read: %v, %v", e.Panels, e.Actions)
	}

	tt := ui.NewTester(app.ViewForTest(), 1180, 780)
	tt.Frame()
	p := app.text.S.Plug
	if want := plugText(p.PageTakesPlace, "page", app.text.S.Usage, "reads", p.ReadUsageSummary); !tt.HasText(want) {
		t.Errorf("the row did not say its page takes the Usage page's place; texts were %v", tt.Texts())
	}
	if want := plugText(p.PanelsAdds, "where", p.SlotProvider, "reads", p.ReadUsageSummary); !tt.HasText(want) {
		t.Errorf("the row did not say where its panels go; texts were %v", tt.Texts())
	}
	if want := plugText(p.ActionsAdds, "where", p.MenuProviders); !tt.HasText(want) {
		t.Errorf("the row did not say which menu its items go in; texts were %v", tt.Texts())
	}
	// both words for what it is, as magpie's own row carries them
	chips := map[string]int{}
	for _, s := range tt.Texts() {
		if s == p.KindPage || s == p.KindPanels {
			chips[s]++
		}
	}
	if chips[p.KindPage] != 1 || chips[p.KindPanels] != 1 {
		t.Errorf("the row did not carry both chips (%v); texts were %v", chips, tt.Texts())
	}
}

// TestPluginLimitedText: GitHub's rate limit used up is said in the page's
// own words, with the time it lifts and whether a token was carried.
func TestPluginLimitedText(t *testing.T) {
	app := NewForTest()
	p := app.text.S.Plug
	cases := []struct {
		in   plugin.GitHubLimit
		want string
	}{
		{plugin.GitHubLimit{Token: true}, p.LimitToken},
		{plugin.GitHubLimit{}, p.LimitAnon},
	}
	for _, c := range cases {
		if got := app.plugLimitedText(&c.in); got != c.want {
			t.Errorf("plugLimitedText(%+v) = %q, want %q", c.in, got, c.want)
		}
	}
	at := time.Date(2026, 10, 18, 14, 30, 0, 0, time.UTC)
	until := plugin.GitHubLimit{Until: at.Format(time.RFC3339)}
	if got, want := app.plugLimitedText(&until), plugText(p.LimitAnonUntil, "time", plugClock(app.lang, at)); got != want {
		t.Errorf("plugLimitedText with a time = %q, want %q", got, want)
	}
	tok := plugin.GitHubLimit{Token: true, Until: at.Format(time.RFC3339)}
	if got, want := app.plugLimitedText(&tok), plugText(p.LimitTokenUntil, "time", plugClock(app.lang, at)); got != want {
		t.Errorf("plugLimitedText with a token and a time = %q, want %q", got, want)
	}
}

// TestPluginStringsAreComplete: the Plugins page's words follow the interface's
// rule — every language has every string, and the translations keep the
// {placeholders}, so no sentence draws with a hole in it.
func TestPluginStringsAreComplete(t *testing.T) {
	en := reflect.ValueOf(pluginEnglish)
	zh := reflect.ValueOf(pluginChinese)
	if en.NumField() != zh.NumField() {
		t.Fatalf("the two languages have different shapes: %d and %d fields", en.NumField(), zh.NumField())
	}
	for i := 0; i < en.NumField(); i++ {
		field := en.Type().Field(i)
		if field.Type.Kind() != reflect.String {
			continue
		}
		want, got := en.Field(i).String(), zh.Field(i).String()
		if want == "" {
			t.Errorf("PluginStrings.%s has no English", field.Name)
			continue
		}
		if got == "" {
			t.Errorf("PluginStrings.%s has no Chinese", field.Name)
			continue
		}
		if !slices.Equal(placeholdersOf(want), placeholdersOf(got)) {
			t.Errorf("PluginStrings.%s: English has %v, Chinese has %v\n  en: %q\n  zh: %q",
				field.Name, placeholdersOf(want), placeholdersOf(got), want, got)
		}
	}
}

// TestPluginsBrandingIsEcho: nothing the page says calls the product magpie.
//
// The product's name is Echo wherever a reader can see it, and the page's
// words are all a reader's: an address (a repository's path) is not a name
// and keeps its own spelling.
func TestPluginsBrandingIsEcho(t *testing.T) {
	check := func(kind string, v reflect.Value, skip map[string]bool) {
		for i := 0; i < v.NumField(); i++ {
			field := v.Type().Field(i)
			if field.Type.Kind() != reflect.String || skip[field.Name] {
				continue
			}
			if strings.Contains(strings.ToLower(v.Field(i).String()), "magpie") {
				t.Errorf("%s.%s still says magpie: %q", kind, field.Name, v.Field(i).String())
			}
		}
	}
	// the community's repository is an address, not the product's name
	skip := map[string]bool{"CommunityTip": true}
	check("PluginStrings(en)", reflect.ValueOf(pluginEnglish), skip)
	check("PluginStrings(zh)", reflect.ValueOf(pluginChinese), skip)
}

func TestPluginUpdatesBadgeAndTab(t *testing.T) {
	app := NewForTest()
	app.LoadNow()
	app.page = PagePlugins
	app.data.Plugins.Updates = 2
	app.data.Plugins.Installed = []PluginEntryRow{{Spec: "@test/pkg", Name: "Pkg"}}

	tt := ui.NewTester(app.ViewForTest(), 1180, 780)
	tt.Frame()

	p := app.text.S.Plug
	// Tab title remains TabInstalled
	if !tt.HasText(p.TabInstalled) {
		t.Errorf("expected tab to retain %q, texts were %v", p.TabInstalled, tt.Texts())
	}
	// Badge shows "2 待更新"
	if !tt.HasText("2 待更新") {
		t.Errorf("expected badge with '2 待更新', texts were %v", tt.Texts())
	}
}

func TestPluginDiscoverIntroAndTopManualForm(t *testing.T) {
	app := NewForTest()
	app.LoadNow()
	app.page = PagePlugins
	app.plugTab = PlugTabDiscover
	app.plugMarket = &plugMarketData{
		Listings: []plugin.Listing{
			{Package: "@magpie-community/opencode-zen-free-auth", Name: "OpenCode Zen Free", Icon: "opencode", Community: true},
		},
		At: time.Now(),
	}

	tt := ui.NewTester(app.ViewForTest(), 1180, 780)
	tt.Frame()

	p := app.text.S.Plug
	// Has intro title
	if !tt.HasText(p.IntroTitle) {
		t.Errorf("expected intro title %q", p.IntroTitle)
	}
	// Has manual form label "已有想装的？"
	if !tt.HasText(p.HaveOne) {
		t.Errorf("expected top manual form %q", p.HaveOne)
	}
}

func TestPluginDetailDismiss(t *testing.T) {
	app := NewForTest()
	app.LoadNow()
	app.page = PagePlugins
	app.plugDetail = &plugDetail{Spec: "test-plugin", Name: "Test Plugin"}

	tt := ui.NewTester(app.ViewForTest(), 1180, 780)
	tt.Frame()

	if app.plugDetail == nil {
		t.Fatal("expected plugDetail to be open")
	}

	// Press escape to test dismissal
	tt.Key(0, ui.KeyEscape)
	tt.Frame()

	if app.plugDetail != nil {
		t.Errorf("expected plugDetail to be dismissed after escape")
	}
}

func TestPluginRealIconsResolution(t *testing.T) {
	app := NewForTest()
	// Test icon resolution for common plugins
	if got := plugIconByName("Kimi Work", "@magpie-community/opencode-kimi-auth"); got != "kimi" {
		t.Errorf("got %q, want 'kimi'", got)
	}
	if got := plugIconByName("OpenCode Zen Free", "@magpie-community/opencode-zen-free-auth"); got != "opencode" {
		t.Errorf("got %q, want 'opencode'", got)
	}
	if got := plugIconByName("Devin", "@magpie-community/opencode-devin-auth"); got != "devin" {
		t.Errorf("got %q, want 'devin'", got)
	}
	if got := plugIconByName("Zed", "@magpie-community/opencode-zed-auth"); got != "zed" {
		t.Errorf("got %q, want 'zed'", got)
	}
	pic := iconPictureFor("kimi")
	if pic == nil || pic == genericIcon() {
		t.Errorf("iconPictureFor('kimi') should return real icon, got %v", pic)
	}
	picOpenCode := iconPictureFor("opencode")
	if picOpenCode == nil || picOpenCode == genericIcon() {
		t.Errorf("iconPictureFor('opencode') should return real icon, got %v", picOpenCode)
	}
	_ = app
}
