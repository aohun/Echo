//go:build nogui && echo

package shell

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/yetone/magpie/internal/provider"
)

// TestImportFingerprintIgnoresTheGeneratedID: a fingerprint must not change
// when settle gives the configuration another id — the id is regenerated as
// other sources arrive, and a dismissal would silently stop matching. The
// rest of the provider (its key, endpoints, name) must change it.
func TestImportFingerprintIgnoresTheGeneratedID(t *testing.T) {
	it := provider.AppImport{
		Ref:      "claude/settings",
		Provider: provider.Provider{ID: "imported-2", Name: "Relay", Key: "sk-x", Chat: "https://relay.example/v1"},
	}
	a := importFingerprint("claude-code", it)

	it.Provider.ID = "imported-3" // settle's id moved under it
	b := importFingerprint("claude-code", it)
	if a != b {
		t.Error("the fingerprint moved with the generated id: a dismissal would stop matching")
	}

	it.Provider.Key = "sk-y"
	if a == importFingerprint("claude-code", it) {
		t.Error("the fingerprint ignored the key: two configurations would share one dismissal")
	}

	if a == importFingerprint("codex", it) {
		t.Error("the fingerprint ignored the source: two apps' reads would share one dismissal")
	}
}

// TestDiscoveryIgnoresSurviveAChangeAndARead: putting configurations away
// keeps them quiet, and the file they are kept in is read back by a fresh
// run — magpie's rule that a dismissal survives the window.
func TestDiscoveryIgnoresSurviveAChangeAndARead(t *testing.T) {
	dir := t.TempDir()
	setPaths(dir)

	// A read before anything is saved: none are put off.
	if got := len(ignoredRead()); got != 0 {
		t.Fatalf("a fresh run put off %d configurations, want 0", got)
	}

	ignoredSave([]string{"abc", "def"})
	ignoredSave([]string{"ghi"}) // a later dismissal keeps the earlier ones

	ignoredIDs.mu.Lock()
	ignoredIDs.read = false // a fresh run of the app reads the file again
	ignoredIDs.mu.Unlock()
	ids := ignoredRead()
	for _, want := range []string{"abc", "def", "ghi"} {
		if !ids[want] {
			t.Errorf("put-off fingerprint %q did not survive the run", want)
		}
	}

	// The file is a JSON array of strings, nothing else in it: opaque
	// fingerprints, never credentials or paths.
	b, err := os.ReadFile(filepath.Join(dir, "provider-discovery-ignored.json"))
	if err != nil {
		t.Fatalf("reading the file back: %v", err)
	}
	var got []string
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("the file is not a JSON array of strings: %v", err)
	}
	if len(got) != 3 {
		t.Errorf("the file holds %d fingerprints, want 3", len(got))
	}
}

// TestProvidersPageDrawsDiscoveryHint: with configurations found and
// nothing else on the page, the empty page says how many were found and
// asks, and putting them away clears the ask.
func TestProvidersPageDrawsDiscoveryHint(t *testing.T) {
	dir := t.TempDir()
	setPaths(dir)

	app := NewForTest()
	app.LoadNow()
	// The look for other apps' configurations is already done: what it
	// found is set below, and a fresh scan (which a kick would start)
	// reads the test home, not a fixture.
	app.pvDiscoveryAt = time.Now()
	app.pvDiscovery.candidates = []discoveryCandidate{
		{"fp-1", "Codex"},
		{"fp-2", "CC Switch"},
	}
	tt := ui.NewTester(app.ViewForTest(), 1180, 780)
	app.ShowForTest(PageProviders)
	tt.Frame()

	for _, want := range []string{"发现 2 项本地供应商配置", "查看并导入…", "忽略"} {
		if !tt.HasText(want) {
			t.Errorf("the empty page's hint is missing %q; texts were %v", want, tt.Texts())
		}
	}

	// Put them away: the ask is gone, and only the empty page's own words
	// are left of the discovery's.
	ignoredSave([]string{"fp-1", "fp-2"})
	tt.Frame()
	for _, gone := range []string{"发现 2 项本地供应商配置", "查看并导入…"} {
		if tt.HasText(gone) {
			t.Errorf("the hint stayed after it was put off: %q is still drawn", gone)
		}
	}
	if !tt.HasText("暂无内容") {
		t.Error("the empty page's own words are gone with the hint")
	}
}

// TestProvidersPageDrawsCompactHint: with providers of its own on the
// page, the discovery hint is the one compact link beside Add provider.
func TestProvidersPageDrawsCompactHint(t *testing.T) {
	dir := t.TempDir()
	setPaths(dir)

	app := NewForTest()
	app.data.Providers = []ProviderRow{{ID: "p1", Name: "Relay", Models: 2, Ready: true}}
	// The look already done: what it found is set here, no scan behind it.
	app.pvDiscoveryAt = time.Now()
	app.pvDiscovery.candidates = []discoveryCandidate{{"fp-1", "Codex"}}
	tt := ui.NewTester(app.ViewForTest(), 1180, 780)
	app.ShowForTest(PageProviders)
	tt.Frame()

	// Echo no longer shows a footer's compact import link; the page just
	// lists what it has and the discovery hint is not drawn. What is here
	// is the relay row and the standard chrome around it.
	if !tt.HasText("Relay") {
		t.Errorf("the row is missing; texts were %v", tt.Texts())
	}
	for _, gone := range []string{"Import local configurations (1)", "Ignore"} {
		if tt.HasText(gone) {
			t.Errorf("the compact hint or its actions still show: %q", gone)
		}
	}
}

// setPaths points the shell's own state at a test home, so a test of the
// dismissal writes neither into the reader's own nor into another test's.
// The read-back cache is dropped with it: a fresh run reads afresh.
func setPaths(dir string) {
	mygo.App.SetPath(mygo.PathUserData, dir)
	ignoredIDs.mu.Lock()
	ignoredIDs.read = false
	ignoredIDs.ids = nil
	ignoredIDs.mu.Unlock()
}
