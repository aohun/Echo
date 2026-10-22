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
// of one runs unchanged. The entries list (the models with their icons
// and the routing groups first) is the one cut still mapped in a later
// milestone; a guessed ordering is worse than an empty list.
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
	return g, true
}
