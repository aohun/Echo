package magpieclient

// The Routing page's history over HTTP (docs/08 M4): the gateway's own
// record of each routed request it kept on disk — one day gone by, named
// by its day. The live trace is /api/gateway/trace's (gateway_trace.go).

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

// GatewayHistory reads one named day of the gateway's routing history.
// The instance answers the days it keeps beside the routes; the empty day
// names none of them and answers no routes — the live trace is
// /api/gateway/trace's, not this.
func (c *Client) GatewayHistory(ctx context.Context, day string) (GatewayHistoryDoc, error) {
	var out GatewayHistoryDoc
	err := c.do(ctx, http.MethodGet, "/api/gateway/history?day="+day, nil, &out)
	return out, err
}
