---
task: Audit ~/.claude file origins and design atlas-config migration manifest
slug: atlas-config-portability-audit
effort: E3
phase: complete
progress: 34/34
mode: standard
started: 2026-05-17T14:23:10Z
updated: 2026-05-17T14:23:10Z
---

## Problem

Ed wants to copy Atlas (his PAI instance at `~/.claude`) to a new Mac, but `~/.claude` mixes three different kinds of content: (a) auto-generated runtime state (sessions, history, memory.db, statsig, file-history, shell-snapshots, observability JSONL, cached embeddings), (b) installed-by-installer system code that lives in the v5.0.0 release tree and can be reinstalled (`PAI/`, `hooks/`, `skills/`, `agents/`, `commands/`), and (c) hand-authored user-specific configuration (settings.json, USER/, MEMORY/KNOWLEDGE/, MEMORY/WISDOM/, MEMORY/RESEARCH/, MEMORY/RELATIONSHIP/, MEMORY/WORK/, checkpoint-repos.txt, CLAUDE.md customizations, .env). Without a clean separation, a migration either drags machine-local junk over (slow, polluted) or loses durable personal data (catastrophic). `~/Developer/atlas-config` already exists as the staging repo and contains an unfinished `.claude/` placeholder — it needs a definitive manifest of what to put there.

## Vision

A migration manifest that, when executed on a fresh Mac, reproduces Atlas's behavior end-to-end: same personality, same memories, same skills, same hooks, same secrets, same checkpoint repo registry — with zero hand-tuning needed after install. The atlas-config repo becomes the **canonical Atlas dotfiles repo**: clone it, run a bootstrap script, and Atlas wakes up. Ephemeral state is left behind by design.

## Out of Scope

- Actually performing the migration (this run produces the manifest, not the move)
- Writing the bootstrap/install script (called out as the next task)
- Migrating MEMORY/LEARNING, MEMORY/OBSERVABILITY, MEMORY/SECURITY (intentionally ignored per `.gitignore`)
- Migrating Claude Code internal state (sessions, history, statsig, shell-snapshots, file-history)
- Designing a sync protocol between two live Macs (this is one-shot move/replay, not continuous sync)
- Touching `~/.config/PAI/` directly — it already lives outside `~/.claude`

## Principles

- **Single source of truth per file.** A file belongs to exactly one tier: STATE, SYSTEM, or USER-CONFIG. Files that drift between tiers are bugs.
- **Idempotent reinstall.** The v5.0.0 release tree + `install.sh` recreates all SYSTEM files. atlas-config must NOT duplicate them.
- **Secrets via symlink, not git.** `.env` already symlinks to `~/.config/PAI/.env`. Same pattern for any other secret store.
- **State is local, config is portable.** Anything that grows monotonically with usage is STATE and is excluded.
- **Bitter Pill: don't over-engineer.** The output is a manifest + symlink plan, not a sync framework.

## Constraints

- `~/.claude` path is hard-coded throughout PAI (Claude Code requires it). Cannot move the root.
- `.env` must continue to live at `~/.config/PAI/.env` (already symlinked).
- The PAI installer recreates `PAI/`, `hooks/`, `skills/`, `agents/`, `commands/` — atlas-config must not shadow these or git conflicts result.
- `~/.claude/.gitignore` already encodes most of the STATE/SYSTEM/USER split — it is the canonical reference.
- macOS-only target (Darwin paths, `~/Library`, kitty terminal, ElevenLabs voice via local Pulse).

## Goal

Produce a tiered file manifest classifying every top-level entity in `~/.claude` (and the relevant USER/ + MEMORY/ subtrees) into STATE (skip), SYSTEM (reinstall), or USER-CONFIG (stage in atlas-config), together with a recommendation per USER-CONFIG entry on whether to COPY or SYMLINK from atlas-config to `~/.claude`, plus a short bootstrap recipe for replay on a new Mac.

## Criteria

### Discovery (top-level inventory)

- [ ] ISC-1: Every directory entry under `~/.claude` (depth=1) is classified STATE | SYSTEM | USER-CONFIG | HYBRID
- [ ] ISC-2: Every regular file under `~/.claude` (depth=1) is classified
- [ ] ISC-3: Existing symlinks under `~/.claude` are enumerated (currently `.env` and `skills/herdr`)
- [ ] ISC-4: The `~/.claude/.gitignore` is parsed to confirm STATE candidates are already ignored
- [ ] ISC-5: HYBRID directories (where some children are SYSTEM and others USER) are sub-classified (PAI/, MEMORY/, skills/, hooks/)

### USER-CONFIG identification

