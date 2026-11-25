package main

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

func reviewModel(t *testing.T, items []itemRow) (*tuiModel, *[][]string) {
	t.Helper()
	project := t.TempDir()
	m := newTUIModel(t.TempDir(), t.TempDir(), project, darkPalette)
	m.runner = &fakeRunner{}
	var handed [][]string
	m.handOff = func(_ string, args []string) tea.Cmd {
		handed = append(handed, args)
		return nil
	}
	m.Update(tea.WindowSizeMsg{Width: 110, Height: 34})
	m.Update(loadedMsg{items: items})
	return m, &handed
}

func TestTUIUpdateAllListsWhatUpdateMoves(t *testing.T) {
	m, handed := reviewModel(t, []itemRow{
		{ID: "held", Source: "user", Held: true},
		{ID: "off", Source: "user", Disabled: true},
		{ID: "plain", Source: "distro"},
	})
	typeKeys(m, "U")
	if !slices.Equal(m.confirmIDs, []string{"off", "plain"}) {
		t.Fatalf("U lists %q, want the unheld items, disabled included", m.confirmIDs)
	}
	typeKeys(m, "enter")
	if len(*handed) != 1 || strings.Join((*handed)[0], " ") != "update" {
		t.Fatalf("handed %q", *handed)
	}
}

func TestTUIVersionEditKeepsWorkingRefOnFailure(t *testing.T) {
	m, handed := reviewModel(t, []itemRow{{ID: "-x", Source: "user", Scope: "global", Ref: "main"}})
	cfg := userConfigFile(m.home)
	write(t, cfg, "[components.-x]\nrepo = \"github.com/o/r\"\nref = \"main\"\n")
	m.projectRoot = ""
	typeKeys(m, "e", "enter", "enter")
	for range "main" {
		typeKeys(m, "backspace")
	}
	typeKeys(m, "nope", "enter", "enter")
	if len(*handed) != 1 || strings.Join((*handed)[0], " ") != "update -- -x" {
		t.Fatalf("handed %q, want update -- -x", *handed)
	}
	if got := readString(t, cfg); !strings.Contains(got, "ref = \"main\"") {
		t.Fatalf("a ref the pin never moved to was kept:\n%s", got)
	}
	if m.note != "-x stays on main: the pin did not move to nope." {
		t.Fatalf("note %q", m.note)
	}
	write(t, filepath.Join(userConfigDir(m.home), "gap.lock"), "")
	if err := writeLock(userConfigDir(m.home), lockFile{Components: map[string]lockEntryM1{"-x": {Ref: "v2", Commit: strings.Repeat("a", 40), Tree: strings.Repeat("b", 40)}}}); err != nil {
		t.Fatal(err)
	}
	*handed = nil
	m.mode = modeBrowse
	typeKeys(m, "e", "enter", "enter")
	for range "main" {
		typeKeys(m, "backspace")
	}
	typeKeys(m, "v2", "enter", "enter")
	if got := readString(t, cfg); !strings.Contains(got, "ref = \"v2\"") {
		lk, err := readLock(userConfigDir(m.home))
		t.Fatalf("a ref the pin moved to was dropped (note %q, lock %+v %v, handed %q):\n%s", m.note, lk, err, *handed, got)
	}
}

func TestTUIDistroEditHasNoVersion(t *testing.T) {
	m, _ := reviewModel(t, []itemRow{{ID: "tools", Source: "distro", Scope: "global", Ref: "main"}})
	typeKeys(m, "e")
	if strings.Contains(ansi.Strip(m.render()), "Version") {
		t.Fatal("distro components must change version through gap pin, not the form")
	}
}

func TestTUIFormFitsSmallTerminal(t *testing.T) {
	m, _ := reviewModel(t, []itemRow{{ID: "echo", Source: "user", Scope: "global", Ref: "main"}})
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	typeKeys(m, "e")
	if n := len(strings.Split(m.render(), "\n")); n > 24 {
		t.Fatalf("edit form is %d lines in 80x24:\n%s", n, ansi.Strip(m.render()))
	}
}

func TestProjectConfigNeedsExistingRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "gone")
	if err := ensureProjectConfig(root); err == nil || !strings.Contains(err.Error(), "no longer exists") {
		t.Fatalf("err = %v", err)
	}
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatal("a missing project folder was recreated")
	}
}

