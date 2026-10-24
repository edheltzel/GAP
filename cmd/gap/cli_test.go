package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHelpListsCommands(t *testing.T) {
	buf := new(bytes.Buffer)
	cmd := rootCmd()
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	for _, name := range []string{"sync", "status", "restore", "pin", "doctor", "init"} {
		if !strings.Contains(got, name) {
			t.Fatalf("help is missing %q:\n%s", name, got)
		}
	}
}

func TestUnknownCommandFails(t *testing.T) {
	cmd := rootCmd()
	cmd.SetOut(new(bytes.Buffer))
	cmd.SetErr(new(bytes.Buffer))
	cmd.SetArgs([]string{"nope"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("unknown command exited clean")
	}
}

func TestSyncHelpListsFlags(t *testing.T) {
	buf := new(bytes.Buffer)
	cmd := rootCmd()
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"sync", "--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	for _, name := range []string{"--remove-codex-legacy", "--profile"} {
		if !strings.Contains(got, name) {
			t.Fatalf("sync help is missing %q:\n%s", name, got)
		}
	}
}

func TestSyncRejectsExtraArgs(t *testing.T) {
	cmd := rootCmd()
	cmd.SetOut(new(bytes.Buffer))
	cmd.SetErr(new(bytes.Buffer))
	cmd.SetArgs([]string{"sync", "extra"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("sync accepted an extra argument")
	}
}

func TestInitUsesParsedDirectory(t *testing.T) {
	for _, dir := range []string{".", "project", "--adopt", "--other", "-x"} {
		for _, adopt := range []bool{false, true} {
			name := dir
			if adopt {
				name += "/adopt"
			}
			t.Run(name, func(t *testing.T) {
				t.Setenv("PATH", "/usr/bin:/bin")
				parent := t.TempDir()
				t.Chdir(parent)
				target := filepath.Join(parent, dir)
				const original = "# Existing project\n"
				write(t, filepath.Join(target, "AGENTS.md"), original)
				args := []string{"init"}
				if adopt {
					args = append(args, "--adopt")
				}
				if dir != "." {
					args = append(args, "--", dir)
				}
				out, stderr := captureStd(t, func() {
					if code := run(args); code != 0 {
						t.Errorf("init exit=%d", code)
					}
				})
				config, err := parseTOML(readString(t, filepath.Join(target, ".gap", "gap.toml")))
				if err != nil || config["slug"] != filepath.Base(target) || config["profile"] != "project" {
					t.Fatalf("wrong project initialized: %#v, %v\n%s\n%s", config, err, out, stderr)
				}
				got := readString(t, filepath.Join(target, "AGENTS.md"))
				if adopt && got != original {
					t.Fatalf("adopt changed existing policy: %q", got)
				}
				if !adopt && (!strings.HasPrefix(got, original) || !strings.Contains(got, "<!-- gap:begin -->")) {
					t.Fatalf("init omitted the policy pointer: %q", got)
				}
				if dir != "." {
					if _, err := os.Stat(filepath.Join(parent, ".gap")); !os.IsNotExist(err) {
						t.Fatalf("init wrote configuration in the parent: %v", err)
					}
				}
			})
		}
	}
}

func TestInitRejectsArguments(t *testing.T) {
	for _, args := range [][]string{{"init", "--unknown"}, {"init", "one", "two"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			t.Chdir(t.TempDir())
			captureStd(t, func() {
				if code := run(args); code != 1 {
					t.Errorf("invalid arguments exit=%d, want 1", code)
				}
			})
			if _, err := os.Stat(".gap"); !os.IsNotExist(err) {
				t.Fatalf("invalid arguments initialized a project: %v", err)
			}
		})
	}
}
