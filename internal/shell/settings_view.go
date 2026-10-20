//go:build nogui && echo

package shell

// The Settings page: magpie's own Settings, drawn in native UI.
//
// Its tabs are magpie's (SET_TABS in app.js): the General (appearance,
// language, start-up, the Codex naming), the Allowances (what the menu bar
// shows and the alerts, the warm-ups and the check-ins), the Network and
// sharing, the Models, the Privacy, the Observability, the Sync and
// backup, the Notifications and the About.
//
// The reads are magpie's too (settings_data.go's readSettings); this file
// and the ones beside it are the drawing and the writes. A write goes
// through saveSettings — which keeps what other pages own — and lands with
// settingsWrite, which reads the page again when it is taken.

import (
	"fmt"
	"runtime"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/yetone/magpie/internal/autostart"
	"github.com/yetone/magpie/internal/settings"
)

// setTabs is the head's tabs, in magpie's order. Notifications is absent
// for now: its one row (the Codex reset alerts) is a settings field this
// tree's settings package doesn't have yet — it came with a later upstream.
var setTabs = []SetTab{
	SetTabGeneral,
	SetTabUsage,
	SetTabNetwork,
	SetTabModels,
	SetTabPrivacy,
	SetTabOTel,
	SetTabSync,
	SetTabAbout,
}

// setTabIdx maps a.setTab to the index the tab strip draws.
func setTabIdx(tab SetTab) int {
	for i, t := range setTabs {
		if t == tab {
			return i
		}
	}
	return 0
}

// setTabLabel is a tab's name in the reader's language.
func (a *App) setTabLabel(tab SetTab) string {
	switch tab {
	case SetTabGeneral:
		return a.text.S.Set.General
	case SetTabUsage:
		return a.text.S.Set.Allowances
	case SetTabNetwork:
		return a.text.S.Set.Network
	case SetTabModels:
		return a.text.S.Set.ModelsTab
	case SetTabPrivacy:
		return a.text.S.Set.Privacy
	case SetTabOTel:
		return a.text.S.Set.OTel
	case SetTabSync:
		return a.text.S.Set.Sync
	case SetTabNotify:
		return a.text.S.Set.Notify
	case SetTabAbout:
		return a.text.S.Set.About
	}
	return string(tab)
}

// settingsView draws the page: the section's name, then the section the
// navigation picked. The sections are the sidebar's own menu items
// (magpie's set, 常规 the only one shown for now), so there is no tab
// strip here.
func (a *App) settingsView(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Padding(8, 24, 0, 24).Gap(14).Children(func() {
		ui.Text(c, a.setTabLabel(a.setTab)).FontSize(13.5).FontWeight(600).TextColor(t.Text).SingleLine()
		switch a.setTab {
		case SetTabUsage:
			a.settingsAllowances(c, t)
		case SetTabNetwork:
			a.settingsNetwork(c, t)
		case SetTabModels:
			a.settingsModels(c, t)
		case SetTabPrivacy:
			a.settingsPrivacy(c, t)
		case SetTabOTel:
			a.settingsOTel(c, t)
		case SetTabSync:
			a.settingsSync(c, t)
		case SetTabAbout:
			a.settingsAbout(c, t)
		default:
			a.settingsGeneral(c, t)
		}
	})
}

// ---- the rows every tab draws ----

// setCard is one group of rows: a titled card holding them, as magpie's
// sections are.
func (a *App) setCard(c *ui.Context, t *ui.Theme, title string, rows func()) {
	ui.Column(c).FillWidth().Gap(0).Padding(0, 0, 0, 0).Children(func() {
		if title != "" {
			ui.Text(c, title).FontSize(12).FontWeight(600).TextColor(t.TextMuted).Padding(0, 0, 6, 0)
		}
		ui.Column(c).FillWidth().Gap(0).Radius(10).Border(1, t.Border).
			Background(a.fieldBg(t)).Children(rows)
	})
}

// setRow draws one preference row: its title and its sub line at the left,
// the control at the right — fixed there, never shifted by the texts'
// length. control, when nil, the row shows the sub alone.
func (a *App) setRow(c *ui.Context, t *ui.Theme, title, sub string, control func()) {
	ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(16).
		Padding(10, 14).BorderWidth(0, 0, 0.5, 0).BorderColor(t.Border).Children(func() {
		ui.Column(c).Grow(1).Shrink(1).Gap(2).Children(func() {
			if title != "" {
				ui.Text(c, title).FontSize(13).FontWeight(500).TextColor(t.Text).Wrap()
			}
			if sub != "" {
				ui.Text(c, sub).FontSize(11.5).TextColor(t.TextMuted).Wrap()
			}
		})
		if control != nil {
			ui.Row(c).Shrink(0).AlignItems(ui.Center).Gap(8).Children(control)
		}
	})
}

