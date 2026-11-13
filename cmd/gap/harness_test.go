package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func unsetEnv(t *testing.T, key string) {
	t.Helper()
	t.Setenv(key, "")
	if err := os.Unsetenv(key); err != nil {
		t.Fatal(err)
	}
}

func TestJcodeHomeEmptyRefused(t *testing.T) {
	jcode := harnessNamed("jcode")
	unsetEnv(t, "JCODE_HOME")
	if got, err := harnessHome(jcode, "/h"); err != nil || got != filepath.Join("/h", ".jcode") {
		t.Fatalf("unset: %q %v", got, err)
	}
	t.Setenv("JCODE_HOME", "")
	if _, err := harnessHome(jcode, "/h"); err == nil || !strings.Contains(err.Error(), "unset it") {
		t.Fatalf("set but empty: %v", err)
	}
	if dests, err := policyDests("/h", policyPath); err == nil || dests != nil {
		t.Fatalf("policyDests with empty JCODE_HOME: %v %v", dests, err)
	}
	t.Setenv("CODEX_HOME", "")
	if got, err := resolveCodexHome(os.Getenv("CODEX_HOME"), "/h"); err != nil || got != filepath.Join("/h", ".codex") {
		t.Fatalf("empty CODEX_HOME must stay unset: %q %v", got, err)
	}
}
