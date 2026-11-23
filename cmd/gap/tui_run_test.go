package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

func init() {
	if os.Getenv("GAP_TEST_AS_GAP") != "1" {
		return
	}
	sourceRootFor = os.Getenv("GAP_TEST_SOURCE")
	os.Exit(run(os.Args[1:]))
}

func actAsGap(t *testing.T, repo string) {
	t.Helper()
	t.Setenv("GAP_TEST_AS_GAP", "1")
	t.Setenv("GAP_TEST_SOURCE", repo)
	t.Setenv("PATH", "/usr/bin:/bin")
}

func addSecondUtility(t *testing.T, e *utilityEnv, id string) string {
	t.Helper()
	remote := filepath.Join(t.TempDir(), id)
	write(t, filepath.Join(remote, "skills", id+"-skill", "SKILL.md"), "---\nname: "+id+"-skill\ndescription: test\n---\n")
	write(t, filepath.Join(remote, utilityManifestName), "schema = \"1\"\nname = \""+id+"\"\nskills = [\""+id+"-skill\"]\n")
	commitAll(t, remote)
	return remote
}

func trustedUtilities(t *testing.T, e *utilityEnv, ids ...string) {
	t.Helper()
	var remotes []string
	for _, id := range ids {
		remotes = append(remotes, addSecondUtility(t, e, id))
	}
	write(t, userConfigFile(e.home), "[trust]\nrepos = [\""+strings.Join(remotes, "\", \"")+"\"]\n")
	captureStd(t, func() {
		for i, id := range ids {
			if err := runAdd(addGlobalArgs(remotes[i], "--id", id)); err != nil {
				t.Fatal(err)
			}
		}
	})
}

func collect(t *testing.T, r jobRunner, jobs []tuiJob, onStart func(int)) []tea.Msg {
	t.Helper()
	ch := make(chan tea.Msg, 1024)
	r.start(jobs, func(msg tea.Msg) { ch <- msg })
	var msgs []tea.Msg
	deadline := time.After(60 * time.Second)
	for {
		select {
		case msg := <-ch:
			msgs = append(msgs, msg)
			if s, ok := msg.(jobStartedMsg); ok && onStart != nil {
				onStart(s.i)
			}
			if _, ok := msg.(jobsFinishedMsg); ok {
				return msgs
			}
		case <-deadline:
			t.Fatalf("runner did not finish: %+v", msgs)
		}
	}
}

func doneOf(msgs []tea.Msg) map[int]jobDoneMsg {
	out := map[int]jobDoneMsg{}
	for _, msg := range msgs {
		if d, ok := msg.(jobDoneMsg); ok {
			out[d.i] = d
		}
	}
	return out
}

func TestTUIRunBatchKeepsGoing(t *testing.T) {
	e := newUtilityEnv(t, "schema = \"1\"\nname = \"util\"\nskills = [\"util-skill\"]\n")
	trustedUtilities(t, e, "ua", "ub")
	actAsGap(t, e.repo)
	msgs := collect(t, &execRunner{}, []tuiJob{
		{label: "remove ua", args: []string{"remove", "ua"}},
		{label: "remove nope", args: []string{"remove", "nope"}},
		{label: "remove ub", args: []string{"remove", "ub"}},
	}, nil)
	done := doneOf(msgs)
	if len(done) != 3 || done[0].code != 0 || done[1].code == 0 || done[2].code != 0 {
		t.Fatalf("done = %+v", done)
	}
	var failure []string
	for _, msg := range msgs {
		if o, ok := msg.(jobOutputMsg); ok && o.i == 1 {
			failure = append(failure, o.line)
		}
	}
	if !strings.Contains(lastLine(failure), "utility nope is not added") {
		t.Fatalf("failure output = %q", failure)
	}
	list, _ := captureStd(t, func() { _ = listUtilities() })
	if strings.Contains(list, "ua") || strings.Contains(list, "ub") {
		t.Fatalf("removed utilities still listed:\n%s", list)
	}
	for _, id := range []string{"ua", "ub"} {
		backups, _ := filepath.Glob(filepath.Join(e.home, ".gap", "backups", "*", ".agents", "skills", id+"-skill"))
		if len(backups) != 1 {
			t.Fatalf("backup for %s = %v", id, backups)
		}
	}
}

