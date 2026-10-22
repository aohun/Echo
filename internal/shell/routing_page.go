//go:build nogui && echo

package shell

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/yetone/magpie/internal/provider"
)

// routingView renders the complete Routing page.
func (a *App) routingView(c *ui.Context) {
	t := c.Theme()
	rd := a.data.Routing

	// Page root container strictly adheres to layout rules:
	// Generous sidebar spacing (Padding 8, 24, 0, 24) -> total distance from sidebar divider >= 36-40px.
	ui.Column(c).Fill().Padding(8, 24, 0, 24).Gap(16).Children(func() {
		// Notice banner if active
		if a.rtNotice != "" {
			a.routingNoticeBanner(c, t)
		}

		// 1. Header & Controls
		a.routingHeader(c, t, rd)

		// 2. The stage: what routing does with each request, live
		a.routingFlow(c, t, rd)

		// 3. Quick Templates Bar
		if len(rd.Templates) > 0 {
			a.routingTemplatesBar(c, t, rd.Templates)
		}

		// 4. Routing Groups Section
		a.routingGroupsSection(c, t, rd)

		// 5. Account & Key Pools Section
		if len(rd.Pools) > 0 {
			a.routingPoolsSection(c, t, rd.Pools)
		}

		// 6. Live Activity & Recent Traces Section
		a.routingTraceSection(c, t, rd)
	})
}

// routingNoticeBanner renders a notification banner at the top of the page.
func (a *App) routingNoticeBanner(c *ui.Context, t *ui.Theme) {
	bg := ui.RGBA(52, 199, 89, 0.12)
	borderC := ui.RGBA(52, 199, 89, 0.3)
	textC := ui.Hex("#34c759")
	if a.rtNoticeErr {
		bg = ui.RGBA(255, 59, 48, 0.12)
		borderC = ui.RGBA(255, 59, 48, 0.3)
		textC = ui.Hex("#ff3b30")
	}

	ui.Row(c).AlignItems(ui.Center).Gap(8).Padding(8, 14).Radius(8).
		Background(bg).Border(1, borderC).FillWidth().Children(func() {
		ui.Text(c, a.rtNotice).FontSize(12).FontWeight(500).TextColor(textC).Grow(1).Shrink(1).SingleLine()
		closeBtn := ui.Text(c, "✕").FontSize(11).TextColor(textC).Cursor(ui.CursorPointer)
		if closeBtn.Clicked() {
			a.rtNotice = ""
			c.Invalidate()
		}
	})
}

