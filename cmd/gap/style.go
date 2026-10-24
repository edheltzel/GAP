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

func (r *report) heading(text string) {
	r.println(r.tone(text, "#7D56F4"))
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

// statusLine keeps the exact "id ref= commit= store=" text the tests match,
// coloring only the store word on a terminal.
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

// table prints aligned columns. On a terminal the header is colored and the
// rows sit under a rule; piped, it is plain space-separated columns.
func (r *report) table(headers []string, rows [][]string) {
	if !r.tty {
		r.println(strings.Join(headers, "  "))
		for _, row := range rows {
			r.println(strings.Join(row, "  "))
		}
		return
	}
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
