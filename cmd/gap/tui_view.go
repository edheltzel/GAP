package main

import (
	"fmt"
	"math"
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
	"github.com/charmbracelet/x/ansi"
)

func (m *tuiModel) View() tea.View {
	v := tea.NewView(m.render())
	v.AltScreen = true
	v.WindowTitle = "gap"
	return v
}

func (m *tuiModel) bodyHeight() int { return max(m.height-5, 6) }

func (m *tuiModel) stacked() bool { return m.width < 70 }

func (m *tuiModel) listWidth() int {
	if m.stacked() {
		return m.width
	}
	return min(m.width*3/5, 84)
}

func (m *tuiModel) running() bool { return m.mode == modeRun || m.mode == modeAsk }

func (m *tuiModel) refreshPane() {
	inner := m.bodyHeight() - 3
	switch {
	case m.running() || m.mode == modeSummary:
		if m.stacked() {
			m.vp.SetWidth(max(m.width-4, 1))
			m.vp.SetHeight(max(inner-1, 1))
		} else {
			m.vp.SetWidth(max(m.width-m.listWidth()-4, 1))
			m.vp.SetHeight(inner)
		}
		if m.jobSel < len(m.jobs) {
			lines := m.jobs[m.jobSel].lines
			if len(lines) == 0 {
				lines = []string{m.st.faint.Render("Waiting to run.")}
			}
			m.vp.SetContentLines(lines)
			m.vp.GotoBottom()
		}
	case m.tab == tabStatus:
		m.vp.SetWidth(max(m.width-4, 1))
		m.vp.SetHeight(inner + 1)
		m.vp.SetContentLines(strings.Split(m.status, "\n"))
	}
}

func (m *tuiModel) header() string {
	name := m.st.title.Render(" gap ")
	title := name + " " + m.st.faint.Render("components, utilities, projects")
	stat := fmt.Sprintf("%d items  %d marked  %d projects", len(m.items), len(m.marked), len(m.projects))
	for _, t := range []string{title, name} {
		if pad := m.width - lipgloss.Width(t) - lipgloss.Width(stat); pad >= 1 {
			return t + strings.Repeat(" ", pad) + m.st.faint.Render(stat)
		}
	}
	return ansi.Truncate(title, m.width, "\u2026")
}

func (m *tuiModel) panes(list, detail func() string) string {
	if m.stacked() {
		return list()
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, list(), detail())
}

func (m *tuiModel) render() string {
	var tabs []string
	for i, name := range tabNames {
		label := fmt.Sprintf("%d %s", i+1, name)
		if i == m.tab {
			tabs = append(tabs, m.st.tabOn.Render(label))
		} else {
			tabs = append(tabs, m.st.tabOff.Render(label))
		}
	}
	var body string
	switch m.mode {
	case modeConfirm:
		body = m.confirmView()
	case modeRun:
		body = m.runView()
	case modeAsk:
		body = m.askView()
	case modeSummary:
		body = m.summaryView()
	default:
		switch m.tab {
		case tabItems:
			body = m.panes(m.itemList, m.itemDetail)
		case tabStatus:
			body = m.statusView()
		case tabProjects:
			body = m.panes(m.projectList, m.projectDetail)
		}
	}
	note := " " + m.note
	if m.loadErr != nil && m.note == "" {
		note = " " + m.st.bad.Render(firstLine(m.loadErr.Error()))
	}
	m.help.SetWidth(max(m.width-1, 1))
	fit := func(s string) string { return ansi.Truncate(s, m.width, "\u2026") }
	return lipgloss.JoinVertical(lipgloss.Left, m.header(), fit(" "+strings.Join(tabs, "   ")), body, fit(note), fit(" "+m.help.ShortHelpView(m.helpKeys())))
}

func firstLine(s string) string {
	line, _, _ := strings.Cut(s, "\n")
	return line
}

