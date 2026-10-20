//go:build nogui && echo

package shell

import (
	"fmt"
	"os"
)

// EchoPort is the port Echo's gateway listens on.
//
// It is Echo's own, not magpie's 3425, and that is the point: two products
// serving the same port hand over to whichever started first. An Echo
// launched beside magpie would then show a window whose gateway is magpie's
// — magpie's providers, magpie's routes, and every request an agent sends
// going through it rather than through Echo.
//
// MAGPIE_ADDR still comes first: a launch that names an address of its own
// (a container, a second copy on a port of its own) must keep it.
const EchoPort = 3426

// echoAddr is where Echo's gateway listens: EchoPort on loopback, unless
// MAGPIE_ADDR names an address.
//
// The port goes into MAGPIE_ADDR before the gateway starts, because that
// variable is the one lever the whole domain reads the address through: the
// gateway's listener, the URL it reports, every agent's config, and every
// page. Changing it in one place is what keeps those together — a shell that
// only drew a different number would leave the gateway listening on 3425.
func echoAddr() string {
	if a := os.Getenv("MAGPIE_ADDR"); a != "" {
		return a
	}
	return fmt.Sprintf("127.0.0.1:%d", EchoPort)
}
