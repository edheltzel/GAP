# Claude Code — Atlas Integration

Atlas's primary coding agent. Claude Code provides the runtime for the PAI Algorithm, skills, hooks, and memory system.

## Stow Package

**Source:** `claudecode/.claude/`
**Target:** `~/.claude/`

```bash
just up   # restow after changes
```

## Key Files

| File | Purpose |
|------|---------|
| `settings.json` | Hooks, identity, permissions, statusline, MCP servers |
| `CLAUDE.md` | Global agent instructions — loaded at every session start |
| `skills/PAI/SKILL.md` | Core PAI system (Algorithm, ISC, PRDs) |
| `hooks/` | 24 lifecycle event handlers (SessionStart, Stop, PostToolUse, etc.) |
| `agents/` | Named agent personas for subagent spawning |
| `VoiceServer/` | Multi-provider TTS server on port 8888 |
| `lib/` | Shared TypeScript utilities (paths, voice, etc.) |
| `atlas-statusline.sh` | Eldritch-themed statusline for Claude Code UI |

## How Atlas Runs in Claude Code

1. **Session starts** → `SessionStart` hooks fire (context load, voice greeting, capability recommendation)
2. **User sends prompt** → `UserPromptSubmit` hook enforces Algorithm format
3. **Algorithm executes** → 7-phase cycle (Observe → Think → Plan → Build → Execute → Verify → Learn)
4. **Skills invoked** → `/research`, `/commit`, `/browser`, etc. via `Skill` tool
5. **PRDs track work** → `~/.claude/MEMORY/WORK/{slug}/PRD.md` as session artifacts
6. **Plans stored** → `.atlas/plans/` at project root (mandatory)
7. **Session ends** → `Stop` hooks fire (session extract, learning capture)

## Plans and PRDs

- **Plans** (handoffs, design docs, implementation plans): `.atlas/plans/` at project root — mandatory, git-tracked
- **PRDs** (Algorithm session artifacts): `~/.claude/MEMORY/WORK/` — internal, not git-tracked

## Voice

Claude Code uses the shared VoiceServer at `localhost:8888`. Phase transitions, greetings, and notifications are spoken via inline `curl` calls.

## Permissions

Configured in `settings.json` under `permissions`. Covers bash commands, MCP tools, and file operations. See `settings.json` for the full allowlist.
