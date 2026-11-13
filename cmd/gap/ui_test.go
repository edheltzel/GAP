package main

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

var listRows = [][]string{
	{"do-skills", "distro", "edheltzel/do-skills", "master", "bb80228", "run"},
	{"mine", "user", "someone/mine", "v1", "unpinned", "ask"},
}

func ttyReport(buf *bytes.Buffer) *report {
	return &report{out: buf, tty: true, width: 120}
}

func assertFramed(t *testing.T, got string, headers []string) {
	t.Helper()
	plain := ansi.Strip(got)
	for _, want := range []string{"╭", "╰", "├"} {
		if !strings.Contains(plain, want) {
			t.Fatalf("table lost its %q border:\n%s", want, plain)
		}
	}
	for _, h := range headers {
		if !strings.Contains(plain, "│ "+h) {
			t.Fatalf("table lost header %q:\n%s", h, plain)
		}
	}
}

func TestListTerminalTable(t *testing.T) {
	var buf bytes.Buffer
	if err := printList(ttyReport(&buf), listRows); err != nil {
		t.Fatal(err)
	}
	assertFramed(t, buf.String(), []string{"id", "source", "repo", "ref", "commit", "hooks"})
	plain := ansi.Strip(buf.String())
	if !strings.HasPrefix(plain, "gap list") || !strings.HasSuffix(plain, "2 total\n") || !strings.Contains(plain, "edheltzel/do-skills") {
		t.Fatalf("terminal list = \n%s", plain)
	}
}

func TestListPipedUnchanged(t *testing.T) {
	var buf bytes.Buffer
	if err := printList(newReport(&buf), listRows); err != nil {
		t.Fatal(err)
	}
	want := "ID         SOURCE  REPO                 REF     COMMIT    HOOKS\n" +
		"do-skills  distro  edheltzel/do-skills  master  bb80228   run\n" +
		"mine       user    someone/mine         v1      unpinned  ask\n"
	if buf.String() != want {
		t.Fatalf("piped list = %q, want %q", buf.String(), want)
	}
}

func TestSearchTerminalTable(t *testing.T) {
	rows := [][]string{{"echo", "Local voice notifications", "macos, voice", "included with GAP; gap sync installs it"}}
	var buf bytes.Buffer
	printSearch(ttyReport(&buf), "voice", rows)
	assertFramed(t, buf.String(), []string{"id", "description", "tags", "add"})
	plain := ansi.Strip(buf.String())
	if !strings.HasPrefix(plain, `gap search  "voice"`) || !strings.HasSuffix(plain, "1 match\n") {
		t.Fatalf("terminal search = \n%s", plain)
	}
	buf.Reset()
	printSearch(ttyReport(&buf), "", append(rows, rows[0]))
	if plain := ansi.Strip(buf.String()); !strings.Contains(plain, "utility catalog") || !strings.HasSuffix(plain, "2 matches\n") {
		t.Fatalf("terminal search without a query = \n%s", plain)
	}
	buf.Reset()
	printSearch(newReport(&buf), "voice", rows)
	if want := "echo - Local voice notifications\n  included with GAP; gap sync installs it\n"; buf.String() != want {
		t.Fatalf("piped search = %q, want %q", buf.String(), want)
	}
}

func TestStatusTerminalSummary(t *testing.T) {
	rows := []statusRow{
		{id: "do-skills", ref: "master", commit: "bb80228", store: storeOK, drifted: 1, skills: [][]string{{"do-todo", "drift=2"}, {"do-commit", "drift=0"}}},
		{id: "echo", ref: "v0.13.0", commit: "f800126", store: storeOK, skills: [][]string{{"echo-voice", "drift=0"}}},
		{id: "recall", ref: "main", commit: "unpinned", store: storeUnpinned},
	}
	var buf bytes.Buffer
	ttyReport(&buf).statusSummary(rows)
	assertFramed(t, buf.String(), []string{"id", "ref", "commit", "store", "drift", "skill"})
	plain := ansi.Strip(buf.String())
	if !strings.HasPrefix(plain, "gap status") || !strings.HasSuffix(plain, "3 components, 1 with drift, 1 store problem\n") {
		t.Fatalf("terminal status = \n%s", plain)
	}
	if !strings.Contains(plain, "do-todo") || strings.Contains(plain, "do-commit") || strings.Contains(plain, "echo-voice") {
		t.Fatalf("terminal status must list only drifted skills:\n%s", plain)
	}
	if strings.Count(plain, "╭") != 2 {
		t.Fatalf("terminal status needs one summary table and one drift table:\n%s", plain)
	}
}

func TestNoColorDisablesColor(t *testing.T) {
	for _, env := range [][2]string{{"NO_COLOR", "1"}, {"TERM", "dumb"}} {
		t.Run(env[0], func(t *testing.T) {
			t.Setenv("NO_COLOR", "")
			t.Setenv("TERM", "xterm-256color")
			var colored bytes.Buffer
			printList(ttyReport(&colored), listRows)
			if !strings.Contains(colored.String(), "\x1b[") {
				t.Fatalf("terminal list carried no color:\n%q", colored.String())
			}
			t.Setenv(env[0], env[1])
			var buf bytes.Buffer
			out := ttyReport(&buf)
			out.plain = plainEnv()
			printList(out, listRows)
			out.statusSummary([]statusRow{{id: "x", ref: "m", commit: "abc", store: storeOK, drifted: 1, skills: [][]string{{"s", "drift=1"}}}})
			out.done(true, "ok")
			if strings.Contains(buf.String(), "\x1b") {
				t.Fatalf("%s=%s still colored:\n%q", env[0], env[1], buf.String())
			}
			assertFramed(t, buf.String(), []string{"id"})
		})
	}
}

func TestSpinnerOnlyOnTerminal(t *testing.T) {
	prev := spinnerOut
	t.Cleanup(func() { spinnerOut = prev })
	for _, tty := range []bool{false, true} {
		var buf bytes.Buffer
		spinnerOut = func() (io.Writer, bool) { return &buf, tty }
		boom := errors.New("boom")
		got, err := spin("Fetching x", func() (string, error) { return "v", boom })
		if got != "v" || err != boom {
			t.Fatalf("spin returned %q, %v", got, err)
		}
		if !tty && buf.Len() != 0 {
			t.Fatalf("piped spinner printed %q", buf.String())
		}
		if tty && (!strings.Contains(buf.String(), "Fetching x") || !strings.HasSuffix(buf.String(), "\r\x1b[2K")) {
			t.Fatalf("terminal spinner = %q", buf.String())
		}
	}
}
