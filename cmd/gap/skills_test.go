package main

import (
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestSkillsInstallPostconditions(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	npx := filepath.Join(t.TempDir(), "npx")
	write(t, npx, `#!/bin/sh
canonical=false
for arg in "$@"; do
  if [ "$arg" = codex ]; then canonical=true; fi
done
while [ $# -gt 0 ]; do
  if [ "$1" = -s ] && [ "$2" != typo ]; then
    if [ "$canonical" = true ]; then
      mkdir -p "$HOME/.agents/skills/$2" "$HOME/.claude/skills"
      echo installed > "$HOME/.agents/skills/$2/SKILL.md"
      ln -sfn "$HOME/.agents/skills/$2" "$HOME/.claude/skills/$2"
    else
      mkdir -p "$HOME/.claude/skills/$2"
      echo installed > "$HOME/.claude/skills/$2/SKILL.md"
    fi
  fi
  shift
done
`)
	if err := os.Chmod(npx, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_STATE_HOME", "")
	baseline := t.TempDir()
	write(t, filepath.Join(baseline, "SKILL.md"), "installed\n")
	hash, err := treeHash(baseline)
	if err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(home, ".agents", ".skill-lock.json"), `{"skills":{"good":{"source":"owner/repo","sourceUrl":"https://github.com/owner/repo.git","skillFolderHash":"`+hash+`"}}}`)
	if !skillsAdd(npx, t.TempDir(), home, "owner/repo", []string{"good"}) {
		t.Fatal("a complete matching install must succeed")
	}
	if skillsAdd(npx, t.TempDir(), home, "owner/repo", []string{"good", "typo"}) {
		t.Fatal("a successful subset must not satisfy all requested installations")
	}
	if !installed(home, "good") || installed(home, "typo") {
		t.Fatal("fixture must install exactly the matching subset")
	}
}

func TestSkillsReplacement(t *testing.T) {
	for _, mirrorLink := range []bool{false, true} {
		for _, mode := range []string{"failed", "incomplete", "mirror-mismatch", "complete"} {
			t.Run(mode+"-mirror-link-"+map[bool]string{false: "false", true: "true"}[mirrorLink], func(t *testing.T) {
				home := t.TempDir()
				t.Setenv("HOME", home)
				dirs := skillDirs(home, "do-x")
				write(t, filepath.Join(dirs[0], "SKILL.md"), "canonical edit\n")
				link, _ := filepath.Rel(filepath.Dir(dirs[1]), dirs[0])
				if mirrorLink {
					if err := os.MkdirAll(filepath.Dir(dirs[1]), 0o755); err != nil {
						t.Fatal(err)
					}
					if err := os.Symlink(link, dirs[1]); err != nil {
						t.Fatal(err)
					}
				} else {
					write(t, filepath.Join(dirs[1], "SKILL.md"), "mirror edit\n")
				}
				npx := filepath.Join(t.TempDir(), "npx")
				repo := t.TempDir()
				write(t, filepath.Join(repo, "skills", "global", "core", "do-x", "SKILL.md"), "replacement\n")
				body := "#!/bin/sh\nmkdir -p \"$HOME/.agents/skills/do-x\"\necho replacement > \"$HOME/.agents/skills/do-x/SKILL.md\"\n"
				switch mode {
				case "failed":
					body += "exit 1\n"
				case "complete":
					body += "ln -s \"$HOME/.agents/skills/do-x\" \"$HOME/.claude/skills/do-x\"\n"
				case "mirror-mismatch":
					body += "mkdir -p \"$HOME/.claude/skills/do-x\"\necho different > \"$HOME/.claude/skills/do-x/SKILL.md\"\n"
				}
				write(t, npx, body)
				if err := os.Chmod(npx, 0o755); err != nil {
					t.Fatal(err)
				}
				ok, err := replaceSkill(npx, repo, home, ".", "do-x")
				if err != nil || ok != (mode == "complete") {
					t.Fatalf("replace: ok=%v err=%v", ok, err)
				}
				if mode == "complete" {
					for _, dir := range dirs {
						if readString(t, filepath.Join(dir, "SKILL.md")) != "replacement\n" {
							t.Fatal("replacement did not reach both destinations")
						}
					}
					backups, _ := filepath.Glob(filepath.Join(home, ".gap", "backups", "*", ".agents", "skills", "do-x", "SKILL.md"))
					if len(backups) != 1 || readString(t, backups[0]) != "canonical edit\n" {
						t.Fatal("successful replacement must preserve the original backup")
					}
					return
				}
				if readString(t, filepath.Join(dirs[0], "SKILL.md")) != "canonical edit\n" {
					t.Fatal("failed replacement lost canonical edits")
				}
				if mirrorLink {
					if got, err := os.Readlink(dirs[1]); err != nil || got != link {
						t.Fatal("failed replacement did not restore the original mirror link")
					}
				} else if readString(t, filepath.Join(dirs[1], "SKILL.md")) != "mirror edit\n" {
					t.Fatal("failed replacement lost independent mirror edits")
				}
			})
		}
	}
}

func TestSkillsBaselines(t *testing.T) {
	for _, state := range []string{"missing", "unreadable", "malformed", "missing-entry", "missing-hash", "xdg", "local-missing"} {
		t.Run(state, func(t *testing.T) {
			home, repo := t.TempDir(), t.TempDir()
			t.Setenv("XDG_STATE_HOME", "")
			dirs := skillDirs(home, "do-x")
			for _, dir := range dirs {
				write(t, filepath.Join(dir, "SKILL.md"), "same\n")
			}
			hash, err := treeHash(dirs[0])
			if err != nil {
				t.Fatal(err)
			}
			lock := filepath.Join(home, ".agents", ".skill-lock.json")
			source := "owner/repo"
			switch state {
			case "unreadable":
				if err := os.MkdirAll(lock, 0o755); err != nil {
					t.Fatal(err)
				}
			case "malformed":
				write(t, lock, "{")
			case "missing-entry":
				write(t, lock, `{"skills":{}}`)
			case "missing-hash":
				write(t, lock, `{"skills":{"do-x":{}}}`)
			case "xdg":
				stateHome := filepath.Join(home, "state")
				t.Setenv("XDG_STATE_HOME", stateHome)
				write(t, filepath.Join(stateHome, "skills", ".skill-lock.json"), `{"skills":{"do-x":{"source":"owner/repo","sourceUrl":"https://github.com/owner/repo.git","skillFolderHash":"`+hash+`"}}}`)
			case "local-missing":
				source = "."
			}
			ok, err := checkDrift(repo, home, []manifestEntry{{Source: source, Skills: []string{"do-x"}}})
			if err != nil || ok != (state == "xdg") {
				t.Fatalf("baseline check: ok=%v err=%v", ok, err)
			}
		})
	}
}

func TestSyncReinstallsMismatchedSource(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "")
	for _, tc := range []struct {
		name, manifest, lockSource string
	}{
		{name: "fork", manifest: "kunchenguid/gh-axi", lockSource: "fork/gh-axi"},
		{name: "doskills", manifest: ".", lockSource: "edheltzel/Do-Skills"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home, repo := t.TempDir(), t.TempDir()
			t.Setenv("HOME", home)
			skill := "gh-axi"
			body := "from-checkout\n"
			if tc.manifest == "." {
				skill = "do-x"
				write(t, filepath.Join(repo, "skills", "global", "core", skill, "SKILL.md"), body)
			}
			for _, dir := range skillDirs(home, skill) {
				write(t, filepath.Join(dir, "SKILL.md"), body)
			}
			hash, err := treeHash(skillDirs(home, skill)[0])
			if err != nil {
				t.Fatal(err)
			}
			write(t, filepath.Join(home, ".agents", ".skill-lock.json"), `{"skills":{"`+skill+`":{"source":"`+tc.lockSource+`","sourceType":"github","sourceUrl":"https://github.com/`+tc.lockSource+`.git","skillPath":"skills/`+skill+`/SKILL.md","skillFolderHash":"`+hash+`"}}}`)
			npxDir := t.TempDir()
			npxLog := filepath.Join(npxDir, "npx.log")
			write(t, filepath.Join(npxDir, "npx"), "#!/bin/sh\necho \"$*\" >> "+npxLog+"\n"+`prev=""
src=""
for arg in "$@"; do
  if [ "$prev" = add ]; then src="$arg"; fi
  if [ "$prev" = -s ]; then
    mkdir -p "$HOME/.agents/skills/$arg" "$HOME/.claude/skills"
    printf '%s\n' 'from-checkout' > "$HOME/.agents/skills/$arg/SKILL.md"
    ln -sfn "$HOME/.agents/skills/$arg" "$HOME/.claude/skills/$arg"
    python3 - "$HOME" "$arg" "$src" <<'PY'
import json, sys
home, name, src = sys.argv[1:]
path = home + "/.agents/.skill-lock.json"
lock = json.load(open(path))
entry = lock["skills"][name]
entry["source"] = src
entry["sourceType"] = "local" if src.startswith("/") else "github"
entry["sourceUrl"] = src if src.startswith("/") else "https://github.com/" + src + ".git"
json.dump(lock, open(path, "w"))
PY
  fi
  prev="$arg"
done
`)
			if err := os.Chmod(filepath.Join(npxDir, "npx"), 0o755); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", npxDir+string(os.PathListSeparator)+os.Getenv("PATH"))
			old := os.Stdout
			r, w, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			os.Stdout = w
			ok := installSkills(repo, home, []manifestEntry{{Source: tc.manifest, Skills: []string{skill}}})
			w.Close()
			os.Stdout = old
			out, _ := io.ReadAll(r)
			if !ok {
				t.Fatalf("reinstall must succeed\n%s", out)
			}
			if _, err := os.Stat(npxLog); err != nil {
				t.Fatal("matching hash from another source must reinstall, not skip")
			}
			if !strings.Contains(string(out), "Migrated "+skill+"\n") && !strings.Contains(string(out), "Migrated "+skill+" ") {
				t.Fatalf("expected one migration line, got %q", out)
			}
			backups, _ := filepath.Glob(filepath.Join(home, ".gap", "backups", "*", ".agents", "skills", skill, "SKILL.md"))
			if len(backups) != 1 {
				t.Fatal("reinstall must back up the previous skill")
			}
			driftOK, err := checkDrift(repo, home, []manifestEntry{{Source: tc.manifest, Skills: []string{skill}}})
			if err != nil || !driftOK {
				t.Fatalf("after reinstall, drift check must pass: ok=%v err=%v", driftOK, err)
			}
		})
	}
}

