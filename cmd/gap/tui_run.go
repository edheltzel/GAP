package main

import (
	"bufio"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"
)

type tuiJob struct {
	label   string
	args    []string
	removes string
	fn      func() error
}

type jobStartedMsg struct{ i int }

type jobOutputMsg struct {
	i    int
	line string
}

type jobDoneMsg struct {
	i         int
	code      int
	err       error
	cancelled bool
}

type jobsFinishedMsg struct{}

type statusTextMsg struct {
	gen  int
	text string
	err  error
}

var (
	tuiExecutable = os.Executable
	tuiStopGrace  = 5 * time.Second
)

func gapCommand(ctx context.Context, args ...string) (*exec.Cmd, error) {
	exe, err := tuiExecutable()
	if err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(ctx, exe, args...)
	cmd.Env = append(os.Environ(), "NO_COLOR=1")
	ownGroup(cmd)
	cmd.Cancel = func() error { return interruptGroup(cmd) }
	cmd.WaitDelay = tuiStopGrace
	return cmd, nil
}

type jobRunner interface {
	start(jobs []tuiJob, send func(tea.Msg))
	cancel()
}

type execRunner struct {
	mu      sync.Mutex
	stop    context.CancelFunc
	stopped bool
	batch   sync.WaitGroup
}

func (r *execRunner) wait() {
	r.batch.Wait()
}

func (r *execRunner) cancel() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.stopped = true
	if r.stop != nil {
		r.stop()
	}
}

func (r *execRunner) start(jobs []tuiJob, send func(tea.Msg)) {
	r.mu.Lock()
	r.stopped = false
	r.mu.Unlock()
	r.batch.Add(1)
	go func() {
		defer r.batch.Done()
		for i, job := range jobs {
			r.mu.Lock()
			if r.stopped {
				r.mu.Unlock()
				break
			}
			ctx, stop := context.WithCancel(context.Background())
			r.stop = stop
			r.mu.Unlock()
			send(jobStartedMsg{i: i})
			code, err := runJob(ctx, i, job, send)
			r.mu.Lock()
			var exit *exec.ExitError
			signalled := errors.As(err, &exit) && exit.ExitCode() == -1
			cancelled := r.stopped || ctx.Err() != nil || signalled
			r.stop = nil
			r.mu.Unlock()
			stop()
			send(jobDoneMsg{i: i, code: code, err: err, cancelled: cancelled && code != 0})
		}
		send(jobsFinishedMsg{})
	}()
}

func runJob(ctx context.Context, i int, job tuiJob, send func(tea.Msg)) (int, error) {
	if job.fn != nil {
		if err := job.fn(); err != nil {
			send(jobOutputMsg{i: i, line: err.Error()})
			return 1, err
		}
		return 0, nil
	}
	cmd, err := gapCommand(ctx, job.args...)
	if err != nil {
		return 1, err
	}
	pr, pw := io.Pipe()
	cmd.Stdout, cmd.Stderr = pw, pw
	if err := cmd.Start(); err != nil {
		pw.Close()
		pr.Close()
		send(jobOutputMsg{i: i, line: err.Error()})
		return 1, err
	}
	done := make(chan struct{})
	go func() {
		lines := bufio.NewScanner(pr)
		lines.Buffer(make([]byte, 64*1024), 1024*1024)
		for lines.Scan() {
			send(jobOutputMsg{i: i, line: lines.Text()})
		}
		io.Copy(io.Discard, pr)
		close(done)
	}()
	err = cmd.Wait()
	pw.Close()
	<-done
	if ctx.Err() != nil {
		stopGroup(cmd.Process.Pid, tuiStopGrace)
	}
	var exit *exec.ExitError
	switch {
	case err == nil:
		return 0, nil
	case errors.As(err, &exit):
		return max(exit.ExitCode(), 1), err
	}
	return 1, err
}

type childGroup struct {
	mu     sync.Mutex
	ctx    context.Context
	stop   context.CancelFunc
	active sync.WaitGroup
}

func newChildGroup() *childGroup {
	ctx, stop := context.WithCancel(context.Background())
	return &childGroup{ctx: ctx, stop: stop}
}

func (g *childGroup) join() (context.Context, func(), bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.ctx.Err() != nil {
		return nil, nil, false
	}
	g.active.Add(1)
	return g.ctx, g.active.Done, true
}

func (g *childGroup) close() {
	g.mu.Lock()
	g.stop()
	g.mu.Unlock()
	g.active.Wait()
}

func statusText(g *childGroup) tea.Cmd {
	return func() tea.Msg {
		ctx, done, ok := g.join()
		if !ok {
			return nil
		}
		defer done()
		cmd, err := gapCommand(ctx, "status")
		if err != nil {
			return statusTextMsg{err: err}
		}
		out, err := cmd.CombinedOutput()
		if ctx.Err() != nil && cmd.Process != nil {
			stopGroup(cmd.Process.Pid, tuiStopGrace)
		}
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			err = nil
		}
		return statusTextMsg{text: strings.TrimRight(string(out), "\n"), err: err}
	}
}
