//go:build nogui && echo

package shell

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/yetone/magpie/internal/provider"
	"github.com/yetone/magpie/internal/settings"
	"github.com/yetone/magpie/internal/usage"
	"github.com/yetone/magpie/internal/usageapi"
)

// The provider editor: the dialog a sign-in succeeds to, and 编辑 on a
// provider's row. It is magpie's own provider editor — the accounts in
// use, the models the agents are offered, the windows and limits the
// requests keep — drawn in native UI, its rows the same grid the group
// modal's are.

// providerEdit is the editor's draft: what its fields hold while it is
// open, read from the provider when it opens and written back on Save.
type providerEdit struct {
	ID      string
	Name    string
	IsSub   bool // a subscription the reader signs in to
	Keyed   bool // a vendor used with a key
	Key     string
	Preset  string
	Moved   bool // served by its plugin now: said, not editable here
	Err     string
	Removed bool

	// the models the agents are offered, and the provider's list to pick
	// from; Chosen is by model id. WasAll is a provider that names no
	// list: every model through, the ones to come later too. Info is a
	// plugin model's rate and window, said on its pill. View is which of
	// the list's three looks is showing; UsageModels is the models with
	// calls recorded, read when the usage look is first picked.
	Models      []string
	Chosen      map[string]bool
	Info        map[string]pvModelInfo
	WasAll      bool
	View        int // 0 the vendor's list, 1 the picked, 2 the ones with usage
	UsageModels []string
	UsageRead   bool
	Refreshing  bool            // 重新获取模型 in flight
	Testing     bool            // 测试模型 in flight
	TestRes     map[string]uint8 // 0 untested, 1 answered, 2 didn't
	Note        string          // what Refresh and Test said, said under them
	NewID       string          // the 添加模型 field's words
	Unlisted    bool

	// snap is the draft as it opened: a close beside it is a change the
	// reader is asked to throw away
	snap string

	// the three token maps as their inputs hold them: "128k" for every
	// model, "gpt-6=1m" for one, comma separated
	Contexts string
	Outputs  string
	Compacts string

	Fallback string // one provider/model a line

	ProxyMode string // "", "direct" or "custom"
	ProxyURL  string

	Concurrency string // "" = 不限制
	RPM         string
	QueueCap    string // "" = as saved
	QueueWait   string // "" = as saved
	PriceRate   string // "" = as saved
}

// openProviderEditor loads a provider into the editor and opens it over
// the page.
func (a *App) openProviderEditor(id string) {
	e := a.providerEditLoad(id)
	if e == nil {
		return
	}
	a.pvEdit = e
	a.invalidate()
}

// closeProviderEditor puts the editor away.
func (a *App) closeProviderEditor() {
	a.pvEdit = nil
	a.pvDiscard = false
	a.invalidate()
}

// providerEditLoad reads the provider and what is saved of it into a
// draft; nil when there is no such provider.
func (a *App) providerEditLoad(id string) *providerEdit {
	p, err := provider.Find(id)
	if err != nil {
		return nil
	}
	e := &providerEdit{
		ID:       p.ID,
		Name:     p.Name,
		IsSub:    p.Account != nil,
		Keyed:    p.Key != "",
		Key:      p.Key,
		Preset:   p.Preset,
		Moved:    provider.Moved(p.ID),
		Models:   append([]string{}, p.Models...),
		Chosen:   map[string]bool{},
		Info:     map[string]pvModelInfo{},
		Unlisted: p.Unlisted,
	}
	// a plugin's provider's list is the plugin's own: the models it
	// serves, with the names they are shown by, their rates and windows,
	// whether it names a subset now or lets every one of them through.
	// The plugin's list is the truth here — the catalog's prefixed ids
	// for the same models are not appended beside it.
	pluginList := false
	if pp, ok := pluginProviderFor(p.ID); ok && len(pp.Models) > 0 {
		pluginList = true
		e.Models = e.Models[:0]
		for _, m := range pp.Models {
			e.Models = append(e.Models, m.ID)
			e.Info[m.ID] = pvModelInfo{Name: m.Name, Rate: m.Rate, Free: m.Free, Context: m.Context}
		}
	}
	e.WasAll = len(p.Models) == 0
	for _, m := range p.Models {
		e.Chosen[m] = true
	}
	if e.WasAll {
		for _, m := range e.Models {
			e.Chosen[m] = true
		}
	}
	// the provider's own catalog list, for models it serves but doesn't
	// list: picked beside the ones it does — a plugin's provider keeps
	// its plugin's list alone
	if !pluginList {
		for _, am := range a.data.Routing.AvailableModels {
			base, _ := provider.MemberEffort(am.ID)
			if prov, _, ok := strings.Cut(am.ID, "/"); ok && prov == p.ID && !slices.Contains(e.Models, base) {
				e.Models = append(e.Models, base)
			}
		}
	}
	e.Contexts = pvTokensText(p.Contexts)
	e.Outputs = pvTokensText(provider.OutputsOf(p.ID))
	e.Compacts = pvTokensText(provider.CompactsOf(p.ID))
	e.Fallback = strings.Join(p.Fallback, "\n")
	switch p.Proxy {
	case "":
		e.ProxyMode = ""
	case "direct":
		e.ProxyMode = "direct"
	default:
		e.ProxyMode, e.ProxyURL = "custom", p.Proxy
	}
	if p.MaxConcurrency != nil {
		e.Concurrency = strconv.Itoa(*p.MaxConcurrency)
	}
	e.RPM = pvIntText(p.MaxRPM)
	e.QueueCap = pvIntText(p.QueueLimit)
	e.QueueWait = pvIntText(p.QueueWait)
	e.PriceRate = pvRateText(p.PriceRate)
	e.snap = e.pvSnap()
	return e
}

