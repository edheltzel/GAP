package main

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
)

// table prints a borderless table with a styled header row. Callers use it only on a terminal.
func (r *report) table(headers []string, rows [][]string) {
	header := r.strong(r.colors().Accent)
	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(r.style(r.colors().Border)).
		BorderTop(false).BorderLeft(false).BorderRight(false).BorderBottom(false).
		Headers(headers...).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return header
			}
			return lipgloss.NewStyle().PaddingRight(2)
		})
	for _, row := range rows {
		t.Row(row...)
	}
	fmt.Fprintln(r.out, t)
}

// statusRow is one component in gap status: its pin, store state, and per-skill drift lines.
type statusRow struct {
	id, ref, commit, store string
	skills                 [][]string
	drifted                int
}

// statusSummary prints gap status on a terminal: one table of components, then the drifted
// skills of each component with drift, then a count footer.
func (r *report) statusSummary(rows []statusRow) {
	r.title("gap status", "pins, stores, and skill drift")
	table := make([][]string, 0, len(rows))
	drifted, broken := 0, 0
	for _, row := range rows {
		hex := colorOK
		if row.store != storeOK {
			hex = colorBad
			broken++
		}
		drift := "-"
		if row.store == storeOK {
			drift = fmt.Sprint(row.drifted)
			if row.drifted > 0 {
				drift = r.tone(drift, colorWarn)
				drifted++
			}
		}
		table = append(table, []string{r.tone(row.id, colorAccent), row.ref, row.commit, r.tone(row.store, hex), drift})
	}
	r.framed([]string{"id", "ref", "commit", "store", "drift"}, table)
	for _, row := range rows {
		if row.drifted == 0 {
			continue
		}
		var skills [][]string
		for _, s := range row.skills {
			if s[1] != "drift=0" {
				skills = append(skills, []string{s[0], strings.TrimPrefix(s[1], "drift=")})
			}
		}
		r.printf("\n%s  %s\n", r.bold(row.id, colorAccent), r.tone("skills that differ from the pinned store", colorDim))
		r.framed([]string{"skill", "copies drifted"}, skills)
	}
	r.footer(fmt.Sprintf("%s, %d with drift, %s", plural(len(rows), "component"), drifted, plural(broken, "store problem")))
}

// framed prints a rounded-border table with a styled header row, shrunk to the terminal width when wider.
// Callers use it only on a terminal.
func (r *report) framed(headers []string, rows [][]string) {
	header := r.strong(r.colors().Accent).Padding(0, 1)
	cell := lipgloss.NewStyle().Padding(0, 1)
	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(r.style(r.colors().Border)).
		Headers(headers...).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return header
			}
			return cell
		})
	for _, row := range rows {
		t.Row(row...)
	}
	if r.width > 0 && lipgloss.Width(t.String()) > r.width {
		t = t.Width(r.width)
	}
	fmt.Fprintln(r.out, t)
}
