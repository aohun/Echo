//go:build nogui && echo

package shell

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestMovedSubscriptionSignInSaysWhatsWrong pins the fix for the silent
// click: a subscription moved onto its plugin signs in through the plugin
// (provider.StartSignIn refuses, and StartSignIn's error used to be
// swallowed, so Qoder CN's 登录 did nothing at all). Without the plugin
// installed, the sheet says so; with it, the ask begins.
func TestMovedSubscriptionSignInSaysWhatsWrong(t *testing.T) {
	app := NewForTest()
	app.LoadNow()

	app.startPluginSignIn("qoder-cn")
	if app.pvAsk != nil {
		t.Fatalf("expected no ask without the plugin installed, got %+v", app.pvAsk)
	}
	if app.pvSignIn == nil || app.pvSignIn.Err == "" {
		t.Fatalf("expected the plugin's absence to be said in the sheet, got %+v", app.pvSignIn)
	}

	// the sheet's draw with the error showing survives a frame
	app.pvAdding = &addSheet{Picker: &addPick{Kind: AddSub, Key: "qoder-cn", Name: "Qoder CN"}}
	tt := ui.NewTester(app.ViewForTest(), 1180, 780)
	tt.Frame()
	if !tt.HasText("插件没有装") {
		t.Errorf("expected the sheet to say the plugin isn't installed, texts: %v", tt.Texts())
	}
}

// TestRiskySubscriptionWarnsBeforeItsSignIn pins the flow magpie's add
// sheet has: a risky subscription's sign-in is confirmed first (the warn
// of an account the vendor may act on), and the editor's view carries no
// search field — the search is the list's own.
func TestRiskySubscriptionWarnsBeforeItsSignIn(t *testing.T) {
	app := NewForTest()
	app.LoadNow()

	var qoder addPick
	for _, p := range addPicks() {
		if p.Key == "qoder-cn" {
			qoder = p
		}
	}
	if !qoder.Risky || qoder.RiskNote == "" {
		t.Fatalf("expected qoder-cn to be risky with its note, got %+v", qoder)
	}

	app.pvAdding = &addSheet{Picker: &qoder, Risk: "qoder-cn"}
	tt := ui.NewTester(app.ViewForTest(), 1180, 780)
	tt.Frame()

	if !tt.HasText("Qoder CN 账号可能被封禁") {
		t.Errorf("expected the warn's title, texts: %v", tt.Texts())
	}
	if !tt.HasText("仍然登录") {
		t.Errorf("expected the warn's confirm button, texts: %v", tt.Texts())
	}
	if tt.HasText(app.text.S.FindVendor) {
		t.Errorf("expected no search field over a pick's editor, texts: %v", tt.Texts())
	}
	if app.pvSignIn != nil || app.pvAsk != nil {
		t.Fatalf("expected the sign-in to wait for the confirm, got %+v / %+v", app.pvSignIn, app.pvAsk)
	}
}
