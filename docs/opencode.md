# OpenCode — Atlas Integration

Atlas's secondary coding agent. OpenCode provides an alternative terminal UI with its own agent, command, skill, and plugin system.

## Stow Package

**Source:** `opencode/.config/opencode/`
**Target:** `~/.config/opencode/`

```bash
just up   # restow after changes
```

## Key Files

| File | Purpose |
|------|---------|
| `opencode.json` | Main config — models, MCP servers, permissions |
| `README.md` | Global agent instructions and setup guide |
| `agent/` | 15+ subagent definitions (Algorithm, Architect, Artist, etc.) |
| `command/` | Slash commands (`/commit`, `/voice`, `/x-plan-create`, etc.) |
| `skill/` | Loadable skills (frontend-design, etc.) |
| `plugins/` | Auto-loaded plugins (voice notifications) |
| `themes/` | Custom color themes (Eldritch variants) |
| `superpowers/` | Superpowers plugin — brainstorming, plans, TDD, debugging workflows |

## How Atlas Runs in OpenCode

1. **Session starts** → Voice plugin speaks startup catchphrase
2. **User sends prompt** → Agent processes with configured model
3. **Skills/commands invoked** → `/x-plan-create`, `/commit`, `/voice`, etc.
4. **Subagents spawned** → `codebase-analyzer`, `workflow-locator`, researchers, etc.
5. **Plans stored** → `.atlas/plans/` at project root (mandatory)

## Plans

- **Create:** `/x-plan-create` — interactive plan creation with research agents
- **Implement:** `/x-plan-implement` — execute an approved plan phase by phase
- **Standup:** `/Standup` — review active plans, pick what to work on, archive handoffs on pickup
- **List:** `/Standup list-plans` — quick read-only listing of active and archived plans
- **Migrate:** `/Standup migrate` — move legacy `.atlas-plans/` → `.atlas/`
- **Archive:** handled automatically — plans archive on completion, handoffs archive on pickup via `/Standup`

All plan commands target `.atlas/plans/` at the project root.

## Voice

OpenCode integrates with the shared VoiceServer (`localhost:8888`) via the `voice-notification.ts` plugin. Session start, task completion, permission requests, and test results trigger voice announcements.

## Models

Configured in `opencode.json`:
- **Plan agent:** `opencode-go/glm-5`
- **Build agent:** `opencode-go/kimi-k2.5`

## MCP Servers

- **context7** — Library documentation lookup
- **recall-memory** — Persistent memory (shared with Claude Code)
- **pencil** — Pencil desktop integration
