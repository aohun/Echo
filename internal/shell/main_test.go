//go:build nogui && echo

package shell

import (
	"testing"

	"github.com/yetone/magpie/internal/testenv"
	// The built-in plugins, as the app's own main.go gathers them. magpie's
	// rule is that core never imports a plugin and main.go alone gathers
	// them — and that a test may too, which is what this is. Without it the
	// Plugins page is empty and every plugin-backed feature is missing, so
	// the shell would be tested against a smaller product than it ships.
	_ "github.com/yetone/magpie/plugins/all"
)

// TestMain gives the package a home of its own, so no test reads or writes
// the reader's own agent configs, keychain or settings — the rule magpie's
// tests follow (risk O12 in docs/06-open-questions-and-risks.md).
func TestMain(m *testing.M) { testenv.Main(m) }
