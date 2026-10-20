//go:build nogui && echo

package shell

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/yetone/magpie/internal/provider"
)

// TestQuotaRingWords checks the two lines under a ring: what the window is
// for, and how long it runs — magpie's own words.
func TestQuotaRingWords(t *testing.T) {
	cases := []struct {
		window provider.QuotaWindow
		family string
		span   string
	}{
		{provider.QuotaWindow{Name: "5 hours"}, "", "5h"},
		{provider.QuotaWindow{Name: "7 days"}, "", "7d"},
		{provider.QuotaWindow{Name: "2 weeks"}, "", "2w"},
		{provider.QuotaWindow{Name: "5 小时"}, "", "5 小时"},
		{provider.QuotaWindow{Name: "本周"}, "", "本周"},
		{provider.QuotaWindow{Name: "Gemini 3.8 Flash (High)", Family: "Gemini"}, "Gemini", "Gemini 3.8 Flash (High)"},
		{provider.QuotaWindow{Name: "Gemini · 5 hours", Pool: "Gemini"}, "Gemini", "5h"},
		{provider.QuotaWindow{Name: "Gemini · 7 days", Pool: "Gemini"}, "Gemini", "7d"},
	}
	for _, c := range cases {
		family, span := quotaRingWords(c.window)
		if family != c.family || span != c.span {
			t.Errorf("quotaRingWords(%+v) = %q, %q; want %q, %q", c.window, family, span, c.family, c.span)
		}
	}
}

// TestQuotaDialDraws checks the dial: the share in the middle of the ring,
// and the ring itself drawn by Onevra's ProgressRing.
func TestQuotaDialDraws(t *testing.T) {
	app := NewForTest()
	app.LoadNow()

	tt := ui.NewTester(func(c *ui.Context) {
		quotaDial(c, c.Theme(), 42)
	}, 120, 60)
	tt.Frame()
	if !tt.HasText("42%") {
		t.Errorf("expected the dial to draw the share in its middle; texts were %v", tt.Texts())
	}
}
