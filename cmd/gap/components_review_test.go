package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestComponentExactPins(t *testing.T) {
	home, repo, remote := t.TempDir(), t.TempDir(), filepath.Join(t.TempDir(), "remote")
	t.Setenv("HOME", home)
	t.Setenv("XDG_STATE_HOME", "")
	first, firstTree := gitFixture(t, remote, "first\n")
	write(t, filepath.Join(remote, "skills", "global", "core", "do-todo", "SKILL.md"), "second\n")
	commands := [][]string{
		{"-c", "user.name=gap", "-c", "user.email=gap@example.com", "commit", "-qam", "second"},
		{"branch", "release/master"},
		{"branch", "release/only"},
		{"branch", "shared", first},
		{"tag", "shared"},
		{"branch", "same", first},
		{"tag", "same", first},
		{"-c", "user.name=gap", "-c", "user.email=gap@example.com", "tag", "-a", "release/v1", "-m", "nested"},
		{"-c", "user.name=gap", "-c", "user.email=gap@example.com", "tag", "-a", "v1", first, "-m", "exact"},
	}
	for _, args := range commands {
		if _, err := git(remote, args...); err != nil {
			t.Fatal(err)
		}
	}
	second, err := git(remote, "rev-parse", "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	secondTree, err := git(remote, "rev-parse", "HEAD^{tree}")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := git(remote, "update-ref", "refs/heads/master", first); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(repo, "gap.toml"), "schema = \"2\"\n[components.do-skills]\nrepo = \""+remote+"\"\nref = \"master\"\nnames = [\"do-todo\"]\n")
	if err := writeLock(repo, lockFile{Components: map[string]lockEntryM1{"do-skills": {Ref: "master", Commit: first, Tree: firstTree}}}); err != nil {
		t.Fatal(err)
	}
	sourceRootFor = repo
	t.Cleanup(func() { sourceRootFor = "" })
	for _, tc := range []struct {
		ref, commit, tree string
	}{
		{"master", first, firstTree},
		{"HEAD", first, firstTree},
		{"refs/heads/master", first, firstTree},
		{"release/master", second, secondTree},
		{"v1", first, firstTree},
		{"refs/tags/v1", first, firstTree},
		{"release/v1", second, secondTree},
		{"refs/heads/shared", first, firstTree},
		{"refs/tags/shared", second, secondTree},
		{first, first, firstTree},
		{"shared", "", ""},
		{"same", "", ""},
		{"only", "", ""},
		{"mast*", "", ""},
	} {
		t.Run(tc.ref, func(t *testing.T) {
			before := readString(t, filepath.Join(repo, "gap.lock"))
			code := run([]string{"pin", "do-skills@" + tc.ref})
			if tc.commit == "" {
				if code != 1 || readString(t, filepath.Join(repo, "gap.lock")) != before {
					t.Fatalf("invalid or ambiguous ref changed the pin: code=%d", code)
				}
				return
			}
			lk, err := readLock(repo)
			want := lockEntryM1{Ref: tc.ref, Commit: tc.commit, Tree: tc.tree}
			if code != 0 || err != nil || lk.Components["do-skills"] != want {
				t.Fatalf("pin: code=%d lock=%+v err=%v, want %+v", code, lk, err, want)
			}
			stdout, _ := captureStd(t, func() { run([]string{"status"}) })
			if !strings.Contains(stdout, "do-skills ref="+tc.ref+" commit="+tc.commit[:7]+" store=") {
				t.Fatalf("status must report the locked ref: %s", stdout)
			}
		})
	}
}

