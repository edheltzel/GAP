# Atlas Config — Agent Instructions

## MANDATORY: Use td for Task Management

You must run td usage --new-session at conversation start (or after /clear) to see current work.
Use td usage -q for subsequent reads.

This repository contains the PAI (Personal AI Infrastructure) configuration for Atlas.

## For Agents

If you are an AI agent working in this repo, read the following files for full system context:

1. **`.claude/skills/PAI/SKILL.md`** — The complete PAI system documentation including the Algorithm, capabilities, and execution rules.
2. **`.claude/CLAUDE.md`** — Context injection and session initialization.
3. **`.claude/settings.json`** — Identity configuration, hooks, and permissions.

Do not modify files in `.claude/MEMORY/` without understanding the memory system (documented in `SKILL.md`).

## Local Modifications (PAIUpdate Protection)

The following files contain **local customizations** that must NOT be overwritten during a PAI update (`/pai-upgrade`). When pulling upstream PAI changes, preserve these modifications:

| File | What Was Customized |
|------|-------------------|
| `.claude/skills/PAI/Tools/pai.ts` | Added `--dangerously-skip-permissions` flag to `cmdLaunch()`. Removed hardcoded voice notification (handled by async hook instead). |
| `.claude/hooks/StartupGreeting.hook.ts` | Removed banner display from hook (banner only shows via `pai.ts` CLI path). Kept Kitty env persistence. |
| `.claude/hooks/VoiceGreeting.hook.ts` | **New file** — async SessionStart hook that speaks the startup catchphrase for ALL launch paths (`pai`, `atlas`, `claude`). |
| `.claude/settings.json` | Registered `VoiceGreeting.hook.ts` as an async SessionStart hook in the hooks section. |
| `~/.zshrc` | Added `atlas` alias alongside `pai` alias. |

**During PAIUpdate:** If upstream changes touch any of these files, merge carefully — do not blindly replace. The voice greeting architecture depends on the hook being async and `pai.ts` NOT having its own voice call.

## For Humans

See **[README.md](README.md)** for a human-friendly overview of PAI, Atlas, all available skills, and setup instructions.
