//go:build nogui && echo

package shell

// The Usage page's Overview: the allowances, the period's totals and its
// timeline, and what each agent, model, subscription account and gateway
// key spent — magpie's own Overview, drawn natively.

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/yetone/magpie/internal/provider"
	"github.com/yetone/magpie/internal/usage"
	"github.com/yetone/magpie/internal/usageapi"
)

func (a *App) usageOverview(c *ui.Context, t *ui.Theme) {
	d := a.data.Usage
	if d.Query.Tab != UsTabOverview {
		a.usSkeleton(c, t)
		return
	}
	a.usQuotas(c, t, d.Quotas, d.QuotasReading)
	if d.Overview.Calls == 0 {
		a.usEmpty(c, t, a.usNoCallsIn(a.usPeriod), "把某个 Agent 指向目录里的模型并开始使用，经过网关的每一次调用都会记在这里。")
		a.usNote(c, t, "")
		return
	}
	a.usOverviewKPIs(c, t, d.Overview)
	a.usOverviewChart(c, t, d.Overview)

	a.usSectionHead(c, t, "agents", "Agents", fmt.Sprintf("%d 个", len(d.Overview.Agents)), true)
	if !a.usFolds["agents"] {
		a.usAgentList(c, t, d.Overview)
	}
	a.usSectionHead(c, t, "models", "Models", fmt.Sprintf("%d 个", len(d.Overview.Models)), true)
	if !a.usFolds["models"] {
		a.usUsageList(c, t, d.Overview.Models, "model")
	}
	if len(d.Overview.Accounts) > 0 {
		a.usSectionHead(c, t, "accounts", "Accounts", fmt.Sprintf("%d 个", len(d.Overview.Accounts)), true)
		if !a.usFolds["accounts"] {
			a.usUsageList(c, t, d.Overview.Accounts, "account")
		}
	}
	if len(d.Overview.CallerKeys) > 0 {
		a.usSectionHead(c, t, "keys", "网关密钥", fmt.Sprintf("%d 个", len(d.Overview.CallerKeys)), true)
		if !a.usFolds["keys"] {
			a.usUsageList(c, t, d.Overview.CallerKeys, "key")
		}
	}
	a.usNote(c, t, "每一次经过网关的调用都按供应商自己上报的用量计 · "+d.Overview.Path)
}

// usSkeleton is a segment whose read has not landed: the shape of what is
// coming, not an empty page that looks broken.
func (a *App) usSkeleton(c *ui.Context, t *ui.Theme) {
	ui.Row(c).FillWidth().Gap(10).Padding(16, 20).Children(func() {
		for i := 0; i < 4; i++ {
			ui.Column(c).Grow(1).Basis(0).Height(74).Radius(9).
				Background(a.fieldBg(t)).Border(1, t.Border)
		}
	})
	ui.Row(c).AlignItems(ui.Center).Gap(8).Padding(30, 20).Children(func() {
		ui.Spinner(c).Size(14, 14)
		ui.Text(c, "正在读取用量记录…").FontSize(12).TextColor(t.TextMuted).SingleLine()
	})
}

// usNoCallsIn is what an empty page says of the period it is on.
func (a *App) usNoCallsIn(p usage.Period) string {
	switch p {
	case usage.Today:
		return "今天还没有调用。"
	case usage.Week:
		return "最近 7 天没有调用。"
	case usage.All:
		return "还没有任何调用。"
	}
	return "最近 30 天没有调用。"
}

// ---- allowances ----