// pvIntText is a count as its input holds it: "" for the unset zero.
func pvIntText(n int) string {
	if n == 0 {
		return ""
	}
	return strconv.Itoa(n)
}

// pvRateText is a price rate as its input holds it: "" for the official
// price's 1 and the unset 0 both.
func pvRateText(r float64) string {
	if r == 0 || r == 1 {
		return ""
	}
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.3f", r), "0"), ".")
}

// pvTokensText is a token map as the editor's input holds it: "*" first
// as a bare value, the rest as model=value, comma separated.
func pvTokensText(m map[string]int) string {
	if len(m) == 0 {
		return ""
	}
	parts := []string{}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	for _, k := range keys {
		v := rtCtxShort(m[k])
		if k == "*" {
			parts = append(parts, v)
		} else {
			parts = append(parts, k+"="+v)
		}
	}
	return strings.Join(parts, ", ")
}

// pvParseTokensList reads an input's text as a token map: "128k" sets
// every model's ("*"), "gpt-6=1m" one model's, comma separated. The
// second return is the part that would not read, "" when all did.
func pvParseTokensList(s string) (map[string]int, string) {
	out := map[string]int{}
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		key, val := "*", part
		if k, v, ok := strings.Cut(part, "="); ok {
			key, val = strings.TrimSpace(k), strings.TrimSpace(v)
		}
		n, ok := rtParseTokens(val)
		if !ok {
			return nil, part
		}
		out[key] = n
	}
	return out, ""
}

// pvAtoi reads a whole number as its input holds it: "" reads as the
// unset false.
func pvAtoi(s string) (int, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, false
	}
	return n, true
}

// pvSnap is the draft as one string, what dirty compares.
func (e *providerEdit) pvSnap() string {
	chosen := make([]string, 0, len(e.Chosen))
	for _, m := range e.Models {
		if e.Chosen[m] {
			chosen = append(chosen, m)
		}
	}
	unlisted := "0"
	if e.Unlisted {
		unlisted = "1"
	}
	return strings.Join([]string{e.Key, unlisted, e.Contexts, e.Outputs, e.Compacts,
		e.Fallback, e.ProxyMode, e.ProxyURL, e.Concurrency, e.RPM, e.QueueCap, e.QueueWait, e.PriceRate,
		strings.Join(chosen, "\x00"), strings.Join(e.Models, "\x00")}, "\x01")
}

// dirty is whether the draft has moved since it opened.
func (e *providerEdit) dirty() bool { return e.snap != e.pvSnap() }

// pvEditCloseAttempt is every way into the editor tried to close it: one
// with changes still in it is asked first (the discard ask), one without
// closes.
func (a *App) pvEditCloseAttempt() {
	if a.pvEdit != nil && a.pvEdit.dirty() {
		a.pvDiscard = true
		a.invalidate()
		return
	}
	a.closeProviderEditor()
}

