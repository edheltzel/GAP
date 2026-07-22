# atlas-config

Minimal config for the agents Ed uses. Harness config is still linked with [GNU Stow](https://www.gnu.org/software/stow) and [`just`](https://github.com/casey/just); shared skills and OMP marketplace plugins are provisioned from tracked manifests.

## Layout

| Surface | Source | Installed to |
| ------------ | ----------------------- | ---------------- |
| `claude`     | `claude/.claude/`       | `~/.claude/`     |
| `pi`         | `pi/.pi/agent/`         | `~/.pi/agent/`   |
| `omp`        | `omp/.omp/agent/`       | `~/.omp/agent/`  |
| Skills       | `skills.txt`, `install-skills.sh` | skills.sh global install targets |
| OMP plugins  | `omp-plugins.txt`, `install-omp-plugins.sh` | `~/.omp/plugins/` runtime registry and cache |

Plans, handoffs, worktrees, and artifacts live under `.agents/atlas/` at the repo root (`plans/`, `handoffs/`, `artifacts/`, `worktrees/`). The entire `.agents/atlas/` folder is gitignored — local-only, does not sync via git.

## Commands

```bash
just                 # list all available commands (bare `just`)
just link            # first-time setup — link packages and install both manifests
just up              # restow every package and install both manifests
just claude          # restow the claude package only
just omp             # restow OMP config and install omp-plugins.txt
omp update           # install the latest published OMP release
just pi              # restow the pi package only (no-folding)
just unlink          # remove all package symlinks
just status          # dry run — show what stow would do
just skills          # list declared skills grouped by repo, with install status
just skills-status   # dry run — print pinned sources and integrity checks
just skills-install  # install skills listed in skills.txt via skills.sh
just omp-plugins-status  # dry run — print OMP marketplace add/update/install commands
just omp-plugins-install # install or upgrade plugins listed in omp-plugins.txt
just push            # commit any changes and push to origin
```

## Notes

- Agent instructions use DOX: start at root `AGENTS.md`, follow any Child DOX Index entries toward the files being edited, and update the owning AGENTS.md after meaningful contract changes.

- Atlas Flow is tracked as the atlas-flow epic (issues #2–#11); review it before changing Pi's global package set or implementing `/atlas-flow`.
- `pi/` is active curated Pi config, not a placeholder.
- `oldSetup/`, `.audit-backup/`, and any `auth.json` files are gitignored — runtime state and local backups never get committed.
- OMP runtime state stays local; only curated files like `config.yml`, `models.yml`, `mcp.json`, keybindings, skills, commands, tools, extensions, and themes are stowed. `omp-plugins.txt` tracks desired marketplace packages while OMP keeps its installed registry and cache under `~/.omp/plugins/`. See `docs/agents/omp.md`.
- `just link` and `just up` provision shared skills from `skills.txt`; package-only `just claude`, `just omp`, and `just pi` commands only restow or provision their package-specific state.
- Lavish stage 1 pins global provisioning to upstream commit `55045850f61e7b23c8dbf8cc30d92bd6e31649d2` and verifies blob `1e792b5bfc34127ca6f00bbb44fea306ce77ea52`. The project Lavish file, Claude link, and lock entry remain intentionally until stage 2 evidence and approval.
- Worktrees (`EnterWorktree` / `ExitWorktree`) write to `.agents/atlas/worktrees/`, which is also gitignored.