// usQuotas draws the subscription allowances: one card a provider, an
// account under it, each window a meter. A card can be read again from its
// vendor, a Codex account's reset spent, and a WorkBuddy, Trae CN, MiniMax
// Code or Qoder account's daily check-in pressed — the writes magpie's own
// cards carry.
func (a *App) usQuotas(c *ui.Context, t *ui.Theme, quotas []provider.SubscriptionQuota, reading bool) {
	if len(quotas) == 0 && !reading {
		return
	}
	groups := groupQuotas(quotas)

	ui.Row(c).AlignItems(ui.Center).Gap(8).Padding(14, 20, 6, 20).Children(func() {
		foldBtn := ui.Row(c).AlignItems(ui.Center).Gap(4).Cursor(ui.CursorPointer).
			Focusable().Role(ui.RoleButton).Label("订阅额度")
		if foldBtn.Clicked() {
			a.usQuotaFold = !a.usQuotaFold
			c.Invalidate()
		}
		foldBtn.Children(func() {
			chev := "▾"
			if a.usQuotaFold {
				chev = "▸"
			}
			ui.Text(c, chev).FontSize(10).TextColor(t.TextMuted).SingleLine()
			ui.Text(c, "订阅额度").FontSize(12.5).FontWeight(600).TextColor(t.Text).SingleLine()
		})
		if reading {
			ui.Row(c).AlignItems(ui.Center).Gap(4).Children(func() {
				ui.Spinner(c).Size(10, 10)
				ui.Text(c, "正在读取").FontSize(10.5).TextColor(t.TextMuted).SingleLine()
			})
		}
		ui.Spacer(c)

		// Used or left: the same meters, the other side of them.
		if a.usQuotaLeft {
			if ui.Button(c, "看已用").Clicked() {
				a.usQuotaLeft = false
				c.Invalidate()
			}
		} else {
			if ui.Button(c, "看剩余").Clicked() {
				a.usQuotaLeft = true
				c.Invalidate()
			}
		}

		maskLabel := "隐藏账号"
		if a.usHideAccounts {
			maskLabel = "显示账号"
		}
		if ui.Button(c, maskLabel).Clicked() {
			a.usHideAccounts = !a.usHideAccounts
			c.Invalidate()
		}
	})

	if a.usQuotaFold {
		return
	}

	const perRow = 3
	for i := 0; i < len(groups); i += perRow {
		row := groups[i:min(i+perRow, len(groups))]
		ui.Row(c).AlignItems(ui.Stretch).Gap(10).Padding(2, 20, 6, 20).FillWidth().Children(func() {
			for _, g := range row {
				a.usQuotaCard(c, t, g)
			}
		})
	}
}

// quotaGroup is one provider's card: its accounts, in magpie's order.
type quotaGroup struct {
	Provider string
	Name     string
	Icon     string
	Plan     string
	Quotas   []provider.SubscriptionQuota
}

// groupQuotas groups the cards by provider, keeping magpie's order: the
// first card of a provider names it, and each of its accounts follows.
func groupQuotas(quotas []provider.SubscriptionQuota) []quotaGroup {
	var out []quotaGroup
	at := map[string]int{}
	for _, q := range quotas {
		if i, ok := at[q.Provider]; ok {
			out[i].Quotas = append(out[i].Quotas, q)
			continue
		}
		at[q.Provider] = len(out)
		out = append(out, quotaGroup{Provider: q.Provider, Name: q.Name, Icon: q.Icon, Plan: q.Plan, Quotas: []provider.SubscriptionQuota{q}})
	}
	return out
}

// usQuotaCard is one provider's allowances: its name, then each account
// with its windows.
func (a *App) usQuotaCard(c *ui.Context, t *ui.Theme, g quotaGroup) {
	card := ui.Column(c).Grow(1).Basis(0).Padding(10, 12).Radius(10).
		Background(a.fieldBg(t)).Border(1, t.Border).Gap(8)
	card.Children(func() {
		ui.Row(c).AlignItems(ui.Center).Gap(7).Children(func() {
			ui.Image(c, iconPictureFor(g.Icon)).Size(18, 18).Radius(4)
			ui.Text(c, g.Name).FontSize(12.5).FontWeight(650).TextColor(t.Text).Grow(1).Shrink(1).SingleLine()
			if g.Plan != "" {
				a.usMutedPill(c, t, g.Plan)
			}
			// the card read again, from its vendor alone
			refresh := ui.Row(c).Center().Width(20).Height(20).Radius(5).
				Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).
				Label("刷新 " + g.Name).Tooltip("只重新读取这张卡片")
			if refresh.Hovered() {
				refresh.Background(a.hoverBg(t))
			}
			if refresh.Clicked() {
				go func() {
					qs := RefreshQuotaCard(g.Provider, "")
					if a.win != nil {
						a.win.Update(func() {
							a.data.Usage.Quotas = qs
							a.win.Invalidate()
						})
					}
				}()
			}
			refresh.Children(func() {
				ui.Icon(c, refreshIcon()).Size(11, 11).TextColor(t.TextMuted)
			})
		})

		for _, q := range g.Quotas {
			a.usQuotaAccount(c, t, q)
		}
	})
}

