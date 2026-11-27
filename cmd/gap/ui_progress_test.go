package main

import (
	"strings"
	"testing"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

func updateProgress(m progressModel, msg tea.Msg) (progressModel, tea.Cmd) {
	next, cmd := m.Update(msg)
	return next.(progressModel), cmd
}

func TestProgressBarModel(t *testing.T) {
	m := newProgressModel(darkPalette, true, "sync", 0, 120, 24)
	now := m.start
	m.clock = func() time.Time { return now }

	// Pre-seeded planned steps.
	m, _ = updateProgress(m, progressPlanMsg{steps: []progressStep{
		{Label: "fetch sources"},
		{Label: "policy"},
		{Label: "skills: check drift"},
	}})
	if m.total != 3 || len(m.entries) != 3 {
		t.Fatalf("plan did not seed 3 steps: total=%d len=%d", m.total, len(m.entries))
	}

	// Start first step and report sub-item progress + orthogonal detail.
	m, _ = updateProgress(m, progressStepMsg{label: "fetch sources"})
	now = now.Add(3 * time.Second)
	m, _ = updateProgress(m, progressItemMsg{done: 4, total: 15, name: "do-skills"})
	view := ansi.Strip(m.View().Content)
	for _, want := range []string{
		"fetch sources",
		"4/15 do-skills",
		"· policy",
		"· skills: check drift",
		"~12s step left",
		"\u2501",
		"\u2591",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("active view lacks %q:\n%s", want, view)
		}
	}

	// detail() stays orthogonal to subDone/subTotal so ETA and item count remain intact.
	m, _ = updateProgress(m, progressDetailMsg{text: "Fetching do-skills"})
	view = ansi.Strip(m.View().Content)
	for _, want := range []string{"4/15 do-skills", "Fetching do-skills", "~12s step left"} {
		if !strings.Contains(view, want) {
			t.Fatalf("detail() clobbered item or ETA (missing %q):\n%s", want, view)
		}
	}
	m, _ = updateProgress(m, progressDetailMsg{text: ""})

	// Spinner tick advances spinner frame while preserving clock and ETA.
	beforeSpin := m.spin.View()
	m, _ = updateProgress(m, spinner.TickMsg{ID: m.spin.ID(), Time: now.Add(100 * time.Millisecond)})
	if m.spin.View() == beforeSpin {
		t.Fatalf("spinner tick did not advance frame (still %q)", beforeSpin)
	}

	// Moving to next step marks previous step done with duration and clears item/detail.
	now = now.Add(1500 * time.Millisecond)
	m, _ = updateProgress(m, progressStepMsg{label: "policy"})
	view = ansi.Strip(m.View().Content)
	if !strings.Contains(view, "  \u2713 fetch sources  4.5s") {
		t.Fatalf("completed step missing from checklist:\n%s", view)
	}
	if strings.Contains(view, "4/15 do-skills") {
		t.Fatalf("item outlived its step:\n%s", view)
	}
	if !strings.Contains(view, " 33%  1/3") || !strings.Contains(view, "~9s left") {
		t.Fatalf("step-level progress or ETA missing:\n%s", view)
	}

	// Dynamic step appends cleanly and grows total if needed.
	now = now.Add(2 * time.Second)
	m, _ = updateProgress(m, progressStepMsg{label: "extra", hue: darkPalette.group("", "echo")})
	if m.total != 4 || len(m.entries) != 4 {
		t.Fatalf("dynamic step did not append: total=%d len=%d", m.total, len(m.entries))
	}

	// Negative add drops trailing pending steps.
	m, _ = updateProgress(m, progressPlanMsg{steps: []progressStep{
		{Label: "policy"},
		{Label: "skills: upgrade unedited"},
		{Label: "skills: install missing"},
		{Label: "skills: check drift"},
	}})
	m, _ = updateProgress(m, progressStepMsg{label: "policy"})
	m, _ = updateProgress(m, progressAddMsg{delta: -3})
	if m.total != 1 || len(m.entries) != 1 {
		t.Fatalf("negative add did not prune pending steps: total=%d len=%d", m.total, len(m.entries))
	}

	// Narrow width truncation never exceeds width.
	m, _ = updateProgress(m, tea.WindowSizeMsg{Width: 32, Height: 10})
	for _, line := range strings.Split(m.View().Content, "\n") {
		if w := ansi.StringWidth(line); w > 32 {
			t.Fatalf("narrow line exceeded 32 cols (%d): %q", w, ansi.Strip(line))
		}
	}

	// Suspended renders empty; stopping renders completed steps and returns tea.Quit.
	m, _ = updateProgress(m, progressSuspendMsg{suspended: true})
	if got := m.View().Content; got != "" {
		t.Fatalf("suspended view = %q, want empty", got)
	}
	m, _ = updateProgress(m, progressSuspendMsg{suspended: false})
	now = now.Add(time.Second)
	m, _ = updateProgress(m, progressStepMsg{label: ""})
	m, cmd := updateProgress(m, progressStopMsg{})
	if cmd == nil {
		t.Fatal("stop message did not return tea.Quit")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatalf("stop cmd returned %T, want tea.QuitMsg", cmd())
	}
	final := ansi.Strip(m.View().Content)
	if !strings.Contains(final, "\u2713 policy") || strings.Contains(final, "\u2501") {
		t.Fatalf("final view should keep completed steps without bar:\n%s", final)
	}

	// Nil progressBar safety.
	var nilBar *progressBar
	nilBar.plan([]progressStep{{Label: "x"}})
	nilBar.step("x", nil)
	nilBar.mark(false)
	nilBar.item(1, 2, "a")
	nilBar.detail("working")
	nilBar.add(1)
	if nilBar.println("hi") {
		t.Fatal("nilBar.println returned true")
	}
	ran := false
	nilBar.suspended(func() { ran = true })
	if !ran {
		t.Fatal("nilBar.suspended did not run callback")
	}
	nilBar.stop()
}

