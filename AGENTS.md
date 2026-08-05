# Atlas Config - Agent Instructions

This repo is a minimal, stow-managed config for Ed's agents - `claude`, `pi`, `omp`, and `jcode`. The `claude` package stow-manages `~/.claude/skills/` and `~/.claude/agents/` (native skill and agent store, adopted 2026-07-31 after the LifeOS uninstall) alongside the curated Claude Code config.

## DOX framework

This repository uses DOX as the AGENTS.md hierarchy for durable agent instructions.

### Core contract

- AGENTS.md files are binding work contracts for their subtrees.
- Work products, source materials, instructions, records, assets, and durable docs must stay understandable from the nearest applicable AGENTS.md plus every parent AGENTS.md above it.

### Read before editing

1. Read this root AGENTS.md.
2. Identify every file or folder you expect to touch.
3. Walk from the repository root to each target path.
4. Read every AGENTS.md found along each route.
5. If a parent AGENTS.md lists a child AGENTS.md whose scope contains the path, read that child and continue from there.
6. Use the nearest AGENTS.md as the local contract and parent docs for repo-wide rules.
7. If docs conflict, the closer doc controls local work details, but no child doc may weaken DOX.

Do not rely on memory. Re-read the applicable DOX chain in the current session before editing.

### Update after editing

Every meaningful change requires a DOX pass before the task is done.

Update the closest owning AGENTS.md when a change affects:

- purpose, scope, ownership, or responsibilities
- durable structure, contracts, workflows, or operating rules
- required inputs, outputs, permissions, constraints, side effects, or artifacts
- user preferences about behavior, communication, process, organization, or quality
- AGENTS.md creation, deletion, move, rename, or index contents

Update parent docs when parent-level structure, ownership, workflow, or child index changes. Update child docs when parent changes alter local rules. Remove stale or contradictory text immediately. Small edits that do not change behavior or contracts may leave docs unchanged, but the DOX pass still must happen.

### Hierarchy

- Root AGENTS.md is the DOX rail: project-wide instructions, global preferences, durable workflow rules, and the top-level Child DOX Index.
- Child AGENTS.md files own domain-specific instructions and their own Child DOX Index.
- Each parent explains what its direct children cover and what stays owned by the parent.
- The closer a doc is to the work, the more specific and practical it must be.

### Child doc shape

Create a child AGENTS.md when a folder becomes a durable boundary with its own purpose, rules, responsibilities, workflow, materials, or quality standards.

Default section order:

- Purpose
- Ownership
- Local Contracts
- Work Guidance
- Verification
- Child DOX Index

### Style

- Keep docs concise, current, and operational.
- Document stable contracts, not diary entries.
- Put broad rules in parent docs and concrete details in child docs.
- Prefer direct bullets with explicit names.
- Do not duplicate rules across many files unless each scope needs a local version.
- Delete stale notes instead of explaining history.
- Trim obvious statements, repeated rules, misplaced detail, and warnings for risks that no longer exist.

### Closeout

1. Re-check changed paths against the DOX chain.
2. Update nearest owning docs and any affected parents or children.
3. Refresh every affected Child DOX Index.
4. Remove stale or contradictory text.
5. Run existing verification when relevant.
6. Report any docs intentionally left unchanged and why.

### Child DOX Index

- `packages/AGENTS.md` - installable agent package rules and package child index.
- `docs/AGENTS.md` - durable documentation rules.
- Root-owned durable scopes include `global/AGENTS.md`, `agents.toml`, `manage-agents.py`, `claude/`, `pi/`, `omp/`, `jcode/`, `.claude-plugin/`, `.agents/`, `skills.txt`, and `install-skills.sh`.
- Harness-local operating rules live in each package's tracked `rules/RULES.md`; they are runtime instruction indexes, not repository DOX roots.

## LifeOS status

LifeOS (formerly PAI) was fully uninstalled on 2026-07-31: Pulse daemons, menu bar app, hooks, and the `~/.claude/LIFEOS` system tree are gone. Only the user data trees survive locally (`~/.claude/LIFEOS/USER` and `~/.claude/LIFEOS/MEMORY`, plus top-level `~/.claude/MEMORY`); nothing in this repo may depend on LifeOS machinery. The pre-uninstall skill/agent snapshot is recoverable from git history (commit `3588dda`).

## Scope

**atlas-config is single-machine-canonical.** The `merge=ours` rules on `claude/.claude/settings.json` and `pi/.pi/agent/settings.json` (see `.gitattributes`) mean changes to those files do not propagate across machines via `git pull` - local edits always win. Multi-machine sync requires explicit `git checkout origin/master -- <path>` on the receiving Mac, then a re-personalization pass if needed. This is intentional: these settings files carry per-session mutating fields (`counts`, `feedbackSurveyState.lastShownTime`, UI/model/package prefs) that would otherwise churn the repo.

