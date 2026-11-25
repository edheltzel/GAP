package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

func TestSetComponentFieldsKeepsOtherLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gap.toml")
	write(t, path, "# mine\n[components.echo]\nref = \"main\" # keep\ncolor = \"#111111\"\n\n[components.other]\nenabled = false\n")
	if err := setComponentFields(path, "echo", []configField{{"enabled", false}, {"pin", true}, {"color", nil}}); err != nil {
		t.Fatal(err)
	}
	if err := setComponentFields(path, "new", []configField{{"color", "#5FD7A7"}}); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(path)
	want := "# mine\n[components.echo]\npin = true\nenabled = false\nref = \"main\" # keep\n\n[components.other]\nenabled = false\n\n[components.new]\ncolor = \"#5FD7A7\"\n"
	if string(b) != want {
		t.Fatalf("got\n%s\nwant\n%s", b, want)
	}
	if err := setComponentFields(path, "echo", []configField{{"color", 3}}); err == nil {
		t.Fatal("non-string value was written")
	}
}

func drive(m *tuiModel, msg tea.Msg) {
	queue := []tea.Msg{msg}
	for len(queue) > 0 {
		next := queue[0]
		queue = queue[1:]
		_, cmd := m.Update(next)
		queue = append(queue, runQuick(cmd)...)
	}
}

func runQuick(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	got := make(chan tea.Msg, 1)
	go func() { got <- cmd() }()
	select {
	case msg := <-got:
		if batch, ok := msg.(tea.BatchMsg); ok {
			var out []tea.Msg
			for _, c := range batch {
				out = append(out, runQuick(c)...)
			}
			return out
		}
		if msg == nil {
			return nil
		}
		switch msg.(type) {
		case loadedMsg, statusTextMsg:
			return nil
		}
		return []tea.Msg{msg}
	case <-time.After(30 * time.Millisecond):
		return nil
	}
}

func typeKeys(m *tuiModel, keys ...string) {
	for _, k := range keys {
		var msg tea.KeyPressMsg
		switch k {
		case "enter":
			msg = tea.KeyPressMsg{Code: tea.KeyEnter}
		case "down":
			msg = tea.KeyPressMsg{Code: tea.KeyDown}
		case "left":
			msg = tea.KeyPressMsg{Code: tea.KeyLeft}
		case "esc":
			msg = tea.KeyPressMsg{Code: tea.KeyEscape}
		case "backspace":
			msg = tea.KeyPressMsg{Code: tea.KeyBackspace}
		default:
			for _, r := range k {
				drive(m, tea.KeyPressMsg{Code: r, Text: string(r)})
			}
			continue
		}
		drive(m, msg)
	}
}

func TestTUIEditSavesToProject(t *testing.T) {
	home, project := t.TempDir(), t.TempDir()
	m := newTUIModel(home, t.TempDir(), project, darkPalette)
	m.runner = &fakeRunner{}
	m.loadQuick = func() loadedMsg { return loadedMsg{} }
	m.fetchStatus = func(context.Context) tea.Cmd { return nil }
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m.Update(loadedMsg{items: []itemRow{{ID: "echo", Source: "distro", Scope: "global", Ref: "main"}}})
	typeKeys(m, "e")
	if m.mode != modeForm || !strings.Contains(ansi.Strip(m.render()), "Edit echo") {
		t.Fatalf("e did not open the edit form:\n%s", ansi.Strip(m.render()))
	}
	typeKeys(m, "down", "enter", "left", "enter", "enter", "#ff79c6", "enter")
	if m.mode != modeBrowse {
		t.Fatalf("form still open:\n%s", ansi.Strip(m.render()))
	}
	b, err := os.ReadFile(projectConfigFile(project))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"profile = \"project\"", "[components.echo]", "enabled = false", "color = \"#FF79C6\""} {
		if !strings.Contains(string(b), want) {
			t.Fatalf("project gap.toml lacks %q:\n%s", want, b)
		}
	}
	if _, err := os.Stat(userConfigFile(home)); !os.IsNotExist(err) {
		t.Fatalf("global gap.toml was written: %v", err)
	}
	if !strings.Contains(m.note, filepath.Join(filepath.Base(project), ".gap", "gap.toml")) {
		t.Fatalf("note %q does not name the file", m.note)
	}
	typeKeys(m, "e", "esc")
	if m.mode != modeBrowse || m.note != "Nothing changed." {
		t.Fatalf("esc did not cancel: mode %v note %q", m.mode, m.note)
	}
}

func TestTUIAddFormHandsOverToGapAdd(t *testing.T) {
	m := newTUIModel(t.TempDir(), t.TempDir(), "", darkPalette)
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m.Update(loadedMsg{})
	typeKeys(m, "a")
	if m.mode != modeForm || !strings.Contains(ansi.Strip(m.render()), "Add a utility") {
		t.Fatalf("a did not open the add form:\n%s", ansi.Strip(m.render()))
	}
	typeKeys(m, "enter")
	if !strings.Contains(ansi.Strip(m.render()), "enter a repo") {
		t.Fatalf("empty repo accepted:\n%s", ansi.Strip(m.render()))
	}
}
