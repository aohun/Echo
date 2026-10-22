//go:build nogui && echo

package shell

import (
	"cmp"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/yetone/magpie/internal/agent"
	"github.com/yetone/magpie/internal/profile"
	"github.com/yetone/magpie/internal/provider"
	"github.com/yetone/magpie/internal/settings"
)

// AgentRow is one coding agent as the Agents page lists it.
type AgentRow struct {
	ID          string
	Name        string
	Icon        string
	Found       bool
	Path        string
	Plugin      string
	Wired       bool
	Drift       bool
	DriftText   string
	Version     string
	Subtitle    string
	ModelCount  int
	DetailsText string
	FieldKey    string
	CurrentVal  string
	CurrentName string
	CurrentIcon string
	Effort      string
	ModelList   bool // the agent's own menu is the list of models
	ModelNames  []string
	Options     []AgentModelOpt
	Groups      []AgentGroupChip
	Notice      string
}

// AgentModelOpt is one option for picking a model in an agent row.
type AgentModelOpt struct {
	ID       string
	Label    string
	Provider string
	Icon     string
	Selected bool
}

// AgentGroupChip is a chip in an expanded agent row representing a provider/group.
type AgentGroupChip struct {
	Name  string
	Icon  string
	Icons []string
	Count int
}

// AgentInstallRow is an uninstalled agent with its install command.
type AgentInstallRow struct {
	ID      string
	Name    string
	Icon    string
	Via     string
	Command string
}

// ProfileRow is a saved configuration scheme (方案).
type ProfileRow struct {
	Name    string
	Summary string
	InUse   bool
}

var (
	agCLICache sync.Map // agent id -> agent.CLI

	refreshOnce sync.Once
	refreshPic  *ui.SVG

	reapplyOnce sync.Once
	reapplyPic  *ui.SVG

	dragOnce sync.Once
	dragPic  *ui.SVG
)

func refreshIcon() *ui.SVG {
	refreshOnce.Do(func() {
		refreshPic = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" width="13" height="13"><path d="M13.5 8a5.5 5.5 0 1 1-1.6-3.9L14 6m0-4v4h-4" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg>`))
	})
	return refreshPic
}

func reapplyIcon() *ui.SVG {
	reapplyOnce.Do(func() {
		reapplyPic = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" width="11" height="11"><path d="M13 8a5 5 0 1 1-1.5-3.5L13 6m0-3.5v3.5H9.5" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg>`))
	})
	return reapplyPic
}

func dragHandleIcon() *ui.SVG {
	dragOnce.Do(func() {
		dragPic = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" width="12" height="12"><circle cx="6" cy="4" r="1.2" fill="currentColor"/><circle cx="10" cy="4" r="1.2" fill="currentColor"/><circle cx="6" cy="8" r="1.2" fill="currentColor"/><circle cx="10" cy="8" r="1.2" fill="currentColor"/><circle cx="6" cy="12" r="1.2" fill="currentColor"/><circle cx="10" cy="12" r="1.2" fill="currentColor"/></svg>`))
	})
	return dragPic
}

var (
	checkOnce sync.Once
	checkPic  *ui.SVG

	searchOnce sync.Once
	searchPic  *ui.SVG

	allProvidersOnce sync.Once
	allProvidersPic  *ui.SVG

	routingFanOnce sync.Once
	routingFanPic  *ui.SVG
)

func checkIcon() *ui.SVG {
	checkOnce.Do(func() {
		checkPic = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" width="13" height="13"><path d="m3.5 8.5 3 3 6-7" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/></svg>`))
	})
	return checkPic
}

func searchIcon() *ui.SVG {
	searchOnce.Do(func() {
		searchPic = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" width="12" height="12"><circle cx="7" cy="7" r="4.5" fill="none" stroke="currentColor" stroke-width="1.6"/><path d="m10.5 10.5 3 3" stroke="currentColor" stroke-width="1.6" stroke-linecap="round"/></svg>`))
	})
	return searchPic
}

func allProvidersIcon() *ui.SVG {
	allProvidersOnce.Do(func() {
		allProvidersPic = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" width="13" height="13"><path d="M2.5 4.5h11M2.5 8h11M2.5 11.5h11" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round"/></svg>`))
	})
	return allProvidersPic
}

func routingFanIcon() *ui.SVG {
	routingFanOnce.Do(func() {
		routingFanPic = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" width="13" height="13"><circle cx="3.5" cy="8" r="1.8" fill="none" stroke="currentColor" stroke-width="1.5"/><circle cx="12" cy="4" r="1.8" fill="none" stroke="currentColor" stroke-width="1.5"/><circle cx="12" cy="12" r="1.8" fill="none" stroke="currentColor" stroke-width="1.5"/><path d="M5.3 7.2l5-2.4M5.3 8.8l5 2.4" fill="none" stroke="currentColor" stroke-width="1.5"/></svg>`))
	})
	return routingFanPic
}

type modelModalGroup struct {
	name       string
	isRoute    bool
	icon       string
	entries    []provider.Entry
	shownCount int
	totalCount int
}

var picksIn = map[string]string{
	"codex":        "/model",
	"claude":       "/model",
	"opencode":     "/models",
	"pi":           "/model",
	"droid":        "/model",
	"kimi":         "/model",
	"grok":         "/model",
	"minimax-code": "/model",
	"dsh":          "/model",
	"crush":        "ctrl+p",
	"omp":          "/model",
	"qwen":         "/model",
	"mimocode":     "/model",
	"deveco":       "/models",
}

func cachedAgentCLI(a *agent.Agent) (agent.CLI, bool) {
	if val, ok := agCLICache.Load(a.ID); ok {
		return val.(agent.CLI), true
	}
	return agent.CLI{}, false
}

// loadCLIsAsync updates the CLI versions and update status asynchronously in the background,
// matching magpie's lazy loadCLIs behavior without delaying page rendering.
func (a *App) loadCLIsAsync() {
	go func() {
		clis, _ := agent.CLIs(3 * time.Second)
		var changed bool
		for id, c := range clis {
			if old, ok := agCLICache.Load(id); !ok || old.(agent.CLI) != c {
				agCLICache.Store(id, c)
				changed = true
			}
		}
		if changed && a.win != nil {
			a.win.Update(func() {
				for i := range a.data.Agents {
					if c, ok := agCLICache.Load(a.data.Agents[i].ID); ok {
						cli := c.(agent.CLI)
						if cli.Version != "" && cli.App != "" {
							a.data.Agents[i].Version = fmt.Sprintf("CLI %s App %s", cli.Version, cli.App)
						} else if cli.Version != "" {
							a.data.Agents[i].Version = cli.Version
						} else if cli.App != "" {
							a.data.Agents[i].Version = fmt.Sprintf("App %s", cli.App)
						}
					}
				}
				a.win.Invalidate()
			})
		}
	}()
}

// readAgentPageData reads all detected agents, uninstalled installs, and schemes.
func readAgentPageData() ([]AgentRow, []AgentInstallRow, []ProfileRow) {
	// The experiment's seam (docs/08 M4): the rows, the install list and
	// the schemes from a magpie web instance when the environment points
	// at one; embedded otherwise.
	if c := remoteGateway(); c != nil {
		if rows, installs, profiles, ok := readAgentsRemote(c); ok {
			// The state names only the agents it found; the roster of the
			// known ones is Echo's own, the same either way.
			return notFoundAgentRows(rows), installs, profiles
		}
		// The instance went away; fall through to the embedded read.
	}
	return readAgentsEmbedded()
}

