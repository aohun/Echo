//go:build nogui && echo

package shell

// The Agents page's remote read against a fake instance (docs/08 M4):
// the rows, the installs and the schemes the state document builds, a
// write asked of the instance, and the embedded read back when the
// instance is gone.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

// agentFakeState is the state document the fake answers, modelled on
// magpie's own stateJSON: a wired agent with a model field, an effort
// and the model count; a wired agent whose config drifted; two schemes.
const agentFakeState = `{
  "agents": [
    {
      "id": "wired", "name": "Wired Agent", "icon": "codex-color", "path": "~/fake/.codex", "wired": true,
      "fields": [
        {"key": "model", "label": "model", "value": "fake/gpt-5", "options": [
          {"value": "fake/gpt-5", "label": "GPT-5", "icon": "openai-color"},
          {"value": "fake/m2", "label": "M2", "icon": "claude-color"}
        ]},
        {"key": "effort", "label": "effort", "value": "high"}
      ],
      "models": {"shown": 3, "listed": 5, "names": ["GPT-5", "M2", "M3"],
        "by": [{"name": "Fake", "icon": "openai-color", "n": 2}, {"name": "路由组", "icons": ["openai-color", "claude-color"], "n": 1}]}
    },
    {
      "id": "drifted", "name": "Drifted Agent", "icon": "claude-color", "path": "~/fake/.claude", "wired": true,
      "drift": {"kind": "unwired", "field": "model", "want": "fake/sonnet", "detail": "config rewritten"},
      "fields": [{"key": "model", "label": "model", "value": "gpt-5.4", "options": [{"value": "gpt-5.4", "label": "gpt-5.4"}]}]
    }
  ],
  "profiles": [
    {"name": "方案1", "summary": "wired fake/gpt-5", "agents": [
      {"id": "wired", "name": "Wired Agent", "fields": [
        {"key": "model", "value": "fake/gpt-5"},
        {"key": "effort", "value": "high"}
      ]}
    ]},
    {"name": "方案2", "summary": "", "agents": []}
  ],
  "settings": {"agentOrder": ["drifted", "wired"]}
}`