// routingHeader renders the title, live indicator, stats summary, and action buttons.
func (a *App) routingHeader(c *ui.Context, t *ui.Theme, rd RoutingData) {
	ui.Row(c).AlignItems(ui.Center).FillWidth().Children(func() {
		// Title & Subtitle
		ui.Column(c).Grow(1).Shrink(1).Gap(3).Children(func() {
			ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
				ui.Text(c, "路由").FontSize(18).FontWeight(700).TextColor(t.Text).SingleLine()
				// Live green pulsing dot
				ui.Row(c).AlignItems(ui.Center).Gap(5).Padding(2, 7).Radius(10).
					Background(ui.RGBA(52, 199, 89, 0.12)).Children(func() {
					ui.Row(c).Width(6).Height(6).Radius(3).Background(ui.Hex("#34c759"))
					ui.Text(c, "实时 · 网关正在做的决定").FontSize(10.5).FontWeight(500).TextColor(ui.Hex("#34c759")).SingleLine()
				})
			})

			// Stats pills
			ui.Row(c).AlignItems(ui.Center).Gap(10).Children(func() {
				ui.Text(c, fmt.Sprintf("总请求数：%d", rd.Totals.Requests)).
					FontSize(11.5).TextColor(t.TextMuted).SingleLine()
				ui.Text(c, "·").FontSize(11.5).TextColor(t.TextMuted).SingleLine()
				ui.Text(c, fmt.Sprintf("重新路由：%d", rd.Totals.Rerouted)).
					FontSize(11.5).TextColor(t.TextMuted).SingleLine()
				ui.Text(c, "·").FontSize(11.5).TextColor(t.TextMuted).SingleLine()
				ui.Text(c, fmt.Sprintf("异常报错：%d", rd.Totals.Errors)).
					FontSize(11.5).TextColor(t.TextMuted).SingleLine()
			})
		})

		// Right-aligned actions (Pinned to the right edge with Shrink(0))
		ui.Row(c).AlignItems(ui.Center).Gap(8).Shrink(0).Children(func() {
			// Hide accounts toggle
			maskBtn := ui.Row(c).AlignItems(ui.Center).Gap(5).Padding(5, 10).Radius(6).
				Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).Label("隐藏账号")
			if a.rtHideAccounts {
				maskBtn.Background(t.Accent).Border(1, t.Accent)
			} else {
				maskBtn.Background(a.fieldBg(t)).Border(1, t.Border)
			}
			if maskBtn.Clicked() {
				a.rtHideAccounts = !a.rtHideAccounts
				c.Invalidate()
			}
			maskBtn.Children(func() {
				maskC := t.Text
				if a.rtHideAccounts {
					maskC = ui.Hex("#ffffff")
				}
				ui.Text(c, "隐藏账号标识").FontSize(11.5).FontWeight(500).TextColor(maskC).SingleLine()
			})

			// Auto groups toggle
			autoBtn := ui.Row(c).AlignItems(ui.Center).Gap(5).Padding(5, 10).Radius(6).
				Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).Label("自动分组")
			if rd.AutoGroupsOn {
				autoBtn.Background(ui.RGBA(52, 199, 89, 0.12)).Border(1, ui.RGBA(52, 199, 89, 0.3))
			} else {
				autoBtn.Background(a.fieldBg(t)).Border(1, t.Border)
			}
			if autoBtn.Clicked() {
				_ = routingFound(!rd.AutoGroupsOn)
				a.Reload()
			}
			autoBtn.Children(func() {
				autoColor := t.TextMuted
				if rd.AutoGroupsOn {
					autoColor = ui.Hex("#34c759")
				}
				ui.Text(c, "自动识别分组: 开启").FontSize(11.5).FontWeight(500).TextColor(autoColor).SingleLine()
			})

			// Import button
			importBtn := ui.Row(c).AlignItems(ui.Center).Padding(5, 12).Radius(6).
				Background(a.fieldBg(t)).Border(1, t.Border).
				Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).Label("导入")
			if importBtn.Clicked() {
				a.rtImportModal = &rtImportModalState{}
				c.Invalidate()
			}
			importBtn.Children(func() {
				ui.Text(c, "从文本导入").FontSize(11.5).FontWeight(500).TextColor(t.Text).SingleLine()
			})

			// New group button (Primary)
			newBtn := ui.Row(c).AlignItems(ui.Center).Gap(4).Padding(5, 14).Radius(6).
				Background(t.Accent).Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).Label("新建分组")
			if newBtn.Clicked() {
				a.openCreateGroupModal()
			}
			newBtn.Children(func() {
				ui.Text(c, "+ 新建分组").FontSize(11.5).FontWeight(600).TextColor(ui.Hex("#ffffff")).SingleLine()
			})
		})
	})
}

