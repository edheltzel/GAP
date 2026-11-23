package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
)

func registryRoots(t *testing.T, home string) []string {
	t.Helper()
	list, err := readProjects(home)
	if err != nil {
		t.Fatal(err)
	}
	var roots []string
	for _, p := range list {
		roots = append(roots, p.Root)
	}
	return roots
}

func TestProjectsRegistryInit(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PATH", "/usr/bin:/bin")
	repo := t.TempDir()
	for range 2 {
		if code := run([]string{"init", repo}); code != 0 {
			t.Fatalf("init: %d", code)
		}
	}
	real := canonicalRoot(repo)
	if got := registryRoots(t, home); len(got) != 1 || got[0] != real {
		t.Fatalf("registry = %v, want [%s]", got, real)
	}
	if strings.Count(readString(t, projectsPath(home)), real) != 1 {
		t.Fatal("a second init duplicated the root")
	}
}

func TestProjectsRegistryFailedInitRecordsNothing(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PATH", "/usr/bin:/bin")
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, "AGENTS.md"), 0o755); err != nil {
		t.Fatal(err)
	}
	var code int
	captureStd(t, func() { code = run([]string{"init", repo}) })
	if code == 0 {
		t.Fatal("init succeeded with a directory named AGENTS.md")
	}
	if _, err := os.Stat(projectsPath(home)); !os.IsNotExist(err) {
		t.Fatalf("failed init recorded the project: %v", registryRoots(t, home))
	}
}

func TestProjectsRegistrySymlinkedRoots(t *testing.T) {
	home := t.TempDir()
	real := filepath.Join(t.TempDir(), "repo")
	if err := os.Mkdir(real, 0o755); err != nil {
		t.Fatal(err)
	}
	links := t.TempDir()
	link, homeLink := filepath.Join(links, "repo"), filepath.Join(links, "home")
	for target, name := range map[string]string{real: link, home: homeLink} {
		if err := os.Symlink(target, name); err != nil {
			t.Fatal(err)
		}
	}
	for _, root := range []string{link, real, homeLink} {
		if err := recordProject(home, root); err != nil {
			t.Fatal(err)
		}
	}
	if got := registryRoots(t, home); len(got) != 1 || got[0] != canonicalRoot(real) {
		t.Fatalf("registry = %v, want only %s", got, canonicalRoot(real))
	}
	if err := forgetProject(home, link); err != nil {
		t.Fatal(err)
	}
	if got := registryRoots(t, home); len(got) != 0 {
		t.Fatalf("forget through the link left %v", got)
	}
}

func TestProjectsRegistrySyncRecordsOnlyAfterProceeding(t *testing.T) {
	for _, tc := range []struct {
		name   string
		answer string
		want   bool
	}{{"quit", "q\n", false}, {"all", "\n", true}} {
		t.Run(tc.name, func(t *testing.T) {
			home := selectionFixture(t)
			project, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			write(t, projectConfigFile(project), "profile = \"project\"\nslug = \"p\"\n")
			answers(t, true, tc.answer)
			captureStd(t, func() { run([]string{"sync"}) })
			if exists(projectsPath(home)) != tc.want {
				t.Fatalf("registry written = %v, want %v", exists(projectsPath(home)), tc.want)
			}
			if tc.want {
				if got := registryRoots(t, home); len(got) != 1 || got[0] != canonicalRoot(project) {
					t.Fatalf("registry = %v", got)
				}
			}
		})
	}
}

type astroEnv struct {
	home, distroRoot, project string
	commit                    string
}

func newAstroEnv(t *testing.T) astroEnv {
	t.Helper()
	home, distroRoot, project, remote := t.TempDir(), t.TempDir(), t.TempDir(), t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_STATE_HOME", "")
	commit, tree := gitFixture(t, remote, "project skill\n")
	write(t, filepath.Join(distroRoot, "gap.toml"), "schema = \"2\"\n[components.tools]\nrepo = \""+remote+"\"\nref = \"master\"\nnames = [\"do-todo\"]\n[components.tools.skills.do-todo]\nload = \"project\"\n[components.tools.skills.do-todo.when]\nfiles = [\"astro.config.*\"]\n")
	if err := writeLock(distroRoot, lockFile{Components: map[string]lockEntryM1{"tools": {Ref: "master", Commit: commit, Tree: tree}}}); err != nil {
		t.Fatal(err)
	}
	readyEnvReviewDistro(t, distroRoot)
	if _, err := git(project, "init", "-q", "-b", "master"); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(project, "astro.config.mjs"), "export default {}\n")
	if err := recordProject(home, t.TempDir()); err != nil {
		t.Fatal(err)
	}
	t.Chdir(project)
	return astroEnv{home: home, distroRoot: distroRoot, project: project, commit: commit}
}

