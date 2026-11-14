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

func TestUtilitySkillBaseline(t *testing.T) {
	for _, outcome := range []string{"declined", "publication-failed", "partial"} {
		t.Run(outcome, func(t *testing.T) {
			manifest := declarativeUtility + "[hooks]\ninstall = \"true\"\nupdate = \"touch \\\"$HOME/update-ran\\\"\"\n"
			if outcome == "partial" {
				manifest = "schema = \"1\"\nname = \"util\"\nskills = [\"util-skill\", \"other-skill\"]\n"
			}
			e := newUtilityEnv(t, manifest)
			quietStdout(t)
			if outcome == "partial" {
				write(t, filepath.Join(e.remote, "skills", "other-skill", "SKILL.md"), "other skill\n")
				commitAll(t, e.remote)
			}
			e.tty, e.answers = true, []string{"y"}
			if run([]string{"add", e.remote, "--id", "util"}) != 0 {
				t.Fatal("initial installation failed")
			}
			original := readString(t, filepath.Join(e.home, ".agents", "skills", "util-skill", "SKILL.md"))
			write(t, filepath.Join(e.remote, "skills", "util-skill", "SKILL.md"), "version Q\n")
			commitAll(t, e.remote)
			ledgerPath := filepath.Join(e.home, ".gap", "published.json")
			var ledger string
			if outcome == "publication-failed" {
				ledger = readString(t, ledgerPath)
				if err := os.Remove(ledgerPath); err != nil {
					t.Fatal(err)
				}
				if err := os.Mkdir(ledgerPath, 0o755); err != nil {
					t.Fatal(err)
				}
				e.answers = []string{"y"}
			} else {
				e.tty = false
			}
			if outcome == "partial" {
				write(t, filepath.Join(e.home, ".claude", "skills", "other-skill", "SKILL.md"), "user edit\n")
				write(t, userConfigFile(e.home), readString(t, userConfigFile(e.home))+fmt.Sprintf("\n[trust]\nrepos = [%q]\n", e.remote))
			}
			if run([]string{"update", "util"}) != 1 {
				t.Fatal("incomplete update reported success")
			}
			want := "version Q\n"
			if outcome == "publication-failed" {
				want = original
				if !e.exists("update-ran") {
					t.Fatal("publication failure did not follow a completed hook")
				}
				if err := os.Remove(ledgerPath); err != nil {
					t.Fatal(err)
				}
				write(t, ledgerPath, ledger)
			}
			for _, dir := range skillDirs(e.home, "util-skill") {
				if got := readString(t, filepath.Join(dir, "SKILL.md")); got != want {
					t.Fatalf("intermediate skill = %q, want %q", got, want)
				}
			}
			if outcome != "partial" {
				write(t, userConfigFile(e.home), readString(t, userConfigFile(e.home))+fmt.Sprintf("\n[trust]\nrepos = [%q]\n", e.remote))
			}
			if outcome != "publication-failed" {
				want = "version R\n"
				write(t, filepath.Join(e.remote, "skills", "util-skill", "SKILL.md"), want)
				commitAll(t, e.remote)
			} else {
				want = "version Q\n"
			}
			e.tty = false
			wantCode := 0
			if outcome == "partial" {
				wantCode = 1
			}
			if code := run([]string{"update", "util"}); code != wantCode {
				t.Errorf("retry exited %d, want %d", code, wantCode)
			}
			for _, dir := range skillDirs(e.home, "util-skill") {
				if got := readString(t, filepath.Join(dir, "SKILL.md")); got != want {
					t.Errorf("retry skill = %q, want %q", got, want)
				}
			}
			if outcome == "partial" && readString(t, filepath.Join(e.home, ".claude", "skills", "other-skill", "SKILL.md")) != "user edit\n" {
				t.Error("retry overwrote an edited sibling skill")
			}
		})
	}
}

