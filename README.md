# atlas-config

Minimal config for the agents Ed uses, managed with [GNU Stow](https://www.gnu.org/software/stow/) and [`just`](https://github.com/casey/just).

This repo intentionally starts small. Items are cherry-picked and added over time as needed.

## Layout

| Package      | Source                  | Stowed to        |
| ------------ | ----------------------- | ---------------- |
| `claude`     | `claude/.claude/`       | `~/.claude/`     |
| `pi`         | `pi/.pi/agent/`         | `~/.pi/agent/`   |
| `codex`      | `codex/.codex/`         | `~/.codex/`      |
| `omp`        | `omp/.omp/agent/`       | `~/.omp/agent/`  |

Plans, handoffs, worktrees, and artifacts live under `.agents/` at the repo root (see `.agents/plans/`, `.agents/atlas/handoffs/`).

## Commands

```bash
just link     # first-time setup — create symlinks for all packages
just up       # restow after adding/removing files (alias: just update)
just unlink   # remove all symlinks
just status   # dry run — show what stow would do
just push     # commit any changes and push to origin
```

## Notes

- Atlas Flow research lives at `.agents/plans/2026-05-16-atlas-flow-research-digest.md`; read it before changing Pi's global package set or implementing `/atlas-flow`.
- `pi/` is a placeholder while the Pi agent is reconfigured. Add files there as needed and `just up` to stow them.
- `oldSetup/`, `.audit-backup/`, and any `auth.json` files are gitignored — runtime state and local backups never get committed.
- Codex runtime state stays local; only curated files like `config.toml` and `hooks.json` are stowed.
- OMP runtime state stays local; only curated files like `config.yml`, `models.yml`, `mcp.json`, keybindings, skills, commands, tools, extensions, and themes should be stowed. See `docs/agents/omp.md`.
- Worktrees (`EnterWorktree` / `ExitWorktree`) write to `.agents/atlas/worktrees/`, which is also gitignored.
