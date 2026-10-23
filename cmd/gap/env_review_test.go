package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnvReviewInit(t *testing.T) {
	recallStub(t, "exit 0\n")
	repo := t.TempDir()
	write(t, filepath.Join(repo, "AGENTS.md"), "project rules\n")
	if code := run([]string{"init", "--adopt", repo}); code != 0 {
		t.Fatalf("adopt: %d", code)
	}
	config := readString(t, filepath.Join(repo, ".gap", "gap.toml"))
	doc, err := parseTOML(config)
	if err != nil || doc["profile"] != "project" || doc["slug"] != filepath.Base(repo) {
		t.Fatalf("project configuration: %+v %v", doc, err)
	}
	pointer := readString(t, filepath.Join(repo, "AGENTS.md"))
	if !strings.HasPrefix(pointer, "project rules\n") || !strings.Contains(pointer, "https://github.com/edheltzel/GAP/blob/master/global/AGENTS.md") {
		t.Fatalf("policy pointer: %s", pointer)
	}
	if code := run([]string{"init", "--adopt", repo}); code != 0 || readString(t, filepath.Join(repo, "AGENTS.md")) != pointer || readString(t, filepath.Join(repo, ".gap", "gap.toml")) != config {
		t.Fatal("repeated adoption changed existing project files")
	}
	for _, marker := range []string{"<!-- gap:begin -->", "<!-- gap:end -->"} {
		write(t, filepath.Join(repo, "AGENTS.md"), marker+"\nprotected\n")
		if code := run([]string{"init", repo}); code != 0 || readString(t, filepath.Join(repo, "AGENTS.md")) != marker+"\nprotected\n" {
			t.Fatal("init modified a marked file")
		}
	}
}

func TestEnvReviewEffectiveSelection(t *testing.T) {
	home, distroRoot, project, remote := t.TempDir(), t.TempDir(), t.TempDir(), t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_STATE_HOME", "")
	commit, tree := gitFixture(t, remote, "project skill\n")
	write(t, filepath.Join(distroRoot, "gap.toml"), "schema = \"2\"\n[components.tools]\nrepo = \""+remote+"\"\nref = \"master\"\nnames = [\"do-todo\"]\n[components.tools.skills.do-todo]\nload = \"project\"\n[components.tools.skills.do-todo.when]\nfiles = [\"wrong.config\"]\n")
	if err := writeLock(distroRoot, lockFile{Components: map[string]lockEntryM1{"tools": {Ref: "master", Commit: commit, Tree: tree}}}); err != nil {
		t.Fatal(err)
	}
	lock := readString(t, filepath.Join(distroRoot, "gap.lock"))
	readyEnvReviewDistro(t, distroRoot)
	if _, err := git(project, "init", "-q", "-b", "master"); err != nil {
		t.Fatal(err)
	}
	t.Chdir(project)
	write(t, filepath.Join(home, ".config", "gap", "gap.toml"), "[components.tools]\nenabled = false\n")
	write(t, filepath.Join(project, ".gap", "gap.toml"), "[components.tools]\nenabled = true\n[components.tools.skills.do-todo.when]\nfiles = [\"astro.config.*\"]\n")
	for _, match := range []bool{false, true} {
		if match {
			write(t, filepath.Join(project, "astro.config.mjs"), "export default {}\n")
		}
		if code := run([]string{"sync"}); code != 0 {
			t.Fatalf("sync selection: %d", code)
		}
		for _, dir := range skillDirs(project, "do-todo") {
			_, err := os.Stat(filepath.Join(dir, "SKILL.md"))
			if match && err != nil || !match && !os.IsNotExist(err) {
				t.Fatalf("project match=%v destination %s: %v", match, dir, err)
			}
		}
	}
	if _, err := os.Stat(filepath.Join(home, ".agents")); !os.IsNotExist(err) {
		t.Fatal("project skill leaked into the home directory")
	}
	nested := filepath.Join(project, "src")
	write(t, filepath.Join(nested, "entry.txt"), "entry\n")
	t.Chdir(nested)
	if code := run([]string{"status"}); code != 0 {
		t.Fatalf("project status: %d", code)
	}
	write(t, filepath.Join(project, ".agents", "skills", "do-todo", "SKILL.md"), "edit\n")
	if code := run([]string{"restore"}); code != 0 || readString(t, filepath.Join(project, ".agents", "skills", "do-todo", "SKILL.md")) != "project skill\n" {
		t.Fatal("project restore did not repair the project destination")
	}
	write(t, filepath.Join(project, ".gap", "gap.toml"), "[components.tools]\nenabled = true\n[components.tools.skills.do-todo]\nload = \"off\"\n")
	for _, dir := range skillDirs(project, "do-todo") {
		if err := os.RemoveAll(dir); err != nil {
			t.Fatal(err)
		}
	}
	if code := run([]string{"status"}); code != 0 {
		t.Fatalf("disabled skill status: %d", code)
	}
	if code := run([]string{"restore"}); code != 0 {
		t.Fatalf("disabled skill restore: %d", code)
	}
	if _, err := os.Stat(filepath.Join(home, ".agents", "skills", "do-todo")); !os.IsNotExist(err) {
		t.Fatal("restore installed an off skill globally")
	}
	if readString(t, filepath.Join(distroRoot, "gap.lock")) != lock {
		t.Fatal("effective selection changed gap.lock")
	}
	write(t, filepath.Join(project, ".gap", "gap.toml"), "[components.tools]\nenabled = true\nref = \"release\"\n")
	_, stderr := captureStd(t, func() {
		if code := run([]string{"sync"}); code != 1 {
			t.Errorf("ref override sync: %d", code)
		}
	})
	if !strings.Contains(stderr, "gap pin tools@release") {
		t.Fatalf("ref mismatch did not explain pinning: %s", stderr)
	}
	if code := run([]string{"status"}); code != 1 {
		t.Fatalf("ref override status: %d", code)
	}
	if code := run([]string{"restore"}); code != 1 || readString(t, filepath.Join(distroRoot, "gap.lock")) != lock {
		t.Fatal("ref mismatch restore must refuse without changing the lock")
	}
}

