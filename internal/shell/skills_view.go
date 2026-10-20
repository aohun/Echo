//go:build nogui && echo

package shell

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/yetone/magpie/internal/library"
)

// The Skills page: the library's skills as rows, grouped by where they came
// from, with the install card over them, the skills found in the agents
// under them, and the market below.
//
// It is magpie's own Skills page (app.js's renderSkills and library.js's
// skills half) drawn in native UI: the same groups in the same order, the
// same rows with the same chips, and the same writes behind the clicks. The
// page holds no state of its own beyond what the reader picked — the filter,
// the folds, the picks, the probe — which is what magpie's page keeps in its
// own module too.
//
// The data is read by readLibraryData into a.data.Library, and the writes are
// in skills_data.go. This file is only the drawing.

// skillsView is the page.
func (a *App) skillsView(c *ui.Context) {
	t := c.Theme()

	// The market is asked once, the first time the page is drawn, as
	// magpie's own asks it when its tab shows. A frame the tester builds
	// has no window and no network: it draws what it has.
	if a.win != nil {
		a.skMarketKick()
	}

	// A skill picked that the library no longer has is dropped, so the
	// bar's count is the library's own.
	for n := range a.skPicked {
		if !a.hasSkill(n) {
			delete(a.skPicked, n)
		}
	}
	if len(a.data.Library.Skills) < 2 {
		a.skPicking = false
	}

	ui.Column(c).Fill().Padding(8, 24, 0, 24).Gap(14).Children(func() {
		a.skillsIntro(c, t)
		a.skillsInstallCard(c, t)
		a.skillsProblems(c, t)
		if len(a.data.Library.Skills) > 0 {
			a.skillsList(c, t)
		} else {
			a.skillsNone(c, t)
		}
		if len(a.data.Library.FoundSkills) > 0 {
			a.skillsFound(c, t)
		}
		if len(a.data.Library.NewSkills) > 0 {
			a.skillsNew(c, t)
		}
		if len(a.data.Library.Skills) > 0 || len(a.data.Library.Projects) > 0 {
			a.skillsProjects(c, t)
		}
		a.skillsAside(c, t)
		a.skillsMarket(c, t)
	})
}

// hasSkill is whether the library has a skill by that name.
func (a *App) hasSkill(name string) bool {
	for _, s := range a.data.Library.Skills {
		if s.Name == name {
			return true
		}
	}
	return false
}

// skillsIntro is the line saying what a skill is, in magpie's own words.
func (a *App) skillsIntro(c *ui.Context, t *ui.Theme) {
	ui.Text(c, a.text.S.LibSkillsIntro).FontSize(12).TextColor(t.TextMuted).Wrap()
}

// ---- the install card ---------------------------------------------------

// skillsInstallCard is the source field and the button that looks in it: a
// repository, a GitHub link, or a folder on this computer. What it found is
// drawn under it, with a box on each skill and the agents to give them to.
func (a *App) skillsInstallCard(c *ui.Context, t *ui.Theme) {
	card := ui.Column(c).FillWidth().Gap(8).Padding(10, 12).Radius(10).Border(1, t.Border)
	if t.Dark {
		card.Background(ui.RGBA(255, 255, 255, 0.02))
	} else {
		card.Background(ui.RGBA(0, 0, 0, 0.015))
	}

	card.Children(func() {
		ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
			// the finder is magpie's own: a card-coloured box with the
			// magnifier inside it, and the input filling the rest
			a.skFindField(c, &a.libProbeSource, a.text.S.LibSkillsFindPlaceholder, false)

			looking := a.skInstalling
			label := a.text.S.LibFindSkills
			if looking {
				label = a.text.S.LibLooking
			}
			b := ui.PrimaryButton(c, label).Shrink(0).Disabled(looking)
			if b.Clicked() {
				a.skProbeSource(a.libProbeSource)
			}
		})

		if a.skInstallErr != "" {
			ui.Text(c, a.skInstallErr).FontSize(11.5).TextColor(t.Danger).Wrap()
		}
		if a.skProbe != nil {
			a.skillsProbeResult(c, t)
		}
	})
}

// skillsProbeResult is what the look found: the count, the picks, the filter
// when there are many, the agents, and the button that installs them.
func (a *App) skillsProbeResult(c *ui.Context, t *ui.Theme) {
	p := a.skProbe
	cands := p.Candidates

	box := ui.Column(c).FillWidth().Gap(8).Padding(10, 0, 2, 0)
	box.Children(func() {
		if len(cands) == 0 {
			ui.Text(c, a.text.S.LibNoSkillsThere).FontSize(12).TextColor(t.TextMuted).Wrap()
			return
		}

		// the source, said the way magpie says it: github.com/owner/repo
		// for one from GitHub, the path with ~ for one here
		src := p.Source
		if p.Kind == "github" {
			src = "github.com/" + repoOf(src)
		} else {
			src = shortPath(src)
		}
		head := a.text.S.LibNSkillsIn
		if len(cands) == 1 {
			head = a.text.S.LibOneSkillIn
		}
		head = strings.ReplaceAll(head, "{n}", fmt.Sprint(len(cands)))
		head = strings.ReplaceAll(head, "{src}", src)

		fresh := make([]library.Candidate, 0, len(cands))
		for _, x := range cands {
			if !x.Have {
				fresh = append(fresh, x)
			}
		}

		ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
			ui.Text(c, head).FontSize(12).TextColor(t.TextMuted).Grow(1).Shrink(1).SingleLine()
			if len(fresh) > 1 {
				allOn := true
				for _, x := range fresh {
					if !a.skProbePicks[x.Path] {
						allOn = false
						break
					}
				}
				label := a.text.S.LibSelectAll
				if allOn {
					label = a.text.S.LibSelectNone
				}
				if b := ui.Button(c, label).Shrink(0); b.Clicked() {
					for _, x := range fresh {
						if allOn {
							delete(a.skProbePicks, x.Path)
						} else {
							a.skProbePicks[x.Path] = true
						}
					}
					c.Invalidate()
				}
			}
		})

		if len(cands) > 8 {
			ui.TextInput(c, &a.skProbeFilter).
				FillWidth().
				Placeholder(a.text.S.LibFilter).
				Label(a.text.S.LibFilter)
		}

		// the candidates, a box each; one the library has says so and
		// can't be picked
		q := strings.ToLower(strings.TrimSpace(a.skProbeFilter))
		ui.Column(c).FillWidth().Gap(1).Children(func() {
			for _, x := range cands {
				cd := x
				if q != "" && !strings.Contains(strings.ToLower(cd.Name+" "+cd.Description), q) {
					continue
				}
				row := ui.Row(c).AlignItems(ui.Center).Gap(8).Padding(4, 6).Radius(6)
				if cd.Have {
					row.Opacity(0.55)
				}
				row.Children(func() {
					on := a.skProbePicks[cd.Path]
					cb := ui.Checkbox(c, &on, "").Shrink(0).Disabled(cd.Have)
					if cb.Changed() {
						if on {
							a.skProbePicks[cd.Path] = true
						} else {
							delete(a.skProbePicks, cd.Path)
						}
						c.Invalidate()
					}
					ui.Column(c).Grow(1).Shrink(1).Gap(1).Children(func() {
						ui.Text(c, cd.Name).FontSize(12.5).FontWeight(500).TextColor(t.Text).SingleLine()
						sub := cd.Description
						if sub == "" {
							sub = cd.Path
						}
						if sub != "" {
							ui.Text(c, sub).FontSize(11).TextColor(t.TextMuted).SingleLine()
						}
					})
					if cd.Have {
						a.skTag(c, t, a.text.S.LibInTheLibrary, "")
					}
				})
			}
		})

		// the foot: the agents to give them to, and the install
		ui.Row(c).AlignItems(ui.Center).Gap(8).Padding(6, 0, 0, 0).Children(func() {
			ui.Text(c, a.text.S.LibFor).FontSize(11.5).TextColor(t.TextMuted).Shrink(0)
			a.skAgentChips(c, t, a.skillAgents(), a.skProbeAgents, false, func(id string, on bool) {
				if on {
					a.skProbeAgents[id] = true
				} else {
					delete(a.skProbeAgents, id)
				}
				c.Invalidate()
			}, nil)
			ui.Spacer(c)

			if b := ui.Button(c, a.text.S.Cancel).Shrink(0); b.Clicked() {
				a.skProbe, a.skProbePicks, a.skProbeSrc = nil, nil, ""
				c.Invalidate()
			}

			n := 0
			for _, on := range a.skProbePicks {
				if on {
					n++
				}
			}
			label := a.text.S.LibInstallNSkills
			if n == 1 {
				label = a.text.S.LibInstallOneSkill
			}
			label = strings.ReplaceAll(label, "{n}", fmt.Sprint(n))
			if b := ui.PrimaryButton(c, label).Shrink(0).Disabled(n == 0 || a.skBusy != ""); b.Clicked() {
				a.skInstall()
			}
		})
	})
}

