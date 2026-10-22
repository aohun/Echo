package magpieclient

// The Routing page's live trace over HTTP (docs/08 M4): the gateway's
// own in-memory routes and totals — what the embedded read reads straight
// from the gateway it serves (gw.Trace), and what magpie's page plays
// live. The disk history (/api/gateway/history) is a day gone by, asked
// for by its day; the live page does not read it.

import (
	"context"
	"net/http"
	"time"
)

// GatewayTraceDoc is what /api/gateway/trace answers: the routes changed
// since the sequence named, the totals since the gateway started, and
// mine — whether this instance serves the gateway at all.
type GatewayTraceDoc struct {
	Seq    int64          `json:"seq"`
	Routes []GatewayRoute `json:"routes"`
	Totals struct {
		Requests int `json:"requests"`
		Rerouted int `json:"rerouted"`
		Errors   int `json:"errors"`
	} `json:"totals"`
	Mine bool      `json:"mine"`
	Now  time.Time `json:"now"`
}

// GatewayTrace reads the instance's live routing trace. A wait would hold
// the answer for what changes next; the page that reads this reads again
// on its own cadence instead.
func (c *Client) GatewayTrace(ctx context.Context) (GatewayTraceDoc, error) {
	var out GatewayTraceDoc
	err := c.do(ctx, http.MethodGet, "/api/gateway/trace", nil, &out)
	return out, err
}
