package main

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const (
	source     = "See [x](docs/agents/dox.md).\n"
	policyPath = "/h/.codex/gap/global-agent-policy.md"
	legacyFile = "old policy\n\n<!-- soulstack:start -->\nkeep\n<!-- soulstack:end -->\n<!-- gitbutler-agent-setup:start -->\nbut\n<!-- gitbutler-agent-setup:end -->\n"
)

func TestMain(m *testing.M) {
	docsDir = "/repo/global/docs/agents"
	os.Exit(m.Run())
}

func TestOurs(t *testing.T) {
	payload := adapt(source, false)
	cases := []struct {
		name          string
		existing      string
		exists        bool
		payload, last string
		want          bool
	}{
		{"missing file", "", false, payload, "", true},
		{"matching payload", payload, true, payload, "", true},
		{"source bytes are not the publish", source, true, payload, "", false},
		{"other text", "other\n", true, payload, "", false},
		{"last publish re-syncs", "old publish\n", true, "new publish\n", digest("old publish\n"), true},
		{"hand edit", "hand edit\n", true, "new publish\n", digest("old publish\n"), false},
	}
	for _, c := range cases {
		if got := ours(c.existing, c.exists, c.payload, c.last); got != c.want {
			t.Errorf("%s: ours = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestAdapt(t *testing.T) {
	payload := adapt(source, false)
	if payload != "See [x](/repo/global/docs/agents/dox.md).\n" {
		t.Fatalf("link not rewritten: %q", payload)
	}
	if adapt("no newline", false) != "no newline\n" {
		t.Fatal("missing trailing newline")
	}
	wrapped := adapt(source, true)
	if wrapped != "---\nalwaysApply: true\n---\n\n"+payload {
		t.Fatalf("cursor wrap drifted: %q", wrapped)
	}
}

func TestResolveCodexHome(t *testing.T) {
	for env, want := range map[string]string{"": "/h/.codex", "/abs/codex": "/abs/codex"} {
		if got, err := resolveCodexHome(env, "/h"); err != nil || got != want {
			t.Errorf("CODEX_HOME=%q: got %q, %v", env, got, err)
		}
	}
	for _, bad := range []string{"codex", "~/.codex", "rel/codex"} {
		if _, err := resolveCodexHome(bad, "/h"); err == nil {
			t.Errorf("relative CODEX_HOME must fail: %s", bad)
		}
	}
}

func TestRefuseCheckout(t *testing.T) {
	cases := []struct {
		name   string
		c      checkout
		refuse bool
	}{
		{"non-primary", checkout{"side", "master", ".git", ".git", false}, true},
		{"main is not primary here", checkout{"main", "master", ".git", ".git", false}, true},
		{"primary", checkout{"master", "master", ".git", ".git", false}, false},
		{"GitButler workspace", checkout{"gitbutler/workspace", "master", ".git", ".git", false}, false},
		{"unmerged workspace", checkout{"gitbutler/workspace", "master", ".git", ".git", true}, true},
		{"worktree", checkout{"master", "master", ".git/worktrees/x", ".git", false}, true},
		{"unmerged primary", checkout{"master", "master", ".git", ".git", true}, true},
	}
	for _, c := range cases {
		if got := refuseCheckout("/repo", c.c) != ""; got != c.refuse {
			t.Errorf("%s: refused = %v, want %v", c.name, got, c.refuse)
		}
	}
}

func TestLedgerPrefixEnd(t *testing.T) {
	old := "old policy\n"
	cut, ok := ledgerPrefixEnd(legacyFile, digest(old))
	if !ok || cut != len(old) || !strings.HasPrefix(legacyFile[cut:], "\n<!-- soulstack:start -->") {
		t.Fatalf("ledger cut missed the prefix: %d %v", cut, ok)
	}
	if _, ok := ledgerPrefixEnd(legacyFile, digest("nope\n")); ok {
		t.Fatal("unmatched prefix must refuse")
	}
}

func decodeHooks(t *testing.T, text string) map[string]any {
	t.Helper()
	var doc map[string]any
	if err := json.Unmarshal([]byte(text), &doc); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, text)
	}
	return doc["hooks"].(map[string]any)
}

func TestInsertGapHook(t *testing.T) {
	t.Run("empty hooks gains both events", func(t *testing.T) {
		out, res := insertGapHook("{\n  \"hooks\": {}\n}\n", policyPath)
		if res != hookChanged {
			t.Fatal("missing hooks file failed")
		}
		hooks := decodeHooks(t, out)
		session := hooks["SessionStart"].([]any)[0].(map[string]any)
		if session["matcher"] != "startup|resume|clear|compact" {
			t.Fatalf("matcher missing: %v", session)
		}
		if _, has := hooks["SubagentStart"].([]any)[0].(map[string]any)["matcher"]; has {
			t.Fatal("SubagentStart entry must not be scoped")
		}
	})

	t.Run("live shape keeps other entries and is idempotent", func(t *testing.T) {
		live := `{"hooks":{"SessionStart":[{"hooks":[{"command":"keep","timeout":10,"type":"command"}],"matcher":""}],"Stop":[{"hooks":[{"command":"stop","type":"command"}]}]}}`
		out, res := insertGapHook(live, policyPath)
		if res != hookChanged {
			t.Fatal("live-shaped hooks failed")
		}
		before, after := decodeHooks(t, live), decodeHooks(t, out)
		if !reflect.DeepEqual(after["Stop"], before["Stop"]) {
			t.Fatal("Stop changed")
		}
		sessions := after["SessionStart"].([]any)
		if !reflect.DeepEqual(sessions[0], before["SessionStart"].([]any)[0]) || len(sessions) != 2 {
			t.Fatalf("SessionStart not appended: %v", sessions)
		}
		if _, res := insertGapHook(out, policyPath); res != hookPresent {
			t.Fatal("hook insert was not idempotent")
		}
	})

	t.Run("SessionStart-only install gains SubagentStart", func(t *testing.T) {
		sessionOnly := `{"hooks":{"SessionStart":[{"matcher":"startup|resume|clear|compact","hooks":[{"type":"command","command":"cat '/h/.codex/gap/global-agent-policy.md'","timeout":10,"additionalContextLimit":0}]}],"SubagentStart":[{"hooks":[{"command":"keep","type":"command"}]}]}}`
		out, res := insertGapHook(sessionOnly, policyPath)
		if res != hookChanged {
			t.Fatal("upgrade failed")
		}
		before, after := decodeHooks(t, sessionOnly), decodeHooks(t, out)
		if !reflect.DeepEqual(after["SessionStart"], before["SessionStart"]) {
			t.Fatal("upgrade changed SessionStart")
		}
		subs := after["SubagentStart"].([]any)
		if subs[0].(map[string]any)["hooks"].([]any)[0].(map[string]any)["command"] != "keep" {
			t.Fatal("upgrade shifted SubagentStart entries")
		}
		gap := subs[1].(map[string]any)
		if _, has := gap["matcher"]; has || gap["hooks"].([]any)[0].(map[string]any)["command"] != hookCommand(policyPath) {
			t.Fatalf("SubagentStart GAP entry missing or scoped: %v", gap)
		}
	})

	t.Run("refuses what it cannot edit", func(t *testing.T) {
		for _, bad := range []string{"{", "null", `{"hooks":null}`, `{"hooks":{"SessionStart":{}}}`, `{"hooks":{}} {}`} {
			if _, res := insertGapHook(bad, policyPath); res != hookMalformed {
				t.Errorf("%q must refuse", bad)
			}
		}
	})

	t.Run("keeps HTML characters and big numbers", func(t *testing.T) {
		out, _ := insertGapHook(`{"hooks":{},"x":"a<b>&c","n":12345678901234567890}`, policyPath)
		if !strings.Contains(out, `"a<b>&c"`) || !strings.Contains(out, "12345678901234567890") {
			t.Fatalf("values rewritten: %s", out)
		}
	})
}

func readString(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func write(t *testing.T, path, text string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestForeignSkipOnlyItself(t *testing.T) {
	dir := t.TempDir()
	ledgerFile := filepath.Join(dir, "published.json")
	foreign, other := filepath.Join(dir, "foreign.md"), filepath.Join(dir, "other.md")
	write(t, foreign, "nope\n")
	ledger := map[string]string{}
	wroteForeign, err := publishPolicy(source, foreign, false, ledger, ledgerFile, false, dir)
	if err != nil || wroteForeign {
		t.Fatalf("foreign must be refused: %v %v", wroteForeign, err)
	}
	wroteOther, err := publishPolicy(source, other, false, ledger, ledgerFile, false, dir)
	if err != nil || !wroteOther {
		t.Fatalf("foreign skip blocked the next write: %v", err)
	}
	if readString(t, foreign) != "nope\n" {
		t.Fatal("foreign file was replaced")
	}
	if readString(t, other) != gapMarked(adapt(source, false)) {
		t.Fatal("other destination was not published")
	}
	saved, err := readLedger(ledgerFile)
	if err != nil || saved[other] != digest(adapt(source, false)) || saved[foreign] != "" {
		t.Fatalf("ledger wrong: %v %v", saved, err)
	}
}

func gapMarked(body string) string {
	return "<!-- gap:begin -->\n" + body + "<!-- gap:end -->\n"
}

func withYes(t *testing.T) {
	t.Helper()
	oldTerm, oldLine := termReady, confirmLine
	termReady = func() bool { return true }
	confirmLine = func() string { return "y\n" }
	t.Cleanup(func() { termReady, confirmLine = oldTerm, oldLine })
}

func TestSpliceKeepsForeignContent(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "policy.md")
	ledgerFile := filepath.Join(dir, "published.json")
	head := "user head\n"
	write(t, dest, head)
	ledger := map[string]string{}
	withYes(t)
	wrote, err := publishPolicy(source, dest, false, ledger, ledgerFile, true, dir)
	if err != nil || !wrote {
		t.Fatalf("first splice: wrote %v err %v", wrote, err)
	}
	body := adapt(source, false)
	first := readString(t, dest)
	if first != head+gapMarked(body) {
		t.Fatalf("first splice did not keep the foreign prefix: %q", first)
	}
	tail := "user tail\n"
	write(t, dest, first+tail)
	termReady = func() bool { return false }
	wrote, err = publishPolicy(source, dest, false, ledger, ledgerFile, true, dir)
	if err != nil || !wrote {
		t.Fatalf("second sync: wrote %v err %v", wrote, err)
	}
	if got := readString(t, dest); got != head+gapMarked(body)+tail {
		t.Fatalf("second sync changed content outside the block: %q", got)
	}
	backups, _ := filepath.Glob(filepath.Join(dir, ".gap", "backups", "*", "policy.md"))
	if len(backups) != 1 || readString(t, backups[0]) != head {
		t.Fatalf("first splice must back up the foreign file once: %v", backups)
	}
}

func TestSpliceUpdatesBlockWhenPolicyChanges(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "policy.md")
	ledgerFile := filepath.Join(dir, "published.json")
	head, tail := "keep head\n", "keep tail\n"
	body := adapt(source, false)
	write(t, dest, head+gapMarked(body)+tail)
	ledger := map[string]string{dest: digest(body)}
	next := "See [y](docs/agents/dox.md).\n"
	wrote, err := publishPolicy(next, dest, false, ledger, ledgerFile, false, dir)
	if err != nil || !wrote {
		t.Fatalf("policy change: wrote %v err %v", wrote, err)
	}
	want := adapt(next, false)
	if got := readString(t, dest); got != head+gapMarked(want)+tail {
		t.Fatalf("block update moved foreign content: %q", got)
	}
	if ledger[dest] != digest(want) {
		t.Fatalf("ledger must track the block, got %s", ledger[dest])
	}
	if backups, _ := filepath.Glob(filepath.Join(dir, ".gap", "backups", "*")); len(backups) != 0 {
		t.Fatal("updating a marked block must not back up")
	}
}

func TestSpliceMarkerBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name, head, block, tail string
	}{
		{"extra closer", "head\n", gapMarked("old policy\n"), "local rules\n<!-- gap:end -->\n"},
		{"closer at EOF", "head\n", gapMarked("old policy\n"), "local rules\n<!-- gap:end -->"},
		{"leading closer", "<!-- gap:end -->\nhead\n", gapMarked("old policy\n"), "tail\n"},
		{"leading opener", "<!-- gap:begin -->\nhead\n", gapMarked("old policy\n"), "tail\n"},
		{"trailing opener", "head\n", gapMarked("old policy\n"), "tail\n<!-- gap:begin -->\n"},
		{"multiple blocks", gapMarked("earlier policy\n") + "head\n", gapMarked("old policy\n"), "tail\n<!-- gap:end -->\n"},
		{"CRLF", "head\r\n", "<!-- gap:begin -->\r\nold policy\r\n<!-- gap:end -->\r\n", "local rules\r\n<!-- gap:end -->\r\n"},
		{"closing boundary at EOF", "head\n", "<!-- gap:begin -->\nold policy\n<!-- gap:end -->", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			dest := filepath.Join(dir, "policy.md")
			write(t, dest, tc.head+tc.block+tc.tail)
			wrote, err := publishPolicy("new policy\n", dest, false, map[string]string{}, filepath.Join(dir, "published.json"), false, dir)
			if err != nil || !wrote {
				t.Fatalf("update: wrote %v err %v", wrote, err)
			}
			if got, want := readString(t, dest), tc.head+gapMarked("new policy\n")+tc.tail; got != want {
				t.Fatalf("bytes outside the paired markers changed: got %q, want %q", got, want)
			}
			if backups, err := filepath.Glob(filepath.Join(dir, ".gap", "backups", "*")); err != nil || len(backups) != 0 {
				t.Fatalf("marked update must not create a backup: %v, %v", backups, err)
			}
		})
	}
}

