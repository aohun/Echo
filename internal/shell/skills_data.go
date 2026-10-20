//go:build nogui && echo

package shell

import (
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/yetone/magpie/internal/library"
)

// The Skills page's own reads and writes: how the library's skills are
// grouped, the way each agent is given them, and the calls behind the
// page's buttons.
//
// Everything here is magpie's domain code (internal/library) seen through
// the shell's lens: the page holds no state of its own beyond what the
// reader picked, and a write ends with the library read again.

// skillGroup is one group of the page's list: the reader's own group, a
// repository's skills, or the ones kept on this computer.
type skillGroup struct {
	// Key is the group's identity across frames, as the fold state keeps
	// it: "my:<name>", "gh:<repo>", "owner:<owner>" or "local".
	Key string
	// Repo is the repository the skills came from, "" for the ones on
	// this computer and for the reader's own groups.
	Repo string
	// Mine is the reader's group's name, when the group is one.
	Mine string
	// Owner is set when the group is one author's repositories together.
	Owner bool
	// Skills are the group's skills, by name.
	Skills []SkillRow
	// Total is how many the group has, before the filter: a heading says
	// "n of total" while one is on.
	Total int
}

// skillRepo is the repository a skill came from, however it came in:
// magpie traces one not installed from GitHub too, so a repository's skills
// are one group.
func skillRepo(s SkillRow) string {
	if s.Repo != "" {
		return s.Repo
	}
	switch {
	case s.Kind == "github":
		return repoOf(s.Source)
	case s.Origin != "":
		return repoOf(s.Origin)
	}
	return ""
}

// repoOf cuts a GitHub URL down to owner/repo; anything else is returned as
// it is, so a host's path stays one group.
func repoOf(u string) string {
	u = strings.TrimPrefix(u, "https://github.com/")
	u = strings.TrimPrefix(u, "http://github.com/")
	u = strings.TrimSuffix(u, ".git")
	parts := strings.Split(u, "/")
	if len(parts) < 2 {
		return u
	}
	return parts[0] + "/" + parts[1]
}

// repoURL is a repository as the page opens it: github.com for an owner/repo,
// the host itself for another's path.
func repoURL(repo string) string {
	if repo == "" {
		return ""
	}
	first := strings.SplitN(repo, "/", 2)[0]
	if strings.Contains(first, ".") {
		return "https://" + repo
	}
	return "https://github.com/" + repo
}

