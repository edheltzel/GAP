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
	fmt.Println("ready")
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
	for !strings.Contains(out(), "ready") {
		if time.Now().After(deadline) {
			cmd.Process.Kill()
			t.Fatalf("helper never started:\n%q", out())
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !strings.Contains(out(), "\x1b[1;23r") {
		t.Fatalf("bar did not reserve the last row:\n%q", out())
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
	if !strings.Contains(tail, "\x1b[r") {
		t.Fatalf("interrupt left the scroll region set:\n%q", tail)
	}
}