// ---- what couldn't be written -------------------------------------------

// skillsProblems is the card saying what the last change couldn't write.
// magpie draws the same card over its own page: a chip's amber dot holds
// only its own item's, so a file that wouldn't take a write, a project or
// a skill that wouldn't update is said once here, with the button that
// writes everything again and — for an agent that has a skill of its own
// under the library's name — the choice of whose to keep.
func (a *App) skillsProblems(c *ui.Context, t *ui.Theme) {
	list := a.data.Library.Problems
	if len(list) == 0 {
		return
	}

	card := ui.Column(c).FillWidth().Gap(6).Padding(10, 12).Radius(10).
		Border(1, ui.RGBA(214, 161, 58, 0.45))
	if t.Dark {
		card.Background(ui.RGBA(214, 161, 58, 0.08))
	} else {
		card.Background(ui.RGBA(214, 161, 58, 0.06))
	}

	card.Children(func() {
		ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
			dot := ui.Box(c).Width(7).Height(7).Radius(4).Shrink(0)
			dot.Background(ui.Hex("#d6a13a"))
			ui.Text(c, a.text.S.LibProblemsHead).
				FontSize(12).FontWeight(600).TextColor(t.Text).
				Grow(1).Shrink(1).SingleLine()

			b := ui.Button(c, a.text.S.LibProblemsTryAgain).Shrink(0).Disabled(a.skBusy != "")
			b.Tooltip(a.text.S.LibProblemsTryAgainTip)
			if b.Clicked() {
				a.skSync()
			}
		})

		for _, p := range list {
			problem := p
			ui.Row(c).AlignItems(ui.Center).Gap(6).FillWidth().Children(func() {
				// who: an agent's name, or a project's folder
				if who := a.problemAgentName(problem.Agent); who != "" {
					ui.Text(c, who).FontSize(11.5).FontWeight(600).TextColor(t.Text).Shrink(0)
					ui.Text(c, "·").FontSize(11.5).TextColor(t.TextMuted).Shrink(0)
				}

				own, name := problemOwnSkill(problem)
				what := a.problemWhat(problem.What)
				ui.Text(c, what).FontSize(11.5).TextColor(t.Text).Shrink(0)

				errText := problem.Error
				if own {
					errText = strings.NewReplacer(
						"{agent}", a.problemAgentName(problem.Agent),
						"{name}", name,
					).Replace(a.text.S.LibOwnSkillClash)
				}
				ui.Text(c, errText).FontSize(11.5).TextColor(t.TextMuted).
					Grow(1).Shrink(1).SingleLine()

				if own {
					use := ui.Button(c, a.text.S.LibUseLibrarys).Shrink(0).Disabled(a.skBusy != "")
					use.Tooltip(strings.NewReplacer(
						"{agent}", a.problemAgentName(problem.Agent),
						"{name}", name,
					).Replace(a.text.S.LibUseLibrarysTip))
					if use.Clicked() {
						a.skUseLibrarySkill(name, problem.Agent)
					}

					keep := ui.Button(c, strings.ReplaceAll(a.text.S.LibKeepAgents, "{agent}", a.problemAgentName(problem.Agent))).
						Shrink(0).Disabled(a.skBusy != "")
					keep.Tooltip(strings.NewReplacer(
						"{agent}", a.problemAgentName(problem.Agent),
						"{name}", name,
					).Replace(a.text.S.LibKeepAgentsTip))
					if keep.Clicked() {
						a.skKeepOwnSkill(name, problem.Agent)
					}
				}
			})
		}
	})
}

// problemAgentName is who a problem is about: an agent's name, or a
// project's folder said with ~.
func (a *App) problemAgentName(id string) string {
	if id == "" {
		return ""
	}
	if ag, ok := a.libAgentByID(id); ok {
		return ag.Name
	}
	return shortPath(id)
}

// problemWhat is what a problem is about, in the page's words.
func (a *App) problemWhat(what string) string {
	kind, name, _ := strings.Cut(what, ":")
	switch kind {
	case "instructions":
		return a.text.S.LibProblemInstructions
	case "mcp":
		if name == "" {
			return a.text.S.LibProblemMCPs
		}
		return strings.ReplaceAll(a.text.S.LibProblemMCP, "{name}", name)
	case "skill", "project":
		if name == "" {
			return a.text.S.LibProblemProject
		}
		return strings.ReplaceAll(a.text.S.LibProblemSkill, "{name}", name)
	}
	return what
}

// problemOwnSkill is whether a problem is an agent's own skill under the
// library's name, which the card settles with a choice rather than a retry,
// and the skill's name when it is.
func problemOwnSkill(p ProblemRow) (bool, string) {
	if !p.Own {
		return false, ""
	}
	name, ok := strings.CutPrefix(p.What, "skill:")
	if !ok {
		return false, ""
	}
	return true, name
}

// skillsList is the library's own skills: the head over them (the count, the
// filter, the sort, the bulk actions), the groups, the how bar under them,
// and the pick bar when the reader is picking.
func (a *App) skillsList(c *ui.Context, t *ui.Theme) {
	all := a.skillAgents()
	skills := a.data.Library.Skills
	filtering := strings.TrimSpace(a.libSearch) != ""

	// the head: what the section is, and what can be done to all of it
	ui.Column(c).FillWidth().Gap(8).Children(func() {
		ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
			ui.Text(c, a.text.S.LibInTheLibraryLabel).FontSize(12).FontWeight(600).TextColor(t.TextMuted).Shrink(0)

			if len(skills) > 8 {
				ui.SearchField(c, &a.libSearch).
					Width(200).
					Label(a.text.S.LibSearchSkills)
			}
			ui.Spacer(c)

			if len(skills) > 1 {
				labels := []string{a.text.S.LibSortSource, a.text.S.LibSortAZ, a.text.S.LibSortZA}
				if seg := ui.Segmented(c, &a.libSort, labels...); seg.Changed() {
					c.Invalidate()
				}
			}

			if len(skills) > 1 && len(all) > 0 {
				label := a.text.S.LibSelect
				if a.skPicking {
					label = a.text.S.LibDone
				}
				b := ui.Button(c, label).Shrink(0)
				b.Tooltip(a.text.S.LibSelectSkillsTip)
				if b.Clicked() {
					a.skPicking = !a.skPicking
					a.skPicked = map[string]bool{}
					a.skNaming = false
					c.Invalidate()
				}
				a.skEverySkillButtons(c, t, all)
			}

			if len(a.byAgentRows()) > 0 {
				b := ui.Button(c, a.text.S.LibByAgentPlain).Shrink(0)
				b.Tooltip(a.text.S.LibByAgentTip)
				if b.Clicked() {
					a.skByAgent = true
					c.Invalidate()
				}
			}

			if len(skills) > 1 {
				b := ui.Button(c, a.text.S.LibRemoveAll).Shrink(0)
				b.Tooltip(strings.ReplaceAll(a.text.S.LibRemoveAllTip, "{n}", fmt.Sprint(len(skills))))
				if b.Clicked() {
					a.skAskRemoveAll = true
					c.Invalidate()
				}
			}

			// a check for updates, for the ones that came from GitHub
			if a.anyFromGitHub() {
				label := a.text.S.LibCheckUpdates
				if a.skCheckBusy {
					label = a.text.S.LibLooking
				}
				b := ui.Button(c, label).Shrink(0).Disabled(a.skCheckBusy)
				if b.Clicked() {
					a.skCheck()
				}
			}

			// the ones GitHub changed, or every one from it again
			stale := a.staleSkills()
			if len(stale) > 0 {
				n := len(stale)
				label := strings.ReplaceAll(a.text.S.LibUpdateN, "{n}", fmt.Sprint(n))
				tip := strings.ReplaceAll(a.text.S.LibUpdateNTip, "{n}", fmt.Sprint(n))
				if n == 1 {
					tip = strings.ReplaceAll(a.text.S.LibUpdateOneTip, "{name}", stale[0].Name)
				}
				b := ui.PrimaryButton(c, label).Shrink(0).Disabled(a.skBusy != "")
				b.Tooltip(tip)
				if b.Clicked() {
					names := make([]string, 0, n)
					for _, s := range stale {
						names = append(names, s.Name)
					}
					a.skUpdateAll(names)
				}
			} else if n := len(a.freshFromGitHub()); n > 1 {
				b := ui.Button(c, a.text.S.LibUpdateAll).Shrink(0).Disabled(a.skBusy != "")
				b.Tooltip(strings.ReplaceAll(a.text.S.LibFetchNFromGitHub, "{n}", fmt.Sprint(n)))
				if b.Clicked() {
					a.skUpdateAll(nil)
				}
			}
		})

		if a.libNotice != "" {
			ui.Text(c, a.libNotice).FontSize(11.5).TextColor(t.TextMuted).Wrap()
		}

		// the groups
		a.skillsGroups(c, t, all, filtering)

		// under the list, so the skills stay where they were above it
		if len(all) > 0 {
			a.skillsHowBar(c, t)
		}
		if a.skPicking {
			a.skillsPickBar(c, t, all)
		}
	})
}

