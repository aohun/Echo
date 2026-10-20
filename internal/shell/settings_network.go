//go:build nogui && echo

package shell

// The Settings page's Network and sharing tab: the proxy Echo's own
// requests go through, the GitHub token the library carries, the gateway's
// port, and the local network's sharing — magpie's own, drawn natively.

import (
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/yetone/magpie/internal/access"
	"github.com/yetone/magpie/internal/gui"
	"github.com/yetone/magpie/internal/settings"
)

// settingsNetwork draws the Network tab.
func (a *App) settingsNetwork(c *ui.Context, t *ui.Theme) {
	snap := a.data.Settings
	s := snap.S

	proxyIdx := 0
	if s.Proxy == "direct" {
		proxyIdx = 1
	} else if s.Proxy != "" {
		proxyIdx = 2
	}
	// the custom proxy's draft: seeded from the saved one, kept while it
	// is typed, and re-seeded once a write lands
	if !a.setCustomProxyOpen && a.setCustomProxy == "" && s.Proxy != "" && s.Proxy != "direct" {
		a.setCustomProxy = s.Proxy
	}

	ui.Column(c).FillWidth().Gap(14).Children(func() {
		a.setNoticeBar(c, t)

		// the proxy Echo's own requests to vendors go through
		a.setCard(c, t, a.text.S.Set.ProxyCard, func() {
			proxySub := a.text.S.Set.ProxySub
			if snap.ProxyNow != "" {
				proxySub += " · " + a.text.S.Set.ProxyNowPrefix + snap.ProxyNow + " (" + snap.ProxySource + ")"
			}
			a.setSegRow(c, t, a.text.S.Set.Proxy, proxySub, &proxyIdx, func(i int) {
				vals := []string{"", "direct", strings.TrimSpace(a.setCustomProxy)}
				a.settingsWrite(func() error {
					return saveSettings(func(s *settings.Settings) { s.Proxy = vals[i] })
				}, nil)
			}, a.text.S.Set.ProxyAuto, a.text.S.Set.ProxyDirect, a.text.S.Set.ProxyCustom)
			if proxyIdx == 2 {
				a.setRowWide(c, t, "", a.text.S.Set.ProxyURLSub, func() {
					ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(8).Children(func() {
						ui.TextInput(c, &a.setCustomProxy).Label("http://…").Grow(1).Shrink(1)
						if a.smallBtn(c, t, a.text.S.Set.GHSave) {
							u := strings.TrimSpace(a.setCustomProxy)
							a.settingsWrite(func() error {
								return saveSettings(func(s *settings.Settings) { s.Proxy = u })
							}, func(err error) {
								if err == nil {
									a.setCustomProxy = ""
									a.setCustomProxyOpen = false
								}
							})
						}
					})
				})
			}
		})

		// the GitHub token the library's requests carry
		a.setCard(c, t, a.text.S.Set.GHCard, func() {
			if snap.GHMask != "" {
				a.setButtonRow(c, t, a.text.S.Set.GHToken, func() {
					from := ""
					if snap.GHFrom != "" && snap.GHFrom != "settings" {
						from = " · " + a.text.S.Set.GHFromPrefix + snap.GHFrom
					}
					ui.Text(c, snap.GHMask+from).FontSize(12).TextColor(t.TextMuted).Grow(1).Shrink(1).SingleLine()
					if snap.GHFrom == "settings" {
						if a.smallBtn(c, t, a.text.S.Set.GHRemove) {
							a.settingsWrite(func() error {
								return saveSettings(func(s *settings.Settings) { s.GitHubToken = "" })
							}, nil)
						}
					}
				})
			}
			a.setRowWide(c, t, a.text.S.Set.GHToken, a.text.S.Set.GHTokenSub, func() {
				ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(8).Children(func() {
					if a.setGHTokenDraft == "" {
						a.setGHTokenDraft = ""
					}
					ui.TextInput(c, &a.setGHTokenDraft).Label("ghp_…").Grow(1).Shrink(1)
					if a.smallBtn(c, t, a.text.S.Set.GHSave) {
						tok := strings.TrimSpace(a.setGHTokenDraft)
						if strings.ContainsFunc(tok, func(r rune) bool { return r <= ' ' || r == 0x7f }) {
							a.setSetNotice(a.text.S.Set.GHErr, true)
							return
						}
						a.settingsWrite(func() error {
							return saveSettings(func(s *settings.Settings) { s.GitHubToken = tok })
						}, func(err error) {
							if err == nil {
								a.setGHTokenDraft = ""
								a.setSetNotice(a.text.S.Set.Saved, false)
							} else {
								a.setSetNotice(err.Error(), true)
							}
						})
					}
					if a.smallBtn(c, t, a.text.S.Set.GHCreate) {
						a.openBrowser("https://github.com/settings/tokens/new")
					}
				})
			})
		})

		// the gateway's port, and the local network's sharing
		a.setCard(c, t, a.text.S.Set.GatewayCard, func() {
			portSub := a.text.S.Set.PortSub
			if snap.AddrEnv != "" {
				portSub = a.text.S.Set.PortDisabledSub
			}
			// the port's draft: seeded from the saved one, kept while it
			// is typed, re-seeded once a write lands
			if a.setPortDraft == "" {
				a.setPortDraft = portShown(snap)
			}
			a.setRowWide(c, t, a.text.S.Set.Port, portSub, func() {
				ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(8).Children(func() {
					f := ui.TextInput(c, &a.setPortDraft).Label(strconv.Itoa(settings.DefaultPort)).Width(110).Shrink(0)
					if snap.AddrEnv != "" {
						f.Disabled(true)
					}
					if snap.AddrEnv == "" && a.smallBtn(c, t, a.text.S.Set.PortApply) {
						a.applyPort(a.setPortDraft)
					}
				})
			})
			lanOn := s.LAN
			a.setSwitchRow(c, t, a.text.S.Set.LAN, a.text.S.Set.LANSub, &lanOn, func(on bool) {
				a.settingsWrite(func() error {
					// ConfigureLAN saves the sharing and its key; the
					// gateway this process serves is moved onto it after,
					// as magpie's settings/lan does
					if err := access.ConfigureLAN(on, false); err != nil {
						return err
					}
					if gw := gui.ServedServer(); gw != nil {
						return gw.Relisten()
					}
					return nil
				}, nil)
			})
			if lanOn && len(snap.LANURLs) > 0 {
				for _, u := range snap.LANURLs {
					a.setButtonRow(c, t, "", func() {
						ui.Text(c, u).FontSize(12).Font("monospace").TextColor(t.TextMuted).Grow(1).Shrink(1).SingleLine()
						if a.smallBtn(c, t, a.text.S.Set.Copy) {
							a.copyToClipboard(u)
							a.setSetNotice(a.text.S.Set.Copied, false)
						}
					})
				}
				if snap.LANContainer {
					ui.Row(c).Padding(4, 14).Children(func() {
						ui.Text(c, a.text.S.Set.LANContainer).FontSize(11.5).TextColor(t.TextMuted).Wrap()
					})
				}
			}
		})
	})
}

