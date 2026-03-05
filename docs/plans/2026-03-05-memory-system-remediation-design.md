# Memory System Remediation Design

**Date:** 2026-03-05
**Status:** Approved
**Scope:** Full audit remediation (6 CRITICAL, 9 HIGH, 11 MEDIUM, 8 LOW)
**Approach:** B — Surgical fixes + Health monitoring layer
**Repos:** AtlasMemory (`/Users/ed/Developer/AI/AtlasMemory`), atlas-config (`/Users/ed/Developer/atlas-config`)

## Background

A comprehensive audit of both the Recall (LMF3) memory system and PAI native memory system revealed that neither system's advanced features have ever worked. The Recall extraction pipeline has a fatal env var bug preventing all extraction. The PAI WORK/ directory uses a different schema than what the Algorithm documents. Both systems fail silently with no health monitoring.

## Design Decisions

- **META.yaml is canonical** for PAI WORK/ — don't change what already works
- **Investigate and fix MCP loading** for lmf-memory server
- **Add `mem doctor`** health check to prevent future silent failures

---

## Section 1: Recall Fixes (AtlasMemory Repo)

### 1.1 SessionExtract.ts — CLAUDECODE env var fix

**Files:** `hooks/SessionExtract.ts`
**Bug:** `spawn()` at line 1040 and `execSync()` at lines 686, 782 inherit `CLAUDECODE` env var, blocking `claude -p` calls.
**Fix:** Add `env: { ...process.env, CLAUDECODE: '' }` to all three call sites.

### 1.2 SessionExtract.ts — findClaudeCli enhancement

**File:** `hooks/SessionExtract.ts`, line 612
**Fix:** Add `~/.bun/bin/claude` to the candidates list.

### 1.3 Extraction tracker reset

**File:** `~/.claude/MEMORY/.extraction_tracker.json`
**Fix:** Delete or clear to remove 24-hour lockout on attempted files.

### 1.4 `mem doctor` command (new)

**File:** `src/commands/doctor.ts` (new)
**Purpose:** Health check for all memory subsystems.

Checks:
- DB exists and readable
- Messages count > 0, most recent within 48h
- LoA/decisions/learnings/breadcrumbs counts (warn if 0)
- Extraction output files exist (DISTILLED.md, HOT_RECALL.md, SESSION_INDEX.json, DECISIONS.log)
- EXTRACT_LOG.txt last entry is SUCCESS
- .extraction_tracker.json has no stuck lockouts
- Ollama reachable at localhost:11434
- MCP server responds to `tools/list`
- `claude` CLI findable and not blocked by CLAUDECODE
- Unbounded growth report (voice-greeted/, algorithms/, stale WORK dirs)

Output: Colored pass/fail per check, overall health score. Report-only for cleanup items (no auto-delete).

---

## Section 2: PAI Config Fixes (atlas-config Repo)

### 2.1 settings.json — Move SessionExtract to Stop event

**File:** `claudecode/.claude/settings.json`
**Bug:** SessionExtract registered under `UserPromptSubmit` (line 381-389) instead of `Stop`.
**Fix:** Move the hook group to the `Stop` section, after StopOrchestrator.

### 2.2 PRDSync.hook.ts — Support META.yaml format

**File:** `claudecode/.claude/hooks/PRDSync.hook.ts`
**Bug:** Only triggers on `PRD.md` files. 99% of WORK dirs use `META.yaml`.
**Fix:** Expand trigger condition to also fire on `META.yaml` in `MEMORY/WORK/`. Parse META.yaml fields, map to work.json schema. Keep existing PRD.md support.

### 2.3 CLAUDE.md — Fix phantom tool instructions

**File:** `claudecode/.claude/CLAUDE.md`
**Bug:** Documents 7 MCP tools as available + "MANDATORY" LAW that's impossible to follow.
**Fix:**
- Add conditional note about MCP server availability
- Downgrade `context_for_agent` from MANDATORY to "recommended when available"
- Add `mem` CLI fallback instructions

### 2.4 LoadContext.hook.ts — Wire dead learning readback code

**File:** `claudecode/.claude/hooks/LoadContext.hook.ts`
**Bug:** `loadWisdomFrames()` and `loadFailurePatterns()` from `learning-readback.ts` exist but are never called.
**Fix:** Add calls to these functions in the appropriate section of LoadContext.

