//go:build nogui && echo

package shell

// The Settings page's remote read and writes (docs/08 M4): the settings
// document and the facts its rows show beside it, from a magpie web
// instance's GET /api/settings — the very document magpie's settingsJSON
// marshals, the saved settings carried inline and the facts around them
// — and the page's writes asked of the instance the way magpie's own
// page asks them: POST /api/settings for the saved settings, and the
// settings' own routes for what is set on its own.
//
// What stays embedded even in remote mode:
//   - Version and Update: the shell says its own version and draws its
//     own updater's state; the instance's update pill is not Echo's.
//     Both are read embedded in either mode.
//   - TrayCards: the allowances picker's cards are the quota cache the
//     shell already holds (PeekQuotas), which the document does not
//     carry; the picker draws the same cache in either mode until it is
//     fed by the Usage page's remote reads.
//   - the Sync and backup tab: its status, save and restore move this
//     machine's sync state, and its save and sync-now run the sync in
//     the request without a time limit — longer than the loopback
//     client waits, so a first sync that outlasted it would read as a
//     failure that did not happen. Its endpoints (/api/davsync/*) are
//     left for a later mapping.
//   - the backup file's export and import: they write and read a file
//     the user picked on this machine.
//   - the About tab's update actions: the shell's own updater's, no
//     settings endpoint serves them.

import (
	"context"
	"time"

	"github.com/yetone/magpie/internal/access"
	"github.com/yetone/magpie/internal/autostart"
	"github.com/yetone/magpie/internal/gui"
	"github.com/yetone/magpie/internal/magpieclient"
	"github.com/yetone/magpie/internal/provider"
	"github.com/yetone/magpie/internal/settings"
)

// settingsDoc is the instance's GET /api/settings answer, read only as
// far as the page draws: the saved settings carried inline, the facts
// around them named one by one. A fact the shell says about itself
// (Version, Update) is not among them — those stay embedded, above.
type settingsDoc struct {
	settings.Settings

	Version string `json:"version"`
	Dir     string `json:"dir"`
	Gateway string `json:"gateway"`
	AddrEnv string `json:"addrEnv,omitempty"`
	// Dir is the data folder beside a portable instance
	Portable bool `json:"portable,omitempty"`

	TerminalApps    []gui.TerminalApp `json:"terminalApps,omitempty"`
	TerminalDefault string            `json:"terminalDefault,omitempty"`
	// Login is the system's record, not a setting
	Login bool `json:"login"`

	ProxyNow    string `json:"proxyNow"`
	ProxySource string `json:"proxySource"`

	LANURLs      []string `json:"lanURLs,omitempty"`
	LANContainer bool     `json:"lanContainer,omitempty"`

	FX     gui.FX `json:"fx"`
	GHMask string `json:"githubTokenMask,omitempty"`
	GHFrom string `json:"githubTokenFrom,omitempty"`

	CodexUsers   []string   `json:"codexUsers,omitempty"`
	CodexWarmed  *time.Time `json:"codexWarmed,omitempty"`
	ClaudeWarmed *time.Time `json:"claudeWarmed,omitempty"`

	WorkBuddy       bool                        `json:"workbuddy"`
	WBCheckins      []provider.WorkBuddyCheckin `json:"workbuddyCheckins,omitempty"`
	Trae            bool                        `json:"trae"`
	TraeCheckins    []provider.WorkBuddyCheckin `json:"traeCheckins,omitempty"`
	MiniMax         bool                        `json:"minimax"`
	MiniMaxCheckins []provider.WorkBuddyCheckin `json:"minimaxCheckins,omitempty"`
	Qoder           bool                        `json:"qoder"`
	QoderCheckins   []provider.WorkBuddyCheckin `json:"qoderCheckins,omitempty"`

	VisionAuto     string           `json:"visionAuto,omitempty"`
	VisionModels   []remoteModelRef `json:"visionModels"`
	TitleModels    []remoteModelRef `json:"titleModels"`
	ImageGenAuto   string           `json:"imageGenAuto,omitempty"`
	ImageGenModels []remoteModelRef `json:"imageGenModels"`

	SearchAuto     string               `json:"searchAuto,omitempty"`
	SearchUnused   string               `json:"searchUnused,omitempty"`
	SearchProvider string               `json:"searchProvider,omitempty"`
	SearchChoices  []remoteSearchChoice `json:"searchChoices"`
	SearchRelays   []string             `json:"searchRelays,omitempty"`
	SearchLeftOut  []string             `json:"searchLeftOut,omitempty"`
	SearchAPIs     []remoteSearchAPI    `json:"searchAPIs"`
	SearchVendors  []remoteSearchVendor `json:"searchVendors"`
}

// remoteModelRef is a model as the document names one: the page's
// ModelRef with the document's field names.
type remoteModelRef struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Provider string `json:"provider"`
	PName    string `json:"providerName"`
	Icon     string `json:"icon"`
}

