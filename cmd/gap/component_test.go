package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureComponentCloneIfMissing(t *testing.T) {
	bare, pin, _ := doSkillsBare(t, "from-pin\n", "from-later\n")
	repo := t.TempDir()
	writeDoSkillsLock(t, repo, bare, pin)
	c := mustDoSkillsComponent(t, repo)
	if err := ensureComponent(repo, c); err != nil {
		t.Fatal(err)
	}
	dir := componentDir(repo, c.Name)
	head, err := gitRun(dir, "rev-parse", "HEAD")
	if err != nil || head != pin {
		t.Fatalf("clone-if-missing HEAD=%s pin=%s err=%v", head, pin, err)
	}
	if readString(t, filepath.Join(dir, "skills", "global", "core", "do-x", "SKILL.md")) != "from-pin\n" {
		t.Fatal("clone-if-missing did not check out the pin")
	}
	if err := ensureComponent(repo, c); err != nil {
		t.Fatal(err)
	}
	head, err = gitRun(dir, "rev-parse", "HEAD")
	if err != nil || head != pin {
		t.Fatalf("idempotent ensure moved off the pin: HEAD=%s", head)
	}
}

func TestEnsureComponentFetchToPin(t *testing.T) {
	bare, pin, later := doSkillsBare(t, "from-pin\n", "from-later\n")
	repo := t.TempDir()
	writeDoSkillsLock(t, repo, bare, pin)
	c := mustDoSkillsComponent(t, repo)
	dir := componentDir(repo, c.Name)
	if err := cloneGit(t, bare, dir); err != nil {
		t.Fatal(err)
	}
	if _, err := gitRun(dir, "checkout", "-q", later); err != nil {
		t.Fatal(err)
	}
	head, err := gitRun(dir, "rev-parse", "HEAD")
	if err != nil || head != later {
		t.Fatalf("precondition HEAD=%s later=%s err=%v", head, later, err)
	}
	if err := ensureComponent(repo, c); err != nil {
		t.Fatal(err)
	}
	head, err = gitRun(dir, "rev-parse", "HEAD")
	if err != nil || head != pin {
		t.Fatalf("fetch-to-pin HEAD=%s pin=%s err=%v", head, pin, err)
	}
	if readString(t, filepath.Join(dir, "skills", "global", "core", "do-x", "SKILL.md")) != "from-pin\n" {
		t.Fatal("fetch-to-pin did not restore pin bytes")
	}
}

func TestEnsureComponentDirtyAndOffPin(t *testing.T) {
	bare, pin, later := doSkillsBare(t, "from-pin\n", "from-later\n")
	repo := t.TempDir()
	writeDoSkillsLock(t, repo, bare, pin)
	c := mustDoSkillsComponent(t, repo)
	dir := componentDir(repo, c.Name)
	if err := cloneGit(t, bare, dir); err != nil {
		t.Fatal(err)
	}
	if _, err := gitRun(dir, "checkout", "-q", later); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(dir, "skills", "global", "core", "do-x", "SKILL.md"), "dirty\n")
	if err := ensureComponent(repo, c); err == nil || !strings.Contains(err.Error(), "dirty") {
		t.Fatalf("dirty checkout must fail ensure: %v", err)
	}
	head, err := gitRun(dir, "rev-parse", "HEAD")
	if err != nil || head != later {
		t.Fatalf("dirty checkout was moved: HEAD=%s later=%s err=%v", head, later, err)
	}
	if readString(t, filepath.Join(dir, "skills", "global", "core", "do-x", "SKILL.md")) != "dirty\n" {
		t.Fatal("dirty checkout was reset")
	}
	st, err := inspectComponent(dir, pin)
	if err != nil || !st.Dirty || !st.Off {
		t.Fatalf("status: dirty=%v off=%v err=%v", st.Dirty, st.Off, err)
	}
}