func TestSplicePreservesPermissions(t *testing.T) {
	for _, mode := range []os.FileMode{0o600, 0o640} {
		for _, cursor := range []bool{false, true} {
			for _, state := range []string{"foreign", "marked", "legacy"} {
				t.Run(mode.String()+"/"+map[bool]string{false: "plain", true: "cursor"}[cursor]+"/"+state, func(t *testing.T) {
					dir := t.TempDir()
					dest := filepath.Join(dir, "policy.md")
					ledger := map[string]string{}
					before, want := "private rules\n", "private rules\n"+gapMarked("new policy\n")
					switch state {
					case "marked":
						before += gapMarked("old policy\n")
					case "legacy":
						before, want = "old policy\n", gapMarked("new policy\n")
						if cursor {
							before = "---\nalwaysApply: true\n---\n\n" + before
							want = "---\nalwaysApply: true\n---\n\n" + want
						}
						ledger[dest] = digest(before)
					}
					write(t, dest, before)
					if err := os.Chmod(dest, mode); err != nil {
						t.Fatal(err)
					}
					withYes(t)
					wrote, err := publishPolicy("new policy\n", dest, cursor, ledger, filepath.Join(dir, "published.json"), true, dir)
					if err != nil || !wrote {
						t.Fatalf("publish: wrote %v err %v", wrote, err)
					}
					if got := readString(t, dest); got != want {
						t.Fatalf("publish = %q, want %q", got, want)
					}
					info, err := os.Stat(dest)
					if err != nil {
						t.Fatal(err)
					}
					if info.Mode().Perm() != mode {
						t.Fatalf("permissions = %o, want %o", info.Mode().Perm(), mode)
					}
				})
			}
		}
	}
}

