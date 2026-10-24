package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestDoctorRunsComponentAndReportsHarnesses(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PATH", "/usr/bin:/bin")
	repo, _ := echoFixture(t, home, "#!/bin/sh\nexit 0\n")
	sourceRootFor = repo
	t.Cleanup(func() { sourceRootFor = "" })
	code := 0
	out, _ := captureStd(t, func() { code = run([]string{"doctor"}) })
	if code != 0 || !strings.Contains(out, "echo: scripts/echo-doctor") || !strings.Contains(out, "fixture-ok") {
		t.Fatalf("doctor: %d\n%s", code, out)
	}
	if !strings.Contains(out, "claude: absent;") {
		t.Fatalf("harness report: %s", out)
	}
}

func TestDoctorFailsWhenComponentFails(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	repo, commit := echoFixture(t, home, "#!/bin/sh\nexit 0\n")
	store := storePath(home, "echo", commit)
	write(t, filepath.Join(store, "scripts", "echo-doctor"), "#!/bin/sh\nexit 3\n")
	tree, err := treeHash(store)
	if err != nil {
		t.Fatal(err)
	}
	if err := writeLock(repo, lockFile{Components: map[string]lockEntryM1{"echo": {Ref: "master", Commit: commit, Tree: tree}}}); err != nil {
		t.Fatal(err)
	}
	sourceRootFor = repo
	t.Cleanup(func() { sourceRootFor = "" })
	if code := run([]string{"doctor"}); code != 1 {
		t.Fatalf("failing doctor: %d", code)
	}
}

// echoFixture is a pinned echo store whose scripts/install.sh is the given script.
// The store is a real git worktree so ensureStore accepts it offline.
func echoFixture(t *testing.T, home, script string) (repo, commit string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	remote := filepath.Join(t.TempDir(), "remote")
	write(t, filepath.Join(remote, "scripts", "install.sh"), script)
	write(t, filepath.Join(remote, "scripts", "echo-doctor"), "#!/bin/sh\necho fixture-ok\nexit 0\n")
	if err := os.Chmod(filepath.Join(remote, "scripts", "install.sh"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Join(remote, "scripts", "echo-doctor"), 0o755); err != nil {
		t.Fatal(err)
	}
	commit, tree := gitFixture(t, remote, "unused\n")
	repo = t.TempDir()
	write(t, filepath.Join(repo, "global", "AGENTS.md"), "policy\n")
	write(t, filepath.Join(repo, "gap.toml"), "schema = \"2\"\n[components.echo]\nrepo = \""+remote+"\"\nref = \"master\"\nnames = []\ndoctor = \"scripts/echo-doctor\"\n")
	write(t, filepath.Join(repo, "gap.lock"), "schema = \"2\"\n\n[components.echo]\nref = \"master\"\ncommit = \""+commit+"\"\ntree = \""+tree+"\"\n")
	if _, err := ensureStore(home, component{ID: "echo", Repo: remote, Ref: "master"}, lockEntryM1{Ref: "master", Commit: commit, Tree: tree}); err != nil {
		t.Fatal(err)
	}
	return repo, commit
}

func echoAdapters(t *testing.T, home string) (commit string, adapters []string, present bool) {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(home, ".gap", "state.json"))
	if os.IsNotExist(err) {
		return "", nil, false
	}
	if err != nil {
		t.Fatal(err)
	}
	var state stateFile
	if err := json.Unmarshal(b, &state); err != nil {
		t.Fatal(err)
	}
	entry, ok := state.Components["echo"]
	if !ok {
		return "", nil, false
	}
	return entry.Commit, entry.Adapters, true
}

func TestDoctorRecordsAdapters(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	repo, commit := echoFixture(t, home, "#!/bin/sh\nif [ \"$1\" = --list-installed ]; then printf 'codex\\nclaude\\n'; exit 0; fi\nexit 0\n")
	sourceRootFor = repo
	t.Cleanup(func() { sourceRootFor = "" })
	if code := run([]string{"doctor"}); code != 0 {
		t.Fatalf("doctor: %d", code)
	}
	gotCommit, got, present := echoAdapters(t, home)
	if !present || gotCommit != commit || strings.Join(got, ",") != "claude,codex" {
		t.Fatalf("state.json: present=%v commit=%s adapters=%v", present, gotCommit, got)
	}
}

func TestDoctorLedgerReplaces(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	repo, commit := echoFixture(t, home, "#!/bin/sh\nif [ \"$1\" = --list-installed ]; then printf 'codex\\nclaude\\n'; exit 0; fi\nexit 0\n")
	sourceRootFor = repo
	t.Cleanup(func() { sourceRootFor = "" })
	if code := run([]string{"doctor"}); code != 0 {
		t.Fatalf("doctor: %d", code)
	}
	// Update the lock to accept the fixture store's changed tree without fetching.
	store := filepath.Join(home, ".gap", "store", "echo", commit)
	write(t, filepath.Join(store, "scripts", "install.sh"), "#!/bin/sh\nif [ \"$1\" = --list-installed ]; then printf 'pi\\n'; exit 0; fi\nexit 0\n")
	tree, err := treeHash(store)
	if err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(repo, "gap.lock"), "schema = \"2\"\n\n[components.echo]\nref = \"master\"\ncommit = \""+commit+"\"\ntree = \""+tree+"\"\n")
	if code := run([]string{"doctor"}); code != 0 {
		t.Fatalf("second doctor: %d", code)
	}
	gotCommit, got, present := echoAdapters(t, home)
	if !present || gotCommit != commit || strings.Join(got, ",") != "pi" {
		t.Fatalf("ledger was not replaced: present=%v commit=%s adapters=%v", present, gotCommit, got)
	}
}

func TestDoctorLedgerNoFlag(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	repo, _ := echoFixture(t, home, "#!/bin/sh\necho \"Unknown argument: $1\" >&2\nexit 2\n")
	sourceRootFor = repo
	t.Cleanup(func() { sourceRootFor = "" })
	if code := run([]string{"doctor"}); code != 0 {
		t.Fatalf("a missing flag must not fail the doctor: %d", code)
	}
	if _, _, present := echoAdapters(t, home); present {
		t.Fatal("echo reported nothing, but state.json has an echo entry")
	}
}