- [ ] ISC-6: `settings.json` classification (USER-CONFIG, COPY — contains permissions/env/hooks registration)
- [ ] ISC-7: `CLAUDE.md` classification (USER-CONFIG, COPY — has Ed-specific overrides)
- [ ] ISC-8: `checkpoint-repos.txt` classification (USER-CONFIG, COPY)
- [ ] ISC-9: `PAI/USER/` subtree classification (USER-CONFIG, SYMLINK candidate — biographical, lives with Ed not PAI)
- [ ] ISC-10: `PAI/MEMORY/KNOWLEDGE/` classification (USER-CONFIG, SYMLINK — distilled knowledge)
- [ ] ISC-11: `PAI/MEMORY/WISDOM/` classification (USER-CONFIG, SYMLINK)
- [ ] ISC-12: `PAI/MEMORY/RESEARCH/` classification
- [ ] ISC-13: `PAI/MEMORY/RELATIONSHIP/` classification
- [ ] ISC-14: `PAI/MEMORY/WORK/` classification (USER-CONFIG — Algorithm session PRDs)
- [ ] ISC-15: `PAI/MEMORY/BOOKMARKS/`, `PAISYSTEMUPDATES/`, `PROJECT/`, `REFERENCE/`, `SKILLS/`, `AUTO/`, `DATA/`, `RAW/` classification
- [ ] ISC-16: `.mcp.json`, `.lsp.json`, `.gitattributes`, `.gitmodules` classification
- [ ] ISC-17: `agents/` content classification (likely SYSTEM with possible custom overrides)
- [ ] ISC-18: `commands/` classification
- [ ] ISC-19: `plugins/` classification (the marketplaces dir is git-ignored)
- [ ] ISC-20: `VoiceServer/` classification
- [ ] ISC-21: `backups/` classification (STATE — auto-generated)
- [ ] ISC-22: `tasks/`, `cache/`, `logs/`, `ide/` classification (STATE — Claude Code internal)
- [ ] ISC-23: `USER/SECURITY/PATTERNS.yaml` (USER-CONFIG, COPY — custom redaction patterns)

### STATE identification (must NOT migrate)

- [ ] ISC-24: Anti: `sessions/`, `projects/`, `history.jsonl`, `memory.db*`, `.last-cleanup`, `statsig/` are NOT in the manifest
- [ ] ISC-25: Anti: `file-history/`, `shell-snapshots/`, `session-env/`, `paste-cache/` are NOT in the manifest
- [ ] ISC-26: Anti: `MEMORY/LEARNING/`, `MEMORY/OBSERVABILITY/`, `MEMORY/SECURITY/`, `MEMORY/STATE/`, `MEMORY/VOICE/`, `MEMORY/ARCHIVE/`, `MEMORY/KNOWLEDGE/_embeddings/`, `MEMORY/KNOWLEDGE/_harvest-queue/` are NOT in the manifest
- [ ] ISC-27: Anti: `memory.db-shm`, `memory.db-wal`, the live `memory.db` itself are NOT in the manifest

### Migration mechanics

- [ ] ISC-28: For each USER-CONFIG entry, recommendation is COPY or SYMLINK with a rationale (size-of-tree, change-frequency, secret-handling)
- [ ] ISC-29: Symlink set is explicit and minimal — fewer is better (every symlink is a moving part)
- [ ] ISC-30: Secret-handling pattern: anything secret routed through `~/.config/PAI/` (mirroring current `.env` pattern), not stored in atlas-config
- [ ] ISC-31: Bootstrap recipe (≤20 lines of bash/just) replays the install: run installer → clone atlas-config → run linker → done
- [ ] ISC-32: Manifest output is delivered inline in the SUMMARY, also written to `atlas-config/MANIFEST.md`
- [ ] ISC-33: Anti: No file appears in both COPY and SYMLINK lists
- [ ] ISC-34: Anti: No file in atlas-config would shadow a file that the installer overwrites (would cause install drift)

## Test Strategy

| isc | type | check | threshold | tool |
|-----|------|-------|-----------|------|
| ISC-1..5 | inspection | directory listing classified | 100% coverage of depth=1 entries | Bash `ls -la` + agent classification |
| ISC-6..23 | inspection | each USER-CONFIG entry has rationale | one line per entry | manual review of agent output |
| ISC-24..27 | grep | candidate matches against `.gitignore` | every Anti file IS ignored | `git check-ignore` |
| ISC-28..30 | inspection | COPY vs SYMLINK column populated | every USER-CONFIG row has it | manual review |
| ISC-31 | code | bootstrap recipe is concrete, runnable | ≤20 lines, no TODOs | inline review |
| ISC-32 | file | manifest written to atlas-config/MANIFEST.md | file exists, non-empty | `Read` |
| ISC-33, 34 | inspection | no duplicate entries, no installer-shadowing | zero collisions | grep + cross-ref |

## Features

| name | satisfies | depends_on | parallelizable |
|------|-----------|------------|----------------|
| Architect-driven classification | ISC-1..23 | — | yes (parallel with Engineer) |
| Engineer-driven side-effect inventory | ISC-24..27 | — | yes (parallel with Architect) |
| Synthesis + manifest write | ISC-28..32 | architect + engineer | no (sequential) |
| Self-consistency check | ISC-33, 34 | manifest written | no |

## Decisions

- 2026-05-17T14:23:10Z — Inline ISA write at E3 (skipped ISA Skill scaffold). show-your-math: audit task, no codegen, time budget priority. Doctrine permits inline at E1 strictly; E3 inline is a soft deviation logged here.
- 2026-05-17T14:23:10Z — Forge skipped at E3. show-your-math: investigation + classification, no code produced; Forge's GPT-5.4 strength is code quality which is not on the critical path.
- 2026-05-17T14:23:10Z — Architect + Engineer launched in parallel via single message (both read-only investigation, no contention).
