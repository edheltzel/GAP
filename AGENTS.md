# Atlas Config - Agent Instructions

This repo is a minimal, stow-managed config for Ed's agents: `claude`, `pi`, `omp`, and `jcode`. The Claude package tracks the recovered native skill and agent snapshot alongside authored Claude config; shared external skills remain manifest-managed.

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

- `pi/.pi/agent/AGENTS.md` - Pi package-local rules for curated Pi agent config.
- `packages/AGENTS.md` - installable agent package rules and package child index.
- `docs/AGENTS.md` - durable documentation rules.
- Root-owned durable scopes without child AGENTS.md today: `claude/`, `omp/`, `jcode/`, `.claude-plugin/`, `.agents/`, `skills.txt`, `install-skills.sh`.

## LifeOS status

The LifeOS runtime and system tree were removed on 2026-07-31. Local `USER` and `MEMORY` data survive under `~/.claude/LIFEOS/`, so recovered historical skills may still read those trees. No active status line or current contract may depend on the removed `LIFEOS/VERSION` or `LIFEOS/TOOLS`.

The native Claude inventory is derived from the final unmerged cleanup lineage, then maintained against current harness contracts. Focused tests reject operational references to removed LifeOS machinery and retired task APIs.

## Scope

**atlas-config is single-machine-canonical.** The `merge=ours` rules on `claude/.claude/settings.json`, `pi/.pi/agent/settings.json`, and `omp/.omp/agent/config.yml` (see `.gitattributes`) preserve local versions only when Git performs a merge that invokes the driver. A fast-forward that changes one of these tracked files can stop on local modifications; it does not invoke `merge=ours`. Before updating, capture a patch and restore the machine-local file explicitly when the incoming tree changes it. `just jcode-filter` registers the required repository-local `merge.ours.driver` along with Jcode's clean filter. Multi-machine sync requires an explicit checkout of the remote path followed by re-personalization. This is intentional: these settings files carry per-session mutating fields (`counts`, `feedbackSurveyState.lastShownTime`, UI/model/package prefs) that would otherwise churn the repo.

The Recall MCP database (`~/.agents/Recall/recall.db`, shared by all agents) and the Recall MEMORY/ vault (`~/.agents/Recall/MEMORY/`) also live **outside** atlas-config; they are out-of-band like `~/.env`. Cold-archive these to an external location on a schedule (1Password vault, Time Machine, external drive); they are the only USER content not in git.

**Recall rule ownership:** Atlas Config owns the stowed Claude operating rule at `claude/.claude/rules/memory.md`; Recall owns the upstream behavior contract in `~/Developer/Atlas/Recall/FOR_CLAUDE.md` and the live `recall-memory` MCP schemas. The machine-local `~/.claude/CLAUDE.md` bootstrap must not duplicate the memory block; Claude loads the stowed rule separately. When Recall changes session tiers, MCP arguments, or memory safety rules, revalidate the Atlas rule against both upstream sources and run `just claude`. Keep the Atlas rule concise; do not copy the full Recall guide into this repo.

## Repo layout

| Surface | Source | Installed to | Status |
| --- | --- | --- | --- |
| `claude` | `claude/.claude/` | `~/.claude/` | Curated Claude Code config; stowed with `--no-folding` so runtime state stays local |
| `pi` | `pi/.pi/agent/` | `~/.pi/agent/` | Curated Pi config only; stowed with `--no-folding` so runtime state stays local |
| `omp` | `omp/.omp/agent/` | `~/.omp/agent/` | Curated OMP-native config only; stowed with `--no-folding` so generated integrations and runtime state stay local |
| `jcode` | `jcode/.jcode/` | `~/.jcode/` | Curated portable preferences only; stowed with `--no-folding` so runtime state stays local |
| Native Claude skills/agents | `claude/.claude/{skills,agents}/` | `~/.claude/{skills,agents}/` | Exact curated snapshot from the final LifeOS-removal lineage |
| Shared skills | `skills.txt`, `install-skills.sh` | skills.sh global install targets | External skills provisioned across detected harnesses |
| OMP plugins | `omp-plugins.txt`, `install-omp-plugins.sh` | `~/.omp/plugins/` | Manifest-declared user-scoped OMP marketplace plugins; installed registry and caches stay local |

Project-local Atlas artifacts live under `.agents/atlas/` (`plans/`, `handoffs/`, `artifacts/`, `worktrees/`). The entire `.agents/atlas/` folder is gitignored, local-only, and untracked; these docs do not sync via git. Tracked durable documentation lives under `docs/`.

Before changing Pi's global package set or implementing `/atlas-flow`, review the atlas-flow epic (issues #2-#11 in `edheltzel/atlas-config`) for the minimal-global baseline, flow manifest schema direction, and safety rules.

## Echo system

