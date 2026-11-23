package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestParseAddArgs(t *testing.T) {
	for _, tc := range []struct {
		args           []string
		source         string
		agents, skills []string
		global, yes    bool
		err            string
	}{
		{args: []string{"o/r", "-a", "claude-code", "cursor", "-s", "x"}, source: "o/r", agents: []string{"claude-code", "cursor"}, skills: []string{"x"}},
		{args: []string{"o/r", "--copy=true"}, err: "--copy does not take a value"},
		{args: []string{"o/r", "-a", "-y"}, err: "-a requires an agent name"},
		{args: []string{"o/r", "--all", "-s", "x"}, source: "o/r", agents: []string{"*"}, skills: []string{"*"}, yes: true},
		{args: []string{"o/r", "--bogus"}, source: "o/r"},
		{args: []string{"--", "-g"}, source: "-g"},
		{args: []string{"-gy", "o/r", "-a=codex"}, source: "o/r", agents: []string{"codex"}, global: true, yes: true},
		{args: []string{"o/r", "-g", "--local"}, err: "--global and --project cannot be used together"},
	} {
		t.Run(strings.Join(tc.args, " "), func(t *testing.T) {
			opts, err := parseAddArgs(tc.args)
			if tc.err != "" {
				if err == nil || err.Error() != tc.err {
					t.Fatalf("err = %v, want %q", err, tc.err)
				}
				return
			}
			if err != nil || opts.Source != tc.source || !slices.Equal(opts.Agents, tc.agents) || !slices.Equal(opts.Skills, tc.skills) || opts.Global != tc.global || opts.Yes != tc.yes {
				t.Fatalf("opts = %+v err = %v", opts, err)
			}
		})
	}
}

func TestParseSkillSource(t *testing.T) {
	for _, tc := range []struct {
		input                          string
		kind, url, ref, subpath, skill string
		err                            string
	}{
		{input: "owner/repo@v1.2.3", kind: "github", url: "https://github.com/owner/repo.git", ref: "v1.2.3"},
		{input: "owner/repo@main", kind: "github", url: "https://github.com/owner/repo.git", skill: "main"},
		{input: "owner/repo#dev@x", kind: "github", url: "https://github.com/owner/repo.git", ref: "dev", skill: "x"},
		{input: "https://github.com/o/r/tree/main/skills/a", kind: "github", url: "https://github.com/o/r.git", ref: "main", subpath: "skills/a"},
		{input: "https://gitlab.com/g/s/r/-/tree/v2/x", kind: "gitlab", url: "https://gitlab.com/g/s/r.git", ref: "v2", subpath: "x"},
		{input: "https://dev.azure.com/o/p/_git/r?path=/s&version=GBmain", kind: "git", url: "https://dev.azure.com/o/p/_git/r", ref: "main", subpath: "s"},
		{input: "git@github.com:o/r.git", kind: "git", url: "git@github.com:o/r.git"},
		{input: "https://example.com/skills", err: "gap add installs from git repositories and local folders; https://example.com/skills is a well-known source"},
		{input: "https://raw.githubusercontent.com/o/r/main/SKILL.md", err: "gap add installs from git repositories and local folders; https://raw.githubusercontent.com/o/r/main/SKILL.md is a download source"},
		{input: "owner/repo/../x", err: `Unsafe subpath: "../x" contains path traversal segments. Subpaths must not contain ".." components.`},
	} {
		t.Run(tc.input, func(t *testing.T) {
			got, err := parseSkillSource(tc.input, "/cwd")
			if tc.err != "" {
				if err == nil || err.Error() != tc.err {
					t.Fatalf("err = %v, want %q", err, tc.err)
				}
				return
			}
			want := skillSource{Kind: tc.kind, URL: tc.url, Ref: tc.ref, Subpath: tc.subpath, Skill: tc.skill}
			if err != nil || got != want {
				t.Fatalf("got %+v %v, want %+v", got, err, want)
			}
		})
	}
	t.Setenv("GH_HOST", "ghe.example:443")
	if got, err := parseSkillSource("owner/repo", "/cwd"); err != nil || got.URL != "https://ghe.example/owner/repo.git" {
		t.Fatalf("GH_HOST with the default HTTPS port = %+v %v", got, err)
	}
}

func webDesignRemote(t *testing.T, e *utilityEnv) {
	t.Helper()
	if err := os.RemoveAll(filepath.Join(e.remote, "skills", "util-skill")); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(e.remote, "skills", "web", "SKILL.md"), "---\nname: Web Design\ndescription: Builds web pages\n---\nBody\n")
	write(t, filepath.Join(e.remote, "skills", "web", "ref.md"), "reference\n")
	if err := os.Symlink("ref.md", filepath.Join(e.remote, "skills", "web", "link.md")); err != nil {
		t.Fatal(err)
	}
	commitAll(t, e.remote)
}

