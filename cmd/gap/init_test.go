package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
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
	if agents != "# Existing\n\nKeep this.\n" || readString(t, filepath.Join(repo, "todo.txt")) != "task\n" {
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
	for _, adopt := range []bool{false, true} {
		repo := t.TempDir()
		args := []string{"init"}
		if adopt {
			args = append(args, "--adopt")
		}
		if code := run(append(args, repo)); code != 0 {
			t.Fatalf("init: %d", code)
		}
		if _, err := os.Stat(filepath.Join(repo, "AGENTS.md")); err != nil {
			t.Fatal("init did not create AGENTS.md")
		}
		if _, err := os.Stat(filepath.Join(repo, ".gap", "gap.toml")); err != nil {
			t.Fatal("init did not create .gap/gap.toml")
		}
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
		"search \"gap provisioned " + slug + "\" -t breadcrumbs -p " + slug + " --include-duplicates -l 1",
		"add breadcrumb gap provisioned " + slug + " -p " + slug + " -c setup",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("recall argv:\n%s", strings.Join(got, "\n"))
	}
}

func TestInitBreadcrumbIdempotent(t *testing.T) {
	repo := t.TempDir()
	config := "profile = \"project\"\nslug = \"ledger-project\"\n"
	write(t, filepath.Join(repo, ".gap", "gap.toml"), config)
	t.Setenv("RECALL_HISTORY", filepath.Join(t.TempDir(), "history"))
	recallStub(t, `case "$1" in
search)
  [ "$2" = '"gap provisioned ledger-project"' ] || exit 64
  [ "$3 $4 $5 $6 $7 $8 $9" = '-t breadcrumbs -p ledger-project --include-duplicates -l 1' ] || exit 64
  if [ -f "$RECALL_HISTORY" ] && /usr/bin/grep -Fxq 'gap provisioned ledger-project' "$RECALL_HISTORY"; then
    printf 'Found 1 result(s):\n\n[breadcrumbs#1] [ledger-project] 2026-10-23\n  gap provisioned ledger-project\n'
  else
    printf 'No results found.\n'
  fi
  ;;
recent) if [ -f "$RECALL_HISTORY" ]; then /usr/bin/tail -n 1000 "$RECALL_HISTORY"; fi ;;
add) printf '%s\n' "$3" >> "$RECALL_HISTORY" ;;
*) exit 64 ;;
esac
`)
	if code := run([]string{"init", repo}); code != 0 {
		t.Fatalf("init: %d", code)
	}
	history := readString(t, os.Getenv("RECALL_HISTORY"))
	if history != "gap provisioned ledger-project\n" {
		t.Fatalf("initial breadcrumb: %q", history)
	}
	write(t, os.Getenv("RECALL_HISTORY"), history+strings.Repeat("newer project activity\n", 1000))
	if code := run([]string{"init", repo}); code != 0 {
		t.Fatalf("second init: %d", code)
	}
	if got := strings.Count(readString(t, os.Getenv("RECALL_HISTORY")), "gap provisioned ledger-project\n"); got != 1 {
		t.Fatalf("setup breadcrumbs after 1000 newer notes: %d", got)
	}
	got := recallArgv(t)
	if len(got) != 3 || got[0] != got[2] || readString(t, filepath.Join(repo, ".gap", "gap.toml")) != config {
		t.Fatalf("repeated init changed configuration or added another breadcrumb:\n%v", got)
	}
}

func TestInitBreadcrumbSearchPhrase(t *testing.T) {
	repo := t.TempDir()
	slug := "quoted \"project\"-" + strings.Repeat("x", 120)
	write(t, filepath.Join(repo, ".gap", "gap.toml"), "slug = "+strconv.Quote(slug)+"\n")
	t.Setenv("RECALL_QUERY", `"gap provisioned quoted ""project""-`+strings.Repeat("x", 120)+`"`)
	recallStub(t, `
[ "$1" = search ] && [ "$2" = "$RECALL_QUERY" ] || exit 64
printf 'Found 1 result(s):\n\n[breadcrumbs#42] 2026-10-23\n  gap provisioned quoted "project"-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx...\n'
`)
	_, stderr := captureStd(t, func() {
		if code := run([]string{"init", repo}); code != 0 {
			t.Fatalf("init: %d", code)
		}
	})
	if got := recallArgv(t); len(got) != 1 || stderr != "" {
		t.Fatalf("an existing result with a truncated preview must prevent add: %v, stderr: %s", got, stderr)
	}
}

