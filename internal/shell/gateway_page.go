//go:build nogui && echo

package shell

import (
	"fmt"
	"strings"
	"time"

	"github.com/yetone/magpie/internal/agent"
	"github.com/yetone/magpie/internal/gateway"
	"github.com/yetone/magpie/internal/gui"
	"github.com/yetone/magpie/internal/provider"
	"github.com/yetone/magpie/internal/update"
)

// The Gateway page's own reads: what its card says, the models it lists,
// the ways to connect and the calls that came in. Everything is magpie's
// domain code (gateway, provider) seen through the shell's lens, read off
// the main thread the way the other pages' reads are.

// GatewayRow is the Gateway page's own state: the status card, the models,
// the connect sheet and the recent calls.
//
// It is read as a whole (readGateway) because the four draw together: the
// card's URL is the connect sheet's base, its running state decides
// whether the models and the calls are worth listing at all.
type GatewayRow struct {
	// URL is where the gateway is reached, Running whether it answers
	// there, and Mine whether this process is the one serving it.
	URL     string
	Running bool
	Mine    bool
	// Version is another magpie's, serving it when we are not, Older
	// whether it is older than this one.
	Version string
	Older   bool
	// Models is how many the gateway serves to the agents.
	Models int
	// Routed is how many detected agents are pointed at a model or a
	// routing group of ours now.
	Routed int

	// Models exposed to the agents, groups first then the providers',
	// as the agents' pickers list them (magpie's gatewayModels).
	Entries []GatewayModel
	// Calls are the requests the gateway handled, newest first
	// (gateway.Server.Recent), those this process served.
	Calls []GatewayCall
}

// GatewayModel is one model as the Gateway page lists it: the id an agent
// names, and what there is to say of it.
type GatewayModel struct {
	// ID is what an agent sends magpie for it: "provider/model", or a
	// routing group's "group/<id>".
	ID string
	// Name is the model's display name, when it has one of its own.
	Name string
	// Provider is whose it is — the vendor's or the group's members'.
	Provider string
	// Group is whether it is a routing group rather than a vendor's model.
	Group bool
	// Icon is the provider's picture, a built-in name or "file:<name>".
	Icon string
	// Context and Output are the tokens a prompt and a reply may hold,
	// when known; Efforts the reasoning levels it offers.
	Context, Output int
	Efforts         []string
	// Images is whether it takes images as input.
	Images bool
	// Free is whether its subscription serves it at no cost.
	Free bool
}

// GatewayCall is one request the gateway handled.
type GatewayCall struct {
	Time  time.Time
	Agent string
	Model string
	// From and To are the protocols it came in and went out as; when they
	// are the same only one is shown.
	From, To string
	Status   int
	Millis   int64
	// TTFT is ms from the request to the reply's first content.
	TTFT  int64
	Error string
	// Kind is what the call was for when it wasn't the agent's own turn.
	Kind string
}

// readGateway reads what the Gateway page draws.
//
// The gateway's own state is read from the server this process serves when
// there is one (Server.Recent), and from the address otherwise
// (gateway.ServedBy) — the same cut magpie's own providersJSON makes, where
// a gateway another magpie serves shows that one's version and no calls of
// its own.
func readGateway() GatewayRow {
	g := GatewayRow{URL: gateway.URL()}
	cat := provider.Catalog()
	g.Models = len(cat)
	g.Routed = routedAgents()
	g.Entries = readGatewayModels(cat)
	if gw := guiServed(); gw != nil {
		g.Running, g.Mine = true, true
		for _, c := range gw.Recent() {
			g.Calls = append(g.Calls, gatewayCall(c))
		}
		return g
	}
	s := gateway.ServedBy()
	g.Running, g.Version, g.Older = s.Running, s.Version, s.Running && guiOlder(s.Version)
	return g
}

// guiServed is the gateway this process serves, or nil.
//
// It goes through gui.ServedServer rather than a copy of that pointer, so
// what the Gateway page shows is what the app is actually serving.
func guiServed() *gateway.Server { return gui.ServedServer() }

// guiOlder is whether a version serving the gateway is older than this
// one — magpie's own rule (update.Newer), read through gui so the shell
// keeps no copy of it.
func guiOlder(v string) bool {
	return v != "" && update.Newer(gateway.Version, v)
}

// routedAgents is how many detected agents are on one of our models or
// routing groups now: the same count magpie's status card gives.
func routedAgents() int {
	seen := map[string]bool{}
	for _, u := range agentUses(agent.Detected()) {
		switch {
		case u.inGroup:
			seen[u.agent.ID] = true
		case u.pid != "":
			seen[u.agent.ID] = true
		}
	}
	return len(seen)
}

// readGatewayModels lists what the gateway serves to the agents: the
// routing groups first, then every exposed model of every provider not kept
// unlisted, as magpie's gatewayModels does over provider.Catalog.
//
// A provider switched off has none in the catalog, so its models are left
// out here too — the gateway refuses them, so they are not ids an agent can
// use.
func readGatewayModels(cat []provider.Entry) []GatewayModel {
	out := make([]GatewayModel, 0, len(cat))
	for _, e := range cat {
		m := GatewayModel{
			ID:       e.ID,
			Name:     e.Name,
			Provider: e.Provider.Name,
			Group:    e.Group != "",
			Icon:     e.Provider.Icon,
			Context:  e.Context,
			Output:   e.Output,
			Efforts:  e.Efforts,
			Images:   e.Images,
			Free:     e.Free,
		}
		if m.Provider == "" {
			// a group with no name of its own is named by its members
			m.Provider = "routing group"
		}
		out = append(out, m)
	}
	return out
}

// gatewayCall is a gateway.Call as the page's row draws it.
func gatewayCall(c gateway.Call) GatewayCall {
	return GatewayCall{
		Time:   c.Time,
		Agent:  c.Agent,
		Model:  c.Model,
		From:   string(c.From),
		To:     string(c.To),
		Status: c.Status,
		Millis: c.Millis,
		TTFT:   c.TTFT,
		Error:  c.Error,
		Kind:   c.Kind,
	}
}

// gatewayOlder is whether a version serving the gateway is older than this
// one — magpie's own rule (update.Newer), read through gui so the shell
// keeps no copy of it.
func gatewayOlder(v string) bool { return v != "" && guiOlder(v) }

// gatewayTook is how long a call took, in the page's own words: seconds
// with the milliseconds it ran to when they say more.
func gatewayTook(ms int64) string {
	switch {
	case ms < 1000:
		return fmt.Sprintf("%d ms", ms)
	case ms < 60_000:
		return fmt.Sprintf("%.1fs", float64(ms)/1000)
	default:
		return fmt.Sprintf("%dm %02ds", ms/60_000, (ms/1000)%60)
	}
}

// gatewayWhen is when a call was, as the clock reads: a moment ago, then
// the time of day.
func gatewayWhen(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	}
	return t.Format("15:04:05")
}

// gatewayRoute is the protocols a call went through: one, when it came in
// and went out the same way, else the change.
func gatewayRoute(from, to string) string {
	if from == to || to == "" {
		return from
	}
	if from == "" {
		return to
	}
	return from + " → " + to
}

// gatewaySub is the model's own line under its id: its name when it has
// one of its own, and whose it is.
func gatewaySub(m GatewayModel) string {
	bare := m.ID
	if _, after, ok := strings.Cut(bare, "/"); ok {
		bare = after
	}
	var parts []string
	if m.Name != "" && m.Name != bare {
		parts = append(parts, m.Name)
	}
	if m.Provider != "" {
		parts = append(parts, m.Provider)
	}
	return strings.Join(parts, " · ")
}
