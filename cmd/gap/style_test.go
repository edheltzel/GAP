package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestReportStaysPlainWhenPiped(t *testing.T) {
	var buf bytes.Buffer
	out := newReport(&buf)
	out.statusLine("recall", "main", "4319176", "changed")
	out.driftTable([][]string{{"do-todo", "drift=1"}})
	out.doctorTable([][]string{{"claude", "absent; install this harness separately if needed"}})
	out.next("gap sync    fetch each missing store and install it")
	got := buf.String()
	for _, want := range []string{
		"recall ref=main commit=4319176 store=changed\n",
		"  do-todo drift=1\n",
		"claude: absent; install this harness separately if needed\n",
		"Next\n  gap sync    fetch each missing store and install it\n",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("piped report lost %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "\x1b") {
		t.Fatalf("piped report carried color:\n%q", got)
	}
}

func TestReportTerminalTables(t *testing.T) {
	for _, doctor := range []bool{false, true} {
		var buf bytes.Buffer
		out := &report{out: &buf, tty: true}
		if doctor {
			out.doctorTable([][]string{{"claude", "present"}})
		} else {
			out.driftTable([][]string{{"do-todo", "drift=1"}})
		}
		want := []string{"skill", "drift", "do-todo", "drift=1"}
		if doctor {
			want = []string{"name", "detail", "claude", "present"}
		}
		for _, text := range want {
			if !strings.Contains(buf.String(), text) {
				t.Errorf("terminal table omitted %q: %s", text, buf.String())
			}
		}
	}
}