### 2.5 active-sessions.json cache invalidation fix

**File:** `claudecode/.claude/hooks/LoadContext.hook.ts`
**Bug:** Cache invalidation depends on `current-work.json` mtime, which doesn't exist, so cache is always "valid" (stale sessions persist).
**Fix:** Use WORK directory mtime as invalidation signal instead of current-work.json.

### 2.6 prd-utils.ts — Fix `\Z` regex

**File:** `claudecode/.claude/hooks/lib/prd-utils.ts`, lines 61, 76
**Bug:** `\Z` is invalid in JavaScript regex (Perl/Python anchor).
**Fix:** Replace `\Z` with `$`.

### 2.7 prd-utils.ts — Duplicate frontmatter prevention

**File:** `claudecode/.claude/hooks/lib/prd-utils.ts`
**Bug:** `writeFrontmatterField()` can append duplicate fields.
**Fix:** Add dedup guard — if field exists, update in place; don't append.

---

## Section 3: MCP Investigation + Remaining Fixes

### 3.1 MCP Server Loading Investigation

**Bug:** `lmf-memory` MCP tools registered in `.mcp.json` but not loading in Claude Code sessions.

Investigation steps:
1. Check `mem-mcp` MCP protocol version and transport (stdio)
2. Test with `claude mcp list`
3. Check for missing `--stdio` flag or transport config
4. Verify binary path resolution from Claude Code context
5. Check Claude Code logs for MCP init errors

Fix depends on findings.

### 3.2 LEARNING/ Directory Population

**Bug:** All LEARNING/ subdirs empty. Hooks that should populate them may have format mismatches.

Fix:
- Verify WorkCompletionLearning, AlgorithmTracker, RatingCapture hooks write to correct paths
- Ensure output format matches what `learning-readback.ts` expects (`**Feedback:**` pattern vs JSONL)
- Align hook output format with readback expectations

### 3.3 learning-cache.sh — Add missing trend fields

**Bug:** `trend` and `day_trend` fields absent from learning-cache.sh, causing trend reporting to default to "stable".
**Fix:** Update the hook that generates this file to include these fields.

---

## File Change Summary

### AtlasMemory repo (`/Users/ed/Developer/AI/AtlasMemory`)

| File | Change Type | Description |
|------|-------------|-------------|
| `hooks/SessionExtract.ts` | Edit | Add CLAUDECODE env override (3 locations) + bun CLI path |
| `src/commands/doctor.ts` | New | Health check command |
| `src/index.ts` | Edit | Register doctor command |

### atlas-config repo (`/Users/ed/Developer/atlas-config`)

| File | Change Type | Description |
|------|-------------|-------------|
| `claudecode/.claude/settings.json` | Edit | Move SessionExtract from UserPromptSubmit to Stop |
| `claudecode/.claude/hooks/PRDSync.hook.ts` | Edit | Support META.yaml trigger |
| `claudecode/.claude/CLAUDE.md` | Edit | Fix phantom tools, add CLI fallback |
| `claudecode/.claude/hooks/LoadContext.hook.ts` | Edit | Wire learning readback + fix cache invalidation |
| `claudecode/.claude/hooks/lib/prd-utils.ts` | Edit | Fix \Z regex + dedup guard |

### Runtime files (not in git)

| File | Action | Description |
|------|--------|-------------|
| `~/.claude/MEMORY/.extraction_tracker.json` | Delete | Clear lockout |
| `~/.claude/.mcp.json` | Edit (if needed) | Fix MCP config based on investigation |

---

## Testing Strategy

1. **SessionExtract:** Run `echo '{"cwd":"/Users/ed/Developer/atlas-config"}' | CLAUDECODE=1 bun hooks/SessionExtract.ts` and verify EXTRACT_LOG.txt shows SUCCESS
2. **MCP:** Run `claude mcp list` and verify lmf-memory appears with all 7 tools
3. **PRDSync:** Edit a META.yaml in WORK/ and verify work.json is created/updated
4. **mem doctor:** Run `mem doctor` and verify all checks pass
5. **Learning readback:** Start a new session and verify wisdom frames + failure patterns appear in context
6. **End-to-end:** Complete a full session, verify SessionExtract produces DISTILLED.md + HOT_RECALL.md
