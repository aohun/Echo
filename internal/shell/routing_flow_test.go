//go:build nogui && echo

package shell

import (
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/yetone/magpie/internal/settings"
)

// rtFlowFixture is a Routing page's read as a gateway that has been routing
// for a while answers it: agents sending, accounts answering, one request
// still on the wire, one that failed and went on to the next account.
//
// A test home has no calls in it, so the stage would draw only its empty
// state. This is what draws the rest: the nodes, the wires, and the packets.
func rtFlowFixture(now time.Time) RoutingData {
	rd := RoutingData{
		AutoGroupsOn: true,
		Totals:       RoutingTotals{Requests: 128, Rerouted: 3, Errors: 1},
		Templates:    []RoutingTemplateRow{{Kind: "smart", Name: "智能分流"}},
		Traces: []RoutingTraceRow{
			{
				ID: 9, Time: now.Add(-300 * time.Millisecond), Agent: "claude",
				RequestedModel: "group/smart", ServedModel: "claude-sonnet-4-5",
				ProviderID: "anthropic", ProviderName: "Anthropic", ProviderIcon: "anthropic",
				Status: 200, Done: false, Millis: 900, TTFT: 260, Tokens: 1800,
				Decision: "智能路由",
			},
			{
				ID: 8, Time: now.Add(-4 * time.Second), Agent: "codex",
				RequestedModel: "group/smart", ServedModel: "gpt-5.1",
				ProviderID: "openai", ProviderName: "OpenAI", ProviderIcon: "openai",
				Status: 200, Done: true, Millis: 2200, TTFT: 320, Tokens: 3400,
				Decision: "会话亲和保持",
			},
			{
				ID: 7, Time: now.Add(-9 * time.Second), Agent: "gemini",
				RequestedModel: "gemini-3-pro", ServedModel: "gemini-3-pro",
				ProviderID: "google", ProviderName: "Google", ProviderIcon: "gemini",
				Status: 429, Done: true, Millis: 1600, Error: "rate limited",
				Decision: "备用容灾路由",
			},
			{
				ID: 6, Time: now.Add(-16 * time.Second), Agent: "claude",
				RequestedModel: "group/smart", ServedModel: "claude-opus-4-6",
				ProviderID: "anthropic", ProviderName: "Anthropic", ProviderIcon: "anthropic",
				Status: 200, Done: true, Millis: 5200, TTFT: 410, Tokens: 9800,
				Decision: "智能路由",
			},
		},
	}
	return rd
}

// TestRoutingStageRenders draws the stage with a trace to fly, and checks the
// words it says come from that trace.
func TestRoutingStageRenders(t *testing.T) {
	app := NewForTest()
	app.LoadNow()
	app.ShowForTest(PageRouting)
	app.data.Routing = rtFlowFixture(time.Now())

	tt := ui.NewTester(app.ViewForTest(), 1180, 780)
	tt.Frame()

	for _, want := range []string{"Echo", "网关", "Claude Code", "Codex", "claude-sonnet-4-5", "智能路由"} {
		if !tt.HasText(want) {
			t.Errorf("expected the stage to say %q, texts: %v", want, tt.Texts())
		}
	}

	// the live dot counts what is on the wire
	if !tt.HasText("在飞 1") {
		t.Errorf("expected '在飞 1', texts: %v", tt.Texts())
	}
}

// TestRoutingStageSaysWhatTheNewestRequestDid checks the caption under the
// stage: the agent, the account, how long it took and how fast it began.
func TestRoutingStageSaysWhatTheNewestRequestDid(t *testing.T) {
	app := NewForTest()
	app.LoadNow()
	app.ShowForTest(PageRouting)
	app.data.Routing = rtFlowFixture(time.Now())

	m := rtFlowOf(app.data.Routing, time.Now())
	cap := m.caption(time.Now())
	for _, want := range []string{"Claude Code", "Anthropic / claude-sonnet-4-5", "900ms", "首字 260ms"} {
		if !strings.Contains(cap, want) {
			t.Errorf("caption %q does not say %q", cap, want)
		}
	}
	if !strings.Contains(m.modeLine(), "智能路由") {
		t.Errorf("mode line %q does not name the strategy", m.modeLine())
	}
}

// TestRoutingStageFliesAReplay draws a stage whose newest request has landed:
// it plays that request again rather than stand still.
func TestRoutingStageFliesAReplay(t *testing.T) {
	app := NewForTest()
	app.LoadNow()
	app.ShowForTest(PageRouting)

	rd := rtFlowFixture(time.Now())
	for i := range rd.Traces {
		rd.Traces[i].Time = rd.Traces[i].Time.Add(-10 * time.Minute)
		rd.Traces[i].Done = true
	}
	app.data.Routing = rd

	m := rtFlowOf(rd, time.Now())
	flights, replay := m.flights(rd.Traces, time.Now())
	if !replay {
		t.Fatalf("expected a replay when nothing is in flight")
	}
	if len(flights) != 1 {
		t.Fatalf("expected one replayed flight, got %d", len(flights))
	}
}

