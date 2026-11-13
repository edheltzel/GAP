package main

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"charm.land/bubbles/v2/spinner"
	"github.com/charmbracelet/x/ansi"
)

// spinnerOut is where spin draws. Tests replace it.
var spinnerOut = func() (io.Writer, bool) {
	return os.Stderr, isTerminal(os.Stderr) && !plainEnv()
}

// spin runs fn while a spinner labelled title turns on a terminal's stderr, then clears its line.
// With no terminal, NO_COLOR, or TERM=dumb, it only runs fn and prints nothing.
func spin[T any](title string, fn func() (T, error)) (T, error) {
	w, ok := spinnerOut()
	if !ok {
		return fn()
	}
	r := &report{out: w, tty: true, plain: plainEnv()}
	frames := spinner.MiniDot.Frames
	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		tick := time.NewTicker(spinner.MiniDot.FPS)
		defer tick.Stop()
		for i := 0; ; i++ {
			fmt.Fprintf(w, "\r%s%s %s", ansi.EraseEntireLine, r.tone(frames[i%len(frames)], colorAccent), r.tone(title, colorDim))
			select {
			case <-stop:
				fmt.Fprint(w, "\r"+ansi.EraseEntireLine)
				return
			case <-tick.C:
			}
		}
	}()
	v, err := fn()
	close(stop)
	wg.Wait()
	return v, err
}
