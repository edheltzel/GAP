package main

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

func TestTUIRenderFitsWidth(t *testing.T) {
	long := strings.Repeat("0123456789", 30)
	for _, w := range []int{40, 60, 80, 100, 140} {
		m, _ := tuiFixture(t)
		m.Update(tea.WindowSizeMsg{Width: w, Height: 24})
		widest := 0
		check := func(view string) {
			t.Helper()
			lines := strings.Split(m.render(), "\n")
			if len(lines) > 24 {
				t.Errorf("width %d %s: %d lines, want <= 24", w, view, len(lines))
			}
			for i, l := range lines {
				lw := ansi.StringWidth(l)
				widest = max(widest, lw)
				if lw > w {
					t.Errorf("width %d %s: line %d is %d cols: %q", w, view, i, lw, ansi.Strip(l))
				}
			}
		}
		check("items")
		press(m, "2")
		m.Update(statusTextMsg{text: long + "\nshort"})
		check("status")
		press(m, "3")
		check("projects")
		press(m, "1", "x")
		check("disable confirm")
		press(m, "esc", "j", "space", "space", "d")
		check("confirm")
		press(m, "enter")
		m.Update(jobStartedMsg{i: 0})
		m.Update(jobOutputMsg{i: 0, line: long})
		check("run")
		press(m, "esc")
		check("ask")
		press(m, "esc")
		m.Update(jobDoneMsg{i: 0})
		m.Update(jobStartedMsg{i: 1})
		m.Update(jobOutputMsg{i: 1, line: long})
		m.Update(jobDoneMsg{i: 1, code: 1})
		m.Update(jobsFinishedMsg{})
		check("summary")
		t.Logf("width %d: widest line %d", w, widest)
	}
}

func TestTUIConfirmListsEveryID(t *testing.T) {
	m := newTUIModel(t.TempDir(), t.TempDir(), t.TempDir(), darkPalette)
	f := &fakeRunner{}
	m.runner = f
	m.Update(tea.WindowSizeMsg{Width: 40, Height: 24})
	var rows []itemRow
	var ids []string
	for i := range 32 {
		id := fmt.Sprintf("util-%02d", i)
		ids = append(ids, id)
		rows = append(rows, itemRow{ID: id, Source: "user", Scope: "global", Commit: "1111111", Store: storeOK, Pinned: true})
	}
	m.Update(loadedMsg{items: rows})
	for range ids {
		press(m, "space")
	}
	if len(m.marked) != 32 {
		t.Fatalf("marked %d, want 32", len(m.marked))
	}
	press(m, "d")
	seen := map[string]bool{}
	check := func(step int) {
		t.Helper()
		raw := strings.Split(m.render(), "\n")
		if len(raw) > 24 {
			t.Errorf("step %d: %d lines, want <= 24", step, len(raw))
		}
		var text []string
		for i, l := range raw {
			if lw := ansi.StringWidth(l); lw > 40 {
				t.Errorf("step %d: line %d is %d cols: %q", step, i, lw, ansi.Strip(l))
			}
			inner := strings.Trim(ansi.Strip(l), " \u2502")
			text = append(text, inner)
			if slices.Contains(ids, inner) {
				seen[inner] = true
			}
		}
		joined := strings.Join(text, " ")
		for _, want := range []string{"Remove 32 utilities?", "~/.gap/backups/", "enter confirms, esc goes back", "more (j/k scrolls)"} {
			if !strings.Contains(joined, want) {
				t.Fatalf("step %d: %q missing:\n%s", step, want, strings.Join(text, "\n"))
			}
		}
	}
	check(0)
	for step := 1; step <= 40; step++ {
		press(m, "j")
		check(step)
	}
	for step := 41; step <= 45; step++ {
		press(m, "k")
		check(step)
	}
	if len(seen) != 32 {
		var missing []string
		for _, id := range ids {
			if !seen[id] {
				missing = append(missing, id)
			}
		}
		t.Fatalf("ids never shown: %v", missing)
	}
	press(m, "enter")
	if len(f.started) != 1 || len(f.started[0]) != 32 {
		t.Fatalf("started %d batches, want one of 32", len(f.started))
	}
	for i, job := range f.started[0] {
		if got := strings.Join(job.args, " "); got != "remove -- "+ids[i] {
			t.Fatalf("job %d = %q", i, got)
		}
	}
}