// skillsGroups draws the groups, each a card with a heading over its rows.
// One group with no heading over it when the sort is a flat one.
func (a *App) skillsGroups(c *ui.Context, t *ui.Theme, all []LibAgentRow, filtering bool) {
	groups := a.skillGroups()
	single := len(groups) == 1

	if len(groups) == 0 {
		a.skillsNone(c, t)
		return
	}
	for _, g := range groups {
		gr := g
		hits := skillHits(gr, skillWords(a.libSearch))
		if len(hits) == 0 {
			continue
		}
		gr.Skills = hits
		if gr.Total == 0 {
			gr.Total = len(hits)
		}
		if single {
			a.skillsGroupBody(c, t, gr, all, filtering, true)
			continue
		}
		a.skillsGroupCard(c, t, gr, all, filtering)
	}
}

// skillsNone says the library holds no skill, and how one comes in.
func (a *App) skillsNone(c *ui.Context, t *ui.Theme) {
	ui.Column(c).FillWidth().Gap(4).Padding(20, 0).Children(func() {
		ui.Text(c, a.text.S.LibNoSkills).FontSize(13).FontWeight(500).TextColor(t.Text).SingleLine()
		ui.Text(c, a.text.S.LibNoSkillsHint).FontSize(11.5).TextColor(t.TextMuted).Wrap()
	})
}

// skillsGroupCard is one group: its heading — the repository, the reader's
// own name, or "On this computer" — and the rows under it.
func (a *App) skillsGroupCard(c *ui.Context, t *ui.Theme, g skillGroup, all []LibAgentRow, filtering bool) {
	folded := a.groupFolded(g.Key, filtering)

	card := ui.Column(c).FillWidth().Gap(0).Radius(10).Border(1, t.Border)
	card.Children(func() {
		a.skillsGroupHead(c, t, g, all, folded, filtering)
		if !folded {
			a.skillsGroupBody(c, t, g, all, filtering, false)
		}
	})
}

// skillsGroupHead is the heading over a group's rows.
func (a *App) skillsGroupHead(c *ui.Context, t *ui.Theme, g skillGroup, all []LibAgentRow, folded, filtering bool) {
	head := ui.Row(c).AlignItems(ui.Center).Gap(8).
		Padding(8, 10).FillWidth().
		Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).
		Label(a.text.S.LibExpandAll)
	if head.Hovered() || head.Focused() {
		head.Background(a.skHoverBg(t))
	}

	if head.Clicked() {
		a.foldGroup(g.Key, folded)
		c.Invalidate()
	}

	head.Children(func() {
		if folded {
			ui.Icon(c, chevRight()).Size(10, 10).TextColor(t.TextMuted).Shrink(0)
		} else {
			ui.Icon(c, chevDown()).Size(10, 10).TextColor(t.TextMuted).Shrink(0)
		}

		// the picture: a skill's own when one has it, else a folder for a
		// group of the library's, and the skill glyph for the reader's own
		pic := ""
		for _, s := range g.Skills {
			if s.Icon != "" {
				pic = s.Icon
				break
			}
		}
		if pic != "" {
			if p := iconPictureFor(pic); p != nil {
				ui.Image(c, p).Size(16, 16).Radius(4).Shrink(0)
			}
		} else {
			ui.Icon(c, folderIcon()).Size(14, 14).TextColor(t.TextMuted).Shrink(0)
		}

		// who: the name, and under it the count
		ui.Column(c).Grow(1).Shrink(1).Gap(1).Children(func() {
			ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
				title := g.Mine
				if title == "" {
					title = g.Repo
				}
				if title == "" {
					title = a.text.S.LibOnThisComputer
				}
				name := ui.Text(c, title).FontSize(13).FontWeight(600).TextColor(t.Text).SingleLine()
				if g.Repo != "" {
					name.Font("monospace")
				}
			})

			total := g.Total
			if total == 0 {
				total = len(g.Skills)
			}
			sub := a.text.S.LibOneSkill
			if total != 1 {
				sub = strings.ReplaceAll(a.text.S.LibSkillsCount, "{n}", fmt.Sprint(total))
			}
			// a filter says how many of the group it shows, as magpie's
			// own heading does
			if filtering && len(g.Skills) != total {
				sub = strings.NewReplacer(
					"{n}", fmt.Sprint(len(g.Skills)),
					"{total}", fmt.Sprint(total),
				).Replace(a.text.S.LibNOutOfSkills)
			}
			ui.Text(c, sub).FontSize(11).TextColor(t.TextMuted).SingleLine()
		})

		// a group's own update, so a repository's changed skills are fetched
		// from its heading rather than one row at a time
		if stale := staleIn(g.Skills); len(stale) > 0 {
			n := len(stale)
			label := strings.ReplaceAll(a.text.S.LibUpdateN, "{n}", fmt.Sprint(n))
			b := ui.Button(c, label).Shrink(0).Disabled(a.skBusy != "")
			b.Tooltip(strings.ReplaceAll(a.text.S.LibUpdateNTip, "{n}", fmt.Sprint(n)))
			if b.Clicked() {
				names := make([]string, 0, n)
				for _, s := range stale {
					names = append(names, s.Name)
				}
				a.skUpdateAll(names)
			}
		}

		// the agents that have them, and the writes on a group of the
		// reader's own
		ui.Spacer(c)
		a.skGroupChips(c, t, g.Skills, all, a.skGroupTitle(g))

		// a repository's group opens on GitHub, as magpie's heading does
		if g.Repo != "" && !g.Owner {
			url := repoURL(g.Repo)
			b := a.skIconButton(c, t, globeIcon(), strings.ReplaceAll(a.text.S.LibOpenOnGitHub, "{repo}", g.Repo), false)
			if b.Clicked() {
				a.openBrowser(url)
			}
		}

		if g.Mine != "" {
			if a.skRename == g.Mine {
				ui.Row(c).AlignItems(ui.Center).Gap(4).Shrink(0).Children(func() {
					ui.TextInput(c, &a.skRenameTo).Width(140).Label(a.text.S.LibRename)
					to := strings.TrimSpace(a.skRenameTo)
					if b := ui.PrimaryButton(c, a.text.S.LibDone).Shrink(0).Disabled(to == "" || to == g.Mine); b.Clicked() {
						a.skRenameGroup(g.Mine, to)
						a.skRename, a.skRenameTo = "", ""
						c.Invalidate()
					}
					if b := ui.Button(c, a.text.S.Cancel).Shrink(0); b.Clicked() {
						a.skRename, a.skRenameTo = "", ""
						c.Invalidate()
					}
				})
			} else {
				if b := ui.Button(c, a.text.S.LibRename).Shrink(0); b.Clicked() {
					a.skRename, a.skRenameTo = g.Mine, g.Mine
					c.Invalidate()
				}
				b := ui.Button(c, a.text.S.LibUngroup).Shrink(0)
				b.Tooltip(a.text.S.LibUngroupTip)
				if b.Clicked() {
					a.skUngroup(g.Mine, nil)
				}
			}
		}
	})
}

