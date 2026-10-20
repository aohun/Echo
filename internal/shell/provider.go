//go:build nogui && echo

package shell

import (
	"cmp"
	"strings"

	"github.com/yetone/magpie/internal/agent"
	"github.com/yetone/magpie/internal/gateway"
	"github.com/yetone/magpie/internal/provider"
)

// The Providers page's own reads and writes: what a row draws, and what a
// click on it does. Everything here is magpie's domain code (provider,
// agent, gateway, catalog) seen through the shell's lens — the page's data
// is read once off the main thread (readProviders), and a click acts on the
// domain directly, the way magpie's own page asks its API to.

// ProviderRow is one vendor as the Providers page lists it: a row's worth
// of provider.Provider, plain values the view draws without touching the
// filesystem again.
type ProviderRow struct {
	// ID and Name are the provider's; Name is what the row is titled.
	ID, Name string
	// Icon is the provider's picture: a built-in name ("deepseek-color"),
	// "file:<name>" for a picture the user stored, or "" (the generic mark).
	Icon string
	// Host is the vendor's API host, as Where keeps it for usage.
	Host string
	// Preset is the preset it was made from, "" for a custom one.
	Preset string
	// Models is how many models are exposed to the agents now, and Draws
	// how many image models are kept apart from them.
	Models, Draws int
	// Key is the saved key, masked for display, and whether one is saved.
	Key   string
	Keyed bool
	// Ready is whether the vendor can be used: keyed, needing no key, or
	// signed in. NeedsKey is what a row without a key says: true when one
	// is wanted ("needs a key"), false when none is ("no key").
	Ready    bool
	NeedsKey bool
	// Account is the signed-in plan, when there is one: the pill's text
	// (magpie's accountPlan), the account's user, and whether every login
	// in use was refused by its vendor (signed out).
	Plan     string
	User     string
	SignedIn bool
	Lapsed   bool
	// Agents is what is using this vendor now: the icons of the detected
	// agents on one of its models, each with the model and routing group
	// the row's tile names.
	Agents []AgentUse
	// Off is whether it is switched off (provider.SetOff), and Sponsored
	// whether its preset says magpie's makers are paid for it.
	Off       bool
	Sponsored bool
}

// AgentUse is one agent on this vendor now, as the row's using tile is.
type AgentUse struct {
	// Icon is the agent's bundled icon name.
	Icon string
	// Name is the agent's, for the tile's description.
	Name string
	// Model is what it is on there, and Group the routing group it goes
	// through, when it is on one of those.
	Model, Group string
}

// agentUseOf is what an agent is on now: read once for the page, not again
// for each provider (magpie's agentUses). It is the same cut its
// providerInfo makes of an agent's first field.
type agentUse struct {
	agent *agent.Agent
	pid   string // the provider it is on, "" for a group or its own
	model string
	// group, members and inGroup: on a routing group, one of whose
	// members this vendor may be
	group   provider.Group
	members []provider.Member
	inGroup bool
}

// agentUses reads what every detected agent is on.
func agentUses(agents []*agent.Agent) []agentUse {
	findGroup := provider.GroupFinder()
	out := make([]agentUse, 0, len(agents))
	for _, a := range agents {
		u := agentUse{agent: a}
		var v string
		if len(a.Fields) > 0 {
			v = strings.TrimPrefix(a.Fields[0].Get(), "magpie/")
		}
		switch {
		case strings.HasPrefix(v, provider.GroupPrefix):
			u.group, u.members, u.inGroup = findGroup(v)
		case v != "":
			if pid, model, ok := strings.Cut(v, "/"); ok {
				if _, err := provider.Find(pid); err == nil {
					u.pid, u.model = pid, model
				}
			}
		}
		out = append(out, u)
	}
	return out
}