// pvDiscardView is the discard ask over the editor: the changes unsaved
// go when 放弃 says so, stay when 取 消 does.
func (a *App) pvDiscardView(c *ui.Context) {
	t := c.Theme()
	ui.Overlay(c, func() {
		back := ui.Box(c).Absolute().Left(0).Top(0).Right(0).Bottom(0).Center().
			Background(ui.RGBA(0, 0, 0, 0.4)).Modal()
		if back.OverlayShortcut(0, ui.KeyEscape) {
			a.pvDiscard = false
			a.invalidate()
			return
		}
		back.Children(func() {
			card := ui.Column(c).Width(420).Padding(20, 24).Radius(12).Gap(14).
				Role(ui.RoleDialog).Label("放弃更改")
			card.Background(t.Background)
			card.Shadow(0, 10, 30, 0, ui.RGBA(0, 0, 0, 0.3))
			card.Children(func() {
				ui.Text(c, "放弃更改").FontSize(15).FontWeight(600).TextColor(t.Text)
				ui.Text(c, "是否放弃未保存的修改？放弃后，这些修改将丢失。").FontSize(12.5).TextColor(t.Text).LineHeight(1.5)
				ui.Row(c).AlignItems(ui.Center).Gap(10).Children(func() {
					ui.Spacer(c)
					if b := ui.Button(c, "取消"); b.Clicked() {
						a.pvDiscard = false
						a.invalidate()
					}
					drop := ui.Row(c).AlignItems(ui.Center).Padding(6, 16).Radius(6).Background(ui.Hex("#d93025")).
						Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).Label("放弃")
					clicked := drop.Clicked()
					drop.Children(func() {
						ui.Text(c, "放弃").FontSize(12).FontWeight(600).TextColor(ui.Hex("#ffffff")).SingleLine()
					})
					if clicked {
						a.pvDiscard = false
						a.closeProviderEditor()
					}
				})
			})
		})
	})
}

// pvEditView draws the editor over the page.
func (a *App) pvEditView(c *ui.Context) {
	e := a.pvEdit
	if e == nil {
		return
	}
	t := c.Theme()
	title := "编辑供应商 · " + e.Name

	a.skSheetFrameClose(c, 760, 700, title, a.pvEditCloseAttempt, func() {
		// Modal Header
		ui.Row(c).AlignItems(ui.Center).Gap(10).Padding(16, 20, 12, 20).FillWidth().Children(func() {
			ui.Text(c, title).FontSize(15).FontWeight(600).TextColor(t.Text).Grow(1).Shrink(1).SingleLine()
			closeBtn := ui.Row(c).Padding(4, 8).Radius(6).Background(ui.RGBA(0, 0, 0, 0.05)).
				Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).Label("关闭")
			if closeBtn.Clicked() {
				a.pvEditCloseAttempt()
			}
			closeBtn.Children(func() {
				ui.Text(c, "✕").FontSize(12).TextColor(t.TextMuted).SingleLine()
			})
		})

		a.skSheetRule(c, t)

		ui.Scroll(c).Grow(1).Shrink(1).FillWidth().Padding(16, 20, 16, 20).Children(func() {
			ui.Column(c).FillWidth().Gap(14).Children(func() {
				a.pvEditAccounts(c, t, e)
				if e.Moved {
					rtFieldRowFlat(c, t, "运行方式", func() {
						rtHint(c, t, "社区插件沿用在 Echo 里的账号；内建的登录已下线，请求一直由插件发出。")
					})
				}
				a.pvEditModels(c, t, e)
				a.pvEditTokens(c, t, e)
				rtFieldRow(c, t, "备注", func() {
					ui.TextArea(c, &e.Fallback).Height(64).Placeholder("备用模型，一行一个 provider/model").Label("备注").
						Background(t.Background).FillWidth()
					rtHint(c, t, "这个供应商答不了（额度用尽、被限流或故障）时，请求按这里的顺序改发给这些模型；仅在回复开始前切换。")
				})
				a.pvEditProxy(c, t, e)
				a.pvEditLimits(c, t, e)
			})
		})

		a.skSheetRule(c, t)

		// Modal Footer
		ui.Row(c).AlignItems(ui.Center).Gap(10).Padding(12, 20, 16, 20).FillWidth().Children(func() {
			delBtn := ui.Row(c).AlignItems(ui.Center).Padding(6, 14).Radius(6).
				Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).Label("移除")
			if delBtn.Clicked() {
				if err := provider.Delete(e.ID); err == nil {
					e.Removed = true
					a.closeProviderEditor()
					a.Reload()
				} else {
					e.Err = err.Error()
					c.Invalidate()
				}
			}
			delBtn.Children(func() {
				ui.Text(c, "移除").FontSize(12).TextColor(ui.Hex("#ff3b30")).SingleLine()
			})

			if e.Err != "" {
				ui.Text(c, e.Err).FontSize(11.5).TextColor(ui.Hex("#ff3b30")).Grow(1).Shrink(1).LineHeight(1.35).
					TextAlign(ui.End)
			} else {
				ui.Spacer(c)
			}

			cancelBtn := ui.Row(c).Padding(6, 14).Radius(6).Background(t.Background).Border(1, t.Border).
				Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).Label("取消")
			if cancelBtn.Clicked() {
				a.pvEditCloseAttempt()
			}
			cancelBtn.Children(func() {
				ui.Text(c, "取消").FontSize(12).TextColor(t.Text).SingleLine()
			})

			saveBtn := ui.Row(c).Padding(6, 16).Radius(6).Background(t.Accent).
				Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).Label("保存")
			if saveBtn.Clicked() {
				a.pvEditSave(c, e)
			}
			saveBtn.Children(func() {
				ui.Text(c, "保存").FontSize(12).FontWeight(600).TextColor(ui.Hex("#ffffff")).SingleLine()
			})
		})
	})
}