// usQuotaAccount is one account's windows under its card: who it is,
// each row's meter and reset, and what is left besides them. The rows
// are a family or a pool's, as magpie's own page is, unless this
// account is switched to every model's own.
func (a *App) usQuotaAccount(c *ui.Context, t *ui.Theme, q provider.SubscriptionQuota) {
	if q.User != "" || q.Error != "" || q.Balance != "" || len(q.Windows) > 0 {
		rows := quotaRowsOf(q)
		collapsed := quotaCollapsed(rows)
		models := quotaModelRows(q.Windows)
		key := q.Provider + "|" + q.User
		every := a.usEveryModel != nil && a.usEveryModel[key]
		ui.Column(c).FillWidth().Gap(5).Children(func() {
			head := ui.Row(c).AlignItems(ui.Center).Gap(6)
			head.Children(func() {
				if q.User != "" {
					ui.Text(c, a.usAccountName(q.User)).FontSize(11).FontWeight(550).
						TextColor(t.TextMuted).Grow(1).Shrink(1).SingleLine().
						Tooltip(q.User)
				}
				if q.Plan != "" && q.User != "" {
					a.usMutedPill(c, t, q.Plan)
				}
				// a Codex account's rate-limit resets, spent from here
				if q.Resets != nil && q.Resets.Count > 0 {
					if q.Provider == "codex" {
						use := ui.Row(c).Padding(1, 6).Radius(4).Background(ui.RGBA(0, 122, 255, 0.10)).
							Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).
							Label("使用一次重置").Tooltip(fmt.Sprintf("还剩 %d 次限频重置", q.Resets.Count))
						if use.Clicked() {
							a.usResetConfirmUser = q.User
							c.Invalidate()
						}
						use.Children(func() {
							ui.Text(c, fmt.Sprintf("重置 %d", q.Resets.Count)).FontSize(10).FontWeight(600).
								TextColor(ui.Hex("#007aff")).SingleLine()
						})
					} else {
						a.usMutedPill(c, t, fmt.Sprintf("重置 %d", q.Resets.Count))
					}
				}
				// a family or a pool's figures, or every window of its
				// own, turned here as magpie's page turns them
				if collapsed {
					label := fmt.Sprintf("每个模型 (%d)", len(models))
					tip := "每个模型各自的额度，一层一层"
					if every {
						label, tip = "按组", "一组模型共用的 5 小时与每周额度"
						if !quotaPooled(q.Windows) {
							label, tip = "按族", "一族一个数，取用得最多的那个模型的"
						}
					}
					swith := ui.Row(c).Padding(1, 6).Radius(4).Background(ui.RGBA(0, 122, 255, 0.10)).
						Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).
						Label(label).Tooltip(tip).Shrink(0)
					if swith.Clicked() {
						if a.usEveryModel == nil {
							a.usEveryModel = map[string]bool{}
						}
						a.usEveryModel[key] = !every
						c.Invalidate()
					}
					swith.Children(func() {
						ui.Text(c, label).FontSize(10).FontWeight(550).TextColor(ui.Hex("#007aff")).SingleLine()
					})
				}
			})

			if q.Error != "" {
				ui.Row(c).FillWidth().Padding(5, 8).Radius(6).
					Background(ui.RGBA(255, 59, 48, 0.08)).Children(func() {
					ui.Text(c, q.Error).FontSize(10).TextColor(ui.Hex("#c0392b")).SingleLine().
						Tooltip(q.Error)
				})
			}

			shown := rows
			if every && collapsed {
				shown = models
			}
			for _, r := range shown {
				a.usQuotaWindow(c, t, r)
			}

			if q.Balance != "" {
				ui.Row(c).AlignItems(ui.Center).Gap(5).Children(func() {
					ui.Text(c, "余额").FontSize(10.5).TextColor(t.TextMuted).SingleLine()
					ui.Text(c, q.Balance).FontSize(11).FontWeight(600).TextColor(t.Text).SingleLine()
				})
			}

			// WorkBuddy's, Trae CN's, MiniMax Code's and Qoder's daily
			// check-in, and how this account's last one went
			if q.Checkins {
				a.usCheckinRow(c, t, q)
			}

			if q.ReadAt != nil && len(q.Windows) > 0 {
				ui.Text(c, "更新于 "+usAgo(*q.ReadAt)).FontSize(9.5).TextColor(t.TextMuted).SingleLine()
			}
		})
	}
}

