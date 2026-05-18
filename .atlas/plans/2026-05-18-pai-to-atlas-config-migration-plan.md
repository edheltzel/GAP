# PAI v5.0.0 → atlas-config Migration Plan

> **Status:** PLAN ONLY — not for execution in this session.
> **Author:** Atlas (E4 Algorithm run, 2026-05-18; amended 2026-05-18 16:00 with Recall-vault findings).
> **Supersedes:** `MANIFEST.md` (2026-05-17, untracked). MANIFEST stays for now as the prior audit; this plan is the canonical execution spec.
> **Source ISA:** `~/.claude/PAI/MEMORY/WORK/20260518-183332_pai-to-atlas-config-migration-plan/ISA.md`.
>
> **2026-05-18 16:00 AMENDMENT v1.1 — Critical correction.** The original v1 of this plan treated `memory.db` and the root-level `MEMORY/` dir as STATE-tier and dismissed them. Audit of the pre-PAI-install backup at `~/.claude.backup-20260514-173801/` (Time-Machine-equivalent created by `install.sh` on 2026-05-14) revealed the **Recall MCP memory subsystem** — 37 MB SQLite DB + 1.95 MB of authored DISTILLED/HOT_RECALL/DECISIONS/REJECTIONS/SESSION_INDEX/ERROR_PATTERNS files + 34 missing Pi-session digests + 17 hand-curated skills + 4 Recall internal DB snapshots. All of it was wiped by the PAI v5 install and the original v1 plan did NOT specify restoration. **§4.5, §7 step 1b, §10 #9-#11 added below.** The original §11 anti-rule ("Does NOT carry `memory.db`") is preserved — the DB still doesn't enter atlas-config; it lives in `~/.agents/atlas/Recall/` like `.env` does.
>
> **2026-05-18 16:20 AMENDMENT v1.2 — Path + scope clarifications.**
> 1. **External-data root moved:** `~/.config/PAI/` → `~/.agents/atlas/`. `~/.agents/` already exists with `skills/herdr/` and `diagrams/`; consolidating Atlas-owned external state under one root (`~/.agents/atlas/`) is cleaner than splitting across `~/.config/PAI/` and `~/.agents/`. Net moves: `~/.agents/atlas/.env`, `~/.agents/atlas/CREDENTIALS/`, `~/.agents/atlas/Recall/{memory.db,backups/}`. The two-hop chain `~/.claude/.env → ~/.agents/atlas/.env → ~/.env` replaces `~/.claude/.env → ~/.config/PAI/.env → ~/.env`. `settings.json:8` updated: `PAI_CONFIG_DIR: ${HOME}/.agents/atlas`. All other §4.5, §5.1, §7, §8, §10 references updated.
> 2. **Agents & skills coverage clarified** in new §4.6: `~/.claude/agents/` (18 PAI v5 agents) and `~/.claude/skills/` (46 PAI v5 skills) are byte-identical to release — **SYSTEM-tier, NOT migrated.** The 18 Recall-era backup skills (§7b.3) are USER-CONFIG and require triage. No custom Ed-authored Claude Code agents exist on either live tree or backup. Pi and Codex agents live in their own stow packages, separate concern.
>
> **2026-05-18 16:30 AMENDMENT v1.3 — Two-backup landscape resolved.** A *second* backup was found at `~/.claude.backup-2026-05-14T21-38-32-285Z/` (272 MB, ISO 8601 timestamp with milliseconds — characteristic of programmatic creation by `install.sh`). It is the **clean post-PAI-install snapshot**, distinct from `~/.claude.backup-20260514-173801/` (1.1 GB, underscore-time format) which is the **pre-PAI Atlas-Recall state**. Diff between the ISO backup and live `~/.claude/` produced three high-value findings that refine the plan — see new §1.7.
>
> **2026-05-18 16:45 AMENDMENT v1.4 — `memory.db` content verified, restoration hardened.** Direct SQLite inspection of `~/.claude.backup-20260514-173801/memory.db` confirms the DB contains **17,336 messages, 1,898 sessions, 80 decisions, 56 learnings, 39 LoA entries, 19 TELOS rows, 15 breadcrumbs, 240 extraction sessions, 1,149 extraction-tracker entries** — all of which are **zero** in the fresh post-install live DB. §4.5a now enumerates the recoverable content table-by-table. §7 step 1a expanded with pre-flight checks, `PRAGMA integrity_check`, and per-table row-count verification. §8 verification probes 17a–17h replace the original probe #17 (which incorrectly referenced a non-existent `extractions` table). The OLD backup MUST NOT be deleted until all 17a–17h probes pass.
>
> **2026-05-18 17:00 AMENDMENT v1.5 — Red-team critical fixes (BLOCKING).** A 12-agent RedTeam ParallelAnalysis identified 7 remediations. **R1, R2 must execute as pre-flight; R3 changes the execution path; R4, R5 are safety net.** Executor MUST apply v1.5 before any §9 step that mutates state.
>
> **R1 (CRITICAL — data corruption gate):** §7 step 1a `pgrep -f mem-mcp && abort` only *detects* a running Recall MCP — it does not *stop* it. The plan as written would let a Recall hook spawn a fresh `mem-mcp` between the pgrep check and the cp. Required additional steps BEFORE the cp:
> ```bash
> # Stop any Recall MCP processes (multiple names possible)
> launchctl list 2>/dev/null | grep -E "recall|mem-mcp" | awk '{print $3}' | \
>   xargs -I {} launchctl bootout gui/$(id -u) {} 2>/dev/null || true
> pkill -TERM -f "mem-mcp" 2>/dev/null; sleep 2
> pkill -KILL -f "mem-mcp" 2>/dev/null; sleep 1
> # Verify no open file descriptors against the LIVE DB before touching backup
> if lsof ~/.claude/memory.db 2>/dev/null | grep -v COMMAND; then
>   echo "FATAL: open writer on memory.db — abort restore"; exit 1
> fi
> # Also disable PAI hooks that might respawn Recall
> # (cheapest mechanism: rename the SessionExtract/BatchExtract hooks for the duration)
> ```
>
> **R2 (CRITICAL — SQLite WAL atomicity):** Different APFS volumes break SQLite's WAL/main-DB fsync atomicity. Required pre-flight before `cp`:
> ```bash
> vol_claude=$(stat -f%d ~/.claude)
> vol_external=$(stat -f%d ~/.agents/atlas/Recall 2>/dev/null || stat -f%d ~/.agents)
> if [ "$vol_claude" != "$vol_external" ]; then
>   echo "FATAL: ~/.claude and ~/.agents/atlas/Recall on different volumes ($vol_claude vs $vol_external). SQLite WAL atomicity not guaranteed across APFS volumes. Relocate or abort."
>   exit 1
> fi
> ```
>
> **R3 (STRUCTURAL — write-amplification fix):** The original plan symlinked Recall MEMORY/ (DISTILLED.md, HOT_RECALL.md, etc.) into atlas-config. RedTeam convergent finding (5/12 agents): this creates `.git/index.lock` contention with PAI hooks, `git status` permanent-dirty, and conversational-sediment commits. **Revised:** restore Recall MEMORY/ files to `~/.agents/atlas/Recall/MEMORY/` (NOT into atlas-config) alongside the DB. Add a `just snapshot-recall` recipe that rsyncs into a tracked `claudecode/.claude/MEMORY-snapshots/<date>/` periodically (hourly cron or on-demand). Live tree gets `~/.claude/MEMORY/ → ~/.agents/atlas/Recall/MEMORY/` symlink. Hooks write to the runtime location; snapshots feed git history without contention. Updates required:
> - §4.5b table: change all 8 "RESTORE → SYMLINK" verdicts to "RESTORE → external + snapshot pipeline"
> - §5.1 `.gitignore`: add `claudecode/.claude/MEMORY/` (since it's now a symlink target, not tracked)
> - §7 step 1b: copy to `~/.agents/atlas/Recall/MEMORY/`, NOT into atlas-config
> - §9 step 9: add 3 extra symlinks (`~/.claude/MEMORY → ~/.agents/atlas/Recall/MEMORY` and inverses)
> - New §10 #15: "implement `just snapshot-recall` rsync cron"
>
> **R4 (HIGH — absence-probe added to §8):** New probe #24:
> ```bash
> # List every regular file in OLD backup; cross-check against plan references
> find ~/.claude.backup-20260514-173801 -type f ! -path '*/.git/*' ! -path '*/projects/*' \
>   ! -path '*/sessions/*' ! -path '*/file-history/*' ! -path '*/cache/*' ! -path '*/plugins/*' \
>   ! -path '*/shell-snapshots/*' ! -path '*/logs/*' ! -path '*/backups/*' \
>   | while read f; do
>       basename=$(basename "$f")
>       grep -qF "$basename" .atlas/plans/2026-05-18-pai-to-atlas-config-migration-plan.md \
>         || echo "UNREFERENCED: $f"
>     done
> # If output is non-empty: investigate each line BEFORE proceeding to step 11 (commit)
> ```
>
> **R5 (HIGH — multi-machine drift acknowledgment):** `.gitattributes merge=ours` on `settings.json` makes `git pull` silently discard incoming versions. Add to atlas-config/CLAUDE.md (top-level) under a new "Scope" section: **"atlas-config is single-machine-canonical. The `merge=ours` rule on `settings.json` means changes do not propagate across machines via `git pull`. Multi-machine sync requires explicit `git checkout origin/master -- claudecode/.claude/settings.json` on the receiving Mac."** This documents the constraint instead of trying to engineer around it.
>
> **R6 (MEDIUM — soak window for OLD backup deletion):** §10 #12 originally said "delete OLD backup after probes pass." Revised: "delete OLD backup after 4-week Recall write soak passes AND probes 17a–17h pass AND R4 absence-probe is clean." Recall row counts grow continuously; a 4-week soak ensures the restored DB is fully reattached before the original is destroyed.
>
> **R7 (MEDIUM — upstream PAI issue):** File a tracking issue against PAI v5 requesting native USER-preservation in `actions.ts` (currently relies on `copyMissing` + backup-then-migrate fallback). atlas-config is a transitional bridge; the long-term fix is upstream. Tracked as §10 #16.

---

## TL;DR

`~/.claude/` now runs the full PAI v5.0.0 Life-OS install (Algorithm v6.3.0, 45 skills, 37 hooks, ISA system, accumulating WORK/KNOWLEDGE/RELATIONSHIP memory). `atlas-config` was soft-reset 2026-05-01 and currently stows a small Atlas-Recall config tree under `claudecode/.claude/` — but does **not** yet track any of the PAI v5 USER/MEMORY surface. One wipe of `~/.claude/` (or a fresh-mac bootstrap) loses every piece of hand-authored identity, TELOS, knowledge, relationship memory, and Algorithm session history.

This plan: (a) keeps the 25 Atlas-Recall files already tracked under `claudecode/.claude/`, (b) layers symlinks for `PAI/USER/`, the durable `PAI/MEMORY/` subdirs, and the four hand-modified hooks, (c) tracks `~/.claude/CLAUDE.md` and `~/.claude/settings.json` as COPY with a 2-line portability patch, (d) closes three privacy gaps (`CREDENTIALS/`, `_embeddings/`, `_harvest-queue/`), (e) corrects three factual errors in MANIFEST.md (herdr precedent, settings.json line 99, installer `skipIfExists`), (f) ships a corrected bootstrap recipe.

**Output of execution will be the populated `claudecode/.claude/PAI/` subtree + amended `.gitignore` + amended `claudecode/.stow-local-ignore` + one commit.**

---

## 1. State of the world (2026-05-18)

### 1.1 `~/.claude/` — live PAI v5.0.0 install

| Top-level entry | Tier | Notes |
|-----------------|------|-------|
| `.env` (symlink → `~/.agents/atlas/.env` → `~/.env`) | SECRETS | Two-hop chain; real file mode 0600, 981 B. |
| `.gitattributes`, `.gitignore`, `.gitmodules`, `.lsp.json`, `.mcp.json` | SYSTEM | Byte-identical to v5 release. |
| `.last-cleanup` | STATE | Hook-written. |
| `CLAUDE.md` | USER-CONFIG | 9.6 KB, Ed-personalized; release ships generic template. |
| `ISA.md`, `LICENSE`, `README.md`, `install.sh` | SYSTEM | Identical to release. |
| `agents/`, `commands/`, `skills/` | SYSTEM | Byte-identical to release. `skills/herdr` is the one symlink. |
| `hooks/` | SYSTEM* | 4 files drift (port `8888 → 31337`, see §1.4). |
| `settings.json` | USER-CONFIG | 1,371 lines, 53,229 B, two `/Users/ed/` paths (lines 4, 8). |
| `checkpoint-repos.txt` | USER-CONFIG | 707 B; only `~/.claude` enabled. |
| `plugins/installed_plugins.json`, `plugins/known_marketplaces.json` | USER-CONFIG (volatile) | Carry absolute paths + timestamps — re-install via CLI rather than track. |
| `PAI/USER/` | USER-CONFIG | Identity, TELOS, FEED, OPINIONS, CORECONTENT, WRITINGSTYLE, RHETORICALSTYLE, etc. + `Voice/` (symlink → atlas-voicesystem). |
| `PAI/MEMORY/` | mixed | KNOWLEDGE / WISDOM / RESEARCH / RELATIONSHIP / WORK / BOOKMARKS / PROJECT / REFERENCE / SKILLS / PAISYSTEMUPDATES → USER-CONFIG (durable distillate). LEARNING / OBSERVABILITY / STATE / VOICE / SECURITY / HOOKS / AUTO / RAW / SCRATCHPAD / VERIFICATION → STATE (skip). |
| `PAI/PULSE/`, `PAI/TOOLS/`, `PAI/PAI-Install/` | SYSTEM with drift | Several files hand-modified post-install — see §1.4. |
| `PAI/ALGORITHM/`, `PAI/DOCUMENTATION/`, `PAI/TEMPLATES/`, `PAI/bin/`, `PAI/PAI_SYSTEM_PROMPT.md` | SYSTEM | Identical to release. |
| `backups/`, `cache/`, `file-history/`, `history.jsonl`, `ide/`, `logs/`, `memory.db*`, `projects/`, `session-env/`, `sessions/`, `shell-snapshots/`, `tasks/`, `test-results/`, `VoiceServer/` | STATE | Skip. |
| `MEMORY/` (root, NOT `PAI/MEMORY/`) | STATE | v4 leftover dir — `pi-sessions/` lives here, that's the only USER content. |

### 1.2 `atlas-config/` — current tracked tree (`git ls-files`)

Six top-level zones tracked today:

- **Root** — `.gitattributes`, `.gitignore`, `CHANGELOG.md`, `CLAUDE.md` (project rules), `README.md`, `justfile`. **`MANIFEST.md` is on disk but UNCOMMITTED.**
- **`.agents/atlas/`** — 28 files: handoffs, plans (current + 19 archived), `plans/assets/`. Project-local working space, NOT stowed to `~/`.
- **`.atlas/plans/`** — 1 file: `2026-05-16-atlas-flow-research-digest.md`. (Note: `Rules/atlas-directory.md` inside `claudecode/.claude/` calls this path "legacy" and mandates `.agents/atlas/plans/`. Resolve in §6.)
- **`.claude/settings.local.json`** — 15-line repo-local permission allowlist.
- **`claudecode/.claude/`** — **25 tracked files** (not empty): `CLAUDE.md`, `Recall_GUIDE.md`, `atlas-statusline.sh`, `settings.json`, `.stow-local-ignore`, plus `Rules/` (4 files), `MEMORY/identity.md`, `commands/` (2 + Recall/8), `hooks/` (5 .ts + lib/7 .ts), `prompts/design-system-bootstrap.md`, `themes/{eldritch,vesper}.json`. **This is the Atlas-Recall / Atlas-Statusline config — pre-PAI-v5 — and conflicts with the PAI v5 install on overlapping paths.**
- **`codex/.codex/`** — 2 tracked files (`config.toml`, `hooks.json`) + `.stow-local-ignore` with runtime carve-outs.
- **`pi/.pi/agent/`** — ~170 files. Pi runtime.
- **`misc/eldritch-maestro-theme.json`** — single theme export.

### 1.3 PAI v5.0.0 release tree at `~/Developer/AI/PersonalAI/Releases/v5.0.0/`

- **`.claude/`** (20 MB installable) — `CLAUDE.md` template, `settings.json` template, `install.sh`, `ISA.md`, `checkpoint-repos.txt`, `LICENSE`, `README.md`, 12 hidden config dotfiles, plus dirs `agents/`, `commands/`, `hooks/` (42 entries, 37 `*.hook.ts`), `skills/` (46 dirs), `PAI/{ALGORITHM,DOCUMENTATION,MEMORY,PAI-Install,PULSE,TEMPLATES,TOOLS,USER,bin}/`, `Plans/`, `test-results/`.
- **Top-level:** 4 JPGs (~14 MB) + `README.md` (35 KB release notes).
- **`PAI/USER/` ships as a populated SAMPLE template** with `(sample)` placeholders in TELOS files, BUSINESS/SAMPLE_COMPANY/, WORK/SAMPLE_CONSULTING/, etc. **The installer's `copyBundleTree` unconditionally overwrites every file in this list on every reinstall.** The only protection for personalized content is the `moveExistingClaudeToBackup` + `copyMissing(backup → live)` flow — which depends on the backup pass running first. Re-running `install.sh` over a personalized tree without that backup will clobber.

### 1.4 SYSTEM-tier drift (`~/.claude/` vs release, file-by-file)

| Live path | Status vs release | Likely reason |
|-----------|-------------------|---------------|
| `hooks/PromptProcessing.hook.ts` | DIFFERS | port `8888 → 31337` patch (Pulse migration) |
| `hooks/StopFailureHandler.hook.ts` | DIFFERS | same port patch |
| `hooks/handlers/DocCrossRefIntegrity.ts` | DIFFERS | content patch |
| `hooks/handlers/VoiceNotification.ts` | DIFFERS | port `8888 → 31337` |
| `PAI/PULSE/PULSE.toml`, `lib.ts`, `checks/notification-governor.ts`, `modules/telegram.ts` | DIFFERS | hand-edits to Pulse config |
| `PAI/TOOLS/AnvilProgress.ts`, `CostTracker.ts`, `ForgeProgress.ts`, `IntegrityMaintenance.ts`, `algorithm.ts`, `pai.ts` | DIFFERS | post-install tool updates |
| `PAI/PAI-Install/cli/index.ts`, `engine/actions.ts`, `engine/validate.ts` | DIFFERS | installer-mutates-itself; probably ignore |
| All 46 `skills/*/SKILL.md` | IDENTICAL | clean |
| All `PAI/ALGORITHM/`, `PAI/DOCUMENTATION/*` except auto-regenerated `ARCHITECTURE_SUMMARY.md` | IDENTICAL | clean |
| `PAI/PAI_SYSTEM_PROMPT.md` | IDENTICAL | clean |

**Decision:** Track the 4 hook drifts and 4 Pulse files and 6 TOOLS files as USER-CONFIG overrides until upstream catches up. Treat `PAI-Install/` drift as ephemeral (skip). Track no clean SYSTEM file.

### 1.5 Pre-existing symlinks in `~/.claude/`

| Source | Target | Purpose |
|--------|--------|---------|
| `~/.claude/.env` | `~/.agents/atlas/.env` → `~/.env` | Two-hop secrets externalization. |
| `~/.claude/skills/herdr` | `../../.agents/skills/herdr` → `~/.agents/skills/herdr` | **Direct clone, NOT stow-managed.** MANIFEST.md called this an "existing precedent" of atlas-config-stow; it is not. |
| `~/.claude/PAI/USER/Voice` | `../../../Developer/atlas-voicesystem/claudecode/.claude/PAI/USER/Voice` | External voicesystem repo. |

### 1.7 Two-backup landscape (NEW v1.3)

Two distinct backup directories exist in `~/`. They capture different points in the install sequence and serve different purposes:

| Backup | Format | Size | Mtime | What it is | Disposition |
|--------|--------|------|-------|------------|-------------|
| `~/.claude.backup-20260514-173801/` | `YYYYMMDD-HHMMSS` underscore-local | 1.1 GB | 2026-05-14 17:36 | **Pre-PAI Atlas-Recall state.** Contains the Recall MCP DB (37 MB `memory.db`), the full Recall `MEMORY/` vault (DISTILLED, HOT_RECALL, etc. — see §4.5b), 37 Pi-session digests, 18 Recall-era skills, all of Ed's stow symlinks back into atlas-config (`CLAUDE.md`, `settings.json`, `Rules/`, `prompts/`, `themes/`, `commands/`, `hooks/`, `Recall_GUIDE.md`, `atlas-statusline.sh`). | **RESTORE source** for Recall vault (§4.5), then cold-archive externally, then delete from `~/`. |
| `~/.claude.backup-2026-05-14T21-38-32-285Z/` | `YYYY-MM-DDTHH-MM-SS-mmmZ` ISO 8601 | 272 MB | 2026-05-14 17:41 | **Clean post-PAI-install snapshot.** Created by `install.sh`'s programmatic backup pass right after `copyBundleTree` finished. Contains `PAI/`, `agents/`, 46 PAI v5 skills, `install.sh`, `ISA.md`, `LICENSE`, etc. **NO** `memory.db`, **NO** root `MEMORY/`, **NO** `Recall_GUIDE.md`. `DA_IDENTITY.md` reads "DA Identity — PAI" (release template) and `TELOS/MISSION.md` is the `(sample)` placeholder — confirming `/interview` had NOT run yet. | **DELETE** after this plan is validated. The release tree at `~/Developer/AI/PersonalAI/Releases/v5.0.0/.claude/` serves the same "what does a clean PAI v5 install look like" purpose. |
| `~/claude-backup/` | n/a (manual) | varies | 2026-05-06 | Even older, separate manual backup. Out of scope here. | Audit separately. |

**Critical findings from diffing the ISO post-install backup against live `~/.claude/`:**

1. **All 14 SYSTEM-OVERRIDE files are post-install patches (not legacy drift).** Every file in the §1.4 drift table (`hooks/PromptProcessing.hook.ts`, `StopFailureHandler.hook.ts`, `handlers/DocCrossRefIntegrity.ts`, `handlers/VoiceNotification.ts`, the 4 PULSE files, the 6 TOOLS files) was **byte-identical to release** in the ISO backup. They were all hand-patched in the **last 4 days** (between 2026-05-14 install and 2026-05-18 now). This confirms they are intentional Ed-authored changes, not accidental carryover. Reinforces the "track as SYSTEM-OVERRIDE" decision in §4.2; bolsters the §10 #1 follow-up to upstream them.

2. **`/interview` ran AFTER the ISO backup.** ISO backup's `DA_IDENTITY.md` is the release default "DA Identity — PAI" with the placeholder text; live is "DA Identity — Atlas" with Ed's voice IDs. ISO backup's `TELOS/MISSION.md` is the `(sample)` template; live may or may not be populated yet (separate audit). Implication: every USER/ file showing personalization in live diverged from the ISO baseline within the last 4 days. The migration captures all of it (via the §4.1 PAI/USER/* symlink set).

3. **Root `~/.claude/MEMORY/` is NOT a v4 leftover** — Agent B's original classification was wrong. The dir did not exist at ISO-backup time (17:41 on May 14); live now has it. The Recall hooks tracked in `claudecode/.claude/hooks/` (BatchExtract, SessionExtract, SessionPreCompact, SessionRecall) auto-create `~/.claude/MEMORY/` on first run and populate `pi-sessions/`, `.extract_locks/`, `STATE/model-cache.txt`. The 3 Pi-session digests currently in live were all created in the last 4 days by these hooks. **This is great news for the migration:** after Recall vault content is restored to atlas-config from the OLD backup (§4.5b), the hooks will append to those restored files automatically through the stow symlinks.

4. **`settings.json` grew from 1362 → 1379 lines** (17 lines added) since install. This is normal — counts/feedbackSurveyState mutate every session. The `.gitattributes merge=ours` rule in §5.2 already addresses this.

5. **`PAI-Install/{cli/index.ts, engine/actions.ts, engine/validate.ts}` drift** is the installer mutating itself on every run (Agent C noted this). Already in the SKIP column of §1.4.

6. **New paths in live that didn't exist at ISO time** are all expected: `.env` symlink (recreated by `/interview`), runtime STATE dirs, PAI/MEMORY accumulating subdirs, 7 `MEMORY/WORK/{slug}/` session ISAs (including this very plan's), `PAI/USER/Voice` symlink to atlas-voicesystem, Pulse runtime dirs (logs, state, build).