Echo lives in a separate repo at `~/Developer/Atlas/Echo/`, not inside atlas-config. Its TTS daemon listens on **port 8888**, not Pulse's `31337`. `localhost:31337/notify` returns `Not found` because Pulse's voice module is disabled; all Echo traffic must target `localhost:8888/notify`.

**Install:**

```bash
cd ~/Developer/Atlas/Echo
bash scripts/install.sh --adapter claudecode
```

The installer writes `~/Library/LaunchAgents/com.echo.plist` (Label `com.echo`) and starts the Bun daemon. It also quarantines legacy `com.pai.voice-server` and `com.atlas.voicesystem` plists when found; do not resurrect those services.

Provider chain: edge-tts → ElevenLabs → Kokoro → macOS `say`. **ElevenLabs is disabled by policy** because library voices require paid tier and previously produced ~80 HTTP 402 failures. edge-tts is the active provider with `say` as the floor.

**Live management:** `~/Developer/Atlas/Echo/scripts/{start,stop,restart,status}.sh`, or `launchctl kickstart -k "gui/$UID/com.echo"`. Logs at `~/Library/Logs/echo.log`. Health: `curl http://localhost:8888/health`. Full docs in the Echo repo's `AGENTS.md`.

**Gotcha - wrong-port curls:** PAI Algorithm doctrine (`PAI/ALGORITHM/v6.3.0.md`) and many skills curl `http://localhost:31337/notify`. That's Pulse, which has voice off. Until those are migrated, Echo announcements from the Algorithm side silently no-op. Real Echo endpoint = `:8888`.

## Stow

Multi-package stow with the repo root as `STOW_DIR`. See `justfile` for commands. Claude, OMP, Pi, and Jcode are stowed with `--no-folding`: their runtime directories must remain real directories, with only curated config files symlinked into them. This prevents runtime-generated state and integrations from being written through folded links into the repository. Aggregate restore commands run a conflict-aware Stow preflight before linking any package, then provision locked native Claude tool dependencies, shared skills, and OMP plugins only after every package succeeds. The per-package `.stow-local-ignore` files **replace** stow's built-in defaults; when adding a new one, copy from `claude/.stow-local-ignore` and re-include `\.git`, `\.gitignore`, etc.

Jcode rewrites its tracked `config.toml`; `just jcode-filter` configures the repository-local clean filter that strips theme selection and generated launch-hotkey state before indexing, plus the `merge.ours.driver` required by `.gitattributes`.

Local TypeScript tooling resolves the installed Pi/OMP packages through ignored links created by `just dev-modules`; never commit `node_modules` or install a second harness dependency tree. Native Claude tools with package manifests install locked dependencies beside their real repository files through `just native-skill-tools`; `just native-skill-tools-status` validates locks and installed declared dependencies without mutation. Run `just tooling-preflight` for config parsing, type checking, manifest checks, focused native-skill tests, and executable status-line tests.

Keep package source paths identical in case to their runtime paths. Stow compares symlink ownership lexically even on case-insensitive APFS; Claude rules therefore live at lowercase `claude/.claude/rules/`, matching `~/.claude/rules/`.

**Broken links block stow; `just prune-broken` clears them.** Stow only replaces links that resolve *into* the current stow dir; anything else is "not owned by stow" and aborts the **entire** package restow, so one stale link stops every other link from being created. Stow has no option to clean these up (checked against 2.4.1: `--adopt` only imports real files). `prune-broken` runs automatically before `update`/`claude`/`omp`/`pi`/`jcode`, and can be run alone with optional package names.

It finds targets by parsing `stow -n --restow` conflict output, so it respects `.stow-local-ignore` and folding, then removes only paths that are **both** stow-blocking and dangling; real files and valid links pointing outside the repo survive and still surface as genuine conflicts. Deletions go through `trash` when available. The usual trigger is a moved or renamed repo (see the `~/Developer/Atlas/` migration); if a restow ever fails with `existing target is not owned by stow`, run `just prune-broken` and retry.

Detection depends on stow's conflict wording (`existing target is not owned by stow: <path>`), so a stow upgrade could change it. That fails safe, not dangerous: an unrecognized message prunes nothing and the restow aborts loudly, back to manual cleanup. Nothing can be deleted that stow isn't already refusing to overwrite. If restows start aborting again after a stow bump, check that string first.

## When adding files

- Adding to `claude/.claude/`, `omp/.omp/agent/`, `pi/.pi/agent/`, or `jcode/.jcode/` → run `just up` to restow, provision native Claude tools, and install both tracked manifests.
- Jcode runtime state (`auth*.json`, sessions, memory, caches, models, telemetry, browser/hotkey assets, logs, builds, and machine-specific MCP output) stays in the real `~/.jcode`, not in atlas-config.
- Adding, removing, or changing native Claude skills or agents → edit `claude/.claude/{skills,agents}/`, add or update a focused behavior test, run `tests/claude/native-skills-snapshot.test.sh`, then run `just claude` from the canonical clone. Native tool package directories must include a committed Bun lockfile and appear in `install-native-skill-tools.sh`.
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
- Use git worktrees (`EnterWorktree` / `ExitWorktree`) for implementation work; they land in `.agents/worktrees/`.

