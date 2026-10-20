//go:build nogui && echo

// Package shell is Echo's desktop shell: magpie's gateway and domain packages
// behind an interface MyGo draws itself.
//
// It exists because Go forbids importing a package main: the shipped app
// (cmd/echo) and the acceptance probe (cmd/echoprobe) share this code.
//
// The interface is native UI (package ui): no HTML, no JavaScript and no
// webview. magpie's own page is not reused — Echo reimplements the product in
// MyGo — while everything below the interface is magpie's, unchanged.
//
// Everything here is built with -tags nogui, because magpie's internal/gui
// only compiles without Wails under that tag, and under it magpie's own GUI
// entry point is absent — which is what this package replaces.
//
// The domain packages (internal/gateway, internal/provider, …) are untouched;
// see docs/04-plan-option-a-shell-migration.md for the scope boundary.
package shell

import (
	"os"
	"sync"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/yetone/magpie/internal/library"
	"github.com/yetone/magpie/internal/plugin"
	"github.com/yetone/magpie/internal/provider"
	"github.com/yetone/magpie/internal/sessions"
	"github.com/yetone/magpie/internal/settings"
	"github.com/yetone/magpie/internal/usage"
	"github.com/yetone/magpie/internal/usageapi"
)

// LibTab identifies one of the sub-tabs under the Library page.
type LibTab string

const (
	LibTabInstructions LibTab = "instructions"
	LibTabMCP          LibTab = "mcp"
	LibTabSkills       LibTab = "skills"
	LibTabHooks        LibTab = "hooks"
	LibTabSubagents    LibTab = "subagents"
	LibTabRTK          LibTab = "rtk"
)

// Page identifies one of Echo's pages, the entries of the navigation.
type Page string

// The pages, in the order the navigation lists them: magpie's own views.
const (
	PageAgents    Page = "agents"
	PageSessions  Page = "sessions"
	PageLibrary   Page = "library"
	PageProviders Page = "providers"
	PageRouting   Page = "routing"
	PageGateway   Page = "gateway"
	PageUsage     Page = "usage"
	PagePlugins   Page = "plugins"
	PageSettings  Page = "settings"
)

// SetTab identifies one of the tabs under the Settings page.
type SetTab string

const (
	SetTabGeneral SetTab = "general"
	SetTabUsage   SetTab = "usage"
	SetTabNetwork SetTab = "network"
	SetTabModels  SetTab = "models"
	SetTabPrivacy SetTab = "privacy"
	SetTabOTel    SetTab = "otel"
	SetTabSync    SetTab = "sync"
	SetTabNotify  SetTab = "notify"
	SetTabAbout   SetTab = "about"
)

// PageInfo describes one page: its id, its title and the group it belongs to.
type PageInfo struct {
	Page  Page
	Title string
	Group Group
}

// Group is a heading of the sidebar, as magpie's navigation groups its pages.
type Group string

// The groups, in the order the sidebar lists them.
const (
	GroupWork    Group = "work"
	GroupModels  Group = "models"
	GroupInsight Group = "insights"
	GroupExtend  Group = "extend"
)

