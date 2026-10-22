//go:build nogui && echo

package shell

// The Settings page's Models tab: the image tools' models, the searcher
// and the search APIs, and the reply member naming — magpie's own, drawn
// natively.

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/yetone/magpie/internal/settings"
)

// settingsModels draws the Models tab.
func (a *App) settingsModels(c *ui.Context, t *ui.Theme) {
	snap := a.data.Settings
	s := snap.S

	ui.Column(c).FillWidth().Gap(14).Children(func() {
		a.setNoticeBar(c, t)

		// the two image tools, each a model picker over one card
		a.setCard(c, t, a.text.S.Set.ImagesCard, func() {
			a.setPickRow(c, t, a.text.S.Set.Vision, a.setVisionSub(snap), a.setModelName(snap.VisionAuto, s.Vision), func() {
				a.openModelPicker("vision", a.text.S.Set.Vision, s.Vision, func(id string) {
					a.settingsWrite(func() error {
						return saveSettings(func(s *settings.Settings) { s.Vision = id })
					}, nil)
				})
			})
			a.setPickRow(c, t, a.text.S.Set.ImageGen, a.text.S.Set.ImageGenSub, a.setModelName(snap.ImageGenAuto, s.ImageGen), func() {
				a.openModelPicker("imageGen", a.text.S.Set.ImageGen, s.ImageGen, func(id string) {
					a.settingsWrite(func() error {
						return saveSettings(func(s *settings.Settings) { s.ImageGen = id })
					}, nil)
				})
			})
		})

		// the searcher, the search APIs set for it, and what is tried
		// first when both can
		a.setCard(c, t, a.text.S.Set.SearchCard, func() {
			a.setPickRow(c, t, a.text.S.Set.Searcher, a.setSearcherSub(snap), a.setSearcherName(snap), func() {
				a.openModelPicker("searcher", a.text.S.Set.Searcher, s.Searcher, func(id string) {
					a.settingsWrite(func() error {
						return saveSettings(func(s *settings.Settings) { s.Searcher = id })
					}, nil)
				})
			})
			a.setSearchAPIs(c, t, snap)
			if len(snap.SearchAPIs) > 0 && len(snap.SearchChoices) > 0 {
				firstIdx := 0
				if s.SearchFirst == "api" {
					firstIdx = 1
				}
				a.setSegRow(c, t, a.text.S.Set.SearchFirst, "", &firstIdx, func(i int) {
					vals := []string{"", "api"}
					a.settingsWrite(func() error {
						return saveSettings(func(s *settings.Settings) { s.SearchFirst = vals[i] })
					}, nil)
				}, a.text.S.Set.SearchFirstModel, a.text.S.Set.SearchFirstAPI)
			}
		})

		// the reply's model name: the group member that answered, or the
		// vendor's own
		a.setCard(c, t, "", func() {
			a.setSwitchRow(c, t, a.text.S.Set.MemberModel, a.text.S.Set.MemberModelSub, &s.MemberModel, func(on bool) {
				a.settingsWrite(func() error {
					return saveSettings(func(s *settings.Settings) { s.MemberModel = on })
				}, nil)
			})
		})
	})
}

// setVisionSub is the vision row's sub: what the automatic pick would be.
func (a *App) setVisionSub(snap SettingsSnap) string {
	sub := a.text.S.Set.VisionSub
	if snap.VisionAuto != "" {
		sub += " · " + a.text.S.Set.AutoNowPrefix + snap.VisionAuto
	}
	return sub
}

// setModelName is a model setting as its row's button reads it: the
// choice, the words for none, or the automatic pick named.
func (a *App) setModelName(auto, v string) string {
	if v == "off" {
		return a.text.S.Set.Off
	}
	if v == "" {
		if auto != "" {
			return a.text.S.Set.Automatic + " (" + auto + ")"
		}
		return a.text.S.Set.Automatic
	}
	return v
}

// setSearcherSub is the searcher row's sub: what the automatic pick would
// be, and why a set one isn't serving.
func (a *App) setSearcherSub(snap SettingsSnap) string {
	sub := a.text.S.Set.SearcherSub
	if snap.SearchAuto != "" {
		sub += " · " + a.text.S.Set.AutoNowPrefix + snap.SearchAuto
	}
	if snap.SearchUnused != "" {
		switch snap.SearchUnused {
		case "gone":
			sub += " · " + a.text.S.Set.SearcherGone
		case "off":
			sub += " · " + a.text.S.Set.SearcherOff
		case "cant":
			sub += " · " + a.text.S.Set.SearcherCant
		case "nomodel":
			sub += " · " + a.text.S.Set.SearcherNoModel
		}
	}
	if len(snap.SearchLeftOut) > 0 {
		sub += " · " + a.text.S.Set.SearcherLeftOutPrefix + strings.Join(snap.SearchLeftOut, ", ")
	}
	return sub
}

// setSearcherName is the searcher setting as its row's button reads it.
func (a *App) setSearcherName(snap SettingsSnap) string {
	v := snap.S.Searcher
	if v == "" {
		if snap.SearchAuto != "" {
			return a.text.S.Set.Automatic + " (" + snap.SearchAuto + ")"
		}
		return a.text.S.Set.Automatic
	}
	for _, ch := range snap.SearchChoices {
		if ch.ID == v {
			return ch.Name
		}
		for _, m := range ch.Models {
			if m.ID == v {
				return ch.Name + " · " + m.Name
			}
		}
	}
	return v
}

