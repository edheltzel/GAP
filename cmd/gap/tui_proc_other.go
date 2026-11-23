//go:build !unix

package main

import (
	"os"
	"os/exec"
	"time"
)

func ownGroup(*exec.Cmd) {}

func interruptGroup(cmd *exec.Cmd) error {
	return cmd.Process.Signal(os.Interrupt)
}

func killGroup(pid int) {
	if p, err := os.FindProcess(pid); err == nil {
		p.Kill()
	}
}

func groupGone(int, time.Duration) bool { return true }
