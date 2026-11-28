package main

import (
	"fmt"
	"image/color"
	"math"
	"os"
	"strings"
	"sync"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/term"
)

// progressStep is a planned phase in a sync or update run.
type progressStep struct {
	Label string
	Hue   color.Color
}

type progressState int

const (
	stepPending progressState = iota
	stepActive
	stepDone
	stepWarn
)

type progressEntry struct {
	label      string
	hue        color.Color
	state      progressState
	unresolved bool
	started    time.Time
	took       time.Duration
}

type progressPlanMsg struct{ steps []progressStep }
type progressStepMsg struct {
	label string
	hue   color.Color
}
type progressMarkMsg struct{ ok bool }
type progressItemMsg struct {
	done  int
	total int
	name  string
}
type progressDetailMsg struct{ text string }
type progressAddMsg struct{ delta int }
type progressSuspendMsg struct{ suspended bool }
type progressStopMsg struct{}

// progressModel is an inline Bubble Tea v2 model rendering a live step checklist,
// active item/detail status, gradient progress bar, elapsed time, and ETA.
type progressModel struct {
	pal       palette
	plain     bool
	title     string
	width     int
	height    int
	total     int
	done      int
	entries   []progressEntry
	active    int
	subDone   int
	subTotal  int
	itemName  string
	detail    string
	start     time.Time
	doneAt    time.Time
	stepAt    time.Time
	subDoneAt time.Time
	clock     func() time.Time
	spin      spinner.Model
	suspended bool
	quitting  bool
}

func newProgressModel(pal palette, plain bool, title string, total, width, height int) progressModel {
	clock := time.Now
	now := clock()
	sp := spinner.New(spinner.WithSpinner(spinner.MiniDot))
	return progressModel{
		pal:       pal,
		plain:     plain,
		title:     title,
		width:     width,
		height:    height,
		total:     total,
		active:    -1,
		start:     now,
		doneAt:    now,
		stepAt:    now,
		subDoneAt: now,
		clock:     clock,
		spin:      sp,
	}
}

func (m progressModel) Init() tea.Cmd {
	return m.spin.Tick
}

func (m progressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		if msg.Width > 0 {
			m.width = msg.Width
		}
		if msg.Height > 0 {
			m.height = msg.Height
		}
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spin, cmd = m.spin.Update(msg)
		return m, cmd

	case progressPlanMsg:
		now := m.clock()
		entries := make([]progressEntry, len(msg.steps))
		for i, s := range msg.steps {
			entries[i] = progressEntry{label: s.Label, hue: s.Hue, state: stepPending}
		}
		m.entries = entries
		m.total = len(entries)
		m.done = 0
		m.active = -1
		m.doneAt = now
		m.stepAt = now
		m.subDoneAt = now
		m.subDone, m.subTotal, m.itemName, m.detail = 0, 0, "", ""
		return m, nil

	case progressStepMsg:
		now := m.clock()
		if m.active >= 0 && m.active < len(m.entries) {
			if m.entries[m.active].unresolved {
				m.entries[m.active].state = stepWarn
			} else {
				m.entries[m.active].state = stepDone
			}
			m.entries[m.active].took = max(now.Sub(m.entries[m.active].started), 0)
			m.done++
			m.doneAt = now
			if m.total < m.done {
				m.total = m.done
			}
		}
		m.active = -1
		m.subDone, m.subTotal, m.itemName, m.detail = 0, 0, "", ""
		m.stepAt = now
		m.subDoneAt = now
		if msg.label == "" {
			return m, nil
		}
		for i := range m.entries {
			if m.entries[i].state == stepPending && m.entries[i].label == msg.label {
				m.entries[i].state = stepActive
				m.entries[i].started = now
				if msg.hue != nil {
					m.entries[i].hue = msg.hue
				}
				m.active = i
				return m, nil
			}
		}
		m.entries = append(m.entries, progressEntry{
			label:   msg.label,
			hue:     msg.hue,
			state:   stepActive,
			started: now,
		})
		m.active = len(m.entries) - 1
		if m.total < len(m.entries) {
			m.total = len(m.entries)
		}
		return m, nil

	case progressMarkMsg:
		if m.active >= 0 && m.active < len(m.entries) && !msg.ok {
			m.entries[m.active].unresolved = true
		}
		return m, nil

	case progressItemMsg:
		if msg.done != m.subDone || msg.total != m.subTotal {
			m.subDoneAt = m.clock()
		}
		m.subDone = msg.done
		m.subTotal = msg.total
		m.itemName = msg.name
		return m, nil

	case progressDetailMsg:
		m.detail = msg.text
		return m, nil

	case progressAddMsg:
		m.total = max(m.total+msg.delta, m.done)
		if msg.delta < 0 {
			for len(m.entries) > m.total && m.entries[len(m.entries)-1].state == stepPending {
				m.entries = m.entries[:len(m.entries)-1]
			}
		}
		return m, nil

	case progressSuspendMsg:
		m.suspended = msg.suspended
		if m.suspended {
			return m, tea.Quit
		}
		return m, nil

	case progressStopMsg:
		m.quitting = true
		return m, tea.Quit
	}
	return m, nil
}

