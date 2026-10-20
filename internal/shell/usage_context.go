//go:build nogui && echo

package shell

// The Usage page's Context segment: what each agent's prompts hold, as the
// gateway read them — how full its window runs, what every request starts
// with, what the parts of a prompt are, and the score and tags those work
// out to. The sessions module magpie also draws here is kept off Echo's
// page: the Sessions tab beside it already says all of it.
//
// The reading is gui.ContextOf, magpie's own analysis of the routing
// history, so this segment and magpie's own Context tab can't say
// different things about the same requests. This file is the drawing.

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/yetone/magpie/internal/gui"
)

// usCtxParts are the kinds a prompt is told apart by, in magpie's order.
var usCtxParts = []struct {
	id   string
	name string
	c    ui.Color
}{
	{"system", "系统提示", ui.Hex("#007aff")},
	{"tools", "工具定义", ui.Hex("#af52de")},
	{"memory", "记忆与指令", ui.Hex("#ff9500")},
	{"files", "读过的文件", ui.Hex("#34c759")},
	{"results", "工具返回", ui.Hex("#5ac8fa")},
	{"chat", "对话", ui.Hex("#8e8e93")},
}

// usCtxPartName is a part's name in the reader's language.
func usCtxPartName(kind string) string {
	for _, p := range usCtxParts {
		if p.id == kind {
			return p.name
		}
	}
	return kind
}

// usCtxPartColor is a part's colour.
func usCtxPartColor(kind string) ui.Color {
	for _, p := range usCtxParts {
		if p.id == kind {
			return p.c
		}
	}
	return ui.Hex("#8e8e93")
}

// usCtxScoreNames are what makes a score, in magpie's own words.
var usCtxScoreNames = map[string]string{
	"cache":    "缓存",
	"lean":     "轻起步",
	"pace":     "增长节奏",
	"reliable": "稳定性",
}

// usCtxScoreNotes say what each of those measures.
var usCtxScoreNotes = map[string]string{
	"cache":    "提示词里有多少是供应商从缓存读到的：90% 满分",
	"lean":     "每次请求固定携带的系统提示、工具与记忆，占窗口的比例：5% 以内满分",
	"pace":     "一次请求比上一次多带多少：窗口的 1% 以内满分",
	"reliable": "没有失败的请求比例：两成失败即为零分",
}

// usCtxTagNames are the tags an agent's prompts earn.
var usCtxTagNames = map[string]string{
	"cache-friendly": "缓存友好",
	"cache-misses":   "缓存缺失",
	"lean":           "起步轻",
	"heavy-start":    "起步重",
	"mcp-heavy":      "MCP 偏重",
	"tool-heavy":     "工具返回偏多",
	"file-reader":    "读文件多",
	"memory-heavy":   "记忆偏重",
	"fills-fast":     "填得快",
	"near-full":      "接近占满",
	"compacts":       "会压缩",
	"steady":         "稳定",
	"flaky":          "不稳",
}

// usCtxTagNote says what a tag was earned for.
func usCtxTagNote(tag gui.ContextTag) string {
	switch tag.Key {
	case "cache-friendly", "cache-misses":
		return fmt.Sprintf("提示词里 %.0f%% 从缓存读到", tag.Value*100)
	case "lean", "heavy-start", "mcp-heavy":
		return fmt.Sprintf("每次请求固定携带 %s tokens", FormatCount(int(tag.Value)))
	case "tool-heavy", "file-reader", "memory-heavy":
		return fmt.Sprintf("占提示词的 %.0f%%", tag.Value*100)
	case "fills-fast":
		return fmt.Sprintf("一次请求比上一次多 %s tokens", FormatCount(int(tag.Value)))
	case "near-full":
		return fmt.Sprintf("有会话达到了窗口的 %.0f%%", tag.Value*100)
	case "compacts":
		return fmt.Sprintf("上下文被压缩了 %d 次", int(tag.Value))
	case "flaky":
		return fmt.Sprintf("%.1f%% 的请求失败", tag.Value*100)
	case "steady":
		return "不到 1% 的请求失败"
	}
	return ""
}

