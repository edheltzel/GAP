//go:build darwin || linux || freebsd || netbsd || openbsd || dragonfly

package main

import (
	"os"
	"syscall"
	"unsafe"
)

// stdinIsTerminal asks the kernel for terminal attributes; only a real TTY has them, so /dev/null and pipes fail.
func stdinIsTerminal() bool {
	var t syscall.Termios
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, os.Stdin.Fd(), ioctlReadTermios, uintptr(unsafe.Pointer(&t)))
	return errno == 0
}
