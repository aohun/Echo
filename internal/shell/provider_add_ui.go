//go:build nogui && echo

package shell

import (
	"runtime"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/yetone/magpie/internal/proc"
	"github.com/yetone/magpie/internal/provider"
)

// The add sheet's dialog and the editor it opens: ＋ Add provider's modal,
// the picks it offers, and the form a pick turns into.
//
// magpie's own add sheet sits under the page and expands in place. Echo's
// is a modal over the page — the way an app's own sheet behaves — and the
// editor is a second step in the same dialog: a pick swaps the list of
// presets for the form, back returns to the list, the way magpie's editor
// folds back into the sheet's row.

// openAdd opens the add sheet over the page.
func (a *App) openAdd() {
	a.pvAdding = &addSheet{}
	a.invalidate()
}

// closeAdd puts the add sheet away, whether it holds the list or an editor.
func (a *App) closeAdd() {
	a.pvAdding = nil
	a.invalidate()
}

// addGiveUp is Cancel in an editor, and the sheet's corner's close: the
// sheet is put away whole and nothing of it survives — a pick goes, its
// editor's fields go with it, and a sign-in still under way is given up on.
// Back is the way that keeps them, for a reader who means to come back to
// the same pick; Cancel says the sheet is done, whatever it held.
func (a *App) addGiveUp() {
	if a.pvSignIn != nil {
		provider.CancelSignIn(a.pvSignIn.ID)
		a.pvSignIn = nil
	}
	a.pvAsk = nil
	a.closeAdd()
}

// invalidate asks for a frame, where there is a window to ask. A frame the
// tester builds is its own, so the sheet's opens and closes must not need a
// window to have happened.
func (a *App) invalidate() {
	if a.win != nil {
		a.win.Invalidate()
	}
}

// addOverlay draws the add sheet over the page: the dimmed page behind it,
// the sheet in front of it, and the sheet's own ways out.
//
// It is an overlay of its own rather than ui.Modal, which dismisses a dialog
// when the pointer presses the dimmed page beside it. The sheet keeps what a
// reader put in it — a preset's key half-typed, a custom provider's fields,
// a sign-in under way — and a press that lands beside it would throw all of
// that away for a click that was not aimed at it. It closes from the sheet
// itself: the corner's close, an editor's Cancel, or Escape. The backdrop is
// still a modal's, so the focus and Tab stay inside the sheet while it shows.
func (a *App) addOverlay(c *ui.Context) {
	if a.pvAdding == nil {
		return
	}
	t := c.Theme()
	ui.Overlay(c, func() {
		back := ui.Box(c).Absolute().Left(0).Top(0).Right(0).Bottom(0).Center().
			Background(ui.RGBA(0, 0, 0, 0.4)).Modal()
		// No Clickable and no Clicked on the backdrop: a press that lands
		// on the dimmed page is not a press on the sheet, so it is
		// nothing at all — the page behind stays dimmed and inert.
		if back.OverlayShortcut(0, ui.KeyEscape) {
			a.closeAdd()
			return
		}
		back.Children(func() {
			// The panel is as large as the window lets it be and no
			// larger: a fixed 560 tall on a short window runs past the
			// bottom of the screen, and the sheet's foot lands off it.
			// MaxHeight and MaxWidth keep it inside whatever window it
			// opens in, the way a sheet of its own is kept.
			cw, ch := c.Size()
			panel := ui.Column(c).Width(680).Height(560).
				MaxWidth(cw - 80).MaxHeight(ch - 80).
				Radius(12).Role(ui.RoleDialog).Label(a.text.S.AddProvider)
			if t.Dark {
				panel.Background(ui.RGBA(40, 40, 46, 1))
			} else {
				panel.Background(t.Background)
			}
			panel.Shadow(0, 10, 30, 0, ui.RGBA(0, 0, 0, 0.3))
			panel.Children(func() {
				a.addDialog(c)
			})
		})
	})
}

