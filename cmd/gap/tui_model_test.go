package main

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
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
	if len(f.started) != 1 || len(f.started[0]) != 2 || strings.Join(f.started[0][0].args, " ") != "remove -- ua" || strings.Join(f.started[0][1].args, " ") != "remove -- ub" {
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
	if len(f.started) != 1 || strings.Join(f.started[0][0].args, " ") != "update -- ub" {
		t.Fatalf("update job = %+v", f.started)
	}
	m.Update(jobsFinishedMsg{})
	press(m, "enter", "c")
	if len(f.started) != 2 || strings.Join(f.started[1][0].args, " ") != "check -- ub" {
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
	m.fetchStatus = func(context.Context) tea.Cmd { return nil }
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
	m.loadFull = func(*storeCache) loadedMsg {
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

func numbered(prefix string, n int) []string {
	var out []string
	for i := range n {
		out = append(out, fmt.Sprintf("%s-%03d", prefix, i))
	}
	return out
}

func TestTUIStatusScrollKeys(t *testing.T) {
	m, _ := tuiFixture(t)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 20})
	m.Update(statusTextMsg{text: strings.Join(numbered("line", 100), "\n")})
	press(m, "2")
	if s := screen(m); !strings.Contains(s, "line-000") || strings.Contains(s, "line-099") {
		t.Fatalf("status top:\n%s", s)
	}
	press(m, "G")
	if s := screen(m); !strings.Contains(s, "line-099") || strings.Contains(s, "line-000") {
		t.Fatalf("G did not reach the end:\n%s", s)
	}
	press(m, "g", "g", "j")
	if s := screen(m); strings.Contains(s, "line-000") || !strings.Contains(s, "line-001") {
		t.Fatalf("j did not scroll:\n%s", s)
	}
	press(m, "k")
	if s := screen(m); !strings.Contains(s, "line-000") {
		t.Fatalf("k did not scroll back:\n%s", s)
	}
}

func cmdMsgs(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		var out []tea.Msg
		for _, c := range batch {
			out = append(out, cmdMsgs(c)...)
		}
		return out
	}
	return []tea.Msg{msg}
}

func statusOf(t *testing.T, cmd tea.Cmd) statusTextMsg {
	t.Helper()
	for _, msg := range cmdMsgs(cmd) {
		if s, ok := msg.(statusTextMsg); ok {
			return s
		}
	}
	t.Fatal("no status message")
	return statusTextMsg{}
}

func TestTUIStaleStatusIgnored(t *testing.T) {
	m, _ := tuiFixture(t)
	text := "before the update"
	m.fetchStatus = func(context.Context) tea.Cmd {
		now := text
		return func() tea.Msg { return statusTextMsg{text: now} }
	}
	m.loadQuick = func() loadedMsg { return loadedMsg{} }
	press(m, "2")
	_, older := m.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})
	text = "after the update"
	_, newer := m.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})
	stale, fresh := statusOf(t, older), statusOf(t, newer)
	m.Update(fresh)
	m.Update(stale)
	if s := screen(m); !strings.Contains(s, "after the update") || strings.Contains(s, "before the update") || m.statusLoading {
		t.Fatalf("status:\n%s", s)
	}
}

func TestTUIStatusRunsOnlyWhenShown(t *testing.T) {
	m, _ := tuiFixture(t)
	calls := 0
	m.fetchStatus = func(context.Context) tea.Cmd {
		calls++
		return func() tea.Msg { return statusTextMsg{text: "status"} }
	}
	m.loadQuick = func() loadedMsg { return loadedMsg{} }
	run := func(cmd tea.Cmd) {
		for _, msg := range cmdMsgs(cmd) {
			m.Update(msg)
		}
	}
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})
	if calls != 0 {
		t.Fatalf("refresh on Items ran gap status %d times", calls)
	}
	_, cmd = m.Update(tea.KeyPressMsg{Code: '2', Text: "2"})
	run(cmd)
	if calls != 1 || !strings.Contains(screen(m), "status") {
		t.Fatalf("opening Status ran it %d times", calls)
	}
	press(m, "1")
	if _, cmd = m.Update(tea.KeyPressMsg{Code: '2', Text: "2"}); cmd != nil || calls != 1 {
		t.Fatalf("Status ran again in the same refresh: %d", calls)
	}
	_, cmd = m.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})
	run(cmd)
	if calls != 2 {
		t.Fatalf("refresh on Status ran it %d times, want 2", calls)
	}
}

