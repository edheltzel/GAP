package main

import (
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

func (m *tuiModel) View() tea.View {
	v := tea.NewView(m.render())
	v.AltScreen = true
	v.WindowTitle = "gap"
	return v
}

func (m *tuiModel) bodyHeight() int {
	switch {
	case m.running():
		return max(m.height-m.headHeight()-2, 6)
	case m.mode == modeSummary:
		return max(m.height-m.headHeight()-2, 6)
	}
	return max(m.height-m.headHeight()-4, 6)
}

func (m *tuiModel) bannerOn() bool { return m.height >= 18 && m.width >= 60 }

func (m *tuiModel) headHeight() int {
	if m.bannerOn() {
		return bannerRows
	}
	return 1
}

func (m *tuiModel) runListWidth() int {
	if m.stacked() {
		return m.width
	}
	w := 24
	for _, j := range m.jobs {
		w = max(w, lipgloss.Width(j.title())+24)
	}
	return min(w, max(m.width/2, 24))
}

func (j jobView) title() string {
	if j.name != "" {
		return j.name
	}
	return j.label
}

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
			w := m.runListWidth()
			if m.mode == modeSummary {
				w = m.listWidth()
			}
			m.vp.SetWidth(max(m.width-w-4, 1))
			m.vp.SetHeight(inner)
		}
		if m.jobSel < len(m.jobs) {
			j := m.jobs[m.jobSel]
			lines := j.lines
			if len(lines) == 0 {
				lines = []string{m.st.faint.Render(m.emptyOutput(j))}
			}
			m.vp.SetContentLines(lines)
			if m.running() {
				m.vp.GotoBottom()
			}
		}
	case m.mode == modeDetail:
		m.vp.SetWidth(max(m.width-4, 1))
		m.vp.SetHeight(inner + 1)
		m.vp.SetContentLines(m.wrapLines(m.detailLines()))
	case m.tab == tabStatus:
		m.vp.SetWidth(max(m.width-4, 1))
		m.vp.SetHeight(inner + 1)
		m.vp.SetContentLines(strings.Split(m.status, "\n"))
	}
}

func (m *tuiModel) emptyOutput(j jobView) string {
	switch {
	case j.state == jobPending && m.mode == modeSummary:
		return "Not run."
	case j.state == jobPending:
		return "Waiting to run."
	case j.state == jobRunning:
		return "No output yet."
	}
	return "No output."
}

func (m *tuiModel) stepTitle(inner int, lead string, j jobView, suffix string) string {
	return fitLabel(inner, lead, m.st.selected.Render(j.label), suffix)
}

func fitLabel(inner int, lead, label, suffix string) string {
	inner = max(inner, 1)
	room := inner - ansi.StringWidth(lead) - ansi.StringWidth(suffix)
	if room < 1 {
		return ansi.Truncate(lead+suffix, inner, "\u2026")
	}
	return lead + ansi.Truncate(label, room, "\u2026") + suffix
}

