package main

import (
	"io"
	"os"
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
