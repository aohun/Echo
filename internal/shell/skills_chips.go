//go:build nogui && echo

package shell

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/egoist/mygo/ui"
)

// The pieces the Skills page's rows and groups share: their pictures, the
// chips that say which agents have a skill, and the small marks beside a
// name. They are drawn by skills_view.go and by the sheets.

var (
	skillOnce sync.Once
	skillPic  *ui.SVG

	folderOnce sync.Once
	folderPic  *ui.SVG

	updateOnce sync.Once
	updatePic  *ui.SVG

	trashOnce sync.Once
	trashPic  *ui.SVG

	globeOnce sync.Once
	globePic  *ui.SVG

	downloadOnce sync.Once
	downloadPic  *ui.SVG
)

// skillIcon is the glyph a skill's row takes when it has no picture of its
// own: a page with a folded corner.
func skillIcon() *ui.SVG {
	skillOnce.Do(func() {
		skillPic = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" width="15" height="15"><g fill="none" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round"><path d="M4 1.9h5.2L12.6 5.3v8.8H4z"/><path d="M9.2 1.9v3.4h3.4"/><path d="M6 8.4h4.4M6 10.8h3"/></g></svg>`))
	})
	return skillPic
}

// folderIcon is a group's picture when no skill in it has one.
func folderIcon() *ui.SVG {
	folderOnce.Do(func() {
		folderPic = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" width="14" height="14"><path d="M1.9 4.2A1.3 1.3 0 0 1 3.2 2.9h2.6l1.3 1.6h5.7a1.3 1.3 0 0 1 1.3 1.3v6.1a1.3 1.3 0 0 1-1.3 1.3H3.2a1.3 1.3 0 0 1-1.3-1.3z" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linejoin="round"/></svg>`))
	})
	return folderPic
}

// updateIcon is a row's "fetch it from GitHub again" button.
func updateIcon() *ui.SVG {
	updateOnce.Do(func() {
		updatePic = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" width="12" height="12"><g fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M13.2 8a5.2 5.2 0 1 1-1.6-3.7L13.4 6"/><path d="M13.4 2.4V6h-3.6"/></g></svg>`))
	})
	return updatePic
}

// trashIcon is a row's "remove it" button.
func trashIcon() *ui.SVG {
	trashOnce.Do(func() {
		trashPic = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" width="12" height="12"><g fill="none" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round"><path d="M2.6 4.3h10.8M6.2 4.3V2.7h3.6v1.6"/><path d="M4 4.3l.6 8.5a1 1 0 0 0 1 .9h4.8a1 1 0 0 0 1-.9l.6-8.5"/><path d="M6.6 6.8v4.6M9.4 6.8v4.6"/></g></svg>`))
	})
	return trashPic
}

// globeIcon is a repository's "open it" button.
func globeIcon() *ui.SVG {
	globeOnce.Do(func() {
		globePic = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" width="12" height="12"><g fill="none" stroke="currentColor" stroke-width="1.4"><circle cx="8" cy="8" r="5.6"/><path d="M2.6 8h10.8M8 2.4c1.6 1.6 2.4 3.5 2.4 5.6S9.6 12 8 13.6C6.4 12 5.6 10.1 5.6 8S6.4 4 8 2.4z"/></g></svg>`))
	})
	return globePic
}

