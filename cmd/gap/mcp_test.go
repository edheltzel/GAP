//go:build unix

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
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

func fakeMCPBins(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	git, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(git, filepath.Join(dir, "git")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GAP_MCP_TEST_BINARY", binary)
	for _, key := range []string{"CLAUDE_CONFIG_DIR", "CODEX_HOME", "PI_CODING_AGENT_DIR"} {
		t.Setenv(key, "")
	}
	for _, bin := range []string{"claude", "codex", "pi"} {
		script := "#!/bin/sh\nexec \"$GAP_MCP_TEST_BINARY\" -test.run='^TestMCPCLIProcess$' -- " + bin + " \"$@\"\n"
		if err := os.WriteFile(filepath.Join(dir, bin), []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", dir)
	return dir
}

type testMCPConfig struct {
	Servers map[string]testMCPServer `json:"mcpServers"`
}

type testMCPServer struct {
	Command string `json:"command,omitempty"`
	URL     string `json:"url,omitempty"`
}

func TestMCPCLIProcess(t *testing.T) {
	i := slices.Index(os.Args, "--")
	if i < 0 {
		return
	}
	bin, args := os.Args[i+1], os.Args[i+2:]
	home := os.Getenv("HOME")
	target, file := "", ""
	switch bin {
	case "claude":
		target, file = os.Getenv("CLAUDE_CONFIG_DIR"), ".claude.json"
		if target == "" {
			target = home
		}
	case "codex":
		target, file = os.Getenv("CODEX_HOME"), "servers.json"
		if target == "" {
			target = filepath.Join(home, ".codex")
		}
	case "pi":
		target, file = os.Getenv("PI_CODING_AGENT_DIR"), "mcp.json"
		if target == "" {
			target = filepath.Join(home, ".pi", "agent")
		}
	default:
		os.Exit(2)
	}
	path := filepath.Join(target, file)
	config := testMCPConfig{Servers: map[string]testMCPServer{}}
	b, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		os.Exit(2)
	}
	if len(b) > 0 && json.Unmarshal(b, &config) != nil {
		os.Exit(2)
	}
	if len(args) < 2 || args[0] != "mcp" {
		os.Exit(2)
	}
	action := args[1]
	if action == "list" {
		servers := []map[string]string{}
		for name := range config.Servers {
			servers = append(servers, map[string]string{"name": name})
		}
		json.NewEncoder(os.Stdout).Encode(servers)
		os.Exit(0)
	}

	log, err := os.OpenFile(filepath.Join(home, "mcp.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		os.Exit(2)
	}
	fmt.Fprintln(log, bin+" "+strings.Join(args, " "))
	log.Close()
	if _, err := os.Stat(filepath.Join(home, "fail-"+bin+"-"+action+"-before")); err == nil {
		os.Exit(1)
	}
	n := 2
	for n < len(args) && strings.HasPrefix(args[n], "--") {
		n += 2
	}
	if n >= len(args) {
		os.Exit(2)
	}
	name := args[n]
	switch action {
	case "add":
		server := testMCPServer{}
		if j := slices.Index(args, "--"); j >= 0 && j+1 < len(args) {
			server.Command = args[j+1]
		}
		if j := slices.Index(args, "--url"); j >= 0 && j+1 < len(args) {
			server.URL = args[j+1]
		}
		if bin == "claude" && slices.Contains(args, "http") {
			server.URL = args[n+1]
		}
		config.Servers[name] = server
	case "remove":
		if _, ok := config.Servers[name]; !ok {
			os.Exit(1)
		}
		delete(config.Servers, name)
	default:
		os.Exit(2)
	}
	if err := os.MkdirAll(target, 0o755); err != nil {
		os.Exit(2)
	}
	b, err = json.Marshal(config)
	if err != nil || os.WriteFile(path, b, 0o600) != nil {
		os.Exit(2)
	}
	if _, err := os.Stat(filepath.Join(home, "fail-"+bin+"-"+action+"-after")); err == nil {
		os.Exit(1)
	}
	os.Exit(0)
}

func testMCPServers(t *testing.T, path string) map[string]testMCPServer {
	t.Helper()
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	var config testMCPConfig
	if err := json.Unmarshal(b, &config); err != nil {
		t.Fatal(err)
	}
	return config.Servers
}

func testMCPState(t *testing.T, home string) stateFile {
	t.Helper()
	state, err := readState(home)
	if err != nil {
		t.Fatal(err)
	}
	return state
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
	write(t, filepath.Join(e.home, ".codex", "servers.json"), `{ "mcpServers": { "util-mcp": { "command": "mine" } } }`)
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

func TestUtilityMCPTargetOwnership(t *testing.T) {
	for _, tc := range []struct{ bin, env, dir, file string }{
		{"claude", "CLAUDE_CONFIG_DIR", "", ".claude.json"},
		{"codex", "CODEX_HOME", ".codex", "servers.json"},
		{"pi", "PI_CODING_AGENT_DIR", ".pi/agent", "mcp.json"},
	} {
		for _, custom := range []bool{false, true} {
			for _, action := range []string{"remove", "drop"} {
				t.Run(fmt.Sprintf("%s/custom=%t/%s", tc.bin, custom, action), func(t *testing.T) {
					e := newUtilityEnv(t, mcpUtility)
					fakeMCPBins(t)
					quietStdout(t)
					target := filepath.Join(e.home, tc.dir)
					if custom {
						target = filepath.Join(e.home, "original-config")
						t.Setenv(tc.env, target)
					}
					write(t, userConfigFile(e.home), fmt.Sprintf("[trust]\nrepos = [%q]\n", e.remote))
					if code := run([]string{"add", e.remote, "--id", "util"}); code != 0 {
						t.Fatalf("add exit %d", code)
					}
					key := tc.bin + ":util-mcp"
					if got := testMCPState(t, e.home).Components["util"].MCP[key].Target; got != target {
						t.Fatalf("owned target = %q, want %q", got, target)
					}
					foreign := filepath.Join(e.home, "different-config")
					path := filepath.Join(foreign, tc.file)
					const mine = `{"mcpServers":{"util-mcp":{"command":"mine"}}}`
					write(t, path, mine)
					t.Setenv(tc.env, foreign)
					write(t, filepath.Join(e.remote, "version"), "next")
					commitAll(t, e.remote)
					flag := filepath.Join(e.home, "fail-"+tc.bin+"-add-before")
					write(t, flag, "")
					if code := run([]string{"update", "util"}); code == 0 {
						t.Fatal("failed replacement succeeded")
					}
					if got := testMCPState(t, e.home).Components["util"].MCP[key].Target; got != target {
						t.Fatalf("failed replacement lost target: %q", got)
					}
					if err := os.Remove(flag); err != nil {
						t.Fatal(err)
					}
					if code := run([]string{"update", "util"}); code != 0 {
						t.Fatalf("update retry exit %d", code)
					}
					lk, err := readLock(userConfigDir(e.home))
					if err != nil {
						t.Fatal(err)
					}
					want := storePath(e.home, "util", lk.Components["util"].Commit) + "/server.sh"
					if got := testMCPServers(t, filepath.Join(target, tc.file))["util-mcp"].Command; got != want {
						t.Fatalf("original server = %q, want %q", got, want)
					}
					if action == "drop" {
						write(t, filepath.Join(e.remote, utilityManifestName), "schema = \"1\"\nname = \"util\"\n[hooks]\ndoctor = \"true\"\n")
						commitAll(t, e.remote)
						if code := run([]string{"update", "util"}); code != 0 {
							t.Fatalf("drop exit %d", code)
						}
					} else if code := run([]string{"remove", "util"}); code != 0 {
						t.Fatalf("remove exit %d", code)
					}
					if _, exists := testMCPServers(t, filepath.Join(target, tc.file))["util-mcp"]; exists {
						t.Fatal("original target still has the managed server")
					}
					if got := readString(t, path); got != mine {
						t.Fatalf("foreign configuration changed: %s", got)
					}
				})
			}
		}
	}
}

func TestUtilityMCPForeignOverride(t *testing.T) {
	for _, tc := range []struct{ bin, env, file string }{
		{"claude", "CLAUDE_CONFIG_DIR", ".claude.json"},
		{"codex", "CODEX_HOME", "servers.json"},
		{"pi", "PI_CODING_AGENT_DIR", "mcp.json"},
	} {
		t.Run(tc.bin, func(t *testing.T) {
			e := newUtilityEnv(t, mcpUtility)
			fakeMCPBins(t)
			quietStdout(t)
			target := filepath.Join(e.home, "custom-config")
			t.Setenv(tc.env, target)
			path := filepath.Join(target, tc.file)
			const mine = `{"mcpServers":{"util-mcp":{"command":"mine"}}}`
			write(t, path, mine)
			write(t, userConfigFile(e.home), fmt.Sprintf("[trust]\nrepos = [%q]\n", e.remote))
			if code := run([]string{"add", e.remote, "--id", "util"}); code == 0 {
				t.Fatal("foreign server was accepted")
			}
			if got := readString(t, path); got != mine {
				t.Fatalf("foreign server changed: %s", got)
			}
			if _, owns := testMCPState(t, e.home).Components["util"].MCP[tc.bin+":util-mcp"]; owns {
				t.Fatal("foreign server was claimed")
			}
		})
	}
}

func TestUtilityMCPRemovalRetry(t *testing.T) {
	for _, failure := range []string{"missing-cli", "remove-error"} {
		t.Run(failure, func(t *testing.T) {
			e := newUtilityEnv(t, mcpUtility)
			dir := fakeMCPBins(t)
			quietStdout(t)
			write(t, userConfigFile(e.home), fmt.Sprintf("[trust]\nrepos = [%q]\n", e.remote))
			if code := run([]string{"add", e.remote, "--id", "util"}); code != 0 {
				t.Fatalf("add exit %d", code)
			}
			config := readString(t, userConfigFile(e.home))
			lock := readString(t, filepath.Join(userConfigDir(e.home), "gap.lock"))
			bin := filepath.Join(dir, "codex")
			flag := filepath.Join(e.home, "fail-codex-remove-before")
			if failure == "missing-cli" {
				if err := os.Rename(bin, bin+".disabled"); err != nil {
					t.Fatal(err)
				}
			} else {
				write(t, flag, "")
			}
			mcpLog(t, e.home)
			if code := run([]string{"remove", "util"}); code == 0 {
				t.Fatal("incomplete removal succeeded")
			}
			entry := testMCPState(t, e.home).Components["util"]
			if len(entry.MCP) != 1 || entry.MCP["codex:util-mcp"].Target == "" {
				t.Fatalf("retry ownership = %+v", entry.MCP)
			}
			if got := readString(t, userConfigFile(e.home)); got != config {
				t.Fatal("failed removal dropped utility configuration")
			}
			if got := readString(t, filepath.Join(userConfigDir(e.home), "gap.lock")); got != lock {
				t.Fatal("failed removal dropped the pin")
			}
			if _, exists := testMCPServers(t, filepath.Join(e.home, ".codex", "servers.json"))["util-mcp"]; !exists {
				t.Fatal("failed server disappeared")
			}
			for _, path := range []string{filepath.Join(e.home, ".claude.json"), filepath.Join(e.home, ".pi", "agent", "mcp.json")} {
				if _, exists := testMCPServers(t, path)["util-mcp"]; exists {
					t.Fatal("successful sibling removal was not applied")
				}
			}
			if failure == "missing-cli" {
				if err := os.Rename(bin+".disabled", bin); err != nil {
					t.Fatal(err)
				}
			} else if err := os.Remove(flag); err != nil {
				t.Fatal(err)
			}
			mcpLog(t, e.home)
			if code := run([]string{"remove", "util"}); code != 0 {
				t.Fatalf("retry exit %d", code)
			}
			if got := mcpLog(t, e.home); !slices.Equal(got, []string{"codex mcp remove util-mcp"}) {
				t.Fatalf("retry repeated successful removals: %v", got)
			}
			if _, exists := testMCPState(t, e.home).Components["util"]; exists {
				t.Fatal("retry retained utility state")
			}
		})
	}
}

func TestUtilityMCPAddFailure(t *testing.T) {
	for _, stage := range []string{"initial", "replacement"} {
		for _, timing := range []string{"before", "after"} {
			t.Run(stage+"/"+timing, func(t *testing.T) {
				e := newUtilityEnv(t, "schema = \"1\"\nname = \"util\"\n[mcp.util-mcp]\nurl = \"https://example.com/mcp\"\n")
				fakeMCPBins(t)
				quietStdout(t)
				write(t, userConfigFile(e.home), fmt.Sprintf("[trust]\nrepos = [%q]\n", e.remote))
				if stage == "replacement" {
					if code := run([]string{"add", e.remote, "--id", "util"}); code != 0 {
						t.Fatalf("add exit %d", code)
					}
					write(t, filepath.Join(e.remote, utilityManifestName), "schema = \"1\"\nname = \"util\"\n[mcp.util-mcp]\nurl = \"https://example.com/next\"\n")
					commitAll(t, e.remote)
				}
				flag := filepath.Join(e.home, "fail-codex-add-"+timing)
				write(t, flag, "")
				args := []string{"add", e.remote, "--id", "util"}
				if stage == "replacement" {
					args = []string{"update", "util"}
				}
				if code := run(args); code == 0 {
					t.Fatal("failed add reported success")
				}
				entry := testMCPState(t, e.home).Components["util"]
				_, owns := entry.MCP["codex:util-mcp"]
				server, exists := testMCPServers(t, filepath.Join(e.home, ".codex", "servers.json"))["util-mcp"]
				if owns != (timing == "after" || stage == "replacement") || exists != (timing == "after") {
					t.Fatalf("ownership=%t, server exists=%t after %s failure", owns, exists, timing)
				}
				lk, err := readLock(userConfigDir(e.home))
				if err != nil {
					t.Fatal(err)
				}
				if entry.Installed == lk.Components["util"].Commit {
					t.Fatal("failed registration marked the pin installed")
				}
				if exists {
					want := "https://example.com/mcp"
					if stage == "replacement" {
						want = "https://example.com/next"
					}
					if server.URL != want {
						t.Fatalf("registered URL = %q, want %q", server.URL, want)
					}
				}
				if err := os.Remove(flag); err != nil {
					t.Fatal(err)
				}
				if code := run([]string{"update", "util"}); code != 0 {
					t.Fatalf("retry exit %d", code)
				}
				if got := testMCPState(t, e.home).Components["util"].Installed; got != lk.Components["util"].Commit {
					t.Fatal("successful retry did not complete installation")
				}
				if code := run([]string{"remove", "util"}); code != 0 {
					t.Fatalf("remove exit %d", code)
				}
				if _, exists := testMCPServers(t, filepath.Join(e.home, ".codex", "servers.json"))["util-mcp"]; exists {
					t.Fatal("partially successful add escaped removal")
				}
			})
		}
	}
}

func TestUtilityMCPRepairsMissingServer(t *testing.T) {
	for _, tc := range []struct {
		bin  string
		args []string
		path string
	}{
		{"claude", []string{"mcp", "remove", "--scope", "user", "util-mcp"}, ".claude.json"},
		{"codex", []string{"mcp", "remove", "util-mcp"}, ".codex/servers.json"},
		{"pi", []string{"mcp", "remove", "util-mcp"}, ".pi/agent/mcp.json"},
	} {
		t.Run(tc.bin, func(t *testing.T) {
			e := newUtilityEnv(t, mcpUtility)
			fakeMCPBins(t)
			quietStdout(t)
			write(t, userConfigFile(e.home), fmt.Sprintf("[trust]\nrepos = [%q]\n", e.remote))
			if code := run([]string{"add", e.remote, "--id", "util"}); code != 0 {
				t.Fatalf("add exit %d", code)
			}
			cmd := exec.Command(tc.bin, tc.args...)
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("manual remove: %v %s", err, out)
			}
			mcpLog(t, e.home)
			if code := run([]string{"update", "util"}); code != 0 {
				t.Fatalf("repair exit %d", code)
			}
			if got := mcpLog(t, e.home); len(got) != 1 || !strings.HasPrefix(got[0], tc.bin+" mcp add ") {
				t.Fatalf("repair commands = %v", got)
			}
			if testMCPServers(t, filepath.Join(e.home, tc.path))["util-mcp"].Command == "" {
				t.Fatal("server was not restored")
			}
		})
	}
}

func TestUtilityMCPDeclinedInstallLifecycle(t *testing.T) {
	for _, command := range []string{"add", "update", "sync"} {
		t.Run(command, func(t *testing.T) {
			e := newUtilityEnv(t, mcpUtility+"[hooks]\nuninstall = \"printf ran > \\\"$HOME/uninstalled\\\"\"\n")
			fakeMCPBins(t)
			quietStdout(t)
			if code := run([]string{"add", e.remote, "--id", "util"}); code == 0 {
				t.Fatal("unapproved MCP installation succeeded")
			}
			if command == "update" {
				if code := run([]string{"update", "util"}); code == 0 {
					t.Fatal("unapproved update succeeded")
				}
			}
			if command == "sync" {
				write(t, filepath.Join(e.repo, "global", "skills.json"), "[]\n")
				commitAll(t, e.repo)
				for _, args := range [][]string{
					{"-C", e.repo, "update-ref", "refs/remotes/origin/master", "HEAD"},
					{"-C", e.repo, "symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/master"},
				} {
					cmd := exec.Command("git", args...)
					cmd.Env = gitEnv()
					if out, err := cmd.CombinedOutput(); err != nil {
						t.Fatalf("sync fixture: %v %s", err, out)
					}
				}
				if code := run([]string{"sync", "--only", "util", "--yes"}); code == 0 {
					t.Fatal("unapproved sync succeeded")
				}
			}
			if entry := testMCPState(t, e.home).Components["util"]; entry.Installed != "" || len(entry.MCP) != 0 {
				t.Fatalf("declined installation state = %+v", entry)
			}
			if code := run([]string{"remove", "util"}); code != 0 {
				t.Fatalf("remove ran an unapproved uninstall hook: exit %d", code)
			}
			if e.exists("uninstalled") {
				t.Fatal("uninstall ran without a successful installation")
			}
		})
	}
}

func TestUtilityMCPUnreadableConfig(t *testing.T) {
	for _, tc := range []struct{ bin, path string }{
		{"claude", ".claude.json"},
		{"codex", ".codex/servers.json"},
		{"pi", ".pi/agent/mcp.json"},
	} {
		t.Run(tc.bin, func(t *testing.T) {
			e := newUtilityEnv(t, mcpUtility)
			fakeMCPBins(t)
			quietStdout(t)
			path := filepath.Join(e.home, tc.path)
			write(t, path, "{")
			write(t, userConfigFile(e.home), fmt.Sprintf("[trust]\nrepos = [%q]\n", e.remote))
			if code := run([]string{"add", e.remote, "--id", "util"}); code == 0 {
				t.Fatal("unreadable config was treated as an empty target")
			}
			if got := readString(t, path); got != "{" {
				t.Fatalf("unreadable config changed: %s", got)
			}
			entry := testMCPState(t, e.home).Components["util"]
			if _, owns := entry.MCP[tc.bin+":util-mcp"]; owns || entry.Installed != "" {
				t.Fatalf("failed inspection recorded an installation: %+v", entry)
			}
		})
	}
}
