package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// harness is one agent GAP delivers to. An empty policy is Codex, whose file lives under CODEX_HOME.
// commands is the folder the harness loads user slash commands from; empty means GAP publishes no commands there.
// mcp registers servers through the harness's own CLI. Add a path or CLI only after proving the harness loads it.
// homeEnv names an environment variable that moves the harness home; policy is then relative to that home, which defaults to homeDir under $HOME.
type harness struct {
	name     string
	policy   string
	cursor   bool
	commands string
	mcp      *mcpCLI
	homeEnv  string
	homeDir  string
}

var harnesses = []harness{
	{name: "claude", policy: ".claude/rules/global-agent-policy.md", commands: ".claude/commands", mcp: claudeMCP},
	{name: "pi", policy: ".pi/agent/APPEND_SYSTEM.md", commands: ".pi/agent/prompts", mcp: piMCP},
	{name: "omp", policy: ".omp/agent/APPEND_SYSTEM.md", commands: ".omp/agent/commands"},
	{name: "codex", mcp: codexMCP},
	{name: "cursor", policy: ".cursor/rules/global-agent-policy.mdc", cursor: true},
	{name: "empryo", policy: ".empryo/EMPRYO.md"},
	{name: "jcode", policy: "prompt-overlay.md", homeEnv: "JCODE_HOME", homeDir: ".jcode"},
}

// resolveHarnessHome returns the harness home from its environment variable. Empty means unset; a relative value is refused.
func resolveHarnessHome(name, env, fallback string) (string, error) {
	if env == "" {
		return fallback, nil
	}
	if !filepath.IsAbs(env) {
		return "", fmt.Errorf("%s must be an absolute path: %s", name, env)
	}
	return env, nil
}

func harnessHome(h harness, home string) (string, error) {
	return resolveHarnessHome(h.homeEnv, os.Getenv(h.homeEnv), filepath.Join(home, h.homeDir))
}

type policyDest struct {
	dest   string
	cursor bool
}

func policyDests(home, codexPolicy string) ([]policyDest, error) {
	dests := make([]policyDest, 0, len(harnesses))
	for _, h := range harnesses {
		dest := codexPolicy
		switch {
		case h.homeEnv != "":
			base, err := harnessHome(h, home)
			if err != nil {
				return nil, err
			}
			dest = filepath.Join(base, filepath.FromSlash(h.policy))
		case h.policy != "":
			dest = filepath.Join(home, filepath.FromSlash(h.policy))
		}
		dests = append(dests, policyDest{dest, h.cursor})
	}
	return dests, nil
}

func commandDests(home, name string) []string {
	var dests []string
	for _, h := range harnesses {
		if h.commands != "" {
			dests = append(dests, filepath.Join(home, filepath.FromSlash(h.commands), name))
		}
	}
	return dests
}

func harnessNamed(name string) harness {
	for _, h := range harnesses {
		if h.name == name {
			return h
		}
	}
	return harness{name: name}
}
