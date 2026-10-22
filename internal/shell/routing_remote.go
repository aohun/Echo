//go:build nogui && echo

package shell

// The Routing page's remote read (docs/08 M4): the groups, the models
// they can be made of, the pools and the deciders from a magpie web
// instance's /api/groups, whose groupsJSON is the page's own document —
// the group carried as it is, each member's ready, name and count of
// keys or accounts on cut for the row.
//
// What stays embedded even in remote mode:
//   - Templates: the quick-setup templates are built-ins, no HTTP
//     endpoint serves them and none is needed; the embedded read's list
//     is the same either way.
//   - ClassifierErr: its source is the serving process's classifier
//     failure record, which the document does not carry.
//   - Session titles of non-codex agents in the history: the instance
//     prices and titles the codex sessions it names; other agents'
//     titles are this machine's session files, which the document does
//     not carry.

import (
	"context"
	"strings"
	"time"

	"github.com/yetone/magpie/internal/agent"
	"github.com/yetone/magpie/internal/magpieclient"
	"github.com/yetone/magpie/internal/provider"
)

// provName is what the providers document says of one provider, for the
// routes' and the members' rows: its name, its icon, whether it is off.
type provName struct {
	name string
	icon string
	off  bool
}

func readRoutingRemote(c *magpieclient.Client) (RoutingData, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	doc, err := c.Groups(ctx)
	if err != nil {
		return RoutingData{}, false
	}
	// The providers document rides along: the routes' and the members'
	// rows name each provider, and a member's provider-off says so there.
	pdoc, err := c.Providers(ctx)
	if err != nil {
		return RoutingData{}, false
	}
	names := make(map[string]provName, len(pdoc.Providers))
	for _, p := range pdoc.Providers {
		names[p.ID] = provName{name: p.Name, icon: p.Icon, off: p.Off}
	}
	rd := RoutingData{
		AutoGroupsOn: doc.Found,
		Templates:    routingTemplates(),
	}
	for _, m := range doc.Models {
		rd.AvailableModels = append(rd.AvailableModels, RoutingModelChoice{
			ID:           m.ID,
			Name:         m.Name,
			ProviderID:   m.Provider,
			ProviderName: m.PName,
			Icon:         m.Icon,
			Context:      m.Context,
			Efforts:      m.Efforts,
			CanFast:      m.CanFast,
			Images:       m.Images,
		})
	}
	for _, d := range doc.Deciders {
		rd.DeciderModels = append(rd.DeciderModels, RoutingModelChoice{
			ID:           d.ID,
			Name:         d.Name,
			ProviderID:   d.Provider,
			ProviderName: d.PName,
			Icon:         d.Icon,
		})
	}
	for _, g := range doc.Groups {
		grow := RoutingGroupRow{
			ID:         g.ID,
			Name:       g.Name,
			Routing:    g.Routing,
			Affinity:   g.Affinity,
			Auto:       g.Auto,
			Disabled:   g.Disabled,
			FirstToken: g.FirstToken,
			Patience:   g.Patience,
			Pick:       g.Picked,
			RulesCount: len(g.Rules),
			Classifier: g.Classifier,
			Context:    g.Context,
			Weights:    g.Weights,
			Patterns:   g.Match,
			Ready:      g.Ready,
		}
		for _, m := range g.Info {
			mrow := RoutingMemberRow{
				ID:           m.ID,
				ProviderID:   m.Provider,
				ProviderName: m.Name,
				Icon:         m.Icon,
				Effort:       m.Effort,
				Fast:         m.Fast,
				CanFast:      m.CanFast,
				Weight:       g.Weights[m.ID],
				Ready:        m.Ready,
				IsGroup:      m.Group,
				AccountsOn:   m.On,
				Context:      m.Context,
				Efforts:      m.Efforts,
			}
			if m.Group {
				// a group in the group: the row names it by its id
				// without the prefix, as the embedded read does
				mrow.Model = strings.TrimPrefix(m.ID, provider.GroupPrefix)
				if mrow.ProviderName == "" {
					mrow.ProviderName = mrow.Model
				}
			} else if m.Ready {
				mrow.Model = m.Model
			} else {
				mrow.Model = m.Of
				// not ready and not resolved: often its provider is
				// switched off, which the providers document says
				if pid, model, ok := strings.Cut(m.Of, "/"); ok {
					if n, found := names[pid]; found && n.off {
						mrow.ProviderOff = true
						mrow.ProviderID = pid
						mrow.ProviderName = n.name
						mrow.Icon = n.icon
						mrow.Model = model
					}
				}
			}
			grow.Members = append(grow.Members, mrow)
		}
		rd.Groups = append(rd.Groups, grow)
	}
	for _, p := range doc.Pools {
		rd.Pools = append(rd.Pools, RoutingPoolRow{
			ProviderID:   p.Provider,
			ProviderName: p.Name,
			Icon:         p.Icon,
			Kind:         p.Kind,
			Who:          p.Who,
			Routing:      p.Routing,
			Affinity:     p.Affinity,
			Sink:         p.Sink,
		})
	}

	// The routing history: the instance's gateway saw the traffic, and
	// /api/gateway/history answers its trace of it — the same routes the
	// embedded gateway.History("") would name, priced and titled by the
	// instance.
	hist, err := c.GatewayHistory(ctx, "")
	if err != nil {
		return RoutingData{}, false
	}
	for _, r := range hist.Routes {
		rd.Totals.Requests++
		if len(r.Tries) > 1 {
			rd.Totals.Rerouted++
		}
		if r.Error != "" || (r.Status >= 400 && r.Status != 0) {
			rd.Totals.Errors++
		}
	}
	// Up to 60 most recent, newest first, as the embedded read keeps.
	for i := len(hist.Routes) - 1; i >= 0 && len(rd.Traces) < 60; i-- {
		r := hist.Routes[i].Route
		var cacheRead, cacheWrite int
		for _, u := range r.Usage {
			cacheRead += u.CacheRead
			cacheWrite += u.CacheWrite
		}
		servedModel := r.Served
		if servedModel == "" && len(r.Tries) > 0 {
			servedModel = r.Tries[len(r.Tries)-1].Model
		}
		pName, pIcon := r.Provider, ""
		if n, ok := names[r.Provider]; ok {
			pName, pIcon = n.name, n.icon
		}
		var tries []RoutingTryRow
		for _, tr := range r.Tries {
			tries = append(tries, RoutingTryRow{
				Who:      tr.ID,
				Provider: tr.ID,
				Model:    tr.Model,
				Status:   tr.Status,
				Millis:   tr.Millis,
			})
		}
		rd.Traces = append(rd.Traces, RoutingTraceRow{
			ID:               r.ID,
			Time:             r.Time,
			Agent:            r.Agent,
			Session:          r.Session,
			SessionTitle:     hist.Routes[i].SessionTitle,
			RequestedModel:   r.Model,
			ServedModel:      servedModel,
			ProviderID:       r.Provider,
			ProviderName:     pName,
			ProviderIcon:     pIcon,
			Status:           r.Status,
			Done:             r.Done,
			Error:            r.Error,
			Millis:           r.Millis,
			TTFT:             r.TTFT,
			Cost:             hist.Routes[i].Cost,
			Tokens:           r.Tokens,
			OutputTokens:     r.Output,
			ReasoningTokens:  r.Reasoning,
			CacheReadTokens:  cacheRead,
			CacheWriteTokens: cacheWrite,
			Decision:         rtDecision(r),
			Tries:            tries,
		})
	}
	return rd, true
}

