//go:build nogui && echo

package shell

import (
	"encoding/json"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/yetone/magpie/internal/library"
	"github.com/yetone/magpie/internal/settings"
)

// TestSkillsPageRender draws the Skills page over a seeded library and
// writes it where a reviewer can look at it (ECHO_SHOT_OUT, or the temp
// dir), the way the probe writes its report.
//
// The seed is the test home's own: settings.Dir() is under the temporary
// home testenv makes, so no real library and no real agent config is read.
func TestSkillsPageRender(t *testing.T) {
	seedSkills(t)

	app := NewForTest()
	app.LoadNow()
	app.page, app.libTab = PageLibrary, LibTabSkills
	tt := ui.NewTester(app.ViewForTest(), 1180, 780)
	tt.Frame()

	shot := ui.Render(app.ViewForTest(), 1180, 780, 2)
	if shot == nil {
		t.Fatal("the page did not render")
	}
	out := os.Getenv("ECHO_SHOT_OUT")
	if out == "" {
		out = filepath.Join(os.TempDir(), "echo-skills.png")
	}
	f, err := os.Create(out)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, shot); err != nil {
		t.Fatal(err)
	}
	t.Logf("skills page: %s", out)
	if !tt.HasText(app.text.S.LibInTheLibraryLabel) {
		t.Errorf("the list's heading did not draw; texts were %v", tt.Texts())
	}
	for _, s := range app.data.Library.Skills {
		if !tt.HasText(s.Name) {
			t.Errorf("the skill %q did not draw; texts were %v", s.Name, tt.Texts())
		}
	}
	if len(app.data.Library.Skills) == 0 {
		t.Fatal("the seed wrote no skills: the page drew its empty state")
	}
}

// TestSkillsSheetsRender draws the page in its other states — picking, and
// with the reader and the by-agent sheet open — so a layout that only
// breaks in one of them fails here rather than in the app.
func TestSkillsSheetsRender(t *testing.T) {
	seedSkills(t)
	app := NewForTest()
	app.LoadNow()
	app.page, app.libTab = PageLibrary, LibTabSkills

	// picking, with two skills picked: the bar under the list, its count,
	// and the group it can make
	app.skPicking = true
	app.skPicked = map[string]bool{"git-commit": true, "pdf-tools": true}
	shotPicking := ui.Render(app.ViewForTest(), 1180, 900, 2)
	writeShot(t, "echo-skills-picking.png", shotPicking)
	tt := ui.NewTester(app.ViewForTest(), 1180, 780)
	tt.Frame()
	for _, want := range []string{app.text.S.LibNSelected, app.text.S.LibGroupEllipsis} {
		if !tt.HasText(strings.ReplaceAll(want, "{n}", "2")) {
			t.Errorf("picking: %q did not draw; texts were %v", want, tt.Texts())
		}
	}

	// the reader, over the page
	app.skPicking = false
	app.skReader = "pdf-tools"
	app.skReaderText = "# pdf-tools\n\nRead, split and fill PDFs.\n"
	app.skReaderPath = "/tmp/skills/pdf-tools"
	tt.Frame()
	if !tt.HasText("Read, split and fill PDFs.") {
		t.Errorf("the reader did not draw the SKILL.md; texts were %v", tt.Texts())
	}

	// the by-agent sheet
	app.skReader, app.skReaderText, app.skReaderPath = "", "", ""
	app.skByAgent = true
	tt.Frame()
	writeShot(t, "echo-skills-byagent.png", ui.Render(app.ViewForTest(), 1180, 780, 2))
	if !tt.HasText(app.text.S.LibSkillsByAgent) {
		t.Errorf("the by-agent sheet did not draw; texts were %v", tt.Texts())
	}
	if !tt.HasText(app.text.S.LibByAgentNoOthers) {
		t.Errorf("the by-agent sheet did not draw its note; texts were %v", tt.Texts())
	}

	// the how sheet
	app.skByAgent, app.skAskHow = false, true
	tt.Frame()
	if !tt.HasText(app.text.S.LibHowEachAgent) {
		t.Errorf("the how sheet did not draw; texts were %v", tt.Texts())
	}
	if !tt.HasText(app.text.S.LibLibrarysWay) {
		t.Errorf("the how sheet did not offer the library's way; texts were %v", tt.Texts())
	}

	// the remove-all ask
	app.skAskHow, app.skAskRemoveAll = false, true
	tt.Frame()
	if !tt.HasText(strings.ReplaceAll(app.text.S.LibRemoveAllAsk, "{n}", "4")) {
		t.Errorf("the remove-all ask did not draw; texts were %v", tt.Texts())
	}

	// what the install card found, with a box on each skill and the agents
	app.skAskRemoveAll = false
	app.skProbe = &library.Probe{
		Source: "acme/more-skills", Kind: "github",
		Candidates: []library.Candidate{
			{Path: "alpha", Name: "alpha", Description: "The first one."},
			{Path: "beta", Name: "beta", Description: "The second one."},
			{Path: "git-commit", Name: "git-commit", Have: true},
		},
	}
	app.skProbeSrc = "acme/more-skills"
	app.skProbePicks = map[string]bool{"alpha": true, "beta": true}
	app.skProbeAgents = map[string]bool{}
	for _, ag := range app.skillAgents() {
		app.skProbeAgents[ag.ID] = true
	}
	tt.Frame()
	// the count is composed from the language's own "{n} skills in {src}"
	wantCount := strings.NewReplacer("{n}", "3", "{src}", "github.com/acme/more-skills").
		Replace(app.text.S.LibNSkillsIn)
	if !tt.HasText(wantCount) {
		t.Errorf("the probe did not draw its count %q; texts were %v", wantCount, tt.Texts())
	}
	if !tt.HasText("alpha") || !tt.HasText("beta") {
		t.Errorf("the probe did not draw its candidates; texts were %v", tt.Texts())
	}
	if !tt.HasText(app.text.S.LibInTheLibrary) {
		t.Errorf("the probe did not mark the skill the library has; texts were %v", tt.Texts())
	}
	writeShot(t, "echo-skills-probe.png", ui.Render(app.ViewForTest(), 1180, 900, 2))
}