// remoteSearchChoice is a provider that can search the web for a model
// that can't, as the document carries it.
type remoteSearchChoice struct {
	ID        string           `json:"id"`
	Name      string           `json:"name"`
	Icon      string           `json:"icon"`
	Small     string           `json:"small"`
	SmallName string           `json:"smallName"`
	Models    []remoteModelRef `json:"models"`
	Service   bool             `json:"service"`
	Own       bool             `json:"own"`
}

// remoteSearchAPI is one search API set, as the document carries it.
type remoteSearchAPI struct {
	Vendor string `json:"vendor"`
	Name   string `json:"name"`
	Key    string `json:"key"`
	URL    string `json:"url"`
	Ready  bool   `json:"ready"`
}

// remoteSearchVendor is one search API that can be added.
type remoteSearchVendor struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	KeysURL string `json:"keysURL"`
	NeedURL bool   `json:"needURL"`
}

// settingsDocRemote reads the instance's settings document.
func settingsDocRemote(ctx context.Context, c *magpieclient.Client) (settingsDoc, error) {
	doc := settingsDoc{}
	if err := c.Do(ctx, "GET", "/api/settings", nil, &doc); err != nil {
		return settingsDoc{}, err
	}
	return doc, nil
}

// readSettingsRemote fills what the Settings page draws from a magpie
// web instance's /api/settings: the page's settings as the document
// carries them, and the facts its rows show beside it — the proxy in
// use, the masked GitHub token, the sharing URLs, the pickers' models,
// the warm-ups' accounts and the check-ins.
func readSettingsRemote(c *magpieclient.Client) (SettingsSnap, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	doc, err := settingsDocRemote(ctx, c)
	if err != nil {
		return SettingsSnap{}, false
	}
	snap := SettingsSnap{
		S:               doc.Settings,
		Version:         appVersion(),
		Dir:             doc.Dir,
		Portable:        doc.Portable,
		Gateway:         doc.Gateway,
		AddrEnv:         doc.AddrEnv,
		Login:           doc.Login,
		ProxyNow:        doc.ProxyNow,
		ProxySource:     doc.ProxySource,
		LANURLs:         doc.LANURLs,
		LANContainer:    doc.LANContainer,
		FX:              doc.FX,
		GHFrom:          doc.GHFrom,
		GHMask:          doc.GHMask,
		Update:          gui.UpdateState(),
		CodexUsers:      doc.CodexUsers,
		CodexWarmed:     doc.CodexWarmed,
		ClaudeWarmed:    doc.ClaudeWarmed,
		WorkBuddy:       doc.WorkBuddy,
		WBCheckins:      doc.WBCheckins,
		Trae:            doc.Trae,
		TraeCheckins:    doc.TraeCheckins,
		MiniMax:         doc.MiniMax,
		MiniMaxCheckins: doc.MiniMaxCheckins,
		Qoder:           doc.Qoder,
		QoderCheckins:   doc.QoderCheckins,
		VisionAuto:      doc.VisionAuto,
		VisionModels:    remoteModelRefs(doc.VisionModels),
		TitleModels:     remoteModelRefs(doc.TitleModels),
		ImageGenAuto:    doc.ImageGenAuto,
		ImageGenModels:  remoteModelRefs(doc.ImageGenModels),
		SearchAuto:      doc.SearchAuto,
		SearchUnused:    doc.SearchUnused,
		SearchProvider:  doc.SearchProvider,
		SearchRelays:    doc.SearchRelays,
		SearchLeftOut:   doc.SearchLeftOut,
		TrayCards:       trayCardsCached(),
	}
	for _, a := range doc.SearchAPIs {
		snap.SearchAPIs = append(snap.SearchAPIs, SearchAPIRow{Vendor: a.Vendor, Name: a.Name, Key: a.Key, URL: a.URL, Ready: a.Ready})
	}
	for _, v := range doc.SearchVendors {
		snap.SearchVendors = append(snap.SearchVendors, SearchVendor{ID: v.ID, Name: v.Name, KeysURL: v.KeysURL, NeedURL: v.NeedURL})
	}
	for _, ch := range doc.SearchChoices {
		j := SearchChoice{ID: ch.ID, Name: ch.Name, Icon: ch.Icon, Small: ch.Small, SmallNam: ch.SmallName, Service: ch.Service, Own: ch.Own}
		for _, m := range ch.Models {
			j.Models = append(j.Models, ModelRef{ID: m.ID, Name: m.Name, Prov: m.Provider, PName: m.PName, Icon: m.Icon})
		}
		snap.SearchChoices = append(snap.SearchChoices, j)
	}
	snap.Terminals = doc.TerminalApps
	snap.TerminalDefault = doc.TerminalDefault
	return snap, true
}

