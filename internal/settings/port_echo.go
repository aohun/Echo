//go:build nogui && echo

package settings

import "github.com/yetone/magpie/internal/echoport"

// Echo's gateway is on Echo's own port, and the pages that draw the
// gateway's port must draw that one.
//
// DefaultPort is what a settings file without a port means, and the
// Settings page shows it as the port in the field, and the gateway falls
// back to it when Settings name none. Left at magpie's 3425, Echo's own
// pages would show magpie's port beside a gateway serving on Echo's — and
// a user changing the port in the field would be moved onto magpie's
// port, the one magpie's own app serves.
//
// It is set here, in a file the echo tag alone compiles, so magpie's own
// builds (the Wails app and the terminal CLI) keep magpie's port and are
// otherwise untouched.
func init() { DefaultPort = echoport.Port }