The Recall MCP database (`~/.agents/Recall/recall.db`, shared by all agents) and the Recall MEMORY/ vault (`~/.agents/Recall/MEMORY/`) also live **outside** atlas-config - they are out-of-band like `~/.env`. Cold-archive these to an external location on a schedule (1Password vault, Time Machine, external drive); they are the only USER content not in git.

**Recall rule ownership:** Atlas Config owns the stowed Claude operating rule at `claude/.claude/rules/memory.md`; Recall owns the upstream behavior contract in `~/Developer/Atlas/Recall/FOR_CLAUDE.md` and the live `recall-memory` MCP schemas. The machine-local `~/.claude/CLAUDE.md` bootstrap must not duplicate the memory block; Claude loads the stowed rule separately. When Recall changes session tiers, MCP arguments, or memory safety rules, revalidate the Atlas rule against both upstream sources and run `just claude`. Keep the Atlas rule concise; do not copy the full Recall guide into this repo.

## Repo layout

| Surface | Source | Installed to | Status |
|---|---|---|---|
| `global` | `global/AGENTS.md` | `~/AGENTS.md` | Harness-neutral global contract; read-only to agents unless Ed explicitly requests a global change |
| Agent registry | `agents.toml`, `manage-agents.py` | Native harness instruction entry points | Reconciles shared links, local rule indexes, backups, archive, and restore |
| `claude` | `claude/.claude/` | `~/.claude/` | Curated Claude Code config; runtime settings/hooks may remain machine-local |
| `pi` | `pi/.pi/agent/` | `~/.pi/agent/` | Curated Pi config only; stowed with `--no-folding` so runtime state stays local |
| `omp` | `omp/.omp/agent/` | `~/.omp/agent/` | Curated OMP-native config only; stowed with `--no-folding` so generated integrations and runtime state stay local |
| `jcode` | `jcode/.jcode/` | `~/.jcode/` | Curated portable preferences only; runtime state, credentials, generated integrations, and themes stay local |
| Skills (external) | `skills.txt`, `install-skills.sh` | skills.sh global install targets | Manifest-installed shared skills |
| Skills/agents (native) | `claude/.claude/skills/`, `claude/.claude/agents/` | `~/.claude/skills/`, `~/.claude/agents/` | Stowed native store (adopted 2026-07-31); skills.sh symlinks resolve through it |
| OMP plugins | `omp-plugins.txt`, `install-omp-plugins.sh` | `~/.omp/plugins/` | Manifest-declared user-scoped OMP marketplace plugins; installed registry and caches stay local |

Project-local Atlas artifacts live under `.agents/atlas/` (`plans/`, `handoffs/`, `artifacts/`, `worktrees/`). The entire `.agents/atlas/` folder is gitignored - local-only and untracked; these docs do not sync via git. Tracked durable documentation lives under `docs/`.

Before changing Pi's global package set or implementing `/atlas-flow`, review the atlas-flow epic (issues #2-#11 in `edheltzel/atlas-config`) for the minimal-global baseline, flow manifest schema direction, and safety rules.

## Echo system

Echo lives in a separate repo at `~/Developer/Atlas/Echo/` - a separate installation, not inside atlas-config. Its TTS daemon listens on **port 8888**; Echo is the sole voice/notification system. Pulse (`:31337`) was uninstalled with LifeOS on 2026-07-31 - nothing listens there. All voice traffic must target `localhost:8888/notify`.

**Install:**

```
cd ~/Developer/Atlas/Echo
bash scripts/install.sh --adapter claudecode
```

The installer writes `~/Library/LaunchAgents/com.echo.plist` (Label `com.echo`) and starts the Bun daemon. It also quarantines legacy `com.pai.voice-server` and `com.atlas.voicesystem` plists when found; do not resurrect those services.

Provider chain: edge-tts → ElevenLabs → Kokoro → macOS `say`. **ElevenLabs is disabled by policy** - library voices require paid tier and previously produced ~80 HTTP 402 failures. edge-tts is the active provider with `say` as the floor.

**Live management:** `~/Developer/Atlas/Echo/scripts/{start,stop,restart,status}.sh`, or `launchctl kickstart -k "gui/$UID/com.echo"`. Logs at `~/Library/Logs/echo.log`. Health: `curl http://localhost:8888/health`. Full docs in the Echo repo's `AGENTS.md`.

