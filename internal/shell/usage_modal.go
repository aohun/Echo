//go:build nogui && echo

package shell

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/yetone/magpie/internal/provider"
	"github.com/yetone/magpie/internal/sessions"
	"github.com/yetone/magpie/internal/usageapi"
)

// usageOverlay draws modals and dialogs over the Usage page.
func (a *App) usageOverlay(c *ui.Context) {
	if a.usInspectRow != nil {
		open := true
		ui.Modal(c, &open, func() { a.usageInspectModal(c) })
		if !open {
			a.usCloseInspect()
		}
	}
	if a.usSessOpen != "" {
		open := true
		ui.Modal(c, &open, func() { a.usageSessionModal(c) })
		if !open {
			a.usCloseSession()
		}
	}
	if a.usResetConfirmUser != "" {
		open := true
		ui.Modal(c, &open, func() { a.usageResetModal(c) })
		if !open {
			a.usResetConfirmUser = ""
		}
	}
}

// usCloseInspect closes the request content inspection modal.
func (a *App) usCloseInspect() {
	a.usInspectRow = nil
	a.usInspectContent = nil
	a.usInspectLoading = false
}

// usageInspectModal displays full prompt input and output reply for a request row.
func (a *App) usageInspectModal(c *ui.Context) {
	t := c.Theme()
	row := a.usInspectRow
	if row == nil {
		return
	}

	ui.Column(c).Width(760).MaxHeight(620).Radius(12).
		Background(t.Background).Border(1, t.Border).Children(func() {
		// 1. Header
		ui.Row(c).AlignItems(ui.Center).FillWidth().Padding(14, 18).Children(func() {
			ui.Column(c).Grow(1).Shrink(1).Gap(3).Children(func() {
				ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
					// Status badge
					statusBg := ui.RGBA(52, 199, 89, 0.15)
					statusText := ui.Hex("#34c759")
					statusLabel := "200 OK"
					if row.Failed() || row.Status >= 400 {
						statusBg = ui.RGBA(255, 59, 48, 0.15)
						statusText = ui.Hex("#ff3b30")
						statusLabel = fmt.Sprintf("%d 失败", row.Status)
					}
					ui.Row(c).Padding(2, 6).Radius(4).Background(statusBg).Children(func() {
						ui.Text(c, statusLabel).FontSize(11).FontWeight(600).TextColor(statusText).SingleLine()
					})

					// Model
					modelLabel := row.Model
					if modelLabel == "" {
						modelLabel = row.Requested
					}
					ui.Text(c, modelLabel).FontSize(14).FontWeight(700).TextColor(t.Text).SingleLine()

					// Agent pill
					if row.AgentName != "" {
						ui.Row(c).AlignItems(ui.Center).Gap(4).Padding(2, 6).Radius(4).
							Background(a.fieldBg(t)).Children(func() {
							ui.Text(c, row.AgentName).FontSize(11).TextColor(t.TextMuted).SingleLine()
						})
					}
				})

				// Subtitle metadata
				ui.Row(c).AlignItems(ui.Center).Gap(10).Children(func() {
					ui.Text(c, row.Time.Format("2006-01-02 15:04:05")).FontSize(11).TextColor(t.TextMuted).SingleLine()
					if row.Millis > 0 {
						ui.Text(c, "·").FontSize(11).TextColor(t.TextMuted).SingleLine()
						ui.Text(c, fmt.Sprintf("耗时: %s", FormatDuration(row.Millis))).FontSize(11).TextColor(t.TextMuted).SingleLine()
					}
					totTokens := row.Input + row.Output + row.CacheRead + row.CacheWrite
					if totTokens > 0 {
						ui.Text(c, "·").FontSize(11).TextColor(t.TextMuted).SingleLine()
						ui.Text(c, fmt.Sprintf("Tokens: %s", FormatTokens(totTokens))).FontSize(11).TextColor(t.TextMuted).SingleLine()
					}
					if row.Cost > 0 {
						ui.Text(c, "·").FontSize(11).TextColor(t.TextMuted).SingleLine()
						ui.Text(c, fmt.Sprintf("费用: %s", FormatCost(row.Cost))).FontSize(11).TextColor(ui.Hex("#34c759")).SingleLine()
					}
				})
			})

			// Close button
			closeBtn := ui.Row(c).Padding(4, 8).Radius(6).Background(ui.RGBA(0, 0, 0, 0.05)).
				Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).Label("关闭")
			if closeBtn.Clicked() {
				a.usCloseInspect()
			}
			closeBtn.Children(func() {
				ui.Text(c, "✕").FontSize(12).TextColor(t.TextMuted).SingleLine()
			})
		})

		a.skSheetRule(c, t)

		// 2. Scrollable Body
		ui.Scroll(c).Grow(1).Shrink(1).FillWidth().Padding(16, 20).Children(func() {
			ui.Column(c).FillWidth().Gap(14).Children(func() {
				// Loading indicator
				if a.usInspectLoading {
					ui.Row(c).FillWidth().Padding(30, 0).AlignItems(ui.Center).Children(func() {
						ui.Spacer(c)
						ui.Spinner(c).Size(16, 16)
						ui.Text(c, "正在检索本地会话记录与提示词明细...").FontSize(12.5).TextColor(t.TextMuted).SingleLine()
						ui.Spacer(c)
					})
					return
				}

				// Basic request summary card
				ui.Row(c).FillWidth().Gap(10).Children(func() {
					// Left: Token Breakdown
					ui.Column(c).Grow(1).Shrink(1).Padding(10, 12).Radius(8).
						Background(a.fieldBg(t)).Border(1, t.Border).Gap(4).Children(func() {
						ui.Text(c, "Token 明细").FontSize(11).TextColor(t.TextMuted).SingleLine()
						ui.Row(c).Gap(12).Children(func() {
							ui.Text(c, fmt.Sprintf("输入: %s", FormatTokens(row.Input))).FontSize(11.5).TextColor(t.Text).SingleLine()
							ui.Text(c, fmt.Sprintf("输出: %s", FormatTokens(row.Output))).FontSize(11.5).TextColor(t.Text).SingleLine()
							if row.CacheRead > 0 {
								ui.Text(c, fmt.Sprintf("缓存命中: %s", FormatTokens(row.CacheRead))).FontSize(11.5).TextColor(ui.Hex("#34c759")).SingleLine()
							}
						})
					})

					// Right: Provider & Session
					ui.Column(c).Width(280).Shrink(0).Padding(10, 12).Radius(8).
						Background(a.fieldBg(t)).Border(1, t.Border).Gap(4).Children(func() {
						ui.Text(c, "渠道与会话").FontSize(11).TextColor(t.TextMuted).SingleLine()
						provStr := row.ProviderName
						if provStr == "" {
							provStr = row.Provider
						}
						ui.Text(c, fmt.Sprintf("渠道: %s", provStr)).FontSize(11.5).TextColor(t.Text).SingleLine()
						if row.Session != "" {
							sid := row.Session
							if len(sid) > 24 {
								sid = sid[:10] + "..." + sid[len(sid)-10:]
							}
							ui.Text(c, fmt.Sprintf("会话: %s", sid)).FontSize(11).TextColor(t.TextMuted).SingleLine()
						}
					})
				})

				// Error info if any
				if row.Error != "" {
					ui.Column(c).FillWidth().Padding(10, 12).Radius(8).
						Background(ui.RGBA(255, 59, 48, 0.08)).Border(1, ui.RGBA(255, 59, 48, 0.25)).Gap(4).Children(func() {
						ui.Text(c, "错误信息").FontSize(11).FontWeight(600).TextColor(ui.Hex("#ff3b30")).SingleLine()
						ui.Text(c, row.Error).FontSize(11.5).TextColor(ui.Hex("#ff3b30"))
					})
				}

				// Content inspection results
				cnt := a.usInspectContent
				if cnt == nil || !cnt.Found {
					whyMsg := "该请求未在本地会话文件中找到完整内容"
					if cnt != nil {
						switch cnt.Why {
						case "session":
							whyMsg = "该请求未携带 session ID，仅网关记录了路由元数据"
						case "agent":
							whyMsg = "当前 Agent 暂未提供本地对话历史解析（支持 Claude Code, Claude Desktop, Codex）"
						case "missing":
							whyMsg = "本地会话日志中未检索到对应调用"
						case "read":
							whyMsg = "会话文件读取异常"
						}
					}
					ui.Column(c).FillWidth().Padding(20).Radius(8).
						Background(a.fieldBg(t)).Border(1, t.Border).AlignItems(ui.Center).Gap(6).Children(func() {
						ui.Text(c, "💬 未能提取完整对话内容").FontSize(12.5).FontWeight(600).TextColor(t.Text).SingleLine()
						ui.Text(c, whyMsg).FontSize(11.5).TextColor(t.TextMuted).SingleLine()
					})
					return
				}

				// Render Prompt Inputs
				if len(cnt.Input) > 0 {
					ui.Column(c).FillWidth().Gap(8).Children(func() {
						ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
							ui.Text(c, fmt.Sprintf("📥 用户输入 / 提示词 (%d 项)", len(cnt.Input))).
								FontSize(12.5).FontWeight(600).TextColor(t.Text).SingleLine()
						})

						for _, p := range cnt.Input {
							a.usRenderContentPart(c, t, p, true)
						}
					})
				}

				// Render Replies / Output
				if len(cnt.Output) > 0 {
					ui.Column(c).FillWidth().Gap(8).Children(func() {
						ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
							ui.Text(c, fmt.Sprintf("📤 模型回复 / 输出 (%d 项)", len(cnt.Output))).
								FontSize(12.5).FontWeight(600).TextColor(t.Text).SingleLine()
						})

						for _, p := range cnt.Output {
							a.usRenderContentPart(c, t, p, false)
						}
					})
				}
			})
		})
	})
}

