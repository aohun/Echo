//go:build nogui && echo

package shell

import (
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/transfer"
	"github.com/egoist/mygo/ui"
	"github.com/yetone/magpie/internal/gateway"
	"github.com/yetone/magpie/internal/gui"
	"github.com/yetone/magpie/internal/provider"
)

// view builds the window's interface from the app's state, for every frame.
//
// It runs on the main thread, so it reads a's fields directly and never
// blocks: everything that would touch the filesystem was read by readData
// beforehand.
func (a *App) view(c *ui.Context) {
	// the watchdog's heartbeat: the frame says it is alive
	beatFrame()

	// the text size the General tab sets, as the styling guide has it: a
	// copy of the theme with its font size scaled, which every widget's
	// text and metrics take from
	if th := *c.Theme(); true {
		if sz := a.data.Settings.S.TextSize; sz > 0 && sz != 100 {
			th.FontSize = th.FontSize * float32(sz) / 100
		}
		c.SetTheme(&th)
	}

	// the page open gave way to a built-in plugin switched off: Agents, as
	// magpie's own window does
	if a.data.Plugins.OffPages[string(a.page)] {
		a.page = PageAgents
	}

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

	// The Providers page's dialogs over everything: the Import from other
	// apps picker, and the sign-out ask (magpie's own are modals over the
	// page). Each is drawn only while it is open; dismissed by the
	// backdrop or Escape, it is put away.
	if a.pvImporting != nil {
		open := true
		ui.Modal(c, &open, func() { a.importDialog(c) })
		if !open {
			a.pvImporting = nil
		}
	}
	if a.pvSignOut != nil {
		open := true
		ui.Modal(c, &open, func() { a.signOutDialog(c) })
		if !open {
			a.pvSignOut = nil
		}
	}
	// The add sheet is not a modal of that kind. Its picks and its editors
	// hold what a reader put in them — a key half-typed, a sign-in under
	// way — and a press that lands beside the sheet, on the dimmed page,
	// would put all of it away for a click that was not aimed at it. It
	// closes from the sheet itself: its corner's close, an editor's
	// Cancel, or Escape.
	a.addOverlay(c)

	// A provider's editor over everything: the one a sign-in succeeds to.
	a.pvEditView(c)
	// its discard ask over it, when it tries to close with changes still
	// in the draft
	if a.pvDiscard {
		a.pvDiscardView(c)
	}

	// The Library's sheets: the reader for one skill's SKILL.md, the asks
	// before something is removed, and the way each agent is given its
	// skills. One at a time, as magpie's own modal holds one.
	a.skillsOverlay(c)

	// The Plugins page's own sheet: one plugin's page, over the list.
	a.pluginsOverlay(c)

	// The Routing page's sheets: group editor, import, and trace details.
	a.routingOverlay(c)

	// The Usage page's sheets: a request's conversation, a session's, and
	// the ask before a Codex reset is spent.
	a.usageOverlay(c)

	// The Settings page's sheets and dialogs.
	a.settingsOverlay(c)
}

// sidebarTop renders the top strip of the sidebar: macOS traffic lights
// space, Echo's name, and the window drag region.
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
	if t.Dark {
		textC = ui.Hex("#ededf1")
	}

	// Centering in this row centers on the traffic lights: the window
	// places them (run.go) so the band TitleBar reports is this row's own
	// height, with the lights centered in it.
	//
	// No brand icon: the name lines up with the navigation rows beside it
	// (they indent their content 8px into the same padding), as a Mac
	// app's sidebar title does.
	row := ui.Row(c).Height(h).
		AlignItems(ui.Center).
		Padding(0, 14, 0, leftPad).
		Gap(7).
		DragWindow()

	row.Children(func() {
		ui.Text(c, a.text.S.AppName).
			FontSize(13.5).
			FontWeight(600).
			TextColor(textC).
			Padding(0, 8).
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
			for i, g := range []Group{GroupWork, GroupInsight, GroupExtend} {
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
		a.sidebarBottom(c)
	})
}

// sidebarBottom renders the bottom pinned area of the sidebar:
// Settings navigation item and Gateway status indicator.
func (a *App) sidebarBottom(c *ui.Context) {
	t := c.Theme()
	borderC := ui.RGBA(0, 0, 0, 0.08)
	if t.Dark {
		borderC = ui.RGBA(255, 255, 255, 0.08)
	}

	bottom := ui.Column(c).Shrink(0).
		Padding(8, 10, 10, 10).
		Gap(4).
		BorderWidth(0.5, 0, 0, 0).
		BorderColor(borderC)

	bottom.Children(func() {
		a.sidebarSettingsItem(c, t)
		// the sections as menu items under 设置, while it shows
		if a.page == PageSettings {
			a.sidebarSettingsSubmenu(c)
		}
		a.sidebarGatewayStatus(c, t)
	})
}

// sidebarSettingsSubmenu draws the settings sections under 设置, as menu
// items of their own (magpie's set of tabs, split out of the page's tab
// strip). Only 常规 shows for now — the rest are hidden until their turn.
func (a *App) sidebarSettingsSubmenu(c *ui.Context) {
	t := c.Theme()
	lineC := ui.RGBA(0, 0, 0, 0.12)
	if t.Dark {
		lineC = ui.RGBA(255, 255, 255, 0.12)
	}

	subs := []struct {
		tab   SetTab
		title string
	}{
		{SetTabGeneral, a.text.S.Set.General},
		{SetTabAbout, a.text.S.Set.About},
		// 额度、网络与共享、模型、隐私、可观测性、同步与备份
		// follow here once they are adjusted to the native shell.
	}

	ui.Row(c).Padding(2, 0, 4, 18).Gap(10).Children(func() {
		// the guide line the tree hangs from
		ui.Box(c).Width(1.2).FillHeight().Background(lineC).Radius(0.5)
		ui.Column(c).Grow(1).Shrink(1).Gap(2).Children(func() {
			for _, it := range subs {
				it := it
				selected := a.page == PageSettings && a.setTab == it.tab
				row := ui.Row(c).Key("set-sub-" + string(it.tab)).
					AlignItems(ui.Center).
					Height(28).
					Padding(0, 8).
					Radius(6).
					Cursor(ui.CursorPointer).
					Focusable().
					Role(ui.RoleButton)
				if row.Clicked() {
					a.page = PageSettings
					a.setTab = it.tab
					c.Invalidate()
				}
				var textC ui.Color
				if selected {
					if t.Dark {
						row.Background(ui.RGBA(255, 255, 255, 0.10)).
							Border(0.5, ui.RGBA(255, 255, 255, 0.08))
						textC = ui.Hex("#ededf1")
					} else {
						row.Background(ui.Hex("#ffffff")).
							Border(0.5, ui.RGBA(0, 0, 0, 0.08))
						textC = ui.Hex("#1c1c1f")
					}
				} else if row.Hovered() || row.Focused() {
					if t.Dark {
						row.Background(ui.RGBA(255, 255, 255, 0.05))
						textC = ui.Hex("#ededf1")
					} else {
						row.Background(ui.RGBA(0, 0, 0, 0.04))
						textC = ui.Hex("#1c1c1f")
					}
				} else if t.Dark {
					textC = ui.Hex("#9c9ca6")
				} else {
					textC = ui.Hex("#55555e")
				}
				row.Children(func() {
					txt := ui.Text(c, it.title).
						FontSize(12.5).
						TextColor(textC).
						Grow(1).
						SingleLine()
					if selected {
						txt.FontWeight(500)
					}
				})
			}
		})
	})
}

// sidebarSettingsItem renders the Settings row in the sidebar footer.
func (a *App) sidebarSettingsItem(c *ui.Context, t *ui.Theme) {
	selected := a.page == PageSettings

	row := ui.Row(c).Key("nav-settings").
		AlignItems(ui.Center).
		Gap(9).
		Height(30).
		Padding(0, 8).
		Radius(7).
		Cursor(ui.CursorPointer).
		Focusable().
		Role(ui.RoleButton).
		Label(a.text.S.Settings)

	if row.Clicked() {
		a.page = PageSettings
		c.Invalidate()
	}

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
		ui.Icon(c, gearIcon()).Size(16, 16).TextColor(iconC)
		txt := ui.Text(c, a.text.S.Settings).
			FontSize(13).
			TextColor(textC).
			Grow(1).
			SingleLine()
		if selected {
			txt.FontWeight(500)
		}
	})
}