// skillsGroupBody is a group's rows: its parts when it is big enough to be
// split, else its skills one under another.
func (a *App) skillsGroupBody(c *ui.Context, t *ui.Theme, g skillGroup, all []LibAgentRow, filtering, bare bool) {
	box := ui.Column(c).FillWidth().Gap(0)
	box.Children(func() {
		parts := a.groupParts(g)
		if len(parts) == 0 {
			for _, s := range g.Skills {
				a.skillRow(c, t, s, all)
			}
			return
		}
		for _, p := range parts {
			part := p
			folded := a.groupFolded(part.Key, filtering)
			a.skillPartHead(c, t, g, part, all, folded, filtering)
			if !folded {
				for _, s := range part.Skills {
					a.skillRow(c, t, s, all)
				}
			}
		}
	})
}

// skillPartHead is a part's heading inside a big group: its folder or the
// start its skills share, the count, and the agents that have them.
func (a *App) skillPartHead(c *ui.Context, t *ui.Theme, g skillGroup, p skillPart, all []LibAgentRow, folded, filtering bool) {
	head := ui.Row(c).AlignItems(ui.Center).Gap(8).
		Padding(6, 10, 6, 26).FillWidth().
		Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).
		Label(a.text.S.LibExpandAll)
	if head.Hovered() || head.Focused() {
		head.Background(a.skHoverBg(t))
	}
	if head.Clicked() {
		a.foldGroup(p.Key, folded)
		c.Invalidate()
	}

	head.Children(func() {
		if folded {
			ui.Icon(c, chevRight()).Size(9, 9).TextColor(t.TextMuted).Shrink(0)
		} else {
			ui.Icon(c, chevDown()).Size(9, 9).TextColor(t.TextMuted).Shrink(0)
		}

		name := ui.Text(c, p.Label).FontSize(12).FontWeight(500).TextColor(t.Text).SingleLine()
		if p.Mono {
			name.Font("monospace")
		}

		ui.Text(c, fmt.Sprint(len(p.Skills))).FontSize(11).TextColor(t.TextMuted).Shrink(0)

		if n := len(staleIn(p.Skills)); n > 0 {
			a.skTag(c, t, strings.ReplaceAll(a.text.S.LibUpdateN, "{n}", fmt.Sprint(n)), "update")
		}
		ui.Spacer(c)
		if len(all) > 0 {
			a.skGroupChips(c, t, p.Skills, all, "")
		}
	})
}

// ---- one skill's row ----------------------------------------------------

// skillRow is one skill: its picture, its name with the marks a check left
// on it, its description, where it came from, the buttons that update and
// remove it, and a chip for each agent that can take it.
func (a *App) skillRow(c *ui.Context, t *ui.Theme, s SkillRow, all []LibAgentRow) {
	row := ui.Row(c).Key("sk/"+s.Name).
		AlignItems(ui.Center).Gap(9).
		MinHeight(46).
		Padding(6, 10, 6, 10).
		FillWidth().
		Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).
		Label(strings.ReplaceAll(a.text.S.LibReadSkill, "{name}", s.Name))

	if row.Hovered() || row.Focused() {
		row.Background(a.skHoverBg(t))
	}

	// picking: a click on the row is a click on its box
	if a.skPicking {
		if row.Clicked() {
			if a.skPicked[s.Name] {
				delete(a.skPicked, s.Name)
			} else {
				a.skPicked[s.Name] = true
			}
			c.Invalidate()
		}
		row.Label(strings.ReplaceAll(a.text.S.LibSelect, "{name}", s.Name))
	} else if row.Clicked() {
		a.skOpenReader(s.Name)
	}

	row.Children(func() {
		if a.skPicking {
			on := a.skPicked[s.Name]
			ui.Checkbox(c, &on, "").Shrink(0)
		}

		// the picture: the skill's own, else the skill glyph
		if s.Icon != "" {
			if p := iconPictureFor(s.Icon); p != nil {
				ui.Image(c, p).Size(18, 18).Radius(4).Shrink(0)
			}
		} else {
			ui.Icon(c, skillIcon()).Size(15, 15).TextColor(t.TextMuted).Shrink(0)
		}

		// who: the name with its marks, the description, where it is from
		ui.Column(c).Grow(1).Shrink(1).Gap(2).Children(func() {
			ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
				name := ui.Text(c, s.Name).FontSize(12.5).FontWeight(500).TextColor(t.Text).SingleLine()
				if s.Missing {
					name.TextColor(t.TextMuted).Strikethrough()
				}
				switch s.CheckStatus {
				case "update":
					// the mark is the link to what changed on GitHub
					a.skLinkTag(c, t, a.text.S.LibUpdateAvailable, "update", s.CheckURL, a.text.S.LibChangedSinceInstall)
				case "unknown":
					a.skTag(c, t, a.text.S.LibNotChecked, "muted")
				}
				if len(s.Behind) > 0 {
					// the mark copies the skills again, which is what
					// settles a copy that drifted from the library's
					b := a.skLinkTag(c, t, a.text.S.LibCopyOutOfDate, "update", "", "")
					b.Tooltip(strings.ReplaceAll(a.text.S.LibCopyOutOfDateTip, "{agents}", a.agentNames(s.Behind)))
					if b.Clicked() {
						a.skSync()
					}
				}
			})

			sub := s.Description
			if s.Missing {
				sub = a.text.S.LibItsFolderIsGone
			}
			if sub != "" {
				txt := ui.Text(c, sub).FontSize(11).TextColor(t.TextMuted).SingleLine()
				if s.Missing {
					txt.TextColor(t.Warning)
				}
			}

			// where it came from
			if s.Source != "" {
				ui.Row(c).AlignItems(ui.Center).Gap(4).Children(func() {
					if s.Kind == "github" {
						ui.Text(c, repoOf(s.Source)).Font("monospace").FontSize(10.5).TextColor(t.TextMuted).SingleLine()
					} else {
						ui.Text(c, a.text.S.LibLinkedFrom).FontSize(10.5).TextColor(t.TextMuted).Shrink(0)
						ui.Text(c, shortPath(s.Source)).Font("monospace").FontSize(10.5).TextColor(t.TextMuted).SingleLine()
					}
				})
			}
		})

		// the writes on this row: update, and remove
		ui.Row(c).AlignItems(ui.Center).Gap(4).Shrink(0).Children(func() {
			if s.Kind == "github" || s.Origin != "" {
				busy := a.skBusy == "skills/update:"+s.Name
				b := a.skIconButton(c, t, updateIcon(), a.text.S.LibUpdateSkill, busy)
				if b.Clicked() {
					a.skUpdate(s.Name, false)
				}
			}
			if a.skAskRemove == s.Name {
				if b := ui.Button(c, a.text.S.LibDeleteSkill).Shrink(0); b.Clicked() {
					a.skAskRemove = ""
					a.skRemove(s.Name)
					c.Invalidate()
				}
				if b := ui.Button(c, a.text.S.Cancel).Shrink(0); b.Clicked() {
					a.skAskRemove = ""
					c.Invalidate()
				}
			} else {
				b := a.skIconButton(c, t, trashIcon(), a.text.S.LibDeleteSkill, false)
				if b.Clicked() {
					a.skAskRemove = s.Name
					c.Invalidate()
				}
			}
		})

		// the agents that have it, a chip each, and an All chip ahead of them
		a.skAgentChips(c, t, all, skillAgentSet(s), true, func(id string, on bool) {
			a.skToggleAgent(s, id)
		}, func(on bool) {
			ids := make([]string, 0, len(all))
			for _, ag := range all {
				ids = append(ids, ag.ID)
			}
			if !on {
				ids = nil
			}
			a.skSetAgents(s.Name, ids)
		})
	})
}

// ---- the how bar --------------------------------------------------------

