//go:build unix

package main

import (
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

func TestTUIRunCancelStopsDescendants(t *testing.T) {
	oldExe, oldGrace := tuiExecutable, tuiStopGrace
	t.Cleanup(func() { tuiExecutable, tuiStopGrace = oldExe, oldGrace })
	tuiStopGrace = 300 * time.Millisecond
	dir := t.TempDir()
	pidFile := filepath.Join(dir, "pid")
	script := filepath.Join(dir, "gap")
	write(t, script, "#!/bin/sh\n(trap '' INT TERM; exec /bin/sleep 30) >/dev/null 2>&1 &\necho $! > \""+pidFile+".tmp\"\nmv \""+pidFile+".tmp\" \""+pidFile+"\"\nwait\n")
	if err := os.Chmod(script, 0o755); err != nil {
		t.Fatal(err)
	}
	tuiExecutable = func() (string, error) { return script, nil }
	r := &execRunner{}
	msgs := collect(t, r, []tuiJob{{label: "remove ua", args: []string{"remove", "--", "ua"}}}, func(int) {
		go func() {
			for !exists(pidFile) {
				time.Sleep(20 * time.Millisecond)
			}
			r.cancel()
		}()
	})
	done := doneOf(msgs)
	if len(done) != 1 || !done[0].cancelled {
		t.Fatalf("done = %+v", done)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(readString(t, pidFile)))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { syscall.Kill(pid, syscall.SIGKILL) })
	for end := time.Now().Add(2 * time.Second); syscall.Kill(pid, 0) == nil; time.Sleep(20 * time.Millisecond) {
		if time.Now().After(end) {
			t.Fatalf("descendant %d still runs after the step was reported cancelled", pid)
		}
	}
}

func TestTUIRunCleanExitStopsDescendants(t *testing.T) {
	oldExe, oldGrace := tuiExecutable, tuiStopGrace
	t.Cleanup(func() { tuiExecutable, tuiStopGrace = oldExe, oldGrace })
	tuiStopGrace = 300 * time.Millisecond
	dir := t.TempDir()
	life := filepath.Join(dir, "life")
	if err := syscall.Mkfifo(life, 0o600); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(dir, "gap")
	write(t, script, "#!/bin/sh\n(trap '' INT TERM; exec 3>\""+life+"\"; exec /bin/sleep 30) >/dev/null 2>&1 &\n")
	if err := os.Chmod(script, 0o755); err != nil {
		t.Fatal(err)
	}
	tuiExecutable = func() (string, error) { return script, nil }
	ended := make(chan struct{})
	go func() {
		held, err := os.OpenFile(life, os.O_RDONLY, 0)
		if err != nil {
			return
		}
		defer held.Close()
		io.Copy(io.Discard, held)
		close(ended)
	}()
	msgs := collect(t, &execRunner{}, []tuiJob{{label: "update ua", args: []string{"update", "--", "ua"}}}, nil)
	done := doneOf(msgs)
	if len(done) != 1 || done[0].cancelled || done[0].code == 0 {
		t.Fatalf("done = %+v", done)
	}
	select {
	case <-ended:
	case <-time.After(2 * time.Second):
		t.Fatal("descendant still holds its pipe after the step was reported done")
	}
}

func TestTUIRunForeignGroupDoesNotHoldTheBatch(t *testing.T) {
	oldExe, oldGrace := tuiExecutable, tuiStopGrace
	t.Cleanup(func() { tuiExecutable, tuiStopGrace = oldExe, oldGrace })
	tuiStopGrace = 300 * time.Millisecond
	dir := t.TempDir()
	ready := filepath.Join(dir, "ready")
	pidFile := filepath.Join(dir, "pid")
	script := filepath.Join(dir, "gap")
	write(t, script, "#!/bin/sh\n(/bin/sh -c 'echo $$ >> \""+pidFile+"\"; echo ready >> \""+ready+"\"; exec /bin/sleep 30') &\n")
	if err := os.Chmod(script, 0o755); err != nil {
		t.Fatal(err)
	}
	tuiExecutable = func() (string, error) { return script, nil }
	t.Cleanup(func() {
		pid, err := strconv.Atoi(strings.TrimSpace(readString(t, pidFile)))
		if err == nil {
			syscall.Kill(pid, syscall.SIGKILL)
		}
	})
	start := time.Now()
	msgs := collect(t, &execRunner{}, []tuiJob{{label: "update ua", args: []string{"update", "--", "ua"}}}, nil)
	if time.Since(start) > 6*time.Second {
		t.Fatalf("a writer outside the process group held the batch for %v", time.Since(start))
	}
	if done := doneOf(msgs); len(done) != 1 || done[0].code == 0 {
		t.Fatalf("done = %+v", done)
	}
}

