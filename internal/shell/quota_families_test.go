//go:build nogui && echo

package shell

import (
	"strings"
	"testing"
	"time"

	"github.com/yetone/magpie/internal/provider"
)

// TestQuotaRowsCollapses checks the rows a card draws of an account's
// windows: a pool's own two, its models' said in the tooltip; a family's
// tightest one, the family's name on it; and the windows that name
// neither as they were. It is magpie's own collapsing (familyWindows and
// poolWindows in its app.js), which the panel and the page must agree
// on, or they say different things about the same account.
func TestQuotaRowsCollapses(t *testing.T) {
	soon := time.Now().Add(3 * time.Hour)
	later := time.Now().Add(30 * time.Hour)

	// Antigravity's shape: two pools of models, each pool's own 5-hour
	// and weekly windows, and each model's own per level
	antigravity := []provider.QuotaWindow{
		{Name: "Gemini 3.8 Flash (High)", Used: 100, Family: "Gemini", Pool: "Gemini"},
		{Name: "Gemini 3.8 Flash (Low)", Used: 100, Family: "Gemini", Pool: "Gemini"},
		{Name: "Gemini 3.8 Flash (Medium)", Used: 100, Family: "Gemini", Pool: "Gemini"},
		{Name: "Claude Sonnet 4.6", Used: 5, Family: "Claude", Pool: "Claude & GPT"},
		{Name: "5 hours", Used: 19, ResetsAt: &soon, Pool: "Gemini"},
		{Name: "7 days", Used: 30, ResetsAt: &later, Pool: "Gemini"},
		{Name: "5 hours", Used: 0, ResetsAt: &soon, Pool: "Claude & GPT"},
		{Name: "7 days", Used: 0, ResetsAt: &later, Pool: "Claude & GPT"},
	}
	rows := quotaRows(antigravity)
	want := []struct {
		name string
		used float64
	}{
		{"Gemini · 5 hours", 19},
		{"Gemini · 7 days", 30},
		{"Claude & GPT · 5 hours", 0},
		{"Claude & GPT · 7 days", 0},
	}
	if len(rows) != len(want) {
		t.Fatalf("expected %d rows, got %d: %+v", len(want), len(rows), rows)
	}
	for i, w := range want {
		if rows[i].w.Name != w.name || rows[i].w.Used != w.used {
			t.Errorf("row %d: expected %q at %.2f, got %q at %.2f",
				i, w.name, w.used, rows[i].w.Name, rows[i].w.Used)
		}
		if rows[i].tool == "" {
			t.Errorf("row %d (%q): expected the pool's tooltip", i, w.name)
		}
	}
	if !strings.Contains(rows[0].tool, "Gemini 3.8 Flash (High)") || !strings.Contains(rows[0].tool, "共用的一个额度") {
		t.Errorf("the Gemini pool's tooltip should name the pool and its models; it said %q", rows[0].tool)
	}

	// a family without a pool: one row a family, its tightest window the
	// figure, Gemini then Claude then the bare ones
	familied := []provider.QuotaWindow{
		{Name: "5 hours", Used: 19, ResetsAt: &soon},
		{Name: "Gemini 3.8 Flash (Low)", Used: 40, Family: "Gemini"},
		{Name: "Claude Opus 4", Used: 50, Family: "Claude", ResetsAt: &later},
		{Name: "Gemini 3.8 Flash (High)", Used: 100, Family: "Gemini"},
	}
	rows = quotaRows(familied)
	wantNames := []string{"Gemini", "Claude", "5 hours"}
	wantUsed := []float64{100, 50, 19}
	if len(rows) != len(wantNames) {
		t.Fatalf("expected %d rows, got %d: %+v", len(wantNames), len(rows), rows)
	}
	for i, w := range wantNames {
		if rows[i].w.Name != w || rows[i].w.Used != wantUsed[i] {
			t.Errorf("row %d: expected %q at %.2f, got %q at %.2f",
				i, w, wantUsed[i], rows[i].w.Name, rows[i].w.Used)
		}
	}
	if rows[0].tool == "" || !strings.Contains(rows[0].tool, "Gemini 3.8 Flash (Low)") {
		t.Errorf("the Gemini family's tooltip should say each model's own; it said %q", rows[0].tool)
	}

	// windows that name neither a family nor a pool are as they were
	plain := []provider.QuotaWindow{
		{Name: "5 小时", Used: 42, ResetsAt: &soon},
		{Name: "本周", Used: 91},
	}
	rows = quotaRows(plain)
	if len(rows) != 2 || rows[0].w.Name != "5 小时" || rows[1].w.Name != "本周" {
		t.Errorf("expected the plain windows as they were, got %+v", rows)
	}
	for i, r := range rows {
		if r.tool != "" {
			t.Errorf("row %d: a plain window wants no tooltip, got %q", i, r.tool)
		}
	}
}
