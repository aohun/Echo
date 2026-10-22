package magpieclient

// The Routing page's write actions over HTTP (docs/08 M4): the found,
// save, delete, switch and arrange the page's toggle, editor, row switch
// and drag do. Each mirrors what magpie's own page asks its API to do,
// and each answers with the full groups state so the caller can adopt it.

import (
	"context"
	"net/http"

	"github.com/yetone/magpie/internal/provider"
)

// groupBody is the body magpie's POST /api/groups/{action} takes: the
// group itself, plus the action's own part — the id the group had
// (another is a rename), the found toggle's on, the arrange order, the
// several groups a delete may take at once.
type groupBody struct {
	provider.Group
	From  string   `json:"from,omitempty"`
	On    bool     `json:"on,omitempty"`
	Order []string `json:"order,omitempty"`
	IDs   []string `json:"ids,omitempty"`
}

// GroupsFound turns magpie finding groups on its own on or off.
//
// It is magpie's own found action: off, the agents set to a found group
// are moved to its model from one provider (provider.SetAutoGroups
// through agent.Reseat). The answer is the full groups document.
func (c *Client) GroupsFound(ctx context.Context, on bool) (GroupsDoc, error) {
	var out GroupsDoc
	err := c.do(ctx, http.MethodPost, "/api/groups/found", groupBody{On: on}, &out)
	return out, err
}

// GroupsSave saves the group as the editor's save does, renaming it when
// from names the id it had. The id's own slug rules are the server's to
// apply, as magpie's page lets them be. The answer is the full groups
// document.
func (c *Client) GroupsSave(ctx context.Context, g provider.Group, from string) (GroupsDoc, error) {
	var out GroupsDoc
	err := c.do(ctx, http.MethodPost, "/api/groups/save", groupBody{Group: g, From: from}, &out)
	return out, err
}

// GroupsDelete deletes the named groups, all or none — one at a time when
// the page's row menu does it, several when a selection does. The answer
// is the full groups document.
func (c *Client) GroupsDelete(ctx context.Context, ids []string) (GroupsDoc, error) {
	var out GroupsDoc
	err := c.do(ctx, http.MethodPost, "/api/groups/delete", groupBody{IDs: ids}, &out)
	return out, err
}

// GroupsSwitch switches a group of the user's on or off, kept as it is.
// The answer is the full groups document.
func (c *Client) GroupsSwitch(ctx context.Context, id string, on bool) (GroupsDoc, error) {
	g := groupBody{On: on}
	g.ID = id
	var out GroupsDoc
	err := c.do(ctx, http.MethodPost, "/api/groups/switch", g, &out)
	return out, err
}

// GroupsArrange puts the groups in the order given, the same order
// magpie's arrange endpoint saves (provider.SetGroupOrder), which
// /v1/models follows too. The answer is the full groups document.
func (c *Client) GroupsArrange(ctx context.Context, order []string) (GroupsDoc, error) {
	var out GroupsDoc
	err := c.do(ctx, http.MethodPost, "/api/groups/arrange", groupBody{Order: order}, &out)
	return out, err
}
