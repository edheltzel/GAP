package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseTOMLSubset(t *testing.T) {
	const hex = "0123456789abcdef0123456789abcdef01234567"
	doc := "" +
		"schema = \"2\"\n" +
		"name   = \"edheltzel/GAP\"\n" +
		"\n" +
		"[components.do-skills]\n" +
		"repo = \"github.com/edheltzel/Do-Skills\"\n" +
		"ref  = \"master\"\n" +
		"names = [\"do-todo\", \"do-commit\",]\n" +
		"\n" +
		"[components.recall]\n" +
		"repo = \"github.com/edheltzel/Recall\"\n" +
		"ref  = \"v0.10.2\"\n" +
		"names = [\"do-recall-search\"]\n"
	got, err := parseTOML(doc)
	if err != nil {
		t.Fatal(err)
	}
	if got["schema"] != "2" || got["name"] != "edheltzel/GAP" {
		t.Fatalf("root: %#v", got)
	}
	components := got["components"].(map[string]any)
	skills := components["do-skills"].(map[string]any)["names"].([]string)
	if len(skills) != 2 || skills[0] != "do-todo" || skills[1] != "do-commit" {
		t.Fatalf("skill names: %#v", skills)
	}
	lockDoc := "schema = \"2\"\n\n[components.do-skills]\nref = \"master\"\ncommit = \"" + hex + "\"\ntree = \"" + hex + "\"\n"
	locked, err := parseTOML(lockDoc)
	if err != nil {
		t.Fatal(err)
	}
	entry := locked["components"].(map[string]any)["do-skills"].(map[string]any)
	if entry["commit"] != hex || entry["tree"] != hex {
		t.Fatalf("lock: %#v", entry)
	}
	multi := "names = [\n  \"do-todo\",\n  \"do-commit\", # kept\n]\n"
	gotMulti, err := parseTOML(multi)
	if err != nil {
		t.Fatal(err)
	}
	names, ok := gotMulti["names"].([]string)
	if !ok || len(names) != 2 || names[1] != "do-commit" {
		t.Fatalf("multiline: %#v", gotMulti["names"])
	}
	quoted, err := parseTOML("[components.\"do-skills\"]\n\"weird.key\" = \"ok\"\n")
	if err != nil {
		t.Fatal(err)
	}
	if quoted["components"].(map[string]any)["do-skills"].(map[string]any)["weird.key"] != "ok" {
		t.Fatalf("quoted key: %#v", quoted)
	}
	dotted, err := parseTOML("[components.\"owner.repo\"]\nrepo = \"github.com/owner/repo\"\n")
	if err != nil {
		t.Fatal(err)
	}
	if dotted["components"].(map[string]any)["owner.repo"].(map[string]any)["repo"] != "github.com/owner/repo" {
		t.Fatalf("quoted header segment: %#v", dotted)
	}
	spaced, err := parseTOML("\"a b\".c = \"x\"\n")
	if err != nil {
		t.Fatal(err)
	}
	if spaced["a b"].(map[string]any)["c"] != "x" {
		t.Fatalf("quoted dotted key: %#v", spaced)
	}
	bad := []struct{ text, line string }{
		{"n = 1\n", "line 1"},
		{"d = 1979-05-27\n", "line 1"},
		{"s = \"\"\"\n", "line 1"},
		{"s = \"unterminated\n", "line 1"},
		{"s = \"bad\\q\"\n", "line 1"},
		{"[[arr]]\n", "line 1"},
		{"[a]\nx = \"1\"\n[a]\n", "line 3"},
		{"[a]\nx = \"1\"\nx = \"2\"\n", "line 3"},
		{"k = { a = { b = \"c\" } }\n", "line 1"},
		{"k = { a = \"b\" }\n", "line 1"},
		{"k = true\n", "line 1"},
		{"k = false\n", "line 1"},
		{"[components.bad key]\n", "line 1"},
		{"a.b. = \"x\"\n", "line 1"},
		{"\"bad\\q\" = \"x\"\n", "line 1"},
	}
	for _, c := range bad {
		_, err := parseTOML(c.text)
		if err == nil || !strings.Contains(err.Error(), c.line) {
			t.Errorf("%q: got %v, want %s", c.text, err, c.line)
		}
	}
}