// skillGroups is the list's groups, as magpie's own skillGroups() works
// them out: the reader's own groups first, then the repositories installed
// from GitHub, then the ones on this computer.
func (a *App) skillGroups() []skillGroup {
	if a.libSort != 0 {
		// a flat list, A→Z or Z→A: one group with no heading over it
		skills := append([]SkillRow(nil), a.data.Library.Skills...)
		sort.SliceStable(skills, func(i, j int) bool {
			less := strings.ToLower(skills[i].Name) < strings.ToLower(skills[j].Name)
			if a.libSort == 2 {
				return !less
			}
			return less
		})
		return []skillGroup{{Key: "flat", Skills: skills}}
	}

	// the reader's own groups, each skill in its group rather than its
	// source's
	mine := make([]skillGroup, 0, len(a.data.Library.SkillGroups))
	inMine := map[string]int{}
	for _, g := range a.data.Library.SkillGroups {
		mine = append(mine, skillGroup{Key: "my:" + g.Name, Mine: g.Name})
		for _, n := range g.Skills {
			inMine[n] = len(mine) - 1
		}
	}

	by := map[string]*skillGroup{}
	order := []string{}
	for _, s := range a.data.Library.Skills {
		if i, ok := inMine[s.Name]; ok {
			mine[i].Skills = append(mine[i].Skills, s)
			continue
		}
		repo := skillRepo(s)
		key := "local"
		if repo != "" {
			key = "gh:" + strings.ToLower(repo)
		}
		g, ok := by[key]
		if !ok {
			g = &skillGroup{Key: key, Repo: repo}
			by[key] = g
			order = append(order, key)
		}
		g.Skills = append(g.Skills, s)
	}

	// online first (installed from GitHub, or traced to one), this
	// computer's below
	var up, down []*skillGroup
	for _, k := range order {
		g := by[k]
		if k == "local" {
			continue
		}
		online := false
		for _, s := range g.Skills {
			if s.Kind == "github" || s.Origin != "" {
				online = true
				break
			}
		}
		if online {
			up = append(up, g)
		} else {
			down = append(down, g)
		}
	}

	// one author's repositories are one group, each a part when many
	owners := map[string][]*skillGroup{}
	ownerOrder := []string{}
	for _, g := range up {
		if strings.Contains(g.Repo, ".") && !strings.HasPrefix(g.Repo, "github.com/") {
			// another git host's: left as its own group
			continue
		}
		o := strings.ToLower(strings.SplitN(g.Repo, "/", 2)[0])
		if _, ok := owners[o]; !ok {
			ownerOrder = append(ownerOrder, o)
		}
		owners[o] = append(owners[o], g)
	}
	merged := make([]*skillGroup, 0, len(up))
	done := map[*skillGroup]bool{}
	for _, o := range ownerOrder {
		gs := owners[o]
		if len(gs) < 2 {
			continue
		}
		all := &skillGroup{Key: "owner:" + o, Repo: strings.SplitN(gs[0].Repo, "/", 2)[0], Owner: true}
		for _, g := range gs {
			all.Skills = append(all.Skills, g.Skills...)
			done[g] = true
		}
		merged = append(merged, all)
	}
	for _, g := range up {
		if !done[g] {
			merged = append(merged, g)
		}
	}

	byRepo := func(list []*skillGroup) {
		sort.SliceStable(list, func(i, j int) bool { return list[i].Repo < list[j].Repo })
	}
	byRepo(merged)
	byRepo(down)

	out := make([]skillGroup, 0, len(mine)+len(merged)+len(down)+1)
	for _, g := range mine {
		if len(g.Skills) > 0 {
			out = append(out, g)
		}
	}
	for _, g := range merged {
		out = append(out, *g)
	}
	for _, g := range down {
		out = append(out, *g)
	}
	if g, ok := by["local"]; ok {
		out = append(out, *g)
	}

	for i := range out {
		skills := out[i].Skills
		sort.SliceStable(skills, func(x, y int) bool {
			return strings.ToLower(skills[x].Name) < strings.ToLower(skills[y].Name)
		})
		out[i].Total = len(skills)
	}
	return out
}

// skillWords is the filter's words, lower-cased.
func skillWords(q string) []string {
	return strings.FieldsFunc(strings.ToLower(strings.TrimSpace(q)), func(r rune) bool {
		return r == ' ' || r == '\t' || r == '\n'
	})
}

// skillHits are the group's skills the filter keeps.
func skillHits(g skillGroup, words []string) []SkillRow {
	if len(words) == 0 {
		return g.Skills
	}
	out := make([]SkillRow, 0, len(g.Skills))
	for _, s := range g.Skills {
		hay := strings.ToLower(s.Name + " " + s.Description + " " + skillRepo(s))
		hit := true
		for _, w := range words {
			if !strings.Contains(hay, w) {
				hit = false
				break
			}
		}
		if hit {
			out = append(out, s)
		}
	}
	return out
}

// groupFolded is whether a group shows folded: as the reader left it, and
// folded from the start when the library holds many skills.
func (a *App) groupFolded(key string, filtering bool) bool {
	if a.libFoldedGroups == nil {
		a.libFoldedGroups = map[string]bool{}
	}
	if v, ok := a.libFoldedGroups[key]; ok {
		return v
	}
	return !filtering && len(a.data.Library.Skills) > skillsMany
}

// skillsMany is the count a group is folded from the start at, and split
// into parts at: magpie's own MANY.
const skillsMany = 40

// skillsSplit is the count a group is split at, when its skills sit in more
// than one folder or share a start in their names: magpie's own SPLIT.
const skillsSplit = 8

// skillPart is one part of a big group: the skills that sit in one folder
// of the repository, or that share the start of their names. A group small
// enough to read as it is has none.
type skillPart struct {
	// Key is the part's identity across frames, "group\npart".
	Key string
	// Label is what its heading says: the folder, the shared start, or —
	// with no key — the skills left over.
	Label string
	// Mono is whether the label is a path, drawn in the code font.
	Mono bool
	// Skills are the part's skills.
	Skills []SkillRow
}

