package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

const searchPause = 300 * time.Millisecond

type searchState struct {
	input   textinput.Model
	hits    []skillHit
	sel     int
	seq     int
	busy    bool
	err     error
	queried string
}

type searchTickMsg struct{ seq int }

type searchDoneMsg struct {
	seq   int
	query string
	hits  []skillHit
	err   error
}

func (m *tuiModel) openSearch() tea.Cmd {
	in := textinput.New()
	in.Prompt = "/ "
	in.Placeholder = "search skills.sh"
	in.SetWidth(max(m.width-8, 10))
	m.search = &searchState{input: in}
	m.mode = modeSearch
	return m.search.input.Focus()
}

func (m *tuiModel) searchUpdate(msg tea.Msg) tea.Cmd {
	s := m.search
	switch msg := msg.(type) {
	case searchTickMsg:
		q := strings.TrimSpace(s.input.Value())
		if msg.seq != s.seq || q == "" || q == s.queried {
			return nil
		}
		s.busy, s.queried = true, q
		seq, find := s.seq, m.findSkills
		return func() tea.Msg {
			hits, err := find(context.Background(), q, 30)
			return searchDoneMsg{seq: seq, query: q, hits: hits, err: err}
		}
	case searchDoneMsg:
		if msg.query != s.queried {
			return nil
		}
		s.busy, s.hits, s.err, s.sel = false, msg.hits, msg.err, 0
		return nil
	case tea.KeyPressMsg:
		switch {
		case msg.String() == "ctrl+c":
			m.search, m.mode = nil, modeBrowse
			return m.key(msg)
		case msg.Code == tea.KeyEscape:
			m.search, m.mode = nil, modeBrowse
			return nil
		case msg.Code == tea.KeyDown || key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+n"))):
			s.sel = min(s.sel+1, max(len(s.hits)-1, 0))
			return nil
		case msg.Code == tea.KeyUp || key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+p"))):
			s.sel = max(s.sel-1, 0)
			return nil
		case msg.Code == tea.KeyEnter:
			if s.sel < len(s.hits) {
				return m.installHit(s.hits[s.sel])
			}
			return nil
		}
		before := s.input.Value()
		var cmd tea.Cmd
		s.input, cmd = s.input.Update(msg)
		if s.input.Value() == before {
			return cmd
		}
		s.seq++
		seq := s.seq
		return tea.Batch(cmd, tea.Tick(searchPause, func(time.Time) tea.Msg { return searchTickMsg{seq: seq} }))
	}
	var cmd tea.Cmd
	s.input, cmd = s.input.Update(msg)
	return cmd
}

// installedHit names where a skill from a hit is already installed, or "".
func (m *tuiModel) installedHit(h skillHit) string {
	for _, r := range m.items {
		if !strings.EqualFold(r.Repo, h.Source) {
			continue
		}
		for _, s := range r.Skills {
			if len(s) > 0 && s[0] == h.Skill {
				return r.Scope
			}
		}
		for _, n := range r.Names {
			if n == h.Skill {
				return r.Scope
			}
		}
	}
	return ""
}

func (m *tuiModel) installHit(h skillHit) tea.Cmd {
	project := m.here()
	args := func(scope string) []string {
		where := "--global"
		if scope == scopeProject {
			where = "--project"
		}
		return []string{"add", h.Source, "--skill", h.Skill, where}
	}
	if project == "" {
		m.search, m.mode = nil, modeBrowse
		return m.handOverIn("", args(scopeGlobal)...)
	}
	scope := scopeGlobal
	m.search = nil
	return m.openForm("Install "+h.Skill, func() tea.Cmd {
		dir := ""
		if scope == scopeProject {
			dir = project
		}
		return m.handOverIn(dir, args(scope)...)
	}, huh.NewSelect[string]().Title("Install "+h.Skill+" from "+h.Source).
		Description(fmt.Sprintf("%s installs on skills.sh", humanCount(h.Installs))).
		Options(m.scopeOptions(project)...).Value(&scope))
}

func humanCount(n int) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1e6)
	case n >= 1_000:
		return fmt.Sprintf("%.1fK", float64(n)/1e3)
	}
	return fmt.Sprint(n)
}

func (m *tuiModel) searchView() string {
	s := m.search
	head := s.input.View()
	status := ""
	switch {
	case s.busy:
		status = m.spin.View() + m.st.faint.Render(" searching")
	case s.err != nil:
		status = m.st.bad.Render(s.err.Error())
	case s.queried != "" && len(s.hits) == 0:
		status = m.st.faint.Render("no installable hits for " + s.queried)
	case s.queried == "":
		status = m.st.faint.Render("type to search the skills.sh catalog")
	}
	h := m.bodyHeight()
	listH := max(h-4, 1)
	nameW, srcW := 8, 8
	for _, hit := range s.hits {
		nameW, srcW = max(nameW, ansi.StringWidth(hit.Skill)), max(srcW, ansi.StringWidth(hit.Source))
	}
	nameW, srcW = min(nameW, 32), min(srcW, 34)
	lines := []string{head, status}
	start, end := m.window(len(s.hits), s.sel, listH)
	for i := start; i < end; i++ {
		hit := s.hits[i]
		mark := "  "
		if hit.Trusted {
			mark = m.st.ok.Render("\u2713 ")
		}
		name := fmt.Sprintf("%-*s", nameW, ansi.Truncate(hit.Skill, nameW, "\u2026"))
		name = lipgloss.NewStyle().Foreground(m.st.pal.group("", hit.Source)).Render(name)
		line := m.pointer(i == s.sel) + mark + name + "  " + m.st.faint.Render(fmt.Sprintf("%-*s", srcW, ansi.Truncate(hit.Source, srcW, "\u2026"))) + "  " + fmt.Sprintf("%6s", humanCount(hit.Installs))
		if where := m.installedHit(hit); where != "" {
			line += "  " + m.st.accent.Render("installed "+where)
		}
		lines = append(lines, ansi.Truncate(line, max(m.width-4, 1), "\u2026"))
	}
	return m.st.pane.Width(m.width).Height(h).Render(strings.Join(lines, "\n"))
}
