# PAI Upgrade Execution Plan — 2026-04-17

**Source:** `/PAIUpgrade` report run 2026-04-17
**Branch:** `worktree-pai-upgrade-2026-04-17`
**CC version:** 2.1.113 (≥2.1.110 required for item #2)

## Execution Order

### Tier 1 — CRITICAL (in-flight this session)

| # | Item | Status | Approach |
|---|------|--------|----------|
| 2 | PreToolUse `additionalContext` drop-on-failure bug | ✅ Already fixed upstream | CC 2.1.113 includes v2.1.110 fix. No code change. |
| 1 | InstructionsLoaded.hook.ts confirmed schema | 🚧 Parallel agent | Replace defensive fallback with typed `InstructionsLoadedInput` interface. Preserve existing behavior while tightening field access. |
| 9 | SKILL.md front-matter audit | 🚧 Parallel agent | Sweep `claudecode/.claude/skills/*/SKILL.md`, add missing `name`/`description`/`version` YAML frontmatter. Use claude-api skill as reference. |

### Tier 2 — HIGH (committed 2026-04-17)

| # | Item | Status | Notes |
|---|------|--------|-------|
| 3 | PreCompact hook | ✅ | New `hooks/PreCompact.hook.ts` — reads MEMORY/WORK/<slug>/LOG.md tail (~3000 chars), emits via `hookSpecificOutput.additionalContext`. Never blocks. Registered in settings.json. Verified happy-path + graceful-degradation. |
| 4 | UserPromptSubmit `sessionTitle` | ✅ | Additive edit to `SessionAutoName.hook.ts` — emits sessionTitle in customTitle-sync branch + storeName path. Haiku inference preserved. TS null-narrowing fix included. |
| 7 | PermissionDenied hook | ✅ | New `hooks/PermissionDenied.hook.ts` — appends JSONL to `MEMORY/LEARNING/permission-denials.jsonl`. Observe-only (no retry logic yet). Registered alphabetically between InstructionsLoaded and PermissionRequest. |
| R1 | Verify-before-proceeding | ✅ | Expanded existing `VERIFY LOAD-BEARING ASSUMPTIONS` bullet in OBSERVE (v3.7.0-atlas.md line 238) with "before leaving OBSERVE", rationale, examples. |
| R3 | Compaction-resilient LOG format | ✅ | Added `### LOG Compaction Checkpoints` subsection with structured checkpoint template (Verified implemented / Remaining / Current PRD state / Active assumptions). |

**POST-MERGE STEP:** `RebuildPAI.ts` hardcodes `$HOME/.claude/skills/PAI/` paths, so running it from a worktree writes to the stow target, not the worktree. After merging this branch to main and restowing, run `bun ~/.claude/skills/PAI/Tools/RebuildPAI.ts` to regenerate `SKILL.md` from the updated v3.7.0-atlas.md. The component change is live; only the generated SKILL.md is stale until regenerated.

### Tier 3 — MEDIUM (backlog)

| # | Item | Files |
|---|------|-------|
| 5 | MCP Elicitation + ElicitationResult hooks | new hook + BrightData/chrome-devtools |
| 8 | `xhigh` effort tier in ModelRouter | `hooks/ModelRouter.hook.ts` |
| 12 | `anthropic/maxResultSizeChars` on BrightData MCP | `.mcp.json` |
| 13 | `refreshInterval` for atlas-statusline | `settings.json` |
| 14 | MCP Elicitation support | duplicate of 5? — verify |
| R4 | Visual output in DELIVER phase | Algorithm spec |

### Tier 4 — LOW (awareness / future)

| # | Item | Notes |
|---|------|-------|
| 3d | `defer` decision in SecurityValidator | Only relevant for `-p` pipelines |
| 6 | MCP structured tool output | Requires MCP server rewrites |
| 10 | Monitor anthropics/skills repo | Ongoing |
| 11 | `ENABLE_PROMPT_CACHING_1H` | Irrelevant on Max plan |
| R2 | Parallelization scanning in PLAN | Algorithm spec |

## Rollback

Worktree-based. `ExitWorktree action=remove discard_changes=true` reverts everything.