// readAgentsEmbedded is the page's read from the domain packages this
// process carries.
func readAgentsEmbedded() ([]AgentRow, []AgentInstallRow, []ProfileRow) {
	detected := agent.Detected()
	s := settings.Load()
	detected, _ = settings.Arrange(s, detected, func(a *agent.Agent) string { return a.ID })
	all := agent.All()

	rows := make([]AgentRow, 0, len(all))
	for _, a := range detected {
		rows = append(rows, readOneAgentRow(a))
	}
	rows = notFoundAgentRows(rows)

	installs := agent.Installs()
	installRows := make([]AgentInstallRow, 0, len(installs))
	for _, inst := range installs {
		installRows = append(installRows, agentInstallRowOf(inst))
	}

	profileRows := make([]ProfileRow, 0)
	if ps, err := profile.Load(); err == nil {
		now := make(map[string]map[string]string, len(detected))
		for _, a := range detected {
			now[a.ID] = a.Values()
		}
		for _, n := range profile.Names(ps) {
			p := ps[n]
			profileRows = append(profileRows, ProfileRow{
				Name:    n,
				Summary: profile.Summary(p),
				InUse:   profile.InUse(p, detected, now),
			})
		}
	}

	return rows, installRows, profileRows
}

// notFoundAgentRows appends the rows for the agents the roster knows and
// the found rows don't name: uninstalled, they show under the card with
// their install commands. The roster is Echo's own, the same either way
// the page reads.
func notFoundAgentRows(rows []AgentRow) []AgentRow {
	found := make(map[string]bool, len(rows))
	for _, r := range rows {
		found[r.ID] = true
	}
	for _, a := range agent.All() {
		if !found[a.ID] {
			rows = append(rows, AgentRow{
				ID:    a.ID,
				Name:  a.Name,
				Icon:  a.Icon,
				Found: false,
				Path:  a.Path,
			})
		}
	}
	return rows
}

// agentInstallRowOf is an uninstalled agent's row, its install command
// named by what runs it.
func agentInstallRowOf(inst agent.Install) AgentInstallRow {
	via := "Terminal"
	cmd := ""
	if len(inst.Commands) > 0 {
		cmd = inst.Commands[0].Command
		switch inst.Commands[0].Via {
		case "npm":
			via = "Node.js + npm"
		case "brew":
			via = "Homebrew"
		case "powershell":
			via = "PowerShell"
		case "script":
			via = "Script"
		}
	}
	return AgentInstallRow{
		ID:      inst.ID,
		Name:    inst.Name,
		Icon:    inst.Icon,
		Via:     via,
		Command: cmd,
	}
}

func readOneAgentRow(a *agent.Agent) AgentRow {
	wired := a.Wired()
	drift := a.Drift()
	vals := a.Values()

	var versionStr string
	if cli, ok := cachedAgentCLI(a); ok {
		if cli.Version != "" && cli.App != "" {
			versionStr = fmt.Sprintf("CLI %s App %s", cli.Version, cli.App)
		} else if cli.Version != "" {
			versionStr = cli.Version
		} else if cli.App != "" {
			versionStr = fmt.Sprintf("App %s", cli.App)
		}
	} else {
		appV := a.AppVersion()
		cliV := a.InstalledVersion()
		if cliV != "" && appV != "" {
			versionStr = fmt.Sprintf("CLI %s App %s", cliV, appV)
		} else if cliV != "" {
			versionStr = cliV
		} else if appV != "" {
			versionStr = fmt.Sprintf("App %s", appV)
		}
	}

	listed, _ := provider.ListedFor(a.ListsFor())
	off := provider.ModelOff(a.ListsFor())
	var shownCount int
	var names []string
	groupMap := make(map[string]*AgentGroupChip)
	var groupOrder []string

	for _, e := range listed {
		if off(e.ID) {
			continue
		}
		shownCount++
		names = append(names, cmp.Or(e.Name, e.Model, e.ID))
		gName := e.Provider.Name
		gIcon := e.Provider.Icon
		var gIcons []string
		if e.Group != "" {
			gName = "路由组"
			gIcons = e.Icons
		}
		if chip, exists := groupMap[gName]; exists {
			chip.Count++
		} else {
			chip := &AgentGroupChip{
				Name:  gName,
				Icon:  gIcon,
				Icons: gIcons,
				Count: 1,
			}
			groupMap[gName] = chip
			groupOrder = append(groupOrder, gName)
		}
	}

	groups := make([]AgentGroupChip, 0, len(groupOrder))
	for _, gName := range groupOrder {
		groups = append(groups, *groupMap[gName])
	}

	var subtitle string
	var isDrift bool
	var driftText string
	if drift != nil {
		isDrift = true
		driftText = fmt.Sprintf("%s 已不经过 Echo: 配置已变更", a.Name)
		subtitle = driftText
	} else if wired {
		cmd := picksIn[a.ID]
		if a.ID == "claude" {
			subtitle = "已接入 · 新开的会话生效"
		} else if a.ID == "cursor" || a.ID == "cursor-cli" {
			subtitle = fmt.Sprintf("已接入 · Echo 的模型在 %s 的模型列表里", a.Name)
		} else if cmd != "" && shownCount > 0 {
			subtitle = fmt.Sprintf("已接入 · %s 的 %s 里有 %d 个模型", a.Name, cmd, shownCount)
		} else if a.ListsModels && shownCount > 0 {
			subtitle = fmt.Sprintf("已接入 · %s 的模型菜单里有 %d 个模型", a.Name, shownCount)
		} else {
			subtitle = fmt.Sprintf("已接入 · %s 的模型列表已同步", a.Name)
		}
	} else {
		subtitle = fmt.Sprintf("未接入 · %s 使用其自身设置", a.Name)
	}

	var detailsText string
	if a.ID == "claude" {
		detailsText = "档位 ›"
	} else if shownCount > 0 && !a.ListsModels {
		detailsText = fmt.Sprintf("模型 %d ›", shownCount)
	} else {
		detailsText = "详情 ›"
	}

	fieldKey := "model"
	var currentVal string
	var currentName string
	var currentIcon string
	var options []AgentModelOpt

	var targetField *agent.Field
	for i := range a.Fields {
		f := &a.Fields[i]
		if f.Key == "model" || f.Label == "model" {
			targetField = f
			break
		}
	}
	if targetField == nil && len(a.Fields) > 0 {
		targetField = &a.Fields[0]
	}

	if targetField != nil {
		fieldKey = targetField.Key
		currentVal = vals[fieldKey]

		picks := a.Picks(*targetField, vals)
		for _, p := range picks {
			selected := p.Value == currentVal
			opt := AgentModelOpt{
				ID:       p.Value,
				Label:    cmp.Or(p.Label, p.Value),
				Icon:     p.Icon,
				Selected: selected,
			}
			options = append(options, opt)
			if selected {
				currentName = opt.Label
				currentIcon = opt.Icon
			}
		}
	}

	// the effort the agent runs on, as the tray panel's bars say it
	effort := ""
	for i := range a.Fields {
		f := &a.Fields[i]
		if f.Key == "effort" || f.Label == "effort" || f.Label == "thinking" {
			effort = vals[f.Key]
			break
		}
	}

	// the words on the row's model dropdown: the model the agent is set to
	// when it has one, and otherwise the models it is offered — two of them
	// by name, as magpie shows a list of models, and an ellipsis when there
	// are more. An agent whose own menu is the list of models shows the
	// models, not a pick: it picks none.
	switch {
	case len(names) > 0 && (a.ListsModels || currentName == ""):
		currentName = agentModelsLabel(names)
	case currentName == "":
		if len(options) > 0 {
			currentName = options[0].Label
			currentIcon = options[0].Icon
		} else if currentVal != "" {
			currentName = currentVal
		} else {
			currentName = "默认"
		}
	}

	var notice string
	if a.Notice != nil {
		notice = a.Notice()
	}

	return AgentRow{
		ID:          a.ID,
		Name:        a.Name,
		Icon:        a.Icon,
		Found:       true,
		Path:        a.Path,
		Plugin:      a.Plugin,
		Wired:       wired,
		Drift:       isDrift,
		DriftText:   driftText,
		Version:     versionStr,
		Subtitle:    subtitle,
		ModelCount:  shownCount,
		DetailsText: detailsText,
		FieldKey:    fieldKey,
		CurrentVal:  currentVal,
		CurrentName: currentName,
		CurrentIcon: currentIcon,
		Effort:      effort,
		ModelList:   a.ListsModels,
		ModelNames:  names,
		Options:     options,
		Groups:      groups,
		Notice:      notice,
	}
}

