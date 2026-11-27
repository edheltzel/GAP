package main

import tea "charm.land/bubbletea/v2"

type vimMotion int

const (
	vimNone vimMotion = iota
	vimPending
	vimTop
	vimHalfDown
	vimHalfUp
	vimPageDown
	vimPageUp
	vimHigh
	vimMid
	vimLow
	vimTabNext
	vimTabPrev
)

func (m *tuiModel) vimList() bool {
	return m.mode == modeBrowse && m.tab != tabStatus
}

func (m *tuiModel) vimMotion(msg tea.KeyPressMsg) vimMotion {
	s := msg.String()
	if m.chord == "g" {
		m.chord = ""
		switch {
		case s == "g":
			return vimTop
		case s == "t" && m.mode == modeBrowse:
			return vimTabNext
		case s == "T" && m.mode == modeBrowse:
			return vimTabPrev
		}
		return vimNone
	}
	if s == "g" {
		m.chord = "g"
		return vimPending
	}
	if m.mode == modeSummary {
		return vimNone
	}
	switch s {
	case "ctrl+d":
		return vimHalfDown
	case "ctrl+u":
		return vimHalfUp
	case "ctrl+f":
		return vimPageDown
	case "ctrl+b":
		return vimPageUp
	}
	if !m.vimList() {
		return vimNone
	}
	switch s {
	case "H":
		return vimHigh
	case "M":
		return vimMid
	case "L":
		return vimLow
	}
	return vimNone
}

func (m *tuiModel) chordHint() string {
	switch {
	case m.mode == modeSummary:
		return "gg first step"
	case m.mode == modeBrowse:
		return "gg top, gt/gT next or previous view"
	}
	return "gg top"
}

func (m *tuiModel) applyVim(mv vimMotion) tea.Cmd {
	switch mv {
	case vimPending:
		return nil
	case vimTabNext, vimTabPrev:
		step := 1
		if mv == vimTabPrev {
			step = len(tabNames) - 1
		}
		return m.switchTab((m.tab + step) % len(tabNames))
	}
	if m.mode == modeSummary {
		m.jobSel = 0
		m.refreshPane()
		m.vp.GotoBottom()
		return nil
	}
	if !m.vimList() {
		switch mv {
		case vimTop:
			m.vp.GotoTop()
		case vimHalfDown:
			m.vp.HalfPageDown()
		case vimHalfUp:
			m.vp.HalfPageUp()
		case vimPageDown:
			m.vp.PageDown()
		case vimPageUp:
			m.vp.PageUp()
		}
		return nil
	}
	n := m.count(m.tab)
	if n == 0 {
		return nil
	}
	h := m.rowsFor(m.tab)
	start, end := m.listWindow(m.tab)
	sel := m.sel[m.tab]
	switch mv {
	case vimTop:
		sel, m.top[m.tab] = 0, 0
	case vimHalfDown, vimHalfUp, vimPageDown, vimPageUp:
		d := map[vimMotion]int{vimHalfDown: max(h/2, 1), vimHalfUp: -max(h/2, 1), vimPageDown: h, vimPageUp: -h}[mv]
		sel += d
		m.top[m.tab] = min(max(start+d, 0), max(n-h, 0))
	case vimHigh:
		sel = start
	case vimMid:
		sel = (start + end - 1) / 2
	case vimLow:
		sel = end - 1
	}
	m.sel[m.tab] = min(max(sel, 0), n-1)
	m.top[m.tab], _ = m.listWindow(m.tab)
	m.refreshPane()
	return nil
}

func (m *tuiModel) rowsFor(tab int) int {
	if tab == tabItems {
		return max(m.bodyHeight()-3, 1)
	}
	return max(m.bodyHeight()-2, 1)
}

func (m *tuiModel) listWindow(tab int) (int, int) {
	n, h, sel := m.count(tab), m.rowsFor(tab), m.sel[tab]
	top := min(m.top[tab], sel)
	top = max(top, sel-h+1)
	top = max(min(top, n-h), 0)
	return top, min(top+h, n)
}
