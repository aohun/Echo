//go:build nogui && echo

package shell

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/yetone/magpie/internal/provider"
)

// The Providers page's own look for configurations other apps have set up
// (magpie's discoverLocalProviders and renderProviderDiscovery): Alma, CC
// Switch, Claude Code and Codex are read where the page opens, the ones
// that can come in offered under a hint — "Found N local provider
// configurations" over an empty list, "Import local configurations (N)…"
// beside Add provider over a full one. Only app names and fingerprints
// reach the hint; only the import's picker reads keys, and it reads them
// again on the save.

// importFingerprint is an opaque id for one importable configuration, the
// way magpie's own handler makes it (importapps.go): enough to know a
// configuration is the one that was put off, never enough to reach its
// key, endpoints or path.
func importFingerprint(source string, it provider.AppImport) string {
	p := it.Provider
	p.ID = "" // settle's generated ID can change when another source is added
	b, _ := json.Marshal(struct {
		Source, Ref string
		Provider    provider.Provider
	}{source, it.Ref, p})
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// discoveryCandidate is one configuration the hint is about: its
// fingerprint, and the app it was read in.
type discoveryCandidate struct {
	Fingerprint string
	Source      string
}

// discovery is the page's own scan, as magpie's page holds its own: what
// the apps offered (candidates), and whether a look is out (pending).
// Guarded by its lock: the scan runs on a goroutine, the frame that draws
// the hint reads the result.
type discovery struct {
	mu         sync.Mutex
	candidates []discoveryCandidate
	pending    bool
}

// ignoredIDs is the fingerprints put off, kept in a file of its own under
// the app's user-data folder (beside the window's state file) so they
// outlast the app the way magpie's localStorage copy does. One JSON array
// of hex strings, read once and rewritten on every change.
//
// It is shell state, not a domain setting: internal/settings is magpie's
// and must not change (Plan A's scope).
var ignoredIDs struct {
	mu   sync.Mutex
	ids  map[string]bool
	read bool
}

// ignoredPath is the file the put-off fingerprints are kept in.
func ignoredPath() string {
	dir, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "provider-discovery-ignored.json")
}

// ignoredRead reads the put-off fingerprints once. A damaged file is as
// good as none: an offering is not worth failing over.
func ignoredRead() map[string]bool {
	ignoredIDs.mu.Lock()
	defer ignoredIDs.mu.Unlock()
	if ignoredIDs.read {
		return ignoredIDs.ids
	}
	ignoredIDs.read = true
	ignoredIDs.ids = map[string]bool{}
	b, err := os.ReadFile(ignoredPath())
	if err != nil {
		return ignoredIDs.ids
	}
	var ids []string
	if json.Unmarshal(b, &ids) != nil {
		ignoredIDs.ids = map[string]bool{}
		return ignoredIDs.ids
	}
	for _, id := range ids {
		ignoredIDs.ids[id] = true
	}
	return ignoredIDs.ids
}

// ignoredSave puts fingerprints away: hidden until a configuration they
// were made from changes or is added, as magpie's own dismissal is.
func ignoredSave(ids []string) {
	ignoredRead()
	ignoredIDs.mu.Lock()
	for _, id := range ids {
		ignoredIDs.ids[id] = true
	}
	kept := make([]string, 0, len(ignoredIDs.ids))
	for id := range ignoredIDs.ids {
		kept = append(kept, id)
	}
	ignoredIDs.mu.Unlock()
	sort.Strings(kept)

	if _, err := mygo.App.Path(mygo.PathUserData); err == nil {
		if b, err := json.Marshal(kept); err == nil {
			_ = os.WriteFile(ignoredPath(), b, 0o600)
		}
	}
}

// discoverProviders looks in the apps the page reads by itself, off the
// main thread, at most once a minute as the page is opened or reloaded —
// the same cadence magpie's own keeps. The result lands back on a and the
// window is told to redraw the hint.
//
// A failure is quiet: the look is an offering, never a need, and must not
// block adding by hand (magpie's own comment).
func (a *App) discoverProviders() {
	d := &a.pvDiscovery
	d.mu.Lock()
	if d.pending {
		d.mu.Unlock()
		return
	}
	d.pending = true
	d.mu.Unlock()

	go func() {
		var got []discoveryCandidate
		for _, s := range provider.DiscoverSources() {
			for _, it := range s.Items {
				// the ones that can come in on the default pick, as
				// magpie's own handler filters: a skipped entry, one
				// already in, one turned off in its app, and an id
				// collision that cannot join as another key need the
				// picker's hand
				if it.Skip == "" && it.Status != "same" && it.Off == "" && (it.Status != "taken" || it.KeyOf != "" || it.Reserved) {
					got = append(got, discoveryCandidate{importFingerprint(s.ID, it), s.Name})
				}
			}
		}
		if win := a.win; win != nil {
			win.Update(func() {
				d.mu.Lock()
				d.candidates, d.pending = got, false
				d.mu.Unlock()
			})
			return
		}
		d.mu.Lock()
		d.candidates, d.pending = got, false
		d.mu.Unlock()
	}()
}