func TestUtilityCommandOwnersRequireManifests(t *testing.T) {
	for _, source := range []string{"missing", "changed", "unpinned"} {
		t.Run(source, func(t *testing.T) {
			e := newUtilityEnv(t, declarativeUtility+"[hooks]\ninstall = \"exit 1\"\n")
			quietStdout(t)
			e.tty, e.answers = true, []string{"y"}
			if run([]string{"add", e.remote, "--id", "a"}) != 1 {
				t.Fatal("failing install reported success")
			}
			lk, err := readLock(userConfigDir(e.home))
			if err != nil {
				t.Fatal(err)
			}
			store := storePath(e.home, "a", lk.Components["a"].Commit)
			switch source {
			case "missing":
				if err := os.RemoveAll(store); err != nil {
					t.Fatal(err)
				}
			case "changed":
				write(t, filepath.Join(store, utilityManifestName), "changed cache\n")
			case "unpinned":
				delete(lk.Components, "a")
				if err := writeLock(userConfigDir(e.home), lk); err != nil {
					t.Fatal(err)
				}
			}
			other := filepath.Join(t.TempDir(), "other")
			writeUtilityRepo(t, other, declarativeUtility)
			commitAll(t, other)
			e.tty = false
			if run([]string{"add", other, "--id", "b"}) != 1 {
				t.Error("add accepted unresolved command ownership")
			}
			if _, registered := readConfigComponents(t, userConfigFile(e.home))["b"]; registered {
				t.Error("rejected add registered the conflicting utility")
			}
			for _, dest := range commandDests(e.home, "util-cmd.md") {
				if exists(dest) {
					t.Error("rejected add published a conflicting command")
				}
			}
		})
	}
}

func TestUtilityDoctorDoesNotProbeUserEcho(t *testing.T) {
	for _, trusted := range []bool{false, true} {
		t.Run(fmt.Sprint(trusted), func(t *testing.T) {
			manifest := fmt.Sprintf("schema = \"1\"\nname = \"custom echo\"\n[hooks]\ndoctor = %q\n", "touch \"$HOME/approved-doctor\"")
			e := newUtilityEnv(t, manifest)
			quietStdout(t)
			write(t, filepath.Join(e.remote, "scripts", "install.sh"), "#!/bin/sh\ntouch \"$HOME/undisclosed-probe\"\nprintf 'codex\\n'\n")
			if err := os.Chmod(filepath.Join(e.remote, "scripts", "install.sh"), 0o755); err != nil {
				t.Fatal(err)
			}
			commitAll(t, e.remote)
			if trusted {
				write(t, userConfigFile(e.home), fmt.Sprintf("[trust]\nrepos = [%q]\n", e.remote))
			}
			if run([]string{"add", e.remote, "--id", "echo"}) != 0 {
				t.Fatal("adding custom Echo failed")
			}
			e.tty, e.answers = true, []string{"y"}
			if run([]string{"doctor"}) != 0 || !e.exists("approved-doctor") {
				t.Fatal("the declared doctor hook did not run")
			}
			if e.exists("undisclosed-probe") {
				t.Fatal("doctor ran an undeclared adapter probe for a user utility")
			}
		})
	}
}

