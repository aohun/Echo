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
//   - Totals and Traces: the routing history. The traffic goes through
//     the instance's gateway and Echo's embedded one saw none of it;
//     docs/08 M4 keeps the history read local until the trace route
//     (/api/gateway/trace) is mapped.
//   - ClassifierErr: its source is the serving process's classifier
//     failure record, which the document does not carry.
//   - ProviderOff on a member: the document says only ready or not; a
//     member whose provider is off reads not ready until that cut is
//     mapped.

import (
	"context"
	"strings"
	"time"

	"github.com/yetone/magpie/internal/agent"
	"github.com/yetone/magpie/internal/magpieclient"
	"github.com/yetone/magpie/internal/provider"
)

func readRoutingRemote(c *magpieclient.Client) (RoutingData, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	doc, err := c.Groups(ctx)
	if err != nil {
		return RoutingData{}, false
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
