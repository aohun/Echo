//go:build nogui && echo

package shell

import (
	"image"
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestAgentDriftNoticeIsRed checks the notice a drifted agent carries: the
// words that say it no longer goes through Echo are in magpie's drift red,
// as the row that says it is in magpie, and not the muted grey an ordinary
// subtitle wears.
func TestAgentDriftNoticeIsRed(t *testing.T) {
	app := NewForTest()
	app.LoadNow()
	app.ShowForTest(PageAgents)

	if len(app.data.Agents) == 0 {
		t.Skip("no agent in this environment to drift")
	}

	ag := &app.data.Agents[0]
	ag.Drift = true
	ag.Subtitle = "已不经过 Echo: 配置已变更"

	tt := ui.NewTester(app.ViewForTest(), 1180, 780)
	tt.SetDark(false)
	tt.Frame()

	r, ok := tt.Find(ag.Subtitle)
	if !ok {
		t.Fatalf("the drift notice %q is not drawn", ag.Subtitle)
	}

	want := ui.Hex("#ce2c31")
	if !hasColor(tt.Image(), r, want) {
		t.Errorf("the drift notice is not in magpie's drift red %v", want)
	}
}

// hasColor reports whether the box holds a pixel of that colour: the text
// is antialiased, so the pixels round it differ, but its own are exact.
func hasColor(img *image.RGBA, r ui.Rect, want ui.Color) bool {
	if img == nil {
		return false
	}
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	for y := int(r.Y); y < int(r.Y+r.H) && y < h; y++ {
		for x := int(r.X); x < int(r.X+r.W) && x < w; x++ {
			c := img.RGBAAt(x, y)
			if c.R == want.R && c.G == want.G && c.B == want.B {
				return true
			}
		}
	}
	return false
}
