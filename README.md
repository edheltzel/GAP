# atlas-config

Minimal config for the agents Ed uses. Harness config is still linked with [GNU Stow](https://www.gnu.org/software/stow/) and [`just`](https://github.com/casey/just); skills are manifest-installed through skills.sh from `skills.txt`.

## Layout

| Surface | Source | Installed to |
| ------------ | ----------------------- | ---------------- |
| `claude`     | `claude/.claude/`       | `~/.claude/`     |
| `pi`         | `pi/.pi/agent/`         | `~/.pi/agent/`   |
| `omp`        | `omp/.omp/agent/`       | `~/.omp/agent/`  |
| Skills       | `skills.txt`, `install-skills.sh` | skills.sh global install targets |

Plans, handoffs, worktrees, and artifacts live under `.agents/atlas/` at the repo root (`plans/`, `handoffs/`, `artifacts/`, `worktrees/`). The entire `.agents/atlas/` folder is gitignored — local-only, does not sync via git.

## Commands

```bash
just                 # list all available commands (bare `just`)
just link            # first-time setup — create symlinks and install skills.txt
just up              # restow every package and install skills.txt
just claude          # restow the claude package only
just omp             # restow the omp package only
just pi              # restow the pi package only (no-folding)
just unlink          # remove all package symlinks
just status          # dry run — show what stow would do
just skills          # list declared skills grouped by repo, with install status
just skills-status   # dry run — print skills.sh install commands from skills.txt
just skills-install  # install skills listed in skills.txt via skills.sh
just push            # commit any changes and push to origin
```

## Notes

- Agent instructions use DOX: start at root `AGENTS.md`, follow any Child DOX Index entries toward the files being edited, and update the owning AGENTS.md after meaningful contract changes.

- Atlas Flow is tracked as the atlas-flow epic (issues #2–#11); review it before changing Pi's global package set or implementing `/atlas-flow`.
- `pi/` is active curated Pi config, not a placeholder.
- `oldSetup/`, `.audit-backup/`, and any `auth.json` files are gitignored — runtime state and local backups never get committed.
- OMP runtime state stays local; only curated files like `config.yml`, `models.yml`, `mcp.json`, keybindings, skills, commands, tools, extensions, and themes should be stowed. See `docs/agents/omp.md`.
- Worktrees (`EnterWorktree` / `ExitWorktree`) write to `.agents/atlas/worktrees/`, which is also gitignored.
