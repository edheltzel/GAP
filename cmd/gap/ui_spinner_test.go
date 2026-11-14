package main

import (
	"bytes"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
)

func TestSpinnerFramesFitTerminalWidth(t *testing.T) {
	prevOut, prevWidth := spinnerOut, spinnerWidth
	t.Cleanup(func() { spinnerOut, spinnerWidth = prevOut, prevWidth })
	title := "Resolving main on https://github.com/kunchenguid/chrome-devtools-axi"
	for _, columns := range []int{40, 60} {
		var buf bytes.Buffer
		spinnerOut = func() (io.Writer, bool) { return &buf, true }
		spinnerWidth = func() int { return columns }
		if _, err := spin(title, func() (int, error) { time.Sleep(250 * time.Millisecond); return 0, nil }); err != nil {
			t.Fatal(err)
		}
		out := buf.String()
		if !strings.HasSuffix(out, "\r"+ansi.EraseEntireLine) {
			t.Fatalf("%d columns: cleanup does not clear the line: %q", columns, out)
		}
		frames := strings.Split(strings.TrimSuffix(out, "\r"+ansi.EraseEntireLine), "\r")[1:]
		if len(frames) < 2 {
			t.Fatalf("%d columns: want several frames, got %q", columns, out)
		}
		for _, frame := range frames {
			if !strings.HasPrefix(frame, ansi.EraseEntireLine) || strings.Contains(frame, "\n") {
				t.Fatalf("%d columns: frame does not redraw one line: %q", columns, frame)
			}
			text := ansi.Strip(frame)
			if width := ansi.StringWidth(text); width >= columns {
				t.Fatalf("%d columns: frame is %d columns and wraps: %q", columns, width, text)
			}
			if !strings.HasSuffix(text, "…") || !strings.Contains(text, "Resolving main on https://") {
				t.Fatalf("%d columns: title not truncated with an ellipsis: %q", columns, text)
			}
		}
	}
}
