//go:build nogui && echo

package cli

import (
	"testing"

	"github.com/yetone/magpie/internal/gateway"
	"github.com/yetone/magpie/internal/settings"
)

// The command line reaches Echo's own gateway, not magpie's.
//
// The window used to be the only place that knew Echo's port, and the
// executable used as a command line never went through it: so a terminal
// command wrote magpie's address into the agents' configs, and the
// agents' requests went to magpie — magpie's providers, magpie's routes,
// not Echo's. The commands that ask the gateway (healthcheck, statusline,
// daemon, serve, web) had the same problem, and found magpie's gateway
// and called it their own.
//
// The port is the domain's default now (settings.DefaultPort), so every
// entry point reads the same one without any of them having to know it.
func TestCommandLineReachesEchosGateway(t *testing.T) {
	t.Setenv("MAGPIE_ADDR", "")
	// a home of the test's own, so the settings read are the defaults and
	// not whatever this computer has saved
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", home)

	if got, want := settings.DefaultPort, 3426; got != want {
		t.Fatalf("DefaultPort = %d, want %d: the command line would reach magpie's port", got, want)
	}
	if got, want := gateway.Addr(), "127.0.0.1:3426"; got != want {
		t.Fatalf("the gateway would listen on %q, not %q", got, want)
	}
	if got, want := gateway.URL(), "http://127.0.0.1:3426"; got != want {
		t.Fatalf("gateway.URL() = %q, want %q: an agent's config would name the wrong gateway", got, want)
	}
}

// An address of the reader's own is never replaced.
//
// MAGPIE_ADDR comes first everywhere else in the domain (a container's
// 0.0.0.0:3425, a second copy on a port of its own), and Echo must not be
// the one place it is ignored.
func TestCommandLineKeepsAnAddressOfItsOwn(t *testing.T) {
	t.Setenv("MAGPIE_ADDR", "0.0.0.0:3591")

	if got, want := gateway.Addr(), "0.0.0.0:3591"; got != want {
		t.Fatalf("the gateway would listen on %q, not %q: a launch naming its address lost it", got, want)
	}
}