// usQuotaWindow is one row: its name, the meter, the figure and when it
// starts again. A row that stands for a family or a pool says which in
// its tooltip.
func (a *App) usQuotaWindow(c *ui.Context, t *ui.Theme, r quotaRow) {
	w := r.w
	// Used is the vendor's own percent (0-100), as the CLI's --json says
	// it ("used": 17.25)
	used := w.Used
	if used < 0 {
		used = 0
	}
	if used > 100 {
		used = 100
	}
	show := used
	if a.usQuotaLeft {
		show = 100 - used
	}
	name := w.Name
	if name == "" {
		name = "额度"
	}

	ui.Column(c).FillWidth().Gap(3).Tooltip(r.tool).Children(func() {
		ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
			ui.Text(c, name).FontSize(10.5).TextColor(t.TextMuted).Grow(1).Shrink(1).SingleLine()
			if w.Unlimited {
				ui.Text(c, "不限").FontSize(10.5).FontWeight(600).TextColor(t.Text).SingleLine()
			} else {
				ui.Text(c, fmt.Sprintf("%.0f%%", show)).FontSize(10.5).FontWeight(650).
					TextColor(usQuotaTone(used)).SingleLine()
			}
		})
		if !w.Unlimited {
			bar := ui.Row(c).FillWidth().Height(5).Radius(3).Background(a.usTrackBg(t))
			bar.Children(func() {
				ui.Row(c).WidthPercent(float32(show)).Height(5).Radius(3).Background(usQuotaTone(used))
			})
		}
		foot := ""
		if w.Limit > 0 {
			if a.usQuotaLeft {
				foot = fmt.Sprintf("剩 %s / %s %s", usAmount(w.Limit-w.Amount), usAmount(w.Limit), w.Unit)
			} else {
				foot = fmt.Sprintf("用 %s / %s %s", usAmount(w.Amount), usAmount(w.Limit), w.Unit)
			}
		} else if w.Display != "" {
			foot = w.Display
		}
		if w.ResetsAt != nil {
			when := "重置于 " + usResetIn(*w.ResetsAt)
			if foot != "" {
				foot += " · " + when
			} else {
				foot = when
			}
		}
		if foot != "" {
			ui.Text(c, foot).FontSize(9.5).TextColor(t.TextMuted).SingleLine()
		}
	})
}

// refreshUsageQuotas hands a fresh reading to the Usage page's allowance
// cards, when the page is on them: what a vendor's background read does
// for the panel does for the page too, as the card's own refresh does.
func (a *App) refreshUsageQuotas(qs []provider.SubscriptionQuota) {
	if a.win == nil {
		return
	}
	a.win.Update(func() {
		if a.page == PageUsage && a.usTab == UsTabOverview {
			a.data.Usage.Quotas = qs
			a.win.Invalidate()
		}
	})
}