func TestTUIStatusStopsWhenLeft(t *testing.T) {
	m, _ := tuiFixture(t)
	var reqs []context.Context
	m.fetchStatus = func(ctx context.Context) tea.Cmd {
		reqs = append(reqs, ctx)
		return func() tea.Msg { return statusTextMsg{text: "report"} }
	}
	press(m, "2", "1")
	if len(reqs) != 1 || reqs[0].Err() == nil {
		t.Fatalf("leaving Status did not cancel its request: %d requests", len(reqs))
	}
	_, second := m.Update(tea.KeyPressMsg{Code: '2', Text: "2"})
	if len(reqs) != 2 || second == nil {
		t.Fatalf("reopening Status did not restart it: %d requests", len(reqs))
	}
	for _, msg := range cmdMsgs(second) {
		m.Update(msg)
	}
	press(m, "1")
	if _, cmd := m.Update(tea.KeyPressMsg{Code: '2', Text: "2"}); cmd != nil || len(reqs) != 2 || !strings.Contains(screen(m), "report") {
		t.Fatalf("a finished report was not reused: %d requests\n%s", len(reqs), screen(m))
	}
}

func TestTUIStatusReflowsWhenPaneNarrows(t *testing.T) {
	m, _ := tuiFixture(t)
	n := 0
	m.fetchStatus = func(context.Context) tea.Cmd {
		n++
		text := fmt.Sprintf("report %d at %d", n, m.width)
		return func() tea.Msg { return statusTextMsg{text: text} }
	}
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	_, open := m.Update(tea.KeyPressMsg{Code: '2', Text: "2"})
	w := statusOf(t, open)
	m.Update(w)
	if !strings.Contains(screen(m), "report 1 at 120") {
		t.Fatalf("wide report not shown:\n%s", screen(m))
	}
	_, narrower := m.Update(tea.WindowSizeMsg{Width: 60, Height: 30})
	if narrower == nil || m.statusCols != 56 {
		t.Fatalf("narrowing did not restart status: cols %d cmd %v", m.statusCols, narrower != nil)
	}
	m.Update(w)
	if strings.Contains(screen(m), "report 1") || !m.statusLoading {
		t.Fatalf("the wide report survived the resize:\n%s", screen(m))
	}
	m.Update(statusOf(t, narrower))
	if s := screen(m); !strings.Contains(s, "report 2 at 60") || m.statusLoading {
		t.Fatalf("narrow report not shown:\n%s", s)
	}
}

func TestTUIStatusReflowsWhenHiddenTabResizes(t *testing.T) {
	m, _ := tuiFixture(t)
	n := 0
	m.fetchStatus = func(context.Context) tea.Cmd {
		n++
		text := fmt.Sprintf("report %d at %d", n, m.width)
		return func() tea.Msg { return statusTextMsg{text: text} }
	}
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	_, open := m.Update(tea.KeyPressMsg{Code: '2', Text: "2"})
	m.Update(statusOf(t, open))
	press(m, "1")
	if _, cmd := m.Update(tea.WindowSizeMsg{Width: 60, Height: 30}); cmd != nil || n != 1 {
		t.Fatalf("a hidden Status tab restarted on resize: calls %d", n)
	}
	_, back := m.Update(tea.KeyPressMsg{Code: '2', Text: "2"})
	if back == nil || n != 2 || m.statusCols != 56 || strings.Contains(screen(m), "report 1") {
		t.Fatalf("returning to Status reused the wide report: calls %d cols %d\n%s", n, m.statusCols, screen(m))
	}
	m.Update(statusOf(t, back))
	if s := screen(m); !strings.Contains(s, "report 2 at 60") {
		t.Fatalf("narrow report not shown on return:\n%s", s)
	}
	if _, again := m.Update(tea.KeyPressMsg{Code: '1', Text: "1"}); again != nil {
		t.Fatal("leaving Status restarted it")
	}
	if _, same := m.Update(tea.KeyPressMsg{Code: '2', Text: "2"}); same != nil || n != 2 {
		t.Fatalf("matching width was not reused: calls %d", n)
	}
	press(m, "1")
	_, refreshed := m.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})
	for _, msg := range cmdMsgs(refreshed) {
		m.Update(msg)
	}
	m.Update(tea.WindowSizeMsg{Width: 40, Height: 30})
	_, after := m.Update(tea.KeyPressMsg{Code: '2', Text: "2"})
	if after == nil || strings.Contains(screen(m), "report 1") || strings.Contains(screen(m), "report 2") {
		t.Fatalf("a refresh then a hidden resize kept the old report:\n%s", screen(m))
	}
	m.Update(statusOf(t, after))
	if s := screen(m); !strings.Contains(s, "report 3 at 40") {
		t.Fatalf("report after refresh and resize not shown:\n%s", s)
	}
}