// addBack returns to the picks, as the head's 返回 once did: a sign-in
// under way for the pick goes with it, and the draft with both, so the
// next editor opens on empty fields.
func (a *App) addBack(sh *addSheet) {
	if sh.Picker != nil && sh.Picker.Kind == AddSub {
		if a.pvSignIn != nil && a.pvSignIn.Agent == sh.Picker.Key {
			provider.CancelSignIn(a.pvSignIn.ID)
			a.pvSignIn = nil
		}
		// an ask that hasn't begun a sign-in yet has nothing to
		// cancel: it goes, and its answers with it
		if a.pvAsk != nil && a.pvAsk.Agent == sh.Picker.Key {
			a.pvAsk = nil
		}
	}
	sh.Picker = nil
	sh.draft = nil
	a.invalidate()
}

// addDialog draws the add sheet over the page: the search, the picks in
// their groups, or the form for the pick that was chosen. addOverlay gives
// it the panel it is drawn in.
func (a *App) addDialog(c *ui.Context) {
	t := c.Theme()
	sh := a.pvAdding

	// The sheet fills the panel addOverlay gives it, so the picks under
	// the head have a height to scroll within. Without a definite height
	// here the column is as tall as every pick in it, and the scroll has
	// nothing to clip: the rows drawn past the panel's bottom simply
	// painted over the page behind it.
	box := ui.Column(c).FillWidth().FillHeight()
	box.Children(func() {
		// the head: the sheet's title — or the pick's own while its
		// editor shows — the search beside it, and the corner's close
		ui.Row(c).AlignItems(ui.Center).Gap(8).Padding(14, 16, 10, 16).Children(func() {
			// The head's title: the pick's own name while the editor
			// shows, the sheet's own title otherwise. The custom pick is
			// named by its group's heading, so it says only what it takes.
			title := a.text.S.AddProvider
			if sh.Picker != nil {
				if sh.Picker.Kind == AddCustom {
					title = a.text.S.CustomProvider
				} else {
					title = sh.Picker.Name
				}
			}
			ui.Text(c, title).FontSize(14).FontWeight(600).TextColor(t.Text).SingleLine()
			ui.Spacer(c)
			// The search is the list's own: an editor's picks have none,
			// magpie's editor neither — the words would only sit there
			// over a form they have nothing to narrow.
			if sh.Picker == nil {
				ui.SearchField(c, &sh.Query).Width(200).Label(a.text.S.FindVendor)
			}
			// The way out of the sheet, in the corner a dialog's own
			// close button keeps. The sheet is put away whole: a pick
			// being edited goes with it, and a sign-in started here goes
			// with that — a Close press is a "nothing from this sheet".
			if xc := a.addClose(c, t); xc.Clicked() {
				a.addGiveUp()
				return
			}
		})

		a.addRule(c, t)

		if sh.Picker != nil {
			if sh.Picker.Kind == AddSub {
				a.addEditor(c, sh, t)
				return
			}
			// a form editor: its fields scrolling, the hairline under
			// them, and the foot's 取消 and 添加 at the window's bottom
			// right — magpie's own bar
			ui.Column(c).FillWidth().FillHeight().Children(func() {
				a.addEditor(c, sh, t)
				a.addRule(c, t)
				ui.Row(c).AlignItems(ui.Center).Gap(12).Padding(10, 16, 12, 16).FillWidth().Children(func() {
					ui.Spacer(c)
					cancelBtn := ui.Row(c).AlignItems(ui.Center).Padding(5, 12).Radius(7).
						Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).Label(a.text.S.Cancel)
					clicked := cancelBtn.Clicked()
					cancelBtn.Children(func() {
						ui.Text(c, a.text.S.Cancel).FontSize(12.5).TextColor(t.Text).SingleLine()
					})
					if clicked {
						a.addBack(sh)
						return
					}
					addBtn := ui.Row(c).AlignItems(ui.Center).Padding(5, 14).Radius(7).Background(t.Accent).
						Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).Label(a.text.S.Add)
					addClicked := addBtn.Clicked()
					addBtn.Children(func() {
						ui.Text(c, a.text.S.Add).FontSize(12.5).FontWeight(600).TextColor(ui.Hex("#ffffff")).SingleLine()
					})
					if addClicked {
						a.addSave(c, sh)
					}
				})
			})
			return
		}
		a.addTiles(c, sh, t)
	})
}

