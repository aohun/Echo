//go:build !nogui

package gui

import (
	"embed"
	"io/fs"
)

// assets is the app's own page and the pictures it serves, read from the
// binary: the desktop app shows the page in its window, and `magpie web`
// serves it to a browser.
//
//go:embed assets
var assets embed.FS

// staticFS is the page and its pictures, from the binary.
func staticFS() fs.FS { f, _ := fs.Sub(assets, "assets"); return f }
