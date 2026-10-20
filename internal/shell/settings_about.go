//go:build nogui && echo

package shell

// The About tab of the Settings page, as magpie's About page: the version
// and its update check, the update's own choices, the folder Echo keeps
// things in and the gateway's address, and the community.
//
// magpie's rows are about magpie's own updater (its header pill, its
// mirror, how often it asks). Echo's updater is MyGo's, which keeps its
// own choices — the automatic check and the automatic download — so those
// are what the tab shows; the rest of the page is magpie's.

import (
	"github.com/egoist/mygo/plugins/updater"
	"github.com/egoist/mygo/ui"
)

// settingsAbout draws the About tab: the version and its update check, the
// update's own choices, the folder, the gateway's URL, the community.
func (a *App) settingsAbout(c *ui.Context, t *ui.Theme) {
	snap := a.data.Settings

	// the update row's words: the version, and — when the updater is
	// wired (only Run mounts it) — when it last checked and the button
	// that opens its window (updater.CheckForUpdates). The window says
	// the rest, whichever it finds: a new version, none, or why this
	// build cannot update.
	upSub := snap.Version
	showUpdates := shellUpdatesWired
	if showUpdates {
		if at := updater.LastCheck(); !at.IsZero() {
			upSub += " · " + a.text.S.Set.UpdLastCheck + at.Local().Format("2006-01-02 15:04")
		}
	}
	// the updater's own choices, as it keeps them
	checkIdx, downloadIdx := 0, 0
	if showUpdates {
		if updater.AutomaticChecks() {
			checkIdx = 1
		}
		if updater.AutomaticDownloads() {
			downloadIdx = 1
		}
	}

	ui.Column(c).FillWidth().Gap(14).Children(func() {
		a.setNoticeBar(c, t)

		// the version, and the check that looks for a newer one
		a.setCard(c, t, a.text.S.Set.VersionCard, func() {
			a.setButtonRow(c, t, a.text.S.Set.Version, func() {
				ui.Column(c).Grow(1).Shrink(1).Gap(2).AlignItems(ui.End).Children(func() {
					ui.Text(c, upSub).FontSize(12).TextColor(t.TextMuted).Wrap()
				})
				if showUpdates {
					if a.smallBtn(c, t, a.text.S.Set.UpdCheck) {
						updater.CheckForUpdates()
					}
				}
			})
		})

		// the update's own choices: does Echo look for a newer version by
		// itself, and does it put one in as it finds it
		if showUpdates {
			a.setCard(c, t, a.text.S.Set.UpdatesCard, func() {
				a.setSegRow(c, t, a.text.S.Set.AutoCheck, a.text.S.Set.AutoCheckSub, &checkIdx, func(i int) {
					updater.SetAutomaticChecks(i == 1)
				}, a.text.S.Set.Off, a.text.S.Set.On)
				a.setSegRow(c, t, a.text.S.Set.AutoDownload, a.text.S.Set.AutoDownloadSub, &downloadIdx, func(i int) {
					updater.SetAutomaticDownloads(i == 1)
				}, a.text.S.Set.Off, a.text.S.Set.On)
			})
		}

		// the folder Echo keeps its things in, and the gateway's address
		a.setCard(c, t, "", func() {
			a.setRow(c, t, a.text.S.Set.ConfigDir, snap.Dir, func() {
				if a.smallBtn(c, t, a.text.S.Set.Copy) {
					a.copyToClipboard(snap.Dir)
					a.setSetNotice(a.text.S.Set.Copied, false)
				}
				if a.smallBtn(c, t, a.text.S.Set.Reveal) {
					a.openFolder(snap.Dir)
				}
			})
			a.setButtonRow(c, t, a.text.S.Set.GatewayURL, func() {
				ui.Text(c, snap.Gateway).FontSize(12).TextColor(t.TextMuted).Grow(1).Shrink(1).SingleLine()
				if a.smallBtn(c, t, a.text.S.Set.Copy) {
					a.copyToClipboard(snap.Gateway)
					a.setSetNotice(a.text.S.Set.Copied, false)
				}
			})
		})

		// the community
		a.setCard(c, t, a.text.S.Set.Community, func() {
			a.setButtonRow(c, t, a.text.S.Set.CommunitySub, func() {
				if a.smallBtn(c, t, "GitHub") {
					a.openBrowser("https://github.com/aohun/Echo")
				}
			})
		})
	})
}