func TestEnvReviewComponentCommands(t *testing.T) {
	home, repo, project, remote := t.TempDir(), t.TempDir(), t.TempDir(), t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_STATE_HOME", "")
	if _, err := git(project, "init", "-q", "-b", "master"); err != nil {
		t.Fatal(err)
	}
	t.Chdir(project)
	write(t, filepath.Join(remote, "install.sh"), "#!/bin/sh\nprintf 'installed\\n' >> \"$HOME/install.log\"\n")
	write(t, filepath.Join(remote, "doctor.sh"), "#!/bin/sh\nprintf 'checked\\n' >> \"$HOME/doctor.log\"\nexit 1\n")
	commit, tree := gitFixture(t, remote, "owned by component\n")
	write(t, filepath.Join(repo, "gap.toml"), "schema = \"2\"\n[components.owned]\nrepo = \""+remote+"\"\nref = \"master\"\nnames = []\ninstall = \"sh install.sh\"\ndoctor = \"sh doctor.sh\"\n")
	if err := writeLock(repo, lockFile{Components: map[string]lockEntryM1{"owned": {Ref: "master", Commit: commit, Tree: tree}}}); err != nil {
		t.Fatal(err)
	}
	readyEnvReviewDistro(t, repo)
	if code := run([]string{"sync"}); code != 0 || readString(t, filepath.Join(home, "install.log")) != "installed\n" {
		t.Fatalf("component-owned install: %d", code)
	}
	bin := t.TempDir()
	write(t, filepath.Join(bin, "codex"), "#!/bin/sh\nprintf 'harness executed\\n' >> \"$HOME/harness.log\"\n")
	if err := os.Chmod(filepath.Join(bin, "codex"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	stdout, _ := captureStd(t, func() {
		if code := run([]string{"doctor"}); code != 1 {
			t.Errorf("failed component doctor must fail: %d", code)
		}
	})
	if !strings.Contains(stdout, "codex: present") || readString(t, filepath.Join(home, "doctor.log")) != "checked\n" || readString(t, filepath.Join(home, "install.log")) != "installed\n" {
		t.Fatalf("doctor contract: %s", stdout)
	}
	if _, err := os.Stat(filepath.Join(home, "harness.log")); !os.IsNotExist(err) {
		t.Fatal("doctor executed a harness")
	}
	config := readString(t, filepath.Join(repo, "gap.toml"))
	write(t, filepath.Join(repo, "gap.toml"), strings.Replace(config, "sh doctor.sh", "sh doctor.sh || true", 1))
	if code := run([]string{"doctor"}); code != 0 || readString(t, filepath.Join(home, "install.log")) != "installed\n" {
		t.Fatalf("successful component doctor: %d", code)
	}
}

func TestEnvReviewCrew(t *testing.T) {
	home, repo, project := t.TempDir(), t.TempDir(), t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("CODEX_HOME", "relative-invalid-for-machine-sync")
	if _, err := git(project, "init", "-q", "-b", "master"); err != nil {
		t.Fatal(err)
	}
	t.Chdir(project)
	write(t, filepath.Join(repo, "gap.toml"), "schema = \"2\"\n[components.echo]\nrepo = \"github.com/owner/never-fetch\"\nref = \"master\"\nnames = []\n[components.recall]\nrepo = \"github.com/owner/never-fetch\"\nref = \"main\"\nnames = []\n")
	write(t, filepath.Join(repo, "profiles", "crew.toml"), "profile = \"crew\"\n[components.echo]\nenabled = false\n[components.recall]\nenabled = false\n")
	sourceRootFor = repo
	t.Cleanup(func() { sourceRootFor = "" })
	if code := run([]string{"sync", "--profile", "crew"}); code != 0 {
		t.Fatalf("crew sync: %d", code)
	}
	if entries, err := os.ReadDir(home); err != nil || len(entries) != 0 {
		t.Fatalf("crew wrote machine state: %v %v", entries, err)
	}
	doc, err := parseTOML(readString(t, filepath.Join(project, ".gap", "gap.toml")))
	if err != nil || doc["profile"] != "crew" {
		t.Fatalf("crew project config: %+v %v", doc, err)
	}
	if body := readString(t, filepath.Join(project, "AGENTS.md")); !strings.Contains(body, "https://github.com/edheltzel/GAP/blob/master/global/AGENTS.md") {
		t.Fatalf("crew policy pointer: %s", body)
	}
}

func TestEnvReviewNativeReplacement(t *testing.T) {
	home, store, bin := t.TempDir(), t.TempDir(), t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_STATE_HOME", "")
	write(t, filepath.Join(store, "skills", "global", "core", "do-todo", "SKILL.md"), "pinned\n")
	write(t, filepath.Join(bin, "git"), "#!/bin/sh\nprintf 'git\\n' >> \"$HOME/tool.log\"\nexit 71\n")
	write(t, filepath.Join(bin, "npx"), "#!/bin/sh\nprintf 'npx\\n' >> \"$HOME/tool.log\"\nexit 71\n")
	for _, name := range []string{"git", "npx"} {
		if err := os.Chmod(filepath.Join(bin, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", bin)
	legacy := "{\"skills\":{\"unmanaged\":{\"source\":\"owner/repo\"}}}\n"
	write(t, skillLockPath(home), legacy)
	entry := manifestEntry{Source: "owner/tools", Skills: []string{"do-todo"}, Path: store}
	if !installSkills(store, home, []manifestEntry{entry}) || !installSkills(store, home, []manifestEntry{entry}) {
		t.Fatal("native install required an external tool")
	}
	write(t, filepath.Join(home, ".agents", "skills", "do-todo", "SKILL.md"), "edited\n")
	if ok, err := replaceSkill("", store, home, entry, "do-todo"); err != nil || !ok || readString(t, filepath.Join(home, ".agents", "skills", "do-todo", "SKILL.md")) != "pinned\n" {
		t.Fatalf("native take source: %v %v", ok, err)
	}
	entry.Skills = []string{"missing"}
	if installSkills(store, home, []manifestEntry{entry}) {
		t.Fatal("missing pinned source was accepted")
	}
	if _, err := os.Stat(filepath.Join(home, "tool.log")); !os.IsNotExist(err) {
		t.Fatal("pinned installs invoked npx or repeated drift hashing")
	}
	if readString(t, skillLockPath(home)) != legacy {
		t.Fatal("native install changed the unmanaged lock")
	}
	if err := os.RemoveAll(filepath.Join(home, ".claude", "skills")); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(home, ".claude", "skills"), "blocked\n")
	write(t, filepath.Join(home, ".agents", "skills", "do-todo", "SKILL.md"), "keep my edit\n")
	if ok, err := replaceSkill("", store, home, entry, "do-todo"); err == nil || ok {
		t.Fatalf("replacement unexpectedly succeeded: %v %v", ok, err)
	}
	if readString(t, filepath.Join(home, ".agents", "skills", "do-todo", "SKILL.md")) != "keep my edit\n" || readString(t, filepath.Join(home, ".claude", "skills")) != "blocked\n" {
		t.Fatal("failed replacement lost existing files")
	}
}

func readyEnvReviewDistro(t *testing.T, repo string) {
	t.Helper()
	t.Setenv("CODEX_HOME", "")
	write(t, filepath.Join(repo, "global", "AGENTS.md"), "fixture policy\n")
	write(t, filepath.Join(repo, "global", "skills.json"), "[]\n")
	gitFixture(t, repo, "distro skill\n")
	for _, args := range [][]string{
		{"update-ref", "refs/remotes/origin/master", "HEAD"},
		{"symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/master"},
	} {
		if _, err := git(repo, args...); err != nil {
			t.Fatal(err)
		}
	}
	sourceRootFor = repo
	t.Cleanup(func() { sourceRootFor = "" })
}
