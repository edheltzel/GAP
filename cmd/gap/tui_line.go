package main

import (
	"fmt"
	"slices"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

var tuiCommands = []string{"check", "doctor", "filter", "help", "quit", "refresh", "restore", "search", "status", "sync", "update"}

func (m *tuiModel) rows(tab int) []int {
	var n int
	switch tab {
	case tabItems:
		n = len(m.items)
	case tabProjects:
		n = len(m.projects)
	}
	q := strings.ToLower(m.filter[tab])
	idx := make([]int, 0, n)
	for i := range n {
		if q == "" || strings.Contains(strings.ToLower(m.rowText(tab, i)), q) {
			idx = append(idx, i)
		}
	}
	return idx
}

func (m *tuiModel) total(tab int) int {
	if tab == tabProjects {
		return len(m.projects)
	}
	return len(m.items)
}

func (m *tuiModel) rowText(tab, i int) string {
	if tab == tabProjects {
		return m.projects[i].Root
	}
	r := m.items[i]
	return r.ID
}

func (m *tuiModel) openLine(mode tuiMode, prompt, value string) tea.Cmd {
	in := textinput.New()
	in.Prompt = prompt
	in.SetWidth(max(m.width/3, 10))
	in.SetValue(value)
	in.CursorEnd()
	m.line, m.mode, m.chord = in, mode, ""
	return m.line.Focus()
}

func (m *tuiModel) openFilter() tea.Cmd {
	if m.tab == tabStatus {
		m.note = "Status has no list to filter."
		return nil
	}
	m.filterWas = m.filter[m.tab]
	return m.openLine(modeFilter, "/ ", m.filter[m.tab])
}

func (m *tuiModel) setFilter(q string) {
	m.filter[m.tab] = q
	m.sel[m.tab], m.top[m.tab] = 0, 0
	m.refreshPane()
}

func (m *tuiModel) lineUpdate(msg tea.Msg) tea.Cmd {
	if k, ok := msg.(tea.KeyPressMsg); ok {
		switch {
		case k.String() == "ctrl+c":
			m.mode = modeBrowse
			return m.key(k)
		case k.Code == tea.KeyEscape:
			if m.mode == modeFilter {
				m.setFilter(m.filterWas)
			}
			m.mode = modeBrowse
			return nil
		case k.Code == tea.KeyEnter:
			v := strings.TrimSpace(m.line.Value())
			mode := m.mode
			m.mode = modeBrowse
			if mode == modeFilter {
				if n := len(m.rows(m.tab)); v != "" {
					m.note = matches(n) + " for " + v + ". esc clears."
				}
				return nil
			}
			return m.command(v)
		case k.Code == tea.KeyTab && m.mode == modeCmd:
			m.line.SetValue(complete(m.line.Value()))
			m.line.CursorEnd()
			return nil
		case k.Code == tea.KeyBackspace && m.line.Value() == "":
			m.mode = modeBrowse
			return nil
		}
	}
	before := m.line.Value()
	var cmd tea.Cmd
	m.line, cmd = m.line.Update(msg)
	if m.mode == modeFilter && m.line.Value() != before {
		m.setFilter(strings.TrimSpace(m.line.Value()))
	}
	return cmd
}

func matches(n int) string {
	if n == 1 {
		return "1 match"
	}
	return fmt.Sprintf("%d matches", n)
}

func complete(v string) string {
	name, rest, spaced := strings.Cut(strings.TrimLeft(v, " "), " ")
	if spaced {
		return v
	}
	var hits []string
	for _, c := range tuiCommands {
		if strings.HasPrefix(c, name) {
			hits = append(hits, c)
		}
	}
	if len(hits) == 1 {
		return hits[0] + " " + rest
	}
	return v
}

func (m *tuiModel) lineHint() string {
	if m.mode == modeFilter {
		return matches(len(m.rows(m.tab))) + "  enter keep  esc clear"
	}
	name, _, _ := strings.Cut(strings.TrimSpace(m.line.Value()), " ")
	var hits []string
	for _, c := range tuiCommands {
		if strings.HasPrefix(c, name) {
			hits = append(hits, c)
		}
	}
	if len(hits) == 0 {
		return "no such command"
	}
	return strings.Join(hits, " ") + "  tab complete"
}

func (m *tuiModel) command(line string) tea.Cmd {
	name, arg, _ := strings.Cut(line, " ")
	arg = strings.TrimSpace(arg)
	if name == "" {
		return nil
	}
	var hits []string
	for _, c := range tuiCommands {
		if strings.HasPrefix(c, name) {
			hits = append(hits, c)
		}
	}
	switch {
	case slices.Contains(tuiCommands, name):
		hits = []string{name}
	case name == "q" || name == "q!":
		hits = []string{"quit"}
	case name == "w":
		hits = []string{"sync"}
	}
	if len(hits) != 1 {
		m.note = "Unknown command :" + name + ". ? lists the keys."
		if len(hits) > 1 {
			m.note = ":" + name + " could be " + strings.Join(hits, ", ") + "."
		}
		return nil
	}
	known := func(id string) bool {
		return slices.ContainsFunc(m.items, func(r itemRow) bool { return r.ID == id })
	}
	switch hits[0] {
	case "quit":
		m.quitting = true
		return tea.Quit
	case "help":
		m.mode = modeBrowse
		m.openHelp()
	case "refresh":
		m.note = "Refreshing."
		return m.reload()
	case "sync":
		return m.handOver("sync")
	case "doctor":
		return m.handOver("doctor")
	case "status":
		return m.switchTab(tabStatus)
	case "filter":
		if m.tab == tabStatus {
			m.note = "Status has no list to filter."
			return nil
		}
		m.setFilter(arg)
	case "search":
		cmd := m.openSearch()
		if arg == "" {
			return cmd
		}
		m.search.input.SetValue(arg)
		m.search.input.CursorEnd()
		seq := m.search.seq
		return tea.Batch(cmd, func() tea.Msg { return searchTickMsg{seq: seq} })
	case "check", "update", "restore":
		if arg != "" && !known(arg) {
			m.note = "No item " + arg + "."
			return nil
		}
		if arg == "" && hits[0] == "restore" {
			return m.handOver("restore")
		}
		if arg == "" {
			if hits[0] == "update" {
				m.confirmUpdateAll()
				return nil
			}
			return m.run([]tuiJob{{label: "check all", args: []string{"check"}}})
		}
		if hits[0] == "restore" {
			return m.handOver("restore", arg)
		}
		return m.run([]tuiJob{{label: hits[0] + " " + arg, name: arg, color: m.groupColor(arg), args: []string{hits[0], "--", arg}}})
	}
	return nil
}
