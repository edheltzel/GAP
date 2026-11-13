package main

import "path/filepath"

// harness is one agent GAP delivers to. Paths are relative to $HOME. An empty policy is Codex, whose file lives under CODEX_HOME.
// commands is the folder the harness loads user slash commands from; empty means GAP publishes no commands there.
// mcp registers servers through the harness's own CLI. Add a path or CLI only after proving the harness loads it.
type harness struct {
	name     string
	policy   string
	cursor   bool
	commands string
	mcp      *mcpCLI
}

var harnesses = []harness{
	{name: "claude", policy: ".claude/rules/global-agent-policy.md", commands: ".claude/commands", mcp: claudeMCP},
	{name: "pi", policy: ".pi/agent/APPEND_SYSTEM.md", commands: ".pi/agent/prompts", mcp: piMCP},
	{name: "omp", policy: ".omp/agent/APPEND_SYSTEM.md", commands: ".omp/agent/commands"},
	{name: "codex", mcp: codexMCP},
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

func harnessNamed(name string) harness {
	for _, h := range harnesses {
		if h.name == name {
			return h
		}
	}
	return harness{name: name}
}
