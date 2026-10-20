//go:build nogui && echo

package shell

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/yetone/magpie/internal/davsync"
	"github.com/yetone/magpie/internal/proc"
)

// settingsPickerState holds state for the model picker dialog when picking a
// model for settings (Codex titles, Codex auto-review, Vision, ImageGen, Searcher).
type settingsPickerState struct {
	kind   string // "codexTitles" | "codexAutoReview" | "vision" | "imageGen" | "searcher"
	title  string
	query  string
	group  string // selected group in rail ("" = all)
	value  string // current value
	onPick func(id string)
}

// openFolder opens a folder in Finder / Explorer / file manager.
func (a *App) openFolder(dir string) {
	if dir == "" {
		return
	}
	// resolve tilde if any
	if strings.HasPrefix(dir, "~/") {
		if home, err := os.UserHomeDir(); err == nil && home != "" {
			dir = filepath.Join(home, strings.TrimPrefix(dir, "~/"))
		}
	}
	var cmd string
	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
	case "windows":
		cmd = "explorer"
	default:
		cmd = "xdg-open"
	}
	_ = proc.Command(cmd, dir).Start()
}

// settingsOverlay draws the modals opened from Settings: the model picker,
// sync form, restore confirmation, export dialog, and import dialog.
func (a *App) settingsOverlay(c *ui.Context) {
	if a.setPicker != nil {
		a.settingsPickerOverlay(c)
	}
	if a.setSyncRestore {
		a.settingsRestoreOverlay(c)
	}
}

// settingsPickerOverlay renders the model picker sheet.
func (a *App) settingsPickerOverlay(c *ui.Context) {
	pk := a.setPicker
	if pk == nil {
		return
	}
	t := c.Theme()
	ui.Overlay(c, func() {
		back := ui.Box(c).Absolute().Left(0).Top(0).Right(0).Bottom(0).Center().
			Background(ui.RGBA(0, 0, 0, 0.4)).Modal()
		if back.OverlayShortcut(0, ui.KeyEscape) {
			a.setPicker = nil
			c.Invalidate()
			return
		}
		back.Children(func() {
			cw, ch := c.Size()
			dialog := ui.Column(c).Width(640).Height(520).
				MaxWidth(cw-40).MaxHeight(ch-40).
				Radius(12).Padding(16, 20).Gap(12).
				Role(ui.RoleDialog).Label(pk.title)
			if t.Dark {
				dialog.Background(ui.RGBA(36, 36, 40, 1))
			} else {
				dialog.Background(t.Background)
			}
			dialog.Border(1, t.Border).
				Shadow(0, 16, 40, 0, ui.RGBA(0, 0, 0, 0.35))

			dialog.Children(func() {
				// Header: title and search
				ui.Row(c).AlignItems(ui.Center).Gap(12).Children(func() {
					ui.Text(c, pk.title).FontSize(15).FontWeight(600).TextColor(t.Text).SingleLine()
					ui.Spacer(c)
					ui.SearchField(c, &pk.query).Width(220).Label(a.text.S.Set.FindModel)
					closeBtn := ui.Row(c).Center().Width(24).Height(24).Radius(6).
						Cursor(ui.CursorPointer).Role(ui.RoleButton)
					if closeBtn.Hovered() {
						closeBtn.Background(ui.RGBA(0, 0, 0, 0.06))
					}
					if closeBtn.Clicked() {
						a.setPicker = nil
						c.Invalidate()
					}
					closeBtn.Children(func() {
						ui.Text(c, "✕").FontSize(12).TextColor(t.TextMuted)
					})
				})

				// Body: list of model choices
				a.settingsPickerList(c, t, pk)
			})
		})
	})
}

