package main

import (
	"bytes"
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