// sidebarGatewayStatus renders the Gateway connection status in the sidebar footer.
func (a *App) sidebarGatewayStatus(c *ui.Context, t *ui.Theme) {
	gwRunning := a.data.Gateway.Running || gateway.Running()

	gwAddr := strings.TrimPrefix(a.data.Gateway.URL, "http://")
	if gwAddr == "" {
		gwAddr = strings.TrimPrefix(gateway.URL(), "http://")
	}
	if gwAddr == "" {
		gwAddr = "127.0.0.1:3425"
	}
	gwLabel := a.text.S.Gateway + " " + gwAddr

	row := ui.Row(c).Key("nav-gateway-status").
		AlignItems(ui.Center).
		Gap(8).
		Height(26).
		Padding(0, 8).
		Radius(7).
		Cursor(ui.CursorPointer).
		Focusable().
		Role(ui.RoleButton).
		Label(gwLabel)

	if row.Hovered() || row.Focused() {
		if t.Dark {
			row.Background(ui.RGBA(255, 255, 255, 0.06))
		} else {
			row.Background(ui.RGBA(0, 0, 0, 0.04))
		}
	}

	if row.Clicked() {
		a.page = PageGateway
		c.Invalidate()
	}

	var (
		dotBadgeBg ui.Color
		dotC       ui.Color
		textC      ui.Color
	)

	if gwRunning {
		if t.Dark {
			dotBadgeBg = ui.RGBA(24, 154, 88, 0.22)
			dotC = ui.Hex("#22c55e")
		} else {
			dotBadgeBg = ui.Hex("#e5f6ed")
			dotC = ui.Hex("#189a58")
		}
	} else {
		if t.Dark {
			dotBadgeBg = ui.RGBA(255, 255, 255, 0.08)
			dotC = ui.Hex("#5a5a63")
		} else {
			dotBadgeBg = ui.RGBA(0, 0, 0, 0.06)
			dotC = ui.Hex("#85858d")
		}
	}

	if t.Dark {
		textC = ui.Hex("#8b8b94")
	} else {
		textC = ui.Hex("#85858d")
	}

	row.Children(func() {
		ui.Row(c).Center().Width(16).Height(16).Radius(8).Background(dotBadgeBg).Children(func() {
			ui.Row(c).Width(7).Height(7).Radius(3.5).Background(dotC)
		})
		ui.Text(c, gwLabel).
			FontSize(12).
			TextColor(textC).
			Grow(1).
			SingleLine()
	})
}

// sidebarItem renders one item in the navigation sidebar.
func (a *App) sidebarItem(c *ui.Context, info PageInfo) {
	t := c.Theme()
	selected := info.Page == a.page
	if info.Page == PageLibrary && (a.libMenuOpen || a.page == PageLibrary) {
		selected = false // The active sub-tab (e.g. 技能) is highlighted instead
	}

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
		if info.Page == PageLibrary {
			if a.page != PageLibrary {
				a.page = PageLibrary
				a.libMenuOpen = true
				if a.libTab == "" {
					a.libTab = LibTabSkills
				}
			} else {
				a.libMenuOpen = !a.libMenuOpen
			}
		} else {
			// Plugins with updates waiting opens where they are acted on,
			// as magpie's own navigation does
			if info.Page == PagePlugins && a.data.Plugins.Updates > 0 {
				a.plugTab = PlugTabInstalled
			}
			a.page = info.Page
		}
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
		// the updates waiting, said where they are acted on: the count
		// stands in the navigation and says where to go, as magpie's own
		// dot and its Installed tab's count do
		if info.Page == PagePlugins && a.data.Plugins.Updates > 0 {
			p := &a.text.S.Plug
			label := p.UpdatesOne
			if n := a.data.Plugins.Updates; n > 1 {
				label = plugText(p.UpdatesN, "n", plugNum(n))
			}
			row.Tooltip(p.UpdatesTip)
			ui.Box(c).Height(18).Padding(0, 6).Radius(9).
				Background(t.Accent).Center().Shrink(0).Children(func() {
				ui.Text(c, label).FontSize(10.5).FontWeight(500).
					TextColor(ui.Hex("#ffffff")).SingleLine()
			})
		}
		if info.Page == PageRouting && len(a.data.Routing.Groups) > 0 {
			ui.Box(c).Height(18).Padding(0, 6).Radius(9).
				Background(ui.RGBA(0, 0, 0, 0.05)).Center().Shrink(0).Children(func() {
				ui.Text(c, strconv.Itoa(len(a.data.Routing.Groups))).FontSize(10.5).FontWeight(500).
					TextColor(t.TextMuted).SingleLine()
			})
		}
	})

	if info.Page == PageLibrary && (a.libMenuOpen || a.page == PageLibrary) {
		a.sidebarLibrarySubmenu(c)
	}
}

// sidebarLibrarySubmenu renders the indented tree sub-items under 资源库 (Library)
// with a subtle vertical guide line down the left side and badge counts.
func (a *App) sidebarLibrarySubmenu(c *ui.Context) {
	t := c.Theme()
	if a.libTab == "" {
		a.libTab = LibTabSkills
	}

	lineC := ui.RGBA(0, 0, 0, 0.12)
	if t.Dark {
		lineC = ui.RGBA(255, 255, 255, 0.12)
	}

	// Only the skills tab is drawn under 资源库 (Library): the other
	// resources' tabs are not built yet, so they are left out of the
	// navigation rather than offered as dead entries.
	subItems := []struct {
		tab   LibTab
		title string
		count int
	}{
		{LibTabSkills, a.text.S.LibSkills, a.data.Library.Counts.Skills},
	}

	ui.Row(c).Padding(2, 0, 4, 18).Gap(10).Children(func() {
		// Vertical tree guide line
		ui.Box(c).Width(1.2).FillHeight().Background(lineC).Radius(0.5)

		// Column of sub-items
		ui.Column(c).Grow(1).Shrink(1).Gap(2).Children(func() {
			for _, item := range subItems {
				it := item
				isSelected := a.page == PageLibrary && a.libTab == it.tab

				row := ui.Row(c).Key("lib-sub-"+string(it.tab)).
					AlignItems(ui.Center).
					Height(28).
					Padding(0, 8).
					Radius(6).
					Cursor(ui.CursorPointer).
					Focusable().
					Role(ui.RoleButton)

				if row.Clicked() {
					a.page = PageLibrary
					a.libTab = it.tab
					c.Invalidate()
				}

				var textC ui.Color
				if isSelected {
					if t.Dark {
						row.Background(ui.RGBA(255, 255, 255, 0.10)).
							Border(0.5, ui.RGBA(255, 255, 255, 0.08))
						textC = ui.Hex("#ededf1")
					} else {
						row.Background(ui.Hex("#ffffff")).
							Border(0.5, ui.RGBA(0, 0, 0, 0.08)).
							Shadow(0, 1, 2, 0, ui.RGBA(0, 0, 0, 0.06))
						textC = ui.Hex("#1c1c1f")
					}
				} else {
					if row.Hovered() || row.Focused() {
						if t.Dark {
							row.Background(ui.RGBA(255, 255, 255, 0.05))
							textC = ui.Hex("#ededf1")
						} else {
							row.Background(ui.RGBA(0, 0, 0, 0.04))
							textC = ui.Hex("#1c1c1f")
						}
					} else {
						if t.Dark {
							textC = ui.Hex("#9c9ca6")
						} else {
							textC = ui.Hex("#55555e")
						}
					}
				}

				row.Children(func() {
					txt := ui.Text(c, it.title).
						FontSize(12.5).
						TextColor(textC).
						Grow(1).
						SingleLine()
					if isSelected {
						txt.FontWeight(500)
					}

					// Badge count (e.g. for Skills: 2)
					if it.count > 0 {
						badgeBg := ui.RGBA(0, 0, 0, 0.06)
						badgeTextC := ui.Hex("#666670")
						if t.Dark {
							badgeBg = ui.RGBA(255, 255, 255, 0.10)
							badgeTextC = ui.Hex("#a0a0ab")
						}
						if isSelected {
							if t.Dark {
								badgeBg = ui.RGBA(255, 255, 255, 0.16)
								badgeTextC = ui.Hex("#ffffff")
							} else {
								badgeBg = ui.RGBA(0, 0, 0, 0.08)
								badgeTextC = ui.Hex("#1c1c1f")
							}
						}

						ui.Box(c).
							Height(18).
							Padding(0, 6).
							Radius(9).
							Background(badgeBg).
							Center().
							Shrink(0).
							Children(func() {
								ui.Text(c, fmt.Sprintf("%d", it.count)).
									FontSize(11).
									FontWeight(500).
									TextColor(badgeTextC).
									SingleLine()
							})
					}
				})
			}
		})
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

	// As in sidebarTop: the traffic lights center in the shorter strip
	// TitleBar reports, so the header's content centers on that line, not
	// the whole strip's, to stay level with the lights when the sidebar
	// is hidden.
	lights := h
	if bar.Height > 0 && bar.Height < h {
		lights = bar.Height
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
		Padding(0, 16, h-lights, leftPad).
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

		// Breadcrumbs: Group / Page (or Group / 资源库 / 技能 for Library)
		groupTitle := a.groupTitle(a.pageGroup(a.page))
		pageTitle := a.pageTitle(a.page)

		ui.Row(c).AlignItems(ui.Center).Gap(7).Children(func() {
			if a.page == PageSettings {
				ui.Text(c, pageTitle).
					FontSize(13.5).
					FontWeight(600).
					TextColor(textC).
					SingleLine()
			} else {
				ui.Text(c, groupTitle).
					FontSize(13.5).
					TextColor(mutedC).
					SingleLine()
				ui.Text(c, "/").
					FontSize(13.5).
					TextColor(faintC).
					SingleLine()
				if a.page == PageLibrary {
					var subTitle string
					switch a.libTab {
					case LibTabInstructions:
						subTitle = a.text.S.LibInstructions
					case LibTabMCP:
						subTitle = a.text.S.LibMCPServers
					case LibTabSkills:
						subTitle = a.text.S.LibSkills
					case LibTabHooks:
						subTitle = a.text.S.LibHooks
					case LibTabSubagents:
						subTitle = a.text.S.LibSubagents
					case LibTabRTK:
						subTitle = a.text.S.LibRTK
					default:
						subTitle = a.text.S.LibSkills
					}
					ui.Text(c, pageTitle).
						FontSize(13.5).
						TextColor(mutedC).
						SingleLine()
					ui.Text(c, "/").
						FontSize(13.5).
						TextColor(faintC).
						SingleLine()
					ui.Text(c, subTitle).
						FontSize(13.5).
						FontWeight(600).
						TextColor(textC).
						SingleLine()
				} else {
					ui.Text(c, pageTitle).
						FontSize(13.5).
						FontWeight(600).
						TextColor(textC).
						SingleLine()
				}
			}
		})

		ui.Spacer(c)

		if a.loading {
			ui.Spinner(c).Size(14, 14)
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
				case PageLibrary:
					a.libraryView(c)
				case PageProviders:
					a.providersView(c)
				case PagePlugins:
					a.pluginsView(c)
				case PageGateway:
					a.gatewayView(c)
				case PageRouting:
					a.routingView(c)
				case PageUsage:
					a.usageView(c)
				case PageSettings:
					a.settingsView(c)
				default:
					a.placeholderView(c)
				}
			})
		})
	}

