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
	if err := ensureComponent(repo, c); err != nil {
		t.Fatal(err)
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
	c, ok := paths.bySource["edheltzel/Do-Skills"]
	if !ok {
		t.Fatal("lock missing do-skills")
	}
	return c
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