func TestInitBreadcrumbSuccessfulPaths(t *testing.T) {
	for _, c := range []struct {
		name, existing string
		adopt          bool
	}{
		{"unmarked", "existing rules", false},
		{"begin marker", "<!-- gap:begin -->\nprotected\n", false},
		{"end marker", "<!-- gap:end -->\nprotected\n", false},
		{"adopted", "existing rules", true},
		{"adopted empty", "", true},
	} {
		t.Run(c.name, func(t *testing.T) {
			recallStub(t, "exit 0\n")
			repo := t.TempDir()
			path := filepath.Join(repo, "AGENTS.md")
			write(t, path, c.existing)
			args := []string{"init"}
			if c.adopt {
				args = append(args, "--adopt")
			}
			if code := run(append(args, repo)); code != 0 {
				t.Fatalf("init: %d", code)
			}
			got := readString(t, path)
			if c.name == "unmarked" {
				if !strings.HasPrefix(got, c.existing+"\n\n") || !strings.Contains(got, gapBegin) {
					t.Fatalf("policy pointer: %q", got)
				}
			} else if got != c.existing {
				t.Fatalf("existing file changed: %q", got)
			}
			if got := recallArgv(t); len(got) != 2 || !strings.HasPrefix(got[1], "add breadcrumb ") {
				t.Fatalf("successful provisioning did not record a breadcrumb: %v", got)
			}
		})
	}
}

func TestInitInvalidConfiguration(t *testing.T) {
	for _, config := range []string{
		"[components.recall]\nenabled = false\n",
		"slug = \"\"\n",
		"slug = \"  \"\n",
		"slug = false\n",
		"slug = true\n",
		"slug = []\n",
		"[slug]\nvalue = \"project\"\n",
		"slug = \"unterminated\n",
	} {
		t.Run(config, func(t *testing.T) {
			recallStub(t, "exit 0\n")
			repo := t.TempDir()
			path := filepath.Join(repo, ".gap", "gap.toml")
			write(t, path, config)
			_, stderr := captureStd(t, func() {
				if code := run([]string{"init", repo}); code != 1 {
					t.Errorf("invalid configuration accepted: %d", code)
				}
			})
			if !strings.Contains(stderr, path) || !strings.Contains(stderr, "before running gap init") {
				t.Fatalf("configuration error must explain how to proceed: %s", stderr)
			}
			if _, err := os.Lstat(filepath.Join(repo, "AGENTS.md")); !os.IsNotExist(err) {
				t.Fatal("invalid configuration wrote project files")
			}
			if readString(t, path) != config || len(recallArgv(t)) != 0 {
				t.Fatal("invalid configuration was changed or recorded as provisioned")
			}
		})
	}
}

func TestInitReadConfigurationFailure(t *testing.T) {
	for _, kind := range []string{"directory", "dangling symlink", "blocked parent"} {
		t.Run(kind, func(t *testing.T) {
			recallStub(t, "exit 0\n")
			repo := t.TempDir()
			path := filepath.Join(repo, ".gap", "gap.toml")
			if kind == "blocked parent" {
				write(t, filepath.Dir(path), "not a directory\n")
			} else {
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatal(err)
				}
				if kind == "directory" {
					if err := os.Mkdir(path, 0o755); err != nil {
						t.Fatal(err)
					}
				} else if err := os.Symlink(filepath.Join(repo, "missing"), path); err != nil {
					t.Fatal(err)
				}
			}
			_, stderr := captureStd(t, func() {
				if code := run([]string{"init", repo}); code != 1 {
					t.Errorf("unreadable configuration accepted: %d", code)
				}
			})
			if !strings.Contains(stderr, path) {
				t.Fatalf("read error must identify the configuration: %s", stderr)
			}
			if _, err := os.Lstat(filepath.Join(repo, "AGENTS.md")); !os.IsNotExist(err) || len(recallArgv(t)) != 0 {
				t.Fatal("unreadable configuration wrote policy or a breadcrumb")
			}
		})
	}
}

