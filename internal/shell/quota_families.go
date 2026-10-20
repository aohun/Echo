//go:build nogui && echo

package shell

// An account of a subscription can report one window a model —
// Antigravity's "Gemini 3.8 Flash (High)", "(Low)", "(Medium)"… — which
// drawn as it comes reads as a wall of meters saying the same thing
// several times over. magpie's own cards collapse those first
// (familyWindows and poolWindows in its app.js): the windows that share
// a pool become the pool's own rows, its shortest span first, named by
// the pool with the models it counts said in the tooltip; the rest are
// one row a family, its tightest window — the most used, the soonest to
// start again of those — the figure, and the family's name on it; and
// the windows that name neither are as they were. The page and the
// panel both draw the collapsed rows, so they cannot say different
// things about the same account.

import (
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/yetone/magpie/internal/provider"
)

// quotaRow is one row after collapsing: the window whose figure it
// carries — named by its family or pool when it stands for one — and
// what the tooltip says of it.
type quotaRow struct {
	w    provider.QuotaWindow
	tool string
}

// quotaFamilyFirst is who goes ahead of the others, as magpie's own do.
var quotaFamilyFirst = []string{"Gemini", "Claude"}

// quotaSpanRe reads a window's span off its name ("5 hours", "7 days").
var quotaSpanRe = regexp.MustCompile(`(?i)^(\d+)\s*(hour|day|week)s?$`)

// quotaRows is the rows a card draws of an account's windows.
func quotaRows(ws []provider.QuotaWindow) []quotaRow {
	// a pool's own windows (they name the pool and no family) are rows
	// of their own; a pool's models' fold into their tooltips
	pools := map[string][]provider.QuotaWindow{}
	for _, w := range ws {
		if w.Pool != "" && w.Family == "" {
			pools[w.Pool] = append(pools[w.Pool], w)
		}
	}
	if len(pools) == 0 {
		return quotaFamilyRows(ws)
	}
	names := make([]string, 0, len(pools))
	for name := range pools {
		names = append(names, name)
	}
	sort.SliceStable(names, func(i, j int) bool {
		return quotaFamilyRank(names[i]) < quotaFamilyRank(names[j])
	})
	var out []quotaRow
	for _, pool := range names {
		var members []provider.QuotaWindow
		for _, w := range ws {
			if w.Family != "" && w.Pool == pool {
				members = append(members, w)
			}
		}
		own := pools[pool]
		sort.SliceStable(own, func(i, j int) bool {
			return quotaSpanHours(own[i]) < quotaSpanHours(own[j])
		})
		for _, w := range own {
			named := w
			named.Name = pool + " · " + w.Name
			out = append(out, quotaRow{w: named, tool: quotaPoolTool(pool, members)})
		}
	}
	// what is left — windows in no pool shown here — goes on as families
	var rest []provider.QuotaWindow
	for _, w := range ws {
		if w.Pool != "" && w.Family == "" {
			continue // the pool's own row, drawn above
		}
		if _, in := pools[w.Pool]; w.Pool != "" && in {
			continue // one of the pool's models: its tooltip says them
		}
		rest = append(rest, w)
	}
	return append(out, quotaFamilyRows(rest)...)
}

// quotaFamilyRows is an account's windows one row a family where they
// name one, the rest as they were.
func quotaFamilyRows(ws []provider.QuotaWindow) []quotaRow {
	family := false
	for _, w := range ws {
		if w.Family != "" {
			family = true
			break
		}
	}
	if !family {
		out := make([]quotaRow, len(ws))
		for i, w := range ws {
			out[i] = quotaRow{w: w}
		}
		return out
	}
	// the windows of a family together, the others each its own, first
	// seen first
	type grp struct {
		family string
		ws     []provider.QuotaWindow
	}
	var groups []grp
	at := map[string]int{}
	for _, w := range ws {
		key := "w:" + strconv.Itoa(len(groups))
		if w.Family != "" {
			key = "f:" + w.Family
		}
		if i, ok := at[key]; ok {
			groups[i].ws = append(groups[i].ws, w)
			continue
		}
		at[key] = len(groups)
		groups = append(groups, grp{family: w.Family, ws: []provider.QuotaWindow{w}})
	}
	if len(groups) == len(ws) {
		out := make([]quotaRow, len(ws))
		for i, w := range ws {
			out[i] = quotaRow{w: w}
		}
		return out
	}
	out := make([]quotaRow, 0, len(groups))
	for _, g := range groups {
		if g.family == "" {
			out = append(out, quotaRow{w: g.ws[0]})
			continue
		}
		// the family's figure is its tightest window: the most used, the
		// soonest to start again of those
		top := g.ws[0]
		for _, w := range g.ws[1:] {
			if w.Used > top.Used || (w.Used == top.Used && quotaSoon(w) < quotaSoon(top)) {
				top = w
			}
		}
		named := top
		named.Name = top.Family
		out = append(out, quotaRow{w: named, tool: quotaFamilyTool(named.Name, g.ws)})
	}
	// Gemini, then Claude, then the rest in the order they came
	sort.SliceStable(out, func(i, j int) bool {
		return quotaFamilyRank(out[i].w.Name) < quotaFamilyRank(out[j].w.Name)
	})
	return out
}

