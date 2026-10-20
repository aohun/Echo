//go:build nogui && echo

package shell

// The Settings page's Sync and backup tab: the WebDAV or S3 sync, the
// export and the import, each a form over one card — magpie's own, drawn
// natively and run in-process (davsync, backup).

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/yetone/magpie/internal/backup"
	"github.com/yetone/magpie/internal/davsync"
	"github.com/yetone/magpie/internal/edit"
	"github.com/yetone/magpie/internal/gui"
)

// backupExt is what an export is saved as, magpie's own.
const backupExt = ".magpie-backup"

// settingsSync draws the Sync and backup tab.
func (a *App) settingsSync(c *ui.Context, t *ui.Theme) {
	v := davsync.Status()

	ui.Column(c).FillWidth().Gap(14).Children(func() {
		a.setNoticeBar(c, t)

		a.setCard(c, t, a.text.S.Set.SyncCard, func() {
			// the row itself: what is synced to, when last, what failed
			title := a.text.S.Set.SyncOff
			if v.On {
				title = strings.ToUpper(cmpOr(v.Kind, "webdav")) + " · " + v.URL
			}
			sub := a.text.S.Set.SyncNoneYet
			if !v.Last.IsZero() {
				sub = a.text.S.Set.SyncLastPrefix + syncWhen(v.Last)
			}
			if v.Error != "" {
				sub += " · " + v.Error
			}
			a.setButtonRow(c, t, title, func() {
				ui.Column(c).Grow(1).Shrink(1).Gap(2).Children(func() {
					ui.Text(c, sub).FontSize(11.5).TextColor(t.TextMuted).Wrap()
				})
				if v.On {
					if a.smallBtn(c, t, a.text.S.Set.SyncNow) {
						a.setSyncBusy = true
						go func() {
							ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
							err := davsync.SyncNow(ctx)
							cancel()
							a.endSync(err)
						}()
					}
					if a.smallBtn(c, t, a.text.S.Set.SyncEdit) {
						a.setSyncOpen, a.setSyncErr = "", ""
						a.setSyncRestore, a.setSyncBusy = false, false
						a.fillSyncForm(v)
						c.Invalidate()
					}
				} else if a.smallBtn(c, t, a.text.S.Set.SyncEdit) {
					a.setSyncOpen = "dav"
					a.setSyncErr = ""
					c.Invalidate()
				}
			})

			// the form, shown while it is being set up or edited
			if a.setSyncOpen == "dav" {
				a.davForm(c, t, v)
			}

			// how often it syncs by itself, while it is on
			if v.On {
				autoIdx := 0
				for i, m := range []int{3, 15, 30, 60} {
					if v.Auto == m {
						autoIdx = i + 1
					}
				}
				a.setSegRow(c, t, a.text.S.Set.SyncAuto, "", &autoIdx, func(i int) {
					mins := 0
					if i > 0 {
						mins = []int{3, 15, 30, 60}[i-1]
					}
					a.settingsWrite(func() error { return davsync.SetAuto(mins) }, nil)
				}, a.text.S.Set.Off, "3 min", "15 min", "30 min", "60 min")
				a.setButtonRow(c, t, a.text.S.Set.SyncRestoreBtn, func() {
					ui.Spacer(c)
					if a.smallBtn(c, t, a.text.S.Set.SyncRestoreBtn) {
						a.setSyncRestore = true
						c.Invalidate()
					}
				})
			}
		})

		// the export and the import, each over one card
		a.setCard(c, t, a.text.S.Set.BackupCard, func() {
			a.setButtonRow(c, t, a.text.S.Set.Export, func() {
				ui.Spacer(c)
				if a.smallBtn(c, t, a.text.S.Set.Export) {
					a.setSyncOpen = "export"
					a.setExportResult = ""
					c.Invalidate()
				}
			})
			if a.setSyncOpen == "export" {
				a.exportForm(c, t)
			}
			a.setButtonRow(c, t, a.text.S.Set.Import, func() {
				ui.Spacer(c)
				if a.smallBtn(c, t, a.text.S.Set.Import) {
					a.setSyncOpen = "import"
					a.setImportResult = ""
					c.Invalidate()
				}
			})
			if a.setSyncOpen == "import" {
				a.importForm(c, t)
			}
		})
	})
}