// addRule is the sheet's hairline between its parts, magpie's `.rule`.
func (a *App) addRule(c *ui.Context, t *ui.Theme) {
	hr := ui.Row(c).Padding(0, 16).Height(1).FillWidth().Shrink(0)
	if t.Dark {
		hr.Background(ui.RGBA(255, 255, 255, 0.08))
	} else {
		hr.Background(ui.RGBA(0, 0, 0, 0.08))
	}
}

// addTiles draws the groups of picks the search keeps, each group's heading
// over its rows.
func (a *App) addTiles(c *ui.Context, sh *addSheet, t *ui.Theme) {
	picks := addPicks()
	words := sh.addWords()
	groups := addGroups(picks, words)

	if len(groups) == 0 {
		// nothing matches: the way out is a provider made by hand, which
		// is what magpie's own none case offers first
		ui.Column(c).Padding(16).Gap(10).Children(func() {
			ui.Text(c, a.text.S.NothingCalled).FontSize(12.5).TextColor(t.TextMuted)
			if b := ui.Button(c, a.text.S.AddItAsCustom); b.Clicked() {
				sh.pickCustom()
				c.Invalidate()
			}
		})
		return
	}

	scroll := ui.Scroll(c).Grow(1).Shrink(1)
	scroll.Children(func() {
		for _, id := range groups {
			// a group's heading: its title, and a word under it where
			// one is wanted. It is a line of its own rather than a tab in
			// a row of them, so the picks under it line up in columns
			// instead of running to whatever width their names make.
			ui.Column(c).Padding(14, 16, 4, 16).Gap(2).Children(func() {
				ui.Text(c, a.addSectionTitle(id)).FontSize(12).FontWeight(600).TextColor(t.Text).SingleLine()
				if hint := a.addSectionHint(id); hint != "" {
					ui.Text(c, hint).FontSize(10.5).TextColor(t.TextMuted).SingleLine()
				}
			})

			// the picks: a grid of three to a line, each tile an icon and
			// a name, the way magpie's own add sheet lays its tiles out.
			// A fixed width per tile keeps the columns even when the
			// names run long, which a wrapping row of them does not.
			grid := ui.Grid(c).Columns(3).Padding(0, 16, 4, 16).Gap(2)
			grid.Children(func() {
				for _, p := range picks {
					if p.Section != id || !addHit(p, words) {
						continue
					}
					a.addTile(c, sh, p, t)
				}
			})
		}

		// the sheet ends with room under the last group, so its last row
		// is not against the dialog's edge
		ui.Row(c).Height(10)
	})
}

// pickCustom puts the sheet's editor on a provider made from scratch.
func (sh *addSheet) pickCustom() {
	sh.Picker = &addPick{Kind: AddCustom, Section: "custom", Name: ""}
}