// usRenderContentPart renders a single Part of input or output with fixed right-aligned copy button.
func (a *App) usRenderContentPart(c *ui.Context, t *ui.Theme, p sessions.Part, isInput bool) {
	tagBg := ui.RGBA(0, 122, 255, 0.1)
	tagText := ui.Hex("#007aff")
	tagLabel := "输入文本"

	if !isInput {
		tagBg = ui.RGBA(52, 199, 89, 0.1)
		tagText = ui.Hex("#34c759")
		tagLabel = "模型回复"
	}

	switch p.Kind {
	case "thinking":
		tagBg = ui.RGBA(175, 82, 222, 0.1)
		tagText = ui.Hex("#af52de")
		tagLabel = "思考过程 (Thinking)"
	case "tool_use":
		tagBg = ui.RGBA(255, 149, 0, 0.1)
		tagText = ui.Hex("#ff9500")
		tagLabel = "工具调用: " + p.Name
	case "tool_result":
		tagBg = ui.RGBA(90, 200, 250, 0.1)
		tagText = ui.Hex("#5ac8fa")
		tagLabel = "工具返回: " + p.Name
	case "context":
		tagLabel = "上下文提示"
	case "image":
		tagLabel = "图像输入"
	}

	ui.Column(c).FillWidth().Radius(8).Background(a.fieldBg(t)).Border(1, t.Border).Children(func() {
		// Header bar with badge and right-aligned copy button
		ui.Row(c).AlignItems(ui.Center).FillWidth().Padding(6, 12).Children(func() {
			ui.Row(c).Padding(2, 6).Radius(4).Background(tagBg).Children(func() {
				ui.Text(c, tagLabel).FontSize(10.5).FontWeight(600).TextColor(tagText).SingleLine()
			})

			if p.Model != "" {
				ui.Text(c, p.Model).FontSize(10.5).TextColor(t.TextMuted).SingleLine()
			}

			ui.Spacer(c)

			// Fixed right-aligned Copy button
			copyBtn := ui.Row(c).Padding(3, 8).Radius(4).Background(ui.RGBA(0, 0, 0, 0.05)).
				Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).Label("复制").Shrink(0)
			if copyBtn.Clicked() {
				a.copyToClipboard(p.Text)
				a.usNotice = "已复制内容至剪贴板"
				a.usNoticeErr = false
				c.Invalidate()
			}
			copyBtn.Children(func() {
				ui.Text(c, "复制").FontSize(10.5).TextColor(t.TextMuted).SingleLine()
			})
		})

		// Body text
		ui.Column(c).FillWidth().Padding(8, 12).Children(func() {
			textStr := p.Text
			if len(textStr) > 4000 {
				textStr = textStr[:4000] + "\n\n... (内容过长已截断)"
			}
			ui.Text(c, textStr).FontSize(11.5).TextColor(t.Text)
		})
	})
}

