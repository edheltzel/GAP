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

### Tier 3 — MEDIUM (committed 2026-04-17)

| # | Item | Status | Notes |
|---|------|--------|-------|
| 5 | MCP Elicitation hook | ✅ | New `hooks/ElicitationHandler.hook.ts` — observe-only, appends JSONL to `MEMORY/LEARNING/mcp-elicitations.jsonl`. Defensive schema typing (field names not yet in official docs). Registered in settings.json. ElicitationResult intentionally NOT wired — out of scope. |
| 8 | `xhigh` effort in ModelRouter | ❌ N/A | Current ModelRouter has no effort mapping — only uses `state.sla` as a display label in the system-reminder. The PAIUpgrade report's EFFORT_MAP suggestion was aspirational. Adding one would require inventing structure. Deferred unless Ed explicitly wants to add it. |
| 12 | `maxResultSizeChars` on BrightData | ❌ N/A | Atlas BrightData is a skill-level workflow (curl to Bright Data API), NOT an MCP server. Annotation doesn't apply. |
| 13 | `refreshInterval` for statusline | ✅ | Added `"refreshInterval": 30` to `statusLine` block in settings.json. Live-updates model/token/cost every 30 seconds. |
| 14 | MCP Elicitation support | ✅ | Duplicate of #5 — handled by the ElicitationHandler. |
| R2 | Parallelization scan in PLAN | ✅ | Added to Algorithm v3.7.0-atlas.md PLAN phase. "Never run 3+ independent operations sequentially when they could run concurrently." |
| R4 | Visual output in DELIVER | ✅ | Added to LEARN phase (no DELIVER phase in current Algorithm). Default-to-visuals for architecture/audit/strategic tasks. |

### Tier 4 — LOW (awareness only — no code changes)

| # | Item | Disposition |
|---|------|-------------|
| 3d | `defer` decision in SecurityValidator | Only relevant for `-p` pipeline use. No current driver. Document only. |
| 6 | MCP structured tool output | Requires MCP server rewrites outside Atlas. Track upstream. |
| 10 | Monitor anthropics/skills repo | Ongoing; not a code change. |
| 11 | `ENABLE_PROMPT_CACHING_1H` | Irrelevant on Max plan. Document if Atlas ever uses API key billing. |

### Infrastructure fixes (bonus — committed 2026-04-17)

| Item | Status | Notes |
|------|--------|-------|
| PAI/SKILL.md malformed frontmatter | ✅ | Root cause: `00-frontmatter.md` placed HTML comment before `---`. Swapped order in the component. Build-timestamp injection in RebuildPAI.ts still works verbatim. Regenerated SKILL.md now has `---` at byte 0. |
| RebuildPAI.ts can't regenerate from worktree | ✅ | Added `PAI_BUILD_ROOT` env var (defaults to `$HOME/.claude/skills/PAI`). Running RebuildPAI.ts with `PAI_BUILD_ROOT=<worktree>/claudecode/.claude/skills/PAI` now reads/writes the worktree. Also added `PAI_CONFIG_DIR` for settings.json path override. |

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
