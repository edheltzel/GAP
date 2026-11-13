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
	bin        string
	configEnv  string
	defaultDir string
	add        func(s mcpServer) []string
	remove     func(name string) []string
	exists     func(home, target, name string) (bool, error)
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

type mcpRegistration struct {
	Digest string `json:"digest"`
	Target string `json:"target"`
}

func jsonServerExists(file string) func(home, target, name string) (bool, error) {
	return func(home, target, name string) (bool, error) {
		b, err := os.ReadFile(filepath.Join(target, file))
		if os.IsNotExist(err) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		var doc struct {
			MCPServers map[string]json.RawMessage `json:"mcpServers"`
		}
		if err := json.Unmarshal(b, &doc); err != nil {
			return false, err
		}
		_, ok := doc.MCPServers[name]
		return ok, nil
	}
}

var claudeMCP = &mcpCLI{
	bin:       "claude",
	configEnv: "CLAUDE_CONFIG_DIR",
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
	bin:        "codex",
	configEnv:  "CODEX_HOME",
	defaultDir: ".codex",
	add: func(s mcpServer) []string {
		if s.URL != "" {
			return []string{"mcp", "add", s.Name, "--url", s.URL}
		}
		return stdioArgs([]string{"mcp", "add", s.Name}, "--env", s)
	},
	remove: func(name string) []string { return []string{"mcp", "remove", name} },
	exists: func(home, target, name string) (bool, error) {
		cmd := mcpCommand(home, "CODEX_HOME", target, "codex", []string{"mcp", "list", "--json"})
		out, err := cmd.Output()
		if err != nil {
			return false, err
		}
		var servers []struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(out, &servers); err != nil {
			return false, err
		}
		for _, server := range servers {
			if server.Name == name {
				return true, nil
			}
		}
		return false, nil
	},
}

var piMCP = &mcpCLI{
	bin:        "pi",
	configEnv:  "PI_CODING_AGENT_DIR",
	defaultDir: filepath.Join(".pi", "agent"),
	add: func(s mcpServer) []string {
		if s.URL != "" {
			return []string{"mcp", "add", s.Name, "--url", s.URL}
		}
		return stdioArgs([]string{"mcp", "add", s.Name}, "--env", s)
	},
	remove: func(name string) []string { return []string{"mcp", "remove", name} },
	exists: jsonServerExists("mcp.json"),
}

func (cli *mcpCLI) target(home string) (string, error) {
	target := os.Getenv(cli.configEnv)
	if target == "" {
		target = filepath.Join(home, cli.defaultDir)
	}
	if !filepath.IsAbs(target) {
		return "", fmt.Errorf("%s must be an absolute path: %s", cli.configEnv, target)
	}
	return filepath.Clean(target), nil
}

func mcpCommand(home, configEnv, target, bin string, args []string) *exec.Cmd {
	cmd := exec.Command(bin, args...)
	cmd.Dir = home
	cmd.Env = append(os.Environ(), "HOME="+home, configEnv+"="+target)
	return cmd
}

func runMCP(home string, cli *mcpCLI, target string, args []string) error {
	cmd := mcpCommand(home, cli.configEnv, target, cli.bin, args)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s %s: %w: %s", cli.bin, strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return nil
}

func removeMCP(home string, h harness, target, name string) error {
	if h.mcp == nil {
		return fmt.Errorf("%s has no mcp CLI", h.name)
	}
	if !filepath.IsAbs(target) {
		return fmt.Errorf("%s mcp ownership has no absolute configuration target", h.name)
	}
	if _, err := exec.LookPath(h.mcp.bin); err != nil {
		return err
	}
	exists, err := h.mcp.exists(home, target, name)
	if err != nil || !exists {
		return err
	}
	return runMCP(home, h.mcp, target, h.mcp.remove(name))
}

func mcpKey(h harness, name string) string { return h.name + ":" + name }

func mcpDigest(s mcpServer) string {
	b, _ := json.Marshal(s)
	return digest(string(b))
}

func recordComponentMCP(home, id string, registrations map[string]mcpRegistration) error {
	state, err := readState(home)
	if err != nil {
		return err
	}
	entry := state.Components[id]
	entry.MCP = registrations
	state.Components[id] = entry
	return writeState(home, state)
}

