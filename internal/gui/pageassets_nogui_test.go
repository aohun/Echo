//go:build nogui

package gui

import (
	"os"
	"path/filepath"
)

// pageAssets points the tests at the page, which a build without the
// desktop app does not embed: it is read from the folder beside the
// binary (assets_nogui.go), and the tests read it from the folder it is
// in — the same one a packaged app serves it from.
func pageAssets() {
	if os.Getenv(webAssetsEnv) != "" {
		return
	}
	if dir, err := filepath.Abs("assets"); err == nil {
		os.Setenv(webAssetsEnv, dir)
	}
}
