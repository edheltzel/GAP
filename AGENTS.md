# atlas-config — Agent Instructions

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
- Root-owned durable scopes without child AGENTS.md today: `claude/`, `omp/`, `.claude-plugin/`, `.agents/`, `skills/`.

## PAI v5.0.0 migration

This repo tracks Ed's PAI v5.0.0 configuration (identity, TELOS, durable memory, SYSTEM-OVERRIDE hooks) in the `claude` package. The migration is complete; this tree is the migrated state.

## Scope

**atlas-config is single-machine-canonical.** The `merge=ours` rules on `claudecode/.claude/settings.json` and `pi/.pi/agent/settings.json` (see `.gitattributes`) mean changes to those files do not propagate across machines via `git pull` — local edits always win. Multi-machine sync requires explicit `git checkout origin/master -- <path>` on the receiving Mac, then a re-personalization pass if needed. This is intentional: these settings files carry per-session mutating fields (`counts`, `feedbackSurveyState.lastShownTime`, UI/model/package prefs) that would otherwise churn the repo.

The Recall MCP database (`~/.agents/Recall/recall.db`, shared by all agents) and the Recall MEMORY/ vault (`~/.agents/Recall/MEMORY/`) also live **outside** atlas-config — they are out-of-band like `~/.env`. Cold-archive these to an external location on a schedule (1Password vault, Time Machine, external drive); they are the only USER content not in git.

## Repo layout

| Package | Source | Stowed to | Status |
|---|---|---|---|
| `pi` | `pi/.pi/agent/` | `~/.pi/agent/` | Curated Pi config only; stowed with `--no-folding` so runtime state stays local |
| `omp` | `omp/.omp/agent/` | `~/.omp/agent/` | Curated OMP-native config only; runtime DBs, sessions, blobs, memories, and secrets stay local |

Project-local Atlas artifacts live under `.agents/atlas/` (`plans/`, `handoffs/`, `artifacts/`, `worktrees/`). The entire `.agents/atlas/` folder is gitignored — local-only and untracked (as of 2026-06-30); these docs do not sync via git.

Before changing Pi's global package set or implementing `/atlas-flow`, review the atlas-flow epic (issues #2–#11 in `edheltzel/atlas-config`) for the minimal-global baseline, flow manifest schema direction, and safety rules.

## Voice system

Voice lives in a separate repo at `~/Developer/atlas-voicesystem` — stowed alongside, not inside atlas-config. The TTS daemon listens on **port 8888**, not Pulse's `31337`. `localhost:31337/notify` returns `Not found` because Pulse's voice module is disabled; all voice traffic must target `localhost:8888/notify`.

**Install (two-step):**

```
cd ~/Developer/atlas-voicesystem && stow -v -t ~ claudecode
bash ~/.claude/PAI/USER/Voice/install.sh
```

`stow` creates the symlink `~/.claude/PAI/USER/Voice → ~/Developer/atlas-voicesystem/claudecode/.claude/PAI/USER/Voice`. `install.sh` writes `~/Library/LaunchAgents/com.pai.voice-server.plist` (Label `com.pai.voice-server`, `RunAtLoad`, `KeepAlive`) and starts the Bun daemon.

Provider chain: edge-tts → ElevenLabs → Kokoro → macOS `say`. **ElevenLabs is disabled by policy** — library voices require paid tier and previously produced ~80 HTTP 402 failures. edge-tts is the active provider with `say` as the floor.

**Live management:** `~/.claude/PAI/USER/Voice/{start,stop,restart,status}.sh`, or `launchctl kickstart -k "gui/$UID/com.pai.voice-server"`. Logs at `~/Library/Logs/pai-voice-server.log`. Health: `curl http://localhost:8888/health`. Full docs in the voicesystem repo's `AGENTS.md`.

**Gotcha — wrong-port curls:** PAI Algorithm doctrine (`PAI/ALGORITHM/v6.3.0.md`) and many skills curl `http://localhost:31337/notify`. That's Pulse, which has voice off. Until those are migrated, voice announcements from the Algorithm side silently no-op. Real voice = `:8888`.

## Stow

