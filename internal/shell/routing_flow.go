//go:build nogui && echo

package shell

import (
	"cmp"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/yetone/magpie/internal/settings"
)

// The Routing page's stage: the live picture of what routing does with a
// request — an agent's request carried through magpie to the account routing
// put first, and the answer carried back, or the failure, so the next one
// can be tried.
//
// It is magpie's own stage (internal/gui/assets/plugins/routing/client.js)
// drawn in native UI: the agents in a column at the left, the gateway in
// the middle, the accounts routing weighed in a column at the right, wires
// between them, and a packet flying each leg of a request the gateway
// recorded. Everything drawn comes from that trace — the agent, the model it
// asked for, the one that answered, how long it took, whether it failed.
// Nothing is worked out again or made up: a stage with nothing to fly says
// so, and a stage with nothing in flight replays the newest request rather
// than stand still, saying that it is a replay.

// The stage's measurements, in DIPs: two columns of nodes, the hub between
// them, and the run of wire on either side.
const (
	rtFlowH        = 200 // the stage's height
	rtFlowPadX     = 6   // room at the stage's sides
	rtFlowAgentW   = 156 // an agent's node
	rtFlowColGap   = 62  // the wires' run between the columns
	rtFlowHubW     = 132
	rtFlowHubH     = 108
	rtFlowAgentH   = 60
	rtFlowAgentGap = 6
	rtFlowAcctH    = 38
	rtFlowAcctGap  = 6
	rtFlowMaxAgent = 3 // the agents drawn at once, as many as the column holds
	rtFlowMaxAcct  = 4
	rtFlowScan     = 40              // the traces the stage reads back over
	rtFlowPlay     = 4 * time.Second // the longest a flight is played
	rtFlowLive     = 15 * time.Second
	rtFlowFrame    = 33 * time.Millisecond // a flight is redrawn at 30 a second
)

// rtFlowAgent is one agent's node on the stage: who is sending.
type rtFlowAgent struct {
	ID    string
	Name  string
	Icon  string
	Hue   ui.Color
	Count int
}

// rtFlowAcct is one account's row: the provider and model a request went to,
// as the newest request to it found it.
type rtFlowAcct struct {
	Provider string
	Model    string
	Icon     string
	Millis   int64
	Failed   bool
	Live     bool
	Count    int
}

// rtFlowFlight is one request on its way: from an agent, through magpie, to
// an account and back — or, from one that could not answer, back to magpie
// and on to the next.
type rtFlowFlight struct {
	agent  int
	path   []int // the accounts it was weighed through, in the order it tried them
	start  time.Time
	dur    time.Duration
	hue    ui.Color
	failed bool
	live   bool // still being answered: the packet waits at the account
}

// rtFlowModel is what the stage draws, read from the gateway's trace.
type rtFlowModel struct {
	Agents  []rtFlowAgent
	Accts   []rtFlowAcct
	agentAt map[string]int
	acctAt  map[string]int
	Newest  RoutingTraceRow
	Has     bool
	Live    int
	// where the gateway is listening, under Echo's own name on the hub
	Addr string
	// the running totals the foot counts: what routing has served, how many
	// it had to go on with another account for, and how many of them the
	// agent was told had failed
	Requests int
	Rerouted int
	Errors   int
	// how many agents and accounts the trace holds, drawn or not: the
	// columns hold three and four, and the page says so rather than
	// pretending the rest are not there
	AgentTotal int
	AcctTotal  int
}

// rtPt is a point on the stage, in window DIPs.
type rtPt struct{ X, Y float32 }

// rtAgentHues are the colours an agent's requests fly in: its maker's own
// where it has one, else one of the rest picked by its id, so it stays the
// same from one request to the next. None is red: red on this page is a
// failure.
var rtAgentHues = map[string]ui.Color{
	"claude":      ui.Hex("#d97757"),
	"codex":       ui.Hex("#6366f1"),
	"gemini":      ui.Hex("#0ea5e9"),
	"copilot":     ui.Hex("#a855f7"),
	"cursor":      ui.Hex("#14b8a6"),
	"opencode":    ui.Hex("#eab308"),
	"crush":       ui.Hex("#ec4899"),
	"goose":       ui.Hex("#84cc16"),
	"pi":          ui.Hex("#10b981"),
	"omp":         ui.Hex("#d946ef"),
	"omo":         ui.Hex("#7c3aed"),
	"dsh":         ui.Hex("#06b6d4"),
	"commandcode": ui.Hex("#f97316"),
	"mimocode":    ui.Hex("#3b82f6"),
	"deveco":      ui.Hex("#3c99cc"),
}

