//go:build nogui && echo

package shell

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/yetone/magpie/internal/library"
)

// The sheets the Skills page opens over it: the reader for one skill's
// SKILL.md, the ask before a skill or every skill is removed, the sheet that
// sets the way one agent is given its skills, the sheet that turns every
// skill on or off for one agent, and the market's own sheet.
//
// magpie's page opens them in the library's modal, one at a time, and so
// does Echo: skillsOverlay draws whichever is open, over the page and under
// nothing else.

// skillsOverlay draws whichever sheet the page has open. It is called from
// the view, after the page, so the sheet is over everything.
func (a *App) skillsOverlay(c *ui.Context) {
	switch {
	case a.skReader != "":
		a.skillsReaderSheet(c)
	case a.skAskRemoveAll:
		a.skillsRemoveAllSheet(c)
	case a.skEverySkillOff:
		a.skillsEveryOffSheet(c)
	case a.skAskReplace != "":
		a.skillsReplaceSheet(c)
	case a.skAskRemoveProject != "":
		a.skillsRemoveProjectSheet(c)
	case a.skAskHow:
		a.skillsHowSheet(c)
	case a.skByAgent:
		a.skillsByAgentSheet(c)
	case a.skMarketSheet != "":
		a.skillsMarketSheet(c)
	}
}

// skCloseSheet puts whichever sheet is open away, and nothing of it
// survives: the reader's text goes with it, and the ask it was.
func (a *App) skCloseSheet() {
	a.skReader, a.skReaderText, a.skReaderErr, a.skReaderPath = "", "", "", ""
	a.skAskRemoveAll, a.skEverySkillOff = false, false
	a.skAskReplace, a.skAskHow, a.skByAgent = "", false, false
	a.skMarketSheet = ""
	a.skAskRemoveProject = ""
	a.skRemoveKeep = false
	a.invalidate()
}

// skSheetFrame draws a sheet's panel and gives it the body to fill. It is
// the same frame the add sheet uses: as large as the window lets it be, and
// no larger, so the foot lands on the screen whatever window it opens in.
// Escape puts the sheet away, as it does magpie's own.
func (a *App) skSheetFrame(c *ui.Context, width, height float32, title string, body func()) {
	a.skSheetFrameClose(c, width, height, title, a.skCloseSheet, body)
}

// skSheetFrameClose draws a sheet's panel and gives it the body to fill. It closes
// when clicking outside on the backdrop or pressing Escape.
func (a *App) skSheetFrameClose(c *ui.Context, width, height float32, title string, closeFn func(), body func()) {
	t := c.Theme()
	ui.Overlay(c, func() {
		back := ui.Box(c).Absolute().Left(0).Top(0).Right(0).Bottom(0).Center().
			Background(ui.RGBA(0, 0, 0, 0.4)).Modal()
		backDismissed := back.Clicked() || back.OverlayShortcut(0, ui.KeyEscape)
		back.Children(func() {
			cw, ch := c.Size()
			panel := ui.Column(c).Width(width).Height(height).
				MaxWidth(cw - 80).MaxHeight(ch - 80).
				Radius(12).Role(ui.RoleDialog).Label(title)
			panel.Clicked() // absorbs click inside dialog
			if t.Dark {
				panel.Background(ui.RGBA(40, 40, 46, 1))
			} else {
				panel.Background(t.Background)
			}
			panel.Shadow(0, 10, 30, 0, ui.RGBA(0, 0, 0, 0.3))
			panel.Children(body)
		})
		if backDismissed {
			if closeFn != nil {
				closeFn()
			} else {
				a.skCloseSheet()
			}
			c.Invalidate()
		}
	})
}

// skSheetHead is a sheet's head: its picture and title, and the way out.
func (a *App) skSheetHead(c *ui.Context, t *ui.Theme, icon *ui.SVG, title, sub string) {
	ui.Row(c).AlignItems(ui.Center).Gap(9).Padding(14, 16, 10, 16).FillWidth().Children(func() {
		if icon != nil {
			ui.Icon(c, icon).Size(15, 15).TextColor(t.TextMuted).Shrink(0)
		}
		ui.Column(c).Grow(1).Shrink(1).Gap(2).Children(func() {
			ui.Text(c, title).FontSize(14).FontWeight(600).TextColor(t.Text).SingleLine()
			if sub != "" {
				ui.Text(c, sub).FontSize(11).TextColor(t.TextMuted).SingleLine()
			}
		})
	})
}

