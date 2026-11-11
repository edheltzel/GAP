package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestIsTrusted(t *testing.T) {
	cases := []struct {
		repo  string
		trust []string
		want  bool
	}{
		{"github.com/edheltzel/Echo", []string{"edheltzel/echo"}, true},
		{"github.com/edheltzel/Echo", []string{"edheltzel/*"}, true},
		{"github.com/edheltzel/Echo", []string{"github.com/edheltzel/*"}, true},
		{"github.com/other/Echo", []string{"edheltzel/*"}, false},
		{"github.com/edheltzel-evil/x", []string{"edheltzel/*"}, false},
		{"/tmp/remote", []string{"/tmp/remote"}, true},
		{"github.com/edheltzel/Echo", nil, false},
	}
	for _, c := range cases {
		if got := isTrusted(c.repo, c.trust); got != c.want {
			t.Errorf("isTrusted(%q, %v) = %v", c.repo, c.trust, got)
		}
	}
}

func TestHookTrust(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_STATE_HOME", "")
	repo, project := t.TempDir(), t.TempDir()
	remote := filepath.Join(t.TempDir(), "remote")
	marker := filepath.Join(home, "hook-ran")
	manifest := func(cmd string) string {
		return "schema = \"1\"\nname = \"util\"\nskills = [\"util-skill\"]\n[hooks]\ninstall = \"" + cmd + "\"\n"
	}
	writeUtilityRepo(t, remote, manifest("touch \\\"$HOME/hook-ran\\\""))
	commit, tree := commitAll(t, remote)
	write(t, filepath.Join(repo, "gap.toml"), "schema = \"2\"\n")
	userToml := "[components.util]\nrepo = \"" + remote + "\"\nref = \"master\"\n"
	write(t, filepath.Join(userConfigDir(home), "gap.toml"), userToml)
	pin := func(commit, tree string) {
		t.Helper()
		if err := writeLock(userConfigDir(home), lockFile{Components: map[string]lockEntryM1{"util": {Ref: "master", Commit: commit, Tree: tree}}}); err != nil {
			t.Fatal(err)
		}
	}
	answers := []string{}
	tty := false
	oldReady, oldLine := termReady, confirmLine
	t.Cleanup(func() { termReady, confirmLine = oldReady, oldLine })
	termReady = func() bool { return tty }
	confirmLine = func() string {
		if len(answers) == 0 {
			t.Fatal("unexpected prompt")
		}
		a := answers[0]
		answers = answers[1:]
		return a + "\n"
	}
	run := func() bool {
		t.Helper()
		d, lk, err := effectivePins(repo, home, project, "")
		if err != nil {
			t.Fatal(err)
		}
		pins, err := lockFor(d, lk)
		if err != nil {
			t.Fatal(err)
		}
		_, ok, err := componentEntries(home, project, d, pins)
		if err != nil {
			t.Fatal(err)
		}
		return ok
	}
	ran := func() bool {
		_, err := os.Stat(marker)
		os.Remove(marker)
		return err == nil
	}
	pin(commit, tree)

	if run() || ran() {
		t.Fatal("untrusted hook with no terminal must skip and fail the run")
	}
	tty, answers = true, []string{"y"}
	if !run() || !ran() {
		t.Fatal("approved hook must run")
	}
	if !run() || ran() {
		t.Fatal("unchanged pin must not run the hook again")
	}
	tty = false
	writeUtilityRepo(t, remote, manifest("touch \\\"$HOME/hook-ran\\\"; true"))
	commit, tree = commitAll(t, remote)
	pin(commit, tree)
	if run() || ran() {
		t.Fatal("changed hook with no terminal must skip and fail the run")
	}
	write(t, filepath.Join(userConfigDir(home), "gap.toml"), userToml+"\n[trust]\nrepos = [\""+remote+"\"]\n")
	if !run() || !ran() {
		t.Fatal("trusted repo must run without a prompt")
	}
}