// libraryView renders the Library page based on active subtab (default: skills).
func (a *App) libraryView(c *ui.Context) {
	if a.libTab == "" {
		a.libTab = LibTabSkills
	}
	switch a.libTab {
	case LibTabSkills:
		a.skillsView(c)
	default:
		a.libPlaceholderView(c)
	}
}

// libPlaceholderView renders coming soon for unbuilt library tabs.
func (a *App) libPlaceholderView(c *ui.Context) {
	t := c.Theme()
	var tabTitle string
	switch a.libTab {
	case LibTabInstructions:
		tabTitle = a.text.S.LibInstructions
	case LibTabMCP:
		tabTitle = a.text.S.LibMCPServers
	case LibTabHooks:
		tabTitle = a.text.S.LibHooks
	case LibTabSubagents:
		tabTitle = a.text.S.LibSubagents
	case LibTabRTK:
		tabTitle = a.text.S.LibRTK
	default:
		tabTitle = string(a.libTab)
	}
	ui.Column(c).Padding(t.Space(6), t.Space(5)).Gap(t.Space(2)).Children(func() {
		ui.Text(c, a.text.S.Library+" › "+tabTitle).FontSize(13).Bold().TextColor(t.TextMuted)
		ui.Text(c, a.text.S.ComingSoon).FontSize(15)
		ui.Text(c, a.text.S.Skeleton).FontSize(12).TextColor(t.TextMuted)
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

// providersView is the Providers page: the vendors as rows — logo, name,
// the agents on it, its key or plan, its switch — filtered by the
// segments (all, the ones on, the ones off), searched, a deprecation
// notice over the list when one is due it, and ＋ Add provider beside the
// search.
//
// It is magpie's own Providers page (app.js's renderProviders) drawn in
// native UI: the same rows in the same order with the same pill, and the
// same writes behind the clicks (provider.SetOff, provider.SetOrder).
// The segments and the head's arrangement are Echo's own: magpie folds
// the ones off under the list, Echo's head filters them the way its
// Usage page filters its spans of time.
func (a *App) providersView(c *ui.Context) {
	if len(a.data.Providers) == 0 {
		// magpie's empty page is not bare: its discovery hint, full size,
		// shows over it when other apps hold configurations to bring in.
		t := c.Theme()
		col := ui.Column(c).Fill()
		col.Children(func() {
			a.providersDiscovery(c, false)
			ui.Text(c, a.text.S.NoData).FontSize(13).TextColor(t.TextMuted).Padding(t.Space(4), t.Space(5))
		})
		a.pageDiscoveryKick()
		return
	}

	// The page's own state, held on the app: the search, the filter, and
	// the fold. (Fields of a, so they survive the frame the way a page's
	// own state does in magpie.)

	col := ui.Column(c).Fill()
	col.Children(func() {
		// the head: the segments at the left, the search and ＋ Add
		// provider at the right, as one row the list hangs from.
		//
		// It sits below the header's breadcrumb line, not against it: a
		// gap above and a hairline under it say where the header ends and
		// the page's own controls begin, the way magpie's row-head sits
		// under its top bar.
		t := c.Theme()
		lineC := ui.RGBA(0, 0, 0, 0.08)
		if t.Dark {
			lineC = ui.RGBA(255, 255, 255, 0.08)
		}
		ui.Column(c).FillWidth().BorderWidth(0, 0, 0.5, 0).BorderColor(lineC).Children(func() {
			ui.Row(c).AlignItems(ui.Center).Gap(8).Padding(14, 16, 12, 16).Children(func() {
				a.providerSegments(c)
				ui.Spacer(c)
				ui.SearchField(c, &a.pvQuery).Width(220).Label(a.text.S.FindProvider)
				if ui.Button(c, a.text.S.AddProvider).Clicked() {
					a.openAdd()
				}
			})
		})

		// the rows the filter keeps: every segment is a plain list, the
		// "all" one simply the ones on followed by the ones off — the
		// fold and the footer's import link are not drawn
		switch a.pvFilter {
		case 1: // the ones on
			a.providersList(c, false)
		case 2: // the ones off
			a.providersList(c, true)
		default: // all
			a.providersList(c, false)
			a.providersList(c, true)
		}

		// the sign-ins Echo found but leaves alone, that nobody wonders
		// why an agent that is clearly logged in is not in the list
		a.providersExcluded(c)
	})
	a.pageDiscoveryKick()
}

// providerFilter is how many providers each filter keeps, so a segment
// with nothing behind it can be said to.
func (a *App) providerFilter(n int) int {
	offs := 0
	for _, p := range a.data.Providers {
		if p.Off {
			offs++
		}
	}
	switch n {
	case 1:
		return len(a.data.Providers) - offs
	case 2:
		return offs
	}
	return len(a.data.Providers)
}

// providerSegments draws the filter's segments (all, the ones on, the
// ones off), the count of each beside its label as a badge is.
func (a *App) providerSegments(c *ui.Context) {
	labels := []string{
		a.text.S.FilterAll + " (" + fmt.Sprint(a.providerFilter(0)) + ")",
		a.text.S.FilterOn + " (" + fmt.Sprint(a.providerFilter(1)) + ")",
		a.text.S.FilterOff + " (" + fmt.Sprint(a.providerFilter(2)) + ")",
	}
	segs := ui.Segmented(c, &a.pvFilter, labels...)
	if segs.Changed() {
		c.Invalidate()
	}
}

// pageDiscoveryKick asks for the page's own look at other apps'
// configurations, at most once a minute, as magpie's page runs its own
// discoverLocalProviders where it opens or is refreshed. It runs wherever
// the page rendered, so the hint fills in when the look lands.
func (a *App) pageDiscoveryKick() {
	if time.Since(a.pvDiscoveryAt) < time.Minute || a.pvDiscovery.pending {
		return
	}
	a.pvDiscoveryAt = time.Now()
	a.discoverProviders()
}

// providersDiscovery is the discovery hint: what other apps hold, said
// whole over an empty page, and as one compact link beside Add provider
// over a full one (magpie's renderProviderDiscovery, and its aside's two
// shapes). Nothing to offer draws nothing.
func (a *App) providersDiscovery(c *ui.Context, compact bool) {
	t := c.Theme()
	fresh := a.discoveryFresh()
	n := len(fresh)
	if n == 0 {
		return
	}

	// the sources named once each, in the order the look met them
	var sources []string
	seen := map[string]bool{}
	for _, f := range fresh {
		if !seen[f.Source] {
			seen[f.Source] = true
			sources = append(sources, f.Source)
		}
	}

	// the two asks: the whole hint over an empty page, the compact link
	// beside Add provider over a full one; the count differs as magpie's
	// own two labels do
	count := a.text.S.FoundOneConfig
	if n != 1 {
		count = strings.ReplaceAll(a.text.S.FoundNConfigs, "{n}", fmt.Sprint(n))
	}
	review := a.text.S.ReviewAndImport
	if compact {
		review = strings.ReplaceAll(a.text.S.ImportLocalN, "{n}", fmt.Sprint(n))
	}

	if compact {
		row := ui.Row(c).AlignItems(ui.Center).Gap(12)
		row.Children(func() {
			if b := ui.Button(c, review); b.Clicked() {
				a.openImport(fresh)
			}
		})
		return
	}

	card := ui.Row(c).AlignItems(ui.Center).Gap(10).
		Margin(16, 16, 12, 16).Padding(12, 14).Radius(10).
		Border(1, t.Border)
	card.Children(func() {
		ui.Column(c).Grow(1).Shrink(1).Gap(4).Children(func() {
			ui.Text(c, count).FontSize(12.5).FontWeight(500).TextColor(t.Text)
			ui.Text(c, strings.Join(sources, " · ")).FontSize(11.5).TextColor(t.TextMuted)
		})
		if b := ui.Button(c, review); b.Clicked() {
			a.openImport(fresh)
		}
		if b := ui.Button(c, a.text.S.Ignore); b.Clicked() {
			ids := make([]string, 0, n)
			for _, f := range fresh {
				ids = append(ids, f.Fingerprint)
			}
			ignoredSave(ids)
			c.Invalidate()
		}
	})
}

// providersExcluded draws the sign-ins Echo found but leaves alone, each
// with its way out (magpie's renderExcluded): one brought back as it was,
// one put away until its configuration changes, one signed out for good.
func (a *App) providersExcluded(c *ui.Context) {
	t := c.Theme()
	for _, x := range provider.Excluded() {
		if x.Quiet {
			continue // dismissed: the Add sheet offers it back (#116)
		}
		name := x.Agent
		if x.Name != "" {
			name = x.Name
		}
		head := a.text.S.ExcludedKeptOut
		if x.SignedOut {
			head = a.text.S.ExcludedRemoved
		}
		head = strings.ReplaceAll(head, "{agent}", name)

		row := ui.Row(c).AlignItems(ui.Center).Gap(8).
			Padding(8, 16, 8, 16)
		row.Children(func() {
			ui.Text(c, head).FontSize(11.5).TextColor(t.TextMuted).Shrink(1)
			if x.Provider != "" {
				if b := ui.Button(c, a.text.S.AddBack); b.Clicked() {
					a.providerAction(x.Provider, a.text.S.AddBack)
				}
				if b := ui.Button(c, a.text.S.DontRemind); b.Clicked() {
					if err := provider.QuietAccount(x.Provider); err == nil {
						a.Reload()
					}
				}
				if b := ui.Button(c, a.text.S.SignOut); b.Clicked() {
					a.askSignOut(name, x.Provider)
				}
			}
		})
	}
}

// providerAction does a provider's own write off the main thread and
// reloads the page when it lands (magpie's providerAction: "show" brings
// a removed account back as it was). A failure says so and nothing reloads.
func (a *App) providerAction(id, done string) {
	go func() {
		err := provider.ShowAccount(id)
		if a.win == nil {
			return
		}
		a.win.Update(func() {
			if err != nil {
				_ = err
			}
			a.Reload()
		})
	}()
}

// pvWords is the search's words, lower-cased, as findProviders cuts them.
func (a *App) pvWords() []string {
	return strings.FieldsFunc(strings.ToLower(a.pvQuery), func(r rune) bool { return r == ' ' || r == '\t' })
}

// providersList draws the rows: the ones on, or the ones off when off is
// set. A row is a card in the list's card: magpie's .row.provider, its
// hairline between neighbours and all.
func (a *App) providersList(c *ui.Context, off bool) {
	t := c.Theme()
	words := a.pvWords()
	ui.Column(c).Padding(0, 16, 0, 16).Children(func() {
		drawn := 0
		for _, p := range a.data.Providers {
			p := p
			if p.Off != off {
				continue
			}
			// while searching, only the rows the words match show: the
			// same search magpie's findProviders makes over the row's
			// own text, its id, host, preset and account
			if len(words) > 0 && !providerHit(p, words) {
				continue
			}
			a.providerRow(c, p, t, drawn > 0)
			drawn++
		}
	})
}

// providerHit is whether a row matches the search's words: over its name,
// id, host and preset, and what its account and key say, as magpie's
// providerText gathers.
func providerHit(p ProviderRow, words []string) bool {
	hay := strings.ToLower(strings.Join([]string{
		p.Name, p.ID, p.Host, p.Preset, p.Key, p.Plan, p.User,
	}, " "))
	for _, w := range words {
		if !strings.Contains(hay, w) {
			return false
		}
	}
	return true
}

// providerRow draws one vendor's row.
func (a *App) providerRow(c *ui.Context, p ProviderRow, t *ui.Theme, sep bool) {
	dark := t.Dark
	row := ui.Row(c).Key("pv/"+p.ID).
		AlignItems(ui.Center).
		Gap(10).
		MinHeight(52).
		Padding(6, 12, 6, 12).
		Radius(10).
		Margin(0, 0, 2, 0).
		Cursor(ui.CursorPointer).
		Focusable().
		Role(ui.RoleButton)

	// the hairline between neighbours, magpie's `.row + .row` rule: the
	// line rides each row's top but the list's first
	if sep {
		lineC := ui.RGBA(0, 0, 0, 0.07)
		if dark {
			lineC = ui.RGBA(255, 255, 255, 0.07)
		}
		row.BorderWidth(0.5, 0, 0, 0).BorderColor(lineC)
	}

	if row.Clicked() {
		// the row opens the provider's editor — its details, its models
		// and its limits (magpie's own row opens its editor the same way)
		a.openProviderEditor(p.ID)
		c.Invalidate()
	}
	if row.Hovered() || row.Focused() {
		if dark {
			row.Background(ui.RGBA(255, 255, 255, 0.04))
		} else {
			row.Background(ui.RGBA(0, 0, 0, 0.035))
		}
	}
	if p.Off {
		// a row switched off dims everything under its name (app.css's
		// .provider.off), and its name goes muted
		row.Opacity(0.55)
	}

	row.Children(func() {
		// the logo: the row's handle in magpie's own, whose drag reorders
		// the list. Dragging is not built; the picture is the same.
		pic := iconPictureFor(p.Icon)
		tile := ui.Row(c).Center().Width(30).Height(30).Radius(8)
		if dark {
			tile.Background(ui.RGBA(255, 255, 255, 0.06))
		} else {
			tile.Background(ui.RGBA(0, 0, 0, 0.045))
		}
		tile.Children(func() {
			if pic != nil {
				ui.Image(c, pic).Size(18, 18).Fit(ui.Contain)
			} else {
				ui.Icon(c, genericIcon()).Size(16, 16).TextColor(t.TextMuted)
			}
		})

		// who: the name, its badges, and under it the sub line
		ui.Column(c).Grow(1).Shrink(1).Gap(2).Children(func() {
			ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
				ui.Text(c, p.Name).FontSize(13).FontWeight(500).TextColor(t.Text).SingleLine()
				if p.Sponsored {
					ui.Text(c, a.text.S.Sponsored).FontSize(9.5).FontWeight(600).TextColor(ui.Hex("#b45309")).Background(ui.RGBA(0, 0, 0, 0.06)).Padding(1, 5).Radius(4)
				}
				if p.Off {
					ui.Text(c, a.text.S.SwitchedOffBadge).FontSize(9.5).FontWeight(600).TextColor(t.TextMuted).Background(ui.RGBA(0, 0, 0, 0.06)).Padding(1, 5).Radius(4)
				}
			})
			sub := a.providerSub(p)
			ui.Text(c, sub).FontSize(11.5).TextColor(t.TextMuted).SingleLine()
		})

		// the agents on it now, their icons in tiles
		if len(p.Agents) > 0 {
			ui.Row(c).AlignItems(ui.Center).Gap(4).Children(func() {
				for _, u := range p.Agents {
					tile := ui.Row(c).Center().Width(26).Height(26).Radius(8)
					if dark {
						tile.Background(ui.RGBA(255, 255, 255, 0.06))
					} else {
						tile.Background(ui.RGBA(0, 0, 0, 0.045))
					}
					tile.Children(func() {
						if ap := iconPictureFor(u.Icon); ap != nil {
							ui.Image(c, ap).Size(16, 16).Fit(ui.Contain)
						} else {
							ui.Icon(c, genericIcon()).Size(14, 14).TextColor(t.TextMuted)
						}
					})
				}
			})
		}

		// the key pill: one width so the dots line up down the list, as
		// magpie's .key is 128px for
		a.keyPill(c, p, t)

		// the switch, and the chevron
		on := !p.Off
		if sw := ui.Switch(c, &on); sw.Clicked() {
			id := p.ID
			go func(on bool) {
				if err := SwitchProvider(id, on); err != nil {
					// the save failed: the next read puts the row back
					_ = err
				}
				if a := Current(); a != nil && a.win != nil {
					a.win.Update(func() { a.Reload() })
				}
			}(on)
		}
		ui.Icon(c, chevRight()).Size(11, 11).TextColor(t.TextMuted)
	})
}

// providerSub is the row's second line: where it is signed in, or its host,
// and how many models it serves — magpie's own sub line.
func (a *App) providerSub(p ProviderRow) string {
	var who string
	switch {
	case p.SignedIn && p.Lapsed:
		who = strings.ReplaceAll(a.text.S.SignedOut, "", "")
	case p.SignedIn:
		who = strings.ReplaceAll(a.text.S.SignedInAs, "{user}", p.User)
	default:
		who = p.Host
	}
	models := a.text.S.NoModelsExposed
	switch {
	case p.Models == 1:
		models = strings.ReplaceAll(a.text.S.NModels, "{n}", "1")
	case p.Models > 1:
		models = strings.ReplaceAll(a.text.S.NModels, "{n}", fmt.Sprint(p.Models))
	case p.Draws == 1:
		models = strings.ReplaceAll(a.text.S.NImageModels, "{n}", "1")
	case p.Draws > 1:
		models = strings.ReplaceAll(a.text.S.NImageModels, "{n}", fmt.Sprint(p.Draws))
	}
	return who + " · " + models
}

// keyPill is the row's key status: a dot and the key or plan, in a pill one
// width long so the dots line up down the list (magpie's .key).
func (a *App) keyPill(c *ui.Context, p ProviderRow, t *ui.Theme) {
	var (
		label string
		dot   ui.Color
		textC ui.Color
		bg    ui.Color
	)
	if t.Dark {
		dot, textC, bg = ui.Hex("#34c759"), ui.Hex("#b9b9c1"), ui.RGBA(255, 255, 255, 0.06)
	} else {
		dot, textC, bg = ui.Hex("#34c759"), ui.Hex("#6b6b74"), ui.RGBA(0, 0, 0, 0.045)
	}
	switch {
	case p.SignedIn && p.Lapsed:
		label = a.text.S.SignedOut
		if t.Dark {
			dot, textC, bg = ui.Hex("#d6a13a"), ui.Hex("#d6a13a"), ui.RGBA(214, 161, 58, 0.14)
		} else {
			dot, textC, bg = ui.Hex("#b45309"), ui.Hex("#b45309"), ui.RGBA(180, 83, 9, 0.10)
		}
	case p.SignedIn:
		label = p.Plan
	case p.Keyed:
		label = p.Key
	case p.Ready:
		label = a.text.S.NoKey
		if t.Dark {
			dot, textC = ui.Hex("#8b8b94"), ui.Hex("#8b8b94")
		} else {
			dot, textC = ui.Hex("#9a9aa3"), ui.Hex("#9a9aa3")
		}
	default:
		label = a.text.S.NeedsKey
		if t.Dark {
			dot, textC, bg = ui.Hex("#d6a13a"), ui.Hex("#d6a13a"), ui.RGBA(214, 161, 58, 0.14)
		} else {
			dot, textC, bg = ui.Hex("#b45309"), ui.Hex("#b45309"), ui.RGBA(180, 83, 9, 0.10)
		}
	}
	pill := ui.Row(c).AlignItems(ui.Center).Gap(6).Width(120).Height(24).
		Padding(0, 8).Radius(6).Background(bg)
	pill.Children(func() {
		ui.Row(c).Width(6).Height(6).Radius(3).Background(dot)
		ui.Text(c, label).FontSize(11).TextColor(textC).Grow(1).Shrink(1).SingleLine()
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
		if len(a.data.Plugins.Installed) > 0 {
			return a.data.Plugins.Installed[row].Name
		}
		if row < len(a.data.Plugins.Builtins) {
			return a.data.Plugins.Builtins[row].Name
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
		return len(a.data.Plugins.Installed), len(a.data.Plugins.Installed) > 0
	}
	return 0, false
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

// gatewayView is the Gateway page: the status card, the connect sheet, the
// models list and the recent calls. It is magpie's own page (app.js's
// renderGateway) drawn in native UI.
func (a *App) gatewayView(c *ui.Context) {
	t := c.Theme()
	g := a.data.Gateway

	ui.Column(c).Fill().Padding(8, 24, 0, 24).Children(func() {
		a.gatewayCard(c, t, g)
		a.gatewayConnect(c, t, g)
		a.gatewayModelsList(c, t, g)
		a.gatewayActivity(c, t, g)
	})
}

// gatewayCard is the status card: a dot, the state, the URL.
func (a *App) gatewayCard(c *ui.Context, t *ui.Theme, g GatewayRow) {
	older := g.Running && !g.Mine && g.Older

	dotC := ui.Hex("#d6a13a")
	dotBg := ui.RGBA(214, 161, 58, 0.14)
	if g.Running {
		dotC = ui.Hex("#34c759")
		dotBg = ui.RGBA(52, 199, 89, 0.14)
	}

	// lighter textbox background: between page bg and border.
	fieldBg := a.fieldBg(t)

	card := ui.Row(c).AlignItems(ui.Center).Gap(10).
		Padding(10, 14).Margin(4, 16, 2, 16).Radius(10).
		Border(1, t.Border)
	card.Children(func() {
		ui.Row(c).Width(8).Height(8).Radius(4).Background(dotBg)

		ui.Column(c).Grow(1).Shrink(1).Gap(2).Children(func() {
			ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
				ui.Text(c, a.text.S.Gateway).FontSize(13).FontWeight(600).TextColor(t.Text).SingleLine()
				state := a.text.S.GwNotRunning
				if g.Running {
					state = a.text.S.GwRunning
					if older {
						state = strings.ReplaceAll(a.text.S.GwRunningOlder, "{v}", g.Version)
					} else if !g.Mine {
						state = a.text.S.GwRunningOther
					}
				}
				ui.Text(c, state).FontSize(11).FontWeight(500).TextColor(dotC).SingleLine()
			})
			sub := a.text.S.GwStartIt
			if older {
				sub = strings.ReplaceAll(a.text.S.GwOlderSub, "{v}", g.Version)
			} else if g.Running {
				modelsStr := strings.ReplaceAll(a.text.S.GwNModels, "{n}", fmt.Sprint(g.Models))
				agentsStr := a.text.S.GwNoAgents
				if g.Routed > 0 {
					agentsStr = strings.ReplaceAll(a.text.S.GwNAgents, "{n}", fmt.Sprint(g.Routed))
				}
				sub = modelsStr + " · " + agentsStr + " · " + a.text.S.GwFiveAPIs
			}
			ui.Text(c, sub).FontSize(11).TextColor(t.TextMuted).SingleLine()
		})

		urlPill := ui.Row(c).AlignItems(ui.Center).Gap(6).
			Padding(0, 8).Height(24).Radius(6).
			Background(fieldBg).
			Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).
			Label(a.text.S.GwCopyURL)
		if urlPill.Clicked() {
			a.copyToClipboard(g.URL)
		}
		urlPill.Children(func() {
			ui.Text(c, g.URL).Font("monospace").FontSize(11).TextColor(t.TextMuted).SingleLine()
		})

		// The restart/start button: shown only when there is something to
		// do here — older magpie, gateway not running, or this process
		// serving it. A magpie of the same version already serving it
		// gets no button.
		a.gatewayFixButton(c, t, g)
	})
}

// fieldBg is the colour of a connect-form text box: just above the page
// background, just below the border. The same shade every box uses so
// the form reads as one piece.
func (a *App) fieldBg(t *ui.Theme) ui.Color {
	if t.Dark {
		return ui.Hex("#1f1f23")
	}
	return ui.Hex("#f5f5f7")
}

// gatewayFixButton draws the card's restart/start button when there is one
// to draw: when the gateway isn't running here, or when another (older)
// magpie serves it.
//
// A click asks magpie's gateway fix to stop this process's server and
// start one afresh — the same call /api/gateway/restart takes — and asks
// it to take over when an older magpie sits on the port. While the call
// runs the button shows its in-flight label and can't be clicked again.
func (a *App) gatewayFixButton(c *ui.Context, t *ui.Theme, g GatewayRow) {
	older := g.Running && !g.Mine && g.Older
	// nothing to fix on this side: a same-version magpie already serves
	// it, the gateway is running and we are not — leave only the URL pill.
	if g.Running && !g.Mine && !older {
		return
	}
	label := a.text.S.GwStart
	if g.Running {
		label = a.text.S.GwRestart
	}
	if a.gwFixBusy {
		if g.Running {
			label = a.text.S.GwRestarting
		} else {
			label = a.text.S.GwStarting
		}
	}
	btn := ui.Row(c).AlignItems(ui.Center).Padding(4, 10).Radius(6).
		Border(1, t.Border).
		Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).
		Label(label)
	if btn.Hovered() || btn.Focused() {
		if t.Dark {
			btn.Background(ui.RGBA(255, 255, 255, 0.05))
		} else {
			btn.Background(ui.RGBA(0, 0, 0, 0.04))
		}
	}
	if btn.Clicked() && !a.gwFixBusy {
		a.gwFixBusy = true
		a.invalidate()
		go func() {
			// Stop this process's gateway (no-op if not serving) and
			// start it again. The starter takes over from an older
			// magpie and waits when the port is held by something else.
			gui.StopBackend()
			gui.StartBackend()
			if a.win != nil {
				a.win.Update(func() {
					a.gwFixBusy = false
				})
			}
			// read the gateway state again so the card and the model
			// list show what the new server serves.
			a.Reload()
		}()
	}
	btn.Children(func() {
		ui.Text(c, label).FontSize(11).FontWeight(500).TextColor(t.Text).SingleLine()
	})
}