// remoteModelRefs maps the document's models as the pickers offer them.
func remoteModelRefs(in []remoteModelRef) []ModelRef {
	out := make([]ModelRef, 0, len(in))
	for _, m := range in {
		out = append(out, ModelRef{ID: m.ID, Name: m.Name, Prov: m.Provider, PName: m.PName, Icon: m.Icon})
	}
	return out
}

// settingsSaveRemote saves the settings the way magpie's own page does:
// the instance's current document read, the page's change made on it,
// and the whole document posted back — its server keeps what its other
// pages own, and checks the model choices against its own lists.
func settingsSaveRemote(c *magpieclient.Client, mutate func(s *settings.Settings)) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	doc, err := settingsDocRemote(ctx, c)
	if err != nil {
		return err
	}
	mutate(&doc.Settings)
	return c.Do(ctx, "POST", "/api/settings", &doc.Settings, nil)
}

// settingsSearchAPI sets a web search API, or takes one away with
// remove, as the list's rows do — magpie's settings/search-api.
//
// The experiment's seam (docs/08 M4): when the environment points at a
// magpie web instance the change is asked of it; any remote failure
// falls back to the embedded change below.
func settingsSearchAPI(vendor, key, url string, remove bool) error {
	if c := remoteGateway(); c != nil {
		in := struct {
			Vendor string `json:"vendor"`
			Key    string `json:"key"`
			URL    string `json:"url"`
			Remove bool   `json:"remove"`
		}{vendor, key, url, remove}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := c.Do(ctx, "POST", "/api/settings/search-api", &in, nil); err == nil {
			return nil
		}
	}
	if remove {
		return provider.RemoveSearchAPI(vendor)
	}
	return provider.SetSearchAPI(provider.SearchAPI{Vendor: vendor, Key: key, URL: url})
}

// searchAPISet adds a search API, given a new key or address.
func searchAPISet(vendor, key, url string) error {
	return settingsSearchAPI(vendor, key, url, false)
}

// searchAPIRemove takes one search API away.
func searchAPIRemove(vendor string) error {
	return settingsSearchAPI(vendor, "", "", true)
}

// settingsLogin turns opening at login on or off, as the switch does —
// magpie's settings/login.
//
// The experiment's seam (docs/08 M4): when the environment points at a
// magpie web instance the change is asked of it; any remote failure
// falls back to the embedded change below.
func settingsLogin(on bool) error {
	if c := remoteGateway(); c != nil {
		in := struct{ On bool `json:"on"` }{on}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := c.Do(ctx, "POST", "/api/settings/login", &in, nil); err == nil {
			return nil
		}
	}
	return autostart.Set(on)
}

// settingsLAN shares the gateway on the local network, or stops, as the
// switch does — magpie's settings/lan, which saves the sharing and its
// key and moves the gateway it serves onto it.
//
// The experiment's seam (docs/08 M4): when the environment points at a
// magpie web instance the change is asked of it; any remote failure
// falls back to the embedded change below.
func settingsLAN(on, newKey bool) error {
	if c := remoteGateway(); c != nil {
		in := struct {
			On     bool `json:"on"`
			NewKey bool `json:"newKey"`
		}{on, newKey}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := c.Do(ctx, "POST", "/api/settings/lan", &in, nil); err == nil {
			return nil
		}
	}
	if err := access.ConfigureLAN(on, newKey); err != nil {
		return err
	}
	if gw := gui.ServedServer(); gw != nil {
		return gw.Relisten()
	}
	return nil
}

// settingsPort moves the gateway to the port, as the field's apply does
// — magpie's settings/port, which moves every agent connected with it.
// The answer's port part is what the move did: some agents moved, one
// didn't, said not failed.
//
// The experiment's seam (docs/08 M4): when the environment points at a
// magpie web instance the move is asked of it; any remote failure falls
// back to the embedded move below.
func settingsPort(p int) (gui.PortResult, error) {
	if c := remoteGateway(); c != nil {
		in := struct{ Port int `json:"port"` }{p}
		out := struct {
			Port gui.PortResult `json:"port"`
		}{}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := c.Do(ctx, "POST", "/api/settings/port", &in, &out); err == nil {
			return out.Port, nil
		}
	}
	return gui.SetPort(p)
}

// searchKeyReveal reads one search API's saved key, as the row's Show
// button does — magpie's settings/search-key, whose answer is the one
// secret the row shows, on the reader's click.
//
// The experiment's seam (docs/08 M4): when the environment points at a
// magpie web instance the key is asked of it; any remote failure falls
// back to the embedded read below.
func searchKeyReveal(vendor string) (string, error) {
	if c := remoteGateway(); c != nil {
		in := struct{ Vendor string `json:"vendor"` }{vendor}
		out := struct {
			Key string `json:"key"`
		}{}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := c.Do(ctx, "POST", "/api/settings/search-key", &in, &out); err == nil {
			return out.Key, nil
		}
	}
	for _, a := range provider.StoredSearchAPIs() {
		if a.Vendor == vendor {
			return a.Key, nil
		}
	}
	return "", nil
}
