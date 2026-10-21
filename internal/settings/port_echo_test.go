//go:build nogui && echo

package settings

import "testing"

// The port Echo's pages draw is the port Echo's gateway serves on.
//
// DefaultPort is what a settings file without a port means, and the
// Settings page shows it as the port in the field. Left at magpie's 3425,
// Echo's own page would show magpie's port beside a gateway serving on
// Echo's — and a user changing the port in that field would be moved onto
// magpie's port, the one magpie's own app serves.
func TestDefaultPortIsEchosOwn(t *testing.T) {
	if got, want := DefaultPort, 3426; got != want {
		t.Fatalf("DefaultPort = %d, want %d: the pages would draw magpie's port", got, want)
	}
}

// The address the domain falls back to is Echo's own too: a settings file
// with no port, and no MAGPIE_ADDR, is an Echo serving on Echo's port.
func TestSavedAddrIsEchosOwn(t *testing.T) {
	t.Setenv("MAGPIE_ADDR", "")
	// a home of the test's own, so the settings read are the defaults and
	// not whatever this computer has saved: a saved port would be a port
	// of its own, which is not what this check is about
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", home)
	if got, want := SavedAddr(), "127.0.0.1:3426"; got != want {
		t.Fatalf("SavedAddr() = %q, want %q", got, want)
	}
}