// TestRoutingStageEmptySaysSo draws a gateway that has routed nothing yet.
// TestRoutingStageEmptySaysSo draws a gateway that has routed nothing yet:
// the agent's placeholder, Echo, and a wet place for the accounts.
func TestRoutingStageEmptySaysSo(t *testing.T) {
	app := NewForTest()
	app.LoadNow()
	app.ShowForTest(PageRouting)
	app.data.Routing = RoutingData{}

	tt := ui.NewTester(app.ViewForTest(), 1180, 780)
	tt.Frame()

	for _, want := range []string{
		"等待请求",
		"你的 Agent",
		"还没有请求",
		"经 Echo 的 Agent 发出请求后，这里实时播放",
		"Agent 发给 Echo 的每个请求都会在此显示，均为真实路由。",
	} {
		if !tt.HasText(want) {
			t.Errorf("expected the empty stage to say %q, texts: %v", want, tt.Texts())
		}
	}
	// the hub carries the gateway's own address, and the foot counts from zero
	if !tt.HasText(settings.GatewayAddr()) {
		t.Errorf("expected the hub to say the gateway address %q, texts: %v",
			settings.GatewayAddr(), tt.Texts())
	}
	for _, want := range []string{"请求", "改道", "Agent 收到的错误"} {
		if !tt.HasText(want) {
			t.Errorf("expected the foot to count %q, texts: %v", want, tt.Texts())
		}
	}
}

// TestRoutingStagePacketMoves walks a flight's packet along its legs: out
// from the agent through magpie to the account, then back the way it came.
func TestRoutingStagePacketMoves(t *testing.T) {
	a0 := rtPt{100, 200}
	hL := rtPt{300, 200}
	hR := rtPt{420, 200}
	c0 := rtPt{600, 120}
	stops := []rtPt{c0}
	f := rtFlowFlight{dur: 2 * time.Second, start: time.Unix(0, 0)}

	at := func(ms int) rtPt {
		mark := rtFlowAt(f, time.Unix(0, int64(ms)*int64(time.Millisecond)), a0, hL, hR, stops)
		if !mark.ok {
			t.Fatalf("no packet at %d ms", ms)
		}
		return mark.at
	}

	early, mid, held := at(200), at(500), at(900)
	if !(early.X < mid.X && mid.X < held.X) {
		t.Errorf("the request does not travel rightwards: %v %v %v", early, mid, held)
	}
	if !(early.Y > mid.Y && mid.Y > held.Y) {
		t.Errorf("the request does not climb to the account: %v %v %v", early, mid, held)
	}
	if held != c0 {
		t.Errorf("the request does not reach the account: %v, want %v", held, c0)
	}
	// on the way back it comes left again, to the agent it came from
	back := at(1400)
	if !(back.X < held.X) {
		t.Errorf("the answer does not come back: %v %v", back, held)
	}
	// and once it has landed there is nothing on the wire
	if mark := rtFlowAt(f, time.Unix(0, int64(2100)*int64(time.Millisecond)), a0, hL, hR, stops); mark.ok {
		t.Errorf("a landed flight still draws a packet")
	}
}

// TestRoutingStagePacketWaitsWhileAnswered keeps the packet at the account
// while the request is still being answered.
func TestRoutingStagePacketWaitsWhileAnswered(t *testing.T) {
	c0 := rtPt{600, 120}
	f := rtFlowFlight{dur: 2 * time.Second, start: time.Unix(0, 0), live: true}
	mark := rtFlowAt(f, time.Unix(0, int64(5*time.Second)), rtPt{100, 200}, rtPt{300, 200}, rtPt{420, 200}, []rtPt{c0})
	if !mark.ok || mark.back {
		t.Fatalf("expected a packet at the account, got ok=%v back=%v", mark.ok, mark.back)
	}
	if !mark.hold {
		t.Errorf("expected the packet to be held at the account")
	}
	if mark.at != c0 {
		t.Errorf("packet at %v, want the account at %v", mark.at, c0)
	}
}

