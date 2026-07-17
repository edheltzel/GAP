# Atlas Config — Agent Instructions

This repo is a minimal, stow-managed config for Ed's agents — `claude`, `pi`, and `omp`. Ed's PAI v5.0.0 USER-CONFIG (identity, TELOS, durable MEMORY, SYSTEM-OVERRIDE hooks/Pulse/TOOLS) lives in the `claude` package.

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

- `pi/.pi/agent/AGENTS.md` — Pi package-local rules for curated Pi agent config.
- `packages/AGENTS.md` — installable agent package rules and package child index.
- `docs/AGENTS.md` — durable documentation rules.
- Root-owned durable scopes without child AGENTS.md today: `claude/`, `omp/`, `.claude-plugin/`, `.agents/`, `skills.txt`, `install-skills.sh`.

## PAI v5.0.0 migration

This repo tracks Ed's PAI v5.0.0 configuration (identity, TELOS, durable memory, SYSTEM-OVERRIDE hooks) in the `claude` package. The migration is complete; this tree is the migrated state.

## Scope

**atlas-config is single-machine-canonical.** The `merge=ours` rules on `claude/.claude/settings.json` and `pi/.pi/agent/settings.json` (see `.gitattributes`) mean changes to those files do not propagate across machines via `git pull` — local edits always win. Multi-machine sync requires explicit `git checkout origin/master -- <path>` on the receiving Mac, then a re-personalization pass if needed. This is intentional: these settings files carry per-session mutating fields (`counts`, `feedbackSurveyState.lastShownTime`, UI/model/package prefs) that would otherwise churn the repo.

The Recall MCP database (`~/.agents/Recall/recall.db`, shared by all agents) and the Recall MEMORY/ vault (`~/.agents/Recall/MEMORY/`) also live **outside** atlas-config — they are out-of-band like `~/.env`. Cold-archive these to an external location on a schedule (1Password vault, Time Machine, external drive); they are the only USER content not in git.

**Recall rule ownership:** Atlas Config owns the stowed Claude operating rule at `claude/.claude/rules/memory.md`; Recall owns the upstream behavior contract in `~/Developer/Atlas/Recall/FOR_CLAUDE.md` and the live `recall-memory` MCP schemas. The machine-local `~/.claude/CLAUDE.md` bootstrap must not duplicate the memory block; Claude loads the stowed rule separately. When Recall changes session tiers, MCP arguments, or memory safety rules, revalidate the Atlas rule against both upstream sources and run `just claude`. Keep the Atlas rule concise; do not copy the full Recall guide into this repo.

## Repo layout

| Surface | Source | Installed to | Status |
|---|---|---|---|
| `claude` | `claude/.claude/` | `~/.claude/` | Curated Claude Code config; runtime settings/hooks may remain machine-local |
| `pi` | `pi/.pi/agent/` | `~/.pi/agent/` | Curated Pi config only; stowed with `--no-folding` so runtime state stays local |
| `omp` | `omp/.omp/agent/` | `~/.omp/agent/` | Curated OMP-native config only; runtime DBs, sessions, blobs, memories, and secrets stay local |
| Skills | `skills.txt`, `install-skills.sh` | skills.sh global install targets | Manifest-installed skills; no repo-local `skills/` store |
| OMP plugins | `omp-plugins.txt`, `install-omp-plugins.sh` | `~/.omp/plugins/` | Manifest-declared user-scoped OMP marketplace plugins; installed registry and caches stay local |

Project-local Atlas artifacts live under `.agents/atlas/` (`plans/`, `handoffs/`, `artifacts/`, `worktrees/`). The entire `.agents/atlas/` folder is gitignored — local-only and untracked; these docs do not sync via git. Tracked durable documentation lives under `docs/`.

Before changing Pi's global package set or implementing `/atlas-flow`, review the atlas-flow epic (issues #2–#11 in `edheltzel/atlas-config`) for the minimal-global baseline, flow manifest schema direction, and safety rules.

## Echo system

Echo lives in a separate repo at `~/Developer/Atlas/Echo/` — a separate installation, not inside atlas-config. Its TTS daemon listens on **port 8888**, not Pulse's `31337`. `localhost:31337/notify` returns `Not found` because Pulse's voice module is disabled; all Echo traffic must target `localhost:8888/notify`.

**Install:**

```
cd ~/Developer/Atlas/Echo
bash scripts/install.sh --adapter claudecode
```

The installer writes `~/Library/LaunchAgents/com.echo.plist` (Label `com.echo`) and starts the Bun daemon. It also quarantines legacy `com.pai.voice-server` and `com.atlas.voicesystem` plists when found; do not resurrect those services.

Provider chain: edge-tts → ElevenLabs → Kokoro → macOS `say`. **ElevenLabs is disabled by policy** — library voices require paid tier and previously produced ~80 HTTP 402 failures. edge-tts is the active provider with `say` as the floor.

**Live management:** `~/Developer/Atlas/Echo/scripts/{start,stop,restart,status}.sh`, or `launchctl kickstart -k "gui/$UID/com.echo"`. Logs at `~/Library/Logs/echo.log`. Health: `curl http://localhost:8888/health`. Full docs in the Echo repo's `AGENTS.md`.