// gatewayConnect is the Connect section: the API picker, base URL, key,
// model and example snippet.
func (a *App) gatewayConnect(c *ui.Context, t *ui.Theme, g GatewayRow) {
	flavor := gwFlavors[a.gwFlavor]

	fieldBg := a.fieldBg(t)

	foldBtn := ui.Row(c).AlignItems(ui.Center).Gap(6).
		Padding(10, 16, 4, 16).Cursor(ui.CursorPointer).
		Focusable().Role(ui.RoleButton).Label(a.text.S.GwConnect)
	if foldBtn.Clicked() {
		a.gwConnectFolded = !a.gwConnectFolded
		a.invalidate()
	}
	foldBtn.Children(func() {
		if a.gwConnectFolded {
			ui.Icon(c, chevRight()).Size(11, 11).TextColor(t.TextMuted)
		} else {
			ui.Icon(c, chevDown()).Size(11, 11).TextColor(t.TextMuted)
		}
		ui.Text(c, a.text.S.GwConnect).FontSize(11.5).FontWeight(500).TextColor(t.TextMuted).SingleLine()
	})

	if a.gwConnectFolded {
		return
	}

	col := ui.Column(c).Padding(0, 16, 8, 16).Gap(8)
	col.Children(func() {
		a.gatewayConnectField(c, t, a.text.S.GwAPI, func() {
			segs := ui.Segmented(c, &a.gwFlavor, flavorLabels()...)
			if segs.Changed() {
				a.invalidate()
			}
		}, flavor.note(a.text))

		base := flavor.base(g.URL)
		a.gatewayConnectField(c, t, a.text.S.GwBaseURL, func() {
			val := ui.Row(c).AlignItems(ui.Center).Gap(6).
				Padding(0, 4, 0, 9).Height(26).Radius(7).
				Background(fieldBg)
			val.Children(func() {
				ui.Text(c, base).FontSize(12).TextColor(t.Text).Grow(1).Shrink(1).SingleLine()
				if b := a.copyBtn(c, t, base, a.text.S.GwBaseURL); b.Clicked() {
					a.copyToClipboard(base)
				}
			})
		}, flavor.baseHint(a.text))

		a.gatewayConnectField(c, t, a.text.S.GwAPIKeyLabel, func() {
			val := ui.Row(c).AlignItems(ui.Center).Gap(6).
				Padding(0, 4, 0, 9).Height(26).Radius(7).
				Background(fieldBg)
			val.Children(func() {
				ui.Text(c, a.text.S.GwKeyMagpie).FontSize(12).TextColor(t.Text).Grow(1).Shrink(1).SingleLine()
				if b := a.copyBtn(c, t, "magpie", a.text.S.GwKey); b.Clicked() {
					a.copyToClipboard("magpie")
				}
			})
		}, flavor.keyHint(a.text))

		model := a.gwExampleModel()
		a.gatewayConnectField(c, t, a.text.S.GwModel, func() {
			val := ui.Row(c).AlignItems(ui.Center).Gap(6).
				Padding(0, 4, 0, 9).Height(26).Radius(7).
				Background(fieldBg)
			val.Children(func() {
				ui.Text(c, model).FontSize(12).TextColor(t.Text).Grow(1).Shrink(1).SingleLine()
				if b := a.copyBtn(c, t, model, a.text.S.GwCopyModelID); b.Clicked() {
					a.copyToClipboard(model)
				}
			})
		}, a.gwModelHint())

		lang := a.gwLang
		a.gatewayConnectField(c, t, a.text.S.GwExample, func() {
			ui.Column(c).Gap(8).Children(func() {
				segs := ui.Segmented(c, &a.gwLang, gwLangLabels()...)
				if segs.Changed() {
					a.invalidate()
				}
				snippet := flavor.snippet(lang, base, model, "magpie")
				snipBox := ui.Row(c).AlignItems(ui.Start).Padding(12, 14).Radius(9).
					Background(fieldBg).Border(1, t.Border).
					MinHeight(120).Gap(10)
				snipBox.Children(func() {
					gwSnippetRich(c, t, snippet, gwSnippetForLang(lang)).Grow(1).Shrink(1)
					if b := a.copyBtn(c, t, snippet, a.text.S.GwSnippet).Shrink(0); b.Clicked() {
						a.copyToClipboard(snippet)
					}
				})
			})
		}, a.gwShellNote(lang))
	})
}

