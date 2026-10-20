//go:build nogui && echo

package shell

// The Settings page's Allowances tab: what the menu bar shows of the
// subscriptions and plans, the currency, the three alerts, and the
// warm-ups and check-ins — magpie's own, drawn natively.

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/yetone/magpie/internal/provider"
	"github.com/yetone/magpie/internal/settings"
)

// TrayCard is one subscription or plan the menu-bar allowances picker
// offers, as magpie's pill of cards does: its tray key, what to call it,
// and its icon.
type TrayCard struct {
	ID   string // "provider|user", the Settings.TrayUsages entry
	Name string
	Icon string
}

// warmTabs is the warm-up card's sub-tabs, in magpie's order.
var warmTabs = []struct {
	id    string
	label func(a *App) string
}{
	{"codex", func(a *App) string { return "Codex" }},
	{"claude", func(a *App) string { return "Claude Code" }},
	{"wb", func(a *App) string { return "WorkBuddy" }},
	{"trae", func(a *App) string { return "Trae" }},
	{"minimax", func(a *App) string { return "MiniMax Code" }},
	{"qoder", func(a *App) string { return "Qoder" }},
}

// settingsAllowances draws the Allowances tab.
func (a *App) settingsAllowances(c *ui.Context, t *ui.Theme) {
	snap := a.data.Settings
	s := snap.S

	everyIdx := map[int]int{0: 1, 1: 0, 3: 1, 5: 2, 10: 3, 30: 4}[s.TrayUsageEvery]
	if s.TrayUsageEvery == 0 {
		everyIdx = 1
	}
	curIdx := map[string]int{"usd": 0, "cny": 1}[cmpOr(s.Currency, "usd")]
	quotaIdx := 0
	if s.QuotaLeft {
		quotaIdx = 1
	}
	usageOn := s.UsageAlert > 0
	usagePct := s.UsageAlert
	if usagePct == 0 {
		usagePct = 75
	}
	balOn := s.BalanceAlert > 0
	bal := s.BalanceAlert
	resetIdx := map[int]int{0: 0, 12: 1, 24: 2, 48: 3}[s.ResetReminder]

	ui.Column(c).FillWidth().Gap(14).Children(func() {
		a.setNoticeBar(c, t)

		// the menu bar's allowances: which cards it shows, how often the
		// text is brought up to date
		a.setCard(c, t, a.text.S.Set.MenuBar, func() {
			cards := snap.TrayCards
			if len(cards) == 0 && !a.setTrayReading {
				// nothing cached yet: the vendors are asked behind the
				// page, and the cards are here on the next read
				a.setTrayReading = true
				go func() {
					ReadQuotas(true)
					a.setTrayReading = false
					a.Reload()
				}()
			}
			a.setRowWide(c, t, a.text.S.Set.TrayUsages, a.text.S.Set.TrayUsagesSub, func() {
				if len(cards) == 0 {
					ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
						ui.Text(c, a.text.S.NoData).FontSize(12).TextColor(t.TextMuted)
						if a.setTrayReading {
							ui.Spinner(c).Size(12, 12)
						}
					})
					return
				}
				ui.Column(c).Gap(4).Children(func() {
					for _, card := range cards {
						on := trayUsagesHas(s.TrayUsages, card.ID)
						id := card.ID
						ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(10).Children(func() {
							ui.Text(c, card.Name).FontSize(12.5).TextColor(t.Text).Grow(1).Shrink(1).SingleLine()
							sw := ui.Switch(c, &on).Shrink(0)
							if sw.Clicked() {
								a.settingsWrite(func() error {
									return saveSettings(func(s *settings.Settings) {
										s.TrayUsages = trayUsagesSet(s.TrayUsages, id, on)
										if len(s.TrayUsages) > 0 {
											s.TrayUsage = s.TrayUsages[0]
										}
									})
								}, nil)
							}
						})
					}
				})
			})
			a.setSegRow(c, t, a.text.S.Set.TrayEvery, "", &everyIdx, func(i int) {
				mins := []int{1, 3, 5, 10, 30}
				a.settingsWrite(func() error {
					return saveSettings(func(s *settings.Settings) { s.TrayUsageEvery = mins[i] })
				}, nil)
			}, "1 min", "3 min", "5 min", "10 min", "30 min")
		})

		// how an allowance reads: used or left, whose currency, and what
		// the large counts are called
		a.setCard(c, t, a.text.S.Set.DisplayCard, func() {
			a.setSegRow(c, t, a.text.S.Set.QuotaLeft, "", &quotaIdx, func(i int) {
				a.settingsWrite(func() error {
					return saveSettings(func(s *settings.Settings) { s.QuotaLeft = i == 1 })
				}, nil)
			}, a.text.S.Set.QuotaUsed, a.text.S.Set.QuotaLeftL)
			a.setSegRow(c, t, a.text.S.Set.Currency, a.setFXSub(snap), &curIdx, func(i int) {
				currencies := []string{"usd", "cny"}
				a.settingsWrite(func() error {
					return saveSettings(func(s *settings.Settings) { s.Currency = currencies[i] })
				}, nil)
			}, "USD", "CNY")
			a.setSwitchRow(c, t, a.text.S.Set.ChineseUnits, a.text.S.Set.ChineseUnitsSub, &s.ChineseUnits, func(on bool) {
				a.settingsWrite(func() error {
					return saveSettings(func(s *settings.Settings) { s.ChineseUnits = on })
				}, nil)
			})
		})

		// the three alerts
		a.setCard(c, t, a.text.S.Set.AlertsCard, func() {
			pctDraft := a.setAlertDraft("usage", fmt.Sprint(usagePct))
			a.setRowWide(c, t, a.text.S.Set.UsageAlert, a.text.S.Set.UsageAlertSub, func() {
				ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(8).Children(func() {
					sw := ui.Switch(c, &usageOn).Shrink(0)
					if sw.Clicked() {
						v := 0
						if usageOn {
							v = alertInt(pctDraft, 75)
						}
						a.saveAlert(func(s *settings.Settings) { s.UsageAlert = v })
					}
					ui.TextInput(c, &pctDraft).Label("%").Width(80).Shrink(0)
					if a.smallBtn(c, t, a.text.S.Set.GHSave) {
						a.saveAlert(func(s *settings.Settings) { s.UsageAlert = alertInt(pctDraft, 75) })
					}
				})
			})
			balDraft := a.setAlertDraft("balance", fmt.Sprintf("%g", bal))
			a.setRowWide(c, t, a.text.S.Set.BalanceAlert, a.text.S.Set.BalanceAlertSub, func() {
				ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(8).Children(func() {
					sw := ui.Switch(c, &balOn).Shrink(0)
					if sw.Clicked() {
						v := 0.0
						if balOn {
							v = alertFloat(balDraft, 10)
						}
						a.saveAlert(func(s *settings.Settings) { s.BalanceAlert = v })
					}
					ui.TextInput(c, &balDraft).Label("¥/$").Width(80).Shrink(0)
					if a.smallBtn(c, t, a.text.S.Set.GHSave) {
						a.saveAlert(func(s *settings.Settings) { s.BalanceAlert = alertFloat(balDraft, 10) })
					}
				})
			})
			a.setSegRow(c, t, a.text.S.Set.ResetReminder, a.text.S.Set.ResetSub, &resetIdx, func(i int) {
				hours := []int{0, 12, 24, 48}
				a.settingsWrite(func() error {
					return saveSettings(func(s *settings.Settings) { s.ResetReminder = hours[i] })
				}, nil)
			}, a.text.S.Set.Off, "12 h", "24 h", "48 h")
		})

		// the warm-ups and the check-ins, one sub-tab each over one card
		a.setWarmCard(c, t, snap)
	})
}

