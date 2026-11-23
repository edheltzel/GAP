//go:build darwin || linux || freebsd || netbsd || openbsd || dragonfly

package main

import (
	"os"
	"syscall"
	"unsafe"
)

// fdIsTerminal asks the kernel for terminal attributes; only a real TTY has them, so /dev/null and pipes fail.
func fdIsTerminal(fd uintptr) bool {
	var t syscall.Termios
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, ioctlReadTermios, uintptr(unsafe.Pointer(&t)))
	return errno == 0
}

func stdinIsTerminal() bool { return fdIsTerminal(os.Stdin.Fd()) }

func stdoutIsTerminal() bool { return fdIsTerminal(os.Stdout.Fd()) }