// groupParts splits a group too big to read as one list, as magpie's own
// partsOf works it out: by the folder its skills sit in when they sit in
// more than one, else by the start their names share — a start three or
// more share being a part and the rest one more. Nothing is fetched: it is
// all in where each skill came from and what it is called.
func (a *App) groupParts(g skillGroup) []skillPart {
	if g.Mine != "" || len(g.Skills) <= skillsSplit {
		return nil
	}
	byStart := func() map[string][]SkillRow {
		segs := func(s SkillRow) []string {
			return strings.FieldsFunc(strings.ToLower(s.Name), func(r rune) bool {
				return r == '-' || r == '_' || r == ':' || r == '.' || r == ' ' || r == '\t'
			})
		}
		all := make([][]string, len(g.Skills))
		for i, s := range g.Skills {
			all[i] = segs(s)
		}
		// a start every name has says nothing: the part after it does
		skip := 0
		for {
			ok := true
			for _, x := range all {
				if len(x) <= skip+1 || x[skip] != all[0][skip] {
					ok = false
					break
				}
			}
			if !ok {
				break
			}
			skip++
		}
		lead := map[string]int{}
		for _, x := range all {
			if len(x) > skip+1 {
				lead[x[skip]]++
			}
		}
		m := map[string][]SkillRow{}
		for i, s := range g.Skills {
			k := ""
			if len(all[i]) > skip+1 {
				k = all[i][skip]
			}
			if lead[k] < 3 {
				k = ""
			}
			m[k] = append(m[k], s)
		}
		return m
	}

	if g.Repo != "" {
		// the folder in the repository is known of those installed from it
		// only: one traced to it goes by its name with the rest
		folders := map[string][]SkillRow{}
		allGitHub := true
		for _, s := range g.Skills {
			if s.Kind != "github" {
				allGitHub = false
				break
			}
			folders[skillFolderIn(s)] = append(folders[skillFolderIn(s)], s)
		}
		mono := allGitHub && len(folders) > 1
		m := folders
		if !mono {
			m = byStart()
		}
		if len(m) < 2 {
			return nil
		}
		return a.skillPartsOf(g, m, mono)
	}

	starts := byStart()
	named := map[string][]SkillRow{}
	for k, v := range starts {
		if k != "" {
			named[k] = v
		}
	}
	rest := starts[""]
	folders := map[string][]SkillRow{}
	for _, s := range rest {
		folders[skillFolderIn(s)] = append(folders[skillFolderIn(s)], s)
	}
	if len(folders) > 1 {
		parts := a.skillPartsOf(g, named, false)
		dirParts := a.skillPartsOf(g, folders, true)
		for i := range dirParts {
			dirParts[i].Key = g.Key + "\ndir:" + dirParts[i].Key
		}
		parts = append(parts, dirParts...)
		if len(parts) < 2 {
			return nil
		}
		return parts
	}
	if len(named) > 0 {
		parts := a.skillPartsOf(g, named, false)
		if len(rest) > 0 {
			parts = append(parts, skillPart{Key: g.Key + "\n", Skills: rest})
		}
		if len(parts) < 2 {
			return nil
		}
		return parts
	}
	return nil
}

// skillPartsOf turns a split into the parts the list draws, in the order
// magpie's own does: a path first, then by the key, the leftovers last.
func (a *App) skillPartsOf(g skillGroup, m map[string][]SkillRow, mono bool) []skillPart {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.SliceStable(keys, func(i, j int) bool {
		if (keys[i] == "") != (keys[j] == "") {
			return keys[j] == ""
		}
		return keys[i] < keys[j]
	})
	out := make([]skillPart, 0, len(keys))
	for _, k := range keys {
		label := k
		switch {
		case k != "":
		case mono && g.Repo != "":
			label = a.text.S.LibAtRepoTop
		case mono:
			label = a.text.S.LibKeptInLibrary
		default:
			label = a.text.S.LibOthers
		}
		out = append(out, skillPart{Key: g.Key + "\n" + k, Label: label, Mono: mono && k != "", Skills: m[k]})
	}
	return out
}

