//go:build unix

package main

import (
	"encoding/json"
	"errors"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"testing"
)

const uninstallUtility = hookManifest + "[mcp.util-mcp]\ncommand = [\"{store}/server.sh\", \"--stdio\"]\n"

func newUninstallEnv(t *testing.T) *utilityEnv {
	t.Helper()
	e := newUtilityEnv(t, uninstallUtility)
	t.Setenv("PATH", fakeMCPBins(t)+string(os.PathListSeparator)+"/bin"+string(os.PathListSeparator)+"/usr/bin")
	quietStdout(t)
	tools := filepath.Join(t.TempDir(), "tools")
	commit, tree := gitFixture(t, tools, "distro skill\n")
	write(t, filepath.Join(e.repo, "gap.toml"), "schema = \"2\"\n[components.tools]\nrepo = \""+tools+"\"\nref = \"master\"\nnames = [\"do-todo\"]\n")
	if err := writeLock(e.repo, lockFile{Components: map[string]lockEntryM1{"tools": {Ref: "master", Commit: commit, Tree: tree}}}); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(e.repo, "global", "commands", "hello.md"), "Say hello.\n")
	readyEnvReviewDistro(t, e.repo)
	write(t, filepath.Join(e.home, ".codex", "hooks.json"), `{"hooks":{"Stop":[{"hooks":[{"type":"command","command":"echo mine"}]}]}}`+"\n")
	write(t, filepath.Join(e.home, ".agents", "skills", "mine", "SKILL.md"), "not installed by gap\n")
	write(t, userConfigFile(e.home), "[trust]\nrepos = [\""+e.remote+"\"]\n")
	if code := run([]string{"sync", "--yes"}); code != 0 {
		t.Fatalf("sync exit %d", code)
	}
	if code := run([]string{"add", e.remote, "--id", "util"}); code != 0 {
		t.Fatalf("add exit %d", code)
	}
	mcpLog(t, e.home)
	for _, rel := range []string{".agents/skills/do-todo/SKILL.md", ".agents/skills/util-skill/SKILL.md", ".claude/commands/hello.md", ".claude/commands/util-cmd.md", ".codex/gap/global-agent-policy.md"} {
		if !e.exists(rel) {
			t.Fatalf("setup is missing %s", rel)
		}
	}
	if !strings.Contains(readString(t, filepath.Join(e.home, ".codex", "hooks.json")), "global-agent-policy.md") {
		t.Fatal("sync did not add the Codex hooks")
	}
	return e
}