// addTile is one pick: its picture, its name, and what there is to say of it.
// The custom pick is named by the group's own heading, so it says only what
// it takes.
func (a *App) addTile(c *ui.Context, sh *addSheet, p addPick, t *ui.Theme) {
	row := ui.Row(c).Key("add/"+p.Section+"/"+p.Key).
		AlignItems(ui.Center).Gap(8).Padding(6, 8).Radius(7).
		Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton)
	if row.Clicked() {
		pick := p
		sh.Picker = &pick
		sh.Risk = ""
		c.Invalidate()
		if pick.Kind == AddSub {
			// Entering a subscription's editor is what magpie's tile
			// click is: the sign-in starts itself, and the vendor's
			// page opens when the flow is up. A vendor that may act on
			// the account says so first (the warn), and starts when
			// the reader confirms.
			if a.pvSignIn != nil && a.pvSignIn.Agent == pick.Key {
				// one sign-in per agent at a time: magpie's own
				// StartSignIn cancels the previous one, which makes a
				// second click on a "added" tile look like nothing
				// happened — the first sign-in's URL vanished, the
				// browser refreshed to the new one, and the sheet just
				// shows the same "waiting" again. Skip the start when
				// one is already in flight for this agent; the
				// subEditor already shows its URL and code.
			} else if pick.Risky {
				sh.Risk = pick.Key
			} else {
				a.subSignIn(pick.Key)
			}
		}
		return
	}
	if row.Hovered() || row.Focused() {
		if t.Dark {
			row.Background(ui.RGBA(255, 255, 255, 0.06))
		} else {
			row.Background(ui.RGBA(0, 0, 0, 0.04))
		}
	}

	row.Children(func() {
		// the picture: every pick has one tile for it, the generic mark
		// standing in for the few whose icon the assets have none of, so
		// the names line up in their column whatever a vendor's logo
		// looks like
		tile := ui.Row(c).Center().Width(24).Height(24).Radius(6).
			Background(a.addTileBg(t))
		tile.Children(func() {
			if pic := iconPictureFor(p.Icon); pic != nil {
				ui.Image(c, pic).Size(15, 15).Fit(ui.Contain)
			} else {
				ui.Icon(c, genericIcon()).Size(13, 13).TextColor(t.TextMuted)
			}
		})

		ui.Column(c).Grow(1).Shrink(1).Children(func() {
			ui.Row(c).AlignItems(ui.Center).Gap(5).Children(func() {
				if p.Kind == AddCustom {
					// the custom pick is named by its group's heading
					ui.Text(c, a.text.S.CustomProvider).FontSize(12.5).FontWeight(500).TextColor(t.Text).SingleLine()
				} else {
					ui.Text(c, shortAddName(p.Name)).FontSize(12.5).FontWeight(500).TextColor(t.Text).SingleLine()
				}
				if p.Added {
					ui.Text(c, a.text.S.Added).FontSize(9.5).FontWeight(600).TextColor(t.TextMuted).
						Background(a.badgeBg(t)).Padding(1, 5).Radius(4)
				}
				// the "risky" badge was the same colour as the
				// "already added" one and only ever appeared next to
				// it: it said no more than a subtitle could, and
				// crowded the row. p.Risky stays on the pick so the
				// editor can still say so where it matters, in the
				// sign-in's own hint.
			})
			// the sub line: a preset's host, or a subscription's plans.
			// The custom pick's is its hint, said by the group's heading.
			sub := p.Host
			if sub == "" {
				sub = p.Plans
			}
			if sub != "" {
				ui.Text(c, sub).FontSize(11).TextColor(t.TextMuted).SingleLine()
			}
		})
	})
}

// addClose is the sheet's own close button, in the corner a dialog's keeps:
// a disc holding a cross, which washes as the pointer comes over it. It
// names itself Close for a screen reader rather than showing a bare glyph.
func (a *App) addClose(c *ui.Context, t *ui.Theme) ui.Element {
	btn := ui.Row(c).Center().Width(24).Height(24).Radius(12).
		Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).
		Label(a.text.S.Close)
	if btn.Hovered() || btn.Focused() {
		if t.Dark {
			btn.Background(ui.RGBA(255, 255, 255, 0.10))
		} else {
			btn.Background(ui.RGBA(0, 0, 0, 0.07))
		}
	}
	btn.Children(func() {
		ui.Icon(c, closeIcon()).Size(11, 11).TextColor(t.TextMuted)
	})
	return btn
}

// addTileBg is the tile a pick's picture sits in: the same wash a provider
// row's own icon has, so the sheet and the page read as one.
func (a *App) addTileBg(t *ui.Theme) ui.Color {
	if t.Dark {
		return ui.RGBA(255, 255, 255, 0.06)
	}
	return ui.RGBA(0, 0, 0, 0.045)
}

// badgeBg is the wash behind a pick's badges, in the theme's own terms.
func (a *App) badgeBg(t *ui.Theme) ui.Color {
	if t.Dark {
		return ui.RGBA(255, 255, 255, 0.09)
	}
	return ui.RGBA(0, 0, 0, 0.06)
}

// shortAddName is a name in the sheet without the bracketed rest: a vendor's
// long name says less here than in the editor.
func shortAddName(name string) string {
	if i := strings.LastIndex(name, " ("); i > 0 && strings.HasSuffix(name, ")") {
		return name[:i]
	}
	return name
}

// ---- the editor ----------------------------------------------------------

