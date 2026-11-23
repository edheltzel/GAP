//go:build unix

package main

import (
	"os/exec"
	"syscall"
	"time"
)

func ownGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func interruptGroup(pgid int) error {
	return syscall.Kill(-pgid, syscall.SIGINT)
}

func killGroup(pgid int) {
	syscall.Kill(-pgid, syscall.SIGKILL)
}

func groupGone(pgid int, wait time.Duration) bool {
	for end := time.Now().Add(wait); ; time.Sleep(20 * time.Millisecond) {
		if syscall.Kill(-pgid, 0) != nil {
			return true
		}
		if time.Now().After(end) {
			return false
		}
	}
}
