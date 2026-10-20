//go:build nogui && echo

// Command echoprobe is the shell's acceptance check: it renders Echo's window
// without a display and reports what each page showed.
//
// It is a Go test rather than a binary because the shell is native UI (MyGo's
// package ui): there is no page to ask and no JavaScript to evaluate, so the
// check is a render. `go test -tags "nogui,echo" ./internal/shell/` runs it.
//
// It is not under cmd/: magpie's TestPluginsImportOnlyCore allows the root
// main.go alone to gather the plugins, and a second binary would either break
// that rule or run without the plugins the app actually ships with. The test
// binary already gathers them through the package it links.
package shell

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
)

// pageResult is what one page rendered.
type pageResult struct {
	Page    string   `json:"page"`
	Title   string   `json:"title"`
	Present bool     `json:"present"`
	Marker  string   `json:"marker"`
	Texts   []string `json:"texts"`
}

// report is the probe's outcome.
type report struct {
	Pages  []pageResult `json:"pages"`
	Nav    []string     `json:"nav"`
	Width  int          `json:"width"`
	Height int          `json:"height"`
	OK     bool         `json:"ok"`
	Note   string       `json:"note,omitempty"`
}

// TestProbe renders every page and writes the report the probe is kept for.
//
// The report is written where a release check can read it (ECHO_PROBE_OUT, or
// $TMPDIR/echo-probe.json) and is the same JSON the shell's earlier probe
// binary produced, so an existing check keeps working.
//
// It is a test, so it also fails: a page that renders its chrome but not its
// content fails here, which is the assertion that matters. The file is the
// evidence for a reviewer who was not watching.
func TestProbe(t *testing.T) {
	rep := probeReport(t, 1180, 780)

	out := os.Getenv("ECHO_PROBE_OUT")
	if out == "" {
		out = filepath.Join(os.TempDir(), "echo-probe.json")
	}
	b, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		t.Fatalf("marshalling the report: %v", err)
	}
	if err := os.WriteFile(out, append(b, '\n'), 0o644); err != nil {
		t.Fatalf("writing %s: %v", out, err)
	}
	t.Logf("probe report: %s", out)

	if !rep.OK {
		t.Errorf("the shell did not render: %s", rep.Note)
	}
}

// probeReport renders every page of the shell and reports what it showed.
func probeReport(t *testing.T, width, height int) report {
	t.Helper()
	app := NewForTest()
	// Read the data the pages draw before rendering, as the running app does
	// when its read lands: a page rendered before the read shows only its
	// empty state, and the probe must exercise the real one.
	app.LoadNow()

	rep := report{Width: width, Height: height}
	// One tester for the whole walk, so the pages share one app's state and
	// switching pages exercises the real navigation rather than a fresh app
	// per page.
	tt := ui.NewTester(app.ViewForTest(), width, height)

	for _, p := range AllPages() {
		app.ShowForTest(p)
		tt.Frame()

		title := app.PageTitleForTest(p)
		marker := app.ContentMarkerForTest(p)
		rep.Pages = append(rep.Pages, pageResult{
			Page:    string(p),
			Title:   title,
			Present: title != "" && tt.HasText(title) && marker != "" && tt.HasText(marker),
			Marker:  marker,
			Texts:   tt.Texts(),
		})
	}

	// The Settings page is not in the navigation — the gear opens it — but
	// it is a page like the rest, and every tab of it must render its own
	// content, the tab strip's name being what says it drew.
	for _, tab := range setTabs {
		app.ShowForTest(PageSettings)
		app.setTab = tab
		app.LoadNow()
		tt.Frame()

		title := app.setTabLabel(tab)
		rep.Pages = append(rep.Pages, pageResult{
			Page:    string(PageSettings) + "/" + string(tab),
			Title:   title,
			Present: tt.HasText(title) && tt.HasText(app.ContentMarkerForTest(PageSettings)),
			Marker:  app.ContentMarkerForTest(PageSettings),
			Texts:   tt.Texts(),
		})
	}

	// The navigation must offer every page, or a page is unreachable.
	for _, info := range app.Pages() {
		if !tt.HasText(info.Title) {
			rep.Note = "the navigation does not offer every page"
		}
		rep.Nav = append(rep.Nav, string(info.Page))
	}

	rep.OK = true
	// every navigable page must have rendered, and everything rendered
	// must have shown its content
	for _, info := range app.Pages() {
		found := false
		for _, pr := range rep.Pages {
			found = found || (pr.Page == string(info.Page) && pr.Present)
		}
		if !found {
			rep.OK = false
		}
	}
	for _, pr := range rep.Pages {
		if !pr.Present {
			rep.OK = false
		}
	}
	if !rep.OK && rep.Note == "" {
		rep.Note = "a page rendered without its content; see the per-page texts"
	}
	return rep
}