// App is Echo's state: what the interface is drawn from.
//
// Every field is read and changed on the main thread, by the view and by the
// callbacks MyGo runs there, so it needs no lock. Data that arrives from a
// goroutine reaches it through Window.Update, which runs its function on the
// main thread.
type App struct {
	win  *mygo.Window
	tray *mygo.Tray

	// The tray panel: the popover the menu bar icon opens (tray_panel.go),
	// with its tab and its own read of the allowance cards.
	trayPanel   *mygo.Window
	panelTab    string
	panelQuotas []provider.SubscriptionQuota
	panelReadAt time.Time
	panelStop   chan struct{} // ends the open panel's timed reads

	panelRefreshing bool // the panel's refresh button is reading the vendors
	panelAllAgents  bool // the panel lists every agent, not the first few

	// lang is the language the interface is drawn in, and text its strings.
	lang Lang
	text T

	// page is the page being shown.
	page Page

	// data is what the pages draw, read from magpie's domain packages.
	data Data

	// loading is set while a read of data is in flight.
	loading bool

	// err is the last read's failure, shown instead of the data.
	err error

	// sidebarOpen holds each group's expanded state, keyed by group.
	sidebarOpen map[Group]bool

	// sidebarHidden is true when the sidebar is collapsed by the user.
	sidebarHidden bool

	// quitting is set while the app is ending, so the window's close
	// handler lets the close through instead of hiding the window.
	quitting bool

	// tables holds each page's table state: the columns' order and widths,
	// the sort, and which row is chosen. MyGo keeps such state across
	// frames by the pointer it is given, so one per page must outlive the
	// frame that built it.
	tables map[Page]*ui.ListState

	// The Providers page's own state, as magpie's page holds its own:
	// the search's words (pvQuery), its own look for other apps'
	// configurations (pvDiscovery, drawn by provider_discovery.go), the
	// last time it looked (pvDiscoveryAt, so the minute's cadence holds
	// across reloads), and the two dialogs' open state: the import
	// picker's (pvImporting) and the add sheet's (pvAdding), which holds
	// the editor a pick opens and the sign-in one is started by.
	pvQuery       string
	pvFilter      int // the segments: 0 all, 1 the ones on, 2 the ones off
	pvDiscovery   discovery
	pvDiscoveryAt time.Time
	pvImporting   *importState
	pvAdding      *addSheet
	pvSignOut     *signOutAsk
	pvSignIn      *signIn
	pvAsk         *pluginAsk // a moved subscription's plugin sign-in, its step before the browser
	pvEdit        *providerEdit
	pvDiscard     bool // the editor's ask before its changes are thrown away

	// The Gateway page's own state, as magpie's page holds its own:
	// which API the snippets speak (gwFlavor), which snippet language
	// (gwLang), the model chosen for the snippets (gwModel), the model
	// list's search words (gwModelQuery), whether Connect and the model
	// list are folded (gwConnectFolded, gwModelsFolded), and how many
	// recent calls are shown (gwCallsShown).
	gwFlavor        int    // 0 openai, 1 responses, 2 anthropic, 3 gemini, 4 systemone
	gwLang          int    // 0 shell, 1 curl, 2 python, 3 node
	gwModel         string // the model id chosen for the snippets
	gwModelQuery    string // the model list's search words
	gwConnectFolded bool
	gwModelsFolded  bool
	gwCallsShown    int  // how many recent calls are shown
	gwFixBusy       bool // true while a restart/start is in flight

	// The Agents page's own state:
	agExpanded      string          // agent id of currently expanded row, "" if none
	agFolded        bool            // true if remaining agents beyond active ones are folded
	agInstallFolded bool            // true if uninstalled agents section is folded
	agModalAgentID  string          // agent id for which the model/tier modal dialog is open, "" if closed
	agModalSearch   string          // search filter inside model modal
	agModalFilter   int             // 0: all, 1: enabled/shown, 2: order
	agModalGroup    string          // selected group in left rail ("" = all)
	agModalFolded   map[string]bool // which groups are collapsed in right list
	agModalOnly     bool            // whether "only picked models" switch is active

	// The Library page's own state:
	libTab          LibTab
	libMenuOpen     bool
	libSearch       string
	libSort         int // 0: source/repo, 1: A-Z, 2: Z-A
	libFoldedGroups map[string]bool
	libDetailSkill  string
	libDetailText   string
	libDetailPath   string
	libDetailBusy   bool
	libProbeSource  string
	libProbeBusy    bool
	libProbeResult  *library.Probe
	libProbePicks   map[string]bool
	libProbeAgents  map[string]bool
	libProbeFilter  string
	libProbeErr     string
	libCheckBusy    bool
	libUpdateBusy   map[string]bool
	libRegroupBusy  bool
	libRemoveAsk    string
	libUpdateAsk    string
	libMarketQuery  string
	libMarketSkills []library.MarketSkill
	libMarketBusy   bool
	libMarketErr    string
	libMarketAgents map[string]bool
	libNotice       string
	// libProblems are what the last change couldn't write, kept across
	// the reload that follows it so the page's card can say so: magpie
	// keeps them for the same reason and shows them in the same card.
	libProblems []library.Problem

	// The Skills page's own state, as magpie's page holds its own: the
	// filter over the list (libSearch), which groups are folded
	// (libFoldedGroups), the picker's open state and its picks, the
	// install card's source and what it found there, the market's search,
	// and the reader a row opens.
	skInstalling  bool                  // the install card is looking
	skInstallErr  string                // what the look said, when it failed
	skProbe       *library.Probe        // what the look found
	skProbeSrc    string                // the source it looked at
	skProbePicks  map[string]bool       // candidate path -> picked
	skProbeAgents map[string]bool       // agent id -> picked
	skProbeFilter string                // the probe list's own filter
	skBusy        string                // the write in flight, by name
	skCheckBusy   bool                  // a check for updates is running
	skRegroupBusy bool                  // the groups are being traced again
	skReader      string                // the skill whose SKILL.md shows
	skReaderText  string                // its text, once read
	skReaderErr   string                // why it couldn't be read
	skReaderPath  string                // where its folder is
	skAskRemove   string                // the skill a Remove asks about
	skAskReplace  string                // the skill an update would replace
	skAskHow      bool                  // the "how each agent" sheet is open
	skMarketQuery string                // the market's search words
	skMarketItems []library.MarketSkill // what it found
	skMarketBusy  bool                  // the market is searching
	skMarketErr   string                // what it said, when it failed
	skMarketFetch bool                  // the market was asked once

	// The picking, as magpie's own holds it: a box on every row, a bar
	// under the list, and the bar's own step that asks a group's name.
	skPicking bool            // the rows have a box each, to pick some
	skPicked  map[string]bool // skill name -> picked
	skNaming  bool            // the pick bar asks a group's name
	skName    string          // the name being typed for it

	// The sheets the page opens over it, each named by what it is about.
	skAskRemoveAll  bool        // the "remove every skill" ask
	skByAgent       bool        // "Skills by agent"
	skRename        string      // the group whose name is a box to type a new one
	skRenameTo      string      // what is typed in it
	skMarketSheet   string      // the market skill whose sheet shows, by id
	skEverySkillOff bool        // the "turn every skill off" ask
	skMarketTimer   *time.Timer // the market's ask, waiting for typing to pause

	// The projects under the list: the folder being typed, which of them
	// are open, and the one a remove asks about.
	skAddingProject    bool
	skProjectDir       string
	skProjectErr       string
	skOpenProjects     map[string]bool
	skAskRemoveProject string
	skRemoveKeep       bool

	// The Plugins page's own state, as magpie's page holds its own: the
	// tab it shows (plugTab), the head's search words (plugQuery), the
	// 「国内镜像」 switch's write (plugMirrorBusy), the write in flight by
	// what it is (plugBusy: "add:<spec>", "off:<spec>", …), what the last
	// one said (plugNotice, plugNoticeErr), and the check for updates.
	plugTab        PluginTab
	plugQuery      string
	plugMirrorBusy bool
	plugBusy       string
	plugNotice     string
	plugNoticeErr  bool
	plugChecking   bool
	plugCheckedAt  time.Time
	plugCheckErr   string
	plugChecks     *plugCheckResult

	// The market, asked once when Discover first shows (plugins_market.go):
	// what magpie suggests and what npm says of each — then the
	// repositories tagged on GitHub, which are asked apart because GitHub
	// answers last, and the npm search under the list, and the section's
	// own filter.
	plugMarket      *plugMarketData
	plugMarketBusy  bool
	plugMarketErr   string
	plugTagged      []plugin.Tagged
	plugTaggedBusy  bool
	plugTaggedErr   string
	plugTaggedDone  bool
	plugHeads       map[string]string
	plugGhKind      string // "" all, else the listing kind picked
	plugGhKindIdx   int
	plugGhHave      string // "" all, "have" installed, "none" not
	plugGhHaveIdx   int
	plugGhQuery     string
	plugSearchQ     string       // what the npm search was asked
	plugSearch      []plugin.Hit // what it found
	plugSearchErr   string
	plugSearchBusy  bool
	plugSearchTimer *time.Timer

	// The manual install: what is typed, whether it is being written, and
	// whether Bun is being fetched (the first plugin needs it).
	plugManual     string
	plugManualBusy bool
	plugGettingBun bool

	// The questions the page asks before a write, and the options editor:
	// the spec each is about, what is typed in it and what was wrong.
	plugAskRemove string
	plugAskOff    string
	// plugAskMove is the plugin just switched on again, whose row now asks
	// whether a built-in's accounts move onto it (magpie's move-again ask).
	plugAskMove   string
	plugClearAsk  string
	plugRemoveAsk string
	// The host's own answer on subscriptions two plugins sign in to, asked
	// once when the page first shows (plugHostKick): by spec, whose they are
	// and which plugin runs them.
	plugClashes  map[string][]PluginClashRow
	plugHostBusy bool
	plugHostDone bool
	plugOptions  string
	plugOptsText string
	plugOptsErr  string

	// How the installed list is ordered (0 A→Z, 1 Z→A), as magpie's
	// row-head sort picks (sort.js).
	plugSort int

	// The sheet a plugin's own page shows: which one, its README once
	// read, why it couldn't be, when it last changed, and a heading of the
	// README a link in it named, which the next frame brings to the top.
	plugDetail       *plugDetail
	plugDetailText   string
	plugDetailErr    string
	plugDetailBusy   bool
	plugDetailUpd    string
	plugDetailAnchor string

	// The Routing page's own state:
	rtHideAccounts bool
	rtTraceFilter  int // 0 all, 1 succeeded, 2 rerouted, 3 errors
	rtSearch       string
	rtGroupModal   *rtGroupModalState
	rtImportModal  *rtImportModalState
	rtTraceDetail  *RoutingTraceRow
	rtDeleteGroup  string
	rtNotice       string
	rtNoticeErr    bool
	rtNoticeTimer  *time.Timer
	rtBusy         bool // the stage's own read is in flight
	// the stage's own read of the gateway's trace, which it makes while it
	// is on screen: the page is live, and a stage that never read again
	// would fly requests that ended minutes ago (routing_flow.go)
	rtFlowAt time.Time

	// Usage page state
	//
	// The page is magpie's Usage page drawn natively: the four segments
	// (usTab), the period its head picked, and every filter the Requests
	// and Sessions segments offer. Each is a field of a, as magpie's page
	// keeps its own across a redraw, and usageQuery turns them into the
	// one value the reads are keyed by.
	usTab              string
	usTabIdx           int
	usPeriod           usage.Period
	usPeriodIdx        int
	usSelectedDay      string
	usAgentFilter      string
	usProviderFilter   string
	usPurposeFilter    string
	usAccountFilter    string
	usKeyFilter        string
	usModelFilter      string
	usViaFilter        string
	usStatusFilter     string
	usStatusIdx        int
	usSearchQuery      string
	usSearchTimer      *time.Timer // the search's pause before it reads again
	usHideAccounts     bool
	usQuotaLeft        bool // the meters show what is left, not what is used
	usQuotaFold        bool // the allowance cards are folded away
	usWithCache        bool // the token totals count the cache's tokens
	usLedOffset        int
	usLedMetric        int             // the trend's metric: tokens, cost, requests, speed
	usSplit            int             // the trend's split: model, provider, agent
	usHeatMetric       int             // the heatmap's metric: tokens, requests, cost
	usFolds            map[string]bool // the Overview's lists folded away
	usHiddenAgents     map[string]bool // agents hidden from the Overview's list
	usHiddenOpen       bool            // the hidden agents' line is open
	usSessRange        string
	usSessRangeIdx     int
	usSessAgent        string
	usSessAgentIdx     int
	usSessModel        string
	usSessFolder       string
	usSessQuery        string
	usSessMetric       int    // the activity chart's metric
	usSessTop          int    // the top sessions' measure: tokens, cost, active
	usSessShape        int    // the session shape's telling: messages, minutes, autonomy
	usSessListed       int    // how many sessions the list shows
	usSessOpen         string // the session whose details are open
	usSessOne          *sessions.Session
	usSessTrans        *sessions.Transcript
	usSessLoading      bool
	usSessErr          string
	usCtxDays          int
	usCtxDaysIdx       int
	usCtxAgent         string
	usBusy             bool   // a read of the page is in flight
	usGen              int    // the read whose answer is still wanted
	usFresh            string // the query the data held was read for
	usPending          string // the query a read in flight is for
	usExporting        bool
	usNotice           string
	usNoticeErr        bool
	usNoticeTimer      *time.Timer
	usInspectRow       *usageapi.Row
	usInspectContent   *ContentJSON
	usInspectLoading   bool
	usResetConfirmUser string
	usEveryModel       map[string]bool // provider|user draws each window of its own, not a row a family

	// Settings page state
	setTab               SetTab
	setWarmTab           string // "codex", "claude", "wb", "trae", "minimax", "qoder"
	setPrevPage          Page   // the page open before Settings was opened
	setCustomProxy       string
	setCustomProxyOpen   bool
	setUsagePctDraft     string // the allowance alert's own percent, as typed
	setBalDraft          string // the balance alert's own amount, as typed
	setTrayReading       bool   // the vendors are being asked for the tray cards
	setCompactAtDraft    string
	setPortDraft         string
	setPortErr           string
	setGHTokenDraft      string
	setGHErr             string
	setSearchVendorName  string // the search API add row's vendor, by name
	setSearchKey         string
	setSearchURL         string
	setSearchErr         string
	setRuleKind          string
	setRuleBy            int // 0 prefix, 1 regex
	setRuleMatch         string
	setRuleErr           string
	setWordsDraft        string
	setOTelHeadersDraft  string
	setOTelEndpointDraft string
	setRevealedKey       map[string]string // search vendor -> its revealed key, "" hidden
	setNotice            string
	setNoticeErr         bool
	setNoticeTimer       *time.Timer
	setPicker            *settingsPickerState
	setSyncOpen          string // "" | "dav" | "export" | "import"
	setSyncKind          string // "webdav" | "s3"
	setSyncURL           string
	setSyncUser          string
	setSyncPass          string
	setSyncPhrase        string
	setSyncKeys          bool
	setSyncAgents        bool
	setSyncLib           bool
	setSyncUsage         bool
	setSyncEndpoint      string
	setSyncBucket        string
	setSyncPrefix        string
	setSyncRegion        string
	setSyncKeyID         string
	setSyncSecret        string
	setSyncPathStyle     bool
	setExportPass1       string
	setExportPass2       string
	setExportKeys        bool
	setExportLib         bool
	setExportResult      string
	setImportPath        string
	setImportData        string
	setImportPass        string
	setImportAgents      bool
	setImportLib         bool
	setImportResult      string
	setSyncRestore       bool
	setSyncBusy          bool
	setSyncErr           string
}

