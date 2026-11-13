//go:build unix

package main

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func wantLines(t *testing.T, what, stdout string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(stdout, want) {
			t.Fatalf("%s output is missing %q:\n%s", what, want, stdout)
		}
	}
}

func TestUpdateFeedbackNoUtilities(t *testing.T) {
	e := newUtilityEnv(t, hookManifest)
	write(t, filepath.Join(e.repo, "gap.toml"), "schema = \"2\"\n\n[components.tools]\nrepo = \""+e.remote+"\"\nref = \"master\"\n")
	stdout, _ := captureStd(t, func() {
		if code := run([]string{"update"}); code != 0 {
			t.Errorf("update exit %d", code)
		}
	})
	wantLines(t, "update", stdout, "No utilities added", "gap add owner/repo", "Distro components (tools)", "gap pin <id>@<ref>, then gap sync")
}

func TestUpdateFeedbackPerUtility(t *testing.T) {
	e := newUtilityEnv(t, "schema = \"1\"\nname = \"util\"\nskills = [\"util-skill\"]\n")
	quietStdout(t)
	write(t, userConfigFile(e.home), "[trust]\nrepos = [\""+e.remote+"\"]\n")
	if err := addUtility(e.remote, "util"); err != nil {
		t.Fatal(err)
	}
	first, _ := readLock(userConfigDir(e.home))
	stdout, _ := captureStd(t, func() {
		if code := run([]string{"update"}); code != 0 {
			t.Errorf("update exit %d", code)
		}
	})
	wantLines(t, "update", stdout, "util is up to date @ master ("+first.Components["util"].Commit[:7]+")", "1 utility checked: 0 updated, 1 up to date")
	write(t, filepath.Join(e.remote, "skills", "util-skill", "SKILL.md"), "---\nname: util-skill\ndescription: v2\n---\n")
	commit, _ := commitAll(t, e.remote)
	stdout, _ = captureStd(t, func() {
		if code := run([]string{"update"}); code != 0 {
			t.Errorf("update exit %d", code)
		}
	})
	wantLines(t, "update", stdout, "util moved from "+first.Components["util"].Commit[:7]+" to "+commit[:7]+" @ master", "1 utility checked: 1 updated, 0 up to date")
}

func TestAddRemoveFeedback(t *testing.T) {
	e := newUtilityEnv(t, "schema = \"1\"\nname = \"util\"\nskills = [\"util-skill\"]\n")
	stdout, _ := captureStd(t, func() {
		if code := run([]string{"add", e.remote, "--id", "util"}); code != 0 {
			t.Errorf("add exit %d", code)
		}
	})
	wantLines(t, "add", stdout, "util added @ master (")
	stdout, _ = captureStd(t, func() {
		if code := run([]string{"remove", "util"}); code != 0 {
			t.Errorf("remove exit %d", code)
		}
	})
	wantLines(t, "remove", stdout, "util removed")
}

func TestPinFeedback(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	t.Setenv("HOME", t.TempDir())
	repo := t.TempDir()
	remote := filepath.Join(t.TempDir(), "remote")
	commit, _ := gitFixture(t, remote, "pinned\n")
	write(t, filepath.Join(repo, "gap.toml"), "schema = \"2\"\n[components.do-skills]\nrepo = \""+remote+"\"\nref = \"master\"\nnames = [\"do-todo\"]\n")
	write(t, filepath.Join(repo, "global", "AGENTS.md"), "policy\n")
	sourceRootFor = repo
	t.Cleanup(func() { sourceRootFor = "" })
	pin := func() string {
		stdout, _ := captureStd(t, func() {
			if code := run([]string{"pin", "do-skills@master"}); code != 0 {
				t.Errorf("pin exit %d", code)
			}
		})
		return stdout
	}
	wantLines(t, "pin", pin(), "do-skills pinned @ master ("+commit[:7]+") in gap.lock; run gap sync to install it")
	wantLines(t, "second pin", pin(), "do-skills is already pinned @ master ("+commit[:7]+"); gap.lock unchanged")
}

func TestRestoreFeedback(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_STATE_HOME", "")
	t.Chdir(t.TempDir())
	repo := t.TempDir()
	write(t, filepath.Join(repo, "global", "AGENTS.md"), "policy\n")
	remote := filepath.Join(t.TempDir(), "remote")
	commit, tree := gitFixture(t, remote, "restore\n")
	write(t, filepath.Join(repo, "gap.toml"), "schema = \"2\"\n[components.do-skills]\nrepo = \""+remote+"\"\nref = \"master\"\nnames = [\"do-todo\"]\n")
	write(t, filepath.Join(repo, "gap.lock"), "schema = \"2\"\n\n[components.do-skills]\nref = \"master\"\ncommit = \""+commit+"\"\ntree = \""+tree+"\"\n")
	sourceRootFor = repo
	t.Cleanup(func() { sourceRootFor = "" })
	if _, err := ensureStore(home, component{ID: "do-skills", Repo: remote, Ref: "master"}, lockEntryM1{Ref: "master", Commit: commit, Tree: tree}); err != nil {
		t.Fatal(err)
	}
	for _, dest := range skillDirs(home, "do-todo") {
		write(t, filepath.Join(dest, "SKILL.md"), "restore\n")
	}
	restore := func() string {
		stdout, _ := captureStd(t, func() {
			if code := run([]string{"restore"}); code != 0 {
				t.Errorf("restore exit %d", code)
			}
		})
		return stdout
	}
	wantLines(t, "clean restore", restore(), "Nothing to restore: every installed component skill matches its pin.")
	write(t, filepath.Join(home, ".agents", "skills", "do-todo", "SKILL.md"), "edited\n")
	wantLines(t, "drift restore", restore(), "restored do-todo", "from do-skills @ master ("+commit[:7]+")", "Restored 1 skill.")
}

func TestUninstallFeedbackPiped(t *testing.T) {
	newUninstallEnv(t)
	stdout, _ := captureStd(t, func() {
		if code := run([]string{"uninstall", "--yes"}); code != 0 {
			t.Errorf("uninstall exit %d", code)
		}
	})
	wantLines(t, "uninstall", stdout, "Kept ", "Uninstall complete.")
}
