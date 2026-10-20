//go:build nogui && echo

package shell

// The share rings: how much of an allowance a window has used, and how
// much of the period's calls a model took. The ring is Onevra's
// ProgressRing — the component library has it, so it is used as it is; the
// figure in the middle, and the words under it, are ours.

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/oneo-me/onevra/source/components"
	"github.com/yetone/magpie/internal/provider"
)

// quotaDial is one window's share: the ring, and the figure in it. A window
// that is used up says so in the figure, as magpie's does in the ring.
func quotaDial(c *ui.Context, t *ui.Theme, used float32) ui.Element {
	color := t.Text
	if used >= 99.5 {
		color = t.Warning
	}
	return shareRing(c, t, used, 32, color)
}

// shareRing is a share as a ring: Onevra's ProgressRing, with the figure
// in the middle of it, as the allowance dials have it.
func shareRing(c *ui.Context, t *ui.Theme, used float32, size float32, color ui.Color) ui.Element {
	if used < 0 {
		used = 0
	}
	if used > 100 {
		used = 100
	}
	dial := ui.Box(c).Size(size, size).Shrink(0)
	dial.Children(func() {
		ring := ui.Box(c).Absolute().Left(0).Top(0).Right(0).Bottom(0)
		ring.Children(func() {
			components.ProgressRing(c, float64(used)/100).Size(size, size)
		})
		figure := ui.Box(c).Absolute().Left(0).Top(0).Right(0).Bottom(0).Center().Center()
		figure.Children(func() {
			// the figure in the hole of the ring: it grows with the ring
			ui.Text(c, fmt.Sprintf("%.0f%%", used)).FontSize(size * 9.5 / 32).FontWeight(600).
				TextColor(color).SingleLine()
		})
	})
	return dial
}

// quotaRingWords are the two lines under a ring: what the window is for
// (its family) and how long it runs — magpie's own words.
func quotaRingWords(w provider.QuotaWindow) (string, string) {
	name := w.Name
	family := w.Family
	pool := w.Pool
	// a pool's row names the pool and the window: "Gemini · 5 hours"
	if i := strings.LastIndex(name, "·"); i >= 0 {
		if pool == "" {
			pool = strings.TrimSpace(name[:i])
		}
		name = strings.TrimSpace(name[i+len("·"):])
	}
	if family == "" {
		family = pool
	}
	span := name
	if m := quotaSpanRe.FindStringSubmatch(name); m != nil {
		n, _ := strconv.Atoi(m[1])
		switch strings.ToLower(m[2]) {
		case "hour":
			span = fmt.Sprintf("%dh", n)
		case "day":
			span = fmt.Sprintf("%dd", n)
		case "week":
			span = fmt.Sprintf("%dw", n)
		}
	}
	return family, span
}