// skillFolderIn is the folder in the repository a skill sits in: the path
// before the skill's own folder, for one installed from GitHub, and the
// folder it was linked from for one on this computer.
func skillFolderIn(s SkillRow) string {
	if s.Kind == "github" {
		// …/tree/<ref>/<path to the folder>/<skill>
		u := s.Source
		i := strings.Index(u, "/tree/")
		if i < 0 {
			return ""
		}
		rest := u[i+len("/tree/"):]
		segs := strings.Split(rest, "/")
		if len(segs) < 3 {
			return ""
		}
		return strings.Join(segs[1:len(segs)-1], "/")
	}
	if s.Kind == "folder" {
		src := strings.TrimRight(s.Source, "/\\")
		if j := strings.LastIndexAny(src, "/\\"); j >= 0 {
			return src[:j]
		}
	}
	return ""
}

// skillAgents are the agents on this computer that can take skills: the
// ones the page gives them to, gone ones left out.
func (a *App) skillAgents() []LibAgentRow {
	out := make([]LibAgentRow, 0, len(a.data.Library.Agents))
	for _, ag := range a.data.Library.Agents {
		if ag.Skills == "" || ag.Gone {
			continue
		}
		out = append(out, ag)
	}
	return out
}

// libAgentByID is one agent as the page's rows name it.
func (a *App) libAgentByID(id string) (LibAgentRow, bool) {
	for _, ag := range a.data.Library.Agents {
		if ag.ID == id {
			return ag, true
		}
	}
	return LibAgentRow{}, false
}

// skillGives is whether the skill is given to the agent.
func skillGives(s SkillRow, id string) bool {
	for _, x := range s.Agents {
		if x == id {
			return true
		}
	}
	return false
}

// skillAlways is whether the agent reads the skill itself, from the shared
// folder, whatever the library gives it.
func skillAlways(s SkillRow, id string) bool {
	for _, x := range s.Always {
		if x == id {
			return true
		}
	}
	return false
}

// ---- the writes ---------------------------------------------------------

// skWrite runs one of the library's writes off the main thread, then reads
// the pages again. label names it while it runs, so the button that started
// it says so and a second click is ignored.
//
// What the write couldn't do is kept in libProblems, as magpie keeps its
// Result's: an agent whose file wouldn't take a skill is said in the page's
// card, not only by a chip's amber dot.
func (a *App) skWrite(label string, fn func() (*library.Result, error)) {
	if a.skBusy != "" {
		return
	}
	a.skBusy = label
	a.invalidate()
	go func() {
		res, err := fn()
		run := func() {
			a.skBusy = ""
			if err != nil {
				a.libNotice = err.Error()
			} else {
				a.libNotice = ""
			}
			if res != nil {
				a.libProblems = res.Problems
			}
			// An update refused because the skill was changed here is
			// the page's question, not an error: magpie asks whether to
			// put GitHub's version in place, and keeps the reader's in
			// the backups when they say yes.
			var edited *library.EditedError
			if errors.As(err, &edited) {
				a.skAskReplace = edited.Name
				a.libNotice = ""
			}
		}
		if a.win == nil {
			run()
			return
		}
		a.win.Update(run)
		a.Reload()
	}()
}

// skSetAgents gives the skill to exactly the agents picked.
func (a *App) skSetAgents(name string, agents []string) {
	a.skWrite("skills/agents:"+name, func() (*library.Result, error) {
		return library.SkillAgents(name, agents)
	})
}

// skToggleAgent gives the skill to an agent, or takes it away.
func (a *App) skToggleAgent(s SkillRow, id string) {
	next := make([]string, 0, len(s.Agents)+1)
	had := false
	for _, x := range s.Agents {
		if x == id {
			had = true
			continue
		}
		next = append(next, x)
	}
	if !had {
		next = append(next, id)
	}
	a.skSetAgents(s.Name, next)
}

// skUpdate fetches one skill again from GitHub.
func (a *App) skUpdate(name string, replace bool) {
	a.skWrite("skills/update:"+name, func() (*library.Result, error) {
		return library.UpdateSkill(name, replace)
	})
}

// skUpdateAll fetches every skill that GitHub changed again.
func (a *App) skUpdateAll(names []string) {
	a.skWrite("skills/update-all", func() (*library.Result, error) {
		if len(names) > 0 {
			return library.UpdateSomeSkills(names)
		}
		return library.UpdateSkills()
	})
}