func snapshotTree(t *testing.T, root string) map[string]string {
	t.Helper()
	snap := map[string]string{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		switch {
		case d.Type()&fs.ModeSymlink != 0:
			target, _ := os.Readlink(path)
			snap[path] = "link:" + target
		case d.IsDir():
			snap[path] = "dir"
		default:
			b, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			snap[path] = string(b)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return snap
}

func TestUninstallRemovesOnlyGapRecords(t *testing.T) {
	e := newUninstallEnv(t)
	claude := filepath.Join(e.home, ".claude", "rules", "global-agent-policy.md")
	write(t, claude, "foreign head\n"+readString(t, claude)+"foreign tail\n")
	pi := filepath.Join(e.home, ".pi", "agent", "APPEND_SYSTEM.md")
	write(t, pi, "old whole copy\n")
	ledgerPath := filepath.Join(e.home, ".gap", "published.json")
	ledger, err := readLedger(ledgerPath)
	if err != nil {
		t.Fatal(err)
	}
	ledger[pi] = digest("old whole copy\n")
	if err := writeLedger(ledgerPath, ledger); err != nil {
		t.Fatal(err)
	}
	cursor := filepath.Join(e.home, ".cursor", "rules", "global-agent-policy.mdc")
	write(t, cursor, "mine\n")

	if code := run([]string{"uninstall", "--yes"}); code != 0 {
		t.Fatalf("uninstall exit %d", code)
	}
	if got := readString(t, claude); got != "foreign head\nforeign tail\n" {
		t.Fatalf("foreign text around the block changed: %q", got)
	}
	if readString(t, cursor) != "mine\n" {
		t.Fatal("a foreign file without markers was changed")
	}
	for _, rel := range []string{
		".pi/agent/APPEND_SYSTEM.md", ".omp/agent/APPEND_SYSTEM.md", ".empryo/EMPRYO.md", ".codex/gap/global-agent-policy.md", ".jcode/prompt-overlay.md",
		".claude/commands/hello.md", ".pi/agent/prompts/hello.md", ".omp/agent/commands/hello.md",
		".claude/commands/util-cmd.md", ".agents/skills/util-skill", ".claude/skills/util-skill",
		".agents/skills/do-todo", ".claude/skills/do-todo",
		".gap/state.json", ".gap/published.json", ".gap/store",
	} {
		if _, err := os.Lstat(filepath.Join(e.home, rel)); !os.IsNotExist(err) {
			t.Errorf("%s still exists: %v", rel, err)
		}
	}
	hooks := readString(t, filepath.Join(e.home, ".codex", "hooks.json"))
	if !strings.Contains(hooks, "echo mine") || strings.Contains(hooks, "global-agent-policy.md") {
		t.Fatalf("hooks.json after uninstall:\n%s", hooks)
	}
	if !e.exists("uninstalled") {
		t.Fatal("the utility uninstall hook did not run")
	}
	got := mcpLog(t, e.home)
	slices.Sort(got)
	if want := []string{"claude mcp remove --scope user util-mcp", "codex mcp remove util-mcp", "pi mcp remove util-mcp"}; !slices.Equal(got, want) {
		t.Fatalf("mcp remove calls: %v", got)
	}
	if !e.exists(".agents/skills/mine/SKILL.md") {
		t.Fatal("a skill gap did not install was removed")
	}
	if !e.exists(".gap/backups") {
		t.Fatal("backups were not kept")
	}
}

func TestUninstallFailureKeepsRecords(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	e := newUninstallEnv(t)
	rules := filepath.Join(e.home, ".claude", "rules")
	if err := os.Chmod(rules, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(rules, 0o755) })
	if code := run([]string{"uninstall", "--yes"}); code != 1 {
		t.Fatalf("failed uninstall exit %d, want 1", code)
	}
	for _, rel := range []string{".gap/state.json", ".gap/published.json", ".claude/rules/global-agent-policy.md"} {
		if !e.exists(rel) {
			t.Fatalf("%s was removed after a failed step", rel)
		}
	}
	if err := os.Chmod(rules, 0o755); err != nil {
		t.Fatal(err)
	}
	if code := run([]string{"uninstall", "--yes"}); code != 0 {
		t.Fatalf("rerun exit %d", code)
	}
	if e.exists(".gap/state.json") || e.exists(".claude/rules/global-agent-policy.md") {
		t.Fatal("rerun did not finish the uninstall")
	}
}

func TestUninstallDryRunChangesNothing(t *testing.T) {
	e := newUninstallEnv(t)
	before := snapshotTree(t, e.home)
	stdout, _ := captureStd(t, func() {
		if code := run([]string{"uninstall", "--dry-run"}); code != 0 {
			t.Errorf("dry run exit %d", code)
		}
	})
	after := snapshotTree(t, e.home)
	for path, v := range before {
		if after[path] != v {
			t.Errorf("dry run changed %s", path)
		}
	}
	if len(after) != len(before) {
		t.Errorf("dry run added files: %d -> %d", len(before), len(after))
	}
	for _, want := range []string{"gap uninstall will remove", "util", "hello.md", "Dry run: nothing removed."} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("preview is missing %q:\n%s", want, stdout)
		}
	}
}

func TestHarnessHomeValidation(t *testing.T) {
	for _, key := range []string{"CODEX_HOME", "JCODE_HOME"} {
		t.Run(key, func(t *testing.T) {
			t.Setenv("JCODE_HOME", "")
			e := newUtilityEnv(t, hookManifest)
			readyEnvReviewDistro(t, e.repo)
			captureStd(t, func() {
				if code := run([]string{"sync", "--yes"}); code != 0 {
					t.Fatalf("sync exit %d", code)
				}
			})
			if !e.exists(".jcode/prompt-overlay.md") || !e.exists(".gap/published.json") {
				t.Fatal("sync did not publish the jcode overlay and ledger")
			}
			t.Setenv(key, "relative/home")
			for _, args := range [][]string{{"sync", "--yes"}, {"uninstall", "--dry-run"}, {"uninstall", "--yes"}} {
				t.Run(strings.Join(args, " "), func(t *testing.T) {
					before := snapshotTree(t, e.home)
					_, stderr := captureStd(t, func() {
						if code := run(args); code != 1 {
							t.Errorf("exit %d, want 1", code)
						}
					})
					if !strings.Contains(stderr, key+" must be an absolute path") {
						t.Errorf("missing home validation error: %q", stderr)
					}
					if !maps.Equal(before, snapshotTree(t, e.home)) {
						t.Fatal("invalid harness home changed published files or records")
					}
				})
			}
		})
	}
}

