package main

import (
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

type fakeRunner struct {
	started   [][]tuiJob
	cancelled bool
}

func (f *fakeRunner) start(jobs []tuiJob, _ func(tea.Msg)) { f.started = append(f.started, jobs) }
func (f *fakeRunner) cancel()                              { f.cancelled = true }

func press(m *tuiModel, keys ...string) {
	for _, k := range keys {
		var msg tea.KeyPressMsg
		switch k {
		case "space":
			msg = tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}
		case "enter":
			msg = tea.KeyPressMsg{Code: tea.KeyEnter}
		case "esc":
			msg = tea.KeyPressMsg{Code: tea.KeyEscape}
		case "tab":
			msg = tea.KeyPressMsg{Code: tea.KeyTab}
		default:
			msg = tea.KeyPressMsg{Code: rune(k[0]), Text: k}
		}
		m.Update(msg)
	}
}

func screen(m *tuiModel) string { return ansi.Strip(m.render()) }

func tuiFixture(t *testing.T) (*tuiModel, *fakeRunner) {
	t.Helper()
	home := t.TempDir()
	m := newTUIModel(home, t.TempDir(), t.TempDir(), darkPalette)
	f := &fakeRunner{}
	m.runner = f
	m.Update(tea.WindowSizeMsg{Width: 140, Height: 30})
	m.Update(loadedMsg{
		items: []itemRow{
			{ID: "echo", Source: "distro", Scope: "global", Commit: "abc1234", Store: storeOK, Pinned: true},
			{ID: "ua", Source: "user", Scope: "global", Commit: "1111111", Store: storeOK, Pinned: true, Skills: [][]string{{"ua-skill", "drift=0"}}},
			{ID: "ub", Source: "user", Scope: "project", Commit: "2222222", Store: storeOK, Pinned: true, Drift: 1, New: 3},
		},
		projects: []projectView{
			{Root: filepath.Join(home, "gone"), Missing: true},
			{Root: filepath.Join(home, "here"), Profile: "project", Items: []projectItem{{ID: "ub", Skills: [][]string{{"ub-skill", "drift=1"}}, Drift: 1}}},
		},
	})
	return m, f
}

func TestTUIMassDeleteConfirmsEveryID(t *testing.T) {
	m, f := tuiFixture(t)
	press(m, "j", "space", "space")
	if !m.marked["ua"] || !m.marked["ub"] || len(m.marked) != 2 {
		t.Fatalf("marked = %v", m.marked)
	}
	press(m, "d")
	s := screen(m)
	if m.mode != modeConfirm || !strings.Contains(s, "Remove 2 utilities?") || !strings.Contains(s, "  ua") || !strings.Contains(s, "  ub") || !strings.Contains(s, "~/.gap/backups/") {
		t.Fatalf("confirm screen:\n%s", s)
	}
	if len(f.started) != 0 {
		t.Fatal("removal started before confirmation")
	}
	press(m, "enter")
	if len(f.started) != 1 || len(f.started[0]) != 2 || strings.Join(f.started[0][0].args, " ") != "remove ua" || strings.Join(f.started[0][1].args, " ") != "remove ub" {
		t.Fatalf("jobs = %+v", f.started)
	}
	if m.mode != modeRun || len(m.marked) != 0 {
		t.Fatalf("mode %v marked %v", m.mode, m.marked)
	}
}

func TestTUIConfirmEscapeChangesNothing(t *testing.T) {
	m, f := tuiFixture(t)
	press(m, "j", "d", "esc")
	if m.mode != modeBrowse || len(f.started) != 0 || !strings.Contains(screen(m), "Nothing changed.") {
		t.Fatalf("escape did not back out: mode %v jobs %v", m.mode, f.started)
	}
}

func TestTUIDistroIsLocked(t *testing.T) {
	m, f := tuiFixture(t)
	if s := screen(m); !strings.Contains(s, "locked") || !strings.Contains(s, "cannot be removed") {
		t.Fatalf("distro row not shown as locked:\n%s", s)
	}
	press(m, "space")
	if len(m.marked) != 0 {
		t.Fatal("a distro component was marked")
	}
	press(m, "d")
	if m.mode != modeBrowse || len(f.started) != 0 || !strings.Contains(m.note, "x disables it") {
		t.Fatalf("d on a distro row: mode %v note %q", m.mode, m.note)
	}
	press(m, "x")
	if m.mode != modeConfirm || !strings.Contains(screen(m), "Disable echo?") {
		t.Fatalf("x did not ask:\n%s", screen(m))
	}
	press(m, "enter")
	if len(f.started) != 1 || f.started[0][0].fn == nil || f.started[0][0].label != "disable echo" {
		t.Fatalf("disable job = %+v", f.started)
	}
}