// usQuotaTone colours a window by how much of it is used, in the vendor's
// own percent (0-100): a window at 90% reads the same here as in magpie.
func usQuotaTone(used float64) ui.Color {
	switch {
	case used >= 90:
		return ui.Hex("#ff3b30")
	case used >= 70:
		return ui.Hex("#ff9500")
	}
	return ui.Hex("#34c759")
}

// usAmount writes a credit amount without a trailing .0.
func usAmount(v float64) string {
	if v == math.Trunc(v) {
		return fmt.Sprintf("%.0f", v)
	}
	return fmt.Sprintf("%.1f", v)
}

// usResetIn says how long a window has left, short: 3h 12m, 2d 4h.
func usResetIn(at time.Time) string {
	d := time.Until(at)
	if d <= 0 {
		return "即将"
	}
	return FormatSeconds(int64(d.Seconds()))
}

// usAgo says when something was read, as a reader counts it.
func usAgo(at time.Time) string {
	d := time.Since(at)
	switch {
	case d < time.Minute:
		return "刚刚"
	case d < time.Hour:
		return fmt.Sprintf("%d 分钟前", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%d 小时前", int(d.Hours()))
	}
	return fmt.Sprintf("%d 天前", int(d.Hours()/24))
}

// usAccountName is an account as the page may show it: its name, or a
// masked one when the reader is hiding accounts for a screenshot.
func (a *App) usAccountName(user string) string {
	if !a.usHideAccounts || user == "" {
		return user
	}
	at := strings.LastIndex(user, "@")
	if at <= 0 {
		if len(user) <= 2 {
			return "••"
		}
		return user[:1] + strings.Repeat("•", len(user)-1)
	}
	name, domain := user[:at], user[at:]
	keep := 1
	if len(name) > 3 {
		keep = 2
	}
	return name[:keep] + strings.Repeat("•", max(1, len(name)-keep)) + domain
}

// usCheckinRow is a daily check-in's button and how the last one went.
func (a *App) usCheckinRow(c *ui.Context, t *ui.Theme, q provider.SubscriptionQuota) {
	ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
		label := "每日签到"
		if q.Checkin != nil && q.Checkin.Checked() {
			label = "今日已签到"
		}
		btn := ui.Row(c).Padding(1, 6).Radius(4).Background(ui.RGBA(52, 199, 89, 0.10)).
			Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).Label(label)
		if btn.Clicked() {
			go a.usRunCheckin(q.Provider)
		}
		btn.Children(func() {
			ui.Text(c, label).FontSize(10).FontWeight(600).TextColor(ui.Hex("#1f7a3a")).SingleLine()
		})
		if q.Checkin != nil {
			word := "已签到"
			if !q.Checkin.Checked() {
				word = q.Checkin.Msg
				if word == "" {
					word = "未签到"
				}
			}
			ui.Text(c, word).FontSize(9.5).TextColor(t.TextMuted).Shrink(1).SingleLine()
		}
	})
}

// ---- totals and the timeline ----