// TestSkillsProblemsAndProjects draws the two sections magpie's page has
// over and under its list: the card saying what the last change couldn't
// write, and the projects a folder's agents get some of the skills as.
func TestSkillsProblemsAndProjects(t *testing.T) {
	seedSkills(t)

	app := NewForTest()
	app.LoadNow()
	app.page, app.libTab = PageLibrary, LibTabSkills

	// the problems card: a file that wouldn't take a write, and an agent
	// with a skill of its own under the library's name
	app.libProblems = []library.Problem{
		{Agent: "codex", What: "skill:git-commit", Error: "the file is read-only"},
		{Agent: "claude-code", What: "skill:pdf-tools", Own: true, Error: "its own folder is there"},
	}
	app.LoadNow()
	if got := len(app.data.Library.Problems); got != 2 {
		t.Fatalf("the problems the page holds: want 2, got %d", got)
	}

	tt := ui.NewTester(app.ViewForTest(), 1180, 900)
	tt.Frame()
	if !tt.HasText(app.text.S.LibProblemsHead) {
		t.Errorf("the problems card did not draw its head; texts were %v", tt.Texts())
	}
	if !tt.HasText(app.text.S.LibProblemsTryAgain) {
		t.Errorf("the problems card did not offer to try again; texts were %v", tt.Texts())
	}
	// the skill is named, and the agent whose own folder clashes gets the
	// two ways of settling it
	if !tt.HasText(strings.ReplaceAll(app.text.S.LibProblemSkill, "{name}", "git-commit")) {
		t.Errorf("the problems card did not name the skill; texts were %v", tt.Texts())
	}
	if !tt.HasText(app.text.S.LibUseLibrarys) {
		t.Errorf("the problems card did not offer the library's skill; texts were %v", tt.Texts())
	}

	// the projects: the seeded one, closed, with the skills it has
	if !tt.HasText(app.text.S.LibInProjects) {
		t.Errorf("the projects section did not draw; texts were %v", tt.Texts())
	}
	if len(app.data.Library.Projects) != 1 {
		t.Fatalf("the page holds %d projects: want the seeded one", len(app.data.Library.Projects))
	}
	if !tt.HasText(app.data.Library.Projects[0].Name) {
		t.Errorf("the project's name did not draw; texts were %v", tt.Texts())
	}
	writeShot(t, "echo-skills-problems.png", ui.Render(app.ViewForTest(), 1180, 900, 2))

	// open, it lists every skill in the library with a chip for each agent
	// that reads a project's skills folder
	app.skOpenProjects[app.data.Library.Projects[0].Dir] = true
	tt.Frame()
	if !tt.HasText(app.text.S.LibCopyNotLink) {
		t.Errorf("the open project did not offer copies; texts were %v", tt.Texts())
	}
	for _, s := range app.data.Library.Skills {
		if !tt.HasText(s.Name) {
			t.Errorf("the open project did not list %q; texts were %v", s.Name, tt.Texts())
		}
	}
	writeShot(t, "echo-skills-project-open.png", ui.Render(app.ViewForTest(), 1180, 1400, 2))

	// the ask before it is forgotten
	app.skAskRemoveProject = app.data.Library.Projects[0].Dir
	tt.Frame()
	if !tt.HasText(app.text.S.LibRemoveProjectKeep) {
		t.Errorf("the remove ask did not offer to keep what was placed; texts were %v", tt.Texts())
	}
	app.skAskRemoveProject = ""

	// the card that adds one
	app.skAddingProject = true
	tt.Frame()
	if !tt.HasText(app.text.S.LibProjectFolder) {
		t.Errorf("the add-project card did not draw; texts were %v", tt.Texts())
	}
	if !tt.HasText(app.text.S.LibChooseFolder) {
		t.Errorf("the add-project card did not offer the folder picker; texts were %v", tt.Texts())
	}
}