func TestPinnedSkillDriftAgainstCheckout(t *testing.T) {
	bare, pin, _ := doSkillsBare(t, "from-pin\n", "from-later\n")
	repo := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_STATE_HOME", "")
	writeDoSkillsLock(t, repo, bare, pin)
	c := mustDoSkillsComponent(t, repo)
	if err := ensureComponent(repo, c); err != nil {
		t.Fatal(err)
	}
	entries := []manifestEntry{{Source: "edheltzel/Do-Skills", Skills: []string{"do-x"}}}
	for _, dir := range skillDirs(home, "do-x") {
		write(t, filepath.Join(dir, "SKILL.md"), "from-pin\n")
	}
	ok, err := checkDrift(repo, home, entries)
	if err != nil || !ok {
		t.Fatalf("matching install must pass: ok=%v err=%v", ok, err)
	}
	write(t, filepath.Join(skillDirs(home, "do-x")[0], "SKILL.md"), "hand edit\n")
	ok, err = checkDrift(repo, home, entries)
	if err != nil || ok {
		t.Fatalf("edit against pinned checkout must be drift: ok=%v err=%v", ok, err)
	}
	if readString(t, filepath.Join(skillDirs(home, "do-x")[0], "SKILL.md")) != "hand edit\n" {
		t.Fatal("drift check overwrote the edit")
	}
}

func TestCheckDriftDoesNotInstallFromDirtyCheckout(t *testing.T) {
	bare, pin, later := doSkillsBare(t, "from-pin\n", "from-later\n")
	repo := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_STATE_HOME", "")
	writeDoSkillsLock(t, repo, bare, pin)
	c := mustDoSkillsComponent(t, repo)
	dir := componentDir(repo, c.Name)
	if err := cloneGit(t, bare, dir); err != nil {
		t.Fatal(err)
	}
	if _, err := gitRun(dir, "checkout", "-q", later); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(dir, "skills", "global", "core", "do-x", "SKILL.md"), "dirty-install\n")
	for _, d := range skillDirs(home, "do-x") {
		write(t, filepath.Join(d, "SKILL.md"), "from-pin\n")
	}
	npxLog := filepath.Join(t.TempDir(), "npx.log")
	npx := filepath.Join(t.TempDir(), "npx")
	write(t, npx, "#!/bin/sh\necho installed-from-dirty >> "+npxLog+"\nmkdir -p \"$HOME/.agents/skills/do-x\" \"$HOME/.claude/skills\"\nprintf '%s' 'dirty-install\n' > \"$HOME/.agents/skills/do-x/SKILL.md\"\nln -sfn \"$HOME/.agents/skills/do-x\" \"$HOME/.claude/skills/do-x\"\n")
	if err := os.Chmod(npx, 0o755); err != nil {
		t.Fatal(err)
	}
	oldPath := os.Getenv("PATH")
	t.Setenv("PATH", filepath.Dir(npx)+string(os.PathListSeparator)+oldPath)

	rOut, wOut, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	rErr, wErr, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	oldStdout, oldStderr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = wOut, wErr
	ok, err := checkDrift(repo, home, []manifestEntry{{Source: "edheltzel/Do-Skills", Skills: []string{"do-x"}}})
	replaced, replaceErr := replaceSkill(npx, repo, home, "edheltzel/Do-Skills", "do-x")
	os.Stdout, os.Stderr = oldStdout, oldStderr
	wOut.Close()
	wErr.Close()
	out, _ := io.ReadAll(rOut)
	errOut, _ := io.ReadAll(rErr)
	if err != nil || ok {
		t.Fatalf("dirty pin must fail drift: ok=%v err=%v", ok, err)
	}
	if strings.Contains(string(out), "Skill drifted") {
		t.Fatalf("dirty component skills must be skipped, got stdout %q", out)
	}
	if !strings.Contains(string(errOut), "dirty") {
		t.Fatalf("want dirty report, got stderr %q", errOut)
	}
	if replaced || replaceErr != nil {
		t.Fatalf("take-source must not install from a dirty pin: replaced=%v err=%v", replaced, replaceErr)
	}
	if _, statErr := os.Stat(npxLog); !os.IsNotExist(statErr) {
		t.Fatalf("npx must not run against a dirty pin, log=%q", readString(t, npxLog))
	}
	if readString(t, skillDirs(home, "do-x")[0]+"/SKILL.md") != "from-pin\n" {
		t.Fatal("installed skill was replaced from the dirty checkout")
	}
}