func (a astroEnv) sync(t *testing.T) int {
	t.Helper()
	var code int
	captureStd(t, func() { code = run([]string{"sync"}) })
	return code
}

func TestProjectsRegistrySyncRecordsProjectSkillInstall(t *testing.T) {
	a := newAstroEnv(t)
	if code := a.sync(t); code != 0 {
		t.Fatalf("sync: %d", code)
	}
	if _, err := os.Stat(filepath.Join(a.project, ".agents", "skills", "do-todo", "SKILL.md")); err != nil {
		t.Fatalf("project skill not installed: %v", err)
	}
	if got := registryRoots(t, a.home); !slices.Contains(got, canonicalRoot(a.project)) || len(got) != 2 {
		t.Fatalf("registry = %v, want it to include %s", got, canonicalRoot(a.project))
	}
}

func TestProjectsRegistrySyncSkipsKeptProjectDrift(t *testing.T) {
	a := newAstroEnv(t)
	edited := filepath.Join(a.project, ".agents", "skills", "do-todo", "SKILL.md")
	write(t, edited, "mine\n")
	a.sync(t)
	if readString(t, edited) != "mine\n" {
		t.Fatal("sync replaced an edited project copy without a terminal")
	}
	if got := registryRoots(t, a.home); slices.Contains(got, canonicalRoot(a.project)) {
		t.Fatalf("registry %v includes a repo whose project copy was kept edited", got)
	}
	if err := os.RemoveAll(filepath.Dir(edited)); err != nil {
		t.Fatal(err)
	}
	if code := a.sync(t); code != 0 {
		t.Fatalf("second sync: %d", code)
	}
	if got := registryRoots(t, a.home); !slices.Contains(got, canonicalRoot(a.project)) {
		t.Fatalf("registry = %v after the install", got)
	}
}

func TestProjectInventoryDistroSkillWithoutStore(t *testing.T) {
	a := newAstroEnv(t)
	if code := a.sync(t); code != 0 {
		t.Fatalf("sync: %d", code)
	}
	if err := os.RemoveAll(storePath(a.home, "tools", a.commit)); err != nil {
		t.Fatal(err)
	}
	v := projectInventory(a.home, a.distroRoot, projectRecord{Root: canonicalRoot(a.project)}, nil)
	if v.Err != nil || len(v.Items) != 1 || v.Items[0].ID != "tools" || v.Items[0].Store != storeMissing || !v.unchecked() {
		t.Fatalf("view = %+v", v)
	}
}

func TestProjectsRegistryCoalescesAliases(t *testing.T) {
	home := t.TempDir()
	real := filepath.Join(t.TempDir(), "repo")
	if err := os.Mkdir(real, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "repo")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	gone := filepath.Join(t.TempDir(), "gone")
	if err := writeProjects(home, []projectRecord{{Root: link, Added: "2020-01-01"}, {Root: real, Added: "2021-02-02"}, {Root: gone, Added: "2019-03-03"}}); err != nil {
		t.Fatal(err)
	}
	if err := recordProject(home, real); err != nil {
		t.Fatal(err)
	}
	list, err := readProjects(home)
	if err != nil {
		t.Fatal(err)
	}
	want := []projectRecord{{Root: canonicalRoot(real), Added: "2020-01-01"}, {Root: gone, Added: "2019-03-03"}}
	slices.SortFunc(want, func(a, b projectRecord) int { return strings.Compare(a.Root, b.Root) })
	if !slices.Equal(list, want) {
		t.Fatalf("registry = %+v, want %+v", list, want)
	}
}

func TestProjectsRegistryRecallFailureRecordsNothing(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	recallStub(t, "exit 1\n")
	repo := t.TempDir()
	var code int
	captureStd(t, func() { code = run([]string{"init", repo}) })
	if code == 0 {
		t.Fatal("init succeeded while Recall failed")
	}
	if exists(projectsPath(home)) {
		t.Fatalf("failed init recorded the project: %v", registryRoots(t, home))
	}
}