func TestComponentInputTypes(t *testing.T) {
	const hex = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	const distroFields = "repo = \"github.com/owner/repo\"\nref = \"master\"\nnames = [\"do-todo\"]\n"
	const lockFields = "ref = \"master\"\ncommit = \"" + hex + "\"\ntree = \"" + hex + "\"\n"
	for _, tc := range []struct {
		name, text, field string
		lock              bool
	}{
		{"distro schema type", "schema = [\"2\"]\n", "schema", false},
		{"distro schema missing", "name = \"gap\"\n", "schema", false},
		{"distro name type", "schema = \"2\"\nname = [\"gap\"]\n", "name", false},
		{"distro components type", "schema = \"2\"\ncomponents = \"bad\"\n", "components", false},
		{"distro component type", "schema = \"2\"\n[components]\nbad = [\"bad\"]\n", "bad", false},
		{"distro repo type", "schema = \"2\"\n[components.bad]\n" + strings.Replace(distroFields, "repo = \"github.com/owner/repo\"", "repo = [\"github.com/owner/repo\"]", 1), "repo", false},
		{"distro ref type", "schema = \"2\"\n[components.bad]\n" + strings.Replace(distroFields, "ref = \"master\"", "ref = [\"master\"]", 1), "ref", false},
		{"distro names type", "schema = \"2\"\n[components.bad]\n" + strings.Replace(distroFields, "names = [\"do-todo\"]", "names = \"do-todo\"", 1), "names", false},
		{"distro names table", "schema = \"2\"\n[components.bad]\n" + strings.Replace(distroFields, "names = [\"do-todo\"]", "[components.bad.names]\ndo-todo = \"always\"", 1), "names", false},
		{"distro names missing", "schema = \"2\"\n[components.bad]\nrepo = \"github.com/owner/repo\"\nref = \"master\"\n", "names", false},
		{"lock schema type", "schema = [\"2\"]\n", "schema", true},
		{"lock schema missing", "[components.bad]\n" + lockFields, "schema", true},
		{"lock components type", "schema = \"2\"\ncomponents = \"bad\"\n", "components", true},
		{"lock component type", "schema = \"2\"\n[components]\nbad = [\"bad\"]\n", "bad", true},
		{"lock ref type", "schema = \"2\"\n[components.bad]\n" + strings.Replace(lockFields, "ref = \"master\"", "ref = [\"master\"]", 1), "ref", true},
		{"lock ref empty", "schema = \"2\"\n[components.bad]\n" + strings.Replace(lockFields, "ref = \"master\"", "ref = \"\"", 1), "ref", true},
		{"lock commit type", "schema = \"2\"\n[components.bad]\n" + strings.Replace(lockFields, "commit = \""+hex+"\"", "commit = [\""+hex+"\"]", 1), "commit", true},
		{"lock tree type", "schema = \"2\"\n[components.bad]\n" + strings.Replace(lockFields, "tree = \""+hex+"\"", "tree = [\""+hex+"\"]", 1), "tree", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			var err error
			if tc.lock {
				write(t, filepath.Join(root, "gap.lock"), tc.text)
				_, err = readLock(root)
			} else {
				write(t, filepath.Join(root, "gap.toml"), tc.text)
				_, err = readDistro(root)
			}
			if err == nil || !strings.Contains(err.Error(), tc.field) {
				t.Fatalf("malformed %s must name %s: %v", tc.name, tc.field, err)
			}
		})
	}
	root := t.TempDir()
	write(t, filepath.Join(root, "gap.toml"), "schema = \"2\"\nname = \"gap\"\n[components.good]\n"+distroFields)
	d, err := readDistro(root)
	if err != nil || d.Name != "gap" || len(d.Components["good"].Skills) != 1 || d.Components["good"].Skills[0] != "do-todo" {
		t.Fatalf("valid distro: %+v %v", d, err)
	}
	write(t, filepath.Join(root, "gap.toml"), "schema = \"2\"\n[components.good]\n"+distroFields+"[components.good.skills]\nother = \"always\"\n")
	d, err = readDistro(root)
	if err == nil || !strings.Contains(err.Error(), "other") {
		t.Fatalf("a skill entry must be a table, not a string: %+v %v", d, err)
	}
	write(t, filepath.Join(root, "gap.toml"), "schema = \"2\"\n[components.good]\n"+distroFields+"[components.good.skills.do-todo]\nload = \"always\"\n")
	d, err = readDistro(root)
	if err != nil || d.Components["good"].SkillSet["do-todo"].Load != "always" {
		t.Fatalf("a skill table sets its load: %+v %v", d, err)
	}
}

