# Atlas (PAI) — Migration Manifest

> What `~/.claude` actually contains, what's portable, and how to replay it on a new Mac.
> Generated 2026-05-17 by Atlas. Source audit: `~/.claude/PAI/MEMORY/WORK/atlas-config-portability-audit/ISA.md`.

## The 3-tier model

Every file under `~/.claude` belongs to exactly one tier:

| Tier | Source | Migration | Examples |
|------|--------|-----------|----------|
| **STATE** | Auto-written by Claude Code CLI or PAI hooks; accumulates with use; regen-free | **SKIP** | `sessions/`, `projects/`, `history.jsonl`, `memory.db*`, `MEMORY/OBSERVABILITY/*.jsonl`, `MEMORY/STATE/*.json`, `MEMORY/LEARNING/`, `MEMORY/VOICE/`, `MEMORY/SECURITY/`, `file-history/`, `shell-snapshots/`, `backups/`, `cache/`, `logs/`, `ide/`, `paste-cache/`, `session-env/`, `tasks/`, `.last-cleanup`, `plugins/cache/`, `plugins/marketplaces/` |
| **SYSTEM** | Installed by PAI installer; identical to release tree; reinstall recreates | **REINSTALL** | `PAI/ALGORITHM/`, `PAI/DOCUMENTATION/`, `PAI/TOOLS/`, `PAI/PULSE/`, `PAI/TEMPLATES/`, `PAI/PAI-Install/`, `PAI/bin/`, `PAI/PAI_SYSTEM_PROMPT.md`, `hooks/`, `skills/` (except `herdr` symlink), `agents/`, `commands/`, `install.sh`, `ISA.md`, `LICENSE`, `README.md`, `.gitignore`, `.gitattributes`, `.gitmodules`, `.lsp.json`, `.mcp.json` |
| **USER-CONFIG** | Hand-authored OR distilled-from-state; irreplaceable | **COPY** or **SYMLINK** via atlas-config | See manifest below |
| **SECRETS** | API keys, OAuth tokens, `.env` | **NEVER in atlas-config** — live in `~/.config/PAI/` | `.env`, `PAI/USER/CREDENTIALS/` |

The PAI installer is preservation-first (`PAI/PAI-Install/engine/actions.ts` uses `skipIfExists` on `PAI/USER/**` and `PAI/MEMORY/**`), so reinstalling over an existing tree does not clobber the USER-CONFIG tier.

## Layout: what goes where

