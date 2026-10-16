//go:build nogui && echo

package gui

import "github.com/yetone/magpie/internal/gateway"

// Echo's shell draws its own window, but everything the window is about —
// the gateway, the model lists, the agents' wiring — is magpie's, and starts
// the same way magpie's own app starts it (backend.go).
//
// These two are that startup, exported for the shell to call. They live here
// rather than being written again in the shell because the logic is not
// simple: the gateway is taken over from an older magpie, watched while
// another holds it, and handed over on the way out, and tests here cover
// each of those. Reimplementing it beside them would leave two versions of
// the product's most important behaviour.
//
// The file is tagged nogui && echo, so magpie's own builds — the Wails app
// and the terminal CLI — are untouched by it.

// StartBackend starts what serves the page and the agents: the gateway,
// unless another magpie has it, and the model lists kept warm.
func StartBackend() *gateway.Server { return startBackend() }

// StopBackend stops this process's gateway taking requests, and returns once
// those in flight have finished.
func StopBackend() { stopServing() }