// gatewayConnectField draws one Connect row: label, value, hint.
func (a *App) gatewayConnectField(c *ui.Context, t *ui.Theme, label string, value func(), hint string) {
	ui.Row(c).Gap(14).AlignItems(ui.Start).Children(func() {
		ui.Text(c, label).FontSize(12).TextColor(t.TextMuted).
			Width(72).TextAlign(ui.End).Padding(4, 0, 0, 0).SingleLine()
		ui.Column(c).Grow(1).Shrink(1).Gap(4).Children(func() {
			value()
			if hint != "" {
				ui.Text(c, hint).FontSize(11).TextColor(t.TextMuted).Wrap()
			}
		})
	})
}

// copyBtn is the small copy icon button.
func (a *App) copyBtn(c *ui.Context, t *ui.Theme, _, label string) ui.Element {
	btn := ui.Row(c).Center().Width(22).Height(22).Radius(5).
		Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).
		Label(label)
	if btn.Hovered() || btn.Focused() {
		if t.Dark {
			btn.Background(ui.RGBA(255, 255, 255, 0.08))
		} else {
			btn.Background(ui.RGBA(0, 0, 0, 0.05))
		}
	}
	btn.Children(func() {
		ui.Icon(c, copyIcon()).Size(12, 12).TextColor(t.TextMuted)
	})
	return btn
}