// skRemove takes a skill out of every agent and out of the library.
func (a *App) skRemove(name string) {
	a.skWrite("skills/remove:"+name, func() (*library.Result, error) {
		return library.RemoveSkill(name)
	})
}

// skCheck asks GitHub which skills changed since they were installed.
func (a *App) skCheck() {
	if a.skCheckBusy {
		return
	}
	a.skCheckBusy = true
	a.invalidate()
	go func() {
		_, err := library.CheckSkills()
		run := func() {
			a.skCheckBusy = false
			if err != nil {
				a.libNotice = err.Error()
			}
		}
		if a.win == nil {
			run()
			return
		}
		a.win.Update(run)
		a.Reload()
	}()
}

// skRegroup traces the skills to their repositories again.
func (a *App) skRegroup() {
	if a.skRegroupBusy {
		return
	}
	a.skRegroupBusy = true
	a.invalidate()
	go func() {
		_, err := library.RegroupSkills()
		run := func() {
			a.skRegroupBusy = false
			if err != nil {
				a.libNotice = err.Error()
			}
		}
		if a.win == nil {
			run()
			return
		}
		a.win.Update(run)
		a.Reload()
	}()
}

// skProbe looks at a source for the skills in it.
func (a *App) skProbeSource(source string) {
	source = strings.TrimSpace(source)
	if source == "" || a.skInstalling {
		return
	}
	a.skInstalling, a.skInstallErr = true, ""
	a.skProbeSrc = source
	a.invalidate()
	go func() {
		p, err := library.ProbeSkills(source)
		run := func() {
			a.skInstalling = false
			if err != nil {
				a.skInstallErr = err.Error()
				a.skProbe = nil
				return
			}
			a.skProbe = p
			a.skProbePicks = map[string]bool{}
			// the few fresh ones are picked for the reader, as magpie's
			// own probe picks up to three
			var fresh []string
			for _, cd := range p.Candidates {
				if !cd.Have {
					fresh = append(fresh, cd.Path)
				}
			}
			if len(fresh) <= 3 {
				for _, pth := range fresh {
					a.skProbePicks[pth] = true
				}
			}
			a.skProbeAgents = map[string]bool{}
			for _, ag := range a.skillAgents() {
				a.skProbeAgents[ag.ID] = true
			}
			a.skProbeFilter = ""
		}
		if a.win == nil {
			run()
			return
		}
		a.win.Update(run)
	}()
}

// skInstall adds the picks from the source the probe looked at.
func (a *App) skInstall() {
	p := a.skProbe
	if p == nil {
		return
	}
	paths := make([]string, 0, len(a.skProbePicks))
	for pth, on := range a.skProbePicks {
		if on {
			paths = append(paths, pth)
		}
	}
	if len(paths) == 0 {
		return
	}
	agents := make([]string, 0, len(a.skProbeAgents))
	for id, on := range a.skProbeAgents {
		if on {
			agents = append(agents, id)
		}
	}
	source := a.skProbeSrc
	a.skWrite("skills/install", func() (*library.Result, error) {
		return library.InstallSkills(source, paths, agents)
	})
	if a.win == nil {
		a.skProbe, a.skProbePicks, a.skProbeSrc = nil, nil, ""
		return
	}
	a.skProbe, a.skProbePicks, a.skProbeSrc = nil, nil, ""
	a.invalidate()
}

// skBringIn moves a skill found in an agent into the library.
func (a *App) skBringIn(name string) {
	a.skWrite("skills/import:"+name, func() (*library.Result, error) {
		return library.ImportSkill(name)
	})
}

// skBringInAll brings every found skill in at once.
func (a *App) skBringInAll(names []string) {
	a.skWrite("skills/import-all", func() (*library.Result, error) {
		return library.ImportSkills(names)
	})
}

// skRemoveFound takes a found skill out of the agents that have it.
func (a *App) skRemoveFound(name string) {
	a.skWrite("skills/remove-found:"+name, func() (*library.Result, error) {
		return library.RemoveFoundSkill(name)
	})
}

// skSetHow sets the way the library gives its skills (agent "" for every
// agent, or one agent's own).
func (a *App) skSetHow(agent, how string) {
	a.skWrite("skills/how", func() (*library.Result, error) {
		return library.SetSkillHow(agent, how)
	})
}

