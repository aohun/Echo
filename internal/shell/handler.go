//go:build nogui

// Package shell is Echo's desktop shell: magpie's page and JSON API served
// through MyGo, with MyGo owning the window and the event loop.
//
// It exists because Go forbids importing a package main: the shipped app
// (cmd/echo) and the acceptance probe (cmd/echoprobe) share this code.
//
// Everything here is built with -tags nogui. magpie's internal/gui package only
// compiles without Wails under that tag, and under it magpie's own GUI entry
// point is absent — which is exactly what this package replaces.
//
// The domain packages (internal/gateway, internal/provider, …) are untouched;
// see docs/04-plan-option-a-shell-migration.md for the scope boundary.
package shell

import (
	"net/http"
)

// Handler wraps magpie's page-and-API handler with the bridges the page needs
// under MyGo.
//
// The page reaches Go over fetch("/api/…") on its own origin rather than through
// a Wails binding, so magpie's handler already carries the whole API and nothing
// is stubbed. What it does not serve is the Wails runtime module the page imports
// for its window controls.
//
// The A0 probe found three gaps to close in the handler. This is one of them.
// The other two are already served by magpie's handler: /boot.js, which magpie
// generates from the saved settings, and the content-hash naming of the page's
// own files.
func Handler(inner http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /wails/runtime.js", serveWailsRuntime)
	mux.Handle("/", inner)
	return mux
}

// wailsRuntime forwards the window calls the page makes onto MyGo's own window
// API. The page reaches for this module only to close the window and to toggle
// maximise; everything else it does goes through /api/*.
const wailsRuntime = `export const Window = {
  Close() {
    return window.mygo?.window?.close?.();
  },
  ToggleMaximise() {
    return window.mygo?.window?.toggleMaximize?.();
  },
  Minimise() {
    return window.mygo?.window?.minimize?.();
  },
  Maximise() {
    return window.mygo?.window?.maximize?.();
  },
  UnMaximise() {
    return window.mygo?.window?.unmaximize?.();
  },
  IsMaximised() {
    return window.mygo?.window?.isMaximized?.() ?? false;
  },
  ToggleFullscreen() {
    return window.mygo?.window?.toggleFullScreen?.();
  },
};

export default { Window };
`

func serveWailsRuntime(rw http.ResponseWriter, _ *http.Request) {
	rw.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	rw.Header().Set("Cache-Control", "no-store")
	_, _ = rw.Write([]byte(wailsRuntime))
}