func TestSourceIdentity(t *testing.T) {
	repo := "/tmp/gap-checkout"
	match := func(manifest string, l lockEntry) bool { return sameSource(repo, manifest, l) }
	if !match("kunchenguid/gh-axi", lockEntry{Source: "https://github.com/kunchenguid/gh-axi.git", SourceURL: "https://github.com/kunchenguid/gh-axi.git"}) {
		t.Fatal("a GitHub URL must match owner/repo")
	}
	if match("kunchenguid/gh-axi", lockEntry{Source: "fork/gh-axi", SourceURL: "https://github.com/fork/gh-axi.git"}) {
		t.Fatal("a fork must not match the declared repo")
	}
	if !match("backnotprop/plannotator/apps/skills/extra", lockEntry{Source: "backnotprop/plannotator", SourceURL: "https://github.com/backnotprop/plannotator.git", SkillPath: "apps/skills/extra/plannotator-compound/SKILL.md"}) {
		t.Fatal("a lock skillPath under the manifest subpath must match")
	}
	if match("backnotprop/plannotator/apps/skills/extra", lockEntry{Source: "backnotprop/plannotator", SkillPath: "skills/other/SKILL.md"}) {
		t.Fatal("a different subpath must not match")
	}
	if !match(".", lockEntry{Source: repo, SourceType: "local", SourceURL: repo}) {
		t.Fatal("a local lock path must match the checkout")
	}
	if match(".", lockEntry{Source: "edheltzel/Do-Skills", SourceType: "github", SourceURL: "https://github.com/edheltzel/Do-Skills.git"}) {
		t.Fatal("a GitHub lock must not match the checkout source")
	}
}

