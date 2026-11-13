package main

import (
	"fmt"
	"image/color"
	"io"
	"os"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/term"
)

// report renders command output. Layout and color apply only when the destination
// is a terminal, so piped output stays plain text. plain keeps the terminal layout
// without color for NO_COLOR and TERM=dumb; light picks the light-background palette.
type report struct {
	out   io.Writer
	tty   bool
	plain bool
	light bool
	width int
}

func newReport(w io.Writer) *report {
	f, ok := w.(*os.File)
	r := &report{out: w, tty: ok && isTerminal(f)}
	if r.tty {
		r.width, _, _ = term.GetSize(f.Fd())
		r.plain = plainEnv()
		r.light = !r.plain && f == os.Stdout && !terminalIsDark()
	}
	return r
}

func isTerminal(f *os.File) bool {
	fi, err := f.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}

func (r *report) println(a ...any) { fmt.Fprintln(r.out, a...) }

func (r *report) printf(format string, a ...any) { fmt.Fprintf(r.out, format, a...) }

const (
	colorAccent = "#7D56F4"
	colorOK     = "#04B575"
	colorWarn   = "#FFB86C"
	colorBad    = "#FF5F87"
	colorDim    = "#767676"
)

// hue resolves a color name to the palette for this terminal. Any other hex is used as written.
func (r *report) hue(hex string) color.Color {
	p := r.colors()
	switch hex {
	case colorAccent:
		return p.Accent
	case colorOK:
		return p.OK
	case colorWarn:
		return p.Warn
	case colorBad:
		return p.Bad
	case colorDim:
		return p.Dim
	}
	return lipgloss.Color(hex)
}

// tone colors text on a terminal and returns it unchanged otherwise.
func (r *report) tone(text, hex string) string {
	if !r.tty {
		return text
	}
	return r.style(r.hue(hex)).Render(text)
}

func (r *report) heading(text string) {
	r.println(r.tone(text, colorAccent))
}

func (r *report) bold(text, hex string) string {
	if !r.tty {
		return text
	}
	return r.strong(r.hue(hex)).Render(text)
}

// title opens a command's terminal output with its name and a dim detail. Piped output gets nothing.
func (r *report) title(name, detail string) {
	if !r.tty {
		return
	}
	r.printf("%s  %s\n\n", r.bold(name, colorAccent), r.tone(detail, colorDim))
}

// footer prints a dim summary line under a terminal table. Piped output gets nothing.
func (r *report) footer(text string) {
	if !r.tty {
		return
	}
	r.println(r.tone(text, colorDim))
}

// section opens one sync phase on a terminal. Piped output gets nothing, so scripts see the same lines as before.
func (r *report) section(title, detail string) {
	if !r.tty {
		return
	}
	r.printf("\n%s %s  %s\n", r.tone("▍", colorAccent), r.bold(title, colorAccent), r.tone(detail, colorDim))
}

// wrote reports a written path: the bare path when piped, a checked home-relative path on a terminal.
func (r *report) wrote(path string) {
	if !r.tty {
		r.println(path)
		return
	}
	r.printf("  %s %s\n", r.tone("✓", colorOK), r.tone(tildePath(path), colorDim))
}

// result prints a terminal-only outcome line for a phase.
func (r *report) result(ok bool, text string) {
	if !r.tty {
		return
	}
	mark, hex := "✓", colorOK
	if !ok {
		mark, hex = "✗", colorBad
	}
	r.printf("  %s %s\n", r.tone(mark, hex), text)
}

// done prints an outcome line everywhere: the plain text when piped, a marked line on a terminal.
func (r *report) done(ok bool, text string) {
	if !r.tty {
		r.println(text)
		return
	}
	r.result(ok, text)
}

// warn prints plain on a pipe and as a flagged line on a terminal.
func (r *report) warn(plain, name, detail string) {
	if !r.tty {
		r.println(plain)
		return
	}
	r.printf("  %s %s  %s\n", r.tone("!", colorWarn), r.tone(name, colorWarn), r.tone(detail, colorDim))
}

func tildePath(p string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" || !strings.HasPrefix(p, home+string(os.PathSeparator)) {
		return p
	}
	return "~" + p[len(home):]
}

// preview draws the sync plan as a framed panel on a terminal.
func (r *report) preview(groups []syncGroup, sel map[string]bool) {
	width := 0
	for _, g := range groups {
		width = max(width, len(g.ID))
	}
	lines := make([]string, 0, len(groups))
	for _, g := range groups {
		body, pin, _ := strings.Cut(g.Summary, " @ ")
		id := fmt.Sprintf("%-*s", width, g.ID)
		if !sel[g.ID] {
			lines = append(lines, r.tone("○ "+id+"  "+body+"  skipped", colorDim))
			continue
		}
		line := r.tone("●", colorAccent) + " " + r.bold(id, colorAccent) + "  " + body
		if pin != "" {
			line += "  " + r.tone(pin, colorDim)
		}
		lines = append(lines, line)
	}
	title := r.bold("gap sync", colorAccent) + r.tone("  will publish and install", colorDim)
	box := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)
	if !r.plain {
		box = box.BorderForeground(r.hue(colorAccent))
	}
	content := title + "\n\n" + strings.Join(lines, "\n")
	if r.width > box.GetHorizontalFrameSize() {
		box = box.Width(min(r.width, lipgloss.Width(content)+box.GetHorizontalFrameSize()))
	}
	r.println(box.Render(content))
}

// next lists the commands that move the report forward. Empty does nothing.
func (r *report) next(steps ...string) {
	if len(steps) == 0 {
		return
	}
	r.heading("Next")
	for _, step := range steps {
		r.printf("  %s\n", step)
	}
}

func (r *report) statusLine(id, ref, commit, store string) {
	word := store
	switch store {
	case "ok":
		word = r.tone(store, colorOK)
	case "missing", "changed", "unpinned":
		word = r.tone(store, colorBad)
	}
	r.printf("%s ref=%s commit=%s store=%s\n", r.tone(id, colorAccent), ref, commit, word)
}

// doctorTable prints "name: detail" lines, as a table on a terminal.
func (r *report) doctorTable(rows [][]string) {
	if !r.tty {
		for _, row := range rows {
			r.printf("%s: %s\n", row[0], row[1])
		}
		return
	}
	r.table([]string{"name", "detail"}, rows)
}

// driftTable prints one "name drift=N" line per skill, as a table on a terminal.
func (r *report) driftTable(rows [][]string) {
	if !r.tty {
		for _, row := range rows {
			r.printf("  %s %s\n", row[0], row[1])
		}
		return
	}
	r.table([]string{"skill", "drift"}, rows)
}
