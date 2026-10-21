// Package echoport is Echo's gateway port: the one place that says which
// port Echo's gateway listens on.
//
// It exists because that port was once the shell's business alone
// (internal/shell/gateway_addr.go), and the shell is not the only way into
// Echo: the executable used as a command line never went through it. So
// the window served on Echo's port and the command line on magpie's 3425 —
// a terminal command wrote magpie's address into the agents' configs, and
// the agents' requests went to magpie: magpie's providers, magpie's
// routes, not Echo's.
//
// The port is here rather than in the shell because both must agree, and
// because the domain's own default (settings.DefaultPort) is what the
// pages draw and what a settings file without a port means. This package
// names it; internal/settings and internal/gateway take it from here.
package echoport

import "fmt"

// Port is the port Echo's gateway listens on.
//
// It is Echo's own, not magpie's 3425, and that is the point: two products
// serving the same port hand over to whichever started first. An Echo
// launched beside magpie would then show a window whose gateway is
// magpie's — magpie's providers, magpie's routes, and every request an
// agent sends going through it rather than through Echo.
const Port = 3426

// Addr is where Echo's gateway listens, on loopback.
//
// It is the address of the default port, not of whatever the reader's
// settings or MAGPIE_ADDR say: those are the domain's business
// (settings.GatewayAddr), and this package only names the port they fall
// back to.
func Addr() string { return fmt.Sprintf("127.0.0.1:%d", Port) }