func TestContentHashMatchesSkillsSh(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "SKILL.md"), "---\nname: demo\ndescription: temp hash probe\n---\nhello\n")
	write(t, filepath.Join(dir, "a.txt"), "a\n")
	write(t, filepath.Join(dir, "Zed.txt"), "Z\n")
	write(t, filepath.Join(dir, "references.md"), "ref\n")
	write(t, filepath.Join(dir, "scripts", "run.sh"), "x\n")
	write(t, filepath.Join(dir, "nested", "note.txt"), "n\n")
	write(t, filepath.Join(dir, "node_modules", "pkg", "index.js"), "skip\n")
	write(t, filepath.Join(dir, ".git", "config"), "git\n")
	if err := os.Symlink("SKILL.md", filepath.Join(dir, "link.md")); err != nil {
		t.Fatal(err)
	}
	got, err := contentHash(dir)
	if err != nil {
		t.Fatal(err)
	}
	// skills 1.7.2 recorded this for the tree above via `npx skills add <dir> -g -y -s demo`.
	if got != "29e5ce5cf8661ae41df2a1b7e962d79d17a4fef5509975d6e7445367957e0801" {
		t.Fatalf("content hash %s does not match skills.sh", got)
	}
}

func TestSha256LockIsNotPermanentDrift(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_STATE_HOME", "")
	dirs := skillDirs(home, "ext")
	write(t, filepath.Join(dirs[0], "SKILL.md"), "installed\n")
	if err := os.MkdirAll(filepath.Dir(dirs[1]), 0o755); err != nil {
		t.Fatal(err)
	}
	link, err := filepath.Rel(filepath.Dir(dirs[1]), dirs[0])
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(link, dirs[1]); err != nil {
		t.Fatal(err)
	}
	hash, err := contentHash(dirs[0])
	if err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(home, ".agents", ".skill-lock.json"), `{"skills":{"ext":{"source":"owner/repo","sourceUrl":"https://github.com/owner/repo.git","skillFolderHash":"`+hash+`"}}}`)
	entries := []manifestEntry{{Source: "owner/repo", Skills: []string{"ext"}}}
	ok, err := checkDrift(t.TempDir(), home, entries)
	if err != nil || !ok {
		t.Fatalf("a skills.sh content hash must match the installed copy: ok=%v err=%v", ok, err)
	}
	write(t, filepath.Join(dirs[0], "SKILL.md"), "hand edit\n")
	ok, err = checkDrift(t.TempDir(), home, entries)
	if err != nil || ok || readString(t, filepath.Join(dirs[0], "SKILL.md")) != "hand edit\n" {
		t.Fatal("an edit against a content-hash baseline must be reported and left in place")
	}

	npx := filepath.Join(t.TempDir(), "npx")
	write(t, npx, "#!/bin/sh\nmkdir -p \"$HOME/.agents/skills/good\" \"$HOME/.claude/skills\"\necho installed > \"$HOME/.agents/skills/good/SKILL.md\"\nln -sfn \"$HOME/.agents/skills/good\" \"$HOME/.claude/skills/good\"\n")
	if err := os.Chmod(npx, 0o755); err != nil {
		t.Fatal(err)
	}
	baseline := t.TempDir()
	write(t, filepath.Join(baseline, "SKILL.md"), "installed\n")
	sum, err := contentHash(baseline)
	if err != nil {
		t.Fatal(err)
	}
	fresh := t.TempDir()
	t.Setenv("HOME", fresh)
	write(t, filepath.Join(fresh, ".agents", ".skill-lock.json"), `{"skills":{"good":{"source":"owner/repo","sourceUrl":"https://github.com/owner/repo.git","skillFolderHash":"`+sum+`"}}}`)
	if !skillsAdd(npx, t.TempDir(), fresh, "owner/repo", []string{"good"}) {
		t.Fatal("a content-hash baseline must accept a matching install")
	}
}

