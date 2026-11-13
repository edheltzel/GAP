package main

import "path/filepath"

// harness is one agent GAP delivers to. Paths are relative to $HOME. An empty policy is Codex, whose file lives under CODEX_HOME.
// commands is the folder the harness loads user slash commands from; empty means GAP publishes no commands there.
// Add a path only after proving the harness loads it.
type harness struct {
	name     string
	policy   string
	cursor   bool
	commands string
}

var harnesses = []harness{
	{name: "claude", policy: ".claude/rules/global-agent-policy.md", commands: ".claude/commands"},
	{name: "pi", policy: ".pi/agent/APPEND_SYSTEM.md", commands: ".pi/agent/prompts"},
	{name: "omp", policy: ".omp/agent/APPEND_SYSTEM.md", commands: ".omp/agent/commands"},
	{name: "codex"},
	{name: "cursor", policy: ".cursor/rules/global-agent-policy.mdc", cursor: true},
	{name: "empryo", policy: ".empryo/EMPRYO.md"},
}

type policyDest struct {
	dest   string
	cursor bool
}

func policyDests(home, codexPolicy string) []policyDest {
	dests := make([]policyDest, 0, len(harnesses))
	for _, h := range harnesses {
		dest := codexPolicy
		if h.policy != "" {
			dest = filepath.Join(home, filepath.FromSlash(h.policy))
		}
		dests = append(dests, policyDest{dest, h.cursor})
	}
	return dests
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