func TestAddAnyRepoProjectSymlink(t *testing.T) {
	e := newUtilityEnv(t, "")
	webDesignRemote(t, e)
	if _, err := git(e.project, "init", "-q"); err != nil {
		t.Fatal(err)
	}
	root, err := filepath.EvalSymlinks(e.project)
	if err != nil {
		t.Fatal(err)
	}
	quietStdout(t)
	if code := run([]string{"add", e.remote, "-y", "-a", "claude-code", "-a", "codex"}); code != 0 {
		t.Fatalf("add exited %d", code)
	}
	if code := run([]string{"add", e.remote, "-y", "-a", "cursor"}); code != 0 {
		t.Fatalf("re-add exited %d", code)
	}
	canonical := filepath.Join(root, ".agents", "skills", "web-design")
	for _, rel := range []string{"SKILL.md", "link.md"} {
		if info, err := os.Lstat(filepath.Join(canonical, rel)); err != nil || !info.Mode().IsRegular() {
			t.Fatalf("%s is not a regular file: %v", rel, err)
		}
	}
	if target, err := os.Readlink(filepath.Join(root, ".claude", "skills", "web-design")); err != nil || target != filepath.Join("..", "..", ".agents", "skills", "web-design") {
		t.Fatalf("claude link = %q %v", target, err)
	}
	project := readConfigComponents(t, projectConfigFile(root))["remote"].(map[string]any)
	if !slices.Equal(project["names"].([]string), []string{"web-design"}) || !slices.Equal(project["agents"].([]string), []string{"claude-code", "codex", "cursor"}) || project["copy"] != nil {
		t.Fatalf("project selection = %+v", project)
	}
	if user := readConfigComponents(t, userConfigFile(e.home))["remote"].(map[string]any); user["load"] != "project" {
		t.Fatalf("user registration = %+v", user)
	}
	if code := run([]string{"status"}); code != 0 {
		t.Fatalf("status after add exited %d", code)
	}
	write(t, filepath.Join(canonical, "SKILL.md"), "edited\n")
	if code := run([]string{"status"}); code != 1 {
		t.Fatalf("status with an edited skill exited %d", code)
	}
	if code := run([]string{"restore"}); code != 0 || !strings.Contains(readString(t, filepath.Join(canonical, "SKILL.md")), "name: Web Design") {
		t.Fatalf("restore exited %d", code)
	}
	if code := run([]string{"remove", "remote"}); code != 0 {
		t.Fatalf("remove exited %d", code)
	}
	for _, path := range []string{canonical, filepath.Join(root, ".claude", "skills", "web-design")} {
		if _, err := os.Lstat(path); !os.IsNotExist(err) {
			t.Fatalf("remove left %s: %v", path, err)
		}
	}
	if _, declared := readConfigComponents(t, projectConfigFile(root))["remote"]; declared {
		t.Fatal("remove left the project table")
	}
}

func TestAddManifestRenameReAdd(t *testing.T) {
	e := newUtilityEnv(t, "schema = \"1\"\nname = \"util\"\nskills = [\"alpha\", \"beta\"]\n")
	for _, name := range []string{"alpha", "beta"} {
		write(t, filepath.Join(e.remote, "skills", name, "SKILL.md"), "---\nname: "+name+"\ndescription: "+name+" body\n---\n")
	}
	commitAll(t, e.remote)
	quietStdout(t)
	if code := run([]string{"add", e.remote, "-g", "-y", "-a", "claude-code", "-s", "alpha", "--rename", "gamma"}); code != 0 {
		t.Fatalf("rename from a manifest exited %d", code)
	}
	installed := filepath.Join(e.home, ".claude", "skills", "beta", "SKILL.md")
	if code := run([]string{"add", e.remote, "-g", "-y", "-a", "claude-code", "-s", "alpha", "--rename", "beta"}); code != 0 {
		t.Fatalf("rename onto a declared name exited %d", code)
	}
	if got := readString(t, installed); !strings.Contains(got, "alpha body") {
		t.Fatalf("renamed copy = %q", got)
	}
	if code := run([]string{"add", e.remote, "-g", "-y", "-a", "claude-code", "-s", "beta"}); code != 0 {
		t.Fatalf("re-add exited %d", code)
	}
	if got := readString(t, installed); !strings.Contains(got, "beta body") {
		t.Fatalf("re-adding beta kept the old rename: %q", got)
	}
}

func TestAddBadProjectConfigWritesNothing(t *testing.T) {
	e := newUtilityEnv(t, "")
	webDesignRemote(t, e)
	if _, err := git(e.project, "init", "-q"); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(e.project, ".gap", "gap.toml"), "broken = [\n")
	quietStdout(t)
	if code := run([]string{"add", e.remote, "-y", "-a", "claude-code"}); code != 1 {
		t.Fatalf("add over a malformed project config exited %d", code)
	}
	for _, path := range []string{userConfigFile(e.home), filepath.Join(userConfigDir(e.home), "gap.lock")} {
		if exists(path) {
			t.Fatalf("a failed add wrote %s", path)
		}
	}
}

func TestAddRemovesDestinationsOutsideHome(t *testing.T) {
	for _, command := range [][]string{{"remove", "remote"}, {"uninstall", "--yes"}} {
		t.Run(command[0], func(t *testing.T) {
			e := newUtilityEnv(t, "")
			webDesignRemote(t, e)
			claude := t.TempDir()
			t.Setenv("CLAUDE_CONFIG_DIR", claude)
			t.Setenv("GROK_HOME", filepath.Join(e.home, strings.TrimPrefix(claude, string(filepath.Separator))))
			quietStdout(t)
			if code := run([]string{"add", e.remote, "-g", "-y", "-a", "claude-code", "-a", "grok", "--copy"}); code != 0 {
				t.Fatalf("add exited %d", code)
			}
			installed := []string{filepath.Join(claude, "skills", "web-design"), filepath.Join(os.Getenv("GROK_HOME"), "skills", "web-design")}
			for _, dir := range installed {
				if !exists(filepath.Join(dir, "SKILL.md")) {
					t.Fatalf("add did not install %s", dir)
				}
			}
			if code := run(command); code != 0 {
				t.Fatalf("%s exited %d", command[0], code)
			}
			for _, dir := range installed {
				if exists(dir) {
					t.Fatalf("%s left %s", command[0], dir)
				}
			}
		})
	}
}

func TestAddProjectHookFailureStaysRemovable(t *testing.T) {
	e := newUtilityEnv(t, "schema = \"1\"\nname = \"util\"\nskills = [\"util-skill\"]\n[hooks]\ninstall = \"exit 42\"\n")
	write(t, userConfigFile(e.home), fmt.Sprintf("[trust]\nrepos = [%q]\n", e.remote))
	if _, err := git(e.project, "init", "-q"); err != nil {
		t.Fatal(err)
	}
	root, err := filepath.EvalSymlinks(e.project)
	if err != nil {
		t.Fatal(err)
	}
	quietStdout(t)
	if code := run([]string{"add", e.remote, "-y", "-a", "claude-code", "-a", "codex"}); code != 1 {
		t.Fatalf("add with a failing install hook exited %d", code)
	}
	dirs := skillDirs(root, "util-skill")
	if !exists(filepath.Join(dirs[0], "SKILL.md")) {
		t.Fatal("skills were not installed before the hook ran")
	}
	if code := run([]string{"remove", "remote"}); code != 0 {
		t.Fatalf("remove exited %d", code)
	}
	for _, dir := range dirs {
		if _, err := os.Lstat(dir); !os.IsNotExist(err) {
			t.Fatalf("remove left %s", dir)
		}
	}
}

