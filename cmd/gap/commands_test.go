package main

import (
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestComponentCommandSync(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_STATE_HOME", "")
	repo, project := t.TempDir(), t.TempDir()
	remote := filepath.Join(t.TempDir(), "remote")
	writeUtilityRepo(t, remote, "schema = \"1\"\nname = \"util\"\nskills = [\"util-skill\"]\ncommands = [\"commands/util-cmd.md\"]\n")
	commit, tree := commitAll(t, remote)
	write(t, filepath.Join(repo, "gap.toml"), "schema = \"2\"\n")
	write(t, filepath.Join(userConfigDir(home), "gap.toml"), "[components.util]\nrepo = \""+remote+"\"\nref = \"master\"\n")
	if err := writeLock(userConfigDir(home), lockFile{Components: map[string]lockEntryM1{"util": {Ref: "master", Commit: commit, Tree: tree}}}); err != nil {
		t.Fatal(err)
	}
	d, lk, err := effectivePins(repo, home, project, "")
	if err != nil {
		t.Fatal(err)
	}
	pins, err := lockFor(d, lk)
	if err != nil {
		t.Fatal(err)
	}
	entries, _, err := componentEntries(home, project, d, pins)
	if err != nil {
		t.Fatal(err)
	}
	if !installSkills(home, entries) {
		t.Fatal("skill installation failed")
	}
	ledgerPath := filepath.Join(home, ".gap", "published.json")
	ledger, err := readLedger(ledgerPath)
	if err != nil {
		t.Fatal(err)
	}
	owned, ok, err := publishComponentCommands(repo, home, d, ledger, ledgerPath)
	if err != nil || !ok {
		t.Fatalf("publish: %v %v", ok, err)
	}
	wantCommands := []string{
		filepath.Join(home, ".claude", "commands", "util-cmd.md"),
		filepath.Join(home, ".omp", "agent", "commands", "util-cmd.md"),
		filepath.Join(home, ".pi", "agent", "prompts", "util-cmd.md"),
	}
	for _, dest := range wantCommands {
		if got := readString(t, dest); got != "Run the utility.\n" {
			t.Fatalf("%s = %q", dest, got)
		}
	}
	if err := recordDeclarativeInstalls(home, project, d, owned, nil); err != nil {
		t.Fatal(err)
	}
	state, err := readState(home)
	if err != nil {
		t.Fatal(err)
	}
	got := state.Components["util"]
	slices.Sort(got.Commands)
	if !slices.Equal(got.Commands, wantCommands) || len(got.Skills) != 1 || got.Skills[0] != "util-skill" {
		t.Fatalf("ownership = %+v", got)
	}

	write(t, filepath.Join(repo, "global", "commands", "util-cmd.md"), "global\n")
	if _, _, err := publishComponentCommands(repo, home, d, ledger, ledgerPath); err == nil || !strings.Contains(err.Error(), "published by both") {
		t.Fatalf("clash: %v", err)
	}
}

func TestCommandSync(t *testing.T) {
	const command = "---\nargument-hint: [path]\n---\nRead $ARGUMENTS and [rules](docs/agents/dox.md)."
	for _, tc := range []struct {
		name, checkout, ignored string
	}{
		{"unchanged bytes", "repo", ""},
		{"literal checkout path", "GAP[1]", ""},
		{"ignored command", "repo", "draft.local.md"},
		{"ignored hidden command", "repo", ".local.md"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tmp := t.TempDir()
			repo, home := filepath.Join(tmp, tc.checkout), filepath.Join(tmp, "home")
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
			write(t, filepath.Join(repo, "global", "AGENTS.md"), source)
			write(t, filepath.Join(repo, ".gitignore"), "*.local.md\n*.local.txt\n")
			commands := filepath.Join(repo, "global", "commands")
			write(t, filepath.Join(commands, "state-docs.md"), command)
			write(t, filepath.Join(commands, ".empty.md"), "")
			write(t, filepath.Join(commands, "notes.txt"), "not a command")
			write(t, filepath.Join(commands, "notes.local.txt"), "ignored non-command")
			write(t, filepath.Join(commands, "nested", "nested.md"), "not a top-level command")
			write(t, filepath.Join(commands, "nested", "draft.local.md"), "ignored nested command")
			write(t, filepath.Join(commands, "directory.md", "draft.local.md"), "not a command file")
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
			if tc.ignored != "" {
				write(t, filepath.Join(commands, tc.ignored), "unmerged command")
			}
			bin := filepath.Join(tmp, "gap")
			build := exec.Command("go", "build", "-o", bin, "./cmd/gap")
			build.Dir = repo
			if out, err := build.CombinedOutput(); err != nil {
				t.Fatalf("build: %v\n%s", err, out)
			}
			sync := func() (string, error) {
				t.Helper()
				cmd := exec.Command(bin, "sync")
				cmd.Dir = tmp
				cmd.Env = append(os.Environ(), "HOME="+home, "CODEX_HOME=", "XDG_STATE_HOME=")
				out, err := cmd.CombinedOutput()
				return string(out), err
			}
			out, err := sync()
			if tc.ignored != "" {
				if err == nil || !strings.Contains(out, "global/ or skills/ differs") {
					t.Fatalf("ignored command must refuse before publishing: %v\n%s", err, out)
				}
				if _, err := os.Stat(home); !os.IsNotExist(err) {
					t.Fatalf("refused checkout wrote its home: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("sync: %v\n%s", err, out)
			}
			policy := "See [x](" + filepath.Join(repo, "global", "docs", "agents", "dox.md") + ").\n"
			marked := "<!-- gap:begin -->\n" + policy + "<!-- gap:end -->\n"
			for _, dest := range []string{".claude/rules/global-agent-policy.md", ".pi/agent/APPEND_SYSTEM.md", ".omp/agent/APPEND_SYSTEM.md", ".codex/gap/global-agent-policy.md", ".cursor/rules/global-agent-policy.mdc"} {
				want := marked
				if strings.HasPrefix(dest, ".cursor/") {
					want = "---\nalwaysApply: true\n---\n\n" + marked
				}
				if got := readString(t, filepath.Join(home, dest)); got != want {
					t.Errorf("policy %s = %q, want %q", dest, got, want)
				}
			}
			for pass := 0; pass < 2; pass++ {
				ledger, err := readLedger(filepath.Join(home, ".gap", "published.json"))
				if err != nil {
					t.Fatal(err)
				}
				for _, dir := range []string{".omp/agent/commands", ".claude/commands", ".pi/agent/prompts"} {
					entries, err := os.ReadDir(filepath.Join(home, dir))
					if err != nil || len(entries) != 2 {
						t.Fatalf("expected only two top-level commands in %s: %v, %v", dir, entries, err)
					}
					for name, want := range map[string]string{"state-docs.md": command, ".empty.md": ""} {
						dest := filepath.Join(home, dir, name)
						if got := readString(t, dest); got != want {
							t.Errorf("command %s = %q, want unchanged %q", dest, got, want)
						}
						if ledger[dest] != fmt.Sprintf("%x", sha256.Sum256([]byte(want))) {
							t.Errorf("ledger must hash the exact command bytes: %s", dest)
						}
					}
				}
				if pass == 0 {
					if out, err := sync(); err != nil {
						t.Fatalf("repeated sync: %v\n%s", err, out)
					}
				}
			}
		})
	}
}

const recallPair = "<!-- recall:begin -->\n- keep this note\n<!-- recall:end -->\n"

// recallAround wraps the GAP block with a Recall pair on one side, so a sync or a removal
// that rewrites anything outside the GAP markers fails the byte comparison.
func recallAround(policy, side string) string {
	block := "<!-- gap:begin -->\n" + policy + "<!-- gap:end -->\n"
	if side == "before" {
		return recallPair + block
	}
	return block + recallPair
}

func TestRecallPairSurvivesSync(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	for _, side := range []string{"before", "after"} {
		t.Run(side, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("CODEX_HOME", "")
			unsetEnv(t, "JCODE_HOME")
			repo, _ := echoFixture(t, home, "#!/bin/sh\nexit 0\n")
			sourceRootFor = repo
			t.Cleanup(func() { sourceRootFor = "" })
			gitFixture(t, repo, "distro skill\n")
			for _, args := range [][]string{
				{"update-ref", "refs/remotes/origin/master", "HEAD"},
				{"symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/master"},
			} {
				if _, err := git(repo, args...); err != nil {
					t.Fatal(err)
				}
			}
			policy := adapt(readString(t, filepath.Join(repo, "global", "AGENTS.md")), false)
			dests, err := policyDests(home, filepath.Join(home, ".codex", "gap", "global-agent-policy.md"))
			if err != nil {
				t.Fatal(err)
			}
			want := map[string]string{}
			for _, c := range dests {
				stale, text := recallAround("stale policy\n", side), recallAround(policy, side)
				if c.cursor {
					stale = "---\nalwaysApply: true\n---\n\n" + stale
					text = "---\nalwaysApply: true\n---\n\n" + text
				}
				write(t, c.dest, stale)
				want[c.dest] = text
			}
			if code := run([]string{"sync", "--yes", "--only", "policy"}); code != 0 {
				t.Fatalf("sync exit %d", code)
			}
			for dest, text := range want {
				if got := readString(t, dest); got != text {
					t.Fatalf("%s recall pair changed:\n got %q\nwant %q", dest, got, text)
				}
			}
		})
	}
}

func TestRecallPairSurvivesUninstallStrip(t *testing.T) {
	for _, side := range []string{"before", "after"} {
		t.Run(side, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "AGENTS.md")
			write(t, path, recallAround("policy\n", side))
			action, rest, _, err := classifyPolicy(path, false, nil)
			if err != nil {
				t.Fatal(err)
			}
			if action != policyStrip || rest != recallPair {
				t.Fatalf("uninstall strip: action %v, rest %q, want only %q", action, rest, recallPair)
			}
		})
	}
}