func TestTUISearchDropsStaleHitsAndTakesPaste(t *testing.T) {
	m, handed := reviewModel(t, nil)
	var queries []string
	m.findSkills = func(_ context.Context, q string, _ int) ([]skillHit, error) {
		queries = append(queries, q)
		return []skillHit{{ID: "a/b/" + q, Source: "a/b", Skill: q}}, nil
	}
	typeKeys(m, "/")
	drive(m, tea.PasteMsg{Content: "pdf"})
	drive(m, searchTickMsg{seq: m.search.seq})
	if !slices.Equal(queries, []string{"pdf"}) || len(m.search.hits) != 1 {
		t.Fatalf("paste did not search: queries %q hits %v", queries, m.search.hits)
	}
	stale := m.search.seq
	typeKeys(m, "backspace", "backspace", "backspace")
	if len(m.search.hits) != 0 {
		t.Fatal("clearing the query kept the old hits")
	}
	typeKeys(m, "enter")
	if len(*handed) != 0 {
		t.Fatalf("enter installed a hit for an old query: %q", *handed)
	}
	m.Update(searchDoneMsg{seq: stale, query: "pdf", hits: []skillHit{{Source: "a/b", Skill: "pdf"}}})
	if len(m.search.hits) != 0 {
		t.Fatal("a late answer for an old query came back")
	}
}

func TestTUIInstalledBadgeNeedsFiles(t *testing.T) {
	m, _ := reviewModel(t, []itemRow{{ID: "b", Source: "user", Repo: "github.com/a/b", Names: []string{"pdf"}}})
	hit := skillHit{ID: "a/b/pdf", Source: "a/b", Skill: "pdf"}
	if got := m.installedHit(hit); got != "" {
		t.Fatalf("badge %q with no skill files", got)
	}
	write(t, filepath.Join(m.home, ".agents", "skills", "pdf", "SKILL.md"), "x")
	if got := m.installedHit(hit); got != "global" {
		t.Fatalf("badge %q, want global", got)
	}
	if got := m.installedHit(skillHit{Source: "other/repo", Skill: "pdf"}); got != "" {
		t.Fatalf("badge %q for a same-named skill from another repo", got)
	}
}

func TestSearchInstallUsesGitHubURL(t *testing.T) {
	t.Setenv("GH_HOST", "git.example.com")
	m, handed := reviewModel(t, nil)
	m.projectRoot = ""
	m.installHit(skillHit{Source: "vercel-labs/agent-skills", Skill: "react"})
	if len(*handed) == 0 {
		cmd := m.installHit(skillHit{Source: "vercel-labs/agent-skills", Skill: "react"})
		_ = cmd
	}
	src, err := parseSkillSource("https://github.com/vercel-labs/agent-skills", t.TempDir())
	if err != nil || !strings.HasPrefix(src.URL, "https://github.com/") {
		t.Fatalf("source %+v err %v", src, err)
	}
	if len(*handed) == 0 || (*handed)[0][1] != "https://github.com/vercel-labs/agent-skills" {
		t.Fatalf("handed %q", *handed)
	}
}

func TestComponentTOMLKeepsColorAndExplicitPin(t *testing.T) {
	text := "[components.x]\nrepo = \"github.com/o/r\"\nref = \"main\"\npin = false\ncolor = \"#FF79C6\"\n"
	cc, ok, err := readComponentConfig(text, "x")
	if err != nil || !ok {
		t.Fatal(ok, err)
	}
	out, err := upsertComponent("", "x", cc)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"pin = false", "color = \"#FF79C6\""} {
		if !strings.Contains(out, want) {
			t.Fatalf("rewrite lost %q:\n%s", want, out)
		}
	}
}

func TestReAddKeepsColor(t *testing.T) {
	e := newUtilityEnv(t, "schema = \"1\"\nname = \"util\"\nskills = [\"util-skill\"]\n")
	quietStdout(t)
	write(t, userConfigFile(e.home), "[trust]\nrepos = [\""+e.remote+"\"]\n")
	if err := runAdd(addGlobalArgs(e.remote, "--id", "util")); err != nil {
		t.Fatal(err)
	}
	if err := setComponentFields(userConfigFile(e.home), "util", []configField{{"color", "#FF79C6"}, {"pin", false}}); err != nil {
		t.Fatal(err)
	}
	if err := runAdd(addGlobalArgs(e.remote, "--id", "util")); err != nil {
		t.Fatal(err)
	}
	got := readString(t, userConfigFile(e.home))
	for _, want := range []string{"color = \"#FF79C6\"", "pin = false"} {
		if !strings.Contains(got, want) {
			t.Fatalf("re-add lost %q:\n%s", want, got)
		}
	}
}