func TestAddMigratesLegacyDestinations(t *testing.T) {
	for _, via := range []string{"add", "update"} {
		t.Run(via, func(t *testing.T) {
			e := newUtilityEnv(t, "schema = \"1\"\nname = \"util\"\nskills = [\"util-skill\"]\n")
			config := fmt.Sprintf("[components.util]\nrepo = %q\nref = \"master\"\n", e.remote)
			write(t, userConfigFile(e.home), config)
			quietStdout(t)
			if code := run([]string{"update", "util"}); code != 0 {
				t.Fatalf("legacy install exited %d", code)
			}
			migrate := []string{"add", e.remote, "--id", "util", "-g", "-y", "-a", "codex"}
			if via == "update" {
				write(t, userConfigFile(e.home), config+"agents = [\"codex\"]\n")
				migrate = []string{"update", "util"}
			}
			if code := run(migrate); code != 0 {
				t.Fatalf("%s exited %d", via, code)
			}
			if code := run([]string{"remove", "util"}); code != 0 {
				t.Fatalf("remove exited %d", code)
			}
			for _, dir := range skillDirs(e.home, "util-skill") {
				if _, err := os.Lstat(dir); !os.IsNotExist(err) {
					t.Fatalf("remove left the legacy copy %s", dir)
				}
			}
		})
	}
}

func TestAddRetiresAVerifiedLegacyCopy(t *testing.T) {
	twin := "---\nname: util-skill\ndescription: a same-name folder\n---\n"
	for _, tc := range []struct {
		name, edit   string
		dropManifest bool
		retired      bool
	}{
		{name: "unedited", retired: true},
		{name: "edited", edit: "edited\n"},
		{name: "edit matches a same-name folder after the manifest is dropped", edit: twin, dropManifest: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := newUtilityEnv(t, "schema = \"1\"\nname = \"util\"\nskills = [\"util-skill\"]\n")
			write(t, filepath.Join(e.remote, "skills", "aaa", "SKILL.md"), twin)
			commitAll(t, e.remote)
			write(t, userConfigFile(e.home), fmt.Sprintf("[components.util]\nrepo = %q\nref = \"master\"\n", e.remote))
			quietStdout(t)
			if code := run([]string{"update", "util"}); code != 0 {
				t.Fatalf("legacy install exited %d", code)
			}
			legacy := filepath.Join(e.home, ".claude", "skills", "util-skill")
			if tc.edit != "" {
				write(t, filepath.Join(legacy, "SKILL.md"), tc.edit)
			}
			if tc.dropManifest {
				for _, rel := range []string{utilityManifestName, filepath.Join("skills", "aaa")} {
					if err := os.RemoveAll(filepath.Join(e.remote, rel)); err != nil {
						t.Fatal(err)
					}
				}
				commitAll(t, e.remote)
			}
			t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(e.home, ".new-claude"))
			if code := run([]string{"add", e.remote, "--id", "util", "-g", "-y", "--copy"}); code != 0 {
				t.Fatalf("re-add exited %d", code)
			}
			if _, err := os.Lstat(legacy); os.IsNotExist(err) != tc.retired {
				t.Fatalf("legacy copy retired = %v, want %v", os.IsNotExist(err), tc.retired)
			}
			if !exists(filepath.Join(e.home, ".new-claude", "skills", "util-skill", "SKILL.md")) {
				t.Fatal("the re-add did not install for the new Claude Code folder")
			}
		})
	}
}

func TestAddKeepsTheLastUsableCopy(t *testing.T) {
	e := newUtilityEnv(t, "schema = \"1\"\nname = \"util\"\nskills = [\"util-skill\"]\n")
	quietStdout(t)
	if code := run([]string{"add", e.remote, "-g", "-y", "-a", "claude-code", "-a", "windsurf"}); code != 0 {
		t.Fatalf("add exited %d", code)
	}
	write(t, filepath.Join(e.remote, "skills", "util-skill", "SKILL.md"), "")
	commitAll(t, e.remote)
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(e.home, ".new-claude"))
	run([]string{"add", e.remote, "-g", "-y", "-a", "claude-code", "-a", "windsurf", "--copy"})
	canonical := filepath.Join(e.home, ".agents", "skills", "util-skill", "SKILL.md")
	if !exists(canonical) || readString(t, canonical) == "" {
		t.Fatal("retirement moved the last usable copy")
	}
}

func TestAddAdoptionLeavesForeignFolders(t *testing.T) {
	e := newUtilityEnv(t, "")
	skill := "---\nname: alpha\ndescription: alpha skill\n---\n"
	write(t, filepath.Join(e.remote, "skills", "alpha", "SKILL.md"), skill)
	commitAll(t, e.remote)
	canonical := filepath.Join(e.home, ".agents", "skills", "alpha", "SKILL.md")
	foreign := filepath.Join(e.home, ".claude", "skills", "alpha", "SKILL.md")
	write(t, canonical, skill)
	write(t, foreign, "foreign\n")
	write(t, skillLockPath(e.home), `{"version":3,"skills":{"alpha":{"source":"owner/tools"}}}`)
	config := fmt.Sprintf("[components.util]\nrepo = %q\nref = \"master\"\nagents = [\"codex\"]\n", e.remote)
	write(t, userConfigFile(e.home), config+"names = [\"alpha\", \"missing\"]\n")
	quietStdout(t)
	if code := run([]string{"update", "util"}); code != 1 {
		t.Fatalf("update with a missing skill exited %d", code)
	}
	write(t, userConfigFile(e.home), config+"names = [\"alpha\"]\n")
	write(t, canonical, "edited\n")
	if code := run([]string{"restore", "util"}); code != 0 {
		t.Fatalf("restore exited %d", code)
	}
	if code := run([]string{"remove", "util"}); code != 0 {
		t.Fatalf("remove exited %d", code)
	}
	if got := readString(t, foreign); got != "foreign\n" {
		t.Fatalf("remove took a folder the utility never wrote: %q", got)
	}
}

