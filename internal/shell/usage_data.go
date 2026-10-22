//go:build nogui && echo

package shell

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/yetone/magpie/internal/agent"
	"github.com/yetone/magpie/internal/appdir"
	"github.com/yetone/magpie/internal/edit"
	"github.com/yetone/magpie/internal/gateway"
	"github.com/yetone/magpie/internal/gui"
	"github.com/yetone/magpie/internal/provider"
	"github.com/yetone/magpie/internal/sessions"
	"github.com/yetone/magpie/internal/usage"
	"github.com/yetone/magpie/internal/usageapi"
)

// The Usage page's four segments, as magpie's own page names them.
const (
	UsTabOverview = "overview"
	UsTabRequests = "requests"
	UsTabSessions = "sessions"
	UsTabContext  = "context"
)

func init() {
	if usageapi.Clients == nil {
		usageapi.Clients = func() []usageapi.Client {
			var out []usageapi.Client
			for _, a := range agent.Clients() {
				out = append(out, usageapi.Client{ID: a.ID, Name: a.Name, Icon: a.Icon})
			}
			return out
		}
	}
}

// UsageQuery is what the Usage page is asking for: the segment shown, the
// period its head picked, and every filter the Requests and Sessions
// segments offer.
//
// It is one value so the page can tell whether what it holds was read for
// the filters it is showing (App.usFresh): a pick that changed it reads
// again, a frame that did not draws what it has.
type UsageQuery struct {
	Tab    string
	Period usage.Period
	Day    string

	Agent      string
	Provider   string
	Purpose    string
	Account    string
	CallerKey  string
	Model      string
	Status     string
	Query      string
	Via        string
	Offset     int
	Limit      int
	SessRange  string
	SessAgent  string
	SessModel  string
	SessFolder string
	SessQuery  string
	CtxDays    int
}

// UsageData is one read of the page: only what the segment shown draws, as
// magpie's page reads its own segment alone.
type UsageData struct {
	Query UsageQuery

	// Overview
	Overview      usageapi.Usage
	Quotas        []provider.SubscriptionQuota
	QuotasReading bool

	// Requests
	Ledger  usageapi.Ledger
	Heatmap usage.Heatmap

	// Sessions
	SessionsStats    sessions.Stats
	SessionsOverview sessions.Overview
	SessionsList     []sessions.Summary
	SessionsAgents   []string

	// Context
	Context gui.ContextData
}

// ContentJSON is what was said in a request, or why it can't be told.
type ContentJSON struct {
	Found bool   `json:"found"`
	Why   string `json:"why,omitempty"`
	Model string `json:"model,omitempty"`
	sessions.Content
}

// ReadRequestContent reads the prompt input and output reply of a call from agent session files.
func ReadRequestContent(agentName, sessionID string, from, to, at time.Time) ContentJSON {
	out := ContentJSON{Content: sessions.Content{Input: []sessions.Part{}, Output: []sessions.Part{}}}
	switch {
	case sessionID == "":
		out.Why = "session"
	case agentName != "claude" && agentName != "claude-desktop" && agentName != "codex":
		out.Why = "agent"
	default:
		c, ok := sessions.FindCall(sessionID, from, to, at)
		if !ok {
			out.Why = "missing"
			break
		}
		content, err := sessions.ContentOf(c)
		if err != nil {
			out.Why = "read"
			break
		}
		out.Found, out.Model, out.Content = true, c.Model, content
	}
	return out
}

var (
	quotasCacheMu   sync.Mutex
	quotasLastRead  time.Time
	cachedQuotas    []provider.SubscriptionQuota
	cachedQuotasErr bool
)

// ReadQuotas returns subscription quota cards, utilizing caching to prevent laggy vendor calls.
func ReadQuotas(force bool) []provider.SubscriptionQuota {
	quotasCacheMu.Lock()
	if !force && len(cachedQuotas) > 0 && time.Since(quotasLastRead) < 45*time.Second {
		qs := cachedQuotas
		quotasCacheMu.Unlock()
		return qs
	}
	quotasCacheMu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()

	if force {
		provider.AskClaudeUsage()
		ctx = provider.Asked(ctx)
	}

	qs := provider.WithCheckins(provider.Quotas(ctx))
	qs = usage.WithWindowHolds(qs, usage.Clock())
	qs = provider.WithWindowStarts(qs, usage.Clock())

	quotasCacheMu.Lock()
	cachedQuotas = qs
	quotasLastRead = time.Now()
	quotasCacheMu.Unlock()

	return qs
}

