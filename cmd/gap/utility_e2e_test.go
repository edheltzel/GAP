package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestUtilityLifecycle(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	tmp := t.TempDir()
	repo, home, remote := filepath.Join(tmp, "repo"), filepath.Join(tmp, "home"), filepath.Join(tmp, "remote")
	sources, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, src := range sources {
		if !strings.HasSuffix(src, "_test.go") {
			write(t, filepath.Join(repo, "cmd", "gap", src), readString(t, src))
		}
	}
	write(t, filepath.Join(repo, "go.mod"), readString(t, "../../go.mod"))
	write(t, filepath.Join(repo, "go.sum"), readString(t, "../../go.sum"))
	write(t, filepath.Join(repo, "global", "AGENTS.md"), "policy\n")
	write(t, filepath.Join(repo, "global", "skills.json"), "[]\n")
	write(t, filepath.Join(repo, "gap.toml"), "schema = \"2\"\n")
	for _, args := range [][]string{
		{"init", "-q", "-b", "master", repo},
		{"-C", repo, "add", "."},
		{"-C", repo, "-c", "user.name=gap", "-c", "user.email=gap@example.com", "commit", "-q", "-m", "fixture"},
		{"-C", repo, "update-ref", "refs/remotes/origin/master", "HEAD"},
		{"-C", repo, "symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/master"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Env = gitEnv()
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	bin := filepath.Join(tmp, "gap")
	build := exec.Command("go", "build", "-o", bin, "./cmd/gap")
	build.Dir = repo
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	manifest := func(install string) string {
		return "schema = \"1\"\nname = \"util\"\nskills = [\"util-skill\"]\ncommands = [\"commands/util-cmd.md\"]\n[hooks]\ninstall = \"" + install + "\"\nuninstall = \"touch \\\"$HOME/uninstalled\\\"\"\n"
	}
	writeUtilityRepo(t, remote, manifest("touch \\\"$HOME/installed\\\""))
	commitAll(t, remote)

	gap := func(want int, args ...string) string {
		t.Helper()
		cmd := exec.Command(bin, args...)
		cmd.Dir = tmp
		cmd.Env = append(os.Environ(), "HOME="+home, "CODEX_HOME=", "XDG_STATE_HOME=")
		out, err := cmd.CombinedOutput()
		code := 0
		if exit, ok := err.(*exec.ExitError); ok {
			code = exit.ExitCode()
		} else if err != nil {
			t.Fatalf("gap %v: %v", args, err)
		}
		if code != want {
			t.Fatalf("gap %v exited %d, want %d\n%s", args, code, want, out)
		}
		return string(out)
	}
	exists := func(rel string) bool {
		_, err := os.Stat(filepath.Join(home, rel))
		return err == nil
	}
	took := func(rel string) bool {
		ok := exists(rel)
		os.Remove(filepath.Join(home, rel))
		return ok
	}
	config := filepath.Join(home, ".config", "gap", "gap.toml")
	trust := func(on bool) {
		t.Helper()
		text, _ := readOptional(config)
		text, _, _ = strings.Cut(text, "\n[trust]")
		if on {
			text = strings.TrimRight(text, "\n") + "\n\n[trust]\nrepos = [\"" + remote + "\"]\n"
		}
		write(t, config, text)
	}
	owned := []string{".agents/skills/util-skill/SKILL.md", ".claude/skills/util-skill/SKILL.md", ".claude/commands/util-cmd.md", ".omp/agent/commands/util-cmd.md", ".pi/agent/prompts/util-cmd.md"}

	if out := gap(1, "add", remote, "--id", "util"); !strings.Contains(out, "not approved") {
		t.Fatalf("untrusted add must skip the hook:\n%s", out)
	}
	if took("installed") {
		t.Fatal("unapproved hook ran on add")
	}
	for _, rel := range owned {
		if !exists(rel) {
			t.Fatalf("add did not install %s", rel)
		}
	}
	gap(1, "sync", "--yes")
	if took("installed") {
		t.Fatal("unapproved hook ran on sync")
	}
	trust(true)
	gap(0, "sync", "--yes")
	if !took("installed") {
		t.Fatal("trusted hook did not run on sync")
	}
	snapshot := func() map[string]string {
		got := map[string]string{}
		for _, rel := range append([]string{".gap/state.json", ".config/gap/gap.lock", ".config/gap/gap.toml"}, owned...) {
			info, err := os.Stat(filepath.Join(home, rel))
			if err != nil {
				t.Fatal(err)
			}
			got[rel] = readString(t, filepath.Join(home, rel)) + info.ModTime().String()
		}
		return got
	}
	before := snapshot()
	gap(0, "sync", "--yes")
	if took("installed") {
		t.Fatal("second sync ran the hook again")
	}
	for rel, v := range snapshot() {
		if before[rel] != v {
			t.Fatalf("second sync rewrote %s", rel)
		}
	}
	if out := gap(0, "list"); !strings.Contains(out, "util") || !strings.Contains(out, "trusted") {
		t.Fatalf("list:\n%s", out)
	}

	trust(false)
	write(t, filepath.Join(remote, utilityManifestName), manifest("touch \\\"$HOME/installed\\\"; true"))
	commitAll(t, remote)
	if out := gap(1, "update", "util"); !strings.Contains(out, "not approved") {
		t.Fatalf("changed untrusted hook must skip:\n%s", out)
	}
	if took("installed") {
		t.Fatal("changed unapproved hook ran")
	}
	trust(true)
	gap(0, "update", "util")
	if !took("installed") {
		t.Fatal("trusted update did not run the hook")
	}

	write(t, filepath.Join(home, ".claude", "skills", "util-skill", "SKILL.md"), "my edit\n")
	gap(0, "remove", "util")
	if !took("uninstalled") {
		t.Fatal("uninstall hook did not run")
	}
	for _, rel := range owned {
		if exists(rel) {
			t.Fatalf("remove left %s", rel)
		}
	}
	backups, _ := filepath.Glob(filepath.Join(home, ".gap", "backups", "*", ".claude", "skills", "util-skill", "SKILL.md"))
	if len(backups) != 1 || readString(t, backups[0]) != "my edit\n" {
		t.Fatalf("edited copy backup: %v", backups)
	}
	if components := readConfigComponents(t, config); components["util"] != nil {
		t.Fatalf("registration left after remove: %+v", components["util"])
	}
	lk, err := readLock(userConfigDir(home))
	if err != nil {
		t.Fatal(err)
	}
	if pin, remains := lk.Components["util"]; remains {
		t.Fatalf("pin left after remove: %+v", pin)
	}
	gap(0, "sync", "--yes")
	for _, rel := range owned {
		if exists(rel) {
			t.Fatalf("sync after remove reinstalled %s", rel)
		}
	}
}