// agentModelsLabel is the words for a list of routes and models: the first
// two by name, and an ellipsis when there are more — magpie's rule for the
// models an agent's own menu lists (its menuSaid).
func agentModelsLabel(names []string) string {
	switch len(names) {
	case 0:
		return ""
	case 1:
		return names[0]
	case 2:
		return names[0] + ", " + names[1]
	default:
		return names[0] + ", " + names[1] + " …"
	}
}

// agentModelsSaid is the same list as magpie's panel says it: the first
// two by name, and the count of the rest.
func agentModelsSaid(names []string, shown int) string {
	if len(names) == 0 {
		return ""
	}
	if len(names) > 2 {
		names = names[:2]
	}
	said := strings.Join(names, ", ")
	if shown > len(names) {
		said += fmt.Sprintf(" +%d", shown-len(names))
	}
	return said
}

// driftColor is the red magpie marks a row that no longer goes through the
// app with (--drift in magpie's app.css), and driftSoft the pale red behind
// the pill that offers to put it back (--drift-soft). magpie's own values,
// not the theme's Danger: a drifted agent is a notice, not a form error.
func driftColor(t *ui.Theme) ui.Color {
	if t.Dark {
		return ui.Hex("#ff9592")
	}
	return ui.Hex("#ce2c31")
}

// driftSoft is the pill's own fill, and driftSoftHover the fill while the
// pointer is on it.
func driftSoft(t *ui.Theme) ui.Color {
	if t.Dark {
		return ui.Hex("#3b1219")
	}
	return ui.Hex("#feebec")
}

func driftSoftHover(t *ui.Theme) ui.Color {
	if t.Dark {
		return ui.Hex("#521a24")
	}
	return ui.Hex("#fbdcdf")
}

// agentsView renders the Agent page conforming to native UI layout and spacing rules.
func (a *App) agentsView(c *ui.Context) {
	t := c.Theme()

	// Observe generous sidebar spacing (total distance >= 36-40px from divider)
	ui.Column(c).Fill().Padding(8, 24, 0, 24).Children(func() {
		a.agentsLead(c, t)
		a.agentsCard(c, t)
		a.agentsInstallSection(c, t)
		a.agentsProfilesSection(c, t)
	})

	if a.agModalAgentID != "" {
		a.agentModalDialog(c, t)
	}
}

// agentsLead renders the top description row with refresh button.
func (a *App) agentsLead(c *ui.Context, t *ui.Theme) {
	ui.Row(c).AlignItems(ui.Center).Gap(10).Padding(2, 0, 8, 0).Children(func() {
		ui.Text(c, a.text.S.AgLead).
			FontSize(12).
			TextColor(t.TextMuted).
			Grow(1).
			Shrink(1).
			Wrap()

		refreshBtn := ui.Row(c).Center().Width(24).Height(24).Radius(5).
			Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).
			Label(a.text.S.Loading)
		if refreshBtn.Hovered() {
			if t.Dark {
				refreshBtn.Background(ui.RGBA(255, 255, 255, 0.08))
			} else {
				refreshBtn.Background(ui.RGBA(0, 0, 0, 0.05))
			}
		}
		if refreshBtn.Clicked() {
			go func() {
				agentRescan()
				a.Reload()
			}()
		}
		refreshBtn.Children(func() {
			ui.Icon(c, refreshIcon()).Size(13, 13).TextColor(t.TextMuted)
		})
	})
}

// agentsCard is the main rounded card container holding connected/detected agents.
func (a *App) agentsCard(c *ui.Context, t *ui.Theme) {
	var detected []AgentRow
	for _, ag := range a.data.Agents {
		if ag.Found {
			detected = append(detected, ag)
		}
	}

	card := ui.Column(c).FillWidth().Radius(10).
		Background(t.Background).Border(1, t.Border)

	if len(detected) == 0 {
		card.Padding(24, 16).Center().Children(func() {
			if a.loading {
				ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
					ui.Spinner(c).Size(14, 14)
					ui.Text(c, a.text.S.Loading).FontSize(12.5).TextColor(t.TextMuted)
				})
			} else {
				ui.Text(c, a.text.S.AgNoAgentsFound).FontSize(13).FontWeight(600).TextColor(t.Text)
				ui.Text(c, a.text.S.AgNoAgentsHint).FontSize(12).TextColor(t.TextMuted).Margin(4, 0, 0, 0)
			}
		})
		return
	}

	shown := detected
	maxFold := 8
	if a.agFolded && len(detected) > maxFold {
		shown = detected[:maxFold]
	}

	lineC := ui.RGBA(0, 0, 0, 0.07)
	if t.Dark {
		lineC = ui.RGBA(255, 255, 255, 0.07)
	}

	card.Children(func() {
		for i, ag := range shown {
			ag := ag
			if i > 0 {
				ui.Row(c).FillWidth().Height(0.5).Background(lineC)
			}
			a.agentRow(c, t, ag, i, len(shown))
		}

		// Footer
		if len(detected) > maxFold {
			a.agentsCardFooter(c, t, len(detected)-maxFold, lineC)
		}
	})
}