func TestSpliceEmptyFileOwnership(t *testing.T) {
	for _, tc := range []struct {
		name                               string
		allowPrompt, terminal, owned, want bool
		answer                             string
		prompts                            int
	}{
		{"prompt disabled", false, true, false, false, "y\n", 0},
		{"no terminal", true, false, false, false, "y\n", 0},
		{"declined", true, true, false, false, "n\n", 1},
		{"confirmed", true, true, false, true, "y\n", 1},
		{"ledger owned", false, false, true, true, "", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			dest, ledgerFile := filepath.Join(dir, "policy.md"), filepath.Join(dir, "published.json")
			write(t, dest, "")
			ledger := map[string]string{}
			if tc.owned {
				ledger[dest] = digest("")
			}
			withYes(t)
			termReady = func() bool { return tc.terminal }
			prompts := 0
			confirmLine = func() string { prompts++; return tc.answer }
			wrote, err := publishPolicy("policy\n", dest, false, ledger, ledgerFile, tc.allowPrompt, dir)
			if err != nil || wrote != tc.want || prompts != tc.prompts {
				t.Fatalf("publish = %v, %v; prompts = %d; want %v, %d", wrote, err, prompts, tc.want, tc.prompts)
			}
			want := ""
			if tc.want {
				want = gapMarked("policy\n")
			}
			if got := readString(t, dest); got != want {
				t.Fatalf("empty destination = %q, want %q", got, want)
			}
			backups, err := filepath.Glob(filepath.Join(dir, ".gap", "backups", "*", "policy.md"))
			if err != nil {
				t.Fatal(err)
			}
			if tc.want && !tc.owned {
				if len(backups) != 1 || readString(t, backups[0]) != "" {
					t.Fatalf("confirmed empty foreign file must be backed up: %v", backups)
				}
			} else if len(backups) != 0 {
				t.Fatalf("unexpected backups: %v", backups)
			}
			if !tc.want {
				if _, err := os.Stat(ledgerFile); !os.IsNotExist(err) || len(ledger) != 0 {
					t.Fatalf("refused publish changed the ledger: %v, %v", ledger, err)
				}
			}
		})
	}
}