// RefreshQuotaCard refreshes a single quota card from its vendor.
func RefreshQuotaCard(providerID, user string) []provider.SubscriptionQuota {
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	provider.RefreshUsage(ctx, providerID, user)
	return ReadQuotas(true)
}

// PeekQuotas is what the quota cache holds, read again from no one: what
// a page may show without waiting on a vendor, as the Settings page's
// menu-bar allowances picker does.
func PeekQuotas() []provider.SubscriptionQuota {
	quotasCacheMu.Lock()
	defer quotasCacheMu.Unlock()
	return cachedQuotas
}

// InvalidateQuotas stands this shell's quota cache down: a vendor's
// background read has landed (provider.OnSubscriptionUsage), and the
// provider's cache now holds the fresh cards, so the next read must go
// there rather than serve this one's 45 seconds of what may be the stale
// copy the landed read replaces.
func InvalidateQuotas() {
	quotasCacheMu.Lock()
	quotasLastRead = time.Time{}
	quotasCacheMu.Unlock()
}

// usageFilterOf is the ledger filter a query names.
func usageFilterOf(q UsageQuery) usage.Filter {
	return usage.Filter{
		Day:       q.Day,
		Agent:     q.Agent,
		Provider:  q.Provider,
		Purpose:   q.Purpose,
		Account:   q.Account,
		CallerKey: q.CallerKey,
		Model:     q.Model,
		Query:     q.Query,
		Failed:    q.Status == "failed",
		Retried:   q.Status == "retried",
		Through:   q.Via,
	}
}

// sessDaysOf is how many days back a Sessions range reaches; 0 is all time.
func sessDaysOf(rangeID string) int {
	switch rangeID {
	case "today":
		return 1
	case "7d":
		return 7
	case "90d":
		return 90
	case "all":
		return 0
	}
	return 30
}

// ReadUsageData reads what one segment of the Usage page draws.
//
// It is magpie's own read per segment: the Overview asks for the summary of
// the period and the allowances, Requests for a page of the ledger and the
// heatmap, Sessions for the agents' session files, and Context for the
// routing history's prompts. Nothing else is read, so switching segments
// costs what that segment costs and no more.

// usageReadBox is how long a usage read may take before the last good
// data stands in, like the plugins read's box. A var so a test can
// shorten it.
var usageReadBox = 2 * time.Second

// ReadUsageData reads what one segment of the Usage page draws.
//
// It is magpie's own read per segment: the Overview asks for the summary of
// the period and the allowances, Requests for a page of the ledger and the
// heatmap, Sessions for the agents' session files, and Context for the
// routing history's prompts. Nothing else is read, so switching segments
// costs what that segment costs and no more.
func ReadUsageData(q UsageQuery) UsageData {
	if q.Tab == "" {
		q.Tab = UsTabOverview
	}
	if q.Period == "" {
		q.Period = usage.Month
	}
	if q.Limit <= 0 {
		q.Limit = 50
	}
	if q.SessRange == "" {
		q.SessRange = "30d"
	}
	if q.CtxDays <= 0 {
		q.CtxDays = 7
	}

	// The instance's usage answers can wait on the network through its
	// plugin host (an auth refresh against a vendor timed out under it
	// for twelve seconds, and the reload paid every one): bound the read
	// like the plugins read — the last good data stands in, and the read
	// that finally lands is kept for the next pass.
	done := make(chan UsageData, 1)
	go func() { done <- readUsageInner(q) }()
	select {
	case got := <-done:
		return got
	case <-time.After(usageReadBox):
		go func() {
			if got := <-done; got.Quotas != nil || got.Ledger.Rows != nil || got.SessionsStats.Days != nil {
				if cached, ok := loadDataCache(); ok {
					cached.Usage = got
					saveDataCache(cached)
				}
			}
		}()
		if cached, ok := loadDataCache(); ok {
			got := cached.Usage
			got.Query = q
			return got
		}
		return UsageData{Query: q}
	}
}

