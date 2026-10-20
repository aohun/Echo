//go:build nogui && echo

package shell

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestAgentRowWithoutConnectionHasNoModels checks the row of an agent that
// is not connected: it shows its name, its notice and the switch that
// connects it, and neither the model dropdown nor the details link — an
// agent that is not on Echo has no models to pick and no list to open.
func TestAgentRowWithoutConnectionHasNoModels(t *testing.T) {
	app := NewForTest()
	app.LoadNow()
	app.ShowForTest(PageAgents)

	if len(app.data.Agents) < 2 {
		t.Skip("no agents in this environment")
	}

	// the first row is connected and shows both; the second is not
	ag := &app.data.Agents[0]
	ag.Wired = true
	ag.CurrentName = "Qwen3.8-Max"
	ag.DetailsText = "模型 8 ›"
	off := &app.data.Agents[1]
	off.Wired = false
	off.CurrentName = "Gemini 3.8 Flash"
	off.DetailsText = "模型 9 ›"

	tt := ui.NewTester(app.ViewForTest(), 1180, 780)
	tt.Frame()

	if !tt.HasText(ag.CurrentName) {
		t.Fatalf("the connected row does not show its model")
	}
	if !tt.HasText(ag.DetailsText) {
		t.Fatalf("the connected row does not show its details")
	}

	// the row that is not connected shows neither
	if tt.HasText(off.CurrentName) {
		t.Errorf("the row that is not connected shows the model %q", off.CurrentName)
	}
	if tt.HasText(off.DetailsText) {
		t.Errorf("the row that is not connected shows the details %q", off.DetailsText)
	}
}
