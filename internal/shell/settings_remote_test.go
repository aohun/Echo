//go:build nogui && echo

package shell

// The Settings page's seam tests: the snap read off a magpie web
// instance's /api/settings, the save and the settings' own writes asked
// of it, and the embedded read and save still taken when the
// environment points at nothing. The fake's shapes are magpie's own —
// its settingsJSON (internal/gui/api.go) for the document, and the
// settings routes' request and answer bodies as their handlers read
// and write them.

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/yetone/magpie/internal/gui"
	"github.com/yetone/magpie/internal/settings"
)

// settingsAsked and settingsBody record the last write the fake was
// asked — its path and its body, "" when none — so a test can prove a
// write went remote, and what it carried.
var (
	settingsMu    sync.Mutex
	settingsAsked string
	settingsBody  string
)

func settingsRecord(path string, r *http.Request) {
	b, _ := io.ReadAll(r.Body)
	settingsMu.Lock()
	settingsAsked, settingsBody = path, string(b)
	settingsMu.Unlock()
}

func settingsAskedReset() {
	settingsMu.Lock()
	settingsAsked, settingsBody = "", ""
	settingsMu.Unlock()
}

// settingsDocJSON is the fake's /api/settings answer: the settings
// inline and the facts around them, the page's own rows' worth.
const settingsDocJSON = `{
  "theme": "light",
  "lang": "en",
  "proxy": "http://proxy.lan:8118",
  "port": 3497,
  "version": "v9.9.9",
  "dir": "~/.config/magpie",
  "gateway": "http://127.0.0.1:3497",
  "proxyNow": "http://proxy.lan:8118",
  "proxySource": "settings",
  "login": true,
  "lanURLs": ["http://192.168.1.10:3497"],
  "lanContainer": false,
  "githubTokenMask": "ghp_…abc",
  "githubTokenFrom": "settings",
  "fx": {"rate": 7.2, "stale": true},
  "codexUsers": ["w@example.com"],
  "codexWarmed": "2026-10-22T08:00:00Z",
  "workbuddy": true,
  "workbuddyCheckins": [{"user": "wb@x.com", "day": "2026-10-22", "credit": 5, "streak": 3, "msg": "ok"}],
  "visionAuto": "openai/gpt-5",
  "visionModels": [{"id": "openai/gpt-5", "name": "GPT-5", "provider": "openai", "providerName": "OpenAI", "icon": "openai-color"}],
  "titleModels": [{"id": "group/quick", "name": "Quick", "provider": "", "providerName": "Quick"}],
  "imageGenAuto": "",
  "imageGenModels": [{"id": "openai/img", "name": "Img", "provider": "openai", "providerName": "OpenAI"}],
  "searchProvider": "openai/gpt-5",
  "searchAuto": "openai/gpt-5",
  "searchUnused": "",
  "searchChoices": [{"id": "openai", "name": "OpenAI", "icon": "openai-color", "small": "gpt-5", "smallName": "GPT-5", "models": [{"id": "openai/gpt-5", "name": "GPT-5", "provider": "openai", "providerName": "OpenAI"}]}],
  "searchRelays": ["Relay"],
  "searchLeftOut": ["LeftOut"],
  "searchAPIs": [{"vendor": "tavily", "name": "Tavily", "key": "tvly-…abc", "url": "", "ready": true}],
  "searchVendors": [{"id": "tavily", "name": "Tavily", "keysURL": "https://tavily.com"}],
  "terminalApps": [{"id": "com.apple.Terminal", "name": "Terminal"}],
  "terminalDefault": "Terminal"
}`