// quotaFamilyRank is where a family or pool's first word sorts.
func quotaFamilyRank(name string) int {
	first := name
	if i := strings.IndexAny(name, " &"); i >= 0 {
		first = name[:i]
	}
	for i, f := range quotaFamilyFirst {
		if strings.EqualFold(first, f) {
			return i
		}
	}
	return len(quotaFamilyFirst)
}

// quotaSpanHours is a window's span in hours, for a pool's rows short
// span first; unknown spans last.
func quotaSpanHours(w provider.QuotaWindow) float64 {
	m := quotaSpanRe.FindStringSubmatch(w.Name)
	if m == nil {
		return 1e18
	}
	n, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return 1e18
	}
	switch strings.ToLower(m[2]) {
	case "hour":
		return n
	case "day":
		return n * 24
	case "week":
		return n * 168
	}
	return 1e18
}

// quotaSoon is when a window starts again; never, when unknown.
func quotaSoon(w provider.QuotaWindow) int64 {
	if w.ResetsAt == nil {
		return 1<<62 - 1
	}
	return w.ResetsAt.Unix()
}

// quotaFamilyTool is what a family's row says of itself: its figure is
// the most used of its models, each model's own beneath.
func quotaFamilyTool(family string, tiers []provider.QuotaWindow) string {
	var b strings.Builder
	b.WriteString(family + "：一族里用得最多的那个")
	for _, x := range tiers {
		b.WriteString("\n" + quotaTierLine(x))
	}
	return b.String()
}

// quotaPoolTool is what a pool's row says of itself: one allowance for
// these models, and which they are.
func quotaPoolTool(pool string, members []provider.QuotaWindow) string {
	var b strings.Builder
	b.WriteString(pool + "：这些模型共用的一个额度")
	for _, x := range members {
		b.WriteString("\n" + quotaTierLine(x))
	}
	return b.String()
}

// quotaTierLine is one of a row's windows in words: its name, what it is
// at, and when the used ones start again.
func quotaTierLine(w provider.QuotaWindow) string {
	line := w.Name + " " + quotaUsedWords(w)
	if w.ResetsAt != nil && w.Used > 0 {
		line += " · 重置于 " + usResetIn(*w.ResetsAt)
	}
	return line
}

// quotaUsedWords is a window's figure in words: the share used, with its
// own count beside it when the vendor counts in amounts. Used is the
// vendor's own percent (0-100).
func quotaUsedWords(w provider.QuotaWindow) string {
	if w.Unlimited {
		return "不限"
	}
	used := w.Used
	if used < 0 {
		used = 0
	}
	if used > 100 {
		used = 100
	}
	pct := "已用 " + strconv.FormatFloat(used, 'f', 1, 64) + "%"
	if n := int(used); float64(n) == used {
		pct = "已用 " + strconv.Itoa(n) + "%"
	}
	if w.Limit > 0 {
		return usAmount(w.Amount) + " / " + usAmount(w.Limit) + " " + w.Unit + " · " + pct
	}
	return pct
}

// quotaRowsOf is quotaRows for a whole account, an error account's
// windows drawn as they were.
func quotaRowsOf(q provider.SubscriptionQuota) []quotaRow {
	if q.Error != "" {
		out := make([]quotaRow, len(q.Windows))
		for i, w := range q.Windows {
			out[i] = quotaRow{w: w}
		}
		return out
	}
	return quotaRows(q.Windows)
}

// quotaCollapsed says whether any of the rows stands for a family or a
// pool rather than a window of its own.
func quotaCollapsed(rows []quotaRow) bool {
	for _, r := range rows {
		if r.tool != "" {
			return true
		}
	}
	return false
}

// quotaPooled says whether an account's windows are a pool's: its own
// rows and the models drawing on them.
func quotaPooled(ws []provider.QuotaWindow) bool {
	for _, w := range ws {
		if w.Pool != "" && w.Family == "" {
			return true
		}
	}
	return false
}

// quotaModelRows is an account's windows as they were, a pool's own
// apart: the models' own, for the page's "Every model" switch.
func quotaModelRows(ws []provider.QuotaWindow) []quotaRow {
	poolRow := false
	for _, w := range ws {
		if w.Pool != "" && w.Family == "" {
			poolRow = true
			break
		}
	}
	out := make([]quotaRow, 0, len(ws))
	for _, w := range ws {
		if poolRow && w.Pool != "" && w.Family == "" {
			continue
		}
		out = append(out, quotaRow{w: w})
	}
	return out
}
