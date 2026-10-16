//go:build nogui && echo

package shell

import (
	_ "embed"

	"github.com/egoist/mygo"
)

// trayPNG is the menu bar icon: magpie's own, so Echo wears the mark of the
// product it reimplements rather than a placeholder.
//
//go:embed tray.png
var trayPNG []byte

// makeTray adds Echo's menu bar icon: a click shows the window, and the menu
// offers the pages and Quit.
//
// It must be called on the main thread, after the app is ready, which is where
// Run calls it from.
func (a *App) makeTray() error {
	tray, err := mygo.NewTray(mygo.TrayOptions{
		Icon: trayPNG,
		// A template icon lets macOS tint it for the light and dark menu
		// bar, which is how a menu bar app's mark stays legible.
		IconIsTemplate: true,
		ToolTip:        a.text.S.AppName,
		Menu:           a.trayMenu(),
	})
	if err != nil {
		return err
	}
	a.tray = tray
	// A left click opens the window, as magpie's icon does; the menu is
	// what a right click (or the icon's own menu) offers.
	tray.OnClick(func() { a.Show("") })
	tray.OnRightClick(func() { a.Show("") })
	return nil
}

// trayMenu is the icon's menu: the pages, then what ends the app.
func (a *App) trayMenu() *mygo.Menu {
	items := []*mygo.MenuItem{
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
		&mygo.MenuItem{Label: a.text.S.Quit, Click: func(_ *mygo.MenuItem, _ *mygo.Window) { a.Quit() }},
	)
	return mygo.NewMenu(items)
}