func TestTUIUpdateAndCheckSelected(t *testing.T) {
	m, f := tuiFixture(t)
	if s := screen(m); !strings.Contains(s, "new=3") || !strings.Contains(s, "drift=1") {
		t.Fatalf("badges missing:\n%s", s)
	}
	press(m, "j", "j", "u")
	if len(f.started) != 1 || strings.Join(f.started[0][0].args, " ") != "update ub" {
		t.Fatalf("update job = %+v", f.started)
	}
	m.Update(jobsFinishedMsg{})
	press(m, "enter", "c")
	if len(f.started) != 2 || strings.Join(f.started[1][0].args, " ") != "check ub" {
		t.Fatalf("check job = %+v", f.started)
	}
	m.Update(jobsFinishedMsg{})
	press(m, "enter", "C")
	if len(f.started) != 3 || strings.Join(f.started[2][0].args, " ") != "check" {
		t.Fatalf("check-all job = %+v", f.started)
	}
}

func TestTUIProjectsFlagMissingAndForget(t *testing.T) {
	m, _ := tuiFixture(t)
	gone := m.projects[0].Root
	if err := writeProjects(m.home, []projectRecord{{Root: gone}, {Root: m.projects[1].Root}}); err != nil {
		t.Fatal(err)
	}
	press(m, "3")
	s := screen(m)
	if !strings.Contains(s, "missing") || !strings.Contains(s, "no longer exists") || !strings.Contains(s, "~/here") {
		t.Fatalf("projects view:\n%s", s)
	}
	press(m, "f")
	list, err := readProjects(m.home)
	if err != nil || len(list) != 1 || list[0].Root == gone {
		t.Fatalf("after forget = %v %v", list, err)
	}
	press(m, "j")
	if s := screen(m); !strings.Contains(s, "ub-skill drift=1") {
		t.Fatalf("project detail:\n%s", s)
	}
}

func TestTUIEscapeAsksBeforeCancelling(t *testing.T) {
	m, f := tuiFixture(t)
	press(m, "j", "space", "space", "d", "enter")
	m.Update(jobStartedMsg{i: 0})
	press(m, "esc")
	if s := screen(m); m.mode != modeAsk || !strings.Contains(s, "remove ua is still running") {
		t.Fatalf("esc did not ask:\n%s", s)
	}
	press(m, "esc")
	if m.mode != modeRun || f.cancelled {
		t.Fatal("esc in the prompt must go back without cancelling")
	}
	press(m, "esc", "j", "enter")
	if !f.cancelled || m.mode != modeRun {
		t.Fatalf("stop did not cancel: %v %v", f.cancelled, m.mode)
	}
}

func TestTUISummaryReportsEachJob(t *testing.T) {
	m, _ := tuiFixture(t)
	press(m, "j", "space", "space", "d", "enter")
	for _, msg := range []tea.Msg{
		jobStartedMsg{i: 0}, jobOutputMsg{i: 0, line: "ua removed"}, jobDoneMsg{i: 0},
		jobStartedMsg{i: 1}, jobOutputMsg{i: 1, line: "utility ub: boom"}, jobDoneMsg{i: 1, code: 1},
	} {
		m.Update(msg)
	}
	if s := screen(m); !strings.Contains(s, "utility ub: boom") || !strings.Contains(s, "100%") {
		t.Fatalf("run view:\n%s", s)
	}
	m.Update(jobsFinishedMsg{})
	s := screen(m)
	if m.mode != modeSummary || !strings.Contains(s, "1 ok, 1 failed, 0 not run.") || !strings.Contains(s, "failed (exit 1)") || !strings.Contains(s, "utility ub: boom") {
		t.Fatalf("summary:\n%s", s)
	}
	press(m, "enter")
	if m.mode != modeBrowse || !m.loading {
		t.Fatal("closing the summary must go back and reload")
	}
}

func loadMsgs(cmd tea.Cmd) []loadedMsg {
	if cmd == nil {
		return nil
	}
	switch msg := cmd().(type) {
	case loadedMsg:
		return []loadedMsg{msg}
	case tea.BatchMsg:
		var out []loadedMsg
		for _, c := range msg {
			out = append(out, loadMsgs(c)...)
		}
		return out
	}
	return nil
}

func refresh(t *testing.T, m *tuiModel) loadedMsg {
	t.Helper()
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})
	msgs := loadMsgs(cmd)
	if len(msgs) != 1 || !msgs[0].quick || msgs[0].gen != m.gen {
		t.Fatalf("refresh msgs = %+v gen %d", msgs, m.gen)
	}
	return msgs[0]
}

