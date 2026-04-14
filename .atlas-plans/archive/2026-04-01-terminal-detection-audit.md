# Terminal Detection Audit

**Date:** 2026-04-01
**Triggered by:** Voice summaries silently failing after switch to Ghostty
**Root cause:** Hardcoded terminal allowlist in `StopOrchestrator.hook.ts`

---

## Fix Applied

| File | Before | After | Status |
|------|--------|-------|--------|
| `hooks/StopOrchestrator.hook.ts:42-44` | Allowlist: `['WezTerm', 'kitty', 'iTerm2', ...]` | `!!process.env.TERM_PROGRAM` | **FIXED** |

---

## Full Audit: All Terminal-Specific Code

### 1. StopOrchestrator.hook.ts (voice gate) — FIXED

- **Function:** `isMainSession()` at line 42
- **Was:** Positive allowlist of 6 terminal names
- **Now:** `!!process.env.TERM_PROGRAM` — any terminal = real session
- **Risk eliminated:** Silent voice failure on terminal switch

### 2. VoiceGate.hook.ts (PreToolUse bash blocker) — CLEAN

- **Function:** Blocks subagent voice curls
- **Detection:** Uses `agent_id` from stdin JSON, NOT terminal env vars
- **Verdict:** Terminal-agnostic. No action needed.

### 3. VoiceGreeting.hook.ts (SessionStart greeting) — CLEAN

- **Function:** Speaks catchphrase at startup
- **Detection:** Uses layered checks: `PAI_SUPPRESS_VOICE`, `CLAUDE_CODE_AGENT_TASK_ID`, `CLAUDE_AGENT_TYPE`, `CLAUDE_PROJECT_DIR`, stdin `source` field
- **Verdict:** No terminal allowlist. Works on any terminal. No action needed.

### 4. tab-setter.ts (Kitty tab control) — KITTY-SPECIFIC BY DESIGN

- **Function:** Sets tab titles, colors, and phase indicators via `kitten @` commands
- **Detection:** `process.env.TERM === 'xterm-kitty' || kittyEnv.listenOn`
- **Verdict:** Intentionally Kitty-only — uses Kitty's remote control protocol (`kitten @`). These commands are Kitty API calls, not generic terminal ops. Correctly no-ops on non-Kitty terminals. **No false-negative risk** — it just doesn't run, which is correct behavior.
- **Note:** Tab titles/colors are purely cosmetic. Silent no-op on Ghostty is fine.

### 5. StartupGreeting.hook.ts (Kitty env persistence) — KITTY-SPECIFIC BY DESIGN

- **Function:** Persists `KITTY_LISTEN_ON` and `KITTY_WINDOW_ID` for later hook lookups
- **Detection:** Checks `KITTY_LISTEN_ON` && `KITTY_WINDOW_ID` env vars
- **Verdict:** Only persists if running in Kitty. Correct no-op elsewhere.

### 6. SessionSummary.hook.ts (session cleanup) — CLEAN

- **Function:** Marks work complete, cleans up kitty session files
- **Detection:** Calls `cleanupKittySession()` which just deletes session JSON if it exists
- **Verdict:** No terminal detection — just cleans up files that may or may not exist.

### 7. Banner.ts / pai.ts (terminal width detection) — DEGRADED BUT HARMLESS

- **Function:** Detects terminal width for banner rendering
- **Methods:** WezTerm UNIX socket query, `KITTY_WINDOW_ID` + `kitten @ ls`
- **Fallback:** Falls back to `process.stdout.columns` (POSIX standard)
- **Verdict:** Won't get pixel-precise width on Ghostty but falls back gracefully. No action needed.

### 8. Observability server (kitty @ ls) — KITTY-SPECIFIC BY DESIGN

- **Function:** Gets tab/window layout for the observability dashboard
- **Verdict:** Intentionally Kitty-only feature. No-ops elsewhere.

---

## Summary

| Category | Count | Status |
|----------|-------|--------|
| **Broken (was silently failing)** | 1 | FIXED (StopOrchestrator) |
| **Kitty-specific by design** | 3 | Correct no-op on other terminals |
| **Terminal-agnostic** | 3 | Clean, no issues |
| **Degraded but harmless** | 1 | Falls back to POSIX standard |

### Key Insight

The only problematic pattern was the **positive terminal allowlist** — a detection strategy that breaks on new terminals. All other terminal-specific code either:
- Uses the correct terminal's own env vars (Kitty → `KITTY_LISTEN_ON`) — no false positives possible
- Uses Claude Code's agent metadata (`agent_id`, `source`) — terminal-independent
- Falls back gracefully when the expected terminal isn't present

### Recommendation

No further changes needed. The one allowlist is fixed. All other terminal detection is correctly scoped to Kitty-specific features that should only run on Kitty.
