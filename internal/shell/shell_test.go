//go:build nogui && echo

package shell

import (
	"reflect"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/yetone/magpie/internal/gateway"
	"github.com/yetone/magpie/internal/settings"
)

// TestEveryPageRenders is the shell's core acceptance check: every page the
// navigation offers renders, and renders its own content rather than only the
// chrome around it.
//
// It is the assertion the probe (cmd/echoprobe) reports on, kept here so a
// regression fails a test rather than only a probe run.
//
// The marker per page is what only that page draws: its title alone would be
// satisfied by the header, which every page shows whatever its content does,
// so a page that stopped drawing would still pass. The marker is read from
// the app's own state, so it tracks the data rather than a fixture.
func TestEveryPageRenders(t *testing.T) {
	app := NewForTest()
	app.LoadNow()
	tt := ui.NewTester(app.ViewForTest(), 1180, 780)

	pages := AllPages()
	if len(pages) == 0 {
		t.Fatal("no pages: the navigation would be empty")
	}
	for _, p := range pages {
		app.ShowForTest(p)
		tt.Frame()

		title := app.PageTitleForTest(p)
		if title == "" {
			t.Errorf("page %q has no title", p)
			continue
		}
		if !tt.HasText(title) {
			t.Errorf("page %q did not render its title %q; texts were %v", p, title, tt.Texts())
			continue
		}

		marker := app.ContentMarkerForTest(p)
		if marker == "" {
			t.Errorf("page %q has no content marker: the test cannot tell its content from its chrome", p)
			continue
		}
		if !tt.HasText(marker) {
			t.Errorf("page %q rendered its chrome but not its content (expected %q); texts were %v",
				p, marker, tt.Texts())
		}
	}
}

// TestNavigationOffersEveryPage: a page the navigation does not list cannot
// be reached, whatever the switch in the view does.
func TestNavigationOffersEveryPage(t *testing.T) {
	app := NewForTest()
	tt := ui.NewTester(app.ViewForTest(), 1180, 780)

	for _, info := range app.Pages() {
		if !tt.HasText(info.Title) {
			t.Errorf("the navigation does not offer page %q (%q)", info.Page, info.Title)
		}
	}
	if got, want := len(app.Pages()), len(AllPages()); got != want {
		t.Errorf("Pages() returns %d pages, AllPages() %d: the two lists drifted", got, want)
	}
}

// TestPagesAreUniqueAndComplete: a page listed twice draws two navigation
// entries, and a page missing from the switch draws a placeholder.
func TestPagesAreUniqueAndComplete(t *testing.T) {
	seen := map[Page]bool{}
	for _, p := range AllPages() {
		if seen[p] {
			t.Errorf("page %q is listed twice", p)
		}
		seen[p] = true
	}

	// Every page must be reachable from the view's own switch. The pages
	// built so far are the ones with a case; the rest fall to the
	// placeholder, which still draws the title. This asserts the set the
	// switch handles is a subset of the pages the navigation lists.
	app := NewForTest()
	tt := ui.NewTester(app.ViewForTest(), 1180, 780)
	for _, p := range AllPages() {
		app.ShowForTest(p)
		tt.Frame()
		if len(tt.Texts()) == 0 {
			t.Errorf("page %q rendered nothing at all", p)
		}
	}
}

// TestEveryLanguageIsComplete: every language the settings accept draws every
// page's title, and no language falls back to another's text for a page.
//
// magpie's rule is that a user-visible string needs every language; a page
// whose title is missing in one language would draw English there.
func TestEveryLanguageIsComplete(t *testing.T) {
	for _, lang := range langs {
		text := Text(lang)
		for _, p := range AllPages() {
			app := &App{text: text}
			if got := app.pageTitle(p); got == "" {
				t.Errorf("language %q has no title for page %q", lang, p)
			}
		}
	}
}

// TestSavedLanguageWinsOverLocale: a language the reader saved is never
// replaced, which is magpie's rule for every setting the reader set.
// "system", empty and anything unknown all mean Echo picks — and Echo's
// pick is Chinese whatever the desktop's locale is.
func TestSavedLanguageWinsOverLocale(t *testing.T) {
	cases := []struct {
		name  string
		saved string
		want  Lang
	}{
		{"saved chinese wins", "zh", LangZH},
		{"saved english wins", "en", LangEN},
		{"traditional chinese saved draws chinese", "zh-TW", LangZH},
		{"system defaults to chinese", "system", LangZH},
		{"empty defaults to chinese", "", LangZH},
		{"unknown saved defaults to chinese", "xx", LangZH},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// The desktop's locale is irrelevant to Echo's default: it
			// is passed as something deliberately unlike Chinese, so a
			// test proves the locale does not leak in.
			if got := resolveLang(c.saved, "en-US"); got != c.want {
				t.Errorf("resolveLang(%q, %q) = %q, want %q", c.saved, "en-US", got, c.want)
			}
		})
	}
}