// setRowWide is a row whose control takes the width: a field, a list of
// its own. The texts go above, the control below.
func (a *App) setRowWide(c *ui.Context, t *ui.Theme, title, sub string, body func()) {
	ui.Column(c).FillWidth().Gap(6).Padding(10, 14).
		BorderWidth(0, 0, 0.5, 0).BorderColor(t.Border).Children(func() {
		if title != "" || sub != "" {
			ui.Column(c).Gap(2).Children(func() {
				if title != "" {
					ui.Text(c, title).FontSize(13).FontWeight(500).TextColor(t.Text).Wrap()
				}
				if sub != "" {
					ui.Text(c, sub).FontSize(11.5).TextColor(t.TextMuted).Wrap()
				}
			})
		}
		body()
	})
}

// setSegRow is a row whose control is a segment strip. sel is the index
// held by the page; a click writes it and calls pick.
func (a *App) setSegRow(c *ui.Context, t *ui.Theme, title, sub string, sel *int, pick func(i int), labels ...string) {
	a.setRow(c, t, title, sub, func() {
		if ui.Segmented(c, sel, labels...).Changed() {
			if *sel >= 0 {
				pick(*sel)
			}
			c.Invalidate()
		}
	})
}

// setSwitchRow is a row whose control is a switch over on.
func (a *App) setSwitchRow(c *ui.Context, t *ui.Theme, title, sub string, on *bool, pick func(on bool)) {
	a.setRow(c, t, title, sub, func() {
		sw := ui.Switch(c, on).Shrink(0)
		if sw.Clicked() {
			pick(*on)
			c.Invalidate()
		}
	})
}

// setPickRow is a row whose control opens the model picker: the choice's
// own name as the button reads.
func (a *App) setPickRow(c *ui.Context, t *ui.Theme, title, sub, choice string, open func()) {
	a.setRow(c, t, title, sub, func() {
		btn := ui.Row(c).AlignItems(ui.Center).Gap(6).Padding(5, 10).Radius(6).
			Background(a.fieldBg(t)).Border(1, t.Border).
			Cursor(ui.CursorPointer).Role(ui.RoleButton).Label(choice)
		if btn.Hovered() {
			btn.Background(a.hoverBg(t))
		}
		if btn.Clicked() {
			open()
		}
		btn.Children(func() {
			ui.Text(c, choice).FontSize(12).TextColor(t.Text).SingleLine()
			ui.Icon(c, gearIcon()).Size(10, 10).TextColor(t.TextMuted)
		})
	})
}

// setButtonRow is a row of buttons and texts under a title, as the About
// tab's and the sync form's rows are.
func (a *App) setButtonRow(c *ui.Context, t *ui.Theme, title string, body func()) {
	ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(8).Padding(10, 14).
		BorderWidth(0, 0, 0.5, 0).BorderColor(t.Border).Children(func() {
		if title != "" {
			ui.Text(c, title).FontSize(13).FontWeight(500).TextColor(t.Text).Grow(1).Shrink(1).Wrap()
		} else {
			ui.Spacer(c)
		}
		body()
	})
}

// smallBtn is a small labelled button, as the rows' Apply, Copy and Open
// are. It reports a click.
func (a *App) smallBtn(c *ui.Context, t *ui.Theme, label string) bool {
	btn := ui.Row(c).AlignItems(ui.Center).Padding(4, 10).Radius(6).
		Background(a.fieldBg(t)).Border(1, t.Border).
		Cursor(ui.CursorPointer).Role(ui.RoleButton).Label(label)
	if btn.Hovered() {
		btn.Background(a.hoverBg(t))
	}
	clicked := btn.Clicked()
	btn.Children(func() {
		ui.Text(c, label).FontSize(12).TextColor(t.Text).SingleLine()
	})
	return clicked
}

// setNoticeBar is the page's own notice, telling a write landed or was
// refused, as magpie's status line does.
func (a *App) setNoticeBar(c *ui.Context, t *ui.Theme) {
	if a.setNotice == "" {
		return
	}
	col := ui.Hex("#16a34a")
	if a.setNoticeErr {
		col = ui.Hex("#dc2626")
	}
	ui.Row(c).FillWidth().AlignItems(ui.Center).Padding(6, 12).Radius(8).Children(func() {
		ui.Text(c, a.setNotice).FontSize(12).TextColor(col).Wrap()
	})
}

// setNotice shows a line for three and a half seconds, as magpie's status
// does.
func (a *App) setSetNotice(msg string, isErr bool) {
	if a.setNoticeTimer != nil {
		a.setNoticeTimer.Stop()
	}
	a.setNotice, a.setNoticeErr = msg, isErr
	a.setNoticeTimer = time.AfterFunc(3500*time.Millisecond, func() {
		if a.win == nil {
			return
		}
		a.win.Update(func() {
			a.setNotice = ""
			a.setNoticeTimer = nil
		})
	})
}

