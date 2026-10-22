//go:build nogui && echo

package shell

// The Agents page's remote read (docs/08 M4): the rows, the uninstalled
// installs and the saved schemes from a magpie web instance's /api/state
// and /api/agents/install — the same documents the instance's own page
// draws. The row's words (its subtitle, its dropdown, its details line)
// are re-said here from the facts the state carries, with the same rules
// the embedded row keeps, because readOneAgentRow's cut runs over the
// domain's own Agent, which the document is not.
//
// What stays embedded even in remote mode:
//   - the agents' CLI versions: the row's version line is filled
//     asynchronously by loadCLIsAsync, which scans this machine's
//     binaries — the same machine the loopback instance's agents run
//     on, so the scan is the same fact either way.
//   - the roster rows (an agent magpie knows that isn't here): the
//     state names only the agents it found, and the roster of the known
//     ones is Echo's own, the same either way.
//   - the modal's model-list management (hide and unhide, the only-picked
//     switch, the drag order): the /api surface's POST /api/agent-models
//     carries hidden and order but nothing for the picked cut, and the
//     modal reads the provider domain's lists and the agent's values
//     directly. It stays embedded the way the Providers page keeps its
//     editor writes; the modal's field writes themselves go through the
//     seam below.
//   - the agents' drag order and the profiles' apply and save: saved
//     through the settings and profile packages, whose writes no
//     endpoint of the surface carries whole. The order the page draws
//     still comes from the instance's own settings document.
//   - Plugin and Notice on a row: the state document does not carry
//     them; a row reads without its plugin badge and its restart advice
//     until that cut is mapped.
//   - whether an agent's own menu is the list of models
//     (agent.ListsModels): not in the document either, so a wired
//     agent's subtitle names its count through the command it picks by,
//     or the menu when the command names none, and a row that lists by
//     name says the count in its details line.

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/yetone/magpie/internal/agent"
	"github.com/yetone/magpie/internal/magpieclient"
	"github.com/yetone/magpie/internal/profile"
	"github.com/yetone/magpie/internal/settings"
)

// agentStateDoc is the state document as the page reads it: the agents,
// the schemes and the instance's own settings' order. It is not the
// client's typed State, whose Agent carries only what the tray names —
// the row wants wired, drift and the model count too.
type agentStateDoc struct {
	Agents   []agentDoc   `json:"agents"`
	Profiles []profileDoc `json:"profiles"`
	Settings struct {
		AgentOrder []string `json:"agentOrder"`
	} `json:"settings"`
}

// agentDoc is one detected agent as the state document carries it.
type agentDoc struct {
	ID     string         `json:"id"`
	Name   string         `json:"name"`
	Icon   string         `json:"icon"`
	Path   string         `json:"path"`
	Wired  bool           `json:"wired,omitempty"`
	Drift  *agent.Drift   `json:"drift,omitempty"`
	Fields []fieldDoc     `json:"fields"`
	Models *modelCountDoc `json:"models,omitempty"`
}

// fieldDoc is one of the agent's pickers, its value as it reads now.
type fieldDoc struct {
	Key     string        `json:"key"`
	Label   string        `json:"label"`
	Value   string        `json:"value"`
	Options []agentOption `json:"options"`
}

// agentOption is one choice a picker offers, resolved by the instance.
type agentOption struct {
	Value  string `json:"value"`
	Label  string `json:"label,omitempty"`
	Note   string `json:"note"`
	Icon   string `json:"icon,omitempty"`
	Group  string `json:"group,omitempty"`
	Direct string `json:"direct,omitempty"`
}

// modelCountDoc is how many of the catalog the agent's lists show, with
// the first few by name and the providers' shares for the row's chips.
type modelCountDoc struct {
	Shown  int             `json:"shown"`
	Listed int             `json:"listed"`
	By     []modelGroupDoc `json:"by,omitempty"`
	Names  []string        `json:"names,omitempty"`
}

// modelGroupDoc is one provider's share of the models shown.
type modelGroupDoc struct {
	Name  string   `json:"name"`
	Icon  string   `json:"icon,omitempty"`
	Icons []string `json:"icons,omitempty"`
	N     int      `json:"n"`
}

// profileDoc is one saved scheme as the state document carries it: its
// name, its summary as magpie's own profile.Summary says it, and what it
// holds per agent.
type profileDoc struct {
	Name    string          `json:"name"`
	Summary string          `json:"summary"`
	Agents  []profile.Group `json:"agents"`
}