// rtSpareHues are the rest, for an agent whose maker's colour is not known.
var rtSpareHues = []ui.Color{
	ui.Hex("#8b5cf6"), ui.Hex("#22c55e"), ui.Hex("#0d9488"), ui.Hex("#0891b2"),
	ui.Hex("#ca8a04"), ui.Hex("#db2777"), ui.Hex("#2563eb"), ui.Hex("#65a30d"),
}

// rtAgentHue is the colour an agent's requests and answers fly in.
func rtAgentHue(id string) ui.Color {
	if c, ok := rtAgentHues[id]; ok {
		return c
	}
	var h uint32
	for _, r := range id {
		h = h*31 + uint32(r)
	}
	return rtSpareHues[h%uint32(len(rtSpareHues))]
}

// rtFlowOf reads the stage's agents and accounts off the gateway's trace:
// the accounts the newest requests went to, then the agents that sent them,
// each with the requests it has here.
func rtFlowOf(rd RoutingData, now time.Time) rtFlowModel {
	m := rtFlowModel{agentAt: map[string]int{}, acctAt: map[string]int{}}
	if len(rd.Traces) > 0 {
		m.Newest, m.Has = rd.Traces[0], true
	}
	seenAgent := map[string]bool{}
	seenAcct := map[string]bool{}
	for i, tr := range rd.Traces {
		if i >= rtFlowScan {
			break
		}
		// the accounts, each as the newest request to it found it
		key := rtFlowAcctKey(tr)
		if !seenAcct[key] {
			seenAcct[key] = true
			m.AcctTotal++
			if len(m.Accts) < rtFlowMaxAcct {
				m.acctAt[key] = len(m.Accts)
				m.Accts = append(m.Accts, rtFlowAcct{
					Provider: tr.ProviderName,
					Model:    tr.ServedModel,
					Icon:     tr.ProviderIcon,
					Millis:   tr.Millis,
					Failed:   rtFlowFailed(tr),
					Live:     !tr.Done,
					Count:    1,
				})
			}
		} else if ci, ok := m.acctAt[key]; ok {
			m.Accts[ci].Count++
		}

		// then the agents sending, and how much each has sent here
		id := tr.Agent
		if id == "" {
			id = "agent"
		}
		if !seenAgent[id] {
			seenAgent[id] = true
			m.AgentTotal++
			if len(m.Agents) < rtFlowMaxAgent {
				m.agentAt[id] = len(m.Agents)
				m.Agents = append(m.Agents, rtFlowAgent{
					ID:   id,
					Name: usAgentName(id),
					Icon: usAgentIcon(id),
					Hue:  rtAgentHue(id),
				})
			}
		}
		if ai, ok := m.agentAt[id]; ok {
			m.Agents[ai].Count++
		}
		if !tr.Done && now.Sub(tr.Time) < rtFlowLive {
			m.Live++
		}
	}
	return m
}

// rtFlowAcctKey is a request's account: the provider it went to and the
// model that answered there.
func rtFlowAcctKey(tr RoutingTraceRow) string {
	model := tr.ServedModel
	if model == "" {
		model = tr.RequestedModel
	}
	return tr.ProviderID + "\x00" + model
}

// rtFlowFailed says whether a request's agent got an error.
func rtFlowFailed(tr RoutingTraceRow) bool {
	return tr.Error != "" || tr.Status >= 400
}

// flightOf maps a trace onto the stage's agents and accounts: the accounts
// its tries went to, in the order it went to them, so a request that failed
// over and went on is drawn as the hops it made.
func (m rtFlowModel) flightOf(tr RoutingTraceRow) (rtFlowFlight, bool) {
	id := tr.Agent
	if id == "" {
		id = "agent"
	}
	ai, ok := m.agentAt[id]
	if !ok {
		return rtFlowFlight{}, false
	}
	var path []int
	for _, try := range tr.Tries {
		for i := range m.Accts {
			if m.Accts[i].Model == try.Model && !containsInt(path, i) {
				path = append(path, i)
				break
			}
		}
	}
	if len(path) == 0 {
		ci, ok := m.acctAt[rtFlowAcctKey(tr)]
		if !ok {
			return rtFlowFlight{}, false
		}
		path = []int{ci}
	}
	return rtFlowFlight{agent: ai, path: path, hue: rtAgentHue(id)}, true
}

// containsInt says whether n is in xs.
func containsInt(xs []int, n int) bool {
	for _, x := range xs {
		if x == n {
			return true
		}
	}
	return false
}