func TestUtilityDeclarativeRemovalVersion(t *testing.T) {
	for _, outcome := range []string{"declarative", "declined", "failed", "updated"} {
		t.Run(outcome, func(t *testing.T) {
			manifest := func(version, update string) string {
				text := declarativeUtility + fmt.Sprintf("[hooks]\nuninstall = %q\n", "touch \"$HOME/removed-"+version+"\"")
				if update != "" {
					text += fmt.Sprintf("update = %q\n", update)
				}
				return text
			}
			e := newUtilityEnv(t, manifest("v1", ""))
			quietStdout(t)
			first, err := git(e.remote, "rev-parse", "HEAD")
			if err != nil {
				t.Fatal(err)
			}
			if run([]string{"add", e.remote, "--id", "util"}) != 0 {
				t.Fatal("initial declarative installation failed")
			}
			update := ""
			if outcome == "declined" || outcome == "updated" {
				update = "true"
			} else if outcome == "failed" {
				update = "exit 1"
			}
			write(t, filepath.Join(e.remote, utilityManifestName), manifest("v2", update))
			write(t, filepath.Join(e.remote, "commands", "util-cmd.md"), "v2 command\n")
			second, _ := commitAll(t, e.remote)
			e.tty, e.answers = true, []string{"y"}
			wantCode, installed, wantCommit := 0, "v2", second
			if outcome == "declined" || outcome == "failed" {
				wantCode, installed, wantCommit = 1, "v1", first
			}
			if outcome == "declined" {
				e.answers = []string{"n"}
			}
			if code := run([]string{"update", "util"}); code != wantCode {
				t.Fatalf("update exited %d, want %d", code, wantCode)
			}
			state, err := readState(e.home)
			if err != nil || state.Components["util"].Installed != wantCommit {
				t.Errorf("installed commit = %+v, want %s: %v", state.Components["util"], wantCommit, err)
			}
			if wantCode == 0 {
				for _, path := range commandDests(e.home, "util-cmd.md") {
					if got := readString(t, path); got != "v2 command\n" {
						t.Fatalf("installed command = %q", got)
					}
				}
			}
			if err := os.Rename(e.remote, e.remote+"-offline"); err != nil {
				t.Fatal(err)
			}
			e.answers = []string{"y"}
			if run([]string{"remove", "util"}) != 0 {
				t.Fatal("removal failed")
			}
			if !e.exists("removed-" + installed) {
				t.Fatalf("removal did not use the %s uninstall hook", installed)
			}
		})
	}
}

func TestUtilityRemovalRejectsUnresolvedSharing(t *testing.T) {
	for _, source := range []string{"missing", "changed", "unpinned"} {
		t.Run(source, func(t *testing.T) {
			manifest := fmt.Sprintf("schema = \"1\"\nname = \"util\"\nskills = [\"util-skill\"]\n[hooks]\nuninstall = %q\n", "touch \"$HOME/uninstall-ran\"")
			e := newUtilityEnv(t, manifest)
			quietStdout(t)
			write(t, userConfigFile(e.home), fmt.Sprintf("[trust]\nrepos = [%q]\n", e.remote))
			for _, id := range []string{"a", "b"} {
				if run([]string{"add", e.remote, "--id", id}) != 0 {
					t.Fatalf("add %s failed", id)
				}
			}
			lk, err := readLock(userConfigDir(e.home))
			if err != nil {
				t.Fatal(err)
			}
			store := storePath(e.home, "b", lk.Components["b"].Commit)
			switch source {
			case "missing":
				if err := os.RemoveAll(store); err != nil {
					t.Fatal(err)
				}
			case "changed":
				write(t, filepath.Join(store, "skills", "util-skill", "SKILL.md"), "changed cache\n")
			case "unpinned":
				delete(lk.Components, "b")
				if err := writeLock(userConfigDir(e.home), lk); err != nil {
					t.Fatal(err)
				}
			}
			beforeConfig := readString(t, userConfigFile(e.home))
			beforeState := readString(t, statePath(e.home))
			if run([]string{"remove", "a"}) != 1 {
				t.Error("removal accepted unresolved sharing")
			}
			if e.exists("uninstall-ran") {
				t.Error("uninstall ran before resolving remaining selections")
			}
			for _, path := range skillDirs(e.home, "util-skill") {
				if !exists(path) {
					t.Error("removal took a remaining utility's shared skill")
				}
			}
			if readString(t, userConfigFile(e.home)) != beforeConfig || readString(t, statePath(e.home)) != beforeState {
				t.Error("unresolved removal changed registration or ownership")
			}
		})
	}
}