// readAgentsRemote builds the page's three lists from a magpie web
// instance: the rows from its /api/state, the uninstalled installs from
// its /api/agents/install (the same agent.Install documents the domain
// makes), the schemes from the state's profiles.
func readAgentsRemote(c *magpieclient.Client) ([]AgentRow, []AgentInstallRow, []ProfileRow, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	var doc agentStateDoc
	if err := c.Do(ctx, "GET", "/api/state", nil, &doc); err != nil {
		return nil, nil, nil, false
	}

	// the agents' values as they read now, for the schemes' in-use cut
	now := make(map[string]map[string]string, len(doc.Agents))
	rows := make([]AgentRow, 0, len(doc.Agents))
	for _, a := range doc.Agents {
		fields := make(map[string]string, len(a.Fields))
		for _, f := range a.Fields {
			fields[f.Key] = f.Value
		}
		now[a.ID] = fields
		rows = append(rows, agentRowOfDoc(a))
	}
	// in the order the instance's own settings keep them, its agents'
	// order and not this process's
	shown, hidden := settings.Arrange(settings.Settings{AgentOrder: doc.Settings.AgentOrder}, rows, func(r AgentRow) string { return r.ID })
	rows = append(shown, hidden...)

	var installs []agent.Install
	if err := c.Do(ctx, "GET", "/api/agents/install", nil, &installs); err != nil {
		return nil, nil, nil, false
	}
	installRows := make([]AgentInstallRow, 0, len(installs))
	for _, inst := range installs {
		installRows = append(installRows, agentInstallRowOf(inst))
	}

	profileRows := make([]ProfileRow, 0, len(doc.Profiles))
	for _, p := range doc.Profiles {
		profileRows = append(profileRows, ProfileRow{
			Name:    p.Name,
			Summary: p.Summary,
			InUse:   profileInUseOfDoc(p, now),
		})
	}
	return rows, installRows, profileRows, true
}

// agentRowOfDoc says the row's words the way the embedded readOneAgentRow
// says them, from the document's facts: the same branches, over the
// state's wired, drift, fields and model count.
func agentRowOfDoc(a agentDoc) AgentRow {
	drift := a.Drift != nil
	driftText := ""
	if drift {
		driftText = fmt.Sprintf("%s 已不经过 Echo: 配置已变更", a.Name)
	}

	shown := 0
	var names []string
	var groups []AgentGroupChip
	if mc := a.Models; mc != nil {
		shown = mc.Shown
		names = mc.Names
		for _, g := range mc.By {
			groups = append(groups, AgentGroupChip{Name: g.Name, Icon: g.Icon, Icons: g.Icons, Count: g.N})
		}
	}

	var subtitle string
	if drift {
		subtitle = driftText
	} else if a.Wired {
		cmd := picksIn[a.ID]
		switch {
		case a.ID == "claude":
			subtitle = "已接入 · 新开的会话生效"
		case a.ID == "cursor" || a.ID == "cursor-cli":
			subtitle = fmt.Sprintf("已接入 · Echo 的模型在 %s 的模型列表里", a.Name)
		case cmd != "" && shown > 0:
			subtitle = fmt.Sprintf("已接入 · %s 的 %s 里有 %d 个模型", a.Name, cmd, shown)
		case shown > 0:
			subtitle = fmt.Sprintf("已接入 · %s 的模型菜单里有 %d 个模型", a.Name, shown)
		default:
			subtitle = fmt.Sprintf("已接入 · %s 的模型列表已同步", a.Name)
		}
	} else {
		subtitle = fmt.Sprintf("未接入 · %s 使用其自身设置", a.Name)
	}

	detailsText := "详情 ›"
	if a.ID == "claude" {
		detailsText = "档位 ›"
	} else if shown > 0 {
		detailsText = fmt.Sprintf("模型 %d ›", shown)
	}

	fieldKey := "model"
	var currentVal string
	var currentName string
	var currentIcon string
	var options []AgentModelOpt

	var targetField *fieldDoc
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
		currentVal = targetField.Value
		for _, o := range targetField.Options {
			selected := o.Value == currentVal
			opt := AgentModelOpt{
				ID:       o.Value,
				Label:    cmp.Or(o.Label, o.Value),
				Icon:     o.Icon,
				Selected: selected,
			}
			options = append(options, opt)
			if selected {
				currentName = opt.Label
				currentIcon = opt.Icon
			}
		}
	}

	effort := ""
	for i := range a.Fields {
		f := &a.Fields[i]
		if f.Key == "effort" || f.Label == "effort" || f.Label == "thinking" {
			effort = f.Value
			break
		}
	}

	// the words on the row's model dropdown: the model an option names,
	// else the models the document says the agent's lists show, else what
	// is set, else the default — the embedded read's rule, without the
	// lists-by-name arm the document does not carry
	switch {
	case len(names) > 0 && currentName == "":
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

	return AgentRow{
		ID:          a.ID,
		Name:        a.Name,
		Icon:        a.Icon,
		Found:       true,
		Path:        a.Path,
		Wired:       a.Wired,
		Drift:       drift,
		DriftText:   driftText,
		Subtitle:    subtitle,
		ModelCount:  shown,
		DetailsText: detailsText,
		FieldKey:    fieldKey,
		CurrentVal:  currentVal,
		CurrentName: currentName,
		CurrentIcon: currentIcon,
		Effort:      effort,
		ModelNames:  names,
		Options:     options,
		Groups:      groups,
	}
}