// flights are the requests playing now, or — when none is — the newest one,
// played again in a loop, so the stage shows the way routing goes.
func (m rtFlowModel) flights(traces []RoutingTraceRow, now time.Time) (out []rtFlowFlight, replay bool) {
	for i, tr := range traces {
		if i >= rtFlowScan {
			break
		}
		f, ok := m.flightOf(tr)
		if !ok || tr.Time.IsZero() || now.Before(tr.Time) {
			continue
		}
		f.dur, f.failed, f.live = rtFlightDur(tr), rtFlowFailed(tr), !tr.Done
		el := now.Sub(tr.Time)
		if tr.Done && el > f.dur {
			continue // it landed
		}
		if !tr.Done && el > rtFlowLive {
			continue // it is not coming back
		}
		f.start = tr.Time
		out = append(out, f)
	}
	if len(out) > 0 {
		return out, false
	}
	// nothing in flight: the newest request, replayed, so the stage is not
	// a still picture. The foot says it is a replay.
	for i, tr := range traces {
		if i >= rtFlowScan {
			break
		}
		f, ok := m.flightOf(tr)
		if !ok {
			continue
		}
		f.dur, f.failed = rtFlightDur(tr), rtFlowFailed(tr)
		period := f.dur + 1100*time.Millisecond
		el := now.Sub(tr.Time)
		if el < 0 {
			el = 0
		}
		f.start = now.Add(-(el % period))
		return []rtFlowFlight{f}, true
	}
	return nil, false
}

// rtFlightDur is how long a request takes to fly out and back: as long as it
// took the gateway, kept within what a reader can follow.
func rtFlightDur(tr RoutingTraceRow) time.Duration {
	ms := tr.Millis
	if ms <= 0 {
		ms = 1400
	}
	d := time.Duration(ms) * time.Millisecond
	if d < 1200*time.Millisecond {
		d = 1200 * time.Millisecond
	}
	if d > rtFlowPlay {
		d = rtFlowPlay
	}
	return d
}

// routingFlow is the stage at the top of the Routing page: what routing did
// with the requests the gateway recorded, played as it happens.
func (a *App) routingFlow(c *ui.Context, t *ui.Theme, rd RoutingData) {
	now := time.Now()
	m := rtFlowOf(rd, now)
	// the hub names the gateway that serves: the shell's own when it is
	// the one serving, the instance's when the seam reads one
	m.Addr = cmp.Or(remoteGatewayServed(), settings.GatewayAddr())
	m.Requests, m.Rerouted, m.Errors = rd.Totals.Requests, rd.Totals.Rerouted, rd.Totals.Errors
	a.rtFlowKick()
	_, replay := m.flights(rd.Traces, now)

	card := ui.Column(c).FillWidth().Radius(12).Background(a.fieldBg(t)).
		Border(1, t.Border).Padding(10, 12, 9, 12).Gap(7)
	card.Children(func() {
		a.rtFlowHead(c, t, m, now)
		a.rtFlowStage(c, t, m, rd.Traces)
		a.rtFlowFoot(c, t, m, now, replay)
	})
}

// rtFlowKick reads the gateway's trace again while the stage is on screen:
// the page is live, and a stage that read once would fly requests that ended
// minutes ago. It reads off the main thread, at most every two seconds, and
// never twice at once.
func (a *App) rtFlowKick() {
	if a.win == nil || a.rtBusy || time.Since(a.rtFlowAt) < 2*time.Second {
		return
	}
	a.rtFlowAt, a.rtBusy = time.Now(), true
	go func() {
		rd := readRoutingData()
		if a.win == nil {
			return
		}
		a.win.Update(func() {
			a.rtBusy = false
			a.data.Routing = rd
		})
	}()
}

