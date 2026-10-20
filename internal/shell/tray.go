//go:build nogui && echo

package shell

import (
	"embed"
	"io/fs"
	"runtime"
	"sort"
	"time"

	"github.com/egoist/mygo"
)

// trayPNG is the menu bar icon: the app icon itself, small. It is not a
// template, so it keeps its blue in either menu bar.
//
//go:embed tray.png
var trayPNG []byte

// trayTemplatePNG is the menu bar icon as macOS paints it: the mark in
// black with an alpha channel, which the system tints for a light or a dark
// menu bar — magpie's own tray icon is a template too.
//
//go:embed tray-template.png
var trayTemplatePNG []byte

// trayFlapFS is the echo pulse the menu bar icon plays when the panel
// opens: the mark's back window steps out and back, as magpie's bird beats
// its wing. The frames are the tray's own template pictures.
//
//go:embed trayflap/*.png
var trayFlapFS embed.FS

// trayFlapEvery is how long a frame of the pulse is on the screen.
const trayFlapEvery = 30 * time.Millisecond

// makeTray adds Echo's menu bar icon: a click opens the quick panel (the
// tray popover, tray_panel.go), and a right click offers the pages and Quit.
//
// It must be called on the main thread, after the app is ready, which is where
// Run calls it from.
//
// The menu is installed for a popup only, and taken back after: a menu left
// on the icon consumes the primary click on macOS, which the panel needs.
// On Linux the AppIndicator icons tell nothing of clicks, so there the menu
// stays on the icon and one of its items opens the panel.
func (a *App) makeTray() error {
	// macOS tints a template icon for the menu bar it stands in, so the
	// mark is black and the system paints it; the other trays keep the
	// app's own colours, which they do not tint.
	icon := trayPNG
	template := false
	if runtime.GOOS == "darwin" {
		icon = trayTemplatePNG
		template = true
	}
	tray, err := mygo.NewTray(mygo.TrayOptions{
		Icon:           icon,
		IconIsTemplate: template,
		ToolTip:        a.text.S.AppName,
	})
	if err != nil {
		return err
	}
	a.tray = tray

	if runtime.GOOS == "linux" {
		tray.SetMenu(a.trayMenu())
		return nil
	}
	tray.OnClick(func() {
		a.playTrayFlap()
		a.togglePanel()
	})
	tray.OnRightClick(func() {
		a.closePanel()
		a.tray.SetMenu(a.trayMenu())
		defer a.tray.SetMenu(nil)
		a.tray.PopUpMenu()
	})
	return nil
}

// playTrayFlap plays the echo pulse on the menu bar icon, as magpie's bird
// beats its wing when its panel opens. The frames are the tray's own
// template pictures, so the system keeps tinting them as it tints the mark
// itself; the icon is put back as it was when the pulse is over.
func (a *App) playTrayFlap() {
	if runtime.GOOS != "darwin" || a.tray == nil {
		return
	}
	frames := trayFlapFrames()
	if len(frames) == 0 {
		return
	}
	go func() {
		for _, f := range frames {
			_ = a.tray.SetIcon(f, true)
			time.Sleep(trayFlapEvery)
		}
		_ = a.tray.SetIcon(trayTemplatePNG, true)
	}()
}

// trayFlapFrames are the pulse's pictures, in the order they are played.
func trayFlapFrames() [][]byte {
	entries, err := fs.ReadDir(trayFlapFS, "trayflap")
	if err != nil {
		return nil
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	out := make([][]byte, 0, len(names))
	for _, n := range names {
		data, err := fs.ReadFile(trayFlapFS, "trayflap/"+n)
		if err != nil {
			continue
		}
		out = append(out, data)
	}
	return out
}

// trayMenu is the icon's menu: the quick panel, the pages, then what ends
// the app. The first item is what a Linux icon's reader opens the panel
// with, its icon telling nothing of clicks.
func (a *App) trayMenu() *mygo.Menu {
	items := []*mygo.MenuItem{
		{Label: "快捷面板", Click: func(_ *mygo.MenuItem, _ *mygo.Window) { a.togglePanel() }},
		{Label: a.text.S.Show, Click: func(_ *mygo.MenuItem, _ *mygo.Window) { a.Show("") }},
		{Type: mygo.MenuItemSeparator},
	}
	for _, info := range a.Pages() {
		page := info.Page
		items = append(items, &mygo.MenuItem{
			Label: info.Title,
			Click: func(_ *mygo.MenuItem, _ *mygo.Window) { a.Show(page) },
		})
	}
	items = append(items,
		&mygo.MenuItem{Type: mygo.MenuItemSeparator},
		&mygo.MenuItem{Label: a.text.S.Settings, Click: func(_ *mygo.MenuItem, _ *mygo.Window) { a.Show(PageSettings) }},
		&mygo.MenuItem{Type: mygo.MenuItemSeparator},
		&mygo.MenuItem{Label: a.text.S.Quit, Click: func(_ *mygo.MenuItem, _ *mygo.Window) { a.Quit() }},
	)
	return mygo.NewMenu(items)
}