func TestComponentLocalModeRestore(t *testing.T) {
	for _, repoForm := range []string{"absolute", "file"} {
		for _, mode := range []os.FileMode{0o644, 0o755} {
			t.Run(repoForm+"/"+mode.String(), func(t *testing.T) {
				home, repo, remote := t.TempDir(), t.TempDir(), filepath.Join(t.TempDir(), "remote")
				t.Setenv("HOME", home)
				t.Setenv("XDG_STATE_HOME", "")
				gitFixture(t, remote, "pinned\n")
				rel := filepath.Join("Tools", "run.sh")
				script := filepath.Join(remote, "skills", "global", "core", "do-todo", rel)
				write(t, script, "#!/bin/sh\nexit 0\n")
				if err := os.Chmod(script, mode); err != nil {
					t.Fatal(err)
				}
				for _, args := range [][]string{{"add", "-A"}, {"-c", "user.name=gap", "-c", "user.email=gap@example.com", "commit", "-qm", "script"}} {
					if _, err := git(remote, args...); err != nil {
						t.Fatal(err)
					}
				}
				commit, err := git(remote, "rev-parse", "HEAD")
				if err != nil {
					t.Fatal(err)
				}
				tree, err := git(remote, "rev-parse", "HEAD^{tree}")
				if err != nil {
					t.Fatal(err)
				}
				source := remote
				if repoForm == "file" {
					source = "file://" + remote
				}
				write(t, filepath.Join(repo, "gap.toml"), "schema = \"2\"\n[components.do-skills]\nrepo = \""+source+"\"\nref = \"master\"\nnames = [\"do-todo\"]\n")
				if err := writeLock(repo, lockFile{Components: map[string]lockEntryM1{"do-skills": {Ref: "master", Commit: commit, Tree: tree}}}); err != nil {
					t.Fatal(err)
				}
				argv, npx := filepath.Join(t.TempDir(), "argv"), filepath.Join(t.TempDir(), "npx")
				write(t, npx, componentNpx(argv))
				if err := os.Chmod(npx, 0o755); err != nil {
					t.Fatal(err)
				}
				t.Setenv("PATH", filepath.Dir(npx)+string(os.PathListSeparator)+os.Getenv("PATH"))
				sourceRootFor = repo
				t.Cleanup(func() { sourceRootFor = "" })
				entries, err := loadTestComponents(repo, home, repo)
				if err != nil || !installSkills(repo, home, entries) {
					t.Fatalf("install: %v", err)
				}
				for i, dest := range skillDirs(home, "do-todo") {
					if code := run([]string{"status"}); code != 0 {
						t.Fatalf("clean status: %d", code)
					}
					changed := mode ^ 0o111
					if err := os.Chmod(filepath.Join(dest, rel), changed); err != nil {
						t.Fatal(err)
					}
					stdout, _ := captureStd(t, func() {
						if code := run([]string{"status"}); code != 1 {
							t.Errorf("mode drift status: %d", code)
						}
					})
					if !strings.Contains(stdout, "do-todo drift=1") {
						t.Errorf("mode drift in %s must be counted: %s", dest, stdout)
					}
					if ok, err := checkDrift(repo, home, entries); err != nil || ok {
						t.Errorf("sync must see mode drift: ok=%v err=%v", ok, err)
					}
					args := []string{"restore"}
					if i == 0 {
						args = append(args, "do-skills")
					}
					if code := run(args); code != 0 {
						t.Errorf("mode restore: %d", code)
					}
					for _, dir := range skillDirs(home, "do-todo") {
						info, err := os.Stat(filepath.Join(dir, rel))
						if err != nil || info.Mode().Perm() != mode {
							t.Fatalf("restored mode in %s: %v %v, want %v", dir, info, err, mode)
						}
					}
				}
				write(t, filepath.Join(skillDirs(home, "do-todo")[0], "SKILL.md"), "edited\n")
				if code := run([]string{"restore", "do-skills"}); code != 0 {
					t.Fatalf("content restore with local store lock: %d", code)
				}
				for _, dir := range skillDirs(home, "do-todo") {
					if body := readString(t, filepath.Join(dir, "SKILL.md")); body != "pinned\n" {
						t.Fatalf("restored body: %q", body)
					}
				}
				if ok, err := checkDrift(repo, home, entries); err != nil || !ok {
					t.Fatalf("restored sync: ok=%v err=%v", ok, err)
				}
				write(t, argv, "")
				if code := run([]string{"restore"}); code != 0 || readString(t, argv) != "" {
					t.Fatal("clean restore must skip installation")
				}
			})
		}
	}
}