// importState is the Import from other apps dialog's own state, as
// magpie's importingApps holds its own: the sources as they were read
// (keys masked), which are picked, and which app's tab is showing.
type importState struct {
	sources []provider.AppSource
	picks   map[string]bool // "source\nref" -> picked
	tab     string
}

// tableState returns the table state of a page, making it on first use.
//
// It is keyed by page so that each page keeps its own column widths, sort and
// choice: switching pages and back must not lose them.
func (a *App) tableState(p Page) *ui.ListState {
	if a.tables == nil {
		a.tables = map[Page]*ui.ListState{}
	}
	s, ok := a.tables[p]
	if !ok {
		s = &ui.ListState{}
		a.tables[p] = s
	}
	return s
}

// mu guards the app pointer itself: the tray's callbacks and the window's can
// run on the main thread while a goroutine is asking for a reload.
var (
	mu  sync.Mutex
	app *App
)

// Current returns the running app, or nil before it is built.
func Current() *App {
	mu.Lock()
	defer mu.Unlock()
	return app
}

// setCurrent records the running app, called on the main thread.
func setCurrent(a *App) {
	mu.Lock()
	app = a
	mu.Unlock()
}

// Pages returns every page with its title in the app's language, in the order
// the navigation lists them.
func (a *App) Pages() []PageInfo {
	t := a.text.S
	out := make([]PageInfo, 0, len(allPages))
	for _, p := range allPages {
		// a page a built-in plugin switched off has leaves the navigation,
		// as it does magpie's
		if a.data.Plugins.OffPages[string(p.page)] {
			continue
		}
		out = append(out, PageInfo{Page: p.page, Title: p.title(t), Group: p.group})
	}
	return out
}

