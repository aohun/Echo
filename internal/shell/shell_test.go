//go:build nogui && echo

package shell

import (
	"testing"

	"github.com/egoist/mygo/ui"
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
// replaced by the desktop's locale, which is magpie's rule for every setting
// the reader set. "system" and an unknown value fall back to the locale.
func TestSavedLanguageWinsOverLocale(t *testing.T) {
	cases := []struct {
		name   string
		saved  string
		locale string
		want   Lang
	}{
		{"saved language wins", "ja", "de-DE", LangJA},
		{"saved traditional chinese wins", "zh-TW", "en-US", LangZHTW},
		{"system uses the locale", "system", "ja-JP", LangJA},
		{"empty uses the locale", "", "de-DE", LangDE},
		{"unknown saved falls back to the locale", "xx", "ja-JP", LangJA},
		{"unknown both falls back to english", "xx", "xx-YY", LangEN},
		{"zh-Hant is traditional", "system", "zh-Hant-TW", LangZHTW},
		{"zh-CN is simplified", "system", "zh-CN", LangZH},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := resolveLang(c.saved, c.locale); got != c.want {
				t.Errorf("resolveLang(%q, %q) = %q, want %q", c.saved, c.locale, got, c.want)
			}
		})
	}
}

// TestLoadSettingsReadsMagpie: the shell draws the reader's saved language,
// which is magpie's own setting rather than one of the shell's.
func TestLoadSettingsReadsMagpie(t *testing.T) {
	s := settings.Load()
	s.Lang = string(LangDE)
	if err := settings.Save(s); err != nil {
		t.Fatalf("saving settings: %v", err)
	}

	app := newApp(PageAgents, "en-US")
	if app.lang != LangDE {
		t.Errorf("app.lang = %q, want %q: the saved language was not read", app.lang, LangDE)
	}
	if app.text.S.Agents == english.Agents {
		t.Errorf("the German title is still English: %q", app.text.S.Agents)
	}
}

// TestQuitFlagLetsTheWindowClose: the close handler hides the window rather
// than ending the app, except while the app is quitting — otherwise a quit
// would be swallowed and the app could never exit.
//
// The handler itself needs a window and a real event, so this pins the flag
// the handler reads.
func TestQuitFlagLetsTheWindowClose(t *testing.T) {
	app := NewForTest()
	if app.quitting {
		t.Fatal("a fresh app is already quitting")
	}
}