// routingTemplatesBar renders quick setup buttons for templates.
func (a *App) routingTemplatesBar(c *ui.Context, t *ui.Theme, templates []RoutingTemplateRow) {
	ui.Row(c).AlignItems(ui.Center).Gap(8).Padding(8, 12).Radius(8).
		Background(a.fieldBg(t)).Border(1, t.Border).FillWidth().Children(func() {
		ui.Text(c, "从模板快速创建:").FontSize(11.5).FontWeight(500).TextColor(t.TextMuted).Shrink(0).SingleLine()

		for _, tmpl := range templates {
			curTmpl := tmpl
			disabled := curTmpl.Why != ""
			tBtn := ui.Row(c).AlignItems(ui.Center).Gap(4).Padding(3, 8).Radius(5).
				Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton)

			if disabled {
				tBtn.Background(ui.RGBA(0, 0, 0, 0.03)).Tooltip(curTmpl.Why)
			} else {
				tBtn.Background(ui.RGBA(0, 0, 0, 0.06)).Border(1, t.Border)
			}

			if !disabled && tBtn.Clicked() {
				if _, err := provider.AddTemplate(curTmpl.Kind, curTmpl.Name); err == nil {
					a.setRoutingNotice("已从模板创建分组："+curTmpl.Name, false)
					a.Reload()
				} else {
					a.setRoutingNotice(err.Error(), true)
				}
			}

			tBtn.Children(func() {
				color := t.Text
				if disabled {
					color = t.TextMuted
				}
				ui.Text(c, curTmpl.Name).FontSize(11).TextColor(color).SingleLine()
			})
		}
	})
}

// routingGroupsSection renders the list of routing group cards.
func (a *App) routingGroupsSection(c *ui.Context, t *ui.Theme, rd RoutingData) {
	ui.Column(c).FillWidth().Gap(10).Children(func() {
		// Section Header
		ui.Row(c).AlignItems(ui.Center).FillWidth().Children(func() {
			ui.Text(c, fmt.Sprintf("路由分组 (%d)", len(rd.Groups))).
				FontSize(14).FontWeight(600).TextColor(t.Text).SingleLine()
			ui.Spacer(c)
			ui.Text(c, "Agent 将按此顺序列出路由组 · 可使用 ↑/↓ 调整排序").
				FontSize(11).TextColor(t.TextMuted).SingleLine()
		})

		// Groups List
		if len(rd.Groups) == 0 {
			ui.Row(c).FillWidth().Padding(30, 0).Center().Radius(10).
				Background(a.fieldBg(t)).Border(1, t.Border).Children(func() {
				ui.Text(c, "暂无配置的路由分组，点击上方「+ 新建分组」或选择模板添加").
					FontSize(12).TextColor(t.TextMuted).SingleLine()
			})
		} else {
			for idx, g := range rd.Groups {
				a.routingGroupCard(c, t, g, idx, len(rd.Groups))
			}
		}
	})
}