func TestTUIRunCancelEndsWithinOneGrace(t *testing.T) {
	oldExe, oldGrace := tuiExecutable, tuiStopGrace
	t.Cleanup(func() { tuiExecutable, tuiStopGrace = oldExe, oldGrace })
	tuiStopGrace = time.Second
	script := filepath.Join(t.TempDir(), "gap")
	write(t, script, "#!/bin/sh\ntrap '' INT\n(trap '' INT; exec /bin/sleep 30) &\necho started\nwait\n")
	if err := os.Chmod(script, 0o755); err != nil {
		t.Fatal(err)
	}
	tuiExecutable = func() (string, error) { return script, nil }
	r := &execRunner{}
	at := make(chan time.Time, 1)
	msgs := collect(t, r, []tuiJob{{label: "remove ua", args: []string{"remove", "--", "ua"}}}, func(int) {
		go func() {
			time.Sleep(300 * time.Millisecond)
			at <- time.Now()
			r.cancel()
		}()
	})
	took := time.Since(<-at)
	if done := doneOf(msgs); len(done) != 1 || !done[0].cancelled {
		t.Fatalf("done = %+v", done)
	}
	if took > 1700*time.Millisecond {
		t.Fatalf("stop took %v with a %v grace", took, tuiStopGrace)
	}
}

func TestTUIRunLateCancelStopsDescendants(t *testing.T) {
	oldExe, oldGrace := tuiExecutable, tuiStopGrace
	t.Cleanup(func() { tuiExecutable, tuiStopGrace = oldExe, oldGrace })
	tuiStopGrace = 300 * time.Millisecond
	dir := t.TempDir()
	pidFile := filepath.Join(dir, "pid")
	script := filepath.Join(dir, "gap")
	write(t, script, "#!/bin/sh\n(trap '' INT TERM; exec /bin/sleep 30) >/dev/null 2>&1 &\necho $! > \""+pidFile+"\"\necho started\n")
	if err := os.Chmod(script, 0o755); err != nil {
		t.Fatal(err)
	}
	tuiExecutable = func() (string, error) { return script, nil }
	r := &execRunner{}
	ch := make(chan tea.Msg, 64)
	r.start([]tuiJob{{label: "remove ua", args: []string{"remove", "--", "ua"}}}, func(msg tea.Msg) {
		if _, ok := msg.(jobOutputMsg); ok {
			time.Sleep(300 * time.Millisecond)
			r.cancel()
		}
		ch <- msg
	})
	for msg := range ch {
		if _, ok := msg.(jobsFinishedMsg); ok {
			break
		}
	}
	pid, err := strconv.Atoi(strings.TrimSpace(readString(t, pidFile)))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { syscall.Kill(pid, syscall.SIGKILL) })
	for end := time.Now().Add(2 * time.Second); syscall.Kill(pid, 0) == nil; time.Sleep(20 * time.Millisecond) {
		if time.Now().After(end) {
			t.Fatalf("descendant %d still runs after a stop that came while output was delivered", pid)
		}
	}
}

func TestTUIRunStopDuringOutputStartsGraceAtOnce(t *testing.T) {
	oldExe, oldGrace := tuiExecutable, tuiStopGrace
	t.Cleanup(func() { tuiExecutable, tuiStopGrace = oldExe, oldGrace })
	tuiStopGrace = time.Second
	dir := t.TempDir()
	pidFile := filepath.Join(dir, "pid")
	script := filepath.Join(dir, "gap")
	write(t, script, "#!/bin/sh\n(trap '' INT TERM; exec /bin/sleep 30) >/dev/null 2>&1 &\necho $! > \""+pidFile+"\"\necho started\n")
	if err := os.Chmod(script, 0o755); err != nil {
		t.Fatal(err)
	}
	tuiExecutable = func() (string, error) { return script, nil }
	r := &execRunner{}
	ch := make(chan tea.Msg, 64)
	var stopped time.Time
	var dead time.Duration
	r.start([]tuiJob{{label: "remove ua", args: []string{"remove", "--", "ua"}}}, func(msg tea.Msg) {
		if _, ok := msg.(jobOutputMsg); ok {
			time.Sleep(300 * time.Millisecond)
			stopped = time.Now()
			r.cancel()
			pid, _ := strconv.Atoi(strings.TrimSpace(readString(t, pidFile)))
			for syscall.Kill(pid, 0) == nil && time.Since(stopped) < 3*time.Second {
				time.Sleep(20 * time.Millisecond)
			}
			dead = time.Since(stopped)
		}
		ch <- msg
	})
	for msg := range ch {
		if _, ok := msg.(jobsFinishedMsg); ok {
			break
		}
	}
	if dead > 1600*time.Millisecond {
		t.Fatalf("descendant outlived the %v grace by running %v after Stop while output was still delivered", tuiStopGrace, dead)
	}
}

func TestTUIReadyNeedsTerminalStdout(t *testing.T) {
	null, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer null.Close()
	if !isTerminal(null) {
		t.Skip("/dev/null is not a character device here")
	}
	old := os.Stdout
	os.Stdout = null
	defer func() { os.Stdout = old }()
	if stdoutIsTerminal() {
		t.Fatal("/dev/null on stdout counts as a terminal")
	}
}
