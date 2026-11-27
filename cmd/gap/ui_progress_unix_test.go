//go:build unix

package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"slices"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/vt"
	"github.com/creack/pty"
)

func TestProgressSignalHelper(t *testing.T) {
	if os.Getenv("GAP_TEST_PROGRESS_HELPER") != "1" {
		t.Skip("helper process")
	}
	bar := startProgress("update", 2)
	if bar == nil {
		fmt.Println("no bar")
		os.Exit(3)
	}
	bar.step("echo", nil)
	bar.println("ready")
	time.Sleep(30 * time.Second)
	os.Exit(4)
}

func TestProgressBarResetsTerminalOnInterrupt(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=^TestProgressSignalHelper$")
	cmd.Env = append(os.Environ(), "GAP_TEST_PROGRESS_HELPER=1", "TERM=xterm-256color", "NO_COLOR=1")
	tty, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: 24, Cols: 80})
	if err != nil {
		t.Fatal(err)
	}
	defer tty.Close()
	var mu sync.Mutex
	var raw bytes.Buffer
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := tty.Read(buf)
			mu.Lock()
			raw.Write(buf[:n])
			mu.Unlock()
			if err != nil {
				return
			}
		}
	}()
	out := func() string { mu.Lock(); defer mu.Unlock(); return raw.String() }
	deadline := time.Now().Add(20 * time.Second)
	for {
		stripped := ansi.Strip(out())
		if strings.Contains(stripped, "ready") && strings.Contains(stripped, "update echo") {
			break
		}
		if time.Now().After(deadline) {
			cmd.Process.Kill()
			t.Fatalf("helper never rendered live progress:\n%q", out())
		}
		time.Sleep(20 * time.Millisecond)
	}
	// Ensure no raw ANSI scroll-region escape codes are emitted.
	if strings.Contains(out(), "\x1b[1;23r") {
		cmd.Process.Kill()
		t.Fatalf("unexpected raw scroll-region escape in Bubble Tea progress output:\n%q", out())
	}
	cmd.Process.Signal(os.Interrupt)
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err = <-done:
	case <-time.After(10 * time.Second):
		cmd.Process.Kill()
		t.Fatal("interrupt did not end the run")
	}
	var exit *exec.ExitError
	if !errors.As(err, &exit) {
		t.Fatalf("exit = %v, want death by signal", err)
	}
	if ws, ok := exit.Sys().(syscall.WaitStatus); !ok || !ws.Signaled() || ws.Signal() != syscall.SIGINT {
		t.Fatalf("exit = %v, want SIGINT", err)
	}
	time.Sleep(100 * time.Millisecond)
	tail := out()[strings.Index(out(), "ready"):]
	// Bubble Tea restores cursor visibility (\x1b[?25h) on clean shutdown.
	if !strings.Contains(tail, "\x1b[?25h") {
		t.Fatalf("interrupt did not restore cursor visibility:\n%q", tail)
	}
}

func TestProgressLifecycleHelper(t *testing.T) {
	if os.Getenv("GAP_TEST_PROGRESS_LIFECYCLE") != "1" {
		t.Skip("helper process")
	}
	bar := startProgress("sync", 0)
	if bar == nil {
		fmt.Println("no bar")
		os.Exit(3)
	}
	defer bar.stop()
	bar.plan([]progressStep{
		{Label: "fetch sources"},
		{Label: "policy"},
		{Label: "skills"},
	})
	bar.step("fetch sources", nil)
	bar.item(1, 2, "do-skills")
	_, _ = spin("Fetching do-skills", func() (string, error) {
		time.Sleep(150 * time.Millisecond)
		return "ok", nil
	})
	bar.step("policy", nil)
	bar.println("wrote policy file")
	out := newReport(os.Stdout)
	out.done(true, "do-skills moved from aaaaaaa to bbbbbbb @ main")
	out.println("    bbbbbbb commit subject line")
	out.printf("%s\n", "next heading")
	began := time.Now()
	var handoff time.Duration
	bar.suspended(func() {
		handoff = time.Since(began)
		fmt.Println("prompt-while-suspended")
	})
	bar.item(1, 1, "after-resume")
	bar.mark(false)
	bar.step("skills", nil)
	bar.mark(false)
	bar.step("", nil)
	bar.stop()
	fmt.Printf("handoff-ms=%d\n", handoff.Milliseconds())
}