func TestSpliceMigratesLegacyPublish(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "policy.md")
	ledgerFile := filepath.Join(dir, "published.json")
	old := "old publish\n"
	write(t, dest, old)
	ledger := map[string]string{dest: digest(old)}
	wrote, err := publishPolicy(source, dest, false, ledger, ledgerFile, false, dir)
	if err != nil || !wrote {
		t.Fatalf("legacy migrate: wrote %v err %v", wrote, err)
	}
	body := adapt(source, false)
	if got := readString(t, dest); got != gapMarked(body) {
		t.Fatalf("legacy file was not rewritten as markers: %q", got)
	}
	if ledger[dest] != digest(body) {
		t.Fatal("ledger must track the new block")
	}
	if backups, _ := filepath.Glob(filepath.Join(dir, ".gap", "backups", "*")); len(backups) != 0 {
		t.Fatal("migrating our own publish must not back up")
	}
}

func TestSpliceRefusesWithoutTerminal(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "foreign.md")
	ledgerFile := filepath.Join(dir, "published.json")
	write(t, dest, "mine\n")
	wrote, err := publishPolicy(source, dest, false, map[string]string{}, ledgerFile, true, dir)
	if err != nil || wrote {
		t.Fatalf("non-terminal must refuse: wrote %v err %v", wrote, err)
	}
	if readString(t, dest) != "mine\n" {
		t.Fatal("refused splice changed the file")
	}
}

func TestSpliceCursorFrontmatterStaysFirst(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "global-agent-policy.mdc")
	ledgerFile := filepath.Join(dir, "published.json")
	ledger := map[string]string{}
	wrote, err := publishPolicy(source, dest, true, ledger, ledgerFile, false, dir)
	if err != nil || !wrote {
		t.Fatalf("missing mdc: wrote %v err %v", wrote, err)
	}
	body := adapt(source, false)
	front := "---\nalwaysApply: true\n---\n\n"
	if got := readString(t, dest); got != front+gapMarked(body) {
		t.Fatalf("new mdc frontmatter is not first: %q", got)
	}

	old := adapt(source, true)
	legacy := filepath.Join(dir, "legacy.mdc")
	write(t, legacy, old)
	ledger[legacy] = digest(old)
	wrote, err = publishPolicy(source, legacy, true, ledger, ledgerFile, false, dir)
	if err != nil || !wrote || readString(t, legacy) != front+gapMarked(body) {
		t.Fatalf("legacy mdc must keep frontmatter first: %q err %v", readString(t, legacy), err)
	}

	foreign := "---\ndescription: mine\n---\n\nmine\n"
	other := filepath.Join(dir, "other.mdc")
	write(t, other, foreign)
	withYes(t)
	wrote, err = publishPolicy(source, other, true, ledger, ledgerFile, true, dir)
	if err != nil || !wrote || readString(t, other) != foreign+gapMarked(body) {
		t.Fatalf("foreign mdc frontmatter moved: %q err %v", readString(t, other), err)
	}
}

func TestCodexCleanupGates(t *testing.T) {
	dir := t.TempDir()
	codex := filepath.Join(dir, "codex")
	policy := filepath.Join(codex, "gap", "global-agent-policy.md")
	agents := filepath.Join(codex, "AGENTS.md")
	hooks := filepath.Join(codex, "hooks.json")
	ledgerFile := filepath.Join(dir, "published.json")
	payload := adapt(source, false)

	write(t, policy, "foreign policy\n")
	write(t, agents, "keep agents\n")
	write(t, hooks, `{"hooks":{"Stop":[]}}`+"\n")
	wrote, err := publishPolicy(source, policy, false, map[string]string{}, ledgerFile, false, dir)
	if err != nil || wrote {
		t.Fatal("foreign policy must be refused")
	}
	if ok, err := afterCodexPublish(wrote, policy, payload, codex, dir); err != nil || !ok {
		t.Fatal("refused policy must not fail the hook gate")
	}
	if readString(t, agents) != "keep agents\n" || readString(t, hooks) != `{"hooks":{"Stop":[]}}`+"\n" {
		t.Fatal("refused policy changed Codex files")
	}

	write(t, policy, gapMarked(payload)+"local rules\n<!-- gap:end -->\n")
	write(t, agents, legacyFile)
	write(t, hooks, `{"hooks":{}}`+"\n")
	ledger := map[string]string{agents: digest("old policy\n")}
	if ok, _ := cleanupLegacyCodex(codex, policy, payload, ledger, dir, ledgerFile); ok {
		t.Fatal("cleanup must refuse when the hook is missing")
	}
	if ok, err := afterCodexPublish(true, policy, payload, codex, dir); err != nil || !ok {
		t.Fatalf("default publish failed: %v", err)
	}
	if readString(t, agents) != legacyFile {
		t.Fatal("default run changed AGENTS.md")
	}

	write(t, policy, "foreign policy\n")
	if ok, _ := cleanupLegacyCodex(codex, policy, payload, ledger, dir, ledgerFile); ok || readString(t, agents) != legacyFile {
		t.Fatal("cleanup must refuse when the policy file is foreign")
	}

	write(t, policy, payload)
	if ok, _ := afterCodexPublish(true, policy, payload, codex, dir); ok {
		t.Fatal("hook verification must refuse an unmarked payload")
	}
	if ok, _ := cleanupLegacyCodex(codex, policy, payload, ledger, dir, ledgerFile); ok || readString(t, agents) != legacyFile {
		t.Fatal("cleanup must refuse an unmarked payload")
	}

	write(t, policy, gapMarked(payload)+"local rules\n<!-- gap:end -->\n")
	if ok, err := cleanupLegacyCodex(codex, policy, payload, ledger, dir, ledgerFile); err != nil || !ok {
		t.Fatalf("cleanup failed: %v", err)
	}
	stripped := readString(t, agents)
	if stripped != legacyFile[len("old policy\n"):] {
		t.Fatalf("cleanup did not keep the tool blocks: %q", stripped)
	}
	if ok, _ := cleanupLegacyCodex(codex, policy, payload, ledger, dir, ledgerFile); !ok || readString(t, agents) != stripped {
		t.Fatal("second cleanup must be a no-op")
	}

	write(t, agents, legacyFile)
	if ok, _ := cleanupLegacyCodex(codex, policy, payload, map[string]string{}, dir, ledgerFile); ok || readString(t, agents) != legacyFile {
		t.Fatal("old text with no ledger entry must refuse")
	}

	target := filepath.Join(dir, "real-hooks.json")
	write(t, target, `{"hooks":{}}`+"\n")
	if err := os.Remove(hooks); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, hooks); err != nil {
		t.Fatal(err)
	}
	if ok, _ := afterCodexPublish(true, policy, payload, codex, dir); ok {
		t.Fatal("symlinked hooks.json must refuse")
	}
	if info, err := os.Lstat(hooks); err != nil || !isSymlink(info) || readString(t, target) != `{"hooks":{}}`+"\n" {
		t.Fatal("symlinked hooks.json changed")
	}
}