func TestUtilityRemovalUsesInstalledHook(t *testing.T) {
	for _, outcome := range []string{"declined", "failed", "successful"} {
		t.Run(outcome, func(t *testing.T) {
			manifest := func(version, update string) string {
				return declarativeUtility + fmt.Sprintf("[hooks]\ninstall = %q\nupdate = %q\nuninstall = %q\n",
					"printf "+version+" > \"$HOME/installed-"+version+"\"", update,
					"mv \"$HOME/installed-"+version+"\" \"$HOME/removed-"+version+"\"")
			}
			e := newUtilityEnv(t, manifest("v1", "true"))
			quietStdout(t)
			e.tty, e.answers = true, []string{"y"}
			if run([]string{"add", e.remote, "--id", "util"}) != 0 {
				t.Fatal("initial install failed")
			}
			update := "mv \"$HOME/installed-v1\" \"$HOME/installed-v2\" && printf v2 > \"$HOME/installed-v2\""
			if outcome == "failed" {
				update = "exit 1"
			}
			write(t, filepath.Join(e.remote, utilityManifestName), manifest("v2", update))
			write(t, filepath.Join(e.remote, "commands", "util-cmd.md"), "v2 command\n")
			commit, _ := commitAll(t, e.remote)
			e.answers = []string{"y"}
			wantCode, installed := 0, "v2"
			if outcome != "successful" {
				wantCode, installed = 1, "v1"
			}
			if outcome == "declined" {
				e.answers = []string{"n"}
			}
			if code := run([]string{"update", "util"}); code != wantCode {
				t.Fatalf("update exited %d, want %d", code, wantCode)
			}
			lock, err := readLock(userConfigDir(e.home))
			if err != nil || lock.Components["util"].Commit != commit {
				t.Fatalf("desired pin did not advance: %+v, %v", lock, err)
			}
			if err := os.Rename(e.remote, e.remote+"-offline"); err != nil {
				t.Fatal(err)
			}
			e.answers = []string{"y"}
			if run([]string{"remove", "util"}) != 0 {
				t.Fatal("removal did not clean up the installed hook version")
			}
			if got := readString(t, filepath.Join(e.home, "removed-"+installed)); got != installed {
				t.Fatalf("removed hook side effect = %q, want %q", got, installed)
			}
			if e.exists("installed-v1") || e.exists("installed-v2") {
				t.Fatal("installed hook side effect remains")
			}
			for _, rel := range []string{".agents/skills/util-skill", ".claude/skills/util-skill", ".claude/commands/util-cmd.md", ".omp/agent/commands/util-cmd.md", ".pi/agent/prompts/util-cmd.md"} {
				if e.exists(rel) {
					t.Fatalf("removal left %s", rel)
				}
			}
			if outcome == "declined" {
				backups, err := filepath.Glob(filepath.Join(e.home, ".gap", "backups", "*", ".claude", "commands", "util-cmd.md"))
				if err != nil || len(backups) != 1 || readString(t, backups[0]) != "v2 command\n" {
					t.Fatalf("desired command backup = %v, %v", backups, err)
				}
			}
			if _, remains := readConfigComponents(t, userConfigFile(e.home))["util"]; remains {
				t.Fatal("removal left the utility registered")
			}
		})
	}
}