func TestJcodeSyncAndUninstall(t *testing.T) {
	for _, name := range []string{"default", "override"} {
		t.Run(name, func(t *testing.T) {
			e := newUtilityEnv(t, hookManifest)
			readyEnvReviewDistro(t, e.repo)
			base := filepath.Join(e.home, ".jcode")
			if name == "override" {
				base = t.TempDir()
				t.Setenv("JCODE_HOME", base)
			}
			dest := filepath.Join(base, "prompt-overlay.md")
			write(t, dest, "user head\n<!-- gap:begin -->\nold policy\n<!-- gap:end -->\nuser tail\n")
			captureStd(t, func() {
				if code := run([]string{"sync", "--yes"}); code != 0 {
					t.Fatalf("sync exit %d", code)
				}
			})
			if got := readString(t, dest); got != "user head\n<!-- gap:begin -->\nfixture policy\n<!-- gap:end -->\nuser tail\n" {
				t.Fatalf("published overlay: %q", got)
			}
			captureStd(t, func() {
				if code := run([]string{"uninstall", "--yes"}); code != 0 {
					t.Fatalf("uninstall exit %d", code)
				}
			})
			if got := readString(t, dest); got != "user head\nuser tail\n" {
				t.Fatalf("uninstalled overlay: %q", got)
			}
		})
	}
}

func TestRemoveGapHookKeepsOthers(t *testing.T) {
	policy := "/codex/gap/global-agent-policy.md"
	text, _ := insertGapHook(`{"hooks":{"SessionStart":[{"matcher":"startup|resume|clear|compact","hooks":[{"type":"command","command":"echo mine"}]}]},"x":1}`, policy)
	out, empty, res := removeGapHook(text, policy)
	if res != hookChanged || empty || strings.Contains(out, "global-agent-policy") || !strings.Contains(out, "echo mine") || !strings.Contains(out, `"x": 1`) {
		t.Fatalf("removeGapHook = %v %v\n%s", res, empty, out)
	}
	if _, _, res := removeGapHook(out, policy); res != hookPresent {
		t.Fatalf("second removal = %v", res)
	}
	only, _ := insertGapHook(`{"hooks":{}}`, policy)
	if _, empty, _ := removeGapHook(only, policy); !empty {
		t.Fatal("a hooks file with only GAP hooks should report empty")
	}
}

func sharedUninstallEnv(t *testing.T) *utilityEnv {
	t.Helper()
	e := newUtilityEnv(t, hookManifest)
	quietStdout(t)
	readyEnvReviewDistro(t, e.repo)
	write(t, filepath.Join(e.repo, "global", "skills.json"), `[{"source":"example/shared","skills":["util-skill"]}]`)
	commitAll(t, e.repo)
	if _, err := git(e.repo, "update-ref", "refs/remotes/origin/master", "HEAD"); err != nil {
		t.Fatal(err)
	}
	write(t, userConfigFile(e.home), "[trust]\nrepos = [\""+e.remote+"\"]\n")
	if code := run([]string{"sync", "--skip", "manifest", "--yes"}); code != 0 {
		t.Fatalf("sync exit %d", code)
	}
	if code := run([]string{"add", e.remote, "--id", "util"}); code != 0 {
		t.Fatalf("add exit %d", code)
	}
	write(t, filepath.Join(e.repo, "global", "commands", "util-cmd.md"), "Run the utility.\n")
	return e
}