func (a *App) usageContext(c *ui.Context, t *ui.Theme) {
	d := a.data.Usage
	if d.Query.Tab != UsTabContext {
		a.usSkeleton(c, t)
		return
	}
	data := d.Context
	if len(data.Agents) == 0 {
		a.usEmpty(c, t, "这段时间没有可分析的提示词。",
			"经过网关的请求会记下它携带了什么：系统提示、工具、记忆、读过的文件与对话，各占多少。")
		a.usNote(c, t, "评分与标签都由这段时间的请求算出。")
		return
	}

	a.usSectionHead(c, t, "ctx-agents", "Agents", fmt.Sprintf("%d 个", len(data.Agents)), false)
	for i := 0; i < len(data.Agents); i += 2 {
		row := data.Agents[i:min(i+2, len(data.Agents))]
		ui.Row(c).AlignItems(ui.Stretch).Gap(10).Padding(2, 20, 6, 20).FillWidth().Children(func() {
			for _, ag := range row {
				a.usCtxAgentCard(c, t, ag)
			}
			if len(row) == 1 {
				ui.Column(c).Grow(1).Basis(0)
			}
		})
	}

	// Sessions module is hidden on the Context page
	a.usNote(c, t, fmt.Sprintf("最近 %d 天里，网关读到的提示词构成；评分与标签由这些请求算出。", data.Days))
}

// usCtxGrade words a score, as magpie's ring does.
func usCtxGrade(score int) (string, ui.Color) {
	switch {
	case score >= 85:
		return "出色", ui.Hex("#34c759")
	case score >= 70:
		return "良好", ui.Hex("#30a46c")
	case score >= 50:
		return "一般", ui.Hex("#ff9500")
	}
	return "需要留意", ui.Hex("#ff3b30")
}

// usCtxAgentCard is one agent's card: its score and what makes it, its
// tags, the facts of its prompts, and what they hold.
func (a *App) usCtxAgentCard(c *ui.Context, t *ui.Theme, ag gui.ContextAgent) {
	word, tone := usCtxGrade(ag.Score)
	card := ui.Column(c).Grow(1).Basis(0).Padding(11, 13).Radius(10).
		Background(a.fieldBg(t)).Border(1, t.Border).Gap(9)
	card.Children(func() {
		// who, and the score
		ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
			ui.Image(c, iconPictureFor(usAgentIcon(ag.Agent))).Size(22, 22).Radius(5)
			ui.Column(c).Grow(1).Shrink(1).Gap(1).Children(func() {
				ui.Text(c, usAgentName(ag.Agent)).FontSize(12.5).FontWeight(650).TextColor(t.Text).SingleLine()
				ui.Text(c, fmt.Sprintf("%d 次请求 · %d 个会话", ag.Requests, ag.Sessions)).
					FontSize(10).TextColor(t.TextMuted).SingleLine()
			})
			ui.Column(c).AlignItems(ui.End).Gap(1).Shrink(0).Children(func() {
				ui.Text(c, fmt.Sprint(ag.Score)).FontSize(21).FontWeight(700).TextColor(tone).SingleLine()
				ui.Text(c, word).FontSize(9.5).FontWeight(600).TextColor(tone).SingleLine()
			})
		})

		// what makes the score, a bar each
		if len(ag.Scores) > 0 {
			ui.Column(c).FillWidth().Gap(3).Children(func() {
				for _, s := range ag.Scores {
					s := s
					share := 0.0
					if s.Most > 0 {
						share = s.Points / s.Most
					}
					row := ui.Row(c).AlignItems(ui.Center).Gap(6).FillWidth()
					row.Tooltip(usCtxScoreNotes[s.Key])
					row.Children(func() {
						ui.Text(c, usCtxScoreNames[s.Key]).FontSize(10).TextColor(t.TextMuted).Width(56).Shrink(0).SingleLine()
						a.usShareBar(c, t, share, usCtxScoreTone(share), 90)
						ui.Text(c, fmt.Sprintf("%.0f/%.0f", s.Points, s.Most)).FontSize(9.5).
							TextColor(t.TextMuted).Shrink(1).SingleLine()
					})
				}
			})
		}

		// the tags
		if len(ag.Tags) > 0 {
			ui.Row(c).AlignItems(ui.Center).Gap(4).Children(func() {
				for _, tg := range ag.Tags {
					tg := tg
					fg, bg := usCtxTagTone(tg.Tone)
					chip := ui.Row(c).AlignItems(ui.Center).Padding(1.5, 6).Radius(4).Background(bg).Shrink(0)
					chip.Tooltip(usCtxTagNote(tg))
					chip.Children(func() {
						ui.Text(c, usCtxTagName(tg.Key)).FontSize(9.5).FontWeight(600).TextColor(fg).SingleLine()
					})
				}
			})
		}

		// the facts of its prompts, two lines of three
		facts := []struct{ k, v, note string }{
			{"典型提示词", usCtxK(ag.Median), "中位请求的提示词"},
			{"最大", usCtxK(ag.P90), "九成请求的提示词不超过这个大小"},
			{"起步", usCtxK(ag.Baseline), "系统提示、工具与记忆：每次请求都带的部分"},
			{"每次增长", usCtxGrowth(ag.Growth), "一次请求比上一次多带多少，中位"},
			{"缓存命中", usCtxPct(ag.Cache), "提示词里供应商从缓存读到的比例"},
			{"窗口", usCtxK(ag.Window), "它用过的模型里最大的上下文窗口"},
		}
		for i := 0; i < len(facts); i += 3 {
			line := facts[i:min(i+3, len(facts))]
			ui.Row(c).FillWidth().Gap(6).Children(func() {
				for _, f := range line {
					usCtxFact(c, t, f.k, f.v, f.note)
				}
			})
		}

		// what its prompts hold
		ui.Column(c).FillWidth().Gap(4).Children(func() {
			ui.Text(c, "提示词的构成").FontSize(10).TextColor(t.TextMuted).SingleLine()
			a.usCtxShareBar(c, t, ag.Shares)
			ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
				for _, p := range usCtxParts {
					v := ag.Shares[p.id]
					if v < 0.01 {
						continue
					}
					ui.Row(c).AlignItems(ui.Center).Gap(3).Children(func() {
						ui.Row(c).Width(6).Height(6).Radius(3).Background(p.c)
						ui.Text(c, fmt.Sprintf("%s %.0f%%", p.name, v*100)).FontSize(9).
							TextColor(t.TextMuted).SingleLine()
					})
				}
			})
		})

		// the models it used
		if len(ag.Models) > 0 {
			ui.Row(c).AlignItems(ui.Center).Gap(4).Children(func() {
				for _, m := range ag.Models {
					m := m
					short := m
					if i := strings.LastIndex(m, "/"); i >= 0 {
						short = m[i+1:]
					}
					ui.Text(c, short).FontSize(9).TextColor(t.TextMuted).Shrink(0).SingleLine().Tooltip(m)
				}
				ui.Spacer(c)
				if !ag.LatestTime.IsZero() {
					ui.Text(c, "最近 "+usAgo(ag.LatestTime)).FontSize(9).TextColor(t.TextMuted).SingleLine()
				}
			})
		}
	})
}