// usageResetModal confirms spending 1 rate limit reset for a Codex account.
func (a *App) usageResetModal(c *ui.Context) {
	t := c.Theme()
	user := a.usResetConfirmUser
	if user == "" {
		return
	}

	ui.Column(c).Width(420).Radius(10).Background(t.Background).Border(1, t.Border).
		Padding(18, 20).Gap(14).Children(func() {
		ui.Text(c, "消耗重置额度").FontSize(14).FontWeight(700).TextColor(t.Text).SingleLine()

		ui.Text(c, fmt.Sprintf("确定要为账号 %s 消耗 1 次 Codex 限频重置额度吗？重置后该账号速率限制窗口将立即重新计算。", user)).
			FontSize(12).TextColor(t.TextMuted)

		ui.Row(c).AlignItems(ui.Center).Gap(10).FillWidth().Children(func() {
			ui.Spacer(c)
			// Cancel
			cancelBtn := ui.Row(c).Padding(6, 12).Radius(6).Background(a.fieldBg(t)).Border(1, t.Border).
				Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).Label("取消")
			if cancelBtn.Clicked() {
				a.usResetConfirmUser = ""
			}
			cancelBtn.Children(func() {
				ui.Text(c, "取消").FontSize(11.5).TextColor(t.Text).SingleLine()
			})

			// Confirm
			confirmBtn := ui.Row(c).Padding(6, 14).Radius(6).Background(t.Accent).
				Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).Label("确认重置")
			if confirmBtn.Clicked() {
				a.usExecuteCodexReset(user)
				a.usResetConfirmUser = ""
			}
			confirmBtn.Children(func() {
				ui.Text(c, "确认重置").FontSize(11.5).FontWeight(600).TextColor(ui.Hex("#ffffff")).SingleLine()
			})
		})
	})
}

