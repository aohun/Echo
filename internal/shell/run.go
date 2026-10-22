//go:build nogui && echo

package shell

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/plugins/updater/native"
	"github.com/egoist/mygo/ui"
	"github.com/yetone/magpie/internal/gui"
	"github.com/yetone/magpie/internal/settings"
)

// Options configures the shell.
type Options struct {
	// Page is the page the window opens on; "" opens the first one.
	Page Page
	// Width and Height of the window in DIPs.
	Width, Height int
	// Hidden starts the app without showing the window, which is what a
	// menu bar app does when it is opened at login: the tray icon is there,
	// the window is not. magpie's `tray` launch asks for this.
	Hidden bool
	// NoTray leaves the menu bar icon out. A test or a probe sets it: a run
	// must not leave an icon behind in the reader's menu bar.
	NoTray bool
	// NoBackend leaves the gateway out, for a run that must not serve.
	NoBackend bool
	// Ready, when set, runs on the main thread once the window exists,
	// before it is shown.
	Ready func(a *App, w *mygo.Window)
}

// shellUpdatesWired says the updater plugin is mounted, which only Run
// does: a page may draw its words only then, for the plugin's own calls
// panic out of a render that has no app under it (the probe, a test).
var shellUpdatesWired bool

// Run starts Echo: it reads magpie's settings, starts what serves the agents,
// opens the window and the menu bar icon, and blocks until the app quits.
//
// It must be called from main: MyGo runs its event loop on the main goroutine.
func Run(opts Options) error {
	if opts.Width == 0 {
		opts.Width, opts.Height = 1180, 780
	}
	if opts.Page == "" {
		opts.Page = PageAgents
	}

	mygo.App.SetName("Echo")

	// The update window in native UI — the shell hosts no plugin HTML —
	// checking Echo's own releases (mygo.json's updates): what Settings →
	// About's 检查更新 opens, and what checks in the background once a day
	// in a build that can update itself (one `mygo build` signed).
	mygo.Use(native.Plugin)
	shellUpdatesWired = true

	a := newApp(opts.Page, mygo.App.Locale())
	setCurrent(a)

	// the appearance the reader saved, before the window is built, so the
	// app starts in it (mygo.Theme.SetSource)
	applyThemeSource(settings.Load().Theme)

	// The bundled magpie web starts while the window comes up, so the
	// window shows at once and only the first read waits for the
	// instance (the seam decides once, remoteGateway's sync.Once, so it
	// must be answering by then). A run the environment points itself,
	// and one without the helper next to the binary, decide at once.
	companionReady, stopCompanion := startCompanion()

	// The gateway starts before the window, as magpie's own app starts it:
	// it is the product's reason to exist, and an agent may be waiting on
	// the address. It is also what makes the app a menu bar app rather than
	// a window that happens to open.
	//
	// newApp has set MAGPIE_ADDR to Echo's own address (echoAddr), which is
	// what decides where this one listens.
	//
	// But one gateway serves, never two (gateway.WhileServing): when an
	// instance answers the seam, its gateway is the one the agents' routes
	// point at, and a second one here would keep rewriting the configs
	// under it — the drift a page then reports flaps between them. So the
	// shell's own gateway starts only when no instance answered.
	if !opts.NoBackend {
		go func() {
			<-companionReady
			if remoteGateway() == nil {
				gui.StartBackend()
			}
		}()
	}

	mygo.App.WhenReady(func() {
		a.win = mygo.NewWindow(mygo.WindowOptions{
			Title:         a.text.S.AppName,
			Width:         opts.Width,
			Height:        opts.Height,
			MinWidth:      820,
			MinHeight:     560,
			TitleBarStyle: mygo.TitleBarHiddenInset,
			// The traffic lights are placed explicitly rather than left to
			// AppKit, because their position is what the header is drawn
			// against. Left to AppKit, TitleBar() reports the whole
			// titlebar-plus-toolbar band (66 DIPs here), which the lights
			// are not centered in — so content centered in that band sits
			// below them, which is what the header did. Pinned, TitleBar()
			// reports the band they are centered in (button height plus
			// twice Y), and Y is chosen so that band is the header's own
			// height: 14 + 2*18 = 50. X keeps them where AppKit had them
			// (its own left edge is 19).
			TrafficLightPosition: &mygo.Point{X: 19, Y: 18},
			// StateKey keeps the window's size and place across runs, as
			// magpie keeps its own window's.
			StateKey: "echo",
			Content:  ui.View(a.view),
		})
		// Closing the window hides it rather than ending the app, which is
		// what a menu bar app does: the tray icon brings it back.
		a.win.OnClose(a.onWindowClose)
		a.win.SetMenu(a.menu())

		// the stall watchdog, once the window's frames are the heartbeat
		// it watches
		watchStalls()

		if !opts.NoTray {
			// A tray that cannot be made is not fatal: the window still
			// works, and the failure is reported rather than swallowed.
			if err := a.makeTray(); err != nil {
				log.Printf("echo: menu bar icon: %v", err)
			}
		}

		if opts.Ready != nil {
			opts.Ready(a, a.win)
		}
		if !opts.Hidden {
			a.win.Show()
		}
		// The pages' data is read once the bundled instance has decided
		// (the window is already up, so the wait costs nothing the reader
		// sees but an empty page filling in).
		go func() {
			<-companionReady
			a.win.Update(func() {
				a.Reload()
			})
		}()
	})

	// A quit closes every window before OnWillQuit runs, and the window's
	// close handler hides rather than closes unless quitting is set. So the
	// flag is set here, on the hook that runs first: set any later and the
	// handler cancels the very close the quit is made of, and the app stays
	// up (MyGo's menubar-native example does the same).
	//
	// The gateway is given back on the way out, so the next launch — or the
	// magpie that takes over — finds the port free rather than a socket
	// closing under it.
	var stop func()
	if !opts.NoBackend {
		stop = gui.StopBackend
	}
	wireQuit(mygo.App, a, stop, stopCompanion)

	return mygo.App.Run()
}

