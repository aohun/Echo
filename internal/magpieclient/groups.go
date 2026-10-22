package magpieclient

// The Routing page's groups over HTTP (docs/08 M4): the routing groups,
// the models they can be made of, and each pool of keys or accounts —
// which any of its models routes over already.

import (
	"context"
	"net/http"

	"github.com/yetone/magpie/internal/agent"
	"github.com/yetone/magpie/internal/provider"
)

// GroupsDoc is what the Routing view manages: the routing groups, the
// models they can be made of, and each provider with several keys or
// accounts on — which any of its models routes over already.
type GroupsDoc struct {
	Groups []GroupJSON `json:"groups"`
	Models []ModelRef  `json:"models"`
	Pools  []PoolJSON  `json:"pools"`
	// Deciders: the decision providers' models (Jev), which may only be a
	// group's classifier
	Deciders []ModelRef `json:"deciders"`
	// Found: magpie finds groups on its own (provider.AutoGroupsOn)
	Found bool `json:"found"`
	// Moved: the agents turning found groups off moved off one of them,
	// to its model from one provider (agent.Reseat)
	Moved []agent.Move `json:"moved,omitempty"`
}

type GroupJSON struct {
	provider.Group
	Ready bool         `json:"ready"` // a member is: agents can pick it
	Info  []MemberJSON `json:"memberInfo"`
	// Holds: the groups in it, at any depth — none of which can have it in
	// turn
	Holds []string `json:"holds"`
	// Offers: the reasoning levels agents are offered for it; Shared: those
	// its members have in common, which it offers unless it names its own
	Offers []string `json:"offers"`
	Shared []string `json:"shared"`
	// Picked: the member a manual group sends every request to
	Picked string `json:"picked,omitempty"`
	// Patterns: its patterns, each with how many models it matches now
	// (#766), so one that matches nothing is said on its card
	Patterns []provider.PatternHit `json:"patterns"`
}

type MemberJSON struct {
	ID       string `json:"id"`
	Ready    bool   `json:"ready"`
	Provider string `json:"provider,omitempty"`
	Name     string `json:"name,omitempty"` // the provider's
	Icon     string `json:"icon,omitempty"`
	Model    string `json:"model,omitempty"` // what the vendor is asked for
	On       int    `json:"on"`              // its keys or accounts on
	Group    bool   `json:"group,omitempty"` // a routing group in the group; Name is its
	// Of is the model's id without the effort the member is fixed at, and
	// Effort that effort ("provider/model:low"); "" for one without
	Of     string `json:"of,omitempty"`
	Effort string `json:"effort,omitempty"`
	// Fast: the group sends it in its vendor's fast mode; CanFast: its
	// model has one (provider.CanFast)
	Fast    bool `json:"fast,omitempty"`
	CanFast bool `json:"canFast,omitempty"`
	// Efforts: the reasoning levels this member is offered. A model's own,
	// or — a member that is itself a group — the levels it passes on to
	// agents, which are the ones every model in it has.
	Efforts []string `json:"efforts,omitempty"`
	// what a rule may send it: the tokens it takes, when known, and images
	Context int  `json:"context,omitempty"`
	Images  bool `json:"images,omitempty"`
	// ImagesUnknown: nothing was read of its images either way, so it counts
	// text-only for a describer without its list having said it takes none.
	ImagesUnknown bool `json:"imagesUnknown,omitempty"`
}

type ModelRef struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Provider string `json:"provider"`
	PName    string `json:"providerName"`
	Icon     string `json:"icon,omitempty"`
	Context  int    `json:"context,omitempty"` // the tokens it takes, when known
	// Efforts: the model's reasoning levels, when known — those a group's
	// member of it may be fixed at
	Efforts []string `json:"efforts,omitempty"`
	// CanFast: a group's member of it may be sent in its vendor's fast
	// mode (provider.CanFast)
	CanFast bool `json:"canFast,omitempty"`
	// Images: agents are told it takes images (provider.Entry.Images); a
	// group's editor says of each member what it says of it (#756).
	// ImagesUnknown: nothing was read of it either way (imagesUnknown),
	// sent only when true
	Images        bool `json:"images,omitempty"`
	ImagesUnknown bool `json:"imagesUnknown,omitempty"`
}

type PoolJSON struct {
	Provider string   `json:"provider"`
	Name     string   `json:"name"`
	Icon     string   `json:"icon,omitempty"`
	Kind     string   `json:"kind"` // "account" or "key"
	Who      []string `json:"who"`
	Routing  string   `json:"routing"`
	Affinity string   `json:"affinity"`
	Sink     bool     `json:"sink,omitempty"`
	Protocol string   `json:"protocol,omitempty"`
}

// Groups reads the Routing page's groups: the routing groups, the models
// they can be made of, and each pool of keys or accounts — which any of
// its models routes over already.
func (c *Client) Groups(ctx context.Context) (GroupsDoc, error) {
	var doc GroupsDoc
	if err := c.do(ctx, http.MethodGet, "/api/groups", nil, &doc); err != nil {
		return GroupsDoc{}, err
	}
	return doc, nil
}