Multi-package stow with the repo root as `STOW_DIR`. See `justfile` for commands. Pi is stowed with `--no-folding`: `~/.pi/agent` must remain a real directory, with only curated config files symlinked into it. The per-package `.stow-local-ignore` files **replace** stow's built-in defaults — when adding a new one, copy from `claude/.stow-local-ignore` and re-include `\.git`, `\.gitignore`, etc.

## When adding files

- Adding to `claude/.claude/`, `omp/.omp/agent/`, or curated files under `pi/.pi/agent/` → run `just up` to restow.
- Anything that should not be symlinked into the runtime location → add a pattern to that package's `.stow-local-ignore`.
- Pi runtime/package state (`auth.json`, `sessions/`, `cache/`, `mcp*.json`, `run-history.jsonl`, `memory/`, `npm/`, `git/`) stays in the real `~/.pi/agent`, not in atlas-config.
- OMP runtime state (`agent.db*`, `history.db*`, `models.db*`, `sessions/`, `blobs/`, `terminal-sessions/`, `memories/`, `cache/`, `secrets.yml`, `.env`, auth broker/gateway tokens) stays in the real `~/.omp`, not in atlas-config. See `docs/agents/omp.md`.
- Secrets (`auth.json`, `*.env`, keys) → already covered by `.gitignore`. Do not weaken those rules.

## Notable conventions

- `.agents/atlas/plans/` for active plan/design docs, `.agents/atlas/plans/archive/` for completed.
- `.agents/atlas/handoffs/` for live handoffs, archived to `.agents/atlas/handoffs/archive/` when consumed.
- File naming: `YYYY-MM-DD-description.md`.
- Use git worktrees (`EnterWorktree` / `ExitWorktree`) for implementation work — they land in `.agents/worktrees/`.

## Agent skills

### Skills store (skills.sh)

The [skills.sh](https://skills.sh) global store is repo-backed: the real files live in `skills/` at the repo root, and `~/.agents/skills` is a plain symlink to it (created by `just skills-link`, **not** stow). skills.sh keeps working unchanged — `npx skills add owner/repo` writes through the symlink straight into the repo, and its per-agent symlinks (`~/.claude/skills/<name> → ../../.agents/skills/<name>`, etc.) resolve through the chain.

Workflow: `npx skills add`/`update` as usual → `just skills-sync` to review and commit the diff. The sync target refuses to run if `~/.agents/skills` is no longer a symlink (i.e. skills.sh recreated it as a real dir — relink and reconcile manually). `node_modules/` and `.DS_Store` inside the store are gitignored; do not track Pi CE runtime skills here (they stay under `~/.pi/agent`, see decision 2026-06-09). `git diff` review after `update` is the guard against skills.sh's known false-success/silent-skip bugs — see `.agents/atlas/artifacts/skills-sh-research.md`.

### Issue tracker

Issues are tracked as GitHub issues in `edheltzel/atlas-config` via the `gh` CLI. See `docs/agents/issue-tracker.md`.

### PM tools engine

The `pm-tools` skill ships a config-as-code template layer (`skills/pm-tools/templates/` — archetypes + overlays resolved deterministically) and a reconcile engine (`skills/pm-tools/Tools/Reconcile.ts` — plan-mode default; destructive ops gated behind `--prune --confirm`). Gates: `bun templates/smoke.ts`, `bunx tsc --noEmit`, `bun test` in `Tools/`. Recipes in `skills/pm-tools/references/`.

### Triage labels

Five canonical triage roles using default label strings (`needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, `wontfix`). See `docs/agents/triage-labels.md`.

### OMP config

OMP-native config is stowed from `omp/.omp/agent/` to `~/.omp/agent/`. Track authored config only; keep runtime DBs, sessions, blobs, memories, and secrets local. See `docs/agents/omp.md`.

### Domain docs

Single-context — one `CONTEXT.md` + `docs/adr/` at the repo root. See `docs/agents/domain.md`.

### GitHub identities

Three distinct GitHub principals operate on the `Gentle-Art-Technologies` org — `edheltzel` (human owner), `Atlas-Key` (teammate User, fine-grained PAT `ATLAS_KEY_PAT`), and `gat-atlas-bot` (App `4127929`, ephemeral installation tokens). An agent picks one per action type; act as the teammate with `GH_TOKEN=$ATLAS_KEY_PAT gh …`. See `docs/agents/github-identities.md`.