func TestUninstallRemovesUtilityCopiesRetainedForGlobalSelections(t *testing.T) {
	for _, retry := range []bool{false, true} {
		t.Run(map[bool]string{false: "success", true: "retry"}[retry], func(t *testing.T) {
			e := sharedUninstallEnv(t)
			if retry {
				original := renameBackup
				blocked := filepath.Join(e.home, ".claude", "skills", "util-skill")
				renameBackup = func(src, dst string) error {
					if src == blocked {
						return fs.ErrPermission
					}
					return original(src, dst)
				}
				t.Cleanup(func() { renameBackup = original })
				if code := run([]string{"uninstall", "--yes"}); code != 1 {
					t.Fatalf("blocked cleanup exit %d", code)
				}
				state, err := readState(e.home)
				if err != nil || !slices.Contains(state.Components[pendingUninstall].Skills, "util-skill") {
					t.Fatalf("lost pending cleanup: %+v, %v", state, err)
				}
				if !e.exists(".gap/published.json") || !e.exists(".gap/store") {
					t.Fatal("failed uninstall discarded records")
				}
				renameBackup = original
			}
			if code := run([]string{"uninstall", "--yes"}); code != 0 {
				t.Fatalf("uninstall exit %d", code)
			}
			for _, path := range append(skillDirs(e.home, "util-skill"), commandDests(e.home, "util-cmd.md")...) {
				if _, err := os.Lstat(path); !errors.Is(err, fs.ErrNotExist) {
					t.Errorf("owned copy remains at %s: %v", path, err)
				}
			}
			if e.exists(".gap/state.json") || e.exists(".gap/published.json") {
				t.Fatal("successful uninstall kept cleanup records")
			}
		})
	}
}

func TestUninstallRejectsUnreadableOrMalformedSkillLock(t *testing.T) {
	for _, kind := range []string{"malformed", "directory"} {
		t.Run(kind, func(t *testing.T) {
			e := newUninstallEnv(t)
			path := skillLockPath(e.home)
			if kind == "malformed" {
				write(t, path, "not json")
			} else if err := os.MkdirAll(path, 0o755); err != nil {
				t.Fatal(err)
			}
			before := snapshotTree(t, e.home)
			if code := run([]string{"uninstall", "--yes"}); code != 1 {
				t.Fatalf("invalid lock uninstall exit %d", code)
			}
			after := snapshotTree(t, e.home)
			if len(before) != len(after) {
				t.Fatal("lock discovery failure changed the home")
			}
			for name, value := range before {
				if after[name] != value {
					t.Fatalf("lock discovery failure changed %s", name)
				}
			}
		})
	}
}

