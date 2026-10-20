//go:build nogui && echo

package shell

import (
	"os"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/yetone/magpie/internal/autostart"
	"github.com/yetone/magpie/internal/catalog"
	"github.com/yetone/magpie/internal/gateway"
	"github.com/yetone/magpie/internal/gui"
	"github.com/yetone/magpie/internal/library"
	"github.com/yetone/magpie/internal/netproxy"
	"github.com/yetone/magpie/internal/provider"
	"github.com/yetone/magpie/internal/settings"
)

// appVersion is the version the pages show: the packaged app's own — mygo
// build links it out of mygo.json, so the About page says what was shipped
// — and the build's stamp when the app is not packaged.
func appVersion() string {
	if v := mygo.App.Version(); v != "" {
		return v
	}
	return gui.Version
}

// ModelRef names one model a picker offers: its id, what to call it, and
// the provider (or routing group) it is on.
type ModelRef struct {
	ID    string
	Name  string
	Prov  string // the provider's id, "" for a routing group
	PName string // the provider's or group's name
	Icon  string
}

// SearchChoice is a provider that can search the web for a model that
// can't, as the Searcher picker offers it.
type SearchChoice struct {
	ID       string
	Name     string
	Icon     string
	Small    string
	SmallNam string
	Models   []ModelRef
	Service  bool // a Kimi Code plan: its web search, no model
	Own      bool // a Google sign-in: its own models first
}

// SearchAPIRow is one search API set, as its row shows it.
type SearchAPIRow struct {
	Vendor string
	Name   string
	Key    string // masked
	URL    string
	Ready  bool
}

// SearchVendor is one search API the add row can add.
type SearchVendor struct {
	ID      string
	Name    string
	KeysURL string
	NeedURL bool // one the user runs
}

// SettingsSnap is what the Settings page draws: the saved settings and the
// facts the rows show beside them. Read off the main thread by readData,
// and again after each write, as magpie's settingsState is.
type SettingsSnap struct {
	S settings.Settings

	Version  string
	Dir      string
	Portable bool
	Gateway  string
	AddrEnv  string

	Login bool

	ProxyNow    string
	ProxySource string

	LANURLs      []string
	LANContainer bool

	FX gui.FX

	GHFrom string
	GHMask string

	Update gui.UpdateInfo

	CodexUsers   []string
	CodexWarmed  *time.Time
	ClaudeWarmed *time.Time

	WorkBuddy       bool
	WBCheckins      []provider.WorkBuddyCheckin
	Trae            bool
	TraeCheckins    []provider.WorkBuddyCheckin
	MiniMax         bool
	MiniMaxCheckins []provider.WorkBuddyCheckin
	Qoder           bool
	QoderCheckins   []provider.WorkBuddyCheckin

	VisionAuto   string
	VisionModels []ModelRef

	TitleModels []ModelRef

	ImageGenAuto   string
	ImageGenModels []ModelRef

	SearchAuto     string
	SearchUnused   string
	SearchProvider string
	SearchChoices  []SearchChoice
	SearchRelays   []string
	SearchLeftOut  []string
	SearchAPIs     []SearchAPIRow
	SearchVendors  []SearchVendor

	Terminals       []gui.TerminalApp
	TerminalDefault string

	TrayCards []TrayCard
}

// readSettings gathers what the Settings page draws, as magpie's
// settingsState does. It is called off the main thread.
func readSettings() SettingsSnap {
	s := settings.Load()
	snap := SettingsSnap{
		S:           s,
		Version:     appVersion(),
		Dir:         tildePath(settings.Dir()),
		Portable:    settings.Portable() != "",
		Gateway:     gateway.URL(),
		AddrEnv:     os.Getenv("MAGPIE_ADDR"),
		Login:       autostart.Enabled(),
		ProxyNow:    proxyNowText(),
		ProxySource: proxySource(),
		FX:          gui.FXNow(),
		Update:      gui.UpdateState(),
	}
	if tok, from := library.GitHubToken(); tok != "" {
		snap.GHMask, snap.GHFrom = provider.Mask(tok), from
	}
	snap.LANURLs, snap.LANContainer = gateway.LANURLs(), gateway.ContainerAddrs()
	snap.CodexUsers = gui.CodexUsers()
	if ts := provider.CodexWarmed(); len(ts) > 0 {
		snap.CodexWarmed = latestTime(ts)
	}
	if ts := provider.ClaudeWarmed(); len(ts) > 0 {
		snap.ClaudeWarmed = latestTime(ts)
	}
	snap.WorkBuddy, snap.WBCheckins = provider.HasWorkBuddy(), provider.WorkBuddyCheckins()
	snap.Trae, snap.TraeCheckins = provider.HasTrae(), provider.TraeCheckins()
	snap.MiniMax, snap.MiniMaxCheckins = provider.HasMiniMax(), provider.MiniMaxCheckins()
	snap.Qoder, snap.QoderCheckins = provider.HasQoder(), provider.QoderCheckins()

	names := map[string]string{}
	for _, e := range provider.Served() {
		names[e.ID] = cmpOr(e.Name, e.Model)
	}
	snap.VisionAuto, snap.VisionModels = gateway.AutoVision(), visionModels()
	snap.TitleModels = titleModels()
	snap.ImageGenAuto, snap.ImageGenModels = gateway.AutoDrawer(), imageGenModels()
	readSearch(&snap, names)

	if found, err := gui.DiscoverTerminals(); err == nil {
		snap.Terminals, snap.TerminalDefault = found.Apps, found.Default
	}
	snap.TrayCards = trayCardsCached()
	return snap
}

