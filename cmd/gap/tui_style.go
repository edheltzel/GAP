package main

import "charm.land/lipgloss/v2"

type tuiStyles struct {
	accent, ok, warn, bad, faint, text, selected, title, tabOn, tabOff, pane, border lipgloss.Style
}

func newTUIStyles(p palette) tuiStyles {
	return tuiStyles{
		accent:   lipgloss.NewStyle().Foreground(p.Accent),
		ok:       lipgloss.NewStyle().Foreground(p.OK),
		warn:     lipgloss.NewStyle().Foreground(p.Warn),
		bad:      lipgloss.NewStyle().Foreground(p.Bad),
		faint:    lipgloss.NewStyle().Foreground(p.Dim),
		text:     lipgloss.NewStyle().Foreground(p.Text),
		selected: lipgloss.NewStyle().Foreground(p.Accent).Bold(true),
		title:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Background(p.Accent),
		tabOn:    lipgloss.NewStyle().Bold(true).Foreground(p.Accent).Underline(true),
		tabOff:   lipgloss.NewStyle().Foreground(p.Dim),
		pane:     lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(p.Border).Padding(0, 1),
		border:   lipgloss.NewStyle().Foreground(p.Border),
	}
}
