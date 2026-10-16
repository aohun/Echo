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
	contentBg := ui.Hex("#ffffff")
	if t.Dark {
		contentBg = ui.Hex("#1c1c1f")
	}
	c.Root().Background(contentBg)

	ui.Row(c).Fill().AlignItems(ui.Stretch).Children(func() {
		if !a.sidebarHidden {
			a.sidebar(c)
		}
		ui.Column(c).Grow(1).Shrink(1).FillHeight().Children(func() {
			a.header(c)
			a.pageView(c)
		})
	})
}

// sidebarTop renders the top strip of the sidebar: macOS traffic lights space,
// Echo's brand icon and name, and the window drag region.
func (a *App) sidebarTop(c *ui.Context) {
	t := c.Theme()
	bar := c.TitleBar()

	leftPad := float32(14)
	if runtime.GOOS == "darwin" {
		leftPad = 84
		if bar.Left > 0 && bar.Left+12 > leftPad {
			leftPad = bar.Left + 12
		}
	}

	h := float32(50)
	if bar.Height > h {
		h = bar.Height
	}

	textC := ui.Hex("#1c1c1f")
	iconC := ui.Hex("#1c1c1f")
	if t.Dark {
		textC = ui.Hex("#ededf1")
		iconC = ui.Hex("#ededf1")
	}

	row := ui.Row(c).Height(h).
		AlignItems(ui.Center).
		Padding(0, 14, 0, leftPad).
		Gap(7).
		DragWindow()

	row.Children(func() {
		ui.Icon(c, brandIcon()).Size(16, 16).TextColor(iconC)
		ui.Text(c, a.text.S.AppName).
			FontSize(13.5).
			FontWeight(600).
			TextColor(textC).
			SingleLine()
	})
}

// sidebar is the navigation: Echo's pages, grouped as magpie's are.
func (a *App) sidebar(c *ui.Context) {
	t := c.Theme()
	bg := ui.Hex("#f4f4f6")
	borderC := ui.RGBA(0, 0, 0, 0.08)
	if t.Dark {
		bg = ui.Hex("#1a1a1e")
		borderC = ui.RGBA(255, 255, 255, 0.08)
	}

	side := ui.Column(c).Width(212).Shrink(0).FillHeight().Background(bg).
		BorderWidth(0, 0.5, 0, 0).BorderColor(borderC)
	side.Children(func() {
		a.sidebarTop(c)
		ui.Scroll(c).Grow(1).Shrink(1).Padding(0, 10, 10, 10).Children(func() {
			for i, g := range []Group{GroupWork, GroupModels, GroupInsight, GroupExtend} {
				group := g
				topPad := float32(14)
				if i == 0 {
					topPad = 2
				}
				ui.Row(c).Padding(topPad, 8, 4, 8).Children(func() {
					ui.Text(c, a.groupTitle(group)).
						FontSize(11).
						FontWeight(500).
						TextColor(t.TextMuted).
						SingleLine()
				})

				for _, info := range a.Pages() {
					if info.Group != group {
						continue
					}
					a.sidebarItem(c, info)
				}
			}
		})
	})
}