// trayCardsCached is what the quota cache already holds as the picker's
// cards: never a read, which a frame must not wait on. An empty cache is
// filled behind the page instead (settingsAllowances).
func trayCardsCached() []TrayCard {
	out := []TrayCard{}
	for _, q := range PeekQuotas() {
		user := q.User
		if user == "" {
			user = "*"
		}
		name := q.Name
		if q.User != "" {
			name += " · " + q.User
		}
		out = append(out, TrayCard{ID: q.Provider + "|" + user, Name: name, Icon: q.Icon})
	}
	return out
}

// readSearch fills the snap's search rows and choices, as magpie's
// searchState does.
func readSearch(snap *SettingsSnap, names map[string]string) {
	snap.SearchAPIs, snap.SearchVendors = []SearchAPIRow{}, []SearchVendor{}
	for _, a := range provider.StoredSearchAPIs() {
		row := SearchAPIRow{Vendor: a.Vendor, Name: a.Name(), URL: a.URL, Ready: a.Ready()}
		if a.Key != "" {
			row.Key = provider.Mask(a.Key)
		}
		snap.SearchAPIs = append(snap.SearchAPIs, row)
	}
	for _, v := range provider.SearchVendors {
		snap.SearchVendors = append(snap.SearchVendors, SearchVendor{ID: v.ID, Name: v.Name, KeysURL: v.KeysURL, NeedURL: v.Base == ""})
	}
	snap.SearchProvider = gateway.Searcher()
	snap.SearchAuto, snap.SearchUnused = gateway.AutoSearcher(), gateway.SearcherUnused()
	snap.SearchChoices = []SearchChoice{}
	for _, ch := range gateway.Searchers() {
		j := SearchChoice{ID: ch.Provider.ID, Name: ch.Provider.Name, Icon: ch.Provider.Icon,
			Small: ch.Small, SmallNam: names[ch.Provider.ID+"/"+ch.Small], Service: ch.Service, Own: ch.Own}
		for _, m := range ch.Models {
			id := ch.Provider.ID + "/" + m.ID
			j.Models = append(j.Models, ModelRef{ID: id, Name: cmpOr(names[id], m.Name, m.ID), Prov: ch.Provider.ID, PName: ch.Provider.Name, Icon: ch.Provider.Icon})
		}
		snap.SearchChoices = append(snap.SearchChoices, j)
	}
	for _, p := range gateway.RelaysSaidToSearch() {
		snap.SearchRelays = append(snap.SearchRelays, p.Name)
	}
	for _, p := range provider.All() {
		if !p.On() {
			continue
		}
		known := false
		for _, c := range snap.SearchChoices {
			if c.ID == p.ID {
				known = true
				break
			}
		}
		if known {
			continue
		}
		left := false
		for _, n := range snap.SearchRelays {
			if n == p.Name {
				left = true
				break
			}
		}
		if !left {
			snap.SearchLeftOut = append(snap.SearchLeftOut, p.Name)
		}
	}
}

// visionModels are the models that take images, as the Image recognition
// picker offers them.
func visionModels() []ModelRef {
	out := []ModelRef{}
	for _, e := range provider.Served() {
		if !e.Images || (e.ImageInput != nil && !*e.ImageInput) {
			continue
		}
		if e.Group == "" && !e.Provider.Ready() {
			continue
		}
		m := ModelRef{ID: e.ID, Name: e.Name, Prov: e.Provider.ID, PName: e.Provider.Name, Icon: e.Provider.Icon}
		if e.Group != "" {
			m.Prov, m.PName = "", e.Group
		}
		out = append(out, m)
	}
	return out
}

