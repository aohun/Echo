package gui

// Exported wrappers over the gui package's own machinery, for Echo's shell
// (internal/shell): the settings page's writes and facts that live here —
// the gateway's port, the update state, the exchange rate, the Codex
// accounts and the terminals — rather than in a domain package.

// PortResult is what moving the gateway's port did: the agents whose
// configs moved with it, and what one couldn't.
type PortResult = portJSON

// SetPort moves the gateway to p, as Settings' Gateway port does: it
// checks the port is free, saves it, and moves every agent connected.
func SetPort(p int) (PortResult, error) { return setPort(p) }

// FX is a USD→CNY rate as the currency row shows it.
type FX = fxJSON

// FXNow is the rate known now, never waited for.
func FXNow() FX { return currentFX() }

// UpdateInfo is the updater's state, as the About tab draws it.
type UpdateInfo = updateJSON

// UpdateState is the updater's state now.
func UpdateState() UpdateInfo { return updates.json() }

// UpdateCheck asks the updater to look for a newer version now.
func UpdateCheck() { updates.check() }

// UpdateInstall installs what was downloaded, asking the reader's leave
// when ask is set. It reports whether the install is under way.
func UpdateInstall(ask bool) bool { return updates.install(ask) }

// CodexUsers are the ChatGPT accounts signed in, whose daily warm-up the
// per-account rows name.
func CodexUsers() []string { return codexUsers() }

// TerminalApp is one terminal app a session may open in.
type TerminalApp struct {
	ID   string
	Name string
}

// TerminalDiscovery is the terminals found on this computer.
type TerminalDiscovery struct {
	Apps    []TerminalApp
	Default string
}

// DiscoverTerminals looks for the terminal apps on this computer.
func DiscoverTerminals() (TerminalDiscovery, error) {
	found, err := discoverTerminals()
	if err != nil {
		return TerminalDiscovery{}, err
	}
	out := TerminalDiscovery{Default: found.Default}
	for _, app := range found.Apps {
		out.Apps = append(out.Apps, TerminalApp{ID: app.ID, Name: app.Name})
	}
	return out, nil
}
