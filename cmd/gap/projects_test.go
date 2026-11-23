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
	if got := registryRoots(t, home); len(got) != 1 || got[0] != repo {
		t.Fatalf("registry = %v, want [%s]", got, repo)
	}
	if strings.Count(readString(t, projectsPath(home)), repo) != 1 {
		t.Fatal("a second init duplicated the root")
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
	if got := registryRoots(t, home); len(got) != 1 || got[0] != fresh {
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
