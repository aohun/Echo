//go:build nogui && echo

package shell

// The data-source seam of the Option B experiment (docs/08 M3): the
// pages read their numbers either from the domain packages this process
// carries (embedded, Plan A, the default) or from a magpie web instance
// over loopback HTTP (remote, this branch's subject). One environment
// variable picks, so the two ride the same binary and every page can
// change sides on its own.

import (
	"context"
	"encoding/json"
	"os"
	"sync"
	"time"

	"github.com/yetone/magpie/internal/magpieclient"
)

// remoteEnvAddr and remoteEnvKey name the magpie web instance: its
// origin and its run key. The key is a secret; it lives only here, in
// the process's environment, never in a log or the tree.
const (
	remoteEnvAddr = "ECHO_REMOTE_MAGPIE_ADDR"
	remoteEnvKey  = "ECHO_REMOTE_MAGPIE_KEY"
)

var (
	remoteOnce   sync.Once
	remoteClient *magpieclient.Client
)

// remoteGateway returns the magpie web client when the environment asks
// for one and the instance answers its probe; nil means the page reads
// embedded as it always has.
func remoteGateway() *magpieclient.Client {
	remoteOnce.Do(func() {
		addr, key := os.Getenv(remoteEnvAddr), os.Getenv(remoteEnvKey)
		if addr == "" {
			return
		}
		c := magpieclient.New(addr, key)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := c.Probe(ctx); err != nil {
			return
		}
		remoteClient = c
	})
	return remoteClient
}

// readGatewayRemote builds what the Gateway page draws from a magpie web
// instance's /api/providers: its gateway part carries the card in full —
// URL, running, whose it is, the version, the model count — and the
// recent calls as the gateway's own Call type, so the shell's conversion
// of one runs unchanged. The entries list is the document's own: the
// routing groups it names whole, then each provider's exposed models.
func readGatewayRemote(c *magpieclient.Client) (GatewayRow, bool) {
	g := GatewayRow{}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	doc, err := c.Providers(ctx)
	if err != nil {
		return g, false
	}
	gs := doc.Gateway
	g.URL, g.Running, g.Mine = gs.URL, gs.Running, gs.Mine
	g.Version, g.Older = gs.Version, gs.Running && gs.Older
	g.Models = gs.Models
	for _, call := range gs.Calls {
		g.Calls = append(g.Calls, gatewayCall(call))
	}
	// Routed is the agents on a provider's model now, the same cut the
	// page's own count makes: distinct agents, any provider's agent tile
	// marked current.
	routed := map[string]bool{}
	for _, p := range doc.Providers {
		for _, a := range p.Agents {
			if a.Current {
				routed[a.ID] = true
			}
		}
	}
	g.Routed = len(routed)
	// Entries: the routing groups first, then every exposed model of
	// every provider not kept unlisted or switched off — the cut the
	// embedded read makes over provider.Catalog, rebuilt from the
	// document's two halves: the groups it carries whole, and each
	// provider row's models, exposed ones only.
	entries := make([]GatewayModel, 0, gs.Models)
	for _, grp := range gs.Groups {
		m := GatewayModel{
			ID:      grp.ID,
			Name:    grp.Name,
			Group:   true,
			Context: grp.Context,
			Output:  grp.Output,
			Efforts: grp.Efforts,
			Images:  grp.Images,
		}
		// a group's provider is its members' — the document names them
		m.Provider = "routing group"
		if len(grp.Providers) > 0 {
			m.Provider = grp.Providers[0]
		}
		if len(grp.Icons) > 0 {
			m.Icon = grp.Icons[0]
		}
		entries = append(entries, m)
	}
	for _, p := range doc.Providers {
		if p.Off || p.Unlisted {
			continue
		}
		for _, raw := range p.Models {
			var mj struct {
				ID      string   `json:"id"`
				Name    string   `json:"name"`
				Default string   `json:"default,omitempty"`
				On      bool     `json:"on"`
				Context int      `json:"context,omitempty"`
				Output  int      `json:"output,omitempty"`
				Efforts []string `json:"efforts"`
				Images  bool     `json:"images"`
				Free    bool     `json:"free,omitempty"`
			}
			if err := json.Unmarshal(raw, &mj); err != nil {
				// a model cut that no longer decodes is a contract
				// drift: the page falls back to the embedded read
				return GatewayRow{}, false
			}
			if !mj.On {
				continue
			}
			name := mj.Name
			if name == "" {
				name = mj.Default
			}
			entries = append(entries, GatewayModel{
				ID:       p.ID + "/" + mj.ID,
				Name:     name,
				Provider: p.Name,
				Icon:     p.Icon,
				Context:  mj.Context,
				Output:   mj.Output,
				Efforts:  mj.Efforts,
				Images:   mj.Images,
				Free:     mj.Free,
			})
		}
	}
	g.Entries = entries
	return g, true
}