// TestSkillsProjectWrites checks the writes behind the project's card call
// magpie's own domain code and nothing else: a skill placed for an agent is
// in the project, and taken out again by the same toggle.
func TestSkillsProjectWrites(t *testing.T) {
	seedSkills(t)

	app := NewForTest()
	app.LoadNow()
	app.page, app.libTab = PageLibrary, LibTabSkills

	// a folder of the test home's, which AddProject takes as a project
	dir := filepath.Join(settings.Dir(), "code", "written")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := library.AddProject(dir); err != nil {
		t.Fatalf("adding the project: %v", err)
	}
	app.LoadNow()

	var project *ProjectRow
	for i := range app.data.Library.Projects {
		if app.data.Library.Projects[i].Dir == dir {
			project = &app.data.Library.Projects[i]
		}
	}
	if project == nil {
		t.Fatal("the project the domain added isn't in the page's data")
	}

	// the agents that read a project's skills folder, which the chips are
	all := app.skProjectAgents()
	if len(all) == 0 {
		t.Skip("no agent on this machine reads a project skills folder")
	}
	agent := all[0].ID

	var skill SkillRow
	for _, s := range app.data.Library.Skills {
		if s.Name == "git-commit" {
			skill = s
		}
	}
	if skill.Name == "" {
		t.Fatal("the seed's git-commit isn't in the page's data")
	}

	// on, then off again through the same toggle
	if _, err := library.ProjectSkill(dir, skill.Name, []string{agent}); err != nil {
		t.Fatalf("placing %s for %s: %v", skill.Name, agent, err)
	}
	app.LoadNow()
	got := false
	for _, p := range app.data.Library.Projects {
		if p.Dir != dir {
			continue
		}
		got = containsStr(p.Skills[skill.Name], agent)
	}
	if !got {
		t.Errorf("%s isn't placed in the project for %s after the write", skill.Name, agent)
	}

	if _, err := library.ProjectSkill(dir, skill.Name, nil); err != nil {
		t.Fatalf("taking %s out of the project: %v", skill.Name, err)
	}
	app.LoadNow()
	for _, p := range app.data.Library.Projects {
		if p.Dir == dir && len(p.Skills[skill.Name]) > 0 {
			t.Errorf("%s is still placed in the project: %v", skill.Name, p.Skills[skill.Name])
		}
	}
}

