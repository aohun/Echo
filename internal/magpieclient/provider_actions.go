package magpieclient

// The providers page's write actions over HTTP (docs/08 M4): the switch
// and the arrange the row's switch and drag do. Each mirrors what
// magpie's own page asks its API to do, and each answers with the full
// providers state so the caller can adopt it.

import (
	"context"
	"net/http"
)

// SwitchProvider switches a provider off or on, as the row's switch does.
//
// It is magpie's own switchProvider (provider.SetOff through agent.Reseat):
// the agents on one of its models are moved to others, their files written.
// The answer is the full providers document, adopted by the caller.
func (c *Client) ProviderSwitch(ctx context.Context, id string, off bool) (ProvidersDoc, error) {
	action := "on"
	if off {
		action = "off"
	}
	var out ProvidersDoc
	err := c.do(ctx, http.MethodPost, "/api/provider/"+action, map[string]string{"id": id}, &out)
	return out, err
}

// ProvidersArrange puts the providers in the order given — the same order
// magpie's arrange endpoint saves (provider.SetOrder). The answer is the
// full providers document, adopted by the caller.
func (c *Client) ProvidersArrange(ctx context.Context, order []string) (ProvidersDoc, error) {
	var out ProvidersDoc
	err := c.do(ctx, http.MethodPost, "/api/providers/arrange", map[string]any{"order": order}, &out)
	return out, err
}
