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
	"sync"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/yetone/magpie/internal/settings"
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

	// quitting is set while the app is ending, so the window's close
	// handler lets the close through instead of hiding the window.
	quitting bool

	// tables holds each page's table state: the columns' order and widths,
	// the sort, and which row is chosen. MyGo keeps such state across
	// frames by the pointer it is given, so one per page must outlive the
	// frame that built it.
	tables map[Page]*ui.ListState
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
var allPages = []pageSpec{
	{PageAgents, GroupWork, func(s Strings) string { return s.Agents }},
	{PageSessions, GroupWork, func(s Strings) string { return s.Sessions }},
	{PageLibrary, GroupWork, func(s Strings) string { return s.Library }},
	{PageProviders, GroupModels, func(s Strings) string { return s.Providers }},
	{PageRouting, GroupModels, func(s Strings) string { return s.Routing }},
	{PageGateway, GroupModels, func(s Strings) string { return s.Gateway }},
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

// pageTitle is a page's title in the app's language.
func (a *App) pageTitle(p Page) string {
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
	a := &App{
		page:        page,
		sidebarOpen: map[Group]bool{GroupWork: true, GroupModels: true, GroupInsight: true, GroupExtend: true},
	}
	a.loadSettingsFor(locale)
	return a
}

// Reload reads the pages' data from magpie's domain packages, off the main
// thread, and hands it to the view when it arrives.
//
// The read runs in a goroutine because it touches the filesystem and, for the
// catalog, the network; the view must not wait on it.
func (a *App) Reload() {
	a.loading = true
	if a.win != nil {
		a.win.Invalidate()
	}
	go func() {
		data := readData()
		if a.win == nil {
			return
		}
		a.win.Update(func() {
			a.data, a.loading, a.err = data, false, nil
		})
	}()
}

// LoadNow reads the pages' data on the calling thread, which is what a test
// does: with no window there is no frame to invalidate and nothing to wait
// for, and a test needs the data before it renders.
//
// It is the same read Reload runs, so the pages a test renders are the pages
// the app shows.
func (a *App) LoadNow() {
	a.data, a.loading, a.err = readData(), false, nil
}