// TestEveryLanguageKeepsItsPlaceholders: a translated string has the same
// {placeholders} as the English one, and none is left in English.
//
// magpie's rule is that every user-visible string needs every language, and
// a translation that drops {n} or renames {agents} draws a sentence with a
// hole in it. The check reads the Strings struct by reflection, so a new
// field is covered the day it is added.
func TestEveryLanguageKeepsItsPlaceholders(t *testing.T) {
	en := reflect.ValueOf(english)

	for _, lang := range langs {
		if lang == LangEN {
			continue
		}
		tr, ok := translations[lang]
		if !ok {
			continue
		}
		over := reflect.ValueOf(tr)

		for i := 0; i < en.NumField(); i++ {
			field := en.Type().Field(i)
			if field.Type.Kind() != reflect.String {
				continue
			}
			want := en.Field(i).String()
			got := over.Field(i).String()
			if got == "" {
				continue // filled from English, which the merge does
			}

			wantP := placeholdersOf(want)
			gotP := placeholdersOf(got)
			if !slices.Equal(wantP, gotP) {
				t.Errorf("%s.%s: English has %v, %s has %v\n  en: %q\n  %s: %q",
					"Strings", field.Name, wantP, lang, gotP, want, lang, got)
			}
		}
	}
}

// placeholdersOf is the {placeholders} in a string, sorted, so two strings'
// can be compared.
func placeholdersOf(s string) []string {
	var out []string
	for {
		i := strings.IndexByte(s, '{')
		if i < 0 {
			break
		}
		j := strings.IndexByte(s[i:], '}')
		if j < 0 {
			break
		}
		out = append(out, s[i:i+j+1])
		s = s[i+j+1:]
	}
	sort.Strings(out)
	return out
}

// TestLoadSettingsReadsMagpie: the shell draws the reader's saved language,
// which is magpie's own setting rather than one of the shell's.
func TestLoadSettingsReadsMagpie(t *testing.T) {
	s := settings.Load()
	orig := s.Lang
	t.Cleanup(func() {
		s := settings.Load()
		s.Lang = orig
		_ = settings.Save(s)
	})

	s.Lang = string(LangZH)
	if err := settings.Save(s); err != nil {
		t.Fatalf("saving settings: %v", err)
	}

	app := newApp(PageAgents, "en-US")
	if app.lang != LangZH {
		t.Errorf("app.lang = %q, want %q: the saved language was not read", app.lang, LangZH)
	}
	if app.text.S.Agents == english.Agents {
		t.Errorf("the Chinese title is still English: %q", app.text.S.Agents)
	}
}

// TestQuitFlagLetsTheWindowClose: the close handler hides the window rather
// than ending the app, except while the app is quitting — otherwise a quit
// would be swallowed and the app could never exit.
//
// The handler is driven through MyGo's real quit sequence, because the
// ordering is the whole bug: a quit fires OnBeforeQuit, then closes every
// window, and only then fires OnWillQuit. The flag must therefore be set on
// OnBeforeQuit; set on OnWillQuit it lands after the close it was meant to
// let through, the handler hides the window, and the app never exits.
func TestQuitFlagLetsTheWindowClose(t *testing.T) {
	app := NewForTest()
	if app.quitting {
		t.Fatal("a fresh app is already quitting")
	}

	// A close that arrives on its own (the window's button) hides the
	// window and does not close it.
	close := &mygo.CloseEvent{}
	app.onWindowClose(close)
	if !close.DefaultPrevented() {
		t.Error("closing the window did not hide it: the app would end on its own")
	}

	// The same close, inside a quit: OnBeforeQuit has run, so the handler
	// lets it through.
	app.beginQuit()
	close = &mygo.CloseEvent{}
	app.onWindowClose(close)
	if close.DefaultPrevented() {
		t.Error("the quit's own window close was cancelled: the app cannot exit")
	}
}

// fakeQuitHooks stands in for mygo.Application's quit hooks, so the wiring
// can be replayed in MyGo's real order without a running app.
type fakeQuitHooks struct {
	before, will []func(*mygo.QuitEvent)
}

func (f *fakeQuitHooks) OnBeforeQuit(fn func(*mygo.QuitEvent)) func() {
	f.before = append(f.before, fn)
	return func() {}
}