// agentRow renders one agent item in the card with native drag-and-drop reordering.
func (a *App) agentRow(c *ui.Context, t *ui.Theme, ag AgentRow, index, total int) {
	fieldBg := a.fieldBg(t)

	row := ui.Row(c).AlignItems(ui.Center).Gap(10).
		Padding(8, 14).MinHeight(52)

	// Highlight row when an agent is dragged over it
	if draggedID, over := ui.DragOver[string](row); over && draggedID != ag.ID {
		row.Background(ui.RGBA(0, 122, 255, 0.08)).
			BorderWidth(2, 0, 0, 0).BorderColor(t.Accent)
	}
	// Commit reorder when dropped onto this row
	if droppedID, ok := ui.Drop[string](row); ok && droppedID != ag.ID {
		a.reorderAgentByID(droppedID, ag.ID)
	}

	row.Children(func() {
		// Drag handle: native Drag and secondary ContextMenu
		handle := ui.Row(c).Center().Width(22).Height(34).Radius(4).
			Cursor(ui.CursorGrab).
			Drag(ag.ID)

		if handle.Dragging() {
			handle.Cursor(ui.CursorGrabbing)
			row.Opacity(0.4)
		}
		if handle.Hovered() {
			handle.Background(fieldBg)
		}

		handle.ContextMenu(func(m *ui.Menu) {
			if index > 0 {
				if m.Item(a.text.S.AgMoveToTop).Chosen() {
					a.moveAgent(ag.ID, "top")
				}
				if m.Item(a.text.S.AgMoveUp).Chosen() {
					a.moveAgent(ag.ID, "up")
				}
			}
			if index < total-1 {
				if m.Item(a.text.S.AgMoveDown).Chosen() {
					a.moveAgent(ag.ID, "down")
				}
				if m.Item(a.text.S.AgMoveToBottom).Chosen() {
					a.moveAgent(ag.ID, "bottom")
				}
			}
		})

		handle.Children(func() {
			ui.Icon(c, dragHandleIcon()).Size(12, 12).TextColor(ui.Hex("#9ca3af"))
		})

		// Agent icon (also draggable)
		picBox := ui.Row(c).Center().Cursor(ui.CursorGrab).Drag(ag.ID)
		if picBox.Dragging() {
			row.Opacity(0.4)
		}
		picBox.Children(func() {
			pic := iconPictureFor(ag.Icon)
			ui.Image(c, pic).Size(26, 26).Radius(5)
		})

		// Title & Subtitle column
		ui.Column(c).Grow(1).Shrink(1).Gap(2).Children(func() {
			ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
				ui.Text(c, ag.Name).FontSize(13).FontWeight(600).TextColor(t.Text).SingleLine()
				if ag.Version != "" {
					ui.Text(c, ag.Version).FontSize(11).TextColor(t.TextMuted).SingleLine()
				}
			})

			ui.Row(c).AlignItems(ui.Center).Gap(5).Children(func() {
				dotColor := ui.Hex("#8e8e93")
				if ag.Drift {
					dotColor = driftColor(t)
				} else if ag.Wired {
					dotColor = ui.Hex("#34c759")
				}
				ui.Row(c).Width(6).Height(6).Radius(3).Background(dotColor)

				// a drifted agent's notice is in magpie's drift colour: the
				// red that marks a row which no longer goes through the app
				subColor := t.TextMuted
				if ag.Drift {
					subColor = driftColor(t)
				}
				ui.Text(c, ag.Subtitle).FontSize(11.5).TextColor(subColor).SingleLine()

				if ag.Drift {
					// the reapply button is magpie's drift pill: the soft
					// red behind the drift red, as .ag-fix is in app.css
					reapplyBtn := ui.Row(c).AlignItems(ui.Center).Gap(4).
						Padding(0, 8, 0, 6).Height(20).Radius(999).Cursor(ui.CursorPointer).
						Background(driftSoft(t)).
						Focusable().Role(ui.RoleButton).Label(a.text.S.AgReapply)
					if reapplyBtn.Hovered() {
						reapplyBtn.Background(driftSoftHover(t))
					}
					if reapplyBtn.Clicked() {
						agID := ag.ID
						go func() {
							if err := agentReapply(agID); err == nil {
								a.Reload()
							}
						}()
					}
					reapplyBtn.Children(func() {
						ui.Icon(c, reapplyIcon()).Size(11, 11).TextColor(driftColor(t))
						ui.Text(c, a.text.S.AgReapply).FontSize(11.5).FontWeight(500).TextColor(driftColor(t)).SingleLine()
					})
				}
			})
		})

		// Right Controls: Details link, Model Picker Menu, Toggle Switch
		ui.Row(c).AlignItems(ui.Center).Gap(10).Shrink(0).Children(func() {
			// An agent that is not connected has no models to pick and no
			// list to open: its row keeps the name, the notice and the
			// switch that connects it, and nothing else.
			if ag.Wired {
				// Details button: opens modal popup dialog
				detailsBtn := ui.Row(c).AlignItems(ui.Center).Padding(4, 6).Radius(5).
					Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).
					Label(ag.DetailsText)
				if detailsBtn.Hovered() {
					detailsBtn.Background(fieldBg)
				}
				if detailsBtn.Clicked() {
					a.openAgentModelModal(ag.ID)
					c.Invalidate()
				}
				detailsBtn.Children(func() {
					ui.Text(c, ag.DetailsText).FontSize(11.5).TextColor(t.TextMuted).SingleLine()
				})

				// Model Picker Dropdown Button: one width down the list, as
				// magpie's model column is — the long names cut off inside it
				//
				// The button carries the models the agent is offered: two of
				// them by name and an ellipsis when there are more, and the
				// whole list in its tip and in what a screen reader reads.
				modelLabel := ag.CurrentName
				if len(ag.ModelNames) > 0 {
					modelLabel = strings.Join(ag.ModelNames, ", ")
				}
				modelBtn := ui.Row(c).AlignItems(ui.Center).Gap(6).
					Padding(0, 8).Height(26).Width(210).Shrink(0).Radius(7).
					Border(1, t.Border).Background(fieldBg).
					Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).
					Label(modelLabel).Tooltip(modelLabel)
				if modelBtn.Hovered() {
					modelBtn.Border(1, t.Accent)
				}
				// the button opens the model list dialog itself — the enabled
				// providers and their models, as the details link does
				if modelBtn.Clicked() {
					a.openAgentModelModal(ag.ID)
					c.Invalidate()
				}
				modelBtn.Children(func() {
					if ag.CurrentIcon != "" {
						if pic := iconPictureFor(ag.CurrentIcon); pic != nil {
							ui.Image(c, pic).Size(13, 13).Radius(2)
						}
					}
					ui.Text(c, ag.CurrentName).FontSize(11.5).FontWeight(500).TextColor(t.Text).
						Grow(1).Shrink(1).SingleLine()
					ui.Icon(c, chevDown()).Size(9, 9).TextColor(t.TextMuted)
				})
			}

			// Toggle Switch
			on := ag.Wired
			if sw := ui.Switch(c, &on).Shrink(0); sw.Clicked() {
				agID := ag.ID
				go func(targetOn bool) {
					if err := agentConnect(agID, targetOn); err == nil {
						a.Reload()
					}
				}(on)
			}
		})
	})
}

// agentModalDialog renders the modal dialog for choosing models and setting tiers.
func (a *App) agentModalDialog(c *ui.Context, t *ui.Theme) {
	var curAg *AgentRow
	for i := range a.data.Agents {
		if a.data.Agents[i].ID == a.agModalAgentID {
			curAg = &a.data.Agents[i]
			break
		}
	}
	if curAg == nil {
		a.agModalAgentID = ""
		return
	}
	ag := *curAg

	ui.Overlay(c, func() {
		back := ui.Box(c).Absolute().Left(0).Top(0).Right(0).Bottom(0).Center().
			Background(ui.RGBA(0, 0, 0, 0.4)).Modal()
		if back.OverlayShortcut(0, ui.KeyEscape) {
			a.agModalAgentID = ""
			c.Invalidate()
			a.invalidate()
			return
		}
		back.Children(func() {
			cw, ch := c.Size()
			dialog := ui.Column(c).Width(640).
				MaxWidth(cw-40).MaxHeight(ch-40).
				Radius(12).Padding(16, 20).Gap(12).
				Role(ui.RoleDialog).Label(ag.Name)
			if t.Dark {
				dialog.Background(ui.RGBA(36, 36, 40, 1))
			} else {
				dialog.Background(t.Background)
			}
			dialog.Border(1, t.Border).
				Shadow(0, 16, 40, 0, ui.RGBA(0, 0, 0, 0.35))

			dialog.Children(func() {
				a.agentModalContent(c, t, ag)
			})
		})
	})
}

// openAgentModelModal opens the model list dialog for agID: the enabled
// providers and their models, as the details link and the row's model
// button both show it.
func (a *App) openAgentModelModal(agID string) {
	a.agModalAgentID = agID
	a.agModalSearch = ""
	a.agModalFilter = 0
	a.agModalGroup = ""
	if a.agModalFolded == nil {
		a.agModalFolded = make(map[string]bool)
	}
	listsFor := agID
	if agObj, err := agent.Find(agID); err == nil {
		listsFor = agObj.ListsFor()
	}
	_, only := provider.PickedModels(listsFor)
	a.agModalOnly = only
}