func (m *tuiModel) header() string {
	name := m.st.title.Render(" gap ")
	title := name + " " + m.st.faint.Render("components, utilities, projects")
	stat := fmt.Sprintf("%d items  %d marked  %d projects", len(m.items), len(m.marked), len(m.projects))
	if m.bannerOn() {
		art := banner(m.st.banner, m.st.faint.Render("global agent policy"), m.frame)
		if pad := m.width - lipgloss.Width(art) - lipgloss.Width(stat); pad >= 1 {
			return lipgloss.JoinHorizontal(lipgloss.Top, art, strings.Repeat(" ", pad), m.st.faint.Render(stat))
		}
		return art
	}
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

func (m *tuiModel) runHeader() string {
	tagline := m.st.faint.Render("global agent policy")
	title := m.st.title.Render(" gap ") + " " + tagline
	if m.bannerOn() {
		title = banner(m.st.banner, tagline, m.frame)
	}
	stat := m.st.faint.Render(fmt.Sprintf("%d/%d  %s", m.finished(), len(m.jobs), clock(m.now.Sub(m.start))))
	pad := max(m.width-lipgloss.Width(title)-lipgloss.Width(stat), 1)
	return lipgloss.JoinHorizontal(lipgloss.Top, title, strings.Repeat(" ", pad), stat)
}

func clock(d time.Duration) string {
	d = max(d.Round(time.Second), 0)
	return fmt.Sprintf("%02d:%02d", int(d.Minutes()), int(d.Seconds())%60)
}

func (m *tuiModel) render() string {
	m.help.SetWidth(max(m.width-1, 1))
	fit := func(s string) string { return ansi.Truncate(s, m.width, "\u2026") }
	if m.running() || m.mode == modeSummary {
		body := m.summaryView()
		switch m.mode {
		case modeRun:
			body = m.runView()
		case modeAsk:
			body = m.askView()
		}
		foot := m.help.ShortHelpView(m.helpKeys())
		if m.note != "" {
			foot = m.note + "  " + foot
		}
		return lipgloss.JoinVertical(lipgloss.Left, m.runHeader(), body, m.meter(m.width), fit(" "+foot))
	}
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
	case modeForm:
		body = m.formView()
	case modeSearch:
		body = m.searchView()
	case modeConfirm:
		body = m.confirmView()
	case modeRun:
		body = m.runView()
	case modeAsk:
		body = m.askView()
	case modeSummary:
		body = m.summaryView()
	case modeDetail:
		body = m.st.pane.Width(m.width).Height(m.bodyHeight()).Render(m.vp.View())
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
	if err := errors.Join(m.loadErr, m.projErr); err != nil && m.note == "" {
		note = " " + m.st.bad.Render(firstLine(err.Error()))
	}
	if m.chord != "" {
		note = " " + m.st.accent.Render(m.chord) + m.st.faint.Render("  gg top, gt/gT next or previous view")
	}
	return lipgloss.JoinVertical(lipgloss.Left, m.header(), fit(" "+strings.Join(tabs, "   ")), body, fit(note), fit(" "+m.help.ShortHelpView(m.helpKeys())))
}

func firstLine(s string) string {
	line, _, _ := strings.Cut(s, "\n")
	return line
}

func (m *tuiModel) helpKeys() []key.Binding {
	k := m.keys
	switch m.mode {
	case modeSearch:
		return []key.Binding{key.NewBinding(key.WithKeys("up"), key.WithHelp("\u2191/\u2193", "pick")), key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "install")), key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "close"))}
	case modeForm:
		return []key.Binding{key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab/enter", "next")), k.Back}
	case modeConfirm:
		return []key.Binding{k.Confirm, k.Back}
	case modeRun:
		return []key.Binding{k.Up, k.Down, k.Follow, key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "stop"))}
	case modeAsk:
		return []key.Binding{k.Up, k.Confirm, k.Back}
	case modeSummary:
		return []key.Binding{k.Up, k.Down, key.NewBinding(key.WithKeys("pgup", "pgdown"), key.WithHelp("pgup/pgdn", "output")), key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "close"))}
	case modeDetail:
		return []key.Binding{k.Up, k.Down, k.Top, k.Bottom, k.Back}
	}
	switch m.tab {
	case tabItems:
		return []key.Binding{k.Up, k.Down, k.Open, k.Find, k.Add, k.Edit, k.Editor, k.Update, k.UpdateAll, k.Sync, k.Mark, k.Delete, k.Disable, k.Check, k.CheckAll, k.Refresh, k.Tab, k.Quit}
	case tabProjects:
		return []key.Binding{k.Up, k.Down, k.Open, k.Add, key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "open its gap.toml")), k.Forget, k.Sync, k.Refresh, k.Tab, k.Quit}
	}
	return []key.Binding{k.Up, k.Down, k.Top, k.Bottom, k.Sync, k.Refresh, k.Tab, k.Quit}
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
	start, end := m.listWindow(tabItems)
	for i := start; i < end; i++ {
		r := m.items[i]
		mark := "  "
		if m.marked[r.ID] {
			mark = m.st.warn.Render("\u25cf ")
		}
		id := fmt.Sprintf("%-*s", idW, ansi.Truncate(r.ID, idW, "\u2026"))
		if r.Disabled {
			id = m.st.faint.Render(id)
		} else {
			id = lipgloss.NewStyle().Foreground(m.groupColor(r.ID)).Render(id)
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
	if r.Inactive {
		parts = append(parts, m.st.faint.Render("off here"))
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

func (m *tuiModel) detailLines() []string {
	if m.tab == tabProjects {
		return m.projectDetailLines()
	}
	return m.itemDetailLines()
}

func (m *tuiModel) itemDetail() string {
	return m.paneOf(m.width-m.listWidth(), m.itemDetailLines())
}

func (m *tuiModel) itemDetailLines() []string {
	r, ok := m.item()
	if !ok {
		return nil
	}
	field := func(k, v string) string { return m.st.faint.Render(fmt.Sprintf("%-7s", k)) + " " + v }
	lines := []string{
		lipgloss.NewStyle().Bold(true).Foreground(m.groupColor(r.ID)).Render(r.ID) + " " + m.badges(r),
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
	return lines
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
	for _, home := range []string{m.home, m.homeReal} {
		if rel, err := filepath.Rel(home, root); err == nil && !strings.HasPrefix(rel, "..") {
			return filepath.Join("~", rel)
		}
	}
	return root
}

func (m *tuiModel) projectBadges(p projectView) string {
	switch {
	case p.Missing:
		return m.st.bad.Render("missing")
	case p.Err != nil:
		return m.st.bad.Render("error")
	case m.projChecking:
		return m.st.faint.Render("checking")
	}
	drift := 0
	for _, it := range p.Items {
		drift += it.Drift
	}
	parts := []string{m.st.faint.Render(p.Profile)}
	if drift > 0 {
		parts = append(parts, m.st.warn.Render(fmt.Sprintf("drift=%d", drift)))
	}
	switch {
	case p.unchecked():
		parts = append(parts, m.st.warn.Render("unchecked"))
	case drift == 0:
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
	start, end := m.listWindow(tabProjects)
	for i := start; i < end; i++ {
		p := m.projects[i]
		lines = append(lines, m.pointer(i == m.sel[tabProjects])+m.shortRoot(p.Root)+"  "+m.projectBadges(p))
	}
	return m.paneOf(w, lines)
}

func (m *tuiModel) projectDetail() string {
	return m.paneOf(m.width-m.listWidth(), m.projectDetailLines())
}

func (m *tuiModel) projectDetailLines() []string {
	p, ok := m.project()
	if !ok {
		return nil
	}
	lines := []string{m.st.selected.Render(m.shortRoot(p.Root)) + " " + m.projectBadges(p)}
	if p.Added != "" {
		lines = append(lines, m.st.faint.Render("added  ")+p.Added)
	}
	switch {
	case m.projChecking && !p.Missing:
		return append(lines, "", m.loadingLine("Checking project-scope skills")[0])
	case p.Missing:
		return append(lines, "", m.st.bad.Render("This folder no longer exists."), m.st.faint.Render("f forgets it."))
	case p.Err != nil:
		return append(lines, "", m.st.bad.Render(firstLine(p.Err.Error())))
	}
	lines = append(lines, m.st.faint.Render("profile ")+p.Profile, "")
	if len(p.Items) == 0 {
		lines = append(lines, m.st.faint.Render("No project-scope skills installed here."))
	}
	for _, it := range p.Items {
		lines = append(lines, it.ID)
		if it.Store != storeOK {
			lines = append(lines, "  "+m.st.warn.Render("store "+it.Store+": drift not checked"))
		}
		for _, s := range it.Skills {
			tone := m.st.ok
			if s[1] != "drift=0" {
				tone = m.st.warn
			}
			lines = append(lines, "  "+s[0]+" "+tone.Render(s[1]))
		}
	}
	return append(lines, "", m.st.faint.Render("gap sync inside the repo installs and repairs its copies. f forgets it."))
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

func (m *tuiModel) confirmParts() (head, rows, foot []string, shown int) {
	for _, id := range m.confirmIDs {
		rows = append(rows, m.wrapLines([]string{"  " + id})...)
	}
	title := m.st.selected.Render(m.confirmTitle)
	hint := m.st.faint.Render("enter confirms, esc goes back")
	for _, compact := range []bool{false, true} {
		if compact {
			head, foot = m.wrapLines([]string{title}), m.wrapLines([]string{hint})
		} else {
			head = m.wrapLines([]string{title, ""})
			var tail []string
			if len(rows) > 0 {
				tail = append(tail, "")
			}
			foot = m.wrapLines(append(append(tail, m.confirmNote...), "", hint))
		}
		room := m.bodyHeight() - 2 - len(head) - len(foot)
		if len(rows) == 0 || len(rows) <= room {
			return head, rows, foot, len(rows)
		}
		if room >= 2 || compact {
			return head, rows, foot, max(room-1, 0)
		}
	}
	return head, rows, foot, 0
}

func (m *tuiModel) confirmView() string {
	head, rows, foot, shown := m.confirmParts()
	top := max(min(m.confirmTop, len(rows)-shown), 0)
	w := max(m.width-4, 1)
	lines := append(slices.Clone(head), rows[top:top+shown]...)
	switch {
	case len(rows) > 0 && shown == 0:
		lines = append(lines, m.st.bad.Render(ansi.Truncate("Too small to list them; enlarge the terminal.", w, "\u2026")))
	case shown < len(rows):
		lines = append(lines, m.st.faint.Render(ansi.Truncate(fmt.Sprintf("  \u2026 and %d more (j/k scrolls)", len(rows)-shown), w, "\u2026")))
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

func (m *tuiModel) jobMark(j jobView) (string, string) {
	switch j.state {
	case jobRunning:
		return m.spin.View(), m.st.faint.Render(clock(m.now.Sub(j.began)))
	case jobOK:
		return m.st.ok.Render("\u2713"), m.st.faint.Render(clock(j.ended.Sub(j.began)))
	case jobFailed:
		if j.cancelled {
			return m.st.bad.Render("\u2717"), m.st.bad.Render("cancelled")
		}
		return m.st.bad.Render("\u2717"), m.st.bad.Render(fmt.Sprintf("exit %d", j.code))
	}
	return m.st.faint.Render("\u00b7"), ""
}

func (m *tuiModel) jobStyle(j jobView) lipgloss.Style {
	if j.color == nil {
		return lipgloss.NewStyle().Foreground(m.st.pal.Accent)
	}
	return lipgloss.NewStyle().Foreground(j.color)
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
	if m.jobSel >= len(m.jobs) {
		return m.paneOf(m.width, nil)
	}
	cur := m.jobs[m.jobSel]
	title := m.jobStyle(cur).Bold(true).Render(" " + cur.label + " ")
	if m.stacked() {
		mark, note := m.jobMark(cur)
		inner := max(m.width-4, 1)
		head := m.stepTitle(inner, mark+" ", cur, m.st.faint.Render(fmt.Sprintf("  %d/%d ", m.jobSel+1, len(m.jobs)))+note)
		return m.st.pane.Width(m.width).Height(m.bodyHeight()).Render(head + "\n" + m.vp.View())
	}
	w := m.runListWidth()
	var lines []string
	start, end := m.window(len(m.jobs), m.jobSel, m.bodyHeight()-2)
	for i := start; i < end; i++ {
		j := m.jobs[i]
		mark, note := m.jobMark(j)
		line := m.pointer(i == m.jobSel) + mark + " " + m.jobStyle(j).Render(j.title())
		if note != "" {
			line += strings.Repeat(" ", max(w-4-lipgloss.Width(line)-lipgloss.Width(note), 1)) + note
		}
		lines = append(lines, line)
	}
	left := m.paneOf(w, lines)
	right := m.st.pane.Width(m.width - w).Height(m.bodyHeight()).Render(ansi.Truncate(title, max(m.width-w-4, 1), "\u2026") + "\n" + m.vp.View())
	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

func (m *tuiModel) meter(width int) string {
	pct := 1.0
	if len(m.jobs) > 0 {
		pct = float64(m.finished()) / float64(len(m.jobs))
	}
	cells := max(width-8, 4)
	filled := int(math.Round(float64(cells) * pct))
	hues := lipgloss.Blend1D(max(filled, 1), m.st.pal.Deep, m.st.pal.Accent, m.st.pal.OK)
	var b strings.Builder
	b.WriteString(" ")
	for i := range cells {
		if i < filled {
			b.WriteString(lipgloss.NewStyle().Foreground(hues[i]).Render("\u2501"))
			continue
		}
		b.WriteString(m.st.border.Render("\u2591"))
	}
	b.WriteString(m.st.faint.Render(fmt.Sprintf(" %3.0f%%", pct*100)))
	return b.String()
}

func lastLine(lines []string) string {
	for i := len(lines) - 1; i >= 0; i-- {
		if s := strings.TrimSpace(ansi.Strip(lines[i])); s != "" {
			return s
		}
	}
	return ""
}

func (m *tuiModel) result(j jobView) string {
	switch {
	case j.state == jobOK:
		return m.st.ok.Render("ok")
	case j.state == jobFailed && j.cancelled:
		return m.st.bad.Render("cancelled")
	case j.state == jobFailed:
		return m.st.bad.Render(fmt.Sprintf("failed (exit %d)", j.code))
	}
	return m.st.faint.Render("not run")
}

func (m *tuiModel) shortResult(j jobView) string {
	if j.state == jobFailed && !j.cancelled {
		return m.st.bad.Render(fmt.Sprintf("exit %d", j.code))
	}
	return m.result(j)
}

func (m *tuiModel) tally() string {
	ok, bad := 0, 0
	for _, j := range m.jobs {
		switch j.state {
		case jobOK:
			ok++
		case jobFailed:
			bad++
		}
	}
	line := fmt.Sprintf("%d ok, %d failed, %d not run.", ok, bad, len(m.jobs)-ok-bad)
	if bad > 0 {
		return m.st.bad.Render(line)
	}
	return line
}

func (m *tuiModel) summaryView() string {
	if m.stacked() {
		inner := max(m.width-4, 1)
		title := ""
		if m.jobSel < len(m.jobs) {
			j := m.jobs[m.jobSel]
			step := m.st.faint.Render(fmt.Sprintf("  %d/%d", m.jobSel+1, len(m.jobs)))
			suffix := " " + m.result(j) + step
			if inner-ansi.StringWidth(suffix) < 4 {
				suffix = " " + m.shortResult(j) + step
			}
			title = m.stepTitle(inner, "", j, suffix)
		}
		return m.st.pane.Width(m.width).Height(m.bodyHeight()).Render(title + "\n" + m.vp.View() + "\n" + ansi.Truncate(m.tally(), inner, "\u2026"))
	}
	w := m.listWidth()
	var lines []string
	start, end := m.window(len(m.jobs), m.jobSel, m.bodyHeight()-4)
	for i := start; i < end; i++ {
		j := m.jobs[i]
		mark, _ := m.jobMark(j)
		lines = append(lines, fitLabel(w-4, m.pointer(i == m.jobSel)+mark+" ", m.jobStyle(j).Render(j.label), "  "+m.result(j)))
	}
	lines = append(lines, "", m.tally())
	title := ""
	if m.jobSel < len(m.jobs) {
		j := m.jobs[m.jobSel]
		title = m.stepTitle(m.width-w-4, "", j, " "+m.result(j))
	}
	right := m.st.pane.Width(m.width - w).Height(m.bodyHeight()).Render(title + "\n" + m.vp.View())
	return lipgloss.JoinHorizontal(lipgloss.Top, m.paneOf(w, lines), right)
}