// pvEditSmallBtn is the editor's foot button: a small white pill, dimmed
// while its work is in flight.
func (a *App) pvEditSmallBtn(c *ui.Context, t *ui.Theme, label string, enabled bool, pick func()) {
	btn := ui.Row(c).AlignItems(ui.Center).AlignSelf(ui.End).Padding(5, 12).Radius(6).
		Background(t.Background).Border(1, t.Border).
		Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).Label(label)
	clicked := enabled && btn.Clicked()
	tc := t.Text
	if !enabled {
		tc = t.TextMuted
	}
	btn.Children(func() {
		ui.Text(c, label).FontSize(11.5).FontWeight(500).TextColor(tc).SingleLine()
	})
	if clicked {
		pick()
	}
}

// pvEditAccounts is the accounts the provider signs in with: the one in
// use said, each removable, and another addable — magpie's accounts list.
func (a *App) pvEditAccounts(c *ui.Context, t *ui.Theme, e *providerEdit) {
	if !e.IsSub {
		if e.Keyed {
			rtFieldRow(c, t, "API Key", func() {
				ui.TextInput(c, &e.Key).Label("API Key").Background(t.Background).FillWidth()
			})
		}
		return
	}
	rtFieldRow(c, t, "账号", func() {
		ls := provider.Logins(e.ID)
		for _, l := range ls {
			l := l
			row := ui.Row(c).AlignItems(ui.Center).Gap(8).Padding(6, 10).FillWidth().
				Radius(7).Background(t.Background).Border(1, t.Border)
			row.Children(func() {
				ui.Text(c, "✓").FontSize(11).TextColor(ui.Hex("#34c759")).SingleLine()
				ui.Text(c, l.User).FontSize(12).Font("monospace").TextColor(t.Text).Grow(1).Shrink(1).SingleLine()
				if plan := accountPlan(e.ID, l.Plan); plan != "" {
					ui.Text(c, plan).FontSize(10).TextColor(t.TextMuted).
						Background(ui.RGBA(0, 0, 0, 0.05)).Padding(1, 6).Radius(4).SingleLine()
				}
				switch {
				case l.Lapsed != "":
					ui.Text(c, "已失效").FontSize(10.5).TextColor(ui.Hex("#ff9500")).SingleLine()
				case l.On:
					ui.Text(c, "使用中").FontSize(10.5).TextColor(t.Accent).SingleLine()
				}
				rm := ui.Text(c, "移除").FontSize(11).Cursor(ui.CursorPointer).TextColor(ui.Hex("#ff3b30"))
				if rm.Clicked() {
					if err := provider.ForgetAccount(l.User); err == nil {
						a.Reload()
					} else {
						e.Err = err.Error()
					}
					c.Invalidate()
				}
			})
		}
		addBtn := ui.Row(c).AlignItems(ui.Center).Gap(5).Padding(5, 10).Radius(6).
			Background(t.Background).Border(1, t.Border).
			Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).Label("添加另一个账号")
		if addBtn.Clicked() {
			a.closeProviderEditor()
			a.subSignIn(e.ID)
		}
		addBtn.Children(func() {
			ui.Text(c, "⊕ 添加另一个 "+e.Name+" 账号").FontSize(11.5).FontWeight(500).TextColor(t.Text).SingleLine()
		})
		rtHint(c, t, "网关优先用一个账号；可选多个时，前一个额度用完自动换下一个。")
	})
}

// pvModelInfo is what a model's pill says of it: the name it is shown
// by (a plugin model's, where the id is a machine's), its price rate
// against the official price (0 none), whether its plan serves it for
// nothing, and the window it takes.
type pvModelInfo struct {
	Name    string
	Rate    float64
	Free    bool
	Context int
}

// pvModelLabel is the words a model's pill carries: its name where one
// is known, its id where not.
func pvModelLabel(e *providerEdit, m string) string {
	if info, ok := e.Info[m]; ok && info.Name != "" {
		return info.Name
	}
	return m
}