func TestStatusPrintsRecallCandidates(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	repo, commit := echoFixture(t, home, "#!/bin/sh\nexit 0\n")
	sourceRootFor = repo
	t.Cleanup(func() { sourceRootFor = "" })
	bin := t.TempDir()
	write(t, filepath.Join(bin, "recall"), "#!/bin/sh\nprintf '%s\\n' '{\"candidates\":[{},{}]}'\n")
	if err := os.Chmod(filepath.Join(bin, "recall"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	out, stderr := captureStd(t, func() {
		if code := run([]string{"status"}); code != 0 {
			t.Fatalf("status exit %d", code)
		}
	})
	want := fmt.Sprintf("echo ref=master commit=%s store=ok\nrecall: 2 promotion candidates\n", commit[:7])
	if out != want || stderr != "" {
		t.Fatalf("status with recall:\n got %q\nwant %q\nstderr %q", out, want, stderr)
	}

	write(t, filepath.Join(bin, "recall"), "#!/bin/sh\nexit 1\n")
	out, stderr = captureStd(t, func() {
		if code := run([]string{"status"}); code != 0 {
			t.Fatalf("status exit %d", code)
		}
	})
	if strings.Contains(out, "recall:") || stderr != "" {
		t.Fatalf("a failing recall must print nothing: stdout %q stderr %q", out, stderr)
	}
}

func TestStatusSkipsHungRecall(t *testing.T) {
	bin := t.TempDir()
	write(t, filepath.Join(bin, "recall"), "#!/bin/sh\nsleep 30\n")
	if err := os.Chmod(filepath.Join(bin, "recall"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	old := recallTimeout
	recallTimeout = 200 * time.Millisecond
	t.Cleanup(func() { recallTimeout = old })
	began := time.Now()
	out, _ := captureStd(t, printRecallCandidates)
	if took := time.Since(began); took > 5*time.Second {
		t.Fatalf("a hung recall blocked status for %v", took)
	}
	if out != "" {
		t.Fatalf("a hung recall must print nothing, got %q", out)
	}
}

func TestSourcePrintsCheckout(t *testing.T) {
	repo := t.TempDir()
	sourceRootFor = repo
	t.Cleanup(func() { sourceRootFor = "" })
	out, stderr := captureStd(t, func() {
		if code := run([]string{"source"}); code != 0 {
			t.Fatalf("source exit %d", code)
		}
	})
	if out != repo+"\n" || stderr != "" {
		t.Fatalf("source: got %q stderr %q, want %q", out, stderr, repo+"\n")
	}
}

func TestPublishCommandsDirectoryStates(t *testing.T) {
	repo, home := t.TempDir(), t.TempDir()
	ledger := map[string]string{}
	ledgerPath := filepath.Join(home, ".gap", "published.json")
	if ok, err := publishCommands(repo, home, ledger, ledgerPath); err != nil || !ok {
		t.Fatalf("a missing commands directory must succeed: %v, %v", ok, err)
	}
	if entries, err := os.ReadDir(home); err != nil || len(entries) != 0 || len(ledger) != 0 {
		t.Fatalf("missing commands must leave home and ledger untouched: %v, %v, %v", entries, err, ledger)
	}
	write(t, filepath.Join(repo, "global", "commands"), "not a directory")
	if ok, err := publishCommands(repo, home, ledger, ledgerPath); err == nil || ok {
		t.Fatalf("an invalid commands directory must report its error: %v, %v", ok, err)
	}
}
