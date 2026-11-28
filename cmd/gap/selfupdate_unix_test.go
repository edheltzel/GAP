//go:build unix

package main

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func TestWriteManualPublishesReadablePage(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "gap")
	write(t, exe, "#!/bin/sh\necho .TH GAP 1\n")
	if err := os.Chmod(exe, 0o755); err != nil {
		t.Fatal(err)
	}
	page := filepath.Join(dir, "gap.1")
	write(t, page, "old\n")
	old := syscall.Umask(0o077)
	err := writeManual(exe, page)
	syscall.Umask(old)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(page)
	text, _ := os.ReadFile(page)
	if err != nil || info.Mode().Perm() != 0o644 || string(text) != ".TH GAP 1\n" {
		t.Fatalf("manual must be replaced and stay world readable: %v %v %q", err, info.Mode(), text)
	}
	if left, _ := filepath.Glob(filepath.Join(dir, "gap.1.*")); len(left) != 0 {
		t.Fatalf("staging files left behind: %v", left)
	}
}

func TestSelfUpdateReleaseUsesInstalledVersion(t *testing.T) {
	e := newSelfUpdateEnv(t)
	version = "1.0.0"
	e.advanceOrigin(t)
	releaseServer(t, "v1.1.0", []byte("release 1.1.0"), false)
	installed := "#!/bin/sh\necho gap version v1.2.0\n"
	write(t, e.exe, installed)
	if err := os.Chmod(e.exe, 0o755); err != nil {
		t.Fatal(err)
	}
	code, out := selfUpdateRun(t)
	if code != 0 || readString(t, e.exe) != installed || !strings.Contains(out, "v1.2.0 is newer than the latest release v1.1.0") {
		t.Fatalf("a queued update must compare against the installed binary, not its own build (exit %d):\n%s", code, out)
	}
}