// rtFlowHead names what the stage shows and how routing routes it.
func (a *App) rtFlowHead(c *ui.Context, t *ui.Theme, m rtFlowModel, now time.Time) {
	ui.Row(c).AlignItems(ui.Center).Gap(6).FillWidth().Children(func() {
		ui.Text(c, m.what()).FontSize(12.5).FontWeight(600).TextColor(t.Text).Shrink(0).SingleLine()
		ui.Text(c, m.whatSub(now)).FontSize(11.5).TextColor(t.TextMuted).Grow(1).Shrink(1).SingleLine()

		if !m.Has {
			return // nothing is in flight, and nothing to wait for either
		}

		// the live dot: it breathes while something is on the wire
		pill := ui.Row(c).AlignItems(ui.Center).Gap(5).Padding(2, 8).Radius(9).Shrink(0)
		bg, fg := ui.RGBA(52, 199, 89, 0.12), ui.Hex("#34c759")
		if m.Live == 0 {
			bg = ui.RGBA(0, 0, 0, 0.05)
			fg = t.TextMuted
			if t.Dark {
				bg = ui.RGBA(255, 255, 255, 0.07)
			}
		}
		pill.Background(bg)
		pill.Children(func() {
			dot := ui.Row(c).Width(6).Height(6).Radius(3).Background(fg)
			if m.Live > 0 {
				dot.Opacity(0.45 + 0.55*dot.Loop("rt-flow-live", 1200*time.Millisecond, ui.EaseInOut))
			}
			ui.Text(c, m.liveLabel()).FontSize(10.5).FontWeight(500).TextColor(fg).SingleLine()
		})
	})
	ui.Text(c, m.modeLine()).FontSize(11).TextColor(t.TextMuted).SingleLine()
}

// rtFlowStage draws the wires and the packets, and lays the nodes over them.
// With no request to show it lays the same three columns wet: the agent every
// request comes from, Echo, and a place for the accounts to arrive at.
func (a *App) rtFlowStage(c *ui.Context, t *ui.Theme, m rtFlowModel, traces []RoutingTraceRow) {
	if !m.Has {
		a.rtFlowStageWet(c, t, m)
		return
	}
	stage := ui.Row(c).FillWidth().Height(rtFlowH).Padding(0, rtFlowPadX).
		AlignItems(ui.Center).Gap(rtFlowColGap)
	stage.Draw(func(p *ui.Painter, r ui.Rect) { a.rtFlowPaint(p, r, t, m, traces) })
	stage.Children(func() {
		ui.Column(c).Width(rtFlowAgentW).Shrink(0).Gap(rtFlowAgentGap).Children(func() {
			for _, ag := range m.Agents {
				a.rtFlowAgentNode(c, t, ag, false)
			}
		})
		a.rtFlowHub(c, t, m)
		ui.Column(c).Grow(1).Shrink(1).Gap(rtFlowAcctGap).Children(func() {
			if len(m.Accts) == 0 {
				ui.Text(c, "等待第一个请求…").FontSize(11.5).TextColor(t.TextMuted).
					Padding(0, 10).SingleLine()
				return
			}
			for _, ac := range m.Accts {
				a.rtFlowAcctRow(c, t, ac)
			}
		})
	})
}

// rtFlowStageWet is the stage with nothing to fly: no wires, no packets, the
// agent every request comes from at the left, Echo in the middle, and where
// the accounts will be at the right.
func (a *App) rtFlowStageWet(c *ui.Context, t *ui.Theme, m rtFlowModel) {
	ui.Row(c).FillWidth().Height(rtFlowH).Padding(0, rtFlowPadX).
		AlignItems(ui.Center).Gap(rtFlowColGap).Children(func() {
		ui.Column(c).Width(rtFlowAgentW).Shrink(0).Children(func() {
			a.rtFlowAgentNode(c, t, rtFlowAgent{Name: "你的 Agent", Icon: "generic"}, true)
		})
		a.rtFlowHub(c, t, m)
		ui.Row(c).Grow(1).Shrink(1).Height(rtFlowH-18).AlignItems(ui.Center).
			Justify(ui.Center).Radius(12).BorderStyle(ui.BorderDashed).
			Border(1, t.Border).Children(func() {
			ui.Text(c, "还没有请求").FontSize(11.5).TextColor(t.TextMuted).SingleLine()
		})
	})
}

// rtFlowAgentNode is one agent at the left of the stage: its icon, its name
// under it with the dot its requests fly in, and how much it has sent. With
// nothing sent yet it is the placeholder every request will come from — the
// same card, no count, nothing in colour.
func (a *App) rtFlowAgentNode(c *ui.Context, t *ui.Theme, ag rtFlowAgent, wet bool) {
	node := ui.Column(c).Height(rtFlowAgentH).Padding(6, 10).Radius(10).
		Background(a.rtFlowNodeBg(t)).Border(1, t.Border).Center().Gap(1)
	node.Children(func() {
		if pic := iconPictureFor(ag.Icon); pic != nil {
			ui.Image(c, pic).Size(20, 20).Fit(ui.Contain)
		} else {
			ui.Icon(c, genericIcon()).Size(19, 19).TextColor(t.TextMuted)
		}
		ui.Row(c).AlignItems(ui.Center).Gap(6).Shrink(0).Children(func() {
			dot := ag.Hue
			if wet {
				dot = t.TextMuted
			}
			ui.Row(c).Width(7).Height(7).Radius(4).Background(dot).Shrink(0)
			ui.Text(c, ag.Name).FontSize(12.5).FontWeight(600).TextColor(t.Text).
				Shrink(1).SingleLine()
		})
		if wet {
			return
		}
		ui.Text(c, fmt.Sprintf("%d 次请求", ag.Count)).FontSize(10).TextColor(t.TextMuted).SingleLine()
	})
}

