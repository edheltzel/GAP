//go:build unix

package main

import (
	"os"
	"os/signal"
	"syscall"
)

// restoreOnSignal stops the Bubble Tea progress program when a signal ends the run
// so terminal state is restored, then lets that signal end the process as it would have.
func restoreOnSignal(p *progressBar) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	go func() {
		defer signal.Stop(sigs)
		select {
		case sig := <-sigs:
			p.stop()
			signal.Reset(sig)
			if s, ok := sig.(syscall.Signal); ok {
				syscall.Kill(os.Getpid(), s)
			}
		case <-p.quit:
		}
	}()
}
