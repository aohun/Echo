//go:build nogui && echo

package shell

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/yetone/magpie/internal/appdir"
	"github.com/yetone/magpie/internal/edit"
	"github.com/yetone/magpie/internal/library"
	"github.com/yetone/magpie/internal/plugin"
	"github.com/yetone/magpie/internal/provider"
	"github.com/yetone/magpie/internal/settings"
	"github.com/yetone/magpie/internal/usage"
)

// Data is what the pages draw, read from magpie's domain packages.
//
// Every field is a plain value the view can render without touching the
// filesystem again: readData fills it once, off the main thread.
type Data struct {
	// Agents are the coding agents on this machine, with the ones magpie
	// knows but has not found after them.
	Agents []AgentRow
	// AgentInstalls are the uninstalled agents available for installation.
	AgentInstalls []AgentInstallRow
	// AgentProfiles are the saved configuration schemes (方案).
	AgentProfiles []ProfileRow
	// Providers are the configured vendors, as the Providers page's rows
	// draw them (provider.go).
	Providers []ProviderRow
	// Gateway is what the Gateway page draws (gateway_page.go).
	Gateway GatewayRow
	// Library is what the Library page and sidebar badges draw.
	Library LibraryData
	// Plugins is what the Plugins page draws (plugins_data.go).
	Plugins PluginsData
	// Routing is what the Routing page draws (routing_data.go).
	Routing RoutingData
	// Usage is what the Usage page draws (usage_data.go).
	Usage UsageData
	// Settings is what the Settings page draws (settings_data.go).
	Settings SettingsSnap
}

func lastDataCachePath() string {
	return filepath.Join(appdir.Cache(), "echo-shell-cache.json")
}

func loadDataCache() (Data, bool) {
	b, err := os.ReadFile(lastDataCachePath())
	if err != nil || len(b) == 0 {
		return Data{}, false
	}
	var d Data
	if err := json.Unmarshal(b, &d); err != nil {
		return Data{}, false
	}
	return d, true
}

func saveDataCache(d Data) {
	b, err := json.Marshal(d)
	if err != nil {
		return
	}
	_ = edit.WriteAtomic(lastDataCachePath(), b)
}

// LibraryData is what the Library page and its sidebar badge draw.
type LibraryData struct {
	Dir         string
	Backups     string
	CopySkills  bool
	CheckDue    bool
	Partial     bool
	Counts      LibraryCounts
	Skills      []SkillRow
	FoundSkills []FoundSkillRow
	NewSkills   []NewSkillRow
	SkillGroups []SkillGroupRow
	Agents      []LibAgentRow
	// Projects are the folders whose agents get some of the library's
	// skills as the project's own, and Problems what the last write
	// couldn't do in any of them.
	Projects []ProjectRow
	Problems []ProblemRow
}

// ProblemRow is one thing the last change couldn't write: an agent's
// instructions, its MCP file, a project, or a skill that wouldn't update.
// The page shows them in one card, so a chip's amber dot isn't the only
// place a failure is said.
type ProblemRow struct {
	Agent string // the agent's id, or a project's folder
	What  string // "instructions", "mcp:<name>", "skill:<name>", "project:<name>"
	Error string
	// Own is a skill the agent has a folder of its own for, not the
	// library's: the page offers to use the library's or keep the agent's
	Own bool
}

// ProjectRow is a folder given some of the library's skills as its own.
type ProjectRow struct {
	Dir      string
	Name     string
	Copy     bool
	Skills   map[string][]string // skill name -> the agents it is placed for
	Missing  bool                // the folder is gone
	Problems map[string]string   // by skill name ("" for the project's own)
	Placed   []string            // what magpie put there
}

// SkillGroupRow is a group the reader made of some skills: the page shows
// it above the ones by source, with its skills taken out of theirs.
type SkillGroupRow struct {
	Name   string
	Skills []string
}

// NewSkillRow is a skill a check found in a repository the library's
// skills came from, which the library doesn't have yet.
type NewSkillRow struct {
	ID          string
	Name        string
	Description string
	Repo        string
	Agents      []string
}

type LibraryCounts struct {
	Instructions int
	MCP          int
	Skills       int
	Hooks        int
	Subagents    int
	RTK          int
}

type SkillRow struct {
	Name        string
	Description string
	Source      string
	Kind        string // "github", "folder", or ""
	Origin      string
	Repo        string
	Icon        string
	Agents      []string
	Missing     bool
	Edited      bool
	Problems    map[string]string
	CheckStatus string // "update", "current", "unknown", or ""
	CheckURL    string
	CheckCommit string
	Behind      []string
	Always      []string
}

