//go:build !unix

package main

import (
	"os"
	"os/exec"
	"syscall"
	"time"
)

func ownGroup(*exec.Cmd) {}

func interruptGroup(pid int) error {
	p, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	term := p.Signal(syscall.SIGTERM)
	intr := p.Signal(os.Interrupt)
	if term != nil {
		return intr
	}
	return term
}

func killGroup(pid int) {
	if p, err := os.FindProcess(pid); err == nil {
		p.Kill()
	}
}

func groupGone(int, time.Duration) bool { return true }

var runOnPty = func(*childCmd, int) ([]byte, error, bool) { return nil, nil, false }