// portShown is the port the field starts at: the saved one, 0 read as the
// default.
func portShown(snap SettingsSnap) string {
	if snap.S.Port != 0 {
		return strconv.Itoa(snap.S.Port)
	}
	return strconv.Itoa(settings.DefaultPort)
}

// applyPort moves the gateway to the port the field holds, as magpie's
// settings/port does: the port checked, saved, the gateway moved, and
// every agent connected moved with it.
func (a *App) applyPort(draft string) {
	p, err := strconv.Atoi(strings.TrimSpace(draft))
	if err != nil {
		a.setSetNotice(a.text.S.Set.PortRangeErr, true)
		return
	}
	if a.setPortErr != "" {
		a.setPortErr = ""
	}
	a.settingsWrite(func() error {
		res, err := gui.SetPort(p)
		if err != nil {
			return err
		}
		if res.Error != "" {
			// some agents moved, one didn't: said, not failed
			a.setPortErr = res.Error
		}
		return nil
	}, func(err error) {
		if err == nil {
			a.setPortDraft = ""
			if a.setPortErr != "" {
				a.setSetNotice(a.setPortErr, true)
				a.setPortErr = ""
			} else {
				a.setSetNotice(a.text.S.Set.Saved, false)
			}
		} else {
			a.setSetNotice(err.Error(), true)
		}
	})
}