// titleModels are the models a title or reviewer may be named: every one
// served, a routing group's included.
func titleModels() []ModelRef {
	out := []ModelRef{}
	for _, e := range provider.Served() {
		if e.Group == "" && !(e.Provider.Ready() && !e.Provider.DecideOnly()) {
			continue
		}
		m := ModelRef{ID: e.ID, Name: e.Name, Prov: e.Provider.ID, PName: e.Provider.Name, Icon: e.Provider.Icon}
		if e.Group != "" {
			m.Prov, m.PName = "", e.Group
		}
		out = append(out, m)
	}
	return out
}

// imageGenModels are the models the image tool can draw with.
func imageGenModels() []ModelRef {
	out := []ModelRef{}
	for _, p := range provider.All() {
		if !p.On() || p.DecideOnly() {
			continue
		}
		for _, m := range gateway.Drawers(p) {
			name := m.Name
			if name == "" {
				name = m.ID
			}
			out = append(out, ModelRef{ID: p.ID + "/" + m.ID, Name: name, Prov: p.ID, PName: p.Name, Icon: p.Icon})
		}
	}
	return out
}

// latestTime is the latest of ts, nil when there is none.
func latestTime(ts map[string]time.Time) *time.Time {
	var out *time.Time
	for _, t := range ts {
		if out == nil || t.After(*out) {
			out = &t
		}
	}
	return out
}

// cmpOr is cmp.Or: the first non-empty string.
func cmpOr(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// tildePath shortens the home directory in a path shown, as magpie's
// settingsJSON does.
func tildePath(p string) string {
	if home, err := os.UserHomeDir(); err == nil && home != "" && strings.HasPrefix(p, home) {
		return "~" + strings.TrimPrefix(p, home)
	}
	return p
}

// proxySource is where the proxy row says the requests go through.
func proxySource() string {
	_, source := netproxy.Describe()
	return source
}

// proxyNowText is the proxy in use, as the row's line names it.
func proxyNowText() string {
	p, _ := netproxy.Describe()
	return p
}

// saveSettings applies mutate to the saved settings, preserving what other
// pages own the way magpie's POST /api/settings does, and runs the
// follow-ups a change asks for. It must run off the main thread.
func saveSettings(mutate func(s *settings.Settings)) error {
	cur := settings.Load()
	in := cur
	mutate(&in)
	// what other pages keep here: the agents' arrangement, the window's
	// size, the per-model maps, and everything set on its own elsewhere.
	// CarryPerModel carries the maps whole rather than named one by one,
	// so a map added later is not silently dropped here.
	in.AgentOrder, in.AgentsHidden, in.AgentsShown = cur.AgentOrder, cur.AgentsHidden, cur.AgentsShown
	in.Window, in.WindowMaximised = cur.Window, cur.WindowMaximised
	in.Visible, in.HiddenModels, in.OrderedModels = cur.Visible, cur.HiddenModels, cur.OrderedModels
	in.FastPicks = cur.FastPicks
	settings.CarryPerModel(&in, &cur)
	in.LAN, in.LANKey = cur.LAN, cur.LANKey
	in.LANKeyID = cur.LANKeyID
	in.Port = cur.Port
	in.GitHubToken = cur.GitHubToken
	in.RequestArchive = cur.RequestArchive
	in.RequestArchiveMaxMB = cur.RequestArchiveMaxMB
	in.RedactRules = cur.RedactRules
	in.QuotaLeft = cur.QuotaLeft
	in.UsageOrder = cur.UsageOrder
	in.PlainNames, in.PlainOwnNames = cur.PlainNames, cur.PlainOwnNames
	in.CodexAgentsV1 = cur.CodexAgentsV1
	in.FullContext = cur.FullContext
	in.CompactAt = cur.CompactAt
	in.CodexTitles = cur.CodexTitles
	in.CodexAutoReview = cur.CodexAutoReview
	in.ChinaMirror = cur.ChinaMirror
	in.CodexAutoReset = cur.CodexAutoReset
	in.CodexNoCredits = cur.CodexNoCredits
	in.CodexWarmAtOf = cur.CodexWarmAtOf
	in.TextSize = cur.TextSize
	in.UpdateSkip = cur.UpdateSkip
	in.UpdateMirror = cur.UpdateMirror
	in.GatewayMode = cur.GatewayMode
	if err := settings.Save(in); err != nil {
		return err
	}
	// whether agents are told every model takes images follows Vision
	if strings.TrimSpace(in.Vision) != strings.TrimSpace(cur.Vision) {
		catalog.Touched()
	}
	return nil
}

// settingsWrite runs a settings write off the main thread and, when it
// lands, reads the page again on it. done, when set, runs on the main
// thread with the error before the read, so a row can keep what was typed
// when the write refused it.
func (a *App) settingsWrite(write func() error, done func(error)) {
	go func() {
		err := write()
		if a.win == nil {
			return
		}
		a.win.Update(func() {
			if done != nil {
				done(err)
			}
			a.Reload()
		})
	}()
}