// settingsPickerList lists the candidate models matching the query.
func (a *App) settingsPickerList(c *ui.Context, t *ui.Theme, pk *settingsPickerState) {
	models := a.settingsPickerModels(pk)
	q := strings.ToLower(strings.TrimSpace(pk.query))

	ui.Scroll(c).Grow(1).Shrink(1).Children(func() {
		ui.Column(c).FillWidth().Gap(2).Children(func() {
			// Special options: Automatic, Off, etc.
			switch pk.kind {
			case "vision":
				a.settingsPickerSpecialRow(c, t, pk, "", a.text.S.Set.Automatic, a.data.Settings.VisionAuto)
				a.settingsPickerSpecialRow(c, t, pk, "off", a.text.S.Set.Off, a.text.S.Set.VisionOffDesc)
			case "imageGen":
				a.settingsPickerSpecialRow(c, t, pk, "", a.text.S.Set.Automatic, a.data.Settings.ImageGenAuto)
				a.settingsPickerSpecialRow(c, t, pk, "off", a.text.S.Set.Off, a.text.S.Set.ImageGenOffDesc)
			case "searcher":
				a.settingsPickerSpecialRow(c, t, pk, "", a.text.S.Set.Automatic, a.data.Settings.SearchAuto)
			case "codexTitles":
				a.settingsPickerSpecialRow(c, t, pk, "", a.text.S.Set.CodexTitlesChatGPTOption, a.text.S.Set.CodexTitlesChatGPTDesc)
				a.settingsPickerSpecialRow(c, t, pk, "off", a.text.S.Set.Off, a.text.S.Set.CodexTitlesOffDesc)
			case "codexAutoReview":
				a.settingsPickerSpecialRow(c, t, pk, "", a.text.S.Set.CodexReviewAutoOption, a.text.S.Set.CodexReviewAutoDesc)
			}

			// Models list
			count := 0
			for _, m := range models {
				if q != "" {
					hay := strings.ToLower(m.ID + " " + m.Name + " " + m.PName)
					if !strings.Contains(hay, q) {
						continue
					}
				}
				count++
				a.settingsPickerModelRow(c, t, pk, m)
			}
			if count == 0 && q != "" {
				ui.Row(c).Padding(16, 8).Children(func() {
					ui.Text(c, a.text.S.Set.NoMatchingModels).FontSize(12).TextColor(t.TextMuted)
				})
			}
		})
	})
}

// settingsPickerSpecialRow draws one fixed choice (like Automatic or Off).
func (a *App) settingsPickerSpecialRow(c *ui.Context, t *ui.Theme, pk *settingsPickerState, value, label, sub string) {
	isSelected := pk.value == value
	row := ui.Row(c).AlignItems(ui.Center).Gap(10).
		Height(40).Padding(0, 10).Radius(8).
		Cursor(ui.CursorPointer).Role(ui.RoleButton)
	if isSelected {
		if t.Dark {
			row.Background(ui.RGBA(255, 255, 255, 0.12))
		} else {
			row.Background(ui.RGBA(0, 0, 0, 0.08))
		}
	} else if row.Hovered() {
		if t.Dark {
			row.Background(ui.RGBA(255, 255, 255, 0.06))
		} else {
			row.Background(ui.RGBA(0, 0, 0, 0.04))
		}
	}
	if row.Clicked() {
		pk.onPick(value)
		a.setPicker = nil
		c.Invalidate()
	}
	row.Children(func() {
		ui.Column(c).Grow(1).Shrink(1).Gap(1).Children(func() {
			ui.Text(c, label).FontSize(13).FontWeight(500).TextColor(t.Text).SingleLine()
			if sub != "" {
				ui.Text(c, sub).FontSize(11).TextColor(t.TextMuted).SingleLine()
			}
		})
		if isSelected {
			ui.Text(c, "✓").FontSize(13).FontWeight(600).TextColor(t.Accent).Shrink(0)
		}
	})
}

// settingsPickerModelRow draws one selectable model row.
func (a *App) settingsPickerModelRow(c *ui.Context, t *ui.Theme, pk *settingsPickerState, m ModelRef) {
	isSelected := pk.value == m.ID
	row := ui.Row(c).Key(m.ID).AlignItems(ui.Center).Gap(10).
		Height(40).Padding(0, 10).Radius(8).
		Cursor(ui.CursorPointer).Role(ui.RoleButton)
	if isSelected {
		if t.Dark {
			row.Background(ui.RGBA(255, 255, 255, 0.12))
		} else {
			row.Background(ui.RGBA(0, 0, 0, 0.08))
		}
	} else if row.Hovered() {
		if t.Dark {
			row.Background(ui.RGBA(255, 255, 255, 0.06))
		} else {
			row.Background(ui.RGBA(0, 0, 0, 0.04))
		}
	}
	if row.Clicked() {
		pk.onPick(m.ID)
		a.setPicker = nil
		c.Invalidate()
	}
	row.Children(func() {
		pic := iconPictureFor(m.Icon)
		tile := ui.Row(c).Center().Width(24).Height(24).Radius(6)
		if t.Dark {
			tile.Background(ui.RGBA(255, 255, 255, 0.06))
		} else {
			tile.Background(ui.RGBA(0, 0, 0, 0.045))
		}
		tile.Children(func() {
			if pic != nil {
				ui.Image(c, pic).Size(16, 16).Fit(ui.Contain)
			} else {
				ui.Icon(c, genericIcon()).Size(14, 14).TextColor(t.TextMuted)
			}
		})

		ui.Column(c).Grow(1).Shrink(1).Gap(1).Children(func() {
			ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
				ui.Text(c, m.Name).FontSize(13).FontWeight(500).TextColor(t.Text).SingleLine()
				ui.Text(c, m.ID).Font("monospace").FontSize(11).TextColor(t.TextMuted).SingleLine()
			})
			if m.PName != "" {
				ui.Text(c, m.PName).FontSize(11).TextColor(t.TextMuted).SingleLine()
			}
		})

		if isSelected {
			ui.Text(c, "✓").FontSize(13).FontWeight(600).TextColor(t.Accent).Shrink(0)
		}
	})
}