// copyToClipboard writes text to the system clipboard.
func (a *App) copyToClipboard(text string) {
	mygo.Clipboard.Write(transfer.TextData(text))
}

// gatewayModelsList is the Models section: the exposed model ids.
func (a *App) gatewayModelsList(c *ui.Context, t *ui.Theme, g GatewayRow) {
	entries := a.gwFilteredModels()

	ui.Row(c).AlignItems(ui.Center).Gap(6).Padding(10, 16, 4, 16).Children(func() {
		foldBtn := ui.Row(c).AlignItems(ui.Center).Gap(5).
			Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).
			Label(a.text.S.GwModels)
		if foldBtn.Clicked() {
			a.gwModelsFolded = !a.gwModelsFolded
			a.invalidate()
		}
		foldBtn.Children(func() {
			if a.gwModelsFolded {
				ui.Icon(c, chevRight()).Size(11, 11).TextColor(t.TextMuted)
			} else {
				ui.Icon(c, chevDown()).Size(11, 11).TextColor(t.TextMuted)
			}
			ui.Text(c, a.text.S.GwModels).FontSize(11.5).FontWeight(500).TextColor(t.TextMuted).SingleLine()
		})
		if len(g.Entries) > 0 {
			ui.Text(c, fmt.Sprint(len(g.Entries))).FontSize(11).TextColor(t.TextMuted).SingleLine()
		}
		ui.Spacer(c)
		if b := ui.Button(c, a.text.S.GwCopyAllIDs); b.Clicked() {
			ids := make([]string, 0, len(entries))
			for _, e := range entries {
				ids = append(ids, e.ID)
			}
			a.copyToClipboard(strings.Join(ids, "\n"))
		}
	})

	if a.gwModelsFolded && len(g.Entries) > 0 {
		return
	}

	if len(g.Entries) >= 5 && !a.gwModelsFolded {
		ui.Row(c).Padding(0, 16, 4, 16).Children(func() {
			ui.SearchField(c, &a.gwModelQuery).Label(a.text.S.GwFindModel)
		})
	}

	if len(g.Entries) == 0 {
		ui.Column(c).Padding(8, 16).Children(func() {
			ui.Text(c, a.text.S.GwNoModelsExposed).FontSize(12).FontWeight(500).TextColor(t.Text)
			ui.Text(c, a.text.S.GwNoModelsHint).FontSize(11.5).TextColor(t.TextMuted)
		})
		return
	}

	if len(entries) == 0 {
		ui.Text(c, strings.ReplaceAll(a.text.S.GwNoModelsMatch, "{q}", a.gwModelQuery)).
			FontSize(11.5).TextColor(t.TextMuted).Padding(8, 16)
		return
	}

	list := ui.Column(c).Padding(0, 8, 0, 8)
	list.Children(func() {
		for _, m := range entries {
			a.gatewayModelRow(c, t, m)
		}
	})
}

