package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

type utilityEnv struct {
	home, repo, project, remote string
	tty                         bool
	answers                     []string
}

func newUtilityEnv(t *testing.T, manifest string) *utilityEnv {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	e := &utilityEnv{home: t.TempDir(), repo: t.TempDir(), project: t.TempDir(), remote: filepath.Join(t.TempDir(), "remote")}
	t.Setenv("HOME", e.home)
	t.Setenv("XDG_STATE_HOME", "")
	write(t, filepath.Join(e.repo, "global", "AGENTS.md"), "policy\n")
	write(t, filepath.Join(e.repo, "gap.toml"), "schema = \"2\"\n")
	old := sourceRootFor
	sourceRootFor = e.repo
	oldReady, oldLine := termReady, confirmLine
	t.Cleanup(func() { sourceRootFor, termReady, confirmLine = old, oldReady, oldLine })
	termReady = func() bool { return e.tty }
	confirmLine = func() string {
		if len(e.answers) == 0 {
			t.Fatal("unexpected prompt")
		}
		a := e.answers[0]
		e.answers = e.answers[1:]
		return a + "\n"
	}
	t.Chdir(e.project)
	writeUtilityRepo(t, e.remote, manifest)
	commitAll(t, e.remote)
	return e
}

func (e *utilityEnv) exists(rel string) bool {
	_, err := os.Stat(filepath.Join(e.home, rel))
	return err == nil
}

func quietStdout(t *testing.T) {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	done := make(chan struct{})
	go func() { io.Copy(io.Discard, r); close(done) }()
	t.Cleanup(func() { w.Close(); <-done; os.Stdout = old })
}

const hookManifest = "schema = \"1\"\nname = \"util\"\nskills = [\"util-skill\"]\ncommands = [\"commands/util-cmd.md\"]\n[hooks]\ninstall = \"touch \\\"$HOME/installed\\\"\"\nuninstall = \"touch \\\"$HOME/uninstalled\\\"\"\n"

func readConfigComponents(t *testing.T, path string) map[string]any {
	t.Helper()
	doc, err := parseTOML(readString(t, path))
	if err != nil {
		t.Fatal(err)
	}
	components, ok := doc["components"].(map[string]any)
	if _, exists := doc["components"]; exists && !ok {
		t.Fatalf("components must be a table in %s", path)
	}
	return components
}

func TestUtilityAdd(t *testing.T) {
	e := newUtilityEnv(t, hookManifest)
	quietStdout(t)
	if err := addUtility("github.com/", ""); err == nil {
		t.Fatal("bad spec accepted")
	}
	if err := addUtility(e.remote, ""); !errors.Is(err, errSkipped) {
		t.Fatalf("untrusted add with no terminal: %v", err)
	}
	if e.exists("installed") {
		t.Fatal("unapproved install hook ran")
	}
	components := readConfigComponents(t, userConfigFile(e.home))
	config, _ := components["remote"].(map[string]any)
	if config["repo"] != e.remote || config["ref"] != "master" {
		t.Fatalf("user registration = %+v", config)
	}
	lk, err := readLock(userConfigDir(e.home))
	if err != nil || lk.Components["remote"].Ref != "master" || lk.Components["remote"].Commit == "" {
		t.Fatalf("user lock = %+v %v", lk, err)
	}
	if _, err := os.Stat(filepath.Join(e.repo, "gap.lock")); !os.IsNotExist(err) {
		t.Fatal("add wrote the distro lock")
	}
	for _, rel := range []string{".agents/skills/util-skill/SKILL.md", ".claude/skills/util-skill/SKILL.md", ".claude/commands/util-cmd.md", ".omp/agent/commands/util-cmd.md", ".pi/agent/prompts/util-cmd.md"} {
		if !e.exists(rel) {
			t.Fatalf("missing %s", rel)
		}
	}
	if err := addUtility(e.remote, ""); err == nil || !strings.Contains(err.Error(), "already added") {
		t.Fatalf("second add: %v", err)
	}
	e.tty, e.answers = true, []string{"y"}
	if err := updateUtilities("remote"); err != nil {
		t.Fatalf("approved update: %v", err)
	}
	if !e.exists("installed") {
		t.Fatal("approved install hook did not run")
	}
}

