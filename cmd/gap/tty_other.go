//go:build !(darwin || linux || freebsd || netbsd || openbsd || dragonfly)

package main

// stdinIsTerminal reports false here, so a foreign file is skipped instead of prompted.
func stdinIsTerminal() bool { return false }