All stowed skills and agents were migrated to `http://localhost:8888/notify` on 2026-07-31; no `:31337` references remain. If a new skill arrives curling `:31337`, fix it to `:8888` on sight.

## Stow

Multi-package stow uses the repo root as `STOW_DIR`. See `justfile` for commands. `global/AGENTS.md` is stowed to `~/AGENTS.md`; `manage-agents.py` then links each harness's native instruction entry point to that global file according to `agents.toml`. OMP, Pi, and Jcode are stowed with `--no-folding`: their runtime directories must remain real directories, with only curated config files symlinked into them. This prevents runtime-generated integrations such as Orca's OMP extensions from being written through a folded link into the repository. The per-package `.stow-local-ignore` files **replace** stow's built-in defaults - when adding a new one, copy from `claude/.stow-local-ignore` and re-include `\.git`, `\.gitignore`, etc.

Keep package source paths identical in case to their runtime paths. Stow compares symlink ownership lexically even on case-insensitive APFS; Claude rules therefore live at lowercase `claude/.claude/rules/`, matching `~/.claude/rules/`.

**Broken links block stow - `just prune-broken` clears them.** Stow only replaces links that resolve *into* the current stow dir; anything else is "not owned by stow" and aborts the **entire** package restow, so one stale link stops every other link from being created. Stow has no option to clean these up (checked against 2.4.1: `--adopt` only imports real files). `prune-broken` runs automatically before `update`/`claude`/`omp`/`pi`, and can be run alone with optional package names.

It finds targets by parsing `stow -n --restow` conflict output, so it respects `.stow-local-ignore` and folding, then removes only paths that are **both** stow-blocking and dangling - real files and valid links pointing outside the repo survive and still surface as genuine conflicts. Deletions go through `trash` when available. The usual trigger is a moved or renamed repo (see the `~/Developer/Atlas/` migration); if a restow ever fails with `existing target is not owned by stow`, run `just prune-broken` and retry.

Detection depends on stow's conflict wording (`existing target is not owned by stow: <path>`), so a stow upgrade could change it. That fails safe, not dangerous: an unrecognized message prunes nothing and the restow aborts loudly, back to manual cleanup. Nothing can be deleted that stow isn't already refusing to overwrite. If restows start aborting again after a stow bump, check that string first.

## When adding files

- Adding a coding harness or changing an instruction target -> edit `agents.toml`, follow `README.md#adding-a-coding-harness`, then run `just agents` and `just agents-check`.
- Harness-specific rules belong in the tracked `<package>/.../rules/RULES.md` index or its child rule files. Never put them in `global/AGENTS.md`.
- Adding to `claude/.claude/`, `omp/.omp/agent/`, or curated files under `pi/.pi/agent/` or `jcode/.jcode/` -> run `just up` to restow and provision both tracked manifests.
- Jcode runtime state (`auth*.json`, sessions, memory, caches, models, telemetry, browser/hotkey generated assets, logs, builds, and machine-specific MCP output) stays in the real `~/.jcode`, not in atlas-config.
- Adding, removing, or changing shared skills → edit `skills.txt`, then run `just skills-status` and `just skills-install`.
- Adding, removing, or changing OMP marketplace plugins → edit `omp-plugins.txt`, then run `just omp-plugins-status` and `just omp-plugins-install`.
- Anything that should not be symlinked into the runtime location → add a pattern to that package's `.stow-local-ignore`.
- Archived config directories (`claude/.claude/_archived/`, `omp/.omp/**/_archived/`, `pi/.pi/**/_archived/`) are repo-only and must stay ignored by stow.
- Pi runtime/package state (`auth.json`, `sessions/`, `cache/`, `mcp*.json`, `run-history.jsonl`, `memory/`, `npm/`, `git/`) stays in the real `~/.pi/agent`, not in atlas-config.
- OMP runtime state (`agent.db*`, `history.db*`, `models.db*`, `sessions/`, `blobs/`, `terminal-sessions/`, `memories/`, `cache/`, generated integrations, `secrets.yml`, `.env`, auth broker/gateway tokens) stays in the real `~/.omp`, not in atlas-config. See `docs/agents/omp.md`.
- Secrets (`auth.json`, `*.env`, keys) → already covered by `.gitignore`. Do not weaken those rules.

## Notable conventions

- `docs/plans/` for active plan/design docs, `docs/plans/archive/` for completed.
- `docs/plans/handoffs/` for live handoffs, archived to `docs/plans/handoffs/archive/` when consumed.
- File naming: `YYYY-MM-DD-description.md`.
- Use git worktrees (`EnterWorktree` / `ExitWorktree`) for implementation work - they land in `.agents/worktrees/`.

