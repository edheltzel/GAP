package main

import (
	"context"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

func TestTUISearch(t *testing.T) {
	project := t.TempDir()
	m := newTUIModel(t.TempDir(), t.TempDir(), project, darkPalette)
	var queries []string
	m.findSkills = func(_ context.Context, q string, _ int) ([]skillHit, error) {
		queries = append(queries, q)
		return []skillHit{
			{Source: "vercel-labs/agent-skills", Skill: "react", Installs: 932043, Trusted: true},
			{Source: "pbakaus/agent-reviews", Skill: "agent-reviews", Installs: 1200},
		}, nil
	}
	var handed [][]string
	var dirs []string
	m.handOff = func(dir string, args []string) tea.Cmd {
		dirs, handed = append(dirs, dir), append(handed, args)
		return nil
	}
	m.Update(tea.WindowSizeMsg{Width: 110, Height: 34})
	m.Update(loadedMsg{items: []itemRow{{ID: "agent-reviews", Source: "distro", Scope: "global", Repo: "pbakaus/agent-reviews", Skills: [][]string{{"agent-reviews", "drift=0"}}}}})
	typeKeys(m, "/")
	if m.mode != modeSearch {
		t.Fatalf("/ did not open search, mode %v", m.mode)
	}
	typeKeys(m, "rea", "ct")
	for seq := 1; seq <= m.search.seq; seq++ {
		drive(m, searchTickMsg{seq: seq})
	}
	if !slices.Equal(queries, []string{"react"}) {
		t.Fatalf("queries %q, want one search after the pause", queries)
	}
	view := ansi.Strip(m.render())
	for _, want := range []string{"/ react", "\u2713 react", "vercel-labs/agent-skills", "932.0K", "agent-reviews", "installed global", "enter install"} {
		if !strings.Contains(view, want) {
			t.Fatalf("search view lacks %q:\n%s", want, view)
		}
	}
	typeKeys(m, "enter")
	if m.mode != modeForm || !strings.Contains(ansi.Strip(m.render()), "Install react") {
		t.Fatalf("enter did not ask where to install:\n%s", ansi.Strip(m.render()))
	}
	typeKeys(m, "down", "enter")
	if len(handed) != 1 || strings.Join(handed[0], " ") != "add vercel-labs/agent-skills --skill react --project" || dirs[0] != project {
		t.Fatalf("handed %q in %q", handed, dirs)
	}
	typeKeys(m, "/", "x", "esc")
	if m.mode != modeBrowse || m.search != nil {
		t.Fatalf("esc did not close search")
	}
}
