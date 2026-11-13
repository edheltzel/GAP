package main

import (
	"image/color"
	"os"
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
}

// gapPalette picks each GAP color for the terminal background that c resolves.
func gapPalette(c lipgloss.LightDarkFunc) palette {
	return palette{
		Accent: c(lipgloss.Color("#5A3FC0"), lipgloss.Color("#7D56F4")),
		OK:     c(lipgloss.Color("#00875A"), lipgloss.Color("#04B575")),
		Warn:   c(lipgloss.Color("#B35C00"), lipgloss.Color("#FFB86C")),
		Bad:    c(lipgloss.Color("#D7004F"), lipgloss.Color("#FF5F87")),
		Dim:    c(lipgloss.Color("#6C6C6C"), lipgloss.Color("#767676")),
		Border: c(lipgloss.Color("#BCBCBC"), lipgloss.Color("#444444")),
		Text:   c(lipgloss.Color("#1C1C1C"), lipgloss.Color("#DADADA")),
	}
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