// fillSyncForm puts the server's view in the form's fields, so an edit
// starts from what is there.
func (a *App) fillSyncForm(v davsync.View) {
	a.setSyncOpen = "dav"
	a.setSyncKind = cmpOr(v.Kind, "webdav")
	a.setSyncURL = v.URL
	a.setSyncUser = v.User
	a.setSyncEndpoint = v.Endpoint
	a.setSyncRegion = v.Region
	a.setSyncPathStyle = v.PathStyle
	a.setSyncKeys, a.setSyncAgents, a.setSyncLib, a.setSyncUsage = v.Keys, v.Agents, v.Library, v.Usage
}

// davForm is the sync server's own form: which kind, its fields, the
// passphrase, and what goes in the backup.
func (a *App) davForm(c *ui.Context, t *ui.Theme, v davsync.View) {
	kindIdx := 0
	if a.setSyncKind == "s3" {
		kindIdx = 1
	}
	ui.Column(c).FillWidth().Gap(8).Padding(10, 14, 12, 14).Children(func() {
		ui.Row(c).AlignItems(ui.Center).Gap(10).Children(func() {
			ui.Text(c, a.text.S.Set.SyncTo).FontSize(12.5).FontWeight(500).TextColor(t.Text)
			if ui.Segmented(c, &kindIdx, "WebDAV", "S3").Changed() {
				a.setSyncKind = []string{"webdav", "s3"}[kindIdx]
				c.Invalidate()
			}
		})

		field := func(label string, val *string) {
			ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(8).Children(func() {
				ui.Text(c, label).FontSize(12).TextColor(t.TextMuted).Width(90).Shrink(0)
				ui.TextInput(c, val).Label(label).Grow(1).Shrink(1)
			})
		}
		if a.setSyncKind == "s3" {
			field(a.text.S.Set.SyncEndpoint, &a.setSyncEndpoint)
			field(a.text.S.Set.SyncRegion, &a.setSyncRegion)
			field(a.text.S.Set.SyncBucket, &a.setSyncURL)
		} else {
			field(a.text.S.Set.SyncAddress, &a.setSyncURL)
			field(a.text.S.Set.SyncUser, &a.setSyncUser)
			field(a.text.S.Set.SyncPass, &a.setSyncPass)
		}
		field(a.text.S.Set.SyncPhrase, &a.setSyncPhrase)

		// what goes in the backup
		ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(14).Children(func() {
			ui.Text(c, a.text.S.Set.SyncAlso).FontSize(12).TextColor(t.TextMuted)
			if ui.Checkbox(c, &a.setSyncKeys, a.text.S.Set.SyncKeysL).Changed() {
				c.Invalidate()
			}
			if ui.Checkbox(c, &a.setSyncAgents, a.text.S.Set.SyncAgentsL).Changed() {
				c.Invalidate()
			}
			if ui.Checkbox(c, &a.setSyncLib, a.text.S.Set.SyncLibL).Changed() {
				c.Invalidate()
			}
			if ui.Checkbox(c, &a.setSyncUsage, a.text.S.Set.SyncUsageL).Changed() {
				c.Invalidate()
			}
		})

		if a.setSyncErr != "" {
			ui.Text(c, a.setSyncErr).FontSize(11.5).TextColor(ui.Hex("#dc2626")).Wrap()
		}
		ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
			if b := ui.Button(c, a.text.S.Set.SyncSave); b.Clicked() {
				a.saveSync()
			}
			if v.On {
				if b := ui.Button(c, a.text.S.Set.SyncTurnOff); b.Clicked() {
					a.setSyncErr = ""
					a.settingsWrite(func() error { return davsync.Off() }, func(err error) {
						if err != nil {
							a.setSyncErr = err.Error()
						}
					})
				}
			}
			if b := ui.Button(c, a.text.S.Cancel); b.Clicked() {
				a.setSyncOpen = ""
				c.Invalidate()
			}
		})
	})
}