// pageSpec is one page before it is given a title: its id, its group, and how
// its title is read in the language being drawn in.
type pageSpec struct {
	page  Page
	group Group
	title func(Strings) string
}

// allPages is every page, in the order the navigation lists them. It is the
// one place the order lives: Pages and AllPages both read it.
// Note: PageSessions and GroupModels are temporarily hidden, with Providers,
// Routing, and Gateway moved under GroupWork.
var allPages = []pageSpec{
	{PageAgents, GroupWork, func(s Strings) string { return s.Agents }},
	{PageLibrary, GroupWork, func(s Strings) string { return s.Library }},
	{PageProviders, GroupWork, func(s Strings) string { return s.Providers }},
	{PageRouting, GroupWork, func(s Strings) string { return s.Routing }},
	{PageGateway, GroupWork, func(s Strings) string { return s.Gateway }},
	{PageUsage, GroupInsight, func(s Strings) string { return s.Usage }},
	{PagePlugins, GroupExtend, func(s Strings) string { return s.Plugins }},
}

// groupTitle is a group's heading in the app's language.
func (a *App) groupTitle(g Group) string {
	t := a.text.S
	switch g {
	case GroupWork:
		return t.GroupWork
	case GroupModels:
		return t.GroupModels
	case GroupInsight:
		return t.GroupInsight
	case GroupExtend:
		return t.GroupExtend
	}
	return string(g)
}

