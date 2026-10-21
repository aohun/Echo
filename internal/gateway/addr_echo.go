//go:build nogui && echo

package gateway

import (
	"fmt"

	"github.com/yetone/magpie/internal/echoport"
)

// Echo's gateway is on Echo's own address, not magpie's.
//
// DefaultAddr is what a build without an address of its own serves on: a
// test that wires an agent to the gateway's default address, and a page
// that reports it. Left at magpie's 3425, that address would be the one
// magpie's own app serves — so an agent wired to it would reach magpie.
//
// It is set here, in a file the echo tag alone compiles, so magpie's own
// builds keep magpie's address and are otherwise untouched.
func init() { DefaultAddr = fmt.Sprintf("127.0.0.1:%d", echoport.Port) }