func TestUtilityRemovalRecoversUnverifiedInstalledCache(t *testing.T) {
	for _, cache := range []string{"edited", "recommitted", "missing"} {
		t.Run(cache, func(t *testing.T) {
			manifest := declarativeUtility + "[hooks]\nuninstall = \"touch \\\"$HOME/removed-v1\\\"\"\n"
			e := newUtilityEnv(t, manifest)
			quietStdout(t)
			if run([]string{"add", e.remote, "--id", "util"}) != 0 {
				t.Fatal("initial installation failed")
			}
			state, err := readState(e.home)
			if err != nil {
				t.Fatal(err)
			}
			store := storePath(e.home, "util", state.Components["util"].Installed)
			write(t, filepath.Join(e.remote, utilityManifestName), manifest+"update = \"true\"\n")
			commitAll(t, e.remote)
			if run([]string{"update", "util"}) != 1 {
				t.Fatal("unapproved update should leave the previous version installed")
			}
			if cache == "missing" {
				if err := os.Rename(store, store+"-missing"); err != nil {
					t.Fatal(err)
				}
			} else {
				write(t, filepath.Join(store, utilityManifestName), declarativeUtility+"[hooks]\nuninstall = \"touch \\\"$HOME/unverified-uninstall\\\"\"\n")
				if cache == "recommitted" {
					commitAll(t, store)
				}
			}
			if err := os.Rename(e.remote, e.remote+"-offline"); err != nil {
				t.Fatal(err)
			}
			beforeConfig := readConfigComponents(t, userConfigFile(e.home))
			beforeState, err := readState(e.home)
			if err != nil {
				t.Fatal(err)
			}
			e.tty, e.answers = true, []string{"y"}
			if run([]string{"remove", "util"}) != 1 {
				t.Fatal("removal accepted an unverified installed cache without a remote")
			}
			afterState, err := readState(e.home)
			if err != nil || !reflect.DeepEqual(beforeState, afterState) || !reflect.DeepEqual(beforeConfig, readConfigComponents(t, userConfigFile(e.home))) {
				t.Fatalf("failed removal changed registration or ownership: %v", err)
			}
			if e.exists("removed-v1") || e.exists("unverified-uninstall") {
				t.Fatal("removal ran a hook before verifying the installed cache")
			}
			if err := os.Rename(e.remote+"-offline", e.remote); err != nil {
				t.Fatal(err)
			}
			if run([]string{"remove", "util"}) != 0 {
				t.Fatal("removal did not recover the installed version from the remote")
			}
			if !e.exists("removed-v1") || e.exists("unverified-uninstall") {
				t.Fatal("removal did not use the verified installed version's hook")
			}
			if _, remains := readConfigComponents(t, userConfigFile(e.home))["util"]; remains {
				t.Fatal("removal left the utility registered")
			}
		})
	}
}

func TestUtilityRemovalWithoutInstalledHook(t *testing.T) {
	for _, outcome := range []string{"declined", "failed"} {
		t.Run(outcome, func(t *testing.T) {
			manifest := hookManifest
			if outcome == "failed" {
				manifest = declarativeUtility + "[hooks]\ninstall = \"exit 1\"\nuninstall = \"touch \\\"$HOME/uninstalled\\\"\"\n"
			}
			e := newUtilityEnv(t, manifest)
			quietStdout(t)
			if outcome == "failed" {
				e.tty, e.answers = true, []string{"y"}
			}
			if run([]string{"add", e.remote, "--id", "util"}) != 1 {
				t.Fatal("initial installation should fail or skip its hook")
			}
			e.tty = true
			if run([]string{"remove", "util"}) != 0 {
				t.Fatal("removal should not need an uninstall hook for an uninstalled utility")
			}
			if e.exists("uninstalled") {
				t.Fatal("uninstall hook ran without a successful installation")
			}
		})
	}
}

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
		groups := syncGroups(distro{Components: map[string]component{"util": tc.c}}, nil, "", 6, 0)
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