// routingGroupCard renders an individual group card with full actions.
func (a *App) routingGroupCard(c *ui.Context, t *ui.Theme, g RoutingGroupRow, idx, total int) {
	card := ui.Column(c).FillWidth().Padding(12, 16).Radius(10).
		Background(a.fieldBg(t)).Border(1, t.Border).Gap(10)

	card.Children(func() {
		// Top row: Switch, Name, Slug ID badge, Strategy pill, Affinity pill, Actions
		ui.Row(c).AlignItems(ui.Center).Gap(8).FillWidth().Children(func() {
			// On/Off toggle switch
			sw := ui.Row(c).Width(32).Height(18).Radius(9).
				AlignItems(ui.Center).Padding(0, 2).
				Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).Label("启用/停用")
			if !g.Disabled {
				sw.Background(ui.Hex("#34c759"))
			} else {
				sw.Background(ui.Hex("#8e8e93"))
			}
			if sw.Clicked() {
				_ = routingSwitch(g.ID, g.Disabled)
				a.Reload()
			}
			sw.Children(func() {
				if !g.Disabled {
					ui.Spacer(c)
				}
				ui.Box(c).Width(14).Height(14).Radius(7).Background(ui.Hex("#ffffff"))
				if g.Disabled {
					ui.Spacer(c)
				}
			})

			// Group Title
			ui.Text(c, g.Name).FontSize(13.5).FontWeight(600).TextColor(t.Text).SingleLine()

			// ID chip (clickable to copy)
			idChip := ui.Row(c).AlignItems(ui.Center).Gap(4).Padding(2, 6).Radius(4).
				Background(ui.RGBA(0, 0, 0, 0.05)).Cursor(ui.CursorPointer).Role(ui.RoleButton)
			if idChip.Clicked() {
				a.copyToClipboard("group/" + g.ID)
				a.setRoutingNotice("已复制标识 group/"+g.ID, false)
			}
			idChip.Children(func() {
				ui.Text(c, "group/"+g.ID).FontSize(10.5).Font("monospace").TextColor(t.TextMuted).SingleLine()
			})

			// Auto group tag
			if g.Auto {
				ui.Box(c).Padding(2, 6).Radius(4).Background(ui.RGBA(0, 122, 255, 0.12)).Children(func() {
					ui.Text(c, "自动发现").FontSize(10).TextColor(t.Accent).SingleLine()
				})
			}

			// Strategy badge
			ui.Box(c).Padding(2, 6).Radius(4).Background(ui.RGBA(0, 0, 0, 0.05)).Children(func() {
				ui.Text(c, rtStrategyLabel(g.Routing)).FontSize(10.5).FontWeight(500).TextColor(t.Text).SingleLine()
			})

			// Affinity badge
			if g.Affinity != "" {
				ui.Box(c).Padding(2, 6).Radius(4).Background(ui.RGBA(0, 0, 0, 0.05)).Children(func() {
					ui.Text(c, rtAffinityLabel(g.Affinity)).FontSize(10.5).TextColor(t.TextMuted).SingleLine()
				})
			}

			ui.Spacer(c)

			// Right-pinned action buttons (Fixed to the far right edge)
			ui.Row(c).AlignItems(ui.Center).Gap(6).Shrink(0).Children(func() {
				// Reorder buttons
				if idx > 0 {
					upBtn := ui.Row(c).Padding(2, 6).Radius(4).Background(ui.RGBA(0, 0, 0, 0.04)).
						Cursor(ui.CursorPointer).Role(ui.RoleButton).Label("上移")
					if upBtn.Clicked() {
						a.rtMoveGroup(idx, -1)
					}
					upBtn.Children(func() {
						ui.Text(c, "↑").FontSize(11).TextColor(t.TextMuted).SingleLine()
					})
				}
				if idx < total-1 {
					downBtn := ui.Row(c).Padding(2, 6).Radius(4).Background(ui.RGBA(0, 0, 0, 0.04)).
						Cursor(ui.CursorPointer).Role(ui.RoleButton).Label("下移")
					if downBtn.Clicked() {
						a.rtMoveGroup(idx, 1)
					}
					downBtn.Children(func() {
						ui.Text(c, "↓").FontSize(11).TextColor(t.TextMuted).SingleLine()
					})
				}

				// Edit button
				editBtn := ui.Row(c).Padding(3, 8).Radius(5).Background(ui.RGBA(0, 0, 0, 0.05)).
					Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).Label("编辑")
				if editBtn.Clicked() {
					a.openEditGroupModal(g)
				}
				editBtn.Children(func() {
					ui.Text(c, "编辑").FontSize(11).TextColor(t.Text).SingleLine()
				})

				// Copy button
				copyBtn := ui.Row(c).Padding(3, 8).Radius(5).Background(ui.RGBA(0, 0, 0, 0.05)).
					Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).Label("复制")
				if copyBtn.Clicked() {
					_, _ = provider.CopyGroup(g.ID, g.Name+" (副本)")
					a.setRoutingNotice("已复制分组："+g.Name, false)
					a.Reload()
				}
				copyBtn.Children(func() {
					ui.Text(c, "复制").FontSize(11).TextColor(t.Text).SingleLine()
				})

				// Share button
				shareBtn := ui.Row(c).Padding(3, 8).Radius(5).Background(ui.RGBA(0, 0, 0, 0.05)).
					Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).Label("分享")
				if shareBtn.Clicked() {
					if text, err := provider.ShareGroup(g.ID); err == nil {
						a.copyToClipboard(text)
						a.setRoutingNotice("已复制分组分享文本到剪贴板", false)
					}
				}
				shareBtn.Children(func() {
					ui.Text(c, "分享").FontSize(11).TextColor(t.Text).SingleLine()
				})

				// Delete button
				isDeleting := a.rtDeleteGroup == g.ID
				delBtn := ui.Row(c).Padding(3, 8).Radius(5).
					Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).Label("删除")
				if isDeleting {
					delBtn.Background(ui.Hex("#ff3b30"))
				} else {
					delBtn.Background(ui.RGBA(255, 59, 48, 0.08))
				}
				if delBtn.Clicked() {
					if isDeleting {
						_ = routingDelete(g.ID)
						a.rtDeleteGroup = ""
						a.setRoutingNotice("已删除分组："+g.Name, false)
						a.Reload()
					} else {
						a.rtDeleteGroup = g.ID
						c.Invalidate()
					}
				}
				delBtn.Children(func() {
					dtc := ui.Hex("#ff3b30")
					delLabel := "删除"
					if isDeleting {
						dtc = ui.Hex("#ffffff")
						delLabel = "确认删除?"
					}
					ui.Text(c, delLabel).FontSize(11).TextColor(dtc).SingleLine()
				})
			})
		})

		// Bottom row: Info badges (Classifier, Rules, Context, Patterns)
		var badges []string
		if g.Classifier != "" {
			badges = append(badges, "分类器: "+g.Classifier)
		}
		if g.RulesCount > 0 {
			badges = append(badges, fmt.Sprintf("%d 条条件规则", g.RulesCount))
		}
		if g.Context > 0 {
			badges = append(badges, fmt.Sprintf("上下文: %d Tokens", g.Context))
		}
		if len(g.Patterns) > 0 {
			badges = append(badges, "匹配: "+strings.Join(g.Patterns, ", "))
		}

		if len(badges) > 0 || g.ClassifierErr != "" {
			ui.Row(c).AlignItems(ui.Center).Gap(8).FillWidth().Children(func() {
				for _, b := range badges {
					ui.Text(c, b).FontSize(10.5).TextColor(t.TextMuted).SingleLine()
					ui.Text(c, "·").FontSize(10).TextColor(t.TextMuted).SingleLine()
				}
				if g.ClassifierErr != "" {
					ui.Text(c, "分类器异常: "+g.ClassifierErr).FontSize(10.5).TextColor(ui.Hex("#ff9500")).SingleLine()
				}
			})
		}
	})
}

