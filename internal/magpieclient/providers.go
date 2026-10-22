package magpieclient

// The instance's providers document (GET /api/providers): the gateway's
// status and recent calls as the gateway's own Call type, and the
// providers with what their rows draw. Only the cuts Echo reads are
// typed; the editor's rest of the document stays untyped.

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/yetone/magpie/internal/gateway"
)

// GatewayStatus is the document's gateway part: the card the Gateway
// page draws. Calls are the instance's recent calls, decoded as the
// gateway's own type — the shell's conversion of one runs unchanged.
type GatewayStatus struct {
	URL     string         `json:"url"`
	LAN     bool           `json:"lan"`
	LANURLs []string       `json:"lanURLs,omitempty"`
	Open    bool           `json:"open,omitempty"`
	Running bool           `json:"running"`
	Mine    bool           `json:"mine"`
	Window  bool           `json:"window"`
	Version string         `json:"version,omitempty"`
	Older   bool           `json:"older,omitempty"`
	Models  int            `json:"models"`
	Calls   []gateway.Call `json:"calls"`
}

// ProviderAgents is one provider's using agents, as a row's tiles draw
// them.
type ProviderAgents struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Icon    string `json:"icon"`
	Current bool   `json:"current"`
	Model   string `json:"model,omitempty"`
	Group   string `json:"group,omitempty"`
}

// ProviderSummary is one provider of the document as far as the gateway
// card and the providers page's first cut read it: who it is, whether it
// is on, and the agents on it now.
type ProviderSummary struct {
	ID     string           `json:"id"`
	Name   string           `json:"name"`
	Icon   string           `json:"icon"`
	Off    bool             `json:"off,omitempty"`
	Agents []ProviderAgents `json:"agents"`
	// Raw keeps the editor's rest of the document for the page that
	// needs it, undecoded.
	Raw map[string]any `json:"-"`
}

// UnmarshalJSON keeps the summary's fields and holds the rest in Raw.
func (p *ProviderSummary) UnmarshalJSON(b []byte) error {
	type plain ProviderSummary
	if err := json.Unmarshal(b, (*plain)(p)); err != nil {
		return err
	}
	return json.Unmarshal(b, &p.Raw)
}

// ProvidersDoc is the instance's providers document's first cut.
type ProvidersDoc struct {
	Providers []ProviderSummary `json:"providers"`
	Gateway   GatewayStatus     `json:"gateway"`
}

// Providers reads the document. It is the Gateway card's source of truth
// over HTTP — /api/state does not carry the gateway — and the providers
// page's starting cut.
func (c *Client) Providers(ctx context.Context) (ProvidersDoc, error) {
	var doc ProvidersDoc
	if err := c.do(ctx, http.MethodGet, "/api/providers", nil, &doc); err != nil {
		return ProvidersDoc{}, err
	}
	return doc, nil
}
