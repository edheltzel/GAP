# Codebase Audit Cleanup Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Remove all confirmed orphaned files, fix broken references, correct case-sensitive paths, and update stale documentation — based on the comprehensive multi-agent audit with red-team validation.

**Architecture:** Deletions first (safe, confirmed by red-team), then doc fixes, then path corrections. Each task is independently committable. No runtime behavior changes — only removing dead code and fixing references.

**Tech Stack:** GNU Stow (symlinks), TypeScript (hooks/tools), Markdown (skills/docs), Vue (Observability client)

**Source:** Audit reports at `~/.claude/MEMORY/WORK/20260303-042800_codebase-orphan-audit/`

**Protected files (NEVER touch):** `Banner.ts`, `pai.ts`, `VoiceGreeting.hook.ts`, `VoiceGate.hook.ts`, `atlas-statusline.sh`, `settings.json` (wholesale)

---

### Task 1: Delete orphaned hook libs (3 files)

**Files:**
- Delete: `claudecode/.claude/hooks/lib/hook-io.ts`
- Delete: `claudecode/.claude/hooks/lib/metadata-extraction.ts`
- Delete: `claudecode/.claude/hooks/lib/prd-template.ts`

**Step 1: Verify no imports exist before deletion**

Run:
```bash
rg "hook-io|metadata-extraction|prd-template" claudecode/.claude/ --type ts -l
```
Expected: Only matches within the 3 files themselves (self-references) or zero matches.

**Step 2: Delete the files**

```bash
rm claudecode/.claude/hooks/lib/hook-io.ts
rm claudecode/.claude/hooks/lib/metadata-extraction.ts
rm claudecode/.claude/hooks/lib/prd-template.ts
```

**Step 3: Restow and verify no broken symlinks**

```bash
just update
ls -la ~/.claude/hooks/lib/ | grep -c "broken"
```
Expected: 0 broken symlinks.

**Step 4: Commit**

```bash
git add -u claudecode/.claude/hooks/lib/
git commit -m "chore(hooks): remove orphaned lib files (hook-io, metadata-extraction, prd-template)

These v4.0.3 additions were never wired into any hook or handler.
Confirmed zero importers across all 6 loading mechanisms.
Red-team validated: HIGH confidence orphans."
```

---

### Task 2: Delete orphaned lib/migration package (5 files)

**Files:**
- Delete: `claudecode/.claude/lib/migration/index.ts`
- Delete: `claudecode/.claude/lib/migration/scanner.ts`
- Delete: `claudecode/.claude/lib/migration/extractor.ts`
- Delete: `claudecode/.claude/lib/migration/merger.ts`
- Delete: `claudecode/.claude/lib/migration/validator.ts`

**Step 1: Verify no external imports**

Run:
```bash
rg "lib/migration" claudecode/.claude/ --type ts -l
```
Expected: Only matches within `lib/migration/` itself.

**Step 2: Delete the entire directory**

```bash
rm -r claudecode/.claude/lib/migration/
```

**Step 3: Check if lib/ is now empty and remove if so**

```bash
ls claudecode/.claude/lib/
```
If empty, `rmdir claudecode/.claude/lib/`.

**Step 4: Restow**

```bash
just update
```

**Step 5: Commit**

```bash
git add -u claudecode/.claude/lib/
git commit -m "chore(lib): remove orphaned migration package (5 files)

Entire lib/migration/ tree had zero external importers.
Functionality duplicated by PAI-Install/engine/detect.ts and actions.ts.
Red-team validated: HIGH confidence orphan tree."
```

---

### Task 3: Delete orphaned skills/CORE directory (~27 files)

**Files:**
- Delete: `claudecode/.claude/skills/CORE/` (entire directory)

**Step 1: Verify no references exist**

Run:
```bash
rg "skills/CORE" claudecode/.claude/ -l
```
Expected: Zero matches.

**Step 2: Delete the directory**

```bash
rm -r claudecode/.claude/skills/CORE/
```

**Step 3: Restow**

```bash
just update
```

**Step 4: Commit**

```bash
git add -u claudecode/.claude/skills/CORE/
git commit -m "chore(skills): remove orphaned CORE directory

No SKILL.md, unreachable from any loading mechanism.
Contains duplicate ACTIONS/PIPELINES superseded by PAI/ACTIONS.
Red-team validated: HIGH confidence orphan directory."
```

