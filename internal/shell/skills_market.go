//go:build nogui && echo

package shell

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/yetone/magpie/internal/library"
)

// The market under the Skills page: what skills.sh offers, searched or
// popular, each a card with its source, its install count, and the button
// that adds it.
//
// It is magpie's own Discover section (library.js's discover and skillCard)
// drawn in native UI: the same cards in the same order, the same search that
// asks once typing pauses, and the same writes behind Add.

// skMarketKick asks the market once, when the page is first drawn.
func (a *App) skillsMarket(c *ui.Context, t *ui.Theme) {
	q := strings.TrimSpace(a.skMarketQuery)

	ui.Column(c).FillWidth().Gap(8).Children(func() {
		ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
			ui.Text(c, a.text.S.LibDiscover).FontSize(12).FontWeight(600).TextColor(t.TextMuted).Shrink(0)
			ui.Spacer(c)

			note := a.text.S.LibPopularOnSkillsSh
			if q != "" && a.skMarketItems != nil {
				if len(a.skMarketItems) == 1 {
					note = a.text.S.LibOneResult
				} else {
					note = strings.ReplaceAll(a.text.S.LibNResults, "{n}", fmt.Sprint(len(a.skMarketItems)))
				}
			}
			ui.Text(c, note).FontSize(10.5).TextColor(t.TextMuted).SingleLine()
		})

		ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
			// ui.SearchField's placeholder is "Search", and magpie's market
			// says what it searches. The field is built here so that the
			// placeholder is the page's own.
			f := a.skSearchField(c, &a.skMarketQuery, a.text.S.LibSkillsSearchSh)
			// magpie asks once typing pauses: a query a keystroke long
			// would ask on every letter
			if f.Changed() {
				a.skMarketKickAsk()
			}
			if f.Submitted() {
				a.skMarketNow()
			}
			if a.skMarketBusy {
				ui.Spinner(c).Size(12, 12).Shrink(0)
			}
		})

		if a.skMarketErr != "" {
			ui.Text(c, a.skMarketErr).FontSize(11.5).TextColor(t.Danger).Wrap()
		}

		switch {
		case a.skMarketItems == nil && a.skMarketBusy:
			a.skillsMarketSkeleton(c, t)
		case a.skMarketItems == nil:
			// nothing asked for yet
		case len(a.skMarketItems) == 0:
			msg := a.text.S.LibNothingToShow
			if q != "" {
				msg = strings.ReplaceAll(a.text.S.LibNothingMatches, "{q}", q)
			}
			ui.Text(c, msg).FontSize(11.5).TextColor(t.TextMuted).Wrap()
		default:
			a.skillsMarketCards(c, t)
		}
	})
}

// skSearchField is the market's finder: magpie's .mk-find — one white box
// with a border, the magnifier inside it on the left, and the input filling
// the rest.
func (a *App) skSearchField(c *ui.Context, query *string, placeholder string) ui.Element {
	return a.skFindField(c, query, placeholder, true)
}

// skFindField is magpie's finder drawn once for both places that need it: the
// skills finder (library.css's .lib-find) and the market's search
// (.mk-find). The box is the field — white on light, the card's colour on
// dark, with the magnifier inside it — and the input sits in it with no look
// of its own, which is what puts the glyph inside the box rather than beside
// it. showClear adds the button that empties it while it holds anything.
//
// The input is returned, so the caller watches it for Changed and Submitted.
func (a *App) skFindField(c *ui.Context, query *string, placeholder string, showClear bool) ui.Element {
	t := c.Theme()

	box := ui.Row(c).AlignItems(ui.Center).Gap(8).
		Padding(0, 12).Height(36).Radius(10).
		Border(1, t.Border).Background(skCardBg(t)).
		Grow(1).Shrink(1)

	var field ui.Element
	box.Children(func() {
		ui.Icon(c, searchIcon()).Size(13, 13).TextColor(t.TextMuted).Shrink(0)
		field = ui.TextInputBase(c, query).
			Grow(1).Shrink(1).
			Placeholder(placeholder).
			Label(placeholder)
		if showClear && *query != "" {
			if b := a.skIconButton(c, t, closeIcon(), a.text.S.LibClear, false); b.Clicked() {
				*query = ""
				c.Invalidate()
			}
		}
	})
	return field
}

// skillsMarketCards draws the market's cards, three to a row.
func (a *App) skillsMarketCards(c *ui.Context, t *ui.Theme) {
	items := a.skMarketItems
	perRow := 3

	ui.Column(c).FillWidth().Gap(8).Children(func() {
		for i := 0; i < len(items); i += perRow {
			end := i + perRow
			if end > len(items) {
				end = len(items)
			}
			chunk := items[i:end]
			ui.Row(c).AlignItems(ui.Stretch).Gap(8).FillWidth().Children(func() {
				for _, it := range chunk {
					item := it
					ui.Column(c).Grow(1).Shrink(1).Basis(0).Gap(6).
						Padding(10, 12).Radius(10).Border(1, t.Border).
						Children(func() {
							a.skillMarketCard(c, t, item)
						})
				}
				// keep the last row's cards the same width as the rest
				for i := end - i; i < perRow; i++ {
					ui.Box(c).Grow(1).Shrink(1).Basis(0)
				}
			})
		}
	})
}