// skAddNew adds the skills a check found in the library's repositories.
func (a *App) skAddNew(ids []string) {
	a.skWrite("skills/add-new", func() (*library.Result, error) {
		return library.AddNewSkills(ids)
	})
}

// skIgnoreNew sets the skills a check found aside, so it doesn't offer
// them again.
func (a *App) skIgnoreNew(ids []string) {
	a.skWrite("skills/ignore-new", func() (*library.Result, error) {
		return nil, library.IgnoreNewSkills(ids)
	})
}

// skSync makes the copies that differ from the library's skills again.
func (a *App) skSync() {
	a.skWrite("all/sync", func() (*library.Result, error) {
		return library.Sync()
	})
}

// skOpenReader reads a skill's SKILL.md for the reader to see.
func (a *App) skOpenReader(name string) {
	a.skReader, a.skReaderText, a.skReaderErr = name, "", ""
	a.skReaderPath = library.SkillPath(name)
	a.invalidate()
	go func() {
		text, err := library.SkillText(name)
		run := func() {
			if err != nil {
				a.skReaderErr = err.Error()
				return
			}
			a.skReaderText = text
		}
		if a.win == nil {
			run()
			return
		}
		a.win.Update(run)
	}()
}

// skMarket asks skills.sh for what matches the market's search.
func (a *App) skMarket(q string) {
	if a.skMarketBusy {
		return
	}
	a.skMarketBusy = true
	a.invalidate()
	go func() {
		items, err := library.MarketSkills(q)
		run := func() {
			a.skMarketBusy = false
			if err != nil {
				a.skMarketErr = err.Error()
				return
			}
			a.skMarketErr = ""
			a.skMarketItems = items
		}
		if a.win == nil {
			run()
			return
		}
		a.win.Update(run)
	}()
}

// skMarketKick asks the market once, when the page is first drawn.
func (a *App) skMarketKick() {
	if a.skMarketFetch || a.skMarketBusy {
		return
	}
	a.skMarketFetch = true
	a.skMarket("")
}

// skMarketKickAsk asks the market for what was just typed, once typing
// pauses — magpie's own 320ms. The timer is the app's, so a second letter
// replaces the ask rather than racing it.
func (a *App) skMarketKickAsk() {
	if a.skMarketTimer != nil {
		a.skMarketTimer.Stop()
	}
	// a query one letter long would match nearly everything, as magpie's
	// own page leaves it to the popular list
	if len(strings.TrimSpace(a.skMarketQuery)) == 1 {
		return
	}
	a.skMarketTimer = time.AfterFunc(320*time.Millisecond, func() {
		if a := Current(); a != nil && a.win != nil {
			a.win.Update(func() { a.skMarket(a.skMarketQuery) })
		}
	})
}

// skMarketNow asks at once, for a search that was submitted.
func (a *App) skMarketNow() {
	if a.skMarketTimer != nil {
		a.skMarketTimer.Stop()
		a.skMarketTimer = nil
	}
	a.skMarket(a.skMarketQuery)
}

// skInstallMarket adds a skill from skills.sh to the library.
func (a *App) skInstallMarket(source, id string) {
	agents := make([]string, 0)
	for _, ag := range a.skillAgents() {
		agents = append(agents, ag.ID)
	}
	a.skWrite("market/skill:"+id, func() (*library.Result, error) {
		return library.InstallMarketSkill(source, id, agents)
	})
}

// ---- the projects -------------------------------------------------------

// skProjectAgents are the agents that read a project's own skills folder:
// the ones with a ProjectSkills path, which is what a project's rows give
// their chips.
func (a *App) skProjectAgents() []LibAgentRow {
	out := make([]LibAgentRow, 0, len(a.data.Library.Agents))
	for _, ag := range a.data.Library.Agents {
		if ag.ProjectSkills != "" && !ag.Gone {
			out = append(out, ag)
		}
	}
	return out
}

// skProjectAgentsThatReadNone are the agents that take skills but read no
// project folder the library knows of, which the project's card says.
func (a *App) skProjectAgentsThatReadNone() []string {
	var out []string
	for _, ag := range a.skillAgents() {
		if ag.ProjectSkills == "" {
			out = append(out, ag.Name)
		}
	}
	sort.Strings(out)
	return out
}

