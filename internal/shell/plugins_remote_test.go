//go:build nogui && echo

package shell

// The Plugins page's remote read and writes against a fake instance
// (docs/08 M4). The fake's JSON is modelled on magpie's own shapes
// (internal/gui/plugins.go's pluginsJSON and pluginEntryJSON over
// plugin.Entry).

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/yetone/magpie/internal/settings"
)

// fakePluginsMagpie is the fake instance with the Plugins page's
// endpoints: it records what it was asked, so a test can see which reads
// and writes went remote and which stayed home.
func fakePluginsMagpie(t *testing.T) (*httptest.Server, *map[string]int, *string) {
	hits := map[string]int{}
	var addBody string
	var mu sync.Mutex
	count := func(path string) {
		mu.Lock()
		hits[path]++
		mu.Unlock()
	}
	mux := http.NewServeMux()
	guard := func(rw http.ResponseWriter, r *http.Request, path string) bool {
		ck, err := r.Cookie("magpie_web_" + portOf(r.Host))
		if err != nil || ck.Value != "test-key-0123456789" {
			rw.WriteHeader(http.StatusUnauthorized)
			return false
		}
		count(path)
		return true
	}
	mux.HandleFunc("GET /api/settings", func(rw http.ResponseWriter, r *http.Request) {
		if !guard(rw, r, "settings") {
			return
		}
		rw.Write([]byte(`{"version":"v9.9.9","gateway":"http://127.0.0.1:3497","dir":"/tmp/there"}`))
	})
	// the installed set, modelled on pluginsState's document: one npm
	// plugin with what npm said of it, one switched off
	mux.HandleFunc("GET /api/plugins", func(rw http.ResponseWriter, r *http.Request) {
		if !guard(rw, r, "plugins") {
			return
		}
		rw.Write([]byte(`{
			"plugins": [{
				"spec": "@magpie-community/opencode-zed-auth",
				"version": "0.1.0",
				"latest": "0.2.0",
				"moved": ["zed"],
				"middleware": {"hooks": ["request"]},
				"isMiddleware": true,
				"middlewareOnly": true,
				"optionsExample": {"key": "v"}
			}, {
				"spec": "@magpie-community/opencode-cursor-auth",
				"off": true,
				"version": "1.0.0"
			}],
			"bun": true,
			"bunVersion": "1.3.0",
			"movable": [{"id": "grok", "name": "Grok", "package": "@magpie-community/opencode-grok-auth", "accounts": 2}],
			"mirror": true
		}`))
	})
	// the updates waiting, modelled on PendingUpdates's document
	mux.HandleFunc("GET /api/plugins/updates", func(rw http.ResponseWriter, r *http.Request) {
		if !guard(rw, r, "plugins/updates") {
			return
		}
		rw.Write([]byte(`{"checked": "2026-10-01T00:00:00Z", "waiting": [{"spec": "@magpie-community/opencode-zed-auth", "package": "@magpie-community/opencode-zed-auth", "version": "0.1.0", "latest": "0.2.0"}], "updated": []}`))
	})
	// add answers with the state, as the handler's writes do; the body
	// is recorded for the test to read
	mux.HandleFunc("POST /api/plugins/add", func(rw http.ResponseWriter, r *http.Request) {
		if !guard(rw, r, "plugins/add") {
			return
		}
		var in struct {
			Spec string `json:"spec"`
		}
		json.NewDecoder(r.Body).Decode(&in)
		mu.Lock()
		addBody = in.Spec
		mu.Unlock()
		rw.Write([]byte(`{"plugins":[],"bun":true,"bunVersion":"1.3.0","movable":[],"mirror":false}`))
	})
	// the check answers with what it found and the refreshed state
	mux.HandleFunc("POST /api/plugins/check", func(rw http.ResponseWriter, r *http.Request) {
		if !guard(rw, r, "plugins/check") {
			return
		}
		rw.Write([]byte(`{"at": "2026-10-01T00:00:00Z", "plugins": [{"spec": "@magpie-community/opencode-zed-auth", "package": "@magpie-community/opencode-zed-auth", "version": "0.1.0", "latest": "0.2.0", "status": "update"}], "state": {"plugins": []}}`))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, &hits, &addBody
}

// fakePluginsDead is a fake whose Plugins endpoints all fail: the reads
// and writes then fall back to the embedded ones.
func fakePluginsDead(t *testing.T) *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/settings", func(rw http.ResponseWriter, r *http.Request) {
		rw.Write([]byte(`{"version":"v9.9.9","gateway":"http://127.0.0.1:3497","dir":"/tmp/there"}`))
	})
	mux.HandleFunc("GET /api/plugins", func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(http.StatusInternalServerError)
	})
	mux.HandleFunc("GET /api/plugins/updates", func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(http.StatusInternalServerError)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestReadPluginsRemoteMapsTheDocument(t *testing.T) {
	srv, hits, _ := fakePluginsMagpie(t)
	c := withRemote(t, srv.URL)
	d, ok := readPluginsRemote(c)
	if !ok {
		t.Fatal("remote read failed against a live fake")
	}
	if (*hits)["plugins"] != 1 || (*hits)["plugins/updates"] != 1 {
		t.Errorf("the read asked plugins %d and updates %d times", (*hits)["plugins"], (*hits)["plugins/updates"])
	}
	if len(d.Installed) != 2 {
		t.Fatalf("installed rows %d", len(d.Installed))
	}
	row := d.Installed[0]
	if row.Spec != "@magpie-community/opencode-zed-auth" || row.Name != row.Spec {
		t.Errorf("row name: %+v", row)
	}
	if row.Version != "0.1.0" || row.Latest != "0.2.0" || row.Off {
		t.Errorf("row versions: %+v", row)
	}
	if row.Middleware == nil || len(row.Middleware.Hooks) != 1 || row.Middleware.Hooks[0] != "request" {
		t.Errorf("middleware decoded as %+v", row.Middleware)
	}
	if !row.IsMiddleware || !row.MiddlewareOnly {
		t.Errorf("middleware flags: isMiddleware=%v only=%v", row.IsMiddleware, row.MiddlewareOnly)
	}
	if len(row.Moved) != 1 || row.Moved[0] != "zed" {
		t.Errorf("moved decoded as %v", row.Moved)
	}
	if row.OptionsExample["key"] != "v" {
		t.Errorf("options example decoded as %v", row.OptionsExample)
	}
	// the document's providers are names alone: the subscription cut is
	// left unmapped, so the row carries neither subs nor the names
	if len(row.Providers) != 0 {
		t.Errorf("providers mapped from names alone: %+v", row.Providers)
	}
	if !d.Installed[1].Off || d.Installed[1].Error != "" {
		t.Errorf("second row decoded as %+v", d.Installed[1])
	}
	if !d.Bun || d.BunVer != "1.3.0" || !d.Mirror {
		t.Errorf("the page's head decoded as bun=%v ver=%q mirror=%v", d.Bun, d.BunVer, d.Mirror)
	}
	if len(d.Movable) != 1 || d.Movable[0].ID != "grok" || d.Movable[0].Accounts != 2 {
		t.Errorf("movable decoded as %+v", d.Movable)
	}
	if d.Updates != 1 || len(d.UpdateNames) != 1 || d.UpdateNames[0] != "@magpie-community/opencode-zed-auth" {
		t.Errorf("updates decoded as %d / %v", d.Updates, d.UpdateNames)
	}
	// the built-ins cut keeps the embedded read (no endpoint serves it):
	// the same rows the embedded read gives, and a filled On map
	rows, broken := pluginBuiltins()
	if len(d.Builtins) != len(rows) || d.Broken != broken {
		t.Errorf("built-ins cut is not the embedded read: %d rows, broken %q", len(d.Builtins), d.Broken)
	}
	if d.On == nil || d.OffPages == nil {
		t.Error("the built-ins cut left the navigation's maps nil")
	}
}

func TestPluginsAddActionGoesRemote(t *testing.T) {
	srv, hits, addBody := fakePluginsMagpie(t)
	withRemote(t, srv.URL)
	if err := plugAddSeam(context.Background(), "@magpie-community/opencode-zed-auth"); err != nil {
		t.Fatalf("add: %v", err)
	}
	if (*hits)["plugins/add"] != 1 {
		t.Errorf("the add was not asked of the instance (%d hits)", (*hits)["plugins/add"])
	}
	if *addBody != "@magpie-community/opencode-zed-auth" {
		t.Errorf("the add carried spec %q", *addBody)
	}
}

func TestPluginsCheckGoesRemote(t *testing.T) {
	srv, hits, _ := fakePluginsMagpie(t)
	withRemote(t, srv.URL)
	ck := plugCheckSeam(context.Background())
	if (*hits)["plugins/check"] != 1 {
		t.Errorf("the check was not asked of the instance (%d hits)", (*hits)["plugins/check"])
	}
	if ck.At.IsZero() || len(ck.Plugins) != 1 || ck.Plugins[0].Status != "update" {
		t.Errorf("check decoded as %+v", ck)
	}
}

// The mirror's embedded fallback runs this process's settings write, in
// the test home, when no instance is set.
func TestPluginsMirrorFallsBackToEmbedded(t *testing.T) {
	resetRemote(t)
	if err := plugMirrorSeam(true); err != nil {
		t.Fatalf("mirror: %v", err)
	}
	if !settings.Load().ChinaMirror {
		t.Error("the embedded mirror write did not land")
	}
	if err := plugMirrorSeam(false); err != nil {
		t.Fatalf("mirror back: %v", err)
	}
	if settings.Load().ChinaMirror {
		t.Error("the embedded mirror write did not turn back off")
	}
}

// With the environment pointing at an instance whose Plugins endpoints
// fail, the page read falls back to the embedded one and the test home
// answers it.
func TestReadPluginsDataFallsBackWhenRemoteFails(t *testing.T) {
	srv := fakePluginsDead(t)
	withRemote(t, srv.URL)
	d := readPluginsData()
	if len(d.Installed) != 0 {
		t.Errorf("embedded read in the test home has plugins: %+v", d.Installed)
	}
	if d.On == nil || d.OffPages == nil {
		t.Error("the embedded read left the navigation's maps nil")
	}
}
