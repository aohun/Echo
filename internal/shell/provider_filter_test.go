//go:build nogui && echo

package shell

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// TestProvidersFilterSegments: the segments keep the rows they name — all
// keeps every row with the ones off under their fold, the on segment the
// ones on alone, the off segment the ones off alone — and a filter with
// nothing behind it shows no rows.
//
// pvFilter is driven as a click on a segment would drive it, and the frame
// is read as the screen would show it.
func TestProvidersFilterSegments(t *testing.T) {
	app := NewForTest()
	app.lang, app.text = LangEN, Text(LangEN)
	app.LoadNow()
	// The look for other apps' configurations has run: none are set here,
	// and a kick from the frame must not start one behind the test's back.
	app.pvDiscoveryAt = time.Now()
	app.data.Providers = []ProviderRow{
		{ID: "on1", Name: "On One", Models: 1, Ready: true},
		{ID: "on2", Name: "On Two", Models: 1, Ready: true},
		{ID: "off1", Name: "Off One", Models: 1, Off: true},
	}
	tt := ui.NewTester(app.ViewForTest(), 1180, 780)
	app.ShowForTest(PageProviders)

	// All: every row, the ones on followed by the ones off — no fold.
	tt.Frame()
	writeShot(t, "echo-providers-rows.png", tt.Image())
	for _, want := range []string{"On One", "On Two", "Off One"} {
		if !tt.HasText(want) {
			t.Errorf("the all view is missing %q; texts were %v", want, tt.Texts())
		}
	}
	if tt.HasText("Turned off") {
		t.Error("the all view still shows the fold heading")
	}

	// The on segment: the ones on, and no fold — there is nothing under it.
	app.pvFilter = 1
	tt.Frame()
	if !tt.HasText("On One") || !tt.HasText("On Two") {
		t.Errorf("the on view lost a row; texts were %v", tt.Texts())
	}
	if tt.HasText("Off One") {
		t.Error("the on view shows a row that is off")
	}

	// The off segment: the one off, and no fold.
	app.pvFilter = 2
	tt.Frame()
	if !tt.HasText("Off One") {
		t.Errorf("the off view is missing the one off; texts were %v", tt.Texts())
	}
	if tt.HasText("On One") {
		t.Error("the off view shows a row that is on")
	}
}

// TestProviderSegmentsShowCounts: each segment says how many rows it keeps,
// so a segment with nothing behind it is said to before it is chosen.
func TestProviderSegmentsShowCounts(t *testing.T) {
	app := NewForTest()
	app.lang, app.text = LangEN, Text(LangEN)
	app.LoadNow()
	app.pvDiscoveryAt = time.Now()
	app.data.Providers = []ProviderRow{
		{ID: "on1", Name: "On One", Ready: true},
		{ID: "off1", Name: "Off One", Off: true},
	}
	tt := ui.NewTester(app.ViewForTest(), 1180, 780)
	app.ShowForTest(PageProviders)
	tt.Frame()

	for _, want := range []string{
		"All (2)",
		"On (1)",
		"Off (1)",
	} {
		if !tt.HasText(want) {
			t.Errorf("the segments do not say %q; texts were %v", want, tt.Texts())
		}
	}
}