func TestAddSharedRemovalKeepsDestinationsOwned(t *testing.T) {
	e := newUtilityEnv(t, "schema = \"1\"\nname = \"util\"\nskills = [\"util-skill\"]\n")
	quietStdout(t)
	if code := run([]string{"add", e.remote, "--id", "a", "-g", "-y", "-a", "claude-code", "-a", "codex"}); code != 0 {
		t.Fatalf("add a exited %d", code)
	}
	write(t, userConfigFile(e.home), readString(t, userConfigFile(e.home))+fmt.Sprintf("[components.b]\nrepo = %q\nref = \"master\"\nagents = [\"codex\"]\n", e.remote))
	if code := run([]string{"update", "b"}); code != 0 {
		t.Fatalf("update b exited %d", code)
	}
	for _, id := range []string{"a", "b"} {
		if code := run([]string{"remove", id}); code != 0 {
			t.Fatalf("remove %s exited %d", id, code)
		}
	}
	for _, dir := range skillDirs(e.home, "util-skill") {
		if _, err := os.Lstat(dir); !os.IsNotExist(err) {
			t.Fatalf("removing both owners left %s", dir)
		}
	}
}

func TestAddSharedTransferLeavesForeignFolders(t *testing.T) {
	e := newUtilityEnv(t, "")
	quietStdout(t)
	if code := run([]string{"add", e.remote, "--id", "a", "-g", "-y", "-a", "codex", "-s", "util-skill"}); code != 0 {
		t.Fatalf("add a exited %d", code)
	}
	foreign := filepath.Join(e.home, ".claude", "skills", "util-skill", "SKILL.md")
	write(t, foreign, "foreign\n")
	write(t, userConfigFile(e.home), readString(t, userConfigFile(e.home))+fmt.Sprintf("[components.b]\nrepo = %q\nref = \"master\"\nagents = [\"codex\"]\nnames = [\"util-skill\", \"missing\"]\n", e.remote))
	if code := run([]string{"update", "b"}); code != 1 {
		t.Fatalf("update b with a missing skill exited %d", code)
	}
	for _, id := range []string{"a", "b"} {
		if code := run([]string{"remove", id}); code != 0 {
			t.Fatalf("remove %s exited %d", id, code)
		}
	}
	if got := readString(t, foreign); got != "foreign\n" {
		t.Fatalf("remove took a folder neither utility wrote: %q", got)
	}
}

func TestAddWithoutDestinationsClaimsNothing(t *testing.T) {
	e := newUtilityEnv(t, "schema = \"1\"\nname = \"util\"\nskills = [\"util-skill\"]\n")
	for _, dir := range skillDirs(e.home, "util-skill") {
		write(t, filepath.Join(dir, "SKILL.md"), "foreign\n")
	}
	write(t, userConfigFile(e.home), fmt.Sprintf("[components.util]\nrepo = %q\nref = \"master\"\nagents = [\"eve\"]\n", e.remote))
	quietStdout(t)
	if code := run([]string{"update", "util"}); code != 1 {
		t.Fatalf("update with no global destination exited %d", code)
	}
	if code := run([]string{"remove", "util"}); code != 0 {
		t.Fatalf("remove exited %d", code)
	}
	for _, dir := range skillDirs(e.home, "util-skill") {
		if got := readString(t, filepath.Join(dir, "SKILL.md")); got != "foreign\n" {
			t.Fatalf("remove took %s, which the utility never wrote", dir)
		}
	}
}

func TestAddDroppedAgentsKeepTheLedger(t *testing.T) {
	e := newUtilityEnv(t, "")
	webDesignRemote(t, e)
	quietStdout(t)
	if code := run([]string{"add", e.remote, "-g", "-y", "-a", "codex"}); code != 0 {
		t.Fatalf("add exited %d", code)
	}
	write(t, userConfigFile(e.home), fmt.Sprintf("[components.remote]\nrepo = %q\nref = \"master\"\nnames = [\"web-design\"]\n", e.remote))
	if code := run([]string{"restore", "remote"}); code != 0 {
		t.Fatalf("restore exited %d", code)
	}
	if code := run([]string{"remove", "remote"}); code != 0 {
		t.Fatalf("remove exited %d", code)
	}
	for _, dir := range skillDirs(e.home, "web-design") {
		if _, err := os.Lstat(dir); !os.IsNotExist(err) {
			t.Fatalf("remove left %s", dir)
		}
	}
}

func TestAddReAddVerifiesTheStore(t *testing.T) {
	e := newUtilityEnv(t, "")
	webDesignRemote(t, e)
	quietStdout(t)
	if code := run([]string{"add", e.remote, "-g", "-y", "-a", "claude-code"}); code != 0 {
		t.Fatalf("add exited %d", code)
	}
	lk, err := readLock(userConfigDir(e.home))
	if err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(storePath(e.home, "remote", lk.Components["remote"].Commit), "skills", "web", "SKILL.md"), "---\nname: Web Design\ndescription: tampered\n---\n")
	if code := run([]string{"add", e.remote, "-g", "-y", "-a", "claude-code"}); code != 0 {
		t.Fatalf("re-add exited %d", code)
	}
	if got := readString(t, filepath.Join(e.home, ".claude", "skills", "web-design", "SKILL.md")); strings.Contains(got, "tampered") {
		t.Fatal("re-add installed from a store that no longer matches the pin")
	}
}

func TestAddKeepsTheSelectedSkill(t *testing.T) {
	e := newUtilityEnv(t, "")
	webDesignRemote(t, e)
	write(t, filepath.Join(e.remote, "skills", "web-lower", "SKILL.md"), "---\nname: web-design\ndescription: lowercase twin\n---\n")
	commitAll(t, e.remote)
	quietStdout(t)
	if code := run([]string{"add", e.remote, "-g", "-y", "-a", "claude-code", "-s", "Web Design", "-s", "web-design"}); code != 1 {
		t.Fatalf("two skills for one folder exited %d", code)
	}
	if code := run([]string{"add", e.remote, "-g", "-y", "-a", "claude-code", "-s", "web-design"}); code != 0 {
		t.Fatalf("add exited %d", code)
	}
	if got := readString(t, filepath.Join(e.home, ".claude", "skills", "web-design", "SKILL.md")); !strings.Contains(got, "lowercase twin") {
		t.Fatalf("installed the other skill: %q", got)
	}
}

