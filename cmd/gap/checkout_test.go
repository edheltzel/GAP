package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fakeCheckout(t *testing.T, root string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(root, "global", "AGENTS.md"), "policy\n")
	write(t, filepath.Join(root, "cmd", "gap", "main.go"), "package main\n")
}

func TestCheckoutRootPresent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("GAP_HOME", "")
	want := filepath.Join(home, ".gap", "checkout")
	fakeCheckout(t, want)
	got, err := checkoutRoot()
	if err != nil || got != want {
		t.Fatalf("checkoutRoot = %q, %v; want %q", got, err, want)
	}
}

func TestCheckoutRootMissing(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("GAP_HOME", "")
	_, err := checkoutRoot()
	if err == nil || !strings.Contains(err.Error(), "install.sh") {
		t.Fatalf("missing checkout: %v", err)
	}
}

func TestCheckoutRootWrongRepo(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("GAP_HOME", "")
	root := filepath.Join(home, ".gap", "checkout")
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(root, "README.md"), "other\n")
	if _, err := checkoutRoot(); err == nil || !strings.Contains(err.Error(), "AGENTS.md") {
		t.Fatalf("wrong repo: %v", err)
	}
	write(t, filepath.Join(root, "global", "AGENTS.md"), "policy\n")
	if _, err := checkoutRoot(); err == nil || !strings.Contains(err.Error(), "main.go") {
		t.Fatalf("no cmd/gap: %v", err)
	}
	if err := os.RemoveAll(filepath.Join(root, ".git")); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(root, "cmd", "gap", "main.go"), "package main\n")
	if _, err := checkoutRoot(); err == nil || !strings.Contains(err.Error(), ".git") {
		t.Fatalf("not a git checkout: %v", err)
	}
}

func TestCheckoutRootGapHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	fakeCheckout(t, filepath.Join(home, ".gap", "checkout"))
	gapHome := t.TempDir()
	t.Setenv("GAP_HOME", gapHome)
	if _, err := checkoutRoot(); err == nil {
		t.Fatal("GAP_HOME without a checkout must not fall back to ~/.gap")
	}
	want := filepath.Join(gapHome, "checkout")
	fakeCheckout(t, want)
	got, err := checkoutRoot()
	if err != nil || got != want {
		t.Fatalf("checkoutRoot = %q, %v; want %q", got, err, want)
	}
	t.Setenv("GAP_HOME", "relative")
	if _, err := checkoutRoot(); err == nil || !strings.Contains(err.Error(), "absolute") {
		t.Fatalf("relative GAP_HOME: %v", err)
	}
}