// settingsPickerModels returns the candidate models based on what is being picked.
func (a *App) settingsPickerModels(pk *settingsPickerState) []ModelRef {
	switch pk.kind {
	case "vision":
		return a.data.Settings.VisionModels
	case "imageGen":
		return a.data.Settings.ImageGenModels
	case "codexTitles", "codexAutoReview":
		return a.data.Settings.TitleModels
	case "searcher":
		var out []ModelRef
		for _, ch := range a.data.Settings.SearchChoices {
			for _, m := range ch.Models {
				out = append(out, m)
			}
		}
		return out
	}
	return nil
}

// settingsRestoreOverlay confirms restoring setup from the sync server.
func (a *App) settingsRestoreOverlay(c *ui.Context) {
	t := c.Theme()
	ui.Overlay(c, func() {
		back := ui.Box(c).Absolute().Left(0).Top(0).Right(0).Bottom(0).Center().
			Background(ui.RGBA(0, 0, 0, 0.4)).Modal()
		if back.OverlayShortcut(0, ui.KeyEscape) {
			a.setSyncRestore = false
			c.Invalidate()
			return
		}
		back.Children(func() {
			dialog := ui.Column(c).Width(480).
				Radius(12).Padding(20).Gap(14).
				Role(ui.RoleDialog).Label(a.text.S.Set.RestoreAskTitle)
			if t.Dark {
				dialog.Background(ui.RGBA(36, 36, 40, 1))
			} else {
				dialog.Background(t.Background)
			}
			dialog.Border(1, t.Border).
				Shadow(0, 16, 40, 0, ui.RGBA(0, 0, 0, 0.35))

			dialog.Children(func() {
				ui.Text(c, a.text.S.Set.RestoreAskTitle).FontSize(16).FontWeight(600).TextColor(t.Text)
				ui.Text(c, a.text.S.Set.RestoreAskDesc).FontSize(12.5).TextColor(t.TextMuted).Wrap()
				ui.Text(c, a.text.S.Set.RestoreAskNote).FontSize(12).TextColor(t.TextMuted).Wrap()

				if a.setSyncErr != "" {
					ui.Text(c, a.setSyncErr).FontSize(12).TextColor(ui.Hex("#dc2626"))
				}

				ui.Row(c).AlignItems(ui.Center).Gap(8).Padding(8, 0, 0, 0).Children(func() {
					ui.Spacer(c)
					if ui.Button(c, a.text.S.Cancel).Clicked() {
						a.setSyncRestore = false
						a.setSyncErr = ""
						c.Invalidate()
					}
					restoreBtn := ui.Button(c, a.text.S.Set.RestoreConfirm)
					if a.setSyncBusy {
						restoreBtn.Disabled(true)
					}
					if restoreBtn.Clicked() && !a.setSyncBusy {
						a.setSyncBusy = true
						c.Invalidate()
						go func() {
							ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
							defer cancel()
							_, err := davsync.Restore(ctx)
							if a.win != nil {
								a.win.Update(func() {
									a.setSyncBusy = false
									if err != nil {
										a.setSyncErr = err.Error()
									} else {
										a.setSyncRestore = false
										a.setSyncErr = ""
									}
									a.Reload()
								})
							}
						}()
					}
				})
			})
		})
	})
}