// rtFlowHub is Echo in the middle: the bird, its name, where its gateway
// listens, and how the request on the wire was routed.
func (a *App) rtFlowHub(c *ui.Context, t *ui.Theme, m rtFlowModel) {
	hub := ui.Column(c).Width(rtFlowHubW).Height(rtFlowHubH).Shrink(0).
		Center().Radius(12).Background(a.rtFlowNodeBg(t)).
		Border(1.5, t.Text).Gap(1)
	hub.Children(func() {
		ui.Icon(c, birdIcon()).Size(24, 24).TextColor(t.Text)
		ui.Text(c, "Echo").FontSize(12.5).FontWeight(700).TextColor(t.Text).SingleLine()
		ui.Text(c, m.Addr).FontSize(10).TextColor(t.TextMuted).SingleLine()

		if label := m.pickLabel(); label != "" {
			ui.Box(c).Padding(1, 8).Radius(9).Background(t.Text).Children(func() {
				ui.Text(c, label).FontSize(10).FontWeight(500).TextColor(a.rtFlowNodeBg(t)).SingleLine()
			})
		}
	})
}

// rtFlowAcctRow is one account at the right: the model that answered, where
// it was answered, and how long it took.
func (a *App) rtFlowAcctRow(c *ui.Context, t *ui.Theme, ac rtFlowAcct) {
	row := ui.Row(c).Height(rtFlowAcctH).AlignItems(ui.Center).Gap(8).
		Padding(5, 9).Radius(9).Background(a.rtFlowNodeBg(t)).Border(1, t.Border)
	row.Children(func() {
		if pic := iconPictureFor(ac.Icon); pic != nil {
			ui.Image(c, pic).Size(15, 15).Fit(ui.Contain)
		} else {
			ui.Icon(c, genericIcon()).Size(14, 14).TextColor(t.TextMuted)
		}
		ui.Column(c).Grow(1).Shrink(1).Gap(0).Justify(ui.Center).Children(func() {
			model := ac.Model
			if model == "" {
				model = "—"
			}
			ui.Text(c, model).FontSize(11.5).FontWeight(600).TextColor(t.Text).SingleLine()
			sub := ac.Provider
			if ac.Count > 1 {
				sub += fmt.Sprintf(" · %d 次", ac.Count)
			}
			ui.Text(c, sub).FontSize(10).TextColor(t.TextMuted).SingleLine()
		})

		// what it answered, as the newest request found it
		tone, note := ui.Hex("#34c759"), FormatDuration(ac.Millis)
		switch {
		case ac.Live:
			tone, note = ui.Hex("#0a84ff"), "应答中"
		case ac.Failed:
			tone, note = ui.Hex("#ff3b30"), "失败"
		}
		ui.Row(c).AlignItems(ui.Center).Gap(4).Shrink(0).Children(func() {
			ui.Row(c).Width(6).Height(6).Radius(3).Background(tone)
			ui.Text(c, note).FontSize(10).TextColor(tone).SingleLine()
		})
	})
}

// rtFlowFoot is the line under the stage: what the newest request did (or,
// before there is one, that what will be played here is the gateway's own
// trace), and the three numbers the stage keeps — what it has served, how
// many it had to send on to another account, and how many of those the
// agent was told had failed. All three are the gateway's own count, and a
// fresh gateway reads 0 0 0.
func (a *App) rtFlowFoot(c *ui.Context, t *ui.Theme, m rtFlowModel, now time.Time, replay bool) {
	caption := m.caption(now)
	if replay && caption != "" {
		caption = "重放最近一次 · " + caption
	}
	ui.Row(c).AlignItems(ui.Center).Gap(12).FillWidth().Children(func() {
		if caption != "" {
			ui.Text(c, caption).FontSize(11).TextColor(t.TextMuted).Grow(1).Shrink(1).SingleLine()
		}
		ui.Row(c).AlignItems(ui.Center).Gap(16).Shrink(0).Children(func() {
			a.rtFlowStat(c, t, m.Requests, "请求")
			a.rtFlowStat(c, t, m.Rerouted, "改道")
			a.rtFlowStat(c, t, m.Errors, "Agent 收到的错误")
		})
	})
}

