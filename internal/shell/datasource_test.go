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
	"github.com/yetone/magpie/internal/provider"
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
	if g.Models != 3 {
		t.Errorf("row models %d", g.Models)
	}
	if g.Routed != 2 {
		t.Errorf("row routed %d, want the two distinct current agents", g.Routed)
	}
	if len(g.Calls) != 1 {
		t.Fatalf("row calls %d", len(g.Calls))
	}
	call := g.Calls[0]
	if call.Agent != "claude" || call.Model != "openai/gpt-5" || call.Status != 200 {
		t.Errorf("call decoded as %+v", call)
	}
	// The entries list is the one cut still to map.
	if g.Entries != nil {
		t.Error("remote row guessed at the models list; that maps in a later milestone")
	}
}

func TestRemoteProvidersRows(t *testing.T) {
	srv := fakeMagpie(t)
	withRemote(t, srv.URL)
	c := remoteGateway()
	rows, ok := readProvidersRemote(c)
	if !ok {
		t.Fatal("readProvidersRemote failed against a live fake")
	}
	if len(rows) != 3 {
		t.Fatalf("rows %d", len(rows))
	}
	keyed := rows[0]
	if keyed.ID != "openai" || keyed.Models != 3 || keyed.Draws != 1 {
		t.Errorf("keyed row: %+v", keyed)
	}
	if !keyed.Keyed || keyed.Key != "sk-…abc" || keyed.NeedsKey {
		t.Errorf("keyed row's key: keyed=%v key=%q needs=%v", keyed.Keyed, keyed.Key, keyed.NeedsKey)
	}
	if len(keyed.Agents) != 1 || keyed.Agents[0].Model != "openai/gpt-5" || keyed.Agents[0].Name != "Claude Code" {
		t.Errorf("keyed row's agents: %+v", keyed.Agents)
	}
	signed := rows[1]
	if !signed.SignedIn || signed.User != "w@example.com" || signed.Plan == "" {
		t.Errorf("signed-in row: signed=%v user=%q plan=%q", signed.SignedIn, signed.User, signed.Plan)
	}
	if signed.Key != "" || signed.NeedsKey {
		t.Errorf("signed-in row carries a key: %q needs=%v", signed.Key, signed.NeedsKey)
	}
	if len(signed.Agents) != 1 || signed.Agents[0].Group != "" || signed.Agents[0].Model != "claude/sonnet" {
		t.Errorf("signed-in row's agents: %+v", signed.Agents)
	}
	off := rows[2]
	if !off.Off || off.Models != 4 {
		t.Errorf("off row: off=%v models=%d", off.Off, off.Models)
	}
	if off.Keyed || !off.NeedsKey {
		t.Errorf("off row's key: keyed=%v needs=%v", off.Keyed, off.NeedsKey)
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

func TestRemoteRoutingGroups(t *testing.T) {
	srv := fakeMagpie(t)
	withRemote(t, srv.URL)
	rd := readRoutingData()
	if !rd.AutoGroupsOn {
		t.Error("auto groups: the document says found, the read says off")
	}
	if len(rd.Templates) == 0 {
		t.Error("templates stay embedded; the remote read dropped them")
	}
	if len(rd.Groups) != 2 {
		t.Fatalf("groups %d", len(rd.Groups))
	}
	fast := rd.Groups[0]
	if fast.ID != "fast" || fast.Routing != "manual" || !fast.Ready || fast.Pick != "openai/gpt-5:high" {
		t.Errorf("fast row: %+v", fast)
	}
	if len(fast.Weights) != 1 || fast.Weights["openai/gpt-5:high"] != 3 {
		t.Errorf("fast row's weights: %v", fast.Weights)
	}
	if len(fast.Members) != 2 {
		t.Fatalf("fast row's members %d", len(fast.Members))
	}
	m := fast.Members[0]
	if m.Model != "gpt-5" || m.Effort != "high" || m.ProviderName != "OpenAI" || m.AccountsOn != 2 ||
		!m.Ready || !m.CanFast || m.Context != 128000 || m.Weight != 3 || m.IsGroup {
		t.Errorf("fast row's model member: %+v", m)
	}
	g := fast.Members[1]
	if !g.IsGroup || g.Model != "quick" || g.ProviderName != "Quick" || g.AccountsOn != 1 || !g.Ready {
		t.Errorf("fast row's group member: %+v", g)
	}
	off := rd.Groups[1]
	if !off.Disabled || len(off.Members) != 1 || off.Members[0].Ready || off.Members[0].Model != "deepseek/r2" {
		t.Errorf("quick row: %+v", off)
	}
	if len(rd.AvailableModels) != 1 {
		t.Fatalf("available models %d", len(rd.AvailableModels))
	}
	am := rd.AvailableModels[0]
	if am.ID != "openai/gpt-5" || am.ProviderID != "openai" || am.ProviderName != "OpenAI" ||
		!am.CanFast || !am.Images || am.Context != 128000 {
		t.Errorf("available model: %+v", am)
	}
	if len(rd.DeciderModels) != 1 || rd.DeciderModels[0].ID != "jev/sonnet" {
		t.Errorf("decider models: %+v", rd.DeciderModels)
	}
	if len(rd.Pools) != 1 || rd.Pools[0].Kind != "account" || len(rd.Pools[0].Who) != 2 ||
		rd.Pools[0].ProviderID != "openai" || !rd.Pools[0].Sink {
		t.Errorf("pools: %+v", rd.Pools)
	}
}

func TestRemoteRoutingWritesGoToTheInstance(t *testing.T) {
	srv := fakeMagpie(t)
	withRemote(t, srv.URL)
	if c := remoteGateway(); c == nil {
		t.Fatal("the instance answered its probe, yet remote stayed off")
	}
	groupsAction = ""
	if err := routingFound(true); err != nil {
		t.Fatalf("found: %v", err)
	}
	if groupsAction != "/api/groups/found" {
		t.Errorf("found asked %q", groupsAction)
	}
	groupsAction = ""
	if err := routingSwitch("fast", false); err != nil {
		t.Fatalf("switch: %v", err)
	}
	if groupsAction != "/api/groups/switch" {
		t.Errorf("switch asked %q", groupsAction)
	}
	groupsAction = ""
	if err := routingDelete("fast"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if groupsAction != "/api/groups/delete" {
		t.Errorf("delete asked %q", groupsAction)
	}
	groupsAction = ""
	if err := routingSaveGroup(provider.Group{ID: "quick2", Name: "Quick 2"}, "quick"); err != nil {
		t.Fatalf("save: %v", err)
	}
	if groupsAction != "/api/groups/save" {
		t.Errorf("save asked %q", groupsAction)
	}
}

// groupsAction records the path of the last group action the fake was
// asked, "" when none — the tests read it to prove a write went remote.
var groupsAction string

func fakeMagpie(t *testing.T) *httptest.Server {
	mux := http.NewServeMux()
	guard := func(rw http.ResponseWriter, r *http.Request, path string) bool {
		ck, err := r.Cookie("magpie_web_" + portOf(r.Host))
		if err != nil || ck.Value != "test-key-0123456789" {
			rw.WriteHeader(http.StatusUnauthorized)
			return false
		}
		return true
	}
	mux.HandleFunc("GET /api/settings", func(rw http.ResponseWriter, r *http.Request) {
		if !guard(rw, r, "settings") {
			return
		}
		rw.Header().Set("Content-Type", "application/json")
		rw.Write([]byte(`{"version":"v9.9.9","gateway":"http://127.0.0.1:3497","dir":"/tmp/there"}`))
	})
	mux.HandleFunc("GET /api/providers", func(rw http.ResponseWriter, r *http.Request) {
		if !guard(rw, r, "providers") {
			return
		}
		rw.Header().Set("Content-Type", "application/json")
		rw.Write([]byte(`{"providers":[` +
			`{"id":"openai","name":"OpenAI","host":"api.openai.com","exposed":3,"draws":1,"key":{"set":true,"masked":"sk-…abc"},"ready":true,"agents":[{"id":"claude","name":"Claude Code","icon":"claudecode-color","current":true,"model":"openai/gpt-5"}],"models":[{},{},{}]},` +
			`{"id":"claude","name":"Claude","preset":"claude","account":{"agent":"claude","user":"w@example.com","plan":"pro"},"ready":true,"agents":[{"id":"codex","name":"Codex","icon":"codex-color","current":true,"model":"claude/sonnet"}]},` +
			`{"id":"deepseek","name":"DeepSeek","key":{"set":false,"masked":"","optional":false},"ready":false,"off":true,"agents":[],"models":[{},{},{},{}]}` +
			`],"presets":[],"excluded":[],"gateway":{"url":"http://127.0.0.1:3497","running":true,"mine":false,"version":"v9.9.9","models":3,"calls":[{"time":"2026-10-22T08:00:00Z","agent":"claude","model":"openai/gpt-5","provider":"openai","from":"anthropic","to":"openai","status":200,"ms":120,"ttft":40}]}}`))
	})
	mux.HandleFunc("GET /api/groups", func(rw http.ResponseWriter, r *http.Request) {
		if !guard(rw, r, "groups") {
			return
		}
		rw.Header().Set("Content-Type", "application/json")
		rw.Write([]byte(`{"groups":[` +
			`{"id":"fast","name":"Fast","routing":"manual","ready":true,"picked":"openai/gpt-5:high","weights":{"openai/gpt-5:high":3},` +
			`"memberInfo":[` +
			`{"id":"openai/gpt-5:high","of":"openai/gpt-5","effort":"high","ready":true,"provider":"openai","name":"OpenAI","icon":"openai-color","model":"gpt-5","on":2,"canFast":true,"context":128000,"efforts":["low","high"]},` +
			`{"id":"group/quick","group":true,"ready":true,"name":"Quick","icon":"claude-color","on":1}]},` +
			`{"id":"quick","name":"Quick","routing":"ordered","ready":false,"disabled":true,` +
			`"memberInfo":[{"id":"deepseek/r2","of":"deepseek/r2","ready":false}]}],` +
			`"models":[{"id":"openai/gpt-5","name":"GPT-5","provider":"openai","providerName":"OpenAI","icon":"openai-color","context":128000,"efforts":["low","high"],"canFast":true,"images":true}],` +
			`"pools":[{"provider":"openai","name":"OpenAI","icon":"openai-color","kind":"account","who":["a@x.com","b@x.com"],"routing":"rotate","affinity":"session","sink":true}],` +
			`"deciders":[{"id":"jev/sonnet","name":"Jev","provider":"jev","providerName":"Jev","icon":"jev-color"}],` +
			`"found":true}`))
	})
	mux.HandleFunc("POST /api/groups/{action}", func(rw http.ResponseWriter, r *http.Request) {
		if !guard(rw, r, "groups action") {
			return
		}
		groupsAction = r.URL.Path
		rw.Header().Set("Content-Type", "application/json")
		rw.Write([]byte(`{"groups":[],"models":[],"pools":[],"deciders":[],"found":true}`))
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