func TestCheckDriftReportsDirtyCheckout(t *testing.T) {
	bare, pin, _ := doSkillsBare(t, "from-pin\n", "from-later\n")
	repo := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_STATE_HOME", "")
	writeDoSkillsLock(t, repo, bare, pin)
	c := mustDoSkillsComponent(t, repo)
	if err := ensureComponent(repo, c); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(componentDir(repo, c.Name), "extra.txt"), "dirt\n")
	for _, dir := range skillDirs(home, "do-x") {
		write(t, filepath.Join(dir, "SKILL.md"), "from-pin\n")
	}
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stderr
	os.Stderr = w
	ok, err := checkDrift(repo, home, []manifestEntry{{Source: "edheltzel/Do-Skills", Skills: []string{"do-x"}}})
	os.Stderr = old
	w.Close()
	out, _ := io.ReadAll(r)
	if err != nil || ok {
		t.Fatalf("dirty pin checkout must fail drift: ok=%v err=%v", ok, err)
	}
	if !strings.Contains(string(out), "dirty") {
		t.Fatalf("want dirty report, got %q", out)
	}
}

func TestPinCommitMustBeHexSHA(t *testing.T) {
	repo := t.TempDir()
	good := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	for _, bad := range []string{"master", "-u", "-fetch", "78E2C15BFBC484D74A2D38EF3FA727A99272BF25", "abc", strings.Repeat("a", 39), strings.Repeat("a", 41), "gggggggggggggggggggggggggggggggggggggggg"} {
		writeDoSkillsLock(t, repo, "https://github.com/edheltzel/Do-Skills.git", bad)
		if _, err := loadSkillPaths(repo); err == nil || !strings.Contains(err.Error(), "40-character lowercase hex SHA") {
			t.Fatalf("load commit %q: %v", bad, err)
		}
		err := ensureComponent(repo, component{Name: "do-skills", Repo: repo, Ref: "master", Commit: bad, Source: "edheltzel/Do-Skills"})
		if err == nil || !strings.Contains(err.Error(), "40-character lowercase hex SHA") {
			t.Fatalf("ensure commit %q: %v", bad, err)
		}
		if _, err := os.Stat(componentDir(repo, "do-skills")); !os.IsNotExist(err) {
			t.Fatalf("bad commit %q reached a git checkout", bad)
		}
	}
	writeDoSkillsLock(t, repo, "https://github.com/edheltzel/Do-Skills.git", good)
	if _, err := loadSkillPaths(repo); err != nil {
		t.Fatal(err)
	}
}