func TestLockRoundTrip(t *testing.T) {
	const hex = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	const hex2 = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	lk := lockFile{Components: map[string]lockEntryM1{
		"zeta":      {Ref: "v1", Commit: hex2, Tree: hex2},
		"do-skills": {Ref: "master", Commit: hex, Tree: hex},
	}}
	root := t.TempDir()
	if err := writeLock(root, lk); err != nil {
		t.Fatal(err)
	}
	got := readString(t, filepath.Join(root, "gap.lock"))
	want := "schema = \"2\"\n\n[components.do-skills]\nref = \"master\"\ncommit = \"" + hex + "\"\ntree = \"" + hex + "\"\n\n[components.zeta]\nref = \"v1\"\ncommit = \"" + hex2 + "\"\ntree = \"" + hex2 + "\"\n"
	if got != want {
		t.Fatalf("lock bytes:\n%s", got)
	}
	if err := writeLock(root, lk); err != nil {
		t.Fatal(err)
	}
	if again := readString(t, filepath.Join(root, "gap.lock")); again != want {
		t.Fatal("lock write is not stable")
	}
	back, err := readLock(root)
	if err != nil || back.Components["do-skills"].Commit != hex || back.Components["zeta"].Ref != "v1" {
		t.Fatalf("read: %+v %v", back, err)
	}
}

func gitFixture(t *testing.T, dir, body string) (commit, tree string) {
	t.Helper()
	write(t, filepath.Join(dir, "skills", "global", "core", "do-todo", "SKILL.md"), body)
	run := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = gitEnv()
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	run("init", "-q", "-b", "master")
	run("-c", "user.name=gap", "-c", "user.email=gap@example.com", "add", "-A")
	run("-c", "user.name=gap", "-c", "user.email=gap@example.com", "commit", "-q", "-m", "skill")
	return run("rev-parse", "HEAD"), run("rev-parse", "HEAD^{tree}")
}

func TestEnsureStore(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	home := t.TempDir()
	remote := filepath.Join(t.TempDir(), "remote")
	commit, tree := gitFixture(t, remote, "todo\n")
	c := component{ID: "do-skills", Repo: remote, Ref: "master", Skills: []string{"do-todo"}}
	l := lockEntryM1{Ref: "master", Commit: commit, Tree: tree}
	dir, err := ensureStore(home, c, l)
	if err != nil {
		t.Fatal(err)
	}
	if got := readString(t, filepath.Join(dir, "skills", "global", "core", "do-todo", "SKILL.md")); got != "todo\n" {
		t.Fatalf("store body: %q", got)
	}
	if err := os.RemoveAll(remote); err != nil {
		t.Fatal(err)
	}
	again, err := ensureStore(home, c, l)
	if err != nil || again != dir {
		t.Fatalf("offline reuse: %s %v", again, err)
	}
	skill := filepath.Join(dir, "skills", "global", "core", "do-todo", "SKILL.md")
	if err := os.Chmod(skill, 0o644); err != nil {
		t.Fatal(err)
	}
	write(t, skill, "tampered\n")
	if _, err := ensureStore(home, c, l); err == nil {
		t.Fatal("tampered store must be rejected")
	}
}