// addEditor draws the form a pick is added by: the preset's own, a provider
// made from scratch, or the sign-in a subscription is started by.
func (a *App) addEditor(c *ui.Context, sh *addSheet, t *ui.Theme) {
	switch sh.Picker.Kind {
	case AddCustom:
		a.customEditor(c, sh, t)
	case AddSub:
		a.subEditor(c, sh, t)
	default:
		if pr := provider.Preset(sh.Picker.Key); pr != nil {
			a.presetEditor(c, sh, t, pr)
		}
	}
}

// editDraft is the editor's fields: what the reader typed, before it is a
// provider. p is the provider the form is for — the preset's own, or the
// custom one being made. api is the protocol a custom base URL is asked
// at: magpie's own four.
type editDraft struct {
	p         provider.Provider
	api       string
	name      string
	base      string
	key       string
	modelText string
	models    []string
}

// draftFor is the draft a pick's editor edits: read once, so the fields
// survive the frame the way magpie's draft does.
func (sh *addSheet) draftFor(p provider.Provider) *editDraft {
	if sh.draft == nil {
		sh.draft = &editDraft{p: p}
	}
	return sh.draft
}

// presetEditor is a preset's form: its key where it wants one, the vendor's
// host said, and the models it asks for.
func (a *App) presetEditor(c *ui.Context, sh *addSheet, t *ui.Theme, pr *provider.PresetDef) {
	d := sh.draftFor(provider.Provider{
		ID: pr.ID, Name: pr.Name, Icon: pr.Icon, Preset: pr.ID,
		Chat: pr.Chat, Responses: pr.Responses, Anthropic: pr.Anthropic, Decide: pr.Decide,
		Catalog: pr.Catalog, Website: pr.Website, KeysURL: pr.KeysURL,
	})

	form := ui.Scroll(c).Grow(1).Shrink(1).Padding(12, 16)
	form.Children(func() {
		if note := pr.Note; note != "" {
			ui.Text(c, note).FontSize(11.5).TextColor(t.TextMuted).Padding(0, 0, 8, 0)
		}

		// the key, where the preset wants one: a local server's is
		// optional, which is said rather than asked
		if !pr.NoKey {
			ui.Column(c).Gap(4).Padding(0, 0, 8, 0).Children(func() {
				ui.Text(c, a.text.S.APIKey).FontSize(11.5).FontWeight(500).TextColor(t.Text)
				if pr.KeyHint != "" {
					ui.Text(c, pr.KeyHint).FontSize(10.5).TextColor(t.TextMuted)
				} else if pr.KeysURL != "" {
					ui.Text(c, pr.KeysURL).FontSize(10.5).TextColor(t.TextMuted)
				}
				ui.TextInput(c, &d.key).Label(a.text.S.APIKey).Background(t.Background)
			})
		} else if pr.KeyHint != "" {
			ui.Text(c, pr.KeyHint).FontSize(11.5).TextColor(t.TextMuted).Padding(0, 0, 8, 0)
		}

		// the models: a preset with a list of its own asks which of them
		// are exposed, one id to a line
		if len(pr.Models) > 0 {
			ui.Column(c).Gap(6).Padding(0, 0, 8, 0).Children(func() {
				ui.Text(c, a.text.S.Models).FontSize(11.5).FontWeight(500).TextColor(t.Text)
				ui.Text(c, a.text.S.PasteModelIDs).FontSize(10.5).TextColor(t.TextMuted)
				ui.TextArea(c, &d.modelText).Height(80).Background(t.Background)
			})
		}
	})
}