// TestSkillGroupCountsWhileFiltering checks a group's heading says how many
// of its skills the filter shows, as magpie's own heading does.
func TestSkillGroupCountsWhileFiltering(t *testing.T) {
	seedSkills(t)

	app := NewForTest()
	app.LoadNow()
	app.page, app.libTab = PageLibrary, LibTabSkills

	app.libSearch = "commit"
	tt := ui.NewTester(app.ViewForTest(), 1180, 900)
	tt.Frame()

	want := strings.NewReplacer("{n}", "1", "{total}", "2").Replace(app.text.S.LibNOutOfSkills)
	if !tt.HasText(want) {
		t.Errorf("the filtered group's heading did not say %q; texts were %v", want, tt.Texts())
	}
}

// TestSkillsFoundInAgents draws the section under the list: the skills the
// agents have of their own, which only the full library read brings in
// (magpie's "In your agents"). The glance the page draws first leaves them
// out, so a page stuck on the glance would draw no section at all.
func TestSkillsFoundInAgents(t *testing.T) {
	seedSkills(t)

	app := NewForTest()
	app.LoadNow()

	// a skill one agent has of its own, which the library doesn't list
	folder := ""
	for _, ag := range app.data.Library.Agents {
		if ag.Skills != "" {
			folder = ag.Skills
			break
		}
	}
	if folder == "" {
		t.Skip("no agent on this machine has a skills folder")
	}
	dir := filepath.Join(folder, "found-here")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	md := "---\nname: found-here\ndescription: A skill the agents have of their own.\n---\n\n# found-here\n"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(md), 0o644); err != nil {
		t.Fatal(err)
	}

	app.LoadNow()
	app.page, app.libTab = PageLibrary, LibTabSkills

	tt := ui.NewTester(app.ViewForTest(), 1180, 900)
	tt.Frame()
	if !tt.HasText(app.text.S.LibInAgents) {
		t.Errorf("the found-in-agents section did not draw; texts were %v", tt.Texts())
	}
	if !tt.HasText("found-here") {
		t.Errorf("the skill the agents have of their own did not draw; texts were %v", tt.Texts())
	}
	if !tt.HasText(app.text.S.LibBringIn) {
		t.Errorf("the section did not offer to bring it in; texts were %v", tt.Texts())
	}
	writeShot(t, "echo-skills-found.png", ui.Render(app.ViewForTest(), 1180, 1100, 2))
}

// TestSkillChipsStackAndFan checks a row's agent chips are drawn as a hand of
// cards — overlapping — and spread out when the pointer comes to them, with
// the All chip ahead of them, as magpie's own rows do.
func TestSkillChipsStackAndFan(t *testing.T) {
	seedSkills(t)

	app := NewForTest()
	app.LoadNow()
	app.page, app.libTab = PageLibrary, LibTabSkills

	tt := ui.NewTester(app.ViewForTest(), 1180, 900)
	tt.Frame()

	agents := app.skillAgents()
	if len(agents) < 2 {
		t.Skip("fewer than two agents can take skills on this machine")
	}
	first := agents[0].Name
	last := agents[len(agents)-1].Name

	stackedFirst, ok := tt.Find(first)
	if !ok {
		t.Fatalf("the first agent's chip did not draw; texts were %v", tt.Texts())
	}
	stackedLast, ok := tt.Find(last)
	if !ok {
		t.Fatalf("the last agent's chip did not draw; texts were %v", tt.Texts())
	}

	// the All chip sits ahead of the hand
	all, ok := tt.Find(app.text.S.FilterAll)
	if !ok {
		t.Fatalf("the All chip did not draw; texts were %v", tt.Texts())
	}
	if all.X >= stackedFirst.X {
		t.Errorf("the All chip is at x=%v, not ahead of the hand at x=%v", all.X, stackedFirst.X)
	}
	// the heading's own button, past the chips: fanning must leave it alone
	rename, ok := tt.Find(app.text.S.LibRename)
	if !ok {
		t.Fatalf("the group's Rename button did not draw; texts were %v", tt.Texts())
	}

	// stacked: the hand is narrower than a chip per agent with a gap between
	stackedSpan := stackedLast.X - stackedFirst.X
	if spread := float32(len(agents)-1) * (stackedFirst.W + 2); stackedSpan >= spread {
		t.Errorf("the hand spans %v at rest, not stacked under %v", stackedSpan, spread)
	}
	writeShot(t, "echo-skills-chips-stacked.png", tt.Image())

	// the pointer on the hand fans it out
	tt.Move(stackedLast.X+stackedLast.W/2, stackedLast.Y+stackedLast.H/2)
	tt.Frame()
	fannedFirst, ok := tt.Find(first)
	if !ok {
		t.Fatalf("the first agent's chip did not draw while fanned; texts were %v", tt.Texts())
	}
	fannedLast, ok := tt.Find(last)
	if !ok {
		t.Fatalf("the last agent's chip did not draw while fanned; texts were %v", tt.Texts())
	}
	fannedSpan := fannedLast.X - fannedFirst.X
	if fannedSpan <= stackedSpan {
		t.Errorf("the hand did not fan out: it spans %v over the pointer, %v at rest", fannedSpan, stackedSpan)
	}
	if gap := fannedLast.X - fannedFirst.X - float32(len(agents)-2)*(fannedFirst.W+2); gap < fannedFirst.W {
		t.Errorf("the fanned hand leaves the chips overlapping: %v between the first and last", gap)
	}
	// the room was reserved, so the heading's own pieces stayed where they were
	if fannedAll, ok := tt.Find(app.text.S.FilterAll); !ok {
		t.Errorf("the All chip went out of the frame while fanned; texts were %v", tt.Texts())
	} else if fannedAll.X != all.X {
		t.Errorf("the All chip moved from x=%v to x=%v as the hand fanned", all.X, fannedAll.X)
	}
	if moved, ok := tt.Find(app.text.S.LibRename); !ok {
		t.Errorf("the Rename button went out of the frame while fanned; texts were %v", tt.Texts())
	} else if moved.X != rename.X {
		t.Errorf("the Rename button moved from x=%v to x=%v as the hand fanned", rename.X, moved.X)
	}
	writeShot(t, "echo-skills-chips-fanned.png", tt.Image())
}

