package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitAdoptWritesOnlyNewFiles(t *testing.T) {
	t.Setenv("PATH", "/usr/bin:/bin")
	repo := t.TempDir()
	write(t, filepath.Join(repo, "AGENTS.md"), "# Existing\n\nKeep this.\n")
	write(t, filepath.Join(repo, "todo.txt"), "task\n")
	if err := os.MkdirAll(filepath.Join(repo, "forge"), 0o755); err != nil {
		t.Fatal(err)
	}
	if code := run([]string{"init", "--adopt", repo}); code != 0 {
		t.Fatalf("init: %d", code)
	}
	if got := readString(t, filepath.Join(repo, ".gap", "gap.toml")); !strings.Contains(got, "slug = ") {
		t.Fatalf("gap.toml: %q", got)
	}
	agents := readString(t, filepath.Join(repo, "AGENTS.md"))
	if !strings.HasPrefix(agents, "# Existing\n") || !strings.Contains(agents, "<!-- gap:begin -->") {
		t.Fatalf("AGENTS.md was rewritten:\n%s", agents)
	}
	before := agents
	if code := run([]string{"init", "--adopt", repo}); code != 0 {
		t.Fatalf("second init: %d", code)
	}
	if got := readString(t, filepath.Join(repo, "AGENTS.md")); got != before {
		t.Fatal("a second init changed AGENTS.md")
	}
}

func TestInitOnEmptyRepo(t *testing.T) {
	t.Setenv("PATH", "/usr/bin:/bin")
	repo := t.TempDir()
	if code := run([]string{"init", repo}); code != 0 {
		t.Fatalf("init: %d", code)
	}
	if _, err := os.Stat(filepath.Join(repo, "AGENTS.md")); err != nil {
		t.Fatal("init did not create AGENTS.md")
	}
	if _, err := os.Stat(filepath.Join(repo, ".gap", "gap.toml")); err != nil {
		t.Fatal("init did not create .gap/gap.toml")
	}
}

// recallStub records every invocation as one line of argv, joined by spaces.
func recallStub(t *testing.T, body string) {
	t.Helper()
	bin := t.TempDir()
	write(t, filepath.Join(bin, "recall"), "#!/bin/sh\necho \"$*\" >> \"$RECALL_LOG\"\n"+body)
	if err := os.Chmod(filepath.Join(bin, "recall"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("RECALL_LOG", filepath.Join(t.TempDir(), "log"))
}

func recallArgv(t *testing.T) []string {
	t.Helper()
	b, err := os.ReadFile(os.Getenv("RECALL_LOG"))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	return strings.Split(strings.TrimRight(string(b), "\n"), "\n")
}

func TestInitBreadcrumb(t *testing.T) {
	recallStub(t, "exit 0\n")
	repo := t.TempDir()
	if code := run([]string{"init", repo}); code != 0 {
		t.Fatalf("init: %d", code)
	}
	slug := filepath.Base(repo)
	got := recallArgv(t)
	want := []string{
		"recent breadcrumbs -p " + slug + " -l 1000",
		"add breadcrumb gap provisioned " + slug + " -p " + slug + " -c setup",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("recall argv:\n%s", strings.Join(got, "\n"))
	}
}

func TestInitBreadcrumbIdempotent(t *testing.T) {
	slug := ""
	repo := t.TempDir()
	slug = filepath.Base(repo)
	recallStub(t, "case \"$1\" in\nrecent) echo \"#1 ["+slug+"] 2026-10-23\"; echo \"  gap provisioned "+slug+"\";;\nesac\nexit 0\n")
	if code := run([]string{"init", repo}); code != 0 {
		t.Fatalf("init: %d", code)
	}
	got := recallArgv(t)
	if len(got) != 1 || !strings.HasPrefix(got[0], "recent breadcrumbs -p "+slug+" ") {
		t.Fatalf("a second breadcrumb must not be added:\n%v", got)
	}
}

func TestInitBreadcrumbNoRecall(t *testing.T) {
	t.Setenv("PATH", "/usr/bin:/bin")
	repo := t.TempDir()
	out, _ := captureStd(t, func() {
		if code := run([]string{"init", repo}); code != 0 {
			t.Fatalf("init without recall: %d", code)
		}
	})
	if !strings.Contains(out, "recall is not installed; no breadcrumb written") {
		t.Fatalf("missing notice:\n%s", out)
	}
	if _, err := os.Stat(filepath.Join(repo, ".gap", "gap.toml")); err != nil {
		t.Fatal("init did not write .gap/gap.toml when recall is missing")
	}
}