// TestSyncEndToEnd builds the CLI inside a throwaway clone on its primary branch, then runs it
// from outside any repo against a temp HOME. The clone gets this tree's Go files so uncommitted code is tested.
func TestSyncEndToEnd(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	tmp := t.TempDir()
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	clone := filepath.Join(tmp, "repo")
	for _, args := range [][]string{
		{"clone", "-q", root, clone},
		{"-C", clone, "checkout", "-q", "-B", "master", "HEAD"},
		{"-C", clone, "update-ref", "refs/remotes/origin/master", "HEAD"},
		{"-C", clone, "symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/master"},
	} {
		if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	sources, _ := filepath.Glob("*.go")
	for _, src := range append(sources, "../../go.mod") {
		rel, _ := filepath.Rel(".", src)
		write(t, filepath.Join(clone, "cmd", "gap", rel), readString(t, src))
	}
	bin := filepath.Join(tmp, "gap")
	build := exec.Command("go", "build", "-o", bin, "./cmd/gap")
	build.Dir = clone
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	// commitMerged commits the clone's working tree and marks it merged (origin/master = HEAD).
	commitMerged := func(msg string) {
		t.Helper()
		for _, args := range [][]string{
			{"-C", clone, "add", "-A", "global", "skills"},
			{"-C", clone, "-c", "user.name=gap", "-c", "user.email=gap@example.com", "commit", "-q", "-m", msg},
			{"-C", clone, "update-ref", "refs/remotes/origin/master", "HEAD"},
		} {
			if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
				t.Fatalf("git %v: %v\n%s", args, err, out)
			}
		}
	}
	write(t, filepath.Join(clone, "global", "skills.json"), `[{"source":".","skills":["do-a","do-b"]},{"source":"owner/ext","skills":["ext"]}]`+"\n")
	for _, name := range []string{"do-a", "do-b"} {
		write(t, filepath.Join(clone, "skills", "global", "core", name, "SKILL.md"), name+"\n")
	}
	commitMerged("test manifest")
	write(t, filepath.Join(clone, "global", "commands", "state-docs.md"), "published command\n")
	commitMerged("test command")
	home := filepath.Join(tmp, "home")
	codex := filepath.Join(home, ".codex")
	write(t, filepath.Join(codex, "hooks.json"), `{"hooks":{"SubagentStart":[{"hooks":[{"type":"command","command":"keep"}]}]}}`+"\n")
	write(t, filepath.Join(codex, "AGENTS.md"), legacyFile)
	unlisted := filepath.Join(home, ".agents", "skills", "mine", "SKILL.md")
	write(t, unlisted, "mine\n")
	extSource := t.TempDir()
	write(t, filepath.Join(extSource, "SKILL.md"), "ext\n")
	extHash, err := treeHash(extSource)
	if err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(home, ".agents", ".skill-lock.json"), `{"skills":{"ext":{"source":"owner/ext","sourceUrl":"https://github.com/owner/ext.git","skillFolderHash":"`+extHash+`"}}}`)

	// Stub npx: logs its argv and installs each -s skill the way skills.sh does (folder + Claude symlink).
	stub, npxLog := filepath.Join(tmp, "stub"), filepath.Join(tmp, "npx.log")
	write(t, filepath.Join(stub, "npx"), `#!/bin/sh
echo "$*" >> `+npxLog+`
while [ $# -gt 0 ]; do
  if [ "$1" = -s ]; then
    mkdir -p "$HOME/.agents/skills/$2" "$HOME/.claude/skills"
    echo "$2" > "$HOME/.agents/skills/$2/SKILL.md"
    ln -sfn "$HOME/.agents/skills/$2" "$HOME/.claude/skills/$2"
    shift
  fi
  shift
done
`)
	if err := os.Chmod(filepath.Join(stub, "npx"), 0o755); err != nil {
		t.Fatal(err)
	}
	pathEnv := "PATH=" + stub + string(os.PathListSeparator) + os.Getenv("PATH")
	npxCalls := func() []string {
		b, _ := os.ReadFile(npxLog)
		return strings.Fields(strings.ReplaceAll(strings.TrimSpace(string(b)), " ", "_"))
	}
	var extraEnv []string
	var lastOut string
	cli := func(args ...string) int {
		cmd := exec.Command(bin, append([]string{"sync"}, args...)...)
		cmd.Dir = tmp
		cmd.Env = append(append(os.Environ(), "HOME="+home, "CODEX_HOME=", "XDG_STATE_HOME=", pathEnv), extraEnv...)
		out, err := cmd.CombinedOutput()
		lastOut = string(out)
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			return exit.ExitCode()
		}
		if err != nil {
			t.Fatal(err)
		}
		return 0
	}

	if code := cli(); code != 0 {
		t.Fatalf("publish exited %d", code)
	}
	for _, dest := range []string{".claude/rules/global-agent-policy.md", ".pi/agent/APPEND_SYSTEM.md", ".omp/agent/APPEND_SYSTEM.md", ".codex/gap/global-agent-policy.md", ".cursor/rules/global-agent-policy.mdc"} {
		if _, err := os.Lstat(filepath.Join(home, dest)); err != nil {
			t.Errorf("not published: %s", dest)
		}
	}
	ompCmd := filepath.Join(home, ".omp", "agent", "commands", "state-docs.md")
	claudeCmd := filepath.Join(home, ".claude", "commands", "state-docs.md")
	for _, dest := range []string{ompCmd, claudeCmd} {
		if readString(t, dest) != "published command\n" {
			t.Fatalf("command not published: %s", dest)
		}
	}
	if calls := npxCalls(); len(calls) != 2 || !strings.Contains(calls[0], "add_"+clone+"_-g_-y_-a_claude-code_-a_codex_-s_do-a_-s_do-b") || !strings.Contains(calls[1], "add_owner/ext_") {
		t.Fatalf("first run must install each source's missing skills once: %q", calls)
	}
	if readString(t, unlisted) != "mine\n" {
		t.Fatal("unlisted skill changed")
	}
	if code := cli(); code != 0 || len(npxCalls()) != 2 {
		t.Fatalf("second run must not call skills.sh (exit %d, calls %q)\n%s", code, npxCalls(), lastOut)
	}
	if readString(t, ompCmd) != "published command\n" || readString(t, claudeCmd) != "published command\n" {
		t.Fatal("second run changed a published command")
	}
	write(t, ompCmd, "foreign command\n")
	if code := cli(); code != 1 || !strings.Contains(lastOut, "Refusing to replace foreign file") || readString(t, ompCmd) != "foreign command\n" {
		t.Fatalf("foreign command must be refused (exit %d)\n%s", code, lastOut)
	}
	if readString(t, claudeCmd) != "published command\n" {
		t.Fatal("foreign skip replaced the other command destination")
	}
	if err := os.Remove(ompCmd); err != nil {
		t.Fatal(err)
	}
	for _, missing := range skillDirs(home, "do-a") {
		canonical, mirror := skillDirs(home, "do-a")[0], skillDirs(home, "do-a")[1]
		if err := os.RemoveAll(mirror); err != nil {
			t.Fatal(err)
		}
		write(t, filepath.Join(mirror, "SKILL.md"), "mirror edit\n")
		write(t, filepath.Join(canonical, "SKILL.md"), "canonical edit\n")
		if err := os.RemoveAll(missing); err != nil {
			t.Fatal(err)
		}
		calls := len(npxCalls())
		if code := cli(); code != 1 || len(npxCalls()) != calls {
			t.Fatalf("partial installs must require review without running npx (exit %d)\n%s", code, lastOut)
		}
		remaining := canonical
		want := "canonical edit\n"
		if missing == canonical {
			remaining, want = mirror, "mirror edit\n"
		}
		if readString(t, filepath.Join(remaining, "SKILL.md")) != want {
			t.Fatal("partial install overwrote the surviving destination")
		}
		write(t, filepath.Join(canonical, "SKILL.md"), "do-a\n")
		if err := os.RemoveAll(mirror); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(canonical, mirror); err != nil {
			t.Fatal(err)
		}
	}

	// No Node: policy still publishes, the skills step is skipped, exit 1.
	noNode := filepath.Join(tmp, "nonode")
	gitPath, _ := exec.LookPath("git")
	if err := os.MkdirAll(noNode, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(gitPath, filepath.Join(noNode, "git")); err != nil {
		t.Fatal(err)
	}
	claudeRules := filepath.Join(home, ".claude", "rules", "global-agent-policy.md")
	if err := os.RemoveAll(filepath.Join(home, ".agents", "skills", "ext")); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(claudeRules); err != nil {
		t.Fatal(err)
	}
	extraEnv = []string{"PATH=" + noNode}
	if code := cli(); code != 1 {
		t.Fatalf("missing npx must exit 1, got %d", code)
	}
	if _, err := os.Stat(claudeRules); err != nil {
		t.Fatal("policy must still publish without npx")
	}
	extraEnv = nil
	write(t, filepath.Join(home, ".agents", "skills", "ext", "SKILL.md"), "ext\n")
	hooks := decodeHooks(t, readString(t, filepath.Join(codex, "hooks.json")))
	subs, sessions := hooks["SubagentStart"].([]any), hooks["SessionStart"].([]any)
	if len(subs) != 2 || len(sessions) != 1 || subs[0].(map[string]any)["hooks"].([]any)[0].(map[string]any)["command"] != "keep" {
		t.Fatalf("hooks.json not appended as expected: %v", hooks)
	}
	if readString(t, filepath.Join(codex, "AGENTS.md")) != legacyFile {
		t.Fatal("default run changed AGENTS.md")
	}
	if code := cli("--remove-codex-legacy"); code != 1 || readString(t, filepath.Join(codex, "AGENTS.md")) != legacyFile {
		t.Fatal("flag must refuse old text with no ledger entry")
	}
	// Only merged global/AGENTS.md is published, never a file named in the environment.
	evil := filepath.Join(tmp, "evil.md")
	write(t, evil, "not merged\n")
	extraEnv = []string{"GAP_SOURCE=" + evil}
	cli()
	if strings.Contains(readString(t, filepath.Join(home, ".claude", "rules", "global-agent-policy.md")), "not merged") {
		t.Error("GAP_SOURCE published a file the merge gate never checked")
	}
	extraEnv = nil

	// skills/ is merge-gated like global/: an edited tracked skill or an untracked file in skills/ refuses.
	skill := filepath.Join(clone, "skills", "global", "core", "do-a", "SKILL.md")
	write(t, skill, "unmerged edit\n")
	if code := cli(); code != 1 || !strings.Contains(lastOut, "skills/") {
		t.Fatalf("an unmerged skill edit must refuse (exit %d)\n%s", code, lastOut)
	}
	write(t, skill, "do-a\n")
	extra := filepath.Join(clone, "skills", "global", "core", "do-a", "notes.md")
	write(t, extra, "untracked\n")
	if code := cli(); code != 1 || !strings.Contains(lastOut, "skills/") {
		t.Fatalf("an untracked file in skills/ must refuse (exit %d)\n%s", code, lastOut)
	}
	if err := os.Remove(extra); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(clone, "skills", ".gitignore"), "ignored/\nnode_modules/\n")
	write(t, filepath.Join(clone, "global", ".gitignore"), "ignored/\nskills.json\n.DS_Store\n")
	commitMerged("ignore generated files")
	write(t, filepath.Join(clone, "skills", "ignored", "payload"), "unmerged ignored content\n")
	if code := cli(); code != 1 || !strings.Contains(lastOut, "differs from origin/") {
		t.Fatalf("an ignored file skills.sh would install must refuse (exit %d)\n%s", code, lastOut)
	}
	if err := os.RemoveAll(filepath.Join(clone, "skills", "ignored")); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(clone, "global", "ignored", "payload"), "unread\n")
	write(t, filepath.Join(clone, "global", ".DS_Store"), "store\n")
	write(t, filepath.Join(clone, "skills", "global", "core", "do-a", "node_modules", "pkg", "index.js"), "dep\n")
	if code := cli(); code != 0 {
		t.Fatalf("ignored node_modules and .DS_Store must not block (exit %d)\n%s", code, lastOut)
	}
	if err := os.RemoveAll(filepath.Join(clone, "global", "ignored")); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(clone, "global", ".DS_Store")); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(clone, "skills", "global", "core", "do-a", "node_modules")); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"-C", clone, "rm", "-q", "--cached", "global/skills.json"},
		{"-C", clone, "-c", "user.name=gap", "-c", "user.email=gap@example.com", "commit", "-q", "-m", "untrack manifest for ignore"},
		{"-C", clone, "update-ref", "refs/remotes/origin/master", "HEAD"},
	} {
		if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if code := cli(); code != 1 || !strings.Contains(lastOut, "global/") {
		t.Fatalf("an ignored global/skills.json must refuse (exit %d)\n%s", code, lastOut)
	}
	if out, err := exec.Command("git", "-C", clone, "add", "--force", "global/skills.json").CombinedOutput(); err != nil {
		t.Fatalf("restore manifest: %v\n%s", err, out)
	}
	commitMerged("restore ignored manifest")
	if code := cli(); code != 0 {
		t.Fatalf("merged skills must publish (exit %d)\n%s", code, lastOut)
	}

	// An untracked global/skills.json on a merged checkout would install unmerged sources; it must refuse.
	for _, args := range [][]string{
		{"-C", clone, "rm", "-q", "--cached", "global/skills.json"},
		{"-C", clone, "-c", "user.name=gap", "-c", "user.email=gap@example.com", "commit", "-q", "-m", "untrack manifest"},
		{"-C", clone, "update-ref", "refs/remotes/origin/master", "HEAD"},
	} {
		if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if code := cli(); code != 1 || !strings.Contains(lastOut, "global/") {
		t.Fatalf("an untracked global/skills.json must refuse (exit %d)\n%s", code, lastOut)
	}
	if out, err := exec.Command("git", "-C", clone, "add", "--force", "global/skills.json").CombinedOutput(); err != nil {
		t.Fatalf("restore manifest: %v\n%s", err, out)
	}
	commitMerged("restore manifest")

	write(t, filepath.Join(clone, "global", "AGENTS.md"), "unmerged\n")
	if code := cli(); code != 1 {
		t.Fatal("unmerged global/ must refuse")
	}

	// A GIT_DIR inherited from a hook or a dotfiles shell must not point the gate at another repo.
	other := filepath.Join(tmp, "other")
	for _, args := range [][]string{
		{"init", "-q", "-b", "main", other},
		{"-C", other, "-c", "user.name=gap", "-c", "user.email=gap@example.com", "commit", "-q", "--allow-empty", "-m", "base"},
		{"-C", other, "update-ref", "refs/remotes/origin/main", "HEAD"},
		{"-C", other, "symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/main"},
	} {
		if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	extraEnv = []string{"GIT_DIR=" + filepath.Join(other, ".git")}
	if code := cli(); code != 1 {
		t.Error("GIT_DIR bypassed the unmerged refusal")
	}
}

func TestCheckDriftWithoutTerminal(t *testing.T) {
	repo, home := t.TempDir(), t.TempDir()
	t.Setenv("XDG_STATE_HOME", "")
	src := filepath.Join(repo, "skills", "global", "core", "do-x", "SKILL.md")
	installedSkill := filepath.Join(home, ".agents", "skills", "do-x", "SKILL.md")
	extSkill := filepath.Join(home, ".agents", "skills", "ext", "SKILL.md")
	for _, p := range []string{src, installedSkill, extSkill} {
		write(t, p, "same\n")
	}
	if err := os.MkdirAll(filepath.Join(home, ".claude", "skills"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"do-x", "ext"} {
		if err := os.Symlink(filepath.Join(home, ".agents", "skills", name), filepath.Join(home, ".claude", "skills", name)); err != nil {
			t.Fatal(err)
		}
	}
	extHash, err := treeHash(filepath.Dir(extSkill))
	if err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(home, ".agents", ".skill-lock.json"), `{"skills":{"ext":{"source":"owner/ext","sourceUrl":"https://github.com/owner/ext.git","skillFolderHash":"`+extHash+`"}}}`)
	entries := []manifestEntry{{Source: ".", Skills: []string{"do-x"}}, {Source: "owner/ext", Skills: []string{"ext"}}}
	drift := func() bool {
		t.Helper()
		ok, err := checkDrift(repo, home, entries)
		if err != nil {
			t.Fatal(err)
		}
		return ok
	}

	if !drift() {
		t.Fatal("matching skills must not drift")
	}
	write(t, installedSkill, "hand edit\n")
	if drift() || readString(t, installedSkill) != "hand edit\n" {
		t.Fatal("an edited local skill must be reported and left unchanged")
	}
	if drift() {
		t.Fatal("a later run must report the edit again")
	}
	write(t, extSkill, "external edit\n")
	if drift() || readString(t, extSkill) != "external edit\n" {
		t.Fatal("an external skill that no longer matches the skills.sh lock must be reported and left unchanged")
	}
	write(t, installedSkill, "same\n")
	write(t, extSkill, "same\n")
	mirror := skillDirs(home, "do-x")[1]
	if err := os.Remove(mirror); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(mirror, "SKILL.md"), "mirror edit\n")
	if drift() || readString(t, filepath.Join(mirror, "SKILL.md")) != "mirror edit\n" {
		t.Fatal("an independent Claude copy must be checked and protected")
	}
	write(t, filepath.Join(mirror, "SKILL.md"), "same\n")
	write(t, filepath.Join(mirror, ".gitignore"), "ignored\n")
	write(t, filepath.Join(filepath.Dir(src), ".gitignore"), "ignored\n")
	write(t, filepath.Join(filepath.Dir(installedSkill), ".gitignore"), "ignored\n")
	write(t, filepath.Join(mirror, "ignored"), "hidden edit\n")
	if drift() {
		t.Fatal("ignored installed content must count as drift")
	}
}

const empryoFixture = "user intro\n\n<!-- soulstack:start -->\n# Takeaways\nkeep this\n<!-- soulstack:end -->\n\nuser outro\n"

func TestEmpryoDestination(t *testing.T) {
	want := filepath.Join("/h", ".empryo", "EMPRYO.md")
	for _, d := range policyDests("/h", policyPath) {
		if d.dest == want {
			if d.cursor {
				t.Fatal("Empryo must not get the Cursor frontmatter")
			}
			return
		}
	}
	t.Fatalf("policyDests misses %s", want)
}

func TestEmpryoSpliceKeepsSoulStack(t *testing.T) {
	home := t.TempDir()
	dest := filepath.Join(home, ".empryo", "EMPRYO.md")
	write(t, dest, empryoFixture)
	ledgerFile := filepath.Join(home, ".gap", "published.json")
	ledger := map[string]string{}
	withYes(t)
	wrote, err := publishPolicy(source, dest, false, ledger, ledgerFile, true, home)
	if err != nil || !wrote {
		t.Fatalf("splice: wrote %v err %v", wrote, err)
	}
	body := adapt(source, false)
	first := readString(t, dest)
	if first != empryoFixture+gapMarked(body) {
		t.Fatalf("splice changed bytes outside the gap block: %q", first)
	}
	sum := ledger[dest]
	termReady = func() bool { return false }
	wrote, err = publishPolicy(source, dest, false, ledger, ledgerFile, true, home)
	if err != nil || !wrote {
		t.Fatalf("re-sync: wrote %v err %v", wrote, err)
	}
	if got := readString(t, dest); got != first {
		t.Fatalf("re-sync changed the file: %q", got)
	}
	saved, err := readLedger(ledgerFile)
	if err != nil || saved[dest] != sum || sum != digest(body) {
		t.Fatalf("ledger digest not stable: %q %q %v", sum, saved[dest], err)
	}
}

func TestEmpryoForeignRefusedWithoutTerminal(t *testing.T) {
	home := t.TempDir()
	codex := filepath.Join(home, ".codex", "gap", "global-agent-policy.md")
	empryo := filepath.Join(home, ".empryo", "EMPRYO.md")
	write(t, empryo, empryoFixture)
	old := termReady
	termReady = func() bool { return false }
	t.Cleanup(func() { termReady = old })
	ledgerFile := filepath.Join(home, ".gap", "published.json")
	ledger := map[string]string{}
	for _, d := range policyDests(home, codex) {
		wrote, err := publishPolicy(source, d.dest, d.cursor, ledger, ledgerFile, true, home)
		if err != nil {
			t.Fatal(err)
		}
		if wrote == (d.dest == empryo) {
			t.Fatalf("%s: wrote %v", d.dest, wrote)
		}
	}
	if readString(t, empryo) != empryoFixture {
		t.Fatal("foreign EMPRYO.md was changed without a terminal")
	}
}

func TestEmpryoPinAdvice(t *testing.T) {
	for _, tc := range []struct {
		name, config string
		advise       bool
	}{
		{"no Empryo", "", false},
		{"pins match", `{"skills":{"autoLoad":["do-todo","do-commit","but-axi"]}}`, false},
		{"same pins, other order", `{"skills":{"autoLoad":["but-axi","do-todo","do-commit"]}}`, false},
		{"a subset", `{"skills":{"autoLoad":["do-todo","do-commit"]}}`, true},
		{"35 pins", `{"skills":{"autoLoad":["do-commit","do-art","do-todo"]}}`, true},
		{"no pins", `{"theme":{}}`, true},
		{"unreadable", `{`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			path := filepath.Join(home, ".empryo", "config.json")
			if tc.config != "" {
				write(t, path, tc.config)
			}
			got := empryoPinAdvice(home)
			if (got != "") != tc.advise {
				t.Fatalf("advice = %q, want %v", got, tc.advise)
			}
			if tc.advise && !strings.Contains(got, `value:'["do-todo","do-commit","but-axi"]', scope:"global"`) {
				t.Fatalf("advice must carry the exact settings call: %q", got)
			}
			if tc.config != "" && readString(t, path) != tc.config {
				t.Fatal("gap must never write Empryo's config")
			}
		})
	}
}
