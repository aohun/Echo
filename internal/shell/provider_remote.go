//go:build nogui && echo

package shell

// The Providers page's remote read (docs/08 M4): the rows from a magpie
// web instance's /api/providers, whose providerJSON carries the facts a
// row draws — the masked key, ready, the signed-in account's plan and
// user, the agents on it now. The instance's editor-shaped document is
// decoded only as far as the row needs; the rest of each entry rides
// along undecoded in the client's Raw.

// What stays embedded even in remote mode:
//   - Lapsed: its source is the logins' refused states, which the
//     document's account cut does not carry; a row shows not-lapsed
//     until that cut is mapped.
//   - the page's writes: a switch or a drag still acts on the embedded
//     domain (provider.SetOff, provider.SetOrder). In remote mode they
//     are disabled at the call sites, not sent to the wrong process.

import (
	"context"
	"time"

	"github.com/yetone/magpie/internal/magpieclient"
)

func readProvidersRemote(c *magpieclient.Client) ([]ProviderRow, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	doc, err := c.Providers(ctx)
	if err != nil {
		return nil, false
	}
	rows := make([]ProviderRow, 0, len(doc.Providers))
	for _, p := range doc.Providers {
		r := ProviderRow{
			ID:        p.ID,
			Name:      p.Name,
			Icon:      p.Icon,
			Host:      p.Host,
			Preset:    p.Preset,
			Keyed:     p.Key.Set,
			Ready:     p.Ready,
			Off:       p.Off,
			Sponsored: p.Sponsored,
		}
		if acct := p.Account; acct != nil {
			r.SignedIn = true
			r.User = acct.User
			r.Plan = accountPlan(acct.Agent, acct.Plan)
		} else {
			r.Key = p.Key.Masked
			r.NeedsKey = !p.Ready
		}
		// A provider switched off serves nothing: its count is its full
		// list's, not the exposed cut (the embedded read's own rule).
		if p.Off {
			r.Models = len(p.Models)
		} else {
			r.Models, r.Draws = p.Exposed, p.Draws
		}
		for _, a := range p.Agents {
			if !a.Current {
				continue
			}
			r.Agents = append(r.Agents, AgentUse{Icon: a.Icon, Name: a.Name, Model: a.Model, Group: a.Group})
		}
		rows = append(rows, r)
	}
	return rows, true
}
