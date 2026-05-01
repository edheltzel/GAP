# atlas-config

Minimal config for the agents Ed uses, managed with [GNU Stow](https://www.gnu.org/software/stow/) and [`just`](https://github.com/casey/just).

This repo intentionally starts small. Items are cherry-picked and added over time as needed — no pre-built skill/hook bundles, no auto-loaded frameworks. If it's here, it's because it earned its keep.

## Layout

| Package      | Source                  | Stowed to        |
| ------------ | ----------------------- | ---------------- |
| `claudecode` | `claudecode/.claude/`   | `~/.claude/`     |
| `pi`         | `pi/.pi/agent/`         | `~/.pi/agent/`   |

Plans, handoffs, and worktrees live under `.atlas/` at the repo root (see `.atlas/plans/`, `.atlas/handoffs/`).

## Commands

```bash
just link     # first-time setup — create symlinks for all packages
just up       # restow after adding/removing files (alias: just update)
just unlink   # remove all symlinks
just status   # dry run — show what stow would do
just push     # commit any changes and push to origin
```

## Notes

- `pi/` is a placeholder while the Pi agent is reconfigured. Add files there as needed and `just up` to stow them.
- `oldSetup/`, `.audit-backup/`, and any `auth.json` files are gitignored — runtime state and local backups never get committed.
- Worktrees (`EnterWorktree` / `ExitWorktree`) write to `.atlas/worktrees/`, which is also gitignored.