func TestProgressBarUnresolvedStepShowsWarning(t *testing.T) {
	m := newProgressModel(darkPalette, true, "sync", 0, 120, 24)
	now := m.start
	m.clock = func() time.Time { return now }
	m, _ = updateProgress(m, progressPlanMsg{steps: []progressStep{
		{Label: "skills: install missing"},
		{Label: "skills: check drift"},
	}})
	m, _ = updateProgress(m, progressStepMsg{label: "skills: install missing"})
	now = now.Add(time.Second)
	m, _ = updateProgress(m, progressStepMsg{label: "skills: check drift"})
	now = now.Add(time.Second)
	m, _ = updateProgress(m, progressMarkMsg{ok: false})
	m, _ = updateProgress(m, progressStepMsg{label: ""})
	if m.done != 2 {
		t.Fatalf("done = %d, want 2", m.done)
	}
	m, _ = updateProgress(m, progressStopMsg{})
	final := ansi.Strip(m.View().Content)
	if !strings.Contains(final, "  \u2713 skills: install missing") {
		t.Fatalf("clean step missing checkmark:\n%s", final)
	}
	if !strings.Contains(final, "  ! skills: check drift") {
		t.Fatalf("unresolved step missing warning mark:\n%s", final)
	}
	if strings.Contains(final, "\u2713 skills: check drift") {
		t.Fatalf("unresolved step must not render checkmark:\n%s", final)
	}
}

func TestProgressBarETADoesNotInflateOnIdleTicks(t *testing.T) {
	m := newProgressModel(darkPalette, true, "sync", 0, 120, 24)
	now := m.start
	m.clock = func() time.Time { return now }
	m, _ = updateProgress(m, progressPlanMsg{steps: []progressStep{
		{Label: "skills: check drift"},
		{Label: "policy"},
		{Label: "final"},
	}})
	m, _ = updateProgress(m, progressStepMsg{label: "skills: check drift"})
	m, _ = updateProgress(m, progressItemMsg{done: 1, total: 10, name: "s1"})
	// Item 1 completes in 1s -> item 2 starts at t=1s with 9 items left (~9s step left).
	now = now.Add(time.Second)
	m, _ = updateProgress(m, progressItemMsg{done: 2, total: 10, name: "s2"})
	eta1, stepScope := m.remaining(now)
	if !stepScope || eta1 != 9*time.Second {
		t.Fatalf("initial item ETA = (%v, %v), want (9s, true)", eta1, stepScope)
	}
	// Halfway through item 2 (not overdue): ETA must decrease, not inflate to 14s.
	now = now.Add(500 * time.Millisecond)
	eta2, _ := m.remaining(now)
	if eta2 >= eta1 {
		t.Fatalf("idle tick inflated item ETA from %v to %v", eta1, eta2)
	}
	// Even when item 2 is overdue, ETA stays bounded by remaining items (8s).
	now = now.Add(3 * time.Second)
	eta3, _ := m.remaining(now)
	if eta3 != 8*time.Second {
		t.Fatalf("overdue item ETA = %v, want 8s", eta3)
	}

	// Step-level ETA likewise counts down on idle ticks instead of inflating.
	now = now.Add(500 * time.Millisecond)
	m, _ = updateProgress(m, progressStepMsg{label: "policy"})
	stepETA1, isStep := m.remaining(now)
	if isStep || stepETA1 <= 0 {
		t.Fatalf("initial command ETA = (%v, %v), want (>0, false)", stepETA1, isStep)
	}
	now = now.Add(time.Second)
	stepETA2, _ := m.remaining(now)
	if stepETA2 >= stepETA1 {
		t.Fatalf("idle tick inflated command ETA from %v to %v", stepETA1, stepETA2)
	}
}

func TestProgressBarUpdateMonotonicPct(t *testing.T) {
	// Mirrors updateUtilities for 1 utility with 2 skills: 1 + 4*1 = 5 steps.
	m := newProgressModel(darkPalette, true, "update", 5, 120, 24)
	msgs := []tea.Msg{
		progressStepMsg{label: "check util"},
		progressStepMsg{label: "fetch sources"},
		progressItemMsg{done: 1, total: 1, name: "util"},
		progressStepMsg{label: "util"},
		progressStepMsg{label: "util: install skills"},
		progressItemMsg{done: 1, total: 2, name: "s1"},
		progressItemMsg{done: 2, total: 2, name: "s2"},
		progressStepMsg{label: "util: check drift"},
		progressItemMsg{done: 1, total: 2, name: "s1"},
		progressItemMsg{done: 2, total: 2, name: "s2"},
		progressStepMsg{label: ""},
	}
	prev := -1.0
	for i, msg := range msgs {
		m, _ = updateProgress(m, msg)
		got := m.pct()
		if got < prev {
			t.Fatalf("step %d (%T) moved progress backward: %.3f -> %.3f", i, msg, prev, got)
		}
		prev = got
	}
	if prev != 1.0 {
		t.Fatalf("final pct = %.3f, want 1.0", prev)
	}
}

func TestProgressBarOffWhenPiped(t *testing.T) {
	if p := startProgress("sync", 3); p != nil {
		p.stop()
		t.Fatal("startProgress started a program without a terminal")
	}
}
