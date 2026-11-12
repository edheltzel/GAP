package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSyncPinnedCommitPreview(t *testing.T) {
	const commit = "af85e07bf83baee26ffb79daa425eb84356b35b4"
	for _, tc := range []struct {
		name, ref, commit, label string
	}{
		{"commit ref", commit, commit, " @ " + commit + " (af85e07)"},
		{"branch ref", "master", commit, " @ master (af85e07)"},
		{"unpinned", "master", "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			groups := syncGroups(distro{Components: map[string]component{"owned": {}}}, map[string]lockEntryM1{"owned": {Ref: tc.ref, Commit: tc.commit}}, nil, "", 6, 0)
			stdout, stderr := captureStd(t, func() {
				printPreview(groups, map[string]bool{"policy": true, "owned": true})
			})
			want := "gap sync will:\n  policy  publish AGENTS.md to 6 harness files\n  owned   nothing to install" + tc.label + "\n"
			if stdout != want || stderr != "" {
				t.Fatalf("preview = %q, stderr = %q; want %q", stdout, stderr, want)
			}
		})
	}
}

func TestSyncPickerPreview(t *testing.T) {
	for _, cancel := range []bool{false, true} {
		t.Run(map[bool]string{false: "selected plan", true: "cancelled"}[cancel], func(t *testing.T) {
			answers(t, true)
			oldReady, oldPick := pickReady, pickGroups
			pickReady = func() bool { return true }
			pickGroups = func(_ []syncGroup, _ map[string]bool) (map[string]bool, error) {
				if cancel {
					return nil, errSyncCancelled
				}
				return map[string]bool{"policy": false, "owned": true}, nil
			}
			t.Cleanup(func() { pickReady, pickGroups = oldReady, oldPick })
			var sel map[string]bool
			var err error
			stdout, stderr := captureStd(t, func() {
				sel, err = chooseGroups([]syncGroup{{"policy", "publish policy"}, {"owned", "run installer"}}, nil, nil, false)
			})
			if cancel {
				if !errors.Is(err, errSyncCancelled) || sel != nil || stdout != "" || stderr != "" {
					t.Fatalf("cancel = %v, %v; stdout %q stderr %q", sel, err, stdout, stderr)
				}
				return
			}
			want := "gap sync will:\n  policy  publish policy  (skipped)\n  owned   run installer\n"
			if err != nil || sel["policy"] || !sel["owned"] || stdout != want || stderr != "" {
				t.Fatalf("selection = %v, %v; stdout %q stderr %q; want %q", sel, err, stdout, stderr, want)
			}
		})
	}
}

func TestDumbTerminalSkipsPicker(t *testing.T) {
	for term, want := range map[string]bool{"xterm-256color": true, "": true, "dumb": false} {
		if got := usesPicker(term, true, true); got != want {
			t.Fatalf("usesPicker(%q) = %v, want %v", term, got, want)
		}
	}
	if usesPicker("xterm", true, false) || usesPicker("xterm", false, true) {
		t.Fatal("picker needs a terminal on stdin and stdout")
	}
}

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
		{"picker quit while choosing", []string{"sync"}, true, []string{"c\n", "q\n"}, 0, false, false, "Nothing synced.", ""},
		{"picker typo re-asks", []string{"sync"}, true, []string{"c\n", "nope\n", "n\n", "y\n"}, 0, false, true, "Answer y, n, or q.", ""},
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

func TestSyncSelectionRequiresSubmittedLine(t *testing.T) {
	for _, tc := range []struct {
		name, input     string
		policy, install bool
	}{
		{"empty EOF", "", false, false},
		{"space EOF", " ", false, false},
		{"all EOF", "a", false, false},
		{"first group empty EOF", "c\n", false, false},
		{"first group space EOF", "c\n ", false, false},
		{"first group yes EOF", "c\ny", false, false},
		{"last group space EOF", "c\ny\n ", false, false},
		{"last group yes EOF", "c\ny\ny", false, false},
		{"submitted default", "\n", true, true},
		{"submitted all", "a\n", true, true},
		{"submitted groups", "c\nn\ny\n", false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home := selectionFixture(t)
			t.Setenv("TERM", "dumb")
			answers(t, true)
			confirmLine = stdinLine
			r, w, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			old := os.Stdin
			os.Stdin = r
			t.Cleanup(func() { os.Stdin = old; r.Close() })
			if _, err := w.WriteString(tc.input); err != nil {
				w.Close()
				t.Fatal(err)
			}
			w.Close()
			var code int
			stdout, stderr := captureStd(t, func() { code = run([]string{"sync"}) })
			if code != 0 || stderr != "" {
				t.Fatalf("exit %d; stdout %q stderr %q", code, stdout, stderr)
			}
			policy := filepath.Join(home, ".claude", "rules", "global-agent-policy.md")
			install := filepath.Join(home, "install.log")
			if exists(policy) != tc.policy || exists(install) != tc.install {
				t.Fatalf("policy=%v install=%v, want %v %v", exists(policy), exists(install), tc.policy, tc.install)
			}
			if !tc.policy && !tc.install {
				if !strings.Contains(stdout, "Nothing synced.") {
					t.Fatalf("missing cancellation result: %q", stdout)
				}
				entries, err := os.ReadDir(home)
				if err != nil || len(entries) != 0 {
					t.Fatalf("cancelled sync wrote to home: %v, %v", entries, err)
				}
			}
		})
	}
}

func TestStdinLineKeepsBufferedAnswers(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdin
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = old; r.Close() })
	if _, err := w.WriteString("c\nn\ny\ny"); err != nil {
		t.Fatal(err)
	}
	w.Close()
	for _, want := range []string{"c\n", "n\n", "y\n", "", ""} {
		if got := stdinLine(); got != want {
			t.Fatalf("stdinLine = %q, want %q", got, want)
		}
	}
}