func TestForeignLockMigration(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "")
	for _, tc := range []struct {
		name, body, recorded string
		edited, symlink      bool
	}{
		{name: "edited", body: "hand edit\n", recorded: "from-old\n", edited: true},
		{name: "unverifiable", body: "hand edit\n", edited: true},
		{name: "unedited-symlink", body: "from-checkout\n", recorded: "from-checkout\n", symlink: true},
		{name: "unedited-sha256", body: "from-checkout\n", recorded: "from-checkout\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home, repo := t.TempDir(), t.TempDir()
			t.Setenv("HOME", home)
			skill := "do-x"
			write(t, filepath.Join(repo, "skills", "global", "core", skill, "SKILL.md"), "from-checkout\n")
			dirs := skillDirs(home, skill)
			write(t, filepath.Join(dirs[0], "SKILL.md"), tc.body)
			if tc.symlink {
				if err := os.MkdirAll(filepath.Dir(dirs[1]), 0o755); err != nil {
					t.Fatal(err)
				}
				link, err := filepath.Rel(filepath.Dir(dirs[1]), dirs[0])
				if err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(link, dirs[1]); err != nil {
					t.Fatal(err)
				}
			} else {
				write(t, filepath.Join(dirs[1], "SKILL.md"), tc.body)
			}
			hash := ""
			if tc.recorded != "" {
				baseline := t.TempDir()
				write(t, filepath.Join(baseline, "SKILL.md"), tc.recorded)
				var err error
				if tc.name == "unedited-sha256" {
					hash, err = contentHash(baseline)
				} else {
					hash, err = treeHash(baseline)
				}
				if err != nil {
					t.Fatal(err)
				}
			}
			write(t, filepath.Join(home, ".agents", ".skill-lock.json"), `{"skills":{"do-x":{"source":"edheltzel/Do-Skills","sourceType":"github","sourceUrl":"https://github.com/edheltzel/Do-Skills.git","skillPath":"skills/global/core/do-x/SKILL.md","skillFolderHash":"`+hash+`"}}}`)
			npxDir := t.TempDir()
			logPath := filepath.Join(npxDir, "npx.log")
			write(t, filepath.Join(npxDir, "npx"), "#!/bin/sh\necho called >> "+logPath+"\n"+`prev=""
src=""
for arg in "$@"; do
  if [ "$prev" = add ]; then src="$arg"; fi
  if [ "$prev" = -s ]; then
    mkdir -p "$HOME/.agents/skills/$arg" "$HOME/.claude/skills"
    printf '%s\n' 'from-checkout' > "$HOME/.agents/skills/$arg/SKILL.md"
    ln -sfn "$HOME/.agents/skills/$arg" "$HOME/.claude/skills/$arg"
    python3 - "$HOME" "$arg" "$src" <<'PY'
import json, sys
home, name, src = sys.argv[1:]
path = home + "/.agents/.skill-lock.json"
lock = json.load(open(path))
entry = lock["skills"][name]
entry["source"] = src
entry["sourceType"] = "local" if src.startswith("/") else "github"
entry["sourceUrl"] = src if src.startswith("/") else "https://github.com/" + src + ".git"
json.dump(lock, open(path, "w"))
PY
  fi
  prev="$arg"
done
`)
			if err := os.Chmod(filepath.Join(npxDir, "npx"), 0o755); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", npxDir+string(os.PathListSeparator)+os.Getenv("PATH"))
			installedOK := installSkills(repo, home, []manifestEntry{{Source: ".", Skills: []string{skill}}})
			_, called := os.Stat(logPath)
			if tc.edited {
				if !installedOK {
					t.Fatal("deferring an edited foreign skill must not fail install")
				}
				if called == nil {
					t.Fatal("an edited foreign skill must not be reinstalled")
				}
				for _, dir := range dirs {
					if readString(t, filepath.Join(dir, "SKILL.md")) != tc.body {
						t.Fatal("edited foreign skill was overwritten")
					}
				}
				ok, err := checkDrift(repo, home, []manifestEntry{{Source: ".", Skills: []string{skill}}})
				if err != nil || ok {
					t.Fatalf("non-interactive review must fail closed: ok=%v err=%v", ok, err)
				}
				for _, dir := range dirs {
					if readString(t, filepath.Join(dir, "SKILL.md")) != tc.body {
						t.Fatal("drift review overwrote the edit")
					}
				}
				return
			}
			if !installedOK || called != nil {
				t.Fatal("an unedited foreign skill must be migrated")
			}
			if readString(t, filepath.Join(dirs[0], "SKILL.md")) != "from-checkout\n" {
				t.Fatal("migration did not install the manifest source")
			}
			backups, _ := filepath.Glob(filepath.Join(home, ".gap", "backups", "*", ".agents", "skills", skill, "SKILL.md"))
			if len(backups) != 1 || readString(t, backups[0]) != tc.body {
				t.Fatal("migration must keep a backup of the previous copy")
			}
			ok, err := checkDrift(repo, home, []manifestEntry{{Source: ".", Skills: []string{skill}}})
			if err != nil || !ok {
				t.Fatalf("after migration, drift check must pass: ok=%v err=%v", ok, err)
			}
		})
	}
}