func TestComponentSyncRestore(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_STATE_HOME", "")
	repo := t.TempDir()
	remote := filepath.Join(t.TempDir(), "remote")
	commit, tree := gitFixture(t, remote, "from-store\n")
	write(t, filepath.Join(repo, "gap.toml"), "schema = \"2\"\nname = \"edheltzel/GAP\"\n\n[components.do-skills]\nrepo = \""+remote+"\"\nref = \"master\"\nnames = [\"do-todo\"]\n")
	write(t, filepath.Join(repo, "gap.lock"), "schema = \"2\"\n\n[components.do-skills]\nref = \"master\"\ncommit = \""+commit+"\"\ntree = \""+tree+"\"\n")
	npx := filepath.Join(t.TempDir(), "npx")
	write(t, npx, `#!/bin/sh
src=
while [ $# -gt 0 ]; do
  if [ "$1" = add ]; then src=$2; fi
  if [ "$1" = -s ]; then
    mkdir -p "$HOME/.agents/skills/$2" "$HOME/.claude/skills/$2"
    cp "$src/skills/global/core/$2/SKILL.md" "$HOME/.agents/skills/$2/SKILL.md"
    cp "$src/skills/global/core/$2/SKILL.md" "$HOME/.claude/skills/$2/SKILL.md"
    chmod 644 "$HOME/.agents/skills/$2/SKILL.md" "$HOME/.claude/skills/$2/SKILL.md"
  fi
  shift
done
`)
	if err := os.Chmod(npx, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", filepath.Dir(npx)+string(os.PathListSeparator)+os.Getenv("PATH"))
	entries, err := loadComponents(repo, home)
	if err != nil || len(entries) != 1 || entries[0].Source != "." || entries[0].Path == "" {
		t.Fatalf("entries: %+v %v", entries, err)
	}
	if !installSkills(repo, home, entries) {
		t.Fatal("install")
	}
	for _, dest := range skillDirs(home, "do-todo") {
		if got := readString(t, filepath.Join(dest, "SKILL.md")); got != "from-store\n" {
			t.Fatalf("%s = %q", dest, got)
		}
	}
	agents := readString(t, filepath.Join(home, ".agents", "skills", "do-todo", "SKILL.md"))
	claude := readString(t, filepath.Join(home, ".claude", "skills", "do-todo", "SKILL.md"))
	write(t, filepath.Join(home, ".agents", "skills", "do-todo", "SKILL.md"), "drift\n")
	if err := os.RemoveAll(remote); err != nil {
		t.Fatal(err)
	}
	old := os.Args
	// restoreAll uses sourceRoot, so drive the pieces the command would.
	os.Args = old
	d, lk, err := readPins(repo)
	if err != nil {
		t.Fatal(err)
	}
	dir, err := ensureStore(home, d.Components["do-skills"], lk.Components["do-skills"])
	if err != nil {
		t.Fatal(err)
	}
	if _, err := replaceSkill(npx, repo, home, manifestEntry{Source: ".", Path: dir}, "do-todo"); err != nil {
		t.Fatal(err)
	}
	if got := readString(t, filepath.Join(home, ".agents", "skills", "do-todo", "SKILL.md")); got != agents {
		t.Fatalf("restore agents: %q", got)
	}
	if got := readString(t, filepath.Join(home, ".claude", "skills", "do-todo", "SKILL.md")); got != claude {
		t.Fatalf("restore claude: %q", got)
	}
	beforeAgents := readString(t, filepath.Join(home, ".agents", "skills", "do-todo", "SKILL.md"))
	beforeClaude := readString(t, filepath.Join(home, ".claude", "skills", "do-todo", "SKILL.md"))
	if !installSkills(repo, home, entries) {
		t.Fatal("second install")
	}
	if got := readString(t, filepath.Join(home, ".agents", "skills", "do-todo", "SKILL.md")); got != beforeAgents {
		t.Fatal("second sync changed agents")
	}
	if got := readString(t, filepath.Join(home, ".claude", "skills", "do-todo", "SKILL.md")); got != beforeClaude {
		t.Fatal("second sync changed claude")
	}
}

func TestStatusExit(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	repo := t.TempDir()
	write(t, filepath.Join(repo, "global", "AGENTS.md"), "policy\n")
	remote := filepath.Join(t.TempDir(), "remote")
	commit, tree := gitFixture(t, remote, "status\n")
	write(t, filepath.Join(repo, "gap.toml"), "schema = \"2\"\n[components.do-skills]\nrepo = \""+remote+"\"\nref = \"master\"\nnames = [\"do-todo\"]\n")
	write(t, filepath.Join(repo, "gap.lock"), "schema = \"2\"\n\n[components.do-skills]\nref = \"master\"\ncommit = \""+commit+"\"\ntree = \""+tree+"\"\n")
	sourceRootFor = repo
	t.Cleanup(func() { sourceRootFor = "" })
	if _, err := ensureStore(home, component{ID: "do-skills", Repo: remote, Ref: "master"}, lockEntryM1{Ref: "master", Commit: commit, Tree: tree}); err != nil {
		t.Fatal(err)
	}
	for _, dest := range skillDirs(home, "do-todo") {
		write(t, filepath.Join(dest, "SKILL.md"), "status\n")
	}
	if code := run([]string{"status"}); code != 0 {
		t.Fatalf("clean status: %d", code)
	}
	storeSkill := filepath.Join(storePath(home, "do-skills", commit), "skills", "global", "core", "do-todo", "SKILL.md")
	if err := os.Chmod(storeSkill, 0o644); err != nil {
		t.Fatal(err)
	}
	write(t, storeSkill, "tampered\n")
	write(t, filepath.Join(home, ".agents", "skills", "do-todo", "SKILL.md"), "tampered\n")
	write(t, filepath.Join(home, ".claude", "skills", "do-todo", "SKILL.md"), "tampered\n")
	if code := run([]string{"status"}); code != 1 {
		t.Fatalf("tampered store status: %d", code)
	}
	if _, err := ensureStore(home, component{ID: "do-skills", Repo: remote, Ref: "master"}, lockEntryM1{Ref: "master", Commit: commit, Tree: tree}); err != nil {
		t.Fatal(err)
	}
	for _, dest := range skillDirs(home, "do-todo") {
		write(t, filepath.Join(dest, "SKILL.md"), "status\n")
	}
	if code := run([]string{"status"}); code != 0 {
		t.Fatalf("refetched status: %d", code)
	}
	write(t, filepath.Join(home, ".agents", "skills", "do-todo", "SKILL.md"), "edited\n")
	if code := run([]string{"status"}); code != 1 {
		t.Fatalf("drift status: %d", code)
	}
	if err := os.RemoveAll(filepath.Join(home, ".gap", "store")); err != nil {
		t.Fatal(err)
	}
	if code := run([]string{"status"}); code != 1 {
		t.Fatalf("missing store status: %d", code)
	}
}

func TestPinWritesLock(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	repo := t.TempDir()
	remote := filepath.Join(t.TempDir(), "remote")
	commit, tree := gitFixture(t, remote, "pinned\n")
	write(t, filepath.Join(repo, "gap.toml"), "schema = \"2\"\n[components.do-skills]\nrepo = \""+remote+"\"\nref = \"master\"\nnames = [\"do-todo\"]\n")
	write(t, filepath.Join(repo, "global", "AGENTS.md"), "policy\n")
	sourceRootFor = repo
	t.Cleanup(func() { sourceRootFor = "" })
	if code := run([]string{"pin", "do-skills@master"}); code != 0 {
		t.Fatalf("pin: %d", code)
	}
	back, err := readLock(repo)
	if err != nil || back.Components["do-skills"].Commit != commit || back.Components["do-skills"].Tree != tree {
		t.Fatalf("pin lock: %+v %v", back, err)
	}
	if _, err := lockFor(distro{Components: map[string]component{"other": {ID: "other", Ref: "v1"}}}, back); err == nil || !strings.Contains(err.Error(), "gap pin other@v1") {
		t.Fatalf("missing pin: %v", err)
	}
}

func TestManifestGateFiles(t *testing.T) {
	for _, path := range []string{"gap.toml", "gap.lock", "global/AGENTS.md"} {
		if !ignoredPathBlocks(path) {
			t.Errorf("%s must block", path)
		}
	}
	if ignoredPathBlocks("notes.md") {
		t.Error("unrelated root file must not block")
	}
}

func TestGitHubComponentUsesStore(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_STATE_HOME", "")
	repo := t.TempDir()
	write(t, filepath.Join(repo, "global", "AGENTS.md"), "policy\n")
	remote := filepath.Join(t.TempDir(), "remote")
	commit, tree := gitFixture(t, remote, "from-store\n")
	write(t, filepath.Join(repo, "gap.toml"), "schema = \"2\"\nname = \"edheltzel/GAP\"\n\n[components.do-skills]\nrepo = \"github.com/edheltzel/Do-Skills\"\nref = \"master\"\nnames = [\"do-todo\"]\n")
	write(t, filepath.Join(repo, "gap.lock"), "schema = \"2\"\n\n[components.do-skills]\nref = \"master\"\ncommit = \""+commit+"\"\ntree = \""+tree+"\"\n")
	argv := filepath.Join(t.TempDir(), "argv")
	npx := filepath.Join(t.TempDir(), "npx")
	write(t, npx, componentNpx(argv))
	if err := os.Chmod(npx, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", filepath.Dir(npx)+string(os.PathListSeparator)+os.Getenv("PATH"))
	c := component{ID: "do-skills", Repo: remote, Ref: "master", Skills: []string{"do-todo"}}
	dir, err := ensureStore(home, c, lockEntryM1{Ref: "master", Commit: commit, Tree: tree})
	if err != nil {
		t.Fatal(err)
	}
	entry := manifestEntry{Source: "edheltzel/Do-Skills", Skills: []string{"do-todo"}, Path: dir}
	if !installSkills(repo, home, []manifestEntry{entry}) {
		t.Fatal("install")
	}
	got := readString(t, argv)
	if !strings.Contains(got, dir) || strings.Contains(got, "edheltzel/Do-Skills") {
		t.Fatalf("install argv:\n%s", got)
	}
	if err := os.WriteFile(argv, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	drift := filepath.Join(home, ".agents", "skills", "do-todo", "SKILL.md")
	if err := os.Chmod(drift, 0o644); err != nil {
		t.Fatal(err)
	}
	write(t, drift, "drift\n")
	sourceRootFor = repo
	t.Cleanup(func() { sourceRootFor = "" })
	if code := run([]string{"restore", "do-skills"}); code != 0 {
		t.Fatalf("restore: %d", code)
	}
	got = readString(t, argv)
	if !strings.Contains(got, dir) || strings.Contains(got, "edheltzel/Do-Skills") {
		t.Fatalf("restore argv:\n%s", got)
	}
}

func TestAnnotatedTagPeels(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	remote := filepath.Join(t.TempDir(), "remote")
	commit, _ := gitFixture(t, remote, "tagged\n")
	tag := exec.Command("git", "-C", remote, "-c", "user.name=gap", "-c", "user.email=gap@example.com", "tag", "-a", "v1", "-m", "annotated")
	tag.Env = gitEnv()
	if out, err := tag.CombinedOutput(); err != nil {
		t.Fatalf("tag: %v\n%s", err, out)
	}
	obj := exec.Command("git", "-C", remote, "rev-parse", "v1")
	obj.Env = gitEnv()
	tagObj, err := obj.Output()
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(tagObj)) == commit {
		t.Fatal("annotated tag must not be the commit")
	}
	got, err := resolveRef(remote, "v1")
	if err != nil || got != commit {
		t.Fatalf("peel: %s %v, want %s", got, err, commit)
	}
}

// componentNpx copies the named skill and writes the lock skills.sh writes for a local directory:
// source and sourceUrl are the -s install dir, sourceType is local, skillFolderHash is the content hash.
func componentNpx(argv string) string {
	return `#!/bin/sh
printf '%s\n' "$@" >> ` + argv + `
src=
while [ $# -gt 0 ]; do
  if [ "$1" = add ]; then src=$2; fi
  if [ "$1" = -s ]; then
    mkdir -p "$HOME/.agents/skills/$2" "$HOME/.claude/skills/$2"
    cp -Rp "$src/skills/global/core/$2/." "$HOME/.agents/skills/$2/"
    cp -Rp "$src/skills/global/core/$2/." "$HOME/.claude/skills/$2/"
    digest=$(GAP_TEST_CONTENT_HASH="$src/skills/global/core/$2" ` + shellQuote(os.Args[0]) + `) || exit 1
    python3 - "$HOME" "$2" "$src" "$digest" <<'PY'
import json, os, sys
home, name, src, digest = sys.argv[1:]
lock_path = os.path.join(home, ".agents", ".skill-lock.json")
os.makedirs(os.path.dirname(lock_path), exist_ok=True)
lock = {"skills": {}}
if os.path.exists(lock_path):
    lock = json.load(open(lock_path))
lock.setdefault("skills", {})[name] = {
    "source": src, "sourceType": "local", "sourceUrl": src, "skillFolderHash": digest,
}
json.dump(lock, open(lock_path, "w"))
PY
  fi
  shift
done
`
}

func TestGitHubStoreLockSecondSync(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_STATE_HOME", "")
	repo := t.TempDir()
	remote := filepath.Join(t.TempDir(), "remote")
	commit, tree := gitFixture(t, remote, "from-store\n")
	argv := filepath.Join(t.TempDir(), "argv")
	npx := filepath.Join(t.TempDir(), "npx")
	write(t, npx, componentNpx(argv))
	if err := os.Chmod(npx, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", filepath.Dir(npx)+string(os.PathListSeparator)+os.Getenv("PATH"))
	store, err := ensureStore(home, component{ID: "do-skills", Repo: remote, Ref: "master", Skills: []string{"do-todo"}}, lockEntryM1{Ref: "master", Commit: commit, Tree: tree})
	if err != nil {
		t.Fatal(err)
	}
	entry := manifestEntry{Source: "edheltzel/Do-Skills", Skills: []string{"do-todo"}, Path: store}
	if !installSkills(repo, home, []manifestEntry{entry}) {
		t.Fatal("first sync")
	}
	locked, err := readSkillLock(home)
	if err != nil {
		t.Fatal(err)
	}
	got := locked["do-todo"]
	if got.Source != store || got.SourceType != "local" || got.SourceURL != store {
		t.Fatalf("lock: %+v want source %s", got, store)
	}
	before := readString(t, filepath.Join(home, ".agents", ".skill-lock.json"))
	if err := os.WriteFile(argv, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if !installSkills(repo, home, []manifestEntry{entry}) {
		t.Fatal("second sync")
	}
	if calls := readString(t, argv); strings.TrimSpace(calls) != "" {
		t.Fatalf("second sync called skills.sh:\n%s", calls)
	}
	if after := readString(t, filepath.Join(home, ".agents", ".skill-lock.json")); after != before {
		t.Fatal("second sync migrated a store-path lock")
	}
}

func TestPinMovesStoreIdentity(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_STATE_HOME", "")
	repo := t.TempDir()
	remote := filepath.Join(t.TempDir(), "remote")
	first, tree := gitFixture(t, remote, "first\n")
	write(t, filepath.Join(remote, "skills", "global", "core", "do-todo", "SKILL.md"), "second\n")
	bump := exec.Command("git", "-C", remote, "-c", "user.name=gap", "-c", "user.email=gap@example.com", "commit", "-q", "-am", "second")
	bump.Env = gitEnv()
	if out, err := bump.CombinedOutput(); err != nil {
		t.Fatalf("commit: %v\n%s", err, out)
	}
	secondCommit, err := git(remote, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	secondTree, err := git(remote, "rev-parse", "HEAD^{tree}")
	if err != nil {
		t.Fatal(err)
	}
	old, err := ensureStore(home, component{ID: "do-skills", Repo: remote, Ref: "master", Skills: []string{"do-todo"}}, lockEntryM1{Ref: "master", Commit: first, Tree: tree})
	if err != nil {
		t.Fatal(err)
	}
	pinned, err := ensureStore(home, component{ID: "do-skills", Repo: remote, Ref: "master", Skills: []string{"do-todo"}}, lockEntryM1{Ref: "master", Commit: secondCommit, Tree: secondTree})
	if err != nil {
		t.Fatal(err)
	}
	for _, dir := range skillDirs(home, "do-todo") {
		write(t, filepath.Join(dir, "SKILL.md"), "first\n")
	}
	hash, err := contentHash(localSkillDir(old, "do-todo"))
	if err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(home, ".agents", ".skill-lock.json"), `{"skills":{"do-todo":{"source":"`+old+`","sourceType":"local","sourceUrl":"`+old+`","skillFolderHash":"`+hash+`"}}}`)
	entry := manifestEntry{Source: "edheltzel/Do-Skills", Skills: []string{"do-todo"}, Path: pinned}
	if _, foreign := foreignLockEntry(pinned, home, entry, "do-todo"); foreign {
		t.Fatal("an older commit of the same store is the same identity")
	}
	argv := filepath.Join(t.TempDir(), "argv")
	npx := filepath.Join(t.TempDir(), "npx")
	write(t, npx, componentNpx(argv))
	if err := os.Chmod(npx, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", filepath.Dir(npx)+string(os.PathListSeparator)+os.Getenv("PATH"))
	ok, err := checkDrift(repo, home, []manifestEntry{entry})
	if err != nil || ok {
		t.Fatalf("drift vs the new store must be ordinary drift, not a foreign lock: ok=%v err=%v", ok, err)
	}
	replaced, err := replaceSkill(npx, repo, home, entry, "do-todo")
	if err != nil || !replaced {
		t.Fatalf("reinstall: %v %v", replaced, err)
	}
	if !strings.Contains(readString(t, argv), pinned) {
		t.Fatal("reinstall did not use the pinned store")
	}
	if got := readString(t, filepath.Join(home, ".agents", "skills", "do-todo", "SKILL.md")); got != "second\n" {
		t.Fatalf("reinstall body: %q", got)
	}
}

func TestCheckoutLockMigrates(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_STATE_HOME", "")
	repo := t.TempDir()
	write(t, filepath.Join(repo, "global", "AGENTS.md"), "policy\n")
	remote := filepath.Join(t.TempDir(), "remote")
	commit, tree := gitFixture(t, remote, "from-store\n")
	store, err := ensureStore(home, component{ID: "do-skills", Repo: remote, Ref: "master", Skills: []string{"do-todo"}}, lockEntryM1{Ref: "master", Commit: commit, Tree: tree})
	if err != nil {
		t.Fatal(err)
	}
	for _, dir := range skillDirs(home, "do-todo") {
		write(t, filepath.Join(dir, "SKILL.md"), "from-store\n")
	}
	hash, err := contentHash(localSkillDir(store, "do-todo"))
	if err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(home, ".agents", ".skill-lock.json"), `{"skills":{"do-todo":{"source":"`+repo+`","sourceType":"local","sourceUrl":"`+repo+`","skillFolderHash":"`+hash+`"}}}`)
	argv := filepath.Join(t.TempDir(), "argv")
	npx := filepath.Join(t.TempDir(), "npx")
	write(t, npx, componentNpx(argv))
	if err := os.Chmod(npx, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", filepath.Dir(npx)+string(os.PathListSeparator)+os.Getenv("PATH"))
	entry := manifestEntry{Source: "edheltzel/Do-Skills", Skills: []string{"do-todo"}, Path: store}
	if !checkoutLock(store, repo) {
		t.Fatal("a GAP checkout lock must be migratable onto the store")
	}
	if _, foreign := foreignLockEntry(store, home, entry, "do-todo"); !foreign {
		t.Fatal("a checkout-path lock must stay foreign until the copy is migrated")
	}
	if !installSkills(repo, home, []manifestEntry{entry}) {
		t.Fatal("migration install")
	}
	if !strings.Contains(readString(t, argv), store) {
		t.Fatal("a checkout-path lock with a matching hash must migrate")
	}
	locked, err := readSkillLock(home)
	if err != nil || locked["do-todo"].Source != store {
		t.Fatalf("migrated lock: %+v %v", locked["do-todo"], err)
	}
}

func TestPlaceStoreKeepsPreviousOnFailure(t *testing.T) {
	root := t.TempDir()
	dest, tmp := filepath.Join(root, "commit"), filepath.Join(root, "fetch-1")
	write(t, filepath.Join(dest, "SKILL.md"), "verified\n")
	write(t, filepath.Join(tmp, "SKILL.md"), "fresh\n")
	old := storeRename
	t.Cleanup(func() { storeRename = old })
	storeRename = func(string, string) error { return os.ErrPermission }
	if err := placeStore(tmp, dest); !errors.Is(err, os.ErrPermission) {
		t.Fatalf("placeStore = %v", err)
	}
	if got := readString(t, filepath.Join(dest, "SKILL.md")); got != "verified\n" {
		t.Fatalf("previous store lost: %q", got)
	}
	if left, _ := filepath.Glob(dest + ".old-*"); len(left) != 0 {
		t.Fatalf("aside left behind: %v", left)
	}
	storeRename = old
	if err := placeStore(tmp, dest); err != nil {
		t.Fatal(err)
	}
	if got := readString(t, filepath.Join(dest, "SKILL.md")); got != "fresh\n" {
		t.Fatalf("store not replaced: %q", got)
	}
	if left, _ := filepath.Glob(dest + ".old-*"); len(left) != 0 {
		t.Fatalf("aside left behind: %v", left)
	}
}

func TestGitHubComponentRepoForms(t *testing.T) {
	url, err := cloneURL("github.com/edheltzel/Do-Skills")
	if err != nil || url != "https://github.com/edheltzel/Do-Skills.git" {
		t.Fatalf("cloneURL = %q, %v", url, err)
	}
	if got := componentSource("github.com/edheltzel/Do-Skills"); got != "edheltzel/Do-Skills" {
		t.Fatalf("componentSource = %q", got)
	}
	a, _, okA := githubIdentity("github.com/edheltzel/Do-Skills")
	b, _, okB := githubIdentity("https://github.com/edheltzel/Do-Skills.git")
	if !okA || !okB || a != b {
		t.Fatalf("host form %q and clone URL %q must be one identity", a, b)
	}
}
