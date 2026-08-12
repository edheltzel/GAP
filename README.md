# atlas-config

Minimal config for the agents Ed uses. Harness config is linked with [GNU Stow](https://www.gnu.org/software/stow) and [`just`](https://github.com/casey/just); the repository tracks native Claude skills and stows its repository-owned contents, while external skills and OMP marketplace plugins are provisioned from tracked manifests.

## Layout

| Surface | Source | Installed to |
| ------------ | ----------------------- | ---------------- |
| `claude` | `claude/.claude/` | `~/.claude/` |
| `pi` | `pi/.pi/agent/` | `~/.pi/agent/` |
| `omp` | `omp/.omp/agent/` | `~/.omp/agent/` |
| `jcode` | `jcode/.jcode/` | `~/.jcode/` |
| Native Claude skills | `claude/.claude/skills/` | `~/.claude/skills/` |
| External skills | `skills.txt`, `install-skills.sh` | skills.sh global install targets |
| OMP plugins | `omp-plugins.txt`, `install-omp-plugins.sh` | `~/.omp/plugins/` runtime registry and cache |

Plans, handoffs, worktrees, and artifacts live under `.agents/atlas/` at the repo root (`plans/`, `handoffs/`, `artifacts/`, `worktrees/`). The entire `.agents/atlas/` folder is gitignored, local-only, and does not sync via git.

## Commands

```bash
just                 # list all available commands (bare `just`)
just link            # link packages, provision native tools, and install both manifests
just up              # restow packages, provision native tools, and install both manifests
just stow-preflight  # fail on genuine ownership conflicts before package mutation
just claude          # preflight and restow the Claude package only (no-folding)
just omp             # restow OMP config (no-folding) and install omp-plugins.txt
omp update           # install the latest published OMP release
just pi              # restow the pi package only (no-folding)
just jcode           # restow curated Jcode config only (no-folding)
just unlink          # remove all package symlinks
just status          # require active managed links and fail on conflicts or dangling links
just dev-modules     # link ignored, repo-local harness modules for TypeScript tooling
just native-skill-tools # install locked dependencies for native Claude skill tools
just native-skill-tools-status # validate native tool locks and installed dependencies
just tooling-preflight # validate config, types, skills, and status-line behavior
just installed-skills-status # fail on missing skills or dangling managed skill links
just ownership-status # require exact Firstmate and app-owned tldraw skill stores
just preflight       # run Stow, tooling, installed-skill, and ownership validation
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
- Claude, Pi, and OMP status lines use harness-native version identity and never read the removed `LIFEOS/VERSION`. Claude's status command is safe without a controlling TTY and renders `…/<parent>/<current>` like Pi and OMP.
- `claude/.claude/skills/` preserves the 48-directory inventory recovered from the unmerged LifeOS-removal lineage and maintained with focused regression tests. Config actively stows 43 skills; five Recall pointer records are ignored while `skills.txt` owns their active runtime copies. Recoverable historical external pointers are declared in `skills.txt`; top-level external-skill links are not tracked in the Claude package. Firstmate's internal skills remain canonical in its sibling repository, not in Config or `skills.txt`. See `docs/agents/native-claude-skills.md`.
- Anti-slop uses split ownership: pinned entries from the canonical Pi Extensions checkout install five skills only into Claude Code, while Pi discovers them from `git:github.com/edheltzel/pi-extensions`. Per-entry `agents=` targets are intersected with `SKILLS_AGENTS`, preventing duplicate Pi links.
- `just preflight` requires active Stow links, installed manifest/native skills, direct Pi package discovery, zero duplicate Pi anti-slop copies, no dangling managed skill links, and exact external ownership. Firstmate or tldraw absence fails this strict check. tldraw Desktop owns its installed skill, agent, commands, hooks, and bundled tools; Config does not track duplicate artifacts.
- `jcode/` tracks portable preferences in writable `config.toml`; the `jcode-config` clean filter removes theme selection and generated launch-hotkey state before indexing.
- Jcode credentials, sessions, memory, caches, telemetry, generated browser assets, MCP discovery output, and its local `skills/` cache stay machine-local. skills.sh currently exposes no Jcode target, so Config validates shared skills in `~/.agents/skills` instead of requiring duplicate Jcode copies.
- `oldSetup/`, `.audit-backup/`, and any `auth.json` files are gitignored; runtime state and local backups never get committed.
- OMP runtime state stays local; only curated files like `config.yml`, `models.yml`, `mcp.json`, keybindings, skills, commands, tools, extensions, and themes are stowed. `omp-plugins.txt` tracks desired marketplace packages while OMP keeps its installed registry and cache under `~/.omp/plugins/`. See `docs/agents/omp.md`.
- `just link` and `just up` preflight every Stow package, link all packages, provision locked native Claude tool dependencies beside their repository files, then provision shared skills and OMP plugins. This ordering prevents package or manifest updates when a Stow conflict would leave a partial restore. `just claude` also provisions native Claude tool dependencies; the other package-only commands affect only their package-specific state.
- External skill installs target `claude-code,codex,pi,grok` by default; set `SKILLS_AGENTS` to narrow that to a different explicit target filter. Manifest `agents=` ownership remains authoritative. `just dev-modules` creates ignored links to the installed Pi/OMP type packages so LSP and `tsc` use the same versions as the live harnesses.
- Lavish is globally owned through `skills.txt`, which installs upstream commit `55045850f61e7b23c8dbf8cc30d92bd6e31649d2` from a disposable checkout and verifies blob `1e792b5bfc34127ca6f00bbb44fea306ce77ea52` during `just skills-status` and `just skills-install`.
- Fresh or disposable checkouts should run `just skills-status` and then `just skills-install` before using Lavish; `just pi` only restows package-specific state, while `just claude` also provisions repository-owned native tool dependencies.
- To roll back the project-ownership removal, revert its commit and rerun `just skills-install` to restore the pinned global copy and generated agent links; do not reintroduce a Pi package copy.
- Worktrees (`EnterWorktree` / `ExitWorktree`) write to `.agents/atlas/worktrees/`, which is also gitignored.
