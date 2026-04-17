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

### Tier 2 — HIGH (next session after Tier 1 reviewed)

| # | Item | Files |
|---|------|-------|
| 3 | PreCompact hook for Algorithm state preservation | new `hooks/PreCompact.hook.ts`, `settings.json` |
| 4 | UserPromptSubmit `sessionTitle` for zero-cost naming | `hooks/SessionAutoName.hook.ts` |
| 7 | PermissionDenied hook for denial tracking | new `hooks/PermissionDenied.hook.ts`, `settings.json` |
| R1 | Verify-before-proceeding pattern in OBSERVE | `skills/PAI/Components/Algorithm/v3.7.0-atlas.md` |
| R3 | Compaction-resilient LOG format | Algorithm LOG spec |

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
