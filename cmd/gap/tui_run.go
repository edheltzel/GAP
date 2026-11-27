package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"image/color"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"
)

type tuiJob struct {
	label   string
	name    string
	color   color.Color
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
	mu        sync.Mutex
	kill      *time.Timer
	out       io.Closer
	in        io.Closer
	escalated bool
	done      bool
	signalled bool
}

type handedBackMsg struct {
	err    error
	edited string
	note   string
	after  func(error) string
}

// handoff runs a gap command on the real terminal, so its picker, progress bar, and prompts work, then waits for enter.
type handoff struct {
	cmd      *exec.Cmd
	in       io.Reader
	out, err io.Writer
	after    func(error) string
	note     string
	ran      bool
}

func (h *handoff) SetStdin(r io.Reader)  { h.in = r }
func (h *handoff) SetStdout(w io.Writer) { h.out = w }
func (h *handoff) SetStderr(w io.Writer) { h.err = w }

// Run gives the terminal to the command. ctrl+c reaches the command but not gap tui, so after
// always runs, before the enter prompt, even when the command is interrupted.
func (h *handoff) Run() error {
	h.ran = true
	h.cmd.Stdin, h.cmd.Stdout, h.cmd.Stderr = h.in, h.out, h.err
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)
	runErr := h.cmd.Run()
	signal.Stop(sigs)
	if h.after != nil {
		h.note = h.after(runErr)
	}
	fmt.Fprint(h.out, "\nPress enter to return to gap tui.")
	buf := make([]byte, 1)
	for {
		if n, err := h.in.Read(buf); err != nil || (n == 1 && (buf[0] == '\n' || buf[0] == '\r')) {
			break
		}
	}
	return runErr
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
	c.signalled = true
	c.kill = time.AfterFunc(tuiStopGrace, func() {
		c.mu.Lock()
		defer c.mu.Unlock()
		if c.done {
			return
		}
		c.escalated = true
		killGroup(pgid)
	})
	return interruptGroup(pgid)
}

func (c *childCmd) wasEscalated() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.escalated
}

func (c *childCmd) settle(cancelled bool) (gone, forced, signalled bool) {
	if c.Process == nil {
		return true, false, false
	}
	pgid := c.Process.Pid
	if groupGone(pgid, 0) {
		c.disarm()
		return true, false, false
	}
	c.interrupt()
	gone = groupGone(pgid, tuiStopGrace)
	if !gone {
		c.mu.Lock()
		c.escalated = true
		c.mu.Unlock()
		killGroup(pgid)
		gone = groupGone(pgid, 2*time.Second)
	}
	c.disarm()
	c.mu.Lock()
	signalled = c.signalled
	c.mu.Unlock()
	return gone, c.wasEscalated(), signalled
}

func (c *childCmd) closeIn() {
	c.mu.Lock()
	in := c.in
	c.in = nil
	c.mu.Unlock()
	if in != nil {
		in.Close()
	}
}

func (c *childCmd) disarm() {
	c.mu.Lock()
	c.done = true
	kill := c.kill
	c.kill = nil
	c.mu.Unlock()
	if kill != nil {
		kill.Stop()
	}
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
			child, code, err, drained := runJob(ctx, i, job, send, r.started)
			r.mu.Lock()
			var exit *exec.ExitError
			signalled := errors.As(err, &exit) && exit.ExitCode() == -1
			cancelled := r.stopped || ctx.Err() != nil || signalled
			r.stop, r.child = nil, nil
			r.mu.Unlock()
			lingering, forced, signalledGroup := false, false, false
			if child != nil {
				var gone bool
				gone, forced, signalledGroup = child.settle(cancelled)
				lingering = !gone
			}
			stop()
			if drained != nil {
				select {
				case <-drained:
				case <-time.After(2 * time.Second):
					child.closeIn()
					<-drained
				}
			}
			if !cancelled && (forced || lingering || signalledGroup) {
				code = 1
				why := "was interrupted"
				if forced {
					why = "had to be killed"
				}
				if lingering {
					why = "could not be killed"
				}
				err = fmt.Errorf("a hook process kept running after the command exited and %s", why)
			}
			send(jobDoneMsg{i: i, code: code, err: err, cancelled: cancelled && code != 0})
		}
		send(jobsFinishedMsg{})
	}()
}

func runJob(ctx context.Context, i int, job tuiJob, send func(tea.Msg), started func(*childCmd)) (*childCmd, int, error, <-chan struct{}) {
	if job.fn != nil {
		if err := job.fn(); err != nil {
			send(jobOutputMsg{i: i, line: err.Error()})
			return nil, 1, err, nil
		}
		return nil, 0, nil, nil
	}
	cmd, err := gapCommand(ctx, job.args...)
	if err != nil {
		return nil, 1, err, nil
	}
	pr, pw, err := os.Pipe()
	if err != nil {
		return nil, 1, err, nil
	}
	cmd.Stdout, cmd.Stderr = pw, pw
	cmd.out, cmd.in = pw, pr
	if err := cmd.Start(); err != nil {
		pw.Close()
		pr.Close()
		send(jobOutputMsg{i: i, line: err.Error()})
		return nil, 1, err, nil
	}
	pw.Close()
	started(cmd)
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer pr.Close()
		lines := bufio.NewScanner(pr)
		lines.Buffer(make([]byte, 64*1024), 1024*1024)
		for lines.Scan() {
			send(jobOutputMsg{i: i, line: lines.Text()})
		}
		io.Copy(io.Discard, pr)
	}()
	err = cmd.Wait()
	var exit *exec.ExitError
	switch {
	case err == nil:
		return cmd, 0, nil, done
	case errors.As(err, &exit):
		return cmd, max(exit.ExitCode(), 1), err, done
	}
	return cmd, 1, err, done
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
		out, err := runStatus(cmd)
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

// runStatus runs gap status. A terminal shows the framed table, so the child gets one when the
// platform can hand it a pty; the pipe fallback is what a non-terminal status prints.
func runStatus(cmd *childCmd) ([]byte, error) {
	if out, err, ok := runOnPty(cmd); ok {
		return out, err
	}
	return cmd.CombinedOutput()
}
