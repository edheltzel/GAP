package main

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
)

func TestProgressBarPinsLastRow(t *testing.T) {
	var buf bytes.Buffer
	rows := 24
	p := newProgressBar(&buf, func() (int, int) { return 80, rows }, darkPalette, true, "sync", 3)
	now := p.start
	p.clock = func() time.Time { return now }
	if got := buf.String(); !strings.HasPrefix(got, "\n\x1b[1A") || !strings.Contains(got, "\x1b[1;23r") || !strings.Contains(got, "\x1b[24;1H") {
		t.Fatalf("start did not reserve the last row: %q", got)
	}
	p.step("policy", nil)
	p.step("echo", darkPalette.group("", "echo"))
	now = now.Add(65 * time.Second)
	buf.Reset()
	p.step("skills", nil)
	line := ansi.Strip(buf.String())
	for _, want := range []string{" 67%", "2/3", "01:05", "sync skills", "\u2501", "\u2591"} {
		if !strings.Contains(line, want) {
			t.Fatalf("bar lacks %q: %q", want, line)
		}
	}
	rows = 30
	buf.Reset()
	p.add(1)
	if got := buf.String(); !strings.Contains(got, "\x1b[1;29r") || !strings.Contains(got, "2/4") {
		t.Fatalf("resize did not move the region: %q", got)
	}
	buf.Reset()
	p.stop()
	p.stop()
	p.step("late", nil)
	if got := buf.String(); got != "\x1b7\x1b[30;1H\x1b[2K\x1b[r\x1b8" {
		t.Fatalf("stop = %q, want one clear and region reset", got)
	}
	var nilBar *progressBar
	nilBar.step("x", nil)
	nilBar.add(1)
	nilBar.stop()
}

func TestProgressBarOffWhenPiped(t *testing.T) {
	if p := startProgress("sync", 3); p != nil {
		p.stop()
		t.Fatal("startProgress drew a bar without a terminal")
	}
}
