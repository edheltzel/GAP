package main

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"charm.land/bubbles/v2/spinner"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/term"
)

// spinnerOut is where spin draws. Tests replace it.
var spinnerOut = func() (io.Writer, bool) {
	return os.Stderr, isTerminal(os.Stderr) && !plainEnv()
}

// spinnerWidth is the column count of the terminal spin draws on, or 0 when unknown. Tests replace it.
var spinnerWidth = func() int {
	w, _, err := term.GetSize(os.Stderr.Fd())
	if err != nil {
		return 0
	}
	return w
}

// spinnerFrame renders one spinner frame that fits on one physical line of a width-column terminal.
// The title is cut by display columns with an ellipsis, and the frame stops short of the last column so it never wraps.
func spinnerFrame(r *report, frame, title string, width int) string {
	if width > 0 {
		title = ansi.Truncate(title, max(width-1-ansi.StringWidth(frame)-1, 0), "…")
	}
	return r.tone(frame, colorAccent) + " " + r.tone(title, colorDim)
}

// spin runs fn while a spinner labelled title turns on a terminal's stderr, then clears its line.
// When the inline Bubble Tea progress program is active, it delegates the title as live step detail
// instead of spawning a competing stderr spinner.
// With no terminal, NO_COLOR, or TERM=dumb, it only runs fn and prints nothing.
func spin[T any](title string, fn func() (T, error)) (T, error) {
	if activeProgress != nil {
		activeProgress.detail(title)
		defer activeProgress.detail("")
		return fn()
	}
	w, ok := spinnerOut()
	if !ok {
		return fn()
	}
	r := &report{out: w, tty: true, plain: plainEnv()}
	frames := spinner.MiniDot.Frames
	width := spinnerWidth()
	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		tick := time.NewTicker(spinner.MiniDot.FPS)
		defer tick.Stop()
		for i := 0; ; i++ {
			fmt.Fprint(w, "\r"+ansi.EraseEntireLine+spinnerFrame(r, frames[i%len(frames)], title, width))
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

// startWorking turns a spinner labelled title on a terminal's stderr until the returned stop runs.
// It follows spin: nothing off a terminal, with NO_COLOR or TERM=dumb, or while a progress bar is active.
func startWorking(title string) func() {
	w, ok := spinnerOut()
	if activeProgress != nil || !ok {
		return func() {}
	}
	r := &report{out: w, tty: true, plain: plainEnv()}
	frames := spinner.MiniDot.Frames
	width := spinnerWidth()
	stop, done := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(done)
		tick := time.NewTicker(spinner.MiniDot.FPS)
		defer tick.Stop()
		for i := 0; ; i++ {
			fmt.Fprint(w, "\r"+ansi.EraseEntireLine+spinnerFrame(r, frames[i%len(frames)], title, width))
			select {
			case <-stop:
				fmt.Fprint(w, "\r"+ansi.EraseEntireLine)
				return
			case <-tick.C:
			}
		}
	}()
	var once sync.Once
	return func() {
		once.Do(func() {
			close(stop)
			<-done
		})
	}
}
