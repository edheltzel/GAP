//go:build !unix

package main

import (
	"os"
	"os/exec"
	"time"
)

func ownGroup(*exec.Cmd) {}

func interruptGroup(pid int) error {
	p, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return p.Signal(os.Interrupt)
}

func killGroup(pid int) {
	if p, err := os.FindProcess(pid); err == nil {
		p.Kill()
	}
}

func groupGone(int, time.Duration) bool { return true }
