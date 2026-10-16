# Operations

Operating AI agents - delegation, handoffs, prompt audits, and project status check-ins.

## Installation

These skills live in this repo (`edheltzel/agent-config`, being renamed GAP). `gap sync` installs every skill listed in `global/skills.json`.

```bash
gap sync
```

One skill by exact name:

```bash
npx skills add edheltzel/agent-config -s <skill-name>
```

## Skills

| Skill | Description |
|-------|-------------|
| [`do-bitter-pill`](./do-bitter-pill/) | Audits AI instruction sets for over-prompting. |
| [`do-delegation`](./do-delegation/) | Routes independent work through the harness's subagent tool: background execution, role briefs, worktree isolation, and coordinator-managed synthesis. |
| [`do-handoff`](./do-handoff/) | Write a pointer-first session handoff, or resume from a source the user names. |
| [`do-regroup`](./do-regroup/) | Use when checking project status. |

## See Also

- [Global skills](../README.md)
- [Skill catalog](../../../README.md)
