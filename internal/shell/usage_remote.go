//go:build nogui && echo

package shell

// The Usage page's remote read (docs/08 M4): the overview, the ledger and
// the quota cards from a magpie web instance. Its /api/usage and
// /api/usage/requests answer the very shapes usageapi marshals — the
// client decodes them straight in, no re-drawing — and /api/usage/quotas
// answers the providers' quota cards.
//
// What stays embedded even in remote mode: the Sessions and Context
// segments (they read this machine's agent session files) and the
// requests heatmap (no endpoint serves it yet). A page that asks for one
// of those falls back to the embedded read below.

import (
	"context"
	"net/url"
	"strconv"
	"time"

	"github.com/yetone/magpie/internal/magpieclient"
	"github.com/yetone/magpie/internal/provider"
)

func readUsageRemote(c *magpieclient.Client, q UsageQuery) (UsageData, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	ud := UsageData{Query: q}
	switch q.Tab {
	case UsTabSessions, UsTabContext:
		// Local only: the agents' session files and the routing
		// history's prompts are this machine's.
		return UsageData{}, false
	case UsTabRequests:
		v := usageParams(q)
		v.Set("offset", strconv.Itoa(max(0, q.Offset)))
		v.Set("limit", strconv.Itoa(max(1, q.Limit)))
		if err := c.Do(ctx, "GET", "/api/usage/requests?"+v.Encode(), nil, &ud.Ledger); err != nil {
			return UsageData{}, false
		}
		// The heatmap has no remote endpoint; the segment draws without
		// it until one is mapped.
	default:
		v := url.Values{"period": {string(q.Period)}}
		if err := c.Do(ctx, "GET", "/api/usage?"+v.Encode(), nil, &ud.Overview); err != nil {
			return UsageData{}, false
		}
		qs, err := readQuotasRemote(ctx, c)
		if err != nil {
			return UsageData{}, false
		}
		ud.Quotas = qs
		// The instance reads its own cards; its "reading" header says so
		// for its page, and Echo draws the cards it was handed as they
		// are. A card that lands late shows on the next read.
	}
	return ud, true
}

// readQuotasRemote reads the instance's quota cards: the same
// SubscriptionQuota documents the embedded read builds, from the vendors
// the instance reaches. Up to the handler's own 12s.
func readQuotasRemote(ctx context.Context, c *magpieclient.Client) ([]provider.SubscriptionQuota, error) {
	qs := []provider.SubscriptionQuota{}
	if err := c.Do(ctx, "GET", "/api/usage/quotas", nil, &qs); err != nil {
		return nil, err
	}
	return qs, nil
}

// usageParams is the UsageQuery as the ledger's query string: the same
// names magpie's ledgerFilter reads.
func usageParams(q UsageQuery) url.Values {
	v := url.Values{}
	v.Set("period", string(q.Period))
	set := func(name, val string) {
		if val != "" {
			v.Set(name, val)
		}
	}
	set("day", q.Day)
	set("agent", q.Agent)
	set("provider", q.Provider)
	set("purpose", q.Purpose)
	set("account", q.Account)
	set("callerKey", q.CallerKey)
	set("model", q.Model)
	set("q", q.Query)
	set("via", q.Via)
	if q.Status == "failed" {
		v.Set("failed", "1")
	}
	if q.Status == "retried" {
		v.Set("retried", "1")
	}
	return v
}
