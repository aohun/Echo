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
// instance: the status card in full — URL, running, whose it is, the
// version — and the routing totals from the trace. The page's own model
// list and its recent calls still come from the embedded reader: their
// HTTP shapes (/api/groups's models, the trace's routes under traffic)
// are mapped page by page in the milestones after this one.
func readGatewayRemote(c *magpieclient.Client) (GatewayRow, bool) {
	g := GatewayRow{}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	info, err := c.Probe(ctx)
	if err != nil {
		return g, false
	}
	g.URL = info.Gateway
	g.Running, g.Version = true, info.Version
	// The instance serves the gateway; Echo does not.
	g.Mine = false
	// Models, Entries and Calls stay at their zero values: their sources
	// (/api/groups's model list, the trace's routes under real traffic)
	// are mapped page by page in the milestones after this one, and a
	// guessed mapping is worse than an empty list.
	return g, true
}