// sidebarItem renders one item in the navigation sidebar.
func (a *App) sidebarItem(c *ui.Context, info PageInfo) {
	t := c.Theme()
	selected := info.Page == a.page

	row := ui.Row(c).Key(string(info.Page)).
		AlignItems(ui.Center).
		Gap(9).
		Height(30).
		Padding(0, 8).
		Radius(7).
		Cursor(ui.CursorPointer).
		Focusable().
		Role(ui.RoleButton)

	if row.Clicked() {
		a.page = info.Page
		c.Invalidate()
	}

	icon := pageIcon(info.Page)

	var (
		textC ui.Color
		iconC ui.Color
	)

	if selected {
		if t.Dark {
			row.Background(ui.Hex("#25262c")).
				Border(0.5, ui.RGBA(255, 255, 255, 0.08)).
				Shadow(0, 1, 2, 0, ui.RGBA(0, 0, 0, 0.3))
			textC = ui.Hex("#ededf1")
			iconC = ui.Hex("#ededf1")
		} else {
			row.Background(ui.Hex("#ffffff")).
				Border(0.5, ui.RGBA(0, 0, 0, 0.08)).
				Shadow(0, 1, 2, 0, ui.RGBA(0, 0, 0, 0.06))
			textC = ui.Hex("#1c1c1f")
			iconC = ui.Hex("#1c1c1f")
		}
	} else {
		if row.Hovered() || row.Focused() {
			if t.Dark {
				row.Background(ui.RGBA(255, 255, 255, 0.06))
				textC = ui.Hex("#ededf1")
				iconC = ui.Hex("#ededf1")
			} else {
				row.Background(ui.RGBA(0, 0, 0, 0.04))
				textC = ui.Hex("#1c1c1f")
				iconC = ui.Hex("#1c1c1f")
			}
		} else {
			if t.Dark {
				textC = ui.Hex("#b9b9c1")
				iconC = ui.Hex("#8b8b94")
			} else {
				textC = ui.Hex("#4b5563")
				iconC = ui.Hex("#85858d")
			}
		}
	}

	row.Children(func() {
		if icon != nil {
			ui.Icon(c, icon).Size(16, 16).TextColor(iconC)
		}
		txt := ui.Text(c, info.Title).
			FontSize(13).
			TextColor(textC).
			Grow(1).
			SingleLine()
		if selected {
			txt.FontWeight(500)
		}
	})
}

// header is the strip above the page: breadcrumbs, sidebar toggle, settings, and window dragging.
func (a *App) header(c *ui.Context) {
	t := c.Theme()
	bar := c.TitleBar()

	h := float32(50)
	if bar.Height > h {
		h = bar.Height
	}

	leftPad := float32(16)
	if a.sidebarHidden && runtime.GOOS == "darwin" {
		leftPad = 84
		if bar.Left > 0 && bar.Left+12 > leftPad {
			leftPad = bar.Left + 12
		}
	}

	borderC := ui.RGBA(0, 0, 0, 0.08)
	textC := ui.Hex("#1c1c1f")
	mutedC := ui.Hex("#85858d")
	faintC := ui.Hex("#9a9aa3")
	if t.Dark {
		borderC = ui.RGBA(255, 255, 255, 0.08)
		textC = ui.Hex("#ededf1")
		mutedC = ui.Hex("#8b8b94")
		faintC = ui.Hex("#5a5a63")
	}

	headerRow := ui.Row(c).Height(h).
		AlignItems(ui.Center).
		Padding(0, 16, 0, leftPad).
		Gap(10).
		Background(t.Background).
		BorderWidth(0, 0, 0.5, 0).
		BorderColor(borderC).
		DragWindow()

	headerRow.Children(func() {
		// Toggle sidebar button
		toggleBtn := ui.Row(c).
			Center().
			Width(28).
			Height(28).
			Radius(5).
			Cursor(ui.CursorPointer).
			Role(ui.RoleButton)

		if toggleBtn.Hovered() {
			if t.Dark {
				toggleBtn.Background(ui.RGBA(255, 255, 255, 0.08))
			} else {
				toggleBtn.Background(ui.RGBA(0, 0, 0, 0.05))
			}
		}
		if toggleBtn.Clicked() {
			a.sidebarHidden = !a.sidebarHidden
			c.Invalidate()
		}
		toggleBtn.Children(func() {
			ui.Icon(c, toggleIcon()).Size(15, 15).TextColor(mutedC)
		})

		// Breadcrumbs: Group / Page
		groupTitle := a.groupTitle(a.pageGroup(a.page))
		pageTitle := a.pageTitle(a.page)

		ui.Row(c).AlignItems(ui.Center).Gap(7).Children(func() {
			ui.Text(c, groupTitle).
				FontSize(13.5).
				TextColor(mutedC).
				SingleLine()
			ui.Text(c, "/").
				FontSize(13.5).
				TextColor(faintC).
				SingleLine()
			ui.Text(c, pageTitle).
				FontSize(13.5).
				FontWeight(600).
				TextColor(textC).
				SingleLine()
		})

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
}

// pageView draws the page being shown.
func (a *App) pageView(c *ui.Context) {
	// One scroll for every page: the pages are lists, and a list scrolls
	// itself, but a page's own header and foot do not.
	ui.Scroll(c).Grow(1).Shrink(1).Children(func() {
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