// readUsageInner is the read itself: the instance's answer when the seam
// points at one, the embedded read otherwise.
func readUsageInner(q UsageQuery) UsageData {
	ud := UsageData{Query: q}

	if c := remoteGateway(); c != nil {
		if r, ok := readUsageRemote(c, q); ok {
			return r
		}
	}

	switch q.Tab {
	case UsTabRequests:
		f := usageFilterOf(q)
		ud.Ledger = usageapi.LedgerPage(q.Period, f, q.Offset, q.Limit)
		ud.Heatmap = usage.HeatmapOf(f)
	case UsTabSessions:
		days := sessDaysOf(q.SessRange)
		st := sessions.StatsFor(days)
		ud.SessionsStats = st
		// the overview is the range's own answer under the three filters
		// the segment offers, "" meaning any
		ag := q.SessAgent
		if ag == "all" {
			ag = ""
		}
		ud.SessionsOverview = st.Overview(ag, q.SessModel, q.SessFolder, 10)
		ud.SessionsList = filterSessions(st, q)
		ud.SessionsAgents = sessionAgents(st)
	case UsTabContext:
		ud.Context = gui.ContextOf(q.CtxDays)
	default:
		ud.Overview = usageapi.Page(q.Period)
		ud.Quotas = ReadQuotas(false)
		ud.QuotasReading = provider.SubscriptionUsageReading()
	}

	return ud
}

// sessionAgents is every agent with a session in the range, by name.
func sessionAgents(st sessions.Stats) []string {
	set := map[string]bool{}
	for _, s := range st.Sessions {
		if s.Agent != "" {
			set[s.Agent] = true
		}
	}
	out := make([]string, 0, len(set))
	for a := range set {
		out = append(out, a)
	}
	sort.Strings(out)
	return out
}

// filterSessions is the sessions the Sessions segment lists: the range's
// own (Stats.Keys, not a guess from each session's last line), under the
// agent, model, folder and search the head offers.
func filterSessions(st sessions.Stats, q UsageQuery) []sessions.Summary {
	inRange := map[string]bool{}
	for _, k := range st.Keys {
		inRange[k] = true
	}
	needle := strings.ToLower(strings.TrimSpace(q.SessQuery))
	out := make([]sessions.Summary, 0, len(st.Sessions))
	for _, s := range st.Sessions {
		if len(inRange) > 0 && !inRange[s.Key] {
			continue
		}
		if q.SessAgent != "" && q.SessAgent != "all" && s.Agent != q.SessAgent {
			continue
		}
		if q.SessFolder != "" && s.Cwd != q.SessFolder {
			continue
		}
		if q.SessModel != "" && !slices.Contains(s.Models, q.SessModel) {
			continue
		}
		if needle != "" {
			hay := strings.ToLower(s.Title + "\x00" + s.Cwd + "\x00" + s.ID + "\x00" + strings.Join(s.Models, "\x00"))
			if !strings.Contains(hay, needle) {
				continue
			}
		}
		out = append(out, s)
	}
	return out
}

// ExportUsageCSV exports filtered ledger rows as a CSV file to ~/Downloads.
func ExportUsageCSV(period usage.Period, f usage.Filter) (string, int, error) {
	now := usage.Clock()
	rows := usage.LedgerOfAt(period, f, now).Rows
	var b bytes.Buffer
	if err := usage.WriteCSV(&b, rows); err != nil {
		return "", 0, err
	}

	home, err := appdir.Home()
	if err != nil {
		home, _ = os.UserHomeDir()
	}
	dir := filepath.Join(home, "Downloads")
	_ = os.MkdirAll(dir, 0755)

	pStr := strings.ReplaceAll(string(period), ":", "-")
	if f.Day != "" {
		pStr = "day-" + f.Day
	}
	stamp := fmt.Sprintf("Echo-requests-%s-%s", pStr, now.Format("20060102-150405"))
	name := filepath.Join(dir, stamp+".csv")

	for i := 2; ; i++ {
		if _, err := os.Stat(name); err != nil {
			break
		}
		name = filepath.Join(dir, fmt.Sprintf("%s-%d.csv", stamp, i))
	}

	if err := edit.WriteAtomic(name, b.Bytes()); err != nil {
		return "", 0, err
	}

	return name, len(rows), nil
}