// gatewayModelRow is one model id in the Gateway list.
func (a *App) gatewayModelRow(c *ui.Context, t *ui.Theme, m GatewayModel) {
	selected := m.ID == a.gwExampleModel()
	row := ui.Row(c).Key("gw/"+m.ID).
		AlignItems(ui.Center).Gap(10).
		MinHeight(40).Padding(4, 8).Radius(8).
		Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton)

	if row.Clicked() {
		a.gwModel = m.ID
		a.invalidate()
	}
	if row.Hovered() || row.Focused() || selected {
		if t.Dark {
			row.Background(ui.RGBA(255, 255, 255, 0.04))
		} else {
			row.Background(ui.RGBA(0, 0, 0, 0.035))
		}
	}

	row.Children(func() {
		pic := iconPictureFor(m.Icon)
		tile := ui.Row(c).Center().Width(24).Height(24).Radius(6)
		if t.Dark {
			tile.Background(ui.RGBA(255, 255, 255, 0.06))
		} else {
			tile.Background(ui.RGBA(0, 0, 0, 0.045))
		}
		tile.Children(func() {
			if pic != nil {
				ui.Image(c, pic).Size(16, 16).Fit(ui.Contain)
			} else {
				ui.Icon(c, genericIcon()).Size(14, 14).TextColor(t.TextMuted)
			}
		})

		ui.Column(c).Grow(1).Shrink(1).Gap(1).Children(func() {
			ui.Row(c).AlignItems(ui.Center).Gap(5).Children(func() {
				ui.Text(c, m.ID).Font("monospace").FontSize(12).TextColor(t.Text).
					Grow(1).Shrink(1).SingleLine()
				if m.Free {
					ui.Text(c, a.text.S.GwFree).FontSize(9.5).FontWeight(600).
						TextColor(ui.Hex("#16a34a")).Background(ui.RGBA(22, 163, 74, 0.12)).
						Padding(1, 5).Radius(4)
				}
			})
			sub := gatewaySub(m)
			if sub != "" {
				ui.Text(c, sub).FontSize(11).TextColor(t.TextMuted).SingleLine()
			}
		})

		a.gatewayModelInfo(c, t, m)

		if b := a.copyBtn(c, t, m.ID, a.text.S.GwCopyModelID); b.Clicked() {
			a.copyToClipboard(m.ID)
		}
	})
}

// gatewayModelInfo draws the model's chips: reasoning levels, images,
// context.
func (a *App) gatewayModelInfo(c *ui.Context, t *ui.Theme, m GatewayModel) {
	var chips []string
	if len(m.Efforts) > 0 {
		chips = append(chips, gatewayEffortSpan(m.Efforts))
	}
	if m.Images {
		chips = append(chips, "img")
	}
	if m.Context > 0 {
		chips = append(chips, ctxShort(m.Context))
	}
	if len(chips) == 0 {
		return
	}
	ui.Row(c).AlignItems(ui.Center).Gap(4).Children(func() {
		for _, chip := range chips {
			ui.Text(c, chip).FontSize(9.5).FontWeight(600).TextColor(t.TextMuted).
				Background(t.Border).Padding(0, 5).Radius(4).SingleLine()
		}
	})
}

// gatewayActivity is the Recent calls section.
func (a *App) gatewayActivity(c *ui.Context, t *ui.Theme, g GatewayRow) {
	ui.Row(c).AlignItems(ui.Center).Padding(10, 16, 4, 16).Children(func() {
		ui.Text(c, a.text.S.GwRecentCalls).FontSize(11.5).FontWeight(500).TextColor(t.TextMuted).SingleLine()
		ui.Spacer(c)
		if g.Running && !g.Mine {
			ui.Text(c, a.text.S.GwCallsByOther).FontSize(11).TextColor(t.TextMuted).SingleLine()
		}
	})

	calls := g.Calls
	if len(calls) > a.gwCallsShown {
		calls = calls[:a.gwCallsShown]
	}

	if len(calls) == 0 {
		ui.Text(c, a.text.S.GwNoRequests).FontSize(11.5).TextColor(t.TextMuted).
			Padding(8, 16)
		return
	}

	list := ui.Column(c).Padding(0, 8, 0, 8)
	list.Children(func() {
		for _, call := range calls {
			a.gatewayCallRow(c, t, call)
		}
	})

	if len(g.Calls) > len(calls) {
		n := len(g.Calls) - len(calls)
		if n > 20 {
			n = 20
		}
		btnRow := ui.Row(c).Center().Padding(2, 16, 6, 16)
		btnRow.Children(func() {
			if b := ui.Button(c, strings.ReplaceAll(a.text.S.GwShowMore, "{n}", fmt.Sprint(n))); b.Clicked() {
				a.gwCallsShown += 20
				a.invalidate()
			}
		})
	}
}

// gatewayCallRow is one recent call.
func (a *App) gatewayCallRow(c *ui.Context, t *ui.Theme, call GatewayCall) {
	row := ui.Row(c).AlignItems(ui.Center).Gap(10).
		MinHeight(24).Padding(2, 8).Radius(6)
	if row.Hovered() {
		if t.Dark {
			row.Background(ui.RGBA(255, 255, 255, 0.04))
		} else {
			row.Background(ui.RGBA(0, 0, 0, 0.035))
		}
	}
	row.Children(func() {
		ui.Icon(c, chevRight()).Size(10, 10).TextColor(t.TextMuted)
		ui.Text(c, call.Time.Format("15:04:05")).FontSize(11).TextColor(t.TextMuted).SingleLine()
		agent := call.Agent
		if agent == "" {
			agent = "—"
		}
		ui.Text(c, agent).FontSize(11.5).TextColor(t.TextMuted).
			MaxWidth(90).SingleLine()
		ui.Text(c, call.Model).Font("monospace").FontSize(11).TextColor(t.Text).
			Grow(1).Shrink(1).SingleLine()
		ui.Text(c, gatewayRoute(call.From, call.To)).FontSize(11).TextColor(t.TextMuted).SingleLine()
		ui.Spacer(c)
		statusC := ui.Hex("#16a34a")
		if call.Status >= 400 {
			statusC = ui.Hex("#dc2626")
		}
		status := fmt.Sprintf("%d · %s", call.Status, gatewayTook(call.Millis))
		if call.Error != "" {
			status = fmt.Sprintf("%d %s", call.Status, call.Error)
		} else if call.TTFT > 0 {
			status += " · " + strings.ReplaceAll(a.text.S.GwTTFT, "{ms}", gatewayTook(call.TTFT))
		}
		ui.Text(c, status).FontSize(11).TextColor(statusC).SingleLine()
	})
}

// ---- helpers --------------------------------------------------------------

// gwFilteredModels returns the models matching the search words.
func (a *App) gwFilteredModels() []GatewayModel {
	words := strings.FieldsFunc(strings.ToLower(a.gwModelQuery), func(r rune) bool { return r == ' ' || r == '\t' })
	if len(words) == 0 {
		return a.data.Gateway.Entries
	}
	var out []GatewayModel
	for _, m := range a.data.Gateway.Entries {
		hay := strings.ToLower(m.ID + " " + m.Name + " " + m.Provider)
		match := true
		for _, w := range words {
			if !strings.Contains(hay, w) {
				match = false
				break
			}
		}
		if match {
			out = append(out, m)
		}
	}
	return out
}

// gwExampleModel returns the model chosen for the snippets.
func (a *App) gwExampleModel() string {
	if a.gwModel != "" {
		return a.gwModel
	}
	if len(a.data.Gateway.Entries) > 0 {
		return a.data.Gateway.Entries[0].ID
	}
	return "provider/model"
}

// gwModelHint returns the hint under the model field.
func (a *App) gwModelHint() string {
	if len(a.data.Gateway.Entries) > 0 {
		return a.text.S.GwModelHint
	}
	return a.text.S.GwNoModelsHint
}

// gwShellNote returns the note under the example.
func (a *App) gwShellNote(lang int) string {
	if lang == 0 {
		return a.text.S.GwShellNote
	}
	return ""
}

// gatewayEffortSpan formats reasoning levels as a short string.
func gatewayEffortSpan(levels []string) string {
	if len(levels) > 2 {
		return levels[0] + "–" + levels[len(levels)-1]
	}
	return strings.Join(levels, " / ")
}

// ctxShort formats a token count as K or M.
func ctxShort(n int) string {
	if n == 0 {
		return ""
	}
	if n >= 1_000_000 {
		if n%1_000_000 != 0 {
			return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
		}
		return fmt.Sprintf("%dM", n/1_000_000)
	}
	return fmt.Sprintf("%dK", (n+500)/1000)
}

// ---- flavors and languages -------------------------------------------------

// gwFlavor describes one API the gateway serves.
type gwFlavor struct {
	name     string
	base     func(url string) string
	note     func(t T) string
	baseHint func(t T) string
	keyHint  func(t T) string
	snippet  func(lang int, base, model, key string) string
}

// gwFlavors are the APIs in the order the picker offers them.
var gwFlavors = []gwFlavor{
	{
		name:     "OpenAI",
		base:     func(u string) string { return u + "/v1" },
		note:     func(t T) string { return t.S.GwAPIOpenAINote },
		baseHint: func(t T) string { return strings.ReplaceAll(t.S.GwBaseURLEnvHint, "{env}", "OPENAI_BASE_URL") },
		keyHint:  func(t T) string { return strings.ReplaceAll(t.S.GwAPIKeyLoopbackHint, "{env}", "OPENAI_API_KEY") },
		snippet:  gwOpenAISnippet,
	},
	{
		name:     "Responses",
		base:     func(u string) string { return u + "/v1" },
		note:     func(t T) string { return t.S.GwAPIResponsesNote },
		baseHint: func(t T) string { return strings.ReplaceAll(t.S.GwBaseURLEnvHint, "{env}", "OPENAI_BASE_URL") },
		keyHint:  func(t T) string { return strings.ReplaceAll(t.S.GwAPIKeyLoopbackHint, "{env}", "OPENAI_API_KEY") },
		snippet:  gwResponsesSnippet,
	},
	{
		name:     "Anthropic",
		base:     func(u string) string { return u },
		note:     func(t T) string { return t.S.GwAPIAnthropicNote },
		baseHint: func(t T) string { return strings.ReplaceAll(t.S.GwBaseURLEnvHint, "{env}", "ANTHROPIC_BASE_URL") },
		keyHint:  func(t T) string { return strings.ReplaceAll(t.S.GwAPIKeyLoopbackHint, "{env}", "ANTHROPIC_API_KEY") },
		snippet:  gwAnthropicSnippet,
	},
	{
		name:     "Gemini",
		base:     func(u string) string { return u },
		note:     func(t T) string { return t.S.GwAPIGeminiNote },
		baseHint: func(t T) string { return strings.ReplaceAll(t.S.GwBaseURLEnvHint, "{env}", "GOOGLE_GEMINI_BASE_URL") },
		keyHint:  func(t T) string { return strings.ReplaceAll(t.S.GwAPIKeyLoopbackHint, "{env}", "GEMINI_API_KEY") },
		snippet:  gwGeminiSnippet,
	},
	{
		name:     "System One",
		base:     func(u string) string { return u + "/v1" },
		note:     func(t T) string { return t.S.GwAPISystemOneNote },
		baseHint: func(t T) string { return t.S.GwBaseURLSystemOne },
		keyHint:  func(t T) string { return strings.ReplaceAll(t.S.GwAPIKeyLoopbackHint, "{env}", "SYSTEMONE_API_KEY") },
		snippet:  gwSystemOneSnippet,
	},
}