func (a *App) usOverviewKPIs(c *ui.Context, t *ui.Theme, u usageapi.Usage) {
	tokens := usageTokensOf(u.Totals, a.usWithCache)
	cache := u.CacheRead
	prompt := u.Input + u.CacheRead + u.CacheWrite
	hit := 0.0
	if prompt > 0 {
		hit = float64(u.CacheRead) / float64(prompt)
	}
	callsSub := ""
	if u.Errors > 0 || u.Retried > 0 || u.Unpriced > 0 {
		var parts []string
		if u.Errors > 0 {
			parts = append(parts, fmt.Sprintf("%d 失败", u.Errors))
		}
		if u.Retried > 0 {
			parts = append(parts, fmt.Sprintf("%d 次重试后成功", u.Retried))
		}
		if u.Unpriced > 0 {
			parts = append(parts, fmt.Sprintf("%d 次无价格", u.Unpriced))
		}
		callsSub = strings.Join(parts, " · ")
	}

	ui.Row(c).FillWidth().Gap(8).Padding(8, 20, 10, 20).Children(func() {
		a.usKPI(c, t, FormatCount(tokens), a.usTokenLabel(), fmt.Sprintf("%s 输入 · %s 输出", FormatCount(u.Input), FormatCount(u.Output)), t.TextMuted)
		a.usKPI(c, t, FormatCount(cache), "缓存读取", fmt.Sprintf("命中率 %.0f%%", hit*100), t.TextMuted)
		a.usKPI(c, t, FormatCount(u.Reasoning), "推理", "含在输出里", t.TextMuted)
		subColor := t.TextMuted
		if u.Errors > 0 {
			subColor = ui.Hex("#c0392b")
		}
		a.usKPI(c, t, FormatCount(u.Calls), "次调用", callsSub, subColor)
	})

	// the cache's tokens in the totals or out, as magpie's tokens tile
	// offers: one click either way
	ui.Row(c).AlignItems(ui.Center).Gap(6).Padding(0, 20, 6, 20).Children(func() {
		if ui.Button(c, "Token 口径").Clicked() {
			a.usWithCache = !a.usWithCache
			c.Invalidate()
		}
		word := "不含缓存"
		if a.usWithCache {
			word = "含缓存读取与写入"
		}
		ui.Text(c, word).FontSize(10).TextColor(t.TextMuted).SingleLine()
	})
}

// usTokenLabel is the tokens tile's label, saying which way the cache is
// counted.
func (a *App) usTokenLabel() string {
	if a.usWithCache {
		return "Token（含缓存）"
	}
	return "Token"
}

// usOverviewChart is the period's timeline: one bar a bucket, output on
// top of input, with the peak beside it.
func (a *App) usOverviewChart(c *ui.Context, t *ui.Theme, u usageapi.Usage) {
	if len(u.Series) == 0 {
		return
	}
	pts := u.Series
	peak := 1
	for _, p := range pts {
		if n := usageTokensOf(p.Totals, a.usWithCache); n > peak {
			peak = n
		}
	}
	ui.Row(c).AlignItems(ui.Center).Gap(8).Padding(6, 20, 0, 20).Children(func() {
		ui.Text(c, fmt.Sprintf("峰值 %s", FormatCount(peak))).FontSize(10).TextColor(t.TextMuted).SingleLine()
		ui.Spacer(c)
		ui.Row(c).Width(8).Height(8).Radius(2).Background(ui.Hex("#007aff"))
		ui.Text(c, "输入").FontSize(10).TextColor(t.TextMuted).SingleLine()
		ui.Row(c).Width(8).Height(8).Radius(2).Background(ui.Hex("#34c759"))
		ui.Text(c, "输出").FontSize(10).TextColor(t.TextMuted).SingleLine()
	})

	const chartH = 110.0
	every := 1
	if n := len(pts); n > 8 {
		every = int(math.Ceil(float64(n) / 6))
	}
	ui.Column(c).FillWidth().Gap(4).Padding(0, 20, 8, 20).Children(func() {
		ui.Row(c).AlignItems(ui.End).Gap(2).Height(float32(chartH)).FillWidth().Children(func() {
			for _, p := range pts {
				p := p
				total := usageTokensOf(p.Totals, a.usWithCache)
				in := total - p.Output
				if in < 0 {
					in = 0
				}
				cell := ui.Column(c).Grow(1).Shrink(1).Basis(0).Height(float32(chartH))
				cell.Tooltip(usPointTip(p, a.usWithCache))
				if cell.Hovered() {
					cell.Background(ui.RGBA(0, 122, 255, 0.05))
				}
				cell.Children(func() {
					ui.Spacer(c)
					if h := float32(chartH) * float32(p.Output) / float32(peak); h > 0.5 {
						ui.Row(c).FillWidth().Height(h).Background(ui.Hex("#34c759"))
					}
					if h := float32(chartH) * float32(in) / float32(peak); h > 0.5 {
						ui.Row(c).FillWidth().Height(h).Background(ui.Hex("#007aff"))
					}
				})
			}
		})
		ui.Row(c).FillWidth().Gap(2).Children(func() {
			for i, p := range pts {
				label := ""
				if i%every == 0 || i == len(pts)-1 && (len(pts)-1)%every >= every/2 {
					label = p.Label
				}
				ui.Row(c).Grow(1).Basis(0).Children(func() {
					if label != "" {
						ui.Text(c, label).FontSize(9).TextColor(t.TextMuted).SingleLine()
					}
				})
			}
		})
	})
}