// agentModalContent renders the content inside the agent settings modal dialog.
func (a *App) agentModalContent(c *ui.Context, t *ui.Theme, ag AgentRow) {
	fieldBg := a.fieldBg(t)
	lineC := ui.RGBA(0, 0, 0, 0.08)
	if t.Dark {
		lineC = ui.RGBA(255, 255, 255, 0.08)
	}
	if a.agModalFolded == nil {
		a.agModalFolded = make(map[string]bool)
	}

	listsFor := ag.ID
	var agObj *agent.Agent
	if obj, err := agent.Find(ag.ID); err == nil {
		agObj = obj
		listsFor = obj.ListsFor()
	}
	listed, _ := provider.ListedFor(listsFor)
	off := provider.ModelOff(listsFor)

	// Header row
	ui.Row(c).AlignItems(ui.Start).FillWidth().Children(func() {
		ui.Column(c).Grow(1).Shrink(1).Gap(3).Children(func() {
			title := strings.ReplaceAll(a.text.S.AgModalTitle, "{agent}", ag.Name)
			sub := strings.ReplaceAll(a.text.S.AgModalSub, "{agent}", ag.Name)
			if ag.ID == "claude" {
				title = a.text.S.AgModalClaudeTitle
				sub = a.text.S.AgModalClaudeSub
			}
			ui.Text(c, title).FontSize(15).FontWeight(600).TextColor(t.Text).SingleLine()
			ui.Text(c, sub).FontSize(12).TextColor(t.TextMuted).Wrap()
		})

		closeBtn := ui.Row(c).Center().Width(24).Height(24).Radius(5).
			Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).Label("Close")
		if closeBtn.Hovered() {
			closeBtn.Background(fieldBg)
		}
		if closeBtn.Clicked() {
			a.agModalAgentID = ""
			c.Invalidate()
		}
		closeBtn.Children(func() {
			ui.Icon(c, closeIcon()).Size(10, 10).TextColor(t.TextMuted)
		})
	})

	// Claude Code Tiers Section (if claude)
	if ag.ID == "claude" {
		tiersBox := ui.Column(c).FillWidth().Radius(8).
			Background(fieldBg).Border(1, t.Border).Padding(8, 12).Gap(6)
		tiersBox.Children(func() {
			ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
				ui.Text(c, a.text.S.AgTiersSection).FontSize(12).FontWeight(600).TextColor(t.Text).SingleLine()
				ui.Text(c, a.text.S.AgTiersDesc).FontSize(11).TextColor(t.TextMuted).SingleLine()
			})

			vals := map[string]string{}
			if agObj != nil {
				vals = agObj.Values()
			}

			tiers := []struct {
				key   string
				label string
				hint  string
			}{
				{"opus", "Opus", "/model opus"},
				{"sonnet", "Sonnet", "/model sonnet · " + a.text.S.AgDefaultTier},
				{"haiku", "Haiku", "/model haiku · " + a.text.S.AgSubagentHint},
				{"subagent", "Subagents", "子代理默认模型"},
			}

			for _, tr := range tiers {
				tr := tr
				row := ui.Row(c).AlignItems(ui.Center).Gap(10).MinHeight(24)
				row.Children(func() {
					ui.Text(c, tr.label).FontSize(11.5).FontWeight(500).Width(70).TextColor(t.Text).SingleLine()

					currVal := vals[tr.key]
					currName := currVal
					if currName == "" {
						currName = "默认"
					}
					for _, opt := range ag.Options {
						if opt.ID == currVal {
							currName = opt.Label
							break
						}
					}

					btn := ui.Row(c).AlignItems(ui.Center).Gap(6).
						Padding(1, 8).Height(22).Radius(5).
						Background(t.Background).Border(1, t.Border).
						Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton)
					if btn.Hovered() {
						btn.Border(1, t.Accent)
					}
					btn.Menu(func(m *ui.Menu) {
						for _, opt := range ag.Options {
							opt := opt
							item := m.Item(opt.Label)
							if opt.ID == currVal {
								item.Checked(true)
							}
							if item.Chosen() {
								go func() {
									if err := agentSet(ag.ID, tr.key, opt.ID); err == nil {
										a.Reload()
									}
								}()
							}
						}
					})
					btn.Children(func() {
						ui.Text(c, currName).FontSize(11).FontWeight(500).TextColor(t.Text).SingleLine()
						ui.Icon(c, chevDown()).Size(8, 8).TextColor(t.TextMuted)
					})

					ui.Text(c, tr.hint).FontSize(11).TextColor(t.TextMuted).SingleLine()
				})
			}
		})
	}

	// Top Toolbar: Search input (left) + Segmented pills (right)
	ui.Row(c).AlignItems(ui.Center).Gap(12).FillWidth().Children(func() {
		// Search Input (with search icon inside)
		ui.SearchField(c, &a.agModalSearch).
			Label(a.text.S.AgSearchModels).
			Grow(1).Shrink(1)

		// Segmented Tabs: 全部 | 已显示 | 排序
		segBg := ui.RGBA(0, 0, 0, 0.05)
		if t.Dark {
			segBg = ui.RGBA(255, 255, 255, 0.08)
		}
		seg := ui.Row(c).AlignItems(ui.Center).Padding(2, 2).Radius(8).
			Background(segBg).Shrink(0).Gap(2)
		seg.Children(func() {
			tabs := []struct {
				idx   int
				label string
			}{
				{0, a.text.S.AgAllModels},
				{1, a.text.S.AgEnabledModels},
				{2, a.text.S.AgOrderModels},
			}
			for _, tb := range tabs {
				tb := tb
				isActive := (a.agModalFilter == tb.idx)
				pill := ui.Row(c).Key(fmt.Sprintf("ag-tab-%d", tb.idx)).
					Center().Padding(0, 12).Height(26).Radius(6).
					Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton)
				if isActive {
					pill.Background(t.Background).
						Shadow(0, 1, 3, 0, ui.RGBA(0, 0, 0, 0.12))
				}
				if pill.Clicked() {
					a.agModalFilter = tb.idx
					c.Invalidate()
					a.invalidate()
				}
				pill.Children(func() {
					textColor := t.TextMuted
					fontW := 400
					if isActive {
						textColor = t.Text
						fontW = 500
					}
					ui.Text(c, tb.label).FontSize(12).FontWeight(fontW).
						TextColor(textColor).SingleLine()
				})
			}
		})
	})

	// Group entries by provider / routing group
	var groups []*modelModalGroup
	groupMap := map[string]*modelModalGroup{}
	var totalAll, shownAll int
	var shownEntries []provider.Entry

	for _, e := range listed {
		totalAll++
		isHidden := off(e.ID)
		if !isHidden {
			shownAll++
			shownEntries = append(shownEntries, e)
		}

		gName := e.Provider.Name
		isRoute := false
		gIcon := e.Provider.Icon
		if e.Group != "" {
			gName = a.text.S.GwRoutingGroup
			if gName == "" {
				gName = "路由组"
			}
			isRoute = true
		}

		g, ok := groupMap[gName]
		if !ok {
			g = &modelModalGroup{
				name:    gName,
				isRoute: isRoute,
				icon:    gIcon,
			}
			groupMap[gName] = g
			groups = append(groups, g)
		}
		g.entries = append(g.entries, e)
		g.totalCount++
		if !isHidden {
			g.shownCount++
		}
	}

	// Sort groups: routing group first, then others
	sort.SliceStable(groups, func(i, j int) bool {
		if groups[i].isRoute != groups[j].isRoute {
			return groups[i].isRoute
		}
		return false
	})

	// Two-Column Split Body
	bodyRow := ui.Row(c).FillWidth().Height(360).Gap(10)
	bodyRow.Children(func() {
		// Left Rail (am-rail): Provider Tree / List
		rail := ui.Scroll(c).Width(165).FillHeight().Padding(0, 2, 0, 0)
		rail.Children(func() {
			ui.Column(c).FillWidth().Gap(2).Children(func() {
				// "全部供应商"
				allActive := (a.agModalGroup == "")
				allRow := ui.Row(c).Key("ag-pv-all").
					AlignItems(ui.Center).Height(32).Padding(0, 8).
					Radius(6).Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton)
				if allActive {
					allRow.Background(ui.RGBA(0, 122, 255, 0.1))
				} else if allRow.Hovered() {
					allRow.Background(fieldBg)
				}
				if allRow.Clicked() {
					a.agModalGroup = ""
					a.agModalSearch = ""
					c.Invalidate()
					a.invalidate()
				}
				allRow.Children(func() {
					ui.Icon(c, allProvidersIcon()).Size(13, 13).TextColor(t.Text).Margin(0, 8, 0, 0)
					ui.Text(c, a.text.S.AgAllProviders).FontSize(12.5).FontWeight(500).
						TextColor(t.Text).Grow(1).Shrink(1).SingleLine()
					ui.Text(c, fmt.Sprintf("%d/%d", shownAll, totalAll)).FontSize(11).
						TextColor(t.TextMuted).Shrink(0).SingleLine()
				})

				// Each group / provider
				for _, g := range groups {
					g := g
					isActive := (a.agModalGroup == g.name)
					pRow := ui.Row(c).Key("ag-pv-"+g.name).
						AlignItems(ui.Center).Height(32).Padding(0, 8).
						Radius(6).Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton)
					if isActive {
						pRow.Background(ui.RGBA(0, 122, 255, 0.1))
					} else if pRow.Hovered() {
						pRow.Background(fieldBg)
					}
					if pRow.Clicked() {
						a.agModalGroup = g.name
						a.agModalSearch = ""
						c.Invalidate()
						a.invalidate()
					}
					pRow.Children(func() {
						if g.isRoute {
							ui.Icon(c, routingFanIcon()).Size(14, 14).TextColor(t.Text).Margin(0, 8, 0, 0)
						} else {
							pic := iconPictureFor(g.icon)
							ui.Image(c, pic).Size(16, 16).Radius(3).Margin(0, 8, 0, 0)
						}
						ui.Text(c, g.name).FontSize(12.5).FontWeight(500).
							TextColor(t.Text).Grow(1).Shrink(1).SingleLine()
						ui.Text(c, fmt.Sprintf("%d/%d", g.shownCount, g.totalCount)).FontSize(11).
							TextColor(t.TextMuted).Shrink(0).SingleLine()
					})
				}
			})
		})

		// Divider line
		ui.Row(c).Width(0.5).FillHeight().Background(lineC)

		// Right Pane: Grouped Model List or Order List
		list := ui.Scroll(c).Grow(1).Shrink(1).FillHeight().Padding(0, 2, 0, 6)
		list.Children(func() {
			// Tab 2: Order Mode
			if a.agModalFilter == 2 {
				ui.Column(c).FillWidth().Gap(4).Children(func() {
					hint := strings.ReplaceAll(a.text.S.AgOrderHint, "{agent}", ag.Name)
					ui.Text(c, hint).FontSize(11.5).TextColor(t.TextMuted).Margin(2, 4, 6, 4).Wrap()

					q := strings.ToLower(strings.TrimSpace(a.agModalSearch))
					var renderedCount int
					for i, e := range shownEntries {
						i := i
						e := e
						mName := cmp.Or(e.Name, e.Model, e.ID)
						gName := e.Provider.Name
						if e.Group != "" {
							gName = a.text.S.GwRoutingGroup
							if gName == "" {
								gName = "路由组"
							}
						}
						if a.agModalGroup != "" && gName != a.agModalGroup {
							continue
						}
						if q != "" {
							if !strings.Contains(strings.ToLower(mName), q) &&
								!strings.Contains(strings.ToLower(e.ID), q) &&
								!strings.Contains(strings.ToLower(gName), q) {
								continue
							}
						}
						renderedCount++

						oRow := ui.Row(c).AlignItems(ui.Center).Gap(8).Padding(0, 8).
							Height(34).Radius(6).Cursor(ui.CursorGrab)
						if oRow.Hovered() {
							oRow.Background(fieldBg)
						}

						// Drag and drop reordering
						if draggedID, over := ui.DragOver[string](oRow); over && draggedID != e.ID {
							oRow.Background(ui.RGBA(0, 122, 255, 0.08)).
								BorderWidth(2, 0, 0, 0).BorderColor(t.Accent)
						}
						if droppedID, ok := ui.Drop[string](oRow); ok && droppedID != e.ID {
							a.reorderAgentModel(listsFor, droppedID, e.ID, shownEntries)
						}

						oRow.Children(func() {
							handle := ui.Row(c).Center().Width(18).Height(24).Radius(3).
								Cursor(ui.CursorGrab).Drag(e.ID)
							if handle.Dragging() {
								handle.Cursor(ui.CursorGrabbing)
								oRow.Opacity(0.4)
							}
							handle.Children(func() {
								ui.Icon(c, dragHandleIcon()).Size(11, 11).TextColor(ui.Hex("#9ca3af"))
							})

							ui.Text(c, fmt.Sprint(i+1)).FontSize(11).TextColor(t.TextMuted).Width(18).SingleLine()

							// Maker Logo
							logo := agent.ModelIcon(e.Model)
							if logo == "" {
								logo = agent.ModelIcon(e.ID)
							}
							if logo == "" && e.Group == "" {
								logo = e.Provider.Icon
							}
							if logo != "" && logo != "generic" {
								ui.Image(c, iconPictureFor(logo)).Size(16, 16).Radius(3)
							} else {
								ui.Row(c).Width(16).Height(16)
							}

							ui.Text(c, mName).FontSize(12.5).FontWeight(500).TextColor(t.Text).
								Grow(1).Shrink(1).SingleLine()

							ui.Text(c, gName).FontSize(11).TextColor(t.TextMuted).Shrink(0).SingleLine()
						})
					}

					if renderedCount == 0 {
						ui.Row(c).Padding(24, 0).Center().Children(func() {
							ui.Text(c, a.text.S.NoData).FontSize(12).TextColor(t.TextMuted)
						})
					}
				})
				return
			}

			// Tab 0 & 1: Normal List Mode
			q := strings.ToLower(strings.TrimSpace(a.agModalSearch))
			renderedAny := false

			ui.Column(c).FillWidth().Gap(6).Children(func() {
				for _, g := range groups {
					g := g
					if a.agModalGroup != "" && g.name != a.agModalGroup {
						continue
					}

					// Filter models in group
					var groupModels []provider.Entry
					for _, e := range g.entries {
						isDisabled := off(e.ID)
						if a.agModalFilter == 1 && isDisabled {
							continue
						}
						if q != "" {
							mName := cmp.Or(e.Name, e.Model, e.ID)
							if !strings.Contains(strings.ToLower(mName), q) &&
								!strings.Contains(strings.ToLower(e.ID), q) &&
								!strings.Contains(strings.ToLower(g.name), q) {
								continue
							}
						}
						groupModels = append(groupModels, e)
					}

					if len(groupModels) == 0 {
						continue
					}
					renderedAny = true

					// Group Section
					ui.Column(c).FillWidth().Gap(1).Children(func() {
						// Group Header
						isFolded := a.agModalFolded[g.name]
						gh := ui.Row(c).AlignItems(ui.Center).Gap(6).Height(26).Padding(0, 4).
							Radius(5).Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton)
						if gh.Hovered() {
							gh.Background(fieldBg)
						}
						if gh.Clicked() {
							a.agModalFolded[g.name] = !isFolded
							c.Invalidate()
						}
						gh.Children(func() {
							chev := chevDown()
							if isFolded {
								chev = chevRight()
							}
							ui.Icon(c, chev).Size(8, 8).TextColor(t.TextMuted)

							if g.isRoute {
								ui.Icon(c, routingFanIcon()).Size(14, 14).TextColor(t.Text)
							} else {
								ui.Image(c, iconPictureFor(g.icon)).Size(14, 14).Radius(3)
							}

							ui.Text(c, g.name).FontSize(12.5).FontWeight(600).TextColor(t.Text).SingleLine()
							ui.Text(c, fmt.Sprintf("%d / %d", g.shownCount, g.totalCount)).
								FontSize(11).TextColor(t.TextMuted).SingleLine()
						})

						// Model Items (if not folded)
						if !isFolded {
							for _, e := range groupModels {
								e := e
								mName := cmp.Or(e.Name, e.Model, e.ID)
								isDisabled := off(e.ID)
								isInUse := (e.ID == ag.CurrentVal || e.Model == ag.CurrentVal)

								mRow := ui.Row(c).AlignItems(ui.Center).Gap(8).
									Padding(0, 8, 0, 18).Height(32).Radius(6).
									Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton)
								if mRow.Hovered() {
									mRow.Background(fieldBg)
								}
								if mRow.Clicked() {
									if !isInUse {
										a.toggleAgentModel(ag.ID, e.ID, isDisabled)
									}
								}

								mRow.Children(func() {
									// Model Maker Logo
									logo := agent.ModelIcon(e.Model)
									if logo == "" {
										logo = agent.ModelIcon(e.ID)
									}
									if logo == "" && !g.isRoute {
										logo = e.Provider.Icon
									}
									if logo != "" && logo != "generic" {
										ui.Image(c, iconPictureFor(logo)).Size(16, 16).Radius(3)
									} else {
										ui.Row(c).Width(16).Height(16)
									}

									// Model Name
									nameColor := t.Text
									if isDisabled {
										nameColor = t.TextMuted
									}
									ui.Text(c, mName).FontSize(12.5).TextColor(nameColor).
										Grow(1).Shrink(1).SingleLine()

									if isInUse {
										ui.Row(c).Padding(1, 5).Radius(3).Border(1, t.Accent).Children(func() {
											ui.Text(c, a.text.S.AgInUse).FontSize(10).FontWeight(500).
												TextColor(t.Accent).SingleLine()
										})
									}

									// Context Window Badge (e.g. 1M, 1.0M, 256K, 192K)
									if ctxStr := ctxShort(e.Context); ctxStr != "" {
										ui.Text(c, ctxStr).FontSize(11.5).TextColor(t.TextMuted).
											Width(46).TextAlign(ui.End).SingleLine()
									}

									// Checkmark (blue check if enabled, blank slot if disabled)
									ui.Row(c).Width(18).Center().Children(func() {
										if !isDisabled {
											ui.Icon(c, checkIcon()).Size(13, 13).TextColor(ui.Hex("#007aff"))
										}
									})
								})
							}
						}
					})
				}

				if !renderedAny {
					ui.Row(c).Padding(24, 0).Center().Children(func() {
						ui.Text(c, a.text.S.NoData).FontSize(12).TextColor(t.TextMuted)
					})
				}
			})
		})
	})

	// Footer Bar (am-foot)
	ui.Row(c).FillWidth().Height(0.5).Background(lineC)
	ui.Row(c).AlignItems(ui.Center).FillWidth().Padding(6, 4, 0, 4).Children(func() {
		// Only Picked Switch
		onlyBox := ui.Row(c).AlignItems(ui.Center).Gap(8)
		onlyBox.Children(func() {
			sw := ui.Switch(c, &a.agModalOnly).Shrink(0)
			if sw.Clicked() {
				targetOnly := a.agModalOnly
				go func() {
					_ = provider.SetOnlyPicked(listsFor, targetOnly)
					a.Reload()
				}()
			}
			ui.Text(c, a.text.S.AgOnlyPicked).FontSize(12).TextColor(t.Text).SingleLine()
		})

		ui.Spacer(c)

		if a.agModalFilter == 2 {
			// Order mode: Reset order link
			resetBtn := ui.Row(c).AlignItems(ui.Center).Padding(4, 8).Radius(5).
				Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton)
			if resetBtn.Hovered() {
				resetBtn.Background(fieldBg)
			}
			if resetBtn.Clicked() {
				go func() {
					_ = provider.SetModelOrder(listsFor, nil)
					a.Reload()
				}()
			}
			resetBtn.Children(func() {
				ui.Text(c, a.text.S.AgDefaultOrder).FontSize(12).TextColor(t.TextMuted).SingleLine()
			})
		} else {
			// Hide all
			hideAllBtn := ui.Row(c).AlignItems(ui.Center).Padding(4, 6).Radius(4).
				Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton)
			if hideAllBtn.Hovered() {
				hideAllBtn.Background(fieldBg)
			}
			if hideAllBtn.Clicked() {
				a.disableAllAgentModels(ag.ID, ag.CurrentVal)
			}
			hideAllBtn.Children(func() {
				ui.Text(c, a.text.S.AgDisableAll).FontSize(12).TextColor(t.TextMuted).SingleLine()
			})

			ui.Text(c, "·").FontSize(12).TextColor(t.TextMuted).Margin(0, 4, 0, 4)

			// Show all
			showAllBtn := ui.Row(c).AlignItems(ui.Center).Padding(4, 6).Radius(4).
				Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton)
			if showAllBtn.Hovered() {
				showAllBtn.Background(fieldBg)
			}
			if showAllBtn.Clicked() {
				a.enableAllAgentModels(ag.ID)
			}
			showAllBtn.Children(func() {
				ui.Text(c, a.text.S.AgEnableAll).FontSize(12).TextColor(t.TextMuted).SingleLine()
			})
		}
	})
}

