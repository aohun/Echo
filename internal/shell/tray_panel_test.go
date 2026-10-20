//go:build nogui && echo

package shell

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/yetone/magpie/internal/provider"
	"github.com/yetone/magpie/internal/usage"
	"github.com/yetone/magpie/internal/usageapi"
)

// TestTrayPanelRenders draws the tray panel's two tabs over a read: the
// agents', and the allowances' with windows, an erroring account and the
// bar under it all. The panel is the tray's, so a draw here is what the
// menu bar icon opens, and a panic here is a popover that would take the
// app down on a machine with subscriptions in it.
func TestTrayPanelRenders(t *testing.T) {
	app := NewForTest()
	app.LoadNow()

	readAt := time.Now().Add(-time.Minute)
	resetsAt := time.Now().Add(2 * time.Hour)
	app.panelReadAt = readAt
	app.panelQuotas = []provider.SubscriptionQuota{
		{
			Provider: "claude", Name: "Claude", Icon: "claude", Plan: "Max",
			User: "me@example.com", ReadAt: &readAt,
			Windows: []provider.QuotaWindow{
				{Name: "5 小时", Used: 42, ResetsAt: &resetsAt},
				{Name: "本周", Used: 91},
			},
		},
		{
			Provider: "codex", Name: "Codex", Icon: "codex",
			User: "me@example.com", Error: "读取失败：401",
		},
		{
			// Antigravity's shape: a pool's own two windows, its models'
			// own said only in the tooltip, as magpie's panel is
			Provider: "antigravity", Name: "Antigravity", Icon: "antigravity",
			User: "me@example.com", ReadAt: &readAt,
			Windows: []provider.QuotaWindow{
				{Name: "Gemini 3.8 Flash (High)", Used: 100, Family: "Gemini", Pool: "Gemini"},
				{Name: "Gemini 3.8 Flash (Low)", Used: 100, Family: "Gemini", Pool: "Gemini"},
				{Name: "5 hours", Used: 19, ResetsAt: &resetsAt, Pool: "Gemini"},
				{Name: "7 days", Used: 30, Pool: "Gemini"},
			},
		},
	}

	tt := ui.NewTester(app.panelView, 400, 540)

	// the tab strip: the allowances alone. The other readings — the
	// agents, the usage, the routing — are the main window's business, and
	// their tabs are hidden here.
	app.panelTab = panelTabQuota
	tt.Frame()
	if !tt.HasText(app.text.S.PanelQuota) || !tt.HasText("打开 Echo") || !tt.HasText("退出") {
		t.Errorf("expected the tab and the bar under it; texts were %v", tt.Texts())
	}
	for _, gone := range []string{app.text.S.Usage, app.text.S.Routing, app.text.S.Agents} {
		for _, text := range tt.Texts() {
			if text == gone {
				t.Errorf("expected the panel to hide the %q tab; texts were %v", gone, tt.Texts())
			}
		}
	}

	// the agents' reading is still there, behind the hidden tab
	app.panelTab = panelTabAgents
	tt.Frame()
	if len(app.data.Agents) == 0 {
		if !tt.HasText("没有找到编码 Agent。") {
			t.Errorf("expected the agents tab's empty state; texts were %v", tt.Texts())
		}
	} else if !tt.HasText(app.data.Agents[0].Name) {
		t.Errorf("expected the first agent's row; texts were %v", tt.Texts())
	}

	// the allowances' tab: the provider's card, its windows, its error
	app.panelTab = panelTabQuota
	tt.Frame()
	for _, want := range []string{"CLAUDE", "Max", "me@example.com", "42%", "91%", "重置于", "读取失败：401", "更新于",
		"Gemini", "5h", "7d", "19%", "30%"} {
		if !tt.HasText(want) {
			t.Errorf("expected the allowances tab to draw %q; texts were %v", want, tt.Texts())
		}
	}
	// a pool's models' own windows are its tooltip's, not rings
	for _, gone := range []string{"Gemini 3.8 Flash (High)", "Gemini 3.8 Flash (Low)"} {
		if tt.HasText(gone) {
			t.Errorf("expected %q folded into the pool row's tooltip, drawn as a row", gone)
		}
	}

	// the empty word, when no subscription is on the machine
	app.panelQuotas = nil
	tt.Frame()
	if !tt.HasText("没有可显示的订阅。") {
		t.Errorf("expected the allowances tab's empty state; texts were %v", tt.Texts())
	}

	// the stamp under the cards is the reading's own age, not the
	// panel's: a kept card is not passed off as fresh
	stale := time.Now().Add(-2 * time.Hour)
	app.panelQuotas = []provider.SubscriptionQuota{{
		Provider: "claude", Name: "Claude", Icon: "claude",
		User: "me@example.com", ReadAt: &stale,
		Windows: []provider.QuotaWindow{{Name: "5 小时", Used: 42, ResetsAt: &resetsAt}},
	}}
	app.panelReadAt = time.Now()
	app.panelTab = panelTabQuota
	tt.Frame()
	if !tt.HasText("更新于 2 小时前") {
		t.Errorf("expected the footer to say the reading's own age; texts were %v", tt.Texts())
	}

	// the two tabs as a reviewer looks at them
	app.panelQuotas = []provider.SubscriptionQuota{
		{
			Provider: "claude", Name: "Claude", Icon: "claude", Plan: "Max",
			User: "me@example.com", ReadAt: &readAt,
			Windows: []provider.QuotaWindow{
				{Name: "5 小时", Used: 42, ResetsAt: &resetsAt},
				{Name: "本周", Used: 91},
			},
		},
		{
			Provider: "codex", Name: "Codex", Icon: "codex",
			User: "me@example.com", Error: "读取失败：401",
		},
		{
			Provider: "antigravity", Name: "Antigravity", Icon: "antigravity",
			User: "me@example.com", ReadAt: &readAt,
			Windows: []provider.QuotaWindow{
				{Name: "Gemini 3.8 Flash (High)", Used: 100, Family: "Gemini", Pool: "Gemini"},
				{Name: "Gemini 3.8 Flash (Low)", Used: 100, Family: "Gemini", Pool: "Gemini"},
				{Name: "5 hours", Used: 19, ResetsAt: &resetsAt, Pool: "Gemini"},
				{Name: "7 days", Used: 30, Pool: "Gemini"},
			},
		},
	}
	app.panelTab = panelTabAgents
	writeShot(t, "echo-tray-panel-agents.png", ui.Render(app.panelView, 400, 540, 2))
	app.panelTab = panelTabQuota
	writeShot(t, "echo-tray-panel-quota.png", ui.Render(app.panelView, 400, 540, 2))
}