// rtFlowStat is one of the foot's three numbers, and what it counts.
func (a *App) rtFlowStat(c *ui.Context, t *ui.Theme, n int, label string) {
	ui.Column(c).AlignItems(ui.Center).Gap(1).Shrink(0).Children(func() {
		ui.Text(c, strconv.Itoa(n)).FontSize(15).FontWeight(650).TextColor(t.Text).
			SingleLine()
		ui.Text(c, label).FontSize(10).TextColor(t.TextMuted).SingleLine()
	})
}

// rtFlowNodeBg is the colour of a node on the stage: a step above the card.
func (a *App) rtFlowNodeBg(t *ui.Theme) ui.Color {
	if t.Dark {
		return ui.Hex("#242429")
	}
	return ui.Hex("#ffffff")
}

// ---- the words ------------------------------------------------------------

// what names what the stage shows.
func (m rtFlowModel) what() string {
	if !m.Has {
		return "等待请求"
	}
	if m.AgentTotal == 1 && len(m.Agents) > 0 {
		return m.Agents[0].Name + " 的请求"
	}
	return fmt.Sprintf("%d 个 Agent 的请求", m.AgentTotal)
}

// whatSub says what the newest request asked for and where it went.
func (m rtFlowModel) whatSub(now time.Time) string {
	if !m.Has {
		return ""
	}
	tr := m.Newest
	var parts []string
	if tr.RequestedModel != "" {
		parts = append(parts, "请求 "+tr.RequestedModel)
	}
	if tr.ServedModel != "" && tr.ServedModel != tr.RequestedModel {
		parts = append(parts, "应答 "+tr.ServedModel)
	}
	if m.AcctTotal > 0 {
		parts = append(parts, fmt.Sprintf("%d 个候选", m.AcctTotal))
	}
	if !tr.Time.IsZero() {
		parts = append(parts, rtFlowAgo(now, tr.Time))
	}
	if len(parts) == 0 {
		return ""
	}
	return "· " + strings.Join(parts, " · ")
}

// liveLabel says whether anything is on the wire.
func (m rtFlowModel) liveLabel() string {
	if m.Live > 0 {
		return fmt.Sprintf("在飞 %d", m.Live)
	}
	return "待命"
}

// modeLine explains how routing routes, in the words of the strategy the
// newest request was routed by. With nothing routed yet it says what will
// be played here, and what it takes to make it happen.
func (m rtFlowModel) modeLine() string {
	if !m.Has {
		return "经 Echo 的 Agent 发出请求后，这里实时播放：路由把谁排第一及原因、每次尝试和各自的回应。"
	}
	d := m.Newest.Decision
	note := ""
	switch {
	case d == "" || d == "智能路由":
		note = "智能路由：按各账号的用量与配额节奏挑一个就绪的，失败就换下一个。"
	case len(d) > len("命中规则: ") && d[:len("命中规则: ")] == "命中规则: ":
		note = "命中条件规则：" + d[len("命中规则: "):] + " 决定了这次的走向。"
	case d == "会话亲和保持":
		note = "会话亲和：同一个会话留在同一个账号上，上下文不重建。"
	case d == "账号亲和保持":
		note = "账号亲和：同一个账号继续服务，不换凭据。"
	case d == "备用容灾路由":
		note = "容灾：首选失败后按顺序换下一个账号，直到有人答上来。"
	case d == "就绪直连":
		note = "就绪直连：这个模型只有一个可用账号，请求直接交给它。"
	default:
		note = d + "：按这个策略在就绪的账号之间分配请求。"
	}
	return note
}

// pickLabel is how the request on the wire was routed, told on the hub: the
// decision the newest request came back with, or empty when nothing has been
// routed yet — there is nothing to say.
func (m rtFlowModel) pickLabel() string {
	if !m.Has {
		return ""
	}
	if m.Newest.Decision == "" {
		return "智能路由"
	}
	return m.Newest.Decision
}