// setFXSub is the currency row's sub: the rate known now, as magpie's own
// line reads.
func (a *App) setFXSub(snap SettingsSnap) string {
	fx := snap.FX
	if fx.Rate == 0 {
		return ""
	}
	sub := fmt.Sprintf("1 USD ≈ %.2f CNY", fx.Rate)
	if fx.At != nil {
		sub += " · " + syncWhen(*fx.At)
	}
	if fx.Stale {
		sub += " · " + a.text.S.Set.FXStale
	}
	return sub
}

// syncWhen is a time as the rows' "last …" lines say it: today the hour,
// else the day, as magpie's own does.
func syncWhen(at time.Time) string {
	at = at.Local()
	day := at.Format("2006-01-02")
	today := time.Now().Format("2006-01-02")
	if day == today {
		return at.Format("15:04")
	}
	return day
}

// trayUsagesHas says whether the tray key is picked.
func trayUsagesHas(list []string, id string) bool {
	for _, v := range list {
		if v == id {
			return true
		}
	}
	return false
}

// trayUsagesSet adds the key to the list, or takes it out, keeping the
// order the rest keep.
func trayUsagesSet(list []string, id string, on bool) []string {
	out := []string{}
	for _, v := range list {
		if v != id {
			out = append(out, v)
		}
	}
	if on {
		out = append(out, id)
	}
	return out
}

