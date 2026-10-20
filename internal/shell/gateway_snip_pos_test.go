//go:build nogui && echo

package shell

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestGatewaySnippetCopyButtonPosition asserts that the copy button for
// the example snippet maintains a fixed right-aligned position regardless
// of switching between Shell, curl, Python, or Node snippets.
func TestGatewaySnippetCopyButtonPosition(t *testing.T) {
	app := NewForTest()
	app.LoadNow()
	app.ShowForTest(PageGateway)

	tt := ui.NewTester(app.ViewForTest(), 1180, 780)
	tt.Frame()

	if r, ok := tt.Find(app.text.S.GwConnect); ok {
		if r.X < 232 {
			t.Errorf("gwConnect fold rect X = %v, expected at least 232 (offset from sidebar)", r.X)
		}
	}

	for _, w := range []int{900, 1180, 1400} {
		tt.SetSize(w, 780)
		tt.Frame()
		var firstX float32
		for i, lang := range []snippetLang{langShell, langCurl, langPython, langNode} {
			app.gwLang = int(lang)
			tt.Frame()

			rect, found := tt.Find(app.text.S.GwSnippet)
			if !found {
				t.Fatalf("GwSnippet copy button not found for width %d lang %v", w, lang)
			}
			if i == 0 {
				firstX = rect.X
			} else if rect.X != firstX {
				t.Errorf("width %d lang %v has rect.X = %v, expected %v", w, lang, rect.X, firstX)
			}
		}
	}
}