// setSearchAPIs draws the search APIs' list and its add row: one row per
// set API, its key masked until a click asks for the real one — read off
// the main thread, as every secret shown is.
func (a *App) setSearchAPIs(c *ui.Context, t *ui.Theme, snap SettingsSnap) {
	a.setRowWide(c, t, a.text.S.Set.SearchAPIs, a.text.S.Set.SearchAPIsSub, func() {
		ui.Column(c).FillWidth().Gap(4).Children(func() {
			if len(snap.SearchAPIs) == 0 {
				ui.Text(c, a.text.S.Set.SearchAPINone).FontSize(12).TextColor(t.TextMuted)
			}
			for _, row := range snap.SearchAPIs {
				name, vendor := row.Name, row.Vendor
				ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(8).Children(func() {
					dotC := ui.Hex("#16a34a")
					if !row.Ready {
						dotC = ui.Hex("#dc2626")
					}
					ui.Text(c, "●").FontSize(10).TextColor(dotC).Shrink(0)
					ui.Text(c, name).FontSize(12.5).FontWeight(500).TextColor(t.Text).Grow(1).Shrink(1).SingleLine()
					shown := row.Key
					if real, has := a.setRevealedKey[vendor]; has && real != "" {
						shown = real
					}
					ui.Text(c, shown).FontSize(12).Font("monospace").TextColor(t.TextMuted).Grow(1).Shrink(1).SingleLine()
					if a.smallBtn(c, t, a.setRevealLabel(vendor)) {
						if a.setRevealedKey[vendor] != "" {
							a.setRevealedKey[vendor] = ""
							c.Invalidate()
							return
						}
						a.revealSearchKey(vendor)
					}
					if a.smallBtn(c, t, a.text.S.Set.RuleRemove) {
						a.settingsWrite(func() error { return searchAPIRemove(vendor) }, func(err error) {
							if err != nil {
								a.setSetNotice(err.Error(), true)
							}
						})
					}
				})
			}

			// the add row: a vendor, its key, and — one the user runs —
			// its address
			vendors := []string{}
			for _, v := range snap.SearchVendors {
				vendors = append(vendors, v.Name)
			}
			if len(vendors) == 0 {
				return
			}
			if a.setSearchVendorName == "" {
				a.setSearchVendorName = vendors[0]
			}
			ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(8).Padding(4, 0, 0, 0).Children(func() {
				if ui.Select(c, &a.setSearchVendorName, vendors).Changed() {
					c.Invalidate()
				}
				vendor := a.searchVendorID(snap, a.setSearchVendorName)
				ui.TextInput(c, &a.setSearchKey).Label(a.text.S.Set.SearchAPIKey).Width(180).Shrink(0)
				if a.searchVendorNeedsURL(snap, vendor) {
					ui.TextInput(c, &a.setSearchURL).Label(a.text.S.Set.SearchAPIURL).Width(180).Shrink(0)
				}
				if a.smallBtn(c, t, a.text.S.Add) {
					a.settingsWrite(func() error {
						return searchAPISet(vendor, strings.TrimSpace(a.setSearchKey), strings.TrimSpace(a.setSearchURL))
					}, func(err error) {
						if err == nil {
							a.setSearchKey, a.setSearchURL, a.setSearchErr = "", "", ""
							a.setSetNotice(a.text.S.Set.Saved, false)
						} else {
							a.setSearchErr = err.Error()
							a.setSetNotice(err.Error(), true)
						}
					})
				}
			})
			if a.setSearchErr != "" {
				ui.Text(c, a.setSearchErr).FontSize(11.5).TextColor(ui.Hex("#dc2626")).Wrap()
			}
		})
	})
}

// searchVendorID is a vendor's id by the name the select shows.
func (a *App) searchVendorID(snap SettingsSnap, name string) string {
	for _, v := range snap.SearchVendors {
		if v.Name == name {
			return v.ID
		}
	}
	return name
}

// searchVendorNeedsURL says whether the vendor runs on the user's own
// server, whose address the add row asks for.
func (a *App) searchVendorNeedsURL(snap SettingsSnap, id string) bool {
	for _, v := range snap.SearchVendors {
		if v.ID == id {
			return v.NeedURL
		}
	}
	return false
}

// setRevealLabel is the show/hide button's label for one vendor's key.
func (a *App) setRevealLabel(vendor string) string {
	if a.setRevealedKey[vendor] != "" {
		return a.text.S.Set.SearchKeyHide
	}
	return a.text.S.Set.SearchKeyShow
}

// revealSearchKey reads one vendor's key off the main thread and puts it
// where the row shows it, as magpie's settings/search-key does.
func (a *App) revealSearchKey(vendor string) {
	go func() {
		key, _ := searchKeyReveal(vendor)
		if a.win == nil {
			return
		}
		a.win.Update(func() {
			a.setRevealedKey[vendor] = key
		})
	}()
}