// readProviders lists every configured provider as its row draws, in the
// order they are tried — magpie's own order, on ones first.
func readProviders() []ProviderRow {
	all := provider.All()
	uses := agentUses(agent.Detected())
	rows := make([]ProviderRow, 0, len(all))
	for _, p := range all {
		r := ProviderRow{
			ID:       p.ID,
			Name:     p.Name,
			Icon:     p.Icon,
			Host:     p.Host(),
			Preset:   p.Preset,
			Keyed:    p.Key != "",
			Ready:    p.Ready(),
			SignedIn: p.Account != nil,
			Off:      p.Off,
		}
		if a := p.Account; a != nil {
			r.User = a.User
			r.Plan = accountPlan(a.Agent, a.Plan)
			// every account in use refused by its vendor: signed out, as
			// the accounts list says of each, not a green "signed in"
			inUse, lapsed := 0, 0
			for _, l := range provider.Logins(a.Agent) {
				if l.On || l.Active {
					inUse++
					if l.Lapsed != "" {
						lapsed++
					}
				}
			}
			r.Lapsed = inUse > 0 && lapsed == inUse
		} else {
			r.Key = provider.Mask(p.Key)
			r.NeedsKey = !p.Ready()
		}
		// a provider switched off serves nothing: its count is its picks',
		// not the catalog's, which nothing asked it for
		if p.Off {
			r.Models = len(p.Models)
		} else {
			r.Models = len(p.Exposed())
			r.Draws = len(gateway.Drawers(p))
		}
		// the agents on it now: on one of its models, or on a routing
		// group one of whose members it is (providerInfo's own rule)
		for _, u := range uses {
			var (
				current bool
				model   string
				group   string
			)
			if u.inGroup {
				for _, m := range u.members {
					if m.Provider.ID == p.ID {
						current, model, group = true, m.Model, u.group.Name
						break
					}
				}
			} else if u.pid == p.ID {
				current, model = true, u.model
			}
			if current {
				r.Agents = append(r.Agents, AgentUse{
					Icon:  u.agent.Icon,
					Name:  u.agent.Name,
					Model: model,
					Group: group,
				})
			}
		}
		if pr := provider.Preset(p.Preset); pr != nil {
			r.Sponsored = pr.Sponsored
		}
		rows = append(rows, r)
	}
	return rows
}

// accountPlan is magpie's own accountPlan (app.js): the pill naming the
// vendor and plan, the plan alone when it already names the vendor.
func accountPlan(agentID, plan string) string {
	capitalize := func(s string) string {
		if s == "" {
			return s
		}
		return strings.ToUpper(s[:1]) + s[1:]
	}
	named := func(name, plan string) string {
		if plan == "" {
			return name
		}
		w := strings.Fields(name)
		low := strings.ToLower(plan) + " "
		for i := range w {
			if strings.HasPrefix(low, strings.ToLower(strings.Join(w[i:], " "))+" ") {
				return strings.Join(append(w[:i:i], plan), " ")
			}
		}
		return name + " " + plan
	}
	switch agentID {
	case "codex", "chatgpt-api":
		return named("ChatGPT", capitalize(plan))
	case "copilot":
		return named("GitHub", capitalize(plan))
	case "claude":
		return named("Claude", capitalize(plan))
	case "cursor":
		return named("Cursor", capitalize(plan))
	case "grok":
		return cmp.Or(plan, "SuperGrok")
	case "gemini", "antigravity":
		return cmp.Or(plan, "Google")
	case "zcode":
		return cmp.Or(plan, "GLM Coding Plan")
	case "workbuddy":
		return cmp.Or(plan, "WorkBuddy")
	case "workbuddy-ai":
		return cmp.Or(plan, "WorkBuddy AI")
	case "commandcode-plan":
		return named("Command Code", plan)
	case "qoder":
		return named("Qoder", plan)
	case "qoder-cn":
		return named("Qoder CN", plan)
	case "zed":
		return named("Zed", plan)
	case "mimo-app":
		return named("Xiaomi MiMo", plan)
	}
	if plan != "" {
		return plan
	}
	return "signed in"
}

// SwitchProvider switches a provider off or on, as the row's switch does.
//
// It is magpie's own switchProvider (provider.SetOff through agent.Reseat):
// the agents on one of its models are moved to others, their files written.
func SwitchProvider(id string, on bool) error {
	_, err := agent.Reseat(func() error { return provider.SetOff(id, !on) })
	return err
}

// ArrangeProvider puts the provider at index to among the ones on, the ones
// switched off keeping their places among the rest — the same order
// magpie's arrange endpoint saves (provider.SetOrder).
func ArrangeProvider(id string, to int) error {
	all := provider.All()
	var on []string
	for _, p := range all {
		if !p.Off {
			on = append(on, p.ID)
		}
	}
	from := -1
	for i, x := range on {
		if x == id {
			from = i
			break
		}
	}
	if from < 0 || to < 0 || to >= len(on) || to == from {
		return nil // nothing to do, as magpie's own arrangeProvider says
	}
	// the move itself: take it out, put it where it goes
	out := make([]string, 0, len(on))
	out = append(out, on[:from]...)
	out = append(out, on[from+1:]...)
	out = append(out[:to], append([]string{id}, out[to:]...)...)
	// the order names every provider: the ones off keep their places
	order := make([]string, 0, len(all))
	k := 0
	for _, p := range all {
		if p.Off {
			order = append(order, p.ID)
		} else {
			order = append(order, out[k])
			k++
		}
	}
	return provider.SetOrder(order)
}
