//go:build nogui && echo

package proc

import "testing"

// Echo's own binary is ours, so the gateway's take-over and the port
// guard can tell it from another program: both ask whether the program
// holding a port is one of ours before they touch it, and left out Echo's
// own binary would be read as some other program's.
func TestEchoIsOurs(t *testing.T) {
	for _, path := range []string{
		"/Applications/Echo.app/Contents/MacOS/Echo",
		"/Users/u/Applications/Echo.app/Contents/MacOS/Echo",
	} {
		if !IsMagpie(path) {
			t.Errorf("IsMagpie(%q) = false, want true: Echo's own binary is ours", path)
		}
	}
}

// The shell's echo is not ours.
//
// It is called the same thing as Echo's binary, and the take-over ends
// the program it is sure is ours — so a match on the name alone would
// have it end /bin/echo. Only a program inside an app bundle is ours.
func TestTheShellsEchoIsNotOurs(t *testing.T) {
	for _, path := range []string{
		"/bin/echo",
		"/usr/bin/echo",
		"/usr/local/bin/echo",
	} {
		if IsMagpie(path) {
			t.Errorf("IsMagpie(%q) = true, want false: the shell's echo is not ours", path)
		}
	}
}
