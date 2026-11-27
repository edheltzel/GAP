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

func (m *tuiModel) vimMotion(msg tea.KeyPressMsg) vimMotion {
	s := msg.String()
	if m.chord == "g" {
		m.chord = ""
		switch s {
		case "g":
			return vimTop
		case "t":
			return vimTabNext
		case "T":
			return vimTabPrev
		}
		return vimNone
	}
	switch s {
	case "g":
		m.chord = "g"
		return vimPending
	case "ctrl+d":
		return vimHalfDown
	case "ctrl+u":
		return vimHalfUp
	case "ctrl+f":
		return vimPageDown
	case "ctrl+b":
		return vimPageUp
	case "H":
		return vimHigh
	case "M":
		return vimMid
	case "L":
		return vimLow
	}
	return vimNone
}

func (m *tuiModel) applyVim(mv vimMotion) tea.Cmd {
	switch mv {
	case vimPending:
		return nil
	case vimTabNext, vimTabPrev:
		if m.mode != modeBrowse {
			return nil
		}
		step := 1
		if mv == vimTabPrev {
			step = len(tabNames) - 1
		}
		return m.switchTab((m.tab + step) % len(tabNames))
	}
	if m.mode == modeDetail || (m.mode == modeBrowse && m.tab == tabStatus) {
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
	if m.mode == modeSummary {
		h := max(m.bodyHeight()-2, 1)
		start, end := m.window(len(m.jobs), m.jobSel, h)
		m.jobSel = vimSel(mv, m.jobSel, len(m.jobs), h, start, end)
		m.refreshPane()
		return nil
	}
	if n := m.count(m.tab); n > 0 {
		start, end := m.listWindow(m.tab)
		m.sel[m.tab] = vimSel(mv, m.sel[m.tab], n, m.rowsFor(m.tab), start, end)
		m.top[m.tab], _ = m.listWindow(m.tab)
		m.refreshPane()
	}
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

func vimSel(mv vimMotion, sel, n, h, start, end int) int {
	switch mv {
	case vimTop:
		sel = 0
	case vimHalfDown:
		sel += max(h/2, 1)
	case vimHalfUp:
		sel -= max(h/2, 1)
	case vimPageDown:
		sel += h
	case vimPageUp:
		sel -= h
	case vimHigh:
		sel = start
	case vimMid:
		sel = (start + end - 1) / 2
	case vimLow:
		sel = end - 1
	}
	return min(max(sel, 0), max(n-1, 0))
}
