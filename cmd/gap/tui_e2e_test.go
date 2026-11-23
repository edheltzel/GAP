//go:build unix

package main

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/creack/pty"
)

type ptyGap struct {
	t    *testing.T
	cmd  *exec.Cmd
	tty  *os.File
	mu   sync.Mutex
	raw  bytes.Buffer
	done chan error
}

func gapEnv(repo string, extra ...string) []string {
	var env []string
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(kv, "NO_COLOR=") && !strings.HasPrefix(kv, "TERM=") {
			env = append(env, kv)
		}
	}
	return append(env, append([]string{"GAP_TEST_AS_GAP=1", "GAP_TEST_SOURCE=" + repo, "PATH=/usr/bin:/bin", "TERM=xterm-256color"}, extra...)...)
}

func startPTY(t *testing.T, dir string, env []string, args ...string) *ptyGap {
	t.Helper()
	cmd := exec.Command(os.Args[0], args...)
	cmd.Dir, cmd.Env = dir, env
	tty, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: 40, Cols: 140})
	if err != nil {
		t.Fatal(err)
	}
	g := &ptyGap{t: t, cmd: cmd, tty: tty, done: make(chan error, 1)}
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := tty.Read(buf)
			g.mu.Lock()
			g.raw.Write(buf[:n])
			g.mu.Unlock()
			if err != nil {
				return
			}
		}
	}()
	go func() { g.done <- cmd.Wait() }()
	t.Cleanup(func() { _ = cmd.Process.Kill(); tty.Close() })
	return g
}

func (g *ptyGap) text() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return ansi.Strip(g.raw.String())
}

func (g *ptyGap) waitFor(want string) {
	g.t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if strings.Contains(g.text(), want) {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	g.t.Fatalf("screen never showed %q:\n%s", want, g.text())
}

func (g *ptyGap) keys(keys ...string) {
	g.t.Helper()
	for _, k := range keys {
		if _, err := io.WriteString(g.tty, k); err != nil {
			g.t.Fatal(err)
		}
		time.Sleep(150 * time.Millisecond)
	}
}

func (g *ptyGap) exit() int {
	g.t.Helper()
	select {
	case err := <-g.done:
		if exit, ok := err.(*exec.ExitError); ok {
			return exit.ExitCode()
		}
		if err != nil {
			g.t.Fatal(err)
		}
		return 0
	case <-time.After(30 * time.Second):
		g.t.Fatalf("gap did not exit:\n%s", g.text())
	}
	return -1
}

func TestTUIEntryNeedsATerminal(t *testing.T) {
	repo := t.TempDir()
	for _, args := range [][]string{{}, {"tui"}} {
		cmd := exec.Command(os.Args[0], args...)
		cmd.Env = gapEnv(repo, "HOME="+t.TempDir())
		out, err := cmd.CombinedOutput()
		exit, _ := err.(*exec.ExitError)
		if exit == nil || exit.ExitCode() != 2 {
			t.Fatalf("gap %v piped: %v\n%s", args, err, out)
		}
		want := "Usage:"
		if len(args) == 1 {
			want = "gap tui needs a terminal"
		}
		if !strings.Contains(string(out), want) {
			t.Fatalf("gap %v piped printed:\n%s", args, out)
		}
	}
	g := startPTY(t, repo, gapEnv(repo, "HOME="+t.TempDir(), "NO_COLOR=1"))
	if code := g.exit(); code != 2 || !strings.Contains(g.text(), "Usage:") {
		t.Fatalf("NO_COLOR gap on a terminal: exit %d\n%s", code, g.text())
	}
}

func tuiE2EEnv(t *testing.T) *utilityEnv {
	t.Helper()
	e := newUtilityEnv(t, "schema = \"1\"\nname = \"util\"\nskills = [\"util-skill\"]\n")
	trustedUtilities(t, e, "ua", "ub")
	if err := recordProject(e.home, e.project); err != nil {
		t.Fatal(err)
	}
	return e
}

func TestTUIEndToEndMassDelete(t *testing.T) {
	e := tuiE2EEnv(t)
	before := readString(t, filepath.Join(userConfigDir(e.home), "gap.lock"))
	g := startPTY(t, e.project, gapEnv(e.repo, "HOME="+e.home))
	g.waitFor("ua-skill")
	g.keys(" ", " ", "d")
	g.waitFor("Remove 2 utilities?")
	g.keys("\r")
	g.waitFor("2 ok, 0 failed, 0 not run.")
	g.keys("\r", "3")
	g.waitFor("profile project")
	g.keys("q")
	if code := g.exit(); code != 0 {
		t.Fatalf("exit %d", code)
	}
	list, _ := captureStd(t, func() { _ = listUtilities() })
	if strings.Contains(list, "ua") || strings.Contains(list, "ub") {
		t.Fatalf("gap list after the TUI removed them:\n%s", list)
	}
	for _, id := range []string{"ua", "ub"} {
		if backups, _ := filepath.Glob(filepath.Join(e.home, ".gap", "backups", "*", ".agents", "skills", id+"-skill")); len(backups) != 1 {
			t.Fatalf("backup for %s = %v", id, backups)
		}
	}
	after := readString(t, filepath.Join(userConfigDir(e.home), "gap.lock"))
	if after == before || strings.Contains(after, "ua") || strings.Contains(after, "ub") {
		t.Fatalf("user lock after removal:\n%s", after)
	}
}

func TestTUIEndToEndUpdateOne(t *testing.T) {
	e := tuiE2EEnv(t)
	lockPath := filepath.Join(userConfigDir(e.home), "gap.lock")
	first, err := readLock(userConfigDir(e.home))
	if err != nil {
		t.Fatal(err)
	}
	remoteA := strings.TrimPrefix(readConfigComponents(t, userConfigFile(e.home))["ua"].(map[string]any)["repo"].(string), "file://")
	write(t, filepath.Join(remoteA, "skills", "ua-skill", "SKILL.md"), "---\nname: ua-skill\ndescription: v2\n---\n")
	commit, _ := commitAll(t, remoteA)
	g := startPTY(t, e.project, gapEnv(e.repo, "HOME="+e.home))
	g.waitFor("ua-skill")
	g.keys("u")
	g.waitFor("1 ok, 0 failed, 0 not run.")
	g.keys("\r", "q")
	if code := g.exit(); code != 0 {
		t.Fatalf("exit %d", code)
	}
	second, err := readLock(userConfigDir(e.home))
	if err != nil {
		t.Fatal(err)
	}
	if second.Components["ua"].Commit != commit || second.Components["ub"] != first.Components["ub"] || len(second.Components) != 2 {
		t.Fatalf("lock moved more than ua:\nbefore %+v\nafter  %+v\n%s", first, second, readString(t, lockPath))
	}
	if _, err := os.Stat(filepath.Join(e.repo, "gap.lock")); !os.IsNotExist(err) {
		t.Fatal("updating a utility wrote the distro lock")
	}
	if !strings.Contains(readString(t, filepath.Join(e.home, ".agents", "skills", "ua-skill", "SKILL.md")), "v2") {
		t.Fatal("update did not install the new skill")
	}
}