// caption says what the newest request did, in one line. With nothing routed
// yet it says what the stage will hold once there is: the gateway's own
// requests, nothing made up.
func (m rtFlowModel) caption(now time.Time) string {
	if !m.Has {
		return "Agent 发给 Echo 的每个请求都会在此显示，均为真实路由。"
	}
	tr := m.Newest
	agent := usAgentName(tr.Agent)
	if tr.Agent == "" {
		agent = "agent"
	}
	model := tr.ServedModel
	if model == "" {
		model = tr.RequestedModel
	}
	where := model
	if tr.ProviderName != "" && model != "" {
		where = tr.ProviderName + " / " + model
	}
	out := agent + " → " + where
	if tr.Millis > 0 {
		out += " · " + FormatDuration(tr.Millis)
	}
	if tr.TTFT > 0 {
		out += " · 首字 " + FormatDuration(tr.TTFT)
	}
	if tr.Error != "" {
		out += " · 失败：" + tr.Error
	} else if tr.Done && rtFlowFailed(tr) {
		out += fmt.Sprintf(" · 失败：HTTP %d", tr.Status)
	}
	if !tr.Done {
		out += " · 进行中"
	}
	return out
}

// rtFlowAgo says how long ago a request was, in seconds while that is what
// tells one request from the next.
func rtFlowAgo(now, at time.Time) string {
	d := now.Sub(at)
	switch {
	case d < 0:
		return "刚刚"
	case d < time.Second:
		return "刚刚"
	case d < time.Minute:
		return fmt.Sprintf("%d 秒前", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%d 分钟前", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%d 小时前", int(d.Hours()))
	}
	return fmt.Sprintf("%d 天前", int(d.Hours()/24))
}

// ---- the drawing ----------------------------------------------------------

// rtFlowPaint draws the wires between the columns and the packets on them,
// from the trace the gateway kept.
func (a *App) rtFlowPaint(p *ui.Painter, r ui.Rect, t *ui.Theme, m rtFlowModel, traces []RoutingTraceRow) {
	if r.W <= 0 || r.H <= 0 || len(m.Agents) == 0 {
		return
	}
	now := p.Now()

	agentX := r.X + rtFlowPadX
	hubX := agentX + rtFlowAgentW + rtFlowColGap
	acctX := hubX + rtFlowHubW + rtFlowColGap
	hubCy := r.Y + r.H/2
	agentCy := func(i int) float32 {
		n := float32(len(m.Agents))
		h := n*rtFlowAgentH + (n-1)*rtFlowAgentGap
		return r.Y + (r.H-h)/2 + float32(i)*(rtFlowAgentH+rtFlowAgentGap) + rtFlowAgentH/2
	}
	acctCy := func(i int) float32 {
		n := float32(len(m.Accts))
		h := n*rtFlowAcctH + (n-1)*rtFlowAcctGap
		return r.Y + (r.H-h)/2 + float32(i)*(rtFlowAcctH+rtFlowAcctGap) + rtFlowAcctH/2
	}

	// the wires: an agent into magpie, magpie out to each account
	wire := func(from, to rtPt, c ui.Color) {
		mid := (to.X - from.X) * 0.45
		path := (&ui.Path{}).MoveTo(from.X, from.Y).
			CubeTo(from.X+mid, from.Y, to.X-mid, to.Y, to.X, to.Y)
		p.StrokePath(path, 1.3, c)
	}
	for i := range m.Agents {
		wire(rtPt{agentX + rtFlowAgentW, agentCy(i)}, rtPt{hubX, hubCy}, t.Border)
	}
	for j := range m.Accts {
		wire(rtPt{hubX + rtFlowHubW, hubCy}, rtPt{acctX, acctCy(j)}, t.Border)
	}

	flights, replay := m.flights(traces, now)
	if len(flights) == 0 {
		return
	}
	bg := a.rtFlowNodeBg(t)
	for _, f := range flights {
		a0 := rtPt{agentX + rtFlowAgentW, agentCy(f.agent)}
		hL := rtPt{hubX, hubCy}
		hR := rtPt{hubX + rtFlowHubW, hubCy}
		stops := make([]rtPt, 0, len(f.path))
		for _, ci := range f.path {
			stops = append(stops, rtPt{acctX, acctCy(ci)})
		}

		hue := f.hue
		if replay {
			hue = hue.Alpha(0.5)
		}
		// the way this request goes, lit in the agent's colour
		wire(a0, hL, hue.Alpha(0.55))
		for _, c := range stops {
			wire(hR, c, hue.Alpha(0.55))
		}

		mark := rtFlowAt(f, now, a0, hL, hR, stops)
		if !mark.ok {
			continue
		}
		alpha := float32(1)
		if replay {
			alpha = 0.6
		}
		if mark.hold {
			alpha *= 0.55 + 0.45*float32(math.Sin(float64(now.UnixMilli())/260))
		}
		const sz = 7.0
		box := ui.Rect{X: mark.at.X - sz/2, Y: mark.at.Y - sz/2, W: sz, H: sz}
		if mark.back {
			// coming back: a ring in the agent's colour, red for a failure
			p.Fill(ui.Rect{X: mark.at.X - 2.2, Y: mark.at.Y - 2.2, W: 4.4, H: 4.4}, bg, 2.2)
			ring := hue
			if mark.fail {
				ring = ui.Hex("#ff3b30")
			}
			p.Stroke(box, ring.Alpha(alpha), sz/2, 2)
			continue
		}
		// going out: filled, with a halo so it reads over the wires
		p.Fill(ui.Rect{X: mark.at.X - 7, Y: mark.at.Y - 7, W: 14, H: 14}, hue.Alpha(0.18*alpha), 7)
		p.Fill(box, hue.Alpha(alpha), sz/2)
	}
	// a flight is moving: ask for the next frame while it is
	p.After(rtFlowFrame)
}