**Gotcha — wrong-port curls:** PAI Algorithm doctrine (`PAI/ALGORITHM/v6.3.0.md`) and many skills curl `http://localhost:31337/notify`. That's Pulse, which has voice off. Until those are migrated, Echo announcements from the Algorithm side silently no-op. Real Echo endpoint = `:8888`.

## Stow

Multi-package stow with the repo root as `STOW_DIR`. See `justfile` for commands. Pi is stowed with `--no-folding`: `~/.pi/agent` must remain a real directory, with only curated config files symlinked into it. The per-package `.stow-local-ignore` files **replace** stow's built-in defaults — when adding a new one, copy from `claude/.stow-local-ignore` and re-include `\.git`, `\.gitignore`, etc.

Keep package source paths identical in case to their runtime paths. Stow compares symlink ownership lexically even on case-insensitive APFS; Claude rules therefore live at lowercase `claude/.claude/rules/`, matching `~/.claude/rules/`.

**Broken links block stow — `just prune-broken` clears them.** Stow only replaces links that resolve *into* the current stow dir; anything else is "not owned by stow" and aborts the **entire** package restow, so one stale link stops every other link from being created. Stow has no option to clean these up (checked against 2.4.1: `--adopt` only imports real files). `prune-broken` runs automatically before `update`/`claude`/`omp`/`pi`, and can be run alone with optional package names.

It finds targets by parsing `stow -n --restow` conflict output, so it respects `.stow-local-ignore` and folding, then removes only paths that are **both** stow-blocking and dangling — real files and valid links pointing outside the repo survive and still surface as genuine conflicts. Deletions go through `trash` when available. The usual trigger is a moved or renamed repo (see the `~/Developer/Atlas/` migration); if a restow ever fails with `existing target is not owned by stow`, run `just prune-broken` and retry.

Detection depends on stow's conflict wording (`existing target is not owned by stow: <path>`), so a stow upgrade could change it. That fails safe, not dangerous: an unrecognized message prunes nothing and the restow aborts loudly, back to manual cleanup. Nothing can be deleted that stow isn't already refusing to overwrite. If restows start aborting again after a stow bump, check that string first.

## When adding files

- Adding to `claude/.claude/`, `omp/.omp/agent/`, or curated files under `pi/.pi/agent/` → run `just up` to restow and provision both tracked manifests.
- Adding, removing, or changing shared skills → edit `skills.txt`, then run `just skills-status` and `just skills-install`.
- Adding, removing, or changing OMP marketplace plugins → edit `omp-plugins.txt`, then run `just omp-plugins-status` and `just omp-plugins-install`.
- Anything that should not be symlinked into the runtime location → add a pattern to that package's `.stow-local-ignore`.
- Archived config directories (`claude/.claude/_archived/`, `omp/.omp/**/_archived/`, `pi/.pi/**/_archived/`) are repo-only and must stay ignored by stow.
- Pi runtime/package state (`auth.json`, `sessions/`, `cache/`, `mcp*.json`, `run-history.jsonl`, `memory/`, `npm/`, `git/`) stays in the real `~/.pi/agent`, not in atlas-config.
- OMP runtime state (`agent.db*`, `history.db*`, `models.db*`, `sessions/`, `blobs/`, `terminal-sessions/`, `memories/`, `cache/`, `secrets.yml`, `.env`, auth broker/gateway tokens) stays in the real `~/.omp`, not in atlas-config. See `docs/agents/omp.md`.
- Secrets (`auth.json`, `*.env`, keys) → already covered by `.gitignore`. Do not weaken those rules.

## Notable conventions

- `docs/plans/` for active plan/design docs, `docs/plans/archive/` for completed.
- `docs/plans/handoffs/` for live handoffs, archived to `docs/plans/handoffs/archive/` when consumed.
- File naming: `YYYY-MM-DD-description.md`.
- Use git worktrees (`EnterWorktree` / `ExitWorktree`) for implementation work — they land in `.agents/worktrees/`.

## Agent skills

### Skills manifest (skills.sh)

Custom skills are manifest-managed, not stored as a repo-backed `skills/` directory. The canonical source is `skills.txt`: each line maps a skills.sh source to selected skills with `owner/repo=skill,skill`. `install-skills.sh` reads that manifest and runs `skills add <source> -g -a <agents> -s <skills> -y` for every source.

Workflow: edit `skills.txt` → `just skills-status` for a dry run → `just skills-install` to install through skills.sh. Use `SKILLS_AGENTS=\"claude-code,pi,codex\"` to target specific agents; the default is all detected agents. Native/custom skills that used to live in atlas-config should move to their own skills.sh-compatible repos and be pulled in through `skills.txt`; do not recreate a local `skills/` store.

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

Single-context — one `CONTEXT.md` + `docs/adr/` at the repo root. See `docs/agents/domain.md`.

### GitHub identities

Three distinct GitHub principals operate on the `Gentle-Art-Technologies` org — `edheltzel` (human owner), `Atlas-Key` (teammate User, fine-grained PAT `ATLAS_KEY_PAT`), and `gat-atlas-bot` (App `4127929`, ephemeral installation tokens). An agent picks one per action type; act as the teammate with `GH_TOKEN=$ATLAS_KEY_PAT gh …`. See `docs/agents/github-identities.md`.
