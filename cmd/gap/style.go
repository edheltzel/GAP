package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
)

// report renders status and doctor output. Color is applied only when the
// destination is a terminal, so piped output and the test capture stay plain
// text and the substrings those tests match are unchanged.
type report struct {
	out io.Writer
	tty bool
}

func newReport(w io.Writer) *report {
	f, ok := w.(*os.File)
	return &report{out: w, tty: ok && isTerminal(f)}
}

func isTerminal(f *os.File) bool {
	fi, err := f.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}

func (r *report) println(a ...any) { fmt.Fprintln(r.out, a...) }

func (r *report) printf(format string, a ...any) { fmt.Fprintf(r.out, format, a...) }

// tone colors text on a terminal and returns it unchanged otherwise.
func (r *report) tone(text, hex string) string {
	if !r.tty {
		return text
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color(hex)).Render(text)
}

const (
	colorAccent = "#7D56F4"
	colorOK     = "#04B575"
	colorWarn   = "#FFB86C"
	colorBad    = "#FF5F87"
	colorDim    = "#767676"
)

func (r *report) heading(text string) {
	r.println(r.tone(text, colorAccent))
}

func (r *report) bold(text, hex string) string {
	if !r.tty {
		return text
	}
	return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(hex)).Render(text)
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
	box := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color(colorAccent)).Padding(0, 1)
	r.println(box.Render(title + "\n\n" + strings.Join(lines, "\n")))
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
		word = r.tone(store, "#04B575")
	case "missing", "changed", "unpinned":
		word = r.tone(store, "#FF5F87")
	}
	r.printf("%s ref=%s commit=%s store=%s\n", r.tone(id, "#7D56F4"), ref, commit, word)
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

func (r *report) table(headers []string, rows [][]string) {
	b := lipgloss.NewStyle().Foreground(lipgloss.Color("#444444"))
	h := lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Bold(true)
	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(b).
		BorderTop(false).BorderLeft(false).BorderRight(false).BorderBottom(false).
		Headers(headers...).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return h
			}
			return lipgloss.NewStyle().PaddingRight(2)
		})
	for _, row := range rows {
		t.Row(row...)
	}
	fmt.Fprintln(r.out, t)
}
