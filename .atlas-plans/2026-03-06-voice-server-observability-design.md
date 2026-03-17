# Voice Server Observability Design

**Date:** 2026-03-06
**Status:** Design complete, awaiting implementation
**Visual:** `~/.agent/diagrams/voice-observability-design.html`
**Topology:** `~/.agent/diagrams/voice-server-topology.html`

## Problem

"Atlas, standing by" plays before/during task completion summaries instead of only at session start. Root cause: VoiceGreeting.hook.ts has 5 silent `catch {}` blocks, session dedup bypassed when `sessionId` is null, and VoiceNotification handler has a 3s abort timeout that fires 100% of the time because the voice server is busy playing prior audio.

## Bug Fix (P1)

### Fix 1: Harden session dedup

**File:** `hooks/VoiceGreeting.hook.ts` (lines 57-87)

**Problem:** When stdin JSON parse fails, `sessionId` stays `null`. The dedup block is wrapped in `if (sessionId)` — meaning no session_id = no dedup = greeting fires every time.

**Fix:** Change `sessionId` from `string | null` to `string` with PID-based fallback. Remove the `if (sessionId)` conditional so dedup always runs.

```typescript
// BEFORE:
let sessionId: string | null = null;
// ...
if (sessionId) { /* dedup */ }

// AFTER:
let sessionId: string = `pid-${process.pid}`;
// ...
// dedup always runs (no conditional)
```

### Fix 2: Increase VoiceNotification abort timeout

**File:** `hooks/handlers/VoiceNotification.ts` (line 104)

**Problem:** 3s timeout is too short. TTS generation (2-5s) + playback (3-6s) means the server won't respond within 3s if it's playing prior audio.

**Fix:** Increase from 3000ms to 12000ms.

## Hook Logging (P2)

### Log format convention

All voice hooks use `console.error` with prefix `[ComponentName]`:

```
[VoiceGreeting] session_id: abc-123
[VoiceGreeting] dedup_check: flag exists, skipping
[VoiceGreeting] speaking: "Atlas, standing by" via /notify
[VoiceGate] block: subagent curl (task_id: xyz)
[VoiceNotification] fetch_ok: 200 in 2340ms
```

### VoiceGreeting: 5 catch blocks to fix

| Line | Context | Change |
|------|---------|--------|
| 65 | stdin parse | Add `console.error` with error message |
| 84 | Flag write | Add `console.error` with error message |
| 165 | Agent file read | Add `console.error` with agent type |
| 199 | Agent voice fetch | Add `console.error` with error |
| 248 | Atlas voice fetch | Add `console.error` with error |

### VoiceGreeting: 7 happy-path trace points

1. After stdin parse: session_id + agent_type
2. Layer 1 skip: loop-worker
3. Layer 2 skip: agents-path
4. Dedup hit: already-greeted
5. Dedup miss: first-greet
6. Before fetch: catchphrase + endpoint
7. After fetch: status + timing

### VoiceGate: 3 log points (currently zero)

- `pass: not-voice-curl`
- `pass: main-session`
- `block: subagent {task_id}`

### New shared helper

**File:** `hooks/lib/voice-logger.ts`

```typescript
export function voiceLog(component: string, action: string, detail?: string) {
  console.error(`[${component}] ${action}${detail ? `: ${detail}` : ''}`);
}
```

## Server-Side Observability (P3)

### Structured request log

**File:** `VoiceServer/voice-requests.jsonl` (new)

Fields: `ts`, `endpoint`, `message`, `voice_id`, `emotion`, `provider_tried`, `provider_used`, `tts_ms`, `playback_ms`, `total_ms`, `status`, `error`, `queue_depth`

### New endpoint: GET /stats

Runtime stats: uptime, request counts, avg timing, provider usage, circuit breaker state.

### Log rotation

On server startup: if `voice-requests.jsonl` > 1MB, rename to `.prev` and start fresh. Same for LaunchAgent log.

## Automatic Cleanup

Flag file cleanup runs automatically — no manual intervention needed:

1. **Voice server startup** (`server.ts`): On launch, scan `MEMORY/STATE/voice-greeted/`, delete files older than 7 days. LaunchAgent restarts the server on crash, so this runs at least once per boot.
2. **Log rotation** (also on startup): If `voice-requests.jsonl` > 1MB, rename to `.prev`. Same for LaunchAgent log.

No `--cleanup` flag needed. Humans shouldn't have to remember to clean up logs.

## Diagnostic CLI (P4)

**File:** `VoiceServer/voice-doctor.ts` (new)

Voice-only diagnostic tool. Single command (`bun voice-doctor.ts`) that checks:
- Server health (port 8888, health endpoint)
- Provider status (all 3)
- Circuit breaker state
- Dedup flag file count + age
- Recent voice events (last 5 from JSONL)
- Log file sizes

**Note:** A broader system health command (`/atlas-doctor` covering memory, voice, observability, context, etc.) is planned as a separate effort after voice server issues are resolved. `voice-doctor.ts` will become one subsystem check within that larger tool.

## File Change Inventory

| File | Priority | Changes |
|------|----------|---------|
| `hooks/VoiceGreeting.hook.ts` | P1 | Harden dedup, 5 catch blocks, 7 trace points |
| `handlers/VoiceNotification.ts` | P1 | Increase timeout to 12s, add timing |
| `hooks/VoiceGate.hook.ts` | P2 | 3 log points |
| `hooks/StopOrchestrator.hook.ts` | P2 | Per-handler timing |
| `hooks/UpdateTabTitle.hook.ts` | P2 | Fetch timing |
| `hooks/lib/voice-logger.ts` | P2 | New shared helper |
| `VoiceServer/server.ts` | P3 | JSONL log, /stats endpoint, log rotation, auto flag cleanup on startup |
| `VoiceServer/voice-doctor.ts` | P4 | New voice-only diagnostic CLI (future: subsystem of broader /atlas-doctor) |

## Implementation Order

1. **Phase 1 (P1):** Bug fix — VoiceGreeting dedup + VoiceNotification timeout (15 min)
2. **Phase 2 (P2):** Hook logging — all catch blocks + trace points + VoiceGate (20 min)
3. **Phase 3 (P3):** Server observability — JSONL + /stats + rotation (30 min)
4. **Phase 4 (P4):** CLI diagnostics — voice-doctor.ts (20 min)

Each phase is independently shippable. Recommend shipping P1+P2 together, P3-P4 separately.