// discoveryFresh is the configurations found and not put off, newest look.
func (a *App) discoveryFresh() []discoveryCandidate {
	d := &a.pvDiscovery
	d.mu.Lock()
	defer d.mu.Unlock()
	set := ignoredRead()
	out := make([]discoveryCandidate, 0, len(d.candidates))
	for _, c := range d.candidates {
		if !set[c.Fingerprint] {
			out = append(out, c)
		}
	}
	return out
}

// ---- the import picker -----------------------------------------------------

// openImport opens the Import from other apps dialog over the page,
// reading the sources on a goroutine and drawing the picker when they
// land (magpie's openImportApps). The offered ones come picked, the rest
// of what the apps hold shown beside them. Sources are read again on the
// import itself, so a key never sits in the window longer than the picker
// is open.
func (a *App) openImport(fresh []discoveryCandidate) {
	offered := map[string]bool{}
	for _, f := range fresh {
		offered[f.Fingerprint] = true
	}
	a.pvImporting = &importState{picks: map[string]bool{}}
	if a.win != nil {
		a.win.Invalidate()
	}

	go func() {
		sources := maskedSources()
		if a.win == nil {
			return
		}
		a.win.Update(func() {
			ia := a.pvImporting
			if ia == nil {
				return // the dialog was put away while the read ran
			}
			ia.sources = sources
			// picked by default: the ones the hint offered, on and
			// importable — openImportApps' own selection
			for _, s := range sources {
				for _, it := range s.Items {
					pick := offered[importFingerprint(s.ID, it)] &&
						it.Skip == "" && it.Status != "same" && it.Off == ""
					ia.picks[s.ID+"\n"+it.Ref] = pick
					if pick && ia.tab == "" {
						ia.tab = s.ID
					}
				}
			}
		})
	}()
}

// maskedSources reads every app's configurations as the picker draws
// them, keys masked — the same read magpie's own GET /api/importapps is.
func maskedSources() []provider.AppSource {
	sources := provider.ImportSources()
	for i := range sources {
		for j := range sources[i].Items {
			p := &sources[i].Items[j].Provider
			p.Key, p.Keys = provider.Mask(p.Key), nil
		}
	}
	return sources
}

// importDialog draws the Import from other apps dialog: one tab per app
// Echo can import from, the entries of the one showing, each with its
// name, where in the app it was, and how it stands; the picks are
// imported on Import, nothing before (magpie's renderImportApps).
func (a *App) importDialog(c *ui.Context) {
	t := c.Theme()
	ia := a.pvImporting

	box := ui.Column(c).Width(560).Gap(10).Padding(16).Radius(12)
	if t.Dark {
		box.Background(ui.RGBA(40, 40, 46, 1))
	} else {
		box.Background(t.Background)
	}

	box.Children(func() {
		ui.Text(c, a.text.S.ImportTitle).FontSize(14).FontWeight(600).TextColor(t.Text)

		// the reads still out, or failed where they stand
		if len(ia.sources) == 0 {
			ui.Text(c, a.text.S.ImportReading).FontSize(12.5).TextColor(t.TextMuted)
			a.importBar(c, ia, true)
			return
		}

		ui.Text(c, a.text.S.ImportFromApps).FontSize(12).TextColor(t.TextMuted)

		// one tab per app, so which ones Echo can read is plain at a
		// glance; each shows how many entries it holds
		ui.Row(c).Gap(6).Children(func() {
			for _, s := range ia.sources {
				id := s.ID
				label := fmt.Sprintf("%s (%d)", s.Name, len(s.Items))
				tab := ui.Row(c).Padding(4, 10).Radius(7).Cursor(ui.CursorPointer).
					Focusable().Role(ui.RoleButton).Key("ia/" + id)
				chosen := ia.tab == id
				if tab.Clicked() {
					ia.tab = id
					c.Invalidate()
				}
				if chosen {
					tab.Background(ui.RGBA(0, 0, 0, 0.07))
					if t.Dark {
						tab.Background(ui.RGBA(255, 255, 255, 0.09))
					}
				}
				tab.Children(func() {
					ui.Text(c, label).FontSize(12).TextColor(t.Text).SingleLine()
				})
			}
		})

		// the showing app's entries, importable ones first, as settle
		// ordered them; a skipped entry is said rather than offered
		ui.Column(c).Gap(4).Grow(1).Shrink(1).Children(func() {
			for _, s := range ia.sources {
				if s.ID != ia.tab {
					continue
				}
				for _, it := range s.Items {
					a.importRow(c, ia, s, it, t)
				}
			}
		})

		a.importBar(c, ia, false)
	})
}

