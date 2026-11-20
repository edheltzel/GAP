//go:build unix

package main

import (
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
}

func TestPinMustBeBool(t *testing.T) {
	if _, err := parseComponent("tools", map[string]any{"repo": "github.com/o/r", "ref": "master", "pin": "yes"}); err == nil || !strings.Contains(err.Error(), "pin must be true or false") {
		t.Fatalf("err = %v", err)
	}
}