// TestAddProviderOpensTheModal: ＋ Add provider opens the sheet as a modal
// over the page, as magpie's own add sheet does over its rows, and the
// providers it lists — the subscriptions, the vendors, the relays and what
// runs on this machine — are the same picks magpie's sheet offers
// (provider.Presets, magpie's own presets). Reading the sheet's own dialog
// back through the frame is the assertion: without it a click that opens
// nothing would pass a page that renders fine.
func TestAddProviderOpensTheModal(t *testing.T) {
	app := NewForTest()
	app.lang, app.text = LangEN, Text(LangEN)
	app.LoadNow()
	// The look for other apps' configurations has run: none are set here,
	// and a kick from the frame must not start one behind the test's back.
	app.pvDiscoveryAt = time.Now()
	app.data.Providers = []ProviderRow{
		{ID: "on1", Name: "On One", Ready: true},
	}
	tt := ui.NewTester(app.ViewForTest(), 1180, 780)
	app.ShowForTest(PageProviders)
	tt.Frame()

	app.pvAdding = &addSheet{}
	tt.Frame()

	// The sheet's title and every group's heading are on screen: the
	// subscriptions, the vendors, the relays and the custom group — and
	// not the ones Echo leaves out.
	for _, want := range []string{
		app.text.S.AddProvider,
		app.text.S.Subscriptions,
		app.text.S.Vendors,
		app.text.S.Relays,
		app.text.S.CustomSection,
		app.text.S.CustomProvider,
	} {
		if !tt.HasText(want) {
			t.Errorf("the add sheet does not say %q; texts were %v", want, tt.Texts())
		}
	}
	for _, gone := range []string{
		app.text.S.OnThisMachine,
		"localhost:11434", "localhost:1234", "localhost:8000", "localhost:11234",
		"Remote magpie", app.text.S.AnotherMagpie,
	} {
		if tt.HasText(gone) {
			t.Errorf("the add sheet still offers %q; texts were %v", gone, tt.Texts())
		}
	}
	// A subscription pick says its plans; a preset's row says its host.
	// These are the fields a pick that means nothing in the sheet would
	// leave out, so they show the rows were drawn, not just the headings.
	if !tt.HasText("Claude") {
		t.Errorf("the add sheet has no subscriptions; texts were %v", tt.Texts())
	}
	if !tt.HasText("Anthropic") {
		t.Errorf("the add sheet has no vendor presets; texts were %v", tt.Texts())
	}

	// The search keeps only what matches: a query for a relay leaves the
	// vendors' headings out.
	app.pvAdding.Query = "openrouter"
	app.pvAdding.Picker = nil
	tt.Frame()
	if !tt.HasText("OpenRouter") {
		t.Errorf("the search does not keep the relay; texts were %v", tt.Texts())
	}
	if tt.HasText("Anthropic") {
		t.Errorf("the search kept a row it does not match; texts were %v", tt.Texts())
	}

	// Picking a preset folds the sheet into its editor: the form's own
	// heading is the pick's name and the key field's label is said. No
	// back button in the head — the foot's 取消 is the way back now.
	pick := &addPick{Kind: AddPreset, Key: "anthropic", Name: "Anthropic", Section: "vendor"}
	app.pvAdding.Picker = pick
	tt.Frame()
	if !tt.HasText("Anthropic") {
		t.Errorf("the preset editor does not name the pick; texts were %v", tt.Texts())
	}
	if tt.HasText(app.text.S.Back) {
		t.Errorf("the preset editor still has a back button; texts were %v", tt.Texts())
	}
	if !tt.HasText(app.text.S.Cancel) || !tt.HasText(app.text.S.Add) {
		t.Errorf("the editor's foot names no 取消 and 添加; texts were %v", tt.Texts())
	}
	writeShot(t, "echo-add-editor.png", tt.Image())

	// 取消 from the foot puts the pick away and shows the sheet's
	// picks again — the pick is cleared before the frame draws the head,
	// so the head names the sheet rather than the pick that just went.
	if err := tt.Click(app.text.S.Cancel); err != nil {
		t.Fatalf("clicking 取消: %v", err)
	}
	tt.Frame()
	if app.pvAdding == nil || app.pvAdding.Picker != nil {
		t.Fatalf("取消 left the sheet editing: %+v", app.pvAdding)
	}
	if !tt.HasText(app.text.S.AddProvider) {
		t.Errorf("Back did not show the sheet's own title; texts were %v", tt.Texts())
	}
	if !tt.HasText(app.text.S.Close) {
		t.Errorf("the sheet has no close button; texts were %v", tt.Texts())
	}

	// A press on the dimmed page beside the sheet is not a press on the
	// sheet: the sheet and whatever its editor holds stay. Only the
	// sheet's own close puts it away.
	app.pvAdding.Picker = &addPick{Kind: AddPreset, Key: "anthropic", Name: "Anthropic", Section: "vendor"}
	tt.Frame()
	tt.ClickAt(6, 740)
	tt.Frame()
	if app.pvAdding == nil {
		t.Fatal("a press on the dimmed page closed the add sheet")
	}

	// Escape closes it: the sheet is a dialog, and a dialog's own Escape
	// is the way out the keyboard has.
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if app.pvAdding != nil {
		t.Error("Escape left the add sheet open")
	}

	// The custom editor's 取消 is the way back to the picks: the editor's
	// pick goes, the sheet stays. The sheet itself goes by its corner's
	// close or Escape.
	app.pvAdding = &addSheet{}
	app.pvAdding.Picker = &addPick{Kind: AddCustom, Section: "custom"}
	tt.Frame()
	if !tt.HasText(app.text.S.Cancel) {
		t.Fatalf("the custom editor has no cancel; texts were %v", tt.Texts())
	}
	if err := tt.Click(app.text.S.Cancel); err != nil {
		t.Fatalf("clicking Cancel: %v", err)
	}
	tt.Frame()
	if app.pvAdding == nil || app.pvAdding.Picker != nil {
		t.Error("取消 did not return to the sheet's picks")
	}

	// The sheet's own close, when there is one to click, puts it away too.
	app.pvAdding = &addSheet{}
	tt.Frame()
	if err := tt.Click(app.text.S.Close); err != nil {
		t.Fatalf("clicking the sheet's close: %v", err)
	}
	tt.Frame()
	if app.pvAdding != nil {
		t.Error("closing the add sheet left it open")
	}
}