func (m progressModel) View() tea.View {
	v := tea.NewView(m.render())
	v.AltScreen = false
	return v
}

func (m progressModel) render() string {
	if m.suspended {
		return ""
	}
	w := m.width
	if w <= 0 {
		w = 80
	}
	h := m.height
	if h <= 0 {
		h = 24
	}
	if m.quitting {
		var doneLines []string
		for _, e := range m.entries {
			if e.state == stepDone || e.state == stepWarn {
				doneLines = append(doneLines, ansi.Truncate(m.doneLine(e), w, "\u2026"))
			}
		}
		if len(doneLines) == 0 {
			return ""
		}
		return strings.Join(doneLines, "\n") + "\n"
	}

	var lines []string
	if len(m.entries) > 0 {
		maxList := max(h-3, 1)
		start, end := 0, len(m.entries)
		if len(m.entries) > maxList {
			anchor := m.active
			if anchor < 0 {
				anchor = len(m.entries) - 1
			}
			start = max(anchor-maxList/2, 0)
			end = start + maxList
			if end > len(m.entries) {
				end = len(m.entries)
				start = max(end-maxList, 0)
			}
		}
		for i := start; i < end; i++ {
			lines = append(lines, ansi.Truncate(m.entryLine(m.entries[i]), w, "\u2026"))
		}
	}
	lines = append(lines, m.barLine(w))
	return strings.Join(lines, "\n")
}

func (m progressModel) style(c color.Color) lipgloss.Style {
	if m.plain || c == nil {
		return lipgloss.NewStyle()
	}
	return lipgloss.NewStyle().Foreground(c)
}

func (m progressModel) doneLine(e progressEntry) string {
	hue := e.hue
	if hue == nil {
		hue = m.pal.Text
	}
	mark, markColor := "\u2713", m.pal.OK
	if e.state == stepWarn {
		mark, markColor = "!", m.pal.Warn
	}
	return "  " + m.style(markColor).Render(mark) + " " + m.style(hue).Render(e.label) + "  " + m.style(m.pal.Dim).Render(e.took.Round(100*time.Millisecond).String())
}

func (m progressModel) entryLine(e progressEntry) string {
	switch e.state {
	case stepDone, stepWarn:
		return m.doneLine(e)
	case stepActive:
		hue := e.hue
		if hue == nil {
			hue = m.pal.Text
		}
		line := "  " + m.style(m.pal.Accent).Render(m.spin.View()) + " " + m.style(hue).Bold(!m.plain).Render(e.label)
		if item := m.itemText(); item != "" {
			line += "  " + m.style(m.pal.Dim).Render(item)
		}
		if m.detail != "" {
			line += "  " + m.style(m.pal.Dim).Render("\u00b7 "+m.detail)
		}
		return line
	default:
		return "  " + m.style(m.pal.Dim).Render("\u00b7 "+e.label)
	}
}