// reorderAgentByID moves fromID to the position of toID in a.data.Agents and updates user settings.
func (a *App) reorderAgentByID(fromID, toID string) {
	if fromID == toID || fromID == "" || toID == "" {
		return
	}
	var detected []AgentRow
	for _, ag := range a.data.Agents {
		if ag.Found {
			detected = append(detected, ag)
		}
	}
	fromIdx := -1
	toIdx := -1
	for i, ag := range detected {
		if ag.ID == fromID {
			fromIdx = i
		}
		if ag.ID == toID {
			toIdx = i
		}
	}
	if fromIdx < 0 || toIdx < 0 || fromIdx == toIdx {
		return
	}
	a.commitAgentReorder(fromIdx, toIdx)
}

// commitAgentReorder moves an agent in a.data.Agents and updates user settings.
func (a *App) commitAgentReorder(from, to int) {
	if from == to || from < 0 || to < 0 {
		return
	}
	var detected []AgentRow
	for _, ag := range a.data.Agents {
		if ag.Found {
			detected = append(detected, ag)
		}
	}
	if from >= len(detected) || to >= len(detected) {
		return
	}
	moved := detected[from]
	detected = append(detected[:from], detected[from+1:]...)
	detected = append(detected[:to], append([]AgentRow{moved}, detected[to:]...)...)

	order := make([]string, 0, len(detected))
	for _, ag := range detected {
		order = append(order, ag.ID)
	}

	var notFound []AgentRow
	for _, ag := range a.data.Agents {
		if !ag.Found {
			notFound = append(notFound, ag)
		}
	}
	a.data.Agents = append(detected, notFound...)

	s := settings.Load()
	s.AgentOrder = order
	_ = settings.Save(s)
	a.invalidate()
}

