//go:build nogui && echo

package shell

// The seam's tests (docs/08 M3): with no environment the pages read
// embedded; with ECHO_REMOTE_MAGPIE_* pointing at a fake instance the
// Gateway row comes from it, and an instance that went away falls back.

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/yetone/magpie/internal/magpieclient"
)

func TestRemoteGatewayOffByDefault(t *testing.T) {
	resetRemote(t)
	if c := remoteGateway(); c != nil {
		t.Fatal("no environment, yet a remote client came up")
	}
}

func TestRemoteGatewayChoosesWhenTheInstanceAnswers(t *testing.T) {
	srv := fakeMagpie(t)
	withRemote(t, srv.URL)
	c := remoteGateway()
	if c == nil {
		t.Fatal("the instance answered its probe, yet remote stayed off")
	}
	g, ok := readGatewayRemote(c)
	if !ok {
		t.Fatal("readGatewayRemote failed against a live fake")
	}
	if g.URL != "http://127.0.0.1:3497" {
		t.Errorf("row URL %q", g.URL)
	}
	if !g.Running || g.Mine {
		t.Errorf("row says running=%v mine=%v, want running=true mine=false", g.Running, g.Mine)
	}
	if g.Version != "v9.9.9" {
		t.Errorf("row version %q", g.Version)
	}
	if g.Entries != nil || g.Calls != nil {
		t.Error("remote row guessed at models or calls; those map in a later milestone")
	}
}

func TestRemoteGatewayFallsBackWhenTheInstanceIsGone(t *testing.T) {
	srv := fakeMagpie(t)
	withRemote(t, srv.URL)
	c := remoteGateway()
	if c == nil {
		t.Fatal("remote never came up")
	}
	srv.Close()
	if _, ok := readGatewayRemote(c); ok {
		t.Fatal("a dead instance answered")
	}
	// readGateway itself falls back to embedded; the embedded read runs
	// in the test environment too, where no gateway serves — it must not
	// panic, whatever it finds.
	_ = readGateway()
}

func fakeMagpie(t *testing.T) *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/settings", func(rw http.ResponseWriter, r *http.Request) {
		ck, err := r.Cookie("magpie_web_" + portOf(r.Host))
		if err != nil || ck.Value != "test-key-0123456789" {
			rw.WriteHeader(http.StatusUnauthorized)
			return
		}
		rw.Header().Set("Content-Type", "application/json")
		rw.Write([]byte(`{"version":"v9.9.9","gateway":"http://127.0.0.1:3497","dir":"/tmp/there"}`))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func portOf(host string) string {
	for i := len(host) - 1; i >= 0; i-- {
		if host[i] == ':' {
			return host[i+1:]
		}
	}
	return ""
}

// withRemote points the seam at addr, rewinds the once, and rewinds it
// again when the test ends — a seam left pointing at a dead fake would
// leak into the package's other tests.
func withRemote(t *testing.T, addr string) *magpieclient.Client {
	t.Helper()
	t.Setenv(remoteEnvAddr, addr)
	t.Setenv(remoteEnvKey, "test-key-0123456789")
	resetRemote(t)
	t.Cleanup(func() { resetRemote(t) })
	return remoteGateway()
}

// resetRemote rewinds remoteGateway's once, so each test starts clean.
func resetRemote(t *testing.T) {
	t.Helper()
	remoteOnce = sync.Once{}
	remoteClient = nil
}