func (m progressModel) itemText() string {
	if m.subTotal > 0 {
		if m.itemName != "" {
			return fmt.Sprintf("%d/%d %s", m.subDone, m.subTotal, m.itemName)
		}
		return fmt.Sprintf("%d/%d", m.subDone, m.subTotal)
	}
	return m.itemName
}

func (m progressModel) remaining(now time.Time) (time.Duration, bool) {
	if m.subTotal > 1 && m.subDone > 1 && m.subDone <= m.subTotal {
		completedDur := max(m.subDoneAt.Sub(m.stepAt), 0)
		if completedDur >= time.Second {
			done := float64(m.subDone - 1)
			avg := float64(completedDur) / done
			inFlight := float64(max(now.Sub(m.subDoneAt), 0))
			after := float64(m.subTotal - m.subDone)
			return time.Duration(max(avg-inFlight, 0) + avg*after), true
		}
		return 0, true
	}
	if m.subTotal <= 1 && m.total > 0 && m.done > 0 && m.done < m.total {
		completedDur := max(m.doneAt.Sub(m.start), 0)
		if completedDur >= time.Second {
			avg := float64(completedDur) / float64(m.done)
			if m.active >= 0 {
				inFlight := float64(max(now.Sub(m.doneAt), 0))
				after := float64(max(m.total-m.done-1, 0))
				return time.Duration(max(avg-inFlight, 0) + avg*after), false
			}
			return time.Duration(avg * float64(m.total-m.done)), false
		}
	}
	return 0, false
}

func formatETA(d time.Duration, stepScope bool) string {
	s := int(d.Round(time.Second).Seconds())
	if s < 1 {
		return ""
	}
	suffix := " left"
	if stepScope {
		suffix = " step left"
	}
	if s < 60 {
		return fmt.Sprintf("~%ds%s", s, suffix)
	}
	return fmt.Sprintf("~%dm%02ds%s", s/60, s%60, suffix)
}

func (m progressModel) pct() float64 {
	if m.total > 0 {
		done := float64(m.done)
		if m.subTotal > 0 && m.done < m.total {
			done += float64(min(max(m.subDone-1, 0), m.subTotal)) / float64(m.subTotal)
		}
		return min(done/float64(m.total), 1.0)
	}
	if m.done > 0 {
		return 1.0
	}
	return 0.0
}

func (m progressModel) barLine(width int) string {
	pct := m.pct()
	now := m.clock()
	sinceStart := max(now.Sub(m.start), 0)
	elapsed := sinceStart.Round(time.Second)
	frame := m.spin.View()
	stat := fmt.Sprintf(" %3.0f%%  %d/%d  %02d:%02d", pct*100, m.done, m.total, int(elapsed.Minutes()), int(elapsed.Seconds())%60)
	if eta, stepScope := m.remaining(now); eta > 0 {
		if label := formatETA(eta, stepScope); label != "" {
			stat += "  " + label
		}
	}
	stat += " "

	activeLabel := ""
	var activeHue color.Color
	if m.active >= 0 && m.active < len(m.entries) {
		activeLabel = m.entries[m.active].label
		activeHue = m.entries[m.active].hue
	}
	item := m.itemText()
	tail := frame + " " + m.title
	if activeLabel != "" {
		tail += " " + activeLabel
	}
	if item != "" {
		tail += "  " + item
	}
	if m.detail != "" {
		tail += "  " + m.detail
	}
	cells := max(min(width/2, width-ansi.StringWidth(stat)-ansi.StringWidth(tail)-2), 8)
	filled := int(math.Round(float64(cells) * pct))

	var b strings.Builder
	b.WriteString(" ")
	if filled > 0 {
		hues := lipgloss.Blend1D(filled, m.pal.Deep, m.pal.Accent, m.pal.OK)
		for i := range filled {
			b.WriteString(m.style(hues[i]).Render("\u2501"))
		}
	}
	b.WriteString(m.style(m.pal.Border).Render(strings.Repeat("\u2591", cells-filled)))
	b.WriteString(m.style(m.pal.Dim).Render(stat))
	b.WriteString(m.style(m.pal.Accent).Render(frame) + " ")
	b.WriteString(m.style(m.pal.Accent).Bold(!m.plain).Render(m.title))
	if activeLabel != "" {
		if activeHue == nil {
			activeHue = m.pal.Text
		}
		b.WriteString(" " + m.style(activeHue).Render(activeLabel))
	}
	if item != "" {
		b.WriteString("  " + m.style(m.pal.Dim).Render(item))
	}
	if m.detail != "" {
		b.WriteString("  " + m.style(m.pal.Dim).Render(m.detail))
	}
	return ansi.Truncate(b.String(), width, "\u2026")
}