func (m *tuiModel) helpKeys() []key.Binding {
	k := m.keys
	switch m.mode {
	case modeConfirm:
		return []key.Binding{k.Confirm, k.Back}
	case modeRun:
		return []key.Binding{k.Up, k.Down, key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "stop"))}
	case modeAsk:
		return []key.Binding{k.Up, k.Confirm, k.Back}
	case modeSummary:
		return []key.Binding{key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "close"))}
	}
	switch m.tab {
	case tabItems:
		return []key.Binding{k.Up, k.Down, k.Mark, k.Update, k.Check, k.CheckAll, k.Delete, k.Disable, k.Refresh, k.Tab, k.Quit}
	case tabProjects:
		return []key.Binding{k.Up, k.Down, k.Forget, k.Refresh, k.Tab, k.Quit}
	}
	return []key.Binding{k.Up, k.Down, k.Refresh, k.Tab, k.Quit}
}

func (m *tuiModel) window(n, sel, h int) (int, int) {
	start := min(max(sel-h/2, 0), max(n-h, 0))
	return start, min(start+h, n)
}

func (m *tuiModel) pointer(on bool) string {
	if on {
		return m.st.selected.Render("\u258c ")
	}
	return "  "
}

func (m *tuiModel) paneOf(w int, lines []string) string {
	h := m.bodyHeight() - 2
	for i, l := range lines {
		lines[i] = ansi.Truncate(l, max(w-4, 1), "\u2026")
	}
	return m.st.pane.Width(w).Height(m.bodyHeight()).Render(strings.Join(lines[:min(len(lines), h)], "\n"))
}

func (m *tuiModel) loadingLine(what string) []string {
	return []string{m.spin.View() + " " + m.st.faint.Render(what)}
}

func (m *tuiModel) itemList() string {
	w := m.listWidth()
	if m.loading && len(m.items) == 0 {
		return m.paneOf(w, m.loadingLine("Reading gap.toml, locks, and stores"))
	}
	if len(m.items) == 0 {
		return m.paneOf(w, []string{m.st.faint.Render("Nothing installed. gap add owner/repo adds a utility.")})
	}
	idW := 4
	for _, r := range m.items {
		idW = max(idW, len(r.ID))
	}
	wide := w >= 64
	idW = min(idW, 24)
	if !wide {
		idW = min(idW, max(w-34, 6))
	}
	head := fmt.Sprintf("    %-*s  ", idW, "id")
	if wide {
		head += fmt.Sprintf("%-6s  %-7s  %-8s  ", "source", "scope", "commit")
	} else {
		head += fmt.Sprintf("%-7s  ", "scope")
	}
	lines := []string{m.st.faint.Render(head + "state")}
	start, end := m.window(len(m.items), m.sel[tabItems], m.bodyHeight()-3)
	for i := start; i < end; i++ {
		r := m.items[i]
		mark := "  "
		if m.marked[r.ID] {
			mark = m.st.warn.Render("\u25cf ")
		}
		id := fmt.Sprintf("%-*s", idW, ansi.Truncate(r.ID, idW, "\u2026"))
		if r.Disabled {
			id = m.st.faint.Render(id)
		}
		cols := fmt.Sprintf("%-7s  ", r.Scope)
		if wide {
			cols = fmt.Sprintf("%-6s  %-7s  %-8s  ", r.Source, r.Scope, r.Commit)
		}
		line := m.pointer(i == m.sel[tabItems]) + mark + id + "  " + cols + m.badges(r)
		lines = append(lines, line)
	}
	return m.paneOf(w, lines)
}

func (m *tuiModel) badges(r itemRow) string {
	var parts []string
	switch {
	case r.Disabled:
		parts = append(parts, m.st.faint.Render("disabled"))
	case m.checking:
		parts = append(parts, m.st.faint.Render("checking"))
	case r.Store == storeOK:
		parts = append(parts, m.st.ok.Render("ok"))
	default:
		parts = append(parts, m.st.bad.Render(r.Store))
	}
	if r.Drift > 0 {
		parts = append(parts, m.st.warn.Render(fmt.Sprintf("drift=%d", r.Drift)))
	}
	if r.New > 0 {
		parts = append(parts, m.st.warn.Render(fmt.Sprintf("new=%d", r.New)))
	}
	if r.Held {
		parts = append(parts, m.st.faint.Render("held"))
	}
	if !r.user() {
		parts = append(parts, m.st.faint.Render("locked"))
	}
	return strings.Join(parts, " ")
}