---

### Task 4: Delete orphaned banner prototypes (5 files)

**Files:**
- Delete: `claudecode/.claude/skills/PAI/Tools/BannerMatrix.ts`
- Delete: `claudecode/.claude/skills/PAI/Tools/BannerNeofetch.ts`
- Delete: `claudecode/.claude/skills/PAI/Tools/BannerPrototypes.ts`
- Delete: `claudecode/.claude/skills/PAI/Tools/BannerRetro.ts`
- Delete: `claudecode/.claude/skills/PAI/Tools/BannerTokyo.ts`

NOTE: `Banner.ts` is PROTECTED — do NOT delete it.

**Step 1: Verify no imports**

Run:
```bash
rg "BannerMatrix|BannerNeofetch|BannerPrototypes|BannerRetro|BannerTokyo" claudecode/.claude/ --type ts -l
```
Expected: Only matches within the files themselves.

**Step 2: Delete**

```bash
rm claudecode/.claude/skills/PAI/Tools/Banner{Matrix,Neofetch,Prototypes,Retro,Tokyo}.ts
```

**Step 3: Commit**

```bash
git add -u claudecode/.claude/skills/PAI/Tools/
git commit -m "chore(pai): remove orphaned banner prototypes (5 files)

BannerMatrix, BannerNeofetch, BannerPrototypes, BannerRetro, BannerTokyo
all superseded by the canonical Banner.ts. Zero references anywhere."
```

---

### Task 5: Delete orphaned PAI CLI tools (12 files)

**Files:**
- Delete: `claudecode/.claude/skills/PAI/Tools/FeatureRegistry.ts`
- Delete: `claudecode/.claude/skills/PAI/Tools/GenerateCapabilityIndex.ts`
- Delete: `claudecode/.claude/skills/PAI/Tools/GenerateSkillIndex.ts`
- Delete: `claudecode/.claude/skills/PAI/Tools/GetCounts.ts`
- Delete: `claudecode/.claude/skills/PAI/Tools/IntegrityMaintenance.ts`
- Delete: `claudecode/.claude/skills/PAI/Tools/LoadSkillConfig.ts`
- Delete: `claudecode/.claude/skills/PAI/Tools/NeofetchBanner.ts`
- Delete: `claudecode/.claude/skills/PAI/Tools/OpinionTracker.ts`
- Delete: `claudecode/.claude/skills/PAI/Tools/PAILogo.ts`
- Delete: `claudecode/.claude/skills/PAI/Tools/PipelineMonitor.ts`
- Delete: `claudecode/.claude/skills/PAI/Tools/PipelineOrchestrator.ts`
- Delete: `claudecode/.claude/skills/PAI/Tools/SplitAndTranscribe.ts`

**Step 1: Verify no imports across all mechanisms**

Run:
```bash
rg "FeatureRegistry|GenerateCapabilityIndex|GenerateSkillIndex|GetCounts|IntegrityMaintenance|LoadSkillConfig|NeofetchBanner|OpinionTracker|PAILogo|PipelineMonitor|PipelineOrchestrator|SplitAndTranscribe" claudecode/.claude/ -l --type ts --type md
```
Expected: Only matches within the files themselves or audit reports.

**Step 2: Delete**

```bash
rm claudecode/.claude/skills/PAI/Tools/{FeatureRegistry,GenerateCapabilityIndex,GenerateSkillIndex,GetCounts,IntegrityMaintenance,LoadSkillConfig,NeofetchBanner,OpinionTracker,PAILogo,PipelineMonitor,PipelineOrchestrator,SplitAndTranscribe}.ts
```

**Step 3: Commit**

```bash
git add -u claudecode/.claude/skills/PAI/Tools/
git commit -m "chore(pai): remove 12 orphaned CLI tools

All had zero references across all 6 loading mechanisms.
Functional but undiscoverable — no SKILL.md, TOOLS.md, or workflow routing.
Red-team validated: HIGH confidence orphans."
```

---

### Task 6: Delete MEDIUM-confidence orphaned PAI tools (3 files) + doc-dependencies.json

**Files:**
- Delete: `claudecode/.claude/skills/PAI/Tools/AlgorithmPhaseReport.ts`
- Delete: `claudecode/.claude/skills/PAI/Tools/SkillSearch.ts`
- Delete: `claudecode/.claude/skills/PAI/Tools/PreviewMarkdown.ts`
- Delete: `claudecode/.claude/skills/PAI/doc-dependencies.json`