func TestTUIStatusKeepsReportAcrossHiddenResizeRoundTrip(t *testing.T) {
	m, _ := tuiFixture(t)
	n := 0
	m.fetchStatus = func(context.Context) tea.Cmd {
		n++
		text := fmt.Sprintf("report %d at %d", n, m.width)
		return func() tea.Msg { return statusTextMsg{text: text} }
	}
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	_, open := m.Update(tea.KeyPressMsg{Code: '2', Text: "2"})
	m.Update(statusOf(t, open))
	press(m, "1")
	m.Update(tea.WindowSizeMsg{Width: 60, Height: 30})
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	_, back := m.Update(tea.KeyPressMsg{Code: '2', Text: "2"})
	if s := screen(m); back != nil || n != 1 || !strings.Contains(s, "report 1 at 120") {
		t.Fatalf("a hidden resize round trip dropped the report: calls %d\n%s", n, s)
	}
}

func TestTUIStatusLateRefreshKeepsHiddenReport(t *testing.T) {
	m, _ := tuiFixture(t)
	n := 0
	m.fetchStatus = func(context.Context) tea.Cmd {
		n++
		text := fmt.Sprintf("report %d at %d", n, m.width)
		return func() tea.Msg { return statusTextMsg{text: text} }
	}
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	_, open := m.Update(tea.KeyPressMsg{Code: '2', Text: "2"})
	m.Update(statusOf(t, open))
	_, refresh := m.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})
	var late tea.Msg
	for _, msg := range cmdMsgs(refresh) {
		if _, ok := msg.(statusTextMsg); ok {
			late = msg
			continue
		}
		m.Update(msg)
	}
	if late == nil {
		t.Fatal("refresh did not ask for status")
	}
	press(m, "1")
	m.Update(tea.WindowSizeMsg{Width: 60, Height: 30})
	m.Update(late)
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	m.Update(tea.KeyPressMsg{Code: '2', Text: "2"})
	if s := screen(m); !strings.Contains(s, "report 1 at 120") {
		t.Fatalf("a late refresh response cleared the hidden report:\n%s", s)
	}
}

func TestTUIStatusSupersededResponseIgnored(t *testing.T) {
	m, _ := tuiFixture(t)
	var reqs []context.Context
	n := 0
	m.fetchStatus = func(ctx context.Context) tea.Cmd {
		reqs = append(reqs, ctx)
		n++
		text := fmt.Sprintf("report %d", n)
		return func() tea.Msg { return statusTextMsg{text: text} }
	}
	_, first := m.Update(tea.KeyPressMsg{Code: '2', Text: "2"})
	press(m, "1")
	_, second := m.Update(tea.KeyPressMsg{Code: '2', Text: "2"})
	m.Update(statusOf(t, first))
	if reqs[1].Err() != nil || !m.statusLoading || strings.Contains(screen(m), "report 1") {
		t.Fatalf("a superseded response took over: loading %v\n%s", m.statusLoading, screen(m))
	}
	m.Update(statusOf(t, second))
	if s := screen(m); !strings.Contains(s, "report 2") || m.statusLoading {
		t.Fatalf("current response not shown:\n%s", s)
	}
}

func TestTUIStatusSameTabKeepsRequest(t *testing.T) {
	m, _ := tuiFixture(t)
	var reqs []context.Context
	m.fetchStatus = func(ctx context.Context) tea.Cmd {
		reqs = append(reqs, ctx)
		return func() tea.Msg { return statusTextMsg{text: "report"} }
	}
	press(m, "2", "2")
	if len(reqs) != 1 || reqs[0].Err() != nil || !m.statusLoading {
		t.Fatalf("pressing 2 on Status restarted it: %d requests", len(reqs))
	}
}

func TestTUIQuickLoadClearsOldRegistryError(t *testing.T) {
	m, _ := tuiFixture(t)
	m.fetchStatus = func(context.Context) tea.Cmd { return nil }
	m.tab = tabProjects
	m.Update(loadedMsg{projectsOnly: true, projErr: fmt.Errorf("projects.json: broken")})
	if !strings.Contains(screen(m), "projects.json: broken") {
		t.Fatalf("registry error not shown:\n%s", screen(m))
	}
	m.tab = tabItems
	m.loadQuick = func() loadedMsg { return loadedMsg{items: m.items} }
	m.Update(refresh(t, m))
	m.note = ""
	if s := screen(m); strings.Contains(s, "broken") {
		t.Fatalf("repaired registry still reports an error:\n%s", s)
	}
}