func TestMigrationRollbackRestoresLock(t *testing.T) {
	for _, xdg := range []bool{false, true} {
		t.Run(map[bool]string{false: "home", true: "xdg"}[xdg], func(t *testing.T) {
			home, repo := t.TempDir(), t.TempDir()
			t.Setenv("HOME", home)
			if xdg {
				t.Setenv("XDG_STATE_HOME", filepath.Join(home, "state"))
			} else {
				t.Setenv("XDG_STATE_HOME", "")
			}
			skill := "do-x"
			write(t, filepath.Join(repo, "skills", "global", "core", skill, "SKILL.md"), "from-checkout\n")
			dirs := skillDirs(home, skill)
			for _, dir := range dirs {
				write(t, filepath.Join(dir, "SKILL.md"), "old\n")
			}
			hash, err := treeHash(dirs[0])
			if err != nil {
				t.Fatal(err)
			}
			lockPath := skillLockPath(home)
			write(t, lockPath, `{"version":3,"skills":{"do-x":{"source":"edheltzel/Do-Skills","sourceType":"github","sourceUrl":"https://github.com/edheltzel/Do-Skills.git","skillFolderHash":"`+hash+`","installedAt":"2020-01-01T00:00:00.000Z","pluginName":"keep"},"other":{"source":"stay","pluginName":"untouched"}},"dismissed":{"findSkillsPrompt":true}}`+"\n")
			npxDir := t.TempDir()
			logPath := filepath.Join(npxDir, "npx.log")
			write(t, filepath.Join(npxDir, "npx"), "#!/bin/sh\necho called >> "+logPath+"\npython3 - "+lockPath+" "+skill+` <<'PY'
import json, sys
path, name = sys.argv[1:]
lock = json.load(open(path))
entry = lock["skills"][name]
entry["source"] = "/tmp/gap-checkout"
entry["sourceType"] = "local"
entry["skillFolderHash"] = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
entry.pop("pluginName", None)
json.dump(lock, open(path, "w"))
PY
exit 1
`)
			if err := os.Chmod(filepath.Join(npxDir, "npx"), 0o755); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", npxDir+string(os.PathListSeparator)+os.Getenv("PATH"))
			if installSkills(repo, home, []manifestEntry{{Source: ".", Skills: []string{skill}}}) {
				t.Fatal("a failed migration must not report success")
			}
			for _, dir := range dirs {
				if readString(t, filepath.Join(dir, "SKILL.md")) != "old\n" {
					t.Fatal("rollback must restore the skill bytes")
				}
			}
			assertLockRestored(t, lockPath, hash)
			if xdg {
				if _, err := os.Stat(filepath.Join(home, ".agents", ".skill-lock.json")); !os.IsNotExist(err) {
					t.Fatal("rollback wrote the home lock instead of XDG_STATE_HOME")
				}
			}
			if installSkills(repo, home, []manifestEntry{{Source: ".", Skills: []string{skill}}}) {
				t.Fatal("a second failed migration must still be a failure")
			}
			log, err := os.ReadFile(logPath)
			if err != nil || strings.Count(string(log), "called") != 2 {
				t.Fatalf("a restored foreign lock must be retried, log %q err %v", log, err)
			}
			assertLockRestored(t, lockPath, hash)
		})
	}
}