NOTE: `SecretScan.ts` and `RelationshipReflect.ts` are KEPT — useful tools that should be documented in TOOLS.md instead of deleted (Task 13).

**Step 1: Verify no real consumers**

Run:
```bash
rg "AlgorithmPhaseReport|PreviewMarkdown" claudecode/.claude/ -l --type ts --type md
rg "doc-dependencies.json" claudecode/.claude/ -l
```
Expected: Only self-references.

For SkillSearch — distinguish from the Skill tool concept:
```bash
rg "Tools/SkillSearch|from.*SkillSearch|import.*SkillSearch" claudecode/.claude/ --type ts -l
```
Expected: Only `SkillSearch.ts` itself.

**Step 2: Delete**

```bash
rm claudecode/.claude/skills/PAI/Tools/{AlgorithmPhaseReport,SkillSearch,PreviewMarkdown}.ts
rm claudecode/.claude/skills/PAI/doc-dependencies.json
```

**Step 3: Commit**

```bash
git add -u claudecode/.claude/skills/PAI/
git commit -m "chore(pai): remove MEDIUM-confidence orphans (3 tools + doc-dependencies.json)

AlgorithmPhaseReport.ts, SkillSearch.ts, PreviewMarkdown.ts — zero routing.
doc-dependencies.json — zero references anywhere in codebase."
```

---

### Task 7: Delete dead agent code + artifacts

**Files:**
- Delete: `claudecode/.claude/skills/Agents/Tools/SpawnAgentWithProfile.ts`
- Delete: `claudecode/.claude/skills/Agents/REDESIGN-SUMMARY.md`

**Step 1: Verify SpawnAgentWithProfile has no consumers**

Run:
```bash
rg "SpawnAgentWithProfile" claudecode/.claude/ -l --type ts --type md
```
Expected: Only the file itself and audit reports.

**Step 2: Delete**

```bash
rm claudecode/.claude/skills/Agents/Tools/SpawnAgentWithProfile.ts
rm claudecode/.claude/skills/Agents/REDESIGN-SUMMARY.md
```

**Step 3: Commit**

```bash
git add -u claudecode/.claude/skills/Agents/
git commit -m "chore(agents): remove dead SpawnAgentWithProfile + REDESIGN-SUMMARY

SpawnAgentWithProfile.ts has broken import (AgentProfileLoader deleted in v2.0.0).
REDESIGN-SUMMARY.md is unreferenced historical artifact.
Both confirmed by red-team validation."
```

---

### Task 8: Delete misc orphans (HelloWorld.vue, generate-welcome.ts, VoiceServer/~ artifact)

**Files:**
- Delete: `claudecode/.claude/Observability/apps/client/src/components/HelloWorld.vue`
- Delete: `claudecode/.claude/PAI-Install/generate-welcome.ts`
- Delete: `claudecode/.claude/VoiceServer/~` (directory tree)

**Step 1: Verify HelloWorld is unused**

Run:
```bash
rg "HelloWorld" claudecode/.claude/Observability/apps/client/src/ -l
```
Expected: Only `HelloWorld.vue` itself.

**Step 2: Verify generate-welcome.ts is unused**

Run:
```bash
rg "generate-welcome" claudecode/.claude/PAI-Install/ -l
```
Expected: Zero matches (or only the file itself).

**Step 3: Delete all three**

```bash
rm claudecode/.claude/Observability/apps/client/src/components/HelloWorld.vue
rm claudecode/.claude/PAI-Install/generate-welcome.ts
rm -r claudecode/.claude/VoiceServer/~
```

**Step 4: Restow (VoiceServer is unfolded, needs restow)**

```bash
just update
```

**Step 5: Commit**

```bash
git add -u claudecode/.claude/Observability/ claudecode/.claude/PAI-Install/ claudecode/.claude/VoiceServer/
git commit -m "chore: remove HelloWorld.vue, generate-welcome.ts, VoiceServer/~ artifact

HelloWorld.vue: Vite scaffold leftover, zero importers.
generate-welcome.ts: standalone utility with broken output path.
VoiceServer/~: stow tilde-path mishap, empty Plans/ subdir."
```