// customEditor is a provider made from scratch: its name, the protocol
// its base URL is asked at — magpie's own four, each with its hint — the
// URL, its key, and the models it asks for.
func (a *App) customEditor(c *ui.Context, sh *addSheet, t *ui.Theme) {
	d := sh.draftFor(provider.Provider{})
	if d.api == "" {
		d.api = "openai"
	}

	// the four protocols, magpie's own segmented over the base URL, with
	// the hint each option's tooltip carries
	apiOpts := []struct{ id, label, hint string }{
		{"openai", "OpenAI 兼容", "…/v1 — chat completions，供应商支持时也含 responses"},
		{"responses", "OpenAI Responses", "…/v1 — 用于只提供 Responses API、不提供 chat completions 的供应商"},
		{"anthropic", "Anthropic 兼容", "根地址，即 ANTHROPIC_BASE_URL 的值"},
		{"decide", "System One", "决策 API 的根地址，在其下 POST …/systemone：它的模型只作路由组的分类器，不给 Agent 使用"},
	}
	apiIdx := 0
	for i, o := range apiOpts {
		if o.id == d.api {
			apiIdx = i
		}
	}

	form := ui.Scroll(c).Grow(1).Shrink(1).Padding(12, 16)
	form.Children(func() {
		ui.Text(c, a.text.S.AnyCompatibleURL).FontSize(11.5).TextColor(t.TextMuted).Padding(0, 0, 8, 0)

		ui.Column(c).Gap(4).Padding(0, 0, 8, 0).Children(func() {
			ui.Text(c, a.text.S.Name).FontSize(11.5).FontWeight(500).TextColor(t.Text)
			ui.TextInput(c, &d.name).Label(a.text.S.Name).Background(t.Background)
		})
		ui.Column(c).Gap(6).Padding(0, 0, 8, 0).Children(func() {
			ui.Text(c, a.text.S.APIBase).FontSize(11.5).FontWeight(500).TextColor(t.Text)
			seg := ui.Segmented(c, &apiIdx, apiOpts[0].label, apiOpts[1].label, apiOpts[2].label, apiOpts[3].label).AlignSelf(ui.Start)
			if seg.Changed() {
				d.api = apiOpts[apiIdx].id
				c.Invalidate()
			}
			url := ui.TextInput(c, &d.base).Label(a.text.S.APIBase).Background(t.Background)
			if d.api == "anthropic" || d.api == "decide" {
				url.Placeholder("https://…")
			} else {
				url.Placeholder("https://…/v1")
			}
			ui.Text(c, apiOpts[apiIdx].hint).FontSize(10.5).TextColor(t.TextMuted)
		})
		ui.Column(c).Gap(4).Padding(0, 0, 8, 0).Children(func() {
			ui.Text(c, a.text.S.APIKey).FontSize(11.5).FontWeight(500).TextColor(t.Text)
			ui.TextInput(c, &d.key).Label(a.text.S.APIKey).Background(t.Background)
		})
		ui.Column(c).Gap(6).Padding(0, 0, 8, 0).Children(func() {
			ui.Text(c, a.text.S.Models).FontSize(11.5).FontWeight(500).TextColor(t.Text)
			ui.Text(c, a.text.S.PasteModelIDs).FontSize(10.5).TextColor(t.TextMuted)
			ui.TextArea(c, &d.modelText).Height(80).Background(t.Background)
		})
	})
}

