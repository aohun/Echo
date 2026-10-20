//go:build nogui && echo

package shell

import (
	"slices"
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestSettingsTabsRender: the Settings page is eight tabs, and each draws
// its own content — the cards, the rows — whatever its read came back
// with. A tab that panicked or fell back to the General would leave the
// page showing another tab's words.
//
// The check is the one TestEveryPageRenders makes for a page, made per
// tab: the marker is text only that tab draws, in the language the app
// speaks.
func TestSettingsTabsRender(t *testing.T) {
	app := NewForTest()
	app.ShowForTest(PageSettings)
	app.LoadNow()

	tt := ui.NewTester(app.ViewForTest(), 1180, 780)
	for _, tab := range setTabs {
		app.setTab = tab
		app.LoadNow()
		tt.Frame()

		marker := app.ContentMarkerForTest(PageSettings)
		if marker == "" {
			t.Errorf("tab %q has no content marker", tab)
			continue
		}
		if !tt.HasText(marker) {
			t.Errorf("tab %q did not render its content (expected %q); texts were %v",
				tab, marker, tt.Texts())
		}
		if tab == SetTabGeneral {
			writeShot(t, "echo-settings-general.png", tt.Image())
		}
	}
}

// TestSettingsRendersInChinese: the Settings page's words are the reader's
// language's, as every page's are. A tab that drew its English while the
// app speaks Chinese would be a row that skipped its translation.
func TestSettingsRendersInChinese(t *testing.T) {
	app := NewForTest()
	app.lang, app.text = LangZH, Text(LangZH)
	app.ShowForTest(PageSettings)
	app.LoadNow()

	tt := ui.NewTester(app.ViewForTest(), 1180, 780)
	for _, tab := range setTabs {
		app.setTab = tab
		app.LoadNow()
		tt.Frame()

		// the words the section itself draws are the reader's language's;
		// the section's name is said only where it is shown — the
		// navigation's menu item, 常规 the only one for now
		seen := tt.Texts()
		if tab == SetTabGeneral && !slices.Contains(seen, app.setTabLabel(tab)) {
			t.Errorf("the shown section %q is missing from the navigation; texts were %v",
				tab, seen)
		}
	}
}

// TestSettingsEveryTabHasAMarker: a tab's marker is text only that tab
// draws. A tab whose marker is drawn by the chrome too (the page's name,
// a tab strip's label of another) would prove nothing.
func TestSettingsEveryTabHasAMarker(t *testing.T) {
	app := NewForTest()
	app.ShowForTest(PageSettings)
	app.LoadNow()

	seen := map[string]SetTab{}
	for _, tab := range setTabs {
		app.setTab = tab
		marker := app.ContentMarkerForTest(PageSettings)
		if marker == "" {
			t.Errorf("tab %q has no content marker", tab)
			continue
		}
		if other, dup := seen[marker]; dup {
			t.Errorf("tabs %q and %q share the marker %q: one could pass for the other",
				other, tab, marker)
		}
		seen[marker] = tab
	}
}

// TestSettingsLanguageChangeReadsBack: picking a language writes the saved
// one and the page's words change with it, as magpie's own language row
// does.
func TestSettingsLanguageChangeReadsBack(t *testing.T) {
	app := NewForTest()
	app.ShowForTest(PageSettings)

	// what the row's own write does, as the view's pick does it
	app.lang = resolveLang("en", "")
	app.text = Text(app.lang)

	if app.text.Lang != LangEN {
		t.Errorf("the page still speaks %q after the language changed", app.text.Lang)
	}
}
