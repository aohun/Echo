//go:build nogui

package shell

import (
	"github.com/egoist/mygo"
	"github.com/yetone/magpie/internal/gui"
)

// Options configures the shell.
type Options struct {
	// URL is the page the window opens: "/" for the first tab.
	URL string
	// Width and Height of the window in DIPs.
	Width, Height int
	// ExtraPreload is appended to the page's preload script, after the drag
	// shim. The acceptance probe uses it to record page errors.
	ExtraPreload string
	// Ready, when set, runs on the main thread once the window exists, before
	// the window is shown. It is how the acceptance probe reaches the page.
	Ready func(w *mygo.Window)
}

// Run starts the app: it serves magpie's page and API through MyGo's scheme and
// opens the window. It blocks until the app quits and must be called from main.
func Run(opts Options) error {
	if opts.URL == "" {
		opts.URL = "/"
	}
	if opts.Width == 0 {
		opts.Width, opts.Height = 1180, 780
	}

	mygo.App.SetName("Echo")

	// magpie's page, its /api/* routes and its /boot.js are served from one
	// handler. The page reaches Go over fetch("/api/…") on its own origin, so a
	// single scheme handler carries both the frontend and the API.
	if err := mygo.Protocol.Handle("mygo", Handler(gui.Handler(Host{}, nil))); err != nil {
		return err
	}

	mygo.App.WhenReady(func() {
		w := mygo.NewWindow(mygo.WindowOptions{
			Title:     "Echo",
			URL:       opts.URL,
			Width:     opts.Width,
			Height:    opts.Height,
			MinWidth:  900,
			MinHeight: 560,
			Page:      mygo.PageOptions{PreloadScript: dragShim + opts.ExtraPreload},
		})
		SetMain(w)
		if opts.Ready != nil {
			opts.Ready(w)
		}
	})

	return mygo.App.Run()
}
