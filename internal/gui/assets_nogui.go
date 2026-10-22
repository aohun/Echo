//go:build nogui

package gui

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"
)

// assets is the pictures the native shell draws (internal/gui/assets/icons):
// the vendors', the agents' and the apps'.
//
// The page itself is not embedded here: a build without the desktop app
// draws its own windows, and the page is six megabytes of HTML, CSS and
// JavaScript that it would only carry. `magpie web` serves the page from
// the folder beside the binary (webAssets), where a build that ships it
// puts it.
//
//go:embed assets/icons
var assets embed.FS

// webAssets is the page's folder beside the binary, nil when there is
// none. `magpie web` serves the page from it, and says so when it is
// missing rather than serving a window with nothing in it.
func webAssets() fs.FS {
	if d := os.Getenv(webAssetsEnv); d != "" {
		if st, err := os.Stat(filepath.Join(d, "index.html")); err == nil && !st.IsDir() {
			return os.DirFS(d)
		}
	}
	exe, err := os.Executable()
	if err != nil {
		return nil
	}
	dir := filepath.Dir(exe)
	// beside the binary, and in the bundle's resources folder, which is
	// where a packaged app's resources are installed.
	for _, d := range []string{
		filepath.Join(dir, webAssetsDir),
		filepath.Join(dir, "..", "Resources", webAssetsDir),
	} {
		if st, err := os.Stat(filepath.Join(d, "index.html")); err == nil && !st.IsDir() {
			return os.DirFS(d)
		}
	}
	return nil
}

// webAssetsDir is the folder beside the binary that holds the page, in a
// build that does not embed it.
const webAssetsDir = "web-assets"

// webAssetsEnv names the folder the page is read from, over the one
// beside the binary. A test reads the page from the folder it is in.
const webAssetsEnv = "MAGPIE_WEB_ASSETS"

// staticFS is the page and its pictures: the folder beside the binary
// where they are, and the icons embedded here when there is no folder —
// a build that serves the page always has the folder.
func staticFS() fs.FS {
	if f := webAssets(); f != nil {
		return f
	}
	f, _ := fs.Sub(assets, "assets")
	return f
}