// skillMarketCard is one market skill: its name, its source, its
// description, and its install count beside Add. The card opens the sheet
// that says more of it, as magpie's own does.
func (a *App) skillMarketCard(c *ui.Context, t *ui.Theme, x library.MarketSkill) {
	card := ui.Column(c).FillWidth().Gap(6).
		Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).
		Label(strings.ReplaceAll(a.text.S.LibAboutName, "{name}", x.Name))
	if card.Hovered() || card.Focused() {
		card.Background(a.skHoverBg(t))
	}
	if card.Clicked() {
		a.skMarketSheet = x.ID
		c.Invalidate()
	}

	card.Children(func() {
		ui.Row(c).AlignItems(ui.Center).Gap(7).Children(func() {
			// the picture: the repository's own, else its initial on a tile
			if p := iconPictureFor(x.Icon); p != nil {
				ui.Image(c, p).Size(18, 18).Radius(4).Shrink(0)
			} else {
				tile := ui.Row(c).Center().Width(18).Height(18).Radius(4).Shrink(0)
				if t.Dark {
					tile.Background(ui.RGBA(255, 255, 255, 0.08))
				} else {
					tile.Background(ui.RGBA(0, 0, 0, 0.055))
				}
				tile.Children(func() {
					ui.Text(c, initialOf(x.Source)).FontSize(9.5).FontWeight(700).TextColor(t.TextMuted)
				})
			}

			ui.Text(c, x.Name).FontSize(12.5).FontWeight(600).TextColor(t.Text).SingleLine()
			if x.Official {
				a.skTag(c, t, a.text.S.LibOfficialBadge, "accent")
			}
		})

		ui.Text(c, x.Source).Font("monospace").FontSize(10).TextColor(t.TextMuted).SingleLine()

		desc := x.Description
		if desc == "" {
			desc = a.text.S.LibNoDescription
		}
		ui.Text(c, desc).FontSize(11).TextColor(t.TextMuted).MaxLines(3).Wrap()

		ui.Spacer(c)

		ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
			ui.Icon(c, downloadIcon()).Size(10, 10).TextColor(t.TextMuted).Shrink(0)
			ui.Text(c, compactCount(x.Installs)).FontSize(10.5).TextColor(t.TextMuted).Shrink(0)
			ui.Spacer(c)

			if x.Have != "" {
				a.skTag(c, t, a.text.S.Added, "muted")
				return
			}
			busy := a.skBusy == "market/skill:"+x.ID
			label := a.text.S.Add
			if busy {
				label = a.text.S.LibAdding
			}
			if b := ui.Button(c, label).Shrink(0).Disabled(busy || a.skBusy != ""); b.Clicked() {
				a.skInstallMarket(x.Source, x.SkillID)
			}
		})
	})
}

// skillsMarketSkeleton is the market before its first answer: six cards in
// outline, so the page keeps its shape while the answer comes. magpie's own
// draws the same placeholders.
func (a *App) skillsMarketSkeleton(c *ui.Context, t *ui.Theme) {
	const perRow = 3
	bg := ui.RGBA(0, 0, 0, 0.05)
	if t.Dark {
		bg = ui.RGBA(255, 255, 255, 0.06)
	}

	ui.Column(c).FillWidth().Gap(8).Children(func() {
		for row := 0; row < 2; row++ {
			ui.Row(c).AlignItems(ui.Stretch).Gap(8).FillWidth().Children(func() {
				for i := 0; i < perRow; i++ {
					ui.Column(c).Grow(1).Shrink(1).Basis(0).Gap(6).
						Padding(10, 12).Radius(10).Border(1, t.Border).Children(func() {
						ui.Row(c).AlignItems(ui.Center).Gap(7).Children(func() {
							ui.Box(c).Width(18).Height(18).Radius(4).Shrink(0).Background(bg)
							ui.Column(c).Grow(1).Gap(3).Children(func() {
								ui.Box(c).Height(9).Width(90).Radius(3).Background(bg)
								ui.Box(c).Height(7).Width(60).Radius(3).Background(bg)
							})
						})
						ui.Box(c).Height(8).Width(140).Radius(3).Background(bg)
					})
				}
			})
		}
	})
}

// initialOf is a repository's initial, for a card with no picture.
func initialOf(s string) string {
	s = strings.TrimLeft(s, "/")
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return strings.ToUpper(string(r))
		}
	}
	return "?"
}

// compactCount is a count as magpie's compact() writes it: 1.2K, 3M.
func compactCount(n int) string {
	switch {
	case n >= 1000000:
		v := float64(n) / 1000000
		if n >= 10000000 {
			return fmt.Sprintf("%.0fM", v)
		}
		return strings.TrimSuffix(fmt.Sprintf("%.1f", v), ".0") + "M"
	case n >= 1000:
		v := float64(n) / 1000
		if n >= 10000 {
			return fmt.Sprintf("%.0fK", v)
		}
		return strings.TrimSuffix(fmt.Sprintf("%.1f", v), ".0") + "K"
	}
	return fmt.Sprint(n)
}