// settingsFake serves the settings half of a magpie web instance: the
// document read, the settings save and the settings' own routes. The
// document answers every read; each write is recorded.
func settingsFake(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	guard := func(rw http.ResponseWriter, r *http.Request) bool {
		ck, err := r.Cookie("magpie_web_" + portOf(r.Host))
		if err != nil || ck.Value != "test-key-0123456789" {
			rw.WriteHeader(http.StatusUnauthorized)
			return false
		}
		return true
	}
	answer := func(rw http.ResponseWriter, body string) {
		rw.Header().Set("Content-Type", "application/json")
		rw.Write([]byte(body))
	}
	mux.HandleFunc("GET /api/settings", func(rw http.ResponseWriter, r *http.Request) {
		if !guard(rw, r) {
			return
		}
		answer(rw, settingsDocJSON)
	})
	mux.HandleFunc("POST /api/settings", func(rw http.ResponseWriter, r *http.Request) {
		if !guard(rw, r) {
			return
		}
		settingsRecord("/api/settings", r)
		answer(rw, settingsDocJSON)
	})
	mux.HandleFunc("POST /api/settings/search-api", func(rw http.ResponseWriter, r *http.Request) {
		if !guard(rw, r) {
			return
		}
		settingsRecord("/api/settings/search-api", r)
		answer(rw, settingsDocJSON)
	})
	mux.HandleFunc("POST /api/settings/search-key", func(rw http.ResponseWriter, r *http.Request) {
		if !guard(rw, r) {
			return
		}
		settingsRecord("/api/settings/search-key", r)
		answer(rw, `{"key":"tvly-real"}`)
	})
	mux.HandleFunc("POST /api/settings/login", func(rw http.ResponseWriter, r *http.Request) {
		if !guard(rw, r) {
			return
		}
		settingsRecord("/api/settings/login", r)
		answer(rw, settingsDocJSON)
	})
	mux.HandleFunc("POST /api/settings/lan", func(rw http.ResponseWriter, r *http.Request) {
		if !guard(rw, r) {
			return
		}
		settingsRecord("/api/settings/lan", r)
		answer(rw, settingsDocJSON)
	})
	mux.HandleFunc("POST /api/settings/port", func(rw http.ResponseWriter, r *http.Request) {
		if !guard(rw, r) {
			return
		}
		settingsRecord("/api/settings/port", r)
		answer(rw, `{"settings":{}}`)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestRemoteSettingsRead(t *testing.T) {
	srv := settingsFake(t)
	withRemote(t, srv.URL)
	snap, ok := readSettingsRemote(remoteGateway())
	if !ok {
		t.Fatal("the instance answered, yet the remote read said not")
	}
	if snap.S.Theme != "light" || snap.S.Proxy != "http://proxy.lan:8118" || snap.S.Port != 3497 {
		t.Errorf("the settings carried inline: %+v", snap.S)
	}
	if snap.Dir != "~/.config/magpie" || snap.Gateway != "http://127.0.0.1:3497" {
		t.Errorf("the document's facts: dir %q gateway %q", snap.Dir, snap.Gateway)
	}
	if snap.ProxyNow != "http://proxy.lan:8118" || snap.ProxySource != "settings" {
		t.Errorf("the proxy rows: now %q from %q", snap.ProxyNow, snap.ProxySource)
	}
	if !snap.Login || snap.GHMask != "ghp_…abc" || snap.GHFrom != "settings" {
		t.Errorf("the login and token rows: login %v mask %q from %q", snap.Login, snap.GHMask, snap.GHFrom)
	}
	if snap.FX.Rate != 7.2 || !snap.FX.Stale {
		t.Errorf("the fx rate: %+v", snap.FX)
	}
	if len(snap.VisionModels) != 1 || snap.VisionModels[0].ID != "openai/gpt-5" ||
		snap.VisionModels[0].Prov != "openai" || snap.VisionModels[0].PName != "OpenAI" {
		t.Errorf("the vision models mapped: %+v", snap.VisionModels)
	}
	if snap.VisionAuto != "openai/gpt-5" || len(snap.TitleModels) != 1 || snap.TitleModels[0].ID != "group/quick" {
		t.Errorf("the pickers: vision auto %q titles %+v", snap.VisionAuto, snap.TitleModels)
	}
	if len(snap.SearchAPIs) != 1 || snap.SearchAPIs[0].Vendor != "tavily" || snap.SearchAPIs[0].Key != "tvly-…abc" || !snap.SearchAPIs[0].Ready {
		t.Errorf("the search APIs: %+v", snap.SearchAPIs)
	}
	if len(snap.SearchVendors) != 1 || snap.SearchVendors[0].ID != "tavily" {
		t.Errorf("the search vendors: %+v", snap.SearchVendors)
	}
	if len(snap.SearchChoices) != 1 || snap.SearchChoices[0].SmallNam != "GPT-5" ||
		len(snap.SearchChoices[0].Models) != 1 || snap.SearchChoices[0].Models[0].ID != "openai/gpt-5" {
		t.Errorf("the search choices mapped: %+v", snap.SearchChoices)
	}
	if snap.SearchProvider != "openai/gpt-5" || len(snap.SearchRelays) != 1 || snap.SearchRelays[0] != "Relay" ||
		len(snap.SearchLeftOut) != 1 || snap.SearchLeftOut[0] != "LeftOut" {
		t.Errorf("the searcher rows: provider %q relays %v left out %v", snap.SearchProvider, snap.SearchRelays, snap.SearchLeftOut)
	}
	if len(snap.CodexUsers) != 1 || snap.CodexUsers[0] != "w@example.com" {
		t.Errorf("the codex users: %v", snap.CodexUsers)
	}
	if snap.CodexWarmed == nil || !snap.CodexWarmed.Equal(time.Date(2026, 10, 22, 8, 0, 0, 0, time.UTC)) {
		t.Errorf("the codex warm-up's last start: %v", snap.CodexWarmed)
	}
	if !snap.WorkBuddy || len(snap.WBCheckins) != 1 || snap.WBCheckins[0].User != "wb@x.com" {
		t.Errorf("the workbuddy rows: %v %+v", snap.WorkBuddy, snap.WBCheckins)
	}
	if len(snap.Terminals) != 1 || snap.Terminals[0].ID != "com.apple.Terminal" || snap.TerminalDefault != "Terminal" {
		t.Errorf("the terminals: %+v default %q", snap.Terminals, snap.TerminalDefault)
	}
	// the shell's own facts are read embedded in either mode
	if snap.Version != appVersion() {
		t.Errorf("the version is the shell's own, got %q", snap.Version)
	}
	if snap.Update != gui.UpdateState() {
		t.Errorf("the update state is the shell's own, got %+v", snap.Update)
	}
}

func TestRemoteSettingsSaveGoesToTheInstance(t *testing.T) {
	srv := settingsFake(t)
	withRemote(t, srv.URL)
	settingsAskedReset()
	if err := saveSettings(func(s *settings.Settings) { s.Theme = "dark" }); err != nil {
		t.Fatalf("save: %v", err)
	}
	if settingsAsked != "/api/settings" {
		t.Fatalf("the save asked %q", settingsAsked)
	}
	var posted settings.Settings
	if err := json.Unmarshal([]byte(settingsBody), &posted); err != nil {
		t.Fatalf("the save's body: %v (%s)", err, settingsBody)
	}
	if posted.Theme != "dark" {
		t.Errorf("the save's own change: theme %q", posted.Theme)
	}
	// the change was made on the instance's document, not this
	// process's: what the fake answered came back in the post
	if posted.Proxy != "http://proxy.lan:8118" {
		t.Errorf("the save carried the instance's settings: proxy %q", posted.Proxy)
	}
}

func TestRemoteSettingsWritesGoToTheInstance(t *testing.T) {
	srv := settingsFake(t)
	withRemote(t, srv.URL)

	settingsAskedReset()
	if err := searchAPIRemove("tavily"); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if settingsAsked != "/api/settings/search-api" {
		t.Errorf("the remove asked %q", settingsAsked)
	}
	var rm struct {
		Vendor string `json:"vendor"`
		Remove bool   `json:"remove"`
	}
	if err := json.Unmarshal([]byte(settingsBody), &rm); err != nil || rm.Vendor != "tavily" || !rm.Remove {
		t.Errorf("the remove's body: %s (%v)", settingsBody, err)
	}

	settingsAskedReset()
	if err := searchAPISet("exa", "exa-key", "https://exa.lan"); err != nil {
		t.Fatalf("set: %v", err)
	}
	if settingsAsked != "/api/settings/search-api" {
		t.Errorf("the set asked %q", settingsAsked)
	}

	settingsAskedReset()
	if err := settingsLogin(true); err != nil {
		t.Fatalf("login: %v", err)
	}
	if settingsAsked != "/api/settings/login" {
		t.Errorf("the login asked %q", settingsAsked)
	}

	settingsAskedReset()
	if err := settingsLAN(true, false); err != nil {
		t.Fatalf("lan: %v", err)
	}
	if settingsAsked != "/api/settings/lan" {
		t.Errorf("the lan asked %q", settingsAsked)
	}

	settingsAskedReset()
	res, err := settingsPort(3497)
	if err != nil {
		t.Fatalf("port: %v", err)
	}
	if settingsAsked != "/api/settings/port" {
		t.Errorf("the port asked %q", settingsAsked)
	}
	_ = res

	settingsAskedReset()
	key, err := searchKeyReveal("tavily")
	if err != nil || key != "tvly-real" {
		t.Errorf("the reveal asked %q, key %q err %v", settingsAsked, key, err)
	}
}

// TestSettingsEmbeddedStillTaken proves the page falls back: with no
// instance pointed at, the save lands in this process's settings.
func TestSettingsEmbeddedStillTaken(t *testing.T) {
	t.Setenv(remoteEnvAddr, "")
	resetRemote(t)
	t.Cleanup(func() { resetRemote(t) })

	if err := saveSettings(func(s *settings.Settings) { s.Theme = "dark" }); err != nil {
		t.Fatalf("save: %v", err)
	}
	if got := settings.Load().Theme; got != "dark" {
		t.Errorf("the embedded save did not land: theme %q", got)
	}
	snap := readSettings()
	if snap.Version != appVersion() || snap.S.Theme != "dark" {
		t.Errorf("the embedded read: version %q theme %q", snap.Version, snap.S.Theme)
	}
}

// TestRemoteSettingsSaveFallsBackWhenTheInstanceRefuses proves a
// refused remote save falls back to the embedded one — the seam's own
// rule, as the routing writes'.
func TestRemoteSettingsSaveFallsBackWhenTheInstanceRefuses(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/settings", func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Content-Type", "application/json")
		rw.Write([]byte(settingsDocJSON))
	})
	mux.HandleFunc("POST /api/settings", func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(http.StatusInternalServerError)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	withRemote(t, srv.URL)

	if err := saveSettings(func(s *settings.Settings) { s.Theme = "dark" }); err != nil {
		t.Fatalf("save: %v", err)
	}
	if got := settings.Load().Theme; got != "dark" {
		t.Errorf("the embedded save did not take the refused change: theme %q", got)
	}
}
