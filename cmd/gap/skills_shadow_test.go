package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestJcodeRefreshLinkedDefinition(t *testing.T) {
	home := t.TempDir()
	unsetEnv(t, "JCODE_HOME")
	store := filepath.Join(home, "store")
	write(t, filepath.Join(store, "skills", "linked", "SKILL.md"), "managed\n")
	shadow := filepath.Join(home, ".jcode", "skills", "linked")
	write(t, filepath.Join(shadow, "definition.md"), "stale\n")
	if err := os.Symlink("definition.md", filepath.Join(shadow, "SKILL.md")); err != nil {
		t.Fatal(err)
	}
	if err := refreshJcodeShadows(home, manifestEntry{Path: store, Skills: []string{"linked"}}, "linked"); err != nil {
		t.Fatal(err)
	}
	if got := readString(t, filepath.Join(shadow, "SKILL.md")); got != "managed\n" {
		t.Fatalf("linked definition was not refreshed: %q", got)
	}
}

func TestJcodeRefreshKeepsRecoveryAfterRollbackFailure(t *testing.T) {
	home := t.TempDir()
	unsetEnv(t, "JCODE_HOME")
	store := filepath.Join(home, "store")
	write(t, filepath.Join(store, "skills", "old", "SKILL.md"), "managed\n")
	shadow := filepath.Join(home, ".jcode", "skills", "old")
	write(t, filepath.Join(shadow, "SKILL.md"), "original\n")
	previous := renameJcodeShadow
	t.Cleanup(func() { renameJcodeShadow = previous })
	calls := 0
	var saved string
	renameJcodeShadow = func(from, to string) error {
		calls++
		if calls == 1 {
			saved = to
			return os.Rename(from, to)
		}
		return os.ErrPermission
	}
	err := refreshJcodeShadows(home, manifestEntry{Path: store, Skills: []string{"old"}}, "old")
	if !errors.Is(err, os.ErrPermission) || !strings.Contains(err.Error(), saved) || saved == "" {
		t.Fatalf("refresh error does not identify the saved copy: %v", err)
	}
	if calls != 3 {
		t.Fatalf("rename calls = %d, want save, install and rollback", calls)
	}
	if got := readString(t, filepath.Join(saved, "SKILL.md")); got != "original\n" {
		t.Fatalf("recovery copy = %q", got)
	}
}
