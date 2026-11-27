//go:build unix

package main

import (
	"bytes"
	"errors"
	"io"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/creack/pty"
)

func ownGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func interruptGroup(pgid int) error {
	term := syscall.Kill(-pgid, syscall.SIGTERM)
	intr := syscall.Kill(-pgid, syscall.SIGINT)
	if term != nil {
		return intr
	}
	return term
}

func killGroup(pgid int) {
	syscall.Kill(-pgid, syscall.SIGKILL)
}

var groupAlive = func(pgid int) bool { return syscall.Kill(-pgid, 0) == nil }

// runOnPty runs cmd with its stdout on a pty, so gap status draws the framed table instead of the
// piped lines. The returned text is the screen with ANSI stripped. ok is false when no pty is available.
func runOnPty(cmd *childCmd) ([]byte, error, bool) {
	// pty.StartWithSize sets Setsid and Setctty, which puts the child in its own process group
	// (pgid == pid). Clear Setpgid first so Start does not conflict with Setsid.
	cmd.SysProcAttr = nil
	tty, err := pty.StartWithSize(cmd.Cmd, &pty.Winsize{Rows: 40, Cols: 140})
	if err != nil {
		ownGroup(cmd.Cmd)
		return nil, nil, false
	}
	var raw bytes.Buffer
	_, copyErr := io.Copy(&raw, tty)
	_ = tty.Close()
	waitErr := cmd.Wait()
	text := strings.ReplaceAll(ansi.Strip(raw.String()), "\r\n", "\n")
	if copyErr != nil && !errors.Is(copyErr, syscall.EIO) {
		return []byte(text), copyErr, true
	}
	return []byte(text), waitErr, true
}

func groupGone(pgid int, wait time.Duration) bool {
	if wait == 0 {
		return !groupAlive(pgid)
	}
	quiet := time.Duration(0)
	for end := time.Now().Add(wait); ; time.Sleep(20 * time.Millisecond) {
		if groupAlive(pgid) {
			quiet = 0
		} else {
			quiet += 20 * time.Millisecond
			if quiet >= 200*time.Millisecond {
				return true
			}
		}
		if time.Now().After(end) {
			return false
		}
	}
}
