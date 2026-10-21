//go:build nogui && echo

package gateway

import "testing"

// Echo's gateway is on Echo's own address, not magpie's.
//
// DefaultAddr is what a build without an address of its own serves on: a
// test that wires an agent to the gateway's default address, and a page
// that reports it. Left at magpie's 3425, that address would be the one
// magpie's own app serves — so an agent wired to it would reach magpie.
func TestDefaultAddrIsEchosOwn(t *testing.T) {
	if got, want := DefaultAddr, "127.0.0.1:3426"; got != want {
		t.Fatalf("DefaultAddr = %q, want %q", got, want)
	}
}