```
~/Developer/atlas-config/              # this repo (private dotfiles)
├── justfile                           # already exists — stow link/unlink/push
├── MANIFEST.md                        # this file
├── README.md
│
├── claudecode/                        # stow package → links into ~/
│   └── .claude/
│       ├── CLAUDE.md                  # COPY  — Ed-personalized global instructions
│       ├── settings.json              # COPY  — Atlas voices, paths, hook registry (1371 lines)
│       ├── checkpoint-repos.txt       # COPY  — CheckpointPerISC allowlist
│       │
│       ├── plugins/
│       │   ├── installed_plugins.json # COPY  — list of installed plugins (re-fetched on first start)
│       │   └── known_marketplaces.json# COPY  — marketplace registry
│       │
│       ├── MEMORY/
│       │   └── pi-sessions/           # COPY  (only if pi runs on new Mac)
│       │
│       └── PAI/
│           ├── USER/                  # SYMLINK target — biographical heart of Atlas
│           │   ├── ABOUTME.md, BASICINFO.md, RESUME.md, CONTACTS.md, OPINIONS.md, OUR_STORY.md
│           │   ├── PRINCIPAL_IDENTITY.md, DA_IDENTITY.md     # CRITICAL — defines "Atlas"
│           │   ├── AI_WRITING_PATTERNS.md, WRITINGSTYLE.md, RHETORICALSTYLE.md, CORECONTENT.md
│           │   ├── DEFINITIONS.md, PRONUNCIATIONS.md, FEED.md
│           │   ├── ALGOPREFS.md, ARCHITECTURE.md, TECHSTACKPREFERENCES.md
│           │   ├── BUSINESS/, WORK/, DA/, SHARED/Spinner/
│           │   ├── ACTIONS/, ARTHUR/, Daemon/, FLOWS/, PIPELINES/, TERMINAL/, SKILLCUSTOMIZATIONS/
│           │   ├── Config/PAI_CONFIG.yaml
│           │   ├── SECURITY/PATTERNS.yaml                    # CRITICAL — SecurityPipeline fails without
│           │   ├── TELOS/                                    # MISSION/GOALS/BELIEFS/WISDOM/CHALLENGES/BOOKS/STRATEGIES/...
│           │   └── Voice → ../../../../../atlas-voicesystem/claudecode/.claude/PAI/USER/Voice   # nested symlink (relative)
│           │
│           └── MEMORY/                # SYMLINK target — distilled durable memory
│               ├── KNOWLEDGE/         # distilled wisdom (exclude _embeddings/, _harvest-queue/ via .gitignore)
│               ├── WISDOM/
│               ├── RESEARCH/
│               ├── RELATIONSHIP/      # session-by-session relationship log
│               ├── WORK/              # Algorithm session PRDs (".gitignore says must survive across machines")
│               ├── BOOKMARKS/, PAISYSTEMUPDATES/, PROJECT/, REFERENCE/, SKILLS/, AUTO/, DATA/, RAW/, SCRATCHPAD/, VERIFICATION/
│               └── README.md
│
├── .agents/
│   └── skills/
│       └── herdr/                     # SYMLINK source — already in use (the precedent)
│
├── codex/.codex/                      # stow package for ~/.codex (separate tool)
└── pi/.pi/                            # stow package for ~/.pi    (separate tool)

~/.config/PAI/                         # SECRETS — outside git, outside atlas-config
├── .env                               # → symlinked into ~/.claude/.env
└── CREDENTIALS/                       # → symlink target for ~/.claude/PAI/USER/CREDENTIALS/
```

## Symlink minimal set (7 from atlas-config + 1 from ~/.config/PAI)

Every symlink is a moving part — keep this set minimal.

| # | atlas-config source | → | ~/.claude target | Why this needs to be a SYMLINK (not COPY) |
|---|---|---|---|---|
| 1 | `claudecode/.claude/PAI/USER/` | → | `~/.claude/PAI/USER/` | TELOS, DA_IDENTITY, PRINCIPAL_IDENTITY get edited weekly; symlink = every edit flows back to atlas-config without a sync step |
| 2 | `claudecode/.claude/PAI/MEMORY/KNOWLEDGE/` | → | `~/.claude/PAI/MEMORY/KNOWLEDGE/` | KnowledgeHarvester writes here continuously; symlink accrues to atlas-config |
| 3 | `claudecode/.claude/PAI/MEMORY/WISDOM/` | → | `~/.claude/PAI/MEMORY/WISDOM/` | Distilled wisdom; same accrual logic |
| 4 | `claudecode/.claude/PAI/MEMORY/RESEARCH/` | → | `~/.claude/PAI/MEMORY/RESEARCH/` | Persistent research vault |
| 5 | `claudecode/.claude/PAI/MEMORY/RELATIONSHIP/` | → | `~/.claude/PAI/MEMORY/RELATIONSHIP/` | RelationshipMemory.hook.ts writes per session-start + per significant event |
| 6 | `claudecode/.claude/PAI/MEMORY/WORK/` | → | `~/.claude/PAI/MEMORY/WORK/` | Algorithm session ISAs — `.gitignore` line: "tracked — durable Algorithm record, must survive across machines" |
| 7 | `.agents/skills/herdr/` | → | `~/.claude/skills/herdr` | Already in use; existing precedent |
| 8 | `~/.config/PAI/.env` | → | `~/.claude/.env` | Existing precedent; secret stays out of git |

Also keep the existing `PAI/USER/Voice → ~/Developer/atlas-voicesystem/...` symlink — `atlas-voicesystem` is its own repo, bootstrapped separately.

## COPY list (the small, low-churn, installer-rewritable files)

