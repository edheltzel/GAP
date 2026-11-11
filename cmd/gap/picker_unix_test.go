//go:build unix

package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"
)

func pickerHelper() {
	fmt.Println("ready")
	sel, err := runPicker([]syncGroup{{"policy", "publish policy"}, {"owned", "run installer"}}, map[string]bool{"policy": true, "owned": true}, struct{ io.Reader }{os.Stdin}, io.Discard)
	switch {
	case errors.Is(err, errSyncCancelled):
		fmt.Println("cancelled")
	case err != nil:
		fmt.Println("error", err)
	default:
		fmt.Println("accepted", sel["policy"], sel["owned"])
	}
	os.Exit(0)
}

func TestPickerAcceptsOnlySubmission(t *testing.T) {
	for _, tc := range []struct {
		name string
		end  func(*exec.Cmd, io.WriteCloser) error
		want string
	}{
		{"enter submits", func(_ *exec.Cmd, in io.WriteCloser) error { _, err := io.WriteString(in, "\r"); return err }, "accepted true true"},
		{"sigterm cancels", func(c *exec.Cmd, _ io.WriteCloser) error { return c.Process.Signal(syscall.SIGTERM) }, "cancelled"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=^$")
			cmd.Env = append(os.Environ(), "GAP_TEST_PICKER=1", "TERM=xterm-256color")
			in, err := cmd.StdinPipe()
			if err != nil {
				t.Fatal(err)
			}
			out, err := cmd.StdoutPipe()
			if err != nil {
				t.Fatal(err)
			}
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = cmd.Process.Kill() })
			lines := bufio.NewScanner(out)
			if !lines.Scan() || lines.Text() != "ready" {
				t.Fatalf("helper did not start: %q", lines.Text())
			}
			time.Sleep(500 * time.Millisecond)
			if err := tc.end(cmd, in); err != nil {
				t.Fatal(err)
			}
			done := make(chan string, 1)
			go func() {
				var rest []string
				for lines.Scan() {
					rest = append(rest, lines.Text())
				}
				done <- strings.Join(rest, "\n")
			}()
			select {
			case got := <-done:
				_ = cmd.Wait()
				if got != tc.want {
					t.Fatalf("picker result = %q, want %q", got, tc.want)
				}
			case <-time.After(10 * time.Second):
				t.Fatal("picker did not exit")
			}
		})
	}
}