type FoundSkillRow struct {
	Name        string
	Description string
	Agents      []string
	Others      []string
	Copies      []string
	Link        string
	Shared      string
	Library     string
}

type LibAgentRow struct {
	ID       string
	Name     string
	Icon     string
	Skills   string
	Gone     bool
	MustCopy bool
	How      string
	Own      bool // HowOwn: its own way over the library's
	// ProjectSkills is the folder in a project this agent reads its own
	// skills from, "" for one that reads none: what the project's rows
	// give their chips.
	ProjectSkills string
}

// HowOwn is whether the agent is given its skills its own way, rather than
// the library's.
func (r LibAgentRow) HowOwn() bool { return r.Own }

// readLibraryData reads the library for the pages, and the problems the last
// change couldn't write.
//
// glance reads what doesn't walk each agent's skill folders through
// (library.Glance); the full read (library.Read) is what fills in the skills
// the agents have of their own — "In your agents" — and the copies that are
// out of date. The page draws the glance at once and takes the full read
// after it, which is magpie's own two reads (Glance's doc, #541).
func readLibraryData(problems []library.Problem, glance bool) LibraryData {
	var v *library.View
	var err error
	if glance {
		v, err = library.Glance(problems)
	} else {
		v, err = library.Read(problems)
	}
	if err != nil || v == nil {
		return LibraryData{
			Dir:     library.Dir(),
			Backups: library.BackupDir(),
		}
	}
	instrCount := 0
	if v.Instructions != nil {
		instrCount = len(v.Instructions.Sets)
	}
	ld := LibraryData{
		Dir:        v.Dir,
		Backups:    v.Backups,
		CopySkills: v.CopySkills,
		CheckDue:   v.CheckDue,
		Partial:    v.Partial,
		Counts: LibraryCounts{
			Instructions: instrCount,
			MCP:          len(v.Servers),
			Skills:       len(v.Skills),
			Hooks:        len(v.Hooks),
			Subagents:    len(v.Subagents),
			RTK:          0,
		},
	}
	for _, g := range v.SkillGroups {
		ld.SkillGroups = append(ld.SkillGroups, SkillGroupRow{Name: g.Name, Skills: g.Skills})
	}
	for _, n := range v.NewSkills {
		ld.NewSkills = append(ld.NewSkills, NewSkillRow{
			ID:          n.ID,
			Name:        n.Name,
			Description: n.Description,
			Repo:        n.Repo,
			Agents:      n.Agents,
		})
	}
	for _, a := range v.Agents {
		ld.Agents = append(ld.Agents, LibAgentRow{
			ID:            a.ID,
			Name:          a.Name,
			Icon:          a.Icon,
			Skills:        a.Skills,
			Gone:          a.Gone,
			MustCopy:      a.MustCopy,
			How:           a.How,
			Own:           a.HowOwn,
			ProjectSkills: a.ProjectSkills,
		})
	}
	for _, p := range v.Projects {
		pr := ProjectRow{
			Dir:      p.Dir,
			Name:     p.Name,
			Copy:     p.Copy,
			Skills:   p.Skills,
			Missing:  p.Missing,
			Problems: p.Problems,
			Placed:   p.Placed,
		}
		ld.Projects = append(ld.Projects, pr)
	}
	for _, p := range problems {
		ld.Problems = append(ld.Problems, ProblemRow{
			Agent: p.Agent,
			What:  p.What,
			Error: p.Error,
			Own:   p.Own,
		})
	}
	for _, s := range v.Skills {
		sr := SkillRow{
			Name:        s.Name,
			Description: s.Description,
			Source:      s.Source,
			Kind:        s.Kind,
			Origin:      s.Origin,
			Repo:        s.Repo,
			Icon:        s.Icon,
			Agents:      s.Agents,
			Missing:     s.Missing,
			Edited:      s.Edited,
			Problems:    s.Problems,
			Behind:      s.Behind,
			Always:      s.Always,
		}
		if s.Check != nil {
			sr.CheckStatus = s.Check.Status
			sr.CheckURL = s.Check.URL
			sr.CheckCommit = s.Check.Commit
		}
		ld.Skills = append(ld.Skills, sr)
	}
	for _, f := range v.FoundSkills {
		ld.FoundSkills = append(ld.FoundSkills, FoundSkillRow{
			Name:        f.Name,
			Description: f.Description,
			Agents:      f.Agents,
			Others:      f.Others,
			Copies:      f.Copies,
			Link:        f.Link,
			Shared:      f.Shared,
			Library:     f.Library,
		})
	}
	return ld
}