func TestTUIRunCancelStopsTheRest(t *testing.T) {
	old := tuiExecutable
	t.Cleanup(func() { tuiExecutable = old })
	tuiExecutable = func() (string, error) { return "/bin/sleep", nil }
	r := &execRunner{}
	began := time.Now()
	msgs := collect(t, r, []tuiJob{{label: "one", args: []string{"30"}}, {label: "two", args: []string{"30"}}}, func(i int) {
		if i == 0 {
			go func() { time.Sleep(200 * time.Millisecond); r.cancel() }()
		}
	})
	done := doneOf(msgs)
	if len(done) != 1 || !done[0].cancelled || done[0].code == 0 {
		t.Fatalf("done = %+v", done)
	}
	if time.Since(began) > 10*time.Second {
		t.Fatal("cancel did not stop the running step")
	}
}

func TestTUIRunWaitReturnsAfterCancel(t *testing.T) {
	old := tuiExecutable
	t.Cleanup(func() { tuiExecutable = old })
	tuiExecutable = func() (string, error) { return "/bin/sleep", nil }
	r := &execRunner{}
	ch := make(chan tea.Msg, 1024)
	r.start([]tuiJob{{label: "one", args: []string{"30"}}, {label: "two", args: []string{"30"}}}, func(msg tea.Msg) { ch <- msg })
	select {
	case msg := <-ch:
		if s, ok := msg.(jobStartedMsg); !ok || s.i != 0 {
			t.Fatalf("first message = %#v", msg)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("first step never started")
	}
	r.cancel()
	waited := make(chan struct{})
	go func() { r.wait(); close(waited) }()
	select {
	case <-waited:
	case <-time.After(10 * time.Second):
		t.Fatal("wait did not return after cancel")
	}
	close(ch)
	var done []jobDoneMsg
	finished := false
	for msg := range ch {
		switch msg := msg.(type) {
		case jobDoneMsg:
			done = append(done, msg)
		case jobStartedMsg:
			t.Fatalf("step %d started after cancel", msg.i)
		case jobsFinishedMsg:
			finished = true
		}
	}
	if len(done) != 1 || done[0].i != 0 || !done[0].cancelled || done[0].code == 0 || !finished {
		t.Fatalf("before wait returned: done %+v finished %v", done, finished)
	}
}

func TestTUIStatusChildStopsOnClose(t *testing.T) {
	old := tuiExecutable
	t.Cleanup(func() { tuiExecutable = old })
	slow := filepath.Join(t.TempDir(), "gap")
	write(t, slow, "#!/bin/sh\nexec /bin/sleep 30\n")
	if err := os.Chmod(slow, 0o755); err != nil {
		t.Fatal(err)
	}
	tuiExecutable = func() (string, error) { return slow, nil }
	g := newChildGroup()
	got := make(chan tea.Msg, 1)
	go func() { got <- statusText(g, context.Background())() }()
	time.Sleep(300 * time.Millisecond)
	began := time.Now()
	g.close()
	if time.Since(began) > 10*time.Second {
		t.Fatal("close did not stop the status child")
	}
	select {
	case msg := <-got:
		if _, ok := msg.(statusTextMsg); !ok {
			t.Fatalf("status child never ran: %#v", msg)
		}
	case <-time.After(time.Second):
		t.Fatal("close returned before the status child exited")
	}
	if msg := statusText(g, context.Background())(); msg != nil {
		t.Fatalf("status after close = %#v", msg)
	}
}

func TestTUIStatusRequestStopsWhenCancelled(t *testing.T) {
	old := tuiExecutable
	t.Cleanup(func() { tuiExecutable = old })
	slow := filepath.Join(t.TempDir(), "gap")
	record := filepath.Join(t.TempDir(), "tmpdir")
	write(t, slow, "#!/bin/sh\necho \"$TMPDIR\" > \""+record+"\"\ntouch \"$TMPDIR/index.lock\"\nexec /bin/sleep 30\n")
	if err := os.Chmod(slow, 0o755); err != nil {
		t.Fatal(err)
	}
	tuiExecutable = func() (string, error) { return slow, nil }
	g := newChildGroup()
	t.Cleanup(g.close)
	req, stop := context.WithCancel(context.Background())
	got := make(chan tea.Msg, 1)
	go func() { got <- statusText(g, req)() }()
	time.Sleep(300 * time.Millisecond)
	stop()
	select {
	case msg := <-got:
		if msg != nil {
			t.Fatalf("a cancelled status request reported %#v", msg)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("cancelling the request did not stop gap status")
	}
	tmp := strings.TrimSpace(readString(t, record))
	if tmp == "" || filepath.Clean(tmp) == filepath.Clean(os.TempDir()) {
		t.Fatalf("gap status ran with the shared temp dir %q", tmp)
	}
	if _, err := os.Stat(tmp); !os.IsNotExist(err) {
		t.Fatalf("status temp dir %s left behind: %v", tmp, err)
	}
}

type cancelCounter struct{ cancels int }

func (c *cancelCounter) start([]tuiJob, func(tea.Msg)) {}

func (c *cancelCounter) cancel() { c.cancels++ }

func TestTUIRunCtrlCTwiceStopsAndQuits(t *testing.T) {
	m := newTUIModel(t.TempDir(), t.TempDir(), t.TempDir(), darkPalette)
	r := &cancelCounter{}
	m.runner = r
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m.run([]tuiJob{{label: "x", args: []string{"check"}}})
	ctrlC := tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}
	if ctrlC.String() != "ctrl+c" {
		t.Fatalf("ctrl+c renders as %q", ctrlC.String())
	}
	_, cmd := m.Update(ctrlC)
	if m.mode != modeAsk || r.cancels != 0 || m.quitting || cmd != nil {
		t.Fatalf("first ctrl+c: mode %v cancels %d quitting %v cmd %v", m.mode, r.cancels, m.quitting, cmd != nil)
	}
	_, cmd = m.Update(ctrlC)
	if r.cancels != 1 || !m.quitting || cmd == nil {
		t.Fatalf("second ctrl+c: cancels %d quitting %v cmd %v", r.cancels, m.quitting, cmd != nil)
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatal("second ctrl+c did not quit")
	}
}

func TestTUIChildArgsEndFlags(t *testing.T) {
	newUtilityEnv(t, "schema = \"1\"\nname = \"util\"\nskills = [\"util-skill\"]\n")
	var code int
	stdout, stderr := captureStd(t, func() { code = run([]string{"remove", "--", "-h"}) })
	if code != 1 || !strings.Contains(stdout+stderr, "utility -h is not added") {
		t.Fatalf("remove -- -h exited %d:\n%s%s", code, stdout, stderr)
	}
}

func TestTUIRunInProcessJob(t *testing.T) {
	ran := false
	msgs := collect(t, &execRunner{}, []tuiJob{{label: "fn", fn: func() error { ran = true; return nil }}}, nil)
	if d := doneOf(msgs); !ran || d[0].code != 0 {
		t.Fatalf("fn job: ran %v done %+v", ran, d)
	}
}

func TestTUIRunStatusMatchesGapStatus(t *testing.T) {
	e := newUtilityEnv(t, "schema = \"1\"\nname = \"util\"\nskills = [\"util-skill\"]\n")
	trustedUtilities(t, e, "ua")
	write(t, filepath.Join(e.home, ".agents", "skills", "ua-skill", "SKILL.md"), "edited\n")
	actAsGap(t, e.repo)
	want, _ := captureStd(t, func() { _ = statusAll() })
	got := statusText(newChildGroup(), context.Background())().(statusTextMsg)
	if got.err != nil || got.text != strings.TrimRight(want, "\n") {
		t.Fatalf("status tab text:\n%s\nwant:\n%s\nerr %v", got.text, want, got.err)
	}
	if !strings.Contains(got.text, "drift=1") {
		t.Fatalf("status lost drift:\n%s", got.text)
	}
}