func TestTUIStepResultSurvivesLongLabels(t *testing.T) {
	long := "remove " + strings.Repeat("x", 60)
	m, _ := tuiFixture(t)
	m.Update(tea.WindowSizeMsg{Width: 40, Height: 20})
	m.run([]tuiJob{{label: long}, {label: "disable echo", fn: func() error { return nil }}})
	m.Update(jobStartedMsg{i: 0})
	if s := screen(m); !strings.Contains(s, "1/2") {
		t.Fatalf("run title lost the step number:\n%s", s)
	}
	m.Update(jobDoneMsg{i: 0, code: 1})
	m.Update(jobStartedMsg{i: 1})
	m.Update(jobDoneMsg{i: 1})
	m.Update(jobsFinishedMsg{})
	if s := screen(m); !strings.Contains(s, "failed (exit 1)") || !strings.Contains(s, "1/2") {
		t.Fatalf("narrow summary hid the result:\n%s", s)
	}
	press(m, "j")
	if s := screen(m); !strings.Contains(s, "No output.") || strings.Contains(s, "Waiting to run.") {
		t.Fatalf("finished step without output:\n%s", s)
	}
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 20})
	press(m, "k")
	if s := screen(m); strings.Count(s, "failed (exit 1)") != 2 {
		t.Fatalf("wide summary hid the result:\n%s", s)
	}
	m.Update(tea.WindowSizeMsg{Width: 24, Height: 20})
	if s := screen(m); !strings.Contains(s, "exit 1") || !strings.Contains(s, "1/2") {
		t.Fatalf("24-column summary hid the result or step:\n%s", s)
	}
}

func TestTUIDetailOnNarrowTerminal(t *testing.T) {
	m, _ := tuiFixture(t)
	skills := [][]string{}
	for _, name := range numbered("skill", 30) {
		skills = append(skills, []string{name, "drift=0"})
	}
	m.projects[1].Items = []projectItem{{ID: "ub", Store: storeOK, Skills: skills}}
	m.Update(tea.WindowSizeMsg{Width: 50, Height: 16})
	press(m, "j")
	if s := screen(m); strings.Contains(s, "hooks") {
		t.Fatalf("narrow list already shows details:\n%s", s)
	}
	press(m, "enter")
	if s := screen(m); m.mode != modeDetail || !strings.Contains(s, "ua-skill drift=0") || !strings.Contains(s, "hooks") {
		t.Fatalf("item detail:\n%s", s)
	}
	press(m, "esc", "3", "j", "enter")
	if s := screen(m); !strings.Contains(s, "skill-000") || strings.Contains(s, "skill-029") {
		t.Fatalf("project detail top:\n%s", s)
	}
	press(m, "G")
	if s := screen(m); !strings.Contains(s, "skill-029") {
		t.Fatalf("project detail bottom:\n%s", s)
	}
	for _, l := range strings.Split(m.render(), "\n") {
		if w := ansi.StringWidth(l); w > 50 {
			t.Fatalf("line is %d cols: %q", w, ansi.Strip(l))
		}
	}
	press(m, "esc")
	if m.mode != modeBrowse {
		t.Fatalf("esc left mode %v", m.mode)
	}
}

func TestTUISummaryScrollsToEveryStep(t *testing.T) {
	for _, w := range []int{140, 50} {
		m, _ := tuiFixture(t)
		m.Update(tea.WindowSizeMsg{Width: w, Height: 20})
		var jobs []tuiJob
		for i := range 30 {
			id := fmt.Sprintf("u%02d", i)
			jobs = append(jobs, tuiJob{label: "update " + id, args: []string{"update", "--", id}})
		}
		m.run(jobs)
		for i := range jobs {
			m.Update(jobStartedMsg{i: i})
			m.Update(jobOutputMsg{i: i, line: fmt.Sprintf("out-%02d", i)})
			code := 0
			if i == 25 {
				m.Update(jobOutputMsg{i: i, line: "boom-25"})
				code = 1
			}
			m.Update(jobDoneMsg{i: i, code: code})
		}
		m.Update(jobsFinishedMsg{})
		if s := screen(m); !strings.Contains(s, "boom-25") || !strings.Contains(s, "update u25") || !strings.Contains(s, "29 ok, 1 failed, 0 not run.") {
			t.Fatalf("width %d summary:\n%s", w, s)
		}
		press(m, "g", "g")
		if s := screen(m); !strings.Contains(s, "update u00") || !strings.Contains(s, "out-00") {
			t.Fatalf("width %d first step:\n%s", w, s)
		}
		press(m, "j", "j", "j")
		if s := screen(m); !strings.Contains(s, "out-03") {
			t.Fatalf("width %d step 3 output:\n%s", w, s)
		}
		press(m, "G")
		if s := screen(m); !strings.Contains(s, "update u29") || !strings.Contains(s, "out-29") || m.mode != modeSummary {
			t.Fatalf("width %d last step:\n%s", w, s)
		}
		press(m, "enter")
		if m.mode != modeBrowse {
			t.Fatalf("width %d: enter did not close the summary", w)
		}
	}
}