// skSheetRule is the hairline under a sheet's head.
func (a *App) skSheetRule(c *ui.Context, t *ui.Theme) {
	line := ui.Row(c).Height(1).FillWidth().Padding(0, 16)
	if t.Dark {
		line.Background(ui.RGBA(255, 255, 255, 0.08))
	} else {
		line.Background(ui.RGBA(0, 0, 0, 0.08))
	}
}

// skSheetFoot is a sheet's foot: the way out on the right, and whatever the
// sheet does beside it.
func (a *App) skSheetFoot(c *ui.Context, t *ui.Theme, body func()) {
	ui.Row(c).AlignItems(ui.Center).Gap(8).Padding(10, 16, 14, 16).FillWidth().Children(func() {
		ui.Spacer(c)
		if body != nil {
			body()
		}
	})
}

// ---- one skill's SKILL.md ----------------------------------------------

// skillsReaderSheet shows a skill's SKILL.md, read from its folder, with
// where that folder is.
func (a *App) skillsReaderSheet(c *ui.Context) {
	name := a.skReader
	a.skSheetFrame(c, 720, 560, name, func() {
		t := c.Theme()
		ui.Column(c).FillWidth().FillHeight().Children(func() {
			a.skSheetHead(c, t, skillIcon(), name, "")
			a.skSheetRule(c, t)

			// the text: its own scroll, so a long SKILL.md reads without
			// pushing the foot off the panel
			ui.Scroll(c).Grow(1).Shrink(1).Padding(12, 16).Children(func() {
				switch {
				case a.skReaderErr != "":
					ui.Text(c, a.skReaderErr).FontSize(12).TextColor(t.Danger).Wrap()
				case a.skReaderText == "":
					ui.Text(c, a.text.S.Loading).FontSize(12).TextColor(t.TextMuted)
				default:
					ui.Text(c, a.skReaderText).
						Font("monospace").FontSize(11.5).
						TextColor(t.Text).Wrap().Selectable()
				}
			})

			a.skSheetRule(c, t)
			a.skSheetFoot(c, t, func() {
				if a.skReaderPath != "" {
					ui.Text(c, shortPath(a.skReaderPath)).
						Font("monospace").FontSize(10.5).
						TextColor(t.TextMuted).SingleLine()
					ui.Spacer(c)
				}
				if b := ui.Button(c, a.text.S.LibClose).Shrink(0); b.Clicked() {
					a.skReader, a.skReaderText, a.skReaderErr, a.skReaderPath = "", "", "", ""
					c.Invalidate()
				}
			})
		})
	})
}

// ---- removing ----------------------------------------------------------

// skillsRemoveAllSheet asks before every skill goes, saying who loses what.
func (a *App) skillsRemoveAllSheet(c *ui.Context) {
	skills := a.data.Library.Skills
	n := len(skills)

	// the agents that have any of them, and the ones linked from a folder
	// of the reader's, which are only unlinked
	agentIDs := map[string]bool{}
	linked := 0
	for _, s := range skills {
		if s.Kind == "folder" {
			linked++
		}
		for _, id := range s.Agents {
			agentIDs[id] = true
		}
	}
	names := make([]string, 0, len(agentIDs))
	for _, ag := range a.data.Library.Agents {
		if agentIDs[ag.ID] {
			names = append(names, ag.Name)
		}
	}

	a.skSheetFrame(c, 520, 0, a.text.S.LibRemoveAll, func() {
		t := c.Theme()
		ui.Column(c).FillWidth().Children(func() {
			a.skSheetHead(c, t, trashIcon(), strings.ReplaceAll(a.text.S.LibRemoveAllAsk, "{n}", fmt.Sprint(n)), "")
			ui.Column(c).Padding(2, 16, 6, 16).Gap(8).Children(func() {
				if len(names) > 0 {
					ui.Text(c, strings.ReplaceAll(a.text.S.LibRemoveAllOutOf, "{agents}", strings.Join(names, ", "))).
						FontSize(12).TextColor(t.Text).Wrap()
				} else {
					ui.Text(c, a.text.S.LibRemoveAllNoAgent).FontSize(12).TextColor(t.Text).Wrap()
				}
				if linked > 0 {
					ui.Text(c, strings.ReplaceAll(a.text.S.LibRemoveSkillFolder, "{n}", fmt.Sprint(linked))).
						FontSize(11.5).TextColor(t.TextMuted).Wrap()
				}
			})
			a.skSheetFoot(c, t, func() {
				if b := ui.Button(c, a.text.S.Cancel).Shrink(0); b.Clicked() {
					a.skAskRemoveAll = false
					c.Invalidate()
				}
				if b := ui.PrimaryButton(c, strings.ReplaceAll(a.text.S.LibRemoveAllBtn, "{n}", fmt.Sprint(n))).
					Shrink(0).Disabled(a.skBusy != ""); b.Clicked() {
					all := make([]string, 0, n)
					for _, s := range skills {
						all = append(all, s.Name)
					}
					a.skRemoveAll(all)
					a.skAskRemoveAll = false
					c.Invalidate()
				}
			})
		})
	})
}

