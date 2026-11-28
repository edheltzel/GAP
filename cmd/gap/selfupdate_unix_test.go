//go:build unix

package main

import (
	"os"
	"path/filepath"
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