func TestAddReAddAppliesCopyMode(t *testing.T) {
	e := newUtilityEnv(t, "")
	write(t, filepath.Join(e.remote, "skills", "other", "SKILL.md"), "---\nname: other\ndescription: other skill\n---\n")
	commitAll(t, e.remote)
	quietStdout(t)
	if code := run([]string{"add", e.remote, "-g", "-y", "-a", "claude-code", "-a", "codex", "-s", "util-skill"}); code != 0 {
		t.Fatalf("add exited %d", code)
	}
	link := filepath.Join(e.home, ".claude", "skills", "util-skill")
	if info, err := os.Lstat(link); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("first add did not link %s: %v", link, err)
	}
	if code := run([]string{"add", e.remote, "-g", "-y", "-a", "claude-code", "-a", "codex", "-s", "other", "--copy"}); code != 0 {
		t.Fatalf("re-add exited %d", code)
	}
	if info, err := os.Lstat(link); err != nil || !info.IsDir() {
		t.Fatalf("--copy left %s linked: %v", link, err)
	}
}

func TestAddAnotherProjectKeepsThePin(t *testing.T) {
	e := newUtilityEnv(t, "")
	second := t.TempDir()
	for _, dir := range []string{e.project, second} {
		if _, err := git(dir, "init", "-q"); err != nil {
			t.Fatal(err)
		}
	}
	quietStdout(t)
	if code := run([]string{"add", e.remote, "-y", "-a", "codex"}); code != 0 {
		t.Fatalf("first project add exited %d", code)
	}
	lk, err := readLock(userConfigDir(e.home))
	if err != nil {
		t.Fatal(err)
	}
	pinned := lk.Components["remote"].Commit
	write(t, filepath.Join(e.remote, "skills", "util-skill", "SKILL.md"), "---\nname: util-skill\ndescription: moved upstream\n---\n")
	commitAll(t, e.remote)
	t.Chdir(second)
	if code := run([]string{"add", e.remote, "-y", "-a", "codex"}); code != 0 {
		t.Fatalf("second project add exited %d", code)
	}
	if lk, err = readLock(userConfigDir(e.home)); err != nil || lk.Components["remote"].Commit != pinned {
		t.Fatalf("adding another project moved the shared pin: %v", err)
	}
	root, err := filepath.EvalSymlinks(second)
	if err != nil {
		t.Fatal(err)
	}
	if got := readString(t, filepath.Join(root, ".agents", "skills", "util-skill", "SKILL.md")); strings.Contains(got, "moved upstream") {
		t.Fatal("the second project got a commit the pin does not name")
	}
	if _, err := git(e.remote, "branch", "dev"); err != nil {
		t.Fatal(err)
	}
	write(t, userConfigFile(e.home), strings.Replace(readString(t, userConfigFile(e.home)), `ref = "master"`, `ref = "dev"`, 1))
	if code := run([]string{"add", e.remote, "-y", "-a", "codex"}); code != 1 {
		t.Fatalf("re-add with a ref the pin does not track exited %d", code)
	}
}

func TestAddManifestRenameRewritesTheName(t *testing.T) {
	e := newUtilityEnv(t, "schema = \"1\"\nname = \"util\"\nskills = [\"Web Design\"]\n")
	write(t, filepath.Join(e.remote, "skills", "Web Design", "SKILL.md"), "---\nname: Web Design\ndescription: spaced folder\n---\n")
	commitAll(t, e.remote)
	quietStdout(t)
	if code := run([]string{"add", e.remote, "-g", "-y", "-a", "claude-code", "-s", "Web Design", "--rename", "web-design"}); code != 0 {
		t.Fatalf("add exited %d", code)
	}
	if got := readString(t, filepath.Join(e.home, ".claude", "skills", "web-design", "SKILL.md")); !strings.Contains(got, "name: web-design") {
		t.Fatalf("a manifest rename kept the old name: %q", got)
	}
}

func TestAddListUsesTheProjectPin(t *testing.T) {
	e := newUtilityEnv(t, "")
	if _, err := git(e.project, "init", "-q"); err != nil {
		t.Fatal(err)
	}
	quietStdout(t)
	if code := run([]string{"add", e.remote, "-y", "-a", "codex"}); code != 0 {
		t.Fatalf("add exited %d", code)
	}
	write(t, filepath.Join(e.remote, "skills", "fresh", "SKILL.md"), "---\nname: fresh\ndescription: added upstream\n---\n")
	commitAll(t, e.remote)
	stdout, _ := captureStd(t, func() {
		if code := run([]string{"add", e.remote, "--list"}); code != 0 {
			t.Errorf("list exited %d", code)
		}
	})
	if strings.Contains(stdout, "fresh") {
		t.Fatalf("--list showed a skill the pinned commit lacks:\n%s", stdout)
	}
}

func TestAddReAddReportsAFailedConversion(t *testing.T) {
	e := newUtilityEnv(t, "")
	write(t, filepath.Join(e.remote, "skills", "other", "SKILL.md"), "---\nname: other\ndescription: other skill\n---\n")
	commitAll(t, e.remote)
	quietStdout(t)
	if code := run([]string{"add", e.remote, "-g", "-y", "-a", "claude-code", "-a", "codex", "-s", "util-skill"}); code != 0 {
		t.Fatalf("add exited %d", code)
	}
	write(t, filepath.Join(e.home, ".gap", "backups"), "not a folder\n")
	if code := run([]string{"add", e.remote, "-g", "-y", "-a", "claude-code", "-a", "codex", "-s", "other", "--copy"}); code != 1 {
		t.Fatalf("a failed conversion exited %d", code)
	}
	link := filepath.Join(e.home, ".claude", "skills", "util-skill")
	if info, err := os.Lstat(link); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("a failed conversion lost %s: %v", link, err)
	}
}