func TestUtilityUpdate(t *testing.T) {
	e := newUtilityEnv(t, "schema = \"1\"\nname = \"util\"\nskills = [\"util-skill\"]\n")
	quietStdout(t)
	write(t, userConfigFile(e.home), "[trust]\nrepos = [\""+e.remote+"\"]\n")
	if err := addUtility(e.remote, "util"); err != nil {
		t.Fatal(err)
	}
	first, _ := readLock(userConfigDir(e.home))
	write(t, filepath.Join(e.remote, "skills", "util-skill", "SKILL.md"), "---\nname: util-skill\ndescription: v2\n---\n")
	write(t, filepath.Join(e.remote, utilityManifestName), "schema = \"1\"\nname = \"util\"\nskills = [\"util-skill\"]\n[hooks]\ninstall = \"touch \\\"$HOME/installed\\\"\"\nupdate = \"touch \\\"$HOME/updated\\\"\"\n")
	commit, _ := commitAll(t, e.remote)
	if err := updateUtilities(""); err != nil {
		t.Fatal(err)
	}
	second, _ := readLock(userConfigDir(e.home))
	if second.Components["util"].Commit != commit || first.Components["util"].Commit == commit {
		t.Fatalf("pin did not move: %+v -> %+v", first, second)
	}
	if !strings.Contains(readString(t, filepath.Join(e.home, ".agents", "skills", "util-skill", "SKILL.md")), "v2") {
		t.Fatal("update did not install the new skill")
	}
	if e.exists("installed") || !e.exists("updated") {
		t.Fatal("an update after declarative installation must use the update hook")
	}
	if err := updateUtilities("nope"); err == nil {
		t.Fatal("unknown id updated")
	}
	write(t, filepath.Join(e.repo, "gap.toml"), "schema = \"2\"\n\n[components.distro]\nrepo = \""+e.remote+"\"\nref = \"master\"\n")
	if err := updateUtilities("distro"); err == nil || !strings.Contains(err.Error(), "gap pin") {
		t.Fatalf("distro update: %v", err)
	}
}

func TestUtilityBulkRefChange(t *testing.T) {
	e := newUtilityEnv(t, "schema = \"1\"\nname = \"util\"\nskills = [\"util-skill\"]\n")
	quietStdout(t)
	other := filepath.Join(t.TempDir(), "other")
	write(t, filepath.Join(other, "skills", "other-skill", "SKILL.md"), "---\nname: other-skill\ndescription: test\n---\n")
	write(t, filepath.Join(other, utilityManifestName), "schema = \"1\"\nname = \"other\"\nskills = [\"other-skill\"]\n")
	commitAll(t, other)
	write(t, userConfigFile(e.home), "[trust]\nrepos = [\""+e.remote+"\", \""+other+"\"]\n")
	if err := addUtility(e.remote, "util"); err != nil {
		t.Fatal(err)
	}
	if err := addUtility(other, "other"); err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{e.remote, other} {
		cmd := exec.Command("git", "-C", dir, "branch", "beta")
		cmd.Env = gitEnv()
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("branch: %v\n%s", err, out)
		}
	}
	config := readString(t, userConfigFile(e.home))
	write(t, userConfigFile(e.home), strings.ReplaceAll(config, "ref = \"master\"", "ref = \"beta\""))
	if err := updateUtilities(""); err != nil {
		t.Fatalf("bulk ref change: %v", err)
	}
	lk, _ := readLock(userConfigDir(e.home))
	if lk.Components["util"].Ref != "beta" || lk.Components["other"].Ref != "beta" {
		t.Fatalf("pins = %+v", lk.Components)
	}
}

