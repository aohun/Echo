//go:build nogui

// Command echo is Echo's desktop app: magpie's page and gateway behind a MyGo
// shell.
//
// It is built with -tags nogui because magpie's internal/gui package only
// compiles without Wails under that tag. Under nogui magpie's own GUI entry point
// is absent, which is exactly what this command replaces; the domain packages and
// the TUI are untouched.
package main

import (
	"log"

	"github.com/yetone/magpie/internal/shell"
)

func main() {
	if err := shell.Run(shell.Options{}); err != nil {
		log.Fatal(err)
	}
}