func (m *tuiModel) itemDetail() string {
	w := m.width - m.listWidth()
	r, ok := m.item()
	if !ok {
		return m.paneOf(w, nil)
	}
	field := func(k, v string) string { return m.st.faint.Render(fmt.Sprintf("%-7s", k)) + " " + v }
	lines := []string{
		m.st.selected.Render(r.ID) + " " + m.badges(r),
		field("repo", r.Repo),
		field("ref", r.Ref),
		field("commit", r.Commit),
		field("source", r.Source+", "+r.Scope),
		field("hooks", r.Hooks),
	}
	if len(r.Skills) > 0 {
		lines = append(lines, "", m.st.faint.Render("skills"))
		for _, s := range r.Skills {
			tone := m.st.ok
			if s[1] != "drift=0" {
				tone = m.st.warn
			}
			lines = append(lines, "  "+s[0]+" "+tone.Render(s[1]))
		}
	}
	lines = append(lines, "")
	if r.user() {
		lines = append(lines, m.st.faint.Render("u updates and installs it. d removes it, or every marked utility."))
	} else {
		lines = append(lines, m.st.faint.Render("Distro component: locked, it cannot be removed."), m.st.faint.Render("u moves its pin in gap.lock. x disables or enables it."))
	}
	return m.paneOf(w, lines)
}

func (m *tuiModel) statusView() string {
	if m.statusLoading && m.status == "" {
		return m.paneOf(m.width, m.loadingLine("Running gap status"))
	}
	if m.statusErr != nil {
		return m.paneOf(m.width, []string{m.st.bad.Render(m.statusErr.Error())})
	}
	head := m.st.faint.Render("gap status")
	if m.statusLoading {
		head += " " + m.spin.View()
	}
	return m.st.pane.Width(m.width).Height(m.bodyHeight()).Render(head + "\n" + m.vp.View())
}

func (m *tuiModel) shortRoot(root string) string {
	if rel, err := filepath.Rel(m.home, root); err == nil && !strings.HasPrefix(rel, "..") {
		return filepath.Join("~", rel)
	}
	return root
}

func (m *tuiModel) projectBadges(p projectView) string {
	switch {
	case p.Missing:
		return m.st.bad.Render("missing")
	case p.Err != nil:
		return m.st.bad.Render("error")
	case m.checking:
		return m.st.faint.Render("checking")
	}
	drift := 0
	for _, it := range p.Items {
		drift += it.Drift
	}
	parts := []string{m.st.faint.Render(p.Profile)}
	if drift > 0 {
		parts = append(parts, m.st.warn.Render(fmt.Sprintf("drift=%d", drift)))
	} else {
		parts = append(parts, m.st.ok.Render("ok"))
	}
	return strings.Join(parts, " ")
}

func (m *tuiModel) projectList() string {
	w := m.listWidth()
	if m.loading && len(m.projects) == 0 {
		return m.paneOf(w, m.loadingLine("Reading "+m.shortRoot(projectsPath(m.home))))
	}
	if len(m.projects) == 0 {
		return m.paneOf(w, []string{m.st.faint.Render("No projects yet. gap init, gap add --project, or gap sync in a repo records it.")})
	}
	var lines []string
	start, end := m.window(len(m.projects), m.sel[tabProjects], m.bodyHeight()-2)
	for i := start; i < end; i++ {
		p := m.projects[i]
		lines = append(lines, m.pointer(i == m.sel[tabProjects])+m.shortRoot(p.Root)+"  "+m.projectBadges(p))
	}
	return m.paneOf(w, lines)
}

