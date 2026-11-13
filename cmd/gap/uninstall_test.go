//go:build unix

package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
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
		".pi/agent/APPEND_SYSTEM.md", ".omp/agent/APPEND_SYSTEM.md", ".empryo/EMPRYO.md", ".codex/gap/global-agent-policy.md",
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