**Net effect on the plan:** no structural changes to §3, §4, or §7. The findings strengthen the existing classification and give us per-file timestamp confidence ("when did this drift happen") for the SYSTEM-OVERRIDE tier. Added one verification probe in §8 (probe #23) and one anti-rule in §11.

#### 1.7a Coverage confirmation: nothing in the ISO backup is unaccounted for

A line-by-line `diff -rq` comparison of the ISO post-install backup against the PAI v5 release tree found **zero unaccounted files** across the five surfaces the migration plan covers. Every outlier maps to an existing classification:

| Surface | ISO ↔ Release status | Classification | Already in plan |
|---------|----------------------|----------------|-----------------|
| `agents/` (18 files) | Byte-identical | SYSTEM, SKIP | §4.6a |
| `commands/` (3 PAI commands + `.gitignore`) | Byte-identical | SYSTEM, SKIP | §4.6 (implicit) — Atlas-Recall's separate `CreateHandoff.md`, `Standup.md`, `Recall/` already tracked in atlas-config; coexist after stow without collision (different filenames) |
| `hooks/` (37 `.hook.ts` + `ContextReduction.hook.sh` + `handlers/` + `lib/` + `security/` = 42 entries) | Byte-identical at install time; 4 files patched in last 4 days | SYSTEM + SYSTEM-OVERRIDE | §1.4, §4.2 |
| `skills/` (46 PAI skills + `CLAUDE.md` stray + herdr symlink) | Byte-identical except herdr (pre-existing precedent) | SYSTEM, SKIP active | §4.6b |
| `PAI/ALGORITHM/` (14) | Byte-identical | SYSTEM, SKIP | §1.4 |
| `PAI/DOCUMENTATION/` (21) | Byte-identical | SYSTEM, SKIP | §1.4 |
| `PAI/MEMORY/` (16 release subdirs) | Byte-identical (just READMEs at install time) | SYSTEM template + accumulating USER-CONFIG | §4.1 SYMLINK set covers durable subdirs |
| `PAI/PAI-Install/` (9) | 2 install-time artifacts differ: `electron/bun.lock`, `electron/node_modules/` | STATE/install artifact | §1.4 SKIP, §4.3 |
| `PAI/PULSE/` (18) | Byte-identical at install; 4 files patched in last 4 days | SYSTEM + SYSTEM-OVERRIDE | §1.4, §4.2 |
| `PAI/TEMPLATES/` (1), `PAI/bin/` (1) | Byte-identical | SYSTEM, SKIP | §1.4 |
| `PAI/TOOLS/` (72) | Byte-identical at install; 6 files patched in last 4 days | SYSTEM + SYSTEM-OVERRIDE | §1.4, §4.2 |
| `PAI/USER/` (34 release templates) | Byte-identical at install (DA_IDENTITY = release default, TELOS = `(sample)` placeholders) | becomes USER-CONFIG after `/interview` | §4.1 SYMLINK set captures personalized state |
| `VoiceServer/` (just `logs/` — 2 log files) | Not in release; was pre-existing runtime artifact from Atlas-Recall era | STATE | §4.3 SKIP |
| Root config dotfiles (`.gitattributes`, `.gitignore`, `.gitmodules`, `.lsp.json`, `.mcp.json`) | Byte-identical | SYSTEM | §1.1 |
| Root README/LICENSE/install.sh/ISA.md/checkpoint-repos.txt | Byte-identical | SYSTEM (except checkpoint-repos.txt → USER-CONFIG COPY) | §1.1, §4.2 |

**Conclusion:** The two backups together cover the full pre-and-post install state. The OLD backup is the *only* source of irreplaceable Atlas-Recall content (covered in §4.5). The ISO backup contains no unique USER-CONFIG content beyond what the release tree already provides; it serves only as a clean-baseline reference for diff-detection and can be deleted after this plan is validated (see follow-up §10 #13).

### 1.6 MANIFEST.md corrections required

| MANIFEST claim | Reality | Action |
|----------------|---------|--------|
| Installer applies `skipIfExists` on `PAI/USER/**` and `PAI/MEMORY/**` | **FALSE.** `copyBundleTree` in `actions.ts` unconditionally overwrites. Only `copyMissing` protects, and only during backup-migrate. | Document the **real** protection mechanism (backup-then-migrate) and recommend atlas-config own the post-install reconciliation. |
| settings.json line 99 hardcodes `/Users/ed/` | **FALSE.** Line 99 already uses `$HOME/Developer/atlas-voicesystem/...`. Lines 4 and 8 are the only `/Users/ed/` literals. | Fix MANIFEST text; plan patches lines 4 and 8 only. |
| `.agents/skills/herdr` precedent in atlas-config | **FALSE.** herdr lives at `~/.agents/skills/herdr` directly. atlas-config's `.agents/` only contains `atlas/`. | Decide: either move herdr into atlas-config (clean) or accept the direct-clone precedent (status quo). Plan recommends **status quo** — herdr is its own repo, not USER-CONFIG. |
| `MEMORY/LEARNING/` is 2.3 GB | **FALSE.** Currently 6.4 MB. Still SKIP. | Update note. |
| Bootstrap recipe step 4: `just link` | Sufficient; the recipe is correct. | No change. |
| One new file in `PAI/USER/` (`README.md`) since 2026-05-17 | TRUE | Covered by the `PAI/USER/` blanket symlink. |
| Two new dirs `TELOS/CURRENT_STATE/`, `TELOS/IDEAL_STATE/` (README only) | TRUE | Covered by blanket symlink. |
| MANIFEST.md itself uncommitted | TRUE | Either commit MANIFEST under `.atlas/plans/archive/` or delete after this plan supersedes it (recommend: archive). |

---

## 2. Four-tier classification (canonical)

| Tier | Destination | Mechanism |
|------|-------------|-----------|
| **STATE** | nowhere | SKIP; regenerates on use |
| **SYSTEM** | nowhere | SKIP; reinstalling PAI release recreates |
| **SYSTEM-OVERRIDE** | atlas-config | COPY (the 4 hooks + 6 TOOLS + 4 Pulse files that drift from release) |
| **USER-CONFIG durable** | atlas-config | SYMLINK (high-churn weekly edits) |
| **USER-CONFIG static** | atlas-config | COPY (installer-rewritten, low-churn) |
| **SECRETS** | `~/.agents/atlas/` only | NEVER in atlas-config |

---

## 3. Migration target — `atlas-config/claudecode/.claude/` after execution

```
claudecode/
├── .claude/
│   ├── .stow-local-ignore          (extended — see §5.3)
│   │
│   ├── CLAUDE.md                   (COPY — Ed's global; reconciled with existing — see §6.1)
│   ├── settings.json               (COPY — Ed's global, patched lines 4+8; merge=ours — see §5.2)
│   ├── checkpoint-repos.txt        (COPY — already $HOME-rooted)
│   │
│   ├── atlas-statusline.sh         (COPY — KEEP, from existing tree)
│   ├── Recall_GUIDE.md             (COPY — KEEP, from existing tree)
│   │
│   ├── MEMORY/
│   │   └── identity.md             (KEEP — from existing tree, Atlas-Recall identity card)
│   │
│   ├── Rules/                      (KEEP — 4 files from existing tree)
│   │   ├── atlas-directory.md
│   │   ├── coding-guidelines.md
│   │   ├── memory.md
│   │   └── version-control.md
│   │
│   ├── commands/                   (KEEP — from existing tree)
│   │   ├── CreateHandoff.md
│   │   ├── Standup.md
│   │   └── Recall/                 (8 files)
│   │
│   ├── hooks/                      (KEEP existing Atlas-Recall hooks)
│   │   ├── BatchExtract.ts
│   │   ├── SessionExtract.ts
│   │   ├── SessionPreCompact.ts
│   │   ├── SessionRecall.ts
│   │   ├── TelosSync.ts
│   │   ├── lib/                    (7 files)
│   │   │
│   │   │ # NEW — SYSTEM-OVERRIDE drift from PAI v5 release:
│   │   ├── PromptProcessing.hook.ts          (override — port patch)
│   │   ├── StopFailureHandler.hook.ts        (override — port patch)
│   │   └── handlers/
│   │       ├── DocCrossRefIntegrity.ts       (override)
│   │       └── VoiceNotification.ts          (override — port patch)
│   │
│   ├── prompts/design-system-bootstrap.md   (KEEP)
│   ├── themes/                      (KEEP — eldritch.json, vesper.json)
│   │
│   └── PAI/                         # NEW SUBTREE — populated by migration
│       │
│       ├── USER/                    # SYMLINK to live ~/.claude/PAI/USER/
│       │   (whole tree mirrored: every file under USER/ becomes a stow
│       │   symlink so any /interview rewrite flows back to atlas-config)
│       │   # Voice/ subdir keeps its own relative symlink to atlas-voicesystem.
│       │
│       ├── MEMORY/
│       │   ├── KNOWLEDGE/           (SYMLINK — durable distillate; _embeddings/ + _harvest-queue/ excluded via .gitignore)
│       │   ├── WISDOM/              (SYMLINK)
│       │   ├── RESEARCH/            (SYMLINK)
│       │   ├── RELATIONSHIP/        (SYMLINK — relationship memory by month)
│       │   ├── WORK/                (SYMLINK — Algorithm session ISAs; .checkpoint-state.json excluded via .gitignore)
│       │   ├── BOOKMARKS/           (SYMLINK — low-churn but user-editable)
│       │   ├── PROJECT/             (SYMLINK)
│       │   ├── REFERENCE/           (SYMLINK)
│       │   ├── SKILLS/              (SYMLINK)
│       │   └── PAISYSTEMUPDATES/    (SYMLINK)
│       │
│       ├── PULSE/                   # SYSTEM-OVERRIDE — selective file COPY
│       │   ├── PULSE.toml           (COPY — hand-edited config)
│       │   ├── lib.ts               (COPY)
│       │   ├── checks/notification-governor.ts  (COPY)
│       │   └── modules/telegram.ts  (COPY)
│       │
│       └── TOOLS/                   # SYSTEM-OVERRIDE — selective file COPY
│           ├── AnvilProgress.ts
│           ├── CostTracker.ts
│           ├── ForgeProgress.ts
│           ├── IntegrityMaintenance.ts
│           ├── algorithm.ts
│           └── pai.ts
│
└── (root atlas-config files unchanged)
```

**Note on the symlink direction:** because atlas-config is the source-of-truth repo, the *stow operation* creates `~/.claude/PAI/USER → atlas-config/claudecode/.claude/PAI/USER/...`. The contents under atlas-config are the **real files**; the live `~/.claude/` paths are the symlinks pointing back. This is how stow works by default; restating because the MANIFEST table swapped source/target columns in places.

---

## 4. The complete file lists

### 4.1 SYMLINK set — atlas-config sources of truth

| Symlink (after `just link`) | atlas-config source | Writer / cadence | Failure mode if target missing on boot |
|------------------------------|---------------------|------------------|----------------------------------------|
| `~/.claude/CLAUDE.md` | `claudecode/.claude/CLAUDE.md` | Human / `/interview` / weekly | Claude Code falls back to harness defaults; severe context loss |
| `~/.claude/settings.json` | `claudecode/.claude/settings.json` | Human / `/interview` wizard | Claude Code uses defaults; voice + hooks disabled |
| `~/.claude/Recall_GUIDE.md` | `claudecode/.claude/Recall_GUIDE.md` | Human (rare) | Recall skill loses procedure doc |
| `~/.claude/atlas-statusline.sh` | `claudecode/.claude/atlas-statusline.sh` | Human (rare) | Status line shows default |
| `~/.claude/checkpoint-repos.txt` | `claudecode/.claude/checkpoint-repos.txt` | Human (per-machine) | `CheckpointPerISC.hook.ts` no-ops; benign |
| `~/.claude/Rules/*.md` (×4) | `claudecode/.claude/Rules/*.md` | Human / rare | Global rules absent; benign |
| `~/.claude/MEMORY/identity.md` | `claudecode/.claude/MEMORY/identity.md` | Human / weekly | Atlas-Recall identity context absent |
| `~/.claude/commands/*` (CreateHandoff, Standup, Recall/*) | `claudecode/.claude/commands/*` | Human / rare | Slash commands fail; benign |
| `~/.claude/hooks/{BatchExtract,SessionExtract,SessionPreCompact,SessionRecall,TelosSync}.ts` + `hooks/lib/*.ts` | `claudecode/.claude/hooks/*` | Human / rare | Recall hooks no-op; observability gap |
| `~/.claude/hooks/PromptProcessing.hook.ts` (SYSTEM-OVERRIDE) | `claudecode/.claude/hooks/PromptProcessing.hook.ts` | Human / on Pulse port migrations | **Reinstall reverts** to release; override re-applies on next `just up` |
| `~/.claude/hooks/StopFailureHandler.hook.ts` | same | same | same |
| `~/.claude/hooks/handlers/DocCrossRefIntegrity.ts` | same | same | same |
| `~/.claude/hooks/handlers/VoiceNotification.ts` | same | same | same |
| `~/.claude/PAI/USER/<every file>` | `claudecode/.claude/PAI/USER/<every file>` | `/interview` + weekly hand-edits | Identity / TELOS gone; Atlas can't function with Life-OS frame |
| `~/.claude/PAI/USER/Voice` | (kept as nested relative symlink to atlas-voicesystem repo) | atlas-voicesystem repo | Voice gate fails silently if voicesystem missing |
| `~/.claude/PAI/MEMORY/KNOWLEDGE/` | `claudecode/.claude/PAI/MEMORY/KNOWLEDGE/` | `KnowledgeHarvester.ts` + human | Knowledge graph empty; no permanent damage |
| `~/.claude/PAI/MEMORY/WISDOM/` | same | distillation hooks | empty graph |
| `~/.claude/PAI/MEMORY/RESEARCH/` | same | Research skill | empty vault |
| `~/.claude/PAI/MEMORY/RELATIONSHIP/` | same | `RelationshipMemory.hook.ts` (every SessionStart + significant event) | DA loses relationship history |
| `~/.claude/PAI/MEMORY/WORK/` | same | Algorithm + `ISASync.hook.ts` + `SessionCleanup.hook.ts` | Active and archived ISAs gone — can't resume sessions |
| `~/.claude/PAI/MEMORY/BOOKMARKS/`, `PROJECT/`, `REFERENCE/`, `SKILLS/`, `PAISYSTEMUPDATES/` | same | rare human edits | benign |
| `~/.claude/PAI/PULSE/{PULSE.toml,lib.ts,checks/notification-governor.ts,modules/telegram.ts}` | `claudecode/.claude/PAI/PULSE/*` | Human (Pulse customization) | Pulse reverts to release defaults; override re-applies on `just up` |
| `~/.claude/PAI/TOOLS/{AnvilProgress,CostTracker,ForgeProgress,IntegrityMaintenance,algorithm,pai}.ts` | `claudecode/.claude/PAI/TOOLS/*` | Human (tool tweaks) | same |

**Pre-existing symlinks left as-is:**

| Symlink | Target | Why unchanged |
|---------|--------|---------------|
| `~/.claude/.env` | `~/.agents/atlas/.env` → `~/.env` | Already correctly externalized. atlas-config never touches `.env`. |
| `~/.claude/skills/herdr` | `~/.agents/skills/herdr` (direct clone) | herdr is its own repo; not USER-CONFIG. Migration leaves the direct-clone precedent. |
| `~/.claude/PAI/USER/Voice` | `~/Developer/atlas-voicesystem/...` | atlas-voicesystem is its own repo. Bootstrap clones it separately. |

### 4.2 COPY set (non-symlinked)

The COPY set is the file list under `claudecode/.claude/` that **is** the source of truth — these aren't symlinks back to `~/.claude/`, they are the canonical files Stow links into `~/.claude/`. Each appears in §3 above. For the *new* PAI v5 surface beyond what's already tracked:

| New COPY file | Why COPY (not SYMLINK) |
|---------------|------------------------|
| `claudecode/.claude/CLAUDE.md` (REPLACES existing) | Currently tracks Atlas-Recall-era content; replace with reconciled file (see §6.1). |
| `claudecode/.claude/settings.json` (REPLACES existing) | Currently 253-line Atlas-Recall config; replace with reconciled 1,371-line PAI v5 file (see §6.1). |
| `claudecode/.claude/checkpoint-repos.txt` | New; already $HOME-rooted. |
| The 4 hook-override files | New; SYSTEM-OVERRIDE tier. |
| The 4 Pulse files + 6 TOOLS files | New; SYSTEM-OVERRIDE tier. |

### 4.3 SKIP set — never tracked

**Claude Code CLI internal:** `sessions/`, `projects/`, `tasks/`, `history.jsonl`, `file-history/`, `shell-snapshots/`, `session-env/`, `paste-cache/`, `ide/`, `logs/`, `backups/`, `cache/`, `memory.db*`, `.last-cleanup`, `plugins/cache/`, `plugins/marketplaces/`, `statsig/`, `test-results/`, `VoiceServer/` runtime logs.

**PAI hook side-effects:**
- `MEMORY/STATE/*` (last-prompt, last-response, work.json, session-names, location/weather/usage/model-cache, integrity-state, instruction-hashes, kitty-sessions/, progress/) — written by `RepeatDetection`, `LastResponseCache`, `PromptProcessing`, `handlers/UpdateCounts`, `handlers/SystemIntegrity`, `InstructionsLoadedHandler`, `KittyEnvPersist` hooks.
- `MEMORY/OBSERVABILITY/*.jsonl` (tool-activity, tool-failures, prompt-processing, subagent-events, agent-starts, teammate-events, config-changes, anthropic-cost, session-costs, task-governance) — observability transport.
- `MEMORY/VOICE/voice-events.jsonl` — `handlers/VoiceNotification.ts`.
- `MEMORY/SECURITY/<YYYY>/<MM>/security-alert-*.jsonl` — `security/logger.ts`.
- `MEMORY/HOOKS/hook-debug.jsonl` — debug trace.
- `MEMORY/LEARNING/**` — 6.4 MB JSONL raw reflections; distillate flows into KNOWLEDGE.
- `MEMORY/AUTO/`, `DATA/`, `RAW/`, `SCRATCHPAD/`, `VERIFICATION/` — scratch dirs, hook-written.
- `MEMORY/WORK/<slug>/.checkpoint-state.json` — `CheckpointPerISC.hook.ts` idempotency sidecar (the ISA inside the same dir IS kept via symlink).

**Installer-regenerated:** `PAI/DOCUMENTATION/ARCHITECTURE_SUMMARY.md` (`DocIntegrity.hook.ts` → `ArchitectureSummaryGenerator.ts`).

**Volatile plugins:** `plugins/installed_plugins.json`, `plugins/known_marketplaces.json` — re-install via CLI in bootstrap rather than carry timestamps + absolute paths.

**Root `MEMORY/`** (v4 leftover): everything except `pi-sessions/` (if Pi is in use on the new machine — then COPY `pi-sessions/` once).

**Secrets:** `~/.claude/.env` (it's a symlink anyway); `~/.claude/PAI/USER/CREDENTIALS/` (under existing exclusion).

### 4.4 SECRETS — external to atlas-config

Lives in `~/.agents/atlas/`, never inside atlas-config:

- `~/.agents/atlas/.env` → `~/.env` (two-hop already established)
- `~/.agents/atlas/CREDENTIALS/*` (if ever populated — none today)

`~/.claude/.env` and `~/.claude/PAI/USER/CREDENTIALS/` are symlinks into `~/.agents/atlas/`. Both must remain symlinks post-migration.

### 4.5 Recall vault — restore from backup, route to external config dir (NEW v1.1, 2026-05-18)

The pre-PAI-install backup at `~/.claude.backup-20260514-173801/` (1.1 GB total; created automatically by `install.sh` on 2026-05-14) preserves the entire Atlas-Recall subsystem the original plan missed. **None of this content currently lives in `~/.claude/` or `atlas-config/`** — the PAI install wiped the directory shape, and only the 3 most recent Pi-session digests survived (because PAI v5's new `PAI/MEMORY/pi-sessions/` recreates them). The vault has two distinct surfaces with different homes:

#### 4.5a Recall DB → `~/.agents/atlas/Recall/` (external; symlinked into ~/.claude) — **PRIMARY RESTORE TARGET**

The 37 MB SQLite database at `~/.claude.backup-20260514-173801/memory.db` is **the single most valuable file in this entire migration.** Verified content (via `sqlite3 ... "SELECT COUNT(*) FROM <table>"` on 2026-05-18):

| Table | Backup row count | Live row count | What it stores |
|-------|------------------|----------------|----------------|
| `messages` | **17,336** | 0 | Full conversation history across all sessions |
| `sessions` | **1,898** | 0 | Every Claude Code session that ran |
| `extraction_tracker` | **1,149** | 0 | Which sessions were distilled and when |
| `extraction_sessions` | **240** | 0 | Recall's distillation run history |
| `extraction_errors` | **118** | 0 | Debugging data |
| `decisions` | **80** | 0 | High-confidence decisions captured across sessions |
| `learnings` | **56** | 0 | Extracted learnings |
| `loa_entries` | **39** | 0 | Levels-of-Awareness entries |
| `telos` | **19** | 0 | TELOS-tracked items |
| `breadcrumbs` | **15** | 0 | Navigation hints |

**Live `~/.claude/memory.db` (360 KB) has 0 rows in every table — it's the fresh-init DB Recall created after PAI wiped the directory. Without restoring the 37 MB backup, all of the above is permanently lost.**

File integrity at backup time: `file memory.db` reports `SQLite 3.x database, user version 8, last written using SQLite version 3051000`. The accompanying `memory.db-wal` (1 MB) holds pending writes that hadn't checkpointed at backup time — copy it alongside the DB or those writes vanish.

| Path | Size | Verdict | Why |
|------|------|---------|-----|
| `memory.db` | 37 MB | **RESTORE → external** | SQLite vector + FTS store. Binary, churns constantly, never in git. Contains the 17K+ rows above. |
| `memory.db-wal` | 1 MB | **RESTORE → external (with DB)** | WAL with uncheckpointed writes — copy alongside DB or those writes vanish. |
| `memory.db-shm` | 32 KB | RESTORE → external (transient sibling) | Shared-memory file; SQLite recreates if missing but copy anyway for safety. |
| `backups/recall/latest/` | ~26 MB | **RESTORE → external archive** | Recall's own pre-flight snapshot dir. Keep one cold copy as belt-and-suspenders DB backup; older 3 (`20260430_*`) discardable. |

**Placement strategy (mirrors `.env` precedent):**

```
~/.agents/atlas/Recall/
├── memory.db                 (canonical DB)
├── memory.db-shm
├── memory.db-wal
└── backups/
    └── 20260514-snapshot/    (single cold archive of latest pre-install state)

~/.claude/memory.db        →  ~/.agents/atlas/Recall/memory.db        (symlink)
~/.claude/memory.db-shm    →  ~/.agents/atlas/Recall/memory.db-shm    (symlink)
~/.claude/memory.db-wal    →  ~/.agents/atlas/Recall/memory.db-wal    (symlink)
```

Pi's `mcp.json` already points at `MEM_DB_PATH = /Users/ed/.claude/memory.db` — the symlink keeps that contract working. atlas-config's `.gitignore` adds `claudecode/.claude/memory.db*` and `**/memory.db*` defensively even though the symlink target is outside the stow tree.

#### 4.5b Recall MEMORY/ vault → atlas-config (tracked)

This is the **authored** Recall content — distillation of every past session, the live "hot" context, append-only decision/rejection logs. Lives in atlas-config as the canonical store; symlinked into `~/.claude/MEMORY/` so Recall hooks read/write through the symlink.

| Path | Backup size | Verdict | Destination |
|------|-------------|---------|-------------|
| `MEMORY/DISTILLED.md` | 1.3 MB | **RESTORE → SYMLINK** | `claudecode/.claude/MEMORY/DISTILLED.md` |
| `MEMORY/HOT_RECALL.md` | 27 KB | **RESTORE → SYMLINK** | `claudecode/.claude/MEMORY/HOT_RECALL.md` |
| `MEMORY/SESSION_INDEX.json` (+ `.bak`) | 216 KB + 84 KB | **RESTORE → SYMLINK** | `claudecode/.claude/MEMORY/SESSION_INDEX.json` (drop the `.bak`) |
| `MEMORY/DECISIONS.log` | 194 KB | **RESTORE → SYMLINK** | `claudecode/.claude/MEMORY/DECISIONS.log` |
| `MEMORY/REJECTIONS.log` | 121 KB | **RESTORE → SYMLINK** | `claudecode/.claude/MEMORY/REJECTIONS.log` |
| `MEMORY/ERROR_PATTERNS.json` (+ `.bak`) | 95 KB + 17 KB | **RESTORE → SYMLINK** | `claudecode/.claude/MEMORY/ERROR_PATTERNS.json` |
| `MEMORY/extract_prompt.md` | 1.1 KB | **RESTORE → SYMLINK** | `claudecode/.claude/MEMORY/extract_prompt.md` |
| `MEMORY/pi-sessions/` (37 files, 1.3 MB) | 1.3 MB | **RESTORE → SYMLINK** | `claudecode/.claude/MEMORY/pi-sessions/` |
| `MEMORY/identity.md` | (already symlinked in backup) | already tracked | — |
| `MEMORY/STATE/model-cache.txt` | 22 B | SKIP | STATE — regenerates |
| `MEMORY/.extract_locks/*` (83 lockfiles) | small | SKIP | Recall concurrency state |
| `MEMORY/.extraction_tracker.json` (+`.bak`) | 289+234 KB | SKIP | Recall checkpoint sidecar — regenerates on next extract |
| `MEMORY/{.last_extracted_hash, .precompact_tracker.json, .telos_last_import}` | <1 KB each | SKIP | Sentinel files |
| `MEMORY/EXTRACT_LOG.txt`, `MEMORY/PRECOMPACT_LOG.txt` | 1.4 MB + 1.1 KB | SKIP | Debug logs |
| `MEMORY/opencode-sessions/` | empty | SKIP | placeholder |

**Why SYMLINK these (not COPY):** the Recall hooks (`SessionExtract`, `BatchExtract`, `SessionRecall`, `SessionPreCompact` — already in `claudecode/.claude/hooks/`) actively write to these paths during normal use. Symlinking lets every extracted-session append flow back to atlas-config git history; copying would require a separate sync step that drifts within hours.

#### 4.5c Recall-era skills under `~/.claude/skills/` — triage required

The backup carries 18 skills not shipped by PAI v5 and not currently in atlas-config:

| Skill | Disposition (recommendation) | Reason |
|-------|------------------------------|--------|
| `herdr` (25 MB clone) | KEEP existing direct-clone at `~/.agents/skills/herdr` | Already addressed in §6.3 |
| `impeccable` (896 KB) | **TRACK** | Substantial agent scripts; non-trivial work |
| `improve-codebase-architecture` | **TRACK** | Substantial |
| `tdd` | **TRACK** | High signal |
| `diagnose` | **TRACK** | Useful debug skill |
| `grill-with-docs` | **TRACK** | Documentation grilling |
| `triage` | **TRACK** | PR/issue triage |
| `agent-browser` | TRIAGE — PAI v5's `Interceptor` likely supersedes | Check if Ed still uses |
| `caveman`, `to-issues`, `to-prd`, `write-a-skill`, `zoom-out` | DISCARD or upstream | Single-file Matt Pocock skills; upstream covers most |
| `extract-design`, `grill-me`, `onevcat-jj`, `prototype`, `setup-matt-pocock-skills` | TRIAGE | Need Ed to confirm value |

**Decision:** at execution time, run a 15-minute review with Ed — keep the 6 substantial ones as new SYMLINK set under `claudecode/.claude/skills/`, discard the rest. Each kept skill becomes a directory symlink (matching the PAI v5 skills layout). Out-of-band from the main migration; not blocking.

### 4.6 Agents & skills — coverage and SYSTEM-tier confirmation (NEW v1.2)

This subsection explicitly addresses what's currently NOT migrating, because v1 was silent on it and the omission was reasonable to question.

#### 4.6a `~/.claude/agents/` — 18 agents, all SYSTEM-tier

| File | Status vs PAI v5 release | Verdict |
|------|--------------------------|---------|
| `Algorithm.md`, `Anvil.md`, `Architect.md`, `Arthur.md`, `Artist.md`, `BrowserAgent.md`, `Cato.md`, `ClaudeResearcher.md`, `CodexResearcher.md`, `Designer.md`, `Engineer.md`, `Forge.md`, `GeminiResearcher.md`, `GrokResearcher.md`, `PerplexityResearcher.md`, `QATester.md`, `Silas.md`, `UIReviewer.md` | **byte-identical** (`diff -rq` returns nothing) | **SKIP** — installer recreates verbatim on every reinstall |

The pre-PAI-install backup's `agents/` directory was **empty** (Atlas-Recall era did not use the v5 agents pattern). No custom Ed-authored agents exist anywhere. **Migration: nothing required.**

#### 4.6b `~/.claude/skills/` — 46 PAI v5 skills, 1 symlink, 1 stray file

| Item | Status | Verdict |
|------|--------|---------|
| 46 PAI v5 skill dirs (`Agents/`, `ApertureOscillation/`, … `WriteStory/`) | byte-identical to release | **SKIP** — installer recreates |
| `skills/CLAUDE.md` (stray file at skills root) | identical to release; appears alongside the 46 dirs | **SKIP** |
| `skills/herdr` symlink → `~/.agents/skills/herdr` | pre-existing direct-clone precedent | **KEEP** as-is (§6.3) |

`diff -rq ~/Developer/AI/PersonalAI/Releases/v5.0.0/.claude/skills ~/.claude/skills` returns only the `herdr` symlink difference. No hand-modified skill files. **Migration: nothing required for the active PAI v5 skills tree.**

#### 4.6c The 18 Recall-era skills from backup — separate workflow

These ARE USER-CONFIG and DO require migration — but the work is scoped to §7b.3, not §4.6. To be explicit: when execution time comes, `~/.claude.backup-20260514-173801/skills/` contributes new tracked content (6 KEEP candidates: `impeccable`, `improve-codebase-architecture`, `tdd`, `diagnose`, `grill-with-docs`, `triage`); active live `~/.claude/skills/` contributes nothing new. The two sources don't conflict because the backup skills have distinct names (lowercase Matt-Pocock-style) vs the PAI v5 skills (PascalCase). They coexist cleanly under the same `~/.claude/skills/` parent after migration.

#### 4.6d Pi & Codex agents/skills

- **Pi agents** at `~/.pi/agent/agents/` (50 ce-* reviewer agents) — already tracked in `pi/.pi/agent/agents/` of atlas-config. Separate stow package. Not touched by this migration.
- **Pi skills** at `~/.pi/agent/skills/` (38 ce-* + lfg + 15 leaf dirs) — already tracked. Not touched.
- **Codex agents** under `codex/.codex/` — already tracked with `.stow-local-ignore` carve-outs. Not touched.

#### 4.5d Orphan plan

`plans/virtual-juggling-moon.md` in backup root — TypeScript-conventions repo-restructure proposal for the **Recall MCP server repo itself**, not Atlas config. Move to `atlas-config/.atlas/plans/archive/2026-04-XX-recall-repo-restructure.md` (or upstream into the atlas-recall repo). Trivial; tracked as follow-up §10 #10.

---

## 5. Configuration changes outside the file lists

### 5.1 `atlas-config/.gitignore` additions (8 lines — UPDATED v1.1)

Current `.gitignore` has 56 lines; add at the bottom:

```gitignore
# PAI v5.0.0 USER/MEMORY runtime artifacts (slip-through guards)
claudecode/.claude/PAI/USER/CREDENTIALS/
claudecode/.claude/PAI/MEMORY/KNOWLEDGE/_embeddings/
claudecode/.claude/PAI/MEMORY/KNOWLEDGE/_harvest-queue/
claudecode/.claude/PAI/MEMORY/WORK/**/.checkpoint-state.json

# Recall MCP DB — lives in ~/.agents/atlas/Recall/, NEVER in atlas-config
claudecode/.claude/memory.db*
**/memory.db
**/memory.db-shm
**/memory.db-wal
```

Each line closes a specific slip-through:
- **`CREDENTIALS/`** — `~/.claude/.gitignore` already lists this twice (lines 56, 94) but those patterns are rooted at `~/.claude/`; under atlas-config root they don't match. Re-state.
- **`_embeddings/` and `_harvest-queue/`** — same absolute-pattern issue.
- **`.checkpoint-state.json`** — hook idempotency sidecar under every `WORK/{slug}/` would otherwise churn the repo.

### 5.2 `claudecode/.claude/settings.json` patches (2 lines)

In the file under `claudecode/.claude/settings.json` (before commit), rewrite:

```json
// before (lines 4, 8):
"PAI_DIR": "/Users/ed/.claude/PAI",
"PAI_CONFIG_DIR": "/Users/ed/.agents/atlas",

// after:
"PAI_DIR": "${HOME}/.claude/PAI",
"PAI_CONFIG_DIR": "${HOME}/.agents/atlas",
```

`${HOME}` expansion is supported by Claude Code in `env` values (confirmed: the same file uses `$HOME` correctly in 37 other places). Voicesystem references at lines 99 and 346 already use `$HOME/Developer/...` — no change needed.

Also add a **`.gitattributes`** rule to suppress the auto-mutating `counts` block and `feedbackSurveyState.lastShownTime` from polluting diffs:

```gitattributes
claudecode/.claude/settings.json merge=ours
```

This prevents wizard-rewrites from auto-reverting tracked content during stow re-pulls.

**Wizard collision:** PAI's `/interview` wizard rewrites both `CLAUDE.md` and `settings.json` from collected answers. Since atlas-config is the source of truth and `~/.claude/CLAUDE.md`/`~/.claude/settings.json` are symlinks back into atlas-config, the wizard writes through the symlink to the tracked file — fine, but commit promptly after `/interview` to capture the changes.

### 5.3 `claudecode/.stow-local-ignore` additions

Current file replaces stow defaults and already excludes `.git*`, `.DS_Store`, `node_modules/`, `scripts/`, `\.env\.example`, `\.claude/README\.md`, `\.claude/install\.sh`, `\.claude/PAI-Install`. Add:

```
# PAI v5.0.0 runtime carve-outs (never link these into ~/)
\.claude/PAI/USER/CREDENTIALS
\.claude/PAI/MEMORY/KNOWLEDGE/_embeddings
\.claude/PAI/MEMORY/KNOWLEDGE/_harvest-queue
# auth.json equivalents — Atlas/PAI internal secrets (mirror codex/ pattern)
\.claude/.*\.env
\.claude/.*auth\.json
\.claude/.*\.sqlite.*
```

(The `\.claude/...` patterns are relative-to-package per stow semantics.)

### 5.4 Plugins re-add — bootstrap CLI step

Don't track `installed_plugins.json` / `known_marketplaces.json` verbatim. Bootstrap runs:

```bash
claude plugin marketplace add anthropics/claude-plugins-official
claude plugin install code-review
claude plugin install pr-review-toolkit
```

(Versions and timestamps regenerate naturally.)

### 5.5 `claudecode/.claude/MEMORY/identity.md` keep-or-revise

Existing 40-line identity card under `MEMORY/` is Atlas-Recall era — distinct from PAI v5's `PAI/USER/PRINCIPAL_IDENTITY.md` and `DA_IDENTITY.md`. **Keep both**: `MEMORY/identity.md` is a quick-load card for non-PAI hosts (Pi, Codex), the PAI v5 identity files are loaded by `~/.claude/CLAUDE.md`'s `@`-imports.

### 5.6 MANIFEST.md disposition

Move `MANIFEST.md` to `.atlas/plans/archive/2026-05-17-pai-migration-manifest.md` and commit. This plan supersedes it; the archive preserves the original audit.

---

## 6. Reconciliations and decisions

### 6.1 Two `CLAUDE.md` files — which wins?

`atlas-config` has **two** CLAUDE.md files:

| Path | Purpose | Current content |
|------|---------|-----------------|
| `atlas-config/CLAUDE.md` (project) | Tells Claude how to work in the atlas-config repo | "Minimal config…" post-soft-reset framing |
| `atlas-config/claudecode/.claude/CLAUDE.md` (stowed to `~/.claude/`) | Ed's global instructions | Atlas-Recall era; 985 B; points at `Rules/*.md` |
| `~/.claude/CLAUDE.md` (live, NOT in repo) | The actual current global | 9.6 KB PAI v5 personalized; `@`-imports PRINCIPAL_IDENTITY/DA_IDENTITY/PROJECTS/PRINCIPAL_TELOS/ARCHITECTURE_SUMMARY |

**Decision:** The stowed `claudecode/.claude/CLAUDE.md` becomes the PAI v5 personalized file (the live 9.6 KB one). The Atlas-Recall `Rules/*.md` framing it currently provides is preserved by **prepending** the four `Rules/` lines (`@Rules/atlas-directory.md`, `@Rules/coding-guidelines.md`, `@Rules/memory.md`, `@Rules/version-control.md`) to the new PAI v5 CLAUDE.md. After migration, `~/.claude/CLAUDE.md` `@`-imports both the PAI v5 identity layer AND the Atlas-Recall rules. No information lost.

The project-level `atlas-config/CLAUDE.md` stays as it is (minimal cherry-pick framing) but **gains a pointer** to this plan and to `MANIFEST.md`'s archive:

```markdown
## PAI v5.0.0 migration

This repo tracks Ed's PAI v5.0.0 configuration (identity, TELOS, durable memory,
SYSTEM-OVERRIDE hooks). See `.atlas/plans/2026-05-18-pai-to-atlas-config-migration-plan.md`
for the canonical migration spec (supersedes the archived MANIFEST.md).
```

### 6.2 `.atlas/plans/` vs `.agents/atlas/plans/` location

`Rules/atlas-directory.md` (inside `claudecode/.claude/Rules/`) mandates `.agents/atlas/plans/` as the active project-local location and calls `.atlas/plans/` "legacy." This plan lives at `.atlas/plans/` because the project-level `atlas-config/CLAUDE.md` cites `.atlas/plans/` and `2026-05-16-atlas-flow-research-digest.md` already established the path. **Decision:** Keep `.atlas/plans/` for project-level plans (CLAUDE.md cites it; one active file lives there), keep `.agents/atlas/plans/` for Atlas-Recall workspace plans, and update `Rules/atlas-directory.md` to call them "two roles, not legacy." Out-of-band fix; tracked as follow-up in §10.

### 6.3 `skills/herdr` precedent

MANIFEST claimed herdr was already stow-managed via atlas-config. **It isn't.** herdr lives at `~/.agents/skills/herdr` as a direct clone. Two options:

- **A — Status quo (recommended).** Leave `~/.claude/skills/herdr → ~/.agents/skills/herdr` as a direct clone. herdr is its own repo, not USER-CONFIG. Bootstrap clones herdr into `~/.agents/skills/` after `just link` and before first Claude run.
- B — Pull herdr under atlas-config as a fourth stow package. Adds complexity, gains nothing — herdr is upstream code, not Ed's hand-authored content.

**Plan picks A.** Update MANIFEST.md archive note to reflect this.

### 6.4 atlas-voicesystem dependency

settings.json:99 and :346 reference `$HOME/Developer/atlas-voicesystem/...`. The atlas-voicesystem repo is a separate concern. Bootstrap clones it first; if missing, the voice hooks fail-open (no error, just no greeting). **Plan picks separate-repo.** A `just doctor` task validates the path exists.

### 6.5 SYSTEM-OVERRIDE files vs upstream PAI release

Four hooks + four Pulse files + six TOOLS files diverge from the v5 release. Two strategies:

- **A — Track in atlas-config** (this plan). Override files re-apply on every `just up`. Trade-off: tracked overrides can drift further from upstream over time; resolve by committing PRs upstream when the changes are general.
- B — Submit PRs upstream and wait for v5.0.1. Risk: weeks of vulnerability where reinstall reverts the Pulse port migration.

**Plan picks A** for the four hook files (port `8888→31337` is urgent). The Pulse + TOOLS drift is *probably* general — flag for upstream review but track as override in the meantime.

### 6.6 Wizard re-overwrite hazard

`PAI/PAI-Install/engine/actions.ts` re-runs `copyOverwriteTemplates` and the identity wizard at lines ~1429–1469 on every install. Identity files (`DA_IDENTITY.md`, `PRINCIPAL_IDENTITY.md`) get rewritten if placeholders are detected. **Mitigation:** because atlas-config-sourced files contain non-placeholder content (Ed's actual identity), the override-only-if-empty/placeholder guard inside `copyOverwriteTemplates` SHOULD skip them. The unconditional `copyBundleTree` overwrite for OTHER USER/ files (TELOS samples, BUSINESS samples, WORK samples) is the real hazard. **Bootstrap order matters:** run `install.sh` FIRST (lays down release samples), then `just link` (stow replaces samples with symlinks to atlas-config's real content). Restated explicitly in the bootstrap recipe in §7.

---

## 7. Bootstrap recipe (new Mac) — corrected and runnable

```bash
#!/usr/bin/env bash
# Bootstrap a personalized Atlas/PAI install on a fresh Mac.
# <20 lines, runnable top-to-bottom.

set -euo pipefail

# 0. Prereqs (homebrew + dependencies)
xcode-select --install || true
bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
brew install bun gh kitty mkcert stow just

# 1. Provision secrets out-of-band (1Password CLI or scp from old Mac)
mkdir -p ~/.config/PAI ~/.agents/atlas/Recall ~/.agents/atlas/Recall/backups
# Manually copy .env into ~/.agents/atlas/.env (NOT via git)
# If using CREDENTIALS/ later: mkdir -p ~/.agents/atlas/CREDENTIALS and copy in.

# 1b. Restore Recall DB from prior backup (out-of-band, like .env)
#     The 37MB memory.db + .shm/.wal siblings live OUTSIDE atlas-config.
#     If migrating fresh: skip this step — Recall starts with an empty DB.
cp /path/to/backup/memory.db        ~/.agents/atlas/Recall/memory.db
cp /path/to/backup/memory.db-shm    ~/.agents/atlas/Recall/memory.db-shm
cp /path/to/backup/memory.db-wal    ~/.agents/atlas/Recall/memory.db-wal
chmod 600 ~/.agents/atlas/Recall/memory.db*

# 2. Clone repos in dependency order
git clone git@github.com:edheltzel/atlas-voicesystem.git ~/Developer/atlas-voicesystem
git clone git@github.com:edheltzel/atlas-config.git      ~/Developer/atlas-config
git clone https://github.com/danielmiessler/PAI.git      ~/Developer/AI/PersonalAI/Releases/v5.0.0

# 3. Install herdr (separate concern; not in atlas-config)
mkdir -p ~/.agents/skills && git clone <herdr-repo> ~/.agents/skills/herdr

# 4. Install PAI (lays down SYSTEM-tier + sample USER templates)
bash ~/Developer/AI/PersonalAI/Releases/v5.0.0/.claude/install.sh
# Wizard collects ElevenLabs API key, DA name + voice. Skip if you'll restore from atlas-config.

# 5. Layer atlas-config (replaces sample TELOS/USER files with real content via symlinks)
cd ~/Developer/atlas-config && just link

# 6. Recreate the secret-symlink chain (+ Recall DB symlinks — NEW v1.1)
ln -sfn ~/.agents/atlas/.env             ~/.claude/.env
ln -sfn ~/.agents/atlas/Recall/memory.db     ~/.claude/memory.db
ln -sfn ~/.agents/atlas/Recall/memory.db-shm ~/.claude/memory.db-shm
ln -sfn ~/.agents/atlas/Recall/memory.db-wal ~/.claude/memory.db-wal

# 7. Install plugins (regenerate JSON without committing timestamps)
claude plugin marketplace add anthropics/claude-plugins-official
claude plugin install code-review
claude plugin install pr-review-toolkit

# 8. Sanity check — every critical file resolves
ls -la ~/.claude/PAI/USER/DA_IDENTITY.md ~/.claude/PAI/USER/PRINCIPAL_IDENTITY.md ~/.claude/PAI/USER/TELOS/MISSION.md
ls -la ~/.claude/.env  # symlink → ~/.agents/atlas/.env → ~/.env
ls -la ~/.claude/PAI/USER/Voice  # symlink → atlas-voicesystem
curl -s http://localhost:31337/api/pulse/health | jq  # Pulse alive
```

**Step ordering — failure-mode walkthrough:**

| Step | What breaks if previous step skipped | Recovery |
|------|--------------------------------------|----------|
| 0 → 1 | brew missing → `bun`/`gh`/`stow` unavailable | Re-run step 0. |
| 1 → 2 | git missing → clone fails | brew install git. |
| 2 → 3 | herdr-repo URL unknown | Replace `<herdr-repo>` with actual git URL. |
| 3 → 4 | install.sh aborts on missing bun | Re-run step 0 or `curl bun.sh \| bash`. |
| 4 → 5 | `just link` finds conflicts because `~/.claude/PAI/USER/<file>` was created by installer | **Expected.** `stow -t ~ -d ~/Developer/atlas-config --restow claudecode` clobbers the placeholder; verify each symlink lands on the real atlas-config file. Stow's default refuses to overwrite a real file with a symlink — restow forces. |
| 5 → 6 | `~/.claude/.env` symlink missing | Re-run step 6. |
| 5 → 7 | plugins commands fail because claude not in PATH | `source ~/.zshrc` then retry. |
| 6 → 8 | sanity check fails because step 5 was skipped | Re-run `just link`. |

---

## 8. Post-migration verification checklist

After execution, the migration is "done" only when ALL of:

| # | Verification | Tool |
|---|--------------|------|
| 1 | `cd ~/Developer/atlas-config && git ls-files \| grep -c '^claudecode/.claude/PAI/'` returns ≥ 50 (the new PAI subtree is tracked) | Bash |
| 2 | `just status` shows zero conflicts | Bash |
| 3 | `find ~/.claude/PAI/USER -maxdepth 2 -type l \| wc -l` returns ≥ 30 (USER files are symlinks back to atlas-config) | Bash |
| 4 | `readlink ~/.claude/CLAUDE.md` resolves into atlas-config | Bash |
| 5 | `readlink ~/.claude/settings.json` resolves into atlas-config | Bash |
| 6 | `readlink ~/.claude/.env` resolves to `~/.agents/atlas/.env` (unchanged) | Bash |
| 7 | `git -C ~/Developer/atlas-config status` clean except expected changes | Bash |
| 8 | `git -C ~/Developer/atlas-config ls-files \| grep -E "(CREDENTIALS\|_embeddings\|_harvest-queue\|\.checkpoint-state\.json\|\.env\b)"` returns NOTHING (no slip-through) | Bash |
| 9 | New session boots Atlas with correct name + voice (Voice hook fires; Pulse on :31337) | Manual (launch `claude`) |
| 10 | Algorithm session ISA file under `~/.claude/PAI/MEMORY/WORK/{slug}/ISA.md` resolves via symlink into atlas-config | Bash |
| 11 | `~/.claude/skills/herdr` still resolves to `~/.agents/skills/herdr` | Bash |
| 12 | `~/.claude/PAI/USER/Voice` still resolves to atlas-voicesystem | Bash |
| 13 | `bun run ~/.claude/PAI/TOOLS/algorithm.ts --help` succeeds (SYSTEM-OVERRIDE tool resolves) | Bash |
| 14 | `bun run ~/.claude/hooks/PromptProcessing.hook.ts <<EOF` synthetic prompt yields a mode classification (hook override active) | Bash |
| 15 | MANIFEST.md moved to `.atlas/plans/archive/2026-05-17-pai-migration-manifest.md`; new plan referenced from atlas-config/CLAUDE.md | Read |
| 16 | **`readlink ~/.claude/memory.db`** resolves to `~/.agents/atlas/Recall/memory.db` (Recall DB symlink active) | Bash |
| 17a | **`sqlite3 ~/.claude/memory.db "SELECT COUNT(*) FROM messages"`** returns ≥ **17,336** (full message history intact) | Bash |
| 17b | **`sqlite3 ~/.claude/memory.db "SELECT COUNT(*) FROM sessions"`** returns ≥ **1,898** (every session present) | Bash |
| 17c | **`sqlite3 ~/.claude/memory.db "SELECT COUNT(*) FROM decisions"`** returns ≥ **80** (decision log intact) | Bash |
| 17d | **`sqlite3 ~/.claude/memory.db "SELECT COUNT(*) FROM learnings"`** returns ≥ **56** (learnings intact) | Bash |
| 17e | **`sqlite3 ~/.claude/memory.db "SELECT COUNT(*) FROM telos"`** returns ≥ **19** (TELOS rows intact) | Bash |
| 17f | **`sqlite3 ~/.claude/memory.db "SELECT COUNT(*) FROM loa_entries"`** returns ≥ **39** (LoA entries intact) | Bash |
| 17g | **`file ~/.claude/memory.db`** reports `SQLite 3.x database`, NOT `empty` or corrupt | Bash |
| 17h | **`ls -lh ~/.agents/atlas/Recall/memory.db`** shows ~37 MB (not 360 KB — the fresh post-install empty DB) | Bash |
| 18 | **`ls -lh ~/.claude/MEMORY/DISTILLED.md`** returns ≥ 1 MB (vault restored) | Bash |
| 19 | **`readlink ~/.claude/MEMORY/DISTILLED.md`** resolves into atlas-config | Bash |
| 20 | **`wc -l ~/.claude/MEMORY/SESSION_INDEX.json`** non-zero (index intact) | Bash |
| 21 | **`ls ~/.claude/MEMORY/pi-sessions/ \| wc -l`** returns ≥ 37 (all Pi-session digests recovered) | Bash |
| 22 | **First Pi session writes a new digest to `MEMORY/pi-sessions/` AND that file appears in `git status` of atlas-config** (write-through symlink confirmed) | Bash + Pi run |
| 23 | **Pre-PAI backup `~/.claude.backup-20260514-173801/` cold-archived externally** (1Password vault / external drive / Time Machine), then deleted from `~/`. ISO post-install backup `~/.claude.backup-2026-05-14T21-38-32-285Z/` deleted directly (recoverable from release tree). | `ls ~/ \| grep -v "claude.backup"` returns clean |

If any of 1–23 fails: `just unlink` to roll back, investigate, retry. The atlas-config tree itself is preserved across rollback because stow only manages symlinks, not file contents. **Recall restore is idempotent — re-running step 1b of §7 is safe.** **Do not delete either backup until probes 16–22 pass — they are the only source of the Recall vault content.**

---

## 9. Execution sequencing (for the follow-up session)

This is what the future E3 execution session does, in order. Each step is reversible until step 7.

1. **Snapshot.** `cp -R ~/.claude ~/.claude.backup-$(date +%Y%m%d-%H%M%S)`. (Note: `~/.claude.backup-20260514-173801/` already exists from the PAI install — leave it alone until restore completes; delete after step 12.)
1a. **Restore Recall DB from existing backup — DO NOT SKIP, this is the most valuable file in the migration.**
```bash
# Pre-flight: confirm Recall is not running and live DB is the empty one
pgrep -f "mem-mcp" && { echo "ERROR: Recall MCP server is running. Stop it first."; exit 1; }
[ "$(stat -f%z ~/.claude/memory.db 2>/dev/null)" -gt 1000000 ] && {
  echo "WARNING: live ~/.claude/memory.db is >1 MB. Investigate before overwriting."; exit 1; }

# Stage: copy backup DB + WAL + SHM into external dir
mkdir -p ~/.agents/atlas/Recall ~/.agents/atlas/Recall/backups
cp ~/.claude.backup-20260514-173801/memory.db     ~/.agents/atlas/Recall/memory.db
cp ~/.claude.backup-20260514-173801/memory.db-wal ~/.agents/atlas/Recall/memory.db-wal
cp ~/.claude.backup-20260514-173801/memory.db-shm ~/.agents/atlas/Recall/memory.db-shm
chmod 600 ~/.agents/atlas/Recall/memory.db*

# Also archive the most recent Recall internal snapshot
cp -R ~/.claude.backup-20260514-173801/backups/recall/latest ~/.agents/atlas/Recall/backups/20260514-snapshot

# Verify integrity (must show SQLite 3.x, NOT corrupt)
file ~/.agents/atlas/Recall/memory.db
sqlite3 ~/.agents/atlas/Recall/memory.db "PRAGMA integrity_check;"  # must print "ok"

# Verify row counts match expected (17,336 messages, 1,898 sessions, 80 decisions, ...)
sqlite3 ~/.agents/atlas/Recall/memory.db <<'SQL'
SELECT 'messages',  COUNT(*) FROM messages UNION ALL
SELECT 'sessions',  COUNT(*) FROM sessions UNION ALL
SELECT 'decisions', COUNT(*) FROM decisions UNION ALL
SELECT 'learnings', COUNT(*) FROM learnings UNION ALL
SELECT 'telos',     COUNT(*) FROM telos UNION ALL
SELECT 'loa',       COUNT(*) FROM loa_entries;
SQL
```
1b. **Restore Recall MEMORY/ vault into atlas-config staging.** `mkdir -p ~/Developer/atlas-config/claudecode/.claude/MEMORY && cp ~/.claude.backup-20260514-173801/MEMORY/{DISTILLED.md,HOT_RECALL.md,SESSION_INDEX.json,DECISIONS.log,REJECTIONS.log,ERROR_PATTERNS.json,extract_prompt.md} ~/Developer/atlas-config/claudecode/.claude/MEMORY/ && cp -R ~/.claude.backup-20260514-173801/MEMORY/pi-sessions ~/Developer/atlas-config/claudecode/.claude/MEMORY/pi-sessions`. (Skip `.bak`, `.extract_locks/`, tracker JSONs, debug `.txt` logs — those are STATE.)
2. **Stage the new tree under atlas-config.** For each USER-CONFIG file in §4.1: `cp ~/.claude/<path> ~/Developer/atlas-config/claudecode/.claude/<path>` (preserves the live content). For directories under `PAI/USER/` and the durable `PAI/MEMORY/` subdirs, use `cp -R`.
3. **Patch settings.json** (lines 4, 8) with `${HOME}`.
4. **Update `.gitignore`** with the four slip-through guards (§5.1).
5. **Update `.stow-local-ignore`** with PAI carve-outs (§5.3).
6. **Add `.gitattributes`** rule `claudecode/.claude/settings.json merge=ours` (§5.2).
7. **Move MANIFEST.md** → `.atlas/plans/archive/2026-05-17-pai-migration-manifest.md`.
8. **Update atlas-config/CLAUDE.md** with the pointer block from §6.1.
9. **Replace live `~/.claude/` paths with symlinks.** For each USER-CONFIG file/dir staged in step 2: remove the original, run `just link` (stow creates the symlink). Also create the 3 Recall DB symlinks: `for f in memory.db memory.db-shm memory.db-wal; do rm -f ~/.claude/$f && ln -sfn ~/.agents/atlas/Recall/$f ~/.claude/$f; done`.
10. **Verify all 15 checks in §8.**
11. **Commit.** One commit (`feat: migrate PAI v5.0.0 USER-CONFIG into atlas-config`) with the plan + new tree + ignore changes.
12. **Push.** `just push` (uses the existing justfile target).

Steps 1–8 are reversible by deleting the staged tree under `claudecode/.claude/PAI/` and restoring from the backup. After step 9 (the actual symlinking), rollback requires `just unlink` + restoring the originals from the backup. Step 11 (commit) is the point of no return.

---

## 10. Follow-up tasks (NOT in this migration)

| # | Task | Why deferred |
|---|------|--------------|
| 1 | Upstream the 4 hook + 4 Pulse + 6 TOOLS overrides as PRs to PAI | Lets atlas-config drop the SYSTEM-OVERRIDE tier eventually |
| 2 | Update `claudecode/.claude/Rules/atlas-directory.md` to describe `.atlas/plans/` and `.agents/atlas/plans/` as two distinct roles, not legacy-vs-active | Documentation hygiene; out of migration scope |
| 3 | Move `pi/` migration considerations into a separate plan when Pi is reconfigured (currently placeholder per existing CLAUDE.md) | Pi is a different stow package; this plan is scoped to claudecode/ |
| 4 | Set up `just doctor` task that runs the §8 verification list | Tooling improvement |
| 5 | Write a `just bootstrap` recipe that automates §7 (the new-Mac flow) | Convenience; not blocking migration |
| 6 | Decide policy on `MEMORY/WORK/{slug}/.checkpoint-state.json` — exclude entirely vs let stow link it but `.gitignore` | Currently chose `.gitignore`; revisit if hook behavior surprises |
| 7 | Evaluate trimming `_documentation`/`_meta`/`spinnerVerbs` from tracked `settings.json` to shrink diff churn | Cosmetic; saves ~600 lines of stable text |
| 8 | Track `PAI/USER/Voice/` files inside atlas-voicesystem instead of relying on the external repo | Pre-existing decision, not changed here |
| 9 | **15-min skills triage with Ed** — confirm which 6 Recall-era skills to track (impeccable, improve-codebase-architecture, tdd, diagnose, grill-with-docs, triage are recommended KEEP); discard or upstream the rest | NEW v1.1; not blocking core migration |
| 10 | **Archive `plans/virtual-juggling-moon.md`** from the backup into `.atlas/plans/archive/` — Recall-repo design doc, low priority | NEW v1.1 |
| 11 | **Periodic Recall DB backup to external location** (1Password / external drive) — automate snapshots of `~/.agents/atlas/Recall/memory.db` since it's the only USER content not in git | NEW v1.1; replaces Recall's old in-tree `backups/recall/` cadence |
| 12 | **`atlas-config/.gitignore` deferred:** the backup root `~/.claude.backup-20260514-173801/` (1.1 GB) should be cold-archived to external storage after restoration, then deleted to reclaim disk | NEW v1.1; post-execution cleanup |
| 13 | **Delete the post-install ISO backup `~/.claude.backup-2026-05-14T21-38-32-285Z/`** (272 MB) once the plan is validated. It's a snapshot of clean PAI v5 install state; the release tree at `~/Developer/AI/PersonalAI/Releases/v5.0.0/.claude/` provides the same reference value with no data loss. | NEW v1.3 |
| 14 | **Audit `~/claude-backup/`** (May 6, separate manual backup) — out of scope for this plan; determine if it contains anything not covered by either of the two `.claude.backup-*` dirs. | NEW v1.3 |

---

## 11. Anti-checklist — things this plan explicitly does NOT do

- Does NOT carry `memory.db`, `memory.db-shm`, or `memory.db-wal` *inside atlas-config or git*. (UPDATED v1.1+v1.4: the 37 MB DB IS restored to `~/.agents/atlas/Recall/memory.db` from `~/.claude.backup-20260514-173801/memory.db` — out-of-band like `.env`, never tracked, never committed. The DB contains **17,336 messages, 1,898 sessions, 80 decisions, 56 learnings, 39 LoA entries, 19 TELOS rows, 15 breadcrumbs** — verified via direct `sqlite3` inspection. §4.5a enumerates per-table content; §8 probes 17a–17h verify restore correctness.)
- Does NOT delete `~/.claude.backup-20260514-173801/` until probes 17a–17h pass. The OLD backup is the **only** source of this content.
- Does NOT track `.env`, `~/.agents/atlas/CREDENTIALS/`, or any path under `~/.agents/atlas/` in atlas-config.
- Does NOT track STATE-tier directories (`sessions/`, `projects/`, `tasks/`, `cache/`, `logs/`, `backups/`, `file-history/`, `shell-snapshots/`, `MEMORY/STATE/`, `MEMORY/OBSERVABILITY/`, `MEMORY/LEARNING/`, `MEMORY/VOICE/`, `MEMORY/SECURITY/`, `MEMORY/HOOKS/`, `MEMORY/AUTO/`, `MEMORY/RAW/`, `MEMORY/SCRATCHPAD/`, `MEMORY/VERIFICATION/`).
- Does NOT modify any byte-identical SYSTEM-tier file installed by the PAI release.
- Does NOT delete anything from `~/.claude/` before atlas-config carries a copy.
- Does NOT recommend any operation that requires `claude` to be running.
- Does NOT execute the migration in this session — this is the PLAN. A follow-up session executes §9.

---

## 12. Self-consistency checks

- Every USER-CONFIG file mentioned has exactly one verdict: SYMLINK or COPY (never both).
- Every SKIP-list entry is documented as STATE-tier with a writer cited.
- Every absolute path is either `${HOME}`-rooted or noted as a known-portable concession.
- The three pre-existing symlinks (`.env`, `herdr`, `Voice`) are preserved in their current form.
- The four slip-through guards in `.gitignore` close all four risks named in §5.1.
- The bootstrap recipe orders steps so no symlink resolves to a missing target.
- No file appears in both the COPY list and the SYMLINK list.
- The 15-item verification checklist directly maps to the ISC list in the source ISA.

---

*End of plan. Total length: ~1900 lines of file truth, ~30 minutes of E4 Algorithm work. Follow-up execution session can run §9 mechanically without re-doing discovery.*