// pvEditModels is the models the agents are offered: the provider's list
// under magpie's three looks over it (the vendor's list, the picked, the
// ones with calls recorded), its bulk picks, its pill styles — the picked
// one the accent's soft tint with a check, the rest the white pill — its
// Refresh and Test buttons, and 只通过路由分组使用.
func (a *App) pvEditModels(c *ui.Context, t *ui.Theme, e *providerEdit) {
	rtFieldRowFlat(c, t, fmt.Sprintf("模型 (%d)", len(e.Models)), func() {
		// the three looks, magpie's segmented over the list
		sel := e.View
		seg := ui.Segmented(c, &sel, "供应商列表", "仅所选", "仅看用量").AlignSelf(ui.Start)
		if seg.Changed() {
			e.View = sel
			if sel == 2 && !e.UsageRead {
				// the models with calls recorded are a read of the
				// usage log, taken once for the editor's lifetime
				e.UsageRead = true
				id := e.ID
				go func() {
					with := []string{}
					for _, m := range usageapi.State(usage.All).Models {
						if m.Provider == id {
							with = append(with, m.Model)
						}
					}
					e.UsageModels = with
					c.Invalidate()
				}()
			}
			c.Invalidate()
		}

		// what the look shows of the list
		shown := e.Models
		switch e.View {
		case 1:
			shown = nil
			for _, m := range e.Models {
				if e.Chosen[m] {
					shown = append(shown, m)
				}
			}
		case 2:
			shown = nil
			seen := map[string]bool{}
			for _, m := range e.UsageModels {
				if !seen[m] && slices.Contains(e.Models, m) {
					shown = append(shown, m)
					seen[m] = true
				}
			}
		}
		if len(shown) == 0 {
			if e.View == 2 && e.UsageRead && len(e.UsageModels) == 0 {
				rtHint(c, t, "这个供应商还没有被调用过的模型。")
			} else if e.View == 1 {
				rtHint(c, t, "还没有选择任何模型。")
			} else {
				rtHint(c, t, "供应商列表为空。刷新，或输入一个模型 ID。")
			}
		}

		// the bulk picks, over what the look shows
		if len(shown) > 0 {
			ui.Row(c).Gap(6).AlignSelf(ui.Start).Children(func() {
				allBtn := ui.Row(c).AlignItems(ui.Center).Padding(4, 12).Radius(6).Background(t.Background).Border(1, t.Border).
					Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).Label("全选")
				if allBtn.Clicked() {
					for _, m := range shown {
						e.Chosen[m] = true
					}
					c.Invalidate()
				}
				allBtn.Children(func() {
					ui.Text(c, "全选").FontSize(11.5).TextColor(t.Text).SingleLine()
				})
				noneBtn := ui.Row(c).AlignItems(ui.Center).Padding(4, 12).Radius(6).Background(t.Background).Border(1, t.Border).
					Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).Label("全不选")
				if noneBtn.Clicked() {
					for _, m := range shown {
						delete(e.Chosen, m)
					}
					c.Invalidate()
				}
				noneBtn.Children(func() {
					ui.Text(c, "全不选").FontSize(11.5).TextColor(t.Text).SingleLine()
				})
				free := false
				for _, m := range shown {
					if e.Info[m].Free {
						free = true
						break
					}
				}
				if free {
					freeBtn := ui.Row(c).AlignItems(ui.Center).Padding(4, 12).Radius(6).Background(t.Background).Border(1, t.Border).
						Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).Label("只选免费")
					if freeBtn.Clicked() {
						for _, m := range shown {
							if e.Info[m].Free {
								e.Chosen[m] = true
							} else {
								delete(e.Chosen, m)
							}
						}
						c.Invalidate()
					}
					freeBtn.Children(func() {
						ui.Text(c, "只选免费").FontSize(11.5).TextColor(t.Text).SingleLine()
					})
				}
			})
		}

		// the list itself, magpie's pills: the picked one the accent's
		// soft tint, a check before it and its name in the accent; the
		// rest the white pill in the list's gray
		ui.Row(c).FillWidth().Gap(4).Wrap().Children(func() {
			for _, m := range shown {
				m := m
				active := e.Chosen[m]
				chip := ui.Row(c).AlignItems(ui.Center).Gap(5).Height(24).Padding(0, 9).Radius(999).
					Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton)
				if active {
					chip.Background(ui.Hex("#eeedfd")).Border(1, ui.RGBA(79, 70, 229, 0.45))
				} else {
					chip.Background(t.Background).Border(1, t.Border)
				}
				if chip.Clicked() {
					if active {
						delete(e.Chosen, m)
					} else {
						e.Chosen[m] = true
					}
					c.Invalidate()
				}
				chip.Children(func() {
					if active {
						ui.Text(c, "✓").FontSize(10).FontWeight(500).TextColor(ui.Hex("#4f46e5")).SingleLine()
					}
					name := ui.Text(c, pvModelLabel(e, m)).FontSize(11.5).SingleLine()
					if active {
						name.FontWeight(500).TextColor(ui.Hex("#4f46e5"))
					} else {
						name.TextColor(t.TextMuted)
					}
					// a test's dot, magpie's tdot: green it answered,
					// red it didn't
					switch e.TestRes[m] {
					case 1:
						ui.Box(c).Width(6).Height(6).Radius(3).Background(ui.Hex("#189a58"))
					case 2:
						ui.Box(c).Width(6).Height(6).Radius(3).Background(ui.Hex("#d64545"))
					}
					info, ok := e.Info[m]
					if !ok {
						return
					}
					if info.Free {
						ui.Text(c, "免费").FontSize(9).FontWeight(600).TextColor(ui.Hex("#189a58")).SingleLine()
					}
					if info.Rate > 0 && info.Rate != 1 {
						rate := ui.Text(c, pvRateText(info.Rate)+"x").FontSize(9).SingleLine()
						if info.Free {
							rate.Strikethrough().TextColor(t.TextMuted)
						} else {
							rate.TextColor(t.TextMuted)
						}
					}
					if info.Context >= 1_000_000 {
						ui.Text(c, "1M").FontSize(9).TextColor(t.TextMuted).
							Background(ui.RGBA(0, 0, 0, 0.05)).Padding(0, 4).Radius(3).SingleLine()
					}
				})
			}
		})

		// the foot: a model the list doesn't name is added by its id,
		// the vendor is asked its list again, and a small request goes
		// through each picked model — magpie's own 添加模型, Refresh and
		// Test models
		ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Column(c).Grow(1).Shrink(1).Children(func() {
				ui.TextInput(c, &e.NewID).Placeholder("添加模型 ID…").Label("添加模型").
					Background(t.Background).FillWidth()
			})
			a.pvEditSmallBtn(c, t, "添加模型", true, func() {
				if id := strings.TrimSpace(e.NewID); id != "" && !slices.Contains(e.Models, id) {
					e.Models = append(e.Models, id)
					e.Chosen[id] = true
				}
				e.NewID = ""
				c.Invalidate()
			})
			a.pvEditSmallBtn(c, t, "重新获取模型", !e.Refreshing, func() {
				e.Refreshing, e.Note, e.Err = true, "", ""
				c.Invalidate()
				id := e.ID
				go func() {
					note := ""
					if p, err := provider.Find(id); err != nil {
						note = err.Error()
					} else if ms, dropped, err := p.Refetch(context.Background()); err != nil {
						note = e.Name + " 没能获取模型列表：" + err.Error()
					} else if len(ms) == 0 {
						note = e.Name + "：0 个模型"
					} else {
						// the fresh list, with the catalog extras the
						// refetch doesn't name kept behind it; the picks
						// the vendor no longer lists are gone from the
						// draft too, or its Save would write them back
						list := make([]string, 0, len(ms)+len(e.Models))
						for _, m := range ms {
							list = append(list, m.ID)
						}
						for _, m := range e.Models {
							if !slices.Contains(list, m) {
								list = append(list, m)
							}
						}
						e.Models = list
						for _, d := range dropped {
							delete(e.Chosen, d)
						}
						note = fmt.Sprintf("%s：%d 个模型", e.Name, len(ms))
					}
					e.Refreshing = false
					e.Note = note
					c.Invalidate()
				}()
			})
			a.pvEditSmallBtn(c, t, "测试模型", !e.Testing, func() {
				ids := []string{}
				for _, m := range e.Models {
					if e.Chosen[m] {
						ids = append(ids, m)
					}
				}
				if len(ids) == 0 {
					e.Note = "先选择要测试的模型"
					c.Invalidate()
					return
				}
				e.Testing, e.Note, e.Err = true, "", ""
				e.TestRes = map[string]uint8{}
				c.Invalidate()
				id := e.ID
				go func() {
					note := ""
					if p, err := provider.Find(id); err != nil {
						note = err.Error()
					} else {
						// an image model's test draws a picture, which
						// takes longer — magpie's own 3 minutes
						ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
						res := p.TestModels(ctx, ids)
						cancel()
						bad := 0
						for i, r := range res {
							if r.OK {
								e.TestRes[ids[i]] = 1
							} else {
								e.TestRes[ids[i]] = 2
								bad++
							}
						}
						if bad == 0 {
							note = fmt.Sprintf("全部 %d 个模型都应答了", len(ids))
						} else {
							note = fmt.Sprintf("%d/%d 个模型没有应答", bad, len(ids))
						}
					}
					e.Testing = false
					e.Note = note
					c.Invalidate()
				}()
			})
		})
		if e.Note != "" {
			ui.Text(c, e.Note).FontSize(11).TextColor(t.TextMuted).SingleLine()
		}

		ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
			ui.Row(c).AlignItems(ui.Center).Gap(7).Children(func() {
				box := ui.Box(c).Width(14).Height(14).Radius(3).
					Cursor(ui.CursorPointer).Focusable().Role(ui.RoleCheckBox)
				if e.Unlisted {
					box.Background(t.Accent)
				} else {
					box.Background(t.Background).Border(1, t.Border)
				}
				checked := box.Clicked()
				box.Children(func() {
					if e.Unlisted {
						ui.Text(c, "✓").FontSize(10).TextColor(ui.Hex("#ffffff"))
					}
				})
				if checked {
					e.Unlisted = !e.Unlisted
					c.Invalidate()
				}
				ui.Text(c, "只通过路由分组使用").FontSize(11.5).TextColor(t.Text).SingleLine()
			})
		})

		rtHint(c, t, "Agent 会看到这里选中的模型；供应商以后新列的也会看到，点击模型可取消。")
	})
}

