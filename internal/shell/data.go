//go:build nogui && echo

package shell

import (
	"github.com/yetone/magpie/internal/agent"
	"github.com/yetone/magpie/internal/catalog"
	"github.com/yetone/magpie/internal/pluginrt"
	"github.com/yetone/magpie/internal/provider"
)

// Data is what the pages draw, read from magpie's domain packages.
//
// Every field is a plain value the view can render without touching the
// filesystem again: readData fills it once, off the main thread.
type Data struct {
	// Agents are the coding agents on this machine, with the ones magpie
	// knows but has not found after them.
	Agents []AgentRow
	// Providers are the configured vendors.
	Providers []ProviderRow
	// Sessions, Library, Usage, Routing and Plugins are not read yet; each
	// page says so until its own read is written.
	Plugins []PluginRow
}

// AgentRow is one coding agent as the Agents page lists it.
type AgentRow struct {
	ID   string
	Name string
	// Found is whether the agent is present on this machine.
	Found bool
	// Path is the config file magpie edits for it.
	Path string
	// Plugin is the id of the plugin that adds it, "" for magpie's own.
	Plugin string
}

// ProviderRow is one vendor as the Providers page lists it.
type ProviderRow struct {
	ID   string
	Name string
	// Models is how many models the catalog serves for it.
	Models int
	// Keyed is whether an API key is saved for it.
	Keyed bool
	// SignedIn is whether an account is signed in for it.
	SignedIn bool
	// Off is whether the reader switched the vendor off.
	Off bool
}

// PluginRow is one plugin as the Plugins page lists it.
type PluginRow struct {
	ID      string
	Name    string
	Sub     string
	Enabled bool
}

// readData reads everything the pages draw.
//
// It runs off the main thread: agent.All walks the home directory and the
// catalog, and provider.All reads settings, so neither belongs in a frame.
//
// It never reads a real agent config for its content — only for existence, as
// magpie's own detection does — so pointing it at a test home is enough to keep
// it off the reader's real setup (the rule magpie's tests follow; see
// docs/06-open-questions-and-risks.md, risk O12).
func readData() Data {
	return Data{
		Agents:    readAgents(),
		Providers: readProviders(),
		Plugins:   readPlugins(),
	}
}

// readAgents lists the coding agents magpie knows, present ones first.
//
// agent.All is the same list magpie's own Agents page draws, detected or not;
// agent.Detected filters it to the ones installed here.
func readAgents() []AgentRow {
	found := make(map[string]bool)
	for _, a := range agent.Detected() {
		found[a.ID] = true
	}
	all := agent.All()
	rows := make([]AgentRow, 0, len(all))
	for _, a := range all {
		rows = append(rows, AgentRow{
			ID:     a.ID,
			Name:   a.Name,
			Found:  found[a.ID],
			Path:   a.Path,
			Plugin: a.Plugin,
		})
	}
	// Present ones first, each group keeping magpie's display order.
	out := make([]AgentRow, 0, len(rows))
	for _, r := range rows {
		if r.Found {
			out = append(out, r)
		}
	}
	for _, r := range rows {
		if !r.Found {
			out = append(out, r)
		}
	}
	return out
}

// readProviders lists the configured vendors with what each page shows.
func readProviders() []ProviderRow {
	all := provider.All()
	rows := make([]ProviderRow, 0, len(all))
	for _, p := range all {
		models := 0
		if !p.Off {
			models = len(catalog.Provider(p.ID))
		}
		rows = append(rows, ProviderRow{
			ID:       p.ID,
			Name:     p.Name,
			Models:   models,
			Keyed:    p.Key != "",
			SignedIn: p.Account != nil,
			Off:      p.Off,
		})
	}
	return rows
}

// readPlugins lists the plugins registered with the runtime.
//
// The runtime is magpie's own (internal/pluginrt): plugins/all registers the
// built-ins at startup, so what is registered is what is built in. A plugin
// that is switched off in settings still appears, so the page can offer it.
func readPlugins() []PluginRow {
	ms := pluginrt.Registered()
	rows := make([]PluginRow, 0, len(ms))
	for _, m := range ms {
		rows = append(rows, PluginRow{
			ID:      m.ID,
			Name:    m.Name,
			Sub:     m.Sub,
			Enabled: true,
		})
	}
	return rows
}