// skillsEveryOffSheet asks before every skill is taken from every agent.
func (a *App) skillsEveryOffSheet(c *ui.Context) {
	all := a.skillAgents()
	ids := make([]string, 0, len(all))
	names := make([]string, 0, len(all))
	for _, ag := range all {
		ids = append(ids, ag.ID)
		names = append(names, ag.Name)
	}
	n := len(a.data.Library.Skills)

	a.skSheetFrame(c, 520, 0, a.text.S.LibTurnAllOff, func() {
		t := c.Theme()
		ui.Column(c).FillWidth().Children(func() {
			a.skSheetHead(c, t, skillIcon(), strings.ReplaceAll(a.text.S.LibEverySkillOff, "{n}", fmt.Sprint(n)), "")
			ui.Column(c).Padding(2, 16, 6, 16).Children(func() {
				ui.Text(c, strings.ReplaceAll(a.text.S.LibByAgentNoOthers, "{agents}", strings.Join(names, ", "))).
					FontSize(12).TextColor(t.Text).Wrap()
			})
			a.skSheetFoot(c, t, func() {
				if b := ui.Button(c, a.text.S.Cancel).Shrink(0); b.Clicked() {
					a.skEverySkillOff = false
					c.Invalidate()
				}
				if b := ui.PrimaryButton(c, a.text.S.LibTurnAllOff).Shrink(0).Disabled(a.skBusy != ""); b.Clicked() {
					a.skEverySkill(ids, false)
					a.skEverySkillOff = false
					c.Invalidate()
				}
			})
		})
	})
}

// skillsReplaceSheet asks whether GitHub's version goes in place of a
// skill the reader changed here, as magpie's own ask does: the changed one
// is kept with the backups, so nothing is lost either way.
func (a *App) skillsReplaceSheet(c *ui.Context) {
	name := a.skAskReplace

	a.skSheetFrame(c, 520, 0, name, func() {
		t := c.Theme()
		ui.Column(c).FillWidth().Children(func() {
			a.skSheetHead(c, t, updateIcon(),
				strings.ReplaceAll(a.text.S.LibReplaceChanges, "{name}", name), "")
			ui.Column(c).Padding(2, 16, 6, 16).Children(func() {
				ui.Text(c, strings.ReplaceAll(a.text.S.LibReplaceChangesDesc, "{name}", name)).
					FontSize(12).TextColor(t.TextMuted).Wrap()
			})
			a.skSheetFoot(c, t, func() {
				if b := ui.Button(c, a.text.S.LibKeepMine).Shrink(0); b.Clicked() {
					a.skAskReplace = ""
					c.Invalidate()
				}
				if b := ui.PrimaryButton(c, a.text.S.LibUpdateSkill).Shrink(0).Disabled(a.skBusy != ""); b.Clicked() {
					a.skUpdate(name, true)
					a.skAskReplace = ""
					c.Invalidate()
				}
			})
		})
	})
}

// ---- how each agent gets its skills ------------------------------------