// saveSync takes the form for the server, configures it, and syncs at
// once, so a wrong address or password shows now — magpie's davsync/save.
func (a *App) saveSync() {
	c := davsync.Config{
		URL:        strings.TrimSpace(a.setSyncURL),
		User:       a.setSyncUser,
		Password:   a.setSyncPass,
		Passphrase: a.setSyncPhrase,
		Endpoint:   strings.TrimSpace(a.setSyncEndpoint),
		Region:     strings.TrimSpace(a.setSyncRegion),
		PathStyle:  a.setSyncPathStyle,
		Keys:       a.setSyncKeys,
		Agents:     a.setSyncAgents,
		Usage:      a.setSyncUsage,
		Library:    &a.setSyncLib,
	}
	a.setSyncErr = ""
	a.setSyncBusy = true
	go func() {
		err := davsync.Configure(c)
		if err == nil {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			err = davsync.SyncNow(ctx)
			cancel()
		}
		if a.win == nil {
			return
		}
		a.win.Update(func() {
			a.setSyncBusy = false
			if err != nil {
				a.setSyncErr = err.Error()
			} else {
				a.setSyncOpen = ""
				a.setSetNotice(a.text.S.Set.Saved, false)
			}
			a.Reload()
		})
	}()
}

// endSync takes a finished sync's error, as the row's line says it.
func (a *App) endSync(err error) {
	if a.win == nil {
		return
	}
	a.win.Update(func() {
		a.setSyncBusy = false
		if err != nil {
			a.setSetNotice(err.Error(), true)
		} else {
			a.setSetNotice(a.text.S.Set.Saved, false)
		}
		a.Reload()
	})
}

// exportForm is the export's own form: the passphrase twice, what goes in
// the file, and where it landed.
func (a *App) exportForm(c *ui.Context, t *ui.Theme) {
	ui.Column(c).FillWidth().Gap(8).Padding(6, 14, 12, 14).Children(func() {
		ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(8).Children(func() {
			ui.Text(c, a.text.S.Set.ExportPass).FontSize(12).TextColor(t.TextMuted).Width(90).Shrink(0)
			ui.TextInput(c, &a.setExportPass1).Label(a.text.S.Set.ExportPass).Grow(1).Shrink(1)
		})
		ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(8).Children(func() {
			ui.Text(c, a.text.S.Set.ExportPass2).FontSize(12).TextColor(t.TextMuted).Width(90).Shrink(0)
			ui.TextInput(c, &a.setExportPass2).Label(a.text.S.Set.ExportPass2).Grow(1).Shrink(1)
		})
		ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(14).Children(func() {
			if ui.Checkbox(c, &a.setExportKeys, a.text.S.Set.SyncKeysL).Changed() {
				c.Invalidate()
			}
			if ui.Checkbox(c, &a.setExportLib, a.text.S.Set.SyncLibL).Changed() {
				c.Invalidate()
			}
		})
		if a.setExportResult != "" {
			ui.Text(c, a.setExportResult).FontSize(11.5).TextColor(t.TextMuted).Wrap()
		}
		ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
			if b := ui.Button(c, a.text.S.Set.Export); b.Clicked() {
				a.runExport()
			}
			if b := ui.Button(c, a.text.S.Cancel); b.Clicked() {
				a.setSyncOpen = ""
				c.Invalidate()
			}
		})
	})
}

// runExport writes the sealed backup into the Downloads folder, as
// magpie's backup/export does: never over an earlier one.
func (a *App) runExport() {
	if a.setExportPass1 == "" || a.setExportPass1 != a.setExportPass2 {
		a.setExportResult = a.text.S.Set.ExportErrPass
		c := a.win
		if c != nil {
			c.Invalidate()
		}
		return
	}
	pass, keys, lib := a.setExportPass1, a.setExportKeys, a.setExportLib
	go func() {
		res, err := func() (string, error) {
			b, err := backup.Collect(keys, gui.Version)
			if err != nil {
				return "", err
			}
			if !lib {
				b.Library = nil
			}
			data, err := backup.Seal(b, pass)
			if err != nil {
				return "", err
			}
			name, err := exportPath()
			if err != nil {
				return "", err
			}
			if err := edit.WriteAtomic(name, data); err != nil {
				return "", err
			}
			os.Chmod(name, 0o600)
			return fmt.Sprintf("%s (%d)", name, len(b.Providers)), nil
		}()
		if a.win == nil {
			return
		}
		a.win.Update(func() {
			if err != nil {
				a.setExportResult = err.Error()
			} else {
				a.setExportResult = a.text.S.Set.ExportDonePrefix + res
				a.setExportPass1, a.setExportPass2 = "", ""
			}
		})
	}()
}