// readData reads everything the pages draw. glance is passed to the library's
// own read: true leaves out what walks each agent's skill folders through.
//
// It runs off the main thread: agent.All walks the home directory and the
// catalog, and provider.All reads settings, so neither belongs in a frame.
//
// It never reads a real agent config for its content — only for existence, as
// magpie's own detection does — so pointing it at a test home is enough to keep
// it off the reader's real setup (the rule magpie's tests follow; see
// docs/06-open-questions-and-risks.md, risk O12).
// slowRead logs a page read that outlasted its share of the reload: what
// a slow pass paid can be read rather than felt.
func slowRead(what string, since time.Time) {
	if d := time.Since(since); d > 300*time.Millisecond {
		log.Printf("echo: slow read: %-10s %dms", what, d.Milliseconds())
	}
}

// readDataBox is how long a read of the whole bundle may take before the
// last good data stands in: the instance's answers can wait on the
// network through its plugin host (an auth refresh against a vendor, a
// newer Bun looked up on GitHub), and the reload cannot pay that every
// pass. A var so a test can shorten it.
var readDataBox = 2 * time.Second

// readData reads the bundle behind a watchdog: past the box, the last
// good data stands in at once and the read that finally lands is kept
// for the next pass. Inside, the plugins read runs beside the others —
// it is the one that can wait seconds for its document, so it must not
// stand in front of them either.
func readData(problems []library.Problem, glance bool) Data {
	done := make(chan Data, 1)
	go func() { done <- readDataAll(problems, glance) }()
	select {
	case d := <-done:
		return d
	case <-time.After(readDataBox):
		go func() {
			// keep the read going: when it finally lands, it becomes
			// the last good data for the next pass
			if d := <-done; d.Plugins.Installed != nil || d.Agents != nil {
				saveDataCache(d)
			}
		}()
		if cached, ok := loadDataCache(); ok {
			return cached
		}
		return Data{}
	}
}

func readDataAll(problems []library.Problem, glance bool) Data {
	// The plugins read is the one that can wait seconds for its document
	// (its own box on a magpie web instance, whose plugin host can wait
	// on the network): read it beside the others rather than in front of
	// them. The glance does not even wait for it — the last good cut
	// stands in, and the full read behind the glance brings the fresh
	// rows — so the first paint lands when the fastest of the reads does.
	pluginsCh := make(chan PluginsData, 1)
	go func() { pluginsCh <- readPluginsData() }()
	t0 := time.Now()
	agents, installs, profiles := readAgentPageData()
	slowRead("agents", t0)
	t0 = time.Now()
	providers := readProviders()
	slowRead("providers", t0)
	t0 = time.Now()
	gatewayRow := readGateway()
	slowRead("gateway", t0)
	t0 = time.Now()
	library := readLibraryData(problems, glance)
	slowRead("library", t0)
	t0 = time.Now()
	routing := readRoutingData()
	slowRead("routing", t0)
	t0 = time.Now()
	usageData := ReadUsageData(UsageQuery{Tab: UsTabOverview, Period: usage.Month, Limit: 50, SessRange: "30d", CtxDays: 7})
	slowRead("usage", t0)
	t0 = time.Now()
	settingsRow := readSettings()
	slowRead("settings", t0)
	t0 = time.Now()
	var pluginsRow PluginsData
	if glance {
		// the glance does not wait: the last good cut stands in, and the
		// full read behind it brings the fresh rows
		select {
		case pluginsRow = <-pluginsCh:
		default:
			if cached, ok := loadDataCache(); ok {
				pluginsRow = cached.Plugins
			} else {
				pluginsRow = PluginsData{
					Bun:      plugin.HasBun(),
					BunVer:   plugin.BunInUse(),
					Mirror:   settings.Load().ChinaMirror,
					On:       map[string]bool{},
					Movable:  []provider.MoveCandidate{},
					OffPages: map[string]bool{},
				}
			}
		}
	} else {
		pluginsRow = <-pluginsCh
	}
	d := Data{
		Agents:        agents,
		AgentInstalls: installs,
		AgentProfiles: profiles,
		Providers:     providers,
		Gateway:       gatewayRow,
		Plugins:       pluginsRow,
		Library:       library,
		Routing:       routing,
		Usage:         usageData,
		Settings:      settingsRow,
	}
	saveDataCache(d)
	return d
}