// skillsHowBar is the way the library gives its skills: links, or copies,
// and the button that lets one agent have its own way.
func (a *App) skillsHowBar(c *ui.Context, t *ui.Theme) {
	how := 0 // links
	if a.data.Library.CopySkills {
		how = 1 // copies
	}

	bar := ui.Row(c).AlignItems(ui.Center).Gap(10).Padding(8, 2, 2, 2).FillWidth()
	bar.Children(func() {
		ui.Text(c, a.text.S.LibGiveSkillsAs).FontSize(11.5).TextColor(t.TextMuted).Shrink(0)

		sel := how
		labels := []string{a.text.S.LibLinks, a.text.S.LibCopies}
		if seg := ui.Segmented(c, &sel, labels...); seg.Changed() {
			want := library.HowLink
			if sel == 1 {
				want = library.HowCopy
			}
			a.skSetHow("", want)
		}

		own := 0
		for _, ag := range a.skillAgents() {
			if ag.HowOwn() {
				own++
			}
		}
		label := a.text.S.LibByAgentPlain
		if own > 0 {
			label = strings.ReplaceAll(a.text.S.LibByAgent, "{n}", fmt.Sprint(own))
		}
		b := ui.Button(c, label).Shrink(0)
		if b.Clicked() {
			a.skAskHow = true
			c.Invalidate()
		}

		note := a.text.S.LibLinksNote
		if a.data.Library.CopySkills {
			note = a.text.S.LibCopiesNote
		}
		ui.Text(c, note).FontSize(10.5).TextColor(t.TextMuted).Grow(1).Shrink(1).Wrap()
	})
}

// ---- the pick bar -------------------------------------------------------

// skillsPickBar is the bar under the list while the reader picks: a box for
// every skill the filter shows, the count picked, the agents to turn them on
// or off for, and the group they can be put in.
func (a *App) skillsPickBar(c *ui.Context, t *ui.Theme, all []LibAgentRow) {
	shown := a.shownSkills()
	picked := a.pickedSkills()

	bar := ui.Column(c).FillWidth().Gap(8).Padding(10, 12).Radius(10).Border(1, t.Border)
	bar.Children(func() {
		ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
			every := len(shown) > 0
			for _, s := range shown {
				if !a.skPicked[s.Name] {
					every = false
					break
				}
			}
			on := every
			if ui.Checkbox(c, &on, "").Changed() {
				for _, s := range shown {
					if on {
						a.skPicked[s.Name] = true
					} else {
						delete(a.skPicked, s.Name)
					}
				}
				c.Invalidate()
			}

			label := a.text.S.LibAllN
			if strings.TrimSpace(a.libSearch) != "" {
				label = a.text.S.LibAllNThatMatch
			}
			label = strings.ReplaceAll(label, "{n}", fmt.Sprint(len(shown)))
			ui.Text(c, label).FontSize(11.5).TextColor(t.TextMuted).Shrink(0)

			ui.Spacer(c)

			if n := len(picked); n > 0 {
				ui.Text(c, strings.ReplaceAll(a.text.S.LibNSelected, "{n}", fmt.Sprint(n))).FontSize(11.5).FontWeight(500).TextColor(t.Text).Shrink(0)
			} else {
				ui.Text(c, a.text.S.LibPickToToggle).FontSize(11.5).TextColor(t.TextMuted).SingleLine()
			}
		})

		if len(picked) == 0 {
			ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
				ui.Spacer(c)
				if b := ui.Button(c, a.text.S.LibDone).Shrink(0); b.Clicked() {
					a.skPicking, a.skPicked, a.skNaming = false, map[string]bool{}, false
					c.Invalidate()
				}
			})
			return
		}

		// the group's name, when the reader asked to make one
		if a.skNaming {
			a.skillsNameBar(c, t, picked)
			return
		}

		ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
			a.skGroupChips(c, t, picked, all, "")
			ui.Spacer(c)

			if a.libSort == 0 {
				b := ui.Button(c, a.text.S.LibGroupEllipsis).Shrink(0)
				b.Tooltip(a.text.S.LibGroupTip)
				if b.Clicked() {
					a.skNaming, a.skName = true, ""
					c.Invalidate()
				}
				if own := a.pickedGroups(picked); len(own) > 0 {
					b := ui.Button(c, a.text.S.LibUngroup).Shrink(0)
					if b.Clicked() {
						for _, name := range own {
							names := make([]string, 0, len(picked))
							for _, s := range picked {
								names = append(names, s.Name)
							}
							a.skUngroup(name, names)
						}
					}
				}
			}

			if b := ui.Button(c, a.text.S.LibClear).Shrink(0); b.Clicked() {
				a.skPicked = map[string]bool{}
				c.Invalidate()
			}
			if b := ui.PrimaryButton(c, a.text.S.LibDone).Shrink(0); b.Clicked() {
				a.skPicking, a.skPicked, a.skNaming = false, map[string]bool{}, false
				c.Invalidate()
			}
		})
	})
}

// skillsNameBar is the bar asking the name of the group the picked skills go
// in: a new one, or one there is, which they join.
func (a *App) skillsNameBar(c *ui.Context, t *ui.Theme, picked []SkillRow) {
	have := make([]string, 0, len(a.data.Library.SkillGroups))
	for _, g := range a.data.Library.SkillGroups {
		have = append(have, g.Name)
	}

	ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
		label := a.text.S.LibGroupNSkillsAs
		if len(picked) == 1 {
			label = a.text.S.LibGroupOneSkillAs
		}
		label = strings.ReplaceAll(label, "{n}", fmt.Sprint(len(picked)))
		ui.Text(c, label).FontSize(11.5).TextColor(t.TextMuted).Shrink(0)

		ui.TextInput(c, &a.skName).Width(160).Placeholder(a.text.S.LibGroupName).Label(a.text.S.LibGroupName)

		// the groups there are, to join with a click
		for _, h := range have {
			if len(have) > 4 {
				break
			}
			name := h
			b := ui.Button(c, name).Shrink(0)
			b.Tooltip(strings.ReplaceAll(a.text.S.LibAddToName, "{name}", name))
			if b.Clicked() {
				a.skMakeGroup(name, picked)
				c.Invalidate()
			}
		}

		ui.Spacer(c)

		if b := ui.Button(c, a.text.S.Cancel).Shrink(0); b.Clicked() {
			a.skNaming = false
			c.Invalidate()
		}

		label = a.text.S.LibGroup
		name := strings.TrimSpace(a.skName)
		for _, h := range have {
			if h == name {
				label = strings.ReplaceAll(a.text.S.LibAddToName, "{name}", name)
				break
			}
		}
		if b := ui.PrimaryButton(c, label).Shrink(0).Disabled(name == ""); b.Clicked() {
			a.skMakeGroup(name, picked)
			c.Invalidate()
		}
	})
}

// ---- found in the agents ------------------------------------------------

// skillsFound is the skills the agents have that the library doesn't: each
// with the button that brings it in, and one for all of them.
func (a *App) skillsFound(c *ui.Context, t *ui.Theme) {
	found := a.data.Library.FoundSkills

	ui.Column(c).FillWidth().Gap(6).Children(func() {
		ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
			ui.Text(c, a.text.S.LibInAgents).FontSize(12).FontWeight(600).TextColor(t.TextMuted).Shrink(0)
			ui.Spacer(c)
			ui.Text(c, a.text.S.LibInAgentsDesc).FontSize(10.5).TextColor(t.TextMuted).SingleLine()

			if len(found) > 1 {
				names := make([]string, 0, len(found))
				for _, f := range found {
					names = append(names, f.Name)
				}
				label := a.text.S.LibBringInAll
				if a.skBusy == "skills/import-all" {
					label = a.text.S.LibBringingIn
				}
				if b := ui.Button(c, label).Shrink(0).Disabled(a.skBusy != ""); b.Clicked() {
					a.skBringInAll(names)
				}
			}
		})

		card := ui.Column(c).FillWidth().Gap(0).Radius(10).Border(1, t.Border)
		card.Children(func() {
			for _, f := range found {
				fs := f
				row := ui.Row(c).AlignItems(ui.Center).Gap(9).
					MinHeight(44).Padding(6, 10).FillWidth()

				row.Children(func() {
					ui.Icon(c, skillIcon()).Size(15, 15).TextColor(t.TextMuted).Shrink(0)

					ui.Column(c).Grow(1).Shrink(1).Gap(2).Children(func() {
						ui.Text(c, fs.Name).FontSize(12.5).FontWeight(500).TextColor(t.Text).SingleLine()
						if fs.Description != "" {
							ui.Text(c, fs.Description).FontSize(11).TextColor(t.TextMuted).SingleLine()
						}
						switch {
						case fs.Library != "":
							a.skPathLine(c, t, a.text.S.LibInLibraryFolder, fs.Library)
						case fs.Shared != "":
							a.skPathLine(c, t, a.text.S.LibSharedIn, fs.Shared)
						case fs.Link != "":
							a.skPathLine(c, t, a.text.S.LibLinkedFrom, fs.Link)
						}
					})

					// the agents that have it
					ids := append(append([]string{}, fs.Agents...), fs.Copies...)
					ui.Row(c).AlignItems(ui.Center).Gap(3).Shrink(0).Children(func() {
						for _, id := range ids {
							if ag, ok := a.libAgentByID(id); ok {
								if p := iconPictureFor(ag.Icon); p != nil {
									ui.Image(c, p).Size(16, 16).Radius(4)
								}
							}
						}
					})

					if len(fs.Others) > 0 {
						names := make([]string, 0, len(fs.Others))
						for _, id := range fs.Others {
							if ag, ok := a.libAgentByID(id); ok {
								names = append(names, ag.Name)
							}
						}
						a.skTag(c, t, strings.ReplaceAll(a.text.S.LibDiffersIn, "{agents}", strings.Join(names, ", ")), "warn")
					}

					if b := ui.Button(c, a.text.S.LibBringIn).Shrink(0).Disabled(a.skBusy != ""); b.Clicked() {
						a.skBringIn(fs.Name)
					}
				})
			}
		})
	})
}