// subEditor is a subscription's pick: what signing in means, and the way
// out. The sign-in itself is magpie's own flow, started by a click on the
// pick, which opens the vendor's page in the browser — except one moved
// onto its plugin, whose sign-in is the plugin's own flow (the questions
// it asks, then the page or the key), which the sheet draws as it goes.
func (a *App) subEditor(c *ui.Context, sh *addSheet, t *ui.Theme) {
	pick := sh.Picker
	signing := a.pvSignIn != nil && a.pvSignIn.Agent == pick.Key
	asking := a.pvAsk != nil && a.pvAsk.Agent == pick.Key

	ui.Column(c).Padding(12, 16).Gap(10).Grow(1).Shrink(1).Children(func() {
		if pick.Plans != "" {
			ui.Text(c, pick.Plans).FontSize(11.5).TextColor(t.TextMuted)
		}
		if asking {
			a.pluginAskView(c, t, a.pvAsk)
			return
		}
		// A vendor that may act on the account says so before its
		// sign-in opens — magpie's own warn — and starts when the
		// reader confirms (仍 然 登 录).
		if sh.Risk == pick.Key {
			a.subRiskWarn(c, t, sh, pick)
			return
		}
		if signing && a.pvSignIn.Err == "" {
			ui.Text(c, a.text.S.SigningIn).FontSize(12).TextColor(t.Text)
			if a.pvSignIn.Code != "" {
				ui.Text(c, a.pvSignIn.Code).Font("monospace").FontSize(13).FontWeight(600).TextColor(t.Text)
			}
			if a.pvSignIn.URL != "" {
				ui.Text(c, a.pvSignIn.URL).FontSize(11).TextColor(t.TextMuted)
			}
			ui.Row(c).Gap(8).Children(func() {
				if b := ui.Button(c, a.text.S.OpenInBrowser); b.Clicked() {
					a.openBrowser(a.pvSignIn.URL)
				}
				if b := ui.Button(c, a.text.S.Cancel); b.Clicked() {
					a.addGiveUp()
				}
			})
			return
		}
		if signing {
			// the sign-in failed: why, said where the click was, and
			// the pick's own way back to trying again under it
			ui.Text(c, a.pvSignIn.Err).FontSize(11.5).TextColor(ui.Hex("#ff3b30")).LineHeight(1.4)
		}
		if pick.Added {
			ui.Text(c, a.text.S.AlreadyAdded).FontSize(12).TextColor(t.TextMuted)
		}
		if pick.Hint != "" {
			ui.Text(c, pick.Hint).FontSize(12).TextColor(t.TextMuted)
		}
		ui.Text(c, a.text.S.SignInHint).FontSize(12).TextColor(t.TextMuted)
		ui.Row(c).Gap(8).Children(func() {
			if b := ui.Button(c, a.text.S.SignIn); b.Clicked() {
				a.subSignIn(pick.Key)
				c.Invalidate()
			}
		})
	})
}

// subRiskWarn is the warn a risky subscription's sign-in opens behind:
// what the vendor may do to an account it sees used through Echo, and
// which accounts the subscription is for — magpie's own warn, its
// 取消 and 仍然登录 the two ways out.
func (a *App) subRiskWarn(c *ui.Context, t *ui.Theme, sh *addSheet, pick *addPick) {
	card := ui.Column(c).FillWidth().Padding(12, 14).Radius(8).Gap(6).
		Background(t.Background).Border(1, t.Border)
	card.Children(func() {
		ui.Row(c).AlignItems(ui.Center).Gap(8).FillWidth().Children(func() {
			ui.Icon(c, genericIcon()).Size(15, 15).TextColor(t.Text)
			ui.Text(c, pick.Name+" 账号可能被封禁").FontSize(12.5).FontWeight(600).TextColor(t.Text).SingleLine()
			ui.Spacer(c)
			if b := ui.Button(c, a.text.S.Cancel); b.Clicked() {
				sh.Picker = nil
				sh.Risk = ""
				c.Invalidate()
			}
			goBtn := ui.Row(c).AlignItems(ui.Center).Padding(5, 12).Radius(6).Background(t.Accent).
				Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).Label("仍然登录")
			clicked := goBtn.Clicked()
			if clicked {
				sh.Risk = ""
				a.subSignIn(pick.Key)
			}
			goBtn.Children(func() {
				ui.Text(c, "仍然登录").FontSize(11.5).FontWeight(600).TextColor(ui.Hex("#ffffff")).SingleLine()
			})
		})
		if pick.Hint != "" {
			ui.Text(c, pick.Hint).FontSize(11.5).TextColor(t.Text).LineHeight(1.45)
		}
		if pick.RiskNote != "" {
			ui.Text(c, pick.RiskNote).FontSize(11.5).TextColor(t.TextMuted).LineHeight(1.45)
		}
	})
}

