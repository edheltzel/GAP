package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

// mcpServer is one [mcp.<name>] table in gap-utility.toml: a stdio command or an HTTP url.
// {store} in a command or env value becomes the utility's pinned store path.
type mcpServer struct {
	Name    string
	Command []string
	URL     string
	Env     []string
}

// mcpCLI registers servers through a harness's own mcp command, so GAP never edits that harness's config by hand.
type mcpCLI struct {
	bin    string
	add    func(s mcpServer) []string
	remove func(name string) []string
	exists func(home, name string) bool
}

func manifestMCP(doc map[string]any) ([]mcpServer, error) {
	raw, ok := doc["mcp"]
	if !ok {
		return nil, nil
	}
	table, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("mcp must be a table of servers")
	}
	var servers []mcpServer
	for name, value := range table {
		if strings.HasPrefix(name, "\x00") {
			continue
		}
		if !validID(name) {
			return nil, fmt.Errorf("mcp server name %q must be lowercase letters, digits, and hyphens", name)
		}
		spec, ok := value.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("mcp.%s must be a table", name)
		}
		s := mcpServer{Name: name}
		for key, v := range spec {
			if strings.HasPrefix(key, "\x00") {
				continue
			}
			switch key {
			case "command":
				if s.Command, ok = v.([]string); !ok || len(s.Command) == 0 || s.Command[0] == "" {
					return nil, fmt.Errorf("mcp.%s.command must be a nonempty string array", name)
				}
			case "url":
				if s.URL, ok = v.(string); !ok || !strings.HasPrefix(s.URL, "https://") && !strings.HasPrefix(s.URL, "http://") {
					return nil, fmt.Errorf("mcp.%s.url must be an http or https url", name)
				}
			case "env":
				if s.Env, ok = v.([]string); !ok {
					return nil, fmt.Errorf("mcp.%s.env must be a string array", name)
				}
				for _, kv := range s.Env {
					if k, _, found := strings.Cut(kv, "="); !found || k == "" {
						return nil, fmt.Errorf("mcp.%s.env entry %q must be KEY=value", name, kv)
					}
				}
			default:
				return nil, fmt.Errorf("mcp.%s has unknown key %s", name, key)
			}
		}
		if (len(s.Command) > 0) == (s.URL != "") {
			return nil, fmt.Errorf("mcp.%s needs exactly one of command or url", name)
		}
		if s.URL != "" && len(s.Env) > 0 {
			return nil, fmt.Errorf("mcp.%s: env applies to command servers only", name)
		}
		servers = append(servers, s)
	}
	slices.SortFunc(servers, func(a, b mcpServer) int { return strings.Compare(a.Name, b.Name) })
	return servers, nil
}

func (s mcpServer) expand(store string) mcpServer {
	sub := func(list []string) []string {
		out := make([]string, len(list))
		for i, v := range list {
			out[i] = strings.ReplaceAll(v, "{store}", store)
		}
		return out
	}
	s.Command, s.Env = sub(s.Command), sub(s.Env)
	return s
}

func (s mcpServer) describe() string {
	if s.URL != "" {
		return s.Name + " -> " + s.URL
	}
	return s.Name + " -> " + strings.Join(s.Command, " ")
}

func stdioArgs(head []string, envFlag string, s mcpServer) []string {
	args := slices.Clone(head)
	for _, kv := range s.Env {
		args = append(args, envFlag, kv)
	}
	return append(append(args, "--"), s.Command...)
}

func jsonServerExists(path string) func(home, name string) bool {
	return func(home, name string) bool {
		b, err := os.ReadFile(filepath.Join(home, path))
		if err != nil {
			return false
		}
		var doc struct {
			MCPServers map[string]json.RawMessage `json:"mcpServers"`
		}
		if json.Unmarshal(b, &doc) != nil {
			return false
		}
		_, ok := doc.MCPServers[name]
		return ok
	}
}

var claudeMCP = &mcpCLI{
	bin: "claude",
	add: func(s mcpServer) []string {
		if s.URL != "" {
			return []string{"mcp", "add", "--scope", "user", "--transport", "http", s.Name, s.URL}
		}
		return stdioArgs([]string{"mcp", "add", "--scope", "user", s.Name}, "-e", s)
	},
	remove: func(name string) []string { return []string{"mcp", "remove", "--scope", "user", name} },
	exists: jsonServerExists(".claude.json"),
}

var codexMCP = &mcpCLI{
	bin: "codex",
	add: func(s mcpServer) []string {
		if s.URL != "" {
			return []string{"mcp", "add", s.Name, "--url", s.URL}
		}
		return stdioArgs([]string{"mcp", "add", s.Name}, "--env", s)
	},
	remove: func(name string) []string { return []string{"mcp", "remove", name} },
	exists: func(home, name string) bool {
		cmd := exec.Command("codex", "mcp", "get", name)
		cmd.Env = append(os.Environ(), "HOME="+home)
		return cmd.Run() == nil
	},
}

var piMCP = &mcpCLI{
	bin: "pi",
	add: func(s mcpServer) []string {
		if s.URL != "" {
			return []string{"mcp", "add", s.Name, "--url", s.URL}
		}
		return stdioArgs([]string{"mcp", "add", s.Name}, "--env", s)
	},
	remove: func(name string) []string { return []string{"mcp", "remove", name} },
	exists: jsonServerExists(filepath.Join(".pi", "agent", "mcp.json")),
}

func runMCP(home, bin string, args []string) error {
	cmd := exec.Command(bin, args...)
	cmd.Env = append(os.Environ(), "HOME="+home)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s %s: %w: %s", bin, strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return nil
}