func TestTUIDashIDIsNotAFlag(t *testing.T) {
	m, f := tuiFixture(t)
	m.Update(loadedMsg{items: []itemRow{{ID: "-h", Source: "user", Scope: "global", Commit: "1111111", Store: storeOK, Pinned: true}}})
	press(m, "space")
	if !m.marked["-h"] {
		t.Fatal("-h not marked")
	}
	press(m, "d", "enter")
	if len(f.started) != 1 || strings.Join(f.started[0][0].args, " ") != "remove -- -h" || len(m.marked) != 0 {
		t.Fatalf("jobs %+v marked %v", f.started, m.marked)
	}
	m.Update(jobsFinishedMsg{})
	press(m, "enter", "u")
	m.Update(jobsFinishedMsg{})
	press(m, "enter", "c")
	if len(f.started) != 3 || strings.Join(f.started[1][0].args, " ") != "update -- -h" || strings.Join(f.started[2][0].args, " ") != "check -- -h" {
		t.Fatalf("jobs %+v", f.started)
	}
}

func TestTUIProjectsLoadOnlyWhenShown(t *testing.T) {
	m, _ := tuiFixture(t)
	m.fetchStatus = func(context.Context) tea.Cmd { return nil }
	m.loadQuick = func() loadedMsg {
		return loadedMsg{items: m.items, projects: []projectView{{Root: filepath.Join(m.home, "here")}}}
	}
	m.loadFull = func(*storeCache) loadedMsg { return loadedMsg{items: m.items} }
	calls := 0
	m.loadProjects = func(*storeCache) loadedMsg {
		calls++
		return loadedMsg{projects: []projectView{{Root: filepath.Join(m.home, "here"), Profile: "project"}}}
	}
	quick := refresh(t, m)
	_, next := m.Update(quick)
	for _, msg := range cmdMsgs(next) {
		m.Update(msg)
	}
	if calls != 0 || !m.projChecking {
		t.Fatalf("projects loaded on the Items tab: calls %d checking %v", calls, m.projChecking)
	}
	_, cmd := m.Update(tea.KeyPressMsg{Code: '3', Text: "3"})
	for _, msg := range cmdMsgs(cmd) {
		m.Update(msg)
	}
	if calls != 1 || m.projChecking || !strings.Contains(screen(m), "~/here  project ok") {
		t.Fatalf("projects after showing: calls %d\n%s", calls, screen(m))
	}
	press(m, "1")
	if _, cmd := m.Update(tea.KeyPressMsg{Code: '3', Text: "3"}); cmd != nil || calls != 1 {
		t.Fatalf("projects loaded twice in one refresh: calls %d", calls)
	}
}

func TestTUIVimMotions(t *testing.T) {
	m, _ := tuiFixture(t)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 24})
	m.items = nil
	for i := range 40 {
		m.items = append(m.items, itemRow{ID: fmt.Sprintf("item-%02d", i), Scope: "global"})
	}
	h := m.rowsFor(tabItems)
	ctrl := func(r rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: r, Mod: tea.ModCtrl} }
	m.Update(ctrl('d'))
	if m.sel[tabItems] != h/2 {
		t.Fatalf("ctrl+d moved to %d, want %d", m.sel[tabItems], h/2)
	}
	before, _ := m.listWindow(tabItems)
	m.Update(ctrl('f'))
	if m.sel[tabItems] != h/2+h {
		t.Fatalf("ctrl+f moved to %d, want %d", m.sel[tabItems], h/2+h)
	}
	if after, _ := m.listWindow(tabItems); after != before+h {
		t.Fatalf("ctrl+f scrolled the list from %d to %d, want a full page to %d", before, after, before+h)
	}
	m.Update(ctrl('u'))
	m.Update(ctrl('b'))
	if m.sel[tabItems] != 0 {
		t.Fatalf("ctrl+u then ctrl+b left %d, want 0", m.sel[tabItems])
	}
	press(m, "G")
	start, end := m.listWindow(tabItems)
	press(m, "H")
	if m.sel[tabItems] != start {
		t.Fatalf("H moved to %d, want %d", m.sel[tabItems], start)
	}
	if s, e := m.listWindow(tabItems); s != start || e != end {
		t.Fatalf("H scrolled the list to %d-%d, want %d-%d", s, e, start, end)
	}
	press(m, "L")
	if m.sel[tabItems] != end-1 {
		t.Fatalf("L moved to %d, want %d", m.sel[tabItems], end-1)
	}
	start, end = m.listWindow(tabItems)
	press(m, "M")
	if m.sel[tabItems] != (start+end-1)/2 {
		t.Fatalf("M moved to %d, want %d", m.sel[tabItems], (start+end-1)/2)
	}
	press(m, "g")
	if !strings.Contains(screen(m), "gg top") || m.sel[tabItems] == 0 {
		t.Fatalf("g alone should wait for the next key:\n%s", screen(m))
	}
	press(m, "g")
	if m.sel[tabItems] != 0 || m.chord != "" {
		t.Fatalf("gg left %d, chord %q", m.sel[tabItems], m.chord)
	}
	press(m, "g", "t")
	if m.tab != tabStatus {
		t.Fatalf("gt went to tab %d", m.tab)
	}
	press(m, "g", "T", "g", "T")
	if m.tab != tabProjects {
		t.Fatalf("gT twice went to tab %d", m.tab)
	}
	press(m, "1", "g", "j")
	if m.sel[tabItems] != 1 || m.chord != "" {
		t.Fatalf("g then j should drop the chord and move down, got %d %q", m.sel[tabItems], m.chord)
	}
}

