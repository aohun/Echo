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

	if got, want := echoAddr(), "127.0.0.1:3426"; got != want {
		t.Fatalf("echoAddr() = %q, want %q: Echo would not serve on its own port", got, want)
	}
	if echoAddr() == gateway.SavedURL() {
		t.Fatalf("echoAddr() = %q is magpie's own address: Echo would share magpie's port", echoAddr())
	}
	// The address the domain reports must be the one Echo set, or the pages,
	// the tray and the agents' configs would name a different gateway from
	// the one listening. os.Setenv is what Run does before it starts one.
	os.Setenv("MAGPIE_ADDR", echoAddr())
	t.Cleanup(func() { os.Unsetenv("MAGPIE_ADDR") })
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