---

### Task 9: Fix case-sensitive SKILL.md paths — Aphorisms

**Files:**
- Modify: `claudecode/.claude/skills/Aphorisms/SKILL.md:43-58`

**Step 1: Read current broken lines**

Read `claudecode/.claude/skills/Aphorisms/SKILL.md` lines 40-60.

**Step 2: Fix all 4 workflow references**

Replace:
- `~/.claude/skills/aphorisms/Workflows/Find-aphorism.md` → `~/.claude/skills/Aphorisms/Workflows/FindAphorism.md`
- `~/.claude/skills/aphorisms/Workflows/Add-aphorism.md` → `~/.claude/skills/Aphorisms/Workflows/AddAphorism.md`
- `~/.claude/skills/aphorisms/Workflows/Research-thinker.md` → `~/.claude/skills/Aphorisms/Workflows/ResearchThinker.md`
- `~/.claude/skills/aphorisms/Workflows/Search-aphorisms.md` → `~/.claude/skills/Aphorisms/Workflows/SearchAphorisms.md`

**Step 3: Verify fix**

Run:
```bash
rg "aphorisms/Workflows" claudecode/.claude/skills/Aphorisms/SKILL.md
```
Expected: Zero matches (all should now be `Aphorisms/Workflows`).

**Step 4: Commit**

```bash
git add claudecode/.claude/skills/Aphorisms/SKILL.md
git commit -m "fix(aphorisms): correct case-sensitive workflow paths in SKILL.md

All 4 workflow references used wrong directory case (aphorisms → Aphorisms)
and wrong filename format (kebab-case → TitleCase).
Works on macOS but would fail on Linux."
```

---

### Task 10: Fix case-sensitive SKILL.md paths — BrightData, Sales, USMetrics

**Files:**
- Modify: `claudecode/.claude/skills/BrightData/SKILL.md`
- Modify: `claudecode/.claude/skills/Sales/SKILL.md`
- Modify: `claudecode/.claude/skills/USMetrics/SKILL.md`

**Step 1: Fix BrightData**

Replace `~/.claude/skills/brightdata/Workflows/Four-tier-scrape.md` → `~/.claude/skills/BrightData/Workflows/FourTierScrape.md`

**Step 2: Fix Sales**

Replace all 3 occurrences:
- `Workflows/Create-sales-package.md` → `Workflows/CreateSalesPackage.md`
- `Workflows/Create-narrative.md` → `Workflows/CreateNarrative.md`
- `Workflows/Create-visual.md` → `Workflows/CreateVisual.md`

**Step 3: Fix USMetrics**

Replace:
- `tools/GenerateAnalysis.ts` → `Tools/GenerateAnalysis.ts` (lowercase `tools/` → `Tools/`)
- `update-substrate-metrics.ts` → `UpdateSubstrateMetrics.ts`

**Step 4: Verify all fixes**

Run:
```bash
rg "brightdata/|Create-sales|Create-narrative|Create-visual|tools/Generate|update-substrate" claudecode/.claude/skills/{BrightData,Sales,USMetrics}/SKILL.md
```
Expected: Zero matches.

**Step 5: Commit**

```bash
git add claudecode/.claude/skills/{BrightData,Sales,USMetrics}/SKILL.md
git commit -m "fix(skills): correct case-sensitive paths in BrightData, Sales, USMetrics

All paths used wrong directory/file casing.
Works on macOS (case-insensitive) but would fail on Linux."
```

---

### Task 11: Fix LoadAgentContext.ts case bug

**Files:**
- Modify: `claudecode/.claude/skills/Agents/Tools/LoadAgentContext.ts:29`

**Step 1: Read current line**

Read `claudecode/.claude/skills/Agents/Tools/LoadAgentContext.ts` around line 29.

**Step 2: Fix the path**

Replace: `join(this.claudeHome, "Skills", "Agents")` → `join(this.claudeHome, "skills", "Agents")`

**Step 3: Verify**

Run:
```bash
rg '"Skills"' claudecode/.claude/skills/Agents/Tools/LoadAgentContext.ts
```
Expected: Zero matches.

**Step 4: Commit**

```bash
git add claudecode/.claude/skills/Agents/Tools/LoadAgentContext.ts
git commit -m "fix(agents): correct case-sensitive path in LoadAgentContext.ts

Changed \"Skills\" to \"skills\" to match actual directory name.
Fixes latent cross-platform bug (fails on case-sensitive Linux)."
```