| File | Why COPY (not SYMLINK) |
|------|------------------------|
| `~/.claude/CLAUDE.md` | Installer's `/interview` wizard template-renders this; a live symlink would be silently overwritten |
| `~/.claude/settings.json` | Same — installer's `config-gen.ts` rewrites from wizard answers. Also embeds machine-specific absolute paths that need editing per-machine |
| `~/.claude/checkpoint-repos.txt` | Tiny opt-in allowlist, edit-once per machine |
| `~/.claude/plugins/installed_plugins.json` | Tiny; re-fetched on first start regardless |
| `~/.claude/plugins/known_marketplaces.json` | Tiny |
| `~/.claude/MEMORY/pi-sessions/*.md` | Only if `pi` runs on the new Mac |

## SKIP list — the auto-write side-effects (Engineer-verified, with source citations)

Confirmed by reading hook source:

**Claude Code CLI internal** (`sessions/`, `projects/`, `tasks/`, `history.jsonl`, `file-history/`, `shell-snapshots/`, `session-env/`, `paste-cache/`, `ide/`, `logs/memory-usage.jsonl`, `backups/`, `cache/`, `memory.db*`, `.last-cleanup`, `statsig/`, `plugins/cache/`, `plugins/marketplaces/`) — written by the CLI itself; regenerate on use.