// skillsHowSheet is the way each agent is given its skills: the library's
// way for every one, or an agent's own.
func (a *App) skillsHowSheet(c *ui.Context) {
	a.skSheetFrame(c, 560, 480, a.text.S.LibHowEachAgent, func() {
		t := c.Theme()
		ui.Column(c).FillWidth().FillHeight().Children(func() {
			a.skSheetHead(c, t, skillIcon(), a.text.S.LibHowEachAgent, "")
			ui.Column(c).Padding(2, 16, 8, 16).Children(func() {
				note := a.text.S.LibLinksNote
				if a.data.Library.CopySkills {
					note = a.text.S.LibCopiesNote
				}
				ui.Text(c, note).FontSize(11.5).TextColor(t.TextMuted).Wrap()
			})

			ui.Scroll(c).Grow(1).Shrink(1).Children(func() {
				for _, ag := range a.data.Library.Agents {
					if ag.Skills == "" || ag.Gone {
						continue
					}
					agent := ag
					ui.Row(c).AlignItems(ui.Center).Gap(9).MinHeight(48).
						Padding(6, 16).FillWidth().Children(func() {
						if p := iconPictureFor(agent.Icon); p != nil {
							ui.Image(c, p).Size(20, 20).Radius(4).Shrink(0)
						}
						ui.Column(c).Grow(1).Shrink(1).Gap(1).Children(func() {
							ui.Text(c, agent.Name).FontSize(12.5).FontWeight(500).TextColor(t.Text).SingleLine()
							sub := a.text.S.LibGetsLinks
							if agent.MustCopy {
								sub = a.text.S.LibOnlyCopies
							} else if agent.How == library.HowCopy {
								sub = a.text.S.LibGetsCopies
							}
							ui.Text(c, sub).FontSize(11).TextColor(t.TextMuted).SingleLine()
						})

						// 0 the library's way, 1 links, 2 copies
						sel := 0
						if agent.HowOwn() {
							if agent.How == library.HowCopy {
								sel = 2
							} else {
								sel = 1
							}
						}
						if agent.MustCopy {
							return
						}
						labels := []string{a.text.S.LibLibrarysWay, a.text.S.LibLinks, a.text.S.LibCopies}
						if seg := ui.Segmented(c, &sel, labels...).Shrink(0); seg.Changed() {
							want := ""
							switch sel {
							case 1:
								want = library.HowLink
							case 2:
								want = library.HowCopy
							}
							a.skSetHow(agent.ID, want)
						}
					})
				}
			})

			a.skSheetRule(c, t)
			a.skSheetFoot(c, t, func() {
				if b := ui.PrimaryButton(c, a.text.S.LibDone).Shrink(0); b.Clicked() {
					a.skAskHow = false
					c.Invalidate()
				}
			})
		})
	})
}

// ---- every skill for one agent -----------------------------------------

// skillsByAgentSheet turns every skill on or off for one agent, the others
// keeping theirs.
func (a *App) skillsByAgentSheet(c *ui.Context) {
	rows := a.byAgentRows()
	total := len(a.data.Library.Skills)

	a.skSheetFrame(c, 560, 480, a.text.S.LibSkillsByAgent, func() {
		t := c.Theme()
		ui.Column(c).FillWidth().FillHeight().Children(func() {
			a.skSheetHead(c, t, skillIcon(), a.text.S.LibSkillsByAgent, "")
			ui.Column(c).Padding(2, 16, 8, 16).Children(func() {
				ui.Text(c, a.text.S.LibByAgentNoOthers).FontSize(11.5).TextColor(t.TextMuted).Wrap()
			})

			ui.Scroll(c).Grow(1).Shrink(1).Children(func() {
				for _, r := range rows {
					row := r
					ui.Row(c).AlignItems(ui.Center).Gap(9).MinHeight(48).
						Padding(6, 16).FillWidth().Children(func() {
						if p := iconPictureFor(row.Agent.Icon); p != nil {
							ui.Image(c, p).Size(20, 20).Radius(4).Shrink(0)
						}
						ui.Column(c).Grow(1).Shrink(1).Gap(1).Children(func() {
							ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
								ui.Text(c, row.Agent.Name).FontSize(12.5).FontWeight(500).TextColor(t.Text).SingleLine()
								if row.Agent.Gone {
									tag := a.skTag
									_ = tag
								}
							})
							sub := strings.NewReplacer("{n}", fmt.Sprint(row.Has), "{m}", fmt.Sprint(total)).
								Replace(a.text.S.LibNOfMSkills)
							ui.Text(c, sub).FontSize(11).TextColor(t.TextMuted).SingleLine()
						})

						on := ui.Button(c, a.text.S.LibTurnAllOn).Shrink(0).
							Disabled(row.Missing == 0 || a.skBusy != "")
						on.Tooltip(strings.ReplaceAll(a.text.S.LibGiveAgentEverySkill, "{agent}", row.Agent.Name))
						if on.Clicked() {
							a.skEverySkill([]string{row.Agent.ID}, true)
						}

						off := ui.Button(c, a.text.S.LibTurnAllOff).Shrink(0).
							Disabled(row.Has == 0 || a.skBusy != "")
						off.Tooltip(strings.ReplaceAll(a.text.S.LibTakeAgentEverySkill, "{agent}", row.Agent.Name))
						if off.Clicked() {
							a.skEverySkill([]string{row.Agent.ID}, false)
						}
					})
				}
			})

			a.skSheetRule(c, t)
			a.skSheetFoot(c, t, func() {
				if b := ui.PrimaryButton(c, a.text.S.LibDone).Shrink(0); b.Clicked() {
					a.skByAgent = false
					c.Invalidate()
				}
			})
		})
	})
}