// usCtxFact is one fact of a card: its name over its figure.
func usCtxFact(c *ui.Context, t *ui.Theme, k, v, note string) {
	box := ui.Column(c).Grow(1).Basis(0).Gap(0).Shrink(1)
	box.Tooltip(note)
	box.Children(func() {
		ui.Text(c, k).FontSize(9).TextColor(t.TextMuted).SingleLine()
		ui.Text(c, v).FontSize(11.5).FontWeight(600).TextColor(t.Text).SingleLine()
	})
}

// usCtxScoreTone colours a score's bar by how much of it was earned.
func usCtxScoreTone(share float64) ui.Color {
	switch {
	case share >= 0.8:
		return ui.Hex("#34c759")
	case share >= 0.5:
		return ui.Hex("#ff9500")
	}
	return ui.Hex("#ff3b30")
}

// usCtxTagTone is a tag's colour by its tone.
func usCtxTagTone(tone string) (ui.Color, ui.Color) {
	switch tone {
	case "good":
		return ui.Hex("#1f7a3a"), ui.RGBA(52, 199, 89, 0.12)
	case "warn":
		return ui.Hex("#b45309"), ui.RGBA(255, 149, 0, 0.13)
	}
	return ui.Hex("#007aff"), ui.RGBA(0, 122, 255, 0.10)
}

// usCtxK writes a count of tokens the way the cards do: 12.5K, 1.2M.
func usCtxK(n int) string {
	if n <= 0 {
		return "—"
	}
	if n >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	}
	if n >= 1_000 {
		return fmt.Sprintf("%.1fK", float64(n)/1_000)
	}
	return fmt.Sprint(n)
}

// usCtxGrowth writes how much a request adds to the one before it.
func usCtxGrowth(n int) string {
	if n <= 0 {
		return "—"
	}
	return "+" + usCtxK(n)
}

// usCtxPct writes a share.
func usCtxPct(v float64) string {
	if v <= 0 {
		return "—"
	}
	return fmt.Sprintf("%.0f%%", v*100)
}

// usCtxShareBar is a prompt's parts as one bar, each in its colour.
func (a *App) usCtxShareBar(c *ui.Context, t *ui.Theme, shares map[string]float64) {
	bar := ui.Row(c).FillWidth().Height(7).Radius(4).Background(a.usTrackBg(t))
	bar.Children(func() {
		for _, p := range usCtxParts {
			v := shares[p.id]
			if v <= 0.002 {
				continue
			}
			seg := ui.Row(c).WidthPercent(float32(v) * 100).Height(7).Background(p.c)
			seg.Tooltip(fmt.Sprintf("%s · %.0f%%", p.name, v*100))
		}
	})
}

// usCtxTagName is a tag in the reader's language.
func usCtxTagName(key string) string {
	if n, ok := usCtxTagNames[key]; ok {
		return n
	}
	return key
}