// skProjectDirs are the folders the project's rows place a skill in, each
// once: several agents reading the same folder share one.
func (a *App) skProjectDirs() []string {
	seen := map[string]bool{}
	var out []string
	for _, ag := range a.skProjectAgents() {
		d := ag.ProjectSkills
		if !seen[d] {
			seen[d] = true
			out = append(out, d)
		}
	}
	sort.Strings(out)
	return out
}

// skProjectVia is the agent that already reads a folder the agent a chip
// stands for reads: the same folder given once serves them both, so a chip
// whose folder another agent has is lit by that one's click.
func skProjectVia(on []string, ag LibAgentRow, all []LibAgentRow) string {
	for _, id := range on {
		for _, other := range all {
			if other.ID == id && other.ProjectSkills == ag.ProjectSkills {
				return other.Name
			}
		}
	}
	return ""
}

// skAddProject adds a folder as a project, and opens it, as magpie's own
// page opens the one it just added.
func (a *App) skAddProject(dir string) {
	dir = strings.TrimSpace(dir)
	if dir == "" || a.skBusy != "" {
		return
	}
	a.skWrite("projects/add", func() (*library.Result, error) {
		return library.AddProject(dir)
	})
	if a.skOpenProjects == nil {
		a.skOpenProjects = map[string]bool{}
	}
	a.skOpenProjects[dir] = true
}

// skRemoveProject forgets a project, keeping what was placed in it when
// keep is set.
func (a *App) skRemoveProject(dir string, keep bool) {
	a.skWrite("projects/remove", func() (*library.Result, error) {
		return library.RemoveProject(dir, keep)
	})
}

// skProjectCopy sets whether the project gets copies of the skills rather
// than links to the library's.
func (a *App) skProjectCopy(dir string, copyFiles bool) {
	a.skWrite("projects/copy", func() (*library.Result, error) {
		return library.ProjectCopy(dir, copyFiles)
	})
}

// skProjectSkill places a skill in a project for exactly the agents given.
func (a *App) skProjectSkill(dir, name string, agents []string) {
	a.skWrite("projects/skill:"+dir+":"+name, func() (*library.Result, error) {
		return library.ProjectSkill(dir, name, agents)
	})
}

// skProjectToggle places a skill in a project for an agent, or takes it out.
func (a *App) skProjectToggle(p ProjectRow, s SkillRow, id string) {
	next := make([]string, 0, len(p.Skills[s.Name])+1)
	had := false
	for _, x := range p.Skills[s.Name] {
		if x == id {
			had = true
			continue
		}
		next = append(next, x)
	}
	if !had {
		next = append(next, id)
	}
	a.skProjectSkill(p.Dir, s.Name, next)
}

// skUseLibrarySkill settles an agent's own skill under the library's name
// by setting the agent's aside with the backups and giving it the
// library's, which is what the problems card's "Use the library's" does.
func (a *App) skUseLibrarySkill(name, agent string) {
	a.skWrite("skills/use-library:"+name+":"+agent, func() (*library.Result, error) {
		return library.UseLibrarySkill(name, agent)
	})
}

// skKeepOwnSkill settles it the other way: the agent keeps its own, and the
// library stops giving it this skill.
func (a *App) skKeepOwnSkill(name, agent string) {
	a.skWrite("skills/keep-own:"+name+":"+agent, func() (*library.Result, error) {
		return library.KeepAgentSkill(name, agent)
	})
}

// skChooseProject asks the system's own folder picker for a project's
// folder. A build with no window has no picker, so the field is left for
// the path to be typed in.
func (a *App) skChooseProject() {
	if a.win == nil {
		return
	}
	go func() {
		dirs, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
			Parent:      a.win,
			Title:       a.text.S.LibAddProject,
			ButtonLabel: a.text.S.LibChooseFolder,
			Directory:   true,
			Multiple:    false,
		})
		run := func() {
			if err != nil {
				a.skProjectErr = err.Error()
			} else if len(dirs) > 0 {
				a.skProjectDir, a.skProjectErr = dirs[0], ""
			}
			a.invalidate()
		}
		a.win.Update(run)
	}()
}