// TestTrayPanelHeaderActsOnItsTab checks the panel's first line: the
// refresh button reads what the tab shows, and the open button brings the
// window up on the page that tab stands for — magpie's own panel does the
// same (its refreshKind and its syncTitle).
//
// The tray panel shows the allowances alone, so that is what the buttons
// act on.
func TestTrayPanelHeaderActsOnItsTab(t *testing.T) {
	app := NewForTest()
	app.LoadNow()

	tt := ui.NewTester(app.panelView, 400, 540)
	app.panelTab = panelTabQuota
	tt.Frame()
	if !tt.HasText("重新读取各订阅的额度") {
		t.Errorf("the refresh button should read the allowances; texts were %v", tt.Texts())
	}
	if !tt.HasText("打开主窗口的用量页（额度）") {
		t.Errorf("the open button should open the usage page; texts were %v", tt.Texts())
	}

	// the open button brings the window up on that page
	app.panelOpen()
	if app.page != PageUsage {
		t.Errorf("the open button opened %q, want the Usage page", app.page)
	}
	if app.usTab != UsTabOverview {
		t.Errorf("the open button opened the usage tab %q, want %q", app.usTab, UsTabOverview)
	}
}

// TestTrayPanelUsageAndRouting draws the panel's other two tabs: the usage
// of the period, and the routing groups.
func TestTrayPanelUsageAndRouting(t *testing.T) {
	app := NewForTest()
	app.LoadNow()

	tt := ui.NewTester(app.panelView, 400, 540)

	// the usage: the totals, and the models that spent them
	u := usageapi.Usage{}
	u.Calls = 128
	u.Input = 1_240_000
	u.Output = 380_000
	u.Cost = 3.42
	u.Models = []usageapi.Group{
		{Group: usage.Group{ID: "gpt-5.5", Calls: 54, Cost: 1.8}, Name: "gpt-5.5"},
		{Group: usage.Group{ID: "claude-opus-5-5", Calls: 32, Cost: 1.1}, Name: "claude-opus-5-5"},
	}
	app.data.Usage.Overview = u
	app.panelTab = panelTabUsage
	tt.Frame()
	for _, want := range []string{"128 次调用", "1.24M", "380.0k", "$3.42", "gpt-5.5", "claude-opus-5-5"} {
		if !tt.HasText(want) {
			t.Errorf("expected the usage tab to draw %q; texts were %v", want, tt.Texts())
		}
	}

	// the routing: the groups, what is in them, and the totals
	app.data.Routing.Groups = []RoutingGroupRow{
		{ID: "fast", Name: "fast", Members: []RoutingMemberRow{{ID: "gpt-5.5"}}},
		{ID: "off", Name: "off", Disabled: true},
	}
	app.data.Routing.Totals = RoutingTotals{Requests: 512, Rerouted: 37, Errors: 3}
	app.panelTab = panelTabRouting
	tt.Frame()
	for _, want := range []string{"512 次请求", "改道 37 次", "3 次出错", "fast", "1 个模型", "已停用"} {
		if !tt.HasText(want) {
			t.Errorf("expected the routing tab to draw %q; texts were %v", want, tt.Texts())
		}
	}

	app.panelTab = panelTabUsage
	writeShot(t, "echo-tray-panel-usage.png", ui.Render(app.panelView, 400, 540, 2))
	app.panelTab = panelTabRouting
	writeShot(t, "echo-tray-panel-routing.png", ui.Render(app.panelView, 400, 540, 2))
}
