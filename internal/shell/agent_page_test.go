//go:build nogui && echo

package shell

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestAgentPageRenderingAndSpacing(t *testing.T) {
	app := NewForTest()
	app.LoadNow()
	app.ShowForTest(PageAgents)

	tt := ui.NewTester(app.ViewForTest(), 1180, 780)
	tt.Frame()

	// Verify sidebar spacing (content must not hug sidebar, rect.X >= 232)
	if r, ok := tt.Find(app.text.S.AgLead); ok {
		if r.X < 232 {
			t.Errorf("agLead rect X = %v, expected at least 232 (offset from sidebar)", r.X)
		}
	} else {
		t.Fatalf("agLead text not found on Agent page")
	}

	// Verify detected agents are rendered
	if len(app.data.Agents) > 0 {
		firstAgent := app.data.Agents[0]
		if !tt.HasText(firstAgent.Name) {
			t.Errorf("expected first agent %s to be rendered", firstAgent.Name)
		}
	}

	// Test Agent Modal Dialog for Claude Code
	app.agModalAgentID = "claude"
	tt.Frame()
	if !tt.HasText(app.text.S.AgModalClaudeTitle) {
		t.Errorf("expected modal title '%s' to be rendered", app.text.S.AgModalClaudeTitle)
	}
	if !tt.HasText(app.text.S.AgTiersSection) {
		t.Errorf("expected tiers section '%s' to be rendered", app.text.S.AgTiersSection)
	}
	if !tt.HasText("Opus") || !tt.HasText("Sonnet") || !tt.HasText("Haiku") {
		t.Errorf("expected Claude Code tiers (Opus, Sonnet, Haiku) in modal")
	}
	// Verify tabs, left rail, and footer on Claude modal
	if !tt.HasText(app.text.S.AgAllModels) || !tt.HasText(app.text.S.AgEnabledModels) || !tt.HasText(app.text.S.AgOrderModels) {
		t.Errorf("expected tabs (All, Shown, Order) in model modal")
	}
	if !tt.HasText(app.text.S.AgAllProviders) {
		t.Errorf("expected '%s' in left rail of model modal", app.text.S.AgAllProviders)
	}
	if !tt.HasText(app.text.S.AgOnlyPicked) {
		t.Errorf("expected '%s' in footer switch", app.text.S.AgOnlyPicked)
	}
	if !tt.HasText(app.text.S.AgDisableAll) || !tt.HasText(app.text.S.AgEnableAll) {
		t.Errorf("expected '%s' and '%s' in footer", app.text.S.AgDisableAll, app.text.S.AgEnableAll)
	}

	// Test switching to Order tab (agModalFilter = 2)
	app.agModalFilter = 2
	tt.Frame()
	if !tt.HasText(app.text.S.AgDefaultOrder) {
		t.Errorf("expected '%s' in footer under Order tab", app.text.S.AgDefaultOrder)
	}
	app.agModalFilter = 0

	// Test modal for another detected agent if available
	if len(app.data.Agents) > 0 {
		ag := app.data.Agents[0]
		app.agModalAgentID = ag.ID
		tt.Frame()
		expectedTitle := strings.ReplaceAll(app.text.S.AgModalTitle, "{agent}", ag.Name)
		if ag.ID == "claude" {
			expectedTitle = app.text.S.AgModalClaudeTitle
		}
		if !tt.HasText(expectedTitle) {
			t.Errorf("expected title '%s' for agent %s", expectedTitle, ag.Name)
		}
		if !tt.HasText(app.text.S.AgAllProviders) {
			t.Errorf("expected '%s' in left rail", app.text.S.AgAllProviders)
		}

		// Test modal search filtering
		app.agModalSearch = "non-existent-model-query-xyz"
		tt.Frame()
		if !tt.HasText(app.text.S.NoData) {
			t.Errorf("expected '%s' when search query matches no models", app.text.S.NoData)
		}
		app.agModalSearch = ""

		// Test provider group filtering
		app.agModalGroup = "non-existent-provider-group-abc"
		tt.Frame()
		if !tt.HasText(app.text.S.NoData) {
			t.Errorf("expected '%s' when selected provider group has no models", app.text.S.NoData)
		}
		app.agModalGroup = ""
		tt.Frame()
	}

	// Close modal
	app.agModalAgentID = ""
	tt.Frame()
	if tt.HasText(app.text.S.AgModalClaudeTitle) {
		t.Errorf("expected modal to be closed")
	}

	// Test Drag and Reorder
	if len(app.data.Agents) >= 2 {
		origFirst := app.data.Agents[0].ID
		origSecond := app.data.Agents[1].ID

		// Reorder 0 to 1
		app.commitAgentReorder(0, 1)
		tt.Frame()

		if app.data.Agents[0].ID != origSecond || app.data.Agents[1].ID != origFirst {
			t.Errorf("expected agents to swap order: got %s, %s; want %s, %s",
				app.data.Agents[0].ID, app.data.Agents[1].ID, origSecond, origFirst)
		}

		// Swap back
		app.commitAgentReorder(1, 0)
		tt.Frame()

		// Test reorderAgentByID
		app.reorderAgentByID(origFirst, origSecond)
		tt.Frame()
		if app.data.Agents[0].ID != origSecond || app.data.Agents[1].ID != origFirst {
			t.Errorf("expected agents to swap via reorderAgentByID: got %s, %s; want %s, %s",
				app.data.Agents[0].ID, app.data.Agents[1].ID, origSecond, origFirst)
		}
		app.reorderAgentByID(origSecond, origFirst)
		tt.Frame()
	}

	// Test install section fold / unfold
	app.agInstallFolded = false
	tt.Frame()

	for _, w := range []int{900, 1180, 1400} {
		tt.SetSize(w, 780)
		tt.Frame()
		if len(app.data.AgentInstalls) > 0 {
			inst := app.data.AgentInstalls[0]
			if inst.Command != "" {
				rect, found := tt.Find(inst.Command)
				if !found {
					t.Logf("Command snippet not found for width %d", w)
				} else {
					if rect.X < 232 {
						t.Errorf("width %d command snippet rect.X = %v, expected at least 232", w, rect.X)
					}
				}
			}
		}
	}
}

func TestAgentPageCacheAndFastLoad(t *testing.T) {
	d := readData(nil, false)
	if len(d.Agents) == 0 {
		t.Logf("no agents detected in test environment, but readData succeeded")
	}

	cached, ok := loadDataCache()
	if !ok {
		t.Fatalf("expected loadDataCache to succeed after readData")
	}
	if len(cached.Agents) != len(d.Agents) {
		t.Errorf("cached agents count = %d, want %d", len(cached.Agents), len(d.Agents))
	}

	app := NewForTest()
	app.loadCLIsAsync()
}
