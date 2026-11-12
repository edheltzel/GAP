//go:build unix

package main

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestManifestMCP(t *testing.T) {
	for _, tc := range []struct {
		name, toml, err string
	}{
		{"command", "[mcp.srv]\ncommand = [\"bun\", \"{store}/s.ts\"]\nenv = [\"A=1\"]\n", ""},
		{"url", "[mcp.srv]\nurl = \"https://example.com/mcp\"\n", ""},
		{"neither", "[mcp.srv]\nenv = [\"A=1\"]\n", "exactly one of command or url"},
		{"both", "[mcp.srv]\ncommand = [\"x\"]\nurl = \"https://e.com\"\n", "exactly one of command or url"},
		{"bad name", "[mcp.Srv]\ncommand = [\"x\"]\n", "lowercase"},
		{"bad env", "[mcp.srv]\ncommand = [\"x\"]\nenv = [\"NOEQ\"]\n", "KEY=value"},
		{"url env", "[mcp.srv]\nurl = \"https://e.com\"\nenv = [\"A=1\"]\n", "command servers only"},
		{"bad url", "[mcp.srv]\nurl = \"file:///etc/passwd\"\n", "http or https"},
		{"unknown", "[mcp.srv]\ncommand = [\"x\"]\nshell = true\n", "unknown key shell"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := parseTOML(tc.toml)
			if err != nil {
				t.Fatal(err)
			}
			got, err := manifestMCP(doc)
			if tc.err == "" {
				if err != nil || len(got) != 1 || got[0].Name != "srv" {
					t.Fatalf("got %+v, %v", got, err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.err) {
				t.Fatalf("error = %v, want %q", err, tc.err)
			}
		})
	}
}

const mcpUtility = "schema = \"1\"\nname = \"util\"\n[mcp.util-mcp]\ncommand = [\"{store}/server.sh\", \"--stdio\"]\nenv = [\"TOKEN=x\"]\n"

// fakeMCPBins puts claude, codex, and pi stand-ins first on PATH. Each logs its arguments.
// codex mcp get <name> succeeds when $HOME/codex-has-<name> exists, like a configured server.
func fakeMCPBins(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	for _, bin := range []string{"claude", "codex", "pi"} {
		script := "#!/bin/sh\nif [ \"$1 $2\" = \"mcp get\" ]; then [ -e \"$HOME/codex-has-$3\" ]; exit $?; fi\necho \"" + bin + " $*\" >> \"$HOME/mcp.log\"\n"
		if err := os.WriteFile(filepath.Join(dir, bin), []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func mcpLog(t *testing.T, home string) []string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(home, "mcp.log"))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	os.Remove(filepath.Join(home, "mcp.log"))
	return strings.Split(strings.TrimSpace(string(b)), "\n")
}

func TestUtilityMCPLifecycle(t *testing.T) {
	e := newUtilityEnv(t, mcpUtility)
	fakeMCPBins(t)
	quietStdout(t)
	write(t, userConfigFile(e.home), "[trust]\nrepos = [\""+e.remote+"\"]\n")
	if code := run([]string{"add", e.remote, "--id", "util"}); code != 0 {
		t.Fatalf("add exit %d", code)
	}
	state, err := readState(e.home)
	if err != nil {
		t.Fatal(err)
	}
	lk, err := readLock(userConfigDir(e.home))
	if err != nil {
		t.Fatal(err)
	}
	store := storePath(e.home, "util", lk.Components["util"].Commit)
	want := []string{
		"claude mcp add --scope user util-mcp -e TOKEN=x -- " + store + "/server.sh --stdio",
		"pi mcp add util-mcp --env TOKEN=x -- " + store + "/server.sh --stdio",
		"codex mcp add util-mcp --env TOKEN=x -- " + store + "/server.sh --stdio",
	}
	got := mcpLog(t, e.home)
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Fatalf("add calls:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	if keys := state.Components["util"].MCP; len(keys) != 3 {
		t.Fatalf("state mcp = %v", keys)
	}
	write(t, filepath.Join(e.remote, "version"), "next")
	commitAll(t, e.remote)
	if code := run([]string{"update", "util"}); code != 0 {
		t.Fatalf("update exit %d", code)
	}
	lk, err = readLock(userConfigDir(e.home))
	if err != nil {
		t.Fatal(err)
	}
	next := storePath(e.home, "util", lk.Components["util"].Commit)
	got = mcpLog(t, e.home)
	if len(got) != 6 || !strings.Contains(strings.Join(got, "\n"), next+"/server.sh") {
		t.Fatalf("update should re-register each server at the new store:\n%s", strings.Join(got, "\n"))
	}
	if code := run([]string{"update", "util"}); code != 0 {
		t.Fatalf("second update exit %d", code)
	}
	if got := mcpLog(t, e.home); got != nil {
		t.Fatalf("unchanged servers were registered again: %v", got)
	}
	if got := mcpLog(t, e.home); got != nil {
		t.Fatalf("unchanged servers were registered again: %v", got)
	}
	if code := run([]string{"remove", "util"}); code != 0 {
		t.Fatalf("remove exit %d", code)
	}
	got = mcpLog(t, e.home)
	slices.Sort(got)
	want = []string{"claude mcp remove --scope user util-mcp", "codex mcp remove util-mcp", "pi mcp remove util-mcp"}
	if !slices.Equal(got, want) {
		t.Fatalf("remove calls: %v", got)
	}
}

func TestUtilityMCPLeavesUserServers(t *testing.T) {
	e := newUtilityEnv(t, mcpUtility)
	fakeMCPBins(t)
	quietStdout(t)
	write(t, userConfigFile(e.home), "[trust]\nrepos = [\""+e.remote+"\"]\n")
	write(t, filepath.Join(e.home, ".claude.json"), "{\"mcpServers\":{\"util-mcp\":{\"command\":\"mine\"}}}\n")
	write(t, filepath.Join(e.home, "codex-has-util-mcp"), "")
	if code := run([]string{"add", e.remote, "--id", "util"}); code == 0 {
		t.Fatal("add reported success while two servers were skipped")
	}
	got := mcpLog(t, e.home)
	if len(got) != 1 || !strings.HasPrefix(got[0], "pi mcp add util-mcp") {
		t.Fatalf("only pi should be registered: %v", got)
	}
	state, _ := readState(e.home)
	if _, ok := state.Components["util"].MCP["pi:util-mcp"]; !ok || len(state.Components["util"].MCP) != 1 {
		t.Fatalf("state mcp = %v", state.Components["util"].MCP)
	}
	if code := run([]string{"remove", "util"}); code != 0 {
		t.Fatalf("remove exit %d", code)
	}
	if got := mcpLog(t, e.home); len(got) != 1 || got[0] != "pi mcp remove util-mcp" {
		t.Fatalf("remove touched servers gap did not add: %v", got)
	}
}

func TestUtilityMCPNeedsApproval(t *testing.T) {
	e := newUtilityEnv(t, mcpUtility)
	fakeMCPBins(t)
	quietStdout(t)
	run([]string{"add", e.remote, "--id", "util", "--yes"})
	if got := mcpLog(t, e.home); got != nil {
		t.Fatalf("untrusted utility registered servers without approval: %v", got)
	}
	state, _ := readState(e.home)
	if len(state.Components["util"].MCP) != 0 {
		t.Fatalf("state mcp = %v", state.Components["util"].MCP)
	}
}