func mcpKey(h harness, name string) string { return h.name + ":" + name }

func mcpDigest(s mcpServer) string {
	b, _ := json.Marshal(s)
	return digest(string(b))
}

// publishComponentMCP registers each selected component's MCP servers with every harness whose mcp CLI is installed.
// A server name another component owns, or one the user already configured, is left alone and reported.
func publishComponentMCP(home string, d distro) (bool, error) {
	state, err := readState(home)
	if err != nil {
		return false, err
	}
	gate, err := newHookGate(home)
	if err != nil {
		return false, err
	}
	owner := map[string]string{}
	for id, entry := range state.Components {
		for key := range entry.MCP {
			owner[key] = id
		}
	}
	ok, changed := true, false
	out := newReport(os.Stdout)
	for _, id := range sortedIDs(d.Components) {
		c := d.Components[id]
		entry := state.Components[id]
		if c.Store == "" || len(c.MCP) == 0 && len(entry.MCP) == 0 {
			continue
		}
		type job struct {
			h    harness
			s    mcpServer
			key  string
			sum  string
			owns bool
		}
		var jobs []job
		var stale []string
		for _, h := range harnesses {
			if h.mcp == nil {
				continue
			}
			if _, err := exec.LookPath(h.mcp.bin); err != nil {
				continue
			}
			for _, raw := range c.MCP {
				s := raw.expand(c.Store)
				key, sum := mcpKey(h, s.Name), mcpDigest(s)
				if prev, mine := entry.MCP[key]; mine {
					if prev != sum {
						jobs = append(jobs, job{h, s, key, sum, true})
					}
					continue
				}
				if other := owner[key]; other != "" && other != id {
					fmt.Fprintf(os.Stderr, "%s: mcp server %s in %s belongs to %s; skipped\n", id, s.Name, h.name, other)
					ok = false
					continue
				}
				if h.mcp.exists(home, s.Name) {
					fmt.Fprintf(os.Stderr, "%s: %s already has an mcp server named %s that gap did not add; left alone\n", id, h.name, s.Name)
					ok = false
					continue
				}
				jobs = append(jobs, job{h, s, key, sum, false})
			}
			for key := range entry.MCP {
				if strings.HasPrefix(key, h.name+":") && !slices.ContainsFunc(c.MCP, func(s mcpServer) bool { return mcpKey(h, s.Name) == key }) {
					stale = append(stale, key)
				}
			}
		}
		if len(jobs) == 0 && len(stale) == 0 {
			continue
		}
		if len(jobs) > 0 && !gate.trusted(c) {
			var lines []string
			for _, j := range jobs {
				lines = append(lines, j.h.name+": "+j.s.describe())
			}
			if !gate.approve(c, "mcp", strings.Join(lines, "\n  ")) {
				ok = false
				jobs = nil
			}
		}
		out.section(id, "register mcp servers")
		if entry.MCP == nil {
			entry.MCP = map[string]string{}
		}
		slices.Sort(stale)
		for _, key := range stale {
			hname, name, _ := strings.Cut(key, ":")
			h := harnessNamed(hname)
			if err := runMCP(home, h.mcp.bin, h.mcp.remove(name)); err != nil {
				fmt.Fprintf(os.Stderr, "%s: %v\n", id, err)
				ok = false
				continue
			}
			delete(entry.MCP, key)
			changed = true
			out.result(true, fmt.Sprintf("%s: removed %s", hname, name))
		}
		for _, j := range jobs {
			if j.owns {
				if err := runMCP(home, j.h.mcp.bin, j.h.mcp.remove(j.s.Name)); err != nil {
					fmt.Fprintf(os.Stderr, "%s: %v\n", id, err)
					ok = false
					continue
				}
			}
			if err := runMCP(home, j.h.mcp.bin, j.h.mcp.add(j.s)); err != nil {
				fmt.Fprintf(os.Stderr, "%s: %v\n", id, err)
				ok = false
				if j.owns {
					delete(entry.MCP, j.key)
					changed = true
				}
				continue
			}
			entry.MCP[j.key] = j.sum
			owner[j.key] = id
			changed = true
			out.result(true, fmt.Sprintf("%s: %s", j.h.name, j.s.describe()))
		}
		if len(entry.MCP) == 0 {
			entry.MCP = nil
		}
		state.Components[id] = entry
	}
	if !changed {
		return ok, nil
	}
	fresh, err := readState(home)
	if err != nil {
		return false, err
	}
	for id, entry := range state.Components {
		f := fresh.Components[id]
		f.MCP = entry.MCP
		fresh.Components[id] = f
	}
	return ok, writeState(home, fresh)
}

// removeComponentMCP unregisters every server the state records for one component.
func removeComponentMCP(home string, entry stateEntry) bool {
	ok := true
	keys := make([]string, 0, len(entry.MCP))
	for key := range entry.MCP {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	for _, key := range keys {
		hname, name, _ := strings.Cut(key, ":")
		h := harnessNamed(hname)
		if h.mcp == nil {
			continue
		}
		if _, err := exec.LookPath(h.mcp.bin); err != nil {
			fmt.Fprintf(os.Stderr, "%s is not installed; remove its mcp server %s by hand\n", hname, name)
			ok = false
			continue
		}
		if err := runMCP(home, h.mcp.bin, h.mcp.remove(name)); err != nil {
			fmt.Fprintln(os.Stderr, err)
			ok = false
			continue
		}
		fmt.Printf("Removed mcp server %s from %s\n", name, hname)
	}
	return ok
}
