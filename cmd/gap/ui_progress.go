package main

import (
	"fmt"
	"image/color"
	"io"
	"math"
	"os"
	"strings"
	"sync"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/term"
)

// progressBar pins a bar to the terminal's last row while sync or update runs.
// A scroll region above it keeps every other line, hook output, and prompt
// flowing normally, the way apt draws its progress line.
type progressBar struct {
	mu       sync.Mutex
	w        io.Writer
	size     func() (int, int)
	pal      palette
	plain    bool
	title    string
	total    int
	done     int
	label    string
	hue      color.Color
	start    time.Time
	clock    func() time.Time
	stopped  bool
	lastRows int
	quit     chan struct{}
}

var activeProgress *progressBar

// startProgress returns nil unless stdout and stderr are both a terminal that can move the cursor.
func startProgress(title string, total int) *progressBar {
	if activeProgress != nil || !isTerminal(os.Stdout) || !isTerminal(os.Stderr) || os.Getenv("TERM") == "dumb" {
		return nil
	}
	f := os.Stderr
	p := newProgressBar(f, func() (int, int) {
		w, h, err := term.GetSize(f.Fd())
		if err != nil {
			return 0, 0
		}
		return w, h
	}, newReport(os.Stdout).colors(), plainEnv(), title, total)
	if p != nil {
		activeProgress = p
		p.quit = make(chan struct{})
		restoreOnSignal(p)
	}
	return p
}

func newProgressBar(w io.Writer, size func() (int, int), pal palette, plain bool, title string, total int) *progressBar {
	p := &progressBar{w: w, size: size, pal: pal, plain: plain, title: title, total: total, clock: time.Now}
	if _, h := size(); h < 4 {
		return nil
	}
	p.start = p.clock()
	fmt.Fprint(w, "\n\x1b[1A")
	p.draw()
	return p
}

func (p *progressBar) add(n int) {
	if p == nil {
		return
	}
	p.mu.Lock()
	p.total += n
	p.mu.Unlock()
	p.draw()
}

func (p *progressBar) groupHue(c component) color.Color {
	if p == nil {
		return nil
	}
	return p.pal.group(c.Color, c.ID)
}

// step marks the previous step done and names the one starting now.
func (p *progressBar) step(label string, hue color.Color) {
	if p == nil {
		return
	}
	p.mu.Lock()
	if p.label != "" {
		p.done = min(p.done+1, p.total)
	}
	p.label, p.hue = label, hue
	p.mu.Unlock()
	p.draw()
}

func (p *progressBar) stop() {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.stopped {
		return
	}
	p.stopped = true
	if p.quit != nil {
		close(p.quit)
	}
	if activeProgress == p {
		activeProgress = nil
	}
	_, h := p.size()
	fmt.Fprintf(p.w, "\x1b7\x1b[%d;1H\x1b[2K\x1b[r\x1b8", max(h, p.lastRows))
}

func (p *progressBar) draw() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.stopped {
		return
	}
	w, h := p.size()
	if h < 4 || w < 20 {
		return
	}
	var b strings.Builder
	b.WriteString("\x1b7")
	if h != p.lastRows {
		if p.lastRows > h {
			fmt.Fprintf(&b, "\x1b[%d;1H\x1b[2K", p.lastRows)
		}
		fmt.Fprintf(&b, "\x1b[1;%dr\x1b8\x1b7", h-1)
		p.lastRows = h
	}
	fmt.Fprintf(&b, "\x1b[%d;1H\x1b[2K", h)
	b.WriteString(p.line(w))
	b.WriteString("\x1b8")
	io.WriteString(p.w, b.String())
}

func (p *progressBar) line(width int) string {
	style := func(c color.Color) lipgloss.Style {
		if p.plain || c == nil {
			return lipgloss.NewStyle()
		}
		return lipgloss.NewStyle().Foreground(c)
	}
	pct := 1.0
	if p.total > 0 {
		pct = float64(p.done) / float64(p.total)
	}
	elapsed := p.clock().Sub(p.start).Round(time.Second)
	stat := fmt.Sprintf(" %3.0f%%  %d/%d  %02d:%02d ", pct*100, p.done, p.total, int(elapsed.Minutes()), int(elapsed.Seconds())%60)
	label := " " + p.title
	if p.label != "" {
		label += " " + p.label
	}
	cells := max(min(width/2, width-ansi.StringWidth(stat)-ansi.StringWidth(label)-2), 8)
	filled := int(math.Round(float64(cells) * pct))
	var b strings.Builder
	b.WriteString(" ")
	if filled > 0 {
		hues := lipgloss.Blend1D(filled, p.pal.Deep, p.pal.Accent, p.pal.OK)
		for i := range filled {
			b.WriteString(style(hues[i]).Render("\u2501"))
		}
	}
	b.WriteString(style(p.pal.Border).Render(strings.Repeat("\u2591", cells-filled)))
	b.WriteString(style(p.pal.Dim).Render(stat))
	b.WriteString(style(p.pal.Accent).Bold(!p.plain).Render(p.title))
	if p.label != "" {
		hue := p.hue
		if hue == nil {
			hue = p.pal.Text
		}
		b.WriteString(" " + style(hue).Render(p.label))
	}
	return ansi.Truncate(b.String(), width, "\u2026")
}