func TestAddCopyConversionRetiresTheCanonicalCopy(t *testing.T) {
	for _, kind := range []string{"same pin", "moved pin", "retried", "unusable replacement", "edit matches a cached revision"} {
		t.Run(kind, func(t *testing.T) {
			e := newUtilityEnv(t, "")
			write(t, filepath.Join(e.remote, "skills", "other", "SKILL.md"), "---\nname: other\ndescription: other skill\n---\n")
			write(t, filepath.Join(e.remote, "skills", "util-skill", "ref.md"), "reference the skill needs\n")
			commitAll(t, e.remote)
			quietStdout(t)
			if code := run([]string{"add", e.remote, "-g", "-y", "-a", "claude-code", "-a", "windsurf", "-s", "util-skill"}); code != 0 {
				t.Fatalf("add exited %d", code)
			}
			canonical := filepath.Join(e.home, ".agents", "skills", "util-skill")
			if !exists(filepath.Join(canonical, "SKILL.md")) {
				t.Fatal("symlink mode wrote no canonical copy")
			}
			copies := []string{filepath.Join(e.home, ".claude", "skills", "util-skill"), filepath.Join(e.home, ".codeium", "windsurf", "skills", "util-skill")}
			readd := []string{"add", e.remote, "-g", "-y", "-a", "claude-code", "-a", "windsurf", "-s", "other", "--copy"}
			if kind == "moved pin" {
				write(t, filepath.Join(e.remote, "skills", "util-skill", "SKILL.md"), "---\nname: util-skill\ndescription: newer upstream\n---\n")
				commitAll(t, e.remote)
			}
			if kind != "same pin" && kind != "moved pin" {
				parent := filepath.Dir(canonical)
				if err := os.Chmod(parent, 0o555); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { os.Chmod(parent, 0o755) })
				if code := run(readd); code != 1 {
					t.Fatalf("a blocked retirement exited %d", code)
				}
				if err := os.Chmod(parent, 0o755); err != nil {
					t.Fatal(err)
				}
			}
			switch kind {
			case "unusable replacement":
				for _, dir := range copies {
					if err := os.Remove(filepath.Join(dir, "ref.md")); err != nil {
						t.Fatal(err)
					}
				}
				run(readd)
				if !exists(filepath.Join(canonical, "SKILL.md")) {
					t.Fatal("retirement took the only usable copy")
				}
				return
			case "edit matches a cached revision":
				intermediate := "---\nname: util-skill\ndescription: intermediate\n---\n"
				write(t, filepath.Join(e.remote, "skills", "util-skill", "SKILL.md"), intermediate)
				commitAll(t, e.remote)
				if code := run([]string{"add", e.remote, "--list"}); code != 0 {
					t.Fatalf("list exited %d", code)
				}
				write(t, filepath.Join(e.remote, "skills", "util-skill", "SKILL.md"), "---\nname: util-skill\ndescription: newest\n---\n")
				commitAll(t, e.remote)
				write(t, filepath.Join(canonical, "SKILL.md"), intermediate)
				run(readd)
				if got := readString(t, filepath.Join(canonical, "SKILL.md")); got != intermediate {
					t.Fatalf("retirement moved an edited copy that matched a cached revision: %q", got)
				}
				return
			}
			if code := run(readd); code != 0 {
				t.Fatalf("re-add exited %d", code)
			}
			if _, err := os.Lstat(canonical); !os.IsNotExist(err) {
				t.Fatalf("the unused canonical copy is still live: %v", err)
			}
			for _, dir := range copies {
				if info, err := os.Lstat(dir); err != nil || !info.IsDir() {
					t.Fatalf("%s is not a copy: %v", dir, err)
				}
			}
			if kind == "moved pin" && !strings.Contains(readString(t, filepath.Join(copies[0], "SKILL.md")), "newer upstream") {
				t.Fatal("the converted copy did not move to the new pin")
			}
		})
	}
}

func TestAddRetiresAnOldLinkAndItsTarget(t *testing.T) {
	e := newUtilityEnv(t, "")
	write(t, filepath.Join(e.remote, "skills", "other", "SKILL.md"), "---\nname: other\ndescription: other skill\n---\n")
	commitAll(t, e.remote)
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(e.home, ".000-claude"))
	quietStdout(t)
	if code := run([]string{"add", e.remote, "-g", "-y", "-a", "claude-code", "-a", "windsurf", "-s", "util-skill"}); code != 0 {
		t.Fatalf("add exited %d", code)
	}
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(e.home, ".new-claude"))
	if code := run([]string{"add", e.remote, "-g", "-y", "-a", "claude-code", "-a", "windsurf", "-s", "other", "--copy"}); code != 0 {
		t.Fatalf("re-add exited %d", code)
	}
	for _, path := range []string{filepath.Join(e.home, ".000-claude", "skills", "util-skill"), filepath.Join(e.home, ".agents", "skills", "util-skill")} {
		if _, err := os.Lstat(path); !os.IsNotExist(err) {
			t.Fatalf("%s is still live: %v", path, err)
		}
	}
}

func TestAddRetirementKeepsAliasedDestinations(t *testing.T) {
	for _, kind := range []string{"symlink", "relative"} {
		t.Run(kind, func(t *testing.T) {
			e := newUtilityEnv(t, "")
			write(t, filepath.Join(e.remote, "skills", "other", "SKILL.md"), "---\nname: other\ndescription: other skill\n---\n")
			commitAll(t, e.remote)
			quietStdout(t)
			if code := run([]string{"add", e.remote, "-g", "-y", "-a", "claude-code", "-a", "windsurf", "-s", "util-skill"}); code != 0 {
				t.Fatalf("add exited %d", code)
			}
			alias := ".agents"
			if kind == "symlink" {
				alias = filepath.Join(t.TempDir(), "claude")
				if err := os.Symlink(filepath.Join(e.home, ".agents"), alias); err != nil {
					t.Fatal(err)
				}
			} else {
				t.Chdir(e.home)
			}
			t.Setenv("CLAUDE_CONFIG_DIR", alias)
			if code := run([]string{"add", e.remote, "-g", "-y", "-a", "claude-code", "-a", "windsurf", "-s", "other", "--copy"}); code != 0 {
				t.Fatalf("re-add exited %d", code)
			}
			if !exists(filepath.Join(e.home, ".agents", "skills", "util-skill", "SKILL.md")) {
				t.Fatal("retirement moved the copy Claude Code now reads")
			}
		})
	}
}