// alertInt is a draft read as a percent, defaulting when it isn't one.
func alertInt(draft string, def int) int {
	n, err := strconv.Atoi(strings.TrimSpace(draft))
	if err != nil || n < 0 || n > 100 {
		return def
	}
	return n
}

// alertFloat is a draft read as an amount, defaulting when it isn't one.
func alertFloat(draft string, def float64) float64 {
	n, err := strconv.ParseFloat(strings.TrimSpace(draft), 64)
	if err != nil || n < 0 {
		return def
	}
	return n
}

// setAlertDraft is the field the alert's own input holds; a draft kept
// across the reads so a refused write doesn't lose what was typed.
func (a *App) setAlertDraft(kind, cur string) string {
	switch kind {
	case "usage":
		if a.setUsagePctDraft == "" {
			a.setUsagePctDraft = cur
		}
		return a.setUsagePctDraft
	default:
		if a.setBalDraft == "" {
			a.setBalDraft = cur
		}
		return a.setBalDraft
	}
}

// saveAlert writes one of the alerts and says so.
func (a *App) saveAlert(mutate func(s *settings.Settings)) {
	a.settingsWrite(func() error {
		return saveSettings(mutate)
	}, func(err error) {
		if err == nil {
			a.setSetNotice(a.text.S.Set.Saved, false)
		} else {
			a.setSetNotice(err.Error(), true)
		}
	})
}

// ---- the warm-up and check-in card ----

// setWarmCard draws the warm-up card and its sub-tabs: Codex's and Claude
// Code's warm-ups, and the four services' daily check-ins.
func (a *App) setWarmCard(c *ui.Context, t *ui.Theme, snap SettingsSnap) {
	s := snap.S
	// a sub-tab is offered while its service can use it
	shown := []int{}
	for i, tab := range warmTabs {
		switch tab.id {
		case "codex", "claude":
			shown = append(shown, i)
		case "wb":
			if s.WorkBuddyCheckin || snap.WorkBuddy || len(snap.WBCheckins) > 0 {
				shown = append(shown, i)
			}
		case "trae":
			if s.TraeCheckin || snap.Trae || len(snap.TraeCheckins) > 0 {
				shown = append(shown, i)
			}
		case "minimax":
			if s.MiniMaxCheckin || snap.MiniMax || len(snap.MiniMaxCheckins) > 0 {
				shown = append(shown, i)
			}
		case "qoder":
			if s.QoderCheckin || snap.Qoder || len(snap.QoderCheckins) > 0 {
				shown = append(shown, i)
			}
		}
	}
	// the tab held only when it is still offered
	known := false
	for _, i := range shown {
		if warmTabs[i].id == a.setWarmTab {
			known = true
		}
	}
	if !known && len(shown) > 0 {
		a.setWarmTab = warmTabs[shown[0]].id
	}

	a.setCard(c, t, a.text.S.Set.WarmCard, func() {
		if len(shown) > 1 {
			warmIdx := 0
			labels := []string{}
			for i, tab := range warmTabs {
				if warmTabs[i].id == a.setWarmTab {
					warmIdx = len(labels)
				}
				for _, sh := range shown {
					if sh == i {
						labels = append(labels, tab.label(a))
					}
				}
			}
			ui.Row(c).Padding(10, 14, 0, 14).Children(func() {
				if ui.Segmented(c, &warmIdx, labels...).Changed() {
					if warmIdx >= 0 && warmIdx < len(shown) {
						a.setWarmTab = warmTabs[shown[warmIdx]].id
						c.Invalidate()
					}
				}
			})
		}
		switch a.setWarmTab {
		case "claude":
			a.warmRows(c, t, a.text.S.Set.ClaudeWarm, s.ClaudeWarmup, snap.ClaudeWarmed, func(v string) {
				a.saveAlert(func(s *settings.Settings) { s.ClaudeWarmup = v })
			}, func(v string) {
				a.saveAlert(func(s *settings.Settings) { s.ClaudeWarmAt = v })
			}, s.ClaudeWarmAt, "")
		case "wb":
			a.checkinRow(c, t, a.text.S.Set.WBCheckin, a.text.S.Set.WBCheckinSub, &s.WorkBuddyCheckin, snap.WBCheckins, func(on bool) {
				a.saveAlert(func(s *settings.Settings) { s.WorkBuddyCheckin = on })
			})
		case "trae":
			a.checkinRow(c, t, a.text.S.Set.TraeCheckin, a.text.S.Set.TraeCheckinSub, &s.TraeCheckin, snap.TraeCheckins, func(on bool) {
				a.saveAlert(func(s *settings.Settings) { s.TraeCheckin = on })
			})
		case "minimax":
			a.checkinRow(c, t, a.text.S.Set.MiniMaxCheckin, a.text.S.Set.MiniMaxCheckinSub, &s.MiniMaxCheckin, snap.MiniMaxCheckins, func(on bool) {
				a.saveAlert(func(s *settings.Settings) { s.MiniMaxCheckin = on })
			})
		case "qoder":
			a.checkinRow(c, t, a.text.S.Set.QoderCheckin, a.text.S.Set.QoderCheckinSub, &s.QoderCheckin, snap.QoderCheckins, func(on bool) {
				a.saveAlert(func(s *settings.Settings) { s.QoderCheckin = on })
			})
		default:
			a.warmRows(c, t, a.text.S.Set.CodexWarm, s.CodexWarmup, snap.CodexWarmed, func(v string) {
				a.saveAlert(func(s *settings.Settings) { s.CodexWarmup = v })
			}, func(v string) {
				a.saveAlert(func(s *settings.Settings) { s.CodexWarmAt = v })
			}, s.CodexWarmAt, strings.Join(snap.CodexUsers, ", "))
		}
	})
}

