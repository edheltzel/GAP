package main

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

func TestReportPreviewFitsTerminal(t *testing.T) {
	const commit = "af85e07bf83baee26ffb79daa425eb84356b35b4"
	groups := []syncGroup{
		{"policy", "publish AGENTS.md to 6 harness files and 2 commands"},
		{"do-skills", "76 skills @ " + commit + " (af85e07)"},
		{"manifest", "17 skills from 12 sources in global/skills.json (skills.sh, needs npx)"},
	}
	for _, columns := range []int{40, 80, 120} {
		for _, selected := range []bool{false, true} {
			t.Run(fmt.Sprintf("%d/selected=%t", columns, selected), func(t *testing.T) {
				var buf bytes.Buffer
				out := &report{out: &buf, tty: true, width: columns}
				out.preview(groups, map[string]bool{"policy": true, "do-skills": selected, "manifest": selected})
				lines := strings.Split(strings.TrimSuffix(ansi.Strip(buf.String()), "\n"), "\n")
				var content strings.Builder
				for i, line := range lines {
					if width := lipgloss.Width(line); width > columns {
						t.Fatalf("plan row %d is %d columns in a %d-column terminal:\n%s", i, width, columns, buf.String())
					}
					if i == 0 || i == len(lines)-1 {
						continue
					}
					if !strings.HasPrefix(line, "│") || !strings.HasSuffix(line, "│") {
						t.Fatalf("plan row lost its frame: %q", line)
					}
					content.WriteString(strings.TrimSuffix(strings.TrimPrefix(line, "│"), "│"))
				}
				text := strings.Join(strings.Fields(content.String()), "")
				for _, want := range []string{"gap sync", "policy", "do-skills", "manifest", "publish AGENTS.md to 6 harness files and 2 commands", "76 skills", groups[2].Summary} {
					if !strings.Contains(text, strings.Join(strings.Fields(want), "")) {
						t.Fatalf("wrapped plan lost %q:\n%s", want, buf.String())
					}
				}
				if selected && !strings.Contains(text, commit+"(af85e07)") {
					t.Fatalf("wrapped plan lost the full pin:\n%s", buf.String())
				}
				if !selected && strings.Count(text, "skipped") != 2 {
					t.Fatalf("wrapped plan lost skipped groups:\n%s", buf.String())
				}
			})
		}
	}
}

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