// writeShot writes a render where a reviewer can look at it.
func writeShot(t *testing.T, name string, img *image.RGBA) {
	t.Helper()
	if img == nil {
		t.Errorf("%s: nothing rendered", name)
		return
	}
	out := filepath.Join(os.TempDir(), name)
	f, err := os.Create(out)
	if err != nil {
		t.Errorf("%s: %v", name, err)
		return
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Errorf("%s: %v", name, err)
		return
	}
	t.Logf("%s: %s", name, out)
}

// seedSkills writes a few skills into the test home's library, in the
// library.json the page reads and in the folders they sit in, so the page
// has rows to draw.
func seedSkills(t *testing.T) {
	t.Helper()
	root := filepath.Join(settings.Dir(), "library", "skills")
	names := []struct{ name, desc string }{
		{"git-commit", "Write a commit message that says what changed and why."},
		{"pdf-tools", "Read, split and fill PDFs from the command line."},
		{"docker-build", "Build small images and keep the layers that matter."},
		{"test-first", "Write the failing test before the code that passes it."},
	}
	skills := make([]map[string]any, 0, len(names))
	for _, n := range names {
		dir := filepath.Join(root, n.name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		md := "---\nname: " + n.name + "\ndescription: " + n.desc + "\n---\n\n# " + n.name + "\n\n" + n.desc + "\n"
		if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(md), 0o644); err != nil {
			t.Fatal(err)
		}
		// one of them from a GitHub repository, so the page has a group
		// with a repository's name over it and an update button on its row
		sk := map[string]any{"name": n.name, "agents": []string{}}
		if n.name == "git-commit" || n.name == "test-first" {
			sk["source"] = map[string]any{
				"kind": "github", "repo": "acme/skills", "ref": "main", "path": n.name,
			}
		}
		skills = append(skills, sk)
	}
	// a project whose folder is there, so the card draws as the app's does
	projectDir := filepath.Join(settings.Dir(), "code", "app")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}

	lib := map[string]any{
		"instructions": map[string]any{"agents": []string{}},
		"mcp":          []any{},
		"skills":       skills,
		"skillGroups": []any{
			map[string]any{"name": "Everyday", "skills": []string{"pdf-tools"}},
		},
		"projects": []any{
			map[string]any{
				"dir":    projectDir,
				"skills": map[string][]string{"git-commit": {"claude-code"}},
			},
		},
	}
	b, err := json.MarshalIndent(lib, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(settings.Dir(), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(settings.Dir(), "library.json"), b, 0o644); err != nil {
		t.Fatal(err)
	}
}