// warmRows are one warm-up's two rows: the warm-up on reset, and the daily
// one at a time of day — Codex's and Claude Code's alike.
func (a *App) warmRows(c *ui.Context, t *ui.Theme, title, warmup string, warmed *time.Time, setWarmup, setWarmAt func(string), warmAt, users string) {
	warmIdx := map[string]int{"": 0, "week": 1, "all": 2}[warmup]
	atOn := warmAt != ""
	at := warmAt
	if at == "" {
		at = "06:00"
	}
	sub := a.text.S.Set.WarmSub
	if warmed != nil {
		sub += " · " + a.text.S.Set.WarmLastPrefix + syncWhen(*warmed)
	}
	a.setSegRow(c, t, title, sub, &warmIdx, func(i int) {
		vals := []string{"", "week", "all"}
		setWarmup(vals[i])
	}, a.text.S.Set.Off, a.text.S.Set.WarmWeek, a.text.S.Set.WarmAll)
	a.setRowWide(c, t, a.text.S.Set.WarmAt, a.text.S.Set.WarmAtSub, func() {
		ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(8).Children(func() {
			atIdx := 0
			if atOn {
				atIdx = 1
			}
			if ui.Segmented(c, &atIdx, a.text.S.Set.Off, a.text.S.Set.On).Changed() {
				v := ""
				if atIdx == 1 {
					v = at
				}
				setWarmAt(v)
				c.Invalidate()
			}
			if atOn {
				ui.TextInput(c, &at).Label("15:04").Width(90).Shrink(0)
				if a.smallBtn(c, t, a.text.S.Set.GHSave) {
					if matched := isClock(at); matched {
						setWarmAt(at)
					} else {
						a.setSetNotice(a.text.S.Set.WarmAtErr, true)
					}
				}
			}
		})
	})
	_ = users
}

// isClock says whether s reads "15:04".
func isClock(s string) bool {
	if len(s) != 5 || s[2] != ':' {
		return false
	}
	h, m := s[:2], s[3:]
	for _, part := range []string{h, m} {
		if len(part) != 2 || part < "00" || part > "59" {
			return false
		}
	}
	return h <= "23"
}

// checkinRow is one service's daily check-in: its switch, and what the
// last check-ins said, as the row's sub line lists them.
func (a *App) checkinRow(c *ui.Context, t *ui.Theme, title, sub string, on *bool, checkins []provider.WorkBuddyCheckin, set func(bool)) {
	if len(checkins) > 0 {
		lines := []string{sub}
		for _, w := range checkins {
			if line := checkinLine(w); line != "" {
				lines = append(lines, line)
			}
		}
		sub = strings.Join(lines, " · ")
	}
	a.setSwitchRow(c, t, title, sub, on, set)
}

// checkinLine is one check-in as the row's sub says it: the account, the
// day, and what it got, as magpie's wbCheckinLine does.
func checkinLine(w provider.WorkBuddyCheckin) string {
	if w.User == "" {
		return ""
	}
	line := w.User + ": " + w.Day
	if w.Credit > 0 {
		line += fmt.Sprintf(" · +%g", w.Credit)
	}
	if w.Streak > 0 {
		line += fmt.Sprintf(" · %d", w.Streak)
	}
	if w.Msg != "" {
		line += " · " + w.Msg
	}
	return line
}
