//go:build nogui && echo

package shell

import (
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// TestSettingsAboutPage checks the About tab: magpie's About page — the
// version and its update check, the folder and the gateway's address, and
// the community.
func TestSettingsAboutPage(t *testing.T) {
	app := NewForTest()
	app.LoadNow()
	app.page = PageSettings
	app.setTab = SetTabAbout

	tt := ui.NewTester(app.ViewForTest(), 1100, 800)
	tt.Frame()

	snap := app.data.Settings
	for _, want := range []string{
		app.text.S.Set.Version, snap.Version,
		app.text.S.Set.ConfigDir, snap.Dir,
		app.text.S.Set.GatewayURL, snap.Gateway,
		app.text.S.Set.Community, "GitHub",
		app.text.S.Set.Copy,
	} {
		if want == "" {
			continue
		}
		if !tt.HasText(want) {
			t.Errorf("expected the About tab to draw %q; texts were %v", want, tt.Texts())
		}
	}
	if tt.HasText("Discord") {
		t.Errorf("expected the About tab to draw no Discord button; texts were %v", tt.Texts())
	}
}

// TestMenuHasSettingsAbout checks the application menu: the Settings
// submenu offers the About page, and it opens the Settings on that tab.
func TestMenuHasSettingsAbout(t *testing.T) {
	app := NewForTest()
	app.LoadNow()

	var about *mygo.MenuItem
	for _, item := range app.menu().Items() {
		if item.Label != app.text.S.Settings || item.Submenu == nil {
			continue
		}
		for _, sub := range item.Submenu {
			if sub.Label == app.text.S.Set.About {
				about = sub
			}
		}
	}
	if about == nil {
		t.Fatalf("the Settings menu has no %q item", app.text.S.Set.About)
	}
	if about.Click == nil {
		t.Fatal("the About item does nothing")
	}
	about.Click(about, nil)
	if app.page != PageSettings {
		t.Errorf("the About item opened %q, want the Settings page", app.page)
	}
	if app.setTab != SetTabAbout {
		t.Errorf("the About item opened the %q tab, want %q", app.setTab, SetTabAbout)
	}
}