func TestTUIVimSummaryKeepsOutputKeys(t *testing.T) {
	m, _ := tuiFixture(t)
	m.Update(tea.WindowSizeMsg{Width: 140, Height: 24})
	var jobs []tuiJob
	for i := range 3 {
		jobs = append(jobs, tuiJob{label: fmt.Sprintf("update u%d", i), args: []string{"update", "--", fmt.Sprintf("u%d", i)}})
	}
	m.run(jobs)
	for i := range jobs {
		m.Update(jobStartedMsg{i: i})
		for l := range 80 {
			m.Update(jobOutputMsg{i: i, line: fmt.Sprintf("j%d-line-%02d", i, l)})
		}
		m.Update(jobDoneMsg{i: i})
	}
	m.Update(jobsFinishedMsg{})
	sel := m.jobSel
	off := m.vp.YOffset()
	m.Update(tea.KeyPressMsg{Code: 'u', Mod: tea.ModCtrl})
	if m.jobSel != sel || m.vp.YOffset() >= off {
		t.Fatalf("ctrl+u should scroll the output up, got job %d offset %d (was %d, %d)", m.jobSel, m.vp.YOffset(), sel, off)
	}
	press(m, "H")
	if m.jobSel != sel {
		t.Fatalf("H moved the summary selection to %d", m.jobSel)
	}
	press(m, "g")
	if !strings.Contains(screen(m), "gg first step") || strings.Contains(screen(m), "gt/gT") {
		t.Fatalf("summary chord hint:\n%s", screen(m))
	}
	press(m, "g")
	if s := screen(m); m.jobSel != 0 || !strings.Contains(s, "j0-line-79") {
		t.Fatalf("gg should open the first step at its final lines, job %d:\n%s", m.jobSel, s)
	}
}

func TestTUIVimDetailHint(t *testing.T) {
	m, _ := tuiFixture(t)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 24})
	press(m, "enter", "g")
	if m.mode != modeDetail {
		t.Fatalf("enter did not open detail, mode %d", m.mode)
	}
	if s := screen(m); !strings.Contains(s, "gg top") || strings.Contains(s, "gt/gT") {
		t.Fatalf("detail chord hint:\n%s", s)
	}
	press(m, "t")
	if m.mode != modeDetail || m.tab != tabItems {
		t.Fatalf("gt switched views from detail: mode %d tab %d", m.mode, m.tab)
	}
}

func TestTUIHelpOverlay(t *testing.T) {
	m, _ := tuiFixture(t)
	m.Update(tea.WindowSizeMsg{Width: 140, Height: 30})
	rows := strings.Split(strings.TrimRight(screen(m), " \n"), "\n")
	foot := rows[len(rows)-1]
	if !strings.Contains(foot, "? all keys") || strings.Contains(foot, "check all") || strings.Contains(foot, "\u2026") {
		t.Fatalf("browse footer should be short and point at ?:\n%s", foot)
	}
	press(m, "?")
	s := screen(m)
	for _, want := range []string{"Items keys", "half page", "H/M/L", "check all", "open gap.toml", "disable", "?/esc close"} {
		if !strings.Contains(s, want) {
			t.Fatalf("help overlay lacks %q:\n%s", want, s)
		}
	}
	press(m, "j", "d")
	if !m.showHelp || m.mode != modeBrowse || m.sel[tabItems] != 0 {
		t.Fatalf("keys leaked through the overlay: help %v mode %d sel %d", m.showHelp, m.mode, m.sel[tabItems])
	}
	press(m, "esc")
	if m.showHelp || strings.Contains(screen(m), "Items keys") {
		t.Fatalf("esc did not close the overlay")
	}
	press(m, "3", "?")
	if s := screen(m); !strings.Contains(s, "Projects keys") || !strings.Contains(s, "forget") {
		t.Fatalf("projects help:\n%s", s)
	}
	press(m, "?", "2", "?")
	if s := screen(m); !strings.Contains(s, "Status keys") || strings.Contains(s, "H/M/L") {
		t.Fatalf("status help:\n%s", s)
	}
	press(m, "q")
	if m.showHelp || m.quitting {
		t.Fatalf("q should close the overlay, not quit: help %v quitting %v", m.showHelp, m.quitting)
	}
}