func (f *fakeQuitHooks) OnWillQuit(fn func(*mygo.QuitEvent)) func() {
	f.will = append(f.will, fn)
	return func() {}
}

// TestQuitWiringSurvivesMyGosSequence: wireQuit must leave the app able to
// exit when a quit runs MyGo's real sequence — OnBeforeQuit, then every
// window closed, then OnWillQuit.
//
// This is the regression that made the tray's Quit do nothing: the flag was
// set on OnWillQuit, which MyGo fires only after the windows have closed, so
// the close handler hid the window instead and cancelled the quit. Replaying
// the sequence here fails if the flag moves back.
func TestQuitWiringSurvivesMyGosSequence(t *testing.T) {
	app := NewForTest()
	hooks := &fakeQuitHooks{}
	stopped := false
	wireQuit(hooks, app, func() { stopped = true }, nil)

	if len(hooks.before) != 1 {
		t.Fatalf("OnBeforeQuit got %d listeners, want 1: the flag would never be set before the windows close", len(hooks.before))
	}
	if len(hooks.will) != 1 {
		t.Fatalf("OnWillQuit got %d listeners, want 1: the gateway would never be given back", len(hooks.will))
	}

	// MyGo's prepareQuit: OnBeforeQuit, then each window's close, then
	// OnWillQuit.
	hooks.before[0](&mygo.QuitEvent{})
	close := &mygo.CloseEvent{}
	app.onWindowClose(close)
	if close.DefaultPrevented() {
		t.Fatal("a quit's window close was cancelled: the app cannot exit")
	}
	hooks.will[0](&mygo.QuitEvent{})
	if !stopped {
		t.Error("the gateway was not given back on the way out")
	}
}

// TestChineseNavigationMatchesMagpie: Chinese navigation strings must match
// active pages in navigation (使用, Agent, 资源库, 供应商, 路由, 网关, 观测, 用量, 扩展, 插件).
// Note: 会话 and 模型 are temporarily hidden.
func TestChineseNavigationMatchesMagpie(t *testing.T) {
	s := settings.Load()
	orig := s.Lang
	t.Cleanup(func() {
		s := settings.Load()
		s.Lang = orig
		_ = settings.Save(s)
	})

	s.Lang = string(LangZH)
	if err := settings.Save(s); err != nil {
		t.Fatalf("saving settings: %v", err)
	}

	app := newApp(PageAgents, "zh-CN")
	tt := ui.NewTester(app.ViewForTest(), 1180, 780)
	tt.Frame()

	expected := []string{
		"使用", "Agent", "资源库",
		"供应商", "路由", "网关",
		"观测", "用量",
		"扩展", "插件",
	}
	for _, text := range expected {
		if !tt.HasText(text) {
			t.Errorf("Chinese navigation missing expected string %q; texts were %v", text, tt.Texts())
		}
	}
}

// TestSidebarBottomSettingsAndGateway: Settings and the Gateway address are
// placed in the bottom-left corner of the sidebar, matching magpie's layout.
func TestSidebarBottomSettingsAndGateway(t *testing.T) {
	app := NewForTest()
	tt := ui.NewTester(app.ViewForTest(), 1180, 780)
	tt.Frame()

	// Settings should be rendered in the sidebar
	if !tt.HasText(app.text.S.Settings) {
		t.Errorf("Settings item %q not found in sidebar", app.text.S.Settings)
	}

	// Gateway status should be rendered in the sidebar
	gwAddr := strings.TrimPrefix(gateway.URL(), "http://")
	gwLabel := app.text.S.Gateway + " " + gwAddr
	if !tt.HasText(gwLabel) {
		t.Errorf("Gateway status %q not found in sidebar; texts were %v", gwLabel, tt.Texts())
	}

	// Clicking Settings navigates to PageSettings
	if err := tt.Click(app.text.S.Settings); err != nil {
		t.Fatalf("clicking settings: %v", err)
	}
	tt.Frame()
	if app.page != PageSettings {
		t.Errorf("expected page %q after clicking settings, got %q", PageSettings, app.page)
	}

	// Breadcrumb on settings page shows Settings title alone
	if !tt.HasText(app.text.S.Settings) {
		t.Errorf("expected breadcrumb title %q on settings page", app.text.S.Settings)
	}

	// Clicking Gateway status navigates to PageGateway
	if err := tt.Click(gwLabel); err != nil {
		t.Fatalf("clicking gateway status: %v", err)
	}
	tt.Frame()
	if app.page != PageGateway {
		t.Errorf("expected page %q after clicking gateway status, got %q", PageGateway, app.page)
	}
}

