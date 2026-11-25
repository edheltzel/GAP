//go:build unix

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLazyUpdateCycle(t *testing.T) {
	home, repo, project, remote := t.TempDir(), t.TempDir(), t.TempDir(), t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_STATE_HOME", "")
	if _, err := git(project, "init", "-q", "-b", "master"); err != nil {
		t.Fatal(err)
	}
	t.Chdir(project)
	write(t, filepath.Join(remote, "skills", "kept", "SKILL.md"), "kept v1\n")
	c1, t1 := gitFixture(t, remote, "todo v1\n")
	spec := "schema = \"2\"\n[components.tools]\nrepo = \"" + remote + "\"\nref = \"master\"\nnames = [\"do-todo\", \"kept\"]\n"
	write(t, filepath.Join(repo, "gap.toml"), spec)
	if err := writeLock(repo, lockFile{Components: map[string]lockEntryM1{"tools": {Ref: "master", Commit: c1, Tree: t1}}}); err != nil {
		t.Fatal(err)
	}
	readyEnvReviewDistro(t, repo)
	gap := func(want int, args ...string) (string, string) {
		t.Helper()
		var code int
		stdout, stderr := captureStd(t, func() { code = run(args) })
		if code != want {
			t.Fatalf("gap %v exited %d, want %d:\n%s\n%s", args, code, want, stdout, stderr)
		}
		return stdout, stderr
	}
	gap(0, "sync", "--yes")
	for _, dir := range skillDirs(home, "kept") {
		write(t, filepath.Join(dir, "SKILL.md"), "my edit\n")
	}

	write(t, filepath.Join(remote, "skills", "kept", "SKILL.md"), "kept v2\n")
	c2, _ := gitFixture(t, remote, "todo v2\n")
	stdout, _ := gap(0, "check")
	wantLines(t, "check", stdout, "tools: 1 new on master since "+c1[:7], c2[:7]+" skill", "gap update")
	if lk, _ := readLock(repo); lk.Components["tools"].Commit != c1 {
		t.Fatal("check must not move the pin")
	}
	stdout, _ = gap(1, "status")
	wantLines(t, "status", stdout, "tools ref=master commit="+c1[:7]+" store=ok new=1")

	stdout, _ = gap(0, "update")
	wantLines(t, "update", stdout, "tools moved from "+c1[:7]+" to "+c2[:7]+" @ master", c2[:7]+" skill", "commit gap.lock")
	if lk, _ := readLock(repo); lk.Components["tools"].Commit != c2 {
		t.Fatalf("update must write the new pin to gap.lock")
	}
	stdout, _ = gap(1, "status")
	if strings.Contains(stdout, "new=") {
		t.Fatalf("status after update still reports new commits:\n%s", stdout)
	}

	readyEnvReviewDistro(t, repo)
	stdout, _ = gap(1, "sync", "--yes")
	wantLines(t, "sync", stdout, "Updated do-todo to "+c2[:7])
	for _, dir := range skillDirs(home, "do-todo") {
		if got := readString(t, filepath.Join(dir, "SKILL.md")); got != "todo v2\n" {
			t.Fatalf("untouched %s = %q, want the new version", dir, got)
		}
	}
	for _, dir := range skillDirs(home, "kept") {
		if got := readString(t, filepath.Join(dir, "SKILL.md")); got != "my edit\n" {
			t.Fatalf("edited %s = %q, must stay for drift review", dir, got)
		}
	}

	write(t, filepath.Join(repo, "gap.toml"), spec+"pin = true\n")
	readyEnvReviewDistro(t, repo)
	c3, _ := gitFixture(t, remote, "todo v3\n")
	stdout, _ = gap(0, "check")
	if strings.Contains(stdout, "tools") {
		t.Fatalf("check must skip a pinned component:\n%s", stdout)
	}
	gap(0, "update")
	if lk, _ := readLock(repo); lk.Components["tools"].Commit != c2 {
		t.Fatal("update must skip a component with pin = true")
	}
	stdout, _ = gap(0, "update", "tools")
	wantLines(t, "update tools", stdout, "tools moved from "+c2[:7]+" to "+c3[:7])

	write(t, filepath.Join(repo, "gap.toml"), spec)
	write(t, filepath.Join(project, ".gap", "gap.toml"), "[components.tools]\npin = true\n")
	readyEnvReviewDistro(t, repo)
	c4, _ := gitFixture(t, remote, "todo v4\n")
	gap(0, "update")
	if lk, _ := readLock(repo); lk.Components["tools"].Commit != c3 {
		t.Fatal("update must skip a component held with pin = true in a project override")
	}
	stdout, _ = gap(0, "update", "tools")
	wantLines(t, "update held tools", stdout, "tools moved from "+c3[:7]+" to "+c4[:7])
	os.Remove(filepath.Join(project, ".gap", "gap.toml"))

	old := filepath.Join(home, ".gap", "store", "tools", c1)
	if !verifiedStore(old, c1) {
		t.Fatal("an untouched earlier store must count as a baseline")
	}
	write(t, filepath.Join(old, "skills", "kept", "SKILL.md"), "my edit\n")
	if verifiedStore(old, c1) {
		t.Fatal("an earlier store with edited files must not count as a baseline")
	}
}

func TestPinMustBeBool(t *testing.T) {
	if _, err := parseComponent("tools", map[string]any{"repo": "github.com/o/r", "ref": "master", "pin": "yes"}); err == nil || !strings.Contains(err.Error(), "pin must be true or false") {
		t.Fatalf("err = %v", err)
	}
}

func TestHeldUtilitySkipsUnnamedUpdate(t *testing.T) {
	e := newUtilityEnv(t, "schema = \"1\"\nname = \"util\"\nskills = [\"util-skill\"]\n")
	quietStdout(t)
	write(t, userConfigFile(e.home), "[trust]\nrepos = [\""+e.remote+"\"]\n")
	if err := runAdd(addGlobalArgs(e.remote, "--id", "util")); err != nil {
		t.Fatal(err)
	}
	write(t, userConfigFile(e.home), readString(t, userConfigFile(e.home))+"pin = true\n")
	first, _ := readLock(userConfigDir(e.home))
	write(t, filepath.Join(e.remote, "skills", "util-skill", "SKILL.md"), "---\nname: util-skill\ndescription: v2\n---\n")
	commit, _ := commitAll(t, e.remote)
	if err := updateUtilities(""); err != nil {
		t.Fatal(err)
	}
	if lk, _ := readLock(userConfigDir(e.home)); lk.Components["util"].Commit != first.Components["util"].Commit {
		t.Fatal("unnamed update moved a utility with pin = true")
	}
	if err := updateUtilities("util"); err != nil {
		t.Fatal(err)
	}
	if lk, _ := readLock(userConfigDir(e.home)); lk.Components["util"].Commit != commit {
		t.Fatal("naming a held utility must update it")
	}
}

func TestPendingCommitsPeelsAnnotatedTag(t *testing.T) {
	home, remote := t.TempDir(), t.TempDir()
	commit, _ := gitFixture(t, remote, "v1\n")
	if _, err := git(remote, "-c", "user.name=gap", "-c", "user.email=gap@example.com", "tag", "-a", "v1", "-m", "v1"); err != nil {
		t.Fatal(err)
	}
	tip, commits, diverged, err := pendingCommits(home, component{ID: "tools", Repo: remote, Ref: "v1"}, commit)
	if err != nil || tip != commit || len(commits) != 0 || diverged {
		t.Fatalf("tip=%s commits=%v diverged=%v err=%v, want %s", tip, commits, diverged, err, commit)
	}
}