func TestProjectsRegistryImportsLegacyRoots(t *testing.T) {
	home := t.TempDir()
	legacy := filepath.Join(t.TempDir(), "old")
	if err := writeState(home, stateFile{Components: map[string]stateEntry{"util": {ProjectSkills: map[string][]string{legacy: {"s"}}}}}); err != nil {
		t.Fatal(err)
	}
	if got := registryRoots(t, home); len(got) != 1 || got[0] != legacy {
		t.Fatalf("legacy import = %v", got)
	}
	fresh := t.TempDir()
	if err := recordProject(home, fresh); err != nil {
		t.Fatal(err)
	}
	if got := registryRoots(t, home); len(got) != 2 {
		t.Fatalf("after record = %v", got)
	}
	if err := forgetProject(home, legacy); err != nil {
		t.Fatal(err)
	}
	if got := registryRoots(t, home); len(got) != 1 || got[0] != canonicalRoot(fresh) {
		t.Fatalf("after forget = %v; a forgotten legacy root must not come back", got)
	}
}

func TestProjectsRegistryIgnoresHomeAndRelative(t *testing.T) {
	home := t.TempDir()
	if err := recordProject(home, home); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(projectsPath(home)); !os.IsNotExist(err) {
		t.Fatal("home was recorded as a project")
	}
	if err := recordProject(home, "rel"); err == nil {
		t.Fatal("relative root accepted")
	}
}

func TestProjectsRegistryAddProject(t *testing.T) {
	e := newUtilityEnv(t, "schema = \"1\"\nname = \"util\"\nskills = [\"util-skill\"]\n")
	quietStdout(t)
	if err := runAdd(addProjectArgs(e.remote, "--id", "util")); err != nil {
		t.Fatal(err)
	}
	root, _ := filepath.EvalSymlinks(e.project)
	got := registryRoots(t, e.home)
	if len(got) != 1 {
		t.Fatalf("registry = %v", got)
	}
	if r, _ := filepath.EvalSymlinks(got[0]); r != root {
		t.Fatalf("registry = %v, want %s", got, root)
	}
}

func TestProjectsRegistryConcurrentRecords(t *testing.T) {
	home := t.TempDir()
	base := t.TempDir()
	var want []string
	for i := range 32 {
		want = append(want, filepath.Join(base, fmt.Sprintf("repo%02d", i)))
	}
	errs := make([]error, len(want))
	var wg sync.WaitGroup
	for i, root := range want {
		wg.Go(func() { errs[i] = recordProject(home, root) })
	}
	wg.Wait()
	for _, err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if got := registryRoots(t, home); !slices.Equal(got, want) {
		t.Fatalf("registry kept %d of %d roots: %v", len(got), len(want), got)
	}
}

func TestProjectsRegistryCrossProcessRecords(t *testing.T) {
	home := t.TempDir()
	base := t.TempDir()
	var want []string
	for i := range 8 {
		want = append(want, filepath.Join(base, fmt.Sprintf("proc%d", i)))
	}
	outs := make([][]byte, len(want))
	errs := make([]error, len(want))
	var wg sync.WaitGroup
	for i, root := range want {
		wg.Go(func() {
			cmd := exec.Command(os.Args[0], "-test.run=^TestProjectsRegistryHelperProcess$", "-test.count=1")
			cmd.Env = append(os.Environ(), "GAP_REGISTRY_HELPER=1", "HOME="+home, "GAP_REGISTRY_HOME="+home, "GAP_REGISTRY_ROOT="+root)
			outs[i], errs[i] = cmd.CombinedOutput()
		})
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Fatalf("helper %d: %v\n%s", i, err, outs[i])
		}
	}
	if got := registryRoots(t, home); !slices.Equal(got, want) {
		t.Fatalf("registry kept %d of %d roots: %v", len(got), len(want), got)
	}
}

func TestProjectsRegistryHelperProcess(t *testing.T) {
	if os.Getenv("GAP_REGISTRY_HELPER") != "1" {
		return
	}
	if err := recordProject(os.Getenv("GAP_REGISTRY_HOME"), os.Getenv("GAP_REGISTRY_ROOT")); err != nil {
		t.Fatal(err)
	}
}