// TestRoutingStageManyShot draws three requests on the wire at once, each a
// different way along, to see them share the stage.
func TestRoutingStageManyShot(t *testing.T) {
	app := NewForTest()
	app.LoadNow()
	app.ShowForTest(PageRouting)

	now := time.Now()
	rd := rtFlowFixture(now)
	rd.Traces = []RoutingTraceRow{
		{
			ID: 3, Time: now.Add(-150 * time.Millisecond), Agent: "claude",
			RequestedModel: "group/smart", ServedModel: "claude-sonnet-4-5",
			ProviderID: "anthropic", ProviderName: "Anthropic", ProviderIcon: "anthropic",
			Done: false, Millis: 2600, TTFT: 240, Decision: "智能路由",
		},
		{
			ID: 2, Time: now.Add(-900 * time.Millisecond), Agent: "codex",
			RequestedModel: "group/smart", ServedModel: "gpt-5.1",
			ProviderID: "openai", ProviderName: "OpenAI", ProviderIcon: "openai",
			Done: false, Millis: 2600, TTFT: 300, Decision: "智能路由",
		},
		{
			ID: 1, Time: now.Add(-1900 * time.Millisecond), Agent: "gemini",
			RequestedModel: "gemini-3-pro", ServedModel: "gemini-3-pro",
			ProviderID: "google", ProviderName: "Google", ProviderIcon: "gemini",
			Done: false, Millis: 2600, TTFT: 280, Decision: "智能路由",
		},
	}
	app.data.Routing = rd

	m := rtFlowOf(rd, time.Now())
	flights, replay := m.flights(rd.Traces, time.Now())
	if len(flights) != 3 || replay {
		t.Errorf("expected three requests on the wire, got %d (replay %v)", len(flights), replay)
	}
	writeShot(t, "echo-routing-stage-many.png", ui.Render(app.ViewForTest(), 1180, 640, 2))
}

// TestRoutingStageRerouted draws a request that failed over: it goes out to
// the account that could not answer, comes back to magpie, and goes on to
// the one that did.
func TestRoutingStageRerouted(t *testing.T) {
	app := NewForTest()
	app.LoadNow()
	app.ShowForTest(PageRouting)

	now := time.Now()
	rd := rtFlowFixture(now)
	// the fixture's own requests are long done: only this one is on the wire,
	// but their accounts are the ones on the stage
	for i := range rd.Traces {
		rd.Traces[i].Time = now.Add(-10 * time.Minute)
		rd.Traces[i].Done = true
	}
	rd.Traces = append([]RoutingTraceRow{{
		ID: 5, Time: now.Add(-700 * time.Millisecond), Agent: "codex",
		RequestedModel: "group/smart", ServedModel: "claude-sonnet-4-5",
		ProviderID: "anthropic", ProviderName: "Anthropic", ProviderIcon: "anthropic",
		Done: false, Millis: 2600, TTFT: 300, Decision: "备用容灾路由",
		Tries: []RoutingTryRow{
			{Provider: "google", Model: "gemini-3-pro", Status: 429},
			{Provider: "anthropic", Model: "claude-sonnet-4-5", Status: 200},
		},
	}}, rd.Traces...)
	app.data.Routing = rd

	m := rtFlowOf(rd, time.Now())
	flights, replay := m.flights(rd.Traces, time.Now())
	if replay || len(flights) != 1 {
		t.Fatalf("expected one request on the wire, got %d (replay %v)", len(flights), replay)
	}
	if got := len(flights[0].path); got != 2 {
		t.Fatalf("expected the request to be drawn as two hops, got %d", got)
	}

	// at this age it is coming back from the account that could not answer
	stops := []rtPt{{600, 240}, {600, 120}}
	mark := rtFlowAt(flights[0], time.Now(), rtPt{100, 200}, rtPt{300, 200}, rtPt{420, 200}, stops)
	if !mark.ok || !mark.back || !mark.fail {
		t.Errorf("expected the failed try to be on its way back, got %+v", mark)
	}

	writeShot(t, "echo-routing-stage-rerouted.png", ui.Render(app.ViewForTest(), 1180, 640, 2))
}

// TestRoutingStageShot writes the page with a stage flying, to look at.
func TestRoutingStageShot(t *testing.T) {
	app := NewForTest()
	app.LoadNow()
	app.ShowForTest(PageRouting)
	app.data.Routing = rtFlowFixture(time.Now())

	writeShot(t, "echo-routing-stage.png", ui.Render(app.ViewForTest(), 1180, 900, 2))
}

// TestRoutingStageShots writes the stage with nothing to fly, in a window
// narrow enough to squeeze its columns, and replaying its newest request, to
// look at.
func TestRoutingStageShots(t *testing.T) {
	app := NewForTest()
	app.LoadNow()
	app.ShowForTest(PageRouting)
	app.data.Routing = RoutingData{}
	writeShot(t, "echo-routing-stage-empty.png", ui.Render(app.ViewForTest(), 1180, 640, 2))

	app.data.Routing = rtFlowFixture(time.Now())
	writeShot(t, "echo-routing-stage-narrow.png", ui.Render(app.ViewForTest(), 900, 700, 2))

	rd := rtFlowFixture(time.Now())
	for i := range rd.Traces {
		rd.Traces[i].Time = rd.Traces[i].Time.Add(-20 * time.Minute)
		rd.Traces[i].Done = true
	}
	app.data.Routing = rd
	writeShot(t, "echo-routing-stage-replay.png", ui.Render(app.ViewForTest(), 1180, 640, 2))
}