// usPointTip is one bucket of the timeline in words.
func usPointTip(p usage.Point, withCache bool) string {
	when := p.Label
	switch {
	case p.Time.IsZero():
	case true:
		when = p.Time.Local().Format("1月2日")
	}
	tokens := usageTokensOf(p.Totals, withCache)
	tip := fmt.Sprintf("%s · %s tokens · %d 次调用", when, FormatCount(tokens), p.Calls)
	if p.Cost > 0 {
		tip += " · ≈" + FormatCost(p.Cost)
	}
	return tip
}

// ---- the lists ----

// usAgentList is the Agents list: each agent's share of the period's
// tokens, what it spent, and the way to hide one from the list alone.
func (a *App) usAgentList(c *ui.Context, t *ui.Theme, u usageapi.Usage) {
	total := max(1, usageTokensOf(u.Totals, a.usWithCache))
	var hidden []usageapi.Group
	shown := 0
	for _, g := range u.Agents {
		if a.usHiddenAgents[g.ID] {
			hidden = append(hidden, g)
			continue
		}
		shown++
		a.usUsageRow(c, t, g, total, "agent")
	}
	if shown == 0 && len(hidden) > 0 {
		ui.Text(c, "这一段时间的调用都来自已隐藏的 Agent。").FontSize(11).TextColor(t.TextMuted).Padding(4, 20)
	}
	if len(hidden) == 0 {
		return
	}
	line := ui.Row(c).AlignItems(ui.Center).Gap(5).Padding(5, 20).Cursor(ui.CursorPointer).
		Focusable().Role(ui.RoleButton).Label(fmt.Sprintf("%d 个已隐藏", len(hidden)))
	if line.Clicked() {
		a.usHiddenOpen = !a.usHiddenOpen
		c.Invalidate()
	}
	line.Children(func() {
		ui.Text(c, fmt.Sprintf("%d 个已隐藏", len(hidden))).FontSize(10.5).TextColor(t.TextMuted).SingleLine()
		ui.Text(c, "· 只影响这个列表，合计与图表仍计入").FontSize(10).TextColor(t.TextMuted).SingleLine()
	})
	if a.usHiddenOpen {
		for _, g := range hidden {
			g := g
			ui.Row(c).AlignItems(ui.Center).Gap(8).Padding(4, 24).Children(func() {
				ui.Image(c, iconPictureFor(g.Icon)).Size(16, 16).Radius(4)
				ui.Text(c, g.Name).FontSize(11.5).TextColor(t.TextMuted).Grow(1).SingleLine()
				if ui.Button(c, "取消隐藏").Clicked() {
					delete(a.usHiddenAgents, g.ID)
					c.Invalidate()
				}
			})
		}
	}
}

// usUsageList is the Models, Accounts or Gateway keys list.
func (a *App) usUsageList(c *ui.Context, t *ui.Theme, groups []usageapi.Group, kind string) {
	total := 1
	for _, g := range groups {
		if n := usageTokensOf(g.Totals, a.usWithCache); n > total {
			total = n
		}
	}
	for _, g := range groups {
		a.usUsageRow(c, t, g, total, kind)
	}
}

