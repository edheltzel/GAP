package main

import (
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// skillsShInstall writes a skill the way skills.sh left it: a folder under ~/.agents/skills and a relative Claude symlink to it.
func skillsShInstall(t *testing.T, home, name, body string) {
	t.Helper()
	dirs := skillDirs(home, name)
	write(t, filepath.Join(dirs[0], "SKILL.md"), body)
	if err := os.MkdirAll(filepath.Dir(dirs[1]), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join("..", "..", ".agents", "skills", name), dirs[1]); err != nil {
		t.Fatal(err)
	}
}

func skillLockNames(t *testing.T, home string) []string {
	t.Helper()
	lock, err := readSkillLock(home)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for name := range lock {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

func TestSyncMigratesSkillsShInstalls(t *testing.T) {
	home, repo, project, remote := t.TempDir(), t.TempDir(), t.TempDir(), t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_STATE_HOME", "")
	if _, err := git(project, "init", "-q", "-b", "master"); err != nil {
		t.Fatal(err)
	}
	t.Chdir(project)
	write(t, filepath.Join(remote, "skills", "other", "SKILL.md"), "pinned other\n")
	write(t, filepath.Join(remote, ".agent", "skills", "variant", "SKILL.md"), "wrong variant\n")
	write(t, filepath.Join(remote, ".agents", "skills", "variant", "SKILL.md"), "chosen variant\n")
	commit, tree := gitFixture(t, remote, "pinned\n")
	write(t, filepath.Join(repo, "gap.toml"), "schema = \"2\"\n[components.tools]\nrepo = \""+remote+"\"\nref = \"master\"\npath = \".agents/skills\"\nnames = [\"do-todo\", \"other\", \"variant\"]\n")
	if err := writeLock(repo, lockFile{Components: map[string]lockEntryM1{"tools": {Ref: "master", Commit: commit, Tree: tree}}}); err != nil {
		t.Fatal(err)
	}
	readyEnvReviewDistro(t, repo)
	skillsShInstall(t, home, "do-todo", "pinned\n")
	skillsShInstall(t, home, "variant", "chosen variant\n")
	skillsShInstall(t, home, "other", "edited by hand\n")
	entry := func(name string) string {
		return `"` + name + `":{"source":"owner/tools","sourceType":"github","sourceUrl":"https://github.com/owner/tools.git","skillPath":"skills/` + name + `/SKILL.md","skillFolderHash":"0000000000000000000000000000000000000000"}`
	}
	write(t, skillLockPath(home), `{"version":3,"skills":{`+entry("do-todo")+`,`+entry("variant")+`,`+entry("other")+`,"unrelated":{"source":"someone/else"}},"dismissed":{"x":true}}`)

	stdout, stderr := captureStd(t, func() {
		if code := run([]string{"sync", "--yes"}); code != 1 {
			t.Errorf("an edited skills.sh copy with no terminal must exit 1, got %d", code)
		}
	})
	for _, name := range []string{"do-todo", "variant"} {
		if !strings.Contains(stdout, "Migrated "+name+" from skills.sh (owner/tools) to tools") {
			t.Fatalf("missing migration line for %s:\n%s\n%s", name, stdout, stderr)
		}
		if info, err := os.Lstat(skillDirs(home, name)[1]); err != nil || info.Mode()&os.ModeSymlink == 0 {
			t.Fatalf("a matching copy of %s must be adopted in place, not reinstalled: %v", name, err)
		}
	}
	if strings.Contains(stdout, "Migrated other") || !strings.Contains(stderr, "Skill other was installed by skills.sh from owner/tools; take source moves it to tools") {
		t.Fatalf("an edited copy must go to drift review:\n%s\n%s", stdout, stderr)
	}
	if got := readString(t, filepath.Join(skillDirs(home, "other")[0], "SKILL.md")); got != "edited by hand\n" {
		t.Fatalf("drift with no terminal changed the edited copy: %q", got)
	}
	if got := skillLockNames(t, home); !slices.Equal(got, []string{"other", "unrelated"}) {
		t.Fatalf("lock after first sync = %v", got)
	}
	if !strings.Contains(readString(t, skillLockPath(home)), `"dismissed"`) {
		t.Fatal("migration dropped unrelated lock fields")
	}
	state, err := readState(home)
	if err != nil {
		t.Fatal(err)
	}
	if skills := state.Components["tools"].Skills; !slices.Contains(skills, "do-todo") || !slices.Contains(skills, "variant") {
		t.Fatalf("adopted skills must be owned by the component: %+v", state.Components["tools"])
	}

	input := filepath.Join(t.TempDir(), "answer")
	write(t, input, "t\n")
	in, err := os.Open(input)
	if err != nil {
		t.Fatal(err)
	}
	defer in.Close()
	oldIn, oldTerm := os.Stdin, termReady
	os.Stdin, termReady = in, func() bool { return true }
	t.Cleanup(func() { os.Stdin, termReady = oldIn, oldTerm })
	captureStd(t, func() {
		if code := run([]string{"sync", "--yes"}); code != 0 {
			t.Errorf("take source must finish the migration, got %d", code)
		}
	})
	for _, dir := range skillDirs(home, "other") {
		if got := readString(t, filepath.Join(dir, "SKILL.md")); got != "pinned other\n" {
			t.Fatalf("take source left %s = %q", dir, got)
		}
	}
	if got := skillLockNames(t, home); !slices.Equal(got, []string{"unrelated"}) {
		t.Fatalf("lock after take source = %v", got)
	}
}

func TestInstallSkillsAdoptsOnlyMatchingCopies(t *testing.T) {
	home, store := t.TempDir(), t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_STATE_HOME", filepath.Join(t.TempDir(), "state"))
	write(t, filepath.Join(store, "skills", "same", "SKILL.md"), "pinned\n")
	write(t, filepath.Join(store, "skills", "half", "SKILL.md"), "pinned\n")
	skillsShInstall(t, home, "same", "pinned\n")
	write(t, filepath.Join(skillDirs(home, "half")[0], "SKILL.md"), "pinned\n")
	write(t, filepath.Join(skillDirs(home, "half")[1], "SKILL.md"), "claude edit\n")
	write(t, skillLockPath(home), `{"skills":{"same":{"source":"a/b"},"half":{"source":"a/b"}}}`)
	entry := manifestEntry{Source: "github.com/a/b", Skills: []string{"half", "same"}, Path: store, Owner: "ab"}
	if !installSkills(home, []manifestEntry{entry}) {
		t.Fatal("install")
	}
	if got := skillLockNames(t, home); !slices.Equal(got, []string{"half"}) {
		t.Fatalf("only the matching copy may leave the XDG skills.sh lock: %v", got)
	}
	if got := readString(t, filepath.Join(skillDirs(home, "half")[1], "SKILL.md")); got != "claude edit\n" {
		t.Fatal("adoption touched a copy that differs from the pin")
	}
	write(t, skillLockPath(home), "not json")
	if !installSkills(home, []manifestEntry{entry}) || readString(t, skillLockPath(home)) != "not json" {
		t.Fatal("an unreadable lock is not a migration and must stay as it is")
	}
}

func TestFindSkillDirIsDeterministic(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{".agent/skills/x", ".agents/skills/x", "plugin/skills/x", "a/b/c/d/x"} {
		write(t, filepath.Join(root, filepath.FromSlash(dir), "SKILL.md"), dir+"\n")
	}
	if got := findSkillDir(root, "", "x"); got != filepath.Join(root, ".agent", "skills", "x") {
		t.Fatalf("shallowest match, then lexical order: %s", got)
	}
	if got := findSkillDir(root, ".agents/skills", "x"); got != filepath.Join(root, ".agents", "skills", "x") {
		t.Fatalf("component path must be searched first: %s", got)
	}
	write(t, filepath.Join(root, "z", "x", "SKILL.md"), "shallow\n")
	if got := findSkillDir(root, "", "x"); got != filepath.Join(root, "z", "x") {
		t.Fatalf("a shallower match must win over lexical order: %s", got)
	}
	if got := entrySkillDir(manifestEntry{Path: root, SkillPath: "plugin/skills"}, "x"); got != filepath.Join(root, "plugin", "skills", "x") {
		t.Fatalf("entry path: %s", got)
	}
}

func TestComponentPathKey(t *testing.T) {
	for _, tc := range []struct {
		value any
		want  string
		ok    bool
	}{
		{".agents/skills", ".agents/skills", true},
		{"./plugin//skills/", "plugin/skills", true},
		{"../outside", "", false},
		{"/abs", "", false},
		{"", "", false},
		{[]string{"x"}, "", false},
	} {
		c, err := parseComponent("x", map[string]any{"repo": "github.com/a/b", "ref": "main", "path": tc.value})
		if (err == nil) != tc.ok || c.SkillPath != tc.want {
			t.Fatalf("path %v: %+v %v", tc.value, c.SkillPath, err)
		}
	}
	if entries := selectedEntries(component{Skills: []string{"x"}, SkillPath: "p"}, "/store", ""); len(entries) != 1 || entries[0].SkillPath != "p" {
		t.Fatalf("selected entries must carry the component path: %+v", entries)
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
	if got != "29e5ce5cf8661ae41df2a1b7e962d79d17a4fef5509975d6e7445367957e0801" {
		t.Fatalf("content hash %s does not match skills.sh", got)
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
	skillsShInstall(t, home, "do-art", "art\n")
	write(t, filepath.Join(skillDirs(home, "do-art")[0], "Tools", "README.md"), "keep\n")
	entries := []manifestEntry{{Path: repo, Skills: []string{"do-art"}}}
	ok, err := checkDrift(home, entries)
	if err != nil || !ok {
		t.Fatalf("a do-art-like skill with node_modules must verify: ok=%v err=%v", ok, err)
	}
	dir := skillDirs(home, "do-art")[0]
	write(t, filepath.Join(dir, "Tools", "README.md"), "edit\n")
	ok, err = checkDrift(home, entries)
	if err != nil || ok || readString(t, filepath.Join(dir, "Tools", "README.md")) != "edit\n" {
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
			unsetEnv(t, "JCODE_HOME")
			warnings := shadowWarnings(home, tc.skill)
			got := strings.Join(warnings, "\n")
			if (got != "") != tc.warn || len(warnings) > 1 {
				t.Fatalf("warning = %q, want warn %v", got, tc.warn)
			}
			if tc.warn && (!strings.Contains(got, shadow) || !strings.Contains(got, filepath.Join(home, ".agents", "skills", tc.skill))) {
				t.Fatalf("warning must name both paths: %q", got)
			}
		})
	}
}

func TestShadowJcode(t *testing.T) {
	put := func(t *testing.T, dir, body string) {
		t.Helper()
		write(t, filepath.Join(dir, "SKILL.md"), body)
	}
	for _, tc := range []struct {
		name, shadow string
		env          bool
		warn         bool
	}{
		{"missing", "", false, false},
		{"matching", "same\n", false, false},
		{"differing", "edited\n", false, true},
		{"differing under JCODE_HOME", "edited\n", true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			put(t, filepath.Join(home, ".agents", "skills", "do-todo"), "same\n")
			base := filepath.Join(home, ".jcode")
			unsetEnv(t, "JCODE_HOME")
			if tc.env {
				base = filepath.Join(t.TempDir(), "jc")
				t.Setenv("JCODE_HOME", base)
				put(t, filepath.Join(home, ".jcode", "skills", "do-todo"), "ignored\n")
			}
			shadow := filepath.Join(base, "skills", "do-todo")
			if tc.shadow != "" {
				put(t, shadow, tc.shadow)
			}
			got := shadowWarnings(home, "do-todo")
			if (len(got) == 1) != tc.warn || len(got) > 1 {
				t.Fatalf("warnings = %q, want warn %v", got, tc.warn)
			}
			if tc.warn {
				want := "jcode skill copy " + shadow + " differs from the managed " + filepath.Join(home, ".agents", "skills", "do-todo")
				if got[0] != want {
					t.Fatalf("warning = %q, want %q", got[0], want)
				}
			}
		})
	}
}

func TestEmpryoShadowKeepsDriftResult(t *testing.T) {
	home, store := t.TempDir(), t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_STATE_HOME", "")
	write(t, filepath.Join(store, "skills", "ext", "SKILL.md"), "installed\n")
	for _, dir := range skillDirs(home, "ext") {
		write(t, filepath.Join(dir, "SKILL.md"), "installed\n")
	}
	write(t, filepath.Join(home, ".empryo", "skills", "ext", "SKILL.md"), "shadow edit\n")
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stderr
	os.Stderr = w
	ok, err := checkDrift(home, []manifestEntry{{Path: store, Skills: []string{"ext"}}})
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
		terminal, ok         bool
		warnings             int
	}{
		{"take source exposes shadow", "t", "old\n", true, true, 1},
		{"take source resolves shadow", "t", "new\n", true, true, 0},
		{"keep matching shadow", "k", "old\n", true, true, 0},
		{"keep differing shadow", "k", "new\n", true, true, 1},
		{"skip", "s", "new\n", true, false, 1},
		{"no terminal", "", "new\n", false, false, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home, repo := t.TempDir(), t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("XDG_STATE_HOME", "")
			write(t, filepath.Join(repo, "skills", "global", "core", "do-x", "SKILL.md"), "new\n")
			for _, dir := range skillDirs(home, "do-x") {
				write(t, filepath.Join(dir, "SKILL.md"), "old\n")
			}
			shadow := filepath.Join(home, ".empryo", "skills", "do-x", "SKILL.md")
			write(t, shadow, tc.shadow)
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
			ok, err := checkDrift(home, []manifestEntry{{Path: repo, Skills: []string{"do-x"}}})
			os.Stdin, os.Stderr, termReady = oldIn, oldErr, oldTerm
			if err != nil || ok != tc.ok {
				t.Fatalf("drift result: ok %v err %v", ok, err)
			}
			output := readString(t, out.Name())
			if count := strings.Count(output, "Empryo loads "); count != tc.warnings {
				t.Fatalf("shadow warnings = %d, want %d: %q", count, tc.warnings, output)
			}
			want := "old\n"
			if tc.choice == "t" {
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