func TestOverrideViewKeepsCallerRefs(t *testing.T) {
	home, project := t.TempDir(), t.TempDir()
	write(t, filepath.Join(project, ".gap", "gap.toml"), "[components.tools]\nref = \"dev\"\npin = true\n")
	d := distro{Components: map[string]component{"tools": {ID: "tools", Repo: "github.com/o/tools", Ref: "main"}}}
	view, err := overrideView(d, home, t.TempDir(), project)
	if err != nil {
		t.Fatal(err)
	}
	if !view.Components["tools"].Pin || view.Components["tools"].Ref != "dev" {
		t.Fatalf("view = %+v", view.Components["tools"])
	}
	if d.Components["tools"].Ref != "main" || d.Components["tools"].Pin {
		t.Fatalf("overrideView changed the caller's component: %+v", d.Components["tools"])
	}
}

func TestVersionEditRestoresWhenGapCannotStart(t *testing.T) {
	m, _ := reviewModel(t, []itemRow{{ID: "x", Source: "user", Scope: "global", Ref: "main"}})
	m.handOff = nil
	old := tuiExecutable
	tuiExecutable = func() (string, error) { return "", os.ErrNotExist }
	t.Cleanup(func() { tuiExecutable = old })
	cfg := userConfigFile(m.home)
	write(t, cfg, "[components.x]\nrepo = \"github.com/o/r\"\nref = \"main\"\n")
	m.moveVersion("x", "main", "nope")
	if got := readString(t, cfg); !strings.Contains(got, "ref = \"main\"") {
		t.Fatalf("ref not restored when gap could not start:\n%s", got)
	}
	if !strings.Contains(m.note, "stays on main") {
		t.Fatalf("note %q", m.note)
	}
}

func TestHandoffRestoresBeforeEnterPrompt(t *testing.T) {
	var order []string
	h := &handoff{cmd: exec.Command("false"), in: strings.NewReader("\n"), out: io.Discard, err: io.Discard,
		after: func(err error) string { order = append(order, "after"); return "n" }}
	h.cmd.Stdout = nil
	if err := h.Run(); err == nil {
		t.Fatal("false exited 0")
	}
	if h.note != "n" || len(order) != 1 {
		t.Fatalf("after ran %d times, note %q", len(order), h.note)
	}
}

func TestEditProjectEnabledOverGlobalDisabled(t *testing.T) {
	m, _ := reviewModel(t, []itemRow{{ID: "x", Source: "user", Scope: "global", Ref: "main", OffGlobal: true}})
	typeKeys(m, "e")
	if s := ansi.Strip(m.render()); !strings.Contains(s, "This project") {
		t.Fatalf("form does not start on This project:\n%s", s)
	}
	typeKeys(m, "enter", "right", "enter", "enter", "enter", "enter")
	got := readString(t, projectConfigFile(m.projectRoot))
	if !strings.Contains(got, "enabled = false") {
		t.Fatalf("turning it off in the project wrote:\n%s", got)
	}
}

func TestSearchReopenIgnoresOldAnswers(t *testing.T) {
	m, handed := reviewModel(t, nil)
	m.findSkills = func(context.Context, string, int) ([]skillHit, error) { return nil, nil }
	typeKeys(m, "/", "a")
	old := m.search.seq
	typeKeys(m, "esc", "/", "b")
	m.Update(searchDoneMsg{seq: old, query: "a", hits: []skillHit{{ID: "o/r/a", Source: "o/r", Skill: "a"}}})
	typeKeys(m, "enter")
	if len(m.search.hits) != 0 || len(*handed) != 0 {
		t.Fatalf("an answer from a closed search was used: hits %v handed %q", m.search.hits, *handed)
	}
}

func TestInstalledBadgeMatchesInventoryRepo(t *testing.T) {
	for _, repo := range []string{"o/r", "github.com/o/r", "https://github.com/O/R.git"} {
		m, _ := reviewModel(t, []itemRow{{ID: "r", Source: "user", Repo: repo, Names: []string{"pdf"}}})
		write(t, filepath.Join(m.home, ".agents", "skills", "pdf", "SKILL.md"), "x")
		if got := m.installedHit(skillHit{Source: "o/r", Skill: "pdf"}); got != "global" {
			t.Fatalf("repo %q: badge %q", repo, got)
		}
	}
}
