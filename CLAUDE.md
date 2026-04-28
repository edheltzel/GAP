# Atlas Config — Agent Instructions

## For Agents

If you are an AI agent working in this repo, read the following files for full system context:

1. **`claudecode/.claude/skills/PAI/SKILL.md`** — The complete PAI system documentation including the Algorithm, capabilities, and execution rules.
2. **`claudecode/.claude/CLAUDE.md`** — Context injection and session initialization.
3. **`claudecode/.claude/settings.json`** — Identity configuration, hooks, and permissions.

Do not modify files in `claudecode/.claude/MEMORY/` without understanding the memory system (documented in `SKILL.md`).

For PAI upgrade guidance, local modifications table, v4.0.3 integration decisions, and codebase audit history, see [`docs/pai-update-guide.md`](docs/pai-update-guide.md).

## Hook System

The hook system is documented in [`claudecode/.claude/hooks/README.md`](claudecode/.claude/hooks/README.md). Key references:

- **Hook registry**: 23 stowed hooks + 2 non-stowed + 1 external, across 8 lifecycle events
- **Event-to-hook mapping**: Exact mapping from `settings.json` with matchers, timeouts, and async flags
- **Debugging**: Structured logging via `hookLog()` writes to `~/.claude/MEMORY/HOOKS/hook-debug.jsonl`
- **Critical behavior**: `claude -p` (pipe mode) DOES execute hooks and spawn MCP servers — hooks that spawn `claude -p` must include recursion guards (e.g., `SESSION_EXTRACT_ACTIVE` env var)

When modifying hooks, follow the update protocol in the hooks README (update header docs, README tables, and inter-hook dependencies).

## Voice System

The VoiceServer is documented in [`claudecode/.claude/VoiceServer/README.md`](claudecode/.claude/VoiceServer/README.md). Key references:

- **API**: POST `/notify` on port 8888 — any process that can POST JSON can use it
- **Providers**: Edge TTS (primary) → ElevenLabs (cloud) → Kokoro (local) → macOS say (fallback)
- **Config**: `voices.json` (providers, agent voices), `pronunciations.json` (TTS overrides)
- **Voice conversations**: [VoiceLayer](https://github.com/EtanHey/voicelayer) MCP server provides bidirectional converse mode (Silero VAD + whisper.cpp STT + Edge TTS)
- **Agent-agnostic**: No coding-agent-specific code in the server — integration is consumer-side

## Session Handoff & Standup Workflow

A cross-agent pair of commands for daily project pickup and safe session clearing. Usable from Claude Code, OpenCode, or Pi — all three write to the same `.atlas/` path and dump to the same Recall database, so work from any agent can be resumed in any other.

- **Standup** — the daily entry point. Reviews active plans in `.atlas/plans/`, surfaces live handoffs, and lets you pick what to work on. Archives handoffs automatically on pickup and plans automatically on completion. One subcommand: `list-plans` (quick read-only listing). Files:
  - `claudecode/.claude/commands/Standup.md` (Claude Code) → `/Standup [list-plans]`
  - `opencode/.config/opencode/command/Standup.md` (OpenCode) → `/Standup [list-plans]`
  - `pi-agent/.pi/agent/skills/standup/SKILL.md` (Pi) → `/skill:standup [list-plans]`
- **CreateHandoff** — interviews the user and writes a frontmatter-stamped handoff to `.atlas/handoffs/YYYY-MM-DD-<slug>.md`, then runs `mem dump` (skipped gracefully if the Recall CLI isn't installed). Files:
  - `claudecode/.claude/commands/CreateHandoff.md` (Claude Code) → `/CreateHandoff`
  - `opencode/.config/opencode/command/CreateHandoff.md` (OpenCode) → `/CreateHandoff`
  - `pi-agent/.pi/agent/skills/create-handoff/SKILL.md` (Pi) → `/skill:create-handoff`

Handoff archiving happens on **pickup** via Standup — a resumed session that runs out of context again can create a fresh handoff without colliding with the previous one.

## `.atlas/` Directory Convention

All project-local Atlas artifacts live under `.atlas/`:

| Path                          | Purpose                                    |
|-------------------------------|--------------------------------------------|
| `.atlas/plans/`               | Active plan/design docs                    |
| `.atlas/plans/archive/`       | Completed plans                            |
| `.atlas/handoffs/`            | Session handoff docs                       |
| `.atlas/handoffs/archive/`    | Consumed handoffs (archived on read)       |

## For Humans

See **[README.md](README.md)** for a human-friendly overview of PAI, Atlas, all available skills, and setup instructions.