func twoPhaseFixture(t *testing.T) (*tuiModel, *int) {
	t.Helper()
	m, _ := tuiFixture(t)
	here := filepath.Join(m.home, "here")
	m.fetchStatus = func() tea.Cmd { return nil }
	m.loadQuick = func() loadedMsg {
		return loadedMsg{
			items: []itemRow{
				{ID: "echo", Source: "distro", Scope: "global", Commit: "abc1234", Store: storeUnpinned, Pinned: true},
				{ID: "ua", Source: "user", Scope: "global", Commit: "1111111", Store: storeUnpinned, Pinned: true},
				{ID: "ub", Source: "user", Scope: "project", Commit: "2222222", Store: storeUnpinned, Pinned: true},
			},
			projects: []projectView{{Root: here}},
		}
	}
	fulls := 0
	m.loadFull = func() loadedMsg {
		fulls++
		if fulls == 1 {
			return loadedMsg{items: []itemRow{{ID: "stale", Source: "user", Scope: "global", Commit: "9999999", Store: storeOK, Pinned: true}}}
		}
		return loadedMsg{
			items: []itemRow{
				{ID: "echo", Source: "distro", Scope: "global", Commit: "abc1234", Store: storeOK, Pinned: true},
				{ID: "ua", Source: "user", Scope: "global", Commit: "1111111", Store: storeOK, Pinned: true},
				{ID: "ub", Source: "user", Scope: "project", Commit: "2222222", Store: storeOK, Pinned: true, Drift: 1},
			},
			projects: []projectView{{Root: here, Profile: "project", Items: []projectItem{{ID: "ub", Skills: [][]string{{"ub-skill", "drift=1"}}, Drift: 1}}}},
		}
	}
	return m, &fulls
}

func TestTUIQuickLoadShowsCheckingThenFull(t *testing.T) {
	m, fulls := twoPhaseFixture(t)
	*fulls = 1
	press(m, "j", "space")
	quick := refresh(t, m)
	_, next := m.Update(quick)
	s := screen(m)
	if !m.checking || !strings.Contains(s, "abc1234   checking") || strings.Contains(s, "unpinned") || !m.marked["ua"] {
		t.Fatalf("phase 1 screen (marked %v):\n%s", m.marked, s)
	}
	press(m, "3")
	if s := screen(m); !strings.Contains(s, "~/here  checking") || strings.Contains(s, "drift=") {
		t.Fatalf("phase 1 projects:\n%s", s)
	}
	press(m, "1")
	full := loadMsgs(next)
	if len(full) != 1 || full[0].quick || full[0].gen != m.gen {
		t.Fatalf("phase 2 msgs = %+v", full)
	}
	m.Update(full[0])
	s = screen(m)
	if m.checking || m.loading || strings.Contains(s, "checking") || !strings.Contains(s, "abc1234   ok") || !strings.Contains(s, "drift=1") || !m.marked["ua"] || m.sel[tabItems] != 2 {
		t.Fatalf("phase 2 screen (marked %v sel %d):\n%s", m.marked, m.sel[tabItems], s)
	}
	press(m, "3")
	if s := screen(m); !strings.Contains(s, "~/here  project drift=1") {
		t.Fatalf("phase 2 projects:\n%s", s)
	}
}

func TestTUIStaleFullLoadIgnored(t *testing.T) {
	m, _ := twoPhaseFixture(t)
	first := refresh(t, m)
	_, staleFull := m.Update(first)
	second := refresh(t, m)
	stale := loadMsgs(staleFull)
	if len(stale) != 1 || stale[0].gen == m.gen {
		t.Fatalf("stale msgs = %+v gen %d", stale, m.gen)
	}
	if _, cmd := m.Update(stale[0]); cmd != nil {
		t.Fatal("a stale result started another load")
	}
	if _, cmd := m.Update(first); cmd != nil {
		t.Fatal("a stale quick result started another load")
	}
	if s := screen(m); strings.Contains(s, "stale") || !m.checking {
		t.Fatalf("stale phase 2 applied:\n%s", s)
	}
	_, next := m.Update(second)
	m.Update(loadMsgs(next)[0])
	if s := screen(m); strings.Contains(s, "stale") || !strings.Contains(s, "abc1234   ok") {
		t.Fatalf("newest phase 2 not applied:\n%s", s)
	}
}

func TestTUITabsAndStatus(t *testing.T) {
	m, _ := tuiFixture(t)
	m.Update(statusTextMsg{text: "echo ref=master commit=abc1234 store=ok"})
	press(m, "tab")
	if m.tab != tabStatus || !strings.Contains(screen(m), "echo ref=master commit=abc1234 store=ok") {
		t.Fatalf("status tab:\n%s", screen(m))
	}
	press(m, "tab", "tab")
	if m.tab != tabItems {
		t.Fatalf("tab cycled to %d", m.tab)
	}
	press(m, "q")
	if !m.quitting {
		t.Fatal("q did not quit")
	}
}
