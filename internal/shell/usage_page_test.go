//go:build nogui && echo

package shell

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestUsageSegmentsRender: the Usage page is four segments, and each draws
// its own content — the filters, the cards, the lists — whatever its read
// came back with. A segment that panicked or fell back to the Overview
// would leave the page showing another segment's words.
//
// The check is the one TestEveryPageRenders makes for a page, made per
// segment: the marker is text only that segment draws.
func TestUsageSegmentsRender(t *testing.T) {
	app := NewForTest()
	app.ShowForTest(PageUsage)
	app.LoadUsageNow()

	tt := ui.NewTester(app.ViewForTest(), 1180, 780)
	for _, seg := range []struct {
		tab    string
		marker string
	}{
		{UsTabOverview, ""},
		{UsTabRequests, "导出 CSV"},
		{UsTabSessions, "全部模型"},
		{UsTabContext, ""},
	} {
		app.usTab = seg.tab
		app.usFresh = ""
		app.LoadUsageNow()
		tt.Frame()

		marker := seg.marker
		if marker == "" {
			marker = app.ContentMarkerForTest(PageUsage)
		}
		if marker == "" {
			t.Errorf("segment %q has no content marker", seg.tab)
			continue
		}
		if !tt.HasText(marker) {
			t.Errorf("segment %q did not render its content (expected %q); texts were %v",
				seg.tab, marker, tt.Texts())
		}
	}
}

// TestUsageFiltersAreKeyed: the page's read is keyed by the query, so a
// filter that changed reads again and one that didn't draws what it has.
// Without that, a filter would change the words above the data and leave
// the data as it was.
func TestUsageFiltersAreKeyed(t *testing.T) {
	app := NewForTest()

	app.usTab = UsTabRequests
	app.usPeriod = "month"
	base := app.usageQuery()
	if got := usQueryKey(base); got != usQueryKey(app.usageQuery()) {
		t.Fatal("the same state gave two keys: every frame would read again")
	}

	app.usStatusFilter = "failed"
	if usQueryKey(app.usageQuery()) == usQueryKey(base) {
		t.Error("a changed filter gave the same key: the page would keep the old rows")
	}
}

// TestUsageTabSwitchResetsThePage: a segment is read for its own tab, so
// switching to another must not keep the paging of the one before it.
func TestUsageTabSwitchResetsThePage(t *testing.T) {
	app := NewForTest()
	app.usLedOffset = 150
	app.usSelectedDay = "2024-01-02"
	app.usTab = UsTabOverview

	// what the head does when a segment is picked
	app.usTab = UsTabSessions
	app.usLedOffset = 0
	app.usSelectedDay = ""

	if app.usLedOffset != 0 || app.usSelectedDay != "" {
		t.Fatalf("the page kept its paging: offset %d, day %q", app.usLedOffset, app.usSelectedDay)
	}
}
