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

func stopGroup(int, time.Duration) {}
