//go:build nogui && echo

package shell

import "github.com/egoist/mygo/ui"

// The hooks below let the acceptance check (cmd/echoprobe) drive the shell
// without a window: the same App and the same view, rendered in memory by
// MyGo's ui.NewTester.
//
// They are the shell's own test surface rather than a second implementation:
// ViewForTest is the view Run hands the window, and ShowForTest changes the
// page the way the sidebar does. Keeping one view is what makes the probe
// evidence about the shipped interface.

// NewForTest builds an app as Run does, without a window or a tray.
//
// The locale is "en": a probe run has no desktop to ask, and the language it
// draws in must not depend on the machine it runs on. A saved language in the
// probe's own home still wins, which is the behaviour under test.
func NewForTest() *App { return newApp(PageAgents, "en") }

// ViewForTest is the app's view, for ui.NewTester and ui.Render.
func (a *App) ViewForTest() func(c *ui.Context) { return a.view }

// ShowForTest switches to a page, as a click in the sidebar does.
func (a *App) ShowForTest(p Page) { a.page = p }

// PageTitleForTest is the page's title in the app's language, which every
// page draws in the header whatever its own content is.
func (a *App) PageTitleForTest(p Page) string { return a.pageTitle(p) }

// ContentMarkerForTest is the text only the page's own content draws, which
// tells a page that rendered its content from one that rendered only the
// chrome around it.
func (a *App) ContentMarkerForTest(p Page) string { return a.contentMarker(p) }

// contentMarker is the text only this page's own content draws.
//
// A page with a read of its own shows its first row, or says it has nothing
// when the read came back empty — which is itself proof the page drew rather
// than fell through. A page whose read is not written yet draws the
// placeholder. Either way the marker is a string the chrome never draws, so
// it tells a page that rendered its content from one that rendered only the
// header and sidebar around it.
//
// The acceptance check reads it (shell_test.go, cmd/echoprobe): a page's
// title alone is not enough, because the header draws that whatever the page
// itself does.
func (a *App) contentMarker(p Page) string {
	switch p {
	case PageAgents:
		if len(a.data.Agents) > 0 {
			return a.data.Agents[0].Name
		}
		return a.text.S.NoData
	case PageProviders:
		if len(a.data.Providers) > 0 {
			return a.data.Providers[0].Name
		}
		return a.text.S.NoData
	case PagePlugins:
		if len(a.data.Plugins) > 0 {
			return a.data.Plugins[0].Name
		}
		return a.text.S.NoData
	}
	// A page whose own read is not written yet draws the placeholder, which
	// says so rather than showing an empty page that looks broken.
	return a.text.S.ComingSoon
}

// AllPages is every page, in the order the navigation lists them.
func AllPages() []Page {
	pages := make([]Page, 0, len(allPages))
	for _, spec := range allPages {
		pages = append(pages, spec.page)
	}
	return pages
}