func TestTUIConfirmShortTerminalKeepsHint(t *testing.T) {
	for _, h := range []int{11, 12, 16} {
		m, _ := tuiFixture(t)
		press(m, "j", "space", "space", "d")
		m.Update(tea.WindowSizeMsg{Width: 40, Height: h})
		raw := strings.Split(m.render(), "\n")
		if len(raw) > h {
			t.Errorf("height %d: %d lines", h, len(raw))
		}
		text := ansi.Strip(strings.Join(raw, "\n"))
		for _, want := range []string{"Remove 2 utilities?", "enter confirms, esc goes back"} {
			if !strings.Contains(text, want) {
				t.Errorf("height %d: %q missing:\n%s", h, want, text)
			}
		}
	}
}

func confirmFixture(t *testing.T, w, h int, ids ...string) (*tuiModel, *fakeRunner) {
	t.Helper()
	m, f := tuiFixture(t)
	var rows []itemRow
	for _, id := range ids {
		rows = append(rows, itemRow{ID: id, Source: "user", Scope: "global", Commit: "1111111", Store: storeOK, Pinned: true})
	}
	m.Update(loadedMsg{items: rows})
	for range ids {
		press(m, "space")
	}
	press(m, "d")
	m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	return m, f
}

func TestTUIConfirmShowsLongIDsInFull(t *testing.T) {
	long := "u" + strings.Repeat("x", 88) + "z"
	m, f := confirmFixture(t, 40, 11, "ua", long)
	seen := map[string]bool{}
	for step := range 12 {
		raw := strings.Split(m.render(), "\n")
		if len(raw) > 11 {
			t.Fatalf("step %d: %d lines", step, len(raw))
		}
		text := ansi.Strip(strings.Join(raw, "\n"))
		if !strings.Contains(text, "Remove 2 utilities?") || !strings.Contains(text, "enter confirms, esc goes back") {
			t.Fatalf("step %d:\n%s", step, text)
		}
		for _, l := range raw {
			if ansi.StringWidth(l) > 40 {
				t.Fatalf("step %d: line too wide: %q", step, ansi.Strip(l))
			}
			seen[strings.Trim(ansi.Strip(l), " \u2502")] = true
		}
		press(m, "j")
	}
	var chunks []string
	for _, row := range m.wrapLines([]string{"  " + long}) {
		chunk := strings.TrimSpace(ansi.Strip(row))
		chunks = append(chunks, chunk)
		if !seen[chunk] {
			t.Fatalf("row %q of the long id was never shown; saw %v", chunk, seen)
		}
	}
	if strings.Join(chunks, "") != long {
		t.Fatalf("rows %q do not spell %s", chunks, long)
	}
	press(m, "enter")
	if len(f.started) != 1 {
		t.Fatal("a confirmation with visible targets did not start")
	}
}

func TestTUIConfirmBlockedWhenNoTargetFits(t *testing.T) {
	m, f := confirmFixture(t, 12, 11, "ua", "ub")
	press(m, "enter")
	if len(f.started) != 0 || m.mode != modeConfirm || !strings.Contains(m.note, "Enlarge the terminal") {
		t.Fatalf("started %v mode %v note %q\n%s", f.started, m.mode, m.note, screen(m))
	}
	m.Update(tea.WindowSizeMsg{Width: 40, Height: 11})
	press(m, "enter")
	if len(f.started) != 1 {
		t.Fatal("enter after enlarging did not start")
	}
}

func TestTUIRunSignalledChildIsCancelled(t *testing.T) {
	old := tuiExecutable
	t.Cleanup(func() { tuiExecutable = old })
	killer := filepath.Join(t.TempDir(), "gap")
	write(t, killer, "#!/bin/sh\nkill -TERM $$\n")
	if err := os.Chmod(killer, 0o755); err != nil {
		t.Fatal(err)
	}
	tuiExecutable = func() (string, error) { return killer, nil }
	done := doneOf(collect(t, &execRunner{}, []tuiJob{{label: "remove ua", args: []string{"remove", "ua"}}}, nil))
	if len(done) != 1 || !done[0].cancelled || done[0].code < 1 {
		t.Fatalf("done = %+v", done)
	}
	m, _ := tuiFixture(t)
	press(m, "j", "d", "enter")
	m.Update(jobStartedMsg{i: 0})
	m.Update(done[0])
	m.Update(jobsFinishedMsg{})
	s := screen(m)
	if !strings.Contains(s, "cancelled") || strings.Contains(s, "exit -1") || strings.Contains(s, "failed (exit") {
		t.Fatalf("summary:\n%s", s)
	}
}
