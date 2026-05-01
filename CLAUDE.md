# atlas-config — Agent Instructions

This repo is a minimal, stow-managed config for Ed's agents. The previous Atlas/PAI build (skills, hooks, VoiceServer, Observability, multi-agent coordination) was removed on 2026-05-01 — the goal now is to cherry-pick configuration as needs arise rather than ship a pre-built framework.

## Repo layout

| Package      | Source                  | Stowed to        | Status                                           |
| ------------ | ----------------------- | ---------------- | ------------------------------------------------ |
| `claudecode` | `claudecode/.claude/`   | `~/.claude/`     | Empty — Claude Code itself was freshly installed |
| `pi`         | `pi/.pi/agent/`         | `~/.pi/agent/`   | Placeholder — Pi will be reconfigured over time  |

Project-local artifacts live under `.atlas/` (`plans/`, `handoffs/`, `worktrees/`). `worktrees/` is gitignored.

## Stow

Multi-package stow with the repo root as `STOW_DIR`. See `justfile` for commands. The per-package `.stow-local-ignore` files **replace** stow's built-in defaults — when adding a new one, copy from `claudecode/.stow-local-ignore` and re-include `\.git`, `\.gitignore`, etc.

## When adding files

- Adding to `claudecode/.claude/` or `pi/.pi/agent/` → run `just up` to restow.
- Anything that should not be symlinked into the runtime location → add a pattern to that package's `.stow-local-ignore`.
- Secrets (`auth.json`, `*.env`, keys) → already covered by `.gitignore`. Do not weaken those rules.

## Notable conventions

- `.atlas/plans/` for active plan/design docs, `.atlas/plans/archive/` for completed.
- `.atlas/handoffs/` for live handoffs, archived to `.atlas/handoffs/archive/` when consumed.
- File naming: `YYYY-MM-DD-description.md`.
- Use git worktrees (`EnterWorktree` / `ExitWorktree`) for implementation work — they land in `.atlas/worktrees/`.
