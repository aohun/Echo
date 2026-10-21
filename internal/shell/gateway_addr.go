//go:build nogui && echo

package shell

import (
	"github.com/yetone/magpie/internal/echoport"
	"github.com/yetone/magpie/internal/settings"
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
//
// The number itself lives in internal/echoport, because the window is not
// the only way into Echo: the executable used as a command line reads the
// same port, and the two must never drift.
const EchoPort = echoport.Port

// echoAddr is where Echo's gateway listens: EchoPort on loopback, unless
// MAGPIE_ADDR names an address.
//
// The address is the domain's own (settings.GatewayAddr): MAGPIE_ADDR
// where it is set, else the port the settings name, else Echo's own. That
// is the same address the gateway listens on, the one the pages and the
// tray draw, and the one every agent's config names — so they cannot
// drift apart, and a reader who changed the port in the Settings page sees
// it everywhere.
//
// It is not put into MAGPIE_ADDR to pin it: that variable comes before
// the settings, so setting it would make the port the reader chooses in
// the Settings page ineffective. The port belongs where the reader can
// change it, and the default is Echo's own.
func echoAddr() string { return settings.GatewayAddr() }