// moveAgent moves an agent relative to the list (top, bottom, up, down).
func (a *App) moveAgent(agID string, direction string) {
	var detected []AgentRow
	for _, ag := range a.data.Agents {
		if ag.Found {
			detected = append(detected, ag)
		}
	}
	idx := -1
	for i, ag := range detected {
		if ag.ID == agID {
			idx = i
			break
		}
	}
	if idx < 0 {
		return
	}
	target := idx
	switch direction {
	case "top":
		target = 0
	case "bottom":
		target = len(detected) - 1
	case "up":
		if target > 0 {
			target--
		}
	case "down":
		if target < len(detected)-1 {
			target++
		}
	}
	a.commitAgentReorder(idx, target)
}

// reorderAgentModel moves fromID to the position of toID in model order and updates settings.
func (a *App) reorderAgentModel(listsFor, fromID, toID string, shown []provider.Entry) {
	if fromID == toID || fromID == "" || toID == "" {
		return
	}
	var ids []string
	for _, e := range shown {
		ids = append(ids, e.ID)
	}
	fromIdx, toIdx := -1, -1
	for i, id := range ids {
		if id == fromID {
			fromIdx = i
		}
		if id == toID {
			toIdx = i
		}
	}
	if fromIdx < 0 || toIdx < 0 || fromIdx == toIdx {
		return
	}
	moved := ids[fromIdx]
	ids = append(ids[:fromIdx], ids[fromIdx+1:]...)
	newIDs := make([]string, 0, len(ids)+1)
	newIDs = append(newIDs, ids[:toIdx]...)
	newIDs = append(newIDs, moved)
	newIDs = append(newIDs, ids[toIdx:]...)
	go func() {
		_ = provider.SetModelOrder(listsFor, newIDs)
		a.Reload()
	}()
}

// toggleAgentModel adds or removes modelID from hidden models (or picked models) for agentID.
func (a *App) toggleAgentModel(agID, modelID string, enable bool) {
	listsFor := agID
	if agObj, err := agent.Find(agID); err == nil {
		listsFor = agObj.ListsFor()
	}
	pickedMap, only := provider.PickedModels(listsFor)
	if only {
		var newPicked []string
		for id := range pickedMap {
			if id != modelID {
				newPicked = append(newPicked, id)
			}
		}
		if enable {
			newPicked = append(newPicked, modelID)
		}
		go func(targetAgent string, targetPicked []string) {
			_ = provider.SetPickedModels(targetAgent, targetPicked)
			a.Reload()
		}(listsFor, newPicked)
		return
	}

	hiddenMap := provider.HiddenModels(listsFor)
	var newHidden []string
	if enable {
		for id := range hiddenMap {
			if id != modelID {
				newHidden = append(newHidden, id)
			}
		}
	} else {
		for id := range hiddenMap {
			newHidden = append(newHidden, id)
		}
		newHidden = append(newHidden, modelID)
	}
	go func(targetAgent string, targetHidden []string) {
		_ = provider.SetHiddenModels(targetAgent, targetHidden)
		a.Reload()
	}(listsFor, newHidden)
}