// fakeMagpieAgents is the fake instance with the Agents page's
// endpoints; it records what it was asked, so a test can see which
// writes went remote and which stayed home.
func fakeMagpieAgents(t *testing.T) (*httptest.Server, *atomic.Int64, *atomic.Int64, *atomic.Int64) {
	var sets, rescans, connects atomic.Int64
	mux := http.NewServeMux()
	guard := func(rw http.ResponseWriter, r *http.Request) bool {
		ck, err := r.Cookie("magpie_web_" + portOf(r.Host))
		if err != nil || ck.Value != "test-key-0123456789" {
			rw.WriteHeader(http.StatusUnauthorized)
			return false
		}
		return true
	}
	mux.HandleFunc("GET /api/settings", func(rw http.ResponseWriter, r *http.Request) {
		if !guard(rw, r) {
			return
		}
		rw.Header().Set("Content-Type", "application/json")
		rw.Write([]byte(`{"version":"v9.9.9","gateway":"http://127.0.0.1:3497","dir":"/tmp/there"}`))
	})
	mux.HandleFunc("GET /api/state", func(rw http.ResponseWriter, r *http.Request) {
		if !guard(rw, r) {
			return
		}
		rw.Header().Set("Content-Type", "application/json")
		rw.Write([]byte(agentFakeState))
	})
	mux.HandleFunc("POST /api/set", func(rw http.ResponseWriter, r *http.Request) {
		if !guard(rw, r) {
			return
		}
		var in struct{ Agent, Field, Value string }
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.Agent == "" || in.Field == "" {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
		sets.Add(1)
		rw.Header().Set("Content-Type", "application/json")
		rw.Write([]byte(agentFakeState))
	})
	mux.HandleFunc("POST /api/agents/rescan", func(rw http.ResponseWriter, r *http.Request) {
		if !guard(rw, r) {
			return
		}
		rescans.Add(1)
		rw.Header().Set("Content-Type", "application/json")
		rw.Write([]byte(agentFakeState))
	})
	mux.HandleFunc("POST /api/agents/connect/{id}", func(rw http.ResponseWriter, r *http.Request) {
		if !guard(rw, r) {
			return
		}
		if r.PathValue("id") != "wired" {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
		connects.Add(1)
		rw.Header().Set("Content-Type", "application/json")
		rw.Write([]byte(agentFakeState))
	})
	mux.HandleFunc("POST /api/agents/disconnect/{id}", func(rw http.ResponseWriter, r *http.Request) {
		if !guard(rw, r) {
			return
		}
		if r.PathValue("id") != "wired" {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
		connects.Add(1)
		rw.Header().Set("Content-Type", "application/json")
		rw.Write([]byte(agentFakeState))
	})
	mux.HandleFunc("GET /api/drift", func(rw http.ResponseWriter, r *http.Request) {
		if !guard(rw, r) {
			return
		}
		rw.Header().Set("Content-Type", "application/json")
		rw.Write([]byte(`{"drifted":{"kind":"unwired","field":"model","want":"fake/sonnet"}}`))
	})
	mux.HandleFunc("GET /api/agents/install", func(rw http.ResponseWriter, r *http.Request) {
		if !guard(rw, r) {
			return
		}
		rw.Header().Set("Content-Type", "application/json")
		rw.Write([]byte(`[{"id":"gemini","name":"Gemini CLI","icon":"gemini","commands":[{"via":"npm","command":"npm install -g @google/gemini-cli"}]}]`))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, &sets, &rescans, &connects
}

func TestReadAgentsRemoteRows(t *testing.T) {
	srv, _, _, _ := fakeMagpieAgents(t)
	c := withRemote(t, srv.URL)
	rows, installs, profiles, ok := readAgentsRemote(c)
	if !ok {
		t.Fatal("readAgentsRemote failed against a live fake")
	}
	if len(rows) != 2 {
		t.Fatalf("rows %d", len(rows))
	}
	// the instance's own order, drifted before wired
	if rows[0].ID != "drifted" || rows[1].ID != "wired" {
		t.Fatalf("rows in order %s, %s", rows[0].ID, rows[1].ID)
	}
	wired := rows[1]
	if !wired.Found || !wired.Wired {
		t.Errorf("wired row: found=%v wired=%v", wired.Found, wired.Wired)
	}
	if wired.FieldKey != "model" || wired.CurrentVal != "fake/gpt-5" || wired.CurrentName != "GPT-5" || wired.CurrentIcon != "openai-color" {
		t.Errorf("wired row's pick: key=%q val=%q name=%q icon=%q", wired.FieldKey, wired.CurrentVal, wired.CurrentName, wired.CurrentIcon)
	}
	if wired.Effort != "high" {
		t.Errorf("wired row's effort %q", wired.Effort)
	}
	if wired.ModelCount != 3 || len(wired.ModelNames) != 3 {
		t.Errorf("wired row's models: count=%d names=%v", wired.ModelCount, wired.ModelNames)
	}
	if len(wired.Groups) != 2 || wired.Groups[0].Name != "Fake" || wired.Groups[0].Count != 2 || wired.Groups[1].Name != "路由组" || len(wired.Groups[1].Icons) != 2 {
		t.Errorf("wired row's groups: %+v", wired.Groups)
	}
	if len(wired.Options) != 2 || !wired.Options[0].Selected || wired.Options[1].Selected {
		t.Errorf("wired row's options: %+v", wired.Options)
	}
	if wired.Subtitle == "" {
		t.Error("wired row says nothing")
	}
	drifted := rows[0]
	if !drifted.Drift || drifted.DriftText == "" {
		t.Errorf("drifted row: drift=%v text=%q", drifted.Drift, drifted.DriftText)
	}
	if len(installs) != 1 || installs[0].ID != "gemini" || installs[0].Via != "Node.js + npm" || installs[0].Command == "" {
		t.Errorf("installs: %+v", installs)
	}
	if len(profiles) != 2 {
		t.Fatalf("profiles %d", len(profiles))
	}
	if !profiles[0].InUse {
		t.Errorf("方案1 matches the agents' values now, read as not in use")
	}
	if profiles[1].InUse {
		t.Errorf("方案2 names no agent, read as in use")
	}
}

func TestReadAgentPageDataServesRemoteAndAppendsTheRoster(t *testing.T) {
	srv, sets, _, _ := fakeMagpieAgents(t)
	withRemote(t, srv.URL)
	rows, installs, profiles := readAgentPageData()
	if len(rows) == 0 || rows[0].ID != "drifted" {
		t.Fatalf("the page's rows did not come from the instance: %d rows, first %v", len(rows), rows)
	}
	// the roster's rows follow: agents Echo knows that the instance's
	// state doesn't name
	var notFound int
	for _, r := range rows {
		if !r.Found {
			notFound++
			if r.ID == "wired" || r.ID == "drifted" {
				t.Errorf("found agent %s listed as not found", r.ID)
			}
		}
	}
	if notFound == 0 {
		t.Error("the roster's rows did not follow the instance's")
	}
	if len(installs) != 1 || len(profiles) != 2 {
		t.Errorf("installs %d profiles %d", len(installs), len(profiles))
	}
	if sets.Load() != 0 {
		t.Error("a read wrote to the instance")
	}
}

func TestAgentWritesGoToTheInstance(t *testing.T) {
	srv, sets, rescans, connects := fakeMagpieAgents(t)
	withRemote(t, srv.URL)

	if err := agentSet("wired", "model", "fake/m2"); err != nil {
		t.Fatalf("agentSet: %v", err)
	}
	if sets.Load() != 1 {
		t.Errorf("the set did not go remote: %d", sets.Load())
	}

	agentRescan()
	if rescans.Load() != 1 {
		t.Errorf("the rescan did not go remote: %d", rescans.Load())
	}

	if err := agentConnect("wired", true); err != nil {
		t.Fatalf("agentConnect on: %v", err)
	}
	if err := agentConnect("wired", false); err != nil {
		t.Fatalf("agentConnect off: %v", err)
	}
	if connects.Load() != 2 {
		t.Errorf("the switch did not go remote: %d", connects.Load())
	}

	// the drift pill sets the drifted field again, with the drift's own
	// want, not a guess
	if err := agentReapply("drifted"); err != nil {
		t.Fatalf("agentReapply: %v", err)
	}
	if sets.Load() != 2 {
		t.Errorf("the reapply did not go remote: %d", sets.Load())
	}

	// an agent of the instance's that drifts no more is done there
	if err := agentReapply("wired"); err != nil {
		t.Fatalf("agentReapply without drift: %v", err)
	}
	if sets.Load() != 2 {
		t.Errorf("the reapply of an undrifted agent wrote: %d", sets.Load())
	}
}

func TestAgentSeamFallsBackWhenTheInstanceIsGone(t *testing.T) {
	srv, _, _, _ := fakeMagpieAgents(t)
	withRemote(t, srv.URL)
	srv.Close()

	if _, _, _, ok := readAgentsRemote(remoteGateway()); ok {
		t.Fatal("a dead instance answered")
	}
	// readAgentPageData itself falls back to the embedded read, which
	// runs in the test home too — it must not panic, whatever it finds.
	rows, _, _ := readAgentPageData()
	_ = rows
	if err := agentSet("no-such-agent", "model", "fake/m2"); err == nil {
		t.Error("a set of an agent nobody knows succeeded")
	}
}
