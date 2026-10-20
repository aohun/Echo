//go:build nogui && echo

package shell

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestAgentModelsLabel checks the words on the row's model dropdown: the
// models an agent is offered, two of them by name, and an ellipsis when
// there are more — magpie's rule for a list of models.
func TestAgentModelsLabel(t *testing.T) {
	cases := []struct {
		names []string
		want  string
	}{
		{nil, ""},
		{[]string{"Qwen3.8-Max"}, "Qwen3.8-Max"},
		{[]string{"Qwen3.8-Max", "Gemini 3.8 Flash"}, "Qwen3.8-Max, Gemini 3.8 Flash"},
		{[]string{"Qwen3.8-Max", "Gemini 3.8 Flash", "Auto"}, "Qwen3.8-Max, Gemini 3.8 Flash …"},
		{[]string{"a", "b", "c", "d", "e"}, "a, b …"},
	}
	for _, c := range cases {
		if got := agentModelsLabel(c.names); got != c.want {
			t.Errorf("agentModelsLabel(%v) = %q, want %q", c.names, got, c.want)
		}
	}
}

// TestAgentModelsSaid checks the panel's own wording for the same list: the
// first two by name, and the count of the rest, as magpie's panel says it.
func TestAgentModelsSaid(t *testing.T) {
	cases := []struct {
		names []string
		shown int
		want  string
	}{
		{nil, 0, ""},
		{[]string{"Qwen3.8-Max"}, 1, "Qwen3.8-Max"},
		{[]string{"Qwen3.8-Max", "Gemini 3.8 Flash"}, 2, "Qwen3.8-Max, Gemini 3.8 Flash"},
		{[]string{"Qwen3.8-Max", "Gemini 3.8 Flash", "Auto"}, 16, "Qwen3.8-Max, Gemini 3.8 Flash +14"},
	}
	for _, c := range cases {
		if got := agentModelsSaid(c.names, c.shown); got != c.want {
			t.Errorf("agentModelsSaid(%v, %d) = %q, want %q", c.names, c.shown, got, c.want)
		}
	}
}

// TestAgentModelDropdownShowsTheModels checks the row's dropdown draws the
// words of that rule, and that the whole list stands in its tip: the
// button shows two of the models, the rest are one hover away.
func TestAgentModelDropdownShowsTheModels(t *testing.T) {
	app := NewForTest()
	app.LoadNow()
	app.ShowForTest(PageAgents)

	if len(app.data.Agents) < 1 {
		t.Skip("no agent in this environment")
	}
	ag := &app.data.Agents[0]
	ag.Wired = true
	ag.ModelNames = []string{"Qwen3.8-Max", "Gemini 3.8 Flash", "Auto", "Fast +60"}
	ag.CurrentName = agentModelsLabel(ag.ModelNames)
	ag.CurrentIcon = ""

	tt := ui.NewTester(app.ViewForTest(), 1180, 780)
	tt.Frame()

	if !tt.HasText(ag.CurrentName) {
		t.Fatalf("the dropdown does not show %q", ag.CurrentName)
	}
	if ag.CurrentName != "Qwen3.8-Max, Gemini 3.8 Flash …" {
		t.Errorf("the dropdown shows %q, want the first two models and an ellipsis", ag.CurrentName)
	}
}