func TestUtilityBulkUpdateRepairsLaterCache(t *testing.T) {
	for _, disabledBy := range []string{"enabled", "user", "project"} {
		for _, cache := range []string{"missing", "corrupt"} {
			for _, command := range []string{"bulk", "named"} {
				t.Run(disabledBy+"/"+cache+"/"+command, func(t *testing.T) {
					e := newUtilityEnv(t, declarativeUtility)
					quietStdout(t)
					other := filepath.Join(t.TempDir(), "other")
					const skill = "---\nname: other-skill\ndescription: test\n---\n"
					const manifest = "schema = \"1\"\nname = \"other\"\nskills = [\"other-skill\"]\ncommands = [\"commands/other-cmd.md\"]\n"
					write(t, filepath.Join(other, "skills", "other-skill", "SKILL.md"), skill)
					write(t, filepath.Join(other, "commands", "other-cmd.md"), "Other.\n")
					write(t, filepath.Join(other, utilityManifestName), manifest)
					commitAll(t, other)
					if run([]string{"add", e.remote, "--id", "a"}) != 0 || run([]string{"add", other, "--id", "b"}) != 0 {
						t.Fatal("initial installation failed")
					}
					switch disabledBy {
					case "user":
						config := userConfigFile(e.home)
						write(t, config, strings.Replace(readString(t, config), "[components.b]\n", "[components.b]\nenabled = false\n", 1))
					case "project":
						write(t, filepath.Join(e.project, ".gap", "gap.toml"), "[components.b]\nenabled = false\n")
					}
					destinations := append(skillDirs(e.home, "other-skill"), commandDests(e.home, "other-cmd.md")...)
					for _, dest := range destinations {
						if err := os.RemoveAll(dest); err != nil {
							t.Fatal(err)
						}
					}
					lk, err := readLock(userConfigDir(e.home))
					if err != nil {
						t.Fatal(err)
					}
					store := storePath(e.home, "b", lk.Components["b"].Commit)
					if cache == "missing" {
						if err := os.RemoveAll(store); err != nil {
							t.Fatal(err)
						}
					} else {
						write(t, filepath.Join(store, utilityManifestName), "corrupt cache\n")
					}
					args := []string{"update"}
					if command == "named" {
						args = append(args, "b")
					}
					if code := run(args); code != 0 {
						t.Fatalf("update exited %d", code)
					}
					for rel, want := range map[string]string{
						utilityManifestName:           manifest,
						"skills/other-skill/SKILL.md": skill,
						"commands/other-cmd.md":       "Other.\n",
					} {
						if got := readString(t, filepath.Join(store, rel)); got != want {
							t.Errorf("repaired cache %s = %q, want %q", rel, got, want)
						}
					}
					for _, dest := range destinations {
						if disabledBy != "enabled" {
							if _, err := os.Lstat(dest); !errors.Is(err, os.ErrNotExist) {
								t.Errorf("disabled utility installed %s: %v", dest, err)
							}
							continue
						}
						want := "Other.\n"
						if filepath.Base(dest) == "other-skill" {
							dest = filepath.Join(dest, "SKILL.md")
							want = skill
						}
						if got := readString(t, dest); got != want {
							t.Errorf("installed %s = %q, want %q", dest, got, want)
						}
					}
				})
			}
		}
	}
}

func TestUtilityUpdateRetryKeepsBaseline(t *testing.T) {
	gate := "schema = \"1\"\nname = \"util\"\nskills = [\"util-skill\"]\n[hooks]\ninstall = \"true\"\nupdate = \"test -f \\\"$HOME/ok\\\"\"\n"
	e := newUtilityEnv(t, gate)
	quietStdout(t)
	write(t, userConfigFile(e.home), "[trust]\nrepos = [\""+e.remote+"\"]\n")
	if err := addUtility(e.remote, "util"); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(e.remote, "skills", "util-skill", "SKILL.md"), "---\nname: util-skill\ndescription: v2\n---\n")
	commitAll(t, e.remote)
	if err := updateUtilities("util"); err == nil {
		t.Fatal("failing update hook reported success")
	}
	write(t, filepath.Join(e.home, "ok"), "")
	if err := updateUtilities("util"); err != nil {
		t.Fatalf("retry: %v", err)
	}
	for _, dir := range []string{".agents", ".claude"} {
		if !strings.Contains(readString(t, filepath.Join(e.home, dir, "skills", "util-skill", "SKILL.md")), "v2") {
			t.Fatalf("retry left the old %s skill", dir)
		}
	}
}

func TestUtilityList(t *testing.T) {
	e := newUtilityEnv(t, "schema = \"1\"\nname = \"util\"\nskills = [\"util-skill\"]\n")
	quietStdout(t)
	write(t, userConfigFile(e.home), "[trust]\nrepos = [\""+e.remote+"\"]\n")
	if err := addUtility(e.remote, "util"); err != nil {
		t.Fatal(err)
	}
	r, w, _ := os.Pipe()
	old := os.Stdout
	os.Stdout = w
	err := listUtilities()
	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	io.Copy(&buf, r)
	if err != nil || !strings.Contains(buf.String(), "util") || !strings.Contains(buf.String(), "user") || !strings.Contains(buf.String(), "trusted") {
		t.Fatalf("list: %v\n%s", err, buf.String())
	}
}