## Agent skills

### Skills manifest (skills.sh)

Two skill stores coexist since 2026-07-31: **external/shared skills** are manifest-managed through `skills.txt` (installed by skills.sh into `~/.agents/skills/`, symlinked into the skills dir), and **native skills** live directly in the stowed `claude/.claude/skills/` store (adopted when `~/.claude/skills` moved under stow after the LifeOS uninstall). For external skills the canonical source is `skills.txt`: each line maps a skills.sh source to selected skills with `owner/repo[#immutable-commit]=skill,skill`; an optional `|upstream/path|git-blob-sha` suffix records an integrity check for one selected skill. `install-skills.sh` preserves pinned refs as policy metadata because the current skills.sh CLI treats `#ref` as a branch, installs from the base source, and rejects installed-byte drift against the recorded blob.

Workflow: edit `skills.txt` → `just skills-status` for a dry run → `just skills-install` to install through skills.sh. `just link` and `just up` provision shared skills; package-only `just claude`, `just omp`, and `just pi` commands do not. Use `SKILLS_AGENTS=\"claude-code,pi,codex\"` to target specific agents; the default is all detected agents. Native skills may live directly in the stowed `claude/.claude/skills/` store; migrating one to its own skills.sh-compatible repo pulled in through `skills.txt` is preferred when it becomes shareable across harnesses. The tracked Pi RedTeam and Art adaptations are deliberate harness-specific exceptions owned by `pi/.pi/agent/rules/RULES.md`.

Lavish is globally owned through `skills.txt`: its entry pins upstream commit `55045850f61e7b23c8dbf8cc30d92bd6e31649d2` and verifies Git blob `1e792b5bfc34127ca6f00bbb44fea306ce77ea52` during `just skills-status` and `just skills-install`. The project-scoped `.agents/skills/lavish/SKILL.md`, `.claude/skills/lavish`, and project lock entry were removed after the approved stage 2 fallback decision; do not recreate them or add a renamed fixture without a concrete later test need.

### OMP plugin manifest

OMP-owned marketplace packages are manifest-managed separately from cross-harness skills. The canonical source is `omp-plugins.txt`; each line maps `owner/repo=plugin@marketplace`. `install-omp-plugins.sh` adds or updates the marketplace, then installs or upgrades the user-scoped plugin. The OMP runtime registry and downloaded plugin cache under `~/.omp/plugins/` remain untracked.

Workflow: edit `omp-plugins.txt` → `just omp-plugins-status` for a dry run → `just omp-plugins-install` to apply it. `just omp`, `just link`, and `just up` also provision the manifest. Compound Engineering is OMP-owned through `compound-engineering@compound-engineering-plugin`; keep the Claude installation for Claude Code, and do not copy the Pi `resources_discover` bridge into `omp/.omp/agent/extensions/`.

### Issue tracker

Issues are tracked as GitHub issues in `edheltzel/atlas-config` via the `gh` CLI. See `docs/agents/issue-tracker.md`.

### PM tools engine

`pm-tools` is installed via `skills.txt` from `edheltzel/skills`. Treat its source of truth as the upstream skills.sh repo/runtime installation, not a local `skills/pm-tools/` tree in atlas-config.

### Triage labels

Five canonical triage roles using default label strings (`needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, `wontfix`). See `docs/agents/triage-labels.md`.

### OMP config

OMP-native config is stowed from `omp/.omp/agent/` to `~/.omp/agent/`. Track authored config only; keep runtime DBs, sessions, blobs, memories, and secrets local. Published OMP releases are canonical: install them with `omp update`; never restore a local source link, downstream patch branch, or an older pinned release. See `docs/agents/omp.md`.

### Domain docs

Single-context - one `CONTEXT.md` + `docs/adr/` at the repo root. See `docs/agents/domain.md`.

### GitHub identities

Three GitHub principals operate on the `Gentle-Art-Technologies` org: `edheltzel`, `Atlas-Key`, and `gat-atlas-bot`. Route by authorship and operation: Ed-authored changes stay under `edheltzel` even when an agent performs the mechanics; Atlas/agent-authored work uses the Atlas SSH signing identity for commits and `Atlas-Key` for attributed GitHub API actions; privileged unattended automation uses the App. Never relabel Ed's work merely because an agent executed it. SSH commit signing does not replace API authentication. See `docs/agents/github-identities.md`.

## Maintaining this file

Keep this file for knowledge useful to almost every future agent session in this project.
Do not repeat what the codebase already shows; point to the authoritative file or command instead.
Prefer rewriting or pruning existing entries over appending new ones.
When updating this file, preserve this bar for all agents and keep entries concise.
