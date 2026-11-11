package main

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const declarativeUtility = "schema = \"1\"\nname = \"util\"\nskills = [\"util-skill\"]\ncommands = [\"commands/util-cmd.md\"]\n"

func TestUtilityReviewCommandContainment(t *testing.T) {
	for _, directory := range []bool{false, true} {
		t.Run(fmt.Sprint(directory), func(t *testing.T) {
			dir, outside := t.TempDir(), t.TempDir()
			write(t, filepath.Join(outside, "secret.md"), "private bytes")
			write(t, filepath.Join(dir, utilityManifestName), "schema = \"1\"\nname = \"util\"\ncommands = [\"commands/export.md\"]\n")
			if directory {
				write(t, filepath.Join(outside, "export.md"), "private bytes")
				if err := os.Symlink(outside, filepath.Join(dir, "commands")); err != nil {
					t.Fatal(err)
				}
			} else {
				if err := os.MkdirAll(filepath.Join(dir, "commands"), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(filepath.Join(outside, "secret.md"), filepath.Join(dir, "commands", "export.md")); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := readUtilityManifest(dir); err == nil {
				t.Fatal("manifest accepted a command outside its checkout")
			}
		})
	}
}

func TestUtilityReviewCommandOwners(t *testing.T) {
	for _, disabled := range []bool{false, true} {
		t.Run(fmt.Sprint(disabled), func(t *testing.T) {
			e := newUtilityEnv(t, declarativeUtility)
			quietStdout(t)
			if run([]string{"add", e.remote, "--id", "a"}) != 0 {
				t.Fatal("first add failed")
			}
			if disabled {
				write(t, userConfigFile(e.home), readString(t, userConfigFile(e.home))+"enabled = false\n")
			}
			before := readString(t, commandDests(e.home, "util-cmd.md")[0])
			write(t, filepath.Join(e.remote, "commands", "util-cmd.md"), "second owner's bytes\n")
			commitAll(t, e.remote)
			if run([]string{"add", e.remote, "--id", "b"}) == 0 {
				t.Fatal("second owner was accepted")
			}
			for _, path := range commandDests(e.home, "util-cmd.md") {
				if readString(t, path) != before {
					t.Fatal("collision overwrote the first owner's command")
				}
			}
		})
	}
}

func TestUtilityReviewCollisionPreflight(t *testing.T) {
	repo, home, store := t.TempDir(), t.TempDir(), t.TempDir()
	write(t, filepath.Join(repo, "global", "commands", "z.md"), "global")
	write(t, filepath.Join(store, "a.md"), "first")
	write(t, filepath.Join(store, "z.md"), "collision")
	d := distro{Components: map[string]component{"util": {ID: "util", Store: store, Commands: []string{"a.md", "z.md"}}}}
	if _, _, err := publishComponentCommands(repo, home, d, map[string]string{}, filepath.Join(home, ".gap", "published.json")); err == nil {
		t.Fatal("collision succeeded")
	}
	if exists(commandDests(home, "a.md")[0]) {
		t.Fatal("published a command before discovering a later collision")
	}
}

func TestUtilityReviewRetiredOwnership(t *testing.T) {
	e := newUtilityEnv(t, declarativeUtility)
	quietStdout(t)
	if run([]string{"add", e.remote, "--id", "util"}) != 0 {
		t.Fatal("add failed")
	}
	write(t, filepath.Join(e.remote, utilityManifestName), "schema = \"1\"\nname = \"util\"\ncommands = [\"commands/new.md\"]\n")
	write(t, filepath.Join(e.remote, "commands", "new.md"), "new\n")
	commitAll(t, e.remote)
	if run([]string{"update", "util"}) != 0 || run([]string{"remove", "util"}) != 0 {
		t.Fatal("update/remove failed")
	}
	for _, path := range append(skillDirs(e.home, "util-skill"), append(commandDests(e.home, "util-cmd.md"), commandDests(e.home, "new.md")...)...) {
		if exists(path) {
			t.Fatalf("remove left an owned copy: %s", path)
		}
	}
}

func TestUtilityReviewDeclinedOwnership(t *testing.T) {
	e := newUtilityEnv(t, declarativeUtility)
	quietStdout(t)
	if run([]string{"add", e.remote, "--id", "util"}) != 0 {
		t.Fatal("add failed")
	}
	for _, path := range commandDests(e.home, "util-cmd.md") {
		write(t, path, "user edit\n")
	}
	write(t, filepath.Join(e.remote, "commands", "util-cmd.md"), "new version\n")
	commitAll(t, e.remote)
	if run([]string{"update", "util"}) != 1 || run([]string{"remove", "util"}) != 0 {
		t.Fatal("declined update/remove returned wrong status")
	}
	for _, path := range commandDests(e.home, "util-cmd.md") {
		if exists(path) {
			t.Fatal("declined replacement lost ownership")
		}
	}
}

func TestUtilityReviewForeignSkill(t *testing.T) {
	e := newUtilityEnv(t, declarativeUtility)
	quietStdout(t)
	for _, path := range skillDirs(e.home, "util-skill") {
		write(t, filepath.Join(path, "SKILL.md"), "foreign skill\n")
	}
	if run([]string{"add", e.remote, "--id", "util"}) != 1 {
		t.Error("add reported success over foreign skill copies")
	}
	if run([]string{"remove", "util"}) != 0 {
		t.Fatal("remove failed")
	}
	for _, path := range skillDirs(e.home, "util-skill") {
		if got := readString(t, filepath.Join(path, "SKILL.md")); got != "foreign skill\n" {
			t.Fatal("remove took a foreign skill")
		}
	}
}

func TestUtilityReviewDisabledIdentity(t *testing.T) {
	e := newUtilityEnv(t, declarativeUtility)
	quietStdout(t)
	write(t, filepath.Join(e.repo, "gap.toml"), "schema = \"2\"\n[components.echo]\nrepo = \""+e.remote+"\"\nref = \"master\"\nnames = []\n")
	write(t, userConfigFile(e.home), "[components.echo]\nenabled = false\n")
	before := readString(t, userConfigFile(e.home))
	if run([]string{"add", e.remote, "--id", "echo"}) != 1 || readString(t, userConfigFile(e.home)) != before {
		t.Fatal("add corrupted a disabled distro identity")
	}
	if run([]string{"add", e.remote, "--id", "util"}) != 0 {
		t.Fatal("add failed")
	}
	write(t, filepath.Join(e.project, ".gap", "gap.toml"), "[components.util]\nenabled = false\n")
	stdout, _ := captureStd(t, func() {
		if run([]string{"list"}) != 0 {
			t.Error("list failed")
		}
	})
	if !strings.Contains(stdout, "util") {
		t.Error("list hid a project-disabled utility")
	}
	if run([]string{"update", "util"}) != 0 || run([]string{"remove", "util"}) != 0 {
		t.Fatal("project-disabled utility lost its lifecycle identity")
	}
}

func TestUtilityReviewDisabledNewUtility(t *testing.T) {
	e := newUtilityEnv(t, declarativeUtility)
	write(t, userConfigFile(e.home), "[components.util]\nrepo = \""+e.remote+"\"\nref = \"master\"\nenabled = false\n")
	d, _, err := effectivePins(e.repo, e.home, e.project, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := d.Components["util"]; ok {
		t.Fatal("new disabled utility remained selected")
	}
}

func TestUtilityReviewReservedIDs(t *testing.T) {
	for _, id := range []string{"policy", "manifest"} {
		t.Run(id, func(t *testing.T) {
			e := newUtilityEnv(t, declarativeUtility)
			quietStdout(t)
			if run([]string{"add", e.remote, "--id", id}) != 1 {
				t.Fatal("add accepted a sync group as an id")
			}
			write(t, userConfigFile(e.home), "[components."+id+"]\nrepo = \""+e.remote+"\"\nref = \"master\"\n")
			if _, _, err := effectivePins(e.repo, e.home, e.project, ""); err == nil {
				t.Fatal("user config accepted a sync group as an id")
			}
		})
	}
}

func TestUtilityReviewIncompleteUpdate(t *testing.T) {
	e := newUtilityEnv(t, declarativeUtility)
	quietStdout(t)
	if run([]string{"add", e.remote, "--id", "util"}) != 0 {
		t.Fatal("add failed")
	}
	if err := os.RemoveAll(skillDirs(e.home, "util-skill")[1]); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(e.remote, "skills", "util-skill", "SKILL.md"), "---\nname: util-skill\ndescription: changed\n---\n")
	commitAll(t, e.remote)
	code := run([]string{"update", "util"})
	if code == 0 {
		for _, path := range skillDirs(e.home, "util-skill") {
			if got := readString(t, filepath.Join(path, "SKILL.md")); !strings.Contains(got, "changed") {
				t.Fatal("successful update retained an old copy")
			}
		}
	}
}

func TestUtilityReviewUpdateOnlyHook(t *testing.T) {
	manifest := declarativeUtility + "[hooks]\nupdate = \"touch \\\"$HOME/updated\\\"\"\n"
	e := newUtilityEnv(t, manifest)
	quietStdout(t)
	write(t, userConfigFile(e.home), "[trust]\nrepos = [\""+e.remote+"\"]\n")
	if run([]string{"add", e.remote, "--id", "util"}) != 0 || e.exists("updated") {
		t.Fatal("first declarative install must not run update")
	}
	write(t, filepath.Join(e.remote, "version"), "next")
	commitAll(t, e.remote)
	if run([]string{"update", "util"}) != 0 || !e.exists("updated") {
		t.Fatal("update-only hook did not run after declarative installation")
	}
}

func TestUtilityReviewHookPersistence(t *testing.T) {
	for _, failure := range []string{"fetch", "manifest"} {
		t.Run(failure, func(t *testing.T) {
			e := newUtilityEnv(t, declarativeUtility+"[hooks]\ninstall = \"printf x >> \\\"$HOME/hook.log\\\"\"\n")
			quietStdout(t)
			commit, tree := commitAll(t, e.remote)
			bad := filepath.Join(t.TempDir(), "bad")
			badCommit, badTree := commit, tree
			if failure == "manifest" {
				writeUtilityRepo(t, bad, "schema = \"bad\"\n")
				badCommit, badTree = commitAll(t, bad)
			}
			d := distro{Components: map[string]component{
				"a": {ID: "a", Repo: e.remote, Ref: "master"},
				"b": {ID: "b", Repo: bad, Ref: "master"},
			}}
			pins := map[string]lockEntryM1{"a": {Ref: "master", Commit: commit, Tree: tree}, "b": {Ref: "master", Commit: badCommit, Tree: badTree}}
			for i := 0; i < 2; i++ {
				if _, _, err := componentEntries(e.home, e.project, d, pins); err == nil {
					t.Fatal("bad component succeeded")
				}
			}
			if got := readString(t, filepath.Join(e.home, "hook.log")); got != "x" {
				t.Fatalf("successful hook replayed after %s failure: %q", failure, got)
			}
		})
	}
}

func TestUtilityReviewTOMLRemoval(t *testing.T) {
	for _, spec := range []string{
		"[components.\"util\"]\nrepo = \"r\"\n[components.\"util\".skills.s]\nload = \"off\"\n",
		"[components . util]\nrepo = \"r\"\n",
		"components.util.repo = \"r\"\ncomponents.util.names = [\n\"a\",\n\"b\",\n]\n",
		"[components]\nutil.repo = \"r\"\n",
	} {
		t.Run(spec, func(t *testing.T) {
			text := "profile = \"keep\"\n" + spec + "[components.other]\nenabled = false\n"
			before, err := parseTOML(text)
			if err != nil {
				t.Fatal(err)
			}
			delete(before["components"].(map[string]any), "util")
			got, found := removeTOMLComponent(text, "util")
			after, err := parseTOML(got)
			if !found || err != nil || !reflect.DeepEqual(before, after) {
				t.Fatalf("removal changed the wrong TOML identity: found=%v err=%v\n%s", found, err, got)
			}
		})
	}
}

func TestUtilityReviewPreview(t *testing.T) {
	for _, tc := range []struct {
		c    component
		want string
	}{
		{component{Manifest: true, Commands: []string{"x.md"}}, "1 command"},
		{component{User: true}, "fetch utility manifest"},
	} {
		groups := syncGroups(distro{Components: map[string]component{"util": tc.c}}, nil, nil, "", 6, 0)
		if !strings.Contains(groups[1].Summary, tc.want) {
			t.Fatalf("preview hid utility work: %q", groups[1].Summary)
		}
	}
}

func TestUtilityReviewHookApprovalExpires(t *testing.T) {
	e := newUtilityEnv(t, declarativeUtility+"[hooks]\ninstall = \"sh install.sh\"\n")
	quietStdout(t)
	write(t, filepath.Join(e.remote, "install.sh"), "touch \"$HOME/first\"\n")
	commitAll(t, e.remote)
	e.tty, e.answers = true, []string{"y"}
	if run([]string{"add", e.remote, "--id", "util"}) != 0 || !e.exists("first") {
		t.Fatal("approved install failed")
	}
	write(t, filepath.Join(e.remote, "install.sh"), "touch \"$HOME/second\"\n")
	commitAll(t, e.remote)
	e.tty = false
	if run([]string{"update", "util"}) != 1 || e.exists("second") {
		t.Fatal("cached command approval authorized changed repo contents")
	}
}

func TestUtilityReviewUserRequiresManifest(t *testing.T) {
	store := t.TempDir()
	c := component{ID: "util", User: true, Named: true, Skills: []string{"util-skill"}}
	if _, err := applyManifest(c, store); err == nil {
		t.Fatal("user utility bypassed its manifest")
	}
	c.User = false
	if _, err := applyManifest(c, store); err != nil {
		t.Fatalf("distro compatibility was removed: %v", err)
	}
}

func TestUtilityReviewUserLoadBoundary(t *testing.T) {
	for _, project := range []bool{false, true} {
		t.Run(fmt.Sprint(project), func(t *testing.T) {
			e := newUtilityEnv(t, declarativeUtility)
			config := "[components.util]\nrepo = \"" + e.remote + "\"\nref = \"master\"\n"
			load := "[components.util.skills.util-skill]\nload = \"project\"\n"
			if project {
				write(t, userConfigFile(e.home), config)
				write(t, filepath.Join(e.project, ".gap", "gap.toml"), load)
			} else {
				write(t, userConfigFile(e.home), config+load)
			}
			if _, _, err := effectivePins(e.repo, e.home, e.project, ""); err == nil {
				t.Fatal("user utility accepted project-only skill copies")
			}
		})
	}
}