// ---- the market's own sheet --------------------------------------------

// skillsMarketSheet is one market skill up close: what it is, where it comes
// from, and the agents to give it to.
func (a *App) skillsMarketSheet(c *ui.Context) {
	var x *library.MarketSkill
	for i := range a.skMarketItems {
		if a.skMarketItems[i].ID == a.skMarketSheet {
			x = &a.skMarketItems[i]
			break
		}
	}
	if x == nil {
		a.skMarketSheet = ""
		return
	}
	item := *x

	a.skSheetFrame(c, 560, 0, item.Name, func() {
		t := c.Theme()
		ui.Column(c).FillWidth().Children(func() {
			a.skSheetHead(c, t, nil, item.Name, item.Source)
			ui.Column(c).Padding(2, 16, 8, 16).Gap(8).Children(func() {
				desc := item.Description
				if desc == "" {
					desc = a.text.S.LibNoDescription
				}
				ui.Text(c, desc).FontSize(12).TextColor(t.Text).Wrap()
				ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
					ui.Text(c, a.text.S.LibFrom).FontSize(11).TextColor(t.TextMuted).Shrink(0)
					ui.Text(c, "github.com/"+item.Source).Font("monospace").FontSize(11).TextColor(t.TextMuted).SingleLine()
				})
				if item.Have != "" {
					a.skTag(c, t, strings.ReplaceAll(a.text.S.LibInLibraryAsName, "{name}", item.Have), "muted")
				}
			})
			a.skSheetFoot(c, t, func() {
				if b := ui.Button(c, a.text.S.LibClose).Shrink(0); b.Clicked() {
					a.skMarketSheet = ""
					c.Invalidate()
				}
				if item.Have == "" {
					if b := ui.PrimaryButton(c, a.text.S.LibAddToLibrary).Shrink(0).Disabled(a.skBusy != ""); b.Clicked() {
						a.skInstallMarket(item.Source, item.SkillID)
						a.skMarketSheet = ""
						c.Invalidate()
					}
				}
			})
		})
	})
}

// ---- removing a project ------------------------------------------------

// skillsRemoveProjectSheet asks before a project is forgotten, and says what
// happens to what was placed in it: magpie asks the same way, since skills
// given to a project once are often to stay there.
func (a *App) skillsRemoveProjectSheet(c *ui.Context) {
	dir := a.skAskRemoveProject
	var project *ProjectRow
	for i := range a.data.Library.Projects {
		if a.data.Library.Projects[i].Dir == dir {
			project = &a.data.Library.Projects[i]
			break
		}
	}
	if project == nil {
		a.skAskRemoveProject = ""
		return
	}
	p := *project

	// what magpie put there, which is what a keep keeps
	placed := len(p.Placed) > 0
	for _, agents := range p.Skills {
		if len(agents) > 0 {
			placed = true
			break
		}
	}

	a.skSheetFrame(c, 520, 0, a.text.S.LibRemoveProject, func() {
		t := c.Theme()
		ui.Column(c).FillWidth().Children(func() {
			a.skSheetHead(c, t, trashIcon(),
				strings.ReplaceAll(a.text.S.LibRemoveProjectAsk, "{name}", p.Name), "")

			ui.Column(c).Padding(2, 16, 6, 16).Gap(8).Children(func() {
				desc := a.text.S.LibRemoveProjectDesc
				if placed {
					on := a.skRemoveKeep
					if cb := ui.Checkbox(c, &on, a.text.S.LibRemoveProjectKeep); cb.Changed() {
						a.skRemoveKeep = on
						c.Invalidate()
					}
					desc = a.text.S.LibRemoveProjectForget
					if a.skRemoveKeep {
						desc = a.text.S.LibRemoveProjectKeepDesc
					}
				}
				ui.Text(c, desc).FontSize(12).TextColor(t.TextMuted).Wrap()
			})

			a.skSheetFoot(c, t, func() {
				if b := ui.Button(c, a.text.S.Cancel).Shrink(0); b.Clicked() {
					a.skAskRemoveProject, a.skRemoveKeep = "", false
					c.Invalidate()
				}
				if b := ui.PrimaryButton(c, a.text.S.LibRemoveProject).
					Shrink(0).Disabled(a.skBusy != ""); b.Clicked() {
					a.skRemoveProject(p.Dir, a.skRemoveKeep)
					a.skAskRemoveProject, a.skRemoveKeep = "", false
					c.Invalidate()
				}
			})
		})
	})
}