// routingFound turns auto groups on or off, as the page's toggle does —
// magpie's own found action, whose agents Reseat moves off found groups.
//
// The experiment's seam (docs/08 M4): when the environment points at a
// magpie web instance the toggle is asked of it; any remote failure falls
// back to the embedded toggle below.
func routingFound(on bool) error {
	if c := remoteGateway(); c != nil {
		if _, err := c.GroupsFound(context.Background(), on); err == nil {
			return nil
		}
	}
	_, err := agent.Reseat(func() error { return provider.SetAutoGroups(on) })
	return err
}

// routingSwitch switches one of the user's groups on or off, kept as it
// is, as the row's switch does.
//
// The experiment's seam (docs/08 M4): when the environment points at a
// magpie web instance the switch is asked of it; any remote failure falls
// back to the embedded switch below.
func routingSwitch(id string, on bool) error {
	if c := remoteGateway(); c != nil {
		if _, err := c.GroupsSwitch(context.Background(), id, on); err == nil {
			return nil
		}
	}
	return provider.SwitchGroup(id, on)
}

// routingDelete deletes one group, as the row menu's confirmed delete
// does.
//
// The experiment's seam (docs/08 M4): when the environment points at a
// magpie web instance the delete is asked of it; any remote failure falls
// back to the embedded delete below.
func routingDelete(id string) error {
	if c := remoteGateway(); c != nil {
		if _, err := c.GroupsDelete(context.Background(), []string{id}); err == nil {
			return nil
		}
	}
	return provider.DeleteGroup(id)
}

// routingSaveGroup saves the group as the editor's save does, renaming it
// when from names the id it had — magpie's own save action, whose server
// side saves under the old id and renames.
//
// The experiment's seam (docs/08 M4): when the environment points at a
// magpie web instance the save is asked of it; any remote failure falls
// back to the embedded save below.
func routingSaveGroup(g provider.Group, from string) error {
	if c := remoteGateway(); c != nil {
		if _, err := c.GroupsSave(context.Background(), g, from); err == nil {
			return nil
		}
	}
	if from == "" || from == g.ID {
		return provider.SaveGroup(g)
	}
	to := g.ID
	g.ID = from
	if err := provider.SaveGroup(g); err != nil {
		return err
	}
	return provider.RenameGroup(from, to)
}