// FormatCost formats floating point currency amount nicely.
func FormatCost(c float64) string {
	if c <= 0.0001 && c > 0 {
		return "$0.0001"
	}
	if c < 1.0 {
		return fmt.Sprintf("$%.4f", c)
	}
	return fmt.Sprintf("$%.2f", c)
}

// FormatTokens formats token counts with k/M suffixes.
func FormatTokens(t int) string {
	if t >= 1_000_000 {
		return fmt.Sprintf("%.2fM", float64(t)/1_000_000.0)
	}
	if t >= 1_000 {
		return fmt.Sprintf("%.1fk", float64(t)/1_000.0)
	}
	return strconv.Itoa(t)
}

// FormatCount writes a count the way the page's tiles do: thousands
// separated, so a long history's 750,000 reads as that and not 750000.
func FormatCount(n int) string {
	s := strconv.Itoa(n)
	if len(s) <= 3 {
		return s
	}
	var b strings.Builder
	lead := len(s) % 3
	if lead == 0 {
		lead = 3
	}
	b.WriteString(s[:lead])
	for i := lead; i < len(s); i += 3 {
		b.WriteByte(',')
		b.WriteString(s[i : i+3])
	}
	return b.String()
}

// FormatDuration formats milliseconds into human-readable duration string.
func FormatDuration(ms int64) string {
	if ms <= 0 {
		return "-"
	}
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
	}
	sec := float64(ms) / 1000.0
	if sec < 60 {
		return fmt.Sprintf("%.1fs", sec)
	}
	min := int(sec / 60)
	remainingSec := int(sec) % 60
	return fmt.Sprintf("%dm%ds", min, remainingSec)
}

// FormatLatency formats latency in ms.
func FormatLatency(ms int) string {
	if ms <= 0 {
		return "-"
	}
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
	}
	return fmt.Sprintf("%.2fs", float64(ms)/1000.0)
}

// FormatSeconds writes a span of seconds as the Sessions segment's tiles
// do: 45m, 12h 30m, 3d 4h.
func FormatSeconds(sec int64) string {
	if sec <= 0 {
		return "—"
	}
	switch {
	case sec < 60:
		return fmt.Sprintf("%ds", sec)
	case sec < 3600:
		return fmt.Sprintf("%dm", sec/60)
	case sec < 86400:
		h, m := sec/3600, (sec%3600)/60
		if m == 0 {
			return fmt.Sprintf("%dh", h)
		}
		return fmt.Sprintf("%dh %dm", h, m)
	default:
		d, h := sec/86400, (sec%86400)/3600
		if h == 0 {
			return fmt.Sprintf("%dd", d)
		}
		return fmt.Sprintf("%dd %dh", d, h)
	}
}

// usageTokensOf is a total or a group's tokens as the page counts them:
// in and out, and, with the reader's choice, the cache's read and written.
func usageTokensOf(t usage.Totals, withCache bool) int {
	n := t.Input + t.Output
	if withCache {
		n += t.CacheRead + t.CacheWrite
	}
	return n
}

// cacheHitRate is the share of a prompt the vendor read from its cache:
// input leaves the cached tokens out, so the prompt is the three together.
func cacheHitRate(t usage.Totals) float64 {
	prompt := t.Input + t.CacheRead + t.CacheWrite
	if prompt <= 0 {
		return 0
	}
	return float64(t.CacheRead) / float64(prompt)
}

// outputSpeed is the speed the streamed replies wrote at, from the totals'
// measured decode span: the answer tokens after the first, a second.
func outputSpeed(t usage.Totals) float64 {
	if t.DecodeMs <= 0 || t.DecodeOut <= 0 {
		return 0
	}
	return float64(t.DecodeOut) / (float64(t.DecodeMs) / 1000)
}

// ttftOf is the mean wait for a streamed reply's first token, in ms.
func ttftOf(t usage.Totals) int64 {
	if t.Timed <= 0 {
		return 0
	}
	return t.TTFT / int64(t.Timed)
}

// gatewayServed is the gateway this app serves, when it runs here.
func gatewayServed() *gateway.Server { return guiServed() }