// addSave writes the provider the editor holds: a preset with its key, or a
// provider made from scratch with its name, base and key (magpie's own save
// of a new provider, provider.Add).
func (a *App) addSave(c *ui.Context, sh *addSheet) {
	d := sh.draft
	if d == nil {
		return
	}
	if sh.Picker.Kind == AddCustom {
		name := strings.TrimSpace(d.name)
		base := strings.TrimSpace(d.base)
		key := strings.TrimSpace(d.key)
		if name == "" || base == "" {
			return
		}
		p := provider.Provider{ID: name, Name: name, Key: key}
		// the base goes to the API the protocol picked is asked at —
		// magpie keeps one URL to a protocol
		switch d.api {
		case "responses":
			p.Responses = base
		case "anthropic":
			p.Anthropic = base
		case "decide":
			p.Decide = base
		default:
			p.Chat = base
		}
		go func(id string) {
			if _, err := provider.Add(p); err != nil {
				return
			}
			if models := d.chosenModels(); len(models) > 0 {
				_ = provider.SetModels(id, models)
			}
			a.win.Update(func() {
				a.closeAdd()
				a.Reload()
			})
		}(name)
		return
	}

	pr := provider.Preset(sh.Picker.Key)
	if pr == nil {
		return
	}
	p := d.p
	key := strings.TrimSpace(d.key)
	models := d.chosenModels()
	go func(id string) {
		if key != "" {
			if err := provider.AddKey(id, p.Name, key, ""); err != nil {
				return
			}
		}
		if _, err := provider.Add(p); err != nil {
			return
		}
		if len(models) > 0 {
			_ = provider.SetModels(id, models)
		}
		a.win.Update(func() {
			a.closeAdd()
			a.Reload()
		})
	}(pr.ID)
}

// chosenModels is what the models field holds, one id to a line, blanks left
// out.
func (d *editDraft) chosenModels() []string {
	var out []string
	for _, m := range strings.Split(d.modelText, "\n") {
		if m = strings.TrimSpace(m); m != "" {
			out = append(out, m)
		}
	}
	return out
}

// ---- the subscription's sign-in -----------------------------------------

// startSignIn starts a subscription's sign-in, which is magpie's own flow
// (provider.StartSignIn): the vendor's page opens in the browser and the
// account lands in the list once it is signed in.
func (a *App) startSignIn(agent string) {
	go func() {
		st, err := provider.StartSignIn(agent)
		if err != nil {
			// said in the sheet: the built-in refusing (a moved one
			// signs in through its plugin), a proxy that can't be
			// used, a CLI that can't be looked up
			a.win.Update(func() {
				a.pvSignIn = &signIn{Agent: agent, Err: err.Error()}
				a.invalidate()
			})
			return
		}
		a.win.Update(func() {
			a.pvSignIn = &signIn{ID: st.ID, Agent: st.Agent, URL: st.URL, Code: st.Code}
			a.openBrowser(st.URL)
		})
		a.watchSignIn(st.ID)
	}()
}

// watchSignIn reads the sign-in's progress until it is over: a done one
// reloads the list, where the account now shows; a failed one keeps what
// went wrong, said in the sheet.
func (a *App) watchSignIn(id string) {
	for {
		st, ok := provider.SignInStatus(id)
		if !ok {
			return
		}
		switch st.State {
		case "done", "failed", "canceled":
			a.win.Update(func() {
				if a.pvSignIn != nil && a.pvSignIn.ID == id {
					if st.State == "failed" {
						a.pvSignIn.Err = st.Error
					} else {
						a.pvSignIn = nil
					}
				}
				// a sign-in that worked opens the provider's editor, as
				// magpie's own does: the account in, the settings in
				// front of the reader
				if st.State == "done" {
					a.pvAdding = nil
					a.pvAsk = nil
					a.openProviderEditor(st.Agent)
				}
				a.Reload()
			})
			return
		}
		time.Sleep(time.Second)
	}
}

// openBrowser opens a URL in the reader's own browser, the way magpie's
// own sign-in does: `open` on macOS, `xdg-open` on Linux, and the shell's
// own start on Windows.
func (a *App) openBrowser(url string) {
	if url == "" {
		return
	}
	var cmd string
	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
	case "windows":
		cmd = "rundll32"
	default:
		cmd = "xdg-open"
	}
	args := []string{url}
	if runtime.GOOS == "windows" {
		args = []string{"url.dll,FileProtocolHandler", url}
	}
	_ = proc.Command(cmd, args...).Start()
}

// signIn is the sign-in the sheet started, read as it goes. Err is what
// went wrong starting it or while it ran, said where the click was — a
// sign-in that fails silently is indistinguishable from a dead button.
type signIn struct {
	ID    string
	Agent string
	URL   string
	Code  string
	Err   string
}