// usExecuteCodexReset executes the Codex reset action.
func (a *App) usExecuteCodexReset(user string) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	out, err := provider.UseCodexReset(ctx, user)
	if err != nil {
		a.usNotice = "重置额度失败: " + err.Error()
		a.usNoticeErr = true
		return
	}

	a.usNotice = fmt.Sprintf("Codex 额度重置成功 (状态: %s)", out.Code)
	a.usNoticeErr = false
	a.data.Usage.Quotas = ReadQuotas(true)
}

// usRunCheckin executes the daily checkin action for the specified provider.
func (a *App) usRunCheckin(providerID string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var rs []provider.WorkBuddyCheckin
	name := "每日签到"

	switch providerID {
	case "workbuddy":
		name = "WorkBuddy"
		rs = provider.CheckInWorkBuddy(ctx)
	case "trae":
		name = "Trae CN"
		rs = provider.CheckInTrae(ctx)
	case "minimax":
		name = "MiniMax"
		rs = provider.CheckInMiniMax(ctx)
	case "qoder":
		name = "Qoder"
		rs = provider.CheckInQoder(ctx)
	default:
		name = providerID
		rs = provider.CheckInPlugins(ctx, providerID)
	}

	if len(rs) == 0 {
		a.usNotice = fmt.Sprintf("%s 暂无需要签到的账号", name)
		a.usNoticeErr = false
	} else {
		successCount := 0
		for _, r := range rs {
			if r.Checked() {
				successCount++
			}
		}
		a.usNotice = fmt.Sprintf("%s 签到完成：%d / %d 个账号已签到成功", name, successCount, len(rs))
		a.usNoticeErr = (successCount == 0 && len(rs) > 0)
	}

	a.data.Usage.Quotas = ReadQuotas(true)
}

// usInspectRequest loads request content and opens the inspect modal.
func (a *App) usInspectRequest(r usageapi.Row) {
	a.usInspectRow = &r
	a.usInspectLoading = true
	a.usInspectContent = nil

	go func() {
		from := r.Time
		to := r.Time.Add(time.Duration(max(1000, r.Millis)) * time.Millisecond)
		at := to
		content := ReadRequestContent(r.Agent, r.Session, from, to, at)
		a.usInspectContent = &content
		a.usInspectLoading = false
	}()
}

