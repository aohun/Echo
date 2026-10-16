//go:build nogui && echo

package shell

import (
	"fmt"
	"runtime"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/yetone/magpie/internal/settings"
)

// view builds the window's interface from the app's state, for every frame.
//
// It runs on the main thread, so it reads a's fields directly and never
// blocks: everything that would touch the filesystem was read by readData
// beforehand.
func (a *App) view(c *ui.Context) {
	t := c.Theme()
	c.Root().Background(t.Background)

	ui.Row(c).Fill().Children(func() {
		a.sidebar(c)
		ui.Column(c).Grow(1).Shrink(1).Children(func() {
			a.header(c)
			a.pageView(c)
		})
	})
}

// sidebar is the navigation: Echo's pages, grouped as magpie's are, with the
// gateway's address at its foot.
func (a *App) sidebar(c *ui.Context) {
	t := c.Theme()
	side := ui.Column(c).Width(212).Shrink(0).PaddingY(t.Space(3)).Background(t.Surface)
	side.Children(func() {
		// The brand, as the window's own name rather than a title bar.
		ui.Text(c, a.text.S.AppName).FontSize(15).Bold().Padding(t.Space(1), t.Space(5), t.Space(3))

		page := string(a.page)
		if ui.Sidebar(c, &page, func() {
			// One section per group, in the order the pages list them. A
			// group whose pages all belong to a page that is not built
			// still shows: the skeleton has no gateway mode yet.
			for _, g := range []Group{GroupWork, GroupModels, GroupInsight, GroupExtend} {
				group := g
				open := a.sidebarOpen[group]
				ui.SidebarSection(c, a.groupTitle(group), &open, func() {
					for _, info := range a.Pages() {
						if info.Group != group {
							continue
						}
						item := ui.SidebarItem(c, string(info.Page), nil, info.Title)
						// A page that has data behind it gets a count,
						// so the navigation says something about it.
						if n, ok := a.pageCount(info.Page); ok {
							item.Children(func() { ui.Badge(c, fmt.Sprint(n)) })
						}
					}
				})
				a.sidebarOpen[group] = open
			}
		}).Grow(1).Label(a.text.S.AppName).Changed() {
			a.page = Page(page)
			c.Invalidate()
		}

		a.sidebarFoot(c)
	})
}

// sidebarFoot is the bottom of the navigation: the gateway's address, which is
// the one fact about the running app worth keeping in sight.
func (a *App) sidebarFoot(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Padding(t.Space(2), t.Space(5)).Gap(t.Space(1)).Children(func() {
		ui.Text(c, a.gatewayAddr()).FontSize(11).TextColor(t.TextMuted).SingleLine().
			Tooltip(a.text.S.Gateway)
	})
}

// header is the strip above the page: its title, and what the page can do.
func (a *App) header(c *ui.Context) {
	t := c.Theme()
	ui.Row(c).AlignItems(ui.Center).Padding(t.Space(4), t.Space(5)).Gap(t.Space(3)).
		Background(t.Background).Children(func() {
		ui.Text(c, a.pageTitle(a.page)).FontSize(17).Bold()
		ui.Spacer(c)
		if a.loading {
			ui.Spinner(c).Size(14, 14)
		}
		if ui.Button(c, a.text.S.Settings).Clicked() {
			mygo.App.ShowAboutPanel(mygo.AboutPanelOptions{
				ApplicationName:    a.text.S.AppName,
				ApplicationVersion: mygo.App.Version(),
			})
		}
	})
	ui.Divider(c)
}

// pageView draws the page being shown.
func (a *App) pageView(c *ui.Context) {
	// One scroll for every page: the pages are lists, and a list scrolls
	// itself, but a page's own header and foot do not.
	ui.Scroll(c).Fill().Children(func() {
		ui.Column(c).Fill().Padding(0, 0, 20, 0).Children(func() {
			switch a.page {
			case PageAgents:
				a.agentsView(c)
			case PageProviders:
				a.providersView(c)
			case PagePlugins:
				a.pluginsView(c)
			default:
				a.placeholderView(c)
			}
		})
	})
}

// placeholderView is a page whose own read is not written yet. It says so
// rather than showing an empty page that looks broken.
func (a *App) placeholderView(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Padding(t.Space(6), t.Space(5)).Gap(t.Space(2)).Children(func() {
		ui.Text(c, a.pageTitle(a.page)).FontSize(13).Bold().TextColor(t.TextMuted)
		ui.Text(c, a.text.S.ComingSoon).FontSize(15)
		ui.Text(c, a.text.S.Skeleton).FontSize(12).TextColor(t.TextMuted)
	})
}

