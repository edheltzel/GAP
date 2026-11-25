package main

import (
	"hash/fnv"
	"image/color"
	"os"
	"strings"
	"sync"

	"charm.land/fang/v2"
	"charm.land/lipgloss/v2"
)

// palette is the GAP color set for one background.
type palette struct {
	Accent color.Color
	OK     color.Color
	Warn   color.Color
	Bad    color.Color
	Dim    color.Color
	Border color.Color
	Text   color.Color
	Deep   color.Color
	Glow   color.Color
	OnAcc  color.Color
	Groups []color.Color
}

// gapPalette picks each GAP color for the terminal background that c resolves.
func gapPalette(c lipgloss.LightDarkFunc) palette {
	light := []string{"#6E4FA8", "#B4237A", "#9A6700", "#C4214A", "#00875A", "#2F5FB3", "#8A4FB8", "#00838F"}
	dark := []string{"#B392D7", "#FF79C6", "#F2C55C", "#FF6B8B", "#5FD7A7", "#82AAFF", "#E8D6FF", "#7FDBCA"}
	groups := make([]color.Color, len(dark))
	for i := range dark {
		groups[i] = c(lipgloss.Color(light[i]), lipgloss.Color(dark[i]))
	}
	return palette{
		Accent: c(lipgloss.Color("#6E4FA8"), lipgloss.Color("#B392D7")),
		OK:     c(lipgloss.Color("#00875A"), lipgloss.Color("#5FD7A7")),
		Warn:   c(lipgloss.Color("#9A6700"), lipgloss.Color("#F2C55C")),
		Bad:    c(lipgloss.Color("#C4214A"), lipgloss.Color("#FF6B8B")),
		Dim:    c(lipgloss.Color("#6C6F85"), lipgloss.Color("#7A7F99")),
		Border: c(lipgloss.Color("#BCC0CC"), lipgloss.Color("#4A4E69")),
		Text:   c(lipgloss.Color("#1E1E2E"), lipgloss.Color("#DADAE8")),
		Deep:   c(lipgloss.Color("#9C88C7"), lipgloss.Color("#6E5A9A")),
		Glow:   c(lipgloss.Color("#4B2F86"), lipgloss.Color("#E8D6FF")),
		OnAcc:  c(lipgloss.Color("#FFFFFF"), lipgloss.Color("#1E1E2E")),
		Groups: groups,
	}
}

// group is a component's color: its configured #RRGGBB, otherwise a palette hue picked from its id, so it matches everywhere.
func (p palette) group(configured, id string) color.Color {
	if validColor(configured) {
		return lipgloss.Color(configured)
	}
	h := fnv.New32a()
	h.Write([]byte(id))
	return p.Groups[h.Sum32()%uint32(len(p.Groups))]
}

func validColor(s string) bool {
	if len(s) != 7 || s[0] != '#' {
		return false
	}
	for _, r := range s[1:] {
		if !strings.ContainsRune("0123456789abcdefABCDEF", r) {
			return false
		}
	}
	return true
}

// gapColorScheme themes fang's help and error output with the GAP palette.
func gapColorScheme(c lipgloss.LightDarkFunc) fang.ColorScheme {
	p := gapPalette(c)
	s := fang.DefaultColorScheme(c)
	s.Title = p.Accent
	s.Program = p.Accent
	s.Command = p.OK
	s.Flag = p.Warn
	s.FlagDefault = p.Dim
	s.Comment = p.Dim
	s.DimmedArgument = p.Dim
	s.QuotedString = p.OK
	return s
}

var (
	darkPalette  = gapPalette(lipgloss.LightDark(true))
	lightPalette = gapPalette(lipgloss.LightDark(false))
)

var terminalIsDark = sync.OnceValue(func() bool {
	return lipgloss.HasDarkBackground(os.Stdin, os.Stdout)
})

// plainEnv reports whether NO_COLOR or TERM=dumb asks for output without styling.
func plainEnv() bool {
	return os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb"
}

// colors is the palette for this report: the terminal's background when the report writes to stdout, dark otherwise.
func (r *report) colors() palette {
	if r.light {
		return lightPalette
	}
	return darkPalette
}

// style returns a foreground style, or an empty style when the report is plain.
func (r *report) style(fg color.Color) lipgloss.Style {
	if !r.tty || r.plain {
		return lipgloss.NewStyle()
	}
	return lipgloss.NewStyle().Foreground(fg)
}

// strong is style plus bold, again empty when the report is plain.
func (r *report) strong(fg color.Color) lipgloss.Style {
	if !r.tty || r.plain {
		return lipgloss.NewStyle()
	}
	return r.style(fg).Bold(true)
}