func TestAddRenameAndJSON(t *testing.T) {
	e := newUtilityEnv(t, "")
	webDesignRemote(t, e)
	var stdout string
	stdout, _ = captureStd(t, func() {
		if code := run([]string{"add", e.remote, "-g", "-y", "-a", "claude-code", "-s", "Web Design", "--rename", "web2", "--json"}); code != 0 {
			t.Errorf("add exited %d", code)
		}
	})
	var results []map[string]any
	if err := json.Unmarshal([]byte(stdout), &results); err != nil || len(results) != 1 {
		t.Fatalf("stdout is not one JSON result: %v\n%s", err, stdout)
	}
	if r := results[0]; r["name"] != "web2" || r["status"] != "installed" || r["scope"] != "global" || r["security"] != nil {
		t.Fatalf("result = %+v", r)
	} else if _, present := r["security"]; !present {
		t.Fatalf("result has no security key: %+v", r)
	}
	if body := readString(t, filepath.Join(e.home, ".claude", "skills", "web2", "SKILL.md")); !strings.Contains(body, "name: web2") {
		t.Fatalf("renamed skill = %q", body)
	}
	quietStdout(t)
	if code := run([]string{"status"}); code != 0 {
		t.Fatalf("status after a renamed add exited %d", code)
	}
}

func TestAddListWritesNothing(t *testing.T) {
	e := newUtilityEnv(t, "")
	webDesignRemote(t, e)
	stdout, _ := captureStd(t, func() {
		if code := run([]string{"add", e.remote, "--list"}); code != 0 {
			t.Errorf("list exited %d", code)
		}
	})
	if !strings.Contains(stdout, "Web Design") || !strings.Contains(stdout, "Builds web pages") {
		t.Fatalf("list output:\n%s", stdout)
	}
	for _, path := range []string{userConfigFile(e.home), projectConfigFile(e.project)} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("list wrote %s: %v", path, err)
		}
	}
}

func TestSkillDests(t *testing.T) {
	home, project := t.TempDir(), t.TempDir()
	env := agentEnv{Home: home, ConfigHome: filepath.Join(home, ".config"), Cwd: project, Getenv: func(string) string { return "" }}
	type dest struct{ dir, link string }
	for _, tc := range []struct {
		name      string
		root      string
		global    bool
		entry     manifestEntry
		want      []dest
		eveCopies int
	}{
		{"project symlink", project, false, manifestEntry{Agents: []string{"claude-code", "codex"}}, []dest{
			{filepath.Join(project, ".agents", "skills", "x"), ""},
			{filepath.Join(project, ".claude", "skills", "x"), filepath.Join(project, ".agents", "skills", "x")},
		}, 0},
		{"global symlink", home, true, manifestEntry{Agents: []string{"claude-code", "antigravity", "codex"}}, []dest{
			{filepath.Join(home, ".agents", "skills", "x"), ""},
			{filepath.Join(home, ".claude", "skills", "x"), filepath.Join(home, ".agents", "skills", "x")},
			{filepath.Join(home, ".gemini", "config", "skills", "x"), filepath.Join(home, ".agents", "skills", "x")},
		}, 0},
		{"global copy is the legacy pair", home, true, manifestEntry{Agents: []string{"claude-code", "codex"}, Copy: true}, []dest{
			{filepath.Join(home, ".claude", "skills", "x"), ""},
			{filepath.Join(home, ".agents", "skills", "x"), ""},
		}, 0},
		{"eve subagents", project, false, manifestEntry{Agents: []string{"eve"}, Subagents: []string{"", "rev"}}, []dest{
			{filepath.Join(project, "agent", "skills", "x"), ""},
			{filepath.Join(project, "agent", "subagents", "rev", "skills", "x"), ""},
		}, 2},
		{"no agents keeps the legacy pair", home, true, manifestEntry{}, []dest{
			{filepath.Join(home, ".agents", "skills", "x"), ""},
			{filepath.Join(home, ".claude", "skills", "x"), ""},
		}, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var got []dest
			eve := 0
			for _, d := range skillDests(tc.root, tc.global, "x", tc.entry, env) {
				got = append(got, dest{d.Dir, d.LinkTo})
				if d.Eve {
					eve++
				}
			}
			if !slices.Equal(got, tc.want) || eve != tc.eveCopies {
				t.Fatalf("dests = %+v (eve %d), want %+v (eve %d)", got, eve, tc.want, tc.eveCopies)
			}
		})
	}
}

