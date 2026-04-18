# PI Agent — Atlas Integration

Atlas's third coding agent. PI (Personal Intelligence) runs on the `my-pi` framework with custom tools, subagents, and extensions.

## Stow Package

**Source:** `pi-agent/.pi/agent/`
**Target:** `~/.pi/agent/`

```bash
just up   # restow after changes
```

## Key Files

| File | Purpose |
|------|---------|
| `AGENTS.md` | Global agent instructions — stack awareness, quality rules, git conventions |
| `settings.json` | PI Agent configuration and model settings |
| `git/github.com/noahsaso/my-pi/` | my-pi extensions and skills |

## How Atlas Runs in PI Agent

1. **Session starts** → AGENTS.md loaded as system context
2. **User sends prompt** → Agent processes with configured model
3. **Complex tasks** → Ant Colony auto-triggers for 3+ file changes or 2+ independent workstreams
4. **Subagents available** → `scout`, `planner`, `reviewer`, `worker`, `remover`
5. **Plans stored** → `.atlas/plans/` at project root (mandatory)

## Built-in Tools

| Tool | Purpose |
|------|---------|
| `todo_*` | Task tracking with dependency support |
| `subagent` | Delegate to specialized agents |
| `clipboard_*` | System clipboard read/write |
| `notify` / `ask_user` | System notifications with chime |
| `watch_*` | File/directory change watching |
| `ast_*` | TypeScript-aware code intelligence |
| `generate_image` | Image generation via Google Antigravity |
| `memory_*` | Persistent memory across sessions |

## Skills

- **browser-tools** — Browser automation via Chrome DevTools Protocol
- **superpowers** — 14 workflow skills (brainstorming, writing-plans, TDD, systematic-debugging, verification, etc.)

## Ant Colony

For complex tasks (3+ files, 2+ workstreams), PI Agent automatically uses ant colony coordination. After launching, it uses passive mode — waits for `COLONY_SIGNAL` updates rather than polling.

## Plans

All plans, handoffs, and design documents use `.atlas/plans/` at the project root. This is mandatory — `docs/plans/` is not used.