// pvEditTokens is the three token maps: the window the agents are told,
// the reply's limit, and where a conversation compacts.
func (a *App) pvEditTokens(c *ui.Context, t *ui.Theme, e *providerEdit) {
	type tok struct {
		label   string
		val     *string
		presets []string
		hint    string
	}
	for _, f := range []tok{
		{label: "上下文窗口", val: &e.Contexts, presets: []string{"128k", "200k", "256k", "400k", "512k", "1m"},
			hint: "模型可接受的上下文长度，会告知 Agent；留空时由供应商和 models.dev 决定。"},
		{label: "最大输出", val: &e.Outputs, presets: []string{"8k", "16k", "32k", "64k", "128k"},
			hint: "单次回复最大的 token 数，没设时作为 max_tokens 发给 Anthropic 的 API；留空按厂商和 models.dev。"},
		{label: "压缩阈值", val: &e.Compacts, presets: []string{"128k", "200k", "272k", "400k", "1m"},
			hint: "对话长过这里填的大小时，Codex 和 Claude Code 在该处压缩对话；用完整窗口对话更长，但每轮更慢、更贵。"},
	} {
		f := f
		rtFieldRow(c, t, f.label, func() {
			ui.TextInput(c, f.val).Placeholder("如 128k · 或 gpt-6=1m，逗号分隔").Label(f.label).
				Background(t.Background).FillWidth()
			ui.Row(c).Gap(6).AlignSelf(ui.Start).Children(func() {
				for _, pr := range f.presets {
					pr := pr
					pick := ui.Row(c).AlignItems(ui.Center).Padding(3, 9).Radius(6).Background(t.Background).Border(1, t.Border).
						Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton)
					clicked := pick.Clicked()
					pick.Children(func() {
						ui.Text(c, strings.ToUpper(pr)).FontSize(10.5).TextColor(t.TextMuted).SingleLine()
					})
					if clicked {
						*f.val = pr
						c.Invalidate()
					}
				}
			})
			rtHint(c, t, f.hint)
		})
	}
}

