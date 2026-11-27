//go:build unix

package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
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
	})
	bar.step("fetch sources", nil)
	bar.item(1, 2, "do-skills")
	_, _ = spin("Fetching do-skills", func() (string, error) {
		time.Sleep(150 * time.Millisecond)
		return "ok", nil
	})
	bar.step("policy", nil)
	bar.println("wrote policy file")
	bar.suspended(func() {
		fmt.Println("prompt-while-suspended")
	})
	bar.step("", nil)
	bar.stop()
	fmt.Println("lifecycle-complete")
}

func TestProgressBarLifecyclePTY(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=^TestProgressLifecycleHelper$")
	cmd.Env = append(os.Environ(), "GAP_TEST_PROGRESS_LIFECYCLE=1", "TERM=xterm-256color", "NO_COLOR=1")
	tty, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: 24, Cols: 100})
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
	for _, want := range []string{
		"Fetching do-skills",
		"wrote policy file",
		"prompt-while-suspended",
		"\u2713 fetch sources",
		"\u2713 policy",
		"lifecycle-complete",
	} {
		if !strings.Contains(stripped, want) {
			t.Fatalf("PTY lifecycle output missing %q:\n%s", want, stripped)
		}
	}
}