// ---- in projects --------------------------------------------------------

// skillsProjects is the section under the list: the folders whose agents get
// some of the library's skills as the project's own. magpie draws the same
// section on its own page, a card per project that opens to a row for every
// skill in the library with a chip for each agent that reads a project's
// skills folder.
func (a *App) skillsProjects(c *ui.Context, t *ui.Theme) {
	projects := a.data.Library.Projects

	ui.Column(c).FillWidth().Gap(6).Children(func() {
		ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
			ui.Text(c, a.text.S.LibInProjects).FontSize(12).FontWeight(600).TextColor(t.TextMuted).Shrink(0)
			ui.Spacer(c)

			if !a.skAddingProject {
				b := ui.Button(c, a.text.S.LibAddProject).Shrink(0)
				if b.Clicked() {
					a.skAddingProject = true
					a.skProjectDir = ""
					c.Invalidate()
				}
			}
		})

		if a.skAddingProject {
			a.skillsAddProject(c, t)
		}

		if len(projects) == 0 {
			ui.Text(c, a.text.S.LibProjectsEmpty).FontSize(11).TextColor(t.TextMuted).Wrap()
			return
		}

		for _, p := range projects {
			project := p
			a.skillsProjectCard(c, t, project)
		}
	})
}

// skillsAddProject is the card that takes a folder: typed, or picked with
// the system's own folder picker.
func (a *App) skillsAddProject(c *ui.Context, t *ui.Theme) {
	card := ui.Column(c).FillWidth().Gap(8).Padding(10, 12).Radius(10).Border(1, t.Border)
	if t.Dark {
		card.Background(ui.RGBA(255, 255, 255, 0.02))
	} else {
		card.Background(ui.RGBA(0, 0, 0, 0.015))
	}

	card.Children(func() {
		ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
			ui.Icon(c, folderIcon()).Size(14, 14).TextColor(t.TextMuted).Shrink(0)
			ui.TextInput(c, &a.skProjectDir).
				Grow(1).Shrink(1).
				Placeholder(a.text.S.LibProjectFolder).
				Label(a.text.S.LibProjectFolder)

			// the app has the system's folder picker
			if b := ui.Button(c, a.text.S.LibChooseFolder).Shrink(0); b.Clicked() {
				a.skChooseProject()
			}

			dir := strings.TrimSpace(a.skProjectDir)
			if b := ui.PrimaryButton(c, a.text.S.LibAdd).Shrink(0).
				Disabled(dir == "" || a.skBusy != ""); b.Clicked() {
				a.skAddProject(dir)
				a.skAddingProject = false
				c.Invalidate()
			}
			if b := ui.Button(c, a.text.S.Cancel).Shrink(0); b.Clicked() {
				a.skAddingProject = false
				c.Invalidate()
			}
		})

		if a.skProjectErr != "" {
			ui.Text(c, a.skProjectErr).FontSize(11.5).TextColor(t.Danger).Wrap()
		}
	})
}

// skillsProjectCard is one project: its name and folder, the tag saying it
// gets copies or that its folder is gone, and — open — the skills to place
// in it.
func (a *App) skillsProjectCard(c *ui.Context, t *ui.Theme, p ProjectRow) {
	open := a.skOpenProjects[p.Dir]

	card := ui.Column(c).FillWidth().Gap(0).Radius(10).Border(1, t.Border)
	card.Children(func() {
		head := ui.Row(c).AlignItems(ui.Center).Gap(9).
			Padding(8, 10).FillWidth().
			Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).
			Label(a.text.S.LibPickTheSkillsItGets)
		if head.Hovered() || head.Focused() {
			head.Background(a.skHoverBg(t))
		}
		if head.Clicked() {
			a.skOpenProjects[p.Dir] = !open
			c.Invalidate()
		}

		head.Children(func() {
			if open {
				ui.Icon(c, chevDown()).Size(10, 10).TextColor(t.TextMuted).Shrink(0)
			} else {
				ui.Icon(c, chevRight()).Size(10, 10).TextColor(t.TextMuted).Shrink(0)
			}
			ui.Icon(c, folderIcon()).Size(14, 14).TextColor(t.TextMuted).Shrink(0)

			ui.Column(c).Grow(1).Shrink(1).Gap(1).Children(func() {
				ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
					ui.Text(c, p.Name).FontSize(12.5).FontWeight(600).TextColor(t.Text).SingleLine()
					if p.Missing {
						a.skTag(c, t, a.text.S.LibFolderIsGone, "warn")
					} else if p.Copy {
						a.skTag(c, t, a.text.S.LibCopies, "")
					}
				})
				ui.Text(c, shortPath(p.Dir)).Font("monospace").FontSize(10.5).
					TextColor(t.TextMuted).SingleLine()
			})

			// the skills it has, a pill each, while it is closed
			if !open {
				names := make([]string, 0, len(p.Skills))
				for n := range p.Skills {
					if len(p.Skills[n]) > 0 {
						names = append(names, n)
					}
				}
				sort.Strings(names)
				ui.Row(c).AlignItems(ui.Center).Gap(3).Shrink(0).Children(func() {
					shown := names
					if len(shown) > 4 {
						shown = shown[:4]
					}
					for _, n := range shown {
						a.skTag(c, t, n, "")
					}
					if len(names) > 4 {
						a.skTag(c, t, fmt.Sprintf("+%d", len(names)-4), "")
					}
					if len(names) == 0 {
						a.skTag(c, t, a.text.S.LibNoSkillsYet, "muted")
					}
				})
			}

			ui.Spacer(c)
			if b := a.skIconButton(c, t, trashIcon(), a.text.S.LibRemoveProject, false); b.Clicked() {
				a.skAskRemoveProject = p.Dir
				c.Invalidate()
			}
		})

		if p.Problems[""] != "" {
			ui.Text(c, p.Problems[""]).FontSize(11).TextColor(t.Danger).
				Padding(0, 12, 8, 12).Wrap()
		}

		if !open {
			return
		}

		a.skillsProjectBody(c, t, p)
	})
}

