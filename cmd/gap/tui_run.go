package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
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
	gen, req int
	text     string
	err      error
}

var (
	tuiExecutable = os.Executable
	tuiStopGrace  = 5 * time.Second
)

type childCmd struct {
	*exec.Cmd
	mu   sync.Mutex
	kill *time.Timer
}

func gapCommand(ctx context.Context, args ...string) (*childCmd, error) {
	exe, err := tuiExecutable()
	if err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(ctx, exe, args...)
	cmd.Env = append(os.Environ(), "NO_COLOR=1")
	ownGroup(cmd)
	c := &childCmd{Cmd: cmd}
	cmd.Cancel = c.interrupt
	cmd.WaitDelay = tuiStopGrace + time.Second
	return c, nil
}

func (c *childCmd) interrupt() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.kill != nil || c.Process == nil {
		return nil
	}
	pgid := c.Process.Pid
	c.kill = time.AfterFunc(tuiStopGrace, func() { killGroup(pgid) })
	return interruptGroup(pgid)
}

func (c *childCmd) settle(cancelled bool) bool {
	c.interrupt()
	c.mu.Lock()
	kill := c.kill
	c.mu.Unlock()
	pgid := c.Process.Pid
	gone := groupGone(pgid, tuiStopGrace+time.Second)
	if !gone {
		killGroup(pgid)
		gone = groupGone(pgid, 2*time.Second)
	}
	if kill != nil {
		kill.Stop()
	}
	return gone
}

type jobRunner interface {
	start(jobs []tuiJob, send func(tea.Msg))
	cancel()
}

type execRunner struct {
	mu      sync.Mutex
	stop    context.CancelFunc
	child   *childCmd
	stopped bool
	batch   sync.WaitGroup
}

func (r *execRunner) wait() {
	r.batch.Wait()
}

func (r *execRunner) cancel() {
	r.mu.Lock()
	r.stopped = true
	if r.stop != nil {
		r.stop()
	}
	child := r.child
	r.mu.Unlock()
	if child != nil {
		child.interrupt()
	}
}

func (r *execRunner) started(c *childCmd) {
	r.mu.Lock()
	r.child = c
	stopped := r.stopped
	r.mu.Unlock()
	if stopped {
		c.interrupt()
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
			child, code, err := runJob(ctx, i, job, send, r.started)
			r.mu.Lock()
			var exit *exec.ExitError
			signalled := errors.As(err, &exit) && exit.ExitCode() == -1
			cancelled := r.stopped || ctx.Err() != nil || signalled
			r.stop, r.child = nil, nil
			r.mu.Unlock()
			lingering := false
			if child != nil {
				lingering = !child.settle(cancelled)
			}
			stop()
			if lingering && !cancelled {
				code = 1
				err = fmt.Errorf("a hook process kept running after the command exited")
			}
			send(jobDoneMsg{i: i, code: code, err: err, cancelled: cancelled && code != 0})
		}
		send(jobsFinishedMsg{})
	}()
}

func runJob(ctx context.Context, i int, job tuiJob, send func(tea.Msg), started func(*childCmd)) (*childCmd, int, error) {
	if job.fn != nil {
		if err := job.fn(); err != nil {
			send(jobOutputMsg{i: i, line: err.Error()})
			return nil, 1, err
		}
		return nil, 0, nil
	}
	cmd, err := gapCommand(ctx, job.args...)
	if err != nil {
		return nil, 1, err
	}
	pr, pw := io.Pipe()
	cmd.Stdout, cmd.Stderr = pw, pw
	if err := cmd.Start(); err != nil {
		pw.Close()
		pr.Close()
		send(jobOutputMsg{i: i, line: err.Error()})
		return nil, 1, err
	}
	started(cmd)
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
	var exit *exec.ExitError
	switch {
	case err == nil:
		return cmd, 0, nil
	case errors.As(err, &exit):
		return cmd, max(exit.ExitCode(), 1), err
	}
	return cmd, 1, err
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

func statusText(g *childGroup, req context.Context) tea.Cmd {
	return func() tea.Msg {
		group, done, ok := g.join()
		if !ok {
			return nil
		}
		defer done()
		ctx, cancel := context.WithCancel(group)
		defer cancel()
		defer context.AfterFunc(req, cancel)()
		tmp, err := os.MkdirTemp("", "gap-status-")
		if err != nil {
			return statusTextMsg{err: err}
		}
		defer os.RemoveAll(tmp)
		cmd, err := gapCommand(ctx, "status")
		if err != nil {
			return statusTextMsg{err: err}
		}
		cmd.Env = append(cmd.Env, "TMPDIR="+tmp)
		out, err := cmd.CombinedOutput()
		cmd.settle(ctx.Err() != nil)
		if req.Err() != nil {
			return nil
		}
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			err = nil
		}
		return statusTextMsg{text: strings.TrimRight(string(out), "\n"), err: err}
	}
}