func TestInitFailureDoesNotRecordBreadcrumb(t *testing.T) {
	for _, kind := range []string{"symlink", "directory", "write failure"} {
		t.Run(kind, func(t *testing.T) {
			recallStub(t, "exit 0\n")
			repo := t.TempDir()
			path := filepath.Join(repo, "AGENTS.md")
			switch kind {
			case "symlink":
				target := filepath.Join(repo, "rules.md")
				write(t, target, "protected\n")
				if err := os.Symlink(target, path); err != nil {
					t.Fatal(err)
				}
			case "directory":
				if err := os.Mkdir(path, 0o755); err != nil {
					t.Fatal(err)
				}
			case "write failure":
				write(t, fmt.Sprintf("%s.%d.tmp", path, os.Getpid()), "occupied\n")
			}
			if code := run([]string{"init", repo}); code != 1 {
				t.Fatalf("provisioning failure accepted: %d", code)
			}
			if got := recallArgv(t); len(got) != 0 {
				t.Fatalf("failed provisioning called Recall: %v", got)
			}
		})
	}
}

func TestInitBreadcrumbRecallFailures(t *testing.T) {
	for _, stage := range []string{"search", "add"} {
		for _, mode := range []string{"new", "adopted", "marked"} {
			t.Run(stage+"/"+mode, func(t *testing.T) {
				t.Setenv("RECALL_FAILURE", stage)
				t.Setenv("RECALL_WRITTEN", filepath.Join(t.TempDir(), "breadcrumbs"))
				recallStub(t, `
[ "$1" != "$RECALL_FAILURE" ] || exit 17
case "$1" in
search) printf 'No results found.\n' ;;
add) printf '%s\n' "$3" >> "$RECALL_WRITTEN" ;;
*) exit 64 ;;
esac
`)
				repo := t.TempDir()
				configPath := filepath.Join(repo, ".gap", "gap.toml")
				config := "profile = \"project\"\nslug = \"persisted-project\"\n"
				write(t, configPath, config)
				path := filepath.Join(repo, "AGENTS.md")
				existing := ""
				args := []string{"init"}
				if mode == "adopted" {
					existing = "existing project rules\n"
					args = append(args, "--adopt")
				} else if mode == "marked" {
					existing = "<!-- gap:begin -->\nprotected\n"
				}
				if existing != "" {
					write(t, path, existing)
				}
				args = append(args, repo)
				_, stderr := captureStd(t, func() {
					if code := run(args); code != 1 {
						t.Errorf("Recall %s failure must fail init: %d", stage, code)
					}
				})
				if !strings.Contains(stderr, "recall "+stage+" breadcrumb") || !strings.Contains(stderr, "exit status 17") {
					t.Fatalf("missing Recall failure diagnostic: %s", stderr)
				}
				calls := recallArgv(t)
				if len(calls) != 1 && stage == "search" || len(calls) != 2 && stage == "add" {
					t.Fatalf("unexpected commands after Recall %s failure: %v", stage, calls)
				}
				if _, err := os.Stat(os.Getenv("RECALL_WRITTEN")); !os.IsNotExist(err) {
					t.Fatal("failed Recall command wrote a breadcrumb")
				}
				if readString(t, configPath) != config || existing != "" && readString(t, path) != existing {
					t.Fatal("Recall failure changed existing project files")
				}
				t.Setenv("RECALL_FAILURE", "")
				if code := run(args); code != 0 {
					t.Fatalf("retry after Recall recovered: %d", code)
				}
				if got := readString(t, os.Getenv("RECALL_WRITTEN")); got != "gap provisioned persisted-project\n" {
					t.Fatalf("retry did not write the required breadcrumb: %q", got)
				}
			})
		}
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