// skillsProjectBody is an open project's body: the way it gets its skills,
// a row for each of the library's, and the folders it is placed in.
func (a *App) skillsProjectBody(c *ui.Context, t *ui.Theme, p ProjectRow) {
	all := a.skProjectAgents()

	ui.Column(c).FillWidth().Gap(0).Padding(0, 0, 8, 0).Children(func() {
		// copies, or links
		ui.Row(c).AlignItems(ui.Center).Gap(8).Padding(4, 12, 8, 12).Children(func() {
			on := p.Copy
			if cb := ui.Checkbox(c, &on, a.text.S.LibCopyNotLink).Shrink(0); cb.Changed() {
				a.skProjectCopy(p.Dir, on)
			}
			note := a.text.S.LibLinksNoteProject
			if p.Copy {
				note = a.text.S.LibCopiesNote
			}
			ui.Text(c, note).FontSize(10.5).TextColor(t.TextMuted).Grow(1).Shrink(1).Wrap()
		})

		// a row for every skill in the library
		for _, s := range a.data.Library.Skills {
			skill := s
			problem := p.Problems[skill.Name]
			on := p.Skills[skill.Name]

			row := ui.Row(c).AlignItems(ui.Center).Gap(9).
				MinHeight(46).Padding(6, 12).FillWidth()

			row.Children(func() {
				if skill.Icon != "" {
					if pic := iconPictureFor(skill.Icon); pic != nil {
						ui.Image(c, pic).Size(16, 16).Radius(4).Shrink(0)
					}
				} else {
					ui.Icon(c, skillIcon()).Size(14, 14).TextColor(t.TextMuted).Shrink(0)
				}

				ui.Column(c).Grow(1).Shrink(1).Gap(2).Children(func() {
					ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
						ui.Text(c, skill.Name).FontSize(12.5).FontWeight(500).TextColor(t.Text).SingleLine()
						if problem != "" {
							a.skTag(c, t, a.text.S.LibNotPlaced, "warn")
						}
					})
					sub := skill.Description
					if problem != "" {
						sub = problem
					}
					if sub != "" {
						ui.Text(c, sub).FontSize(11).TextColor(t.TextMuted).SingleLine()
					}
				})

				a.skProjectChips(c, t, p, skill, on, all)
			})
		}

		// where it is placed, and who reads no such folder
		dirs := a.skProjectDirs()
		if len(dirs) > 0 {
			ui.Text(c, strings.ReplaceAll(a.text.S.LibPlacedIn, "{dirs}", strings.Join(dirs, ", "))).
				FontSize(10.5).TextColor(t.TextMuted).Padding(4, 12, 0, 12).Wrap()
		}
		if skip := a.skProjectAgentsThatReadNone(); len(skip) > 0 {
			ui.Text(c, strings.ReplaceAll(a.text.S.LibReadsNoProjectFolder, "{agents}", strings.Join(skip, ", "))).
				FontSize(10.5).TextColor(t.TextMuted).Padding(2, 12, 0, 12).Wrap()
		}
	})
}

// skProjectChips is a project row's chips: one for each agent that reads a
// project's skills folder, lit when the project has the skill for it. An
// agent reading a folder another already has is lit by that one's click.
func (a *App) skProjectChips(c *ui.Context, t *ui.Theme, p ProjectRow, s SkillRow, on []string, all []LibAgentRow) {
	ui.Row(c).AlignItems(ui.Center).Gap(2).Shrink(0).Children(func() {
		for _, ag := range all {
			agent := ag
			has := containsStr(on, agent.ID)
			via := ""
			if !has {
				via = skProjectVia(on, agent, all)
			}

			chip := ui.Row(c).Center().Width(22).Height(22).Radius(6).Shrink(0).
				Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).
				Label(agent.Name)

			switch {
			case has:
				chip.Tooltip(agent.Name)
				if t.Dark {
					chip.Background(ui.RGBA(255, 255, 255, 0.10))
				} else {
					chip.Background(ui.RGBA(0, 0, 0, 0.055))
				}
				chip.Border(1, t.Border)
			case via != "":
				chip.Tooltip(via)
				if t.Dark {
					chip.Background(ui.RGBA(255, 255, 255, 0.055))
				} else {
					chip.Background(ui.RGBA(0, 0, 0, 0.028))
				}
				chip.Border(1, t.Border)
			default:
				chip.Tooltip(agent.Name)
				if chip.Hovered() || chip.Focused() {
					chip.Background(a.skHoverBg(t))
				}
			}

			if chip.Clicked() {
				a.skProjectToggle(p, s, agent.ID)
			}

			chip.Children(func() {
				op := float32(0.35)
				if has || via != "" {
					op = 1
				}
				if pic := iconPictureFor(agent.Icon); pic != nil {
					ui.Image(c, pic).Size(14, 14).Radius(3).Opacity(op)
				} else {
					ui.Icon(c, genericIcon()).Size(12, 12).TextColor(t.TextMuted)
				}
			})
		}
	})
}

// ---- new in the repositories -------------------------------------------

// skillsNew is the skills a check found its repositories to have added
// beside the ones the library has, to add or set aside.
func (a *App) skillsNew(c *ui.Context, t *ui.Theme) {
	items := a.data.Library.NewSkills

	ui.Column(c).FillWidth().Gap(6).Children(func() {
		ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
			ui.Text(c, a.text.S.LibAlsoInRepos).FontSize(12).FontWeight(600).TextColor(t.TextMuted).Shrink(0)
			ui.Spacer(c)
			ui.Text(c, a.text.S.LibAlsoInReposDesc).FontSize(10.5).TextColor(t.TextMuted).SingleLine()

			ids := make([]string, 0, len(items))
			for _, n := range items {
				ids = append(ids, n.ID)
			}
			if b := ui.Button(c, a.text.S.LibIgnoreAll).Shrink(0).Disabled(a.skBusy != ""); b.Clicked() {
				a.skIgnoreNew(ids)
			}
			label := a.text.S.LibAddAll
			if a.skBusy == "skills/add-new" {
				label = a.text.S.LibAdding
			}
			if b := ui.PrimaryButton(c, label).Shrink(0).Disabled(a.skBusy != ""); b.Clicked() {
				a.skAddNew(ids)
			}
		})

		card := ui.Column(c).FillWidth().Gap(0).Radius(10).Border(1, t.Border)
		card.Children(func() {
			for _, n := range items {
				ns := n
				ui.Row(c).AlignItems(ui.Center).Gap(9).MinHeight(42).Padding(6, 10).FillWidth().Children(func() {
					ui.Icon(c, skillIcon()).Size(14, 14).TextColor(t.TextMuted).Shrink(0)
					ui.Column(c).Grow(1).Shrink(1).Gap(1).Children(func() {
						ui.Text(c, ns.Name).FontSize(12.5).FontWeight(500).TextColor(t.Text).SingleLine()
						sub := ns.Description
						if sub == "" {
							sub = ns.Repo
						}
						ui.Text(c, sub).FontSize(11).TextColor(t.TextMuted).SingleLine()
					})
					ui.Text(c, ns.Repo).Font("monospace").FontSize(10.5).TextColor(t.TextMuted).Shrink(0)
				})
			}
		})
	})
}

// ---- the asides --------------------------------------------------------

// skillsAside says which agents have no skills folder, and how Claude
// Desktop reads its list, when either is the case.
func (a *App) skillsAside(c *ui.Context, t *ui.Theme) {
	var skip []string
	for _, ag := range a.data.Library.Agents {
		if ag.Gone || ag.Skills != "" {
			continue
		}
		skip = append(skip, ag.Name)
	}
	if len(skip) > 0 {
		sort.Strings(skip)
		txt := strings.ReplaceAll(a.text.S.LibNoAgentsHaveSkills, "{agents}", strings.Join(skip, ", "))
		ui.Text(c, txt).FontSize(11).TextColor(t.TextMuted).Wrap()
	}

	for _, ag := range a.skillAgents() {
		if ag.ID != "claude-desktop" {
			continue
		}
		keys := "Ctrl+R"
		ui.Text(c, strings.ReplaceAll(a.text.S.LibDesktopReload, "{keys}", keys)).
			FontSize(11).TextColor(t.TextMuted).Wrap()
		break
	}
}

// ---- small pieces -------------------------------------------------------

// skHoverBg is the wash a row or a heading takes under the pointer.
func (a *App) skHoverBg(t *ui.Theme) ui.Color {
	if t.Dark {
		return ui.RGBA(255, 255, 255, 0.04)
	}
	return ui.RGBA(0, 0, 0, 0.035)
}

// skCardBg is the fill of the page's cards, of its fields, and of a chip
// stacked in a row: magpie's own --card, white on light and #232327 on dark.
// The theme's Control surface is a grey the page's cards don't use, so a
// field would read as a slab rather than sitting on the card.
func skCardBg(t *ui.Theme) ui.Color {
	if t.Dark {
		return ui.Hex("#232327")
	}
	return ui.Hex("#ffffff")
}

// skTag is a small mark beside a name: "Update available", "Not checked",
// "Official".
func (a *App) skTag(c *ui.Context, t *ui.Theme, label, kind string) {
	a.skTagIn(c, t, label, kind, false)
}

