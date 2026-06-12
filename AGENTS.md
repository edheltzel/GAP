# atlas-config — Agent Instructions

This repo is a minimal, stow-managed config for Ed's agents. On 2026-05-18 the `Codex/` package was populated with Ed's PAI v5.0.0 USER-CONFIG (identity, TELOS, durable MEMORY, SYSTEM-OVERRIDE hooks/Pulse/TOOLS); prior to that the previous Atlas/PAI build was soft-reset on 2026-05-01.

## PAI v5.0.0 migration

This repo tracks Ed's PAI v5.0.0 configuration (identity, TELOS, durable memory, SYSTEM-OVERRIDE hooks). See `.agents/plans/2026-05-18-pai-to-atlas-config-migration-plan.md` for the canonical migration spec (supersedes the archived MANIFEST at `.agents/atlas/plans/archive/2026-05-17-pai-migration-manifest.md`).

## Scope

**atlas-config is single-machine-canonical.** The `merge=ours` rule on `Codex/.Codex/settings.json` (see `.gitattributes`) means changes to that file do not propagate across machines via `git pull` — local edits always win. Multi-machine sync requires explicit `git checkout origin/master -- Codex/.Codex/settings.json` on the receiving Mac, then a re-personalization pass if needed. This is intentional: `settings.json` carries per-session mutating fields (`counts`, `feedbackSurveyState.lastShownTime`) that would otherwise churn the repo.

The Recall MCP database (`~/.agents/Recall/recall.db`, shared by all agents) and the Recall MEMORY/ vault (`~/.agents/Recall/MEMORY/`) also live **outside** atlas-config — they are out-of-band like `~/.env`. Cold-archive these to an external location on a schedule (1Password vault, Time Machine, external drive); they are the only USER content not in git.

## Repo layout

| Package | Source | Stowed to | Status |
|---|---|---|---|
| `Codex` | `Codex/.Codex/` | `~/.Codex/` | Populated with PAI v5.0.0 USER-CONFIG (2026-05-18) |
| `pi` | `pi/.pi/agent/` | `~/.pi/agent/` | Curated Pi config only; stowed with `--no-folding` so runtime state stays local |
| `omp` | `omp/.omp/agent/` | `~/.omp/agent/` | Curated OMP-native config only; runtime DBs, sessions, blobs, memories, and secrets stay local |

Project-local artifacts live under `.agents/` (`plans/`, `atlas/handoffs/`, `worktrees/`). `worktrees/` is gitignored.

Before changing Pi's global package set or implementing `/atlas-flow`, read `.agents/plans/2026-05-16-atlas-flow-research-digest.md` for the minimal-global baseline, flow manifest schema direction, safety rules, and issue roadmap.

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

Multi-package stow with the repo root as `STOW_DIR`. See `justfile` for commands. Pi is stowed with `--no-folding`: `~/.pi/agent` must remain a real directory, with only curated config files symlinked into it. The per-package `.stow-local-ignore` files **replace** stow's built-in defaults — when adding a new one, copy from `Codex/.stow-local-ignore` and re-include `\.git`, `\.gitignore`, etc.

## When adding files

- Adding to `Codex/.Codex/`, `omp/.omp/agent/`, or curated files under `pi/.pi/agent/` → run `just up` to restow.
- Anything that should not be symlinked into the runtime location → add a pattern to that package's `.stow-local-ignore`.
- Pi runtime/package state (`auth.json`, `sessions/`, `cache/`, `mcp*.json`, `run-history.jsonl`, `memory/`, `npm/`, `git/`) stays in the real `~/.pi/agent`, not in atlas-config.
- OMP runtime state (`agent.db*`, `history.db*`, `models.db*`, `sessions/`, `blobs/`, `terminal-sessions/`, `memories/`, `cache/`, `secrets.yml`, `.env`, auth broker/gateway tokens) stays in the real `~/.omp`, not in atlas-config. See `docs/agents/omp.md`.
- Secrets (`auth.json`, `*.env`, keys) → already covered by `.gitignore`. Do not weaken those rules.

## Notable conventions

- `.agents/plans/` for active plan/design docs, `.agents/atlas/plans/archive/` for completed.
- `.agents/atlas/handoffs/` for live handoffs, archived to `.agents/atlas/handoffs/archive/` when consumed.
- File naming: `YYYY-MM-DD-description.md`.
- Use git worktrees (`EnterWorktree` / `ExitWorktree`) for implementation work — they land in `.agents/worktrees/`.

## Agent skills

### Skills store (skills.sh)

The [skills.sh](https://skills.sh) global store is repo-backed: the real files live in `skills/` at the repo root, and `~/.agents/skills` is a plain symlink to it (created by `just skills-link`, **not** stow). skills.sh keeps working unchanged — `npx skills add owner/repo` writes through the symlink straight into the repo, and its per-agent symlinks (`~/.claude/skills/<name> → ../../.agents/skills/<name>`, etc.) resolve through the chain.

Workflow: `npx skills add`/`update` as usual → `just skills-sync` to review and commit the diff. The sync target refuses to run if `~/.agents/skills` is no longer a symlink (i.e. skills.sh recreated it as a real dir — relink and reconcile manually). `node_modules/` and `.DS_Store` inside the store are gitignored; do not track Pi CE runtime skills here (they stay under `~/.pi/agent`, see decision 2026-06-09). `git diff` review after `update` is the guard against skills.sh's known false-success/silent-skip bugs — see `.agents/atlas/artifacts/skills-sh-research.md`.

### Issue tracker

Issues are tracked as GitHub issues in `edheltzel/atlas-config` via the `gh` CLI. See `docs/agents/issue-tracker.md`.

### Triage labels

Five canonical triage roles using default label strings (`needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, `wontfix`). See `docs/agents/triage-labels.md`.

### OMP config

OMP-native config is stowed from `omp/.omp/agent/` to `~/.omp/agent/`. Track authored config only; keep runtime DBs, sessions, blobs, memories, and secrets local. See `docs/agents/omp.md`.

### Domain docs

Single-context — one `CONTEXT.md` + `docs/adr/` at the repo root. See `docs/agents/domain.md`.