// quitHooks is the part of mygo.Application the quit wiring needs: the two
// hooks that bracket a quit, which the ordering below turns on.
type quitHooks interface {
	OnBeforeQuit(func(*mygo.QuitEvent)) func()
	OnWillQuit(func(*mygo.QuitEvent)) func()
}

// wireQuit registers what ends the app, on the hooks MyGo fires in the order
// that makes them work.
//
// beginQuit goes on OnBeforeQuit — the hook that runs before a quit closes
// any window — because the window's close handler (onWindowClose) hides the
// window unless it has run, and a prevented close cancels the whole quit.
// OnWillQuit fires only after every window has closed, so a flag set there
// arrives too late to let the quit's own close through. stopBackend and
// stopCompanion, which must run after the windows are gone, go there
// instead.
func wireQuit(app quitHooks, a *App, stopBackend, stopCompanion func()) {
	app.OnBeforeQuit(func(_ *mygo.QuitEvent) { a.beginQuit() })
	app.OnWillQuit(func(_ *mygo.QuitEvent) {
		if stopBackend != nil {
			stopBackend()
		}
		// the bundled magpie web dies with the app that started it; one
		// reused from an earlier launch is left to its owner
		if stopCompanion != nil {
			stopCompanion()
		}
	})
}

// beginQuit marks the app as ending, which is what lets the window's close
// handler through.
//
// It runs on MyGo's OnBeforeQuit, the hook that fires before a quit closes
// any window: the close handler (onWindowClose) hides the window unless this
// has run, and a prevented close cancels the whole quit.
func (a *App) beginQuit() { a.quitting = true }

// onWindowClose is the window's close handler: it hides the window rather
// than ending the app, which is what a menu bar app does — the tray icon
// brings it back.
//
// It must let a quit's own close through, or the quit is cancelled by the
// very handler that is meant to keep the app up between quits. That is what
// beginQuit marks, and why it is wired to OnBeforeQuit rather than
// OnWillQuit, which MyGo fires only after every window has closed.
func (a *App) onWindowClose(e *mygo.CloseEvent) {
	if a.quitting {
		return
	}
	e.PreventDefault()
	a.Hide()
}

// menu is the application menu: the standard menus, with Echo's own About.
//
// MyGo installs a default menu when none is set; this one names the app and
// offers the pages, so the reader can reach a page from the menu bar as well
// as from the sidebar.
func (a *App) menu() *mygo.Menu {
	var pages []*mygo.MenuItem
	for _, info := range a.Pages() {
		page := info.Page
		pages = append(pages, &mygo.MenuItem{
			Label: info.Title,
			Click: func(_ *mygo.MenuItem, _ *mygo.Window) { a.Show(page) },
		})
	}

	return mygo.NewMenu([]*mygo.MenuItem{
		{
			Label: a.text.S.AppName,
			Submenu: []*mygo.MenuItem{
				{Role: mygo.RoleAbout},
				{Type: mygo.MenuItemSeparator},
				{Label: a.text.S.Settings, Accelerator: "CmdOrCtrl+,", Click: func(_ *mygo.MenuItem, _ *mygo.Window) { a.Show(PageSettings) }},
				{Type: mygo.MenuItemSeparator},
				{Role: mygo.RoleServices},
				{Type: mygo.MenuItemSeparator},
				{Role: mygo.RoleHide},
				{Role: mygo.RoleHideOthers},
				{Role: mygo.RoleUnhide},
				{Type: mygo.MenuItemSeparator},
				{Role: mygo.RoleQuit, Label: a.text.S.Quit},
			},
		},
		{
			Label: a.text.S.AppName,
			Submenu: []*mygo.MenuItem{
				{Label: a.text.S.Show, Accelerator: "CmdOrCtrl+Shift+O", Click: func(_ *mygo.MenuItem, _ *mygo.Window) { a.Show("") }},
				{Label: a.text.S.Hide, Accelerator: "CmdOrCtrl+W", Click: func(_ *mygo.MenuItem, _ *mygo.Window) { a.Hide() }},
			},
		},
		{Label: a.text.S.Settings, Submenu: append(append([]*mygo.MenuItem{}, pages...), &mygo.MenuItem{
			Type: mygo.MenuItemSeparator,
		}, &mygo.MenuItem{
			// the About page of the Settings, as the menu names it
			Label: a.text.S.Set.About,
			Click: func(_ *mygo.MenuItem, _ *mygo.Window) {
				a.setTab = SetTabAbout
				a.Show(PageSettings)
			},
		})},
		{Role: mygo.RoleEditMenu},
		{Role: mygo.RoleWindowMenu},
	})
}