// progressBar drives an inline Bubble Tea v2 progress program on stderr while sync or update runs.
// All methods are nil-safe so non-terminal runs skip it without branching.
type progressBar struct {
	mu        sync.Mutex
	prog      *tea.Program
	doneCh    chan struct{}
	quit      chan struct{}
	pal       palette
	plain     bool
	stopped   bool
	paused    bool
	pending   []tea.Msg
	model     progressModel
	suspMu    sync.Mutex
	suspDepth int
}

var activeProgress *progressBar

// startWorking draws a one-line spinner on stderr until the returned stop runs; it draws nothing off a terminal.
func startWorking(label string) func() {
	if activeProgress != nil || !term.IsTerminal(os.Stdout.Fd()) || !term.IsTerminal(os.Stderr.Fd()) || os.Getenv("TERM") == "dumb" {
		return func() {}
	}
	frames := spinner.MiniDot.Frames
	quit, done := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(done)
		tick := time.NewTicker(spinner.MiniDot.FPS)
		defer tick.Stop()
		for i := 0; ; i++ {
			fmt.Fprintf(os.Stderr, "\r%s %s", frames[i%len(frames)], label)
			select {
			case <-quit:
				fmt.Fprint(os.Stderr, "\r\x1b[K")
				return
			case <-tick.C:
			}
		}
	}()
	var once sync.Once
	return func() {
		once.Do(func() {
			close(quit)
			<-done
		})
	}
}

// startProgress returns nil unless stdout and stderr are both a terminal that supports live rendering.
func startProgress(title string, total int) *progressBar {
	if activeProgress != nil || !term.IsTerminal(os.Stdout.Fd()) || !term.IsTerminal(os.Stderr.Fd()) || os.Getenv("TERM") == "dumb" {
		return nil
	}
	w, h, err := term.GetSize(os.Stderr.Fd())
	if err != nil || h < 4 || w < 20 {
		return nil
	}
	pal := newReport(os.Stdout).colors()
	plain := plainEnv()
	p := &progressBar{
		quit:  make(chan struct{}),
		pal:   pal,
		plain: plain,
	}
	p.run(newProgressModel(pal, plain, title, total, w, h))
	activeProgress = p
	restoreOnSignal(p)
	return p
}

func (p *progressBar) run(m progressModel) {
	prog := tea.NewProgram(m,
		tea.WithInput(nil),
		tea.WithOutput(os.Stderr),
		tea.WithoutSignalHandler(),
	)
	done := make(chan struct{})
	p.prog, p.doneCh = prog, done
	go func() {
		defer close(done)
		final, _ := prog.Run()
		if fm, ok := final.(progressModel); ok {
			p.mu.Lock()
			p.model = fm
			p.mu.Unlock()
		}
	}()
}