func TestPinnedSourceMatchesVendorPath(t *testing.T) {
	repo := t.TempDir()
	writeDoSkillsLock(t, repo, "/tmp/do-skills.git", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	vendor := componentDir(repo, "do-skills")
	if !sameSource(repo, "edheltzel/Do-Skills", lockEntry{Source: vendor, SourceType: "local", SourceURL: vendor}) {
		t.Fatal("a local lock path must match the pinned checkout")
	}
	if sameSource(repo, "edheltzel/Do-Skills", lockEntry{Source: "edheltzel/Do-Skills", SourceType: "github", SourceURL: "https://github.com/edheltzel/Do-Skills.git"}) {
		t.Fatal("a GitHub lock must not match the pinned checkout path")
	}
	if !sameSource(repo, ".", lockEntry{Source: repo, SourceType: "local", SourceURL: repo}) {
		t.Fatal("source . must still match this checkout")
	}
}

func TestSkillsAddPassesPinnedCheckout(t *testing.T) {
	bare, pin, _ := doSkillsBare(t, "from-pin\n", "from-later\n")
	repo := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeDoSkillsLock(t, repo, bare, pin)
	c := mustDoSkillsComponent(t, repo)
	if err := ensureComponent(repo, c); err != nil {
		t.Fatal(err)
	}
	npxLog := filepath.Join(t.TempDir(), "npx.log")
	npx := filepath.Join(t.TempDir(), "npx")
	write(t, npx, "#!/bin/sh\necho \"$*\" > "+npxLog+"\nmkdir -p \"$HOME/.agents/skills/do-x\" \"$HOME/.claude/skills\"\nprintf '%s' 'from-pin\n' > \"$HOME/.agents/skills/do-x/SKILL.md\"\nln -sfn \"$HOME/.agents/skills/do-x\" \"$HOME/.claude/skills/do-x\"\n")
	if err := os.Chmod(npx, 0o755); err != nil {
		t.Fatal(err)
	}
	if !skillsAdd(npx, repo, home, "edheltzel/Do-Skills", []string{"do-x"}) {
		t.Fatal("install from the pin must succeed")
	}
	got := readString(t, npxLog)
	want := componentDir(repo, c.Name)
	if !strings.Contains(got, want) {
		t.Fatalf("skills add must receive the vendor path %s, got %q", want, got)
	}
	if strings.Contains(got, "add edheltzel/Do-Skills") {
		t.Fatalf("skills add must not receive the unpinned GitHub source, got %q", got)
	}
}

func TestCheckDriftContinuesAfterComponentFailure(t *testing.T) {
	repo := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_STATE_HOME", "")
	writeDoSkillsLock(t, repo, "/no/such/do-skills.git", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	write(t, filepath.Join(repo, "skills", "global", "core", "gap-x", "SKILL.md"), "local\n")
	for _, dir := range skillDirs(home, "gap-x") {
		write(t, filepath.Join(dir, "SKILL.md"), "local\n")
	}
	write(t, filepath.Join(home, ".empryo", "skills", "do-x", "SKILL.md"), "shadow\n")
	write(t, filepath.Join(home, ".empryo", "skills", "gap-x", "SKILL.md"), "shadow\n")
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stderr
	os.Stderr = w
	ok, err := checkDrift(repo, home, []manifestEntry{
		{Source: "edheltzel/Do-Skills", Skills: []string{"do-x"}},
		{Source: ".", Skills: []string{"gap-x"}},
	})
	os.Stderr = old
	w.Close()
	out, _ := io.ReadAll(r)
	if err != nil || ok {
		t.Fatalf("failed pin must fail drift without aborting: ok=%v err=%v", ok, err)
	}
	got := string(out)
	if !strings.Contains(got, "fetch") && !strings.Contains(got, "not a git checkout") && !strings.Contains(got, "do-skills") {
		t.Fatalf("want component failure, got %q", got)
	}
	if !strings.Contains(got, "Empryo loads") {
		t.Fatalf("want Empryo shadow warnings after a failed component, got %q", got)
	}
	if strings.Contains(got, "gap-x") && strings.Contains(got, "drifted") {
		t.Fatalf("matching local skill must still be checked, got %q", got)
	}
}

func TestInstallSkillsReportsSkippedComponent(t *testing.T) {
	bare, pin, later := doSkillsBare(t, "from-pin\n", "from-later\n")
	repo := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeDoSkillsLock(t, repo, bare, pin)
	c := mustDoSkillsComponent(t, repo)
	dir := componentDir(repo, c.Name)
	if err := cloneGit(t, bare, dir); err != nil {
		t.Fatal(err)
	}
	if _, err := gitRun(dir, "checkout", "-q", later); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(dir, "skills", "global", "core", "do-x", "SKILL.md"), "dirty\n")
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stderr
	os.Stderr = w
	ok := installSkills(repo, home, []manifestEntry{{Source: "edheltzel/Do-Skills", Skills: []string{"do-x"}}})
	os.Stderr = old
	w.Close()
	out, _ := io.ReadAll(r)
	if ok {
		t.Fatal("dirty off-pin component must skip install")
	}
	got := string(out)
	if !strings.Contains(got, "dirty") {
		t.Fatalf("want dirty status when skipping install, got %q", got)
	}
	if !strings.Contains(got, "not pinned") {
		t.Fatalf("want off-pin status when skipping install, got %q", got)
	}
	if _, err := os.Stat(filepath.Join(home, ".agents", "skills", "do-x")); !os.IsNotExist(err) {
		t.Fatal("skipped component must not install skills")
	}
}

func TestIgnoredPathBlocksGapLock(t *testing.T) {
	if !ignoredPathBlocks("gap.lock") {
		t.Fatal("an ignored gap.lock must block; gap reads it")
	}
	if !ignoredPathBlocks("gap.toml") {
		t.Fatal("an ignored gap.toml must block; gap reads it")
	}
	if ignoredPathBlocks("global/components.lock") {
		t.Fatal("gap no longer reads global/components.lock")
	}
}

func TestLoadSkillPathsLockShape(t *testing.T) {
	repo := t.TempDir()
	writeDoSkillsLock(t, repo, "https://github.com/edheltzel/Do-Skills.git", "78e2c15bfbc484d74a2d38ef3fa727a99272bf25")
	c := mustDoSkillsComponent(t, repo)
	if c.Ref != "master" || c.Commit != "78e2c15bfbc484d74a2d38ef3fa727a99272bf25" || c.Repo == "" {
		t.Fatalf("lock shape: %+v", c)
	}
	write(t, filepath.Join(repo, gapLockPath), "schema = \"1\"\n[components.do-skills]\nrepo = \"https://github.com/edheltzel/Do-Skills.git\"\nref = \"master\"\ncommit = \"78e2c15bfbc484d74a2d38ef3fa727a99272bf25\"\n")
	if _, err := loadSkillPaths(repo); err == nil || !strings.Contains(err.Error(), `schema must be "2"`) {
		t.Fatalf("schema 1: %v", err)
	}
	writeDoSkillsLock(t, repo, "https://github.com/edheltzel/Do-Skills.git", "78e2c15bfbc484d74a2d38ef3fa727a99272bf25")
	write(t, filepath.Join(repo, gapTOMLPath), "[components.do-skills]\nrepo = \"https://github.com/edheltzel/Do-Skills.git\"\nref = \"main\"\n")
	if _, err := loadSkillPaths(repo); err == nil || !strings.Contains(err.Error(), "repo/ref") {
		t.Fatalf("toml mismatch: %v", err)
	}
}

func TestLockRejectsUnknownKeysAndBadTree(t *testing.T) {
	repo := t.TempDir()
	writeDoSkillsLock(t, repo, "https://github.com/edheltzel/Do-Skills.git", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	write(t, filepath.Join(repo, gapLockPath), `schema = "2"
comment = "nope"

[components.do-skills]
repo = "https://github.com/edheltzel/Do-Skills.git"
ref = "master"
commit = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
`)
	if _, err := loadSkillPaths(repo); err == nil || !strings.Contains(err.Error(), "unknown key") {
		t.Fatalf("unknown lock key: %v", err)
	}
	writeDoSkillsLock(t, repo, "https://github.com/edheltzel/Do-Skills.git", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	write(t, filepath.Join(repo, gapLockPath), `schema = "2"

[components.do-skills]
repo = "https://github.com/edheltzel/Do-Skills.git"
ref = "master"
commit = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
tre = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
`)
	if _, err := loadSkillPaths(repo); err == nil || !strings.Contains(err.Error(), "unknown key") {
		t.Fatalf("unknown component key: %v", err)
	}
	writeDoSkillsLock(t, repo, "https://github.com/edheltzel/Do-Skills.git", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	write(t, filepath.Join(repo, gapTOMLPath), `[components.do-skills]
repo = "https://github.com/edheltzel/Do-Skills.git"
ref = "master"
extra = "nope"
`)
	if _, err := loadSkillPaths(repo); err == nil || !strings.Contains(err.Error(), "unknown key") {
		t.Fatalf("unknown toml key: %v", err)
	}
	writeDoSkillsLock(t, repo, "https://github.com/edheltzel/Do-Skills.git", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	write(t, filepath.Join(repo, gapLockPath), `schema = "2"

[components.do-skills]
repo = "https://github.com/edheltzel/Do-Skills.git"
ref = "master"
commit = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
tree = "not-a-tree"
source = "edheltzel/Do-Skills"
`)
	if _, err := loadSkillPaths(repo); err == nil || !strings.Contains(err.Error(), "40-character lowercase hex SHA") {
		t.Fatalf("bad tree: %v", err)
	}
	write(t, filepath.Join(repo, gapLockPath), `schema = "2"

[components.do-skills]
repo = "https://github.com/edheltzel/Do-Skills.git"
ref = "master"
commit = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
tree = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
source = "edheltzel/Do-Skills"
`)
	if _, err := loadSkillPaths(repo); err != nil {
		t.Fatal(err)
	}
}

func TestComponentNameAndDuplicateSource(t *testing.T) {
	repo := t.TempDir()
	pin := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	url := "https://github.com/edheltzel/Do-Skills.git"
	writeComponent := func(name, lockExtra, tomlName string) {
		t.Helper()
		if tomlName == "" {
			tomlName = name
		}
		write(t, filepath.Join(repo, gapTOMLPath), fmt.Sprintf("[components.%s]\nrepo = %q\nref = \"master\"\n", tomlName, url))
		write(t, filepath.Join(repo, gapLockPath), fmt.Sprintf("schema = \"2\"\n\n[components.%s]\nrepo = %q\nref = \"master\"\ncommit = %q\n%s", name, url, pin, lockExtra))
	}
	writeComponent(`"../../x"`, "", `"../../x"`)
	if _, err := loadSkillPaths(repo); err == nil || !strings.Contains(err.Error(), "must match") {
		t.Fatalf("path name: %v", err)
	}
	writeComponent(`"do/skills"`, "", `"do/skills"`)
	if _, err := loadSkillPaths(repo); err == nil || !strings.Contains(err.Error(), "must match") {
		t.Fatalf("slash name: %v", err)
	}
	writeComponent("Do-Skills", "", "Do-Skills")
	if _, err := loadSkillPaths(repo); err == nil || !strings.Contains(err.Error(), "must match") {
		t.Fatalf("uppercase name: %v", err)
	}
	writeComponent("do_skills", "", "do_skills")
	if _, err := loadSkillPaths(repo); err == nil || !strings.Contains(err.Error(), "must match") {
		t.Fatalf("underscore name: %v", err)
	}
	write(t, filepath.Join(repo, gapTOMLPath), fmt.Sprintf(`[components.do-skills]
repo = %q
ref = "master"

[components.also-do]
repo = %q
ref = "master"
`, url, url))
	write(t, filepath.Join(repo, gapLockPath), fmt.Sprintf(`schema = "2"

[components.do-skills]
repo = %q
ref = "master"
commit = %q
source = "edheltzel/Do-Skills"

[components.also-do]
repo = %q
ref = "master"
commit = %q
source = "edheltzel/Do-Skills"
`, url, pin, url, pin))
	if _, err := loadSkillPaths(repo); err == nil || !strings.Contains(err.Error(), "duplicates source") {
		t.Fatalf("duplicate source: %v", err)
	}
	writeDoSkillsLock(t, repo, url, pin)
	if _, err := loadSkillPaths(repo); err != nil {
		t.Fatal(err)
	}
}

func TestLockField(t *testing.T) {
	repo := t.TempDir()
	url := "https://github.com/edheltzel/Do-Skills.git"
	pin := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	tree := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	writeDoSkillsLock(t, repo, url, pin)
	write(t, filepath.Join(repo, gapLockPath), fmt.Sprintf(`schema = "2"

[components.do-skills]
repo = %q
ref = "master"
commit = %q
tree = %q
`, url, pin, tree))
	got, err := lockField(repo, "do-skills", "commit")
	if err != nil || got != pin {
		t.Fatalf("commit: %q %v", got, err)
	}
	got, err = lockField(repo, "do-skills", "tree")
	if err != nil || got != tree {
		t.Fatalf("tree: %q %v", got, err)
	}
	got, err = lockField(repo, "do-skills", "source")
	if err != nil || got != "edheltzel/Do-Skills" {
		t.Fatalf("derived source: %q %v", got, err)
	}
	got, err = lockField(repo, "do-skills", "repo")
	if err != nil || got != url {
		t.Fatalf("repo: %q %v", got, err)
	}
	got, err = lockField(repo, "do-skills", "ref")
	if err != nil || got != "master" {
		t.Fatalf("ref: %q %v", got, err)
	}
	if _, err := lockField(repo, "do-skills", "nope"); err == nil || !strings.Contains(err.Error(), "unknown lock field") {
		t.Fatalf("unknown field: %v", err)
	}
	if _, err := lockField(repo, "missing", "commit"); err == nil || !strings.Contains(err.Error(), "is not in") {
		t.Fatalf("missing component: %v", err)
	}
	if _, err := lockField(repo, "../x", "commit"); err == nil || !strings.Contains(err.Error(), "must match") {
		t.Fatalf("bad name: %v", err)
	}
}

func TestSourceMatchIgnoresCaseAndGitSuffix(t *testing.T) {
	repo := t.TempDir()
	write(t, filepath.Join(repo, gapTOMLPath), `[components.do-skills]
repo = "https://github.com/edheltzel/do-skills.git"
ref = "master"
`)
	write(t, filepath.Join(repo, gapLockPath), `schema = "2"

[components.do-skills]
repo = "https://github.com/Edheltzel/Do-Skills.git"
ref = "master"
commit = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
`)
	c := mustDoSkillsComponent(t, repo)
	if c.Name != "do-skills" {
		t.Fatalf("case-insensitive lookup: %+v", c)
	}
	src, err := resolveInstallSource(repo, "edheltzel/Do-Skills")
	if err != nil {
		t.Fatal(err)
	}
	if src != componentDir(repo, "do-skills") {
		t.Fatalf("install source=%s", src)
	}
}

func TestComponentSubfolderSourceIsRejected(t *testing.T) {
	repo := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeDoSkillsLock(t, repo, "https://github.com/edheltzel/Do-Skills.git", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	npxLog := filepath.Join(t.TempDir(), "npx.log")
	npx := filepath.Join(t.TempDir(), "npx")
	write(t, npx, "#!/bin/sh\necho \"$*\" > "+npxLog+"\n")
	if err := os.Chmod(npx, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, src := range []string{
		"edheltzel/Do-Skills/skills/global",
		"https://github.com/edheltzel/Do-Skills/tree/master/skills",
	} {
		if _, err := resolveInstallSource(repo, src); err == nil || !strings.Contains(err.Error(), "subfolder") {
			t.Fatalf("%s: %v", src, err)
		}
		if skillsAdd(npx, repo, home, src, []string{"do-x"}) {
			t.Fatalf("%s must not install", src)
		}
	}
	if _, err := os.Stat(npxLog); !os.IsNotExist(err) {
		t.Fatalf("npx must not run, log=%q", readString(t, npxLog))
	}
	got, err := resolveInstallSource(repo, "backnotprop/plannotator/apps/skills/extra")
	if err != nil || got != "backnotprop/plannotator/apps/skills/extra" {
		t.Fatalf("unpinned subpath: %q %v", got, err)
	}
}

func TestMissingLockDoesNotFallBackToGitHub(t *testing.T) {
	repo := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)
	npxLog := filepath.Join(t.TempDir(), "npx.log")
	npx := filepath.Join(t.TempDir(), "npx")
	write(t, npx, "#!/bin/sh\necho \"$*\" > "+npxLog+"\n")
	if err := os.Chmod(npx, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := loadSkillPaths(repo); err == nil || !strings.Contains(err.Error(), "are missing") {
		t.Fatalf("missing lock files: %v", err)
	}
	if skillsAdd(npx, repo, home, "edheltzel/Do-Skills", []string{"do-x"}) {
		t.Fatal("missing lock must not install")
	}
	if _, err := os.Stat(npxLog); !os.IsNotExist(err) {
		t.Fatalf("npx must not run, log=%q", readString(t, npxLog))
	}
}

func TestComponentSourceWithoutLockEntry(t *testing.T) {
	repo := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)
	write(t, filepath.Join(repo, gapTOMLPath), `[components.do-skills]
repo = "https://github.com/edheltzel/Do-Skills.git"
ref = "master"
`)
	write(t, filepath.Join(repo, gapLockPath), "schema = \"2\"\n\n[components]\n")
	if _, err := loadSkillPaths(repo); err == nil || !strings.Contains(err.Error(), "not in") {
		t.Fatalf("toml component without lock entry: %v", err)
	}
	npxLog := filepath.Join(t.TempDir(), "npx.log")
	npx := filepath.Join(t.TempDir(), "npx")
	write(t, npx, "#!/bin/sh\necho \"$*\" > "+npxLog+"\n")
	if err := os.Chmod(npx, 0o755); err != nil {
		t.Fatal(err)
	}
	if skillsAdd(npx, repo, home, "edheltzel/Do-Skills", []string{"do-x"}) {
		t.Fatal("component-backed source without a lock entry must not install")
	}
	if _, err := os.Stat(npxLog); !os.IsNotExist(err) {
		t.Fatalf("npx must not run, log=%q", readString(t, npxLog))
	}
}

func writeDoSkillsLock(t *testing.T, repo, url, commit string) {
	t.Helper()
	write(t, filepath.Join(repo, gapTOMLPath), fmt.Sprintf(`[components.do-skills]
repo = %q
ref = "master"
`, url))
	write(t, filepath.Join(repo, gapLockPath), fmt.Sprintf(`schema = "2"

[components.do-skills]
repo = %q
ref = "master"
commit = %q
source = "edheltzel/Do-Skills"
`, url, commit))
}

func mustDoSkillsComponent(t *testing.T, repo string) component {
	t.Helper()
	paths, err := loadSkillPaths(repo)
	if err != nil {
		t.Fatal(err)
	}
	c, ok := paths.lookup("edheltzel/Do-Skills")
	if !ok {
		t.Fatal("lock missing do-skills")
	}
	return c
}

func writeEmptyLock(t *testing.T, repo string) {
	t.Helper()
	write(t, filepath.Join(repo, gapTOMLPath), "[components]\n")
	write(t, filepath.Join(repo, gapLockPath), "schema = \"2\"\n\n[components]\n")
}

func doSkillsBare(t *testing.T, pinBody, laterBody string) (bare, pin, later string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	src := t.TempDir()
	if _, err := gitRun(src, "init", "-q", "-b", "master"); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(src, "skills", "global", "core", "do-x", "SKILL.md"), pinBody)
	pin = commitGit(t, src, "pin")
	write(t, filepath.Join(src, "skills", "global", "core", "do-x", "SKILL.md"), laterBody)
	later = commitGit(t, src, "later")
	bare = filepath.Join(t.TempDir(), "do-skills.git")
	if out, err := exec.Command("git", "clone", "-q", "--bare", src, bare).CombinedOutput(); err != nil {
		t.Fatalf("bare clone: %v\n%s", err, out)
	}
	return bare, pin, later
}

func cloneGit(t *testing.T, url, dest string) error {
	t.Helper()
	cmd := exec.Command("git", "clone", "-q", url, dest)
	cmd.Env = gitEnv()
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("clone: %w\n%s", err, out)
	}
	return nil
}

func commitGit(t *testing.T, dir, msg string) string {
	t.Helper()
	if _, err := gitRun(dir, "add", "-A"); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("git", "-C", dir, "-c", "user.name=gap", "-c", "user.email=gap@example.com", "commit", "-q", "-m", msg)
	cmd.Env = gitEnv()
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("commit: %v\n%s", err, out)
	}
	sha, err := gitRun(dir, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	return sha
}
