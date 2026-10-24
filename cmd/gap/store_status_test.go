package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStatusStoreStates(t *testing.T) {
	for _, tc := range []struct {
		name, state, hint string
		code              int
	}{
		{"clean", "ok", "", 0},
		{"changed", "changed", "inspect local changes first", 1},
		{"missing", "missing", "fetch each missing source checkout", 1},
		{"unpinned", "unpinned", "gap pin echo@master", 1},
		{"invalid", "", "", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			repo, commit := echoFixture(t, home, "#!/bin/sh\nexit 0\n")
			sourceRootFor = repo
			t.Cleanup(func() { sourceRootFor = "" })
			store := storePath(home, "echo", commit)
			switch tc.name {
			case "changed":
				write(t, filepath.Join(store, "scripts", "echo-doctor"), "changed\n")
			case "missing":
				if err := os.RemoveAll(store); err != nil {
					t.Fatal(err)
				}
			case "unpinned":
				if err := writeLock(repo, lockFile{Components: map[string]lockEntryM1{}}); err != nil {
					t.Fatal(err)
				}
			case "invalid":
				if err := os.RemoveAll(filepath.Join(store, ".git")); err != nil {
					t.Fatal(err)
				}
			}
			var code int
			out, stderr := captureStd(t, func() { code = run([]string{"status"}) })
			if code != tc.code {
				t.Fatalf("exit=%d want=%d\n%s\n%s", code, tc.code, out, stderr)
			}
			if tc.name == "invalid" {
				if !strings.Contains(stderr, "reading store") || strings.Contains(out, "store=missing") {
					t.Fatalf("invalid checkout misreported: %s\n%s", out, stderr)
				}
				return
			}
			if !strings.Contains(out, "store="+tc.state+"\n") {
				t.Fatalf("missing state %s: %s", tc.state, out)
			}
			if tc.hint != "" && !strings.Contains(out, tc.hint) {
				t.Fatalf("missing hint %q: %s", tc.hint, out)
			}
			if tc.code == 0 && strings.Contains(out, "Next") {
				t.Fatalf("healthy store showed repair advice: %s", out)
			}
			out, stderr = captureStd(t, func() { code = run([]string{"doctor"}) })
			if code != tc.code {
				t.Fatalf("doctor exit=%d want=%d: %s\n%s", code, tc.code, out, stderr)
			}
			if tc.state == "missing" || tc.state == "changed" {
				if !strings.Contains(out, "pinned source checkout "+tc.state) || strings.Contains(out, "fixture-ok") {
					t.Fatalf("doctor misreported store: %s", out)
				}
			}
			if tc.state == "unpinned" && !strings.Contains(out, "gap pin echo@master") {
				t.Fatalf("doctor omitted pin command: %s", out)
			}
		})
	}
}
