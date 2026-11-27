package main

import (
	"context"
	"fmt"
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
	press(m, "g", "j")
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
		press(m, "g")
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