func TestUtilityReviewSkillContainment(t *testing.T) {
	for _, kind := range []string{"ancestor", "skill file"} {
		t.Run(kind, func(t *testing.T) {
			e := newUtilityEnv(t, "schema = \"1\"\nname = \"private\"\nskills = [\"internal\"]\n")
			quietStdout(t)
			outside := filepath.Join(e.home, ".empryo")
			write(t, filepath.Join(outside, "skills", "internal", "SKILL.md"), "private skill\n")
			write(t, filepath.Join(outside, "skills", "internal", "notes.md"), "private notes\n")
			if kind == "ancestor" {
				if err := os.Symlink(outside, filepath.Join(e.remote, "skills", "link")); err != nil {
					t.Fatal(err)
				}
			} else {
				if err := os.MkdirAll(filepath.Join(e.remote, "skills", "internal"), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(filepath.Join(outside, "skills", "internal", "SKILL.md"), filepath.Join(e.remote, "skills", "internal", "SKILL.md")); err != nil {
					t.Fatal(err)
				}
			}
			commitAll(t, e.remote)
			if _, err := readUtilityManifest(e.remote); err == nil {
				t.Error("manifest accepted a skill outside its checkout")
			}
			if run([]string{"add", e.remote, "--id", "private"}) != 1 {
				t.Error("add accepted an external skill")
			}
			for _, path := range skillDirs(e.home, "internal") {
				if exists(path) {
					t.Errorf("external skill was copied into %s", path)
				}
			}
		})
	}
}

func TestUtilityReviewLiteralGlobalCommands(t *testing.T) {
	for _, tc := range []struct{ kind, checkout string }{{"command", "GAP[1]"}, {"directory", "GAP[1]"}, {"directory", "GAP"}} {
		t.Run(tc.kind+"/"+tc.checkout, func(t *testing.T) {
			e := newUtilityEnv(t, declarativeUtility)
			quietStdout(t)
			repo := filepath.Join(e.repo, tc.checkout)
			write(t, filepath.Join(repo, "global", "AGENTS.md"), "policy\n")
			write(t, filepath.Join(repo, "gap.toml"), "schema = \"2\"\n")
			sourceRootFor = repo
			path := filepath.Join(repo, "global", "commands", "util-cmd.md")
			if tc.kind == "directory" {
				if err := os.MkdirAll(path, 0o755); err != nil {
					t.Fatal(err)
				}
				if commandCount(repo) != 0 {
					t.Error("preview counted a directory as a command")
				}
				if run([]string{"add", e.remote, "--id", "util"}) != 0 || run([]string{"remove", "util"}) != 0 {
					t.Fatal("a directory incorrectly reserved a command name")
				}
				for _, dest := range commandDests(e.home, "util-cmd.md") {
					if exists(dest) {
						t.Error("a directory prevented removal of a utility command")
					}
				}
				return
			}
			write(t, path, "global command\n")
			ledgerPath := filepath.Join(e.home, ".gap", "published.json")
			if ok, err := publishCommands(repo, e.home, map[string]string{}, ledgerPath); err != nil || !ok {
				t.Fatalf("global publication failed: %v", err)
			}
			if commandCount(repo) != 1 {
				t.Error("preview omitted a literal checkout's command")
			}
			if run([]string{"add", e.remote, "--id", "util"}) != 1 {
				t.Error("global command collision was accepted")
			}
			for _, dest := range commandDests(e.home, "util-cmd.md") {
				if readString(t, dest) != "global command\n" {
					t.Error("global command was overwritten")
				}
			}
		})
	}
}

func TestUtilityReviewSharedOwnership(t *testing.T) {
	for _, order := range [][]string{{"a", "b"}, {"b", "a"}} {
		t.Run(strings.Join(order, " then "), func(t *testing.T) {
			e := newUtilityEnv(t, "schema = \"1\"\nname = \"util\"\nskills = [\"util-skill\"]\n")
			quietStdout(t)
			for _, id := range []string{"a", "b"} {
				if run([]string{"add", e.remote, "--id", id}) != 0 {
					t.Fatalf("add %s failed", id)
				}
			}
			if run([]string{"remove", order[0]}) != 0 {
				t.Fatal("first removal failed")
			}
			for _, path := range skillDirs(e.home, "util-skill") {
				if !exists(path) {
					t.Error("first removal took a shared skill")
				}
			}
			if run([]string{"remove", order[1]}) != 0 {
				t.Fatal("last removal failed")
			}
			for _, path := range skillDirs(e.home, "util-skill") {
				if exists(path) {
					t.Error("last removal orphaned a shared skill")
				}
			}
		})
	}
}

func TestUtilityReviewSharedCommandOwnership(t *testing.T) {
	e := newUtilityEnv(t, "schema = \"1\"\nname = \"util\"\ncommands = [\"commands/util-cmd.md\"]\n")
	quietStdout(t)
	if run([]string{"add", e.remote, "--id", "a"}) != 0 {
		t.Fatal("first add failed")
	}
	if run([]string{"add", e.remote, "--id", "b"}) != 1 {
		t.Fatal("command collision was accepted")
	}
	lk, err := readLock(userConfigDir(e.home))
	if err != nil {
		t.Fatal(err)
	}
	write(t, userConfigFile(e.home), readString(t, userConfigFile(e.home))+fmt.Sprintf("\n[components.b]\nrepo = %q\nref = \"master\"\n", e.remote))
	if err := setUserPin(e.home, "b", lk.Components["a"]); err != nil {
		t.Fatal(err)
	}
	if run([]string{"remove", "a"}) != 0 {
		t.Fatal("first removal failed")
	}
	for _, path := range commandDests(e.home, "util-cmd.md") {
		if !exists(path) {
			t.Error("first removal took a shared command")
		}
	}
	if run([]string{"remove", "b"}) != 0 {
		t.Fatal("last removal failed")
	}
	for _, path := range commandDests(e.home, "util-cmd.md") {
		if exists(path) {
			t.Error("last removal orphaned a shared command")
		}
	}
}

func TestUtilityReviewExistingOverrideIdentity(t *testing.T) {
	for _, declaration := range []string{
		"[components.tools]\nenabled = false\n",
		"[components . \"tools\"]\nenabled = false\n",
		"components.tools.enabled = false\n",
	} {
		t.Run(declaration, func(t *testing.T) {
			e := newUtilityEnv(t, declarativeUtility)
			quietStdout(t)
			write(t, userConfigFile(e.home), declaration)
			if run([]string{"add", e.remote, "--id", "tools"}) != 1 {
				t.Error("add accepted an existing configuration identity")
			}
			if got := readString(t, userConfigFile(e.home)); got != declaration {
				t.Error("refused add modified the existing configuration")
			}
			if _, err := parseTOML(readString(t, userConfigFile(e.home))); err != nil {
				t.Fatalf("add corrupted TOML: %v", err)
			}
		})
	}
}

func TestUtilityReviewProjectUpdateRef(t *testing.T) {
	for _, disabled := range []bool{false, true} {
		t.Run(fmt.Sprint(disabled), func(t *testing.T) {
			e := newUtilityEnv(t, declarativeUtility)
			quietStdout(t)
			if run([]string{"add", e.remote, "--id", "util"}) != 0 {
				t.Fatal("add failed")
			}
			write(t, filepath.Join(e.remote, "skills", "util-skill", "SKILL.md"), "---\nname: util-skill\ndescription: beta\n---\n")
			commit, _ := commitAll(t, e.remote)
			if _, err := git(e.remote, "branch", "beta", commit); err != nil {
				t.Fatal(err)
			}
			write(t, filepath.Join(e.project, ".gap", "gap.toml"), fmt.Sprintf("[components.util]\nref = \"beta\"\nenabled = %t\n", !disabled))
			if run([]string{"update", "util"}) != 0 {
				t.Error("update failed under a project ref override")
			}
			lk, err := readLock(userConfigDir(e.home))
			if err != nil || lk.Components["util"].Ref != "beta" || lk.Components["util"].Commit != commit {
				t.Fatalf("update did not honor the effective ref: %+v, %v", lk, err)
			}
		})
	}
}

func TestUtilityReviewRemovalStatErrors(t *testing.T) {
	for _, rel := range []string{".agents/skills", ".omp/agent/commands"} {
		t.Run(rel, func(t *testing.T) {
			e := newUtilityEnv(t, declarativeUtility)
			quietStdout(t)
			if run([]string{"add", e.remote, "--id", "util"}) != 0 {
				t.Fatal("add failed")
			}
			beforeConfig := readString(t, userConfigFile(e.home))
			beforeState := readString(t, statePath(e.home))
			path := filepath.Join(e.home, filepath.FromSlash(rel))
			if err := os.Rename(path, path+"-saved"); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(filepath.Base(path), path); err != nil {
				t.Fatal(err)
			}
			if run([]string{"remove", "util"}) != 1 {
				t.Error("remove swallowed a filesystem error")
			}
			if readString(t, userConfigFile(e.home)) != beforeConfig || readString(t, statePath(e.home)) != beforeState {
				t.Error("failed removal dropped its registration or ownership")
			}
		})
	}
}