func (m *tuiModel) projectDetail() string {
	w := m.width - m.listWidth()
	p, ok := m.project()
	if !ok {
		return m.paneOf(w, nil)
	}
	lines := []string{m.st.selected.Render(m.shortRoot(p.Root)) + " " + m.projectBadges(p)}
	if p.Added != "" {
		lines = append(lines, m.st.faint.Render("added  ")+p.Added)
	}
	switch {
	case m.checking && !p.Missing:
		lines = append(lines, "", m.loadingLine("Checking project-scope skills")[0])
		return m.paneOf(w, lines)
	case p.Missing:
		lines = append(lines, "", m.st.bad.Render("This folder no longer exists."), m.st.faint.Render("f forgets it."))
		return m.paneOf(w, lines)
	case p.Err != nil:
		lines = append(lines, "", m.st.bad.Render(firstLine(p.Err.Error())))
		return m.paneOf(w, lines)
	}
	lines = append(lines, m.st.faint.Render("profile ")+p.Profile, "")
	if len(p.Items) == 0 {
		lines = append(lines, m.st.faint.Render("No project-scope skills installed here."))
	}
	for _, it := range p.Items {
		lines = append(lines, it.ID)
		for _, s := range it.Skills {
			tone := m.st.ok
			if s[1] != "drift=0" {
				tone = m.st.warn
			}
			lines = append(lines, "  "+s[0]+" "+tone.Render(s[1]))
		}
	}
	lines = append(lines, "", m.st.faint.Render("gap sync inside the repo installs and repairs its copies. f forgets it."))
	return m.paneOf(w, lines)
}

func (m *tuiModel) wrapLines(lines []string) []string {
	var out []string
	for _, l := range lines {
		out = append(out, strings.Split(lipgloss.Wrap(l, max(m.width-4, 1), ""), "\n")...)
	}
	return out
}

func (m *tuiModel) boxed(out []string) string {
	if h := m.bodyHeight() - 2; len(out) > h {
		keep := max(h-2, 0)
		out = append([]string{out[0], m.st.faint.Render("\u2026")}, out[len(out)-keep:]...)[:h]
	}
	box := m.st.pane.Render(strings.Join(out, "\n"))
	return lipgloss.Place(m.width, m.bodyHeight(), lipgloss.Center, lipgloss.Center, box)
}

func (m *tuiModel) centered(lines []string) string {
	return m.boxed(m.wrapLines(lines))
}

func (m *tuiModel) confirmParts() (head, foot []string, shown int) {
	head = m.wrapLines([]string{m.st.selected.Render(m.confirmTitle), ""})
	var tail []string
	if len(m.confirmIDs) > 0 {
		tail = append(tail, "")
	}
	tail = append(append(tail, m.confirmNote...), "", m.st.faint.Render("enter confirms, esc goes back"))
	foot = m.wrapLines(tail)
	n := len(m.confirmIDs)
	if room := m.bodyHeight() - 2 - len(head) - len(foot); n > room {
		return head, foot, max(room-1, 0)
	}
	return head, foot, n
}

func (m *tuiModel) confirmView() string {
	head, foot, shown := m.confirmParts()
	n := len(m.confirmIDs)
	top := max(min(m.confirmTop, n-shown), 0)
	w := max(m.width-4, 1)
	lines := head
	for _, id := range m.confirmIDs[top : top+shown] {
		lines = append(lines, ansi.Truncate("  "+id, w, "\u2026"))
	}
	if shown < n {
		lines = append(lines, m.st.faint.Render(ansi.Truncate(fmt.Sprintf("  \u2026 and %d more (j/k scrolls)", n-shown), w, "\u2026")))
	}
	return m.boxed(append(lines, foot...))
}

func (m *tuiModel) askView() string {
	label := "the run"
	for _, j := range m.jobs {
		if j.state == jobRunning {
			label = j.label
		}
	}
	options := []string{"Keep running", "Stop: cancel " + label + " and skip the rest"}
	lines := []string{label + " is still running.", ""}
	for i, o := range options {
		lines = append(lines, m.pointer(i == m.askChoice)+o)
	}
	lines = append(lines, "", m.st.faint.Render("enter chooses, esc goes back"))
	return m.centered(lines)
}

func (m *tuiModel) jobMark(j jobView) string {
	switch j.state {
	case jobRunning:
		return m.spin.View()
	case jobOK:
		return m.st.ok.Render("\u2713")
	case jobFailed:
		return m.st.bad.Render("\u2717")
	}
	return m.st.faint.Render("\u00b7")
}

func (m *tuiModel) finished() int {
	n := 0
	for _, j := range m.jobs {
		if j.state == jobOK || j.state == jobFailed {
			n++
		}
	}
	return n
}

