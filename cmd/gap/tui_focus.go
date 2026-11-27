package main

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func (m *tuiModel) sideOn() bool {
	return m.side && m.mode == modeBrowse && m.tab != tabStatus && !m.stacked()
}

func (m *tuiModel) sideLines() []string {
	lines := m.itemDetailLines()
	if m.tab == tabProjects {
		lines = m.projectDetailLines()
	}
	w := max(m.width-m.listWidth()-4, 1)
	var out []string
	for _, l := range lines {
		out = append(out, strings.Split(lipgloss.Wrap(l, w, ""), "\n")...)
	}
	return out
}

func (m *tuiModel) sideMax() int {
	return max(len(m.sideLines())-(m.bodyHeight()-2), 0)
}

func (m *tuiModel) focusKey(msg tea.KeyPressMsg) (tea.Cmd, bool) {
	s := msg.String()
	if m.stacked() {
		return nil, false
	}
	if !m.side {
		if s == "l" || s == "right" {
			m.side, m.sideTop, m.chord = true, 0, ""
			return nil, true
		}
		return nil, false
	}
	if m.chord == "g" {
		m.chord = ""
		if s == "g" {
			m.sideTop = 0
			return nil, true
		}
	}
	h := max(m.bodyHeight()-2, 1)
	switch s {
	case "h", "left", "esc":
		m.side = false
	case "j", "down":
		m.sideTop++
	case "k", "up":
		m.sideTop--
	case "ctrl+d":
		m.sideTop += max(h/2, 1)
	case "ctrl+u":
		m.sideTop -= max(h/2, 1)
	case "ctrl+f", "pgdown":
		m.sideTop += h
	case "ctrl+b", "pgup":
		m.sideTop -= h
	case "G", "end":
		m.sideTop = m.sideMax()
	case "home":
		m.sideTop = 0
	case "g":
		m.chord = "g"
		return nil, true
	default:
		return nil, false
	}
	m.sideTop = min(max(m.sideTop, 0), m.sideMax())
	return nil, true
}

func (m *tuiModel) modeName() string {
	switch {
	case m.showHelp:
		return "HELP"
	case m.mode == modeFilter:
		return "FILTER"
	case m.mode == modeCmd:
		return "CMD"
	case m.mode == modeSearch:
		return "SEARCH"
	case m.mode == modeForm:
		return "EDIT"
	case m.mode == modeConfirm, m.mode == modeAsk:
		return "CONFIRM"
	case m.mode == modeRun:
		return "RUN"
	case m.mode == modeSummary:
		return "DONE"
	case m.mode == modeDetail, m.sideOn():
		return "DETAIL"
	}
	return "NORMAL"
}

func (m *tuiModel) modePill() string {
	name := m.modeName()
	bg := m.st.pal.Accent
	switch name {
	case "FILTER", "CMD", "SEARCH", "EDIT":
		bg = m.st.pal.OK
	case "CONFIRM", "RUN":
		bg = m.st.pal.Warn
	}
	return lipgloss.NewStyle().Bold(true).Foreground(m.st.pal.OnAcc).Background(bg).Render(" " + name + " ")
}
