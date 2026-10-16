//go:build nogui

package shell

import (
	"net/url"
	"sync"

	"github.com/egoist/mygo"
	"github.com/yetone/magpie/internal/gui"
)

// Host implements magpie's gui.Windows interface on top of MyGo.
//
// magpie's handler drives the desktop through this interface rather than through
// Wails directly, which is what lets the shell be replaced without touching the
// page or the API. Its methods run on the scheme handler's goroutine, never on
// the main thread, so MyGo's window calls are safe here.
type Host struct{}

// winMu guards Main: it is written on the main thread once the app is ready and
// read by the handler's goroutines.
var (
	winMu sync.Mutex
	Main  *mygo.Window
)

// Window returns the shell's main window, or nil before it exists.
func Window() *mygo.Window {
	winMu.Lock()
	defer winMu.Unlock()
	return Main
}

// SetMain records the main window, called on the main thread.
func SetMain(w *mygo.Window) {
	winMu.Lock()
	Main = w
	winMu.Unlock()
}

// HidePanel has no panel to hide: Echo's shell is the main window alone.
// magpie's tray panel is not part of this probe.
func (Host) HidePanel() {}

// ShowMain brings the window up, on the named tab when view is set.
//
// The page reads its tab from the query string on load (app.js's own
// client-side routing), so a named view reloads the page there. magpie reaches
// the running page instead, through Wails' ExecJS; that is a fidelity gap this
// shell still has.
func (Host) ShowMain(view string) {
	w := Window()
	if w == nil {
		return
	}
	w.Show()
	w.Focus()
	if view != "" {
		_ = w.Page().LoadURL("/?view=" + url.QueryEscape(view))
	}
}

// Quit ends the app.
func (Host) Quit() { mygo.App.Quit() }

// OpenURL hands a link to the system browser.
func (Host) OpenURL(rawURL string) { _ = mygo.Shell.OpenExternal(rawURL) }

// OpenFolder shows a folder in the system file manager.
func (Host) OpenFolder(path string) error { return mygo.Shell.OpenPath(path) }

// ChooseFolder asks for a folder, "" when the user cancels.
func (Host) ChooseFolder(title string) (string, error) {
	chosen, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
		Parent:    Window(),
		Title:     title,
		Directory: true,
	})
	if err != nil || len(chosen) == 0 {
		return "", err
	}
	return chosen[0], nil
}

// Copy puts text on the system clipboard.
func (Host) Copy(text string) bool {
	mygo.Clipboard.WriteText(text)
	return true
}

// FitPanel does nothing, and the tinting calls report that MyGo cannot do it, so
// the page keeps painting the panel itself. Echo has no panel in this probe, and
// MyGo has no established equivalent for magpie's panel fit, tint or tray
// variants (risk O2 in docs/06-open-questions-and-risks.md).
func (Host) FitPanel(height int, g gui.Glide)           {}
func (Host) TintPanel(rgba [4]uint8, ms int) bool       { return false }
func (Host) TintTitleBar(rgba [4]uint8, dark bool) bool { return false }

// SetTextSize zooms the page to percent, which is how magpie's own text size
// setting is applied.
func (Host) SetTextSize(percent int) {
	if w := Window(); w != nil {
		w.Page().SetZoomFactor(float64(percent) / 100)
	}
}