// pvEditProxy is the proxy the provider's requests go through: the
// settings' own, none, or one of its own.
func (a *App) pvEditProxy(c *ui.Context, t *ui.Theme, e *providerEdit) {
	rtFieldRow(c, t, "代理", func() {
		rtPillRow(c, t, []rtOpt{
			{id: "", name: "全局代理"},
			{id: "direct", name: "直连"},
			{id: "custom", name: "自定义"},
		}, e.ProxyMode, func(v string) {
			e.ProxyMode = v
		})
		if e.ProxyMode == "custom" {
			ui.TextInput(c, &e.ProxyURL).Placeholder("http://127.0.0.1:7890").Label("代理地址").
				Background(t.Background).FillWidth()
		}
		rtHint(c, t, "留空跟随设置里的代理。")
	})
}

// pvEditLimits is the load the vendor is asked to carry: its concurrency,
// its rate, and what waits when either is met.
func (a *App) pvEditLimits(c *ui.Context, t *ui.Theme, e *providerEdit) {
	type num struct {
		label string
		val   *string
		hint  string
	}
	for _, f := range []num{
		{label: "并发上限", val: &e.Concurrency, hint: "最多同时在途的请求数；留空为不限制。"},
		{label: "每分钟请求数", val: &e.RPM, hint: "每个账号或 Key 一分钟内最多发出的请求数；留空为不限制。"},
		{label: "排队上限", val: &e.QueueCap, hint: "每个 Key 或账号最多可排队的请求数，再来的请求会被立刻拒绝；0 或留空为不限制。"},
		{label: "排队等待", val: &e.QueueWait, hint: "每个请求等待的最长秒数，超时就被拒绝；0 或留空为一直等。"},
		{label: "价格倍率", val: &e.PriceRate, hint: "费用按官方价格乘以此倍率计算，只用于展示，例如 0.8 或 15；留空即官方价格。"},
	} {
		f := f
		rtFieldRow(c, t, f.label, func() {
			ui.TextInput(c, f.val).Placeholder("不限制").Label(f.label).
				Background(t.Background).FillWidth()
			rtHint(c, t, f.hint)
		})
	}
}