func TestTUIHelpFitsSmallTerminal(t *testing.T) {
	m, _ := tuiFixture(t)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	for _, tab := range []string{"1", "2", "3"} {
		press(m, tab)
		rows := strings.Split(strings.TrimRight(screen(m), " \n"), "\n")
		if foot := rows[len(rows)-1]; !strings.Contains(foot, "? all keys") {
			t.Fatalf("tab %s footer at 80 columns hides ?:\n%s", tab, foot)
		}
	}
	press(m, "1", "?")
	seen := screen(m)
	for range 40 {
		press(m, "j")
		seen += screen(m)
	}
	for _, want := range []string{"half page", "next/previous view", "disable", "update all", "check all", "find skills", "quit", "j/k scroll"} {
		if !strings.Contains(seen, want) {
			t.Fatalf("80x24 help never shows %q:\n%s", want, seen)
		}
	}
	press(m, "esc", "3", "?")
	if s := screen(m); !strings.Contains(s, "find skills") || !strings.Contains(s, "forget") {
		t.Fatalf("projects help lacks shared browse keys:\n%s", s)
	}
	press(m, "esc", "1", "enter", "?")
	if s := screen(m); !strings.Contains(s, "esc/enter/q") || !strings.Contains(s, "pgup/pgdn") {
		t.Fatalf("detail help lacks its back and paging keys:\n%s", s)
	}
}

func TestTUIHelpWrapsNarrowGroups(t *testing.T) {
	m, _ := tuiFixture(t)
	m.Update(tea.WindowSizeMsg{Width: 34, Height: 24})
	press(m, "?")
	for _, l := range strings.Split(m.helpVP.GetContent(), "\n") {
		if w := lipgloss.Width(l); w > m.helpVP.Width() {
			t.Fatalf("help line is %d wide in a %d pane: %q", w, m.helpVP.Width(), ansi.Strip(l))
		}
	}
	all := ansi.Strip(m.helpVP.GetContent())
	for _, want := range []string{"screen", "top/mid/bottom", "next/previous"} {
		if !strings.Contains(all, want) {
			t.Fatalf("narrow help lost %q:\n%s", want, all)
		}
	}
}

func TestTUIFilter(t *testing.T) {
	m, _ := tuiFixture(t)
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	m.items = nil
	for _, id := range []string{"alpha", "beta", "gamma", "alpine"} {
		m.items = append(m.items, itemRow{ID: id, Scope: "global"})
	}
	press(m, "j", "j", "/")
	typeKeys(m, "a", "l")
	if m.mode != modeFilter || m.count(tabItems) != 2 || m.sel[tabItems] != 0 {
		t.Fatalf("live filter: mode %d count %d sel %d", m.mode, m.count(tabItems), m.sel[tabItems])
	}
	s := screen(m)
	if !strings.Contains(s, "alpine") || strings.Contains(s, "gamma") || !strings.Contains(s, "2 matches") {
		t.Fatalf("filtered screen:\n%s", s)
	}
	press(m, "enter", "j")
	if r, _ := m.item(); m.mode != modeBrowse || r.ID != "alpine" {
		t.Fatalf("kept filter should move within matches, got %q mode %d", r.ID, m.mode)
	}
	if !strings.Contains(screen(m), "2 of 4 shown") {
		t.Fatalf("kept filter is not shown:\n%s", screen(m))
	}
	press(m, "/")
	typeKeys(m, "z")
	if !strings.Contains(screen(m), "Nothing matches /alz") {
		t.Fatalf("empty match:\n%s", screen(m))
	}
	press(m, "esc")
	if m.filter[tabItems] != "al" || m.mode != modeBrowse {
		t.Fatalf("esc in the line should restore the kept filter, got %q", m.filter[tabItems])
	}
	press(m, "esc")
	if m.filter[tabItems] != "" || m.count(tabItems) != 4 {
		t.Fatalf("esc in browse should clear the filter, got %q", m.filter[tabItems])
	}
	press(m, "2", "/")
	if m.mode != modeBrowse || !strings.Contains(screen(m), "no list to filter") {
		t.Fatalf("status filter: mode %d\n%s", m.mode, screen(m))
	}
}