// pageGroup returns the Group a Page belongs to.
func (a *App) pageGroup(p Page) Group {
	for _, spec := range allPages {
		if spec.page == p {
			return spec.group
		}
	}
	return GroupWork
}

// pageTitle is a page's title in the app's language.
func (a *App) pageTitle(p Page) string {
	if p == PageSettings {
		return a.text.S.Settings
	}
	for _, info := range a.Pages() {
		if info.Page == p {
			return info.Title
		}
	}
	return string(p)
}

// Show brings the window up, on the named page when p is not empty.
//
// It is the shell's half of magpie's gui.Windows.ShowMain, which the tray menu
// and the deep-link handler use.
func (a *App) Show(p Page) {
	if p != "" {
		a.page = p
	}
	// The panel and the window hold the keyboard one at a time; a window
	// brought up takes it, so the panel goes.
	a.closePanel()
	if a.win == nil {
		return
	}
	a.win.Show()
	a.win.Focus()
	a.win.Invalidate()
}

// Hide takes the window down without ending the app, as magpie's tray does.
func (a *App) Hide() {
	if a.win != nil {
		a.win.Hide()
	}
}

// Quit ends the app.
func (a *App) Quit() { mygo.App.Quit() }

// loadSettings reads magpie's saved settings and applies what the shell draws
// from: the language and the text size.
//
// The read is magpie's own (settings.Load), so Echo shows the reader's saved
// choice rather than a default of its own. A language the reader set is never
// overridden by the desktop's locale; see resolveLang.
func (a *App) loadSettings() {
	a.loadSettingsFor(mygo.App.Locale())
}

