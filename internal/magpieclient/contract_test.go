package magpieclient

// The contract against drift (docs/08 M5): this suite runs against a
// live `magpie web` built from the pinned upstream checkout, and re-runs
// M0's recordings — a shape check on every response Echo reads. When
// magpie moves, a red contract test names the endpoint that drifted, the
// same role magpie's own generated-client contract plays.
//
// The instance runs the way M0 probed it: a throwaway home, a generated
// run key held in memory, the gateway on one port and the web on
// another — nothing of this machine's real configuration is read or
// written. The pinned version is recorded in docs/08-plan-option-b-
// web-client.md; when magpie moves and its contract is re-recorded, move
// the pin with it.

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yetone/magpie/internal/provider"
	"github.com/yetone/magpie/internal/usage"
	"github.com/yetone/magpie/internal/usageapi"
)

const (
	// pinnedMagpieCommit is the checkout the contract speaks, as M0
	// probed it.
	pinnedMagpieCommit = "9f0bd95bad7dd0dbe22c97c7aaa42c1230c21aa1"
	// pinnedMagpieDefault is where the checkout lives on this machine;
	// ECHO_MAGPIE_CHECKOUT moves it.
	pinnedMagpieDefault = "/Users/wayne/Desktop/git/magpie"
)

func TestContractAgainstThePinnedMagpie(t *testing.T) {
	checkout := os.Getenv("ECHO_MAGPIE_CHECKOUT")
	if checkout == "" {
		checkout = pinnedMagpieDefault
	}
	if _, err := os.Stat(filepath.Join(checkout, "go.mod")); err != nil {
		t.Skipf("no magpie checkout at %s: %v", checkout, err)
	}
	if _, err := exec.LookPath("go"); err != nil {
		t.Skipf("no go toolchain to build magpie with: %v", err)
	}
	out, err := exec.Command("git", "-C", checkout, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Skipf("cannot read the checkout's commit: %v", err)
	}
	if got := strings.TrimSpace(string(out)); got != pinnedMagpieCommit {
		t.Skipf("the checkout is at %s; the contract is pinned to %s — re-record and move the pin when magpie moves",
			got, pinnedMagpieCommit)
	}

	bin := filepath.Join(t.TempDir(), "magpie")
	build := exec.Command("go", "build", "-o", bin, ".")
	build.Dir = checkout
	build.Env = append(os.Environ(), "CGO_ENABLED=1")
	if b, err := build.CombinedOutput(); err != nil {
		t.Skipf("building magpie: %v\n%s", err, b)
	}

	// Two free ports: the gateway's and the web's, as M0's probe ran.
	gwPort, webPort := freePort(t), freePort(t)
	home := t.TempDir()
	key, err := NewKey()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(bin, "web", "--no-open", "--addr", "127.0.0.1:"+webPort)
	cmd.Env = append(os.Environ(),
		"HOME="+home,
		"XDG_CONFIG_HOME="+filepath.Join(home, ".config"),
		"XDG_CACHE_HOME="+filepath.Join(home, ".cache"),
		"MAGPIE_ADDR=127.0.0.1:"+gwPort,
		"MAGPIE_WEB_KEY="+key,
		"MAGPIE_WEB_RUNKEY="+key,
	)
	if err := cmd.Start(); err != nil {
		t.Fatalf("starting magpie web: %v", err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	t.Cleanup(func() {
		if cmd.Process != nil {
			cmd.Process.Signal(os.Interrupt)
		}
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			cmd.Process.Kill()
		}
	})

	c := New("http://127.0.0.1:"+webPort, key)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	deadline := time.Now().Add(30 * time.Second)
	for {
		if _, err := c.Probe(ctx); err == nil {
			break
		} else if time.Now().After(deadline) {
			t.Fatalf("the pinned instance never came up: %v", err)
		} else {
			time.Sleep(300 * time.Millisecond)
		}
	}

	// ---- every response Echo reads, shape-checked ----

	// /api/settings: the probe and the version gate.
	if info, err := c.Probe(ctx); err != nil || info.Version == "" {
		t.Errorf("/api/settings: version %q err %v", info.Version, err)
	}

	// /api/state: the Agents page's document.
	if s, err := c.State(ctx); err != nil || s == nil {
		t.Errorf("/api/state: %v", err)
	}

	// /api/drift: the poll the pages ask on their timers.
	if drift, err := c.Drift(ctx); err != nil || drift == nil {
		t.Errorf("/api/drift: %v", err)
	}

	// /api/providers: the Gateway card and the Providers page's rows.
	doc, err := c.Providers(ctx)
	if err != nil {
		t.Fatalf("/api/providers: %v", err)
	}
	// Version is another magpie's, serving the gateway; one serving its
	// own (mine) rightly carries none.
	if doc.Gateway.Running && !doc.Gateway.Mine && doc.Gateway.Version == "" {
		t.Error("/api/providers: a gateway served by another magpie carries no version")
	}
	t.Logf("/api/providers gateway: running=%v version=%q url=%q models=%d calls=%d",
		doc.Gateway.Running, doc.Gateway.Version, doc.Gateway.URL, doc.Gateway.Models, len(doc.Gateway.Calls))
	for _, p := range doc.Providers {
		if p.ID == "" || p.Name == "" {
			t.Errorf("/api/providers: a row without id or name: %+v", p)
			break
		}
	}

	// /api/groups: the Routing page's document, then its writes —
	// save, read back, delete, read again.
	if _, err := c.Groups(ctx); err != nil {
		t.Fatalf("/api/groups: %v", err)
	}
	g, err := c.GroupsSave(ctx, provider.Group{
		ID: "contract-grp", Name: "Contract", Routing: provider.Rotate,
		Members: []string{"openai/gpt-5"},
	}, "")
	if err != nil {
		t.Fatalf("/api/groups/save: %v", err)
	}
	if len(g.Groups) != 1 {
		t.Fatalf("/api/groups/save answered %d groups, want the saved one", len(g.Groups))
	}
	saved := g.Groups[0]
	if saved.ID != "contract-grp" || saved.Routing != string(provider.Rotate) {
		t.Errorf("the saved group came back %+v", saved.Group)
	}
	if len(saved.Info) != 1 || saved.Info[0].Of != "openai/gpt-5" {
		t.Errorf("the saved group's members came back %+v", saved.Info)
	}
	if _, err := c.GroupsDelete(ctx, []string{"contract-grp"}); err != nil {
		t.Fatalf("/api/groups/delete: %v", err)
	}
	if after, err := c.Groups(ctx); err != nil || len(after.Groups) != 0 {
		t.Errorf("after the delete %d groups remain: %v", len(after.Groups), err)
	}

	// /api/groups/arrange: the whole order, answered with the document.
	if _, err := c.GroupsArrange(ctx, []string{}); err != nil {
		t.Errorf("/api/groups/arrange: %v", err)
	}

	// /api/provider/off and /on, on the first provider the document
	// names — the row switch's writes, each answered with the document
	// and put back as it was.
	if len(doc.Providers) > 0 {
		id := doc.Providers[0].ID
		if _, err := c.ProviderSwitch(ctx, id, true); err != nil {
			t.Errorf("/api/provider/off %s: %v", id, err)
		} else if _, err := c.ProviderSwitch(ctx, id, false); err != nil {
			t.Errorf("/api/provider/on %s: %v", id, err)
		}
	}

	// /api/providers/arrange: the page's drag, in the order read.
	if len(doc.Providers) > 1 {
		order := make([]string, len(doc.Providers))
		for i, p := range doc.Providers {
			order[i] = p.ID
		}
		if _, err := c.ProvidersArrange(ctx, order); err != nil {
			t.Errorf("/api/providers/arrange: %v", err)
		}
	}

	// /api/usage, /api/usage/requests, /api/usage/quotas: the Usage
	// page's three remote reads, decoded into the very types the shell
	// decodes them into.
	if err := c.Do(ctx, "GET", "/api/usage?period="+string(usage.Month), nil, &usageapi.Usage{}); err != nil {
		t.Errorf("/api/usage: %v", err)
	}
	if err := c.Do(ctx, "GET", "/api/usage/requests?offset=0&limit=10", nil, &usageapi.Ledger{}); err != nil {
		t.Errorf("/api/usage/requests: %v", err)
	}
	quotas := []provider.SubscriptionQuota{}
	if err := c.Do(ctx, "GET", "/api/usage/quotas", nil, &quotas); err != nil {
		t.Errorf("/api/usage/quotas: %v", err)
	}

	// /api/set: with no agent named, the instance's error is the
	// contract too — an APIError with a body, not a hung request.
	var st State
	err = c.do(ctx, "POST", "/api/set", map[string]string{"agent": "contract-none"}, &st)
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Errorf("/api/set with no such agent: %v, want an APIError", err)
	}
}