// exportPath is the file the next export is saved as: the Downloads
// folder, or home when there is none.
func exportPath() (string, error) {
	home, _ := os.UserHomeDir()
	dir := home
	if d := filepath.Join(home, "Downloads"); backupDir(d) {
		dir = d
	}
	name := filepath.Join(dir, "echo-"+time.Now().Format("2006-01-02")+backupExt)
	for i := 2; ; i++ {
		if _, err := os.Stat(name); err != nil {
			return name, nil
		}
		name = filepath.Join(dir, fmt.Sprintf("echo-%s-%d%s", time.Now().Format("2006-01-02"), i, backupExt))
	}
}

// importForm is the import's own form: the file's path, its passphrase,
// what comes back in.
func (a *App) importForm(c *ui.Context, t *ui.Theme) {
	ui.Column(c).FillWidth().Gap(8).Padding(6, 14, 12, 14).Children(func() {
		ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(8).Children(func() {
			ui.Text(c, a.text.S.Set.ImportPath).FontSize(12).TextColor(t.TextMuted).Width(90).Shrink(0)
			ui.TextInput(c, &a.setImportPath).Label("~/Downloads/echo-…" + backupExt).Grow(1).Shrink(1)
		})
		ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(8).Children(func() {
			ui.Text(c, a.text.S.Set.SyncPhrase).FontSize(12).TextColor(t.TextMuted).Width(90).Shrink(0)
			ui.TextInput(c, &a.setImportPass).Label(a.text.S.Set.SyncPhrase).Grow(1).Shrink(1)
		})
		ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(14).Children(func() {
			if ui.Checkbox(c, &a.setImportAgents, a.text.S.Set.ImportAgentsL).Changed() {
				c.Invalidate()
			}
			if ui.Checkbox(c, &a.setImportLib, a.text.S.Set.SyncLibL).Changed() {
				c.Invalidate()
			}
		})
		if a.setImportResult != "" {
			ui.Text(c, a.setImportResult).FontSize(11.5).TextColor(t.TextMuted).Wrap()
		}
		ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
			if b := ui.Button(c, a.text.S.Set.Import); b.Clicked() {
				a.runBackupImport()
			}
			if b := ui.Button(c, a.text.S.Cancel); b.Clicked() {
				a.setSyncOpen = ""
				c.Invalidate()
			}
		})
	})
}

// runBackupImport reads the file at the path given, opens it with the
// passphrase, and brings in what the ticks keep — magpie's backup/import.
func (a *App) runBackupImport() {
	path := strings.TrimSpace(a.setImportPath)
	pass := a.setImportPass
	agents, lib := a.setImportAgents, a.setImportLib
	go func() {
		res, err := func() (string, error) {
			if path == "" {
				return "", fmt.Errorf("no file named")
			}
			p := path
			if strings.HasPrefix(p, "~/") {
				if home, e := os.UserHomeDir(); e == nil {
					p = filepath.Join(home, p[2:])
				}
			}
			data, err := os.ReadFile(p)
			if err != nil {
				return "", err
			}
			b, err := backup.Open(data, pass)
			if err != nil {
				return "", err
			}
			parts := backup.All
			parts.Agents = agents
			parts.Library = lib
			r, err := backup.Restore(b, parts)
			if err != nil {
				return "", err
			}
			return importSaid(r), nil
		}()
		if a.win == nil {
			return
		}
		a.win.Update(func() {
			if err != nil {
				a.setImportResult = err.Error()
			} else {
				a.setImportResult = res
				a.setImportPass = ""
			}
			a.Reload()
		})
	}()
}

// backupDir says whether p is a directory there.
func backupDir(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.IsDir()
}

// importSaid is what an import tells it brought in, as magpie's own
// answer reads.
func importSaid(r backup.Result) string {
	parts := []string{}
	if n := r.Added + r.Replaced; n > 0 {
		parts = append(parts, fmt.Sprintf("%d providers", n))
	}
	if r.Profiles > 0 {
		parts = append(parts, fmt.Sprintf("%d profiles", r.Profiles))
	}
	if r.Agents > 0 {
		parts = append(parts, fmt.Sprintf("%d agents", r.Agents))
	}
	if r.Library {
		parts = append(parts, "library")
	}
	if len(parts) == 0 {
		return "nothing to bring in"
	}
	return "Brought in " + strings.Join(parts, ", ")
}
