package magpieclient

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// fake is a magpie web instance in a test: it guards its cookie the way
// webGuard does and answers the calls the tests give it.
type fake struct {
	t       *testing.T
	state   string // the state document the writes answer
	seen    map[string]*http.Request
	bodies  map[string]string
	handler http.Handler
}

func newFake(t *testing.T, state string) *fake {
	f := &fake{t: t, state: state, seen: map[string]*http.Request{}, bodies: map[string]string{}}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/settings", func(rw http.ResponseWriter, r *http.Request) {
		if !f.guard(rw, r, "/api/settings") {
			return
		}
		rw.Header().Set("Content-Type", "application/json")
		rw.Write([]byte(`{"version":"v9.9.9","gateway":"http://127.0.0.1:3497","dir":"/tmp/there","extra":true}`))
	})
	mux.HandleFunc("GET /api/state", func(rw http.ResponseWriter, r *http.Request) {
		if !f.guard(rw, r, "/api/state") {
			return
		}
		rw.Write([]byte(f.state))
	})
	mux.HandleFunc("POST /api/set", func(rw http.ResponseWriter, r *http.Request) {
		if !f.guard(rw, r, "/api/set") {
			return
		}
		f.bodies["/api/set"] = readAll(r)
		rw.Write([]byte(f.state))
	})
	mux.HandleFunc("POST /api/agents/rescan", func(rw http.ResponseWriter, r *http.Request) {
		if !f.guard(rw, r, "/api/agents/rescan") {
			return
		}
		rw.Write([]byte(f.state))
	})
	mux.HandleFunc("GET /api/drift", func(rw http.ResponseWriter, r *http.Request) {
		if !f.guard(rw, r, "/api/drift") {
			return
		}
		rw.Write([]byte(`{}`))
	})
	f.handler = mux
	return f
}

// guard checks what every real request carries: the instance's run key,
// in a cookie named for the port the instance answers on. The fake
// listens on whatever port the test server took, so it reads the name's
// prefix and checks the value — the name derivation itself has its own
// test below.
func (f *fake) guard(rw http.ResponseWriter, r *http.Request, path string) bool {
	f.seen[path] = r
	for _, ck := range r.Cookies() {
		if strings.HasPrefix(ck.Name, "magpie_web_") && ck.Value == "test-key-0123456789" {
			return true
		}
	}
	rw.WriteHeader(http.StatusUnauthorized)
	rw.Write([]byte("the link carries this run's key"))
	return false
}

func readAll(r *http.Request) string {
	var buf strings.Builder
	b := make([]byte, 4096)
	for {
		n, err := r.Body.Read(b)
		buf.Write(b[:n])
		if err != nil {
			break
		}
	}
	return buf.String()
}

const stateDoc = `{"agents":[{"id":"claude","name":"Claude Code","icon":"claudecode-color","path":"~/.claude/settings.json","fields":[{"key":"model","label":"model","value":"claude-sonnet-5-5","options":[{"value":"claude-sonnet-5-5","note":"Claude Sonnet 5.5","group":"Claude Code"}]}]}],"catalog":"7f3a","clients":[],"fx":{},"profiles":[],"settings":{"version":"v9.9.9","gateway":"http://127.0.0.1:3497","theme":"system","web":true,"unknown-but-real":42}}`

func newTestClient(f *fake) *Client {
	srv := httptest.NewServer(f.handler)
	f.t.Cleanup(srv.Close)
	return New(srv.URL, "test-key-0123456789")
}