// importRow is one entry of the picker: its check, its name, what it is
// and how it stands (new, already in, its id taken, not importable).
func (a *App) importRow(c *ui.Context, ia *importState, s provider.AppSource, it provider.AppImport, t *ui.Theme) {
	key := s.ID + "\n" + it.Ref
	pickable := it.Skip == "" && it.Status != "same"

	row := ui.Row(c).AlignItems(ui.Center).Gap(8).Padding(6, 8).Radius(8)

	var on bool
	if pickable {
		on = ia.picks[key]
		if row.Clicked() {
			ia.picks[key] = !on
			c.Invalidate()
		}
		row.Cursor(ui.CursorPointer)
	}

	row.Children(func() {
		if pickable {
			checked := ui.Row(c).Width(16).Height(16).Radius(4).Border(1, t.Border)
			if on {
				checked.Background(t.Accent)
			}
		}
		ui.Column(c).Grow(1).Shrink(1).Gap(1).Children(func() {
			ui.Text(c, it.Provider.Name).FontSize(12.5).TextColor(t.Text).SingleLine()
			line := it.From
			switch {
			case it.Skip != "":
				line += " · " + it.Skip
			case it.Status == "same":
				line += " · already in"
			case it.Status == "taken":
				line += " · its id is taken"
			case it.Off != "":
				line += " · " + it.Off
			}
			ui.Text(c, line).FontSize(11).TextColor(t.TextMuted).SingleLine()
		})
	})
}

// importBar is the picker's foot: how many are picked, and the two answers.
func (a *App) importBar(c *ui.Context, ia *importState, loading bool) {
	t := c.Theme()
	n := 0
	for _, on := range ia.picks {
		if on {
			n++
		}
	}
	count := ""
	if n > 0 {
		count = strings.ReplaceAll(a.text.S.ImportN, "{n}", fmt.Sprint(n))
	}
	ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
		ui.Text(c, count).FontSize(11.5).TextColor(t.TextMuted).Grow(1)
		if b := ui.Button(c, a.text.S.Cancel); b.Clicked() {
			a.pvImporting = nil
		}
		importing := ui.Button(c, a.text.S.Import)
		if n == 0 || loading {
			importing.Disabled(true)
		}
		if importing.Clicked() {
			a.runImport(ia)
		}
	})
}

// runImport adds what was picked, reading the sources again so keys are
// never kept from the picker's read, and reloads the page with the names
// added said (magpie's POST /api/importapps). A failure keeps what was
// added before it and says so.
func (a *App) runImport(ia *importState) {
	var picks []provider.AppPick
	for key, on := range ia.picks {
		if !on {
			continue
		}
		source, ref, _ := strings.Cut(key, "\n")
		picks = append(picks, provider.AppPick{Source: source, Ref: ref})
	}
	a.pvImporting = nil
	go func() {
		var (
			added []string
			err   error
		)
		if len(picks) > 0 {
			added, err = provider.ImportFromApps(picks)
		}
		if a.win == nil {
			return
		}
		a.win.Update(func() {
			_ = added // the names said are the page's next read's own
			_ = err
			a.Reload()
		})
	}()
}

// ---- the sign-out ask ------------------------------------------------------

// signOutAsk is the sign-out ask's own state: the account it is about.
type signOutAsk struct{ Name, ID string }

// askSignOut asks before a removed account is signed out for good: what
// Echo keeps of its sign-ins goes, so adding it again signs in afresh
// (magpie's askForgetAccount).
func (a *App) askSignOut(name, id string) {
	a.pvSignOut = &signOutAsk{Name: name, ID: id}
	if a.win != nil {
		a.win.Invalidate()
	}
}

// signOutDialog draws the ask, the two answers at its foot.
func (a *App) signOutDialog(c *ui.Context) {
	t := c.Theme()
	ask := a.pvSignOut

	box := ui.Column(c).Width(440).Gap(10).Padding(16).Radius(12)
	if t.Dark {
		box.Background(ui.RGBA(40, 40, 46, 1))
	} else {
		box.Background(t.Background)
	}
	box.Children(func() {
		ui.Text(c, strings.ReplaceAll(a.text.S.SignOutAsk, "{name}", ask.Name)).
			FontSize(14).FontWeight(600).TextColor(t.Text)
		ui.Text(c, strings.ReplaceAll(a.text.S.SignOutHelp, "{name}", ask.Name)).
			FontSize(12).TextColor(t.TextMuted)
		ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
			ui.Spacer(c)
			if b := ui.Button(c, a.text.S.Cancel); b.Clicked() {
				a.pvSignOut = nil
			}
			if b := ui.Button(c, a.text.S.SignOut); b.Clicked() {
				id := ask.ID
				a.pvSignOut = nil
				go func() {
					// a removed account signed out for good: what Echo
					// keeps of its sign-ins goes, adding it again signs
					// in afresh (#694)
					err := provider.ForgetAccount(id)
					_ = err
					if a.win != nil {
						a.win.Update(func() { a.Reload() })
					}
				}()
			}
		})
	})
}