// rtMoveGroup changes group ordering and persists it.
func (a *App) rtMoveGroup(idx, delta int) {
	groups := a.data.Routing.Groups
	target := idx + delta
	if target < 0 || target >= len(groups) {
		return
	}
	var ids []string
	for _, g := range groups {
		ids = append(ids, g.ID)
	}
	ids[idx], ids[target] = ids[target], ids[idx]
	_ = provider.SetGroupOrder(ids)
	a.Reload()
}

// routingPoolsSection renders the Account & Key Pools configured across providers.
func (a *App) routingPoolsSection(c *ui.Context, t *ui.Theme, pools []RoutingPoolRow) {
	ui.Column(c).FillWidth().Gap(10).Children(func() {
		ui.Row(c).AlignItems(ui.Center).FillWidth().Children(func() {
			ui.Text(c, fmt.Sprintf("多账号与密钥池 (%d)", len(pools))).
				FontSize(14).FontWeight(600).TextColor(t.Text).SingleLine()
			ui.Spacer(c)
			ui.Text(c, "Echo 在同一提供商的多个凭据间平滑调度").
				FontSize(11).TextColor(t.TextMuted).SingleLine()
		})

		ui.Row(c).FillWidth().Gap(10).Wrap().Children(func() {
			for _, pool := range pools {
				curPool := pool
				pCard := ui.Row(c).AlignItems(ui.Center).Gap(8).Padding(8, 12).Radius(8).
					Background(a.fieldBg(t)).Border(1, t.Border)

				pCard.Children(func() {
					pic := iconPictureFor(curPool.Icon)
					if pic != nil {
						ui.Image(c, pic).Size(16, 16).Radius(3)
					}

					ui.Column(c).Gap(1).Children(func() {
						ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
							ui.Text(c, curPool.ProviderName).FontSize(12).FontWeight(600).TextColor(t.Text).SingleLine()
							kindLabel := "多账号"
							if curPool.Kind == "key" {
								kindLabel = "多密钥"
							}
							ui.Box(c).Padding(1, 5).Radius(3).Background(ui.RGBA(0, 0, 0, 0.05)).Children(func() {
								ui.Text(c, kindLabel).FontSize(10).TextColor(t.TextMuted).SingleLine()
							})
						})

						// Render who
						var maskedWho []string
						for _, w := range curPool.Who {
							maskedWho = append(maskedWho, rtMaskWho(w, a.rtHideAccounts))
						}
						ui.Text(c, fmt.Sprintf("%d 个凭据: %s", len(curPool.Who), strings.Join(maskedWho, ", "))).
							FontSize(10.5).TextColor(t.TextMuted).SingleLine()
					})
				})
			}
		})
	})
}