// flavorLabels returns the API picker's labels.
func flavorLabels() []string {
	labels := make([]string, len(gwFlavors))
	for i, f := range gwFlavors {
		labels[i] = f.name
	}
	return labels
}

// gwLangLabels returns the snippet language picker's labels.
func gwLangLabels() []string {
	return []string{"Shell", "curl", "Python", "Node"}
}

// gwLangs are the snippet languages.
var gwLangs = []string{"shell", "curl", "python", "node"}

// Snippet generators per flavor.

func gwOpenAISnippet(lang int, base, model, key string) string {
	switch gwLangs[lang] {
	case "shell":
		return fmt.Sprintf("export OPENAI_BASE_URL=%s\nexport OPENAI_API_KEY=%s", base, key)
	case "curl":
		return fmt.Sprintf("curl %s/chat/completions \\\n  -H \"Authorization: Bearer %s\" \\\n  -H \"Content-Type: application/json\" \\\n  -d '{\"model\": \"%s\", \"messages\": [{\"role\": \"user\", \"content\": \"hi\"}]}'", base, key, model)
	case "python":
		return fmt.Sprintf("from openai import OpenAI\n\nclient = OpenAI(base_url=\"%s\", api_key=\"%s\")\nr = client.chat.completions.create(\n    model=\"%s\",\n    messages=[{\"role\": \"user\", \"content\": \"hi\"}],\n)\nprint(r.choices[0].message.content)", base, key, model)
	case "node":
		return fmt.Sprintf("import OpenAI from \"openai\";\n\nconst client = new OpenAI({ baseURL: \"%s\", apiKey: \"%s\" });\nconst r = await client.chat.completions.create({\n  model: \"%s\",\n  messages: [{ role: \"user\", content: \"hi\" }],\n});\nconsole.log(r.choices[0].message.content);", base, key, model)
	}
	return ""
}

func gwResponsesSnippet(lang int, base, model, key string) string {
	switch gwLangs[lang] {
	case "shell":
		return fmt.Sprintf("export OPENAI_BASE_URL=%s\nexport OPENAI_API_KEY=%s", base, key)
	case "curl":
		return fmt.Sprintf("curl %s/responses \\\n  -H \"Authorization: Bearer %s\" \\\n  -H \"Content-Type: application/json\" \\\n  -d '{\"model\": \"%s\", \"input\": \"hi\"}'", base, key, model)
	case "python":
		return fmt.Sprintf("from openai import OpenAI\n\nclient = OpenAI(base_url=\"%s\", api_key=\"%s\")\nr = client.responses.create(model=\"%s\", input=\"hi\")\nprint(r.output_text)", base, key, model)
	case "node":
		return fmt.Sprintf("import OpenAI from \"openai\";\n\nconst client = new OpenAI({ baseURL: \"%s\", apiKey: \"%s\" });\nconst r = await client.responses.create({ model: \"%s\", input: \"hi\" });\nconsole.log(r.output_text);", base, key, model)
	}
	return ""
}

func gwAnthropicSnippet(lang int, base, model, key string) string {
	switch gwLangs[lang] {
	case "shell":
		return fmt.Sprintf("export ANTHROPIC_BASE_URL=%s\nexport ANTHROPIC_API_KEY=%s", base, key)
	case "curl":
		return fmt.Sprintf("curl %s/v1/messages \\\n  -H \"x-api-key: %s\" \\\n  -H \"anthropic-version: 2023-06-01\" \\\n  -H \"Content-Type: application/json\" \\\n  -d '{\"model\": \"%s\", \"max_tokens\": 1024, \"messages\": [{\"role\": \"user\", \"content\": \"hi\"}]}'", base, key, model)
	case "python":
		return fmt.Sprintf("import anthropic\n\nclient = anthropic.Anthropic(base_url=\"%s\", api_key=\"%s\")\nm = client.messages.create(\n    model=\"%s\",\n    max_tokens=1024,\n    messages=[{\"role\": \"user\", \"content\": \"hi\"}],\n)\nprint(m.content[0].text)", base, key, model)
	case "node":
		return fmt.Sprintf("import Anthropic from \"@anthropic-ai/sdk\";\n\nconst client = new Anthropic({ baseURL: \"%s\", apiKey: \"%s\" });\nconst m = await client.messages.create({\n  model: \"%s\",\n  max_tokens: 1024,\n  messages: [{ role: \"user\", content: \"hi\" }],\n});\nconsole.log(m.content[0].text);", base, key, model)
	}
	return ""
}

func gwGeminiSnippet(lang int, base, model, key string) string {
	switch gwLangs[lang] {
	case "shell":
		return fmt.Sprintf("export GOOGLE_GEMINI_BASE_URL=%s\nexport GEMINI_API_KEY=%s", base, key)
	case "curl":
		return fmt.Sprintf("curl %s/v1beta/models/%s:generateContent \\\n  -H \"x-goog-api-key: %s\" \\\n  -H \"Content-Type: application/json\" \\\n  -d '{\"contents\": [{\"parts\": [{\"text\": \"hi\"}]}]}'", base, model, key)
	case "python":
		return fmt.Sprintf("from google import genai\n\nclient = genai.Client(api_key=\"%s\", http_options={\"base_url\": \"%s\"})\nr = client.models.generate_content(model=\"%s\", contents=\"hi\")\nprint(r.text)", key, base, model)
	case "node":
		return fmt.Sprintf("import { GoogleGenAI } from \"@google/genai\";\n\nconst ai = new GoogleGenAI({ apiKey: \"%s\", httpOptions: { baseUrl: \"%s\" } });\nconst r = await ai.models.generateContent({ model: \"%s\", contents: \"hi\" });\nconsole.log(r.text);", key, base, model)
	}
	return ""
}

func gwSystemOneSnippet(lang int, base, model, key string) string {
	switch gwLangs[lang] {
	case "shell":
		return ""
	case "curl":
		return fmt.Sprintf("curl %s/systemone \\\n  -H \"Authorization: Bearer %s\" \\\n  -H \"Content-Type: application/json\" \\\n  -d '{\"model\": \"%s\", \"state\": {\"message\": \"My order never arrived, refund it\"}, \"questions\": {\"team\": {\"type\": \"choice\", \"instructions\": \"Which team should answer the message?\", \"criteria\": {\"billing\": \"Payments and refunds\", \"shipping\": \"Deliveries\", \"other\": \"Anything else\"}}}}'", base, key, model)
	case "python":
		return fmt.Sprintf("import requests\n\nr = requests.post(\"%s/systemone\",\n    headers={\"Authorization\": \"Bearer %s\"},\n    json={\n        \"model\": \"%s\",\n        \"state\": {\"message\": \"My order never arrived, refund it\"},\n        \"questions\": {\"team\": {\n            \"type\": \"choice\",\n            \"instructions\": \"Which team should answer the message?\",\n            \"criteria\": {\"billing\": \"Payments and refunds\", \"shipping\": \"Deliveries\", \"other\": \"Anything else\"},\n        }},\n    })\na = r.json()[\"answers\"][\"team\"]\nprint(a[\"choice\"], a[\"confidence\"])", base, key, model)
	case "node":
		return fmt.Sprintf("const r = await fetch(\"%s/systemone\", {\n  method: \"POST\",\n  headers: { Authorization: \"Bearer %s\", \"Content-Type\": \"application/json\" },\n  body: JSON.stringify({\n    model: \"%s\",\n    state: { message: \"My order never arrived, refund it\" },\n    questions: { team: {\n      type: \"choice\",\n      instructions: \"Which team should answer the message?\",\n      criteria: { billing: \"Payments and refunds\", shipping: \"Deliveries\", other: \"Anything else\" },\n    } },\n  }),\n});\nconst a = (await r.json()).answers.team;\nconsole.log(a.choice, a.confidence);", base, key, model)
	}
	return ""
}
