# atlas-config — Agent Instructions

This repo is a minimal, stow-managed config for Ed's agents. On 2026-05-18 the `Codex/` package was populated with Ed's PAI v5.0.0 USER-CONFIG (identity, TELOS, durable MEMORY, SYSTEM-OVERRIDE hooks/Pulse/TOOLS); prior to that the previous Atlas/PAI build was soft-reset on 2026-05-01.

## PAI v5.0.0 migration

This repo tracks Ed's PAI v5.0.0 configuration (identity, TELOS, durable memory, SYSTEM-OVERRIDE hooks). See `.atlas/plans/2026-05-18-pai-to-atlas-config-migration-plan.md` for the canonical migration spec (supersedes the archived MANIFEST at `.atlas/plans/archive/2026-05-17-pai-migration-manifest.md`).

## Scope

**atlas-config is single-machine-canonical.** The `merge=ours` rule on `Codex/.Codex/settings.json` (see `.gitattributes`) means changes to that file do not propagate across machines via `git pull` — local edits always win. Multi-machine sync requires explicit `git checkout origin/master -- Codex/.Codex/settings.json` on the receiving Mac, then a re-personalization pass if needed. This is intentional: `settings.json` carries per-session mutating fields (`counts`, `feedbackSurveyState.lastShownTime`) that would otherwise churn the repo.

The Recall MCP database (`~/.agents/atlas/Recall/memory.db`) and the Recall MEMORY/ vault (`~/.agents/atlas/Recall/MEMORY/`) also live **outside** atlas-config — they are out-of-band like `~/.env`. Cold-archive these to an external location on a schedule (1Password vault, Time Machine, external drive); they are the only USER content not in git.

## Repo layout

| Package      | Source                  | Stowed to        | Status                                           |
| ------------ | ----------------------- | ---------------- | ------------------------------------------------ |
| `Codex` | `Codex/.Codex/`   | `~/.Codex/`     | Populated with PAI v5.0.0 USER-CONFIG (2026-05-18) |
| `pi`         | `pi/.pi/agent/`         | `~/.pi/agent/`   | Placeholder — Pi will be reconfigured over time  |

Project-local artifacts live under `.atlas/` (`plans/`, `handoffs/`, `worktrees/`). `worktrees/` is gitignored.

Before changing Pi's global package set or implementing `/atlas-flow`, read `.atlas/plans/2026-05-16-atlas-flow-research-digest.md` for the minimal-global baseline, flow manifest schema direction, safety rules, and issue roadmap.

## Stow

Multi-package stow with the repo root as `STOW_DIR`. See `justfile` for commands. The per-package `.stow-local-ignore` files **replace** stow's built-in defaults — when adding a new one, copy from `Codex/.stow-local-ignore` and re-include `\.git`, `\.gitignore`, etc.

## When adding files

- Adding to `Codex/.Codex/` or `pi/.pi/agent/` → run `just up` to restow.
- Anything that should not be symlinked into the runtime location → add a pattern to that package's `.stow-local-ignore`.
- Secrets (`auth.json`, `*.env`, keys) → already covered by `.gitignore`. Do not weaken those rules.

## Notable conventions

- `.atlas/plans/` for active plan/design docs, `.atlas/plans/archive/` for completed.
- `.atlas/handoffs/` for live handoffs, archived to `.atlas/handoffs/archive/` when consumed.
- File naming: `YYYY-MM-DD-description.md`.
- Use git worktrees (`EnterWorktree` / `ExitWorktree`) for implementation work — they land in `.atlas/worktrees/`.