func (m *tuiModel) runView() string {
	if m.stacked() {
		title := ""
		if m.jobSel < len(m.jobs) {
			j := m.jobs[m.jobSel]
			title = m.jobMark(j) + " " + m.st.selected.Render(j.label) + m.st.faint.Render(fmt.Sprintf("  %d/%d", m.jobSel+1, len(m.jobs)))
		}
		inner := max(m.width-4, 1)
		return m.st.pane.Width(m.width).Height(m.bodyHeight()).Render(ansi.Truncate(title, inner, "\u2026") + "\n" + m.vp.View() + "\n" + m.meter(inner))
	}
	w := m.listWidth()
	var lines []string
	start, end := m.window(len(m.jobs), m.jobSel, m.bodyHeight()-3)
	for i := start; i < end; i++ {
		lines = append(lines, m.pointer(i == m.jobSel)+m.jobMark(m.jobs[i])+" "+m.jobs[i].label)
	}
	lines = append(lines, m.meter(w-4))
	left := m.paneOf(w, lines)
	title := ""
	if m.jobSel < len(m.jobs) {
		title = m.st.selected.Render(m.jobs[m.jobSel].label)
	}
	right := m.st.pane.Width(m.width - w).Height(m.bodyHeight()).Render(ansi.Truncate(title, max(m.width-w-4, 1), "\u2026") + "\n" + m.vp.View())
	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

func (m *tuiModel) meter(width int) string {
	pct := 1.0
	if len(m.jobs) > 0 {
		pct = float64(m.finished()) / float64(len(m.jobs))
	}
	cells := max(width-6, 4)
	filled := int(math.Round(float64(cells) * pct))
	return m.st.accent.Render(strings.Repeat("\u2501", filled)) + m.st.faint.Render(strings.Repeat("\u2591", cells-filled)+fmt.Sprintf(" %3.0f%%", pct*100))
}

func lastLine(lines []string) string {
	for i := len(lines) - 1; i >= 0; i-- {
		if s := strings.TrimSpace(ansi.Strip(lines[i])); s != "" {
			return s
		}
	}
	return ""
}

func (m *tuiModel) summaryView() string {
	t := table.New().Border(lipgloss.RoundedBorder()).BorderStyle(m.st.border).Headers("Step", "Result", "Detail").
		Wrap(false).Width(m.width).FitContent().
		StyleFunc(func(row, _ int) lipgloss.Style {
			if row == table.HeaderRow {
				return lipgloss.NewStyle().Bold(true).Padding(0, 1)
			}
			return lipgloss.NewStyle().Padding(0, 1)
		})
	ok, bad := 0, 0
	stepW, resW := len("Step"), len("Result")
	var rows [][3]string
	for _, j := range m.jobs {
		res, detail := m.st.faint.Render("not run"), ""
		switch {
		case j.state == jobOK:
			ok++
			res = m.st.ok.Render("ok")
		case j.state == jobFailed && j.cancelled:
			bad++
			res = m.st.bad.Render("cancelled")
		case j.state == jobFailed:
			bad++
			res = m.st.bad.Render(fmt.Sprintf("failed (exit %d)", j.code))
			detail = lastLine(j.lines)
		}
		stepW, resW = max(stepW, ansi.StringWidth(j.label)), max(resW, ansi.StringWidth(res))
		rows = append(rows, [3]string{j.label, res, detail})
	}
	detailW := max(m.width-10-stepW-resW, 1)
	for _, r := range rows {
		t.Row(r[0], r[1], ansi.Truncate(r[2], detailW, "\u2026"))
	}
	if h := m.bodyHeight() - 2; len(rows)+4 > h {
		t.Height(h)
	}
	line := fmt.Sprintf("%d ok, %d failed, %d not run.", ok, bad, len(m.jobs)-ok-bad)
	if bad > 0 {
		line = m.st.bad.Render(line)
	}
	return lipgloss.NewStyle().Height(m.bodyHeight()).Render(lipgloss.JoinVertical(lipgloss.Left, t.String(), " "+line, m.st.faint.Render(" enter goes back and refreshes")))
}
