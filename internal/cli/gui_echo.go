//go:build nogui && echo

package cli

import (
	"github.com/yetone/magpie/internal/proc"
	"github.com/yetone/magpie/internal/shell"
)

// hasGUI is true: this build has a desktop app in it — the MyGo shell — so a
// launch with no arguments opens the window rather than the TUI, and the
// updater knows this binary carries the app.
const hasGUI = true

// runGUI opens Echo's window.
//
// It is the same seam magpie's own desktop app plugs into (gui_on.go): the
// command line, the deep links and the platform launch arguments all arrive
// here, and only what happens at the end differs. Echo replaces the shell —
// internal/gui's window and Wails — and keeps everything below it, so the
// command line is magpie's unchanged.
//
// It must run on the main goroutine, which is where Main calls it: MyGo runs
// its event loop there.
func runGUI(showMain bool, link string) error {
	// the desktop app may have been opened from the Finder, with none of the
	// PATH a terminal has
	proc.UserPath()
	return shell.Run(shell.Options{
		Page:   pageForView(link),
		Hidden: !showMain,
	})
}

// runWindow starts the app with its window open on view ("" for the first
// page), which is what `magpie gui settings` and a click on a Windows
// notification ask for.
func runWindow(view string) error {
	return shell.Run(shell.Options{Page: pageForView(view)})
}

// runPanel starts the app with its window up: Echo has one window, so its
// quick panel is that window rather than a second one.
func runPanel() error { return runGUI(true, "") }

// pageForView turns a launch view name into the page the window opens on.
//
// The names come from magpie's own views ("settings", "usage"), which the
// shell's pages are named after; a name the shell has no page for opens the
// first page, which is what magpie does with an unknown one.
func pageForView(view string) shell.Page {
	for _, p := range shell.AllPages() {
		if string(p) == view {
			return p
		}
	}
	return ""
}