// rtFlowMark is where a flight's packet is at a moment, and what it carries:
// a request going out, a failure coming back, or an answer coming back.
type rtFlowMark struct {
	at   rtPt
	back bool // on its way back to the agent
	fail bool // a try that could not answer, or a request that failed
	hold bool // waiting at an account
	ok   bool // there is a packet to draw at all
}

// rtFlowAt is where a flight's packet is at now. A request that failed over
// goes out again from magpie to each account it tried, in the order it tried
// them, and the answer comes back from the one that answered.
func rtFlowAt(f rtFlowFlight, now time.Time, a0, hL, hR rtPt, stops []rtPt) rtFlowMark {
	if f.dur <= 0 || len(stops) == 0 {
		return rtFlowMark{}
	}
	const (
		outAt  = 0.14 // the agent's wire, into magpie
		tryAt  = 0.52 // the accounts it tried
		backAt = 0.58 // the answer, from the one that answered
		landAt = 0.94
	)
	prog := float32(now.Sub(f.start)) / float32(f.dur)
	last := stops[len(stops)-1]
	if f.live && prog >= tryAt {
		// still being answered: it waits at the account
		return rtFlowMark{at: last, hold: true, ok: true}
	}
	switch {
	case prog < 0:
		return rtFlowMark{}
	case prog < outAt: // out of the agent, into magpie
		return rtFlowMark{at: rtLeg(a0, hL, prog/outAt), ok: true}
	case prog < tryAt: // magpie, then each account it tried
		u := (prog - outAt) / (tryAt - outAt) * float32(len(stops))
		i := int(u)
		if i >= len(stops) {
			i = len(stops) - 1
		}
		local := u - float32(i)
		from := hR
		if i == 0 {
			from = hL
		}
		switch {
		case local < 0.6:
			return rtFlowMark{at: rtLeg(from, stops[i], local/0.6), ok: true}
		case i == len(stops)-1: // with the account
			return rtFlowMark{at: stops[i], hold: true, ok: true}
		default: // it could not answer: back to magpie, and on
			return rtFlowMark{at: rtLeg(stops[i], hR, (local-0.6)/0.4), back: true, fail: true, ok: true}
		}
	case prog < backAt: // with the account
		return rtFlowMark{at: last, hold: true, ok: true}
	case prog < landAt: // the answer back: the account, magpie, the agent
		u := (prog - backAt) / (landAt - backAt)
		if u < 0.5 {
			return rtFlowMark{at: rtLeg(last, hR, u*2), back: true, fail: f.failed, ok: true}
		}
		return rtFlowMark{at: rtLeg(hL, a0, (u-0.5)*2), back: true, fail: f.failed, ok: true}
	}
	return rtFlowMark{back: true} // landed
}

// rtLeg is a point along a wire, as a cubic running level out of each end.
func rtLeg(from, to rtPt, u float32) rtPt {
	mid := (to.X - from.X) * 0.45
	return rtCubicAt(from, rtPt{from.X + mid, from.Y}, rtPt{to.X - mid, to.Y}, to, u)
}

// rtCubicAt is a point on a cubic Bézier.
func rtCubicAt(p0, p1, p2, p3 rtPt, u float32) rtPt {
	v := 1 - u
	a := v * v * v
	b := 3 * v * v * u
	c := 3 * v * u * u
	d := u * u * u
	return rtPt{
		X: a*p0.X + b*p1.X + c*p2.X + d*p3.X,
		Y: a*p0.Y + b*p1.Y + c*p2.Y + d*p3.Y,
	}
}
