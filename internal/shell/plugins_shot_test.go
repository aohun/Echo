//go:build nogui && echo

package shell

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/yetone/magpie/internal/plugin"
	"github.com/yetone/magpie/internal/provider"
)

// TestPluginsShots writes the three tabs and the detail sheet as pictures,
// for a look at the page's layout: the left margin the whole shell keeps, the
// cards' grid, and the buttons pinned to the rows' right.
func TestPluginsShots(t *testing.T) {
	seedPlugin(t, "@someone/planner", "1.0.0")

	app := NewForTest()
	app.LoadNow()
	app.page = PagePlugins

	tt := ui.NewTester(app.ViewForTest(), 1180, 780)
	tt.Frame()
	tt.Frame()
	writeShot(t, "echo-plugins-builtin.png", tt.Image())

	app.plugTab = PlugTabInstalled
	app.plugMarket = &plugMarketData{NPM: map[string]plugin.NPM{
		"@someone/planner": {Version: "2.0.0", Weekly: 1200},
	}}
	app.plugClashes = map[string][]PluginClashRow{"@someone/planner": {{
		Clash:  plugin.Clash{ID: "acme", By: "@other/runner", With: []string{"@someone/planner"}},
		Name:   "Acme",
		Others: "@other/runner",
	}}}
	app.data.Plugins.Movable = []provider.MoveCandidate{{
		ID: "acme", Name: "Acme", Package: plugin.Name("@someone/planner"), Accounts: 2,
	}}
	tt.Frame()
	tt.Frame()
	writeShot(t, "echo-plugins-installed.png", tt.Image())

	app.plugTab = PlugTabDiscover
	app.plugMarket = &plugMarketData{
		Listings: []plugin.Listing{{
			Package: "@someone/planner", Name: "Planner", Community: true,
			Summary: map[string]string{"zh": "先想好再回答"},
		}},
		NPM: map[string]plugin.NPM{"@someone/planner": {Version: "2.0.0", Publisher: "someone", Weekly: 1200}},
	}
	app.plugTagged = []plugin.Tagged{{
		Spec: "github:someone/tagger", Repo: "someone/tagger", Owner: "someone",
		Description: "Tags things", Stars: 12, Package: "@someone/tagger",
	}}
	app.plugTaggedDone = true
	tt.Frame()
	tt.Frame()
	writeShot(t, "echo-plugins-discover.png", tt.Image())

	app.plugTab = PlugTabInstalled
	app.plugDetail = &plugDetail{
		Spec: "@someone/planner", Name: "Planner", Kind: "middleware",
		NPM:  &plugin.NPM{Version: "2.0.0", Weekly: 1200, License: "MIT", Publisher: "someone", Description: "Plans first"},
		Tag:  &plugin.Tagged{Repo: "someone/planner", Owner: "someone", Stars: 42, Spec: "github:someone/planner"},
		Have: true,
	}
	app.plugDetailText = "# Planner\n\nPlans **before** it answers.\n\n```js\nexport const middleware = () => {}\n```\n"
	tt.Frame()
	tt.Frame()
	writeShot(t, "echo-plugins-detail.png", tt.Image())
}