// profileInUseOfDoc is the in-use cut the state document allows: the
// profile's saved fields against the agents' values as they read now.
// The model lists and the library's setup a profile also holds are not
// in the document, so a profile held apart by those alone reads in use
// when its fields alone say so.
func profileInUseOfDoc(p profileDoc, now map[string]map[string]string) bool {
	compared := false
	for _, g := range p.Agents {
		for _, it := range g.Fields {
			// a value that reads as a key is left out of the document,
			// and a field that follows another is said by what it follows
			if it.Hidden || it.Follows != "" {
				continue
			}
			cur, ok := now[g.ID]
			if !ok {
				continue
			}
			v, ok := cur[it.Key]
			if !ok {
				continue
			}
			if v != it.Value {
				return false
			}
			compared = true
		}
	}
	return compared
}

// agentRescan looks for the agents on this machine again, as the page's
// refresh button does.
//
// The experiment's seam (docs/08 M4): when the environment points at a
// magpie web instance the rescan is asked of it; any remote failure falls
// back to the embedded rescan below.
func agentRescan() {
	if c := remoteGateway(); c != nil {
		if _, err := c.Rescan(context.Background()); err == nil {
			return
		}
	}
	agent.Rescan()
}

// agentSet writes one of an agent's fields, as the modal's picker does —
// the instance's own /api/set, whose answer is the state to adopt.
//
// The experiment's seam (docs/08 M4): when the environment points at a
// magpie web instance the set is asked of it; any remote failure falls
// back to the embedded set below.
func agentSet(id, field, value string) error {
	if c := remoteGateway(); c != nil {
		if _, err := c.Set(context.Background(), id, field, value); err == nil {
			return nil
		}
	}
	a, err := agent.Find(id)
	if err != nil {
		return err
	}
	f := a.Field(field)
	if f == nil {
		return fmt.Errorf("unknown field %q on %s", field, id)
	}
	// the modal reloads whatever the set said
	_ = f.Set(value)
	return nil
}

// agentConnect wires an agent to the app or takes it out, as the row's
// switch does — the instance's own connect and disconnect actions.
//
// The experiment's seam (docs/08 M4): when the environment points at a
// magpie web instance the switch is asked of it; any remote failure falls
// back to the embedded switch below.
func agentConnect(id string, on bool) error {
	if c := remoteGateway(); c != nil {
		act := "disconnect"
		if on {
			act = "connect"
		}
		// the page posts an empty object, which the handler's decode wants
		if err := c.Do(context.Background(), "POST", "/api/agents/"+act+"/"+id, map[string]string{}, nil); err == nil {
			return nil
		}
	}
	a, err := agent.Find(id)
	if err != nil {
		return err
	}
	if on {
		_, _ = a.ConnectHow()
	} else {
		_ = a.Disconnect()
	}
	return nil
}

// agentReapply sets again what the app set on an agent something else
// rewrote, as the row's drift pill does: the instance says what drifted
// (/api/drift), and setting that field again is its own /api/set.
//
// The experiment's seam (docs/08 M4): when the environment points at a
// magpie web instance the reapply is asked of it; any remote failure falls
// back to the embedded reapply below. An agent of the instance's that
// drifts no more is done there — reapplying the embedded agent instead
// would write this process's copy of the config.
func agentReapply(id string) error {
	if c := remoteGateway(); c != nil {
		drifts, err := c.Drift(context.Background())
		if err == nil {
			raw, drifted := drifts[id]
			if !drifted {
				return nil
			}
			var d agent.Drift
			if jerr := json.Unmarshal(raw, &d); jerr == nil && d.Field != "" {
				if _, serr := c.Set(context.Background(), id, d.Field, d.Want); serr == nil {
					return nil
				}
			}
		}
	}
	a, err := agent.Find(id)
	if err != nil {
		return err
	}
	_ = a.Reapply()
	return nil
}