// routingTraceSection renders live routed requests with filter tabs and details drilldown.
func (a *App) routingTraceSection(c *ui.Context, t *ui.Theme, rd RoutingData) {
	ui.Column(c).FillWidth().Gap(10).Children(func() {
		// Section Header & Filter Pills
		ui.Row(c).AlignItems(ui.Center).FillWidth().Children(func() {
			ui.Text(c, fmt.Sprintf("实时路由动态 (%d)", len(rd.Traces))).
				FontSize(14).FontWeight(600).TextColor(t.Text).SingleLine()
			ui.Spacer(c)

			// Filter segments: All, Succeeded, Rerouted, Errors
			filters := []struct {
				id    int
				label string
			}{
				{0, "全部"},
				{1, "成功"},
				{2, "重新路由"},
				{3, "失败/报错"},
			}

			ui.Row(c).AlignItems(ui.Center).Gap(4).Children(func() {
				for _, flt := range filters {
					active := a.rtTraceFilter == flt.id
					fPill := ui.Row(c).Padding(3, 8).Radius(5).
						Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton)
					if active {
						fPill.Background(t.Accent)
					} else {
						fPill.Background(ui.RGBA(0, 0, 0, 0.04))
					}
					if fPill.Clicked() {
						a.rtTraceFilter = flt.id
						c.Invalidate()
					}
					fPill.Children(func() {
						fc := t.Text
						if active {
							fc = ui.Hex("#ffffff")
						}
						ui.Text(c, flt.label).FontSize(11).TextColor(fc).SingleLine()
					})
				}
			})
		})

		// Filter traces
		var filtered []RoutingTraceRow
		for _, tr := range rd.Traces {
			switch a.rtTraceFilter {
			case 1: // Succeeded
				if tr.Status == 200 || (tr.Status < 400 && tr.Error == "") {
					filtered = append(filtered, tr)
				}
			case 2: // Rerouted
				if len(tr.Tries) > 1 {
					filtered = append(filtered, tr)
				}
			case 3: // Errors
				if tr.Error != "" || tr.Status >= 400 {
					filtered = append(filtered, tr)
				}
			default:
				filtered = append(filtered, tr)
			}
		}

		if len(filtered) == 0 {
			ui.Row(c).FillWidth().Padding(30, 0).Center().Radius(10).
				Background(a.fieldBg(t)).Border(1, t.Border).Children(func() {
				ui.Text(c, "暂无符合条件的实时路由记录").
					FontSize(12).TextColor(t.TextMuted).SingleLine()
			})
		} else {
			ui.Column(c).FillWidth().Gap(6).Children(func() {
				for _, tr := range filtered {
					curTR := tr
					a.routingTraceRowView(c, t, curTR)
				}
			})
		}
	})
}

