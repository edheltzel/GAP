//go:build unix

package main

import (
	"fmt"
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

func TestUpdateFeedbackFailureBeforeInstall(t *testing.T) {
	for _, failure := range []string{"state", "lock", "first ref", "later ref", "manifest"} {
		t.Run(failure, func(t *testing.T) {
			e := newUtilityEnv(t, "schema = \"1\"\nname = \"util\"\nskills = [\"util-skill\"]\n")
			commit, err := git(e.remote, "rev-parse", "HEAD")
			if err != nil {
				t.Fatal(err)
			}
			tree, err := git(e.remote, "rev-parse", "HEAD^{tree}")
			if err != nil {
				t.Fatal(err)
			}
			lk := lockFile{Components: map[string]lockEntryM1{}}
			var config strings.Builder
			for _, id := range []string{"a", "b", "c"} {
				ref := "master"
				if failure == "first ref" && id == "a" || failure == "later ref" && id == "b" {
					ref = "missing-ref"
				}
				fmt.Fprintf(&config, "[components.%s]\nrepo = %q\nref = %q\n", id, e.remote, ref)
				lk.Components[id] = lockEntryM1{Ref: "master", Commit: commit, Tree: tree}
			}
			write(t, userConfigFile(e.home), config.String())
			if err := writeLock(userConfigDir(e.home), lk); err != nil {
				t.Fatal(err)
			}
			switch failure {
			case "state":
				write(t, statePath(e.home), "invalid json\n")
			case "lock":
				write(t, filepath.Join(userConfigDir(e.home), "gap.lock"), "schema = \"invalid\"\n")
			case "later ref":
				write(t, filepath.Join(e.remote, "new-version"), "v2\n")
				commitAll(t, e.remote)
			case "manifest":
				write(t, filepath.Join(e.remote, utilityManifestName), "schema = \"invalid\"\n")
				commitAll(t, e.remote)
			}
			stdout, stderr := captureStd(t, func() {
				if code := run([]string{"update"}); code != 1 {
					t.Errorf("failed update exit %d, want 1", code)
				}
			})
			if stderr == "" {
				t.Fatal("failed update omitted the error")
			}
			unchecked := []string{"a", "b", "c"}
			summary := "0 utilities checked: 0 updated, 0 up to date; 3 not checked; update failed"
			if failure == "later ref" {
				unchecked = []string{"b", "c"}
				summary = "1 utility checked: 1 updated, 0 up to date; 2 not checked; update failed"
				wantLines(t, "partial update", stdout, "a moved from "+commit[:7])
				got, err := readLock(userConfigDir(e.home))
				if err != nil || got.Components["a"].Commit == commit || got.Components["b"].Commit != commit || got.Components["c"].Commit != commit {
					t.Fatalf("partial update pins = %+v, %v", got, err)
				}
			}
			for _, id := range unchecked {
				wantLines(t, "failed update", stdout, id+" was not checked; update stopped")
			}
			wantLines(t, "failed update", stdout, summary)
		})
	}
}

func TestUpdateFeedbackHookFailure(t *testing.T) {
	e := newUtilityEnv(t, "schema = \"1\"\nname = \"util\"\nskills = [\"util-skill\"]\n[hooks]\nupdate = \"exit 1\"\n")
	quietStdout(t)
	write(t, userConfigFile(e.home), "[trust]\nrepos = [\""+e.remote+"\"]\n")
	if err := addUtility(e.remote, "util"); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(e.remote, "new-version"), "v2\n")
	commit, _ := commitAll(t, e.remote)
	stdout, stderr := captureStd(t, func() {
		if code := run([]string{"update", "util"}); code != 1 {
			t.Errorf("failed update exit %d, want 1", code)
		}
	})
	wantLines(t, "failed hook", stdout, "util moved from ", " to "+commit[:7], "1 utility checked: 1 updated, 0 up to date; update failed")
	wantLines(t, "failed hook error", stderr, "component util update")
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
	wantLines(t, "clean restore", restore(), "Nothing to restore: checked 1 selected skill; each matches its pin.")
	write(t, filepath.Join(home, ".agents", "skills", "do-todo", "SKILL.md"), "edited\n")
	wantLines(t, "drift restore", restore(), "restored do-todo", "from do-skills @ master ("+commit[:7]+")", "Restored 1 skill.")
}

func TestRestoreFeedbackScope(t *testing.T) {
	for _, load := range []string{"always", "listed", "off", "command", "project unmatched", "project matched"} {
		t.Run(load, func(t *testing.T) {
			e := newUtilityEnv(t, "schema = \"1\"\nname = \"util\"\nskills = [\"util-skill\", \"other-skill\"]\n")
			write(t, filepath.Join(e.remote, "skills", "other-skill", "SKILL.md"), "other source\n")
			commit, tree := commitAll(t, e.remote)
			level := strings.Fields(load)[0]
			config := fmt.Sprintf("schema = \"2\"\n[components.clean]\nrepo = %q\nref = \"master\"\nnames = [\"util-skill\"]\n[components.clean.skills.util-skill]\nload = %q\n[components.clean.skills.util-skill.when]\nfiles = [\"match.txt\"]\n[components.drifted]\nrepo = %q\nref = \"master\"\nnames = [\"other-skill\"]\n", e.remote, level, e.remote)
			write(t, filepath.Join(e.repo, "gap.toml"), config)
			pin := lockEntryM1{Ref: "master", Commit: commit, Tree: tree}
			if err := writeLock(e.repo, lockFile{Components: map[string]lockEntryM1{"clean": pin, "drifted": pin}}); err != nil {
				t.Fatal(err)
			}
			target := e.home
			if load == "project matched" {
				write(t, filepath.Join(e.project, "match.txt"), "match\n")
				target = e.project
			}
			for _, dest := range skillDirs(target, "util-skill") {
				write(t, filepath.Join(dest, "SKILL.md"), "---\nname: util-skill\ndescription: test\n---\n")
			}
			for _, dest := range skillDirs(e.home, "other-skill") {
				write(t, filepath.Join(dest, "SKILL.md"), "edited\n")
			}
			stdout, _ := captureStd(t, func() {
				if code := run([]string{"restore", "clean"}); code != 0 {
					t.Errorf("restore exit %d", code)
				}
			})
			want := "Nothing to restore: checked 1 selected skill for clean; each matches its pin."
			if load == "off" || load == "command" || load == "project unmatched" {
				want = "Nothing to restore: no component skills selected for clean."
			}
			wantLines(t, "scoped restore", stdout, want)
			for _, dest := range skillDirs(e.home, "other-skill") {
				if got := readString(t, filepath.Join(dest, "SKILL.md")); got != "edited\n" {
					t.Fatalf("unselected skill changed to %q", got)
				}
			}
		})
	}
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