**PAI hook side-effects:**
- `MEMORY/STATE/*` (last-prompt.json, last-response.txt, work.json, session-names.json, usage-cache.json, model-cache.txt, weather-cache.json, location-cache.json, integrity-state.json, instruction-hashes.json, instruction-integrity.jsonl, kitty-sessions/*, progress/*) — written by `hooks/{RepeatDetection,LastResponseCache,PromptProcessing,handlers/UpdateCounts,handlers/SystemIntegrity,InstructionsLoadedHandler,KittyEnvPersist}.hook.ts`. All regenerate on first session.
- `MEMORY/OBSERVABILITY/*.jsonl` (tool-activity, tool-failures, prompt-processing, subagent-events, agent-starts, teammate-events, config-changes, anthropic-cost, session-costs, task-governance) — written by `hooks/{ToolActivityTracker,ToolFailureTracker,PromptProcessing,AgentInvocation,TeammateIdle,ConfigAudit,TaskGovernance}.hook.ts` and `PAI/PULSE/Performance/module.ts`. Raw event logs.
- `MEMORY/VOICE/voice-events.jsonl` — `hooks/handlers/VoiceNotification.ts:77`. Raw.
- `MEMORY/SECURITY/<YYYY>/<MM>/security-alert-*.jsonl` — `hooks/security/logger.ts:45`. Raw.
- `MEMORY/HOOKS/hook-debug.jsonl` — debug trace.
- `MEMORY/WORK/<slug>/.checkpoint-state.json` — `hooks/CheckpointPerISC.hook.ts:80`. Idempotency sidecar. (The ISA.md inside the same dir IS kept via SYMLINK above.)
- `MEMORY/WORK/<slug>/voice.jsonl` — per-session voice log.
- `MEMORY/LEARNING/**` — per `.gitignore`: "2.3 GB — 12K JSONL algorithm reflections (raw input)". DISCARD. Distillate flows into KNOWLEDGE.

**Tool-written distillate that IS kept** (already covered by symlinks above):
- `MEMORY/KNOWLEDGE/*.md` + `_index.md` MOCs — `PAI/TOOLS/KnowledgeHarvester.ts:441,549,619`. Captured by symlink #2.
- `MEMORY/WORK/<slug>/ISA.md` + `META.yaml` — Algorithm + `hooks/SessionCleanup.hook.ts:110,119`. Captured by symlink #6.
- `MEMORY/RELATIONSHIP/<YYYY-MM>/<YYYY-MM-DD>.md` — `hooks/RelationshipMemory.hook.ts:230,269`. Captured by symlink #5.
- `PAI/USER/TELOS/PRINCIPAL_TELOS.md` — `PAI/TOOLS/GenerateTelosSummary.ts` via `hooks/TelosSummarySync.hook.ts`. Captured by symlink #1.

**Installer regenerates** (so the live file doesn't migrate):
- `PAI/DOCUMENTATION/ARCHITECTURE_SUMMARY.md` — `PAI/TOOLS/ArchitectureSummaryGenerator.ts:324`. Hook-regenerated on first session.

## Migration blockers (must be fixed by bootstrap script, not by carrying files)

1. **Absolute paths in `settings.json`** (lines 4, 8, 99) — `PAI_DIR`, `PAI_CONFIG_DIR`, voice-system hook path. Hardcodes `/Users/ed/`. Bootstrap MUST rewrite these for the target machine (or use `${HOME}`).
2. **`atlas-voicesystem` clone path** — `settings.json:99` references `$HOME/Developer/atlas-voicesystem/...`. Must clone that repo to that exact path BEFORE first Claude Code session.
3. **`memory.db`** — SQLite WAL mode. If you do migrate it, copy ONLY when Claude Code is not running, and discard `.db-shm` / `.db-wal`. Recommended: don't migrate; let it rebuild.
4. **`MEMORY/STATE/integrity-state.json` + `instruction-hashes.json`** — drift-detection baselines. Carrying them across machines makes every system doc flag as drifted. Always DISCARD.
5. **`checkpoint-repos.txt`** — lists absolute repo paths. Re-verify each path exists on the new Mac.
6. **`herdr` symlink target** `~/.agents/skills/herdr` — install herdr first, then the symlink resolves.
7. **CREDENTIALS slip-through** — if `PAI/USER/` is symlinked to atlas-config and a tool writes `CREDENTIALS/`, the secrets land in the staging tree. Mitigation: atlas-config's own `.gitignore` must list `claudecode/.claude/PAI/USER/CREDENTIALS/`, AND replace `CREDENTIALS/` with a symlink to `~/.config/PAI/CREDENTIALS/`.
8. **Embeddings slip-through** — `PAI/MEMORY/KNOWLEDGE/_embeddings/` and `_harvest-queue/` are excluded by `~/.claude/.gitignore` (absolute pattern). Under atlas-config's root these patterns don't match. Add `claudecode/.claude/PAI/MEMORY/KNOWLEDGE/_embeddings/` and `_harvest-queue/` to atlas-config's `.gitignore`.

## Bootstrap recipe (new Mac, < 20 lines)

```bash
# 0. Prereqs: install brew, bun, gh, kitty, mkcert
xcode-select --install
bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
brew install bun gh kitty mkcert

# 1. Provision secrets out-of-band (1Password / scp from old Mac)
mkdir -p ~/.config/PAI ~/.config/PAI/CREDENTIALS
# transfer .env and CREDENTIALS/* into ~/.config/PAI/  (NOT via git)

# 2. Clone the three repos in dependency order
git clone <atlas-voicesystem> ~/Developer/atlas-voicesystem
git clone <atlas-config>      ~/Developer/atlas-config
git clone <pai-release>       ~/Developer/AI/PersonalAI/Releases/v5.0.0

# 3. Install PAI (preservation-first; safe to re-run)
bash ~/Developer/AI/PersonalAI/Releases/v5.0.0/.claude/install.sh

# 4. Layer atlas-config on top (justfile uses stow)
cd ~/Developer/atlas-config && just link

# 5. Recreate the one secret symlink the installer doesn't manage
ln -sfn ~/.config/PAI/.env ~/.claude/.env

# 6. Sanity check
ls -la ~/.claude/PAI/USER/DA_IDENTITY.md  ~/.claude/PAI/USER/TELOS/MISSION.md
```

After step 6, launching `claude` should boot Atlas with name, voice, memory, and skills intact. The first session regenerates STATE files (memory.db, instruction-hashes, observability logs).

## Self-consistency checks

- ✅ Every USER-CONFIG entry has either COPY or SYMLINK, never both.
- ✅ No file in the COPY/SYMLINK lists is also installer-managed (would shadow reinstall).
- ✅ Secrets path (`~/.config/PAI/`) is NOT inside atlas-config.
- ✅ All `.gitignore`d paths from `~/.claude/.gitignore` are in the SKIP list.
- ✅ Three existing symlinks documented: `.env`, `skills/herdr`, `PAI/USER/Voice`.

## What this manifest does NOT cover (out of scope)

- Writing the actual bootstrap script as a runnable file (next task).
- Migration sync between two live Macs (this is one-shot replay).
- Recovering `MEMORY/LEARNING/*.jsonl` raw history (intentionally discarded).
- Hooks for Pulse dashboard customization (separate concern).