---

### Task 12: Update hooks/README.md

**Files:**
- Modify: `claudecode/.claude/hooks/README.md`

**Step 1: Read current README**

Read `claudecode/.claude/hooks/README.md` to find the Hook Registry table and footer counts.

**Step 2: Apply fixes**

1. Rename `CriteriaTracker.hook.ts` → `AlgorithmTracker.hook.ts` in diagram and table
2. Remove `SessionReactivator.hook.ts` entry (does not exist)
3. Add missing hooks to registry: `ModelRouter.hook.ts`, `VoiceGreeting.hook.ts`, `VoiceGate.hook.ts`, `PRDSync.hook.ts`
4. Update footer counts: `20` → `23` hooks, `10` → `12` libs (after Task 1 deletes 3)

**Step 3: Verify accuracy**

Run:
```bash
ls claudecode/.claude/hooks/*.hook.ts | wc -l
ls claudecode/.claude/hooks/lib/*.ts | wc -l
```
Expected: 23 hooks, 12 libs (after Task 1).

**Step 4: Commit**

```bash
git add claudecode/.claude/hooks/README.md
git commit -m "docs(hooks): update README to match current hook inventory

- Rename CriteriaTracker → AlgorithmTracker
- Remove non-existent SessionReactivator
- Add ModelRouter, VoiceGreeting, VoiceGate, PRDSync
- Update counts: 23 hooks, 12 libs"
```

---

### Task 13: Document SecretScan.ts and RelationshipReflect.ts in TOOLS.md

