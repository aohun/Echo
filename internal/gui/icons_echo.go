//go:build nogui && echo

package gui

import "io/fs"

// IconsFS is the bundled icons (internal/gui/assets/icons): the vendors',
// the agents' and the apps' pictures magpie's own page serves from
// /icons/*. Echo's native shell draws the same pictures from it.
//
// Like StartBackend and StopBackend (backend_echo.go), this is magpie's own
// asset store handed to the shell rather than a second copy of it: the
// pictures are the product's, and one copy of them is kept.
func IconsFS() fs.FS {
	f, err := fs.Sub(assets, "assets/icons")
	if err != nil {
		return nil
	}
	return f
}