func TestDropSkillLockEntryPreservesForeignRecordsAndErrors(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_STATE_HOME", "")
	path := skillLockPath(home)
	if err := dropSkillLockEntry(home, "ours"); err != nil {
		t.Fatal(err)
	}
	write(t, path, `{"version":3,"skills":{"ours":{"source":"example/ours"},"foreign":{"custom":true}}}`)
	if err := dropSkillLockEntry(home, "ours"); err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal([]byte(readString(t, path)), &got); err != nil {
		t.Fatal(err)
	}
	skills := got["skills"].(map[string]any)
	if _, exists := skills["ours"]; exists || got["version"] != float64(3) || skills["foreign"].(map[string]any)["custom"] != true {
		t.Fatalf("lock cleanup changed foreign records: %+v", got)
	}
	write(t, path, "not json")
	if err := dropSkillLockEntry(home, "ours"); err == nil || readString(t, path) != "not json" {
		t.Fatalf("malformed lock was ignored or changed: %v", err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(path, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := dropSkillLockEntry(home, "ours"); err == nil {
		t.Fatal("unreadable lock was ignored")
	}
}

func TestMoveToBackupAcrossFilesystems(t *testing.T) {
	original := renameBackup
	renameBackup = func(src, dst string) error {
		return &os.LinkError{Op: "rename", Old: src, New: dst, Err: syscall.EXDEV}
	}
	t.Cleanup(func() { renameBackup = original })
	for _, kind := range []string{"file", "directory", "symlink", "changed during copy"} {
		t.Run(kind, func(t *testing.T) {
			home, backup := t.TempDir(), t.TempDir()
			path := filepath.Join(home, "owned")
			switch kind {
			case "directory":
				write(t, filepath.Join(path, "run"), "owned bytes\n")
				if err := os.Chmod(filepath.Join(path, "run"), 0o755); err != nil {
					t.Fatal(err)
				}
			case "symlink":
				write(t, filepath.Join(home, "target"), "foreign bytes\n")
				if err := os.Symlink("target", path); err != nil {
					t.Fatal(err)
				}
			default:
				write(t, path, "owned bytes\n")
			}
			if kind == "changed during copy" {
				old := afterCopyWrite
				afterCopyWrite = func(src, dst string) { write(t, src, "concurrent edit\n") }
				t.Cleanup(func() { afterCopyWrite = old })
			}
			moved, err := moveToBackup(home, backup, path)
			if kind == "changed during copy" {
				if err == nil || moved || readString(t, path) != "concurrent edit\n" {
					t.Fatalf("unverified copy removed the source: %v %v", moved, err)
				}
				return
			}
			if err != nil || !moved {
				t.Fatalf("backup: %v %v", moved, err)
			}
			if _, err := os.Lstat(path); !errors.Is(err, fs.ErrNotExist) {
				t.Fatalf("source remains: %v", err)
			}
			dest := backupTarget(home, backup, path)
			switch kind {
			case "symlink":
				if target, err := os.Readlink(dest); err != nil || target != "target" || readString(t, filepath.Join(home, "target")) != "foreign bytes\n" {
					t.Fatalf("backup followed or changed the link: %q %v", target, err)
				}
			case "directory":
				file := filepath.Join(dest, "run")
				info, err := os.Stat(file)
				if err != nil || info.Mode().Perm() != 0o755 || readString(t, file) != "owned bytes\n" {
					t.Fatalf("directory backup lost bytes or mode: %v", err)
				}
			default:
				if readString(t, dest) != "owned bytes\n" {
					t.Fatal("file backup lost bytes")
				}
			}
		})
	}
}

func TestUninstallLockCleanupFailureKeepsOwnership(t *testing.T) {
	e := sharedUninstallEnv(t)
	original := renameBackup
	renameBackup = func(src, dst string) error {
		write(t, skillLockPath(e.home), "invalid lock")
		return original(src, dst)
	}
	t.Cleanup(func() { renameBackup = original })
	if code := run([]string{"uninstall", "--yes"}); code != 1 {
		t.Fatalf("lock cleanup failure exit %d", code)
	}
	state, err := readState(e.home)
	if err != nil || !slices.Contains(state.Components[pendingUninstall].Skills, "util-skill") || !e.exists(".gap/published.json") {
		t.Fatalf("lock cleanup lost ownership: %+v, %v", state, err)
	}
	renameBackup = original
	write(t, skillLockPath(e.home), `{"skills":{"util-skill":{"source":"example/shared"}}}`)
	if code := run([]string{"uninstall", "--yes"}); code != 0 {
		t.Fatalf("retry exit %d", code)
	}
	lock, err := readSkillLock(e.home)
	if err != nil || len(lock) != 0 || e.exists(".gap/state.json") {
		t.Fatalf("retry left records: %+v, %v", lock, err)
	}
}

func TestUninstallStripsLegacyCodexPrefix(t *testing.T) {
	e := newUninstallEnv(t)
	prefix := "# Old GAP policy\n\nFollow the rules.\n"
	agents := filepath.Join(e.home, ".codex", "AGENTS.md")
	write(t, agents, prefix+"\n# Mine\nKeep this.\n")
	ledgerFile := filepath.Join(e.home, ".gap", "published.json")
	ledger, err := readLedger(ledgerFile)
	if err != nil {
		t.Fatal(err)
	}
	ledger[agents] = digest(prefix)
	if err := writeLedger(ledgerFile, ledger); err != nil {
		t.Fatal(err)
	}
	if code := run([]string{"uninstall", "--yes"}); code != 0 {
		t.Fatalf("uninstall exit %d", code)
	}
	got := readString(t, agents)
	if strings.Contains(got, "Old GAP policy") || !strings.Contains(got, "Keep this.") {
		t.Fatalf("legacy GAP prefix not stripped or user rules lost:\n%s", got)
	}
}

func TestUninstallKeepsHeldCleanupAfterInterruption(t *testing.T) {
	e := newUninstallEnv(t)
	if err := holdUtilityCleanup(e.home, []string{"util"}); err != nil {
		t.Fatal(err)
	}
	state, err := readState(e.home)
	if err != nil {
		t.Fatal(err)
	}
	delete(state.Components, "util")
	if err := writeState(e.home, state); err != nil {
		t.Fatal(err)
	}
	if code := run([]string{"uninstall", "--yes"}); code != 0 {
		t.Fatalf("retry exit %d", code)
	}
	if e.exists(".agents/skills/util-skill") || e.exists(".claude/commands/util-cmd.md") {
		t.Fatal("a utility whose record was lost mid-uninstall left its skill or command behind")
	}
}