// usOpenSession reads a session's own file and its conversation, and opens
// the sheet that shows them.
func (a *App) usOpenSession(s sessions.Summary) {
	key := s.Key
	a.usSessOpen = key
	a.usSessOne = nil
	a.usSessTrans = nil
	a.usSessErr = ""
	a.usSessLoading = true

	go func() {
		one, ok := sessions.Get(key)
		var trans sessions.Transcript
		var err error
		switch {
		case !ok:
			err = fmt.Errorf("找不到这个会话的文件")
		case !one.Transcript:
			err = sessions.ErrNoTranscript
		default:
			trans, err = sessions.TranscriptOf(one)
		}
		apply := func() {
			if a.usSessOpen != key {
				return // another session was opened since
			}
			a.usSessLoading = false
			a.usSessOne = &one
			if err != nil {
				a.usSessErr = err.Error()
				return
			}
			if len(trans.Parts) > 0 {
				a.usSessTrans = &trans
			}
		}
		if a.win == nil {
			apply()
			return
		}
		a.win.Update(apply)
	}()
}

// usCloseSession puts the session's sheet away.
func (a *App) usCloseSession() {
	a.usSessOpen = ""
	a.usSessOne = nil
	a.usSessTrans = nil
	a.usSessErr = ""
	a.usSessLoading = false
}

// usageSessionModal is one session: what it was, what it spent, the command
// that picks it up again, and what was said in it.
func (a *App) usageSessionModal(c *ui.Context) {
	t := c.Theme()
	one := a.usSessOne

	ui.Column(c).Width(820).MaxHeight(640).Radius(12).
		Background(t.Background).Border(1, t.Border).Children(func() {
		// the head: who, what it was about, and the way out
		ui.Row(c).AlignItems(ui.Center).FillWidth().Padding(14, 18).Children(func() {
			ui.Image(c, iconPictureFor(usAgentIcon(sessionAgentOf(one)))).Size(26, 26).Radius(6)
			ui.Column(c).Grow(1).Shrink(1).Gap(3).Children(func() {
				ui.Text(c, sessionTitleOf(one)).FontSize(14).FontWeight(700).TextColor(t.Text).SingleLine()
				ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
					ui.Text(c, usAgentName(sessionAgentOf(one))).FontSize(11).TextColor(t.TextMuted).SingleLine()
					if one != nil && one.Cwd != "" {
						ui.Text(c, one.Cwd).FontSize(10.5).TextColor(t.TextMuted).Shrink(1).SingleLine().
							Tooltip(one.Cwd)
					}
					if one != nil && !one.Last.IsZero() {
						ui.Text(c, "最近 "+usAgo(one.Last)).FontSize(10.5).TextColor(t.TextMuted).SingleLine()
					}
				})
			})
			closeBtn := ui.Row(c).Padding(4, 8).Radius(6).Background(a.fieldBg(t)).
				Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).Label("关闭")
			if closeBtn.Clicked() {
				a.usCloseSession()
			}
			closeBtn.Children(func() {
				ui.Text(c, "✕").FontSize(12).TextColor(t.TextMuted).SingleLine()
			})
		})

		a.skSheetRule(c, t)

		ui.Scroll(c).Grow(1).Shrink(1).FillWidth().Padding(14, 18).Children(func() {
			ui.Column(c).FillWidth().Gap(12).Children(func() {
				// what it spent
				if one != nil {
					ui.Row(c).FillWidth().Gap(8).Children(func() {
						a.sessionFact(c, t, "Token", FormatCount(one.Input+one.Output),
							fmt.Sprintf("%s 入 · %s 出", FormatCount(one.Input), FormatCount(one.Output)))
						a.sessionFact(c, t, "缓存读取", FormatCount(one.CacheRead),
							fmt.Sprintf("%s 写入", FormatCount(one.CacheWrite)))
						a.sessionFact(c, t, "费用", sessionCostOf(one), "按有效模型价格估算")
						a.sessionFact(c, t, "模型", sessionModelsOf(one), "这个会话用过的模型")
					})
					// the command that picks it up again
					if one.Resume != "" {
						ui.Column(c).FillWidth().Padding(8, 10).Radius(8).
							Background(a.fieldBg(t)).Border(1, t.Border).Gap(3).Children(func() {
							ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
								ui.Text(c, "继续这个会话").FontSize(10).TextColor(t.TextMuted).SingleLine()
								ui.Spacer(c)
								copyBtn := ui.Row(c).Padding(2, 8).Radius(4).
									Background(a.fieldBg(t)).Border(1, t.Border).
									Cursor(ui.CursorPointer).Focusable().Role(ui.RoleButton).Label("复制命令")
								if copyBtn.Clicked() {
									a.copyToClipboard(one.Resume)
									a.usSetNotice("已复制继续这个会话的命令", false)
								}
								copyBtn.Children(func() {
									ui.Text(c, "复制").FontSize(10).TextColor(t.TextMuted).SingleLine()
								})
							})
							ui.Text(c, one.Resume).FontSize(11).TextColor(t.Text).SingleLine()
						})
					}
					if one.Path != "" {
						ui.Text(c, one.Path).FontSize(9.5).TextColor(t.TextMuted).SingleLine().Tooltip(one.Path)
					}
				}

				if a.usSessLoading {
					ui.Row(c).FillWidth().Padding(30, 0).AlignItems(ui.Center).Children(func() {
						ui.Spacer(c)
						ui.Spinner(c).Size(16, 16)
						ui.Text(c, "正在读取这个会话的对话…").FontSize(12.5).TextColor(t.TextMuted).SingleLine()
						ui.Spacer(c)
					})
					return
				}

				tr := a.usSessTrans
				if a.usSessErr != "" || tr == nil {
					why := a.usSessErr
					if why == "" {
						why = "这个 Agent 的对话没有留在可读的文件里。"
					}
					ui.Column(c).FillWidth().Padding(20).Radius(8).
						Background(a.fieldBg(t)).Border(1, t.Border).AlignItems(ui.Center).Gap(6).Children(func() {
						ui.Text(c, "看不到这个会话的对话").FontSize(12.5).FontWeight(600).TextColor(t.Text).SingleLine()
						ui.Text(c, why).FontSize(11.5).TextColor(t.TextMuted).SingleLine()
					})
					return
				}

				ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
					ui.Text(c, fmt.Sprintf("对话（%d 项）", len(tr.Parts))).FontSize(12.5).FontWeight(600).TextColor(t.Text).SingleLine()
					if tr.Cut {
						ui.Text(c, "· 只保留了前一部分").FontSize(10).TextColor(t.TextMuted).SingleLine()
					}
				})
				shown := min(len(tr.Parts), 300)
				for _, p := range tr.Parts[:shown] {
					a.usRenderContentPart(c, t, p, p.Role == "user")
				}
				if len(tr.Parts) > shown {
					ui.Text(c, fmt.Sprintf("还有 %d 项没有画出来", len(tr.Parts)-shown)).
						FontSize(10.5).TextColor(t.TextMuted).SingleLine()
				}
			})
		})
	})
}

