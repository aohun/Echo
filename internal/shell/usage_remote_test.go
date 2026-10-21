//go:build nogui && echo

package shell

// The Usage page's remote read against a fake instance (docs/08 M4).

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/yetone/magpie/internal/usageapi"
)

// fakeMagpieUsage is the fake instance with the Usage page's endpoints:
// it records what it was asked, so a test can see which reads went
// remote and which stayed home.
func fakeMagpieUsage(t *testing.T) (*httptest.Server, *map[string]int) {
	hits := map[string]int{}
	mux := http.NewServeMux()
	guard := func(rw http.ResponseWriter, r *http.Request, path string) bool {
		hits[path]++
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
		rw.Write([]byte(`{"version":"v9.9.9","gateway":"http://127.0.0.1:3497","dir":"/tmp/there"}`))
	})
	mux.HandleFunc("GET /api/usage", func(rw http.ResponseWriter, r *http.Request) {
		if !guard(rw, r, "usage") {
			return
		}
		if got := r.URL.Query().Get("period"); got != "month" {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
		rw.Write([]byte(`{"agents":[{"id":"claude","name":"Claude Code","icon":"claudecode-color"}],"models":[],"accounts":[],"callerKeys":[],"path":"~/log","calls":5}`))
	})
	mux.HandleFunc("GET /api/usage/requests", func(rw http.ResponseWriter, r *http.Request) {
		if !guard(rw, r, "usage/requests") {
			return
		}
		q := r.URL.Query()
		if q.Get("period") != "month" || q.Get("offset") != "10" || q.Get("limit") != "25" {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
		if q.Get("agent") != "codex" {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
		rw.Write([]byte(`{"callerKeys":[],"period":"month","rows":[{"agent":"codex","agentName":"Codex","icon":"codex-color","providerName":"OpenAI","calls":2}],"offset":10,"total":1,"bucket":"day","series":[],"by":{},"through":{"calls":2},"direct":{},"reported":{},"agents":[],"providers":[],"purposes":[],"accounts":[]}`))
	})
	mux.HandleFunc("GET /api/usage/quotas", func(rw http.ResponseWriter, r *http.Request) {
		if !guard(rw, r, "usage/quotas") {
			return
		}
		rw.Write([]byte(`[{"id":"claude","name":"Claude Pro"}]`))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, &hits
}

func TestReadUsageRemoteOverviewAndQuotas(t *testing.T) {
	srv, hits := fakeMagpieUsage(t)
	c := withRemote(t, srv.URL)
	ud, ok := readUsageRemote(c, UsageQuery{Tab: UsTabOverview, Period: "month"})
	if !ok {
		t.Fatal("remote read failed against a live fake")
	}
	if len(ud.Overview.Agents) != 1 || ud.Overview.Agents[0].Name != "Claude Code" {
		t.Errorf("overview agents decoded as %+v", ud.Overview.Agents)
	}
	if ud.Overview.Totals.Calls != 5 {
		t.Errorf("overview totals %+v", ud.Overview.Totals)
	}
	if len(ud.Quotas) != 1 || ud.Quotas[0].Name != "Claude Pro" {
		t.Errorf("quota cards decoded as %+v", ud.Quotas)
	}
	if (*hits)["usage/quotas"] != 1 {
		t.Errorf("the overview read hit quotas %d times", (*hits)["usage/quotas"])
	}
}

func TestReadUsageRemoteLedgerCarriesTheQuery(t *testing.T) {
	srv, hits := fakeMagpieUsage(t)
	c := withRemote(t, srv.URL)
	q := UsageQuery{Tab: UsTabRequests, Period: "month", Offset: 10, Limit: 25, Agent: "codex"}
	ud, ok := readUsageRemote(c, q)
	if !ok {
		t.Fatal("remote read failed against a live fake")
	}
	if ud.Ledger.Total != 1 || len(ud.Ledger.Rows) != 1 {
		t.Fatalf("ledger decoded as %d rows / total %d", len(ud.Ledger.Rows), ud.Ledger.Total)
	}
	if ud.Ledger.Rows[0].AgentName != "Codex" {
		t.Errorf("row name %q", ud.Ledger.Rows[0].AgentName)
	}
	if ud.Ledger.Through.Calls != 2 {
		t.Errorf("through totals %+v", ud.Ledger.Through)
	}
	if (*hits)["usage/requests"] != 1 {
		t.Errorf("the ledger read hit requests %d times", (*hits)["usage/requests"])
	}
}

func TestReadUsageDataSessionsTabStaysLocal(t *testing.T) {
	srv, hits := fakeMagpieUsage(t)
	withRemote(t, srv.URL)
	ud := ReadUsageData(UsageQuery{Tab: UsTabSessions, Period: "month"})
	if (*hits)["usage"] != 0 || (*hits)["usage/requests"] != 0 {
		t.Error("the sessions segment went remote; it reads this machine's session files")
	}
	// And the embedded read ran without error against the test home.
	_ = ud
}

func TestReadUsageDataFallsBackWhenTheInstanceIsGone(t *testing.T) {
	srv, _ := fakeMagpieUsage(t)
	withRemote(t, srv.URL)
	srv.Close()
	ud := ReadUsageData(UsageQuery{Tab: UsTabOverview, Period: "month"})
	// The embedded read ran: the query came back and the local summary
	// was taken (an empty test home sums to zero, but the path names it).
	if ud.Query.Tab != UsTabOverview {
		t.Errorf("query lost: %+v", ud.Query)
	}
	if ud.Overview.Path == "" {
		t.Error("the embedded summary did not name its path")
	}
}

// The overview JSON the instance answers decodes into usageapi.Usage —
// the shape contract, spelled out where both sides live.
func TestUsageShapeContract(t *testing.T) {
	var u usageapi.Usage
	if err := json.Unmarshal([]byte(`{"agents":[],"models":[],"accounts":[],"callerKeys":[],"path":"p","calls":1}`), &u); err != nil {
		t.Fatal(err)
	}
	if u.Path != "p" {
		t.Errorf("path %q", u.Path)
	}
}
