//go:build nogui && echo

package shell

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestAgentModelDropdown verifies the row's model button: one width down
// the list, the boxes aligned (magpie's model column), and opening it
// shows the model list dialog, the one the details link opens — the
// enabled providers and their models.
func TestAgentModelDropdown(t *testing.T) {
	app := NewForTest()
	app.LoadNow()
	app.ShowForTest(PageAgents)

	tt := ui.NewTester(app.ViewForTest(), 1180, 780)
	tt.Frame()

	if len(app.data.Agents) < 3 {
		t.Fatalf("need three agents, have %d", len(app.data.Agents))
	}
	// the first rows picked models of different name lengths, as a real
	// account's do. A row that is not connected shows no dropdown at all,
	// so these are connected.
	for i := range 3 {
		app.data.Agents[i].Wired = true
	}
	app.data.Agents[0].CurrentName = "Deepseek-V4.1-Flash"
	app.data.Agents[1].CurrentName = "droid"
	app.data.Agents[2].CurrentName = "droid, Qwen3.8-Flash, Gemini…"
	tt.Frame()

	var widths []float32
	var xs []float32
	for i := range 3 {
		ag := app.data.Agents[i]
		r, ok := tt.Find(ag.CurrentName)
		if !ok {
			t.Fatalf("%s: dropdown text %q not found", ag.ID, ag.CurrentName)
		}
		widths = append(widths, r.W)
		xs = append(xs, r.X)
	}
	// one width down the list, the boxes aligned
	for i := 1; i < 3; i++ {
		if widths[i] != widths[0] || xs[i] != xs[0] {
			t.Errorf("dropdown %d not aligned with the first: widths %v, x %v", i, widths, xs)
		}
	}

	// the model button opens the model list dialog
	ag := app.data.Agents[0]
	r, ok := tt.Find(ag.CurrentName)
	if !ok {
		t.Fatalf("%s: dropdown not found", ag.ID)
	}
	tt.ClickAt(r.X+r.W/2, r.Y+r.H/2)
	tt.Frame()
	t.Logf("after click: agModalAgentID=%q", app.agModalAgentID)
	want := app.text.S.AgModalClaudeTitle
	if ag.ID != "claude" {
		want = strings.ReplaceAll(app.text.S.AgModalTitle, "{agent}", ag.Name)
	}
	if !tt.HasText(want) {
		t.Errorf("expected the model list dialog %q after clicking the dropdown", want)
	}
	if !tt.HasText(app.text.S.AgAllProviders) {
		t.Errorf("expected the providers rail in the dialog")
	}
}
