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
// flowing normally, the way apt draws its progress line. Each finished step
// leaves a line above the bar, and a ticker keeps the spinner and clock moving.
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
	detail   string
	subDone  int
	subTotal int
	hue      color.Color
	start    time.Time
	stepAt   time.Time
	clock    func() time.Time
	stopped  bool
	lastRows int
	quit     chan struct{}
}

var activeProgress *progressBar

var spinnerFrames = []string{"\u280b", "\u2819", "\u2839", "\u2838", "\u283c", "\u2834", "\u2826", "\u2827", "\u2807", "\u280f"}

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
		go p.tick(p.quit)
	}
	return p
}

func newProgressBar(w io.Writer, size func() (int, int), pal palette, plain bool, title string, total int) *progressBar {
	p := &progressBar{w: w, size: size, pal: pal, plain: plain, title: title, total: total, clock: time.Now}
	if _, h := size(); h < 4 {
		return nil
	}
	p.start = p.clock()
	p.stepAt = p.start
	fmt.Fprint(w, "\n\x1b[1A")
	p.draw()
	return p
}

func (p *progressBar) tick(quit <-chan struct{}) {
	t := time.NewTicker(100 * time.Millisecond)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			p.draw()
		case <-quit:
			return
		}
	}
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
	now := p.clock()
	if p.label != "" && !p.stopped {
		p.done = min(p.done+1, p.total)
		io.WriteString(p.w, p.doneLine(now.Sub(p.stepAt)))
	}
	p.label, p.hue, p.detail, p.subDone, p.subTotal, p.stepAt = label, hue, "", 0, 0, now
	p.mu.Unlock()
	p.draw()
}

func (p *progressBar) item(done, total int, name string) {
	if p == nil {
		return
	}
	p.mu.Lock()
	p.subDone, p.subTotal = done, total
	if total > 0 {
		p.detail = fmt.Sprintf("%d/%d %s", done, total, name)
	} else {
		p.detail = name
	}
	p.mu.Unlock()
	p.draw()
}

func (p *progressBar) style(c color.Color) lipgloss.Style {
	if p.plain || c == nil {
		return lipgloss.NewStyle()
	}
	return lipgloss.NewStyle().Foreground(c)
}

func (p *progressBar) doneLine(took time.Duration) string {
	hue := p.hue
	if hue == nil {
		hue = p.pal.Text
	}
	return "  " + p.style(p.pal.OK).Render("\u2713") + " " + p.style(hue).Render(p.label) + "  " + p.style(p.pal.Dim).Render(took.Round(100*time.Millisecond).String()) + "\n"
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

func (p *progressBar) remaining(now time.Time) time.Duration {
	if p.subTotal > 1 && p.subDone > 1 && p.subDone <= p.subTotal {
		stepElapsed := max(now.Sub(p.stepAt), 0)
		if stepElapsed >= time.Second {
			done := float64(p.subDone - 1)
			left := float64(p.subTotal - (p.subDone - 1))
			return time.Duration(float64(stepElapsed) * left / done)
		}
		return 0
	}
	if p.subTotal == 0 && p.total > 0 && p.done > 0 && p.done < p.total {
		elapsed := max(now.Sub(p.start), 0)
		if elapsed >= time.Second {
			return time.Duration(float64(elapsed) * float64(p.total-p.done) / float64(p.done))
		}
	}
	return 0
}

func formatETA(d time.Duration) string {
	s := int(d.Round(time.Second).Seconds())
	if s < 1 {
		return ""
	}
	if s < 60 {
		return fmt.Sprintf("~%ds left", s)
	}
	return fmt.Sprintf("~%dm%02ds left", s/60, s%60)
}

func (p *progressBar) line(width int) string {
	pct := 1.0
	if p.total > 0 {
		done := float64(p.done)
		if p.subTotal > 0 && p.done < p.total {
			done += float64(min(max(p.subDone-1, 0), p.subTotal)) / float64(p.subTotal)
		}
		pct = min(done/float64(p.total), 1.0)
	}
	now := p.clock()
	sinceStart := max(now.Sub(p.start), 0)
	elapsed := sinceStart.Round(time.Second)
	frame := spinnerFrames[int(sinceStart/(100*time.Millisecond))%len(spinnerFrames)]
	stat := fmt.Sprintf(" %3.0f%%  %d/%d  %02d:%02d", pct*100, p.done, p.total, int(elapsed.Minutes()), int(elapsed.Seconds())%60)
	if eta := formatETA(p.remaining(now)); eta != "" {
		stat += "  " + eta
	}
	stat += " "
	label := " " + frame + " " + p.title
	if p.label != "" {
		label += " " + p.label
	}
	if p.detail != "" {
		label += "  " + p.detail
	}
	cells := max(min(width/2, width-ansi.StringWidth(stat)-ansi.StringWidth(label)-2), 8)
	filled := int(math.Round(float64(cells) * pct))
	var b strings.Builder
	b.WriteString(" ")
	if filled > 0 {
		hues := lipgloss.Blend1D(filled, p.pal.Deep, p.pal.Accent, p.pal.OK)
		for i := range filled {
			b.WriteString(p.style(hues[i]).Render("\u2501"))
		}
	}
	b.WriteString(p.style(p.pal.Border).Render(strings.Repeat("\u2591", cells-filled)))
	b.WriteString(p.style(p.pal.Dim).Render(stat))
	b.WriteString(p.style(p.pal.Accent).Render(frame) + " ")
	b.WriteString(p.style(p.pal.Accent).Bold(!p.plain).Render(p.title))
	if p.label != "" {
		hue := p.hue
		if hue == nil {
			hue = p.pal.Text
		}
		b.WriteString(" " + p.style(hue).Render(p.label))
	}
	if p.detail != "" {
		b.WriteString("  " + p.style(p.pal.Dim).Render(p.detail))
	}
	return ansi.Truncate(b.String(), width, "\u2026")
}