// sessionFact is one figure of a session's sheet.
func (a *App) sessionFact(c *ui.Context, t *ui.Theme, label, value, note string) {
	box := ui.Column(c).Grow(1).Basis(0).Shrink(1).Padding(8, 10).Radius(8).
		Background(a.fieldBg(t)).Border(1, t.Border).Gap(1)
	box.Tooltip(note)
	box.Children(func() {
		ui.Text(c, label).FontSize(9.5).TextColor(t.TextMuted).SingleLine()
		ui.Text(c, value).FontSize(12).FontWeight(600).TextColor(t.Text).SingleLine()
	})
}

// sessionModelsOf lists a session's models, short.
func sessionModelsOf(s *sessions.Session) string {
	if len(s.Models) == 0 {
		return "—"
	}
	var names []string
	for i, m := range s.Models {
		if i >= 2 {
			break
		}
		names = append(names, m.Model)
	}
	return strings.Join(names, " · ")
}

// sessionAgentOf is a session's agent, "" when it isn't read yet.
func sessionAgentOf(s *sessions.Session) string {
	if s == nil {
		return ""
	}
	return s.Agent
}

// sessionTitleOf is a session's name, or its id's head.
func sessionTitleOf(s *sessions.Session) string {
	if s == nil {
		return "会话"
	}
	if s.Title != "" {
		return s.Title
	}
	if len(s.ID) > 8 {
		return s.ID[:8]
	}
	return s.ID
}

// sessionCostOf writes a session's cost.
func sessionCostOf(s *sessions.Session) string {
	if s.Cost <= 0 {
		return "—"
	}
	return "≈" + FormatCost(s.Cost)
}