// loadSettingsFor is loadSettings with the desktop's locale passed in, so a
// test can build an app without a running application: App.Locale needs one.
func (a *App) loadSettingsFor(locale string) {
	s := settings.Load()
	a.lang = resolveLang(s.Lang, locale)
	a.text = Text(a.lang)
}

// newApp builds the app's state without a window, which is what a test drives
// through the view directly. Run builds its app the same way, then gives it a
// window; the two must not drift.
func newApp(page Page, locale string) *App {
	if page == "" {
		page = PageAgents
	}
	// Echo's gateway address is set here, before anything reads it, because
	// MAGPIE_ADDR is what the whole domain reads the gateway's address
	// through: the listener, the URL every page and the tray draw, and the
	// configs of the agents Echo points at it. Left unset it is magpie's
	// 3425, which is the port magpie's own app serves — so Echo launched
	// beside magpie would be handed that one's gateway and show a window
	// serving magpie's providers and routes.
	//
	// It is set on every construction rather than only in Run so that the
	// probe, which builds the app the same way without a window, renders the
	// address the shipped app serves.
	os.Setenv("MAGPIE_ADDR", echoAddr())
	a := &App{
		page:            page,
		sidebarOpen:     map[Group]bool{GroupWork: true, GroupModels: true, GroupInsight: true, GroupExtend: true},
		gwCallsShown:    20,
		agFolded:        true,
		agInstallFolded: true,
		skPicked:        map[string]bool{},
		skProbePicks:    map[string]bool{},
		skProbeAgents:   map[string]bool{},
		skOpenProjects:  map[string]bool{},
		usTab:           UsTabOverview,
		usPeriod:        usage.Month,
		usPeriodIdx:     2, // Today, 7 days, 30 days, All
		usSessRange:     "30d",
		usSessRangeIdx:  2, // Today, 7 days, 30 days, 90 days, All
		usSessListed:    200,
		usCtxDays:       7,
		usCtxDaysIdx:    1, // Today, 7 days, 30 days
		usFolds:         map[string]bool{},
		usHiddenAgents:  map[string]bool{},
		setTab:          SetTabGeneral,
		setWarmTab:      "codex",
		setSyncKeys:     true,
		setSyncAgents:   true,
		setSyncLib:      true,
		setExportKeys:   true,
		setExportLib:    true,
		setImportAgents: true,
		setImportLib:    true,
		setRevealedKey:  map[string]string{},
	}
	if cached, ok := loadDataCache(); ok {
		a.data = cached
	}
	a.loadSettingsFor(locale)
	// A vendor's allowance read lands in the background (SubscriptionUsage
	// answers at once with what it holds and reads again behind it), and
	// magpie's own GUI is told the moment one lands, reading again at once
	// rather than standing on the stale copy it showed. The shell hears it
	// here: the tray panel and the Usage page's allowance cards read again
	// then, where before they kept what they had until opened again.
	provider.OnSubscriptionUsage(func() {
		InvalidateQuotas()
		go func() {
			qs := ReadQuotas(false)
			a.updatePanel(func() {
				a.panelQuotas = qs
				a.panelReadAt = time.Now()
			})
			a.refreshUsageQuotas(qs)
		}()
	})
	return a
}