func TestUtilityRemove(t *testing.T) {
	e := newUtilityEnv(t, hookManifest)
	quietStdout(t)
	write(t, userConfigFile(e.home), "[components.keep]\nenabled = false\n")
	e.tty, e.answers = true, []string{"y"}
	if err := addUtility(e.remote, "util"); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(e.home, ".agents", "skills", "shared-skill", "SKILL.md"), "distro\n")
	state, _ := readState(e.home)
	owned := state.Components["util"]
	owned.Skills = append(owned.Skills, "shared-skill")
	state.Components["util"] = owned
	if err := writeState(e.home, state); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(e.repo, "gap.toml"), "schema = \"2\"\n[components.distro]\nrepo = \"github.com/x/y\"\nref = \"master\"\nnames = [\"shared-skill\"]\n")
	edited := filepath.Join(e.home, ".claude", "skills", "util-skill", "SKILL.md")
	write(t, edited, "my edit\n")
	e.answers = []string{"y"}
	if err := removeUtility("util"); err != nil {
		t.Fatal(err)
	}
	if !e.exists("uninstalled") {
		t.Fatal("uninstall hook did not run")
	}
	for _, rel := range []string{".agents/skills/util-skill", ".claude/skills/util-skill", ".claude/commands/util-cmd.md", ".omp/agent/commands/util-cmd.md", ".pi/agent/prompts/util-cmd.md"} {
		if e.exists(rel) {
			t.Fatalf("%s left behind", rel)
		}
	}
	backups, _ := filepath.Glob(filepath.Join(e.home, ".gap", "backups", "*", ".claude", "skills", "util-skill", "SKILL.md"))
	if len(backups) != 1 || readString(t, backups[0]) != "my edit\n" {
		t.Fatalf("edited skill backup: %v", backups)
	}
	if !e.exists(".agents/skills/shared-skill/SKILL.md") {
		t.Fatal("remove took a skill another source selects")
	}
	components := readConfigComponents(t, userConfigFile(e.home))
	keep, _ := components["keep"].(map[string]any)
	if _, remains := components["util"]; remains || keep["enabled"] != false {
		t.Fatalf("user registrations after remove = %+v", components)
	}
	lk, _ := readLock(userConfigDir(e.home))
	state, _ = readState(e.home)
	if _, ok := lk.Components["util"]; ok {
		t.Fatal("lock entry left")
	}
	if _, ok := state.Components["util"]; ok {
		t.Fatal("state entry left")
	}
	if err := removeUtility("util"); err == nil {
		t.Fatal("second remove succeeded")
	}
}

func TestUtilityRemoveWarnsWithoutUninstall(t *testing.T) {
	e := newUtilityEnv(t, "schema = \"1\"\nname = \"util\"\nskills = [\"util-skill\"]\n[hooks]\ninstall = \"true\"\n")
	quietStdout(t)
	e.tty, e.answers = true, []string{"y"}
	if err := addUtility(e.remote, "util"); err != nil {
		t.Fatal(err)
	}
	r, w, _ := os.Pipe()
	old := os.Stderr
	os.Stderr = w
	err := removeUtility("util")
	w.Close()
	os.Stderr = old
	var buf bytes.Buffer
	io.Copy(&buf, r)
	if err != nil || !strings.Contains(buf.String(), "no uninstall hook") {
		t.Fatalf("remove: %v\n%s", err, buf.String())
	}
}

func TestRemoveTOMLComponent(t *testing.T) {
	text := "profile = \"x\"\n\n[components.a] # mine\nrepo = \"r\"\n\n[components.a.skills.s]\nload = \"off\"\n\n[components.ab]\nenabled = false\n"
	got, found := removeTOMLComponent(text, "a")
	doc, err := parseTOML(got)
	if !found || err != nil || doc["profile"] != "x" {
		t.Fatalf("removed configuration: %q, found=%v, error=%v", got, found, err)
	}
	components, _ := doc["components"].(map[string]any)
	keep, _ := components["ab"].(map[string]any)
	if _, remains := components["a"]; remains || keep["enabled"] != false {
		t.Fatalf("component identities after removal: %+v", components)
	}
}