// publishComponentMCP registers each selected component's MCP servers with every harness whose mcp CLI is installed.
// A server name another component owns, or one the user already configured, is left alone and reported.
func publishComponentMCP(home string, d distro) (map[string]bool, error) {
	state, err := readState(home)
	if err != nil {
		return nil, err
	}
	gate, err := newHookGate(home)
	if err != nil {
		return nil, err
	}
	owner := map[string]string{}
	for id, entry := range state.Components {
		for key := range entry.MCP {
			owner[key] = id
		}
	}
	failed := map[string]bool{}
	out := newReport(os.Stdout)
	for _, id := range sortedIDs(d.Components) {
		c := d.Components[id]
		entry := state.Components[id]
		if c.Store == "" || len(c.MCP) == 0 && len(entry.MCP) == 0 {
			continue
		}
		type job struct {
			h      harness
			s      mcpServer
			key    string
			sum    string
			target string
			owns   bool
		}
		var jobs []job
		var stale []string
		for _, h := range harnesses {
			if h.mcp == nil {
				continue
			}
			if _, err := exec.LookPath(h.mcp.bin); err != nil {
				for key := range entry.MCP {
					if strings.HasPrefix(key, h.name+":") {
						fmt.Fprintf(os.Stderr, "%s: %v\n", id, err)
						failed[id] = true
						break
					}
				}
				continue
			}
			for _, raw := range c.MCP {
				s := raw.expand(c.Store)
				key, sum := mcpKey(h, s.Name), mcpDigest(s)
				prev, mine := entry.MCP[key]
				target := prev.Target
				var targetErr error
				if !mine {
					target, targetErr = h.mcp.target(home)
				} else if !filepath.IsAbs(target) {
					targetErr = fmt.Errorf("%s mcp ownership has no absolute configuration target", h.name)
				}
				if targetErr != nil {
					fmt.Fprintf(os.Stderr, "%s: %v\n", id, targetErr)
					failed[id] = true
					continue
				}
				exists, err := h.mcp.exists(home, target, s.Name)
				if err != nil {
					fmt.Fprintf(os.Stderr, "%s: %v\n", id, err)
					failed[id] = true
					continue
				}
				if mine {
					if prev.Digest != sum || !exists {
						jobs = append(jobs, job{h, s, key, sum, target, true})
					}
					continue
				}
				if other := owner[key]; other != "" && other != id {
					fmt.Fprintf(os.Stderr, "%s: mcp server %s in %s belongs to %s; skipped\n", id, s.Name, h.name, other)
					failed[id] = true
					continue
				}
				if exists {
					fmt.Fprintf(os.Stderr, "%s: %s already has an mcp server named %s that gap did not add; left alone\n", id, h.name, s.Name)
					failed[id] = true
					continue
				}
				jobs = append(jobs, job{h, s, key, sum, target, false})
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
				failed[id] = true
				jobs = nil
			}
		}
		out.section(id, "register mcp servers")
		if entry.MCP == nil {
			entry.MCP = map[string]mcpRegistration{}
		}
		slices.Sort(stale)
		for _, key := range stale {
			hname, name, _ := strings.Cut(key, ":")
			h := harnessNamed(hname)
			if err := removeMCP(home, h, entry.MCP[key].Target, name); err != nil {
				fmt.Fprintf(os.Stderr, "%s: %v\n", id, err)
				failed[id] = true
				continue
			}
			delete(entry.MCP, key)
			delete(owner, key)
			if err := recordComponentMCP(home, id, entry.MCP); err != nil {
				return nil, err
			}
			out.result(true, fmt.Sprintf("%s: removed %s", hname, name))
		}
		for _, j := range jobs {
			if j.owns {
				if err := removeMCP(home, j.h, j.target, j.s.Name); err != nil {
					fmt.Fprintf(os.Stderr, "%s: %v\n", id, err)
					failed[id] = true
					continue
				}
			}
			previous := entry.MCP[j.key]
			entry.MCP[j.key] = mcpRegistration{Target: j.target}
			if err := recordComponentMCP(home, id, entry.MCP); err != nil {
				return nil, err
			}
			addErr := runMCP(home, j.h.mcp, j.target, j.h.mcp.add(j.s))
			if addErr != nil {
				fmt.Fprintf(os.Stderr, "%s: %v\n", id, addErr)
				failed[id] = true
				exists, checkErr := j.h.mcp.exists(home, j.target, j.s.Name)
				if checkErr != nil {
					fmt.Fprintf(os.Stderr, "%s: %v\n", id, checkErr)
					j.sum = ""
				}
				if !exists && checkErr == nil {
					if j.owns {
						entry.MCP[j.key] = previous
					} else {
						delete(entry.MCP, j.key)
					}
					if err := recordComponentMCP(home, id, entry.MCP); err != nil {
						return nil, err
					}
					continue
				}
			}
			entry.MCP[j.key] = mcpRegistration{Digest: j.sum, Target: j.target}
			owner[j.key] = id
			if err := recordComponentMCP(home, id, entry.MCP); err != nil {
				return nil, err
			}
			if addErr == nil {
				out.result(true, fmt.Sprintf("%s: %s", j.h.name, j.s.describe()))
			}
		}
	}
	return failed, nil
}

// removeComponentMCP unregisters every server the state records for one component.
func removeComponentMCP(home, id string) error {
	state, err := readState(home)
	if err != nil {
		return err
	}
	entry := state.Components[id]
	ok := true
	keys := make([]string, 0, len(entry.MCP))
	for key := range entry.MCP {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	for _, key := range keys {
		hname, name, _ := strings.Cut(key, ":")
		h := harnessNamed(hname)
		if err := removeMCP(home, h, entry.MCP[key].Target, name); err != nil {
			fmt.Fprintln(os.Stderr, err)
			ok = false
			continue
		}
		delete(entry.MCP, key)
		fmt.Printf("Removed mcp server %s from %s\n", name, hname)
	}
	state.Components[id] = entry
	if err := writeState(home, state); err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("%s: mcp removal incomplete; retry gap remove %s", id, id)
	}
	return nil
}