// pvEditSave writes the draft back: the provider and what is saved beside
// it — its models, its reply limits and its compacts.
func (a *App) pvEditSave(c *ui.Context, e *providerEdit) {
	ctxs, bad := pvParseTokensList(e.Contexts)
	if bad != "" {
		e.Err = "上下文窗口：" + bad + " 不是 128k 这样的 token 数"
		c.Invalidate()
		return
	}
	outs, bad := pvParseTokensList(e.Outputs)
	if bad != "" {
		e.Err = "最大输出：" + bad + " 不是 8k 这样的 token 数"
		c.Invalidate()
		return
	}
	cps, bad := pvParseTokensList(e.Compacts)
	if bad != "" {
		e.Err = "压缩阈值：" + bad + " 不是 272k 这样的 token 数"
		c.Invalidate()
		return
	}
	proxy := e.ProxyMode
	if proxy == "custom" {
		proxy = strings.TrimSpace(e.ProxyURL)
		if err := settings.CheckProxy(proxy); err != nil {
			e.Err = "代理：" + err.Error()
			c.Invalidate()
			return
		}
	}
	p, err := provider.Find(e.ID)
	if err != nil {
		e.Err = err.Error()
		c.Invalidate()
		return
	}
	var maxConc *int
	if n, ok := pvAtoi(e.Concurrency); ok {
		if n < 0 {
			e.Err = "并发上限不能是负数"
			c.Invalidate()
			return
		}
		maxConc = &n
	}
	rpm := p.MaxRPM
	if n, ok := pvAtoi(e.RPM); ok {
		if n < 0 {
			e.Err = "每分钟请求数不能是负数"
			c.Invalidate()
			return
		}
		rpm = n
	}
	ql, qw := p.QueueLimit, p.QueueWait
	if n, ok := pvAtoi(e.QueueCap); ok {
		ql = n
	}
	if n, ok := pvAtoi(e.QueueWait); ok {
		qw = n
	}
	if err := provider.CheckQueue(ql, qw); err != nil {
		e.Err = err.Error()
		c.Invalidate()
		return
	}
	rate := p.PriceRate
	if s := strings.TrimSpace(e.PriceRate); s != "" {
		n, err := strconv.ParseFloat(s, 64)
		if err != nil {
			e.Err = "价格倍率：" + s + " 不是一个像 0.8 或 15 的倍率"
			c.Invalidate()
			return
		}
		if msg := provider.PriceRateOK(n); msg != "" {
			e.Err = "价格倍率：" + msg
			c.Invalidate()
			return
		}
		rate = n
	}
	fallback := []string{}
	for _, line := range strings.Split(e.Fallback, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			fallback = append(fallback, line)
		}
	}

	p.Key = e.Key
	p.Unlisted = e.Unlisted
	p.Contexts = ctxs
	p.Fallback = fallback
	p.Proxy = proxy
	p.MaxConcurrency = maxConc
	p.MaxRPM = rpm
	p.QueueLimit, p.QueueWait = ql, qw
	p.PriceRate = rate
	if err := provider.Save(*p); err != nil {
		e.Err = err.Error()
		c.Invalidate()
		return
	}
	models := []string{}
	for _, m := range e.Models {
		if e.Chosen[m] {
			models = append(models, m)
		}
	}
	// a provider that named no list serves every model, the ones to come
	// later too: an all-picked save keeps it that way rather than fixing
	// today's list
	if e.WasAll && len(models) == len(e.Models) {
		models = nil
	}
	if err := provider.SetModels(e.ID, models); err != nil {
		e.Err = err.Error()
		c.Invalidate()
		return
	}
	if err := provider.SetModelOutputs(e.ID, outs); err != nil {
		e.Err = err.Error()
		c.Invalidate()
		return
	}
	if err := provider.SetModelCompacts(e.ID, cps); err != nil {
		e.Err = err.Error()
		c.Invalidate()
		return
	}
	a.closeProviderEditor()
	a.Reload()
}
