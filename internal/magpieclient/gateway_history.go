package magpieclient

// The Routing page's history over HTTP (docs/08 M4): the gateway's own
// trace of each routed request, the same history magpie's page plays —
// one day of it, or every day when no day is named.

import (
	"context"
	"net/http"

	"github.com/yetone/magpie/internal/gateway"
)

// GatewayRoute is one routed request as the history answers it: the
// gateway's own record, plus what the instance priced it at and the
// codex session's title it resolved.
type GatewayRoute struct {
	gateway.Route
	SessionTitle string  `json:"sessionTitle,omitempty"`
	Cost         float64 `json:"cost"`
	Priced       bool    `json:"priced"`
	Unpriced     int     `json:"unpriced"`
}

// GatewayHistoryDoc is what /api/gateway/history answers: the days the
// history keeps, the named day's routes (every day's when no day is
// named), and cut — whether the instance's history file was truncated.
type GatewayHistoryDoc struct {
	Days   []string       `json:"days"`
	Routes []GatewayRoute `json:"routes"`
	Cut    bool           `json:"cut"`
}

// GatewayHistory reads the gateway's routing history. The empty day
// reads every day, which is the cut Echo's Routing page draws.
func (c *Client) GatewayHistory(ctx context.Context, day string) (GatewayHistoryDoc, error) {
	var out GatewayHistoryDoc
	err := c.do(ctx, http.MethodGet, "/api/gateway/history?day="+day, nil, &out)
	return out, err
}
