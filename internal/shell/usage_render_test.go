//go:build nogui && echo

package shell

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/yetone/magpie/internal/gateway"
	"github.com/yetone/magpie/internal/gui"
	"github.com/yetone/magpie/internal/provider"
	"github.com/yetone/magpie/internal/sessions"
	"github.com/yetone/magpie/internal/usage"
	"github.com/yetone/magpie/internal/usageapi"
)

// usageFixture is a page's read as a machine with a history would answer
// it: the totals, the timeline, the groups, a page of the ledger with its
// heatmap and its shares, the sessions with their overview, and the
// prompts the Context segment scores.
//
// A test home has no calls in it, so every segment draws its empty state
// and none of the drawing that follows it. This is what draws the rest:
// the tiles, the charts, the tables and the cards.
func usageFixture(now time.Time) UsageData {
	day := func(i int) time.Time {
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local).AddDate(0, 0, i)
	}
	totals := usage.Totals{
		Calls: 42, Errors: 2, Retried: 3, Input: 120_000, Output: 30_000,
		CacheRead: 60_000, CacheWrite: 8_000, Reasoning: 4_000,
		Cost: 1.25, Unpriced: 1, Timed: 40, TTFT: 40_000, DecodeMs: 60_000, DecodeOut: 26_000,
	}

	var series []usage.Point
	for i := 0; i < 10; i++ {
		t := day(i - 9)
		series = append(series, usage.Point{
			Label:  t.Format("1月2日"),
			Time:   t,
			Totals: usage.Totals{Calls: 4, Input: 9_000, Output: 2_500, CacheRead: 5_000, Cost: 0.12},
		})
	}

	group := func(id, name, icon string, tokens int) usageapi.Group {
		return usageapi.Group{
			Group: usage.Group{ID: id, Totals: usage.Totals{
				Calls: 12, Input: tokens, Output: tokens / 4, CacheRead: tokens / 2, Cost: 0.4,
			}},
			Name: name, Icon: icon,
		}
	}
	overview := usageapi.Usage{
		Summary: usage.Summary{
			Period: usage.Month, Bucket: "day", Totals: totals, Series: series,
		},
		Agents:     []usageapi.Group{group("claude", "Claude Code", "claude", 90_000), group("codex", "Codex", "codex", 40_000)},
		Models:     []usageapi.Group{group("claude-sonnet-4-5", "claude-sonnet-4-5", "anthropic", 70_000)},
		Accounts:   []usageapi.Group{group("me@example.com", "me@example.com", "anthropic", 30_000)},
		CallerKeys: []usageapi.Group{group("k1", "laptop", "generic", 20_000)},
		Path:       "~/.magpie/usage.jsonl",
	}

	share := func(id string, tokens int) usageapi.Share {
		return usageapi.Share{
			Share: usage.Share{ID: id, Totals: usage.Totals{Calls: 6, Input: tokens, Output: tokens / 3}},
			Name:  id,
		}
	}
	by := map[string][]usageapi.Share{
		"model":    {share("claude-sonnet-4-5", 60_000), share("gpt-5", 30_000)},
		"provider": {share("anthropic", 60_000), share("openai", 30_000)},
		"agent":    {share("claude", 60_000), share("codex", 30_000)},
	}
	var points []usage.SeriesPoint
	for i, p := range series {
		pts := map[string]usage.Part{
			"claude-sonnet-4-5": {Calls: 2, Tokens: 9000 + i, Cost: 0.1, Timed: 2, DecodeMs: 1200, DecodeOut: 400},
			"gpt-5":             {Calls: 1, Tokens: 4000 + i, Cost: 0.05, Timed: 1, DecodeMs: 900, DecodeOut: 300},
		}
		byDim := map[string]map[string]usage.Part{
			"model":    pts,
			"provider": {"anthropic": pts["claude-sonnet-4-5"], "openai": pts["gpt-5"]},
			"agent":    {"claude": pts["claude-sonnet-4-5"], "codex": pts["gpt-5"]},
		}
		points = append(points, usage.SeriesPoint{Point: p, By: byDim})
	}

	row := func(i int, failed bool) usageapi.Row {
		r := usageapi.Row{
			AgentName: "Claude Code", Icon: "claude", ProviderName: "Anthropic", Access: "api",
		}
		r.Record = usage.Record{
			Time: day(0).Add(time.Duration(i) * time.Minute), Agent: "claude",
			Provider: "anthropic", Model: "claude-sonnet-4-5", Requested: "sonnet",
			Served: "claude-sonnet-4-5", Input: 4200, Output: 900, CacheRead: 3100,
			CacheWrite: 200, Reasoning: 120, Millis: 2400, TTFT: 600, FirstText: 700,
			Flow: 1400, Status: 200, Session: "sess-1", RequestID: "req-1",
			Endpoint: "/v1/messages", Tries: 2,
		}
		r.Cost, r.Priced = 0.012, true
		if failed {
			r.Status, r.Error, r.ErrType = 529, "overloaded", "overloaded_error"
			r.Swapped = true
		}
		return r
	}
	rows := []usageapi.Row{row(0, false), row(1, true), row(2, false)}

	var hdays []usage.Day
	for i := 0; i < 400; i += 3 {
		d := day(i - 400)
		hdays = append(hdays, usage.Day{
			Date: d.Format(time.DateOnly), Calls: i % 7, Tokens: i * 40, Cost: float64(i) * 0.001,
		})
	}
	ledger := usageapi.Ledger{
		Period: usage.Month, Rows: rows, Total: 120, Offset: 0, Bucket: "day",
		Totals: totals, Through: totals, Direct: usage.Totals{Calls: 3}, Reported: usage.Totals{Calls: 1},
		Series: points, By: by,
		Agents:    []usageapi.Named{{ID: "claude", Name: "Claude Code", Icon: "claude"}},
		Providers: []usageapi.Named{{ID: "anthropic", Name: "Anthropic"}},
		Accounts:  []usageapi.Account{{ID: "me@example.com", Providers: []string{"anthropic"}}},
	}

	// sessions: two of them, one priced, one not
	sess := func(key, agent, title string, tokens int, cost float64, priced bool) sessions.Summary {
		return sessions.Summary{
			Key: key, Agent: agent, ID: key, Title: title, Cwd: "/Users/me/code/echo",
			Last: day(0).Add(-time.Hour), Cost: cost, Priced: priced, Active: 5400,
			Models: []string{"claude-sonnet-4-5"}, Days: []int{0, 1},
			Prompts: 12, Replies: 12, ToolCalls: 40,
			Tokens: sessions.Tokens{Input: tokens, Output: tokens / 4, CacheRead: tokens / 2},
		}
	}
	list := []sessions.Summary{
		sess("claude:a", "claude", "复刻用量页", 90_000, 1.1, true),
		sess("codex:b", "codex", "", 30_000, 0, false),
	}
	stats := sessions.Stats{
		From: day(-9).Format(time.DateOnly), To: day(0).Format(time.DateOnly),
		Days: []sessions.Day{{
			Date: day(0).Format(time.DateOnly),
			Usage: []sessions.Usage{{
				Agent: "claude", Cwd: "/Users/me/code/echo", Model: "claude-sonnet-4-5",
				Tokens: sessions.Tokens{Input: 60_000, Output: 15_000}, Cost: 0.9, Priced: true,
			}},
			Active: []sessions.Active{{
				Agent: "claude", Cwd: "/Users/me/code/echo", Seconds: 3600,
				Hours: []int64{0, 0, 0, 0, 0, 0, 0, 0, 0, 120, 900, 1200, 300},
			}},
		}},
		Sessions: list,
		Keys:     []string{"claude:a", "codex:b"},
	}
	ov := stats.Overview("", "", "", 10)

	prompt := &gateway.Prompt{
		Window: 200_000, Tokens: 48_000, Counted: true, Turns: 6,
		Parts: []gateway.PromptPart{
			{Kind: "system", Tokens: 3_000, Items: []gateway.PromptItem{{Name: "Claude Code", Tokens: 3_000}}},
			{Kind: "tools", Tokens: 12_000, Items: []gateway.PromptItem{
				{Name: "Read", Tokens: 1_200}, {Name: "Bash", Tokens: 1_100},
				{Name: "mcp · github", Tag: "mcp", N: 12, Tokens: 6_000},
				{Name: "Edit", Tokens: 900}, {Name: "Grep", Tokens: 800},
			}},
			{Kind: "memory", Tokens: 2_000},
			{Kind: "files", Tokens: 9_000},
			{Kind: "results", Tokens: 14_000},
			{Kind: "chat", Tokens: 8_000},
		},
	}
	ctx := gui.ContextData{
		Days: 7,
		Agents: []gui.ContextAgent{{
			Agent: "claude", Requests: 40, Sessions: 2, Errors: 1, Calls: 42,
			Median: 44_000, P90: 88_000, Baseline: 17_000, Window: 200_000, Cache: 0.62,
			Growth: 3_200, PeakFill: 0.9, Compacts: 2, MCP: 6_000,
			Shares: map[string]float64{
				"system": 0.06, "tools": 0.25, "memory": 0.04, "files": 0.19, "results": 0.29, "chat": 0.17,
			},
			Score: 78,
			Scores: []gui.ContextScore{
				{Key: "cache", Points: 14, Most: 25},
				{Key: "lean", Points: 18, Most: 25},
				{Key: "pace", Points: 20, Most: 25},
				{Key: "reliable", Points: 24, Most: 25},
			},
			Tags: []gui.ContextTag{
				{Key: "cache-friendly", Tone: "good", Value: 0.62},
				{Key: "tool-heavy", Tone: "warn", Value: 0.29},
				{Key: "compacts", Tone: "info", Value: 2},
			},
			Models:     []string{"live/claude/claude-sonnet-4-5"},
			LatestID:   7,
			LatestTime: day(0),
		}},
		Sessions: []gui.ContextSession{{
			Agent: "claude", Key: "a", Title: "复刻用量页", Model: "claude-sonnet-4-5",
			First: day(-2), Last: day(0), Requests: 40, Peak: 180_000, Window: 200_000,
			Compacts: 2, Latest: prompt, LatestID: 7,
			Points: []gui.ContextPoint{
				{ID: 1, Time: day(-2), Tokens: 30_000, Window: 200_000, Cache: 12_000},
				{ID: 2, Time: day(-1), Tokens: 90_000, Window: 200_000, Cache: 40_000},
				{ID: 3, Time: day(0), Tokens: 180_000, Window: 200_000, Cache: 90_000},
			},
		}},
	}

	readAt := day(0).Add(time.Hour)
	resetsAt := now.Add(3 * time.Hour)
	quotas := []provider.SubscriptionQuota{
		{
			Provider: "claude", Name: "Claude", Icon: "claude", Plan: "Max",
			User: "me@example.com", ReadAt: &readAt,
			Windows: []provider.QuotaWindow{
				{Name: "5 小时", Used: 42, ResetsAt: &resetsAt},
				{Name: "本周", Used: 91, Amount: 91, Limit: 100, Unit: "次"},
			},
		},
		{
			Provider: "codex", Name: "Codex", Icon: "codex", User: "me@example.com",
			Resets: &provider.ResetCredits{Count: 2},
			Error:  "读取失败：401",
			Windows: []provider.QuotaWindow{
				{Name: "每周", Used: 15},
			},
		},
		{
			Provider: "workbuddy", Name: "WorkBuddy", Icon: "workbuddy",
			User: "me@example.com", Checkins: true,
			Checkin: &provider.WorkBuddyCheckin{Day: day(0).Format(time.DateOnly), Msg: "签到成功"},
			Windows: []provider.QuotaWindow{
				{Name: "额度", Unlimited: true},
			},
		},
		{
			// Antigravity's shape, collapsed a row a pool as magpie's
			// page is, its models' own a switch away
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

	return UsageData{
		Overview:         overview,
		Quotas:           quotas,
		Ledger:           ledger,
		Heatmap:          usage.Heatmap{From: hdays[0].Date, To: day(0).Format(time.DateOnly), Days: hdays},
		SessionsStats:    stats,
		SessionsOverview: ov,
		SessionsList:     list,
		SessionsAgents:   []string{"claude", "codex"},
		Context:          ctx,
	}
}

// TestUsageSegmentsRenderWithData draws every segment over a history: the
// tiles, the timeline, the ranking, the heatmap, the calls' table, the
// sessions' cards, and the Context segment's score cards and prompt.
//
// The empty home of a test never reaches any of it, and a panic here is a
// page that would take the window down on a machine with a month of calls.
func TestUsageSegmentsRenderWithData(t *testing.T) {
	app := NewForTest()
	app.ShowForTest(PageUsage)
	app.LoadUsageNow()

	fixture := usageFixture(time.Now())
	tt := ui.NewTester(app.ViewForTest(), 1180, 780)

	for _, seg := range []struct {
		tab    string
		marker string
		draws  []string
	}{
		{UsTabOverview, "", []string{"Claude Code", "订阅额度", "≈$1.25"}},
		{UsTabRequests, "", []string{"用量热力图", "经过网关", "claude-sonnet-4-5", "导出 CSV"}},
		{UsTabSessions, "", []string{"花费最多的会话", "会话的形状", "工具使用", "复刻用量页"}},
		{UsTabContext, "", []string{"提示词的构成", "工具定义", "缓存友好"}},
	} {
		app.usTab = seg.tab
		app.data.Usage = fixture
		app.data.Usage.Query = app.usageQuery()
		app.usFresh = usQueryKey(app.usageQuery()) // draw it, don't read again

		tt.Frame()

		if !tt.HasText(app.ContentMarkerForTest(PageUsage)) {
			t.Errorf("segment %q drew no content over a history; texts were %v", seg.tab, tt.Texts())
		}
		for _, want := range seg.draws {
			if !tt.HasText(want) {
				t.Errorf("segment %q drew none of what the history holds (missing %q); texts were %v",
					seg.tab, want, tt.Texts())
			}
		}
	}

	// the calls' table, a row at a time, and the sheet it opens
	app.usTab = UsTabRequests
	tt.Frame()
	app.usInspectRequest(fixture.Ledger.Rows[0])
	tt.Frame()
	app.usCloseInspect()

	// each of the other tabs draws over the history too
	app.usTab = UsTabSessions
	tt.Frame()
	app.usTab = UsTabContext
	tt.Frame()

	// the sessions module magpie draws on its Context tab stays hidden on
	// Echo's: its prompt card and its empty state are never on the page
	if tt.HasText("最近一次请求的提示词") || tt.HasText("这段时间没有可分析的会话") {
		t.Errorf("the Context segment drew the sessions module it must hide; texts were %v", tt.Texts())
	}

	// and the allowance card's ask before a Codex reset is spent
	app.usResetConfirmUser = "me@example.com"
	tt.Frame()
	app.usResetConfirmUser = ""

	// the allowance cards a row a pool, Antigravity's models' own folded
	// into its tooltip and a switch to every window of its own beside
	app.usTab = UsTabOverview
	app.data.Usage = fixture
	app.data.Usage.Query = app.usageQuery()
	app.usFresh = usQueryKey(app.usageQuery())
	tt.Frame()
	for _, want := range []string{"Gemini · 5 hours", "Gemini · 7 days", "每个模型 (2)"} {
		if !tt.HasText(want) {
			t.Errorf("expected the overview to draw %q; texts were %v", want, tt.Texts())
		}
	}
	if tt.HasText("Gemini 3.8 Flash (High)") {
		t.Errorf("expected Antigravity's per-model windows folded into the pool rows; texts were %v", tt.Texts())
	}
	if a := app.usEveryModel["antigravity|me@example.com"]; a {
		t.Errorf("the per-model switch started on")
	}
	app.usEveryModel = map[string]bool{"antigravity|me@example.com": true}
	tt.Frame()
	if !tt.HasText("按组") || !tt.HasText("Gemini 3.8 Flash (High)") {
		t.Errorf("expected the account switched to every model's own; texts were %v", tt.Texts())
	}
	app.usEveryModel = nil
}

// TestUsageSheetsRender draws the page's three sheets over the history: a
// call's conversation, a session's, and the ask before a Codex reset is
// spent. A sheet is drawn over the page, so a layout that only breaks in
// one of them is seen here rather than in the app.
func TestUsageSheetsRender(t *testing.T) {
	app := NewForTest()
	app.ShowForTest(PageUsage)
	app.LoadUsageNow()

	fixture := usageFixture(time.Now())
	app.usTab = UsTabRequests
	app.data.Usage = fixture
	app.data.Usage.Query = app.usageQuery()
	app.usFresh = usQueryKey(app.usageQuery())

	// a call's conversation
	row := fixture.Ledger.Rows[0]
	app.usInspectRow = &row
	app.usInspectContent = &ContentJSON{
		Found: true, Model: "claude-sonnet-4-5",
		Content: sessions.Content{
			Input: []sessions.Part{
				{Role: "user", Kind: "text", Text: "把用量页的请求分段复刻成原生界面。"},
				{Role: "user", Kind: "context", Text: "AGENTS.md：本仓库的约定……"},
			},
			Output: []sessions.Part{
				{Role: "assistant", Kind: "thinking", Text: "先看 magpie 的 page.js。"},
				{Role: "assistant", Kind: "tool_use", Name: "Read", Text: `{"path":"page.js"}`},
				{Role: "assistant", Kind: "text", Text: "请求分段有筛选、趋势、热力图与明细表。", Model: "claude-sonnet-4-5"},
			},
		},
	}
	writeShot(t, "echo-usage-inspect.png", ui.Render(app.ViewForTest(), 1180, 900, 2))
	app.usCloseInspect()

	// a session's own sheet
	one := sessions.Session{
		Agent: "claude", ID: "a", Title: "复刻用量页", Cwd: "/Users/me/code/echo",
		Last: time.Now().Add(-time.Hour), Cost: 1.1, Resume: "claude --resume a",
		Path: "/Users/me/.claude/projects/echo/a.jsonl",
		Models: []sessions.Model{{
			Model:  "claude-sonnet-4-5",
			Tokens: sessions.Tokens{Input: 90_000, Output: 22_500, CacheRead: 45_000},
			Cost:   1.1, Priced: true,
		}},
	}
	app.usSessOpen = "claude:a"
	app.usSessOne = &one
	app.usSessTrans = &sessions.Transcript{Parts: []sessions.Part{
		{Role: "user", Kind: "text", Text: "复刻 magpie 的用量页面。"},
		{Role: "assistant", Kind: "text", Text: "好，先读 page.js。"},
		{Role: "assistant", Kind: "tool_use", Name: "Read", Text: "page.js"},
		{Role: "tool", Kind: "tool_result", Name: "Read", Text: "3202 行"},
	}}
	app.usTab = UsTabSessions
	writeShot(t, "echo-usage-session.png", ui.Render(app.ViewForTest(), 1180, 900, 2))
	app.usCloseSession()

	// the ask before a Codex reset is spent
	app.usResetConfirmUser = "me@example.com"
	writeShot(t, "echo-usage-reset.png", ui.Render(app.ViewForTest(), 1180, 700, 2))
	app.usResetConfirmUser = ""
}

// TestUsagePageRender draws each segment over the same history and writes
// it where a reviewer can look at it (the temp dir, as the Skills page's
// own shot is written), so a layout that only breaks with data in it is
// seen rather than only surviving a frame.
func TestUsagePageRender(t *testing.T) {
	app := NewForTest()
	app.ShowForTest(PageUsage)
	app.LoadUsageNow()

	fixture := usageFixture(time.Now())
	for _, seg := range []struct{ tab, name string }{
		{UsTabOverview, "echo-usage-overview.png"},
		{UsTabRequests, "echo-usage-requests.png"},
		{UsTabSessions, "echo-usage-sessions.png"},
		{UsTabContext, "echo-usage-context.png"},
	} {
		app.usTab = seg.tab
		app.data.Usage = fixture
		app.data.Usage.Query = app.usageQuery()
		app.usFresh = usQueryKey(app.usageQuery())
		writeShot(t, seg.name, ui.Render(app.ViewForTest(), 1180, 1600, 2))
	}
}