func TestDiscoverSkills(t *testing.T) {
	skill := func(name string, extra string) string {
		return "---\nname: " + name + "\ndescription: " + name + " skill\n" + extra + "---\n"
	}
	names := func(skills []discoveredSkill) []string {
		var out []string
		for _, s := range skills {
			out = append(out, s.Name)
		}
		return out
	}
	root := t.TempDir()
	write(t, filepath.Join(root, "SKILL.md"), skill("root", ""))
	write(t, filepath.Join(root, "skills", "a", "SKILL.md"), skill("a", ""))
	if got, err := discoverSkills(root, "", false, false, nil); err != nil || !slices.Equal(names(got), []string{"root"}) {
		t.Fatalf("root skill without full depth = %v %v", names(got), err)
	}
	if got, _ := discoverSkills(root, "", true, false, nil); !slices.Equal(names(got), []string{"root", "a"}) {
		t.Fatalf("root skill with full depth = %v", names(got))
	}

	repo := t.TempDir()
	write(t, filepath.Join(repo, "skills", "a", "SKILL.md"), skill("a", ""))
	write(t, filepath.Join(repo, ".claude", "skills", "b", "SKILL.md"), skill("b", ""))
	write(t, filepath.Join(repo, "skills", "hidden", "SKILL.md"), skill("hidden", "metadata:\n  internal: true\n"))
	write(t, filepath.Join(repo, "skills", "z-dup", "SKILL.md"), skill("a", ""))
	write(t, filepath.Join(repo, "skills", "cycle", "SKILL.md"), skill("cycle", "metadata: &meta\n  self: *meta\n"))
	write(t, filepath.Join(repo, "skills", "number", "SKILL.md"), skill("1_000", ""))
	got, err := discoverSkills(repo, "", false, false, nil)
	if err != nil || !slices.Equal(names(got), []string{"a", "cycle", "1_000", "b"}) {
		t.Fatalf("discovered %v %v", names(got), err)
	}
	if got[0].Dir != filepath.Join(repo, "skills", "a") {
		t.Fatalf("duplicate name did not keep the first: %s", got[0].Dir)
	}
	if got, _ := discoverSkills(repo, "", false, true, nil); !slices.Contains(names(got), "hidden") {
		t.Fatalf("requested internal skill hidden: %v", names(got))
	}
	if _, err := discoverSkills(repo, "skills/../..", false, false, nil); err == nil {
		t.Fatal("unsafe subpath accepted")
	}
	outside, linked := t.TempDir(), t.TempDir()
	write(t, filepath.Join(outside, "escape", "SKILL.md"), skill("escape", ""))
	if err := os.Symlink(outside, filepath.Join(linked, "skills")); err != nil {
		t.Fatal(err)
	}
	if got, _ := discoverSkills(linked, "", true, false, nil); len(got) != 0 {
		t.Fatalf("discovered skills outside the repository: %v", names(got))
	}
	mono := t.TempDir()
	write(t, filepath.Join(mono, "shared.json"), `{"paths":["./custom"]}`)
	write(t, filepath.Join(mono, "sub", "skills", "ok", "SKILL.md"), skill("ok", ""))
	write(t, filepath.Join(mono, "sub", "custom", "extra", "SKILL.md"), skill("extra", ""))
	if err := os.Symlink(filepath.Join("..", "shared.json"), filepath.Join(mono, "sub", "skills.sh.json")); err != nil {
		t.Fatal(err)
	}
	if got, _ := discoverSkills(mono, "sub", false, false, nil); !slices.Equal(names(got), []string{"ok", "extra"}) {
		t.Fatalf("a subpath's skills.sh.json linked inside the repository = %v", names(got))
	}
}

func TestDiscoveryReadsNothingOutsideTheRepository(t *testing.T) {
	outside := t.TempDir()
	fifo := filepath.Join(outside, "fifo")
	for _, path := range []string{fifo, filepath.Join(outside, "tree", "skills.sh.json")} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := syscall.Mkfifo(path, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write(t, filepath.Join(outside, "bad.md"), "---\nname: [unclosed\n---\n")
	write(t, filepath.Join(outside, "tree", "skills", "x", "SKILL.md"), "---\nname: x\ndescription: outside\n---\n")
	repo := t.TempDir()
	write(t, filepath.Join(repo, "skills", "ok", "SKILL.md"), "---\nname: ok\ndescription: inside\n---\n")
	for rel, target := range map[string]string{
		"skills-lock.json":                fifo,
		"skills.sh.json":                  fifo,
		".claude-plugin/marketplace.json": fifo,
		".claude-plugin/plugin.json":      fifo,
		".claude/skills":                  filepath.Join(outside, "tree", "skills"),
		"skills/escape/SKILL.md":          filepath.Join(outside, "bad.md"),
		"sub":                             filepath.Join(outside, "tree"),
	} {
		link := filepath.Join(repo, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(target, link); err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct {
		subpath string
		want    []string
	}{
		{"", []string{"ok"}},
		{"sub", nil},
	} {
		var warnings []string
		done := make(chan []string, 1)
		go func() {
			skills, _ := discoverSkills(repo, tc.subpath, false, false, func(w string) { warnings = append(warnings, w) })
			var names []string
			for _, s := range skills {
				names = append(names, s.Name)
			}
			done <- names
		}()
		select {
		case names := <-done:
			if !slices.Equal(names, tc.want) || strings.Contains(strings.Join(warnings, "\n"), "YAML") {
				t.Fatalf("subpath %q: skills %v, warnings %q", tc.subpath, names, warnings)
			}
		case <-time.After(10 * time.Second):
			t.Fatalf("subpath %q: discovery blocked on a file outside the repository", tc.subpath)
		}
	}
}

func TestDiscoverySkipsSpecialFilesInTheRepository(t *testing.T) {
	repo := t.TempDir()
	write(t, filepath.Join(repo, "other", "ok", "SKILL.md"), "---\nname: ok\ndescription: inside\n---\n")
	for _, rel := range []string{"skills", ".claude/skills", "other/pipe", "sub"} {
		fifo := filepath.Join(repo, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(fifo), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := syscall.Mkfifo(fifo, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink("skills", filepath.Join(repo, "linked")); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		subpath string
		want    []string
	}{
		{"", []string{"ok"}},
		{"sub", nil},
		{"linked", nil},
	} {
		done := make(chan []string, 1)
		go func() {
			skills, _ := discoverSkills(repo, tc.subpath, true, false, nil)
			var names []string
			for _, s := range skills {
				names = append(names, s.Name)
			}
			done <- names
		}()
		select {
		case names := <-done:
			if !slices.Equal(names, tc.want) {
				t.Fatalf("subpath %q: skills %v", tc.subpath, names)
			}
		case <-time.After(10 * time.Second):
			t.Fatalf("subpath %q: discovery blocked on a special file in the repository", tc.subpath)
		}
	}
	done := make(chan string, 1)
	go func() { done <- findSkillDir(repo, "ok", "skills", "linked", ".") }()
	select {
	case got := <-done:
		if got != filepath.Join(repo, "other", "ok") {
			t.Fatalf("findSkillDir = %q", got)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("findSkillDir blocked on a special file in the repository")
	}
}
