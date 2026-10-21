//go:build nogui && echo

package echoport

import "testing"

// Echo's gateway is on Echo's own port, not magpie's.
//
// This is the whole of the "not just the UI" claim: the domain reads the
// gateway's address through MAGPIE_ADDR, so an Echo that did not set it
// would serve on magpie's 3425. Two products on one port hand over to
// whichever started first, so an Echo launched beside magpie would show a
// window whose gateway is magpie's — magpie's providers and routes, and
// every request an agent sends going through it.
func TestAddrIsEchosOwnPort(t *testing.T) {
	if got, want := Addr(), "127.0.0.1:3426"; got != want {
		t.Fatalf("Addr() = %q, want %q: Echo would not serve on its own port", got, want)
	}
	if Addr() == "127.0.0.1:3425" {
		t.Fatal("Addr() is magpie's own address: Echo would share magpie's port")
	}
}