// Reload reads the pages' data from magpie's domain packages, off the main
// thread, and hands it to the view when it arrives.
//
// The read runs in a goroutine because it touches the filesystem and, for the
// catalog, the network; the view must not wait on it.
//
// It reads twice, as magpie's own page does. The glance lands first — it is
// everything but walking each agent's skill folders through, which on a
// machine with many big skills takes the longest — so the page draws at once.
// The full read follows it, filling in what only it has: the skills the
// agents have of their own ("In your agents") and the copies out of date.
func (a *App) Reload() {
	a.loading = true
	if a.win != nil {
		a.win.Invalidate()
	}
	problems := a.libProblems
	go func() {
		glance := readData(problems, true)
		if a.win == nil {
			return
		}
		a.win.Update(func() {
			a.data, a.loading, a.err = glance, false, nil
		})
		a.loadCLIsAsync()

		full := readData(problems, false)
		a.win.Update(func() {
			a.data, a.err = full, nil
		})
		// the panel draws the same data: it is up, and its own refresh
		// button asked for this reading
		a.updatePanel(func() {})
	}()
}

// LoadNow reads the pages' data on the calling thread, which is what a test
// does: with no window there is no frame to invalidate and nothing to wait
// for, and a test needs the data before it renders.
//
// It reads the whole page at once — the full read, not the glance followed by
// it — so the pages a test renders are the pages the app settles on.
func (a *App) LoadNow() {
	a.data, a.loading, a.err = readData(a.libProblems, false), false, nil
}