// routingTraceRowView renders one routed request row.
func (a *App) routingTraceRowView(c *ui.Context, t *ui.Theme, tr RoutingTraceRow) {
	row := ui.Row(c).AlignItems(ui.Center).Gap(10).Padding(8, 12).Radius(8).
		Background(a.fieldBg(t)).Border(1, t.Border).FillWidth().
		Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton)

	if row.Clicked() {
		a.rtTraceDetail = &tr
		c.Invalidate()
	}

	row.Children(func() {
		// Status Badge
		statusC := ui.Hex("#34c759")
		statusText := strconv.Itoa(tr.Status)
		if tr.Status >= 400 || tr.Error != "" {
			statusC = ui.Hex("#ff3b30")
			if statusText == "0" {
				statusText = "FAIL"
			}
		} else if len(tr.Tries) > 1 {
			statusC = ui.Hex("#ff9500")
			statusText = "RE"
		} else if statusText == "0" {
			statusText = "200"
		}

		ui.Box(c).Width(38).Height(20).Radius(4).Background(ui.RGBA(0, 0, 0, 0.05)).Center().Children(func() {
			ui.Text(c, statusText).FontSize(10).FontWeight(600).TextColor(statusC).SingleLine()
		})

		// Time
		ui.Text(c, rtTimeWhen(tr.Time)).FontSize(11).TextColor(t.TextMuted).Width(60).Shrink(0).SingleLine()

		// Agent pill
		ui.Box(c).Padding(2, 6).Radius(4).Background(ui.RGBA(0, 0, 0, 0.05)).Shrink(0).Children(func() {
			agentName := tr.Agent
			if agentName == "" {
				agentName = "agent"
			}
			ui.Text(c, agentName).FontSize(10.5).FontWeight(500).TextColor(t.Text).SingleLine()
		})

		// Model route path
		ui.Column(c).Grow(1).Shrink(1).Gap(1).Children(func() {
			ui.Row(c).AlignItems(ui.Center).Gap(4).Children(func() {
				ui.Text(c, tr.RequestedModel).FontSize(12).FontWeight(500).TextColor(t.Text).SingleLine()
				if tr.ServedModel != "" && tr.ServedModel != tr.RequestedModel {
					ui.Text(c, "→").FontSize(11).TextColor(t.TextMuted).SingleLine()
					ui.Text(c, tr.ServedModel).FontSize(12).FontWeight(600).TextColor(t.Accent).SingleLine()
				}
			})

			// Decision & Session Title
			subLine := tr.Decision
			if tr.SessionTitle != "" {
				subLine += " · " + tr.SessionTitle
			}
			if tr.Error != "" {
				subLine = "异常: " + tr.Error
			}
			ui.Text(c, subLine).FontSize(10.5).TextColor(t.TextMuted).SingleLine()
		})

		// Timings & Tokens (Fixed right aligned)
		ui.Row(c).AlignItems(ui.Center).Gap(12).Shrink(0).Children(func() {
			if tr.TTFT > 0 {
				ui.Text(c, fmt.Sprintf("TTFT %dms", tr.TTFT)).FontSize(11).TextColor(t.TextMuted).SingleLine()
			}
			ui.Text(c, fmt.Sprintf("%dms", tr.Millis)).FontSize(11).TextColor(t.Text).SingleLine()

			if tr.Tokens > 0 {
				ui.Text(c, fmt.Sprintf("%d tok", tr.Tokens)).FontSize(11).TextColor(t.TextMuted).SingleLine()
			}
			if tr.Cost > 0 {
				ui.Text(c, fmt.Sprintf("$%.4f", tr.Cost)).FontSize(11).TextColor(ui.Hex("#34c759")).SingleLine()
			}

			// Details arrow
			ui.Text(c, "明细 ↗").FontSize(11).TextColor(t.Accent).SingleLine()
		})
	})
}

// rtTimeWhen returns human-friendly relative time for trace events.
func rtTimeWhen(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < 5*time.Second:
		return "刚刚"
	case d < time.Minute:
		return fmt.Sprintf("%d秒前", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%d分前", int(d.Minutes()))
	default:
		return t.Format("15:04:05")
	}
}