// ---- General ----

// applyThemeSource tells the desktop which appearance Echo's window takes:
// the system's, light, or dark (mygo.Theme.SetSource, which the theme and
// every color the view draws follow). Safe from any goroutine.
func applyThemeSource(theme string) {
	src := mygo.ThemeSystem
	switch theme {
	case "light":
		src = mygo.ThemeLight
	case "dark":
		src = mygo.ThemeDark
	}
	mygo.Theme.SetSource(src)
}

// settingsGeneral draws the General tab: the appearance, the language, the
// start-up and the awake, the Codex naming.
func (a *App) settingsGeneral(c *ui.Context, t *ui.Theme) {
	snap := a.data.Settings
	s := snap.S

	themeIdx := map[string]int{"system": 0, "light": 1, "dark": 2}[cmpOr(s.Theme, "system")]
	langIdx := map[string]int{"system": 0, "en": 1, "zh": 2}[cmpOr(s.Lang, "system")]
	sizeIdx := 1
	for i, n := range []int{85, 100, 115, 130} {
		if s.TextSize == n {
			sizeIdx = i
		}
	}
	trayIdx := map[string]int{"panel": 0, "window": 1}[cmpOr(s.Tray, "panel")]
	nameIdx := 0 // off
	if s.PlainNames && s.PlainOwnNames {
		nameIdx = 1 // not on names I set
	} else if s.PlainNames {
		nameIdx = 2
	}
	longIdx := 0
	if s.FullContext {
		longIdx = 1
	}
	login := snap.Login
	awakeIdx := 0
	if s.KeepAwake {
		awakeIdx = 1
		if s.KeepAwakeDisplay {
			awakeIdx = 2
		}
	}
	awakeSub := a.text.S.Set.KeepAwakeSub
	if s.KeepAwake && s.KeepAwakeDisplay {
		awakeSub = a.text.S.Set.KeepAwakeSubDisplay
	}

	ui.Column(c).FillWidth().Gap(14).Children(func() {
		a.setNoticeBar(c, t)

		a.setCard(c, t, a.text.S.Set.Appearance, func() {
			a.setSegRow(c, t, a.text.S.Set.Theme, "", &themeIdx, func(i int) {
				themes := []string{"system", "light", "dark"}
				// the appearance moves at once, as the reader clicked it:
				// the desktop call and the drawn thumb, before the save
				// is even written
				applyThemeSource(themes[i])
				a.data.Settings.S.Theme = themes[i]
				a.settingsWrite(func() error {
					return saveSettings(func(s *settings.Settings) { s.Theme = themes[i] })
				}, nil)
			}, a.text.S.Set.ThemeSystem, a.text.S.Set.ThemeLight, a.text.S.Set.ThemeDark)
			a.setSegRow(c, t, a.text.S.Set.Language, "", &langIdx, func(i int) {
				langs := []string{"system", "en", "zh"}
				// the words switch at once — the save and the page's
				// reload land when they land
				a.lang = resolveLang(langs[i], "")
				a.text = Text(a.lang)
				a.data.Settings.S.Lang = langs[i]
				a.settingsWrite(func() error {
					return saveSettings(func(s *settings.Settings) { s.Lang = langs[i] })
				}, nil)
			}, a.text.S.Set.ThemeSystem, "English", "中文")
			a.setSegRow(c, t, a.text.S.Set.TextSize, a.text.S.Set.TextSizeSub, &sizeIdx, func(i int) {
				sizes := []int{85, 100, 115, 130}
				a.data.Settings.S.TextSize = sizes[i]
				a.settingsWrite(func() error {
					return saveSettings(func(s *settings.Settings) { s.TextSize = sizes[i] })
				}, nil)
			}, "85%", "100%", "115%", "130%")
			a.setSegRow(c, t, a.text.S.Set.TrayIcon, "", &trayIdx, func(i int) {
				trays := []string{"panel", "window"}
				a.data.Settings.S.Tray = trays[i]
				a.settingsWrite(func() error {
					return saveSettings(func(s *settings.Settings) { s.Tray = trays[i] })
				}, nil)
			}, a.text.S.Set.TrayPanel, a.text.S.Set.TrayWindow)
		})

		a.setCard(c, t, a.text.S.Set.StartUp, func() {
			a.setSwitchRow(c, t, a.text.S.Set.LoginStart, a.text.S.Set.LoginSub, &login, func(on bool) {
				a.settingsWrite(func() error { return autostart.Set(on) }, nil)
			})
			if runtime.GOOS == "darwin" {
				a.setSegRow(c, t, a.text.S.Set.KeepAwake, awakeSub, &awakeIdx, func(i int) {
					a.settingsWrite(func() error {
						return saveSettings(func(s *settings.Settings) {
							s.KeepAwake, s.KeepAwakeDisplay = i > 0, i > 1
						})
					}, nil)
				}, a.text.S.Set.Off, a.text.S.Set.On, a.text.S.Set.KeepDisplay)
			}
			if len(snap.Terminals) > 0 {
				// the terminal a resumed session opens in: the app chosen
				// by its .command association, or one named here
				term := cmpOr(s.SessionTerminal, "system")
				termIdx := 0
				labels := []string{a.text.S.Set.TerminalDefault + " (" + snap.TerminalDefault + ")"}
				for i, ap := range snap.Terminals {
					if ap.ID == term {
						termIdx = i + 1
					}
					labels = append(labels, ap.Name)
				}
				a.setSegRow(c, t, a.text.S.Set.SessionTerminal, "", &termIdx, func(i int) {
					id := ""
					if i > 0 && i-1 < len(snap.Terminals) {
						id = snap.Terminals[i-1].ID
					}
					a.settingsWrite(func() error {
						return saveSettings(func(s *settings.Settings) { s.SessionTerminal = id })
					}, nil)
				}, labels...)
			}
		})

		a.setCard(c, t, a.text.S.Set.NamingCard, func() {
			a.setSegRow(c, t, a.text.S.Set.NameSuffix, a.text.S.Set.NameSuffixSub, &nameIdx, func(i int) {
				a.settingsWrite(func() error {
					return saveSettings(func(s *settings.Settings) {
						s.PlainNames = i > 0
						s.PlainOwnNames = i == 1
					})
				}, nil)
			}, a.text.S.Set.Off, a.text.S.Set.NameOwn, a.text.S.Set.On)
			a.setSwitchRow(c, t, a.text.S.Set.CodexAgents, a.text.S.Set.CodexAgentsSub, &s.CodexAgentsV1, func(on bool) {
				a.settingsWrite(func() error {
					return saveSettings(func(s *settings.Settings) { s.CodexAgentsV1 = on })
				}, nil)
			})
			a.setSegRow(c, t, a.text.S.Set.LongConversations, a.text.S.Set.LongSub, &longIdx, func(i int) {
				a.settingsWrite(func() error {
					return saveSettings(func(s *settings.Settings) { s.FullContext = i == 1 })
				}, nil)
			}, fmt.Sprintf("%s %dk", a.text.S.Set.CompactAt, s.CompactAt), a.text.S.Set.FullWindow)
			a.setPickRow(c, t, a.text.S.Set.CodexTitles, a.text.S.Set.CodexTitlesSub, a.setTitlesName(s.CodexTitles), func() {
				a.openModelPicker("codexTitles", a.text.S.Set.CodexTitles, s.CodexTitles, func(id string) {
					a.settingsWrite(func() error {
						return saveSettings(func(s *settings.Settings) { s.CodexTitles = id })
					}, nil)
				})
			})
			a.setPickRow(c, t, a.text.S.Set.CodexReview, a.text.S.Set.CodexReviewSub, a.setTitlesName(s.CodexAutoReview), func() {
				a.openModelPicker("codexAutoReview", a.text.S.Set.CodexReview, s.CodexAutoReview, func(id string) {
					a.settingsWrite(func() error {
						return saveSettings(func(s *settings.Settings) { s.CodexAutoReview = id })
					}, nil)
				})
			})
		})

		a.setCard(c, t, "", func() {
			a.setButtonRow(c, t, a.text.S.Set.ConfigDir, func() {
				ui.Text(c, snap.Dir).FontSize(12).TextColor(t.TextMuted).Grow(1).Shrink(1).SingleLine()
				if a.smallBtn(c, t, a.text.S.Set.Copy) {
					a.copyToClipboard(snap.Dir)
					a.setSetNotice(a.text.S.Set.Copied, false)
				}
				if a.smallBtn(c, t, a.text.S.Set.Open) {
					a.openFolder(settings.Dir())
				}
			})
			a.setRow(c, t, a.text.S.Set.Version, snap.Version, nil)
		})
	})
}

// setTitlesName is the thread-titles' or reviewer's choice as the row's
// button reads it: what is set, or the words for none.
func (a *App) setTitlesName(v string) string {
	if v == "" {
		return a.text.S.Set.CodexTitlesChatGPTOption
	}
	if v == "off" {
		return a.text.S.Set.Off
	}
	for _, m := range a.data.Settings.TitleModels {
		if m.ID == v {
			return m.Name
		}
	}
	return v
}

// openModelPicker opens the model picker over the page, as magpie's
// settings dialogs do.
func (a *App) openModelPicker(kind, title, value string, onPick func(string)) {
	a.setPicker = &settingsPickerState{kind: kind, title: title, value: value, onPick: onPick}
}

// ---- About ----

// The About tab is in settings_about.go: it is magpie's About page.
