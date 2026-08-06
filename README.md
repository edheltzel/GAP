# atlas-config

Minimal config for the agents Ed uses. Harness config is linked with [GNU Stow](https://www.gnu.org/software/stow) and [`just`](https://github.com/casey/just); the repository tracks the native Claude skill snapshot and stows its repository-owned contents, while shared skills and OMP marketplace plugins are provisioned from tracked manifests.

## Layout

| Surface | Source | Installed to |
| ------------ | ----------------------- | ---------------- |
| `claude` | `claude/.claude/` | `~/.claude/` |
| `pi` | `pi/.pi/agent/` | `~/.pi/agent/` |
| `omp` | `omp/.omp/agent/` | `~/.omp/agent/` |
| `jcode` | `jcode/.jcode/` | `~/.jcode/` |
| Native Claude skills/agents | `claude/.claude/{skills,agents}/` | `~/.claude/{skills,agents}/` |
| Shared skills | `skills.txt`, `install-skills.sh` | skills.sh global install targets |
| OMP plugins | `omp-plugins.txt`, `install-omp-plugins.sh` | `~/.omp/plugins/` runtime registry and cache |

Plans, handoffs, worktrees, and artifacts live under `.agents/atlas/` at the repo root (`plans/`, `handoffs/`, `artifacts/`, `worktrees/`). The entire `.agents/atlas/` folder is gitignored, local-only, and does not sync via git.

## Commands

```bash
just                 # list all available commands (bare `just`)
just link            # first-time setup: link packages and install both manifests
just up              # preflight and restow every package, then install both manifests
just stow-preflight  # fail on genuine ownership conflicts before package mutation
just claude          # preflight and restow the Claude package only (no-folding)
just omp             # restow OMP config (no-folding) and install omp-plugins.txt
omp update           # install the latest published OMP release
just pi              # restow the pi package only (no-folding)
just jcode           # restow curated Jcode config only (no-folding)
just unlink          # remove all package symlinks
just status          # show managed links and fail on conflicts or dangling managed links
just skills          # list declared skills grouped by repo, with install status
just skills-status   # dry run: print pinned sources and integrity checks
just skills-install  # install skills listed in skills.txt via skills.sh
just omp-plugins-status  # dry run: print OMP marketplace add/update/install commands
just omp-plugins-install # install or upgrade plugins listed in omp-plugins.txt
just push            # commit any changes and push to origin
```

## Notes

- Agent instructions use DOX: start at root `AGENTS.md`, follow any Child DOX Index entries toward the files being edited, and update the owning AGENTS.md after meaningful contract changes.

- Atlas Flow is tracked as the atlas-flow epic (issues #2-#11); review it before changing Pi's global package set or implementing `/atlas-flow`.
- `pi/` is active curated Pi config, not a placeholder. Its status line uses Pi's own version and its theme is `eldritch`.
- Claude, Pi, and OMP status lines use harness-native version identity and never read the removed `LIFEOS/VERSION`.
- `claude/.claude/{skills,agents}/` preserves the exact final native snapshot recovered from the unmerged LifeOS-removal lineage. Some restored skills retain known references to removed LifeOS machinery, and five Recall pointer directories are not stowed. Recoverable historical external pointers are declared in `skills.txt`; top-level shared-skill links are not tracked in the Claude package. See `docs/agents/native-claude-skills.md`.
- `jcode/` tracks portable preferences in writable `config.toml`; the `jcode-config` clean filter removes theme selection and generated launch-hotkey state before indexing.
- Jcode credentials, sessions, memory, caches, telemetry, generated browser assets, and MCP discovery output stay machine-local.
- `oldSetup/`, `.audit-backup/`, and any `auth.json` files are gitignored; runtime state and local backups never get committed.
- OMP runtime state stays local; only curated files like `config.yml`, `models.yml`, `mcp.json`, keybindings, skills, commands, tools, extensions, and themes are stowed. `omp-plugins.txt` tracks desired marketplace packages while OMP keeps its installed registry and cache under `~/.omp/plugins/`. See `docs/agents/omp.md`.
- `just link` and `just up` preflight every Stow package, link all packages, then provision shared skills and OMP plugins. This ordering prevents manifest updates when a package conflict would leave a partial restore. Package-only commands affect only their package-specific state.
- Lavish is globally owned through `skills.txt`, which installs upstream commit `55045850f61e7b23c8dbf8cc30d92bd6e31649d2` from a disposable checkout and verifies blob `1e792b5bfc34127ca6f00bbb44fea306ce77ea52` during `just skills-status` and `just skills-install`.
- Fresh or disposable checkouts should run `just skills-status` and then `just skills-install` before using Lavish; `just claude` and `just pi` only restow package-specific state.
- To roll back the project-ownership removal, revert its commit and rerun `just skills-install` to restore the pinned global copy and generated agent links; do not reintroduce a Pi package copy.
- Worktrees (`EnterWorktree` / `ExitWorktree`) write to `.agents/atlas/worktrees/`, which is also gitignored.