// skLinkTag is a mark that can be clicked: magpie's "Update available" is a
// link to what changed on GitHub, and its "Copy out of date" copies the
// skills again.
func (a *App) skLinkTag(c *ui.Context, t *ui.Theme, label, kind, url, tip string) ui.Element {
	e := a.skTagIn(c, t, label, kind, true)
	if tip != "" {
		e.Tooltip(tip)
	}
	if e.Clicked() && url != "" {
		a.openBrowser(url)
	}
	return e
}

func (a *App) skTagIn(c *ui.Context, t *ui.Theme, label, kind string, link bool) ui.Element {
	bg := ui.RGBA(0, 0, 0, 0.05)
	fg := t.TextMuted
	switch kind {
	case "update":
		bg = ui.RGBA(52, 199, 89, 0.14)
		fg = ui.Hex("#1c8c3f")
		if t.Dark {
			fg = ui.Hex("#4ade80")
		}
	case "warn":
		bg = ui.RGBA(214, 161, 58, 0.16)
		fg = ui.Hex("#a06a12")
		if t.Dark {
			fg = ui.Hex("#e0b155")
		}
	case "accent":
		bg = ui.RGBA(0, 122, 255, 0.12)
		fg = t.Accent
	}
	e := ui.Text(c, label).
		FontSize(9.5).FontWeight(600).
		TextColor(fg).Background(bg).
		Padding(1, 5).Radius(4).
		Shrink(0).SingleLine()
	if link {
		e.Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).Label(label)
		if e.Hovered() || e.Focused() {
			e.Underline()
		}
	}
	return e
}

// skIconButton is one of a row's small buttons: an icon, its name for a
// reader who can't see it, and a wash under the pointer.
func (a *App) skIconButton(c *ui.Context, t *ui.Theme, icon *ui.SVG, label string, busy bool) ui.Element {
	b := ui.Row(c).Center().Width(24).Height(24).Radius(5).Shrink(0).
		Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).
		Label(label)
	if b.Hovered() || b.Focused() {
		b.Background(a.skHoverBg(t))
	}
	if busy {
		b.Opacity(0.45)
	}
	b.Children(func() {
		if busy {
			ui.Spinner(c).Size(11, 11)
			return
		}
		ui.Icon(c, icon).Size(12, 12).TextColor(t.TextMuted)
	})
	return b
}

// skPathLine is a "linked from <path>" line under a found skill.
func (a *App) skPathLine(c *ui.Context, t *ui.Theme, what, path string) {
	ui.Row(c).AlignItems(ui.Center).Gap(4).Children(func() {
		ui.Text(c, what).FontSize(10.5).TextColor(t.TextMuted).Shrink(0)
		ui.Text(c, shortPath(path)).Font("monospace").FontSize(10.5).TextColor(t.TextMuted).SingleLine()
	})
}

// ---- the writes the page's own controls call ---------------------------

// foldGroup folds or opens a group, and keeps it as the reader left it.
func (a *App) foldGroup(key string, folded bool) {
	if a.libFoldedGroups == nil {
		a.libFoldedGroups = map[string]bool{}
	}
	a.libFoldedGroups[key] = !folded
}

// shownSkills is every skill the filter shows, in every group, folded or
// not: what the pick bar's "All" counts.
func (a *App) shownSkills() []SkillRow {
	words := skillWords(a.libSearch)
	var out []SkillRow
	for _, g := range a.skillGroups() {
		out = append(out, skillHits(g, words)...)
	}
	return out
}

// pickedSkills are the skills picked, as the library has them now.
func (a *App) pickedSkills() []SkillRow {
	out := make([]SkillRow, 0, len(a.skPicked))
	for _, s := range a.data.Library.Skills {
		if a.skPicked[s.Name] {
			out = append(out, s)
		}
	}
	return out
}

// pickedGroups are the reader's own groups that hold any of the picked.
func (a *App) pickedGroups(picked []SkillRow) []string {
	var out []string
	for _, g := range a.data.Library.SkillGroups {
		for _, s := range picked {
			if containsStr(g.Skills, s.Name) {
				out = append(out, g.Name)
				break
			}
		}
	}
	return out
}

// skMakeGroup puts the picked skills in a group of the reader's, made when
// there is none by that name.
func (a *App) skMakeGroup(name string, picked []SkillRow) {
	names := make([]string, 0, len(picked))
	for _, s := range picked {
		names = append(names, s.Name)
	}
	a.skWrite("skills/group", func() (*library.Result, error) {
		return library.GroupSkills("", name, names)
	})
	if a.libFoldedGroups == nil {
		a.libFoldedGroups = map[string]bool{}
	}
	a.libFoldedGroups["my:"+name] = false // a group just made is shown open
	a.skPicking, a.skNaming = false, false
	a.skPicked = map[string]bool{}
}

// skUngroup lets a group go, or only the skills named out of it.
func (a *App) skUngroup(name string, names []string) {
	a.skWrite("skills/ungroup", func() (*library.Result, error) {
		return library.UngroupSkills(name, names)
	})
}

// skRenameGroup renames a group of the reader's.
func (a *App) skRenameGroup(old, name string) {
	a.skWrite("skills/group", func() (*library.Result, error) {
		return library.GroupSkills(old, name, nil)
	})
}

// skEverySkill turns every skill on or off for every agent that can take
// them, or asks first when it is turning them all off.
func (a *App) skEverySkill(ids []string, on bool) {
	a.skWrite("skills/agents-all", func() (*library.Result, error) {
		return library.EverySkillAgents(ids, on)
	})
}

// skSomeSkills turns the named skills on or off for the named agents, which
// is what a group's chips and the pick bar's do.
func (a *App) skSomeSkills(names, agents []string, on bool) {
	a.skWrite("skills/agents-some", func() (*library.Result, error) {
		return library.SomeSkillsAgents(names, agents, on)
	})
}

// skRemoveAll takes every skill named out of the library at once.
func (a *App) skRemoveAll(names []string) {
	a.skWrite("skills/remove-all", func() (*library.Result, error) {
		return library.RemoveSkills(names)
	})
}

// ---- the helpers the data reads ---------------------------------------

// anyFromGitHub is whether any skill came from GitHub, which is what the
// check for updates is for.
func (a *App) anyFromGitHub() bool {
	for _, s := range a.data.Library.Skills {
		if s.Kind == "github" || s.Origin != "" {
			return true
		}
	}
	return false
}

// staleSkills are the skills GitHub changed since they were installed.
func (a *App) staleSkills() []SkillRow {
	var out []SkillRow
	for _, s := range a.data.Library.Skills {
		if s.CheckStatus == "update" {
			out = append(out, s)
		}
	}
	return out
}

// staleIn is the same for one group's skills.
func staleIn(skills []SkillRow) []SkillRow {
	var out []SkillRow
	for _, s := range skills {
		if s.CheckStatus == "update" {
			out = append(out, s)
		}
	}
	return out
}

// freshFromGitHub are the skills that came from GitHub, which "Update all"
// fetches again.
func (a *App) freshFromGitHub() []SkillRow {
	var out []SkillRow
	for _, s := range a.data.Library.Skills {
		if s.Kind == "github" || s.Origin != "" {
			out = append(out, s)
		}
	}
	return out
}

// skGroupTitle is a group's name as a message about it says it.
func (a *App) skGroupTitle(g skillGroup) string {
	if g.Mine != "" {
		return g.Mine
	}
	if g.Repo != "" {
		return g.Repo
	}
	return ""
}

// skillAgentSet is the agents a skill is given to, as a set the chips read.
func skillAgentSet(s SkillRow) map[string]bool {
	out := make(map[string]bool, len(s.Agents))
	for _, id := range s.Agents {
		out[id] = true
	}
	return out
}

// containsStr is whether a list holds a string.
func containsStr(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

// shortPath cuts a path under the home directory down to ~, as magpie's
// tilde does, so a long folder doesn't run the line off the page.
func shortPath(p string) string {
	if p == "" {
		return ""
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return p
	}
	if strings.HasPrefix(p, home+string(os.PathSeparator)) {
		return "~" + strings.TrimPrefix(p, home)
	}
	return p
}

// agentNames is a list of agent ids as their names, for a tooltip that says
// which agents' copies differ from the library's.
func (a *App) agentNames(ids []string) string {
	names := make([]string, 0, len(ids))
	for _, id := range ids {
		if ag, ok := a.libAgentByID(id); ok {
			names = append(names, ag.Name)
		}
	}
	return strings.Join(names, ", ")
}