func (p *progressBar) send(msg tea.Msg) {
	if p == nil {
		return
	}
	p.mu.Lock()
	if p.paused && !p.stopped {
		p.pending = append(p.pending, msg)
		p.mu.Unlock()
		return
	}
	prog, stopped := p.prog, p.stopped
	p.mu.Unlock()
	if prog == nil || stopped {
		return
	}
	prog.Send(msg)
}

func (p *progressBar) plan(steps []progressStep) {
	if p == nil {
		return
	}
	cp := make([]progressStep, len(steps))
	copy(cp, steps)
	p.send(progressPlanMsg{steps: cp})
}

func (p *progressBar) add(n int) {
	if p == nil {
		return
	}
	p.send(progressAddMsg{delta: n})
}

func (p *progressBar) groupHue(c component) color.Color {
	if p == nil {
		return nil
	}
	return p.pal.group(c.Color, c.ID)
}

// step marks the previous step done and starts the named step. An empty label closes the active step.
func (p *progressBar) step(label string, hue color.Color) {
	if p == nil {
		return
	}
	p.send(progressStepMsg{label: label, hue: hue})
}

// mark records whether the active step completed cleanly. A false mark renders
// the finished step with a warning indicator instead of a checkmark.
func (p *progressBar) mark(ok bool) {
	if p == nil {
		return
	}
	p.send(progressMarkMsg{ok: ok})
}

func (p *progressBar) item(done, total int, name string) {
	if p == nil {
		return
	}
	p.send(progressItemMsg{done: done, total: total, name: name})
}

func (p *progressBar) detail(text string) {
	if p == nil {
		return
	}
	p.send(progressDetailMsg{text: text})
}

// println prints a persistent line above the live inline progress view.
// Returns false when no active unsuspended program handled the line.
func (p *progressBar) println(line string) bool {
	if p == nil {
		return false
	}
	p.suspMu.Lock()
	suspended := p.suspDepth > 0
	p.suspMu.Unlock()
	if suspended {
		return false
	}
	p.mu.Lock()
	prog, stopped := p.prog, p.stopped
	p.mu.Unlock()
	if prog == nil || stopped {
		return false
	}
	prog.Println(strings.TrimRight(line, "\n"))
	return true
}

// suspended temporarily releases the terminal so interactive prompts or subprocesses
// can read stdin and write output without racing the Bubble Tea inline renderer.
func (p *progressBar) suspended(fn func()) {
	if p == nil {
		fn()
		return
	}
	p.suspMu.Lock()
	p.mu.Lock()
	prog, stopped, done := p.prog, p.stopped, p.doneCh
	p.mu.Unlock()
	if prog == nil || stopped {
		p.suspMu.Unlock()
		fn()
		return
	}
	p.suspDepth++
	if p.suspDepth == 1 {
		p.mu.Lock()
		p.paused = true
		p.mu.Unlock()
		prog.Send(progressSuspendMsg{suspended: true})
		<-done
	}
	p.suspMu.Unlock()
	defer p.resume()
	fn()
}

func (p *progressBar) resume() {
	p.suspMu.Lock()
	defer p.suspMu.Unlock()
	p.suspDepth--
	if p.suspDepth > 0 {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.paused = false
	pending := p.pending
	p.pending = nil
	if p.stopped {
		return
	}
	m := p.model
	m.suspended = false
	p.run(m)
	next := p.prog
	go func() {
		for _, msg := range pending {
			next.Send(msg)
		}
	}()
}

func (p *progressBar) stop() {
	if p == nil {
		return
	}
	p.mu.Lock()
	if p.stopped {
		p.mu.Unlock()
		return
	}
	p.stopped = true
	if p.quit != nil {
		close(p.quit)
	}
	if activeProgress == p {
		activeProgress = nil
	}
	prog, done := p.prog, p.doneCh
	p.mu.Unlock()
	if prog != nil {
		prog.Send(progressStopMsg{})
		<-done
	}
}