**Files:**
- Modify: `claudecode/.claude/skills/PAI/TOOLS.md` (or create if it doesn't exist)

**Step 1: Check if TOOLS.md exists**

Run:
```bash
ls claudecode/.claude/skills/PAI/TOOLS.md 2>/dev/null || echo "Does not exist"
```

**Step 2: Add entries for the two tools**

If TOOLS.md exists, add entries. If not, these tools are documented inline in SKILL.md — add a brief reference there instead. Each entry should describe:
- `SecretScan.ts` — TruffleHog wrapper for secret scanning
- `RelationshipReflect.ts` — Periodic relationship growth reflection (writes to OPINIONS.md)

Also verify `SessionProgress.ts` is documented since it's actively surfaced by LoadContext.

**Step 3: Commit**

```bash
git add claudecode/.claude/skills/PAI/TOOLS.md
git commit -m "docs(pai): document SecretScan and RelationshipReflect tools

Both are functional CLI tools that were missing from documentation.
SecretScan wraps TruffleHog. RelationshipReflect writes to OPINIONS.md."
```

---

### Task 14: Update stale config values in settings.json

**Files:**
- Modify: `claudecode/.claude/settings.json`

**Step 1: Read current values**

Find these fields in settings.json:
- `pai.version` (currently `"3.0"`)
- `pai.algorithmVersion` (currently `"3.6.0"`)

**Step 2: Fix version values**

- `pai.version`: Update to `"4.0.3"` (current integrated version)
- `pai.algorithmVersion`: Update to `"3.5.0"` (matches Algorithm/LATEST pointer and CLAUDE.md)

**Step 3: Commit**

```bash
git add claudecode/.claude/settings.json
git commit -m "fix(config): update stale version fields in settings.json

pai.version: 3.0 → 4.0.3 (current integrated version)
pai.algorithmVersion: 3.6.0 → 3.5.0 (matches LATEST pointer)"
```

---

### Task 15: Update README.md makefile → justfile references

**Files:**
- Modify: `README.md` (repo root)

**Step 1: Read README sections referencing make/makefile**

Find references around lines 156, 178, 191-196.

**Step 2: Replace**

- `makefile` → `justfile` in the directory tree listing
- `make` → `just` in the Stow Commands section
- Any `make install`, `make unlink`, etc. → `just link`, `just unlink`, etc.

**Step 3: Verify no stale make references remain**

Run:
```bash
rg "makefile|make " README.md -i
```
Expected: Zero matches (or only references to GNU Make in general context).

**Step 4: Commit**

```bash
git add README.md
git commit -m "docs: update README to reference justfile instead of makefile

Repo uses justfile (not Makefile) for stow commands.
Fixes stale references at lines 156, 178, 191-196."
```

---

### Task 16: Archive completed .atlas-plans

**Files:**
- Modify: `.atlas-plans/iterative-swinging-teapot.md` (set status to COMPLETED)
- Modify: `~/.claude/.atlas-plans/PLAN-hooks.md` (set status to COMPLETED)

**Step 1: Update plan statuses**

For `iterative-swinging-teapot.md`: Change status field from `PLANNED` to `COMPLETED` (stow restructure work was done).

For `PLAN-hooks.md`: Change status from `APPROVED_PENDING_EXECUTION` to `COMPLETED` (hook timeouts are in settings.json).

**Step 2: Commit (repo-local plan only)**

```bash
git add .atlas-plans/iterative-swinging-teapot.md
git commit -m "chore(plans): mark stow restructure plan as completed"
```

Note: `PLAN-hooks.md` is at `~/.claude/.atlas-plans/` (stow target) — update via the stowed source at `claudecode/.claude/.atlas-plans/PLAN-hooks.md` if it exists there.

---

### Task 17: Delete stale .bak file

**Files:**
- Delete: `opencode/.config/opencode/opencode.jsonc.tui-migration.bak`

**Step 1: Verify the main config is stable**

Run:
```bash
ls -la opencode/.config/opencode/opencode.jsonc
```
Expected: File exists and is not empty.

**Step 2: Delete backup**

```bash
rm opencode/.config/opencode/opencode.jsonc.tui-migration.bak
```

**Step 3: Commit**

```bash
git add -u opencode/.config/opencode/
git commit -m "chore(opencode): remove stale tui-migration backup file"
```

---

### Task 18: Restow and final verification

**Step 1: Restow both packages**

```bash
just update
```

**Step 2: Verify no broken symlinks in ~/.claude/**

```bash
find ~/.claude -maxdepth 3 -type l ! -exec test -e {} \; -print 2>/dev/null
```
Expected: Zero broken symlinks (or only known external targets).

**Step 3: Verify settings.json contextFiles still resolve**

```bash
for f in skills/PAI/SKILL.md skills/PAI/AISTEERINGRULES.md skills/PAI/USER/AISTEERINGRULES.md skills/PAI/USER/DAIDENTITY.md; do
  test -f ~/.claude/$f && echo "OK: $f" || echo "BROKEN: $f"
done
```
Expected: All 4 OK.

**Step 4: Verify all 23 hooks still exist**

```bash
ls ~/.claude/hooks/*.hook.ts | wc -l
```
Expected: 23.

**Step 5: Verify PRDFORMAT.md exists (created earlier this session)**

```bash
test -f ~/.claude/skills/PAI/PRDFORMAT.md && echo "OK" || echo "MISSING"
```
Expected: OK.

**Step 6: Final commit (if any stow artifacts changed)**

```bash
git status
```
Only commit if there are meaningful changes from the restow.

---

## Summary

| Task | Type | Files Affected | Risk |
|------|------|----------------|------|
| 1 | Delete orphan hooks/lib | 3 | LOW |
| 2 | Delete orphan lib/migration | 5 | LOW |
| 3 | Delete orphan skills/CORE | ~27 | LOW |
| 4 | Delete orphan banner prototypes | 5 | LOW |
| 5 | Delete orphan PAI CLI tools | 12 | LOW |
| 6 | Delete MEDIUM-confidence orphans | 4 | LOW-MED |
| 7 | Delete dead agent code | 2 | LOW |
| 8 | Delete misc orphans | 3+ | LOW |
| 9 | Fix Aphorisms case paths | 1 | NONE |
| 10 | Fix BrightData/Sales/USMetrics case paths | 3 | NONE |
| 11 | Fix LoadAgentContext.ts case bug | 1 | NONE |
| 12 | Update hooks/README.md | 1 | NONE |
| 13 | Document tools in TOOLS.md | 1 | NONE |
| 14 | Fix stale settings.json versions | 1 | LOW |
| 15 | Fix README make → just refs | 1 | NONE |
| 16 | Archive completed plans | 2 | NONE |
| 17 | Delete stale .bak file | 1 | NONE |
| 18 | Final restow + verification | 0 | NONE |

**Total files deleted:** ~62
**Total files modified:** ~12
**Risk level:** All deletions are red-team validated. No runtime behavior changes.
