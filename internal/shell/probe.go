//go:build nogui && echo

package shell

import (
	"fmt"

	"github.com/egoist/mygo/ui"
)

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
		// The Plugins page draws the tab the reader picked, and each tab's
		// own first words are what say it drew: the built-ins' intro, the
		// market's (which stands whatever the market answered), or what
		// Installed says of what was added.
		switch a.plugTab {
		case PlugTabDiscover:
			return a.text.S.Plug.IntroTitle
		case PlugTabInstalled:
			if len(a.data.Plugins.Installed) > 0 {
				return a.data.Plugins.Installed[0].Name
			}
			return a.text.S.Plug.NoPlugins
		}
		return a.text.S.Plug.BuiltinIntro
	case PageGateway:
		if len(a.data.Gateway.Entries) > 0 {
			return a.data.Gateway.Entries[0].ID
		}
		if a.data.Gateway.URL != "" {
			return a.data.Gateway.URL
		}
		return a.text.S.NoData
	case PageLibrary:
		// The Library draws the tab the reader picked, and its own content
		// is what says it drew: the skills' rows when there are any, else
		// the line saying there are none. A tab not built yet draws the
		// placeholder, which says so rather than looking broken.
		if a.libTab == LibTabSkills || a.libTab == "" {
			if len(a.data.Library.Skills) > 0 {
				return a.data.Library.Skills[0].Name
			}
			return a.text.S.LibNoSkills
		}
		return a.text.S.ComingSoon
	case PageRouting:
		if len(a.data.Routing.Groups) > 0 {
			return a.data.Routing.Groups[0].Name
		}
		return "实时 · 网关正在做的决定"
	case PageUsage:
		// The Usage page draws one of four segments, and each draws a line
		// of its own whatever its read came back with: the filters' own
		// words on Requests and Sessions, the note under Context, the
		// allowance card or the empty period's words on Overview. Those
		// are what tell the page's content from its chrome.
		switch a.usTab {
		case UsTabRequests:
			return "导出 CSV"
		case UsTabSessions:
			return "全部模型"
		case UsTabContext:
			if len(a.data.Usage.Context.Agents) > 0 {
				return "提示词的构成"
			}
			if len(a.data.Usage.Context.Sessions) > 0 {
				return fmt.Sprintf("最近 %d 天里，网关读到的提示词构成；评分与标签由这些请求算出。",
					a.data.Usage.Context.Days)
			}
			return "这段时间没有可分析的提示词。"
		}
		if a.data.Usage.Overview.Calls == 0 {
			return a.usNoCallsIn(a.usPeriod)
		}
		return "次调用"
	case PageSettings:
		// The tab strip's first tab is the page's own content: the chrome
		// draws the page's name (Settings), never the tabs' names.
		return a.setTabLabel(a.setTab)
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