// agentsView lists the coding agents, present ones first.
func (a *App) agentsView(c *ui.Context) {
	t := c.Theme()
	if a.err != nil {
		ui.Text(c, a.err.Error()).FontSize(13).TextColor(t.Danger).Padding(t.Space(4), t.Space(5))
		return
	}
	if len(a.data.Agents) == 0 {
		ui.Text(c, a.text.S.NoData).FontSize(13).TextColor(t.TextMuted).Padding(t.Space(4), t.Space(5))
		return
	}
	a.table(c, []ui.TableColumn{
		{Title: "Name", ID: "name", Width: 0, MinWidth: 120},
		{Title: "ID", ID: "id", Width: 160},
		{Title: "Config", ID: "path", Width: 0},
		{Title: "Found", ID: "found", Width: 90, Align: ui.End},
	}, len(a.data.Agents), func(row, col int) {
		r := a.data.Agents[row]
		switch col {
		case 0:
			ui.Text(c, r.Name).SingleLine()
		case 1:
			ui.Text(c, r.ID).Font("monospace").FontSize(12).TextColor(t.TextMuted).SingleLine()
		case 2:
			ui.Text(c, r.Path).FontSize(12).TextColor(t.TextMuted).SingleLine()
		case 3:
			// A word, not a tick: the column is read by a screen reader,
			// and a lone glyph says nothing.
			if r.Found {
				ui.Text(c, "yes").FontSize(12).TextColor(t.Success)
			} else {
				ui.Text(c, "no").FontSize(12).TextColor(t.TextMuted)
			}
		}
	})
}

// providersView lists the configured vendors.
func (a *App) providersView(c *ui.Context) {
	t := c.Theme()
	if len(a.data.Providers) == 0 {
		ui.Text(c, a.text.S.NoData).FontSize(13).TextColor(t.TextMuted).Padding(t.Space(4), t.Space(5))
		return
	}
	a.table(c, []ui.TableColumn{
		{Title: "Name", ID: "name", Width: 0, MinWidth: 120},
		{Title: "ID", ID: "id", Width: 160},
		{Title: "Models", ID: "models", Width: 90, Align: ui.End},
		{Title: "Key", ID: "key", Width: 70, Align: ui.End},
	}, len(a.data.Providers), func(row, col int) {
		p := a.data.Providers[row]
		switch col {
		case 0:
			ui.Text(c, p.Name).SingleLine()
		case 1:
			ui.Text(c, p.ID).Font("monospace").FontSize(12).TextColor(t.TextMuted).SingleLine()
		case 2:
			ui.Text(c, fmt.Sprint(p.Models)).FontSize(12)
		case 3:
			if p.Keyed || p.SignedIn {
				ui.Text(c, "yes").FontSize(12).TextColor(t.Success)
			} else {
				ui.Text(c, "no").FontSize(12).TextColor(t.TextMuted)
			}
		}
	})
}

// pluginsView lists the plugins built into this binary.
func (a *App) pluginsView(c *ui.Context) {
	t := c.Theme()
	if len(a.data.Plugins) == 0 {
		ui.Text(c, a.text.S.NoData).FontSize(13).TextColor(t.TextMuted).Padding(t.Space(4), t.Space(5))
		return
	}
	a.table(c, []ui.TableColumn{
		{Title: "Name", ID: "name", Width: 0, MinWidth: 140},
		{Title: "ID", ID: "id", Width: 140},
		{Title: "What it is", ID: "sub", Width: 0},
	}, len(a.data.Plugins), func(row, col int) {
		p := a.data.Plugins[row]
		switch col {
		case 0:
			ui.Text(c, p.Name).SingleLine()
		case 1:
			ui.Text(c, p.ID).Font("monospace").FontSize(12).TextColor(t.TextMuted).SingleLine()
		case 2:
			ui.Text(c, p.Sub).FontSize(12).TextColor(t.TextMuted).SingleLine()
		}
	})
}

// table draws rows in columns, with the state a table needs to keep: the
// columns' order and widths, and which row is chosen.
func (a *App) table(c *ui.Context, cols []ui.TableColumn, n int, cell func(row, col int)) {
	state := a.tableState(a.page)
	state.Key = func(row int) any { return row }
	state.Label = func(row int) string {
		if row >= 0 && row < n {
			return a.rowLabel(row)
		}
		return ""
	}
	ui.Table(c, state, cols, n, cell).FillWidth().Padding(0, 16)
}

// rowLabel names a row for assistive technology and for type-to-choose.
func (a *App) rowLabel(row int) string {
	switch a.page {
	case PageAgents:
		if row < len(a.data.Agents) {
			return a.data.Agents[row].Name
		}
	case PageProviders:
		if row < len(a.data.Providers) {
			return a.data.Providers[row].Name
		}
	case PagePlugins:
		if row < len(a.data.Plugins) {
			return a.data.Plugins[row].Name
		}
	}
	return ""
}

// pageCount is how many things a page lists, for the navigation's badge, and
// whether the page has anything to count.
func (a *App) pageCount(p Page) (int, bool) {
	switch p {
	case PageAgents:
		n := 0
		for _, r := range a.data.Agents {
			if r.Found {
				n++
			}
		}
		return n, n > 0
	case PageProviders:
		return len(a.data.Providers), len(a.data.Providers) > 0
	case PagePlugins:
		return len(a.data.Plugins), len(a.data.Plugins) > 0
	}
	return 0, false
}

// gatewayAddr is the address the gateway serves on, or that it is not running.
//
// The address is magpie's own saved setting, so the shell reports what the
// gateway actually serves rather than a guess of its own.
func (a *App) gatewayAddr() string {
	addr := settings.SavedAddr()
	if addr == "" {
		return "—"
	}
	return addr
}

// platformName names the system this build runs on, for the About panel.
func platformName() string {
	switch runtime.GOOS {
	case "darwin":
		return "macOS"
	case "windows":
		return "Windows"
	case "linux":
		return "Linux"
	}
	return runtime.GOOS
}