// usUsageRow is one line of a list: who, its share, what it spent, and its
// cost.
func (a *App) usUsageRow(c *ui.Context, t *ui.Theme, g usageapi.Group, total int, kind string) {
	tokens := usageTokensOf(g.Totals, a.usWithCache)
	share := float64(tokens) / float64(max(1, total))

	row := ui.Row(c).AlignItems(ui.Center).Gap(10).Padding(6, 20).FillWidth()
	if row.Hovered() {
		row.Background(a.hoverBg(t))
	}
	row.Children(func() {
		switch kind {
		case "agent":
			ui.Image(c, iconPictureFor(g.Icon)).Size(20, 20).Radius(4)
		case "model":
			ui.Image(c, iconPictureFor(g.Icon)).Size(18, 18).Radius(4)
		default:
			ui.Image(c, iconPictureFor(g.Icon)).Size(18, 18).Radius(4)
		}

		ui.Column(c).Grow(1).Shrink(1).Gap(1).Children(func() {
			name := g.Name
			if name == "" {
				name = "未记录账号"
			}
			ui.Text(c, name).FontSize(12).FontWeight(550).TextColor(t.Text).SingleLine()
			sub := usGroupSub(g, a.usWithCache)
			if sub != "" {
				ui.Text(c, sub).FontSize(10).TextColor(t.TextMuted).SingleLine()
			}
		})

		// the share of the tokens, a bar in its track
		a.usShareBar(c, t, share, ui.Hex("#007aff"), 90)
		ui.Text(c, fmt.Sprintf("%.0f%%", share*100)).FontSize(10).TextColor(t.TextMuted).Width(34).SingleLine()

		ui.Column(c).Width(150).Shrink(0).Gap(1).Children(func() {
			ui.Text(c, FormatCount(tokens)).FontSize(12).FontWeight(600).TextColor(t.Text).SingleLine()
			inOut := fmt.Sprintf("%s 入 · %s 出", FormatCount(g.Input), FormatCount(g.Output))
			if g.CacheRead > 0 {
				inOut += fmt.Sprintf(" · %s 缓存", FormatCount(g.CacheRead))
			}
			ui.Text(c, inOut).FontSize(9.5).TextColor(t.TextMuted).SingleLine()
		})

		cost := "—"
		costC := t.TextMuted
		if g.Cost > 0 {
			cost = "≈" + FormatCost(g.Cost)
			costC = t.Text
		}
		ui.Text(c, cost).FontSize(11.5).TextColor(costC).Width(78).Shrink(0).SingleLine()

		if kind == "agent" {
			hide := ui.Row(c).Center().Width(22).Height(20).Radius(4).
				Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).
				Label("从列表中隐藏 " + g.Name).Tooltip("从这个列表隐藏")
			if hide.Hovered() {
				hide.Background(a.hoverBg(t))
			}
			if hide.Clicked() {
				a.usHiddenAgents[g.ID] = true
				c.Invalidate()
			}
			hide.Children(func() {
				ui.Text(c, "✕").FontSize(10).TextColor(t.TextMuted).SingleLine()
			})
		}
	})
}

// usGroupSub is a list row's second line: what the calls were, and what
// went wrong among them.
func usGroupSub(g usageapi.Group, withCache bool) string {
	var parts []string
	if g.Sub != "" {
		parts = append(parts, g.Sub)
	}
	parts = append(parts, fmt.Sprintf("%d 次调用", g.Calls))
	if g.Errors > 0 {
		parts = append(parts, fmt.Sprintf("%d 失败", g.Errors))
	}
	if g.Loops > 0 {
		parts = append(parts, fmt.Sprintf("%d 卡死", g.Loops))
	}
	if g.Retried > 0 {
		parts = append(parts, fmt.Sprintf("%d 重试后成功", g.Retried))
	}
	if g.Timed > 0 {
		parts = append(parts, "TTFT "+FormatDuration(g.TTFT/int64(g.Timed)))
	}
	if s := outputSpeed(g.Totals); s > 0 {
		parts = append(parts, fmt.Sprintf("%.0f tok/s", s))
	}
	return strings.Join(parts, " · ")
}
