package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func selectionFixture(t *testing.T) string {
	t.Helper()
	home, repo, project, remote := t.TempDir(), t.TempDir(), t.TempDir(), t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_STATE_HOME", "")
	if _, err := git(project, "init", "-q", "-b", "master"); err != nil {
		t.Fatal(err)
	}
	t.Chdir(project)
	write(t, filepath.Join(remote, "install.sh"), "#!/bin/sh\nprintf 'installed\\n' >> \"$HOME/install.log\"\n")
	commit, tree := gitFixture(t, remote, "owned by component\n")
	write(t, filepath.Join(repo, "gap.toml"), "schema = \"2\"\n[components.owned]\nrepo = \""+remote+"\"\nref = \"master\"\nnames = []\ninstall = \"sh install.sh\"\n")
	if err := writeLock(repo, lockFile{Components: map[string]lockEntryM1{"owned": {Ref: "master", Commit: commit, Tree: tree}}}); err != nil {
		t.Fatal(err)
	}
	readyEnvReviewDistro(t, repo)
	return home
}

func answers(t *testing.T, terminal bool, lines ...string) {
	t.Helper()
	oldTerm, oldLine := termReady, confirmLine
	termReady = func() bool { return terminal }
	confirmLine = func() string {
		if len(lines) == 0 {
			t.Fatal("unexpected prompt")
		}
		line := lines[0]
		lines = lines[1:]
		return line
	}
	t.Cleanup(func() { termReady, confirmLine = oldTerm, oldLine })
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func TestSyncSelection(t *testing.T) {
	policy := func(home string) string { return filepath.Join(home, ".claude", "rules", "global-agent-policy.md") }
	installLog := func(home string) string { return filepath.Join(home, "install.log") }
	for _, tc := range []struct {
		name            string
		args            []string
		terminal        bool
		lines           []string
		code            int
		policy, install bool
		stdout, stderr  string
	}{
		{"no terminal syncs all", []string{"sync"}, false, nil, 0, true, true, "gap sync will:", ""},
		{"skip component", []string{"sync", "--skip", "owned"}, true, nil, 0, true, false, "Skipped this run: owned", ""},
		{"only component", []string{"sync", "--only", "owned"}, false, nil, 0, false, true, "Skipped this run: policy", ""},
		{"picker all", []string{"sync"}, true, []string{"\n"}, 0, true, true, "[a]ll, [c]hoose, [q]uit", ""},
		{"picker choose", []string{"sync"}, true, []string{"c\n", "n\n", "y\n"}, 0, false, true, "Sync owned", ""},
		{"picker quit", []string{"sync"}, true, []string{"q\n"}, 0, false, false, "Nothing synced.", ""},
		{"picker end of input", []string{"sync"}, true, []string{""}, 0, false, false, "Nothing synced.", ""},
		{"picker end of input while choosing", []string{"sync"}, true, []string{"c\n", ""}, 0, false, false, "Nothing synced.", ""},
		{"picker none chosen", []string{"sync"}, true, []string{"c\n", "n\n", "n\n"}, 0, false, false, "Nothing synced.", ""},
		{"unknown group", []string{"sync", "--skip", "nope"}, false, nil, 1, false, false, "", "choose from: policy, owned"},
		{"legacy needs policy", []string{"sync", "--skip", "policy", "--remove-codex-legacy"}, false, nil, 1, false, false, "", "needs the policy group"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home := selectionFixture(t)
			answers(t, tc.terminal, tc.lines...)
			var code int
			stdout, stderr := captureStd(t, func() { code = run(tc.args) })
			if code != tc.code {
				t.Fatalf("exit %d, want %d\n%s%s", code, tc.code, stdout, stderr)
			}
			if exists(policy(home)) != tc.policy || exists(installLog(home)) != tc.install {
				t.Fatalf("policy=%v install=%v, want %v %v", exists(policy(home)), exists(installLog(home)), tc.policy, tc.install)
			}
			if !strings.Contains(stdout, tc.stdout) || !strings.Contains(stderr, tc.stderr) {
				t.Fatalf("stdout %q stderr %q", stdout, stderr)
			}
			if !tc.policy && !tc.install {
				if entries, _ := os.ReadDir(home); len(entries) != 0 {
					t.Fatalf("nothing selected but home has %v", entries)
				}
			}
		})
	}
}