func TestTUICommandLine(t *testing.T) {
	m, _ := tuiFixture(t)
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	press(m, ":")
	typeKeys(m, "s", "y")
	if s := screen(m); m.mode != modeCmd || !strings.Contains(s, ":sy") || !strings.Contains(s, "sync") {
		t.Fatalf("command line: mode %d\n%s", m.mode, s)
	}
	press(m, "tab")
	if m.line.Value() != "sync " {
		t.Fatalf("tab completed to %q", m.line.Value())
	}
	press(m, "esc", ":")
	typeKeys(m, "s")
	press(m, "enter")
	if !strings.Contains(m.note, "could be") {
		t.Fatalf("ambiguous :s note %q", m.note)
	}
	press(m, ":")
	typeKeys(m, "n", "o", "p", "e")
	press(m, "enter")
	if !strings.Contains(m.note, "Unknown command :nope") {
		t.Fatalf("unknown command note %q", m.note)
	}
	press(m, ":")
	typeKeys(m, "f", "i", "l", " ", "z", "z", "z")
	press(m, "enter")
	if m.filter[tabItems] != "zzz" || m.count(tabItems) != 0 {
		t.Fatalf(":filter set %q", m.filter[tabItems])
	}
	press(m, ":")
	typeKeys(m, "c", "h", "e", "c", "k", " ", "n", "o", "-", "s", "u", "c", "h")
	press(m, "enter")
	if m.note != "No item no-such." || m.mode != modeBrowse {
		t.Fatalf(":check unknown id: %q mode %d", m.note, m.mode)
	}
	press(m, ":")
	typeKeys(m, "u", "p")
	press(m, "enter")
	if m.mode != modeConfirm && !strings.Contains(m.note, "Nothing to update") {
		t.Fatalf(":up should open the update-all confirm, mode %d note %q", m.mode, m.note)
	}
	press(m, "esc", ":")
	typeKeys(m, "s", "e", "a", "r", "c", "h", " ", "r", "e", "a", "c", "t")
	press(m, "enter")
	if m.mode != modeSearch || m.search.input.Value() != "react" {
		t.Fatalf(":search should open the catalog search with its query, mode %d", m.mode)
	}
	press(m, "esc", ":", "q", "enter")
	if !m.quitting {
		t.Fatalf(":q did not quit")
	}
}

func TestTUIPaneFocus(t *testing.T) {
	m, _ := tuiFixture(t)
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 24})
	m.st = newTUIStyles(gapPalette(lipgloss.LightDark(true)))
	pill := func() string {
		rows := strings.Split(ansi.Strip(m.render()), "\n")
		return strings.TrimSpace(rows[len(rows)-2])
	}
	if !strings.HasPrefix(pill(), "NORMAL") {
		t.Fatalf("browse pill: %q", pill())
	}
	top := func(p string) string { return strings.Split(p, "\n")[0] }
	listFocused, sideIdle := top(m.itemList()), top(m.itemDetail())
	if listFocused == top(m.paneOf(m.listWidth(), nil)) {
		t.Fatalf("list pane should carry the focus border")
	}
	for i := range 60 {
		m.items[0].Skills = append(m.items[0].Skills, []string{fmt.Sprintf("skill-%02d", i), "drift=0"})
	}
	press(m, "l")
	if !m.sideOn() || !strings.HasPrefix(pill(), "DETAIL") {
		t.Fatalf("l should focus the detail pane, pill %q", pill())
	}
	if top(m.itemDetail()) == sideIdle || top(m.itemList()) == listFocused {
		t.Fatalf("the focus border should move to the detail pane")
	}
	sel := m.sel[tabItems]
	press(m, "j", "j")
	if m.sideTop != 2 || m.sel[tabItems] != sel {
		t.Fatalf("j in the detail pane should scroll it, top %d sel %d", m.sideTop, m.sel[tabItems])
	}
	press(m, "G")
	if m.sideTop != m.sideMax() || m.sideMax() == 0 {
		t.Fatalf("G scrolled to %d of %d", m.sideTop, m.sideMax())
	}
	press(m, "g", "g")
	if m.sideTop != 0 {
		t.Fatalf("gg left the detail at %d", m.sideTop)
	}
	press(m, "h")
	if m.side || !strings.HasPrefix(pill(), "NORMAL") {
		t.Fatalf("h should return focus to the list")
	}
	press(m, "j")
	if m.sel[tabItems] != sel+1 {
		t.Fatalf("j after h should move the list, sel %d", m.sel[tabItems])
	}
	press(m, "l", "2")
	if m.side {
		t.Fatalf("switching views should drop detail focus")
	}
	press(m, "1", "/")
	if !strings.HasPrefix(pill(), "FILTER") {
		t.Fatalf("filter pill: %q", pill())
	}
	press(m, "esc", ":")
	if !strings.HasPrefix(pill(), "CMD") {
		t.Fatalf("command pill: %q", pill())
	}
	press(m, "esc")
	m.Update(tea.WindowSizeMsg{Width: 60, Height: 24})
	press(m, "l")
	if m.side {
		t.Fatalf("l should do nothing when the panes are stacked")
	}
}
