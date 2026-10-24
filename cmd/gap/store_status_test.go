package main

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestStatusPreservesSourceAndStore(t *testing.T) {
	for _, packed := range []bool{false, true} {
		t.Run(fmt.Sprintf("packed=%t", packed), func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			repo, commit := echoFixture(t, home, "#!/bin/sh\nexit 0\n")
			t.Chdir(repo)
			sourceRootFor = repo
			t.Cleanup(func() { sourceRootFor = "" })
			store := storePath(home, "echo", commit)
			if packed {
				if _, err := git(store, "repack", "-ad"); err != nil {
					t.Fatal(err)
				}
			}
			stamp := time.Unix(946684800, 0)
			if err := filepath.Walk(filepath.Join(store, ".git", "objects"), func(path string, info os.FileInfo, err error) error {
				if err != nil || info.IsDir() {
					return err
				}
				return os.Chtimes(path, stamp, stamp)
			}); err != nil {
				t.Fatal(err)
			}
			type fileState struct {
				mode  os.FileMode
				mtime time.Time
				body  string
			}
			snapshot := func() map[string]fileState {
				t.Helper()
				files := map[string]fileState{}
				for _, root := range []string{home, repo} {
					if err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
						if err != nil {
							return err
						}
						state := fileState{mode: info.Mode(), mtime: info.ModTime()}
						if !info.IsDir() {
							body, err := os.ReadFile(path)
							if err != nil {
								return err
							}
							state.body = string(body)
						}
						files[path] = state
						return nil
					}); err != nil {
						t.Fatal(err)
					}
				}
				return files
			}
			before := snapshot()
			var code int
			out, stderr := captureStd(t, func() { code = run([]string{"status"}) })
			want := fmt.Sprintf("echo ref=master commit=%s store=ok\n", commit[:7])
			if code != 0 || out != want || stderr != "" {
				t.Fatalf("healthy status: exit=%d stdout=%q stderr=%q", code, out, stderr)
			}
			after := snapshot()
			for path, state := range before {
				if after[path] != state {
					t.Errorf("status changed %s: mtime %s -> %s", path, state.mtime, after[path].mtime)
				}
			}
			if !reflect.DeepEqual(before, after) {
				t.Fatal("status must preserve source and store contents, modes, and timestamps")
			}
		})
	}
}

func TestStatusGroupsComponentSkills(t *testing.T) {
	home, repo, remote := t.TempDir(), t.TempDir(), t.TempDir()
	t.Setenv("HOME", home)
	t.Chdir(repo)
	commit, tree := gitFixture(t, remote, "skill\n")
	write(t, filepath.Join(repo, "global", "AGENTS.md"), "policy\n")
	config := "schema = \"2\"\n"
	lock := lockFile{Components: map[string]lockEntryM1{}}
	for _, id := range []string{"alpha", "beta", "zeta"} {
		names := "[\"do-todo\"]"
		if id == "zeta" {
			names = "[]"
		}
		config += fmt.Sprintf("[components.%s]\nrepo = %q\nref = \"master\"\nnames = %s\n", id, remote, names)
		if id == "beta" {
			config += "[components.beta.skills.do-todo]\nload = \"project\"\n[components.beta.skills.do-todo.when]\nfiles = [\"gap.toml\"]\n"
		}
		pin := lockEntryM1{Ref: "master", Commit: commit, Tree: tree}
		lock.Components[id] = pin
		if _, err := ensureStore(home, component{ID: id, Repo: remote}, pin); err != nil {
			t.Fatal(err)
		}
	}
	write(t, filepath.Join(repo, "gap.toml"), config)
	if err := writeLock(repo, lock); err != nil {
		t.Fatal(err)
	}
	gitFixture(t, repo, "project skill\n")
	for _, dir := range skillDirs(home, "do-todo") {
		write(t, filepath.Join(dir, "SKILL.md"), "skill\n")
	}
	write(t, filepath.Join(repo, ".agents", "skills", "do-todo", "SKILL.md"), "skill\n")
	sourceRootFor = repo
	t.Cleanup(func() { sourceRootFor = "" })
	out, stderr := captureStd(t, func() {
		if code := run([]string{"status"}); code != 1 {
			t.Errorf("status with one missing project copy exit=%d, want 1", code)
		}
	})
	want := fmt.Sprintf("alpha ref=master commit=%s store=ok\n  do-todo drift=0\nbeta ref=master commit=%s store=ok\n  do-todo drift=1\nzeta ref=master commit=%s store=ok\n", commit[:7], commit[:7], commit[:7])
	if out != want || stderr != "" {
		t.Fatalf("skills must follow their component headers:\ngot %q\nwant %q\nstderr %s", out, want, stderr)
	}
}

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