## Agent skills

### Skill stores

Two skill stores coexist. The native Claude inventory recovered from the LifeOS-removal lineage lives directly under `claude/.claude/skills/`; this repository maintains that tree. External skills are manifest-managed through `skills.txt`, which accepts remote `owner/repo[#immutable-commit]` sources and pinned `local:path#immutable-commit` sibling sources. `|upstream/path|git-blob-sha` records one selected skill's integrity; a final `|agents=name,name` limits harness ownership. Do not track top-level shared-skill symlinks in the Claude Stow package. `install-skills.sh` prepares immutable disposable checkouts and rejects installed-byte drift.

Maintain recovered native skills through focused, behavior-backed changes. Do not preserve removed LifeOS machinery when a current harness adapter or explicit fail-closed boundary can replace it. `docs/agents/native-claude-skills.md` records pointer exceptions, external ownership, and verification.

Firstmate's 19 internal skills are canonical only in the sibling Firstmate repository. Their `metadata.internal: true` contract means they must not enter Config's native inventory or `skills.txt`. `just ownership-status` validates that sibling checkout when present and verifies tldraw Desktop's declared app-owned files and hooks; Config does not track duplicate tldraw artifacts. `just installed-skills-status` fails on missing manifest/native skills or dangling links under managed skill roots, and both checks run in `just preflight`.

Workflow for external skills: edit `skills.txt` → `just skills-status` for a dry run → `just skills-install` to install through skills.sh. `tests/claude/external-skills-inventory.test.sh` pins the recoverable final-lineage inventory. `just link` and `just up` provision native tool dependencies and shared skills after package linking succeeds; `just claude` provisions only its native tool dependencies, `just pi` and `just jcode` only restow their package, and `just omp` also provisions OMP plugins. `SKILLS_AGENTS=\"claude-code,pi,codex\"` filters global targets and is intersected with per-entry ownership. The default lets skills.sh detect agents unless the entry narrows them. The tracked Pi RedTeam and Art adaptations remain deliberate harness-specific exceptions owned by `pi/.pi/agent/AGENTS.md`.

Lavish is globally owned through `skills.txt`: its entry installs upstream commit `55045850f61e7b23c8dbf8cc30d92bd6e31649d2` from a disposable checkout and verifies Git blob `1e792b5bfc34127ca6f00bbb44fea306ce77ea52` during `just skills-status` and `just skills-install`. The project-scoped `.agents/skills/lavish/SKILL.md`, `.claude/skills/lavish`, and project lock entry were removed after the approved stage 2 fallback decision; do not recreate them or add a renamed fixture without a concrete later test need.

Anti-slop has split ownership. Claude receives `anti-slop`, `slop-code`, `slop-review`, `slop-rewrite`, and `slop-verify` from the pinned local PiExtensions commit declared in `skills.txt`; every entry targets only `claude-code`. Pi discovers the same names from the registered `pi-anti-slop` package and must never receive duplicate skills.sh links.

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

OMP-native config is stowed from `omp/.omp/agent/` to `~/.omp/agent/`. Track authored config only; keep runtime DBs, sessions, blobs, memories, and secrets local. OMP rewrites `config.yml`; its `merge=ours` rule protects machine-local provider, model-role, and UI preferences during real merges, not fast-forwards. Patch and restore local changes explicitly when an incoming fast-forward changes the tracked file. Published OMP releases are canonical: install them with `omp update`; never restore a local source link, downstream patch branch, or an older pinned release. See `docs/agents/omp.md`.

### Domain docs

Single-context: one `CONTEXT.md` + `docs/adr/` at the repo root. See `docs/agents/domain.md`.

### GitHub identities

Three distinct GitHub principals operate on the `Gentle-Art-Technologies` org: `edheltzel` (human owner), `Atlas-Key` (teammate User, fine-grained PAT `ATLAS_KEY_PAT`), and `gat-atlas-bot` (App `4127929`, ephemeral installation tokens). An agent picks one per action type; act as the teammate with `GH_TOKEN=$ATLAS_KEY_PAT gh …`. See `docs/agents/github-identities.md`.

## Maintaining this file

Keep this file for knowledge useful to almost every future agent session in this project.
Do not repeat what the codebase already shows; point to the authoritative file or command instead.
Prefer rewriting or pruning existing entries over appending new ones.
When updating this file, preserve this bar for all agents and keep entries concise.