// downloadIcon is the market's install count.
func downloadIcon() *ui.SVG {
	downloadOnce.Do(func() {
		downloadPic = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" width="11" height="11"><g fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M8 2.6v7.6M4.9 7.2 8 10.3l3.1-3.1M2.8 13.2h10.4"/></g></svg>`))
	})
	return downloadPic
}

// ---- the agent chips ----------------------------------------------------

// The hand's geometry: a chip's width, how much of the chip before it a
// stacked one covers, and the gap a fanned one leaves beside it.
const (
	skChipW     = 22
	skChipStack = 7
	skChipGap   = 2
)

// skChipReserve is the room a hand of n chips takes once it fans out, less
// the room it takes stacked: what a row holds open while the hand is still so
// that fanning it moves no other piece of the row, as magpie's own
// .lib-agents::before does. lead says an All chip leads the hand, which adds
// the gap before it once the hand fans.
func skChipReserve(n int, lead bool) float32 {
	fan := float32(n-1) * (skChipW + skChipGap)
	stacked := float32(n-1) * (skChipW - skChipStack)
	if lead {
		fan += skChipGap
	}
	return fan - stacked
}

// skAgentChips is a row's chips: one for each agent that can take skills,
// lit when it has this one and dim when it hasn't. A click turns it on or
// off, which is what set is told.
//
// It is magpie's own .lib-ag chip: the row's agents read at a glance, and
// the one place a skill is given to one agent. In a row (stack) the chips sit
// like a hand of cards — overlapping, and fanned out when the pointer or the
// focus comes to them — with an All chip when two agents can take it.
func (a *App) skAgentChips(c *ui.Context, t *ui.Theme, all []LibAgentRow, set map[string]bool, stack bool, click func(id string, on bool), onAll func(on bool)) {
	if len(all) == 0 {
		return
	}
	row := ui.Row(c).AlignItems(ui.Center).Shrink(0)
	spread := !stack || row.Hovered() || row.Focused()
	if spread {
		row.Gap(skChipGap)
	}

	everyOn := true
	for _, ag := range all {
		if !set[ag.ID] {
			everyOn = false
			break
		}
	}

	lead := onAll != nil && len(all) >= 2
	row.Children(func() {
		if lead {
			tip := a.text.S.LibGiveItToAll
			if everyOn {
				tip = strings.ReplaceAll(a.text.S.LibEveryAgentHasIt, "{n}", fmt.Sprint(len(all)))
			} else {
				tip = strings.ReplaceAll(tip, "{n}", fmt.Sprint(len(all)))
			}
			a.skAllChip(c, t, everyOn, tip, onAll)
		}
		// the room the hand takes fanned out, held open while it is stacked
		// so that fanning it leaves the row where it was
		if stack && !spread {
			ui.Row(c).Width(skChipReserve(len(all), lead)).Height(1).Shrink(0)
		}
		for i, ag := range all {
			agent := ag
			on := set[agent.ID]
			tip := strings.ReplaceAll(a.text.S.LibGiveItTo, "{agent}", agent.Name)
			if on {
				tip = strings.ReplaceAll(a.text.S.LibAgentHasIt, "{agent}", agent.Name)
			}
			first := i == 0 || spread
			a.skAgentChip(c, t, agent, on, stack, first, tip, func() {
				if click != nil {
					click(agent.ID, !on)
				}
			})
		}
	})
}

// skGroupChips is a group's or a part's chips: lit when the agent has every
// one of its skills, half lit when it has some, and a click gives it the
// rest or takes them all away. An All chip gives them to every agent at
// once, as magpie's own group heading does.
func (a *App) skGroupChips(c *ui.Context, t *ui.Theme, skills []SkillRow, all []LibAgentRow, repo string) {
	if len(skills) == 0 || len(all) == 0 {
		return
	}
	n := len(skills)

	row := ui.Row(c).AlignItems(ui.Center).Shrink(0)
	spread := row.Hovered() || row.Focused()
	if spread {
		row.Gap(skChipGap)
	}

	kOf := func(id string) int {
		k := 0
		for _, s := range skills {
			if skillGives(s, id) || skillAlways(s, id) {
				k++
			}
		}
		return k
	}
	everyOn := true
	for _, ag := range all {
		if kOf(ag.ID) != n {
			everyOn = false
			break
		}
	}

	row.Children(func() {
		lead := len(all) >= 2
		if lead {
			tip := a.text.S.LibGiveAllToEvery
			if everyOn {
				tip = a.text.S.LibEveryAgentHasAll
			}
			a.skAllChip(c, t, everyOn, tip, func(on bool) {
				ids := make([]string, 0, len(all))
				for _, ag := range all {
					ids = append(ids, ag.ID)
				}
				names := make([]string, 0, n)
				for _, s := range skills {
					names = append(names, s.Name)
				}
				a.skSomeSkills(names, ids, on)
			})
		}
		// the room the hand takes fanned out, held open while it is stacked
		// so that fanning it leaves the heading where it was
		if !spread {
			ui.Row(c).Width(skChipReserve(len(all), lead)).Height(1).Shrink(0)
		}
		for i, ag := range all {
			agent := ag
			k := kOf(agent.ID)

			tip := strings.ReplaceAll(a.text.S.LibGiveAgentAll, "{agent}", agent.Name)
			switch {
			case k == n:
				tip = strings.ReplaceAll(a.text.S.LibAgentHasAll, "{agent}", agent.Name)
			case k > 0:
				tip = strings.NewReplacer("{agent}", agent.Name, "{n}", fmt.Sprint(k)).Replace(a.text.S.LibAgentHasNOfThem)
			}

			first := i == 0 || spread
			a.skAgentChip(c, t, agent, k > 0, true, first, tip, func() {
				names := make([]string, 0, n)
				for _, s := range skills {
					names = append(names, s.Name)
				}
				if k == n {
					a.skSomeSkills(names, []string{agent.ID}, false)
				} else {
					a.skSomeSkills(names, []string{agent.ID}, true)
				}
			})
		}
	})
}

// skAgentChip draws one agent's chip: its picture, lit when it has the item
// and dim when it hasn't, with tip as its tooltip. stacked gives every chip
// the card's own face — which is what lets a hand of chips cut each out of
// the one under it — and slides all but the first under the chip before.
// It reports whether the chip was clicked.
func (a *App) skAgentChip(c *ui.Context, t *ui.Theme, agent LibAgentRow, on, stacked, first bool, tip string, click func()) bool {
	chip := ui.Row(c).Center().Width(skChipW).Height(skChipW).Radius(6).Shrink(0).
		Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).
		Label(agent.Name)
	chip.Tooltip(tip)

	switch {
	case stacked:
		// an opaque face, so the hand reads as cards rather than a smear;
		// one the agent has is edged in the stronger line, as magpie's is
		chip.Background(skCardBg(t))
		if on {
			chip.Border(1, t.Border)
		} else {
			chip.Border(1, t.Border.Mix(t.Background, 0.5))
		}
	case on:
		if t.Dark {
			chip.Background(ui.RGBA(255, 255, 255, 0.10))
		} else {
			chip.Background(ui.RGBA(0, 0, 0, 0.055))
		}
		chip.Border(1, t.Border)
	}
	if !on && (chip.Hovered() || chip.Focused()) {
		chip.Background(a.skHoverBg(t))
	}
	if stacked && !first {
		chip.Margin(0, 0, 0, -skChipStack)
	}

	clicked := chip.Clicked()
	if clicked && click != nil {
		click()
	}

	op := float32(0.35)
	if on {
		op = 1
	}
	chip.Children(func() {
		if p := iconPictureFor(agent.Icon); p != nil {
			ui.Image(c, p).Size(14, 14).Radius(3).Opacity(op)
		} else {
			ui.Icon(c, genericIcon()).Size(12, 12).TextColor(t.TextMuted)
		}
	})
	return clicked
}

// skAllChip is the "全部" chip ahead of a row's chips: one click gives the
// item to every agent shown, or, when they all have it, takes it from every
// one. It is drawn only for two or more agents, as magpie's own is.
func (a *App) skAllChip(c *ui.Context, t *ui.Theme, allOn bool, tip string, click func(on bool)) {
	chip := ui.Row(c).Center().Height(skChipW).Padding(0, 7).Radius(6).Shrink(0).
		Margin(0, 5, 0, 0).
		Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).
		Label(tip)
	chip.Tooltip(tip)

	var textC ui.Color
	switch {
	case allOn:
		if t.Dark {
			chip.Background(ui.RGBA(255, 255, 255, 0.10))
		} else {
			chip.Background(ui.RGBA(0, 0, 0, 0.055))
		}
		chip.Border(1, t.Border)
		textC = t.Text
	case chip.Hovered() || chip.Focused():
		chip.Background(a.skHoverBg(t))
		textC = t.Text
	default:
		textC = t.TextMuted
	}

	if chip.Clicked() && click != nil {
		click(!allOn)
	}
	chip.Children(func() {
		ui.Text(c, a.text.S.FilterAll).FontSize(11).FontWeight(500).TextColor(textC).SingleLine()
	})
}

// ---- the bulk buttons ---------------------------------------------------

// skEverySkillButtons are "Turn all on" and "Turn all off" for every agent
// that can take skills.
func (a *App) skEverySkillButtons(c *ui.Context, t *ui.Theme, all []LibAgentRow) {
	ids := make([]string, 0, len(all))
	for _, ag := range all {
		ids = append(ids, ag.ID)
	}

	everyOn := true
	for _, s := range a.data.Library.Skills {
		for _, id := range ids {
			if !skillGives(s, id) {
				everyOn = false
				break
			}
		}
		if !everyOn {
			break
		}
	}

	on := ui.Button(c, a.text.S.LibTurnAllOn).Shrink(0).Disabled(everyOn || a.skBusy != "")
	on.Tooltip(strings.ReplaceAll(a.text.S.LibTurnAllOnTip, "{n}", fmt.Sprint(len(ids))))
	if on.Clicked() {
		a.skEverySkill(ids, true)
	}

	anyOn := false
	for _, s := range a.data.Library.Skills {
		for _, id := range ids {
			if skillGives(s, id) {
				anyOn = true
				break
			}
		}
		if anyOn {
			break
		}
	}
	off := ui.Button(c, a.text.S.LibTurnAllOff).Shrink(0).Disabled(!anyOn || a.skBusy != "")
	off.Tooltip(strings.ReplaceAll(a.text.S.LibTurnAllOffTip, "{n}", fmt.Sprint(len(ids))))
	if off.Clicked() {
		a.skAskRemoveAll = false
		a.skEverySkillOff = true
		c.Invalidate()
	}
}

// ---- the "by agent" rows ------------------------------------------------

// byAgentRow is one agent in the "Skills by agent" sheet: how many skills it
// has of the library's, and how many it could take.
type byAgentRow struct {
	Agent   LibAgentRow
	Has     int
	Missing int
}

// byAgentRows is the agents that can take skills, with their counts.
func (a *App) byAgentRows() []byAgentRow {
	total := a.data.Library.Skills
	out := make([]byAgentRow, 0, len(a.data.Library.Agents))
	for _, ag := range a.data.Library.Agents {
		if ag.Skills == "" {
			continue
		}
		r := byAgentRow{Agent: ag}
		for _, s := range total {
			if skillGives(s, ag.ID) {
				r.Has++
			} else {
				r.Missing++
			}
		}
		out = append(out, r)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return strings.ToLower(out[i].Agent.Name) < strings.ToLower(out[j].Agent.Name)
	})
	return out
}