func assertLockRestored(t *testing.T, path, hash string) {
	t.Helper()
	var doc struct {
		Skills    map[string]map[string]any `json:"skills"`
		Dismissed map[string]bool           `json:"dismissed"`
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatal(err)
	}
	got := doc.Skills["do-x"]
	if got["source"] != "edheltzel/Do-Skills" || got["pluginName"] != "keep" || got["skillFolderHash"] != hash || got["installedAt"] != "2020-01-01T00:00:00.000Z" {
		t.Fatalf("lock entry was not restored: %#v", got)
	}
	if doc.Skills["other"]["source"] != "stay" || doc.Skills["other"]["pluginName"] != "untouched" || !doc.Dismissed["findSkillsPrompt"] {
		t.Fatalf("rollback changed the rest of the lock: %#v", doc)
	}
}

func TestForeignExternalDiff(t *testing.T) {
	fixture := t.TempDir()
	write(t, filepath.Join(fixture, "skills", "global", "core", "gh-axi", "SKILL.md"), "upstream\n")
	realGit, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	stub := t.TempDir()
	write(t, filepath.Join(stub, "git"), "#!/bin/sh\nif [ \"$1\" = clone ]; then\n  if [ \"$FAIL_CLONE\" = 1 ]; then exit 1; fi\n  dest=\n  for a in \"$@\"; do dest=\"$a\"; done\n  rm -rf \"$dest\"\n  mkdir -p \"$dest\"\n  cp -R "+fixture+"/. \"$dest\"/\n  exit 0\nfi\nexec "+realGit+" \"$@\"\n")
	if err := os.Chmod(filepath.Join(stub, "git"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", stub+string(os.PathListSeparator)+os.Getenv("PATH"))
	dir, cleanup := openSourceSkill("owner/gh-axi", "gh-axi")
	if cleanup != nil {
		defer cleanup()
	}
	if dir == "" {
		t.Fatal("an external foreign skill must have a comparison directory")
	}
	installed := t.TempDir()
	write(t, filepath.Join(installed, "SKILL.md"), "upstream\nhand\n")
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	showDiff(dir, lockEntry{}, installed)
	w.Close()
	os.Stdout = old
	out, _ := io.ReadAll(r)
	if !strings.Contains(string(out), "hand") {
		t.Fatalf("diff must show the edit take would replace, got %q", out)
	}

	t.Setenv("FAIL_CLONE", "1")
	errR, errW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	oldErr := os.Stderr
	os.Stderr = errW
	miss, missCleanup := openSourceSkill("missing/repo", "gh-axi")
	errW.Close()
	os.Stderr = oldErr
	msg, _ := io.ReadAll(errR)
	if missCleanup != nil {
		missCleanup()
	}
	if miss != "" || !strings.Contains(string(msg), "missing/repo") {
		t.Fatalf("a failed fetch must name the source take would replace, dir %q msg %q", miss, msg)
	}
}

func TestNodeModulesSkillVerifies(t *testing.T) {
	repo, home := t.TempDir(), t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_STATE_HOME", "")
	src := filepath.Join(repo, "skills", "global", "content", "do-art")
	write(t, filepath.Join(src, "SKILL.md"), "art\n")
	write(t, filepath.Join(src, "Tools", "README.md"), "keep\n")
	write(t, filepath.Join(src, "Tools", "node_modules", "pkg", "index.js"), "dep\n")
	write(t, filepath.Join(src, "Tools", "__pycache__", "x.pyc"), "pyc\n")
	write(t, filepath.Join(src, "metadata.json"), "meta\n")
	dirs := skillDirs(home, "do-art")
	write(t, filepath.Join(dirs[0], "SKILL.md"), "art\n")
	write(t, filepath.Join(dirs[0], "Tools", "README.md"), "keep\n")
	if err := os.MkdirAll(filepath.Dir(dirs[1]), 0o755); err != nil {
		t.Fatal(err)
	}
	link, err := filepath.Rel(filepath.Dir(dirs[1]), dirs[0])
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(link, dirs[1]); err != nil {
		t.Fatal(err)
	}
	entries := []manifestEntry{{Source: ".", Skills: []string{"do-art"}}}
	ok, err := checkDrift(repo, home, entries)
	if err != nil || !ok {
		t.Fatalf("a do-art-like skill with node_modules must verify: ok=%v err=%v", ok, err)
	}
	write(t, filepath.Join(dirs[0], "Tools", "README.md"), "edit\n")
	ok, err = checkDrift(repo, home, entries)
	if err != nil || ok || readString(t, filepath.Join(dirs[0], "Tools", "README.md")) != "edit\n" {
		t.Fatal("a copied file edit must still be drift and must be left in place")
	}
}

func TestEmpryoShadowWarning(t *testing.T) {
	put := func(t *testing.T, dir, body string) {
		t.Helper()
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct {
		name, skill, shadow string
		warn                bool
	}{
		{"no shadow", "do-todo", "", false},
		{"identical shadow", "do-todo", "same\n", false},
		{"differing shadow", "do-todo", "edited\n", true},
		{"empryo is ignored", "empryo", "edited\n", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			put(t, filepath.Join(home, ".agents", "skills", tc.skill), "same\n")
			shadow := filepath.Join(home, ".empryo", "skills", tc.skill)
			if tc.shadow != "" {
				put(t, shadow, tc.shadow)
			}
			got := empryoShadow(home, tc.skill)
			if (got != "") != tc.warn {
				t.Fatalf("warning = %q, want warn %v", got, tc.warn)
			}
			if tc.warn && (!strings.Contains(got, shadow) || !strings.Contains(got, filepath.Join(home, ".agents", "skills", tc.skill))) {
				t.Fatalf("warning must name both paths: %q", got)
			}
		})
	}
}

func TestEmpryoShadowKeepsDriftResult(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_STATE_HOME", "")
	dirs := skillDirs(home, "ext")
	write(t, filepath.Join(dirs[0], "SKILL.md"), "installed\n")
	write(t, filepath.Join(dirs[1], "SKILL.md"), "installed\n")
	hash, err := contentHash(dirs[0])
	if err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(home, ".agents", ".skill-lock.json"), `{"skills":{"ext":{"source":"owner/repo","sourceUrl":"https://github.com/owner/repo.git","skillFolderHash":"`+hash+`"}}}`)
	write(t, filepath.Join(home, ".empryo", "skills", "ext", "SKILL.md"), "shadow edit\n")
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stderr
	os.Stderr = w
	ok, err := checkDrift(t.TempDir(), home, []manifestEntry{{Source: "owner/repo", Skills: []string{"ext"}}})
	os.Stderr = old
	w.Close()
	out, _ := io.ReadAll(r)
	if err != nil || !ok {
		t.Fatalf("a shadow warning must not change the drift result: ok=%v err=%v", ok, err)
	}
	if n := strings.Count(string(out), "Empryo loads "); n != 1 {
		t.Fatalf("want exactly one shadow warning, got %d: %q", n, out)
	}
}

func TestEmpryoShadowAfterDriftResolution(t *testing.T) {
	for _, tc := range []struct {
		name, choice, shadow string
		terminal, fail, ok   bool
		warnings             int
	}{
		{"take source exposes shadow", "t", "old\n", true, false, true, 1},
		{"take source resolves shadow", "t", "new\n", true, false, true, 0},
		{"keep matching shadow", "k", "old\n", true, false, true, 0},
		{"keep differing shadow", "k", "new\n", true, false, true, 1},
		{"skip", "s", "new\n", true, false, false, 1},
		{"no terminal", "", "new\n", false, false, false, 1},
		{"failed repair restores matching shadow", "t", "old\n", true, true, false, 0},
		{"failed repair retains differing shadow", "t", "new\n", true, true, false, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home, repo, bin := t.TempDir(), t.TempDir(), t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("XDG_STATE_HOME", "")
			t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
			sourceDir := filepath.Join(repo, "skills", "global", "core", "do-x")
			write(t, filepath.Join(sourceDir, "SKILL.md"), "new\n")
			t.Setenv("GAP_TEST_SOURCE", sourceDir)
			for _, dir := range skillDirs(home, "do-x") {
				write(t, filepath.Join(dir, "SKILL.md"), "old\n")
			}
			shadow := filepath.Join(home, ".empryo", "skills", "do-x", "SKILL.md")
			write(t, shadow, tc.shadow)
			script := `#!/bin/sh
mkdir -p "$HOME/.agents/skills/do-x" "$HOME/.claude/skills/do-x"
cp "$GAP_TEST_SOURCE/SKILL.md" "$HOME/.agents/skills/do-x/SKILL.md"
cp "$GAP_TEST_SOURCE/SKILL.md" "$HOME/.claude/skills/do-x/SKILL.md"
`
			if tc.fail {
				script = "#!/bin/sh\nexit 1\n"
			}
			npx := filepath.Join(bin, "npx")
			write(t, npx, script)
			if err := os.Chmod(npx, 0o755); err != nil {
				t.Fatal(err)
			}
			input := filepath.Join(home, "answer")
			write(t, input, tc.choice+"\n")
			in, err := os.Open(input)
			if err != nil {
				t.Fatal(err)
			}
			defer in.Close()
			out, err := os.Create(filepath.Join(home, "stderr"))
			if err != nil {
				t.Fatal(err)
			}
			defer out.Close()
			oldIn, oldErr, oldTerm := os.Stdin, os.Stderr, termReady
			os.Stdin, os.Stderr = in, out
			termReady = func() bool { return tc.terminal }
			t.Cleanup(func() { os.Stdin, os.Stderr, termReady = oldIn, oldErr, oldTerm })
			ok, err := checkDrift(repo, home, []manifestEntry{{Source: ".", Skills: []string{"do-x"}}})
			os.Stdin, os.Stderr, termReady = oldIn, oldErr, oldTerm
			if err != nil || ok != tc.ok {
				t.Fatalf("drift result: ok %v err %v", ok, err)
			}
			output := readString(t, out.Name())
			if count := strings.Count(output, "Empryo loads "); count != tc.warnings {
				t.Fatalf("shadow warnings = %d, want %d: %q", count, tc.warnings, output)
			}
			want := "old\n"
			if tc.choice == "t" && !tc.fail {
				want = "new\n"
			}
			for _, dir := range skillDirs(home, "do-x") {
				if got := readString(t, filepath.Join(dir, "SKILL.md")); got != want {
					t.Fatalf("managed skill = %q, want %q", got, want)
				}
			}
			if readString(t, shadow) != tc.shadow {
				t.Fatal("drift resolution changed the Empryo shadow")
			}
		})
	}
}
