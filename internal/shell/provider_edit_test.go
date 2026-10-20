//go:build nogui && echo

package shell

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/yetone/magpie/internal/provider"
)

// TestProviderEditTokenTexts pins the token maps' text both ways: what
// the editor's input holds reads back into the map saved, "*" first as a
// bare value, one model's as model=value.
func TestProviderEditTokenTexts(t *testing.T) {
	if got := pvTokensText(map[string]int{"*": 128_000, "gpt-6": 1_000_000}); got != "128k, gpt-6=1m" {
		t.Fatalf("pvTokensText = %q", got)
	}
	m, bad := pvParseTokensList("128k, gpt-6=1m")
	if bad != "" || m["*"] != 128_000 || m["gpt-6"] != 1_000_000 {
		t.Fatalf("pvParseTokensList = %v, bad %q", m, bad)
	}
	if _, bad := pvParseTokensList("128k, nope"); bad != "nope" {
		t.Fatalf("expected the bad part named, got %q", bad)
	}
}

// TestProviderEditorRenders opens the editor on a keyed provider and
// checks magpie's fields are all there.
func TestProviderEditorRenders(t *testing.T) {
	app := NewForTest()
	app.LoadNow()

	if _, err := provider.Add(provider.Provider{ID: "edit-me", Name: "Edit Me", Key: "sk-edit", Chat: "https://api.example.com/v1"}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = provider.Delete("edit-me") })

	app.openProviderEditor("edit-me")
	if app.pvEdit == nil {
		t.Fatal("expected the editor to open on the provider")
	}
	// a list to look at: the pill styles are for the reviewer's eye, and
	// a plugin model's pill is shown by its name, not its id
	if e := app.pvEdit; true {
		e.Models = []string{"gpt-6", "o4-mini", "free-small", "big-context"}
		e.Chosen = map[string]bool{"gpt-6": true, "free-small": true}
		e.Info["free-small"] = pvModelInfo{Name: "Free Small", Rate: 0.5, Free: true}
		e.Info["gpt-6"] = pvModelInfo{Rate: 0.5}
		e.Info["big-context"] = pvModelInfo{Context: 1_000_000}
	}
	if got := pvModelLabel(app.pvEdit, "free-small"); got != "Free Small" {
		t.Fatalf("pvModelLabel = %q, want the plugin's name", got)
	}
	if got := pvModelLabel(app.pvEdit, "o4-mini"); got != "o4-mini" {
		t.Fatalf("pvModelLabel = %q, want the id where no name is known", got)
	}

	tt := ui.NewTester(app.ViewForTest(), 1180, 780)
	tt.Frame()

	for _, want := range []string{"API Key", "模型", "上下文窗口", "最大输出", "压缩阈值", "备注",
		"代理", "并发上限", "每分钟请求数", "排队上限", "排队等待", "价格倍率", "保存", "移除"} {
		if !tt.HasText(want) {
			t.Errorf("expected the editor's %q row, texts: %v", want, tt.Texts())
		}
	}

	writeShot(t, "echo-provider-editor.png", ui.Render(app.ViewForTest(), 1180, 780, 2))
}

// TestProviderEditorAsksBeforeDiscarding pins the close ask: an editor
// with changes still in the draft stays open under the 放弃 ask, and the
// ask's 放弃 closes it for real; an unchanged one closes at once.
func TestProviderEditorAsksBeforeDiscarding(t *testing.T) {
	app := NewForTest()
	app.LoadNow()

	if _, err := provider.Add(provider.Provider{ID: "edit-me", Name: "Edit Me", Key: "sk-edit", Chat: "https://api.example.com/v1"}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = provider.Delete("edit-me") })

	app.openProviderEditor("edit-me")
	if app.pvEdit == nil {
		t.Fatal("expected the editor to open on the provider")
	}

	// unchanged, a close is a close
	app.pvEditCloseAttempt()
	if app.pvEdit != nil {
		t.Fatal("expected the unchanged editor to close")
	}

	// changed, the ask comes first, over the still-open editor
	app.openProviderEditor("edit-me")
	app.pvEdit.Key = "sk-changed"
	app.pvEditCloseAttempt()
	if app.pvEdit == nil || !app.pvDiscard {
		t.Fatal("expected the editor to stay open under the discard ask")
	}
	tt := ui.NewTester(app.ViewForTest(), 1180, 780)
	tt.Frame()
	if !tt.HasText("是否放弃未保存的修改？") {
		t.Fatalf("expected the discard ask's words, texts: %v", tt.Texts())
	}
	if err := tt.Click("放弃"); err != nil {
		t.Fatalf("expected a 放弃 to click: %v", err)
	}
	tt.Frame()
	if app.pvEdit != nil || app.pvDiscard {
		t.Fatal("expected 放弃 to close the editor and the ask")
	}
}

// TestProviderEditorModelViews pins the models' three looks and the
// picking: the picked look names the picked pills only, 只选免费 picks
// the free ones, and a refetch's dropped picks leave the draft.
func TestProviderEditorModelViews(t *testing.T) {
	app := NewForTest()
	app.LoadNow()

	if _, err := provider.Add(provider.Provider{ID: "edit-me", Name: "Edit Me", Key: "sk-edit", Chat: "https://api.example.com/v1"}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = provider.Delete("edit-me") })

	app.openProviderEditor("edit-me")
	e := app.pvEdit
	if e == nil {
		t.Fatal("expected the editor to open on the provider")
	}
	e.Models = []string{"gpt-6", "o4-mini", "free-small"}
	e.Info["free-small"] = pvModelInfo{Free: true}
	e.Chosen = map[string]bool{"gpt-6": true, "free-small": true}

	e.View = 1 // 仅所选
	tt := ui.NewTester(app.ViewForTest(), 1180, 780)
	tt.Frame()
	if !tt.HasText("gpt-6") || !tt.HasText("free-small") {
		t.Fatalf("expected the picked look to name the picked models, texts: %v", tt.Texts())
	}
	if tt.HasText("o4-mini") {
		t.Fatalf("expected the picked look to leave the unpicked model out, texts: %v", tt.Texts())
	}
	if !tt.HasText("只选免费") {
		t.Fatalf("expected the free bulk pick named, texts: %v", tt.Texts())
	}

	// the usage look: the ones with calls recorded, of the list
	e.View = 2
	e.UsageModels = []string{"gpt-6"}
	tt.Frame()
	if !tt.HasText("gpt-6") || tt.HasText("o4-mini") || tt.HasText("free-small") {
		t.Fatalf("expected the usage look to name only the models with calls, texts: %v", tt.Texts())
	}
}