// enableAllAgentModels unhides all models for agentID.
func (a *App) enableAllAgentModels(agID string) {
	listsFor := agID
	if agObj, err := agent.Find(agID); err == nil {
		listsFor = agObj.ListsFor()
	}
	_, only := provider.PickedModels(listsFor)
	if only {
		listed, _ := provider.ListedFor(listsFor)
		var allIDs []string
		for _, e := range listed {
			allIDs = append(allIDs, e.ID)
		}
		go func(targetAgent string, targetPicked []string) {
			_ = provider.SetPickedModels(targetAgent, targetPicked)
			a.Reload()
		}(listsFor, allIDs)
		return
	}
	go func(targetAgent string) {
		_ = provider.SetHiddenModels(targetAgent, nil)
		a.Reload()
	}(listsFor)
}

// disableAllAgentModels hides all models for agentID except the active model.
func (a *App) disableAllAgentModels(agID, inUseID string) {
	listsFor := agID
	if agObj, err := agent.Find(agID); err == nil {
		listsFor = agObj.ListsFor()
	}
	listed, _ := provider.ListedFor(listsFor)
	_, only := provider.PickedModels(listsFor)
	if only {
		var newPicked []string
		if inUseID != "" {
			newPicked = append(newPicked, inUseID)
		}
		go func(targetAgent string, targetPicked []string) {
			_ = provider.SetPickedModels(targetAgent, targetPicked)
			a.Reload()
		}(listsFor, newPicked)
		return
	}
	var newHidden []string
	for _, e := range listed {
		if e.ID != inUseID && e.Model != inUseID {
			newHidden = append(newHidden, e.ID)
		}
	}
	go func(targetAgent string, targetHidden []string) {
		_ = provider.SetHiddenModels(targetAgent, targetHidden)
		a.Reload()
	}(listsFor, newHidden)
}

// agentsCardFooter draws the bottom fold/expand and manage strip of the card.
func (a *App) agentsCardFooter(c *ui.Context, t *ui.Theme, hiddenCount int, lineC ui.Color) {
	ui.Row(c).AlignItems(ui.Center).Padding(8, 16).
		BorderWidth(0.5, 0, 0, 0).BorderColor(lineC).Children(func() {
		ui.Spacer(c)

		foldBtn := ui.Row(c).AlignItems(ui.Center).Padding(3, 8).Radius(5).
			Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton)
		if foldBtn.Clicked() {
			a.agFolded = !a.agFolded
			a.invalidate()
		}
		foldBtn.Children(func() {
			label := a.text.S.AgShowLess
			if a.agFolded {
				label = strings.ReplaceAll(a.text.S.AgShowMore, "{n}", fmt.Sprint(hiddenCount))
			}
			ui.Text(c, label).FontSize(11.5).TextColor(t.TextMuted).SingleLine()
		})

		ui.Spacer(c)

		manageBtn := ui.Row(c).AlignItems(ui.Center).Padding(3, 8).Radius(5).
			Cursor(ui.CursorPointer).Children(func() {
			ui.Text(c, a.text.S.AgManage).FontSize(11.5).TextColor(t.TextMuted).SingleLine()
		})
		_ = manageBtn
	})
}

// agentsInstallSection draws the collapsible section for uninstalled agents.
func (a *App) agentsInstallSection(c *ui.Context, t *ui.Theme) {
	if len(a.data.AgentInstalls) == 0 {
		return
	}

	fieldBg := a.fieldBg(t)
	lineC := ui.RGBA(0, 0, 0, 0.07)
	if t.Dark {
		lineC = ui.RGBA(255, 255, 255, 0.07)
	}

	ui.Column(c).FillWidth().Margin(14, 0, 0, 0).Children(func() {
		head := ui.Row(c).AlignItems(ui.Center).Gap(5).
			Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton)
		if head.Clicked() {
			a.agInstallFolded = !a.agInstallFolded
			a.invalidate()
		}
		head.Children(func() {
			label := a.text.S.AgInstallFold
			if a.agInstallFolded {
				label = strings.ReplaceAll(a.text.S.AgInstallOthers, "{n}", fmt.Sprint(len(a.data.AgentInstalls)))
			}
			ui.Text(c, label).FontSize(12).FontWeight(500).TextColor(t.TextMuted).SingleLine()
		})

		if !a.agInstallFolded {
			box := ui.Column(c).FillWidth().Margin(8, 0, 0, 0).Radius(10).
				Background(t.Background).Border(1, t.Border)
			box.Children(func() {
				for i, inst := range a.data.AgentInstalls {
					inst := inst
					row := ui.Row(c).AlignItems(ui.Center).Gap(10).Padding(8, 14)
					if i > 0 {
						row.BorderWidth(0.5, 0, 0, 0).BorderColor(lineC)
					}
					row.Children(func() {
						pic := iconPictureFor(inst.Icon)
						ui.Image(c, pic).Size(22, 22).Radius(4)

						ui.Text(c, inst.Name).FontSize(12.5).FontWeight(600).
							Width(110).TextColor(t.Text).SingleLine()

						ui.Text(c, inst.Via).FontSize(11).
							TextColor(t.TextMuted).Width(95).SingleLine()

						// Command snippet box with fixed right copy button
						codeBox := ui.Row(c).AlignItems(ui.Center).Grow(1).Shrink(1).
							Padding(3, 8).Radius(6).Background(fieldBg).Gap(8)
						codeBox.Children(func() {
							ui.Text(c, inst.Command).Font("monospace").FontSize(11).
								TextColor(t.Text).Grow(1).Shrink(1).SingleLine()
							if b := a.copyBtn(c, t, inst.Command, inst.Name).Shrink(0); b.Clicked() {
								a.copyToClipboard(inst.Command)
							}
						})
					})
				}
			})
		}
	})
}

// agentsProfilesSection draws the profiles (方案) bar below the cards.
func (a *App) agentsProfilesSection(c *ui.Context, t *ui.Theme) {
	fieldBg := a.fieldBg(t)

	ui.Row(c).AlignItems(ui.Center).Gap(10).Margin(14, 0, 10, 0).Children(func() {
		ui.Text(c, a.text.S.AgScheme).FontSize(12).TextColor(t.TextMuted).SingleLine()

		for _, pf := range a.data.AgentProfiles {
			pf := pf
			pill := ui.Row(c).AlignItems(ui.Center).Gap(4).Padding(3, 10).
				Radius(6).Background(fieldBg).Border(1, t.Border).Cursor(ui.CursorPointer)
			if pf.InUse {
				pill.Background(t.Background).Border(1, t.Accent)
			}
			if pill.Clicked() {
				go func(name string) {
					if ps, err := profile.Load(); err == nil {
						if p, ok := ps[name]; ok {
							_, _ = profile.Apply(p)
							a.Reload()
						}
					}
				}(pf.Name)
			}
			pill.Children(func() {
				txtColor := t.Text
				if pf.InUse {
					txtColor = t.Accent
				}
				ui.Text(c, pf.Name).FontSize(11.5).FontWeight(500).TextColor(txtColor).SingleLine()
			})
		}

		ui.Spacer(c)

		saveBtn := ui.Row(c).AlignItems(ui.Center).Padding(3, 8).Radius(5).
			Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).
			Label(a.text.S.AgSaveCurrent)
		if saveBtn.Hovered() {
			saveBtn.Background(fieldBg)
		}
		if saveBtn.Clicked() {
			go func() {
				if p, err := profile.Snapshot(); err == nil {
					ps, _ := profile.Load()
					name := fmt.Sprintf("方案%d", len(ps)+1)
					_ = profile.Save(name, p)
					a.Reload()
				}
			}()
		}
		saveBtn.Children(func() {
			ui.Text(c, a.text.S.AgSaveCurrent).FontSize(11.5).TextColor(t.TextMuted).SingleLine()
		})
	})
}
