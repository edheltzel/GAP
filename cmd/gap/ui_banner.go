package main

import (
	"strings"

	"charm.land/lipgloss/v2"
)

var wordmark = []string{
	"▄▄▄▄▄▄▄ ▄▄▄▄▄▄▄ ▄▄▄▄▄▄▄",
	"▓       ▓     ▓ ▓     ▓",
	"▒   ▀▀▒ ▒▀▀▀▀▀▒ ▒ ▀▀▀▀▀",
	"░▄▄▄▄▄░ ░     ░ ░      ",
}

const (
	bannerCols = 23
	bannerRows = 4
)

func bannerStyles(p palette) []lipgloss.Style {
	styles := make([]lipgloss.Style, bannerCols)
	for i, c := range lipgloss.Blend1D(bannerCols, p.Deep, p.Accent, p.Glow, p.OK, p.Accent, p.Deep) {
		styles[i] = lipgloss.NewStyle().Foreground(c)
	}
	return styles
}

func banner(styles []lipgloss.Style, tagline string, frame int) string {
	lines := make([]string, 0, len(wordmark))
	for y, row := range wordmark {
		var b strings.Builder
		for x, r := range []rune(row) {
			if r == ' ' {
				b.WriteRune(' ')
				continue
			}
			b.WriteString(styles[(x+frame)%bannerCols].Render(string(r)))
		}
		if y == 0 {
			b.WriteString("  " + tagline)
		}
		lines = append(lines, b.String())
	}
	return strings.Join(lines, "\n")
}