func TestProgressBarLifecyclePTY(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=^TestProgressLifecycleHelper$")
	cmd.Env = append(os.Environ(), "GAP_TEST_PROGRESS_LIFECYCLE=1", "TERM=xterm-256color", "NO_COLOR=1")
	tty, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: 40, Cols: 100})
	if err != nil {
		t.Fatal(err)
	}
	defer tty.Close()
	var mu sync.Mutex
	var raw bytes.Buffer
	doneRead := make(chan struct{})
	go func() {
		defer close(doneRead)
		buf := make([]byte, 4096)
		for {
			n, err := tty.Read(buf)
			mu.Lock()
			raw.Write(buf[:n])
			mu.Unlock()
			if err != nil {
				return
			}
		}
	}()
	if err := cmd.Wait(); err != nil {
		t.Fatalf("lifecycle helper failed: %v\n%s", err, ansi.Strip(raw.String()))
	}
	_ = tty.Close()
	<-doneRead
	stripped := ansi.Strip(raw.String())
	if !strings.Contains(stripped, "Fetching do-skills") {
		t.Fatalf("PTY lifecycle output missing the spinner:\n%s", stripped)
	}
	emu := vt.NewEmulator(100, 40)
	go func() { _, _ = io.Copy(io.Discard, emu) }()
	_, _ = emu.Write(raw.Bytes())
	lines := strings.Split(emu.String(), "\n")
	for i := range lines {
		lines[i] = strings.TrimSpace(lines[i])
	}
	screen := strings.Join(lines, "\n")
	for _, want := range []string{
		"wrote policy file",
		"\u2713 do-skills moved from aaaaaaa to bbbbbbb @ main",
		"bbbbbbb commit subject line",
		"next heading",
		"prompt-while-suspended",
	} {
		if !slices.Contains(lines, want) {
			t.Fatalf("screen lacks the intact line %q:\n%s", want, screen)
		}
	}
	for _, row := range []string{"\u2713 fetch sources", "! policy", "! skills"} {
		if !slices.ContainsFunc(lines, func(l string) bool { return strings.HasPrefix(l, row) }) {
			t.Fatalf("final checklist lost the %q row:\n%s", row, screen)
		}
	}
	var ms int
	if _, err := fmt.Sscanf(stripped[strings.Index(stripped, "handoff-ms="):], "handoff-ms=%d", &ms); err != nil || ms > 250 {
		t.Fatalf("suspend handoff took %dms (%v), want under 250ms", ms, err)
	}
}

func TestProgressNullStdoutHelper(t *testing.T) {
	if os.Getenv("GAP_TEST_PROGRESS_NULL") != "1" {
		t.Skip("helper process")
	}
	null, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		os.Exit(3)
	}
	os.Stdout = null
	bar := startProgress("update", 1)
	fmt.Fprintf(os.Stderr, "bar-started=%v\n", bar != nil)
	newReport(os.Stdout).done(true, "stdout-only outcome")
	bar.stop()
}

func TestProgressSkipsRedirectedStdout(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=^TestProgressNullStdoutHelper$")
	cmd.Env = append(os.Environ(), "GAP_TEST_PROGRESS_NULL=1", "TERM=xterm-256color", "NO_COLOR=1")
	tty, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: 24, Cols: 80})
	if err != nil {
		t.Fatal(err)
	}
	defer tty.Close()
	var raw bytes.Buffer
	done := make(chan struct{})
	go func() { defer close(done); _, _ = io.Copy(&raw, tty) }()
	if err := cmd.Wait(); err != nil {
		t.Fatalf("helper failed: %v\n%s", err, raw.String())
	}
	_ = tty.Close()
	<-done
	out := ansi.Strip(raw.String())
	if !strings.Contains(out, "bar-started=false") || strings.Contains(out, "stdout-only outcome") {
		t.Fatalf("redirected stdout leaked onto the terminal:\n%s", out)
	}
}
