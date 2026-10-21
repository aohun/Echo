//go:build nogui && echo

package shell

import (
	"os"
	"testing"

	"github.com/yetone/magpie/internal/gateway"
)

// Echo's gateway is on Echo's own port, not magpie's.
//
// This is the whole of the "not just the UI" claim: the domain reads the
// gateway's address through MAGPIE_ADDR, so an Echo that did not set it
// would serve on magpie's 3425. Two products on one port hand over to
// whichever started first, so an Echo launched beside magpie would show a
// window whose gateway is magpie's — magpie's providers and routes, and
// every request an agent sends going through it.
func TestEchoGatewayAddressIsEchosOwnPort(t *testing.T) {
	t.Setenv("MAGPIE_ADDR", "")
	// a home of the test's own, so the settings read are the defaults and
	// not whatever this computer has saved
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", home)

	if got, want := echoAddr(), "127.0.0.1:3426"; got != want {
		t.Fatalf("echoAddr() = %q, want %q: Echo would not serve on its own port", got, want)
	}
	if echoAddr() == gateway.SavedURL() {
		t.Fatalf("echoAddr() = %q is magpie's own address: Echo would share magpie's port", echoAddr())
	}
	// The address the domain reports must be the one the window draws, or
	// the pages, the tray and the agents' configs would name a different
	// gateway from the one listening.
	if got, want := gateway.Addr(), echoAddr(); got != want {
		t.Fatalf("the gateway would listen on %q, not %q", got, want)
	}
	if got, want := gateway.URL(), "http://127.0.0.1:3426"; got != want {
		t.Fatalf("gateway.URL() = %q, want %q: the pages would tell agents the wrong address", got, want)
	}
}

// An address of the reader's own is never replaced.
//
// MAGPIE_ADDR comes first everywhere else in the domain (a container's
// 0.0.0.0:3425, a second copy on a port of its own), and Echo must not be
// the one place it is ignored.
func TestEchoKeepsAnAddressOfItsOwn(t *testing.T) {
	t.Setenv("MAGPIE_ADDR", "0.0.0.0:3591")
	if got, want := echoAddr(), "0.0.0.0:3591"; got != want {
		t.Fatalf("echoAddr() = %q, want %q: a launch naming its address lost it", got, want)
	}
}

// Echo does not put its own address into MAGPIE_ADDR.
//
// That variable comes before the settings, so a window that set it would
// make the port the reader chooses in the Settings page ineffective: the
// field would be disabled with "unset MAGPIE_ADDR to set the port here",
// for a variable the reader cannot unset — Echo set it. The port is the
// domain's default instead, so the field works and every entry point
// reads the same address.
func TestEchoDoesNotPinTheAddress(t *testing.T) {
	t.Setenv("MAGPIE_ADDR", "")
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", home)

	NewForTest()

	if v := os.Getenv("MAGPIE_ADDR"); v != "" {
		t.Fatalf("MAGPIE_ADDR = %q: the Settings page would disable the port field for a variable Echo set itself", v)
	}
}