func TestProbeReadsVersionGatewayDir(t *testing.T) {
	f := newFake(t, stateDoc)
	c := newTestClient(f)
	info, err := c.Probe(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if info.Version != "v9.9.9" || info.Gateway != "http://127.0.0.1:3497" || info.Dir != "/tmp/there" {
		t.Errorf("probe read version %q, gateway %q, dir %q", info.Version, info.Gateway, info.Dir)
	}
	if f.seen["/api/settings"] == nil {
		t.Error("probe did not reach /api/settings")
	}
}

func TestStateDecodesTypedAndKeepsRaw(t *testing.T) {
	f := newFake(t, stateDoc)
	c := newTestClient(f)
	s, err := c.State(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Agents) != 1 || s.Agents[0].ID != "claude" {
		t.Fatalf("agents decoded as %+v", s.Agents)
	}
	if s.Agents[0].Fields[0].Value != "claude-sonnet-5-5" {
		t.Errorf("field value %q", s.Agents[0].Fields[0].Value)
	}
	if s.Agents[0].Fields[0].Options[0].Group != "Claude Code" {
		t.Errorf("option group %q", s.Agents[0].Fields[0].Options[0].Group)
	}
	if s.Settings.Raw["unknown-but-real"] != float64(42) {
		t.Errorf("unknown settings keys lost: %v", s.Settings.Raw)
	}
}

func TestSetSendsBodyAndAdoptsAnsweredState(t *testing.T) {
	f := newFake(t, stateDoc)
	c := newTestClient(f)
	s, err := c.Set(context.Background(), "claude", "model", "claude-opus-5-5")
	if err != nil {
		t.Fatal(err)
	}
	if got := f.bodies["/api/set"]; got != `{"agent":"claude","field":"model","value":"claude-opus-5-5"}` {
		t.Errorf("set body %q", got)
	}
	if s == nil || len(s.Agents) != 1 {
		t.Error("set did not answer with the state to adopt")
	}
}

func TestRescanCarriesNoBody(t *testing.T) {
	f := newFake(t, stateDoc)
	c := newTestClient(f)
	if _, err := c.Rescan(context.Background()); err != nil {
		t.Fatal(err)
	}
	if b := f.bodies["/api/agents/rescan"]; b != "" {
		t.Errorf("rescan sent a body %q", b)
	}
}

func TestDriftAnswersEmptyWhenNothingDrifted(t *testing.T) {
	f := newFake(t, stateDoc)
	c := newTestClient(f)
	d, err := c.Drift(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(d) != 0 {
		t.Errorf("drift %v", d)
	}
}

func TestWrongKeyIsAnAPIErrorNotASilentEmpty(t *testing.T) {
	f := newFake(t, stateDoc)
	c := New("http://127.0.0.1:3437", "the-wrong-key-000000000")
	srv := httptest.NewServer(f.handler)
	defer srv.Close()
	c.base = srv.URL
	_, err := c.State(context.Background())
	var api *APIError
	if !errors.As(err, &api) || api.Status != http.StatusUnauthorized {
		t.Fatalf("wrong key gave %v, want 401 APIError", err)
	}
}

func TestTimeoutsSurfaceAsErrors(t *testing.T) {
	f := newFake(t, stateDoc)
	srv := httptest.NewServer(f.handler)
	defer srv.Close()
	c := New(srv.URL, "test-key-0123456789")
	ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
	defer cancel()
	time.Sleep(time.Millisecond)
	if _, err := c.State(ctx); err == nil {
		t.Fatal("an expired context answered as success")
	}
}

func TestCookieNameFollowsTheInstancesPort(t *testing.T) {
	if got := New("http://127.0.0.1:3499", "k").cookieName(); got != "magpie_web_3499" {
		t.Errorf("cookie name %q", got)
	}
	if got := New("http://127.0.0.1", "k").cookieName(); got != "magpie_web_"+DefaultPort {
		t.Errorf("cookie name %q", got)
	}
}

// TestLive runs the client against a real instance when one is offered:
//
//	MAGPIE_WEB_PROBE_URL=http://127.0.0.1:3437 \
//	MAGPIE_WEB_PROBE_KEY=probe-key-0123456789abcdef go test ./internal/magpieclient
//
// The key stays in the environment, never in the tree.
func TestLive(t *testing.T) {
	base, key := probeTarget()
	if base == "" {
		t.Skip("no MAGPIE_WEB_PROBE_URL; the contract tests run against the fake")
	}
	c := New(base, key)
	info, err := c.Probe(context.Background())
	if err != nil {
		t.Fatalf("probe: %v", err)
	}
	if info.Version == "" || info.Gateway == "" {
		t.Errorf("probe read an empty document: %+v", info)
	}
	s, err := c.State(context.Background())
	if err != nil {
		t.Fatalf("state: %v", err)
	}
	if len(s.Agents) == 0 {
		t.Error("a live instance answered with no agents at all")
	}
	if _, err := c.Drift(context.Background()); err != nil {
		t.Fatalf("drift: %v", err)
	}
}

func probeTarget() (base, key string) {
	return os.Getenv("MAGPIE_WEB_PROBE_URL"), os.Getenv("MAGPIE_WEB_PROBE_KEY")
}
