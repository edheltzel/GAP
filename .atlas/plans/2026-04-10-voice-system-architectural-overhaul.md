---
title: Voice System Architectural Overhaul
status: Not Started
percent_complete: 0
last_updated: 2026-04-19
phase: Planning — awaiting approval and open questions
blockers: 8 open questions for Ed before implementation begins (see Section 9)
next_action: get Ed's answers to open questions in Section 9, then begin Phase 0 — worktree + voice/ package skeleton
---

# Voice System Architectural Overhaul

> Complete architectural overhaul of the atlas-config voice infrastructure. Not a feature rollout — a deliberate restructure that fixes coupling, unifies config, eliminates dead code, and makes voice truly agent-agnostic within atlas-config.

**Status:** `PLANNED` — awaiting approval
**Date:** 2026-04-10
**Author:** Atlas + Ed
**Supersedes:** `2026-04-06-voiceserver-standalone-extraction.md` (defer repo extraction), `2026-03-15-voice-notify-cli.md` (reject CLI binary approach)
**Replaces:** `2026-03-06-voice-server-observability-design.md` (P1/P2 already applied; P3/P4 absorbed here)

---

## 1. Executive Summary

### 1.1 The Problem (Not What You Think)

The voice system is not broken. EdgeTTS works, ElevenLabs works, the server is stable. The real problems are **structural**:

1. **Tight coupling.** 16+ files hardcode `localhost:8888`. 5 hooks read `settings.json` directly instead of going through `identity.ts`. Voice identity is dual-sourced (settings.json AND voices.json, with different schemas).
2. **Physical location.** VoiceServer lives inside `claudecode/.claude/VoiceServer/` — deployed via Claude Code's stow package. OpenCode reaches into that path to auto-start the server. This is wrong.
3. **Fragile phase tracking.** `AlgorithmTracker.detectPhaseFromBash()` reads voice curl commands out of Bash tool inputs. Any voice call that bypasses Bash (e.g., a hook's `fetch()`) is invisible to phase tracking.
4. **Silent bugs.** `similarityBoost` in settings.json vs `similarity_boost` in the payload — the voice settings are never actually passed through. `volume` in agent frontmatter is parsed and dropped. The `/notify/personality` endpoint discards the personality and calls the normal handler with `null` voice. The README documents a 3-provider chain that hasn't existed for months.
5. **No observability.** No request log. No /stats endpoint. No voice-doctor. When voice breaks, you `tail` the launchd log and hope.

### 1.2 The Wrong Answers

Two prior plans attempted to fix this by swinging the pendulum:

- **Plan A — Standalone repo extraction** (`2026-04-06`): Extract VoiceServer to its own git repo. Adds a second repo to maintain, a second deployment pipeline, a second README. Solves a distribution problem that doesn't exist yet (voice has no consumers outside atlas-config).
- **Plan B — CLI binary replacement** (`2026-03-15`): Compile the server into a Bun binary. Drops HTTP, circuit breakers, rate limiter. Trades one daemon's queue for N processes fighting a flock mutex. Includes a `--serve` shim that literally re-adds the daemon (defeating the point). Pays Bun cold-start on every invocation.

Both approaches treat "HTTP vs CLI" or "bundled vs extracted" as the problem. The actual problem is **coupling**, not the transport or the repo boundary.

### 1.3 The Right Answer — "Right-Sized Restructure"

**Extract the voice system into a dedicated stow package within atlas-config, unify the configuration, refactor every consumer through a shared client library, and fix the bugs we found along the way.**

This is the minimum change that addresses all the real problems. No second repo. No second paradigm. No throwing away working defensive infrastructure.

| Decision | Why |
|----------|-----|
| **New `voice/` stow package in atlas-config** | True decoupling from claudecode. Stows to `~/.local/share/atlas-voice/` (XDG). Voice is no longer a Claude Code subdirectory. |
| **Keep HTTP + daemon + LaunchAgent** | It works. Circuit breakers, rate limiter, serialized audio queue — these are defensive measures that prevent real failure modes. |
| **Single config at `~/.config/atlas-voice/config.json`** | Eliminates the dual identity store (settings.json + voices.json). Canonical snake_case field names. Hot-reloadable. |
| **Shared client library** | Every consumer imports one module that handles URL, identity, retry, and error. Zero hardcoded URLs after this phase. |
| **Phase tracking via voice client, not Bash sniffing** | AlgorithmTracker reads a phase state file written by the voice client. All voice calls are now visible to phase tracking. |
| **Defer separate repo** | If voice ever needs to leave atlas-config, the stow package structure makes extraction a future `git filter-branch`. |
| **Reject CLI binary** | No real simplification — `--serve` shim re-adds the daemon. Cold start cost per call. flock mutex is new unbattle-tested code. |

---

## 2. Current State — Evidence Summary

Gathered from four parallel exploration agents and one red team validator.

### 2.1 What Actually Exists

**VoiceServer core** (`claudecode/.claude/VoiceServer/server.ts`, **1,277 lines**):

- **4 TTS providers**: EdgeTTS (primary), ElevenLabs, Kokoro, macOS say. Circuit breakers on all except `say`.
- **HTTP endpoints**: `POST /notify`, `POST /notify/personality` (dead shim), `POST /pai`, `GET /health`, `POST /shutdown`, fallthrough catch-all.
- **Rate limiter**: 10 req/60s per IP, but all localhost callers share one bucket because `x-forwarded-for` is never set on direct calls.
- **Request pipeline**: rate-limit → sanitize → strip markers → extract emotion → voice lookup → provider fallback → pronunciation → TTS → audio → notification.
- **Config**: `voices.json` (244 lines, 14 agents, 4 providers), `pronunciations.json` (4 rules, one of which is a literal `{PRINCIPAL.LAST_NAME}` template stub).
- **Hardcoded paths**: `/opt/homebrew/bin/python3`, `/usr/bin/afplay`, `/usr/bin/say`, `/usr/bin/osascript`, `/bin/rm`, `/tmp/voice-*`, `http://127.0.0.1:8880/v1` (Kokoro), `~/.claude/.env` (the only real coupling point).
- **LaunchAgent**: `com.pai.voice-server`, logs to `~/Library/Logs/pai-voice-server.log`.

**Identity resolution chain** (dual-store, messy):

```
settings.json (daidentity section)          voices.json (server identity)
  ├─ name: "Atlas"                            ├─ identity.elevenlabs.voice_id
  ├─ voices.main.voiceId                      ├─ identity.elevenlabs.stability
  ├─ voices.main.similarityBoost  ⚠️          ├─ identity.elevenlabs.similarity_boost ⚠️
  ├─ voices.main.stability                    └─ agents.{14 entries}
  ├─ voices.algorithm.voiceId
  ├─ personality.baseVoice = "" (disables)
  └─ startupCatchphrases[]
```

**⚠️ Bug**: `settings.json` uses `similarityBoost` (camelCase), `VoiceProsody` interface and payload use `similarity_boost` (snake_case). The cast at `identity.ts:120` performs no key translation. Result: `similarity_boost` is **always** falling back to the hardcoded `0.75` default — user settings are silently ignored.

**Voice consumers** (red-team-validated complete list — 16+ files):

| # | File | Purpose |
|---|------|---------|
| 1 | `claudecode/.claude/hooks/handlers/VoiceNotification.ts` | Completion voice (Stop event) |
| 2 | `claudecode/.claude/hooks/VoiceGreeting.hook.ts` | Startup catchphrase (reads settings.json directly) |
| 3 | `claudecode/.claude/hooks/UpdateTabTitle.hook.ts` | Tab title voice |
| 4 | `claudecode/.claude/hooks/AlgorithmTracker.hook.ts` | Phase tracking (reads + posts) |
| 5 | `claudecode/.claude/hooks/VoiceGate.hook.ts` | Subagent voice blocker |
| 6 | `claudecode/.claude/skills/PAI/Tools/pai.ts` | `pai launch` voice |
| 7 | `opencode/.config/opencode/skills/PAI/Tools/pai.ts` | Mirror of above |
| 8 | `claudecode/.claude/PAI-Install/engine/actions.ts` | Installer health/test |
| 9 | `claudecode/.claude/VoiceServer/menubar/pai-voice.5s.sh` | SwiftBar status widget |
| 10 | `claudecode/.claude/VoiceServer/install.sh` | Post-install smoke test |
| 11 | `claudecode/.claude/VoiceServer/start.sh` | Health check wait loop |
| 12 | `claudecode/scripts/lib/agent-converter.ts` | Strips curl blocks (regex) |
| 13 | `claudecode/scripts/lib/skill-converter.ts` | Strips curl blocks (regex) |
| 14 | `claudecode/scripts/convert-pai-to-oc.ts` | 3x replace() calls |
| 15 | `opencode/.config/opencode/plugins-source/voice-notification.ts` | OpenCode voice plugin |
| 16 | `opencode/.config/opencode/plugins-source/algorithm-tracker.ts` | OpenCode phase tracker |
| 17 | `opencode/.config/opencode/skills/PAI/Tools/algorithm.ts` | OpenCode algorithm runner |

All 17 hardcode `localhost:8888` or reference it in regex. None go through a shared abstraction.

### 2.2 Validated Bugs and Surprises

| # | Finding | Source | Impact |
|---|---------|--------|--------|
| 1 | `similarityBoost` / `similarity_boost` mismatch | `identity.ts:120`, `settings.json:976,985` | User voice settings are silently ignored, always use 0.75 default |
| 2 | `/notify/personality` is a dead shim | `server.ts:1151-1181` | Discards personality, calls `sendNotification(..., null)`. Dead code path. |
| 3 | No agent-specific EdgeTTS voice mapping | `voices.json` | All 14 agents sound identical on EdgeTTS (`en-US-AvaNeural`) |
| 4 | `volume` field parsed but not sent | `VoiceGreeting.hook.ts:123,208-213` | Agent frontmatter `volume: 0.85` is ignored |
| 5 | AlgorithmTracker invisible to non-Bash voice | `AlgorithmTracker.hook.ts:47` | The hook's own `fetch()` calls don't trigger phase tracking |
| 6 | Rate limiter single-bucket for localhost | `server.ts:1086` | All local callers share 10 req/60s — can throttle legitimate traffic |
| 7 | README documents 3-provider chain | `VoiceServer/README.md:44-56,138-144,162-199` | EdgeTTS omitted entirely from published docs |
| 8 | Menubar widget unaware of EdgeTTS | `menubar/pai-voice.5s.sh` | Always shows "ElevenLabs AI" or "macOS Say" |
| 9 | server.ts hardcodes `~/.claude/.env` | `server.ts:144` | Only atlas-config coupling in server code |
| 10 | Pronunciation rule `{PRINCIPAL.LAST_NAME}` is a literal template stub | `pronunciations.json` | Never matches anything |

---

## 3. Target Architecture

### 3.1 Directory Structure

```
atlas-config/
├── voice/                                      ← NEW stow package
│   ├── .config/
│   │   └── atlas-voice/
│   │       ├── config.json                     ← unified voice config (tracked)
│   │       ├── config.example.json             ← template for new installs
│   │       └── pronunciations.json             ← moved from VoiceServer/
│   ├── .local/
│   │   ├── share/
│   │   │   └── atlas-voice/
│   │   │       ├── server.ts                   ← moved from VoiceServer/
│   │   │       ├── client/                     ← NEW shared TS client
│   │   │       │   ├── index.ts                ← voiceNotify(), voiceHealth(), getConfig()
│   │   │       │   ├── phase.ts                ← writePhaseState() for AlgorithmTracker
│   │   │       │   └── types.ts                ← VoiceNotifyRequest, VoiceConfig, etc.
│   │   │       ├── voice-doctor.ts             ← health check CLI
│   │   │       ├── install.sh                  ← rewritten for new paths + LaunchAgent name
│   │   │       ├── uninstall.sh
│   │   │       ├── start.sh
│   │   │       ├── stop.sh
│   │   │       ├── restart.sh
│   │   │       ├── status.sh
│   │   │       └── menubar/
│   │   │           ├── atlas-voice.5s.sh       ← renamed from pai-voice.5s.sh
│   │   │           └── install-menubar.sh
│   │   └── bin/
│   │       └── voice-doctor                    ← stowed symlink to ../share/atlas-voice/voice-doctor.ts
│   ├── .stow-local-ignore
│   └── README.md                               ← standalone README for the package
│
├── claudecode/
│   └── .claude/
│       ├── hooks/
│       │   ├── VoiceGreeting.hook.ts           ← refactored to import voice client
│       │   ├── VoiceGate.hook.ts               ← pattern expanded for new client
│       │   ├── UpdateTabTitle.hook.ts          ← refactored
│       │   ├── AlgorithmTracker.hook.ts        ← reads phase state file, not Bash
│       │   └── handlers/
│       │       └── VoiceNotification.ts        ← refactored
│       ├── VoiceServer/                        ← REMOVED
│       └── settings.json                       ← daidentity.voices moved to atlas-voice config
│
├── opencode/
│   └── .config/
│       └── opencode/
│           ├── plugins-source/
│           │   ├── voice-notification.ts       ← refactored to import voice client
│           │   └── algorithm-tracker.ts        ← reads phase state file
│           └── skills/PAI/Tools/
│               ├── pai.ts                      ← refactored
│               └── algorithm.ts                ← refactored
│
└── justfile                                    ← pkgs now includes "voice"
```

**Stow targets** (from `voice/` package):

| Source | Target |
|--------|--------|
| `voice/.config/atlas-voice/*` | `~/.config/atlas-voice/*` |
| `voice/.local/share/atlas-voice/*` | `~/.local/share/atlas-voice/*` |
| `voice/.local/bin/voice-doctor` | `~/.local/bin/voice-doctor` |

`~/.local/share/atlas-voice/` becomes the canonical runtime directory. `~/.config/atlas-voice/` is the canonical config directory. Both are XDG Base Directory compliant.

### 3.2 Unified Config Schema

**`~/.config/atlas-voice/config.json`** (single source of truth):

```jsonc
{
  "$schema": "https://atlas-voice.example.com/schema/v1.json",
  "version": "1.0.0",
  "server": {
    "url": "http://localhost:8888",
    "port": 8888,
    "bind": "127.0.0.1",
    "log_path": "~/Library/Logs/atlas-voice.log",
    "request_log_path": "~/.local/share/atlas-voice/logs/requests.jsonl",
    "rate_limit": { "requests": 10, "window_seconds": 60 }
  },
  "providers": {
    "edgetts": {
      "enabled": true,
      "defaultVoice": "en-US-AvaNeural",
      "rate": "+0%",
      "python_path": "auto"           // "auto" = which python3, or explicit path
    },
    "elevenlabs": {
      "enabled": false,
      "api_key_env": "ELEVENLABS_API_KEY",
      "defaultVoiceId": "s3TPKV1kjDlVtZbl4Ksh",
      "model": "eleven_turbo_v2_5"
    },
    "kokoro": {
      "enabled": false,
      "endpoint": "http://127.0.0.1:8880/v1",
      "defaultVoice": "af_sky"
    },
    "say": {
      "enabled": true,
      "voice": "Daniel (Enhanced)",
      "rate_wpm": 175
    }
  },
  "defaultProvider": "edgetts",
  "fallbackOrder": ["edgetts", "elevenlabs", "kokoro", "say"],
  "default_volume": 0.8,
  "identity": {
    "name": "Atlas",
    "displayName": "ATLAS",
    "catchphrases": [
      "Atlas, standing by",
      "Atlas online",
      "Ready when you are"
    ],
    "voices": {
      "main": {
        "elevenlabs": {
          "voice_id": "21m00Tcm4TlvDq8ikWAM",
          "stability": 0.35,
          "similarity_boost": 0.8,     // ✓ canonical snake_case
          "style": 0.9,
          "speed": 1.1,
          "use_speaker_boost": true
        },
        "kokoro": {
          "voice": "af_heart",
          "speed": 1.1
        }
      },
      "algorithm": { /* same shape as main */ }
    }
  },
  "agents": {
    "engineer": {
      "persona": { "name": "Marcus Webb", "title": "The Battle-Scarred Leader" },
      "elevenlabs": {
        "voice_id": "Fahco4VZzobUeiPqni1S",
        "stability": 0.62,
        "similarity_boost": 0.80,
        "style": 0.08,
        "speed": 0.98,
        "use_speaker_boost": true,
        "volume": 0.85                  // ✓ now honored
      },
      "kokoro": { "voice": "am_michael", "speed": 1.0 },
      "edgetts": { "voice": "en-US-GuyNeural" }  // ✓ NEW: agent-specific EdgeTTS
    }
    // ... 13 more agents
  }
}
```

**Key schema decisions:**

1. **Canonical snake_case everywhere.** `similarity_boost`, `use_speaker_boost`, `voice_id`. No more camelCase.
2. **Per-provider voice settings per agent.** Each agent can specify distinct ElevenLabs, Kokoro, AND EdgeTTS voices. Fixes the "all agents sound the same on EdgeTTS" problem.
3. **`volume` is now honored.** Explicit field, passed through to afplay.
4. **Server URL is configurable.** Every consumer reads from `server.url`.
5. **Python path is "auto" by default.** Uses `which python3` instead of hardcoding `/opt/homebrew/bin/python3`.

**Schema migration:** A one-time migration script reads `daidentity` from `settings.json` and `voices.json`, merges them into `config.json`, translates camelCase → snake_case, and leaves `settings.json` with only non-voice identity fields (`name`, `displayName`, `color`, etc.). `daidentity.voices.*` is removed from `settings.json`.

### 3.3 The Voice Client Library

**`~/.local/share/atlas-voice/client/index.ts`** — imported by every consumer:

```typescript
// Public API
export async function voiceNotify(opts: VoiceNotifyOptions): Promise<VoiceResult>
export async function voiceHealth(): Promise<HealthStatus>
export function getVoiceConfig(): VoiceConfig          // cached, hot-reloaded
export function getVoiceUrl(): string                   // resolved from config.server.url
export function getIdentity(): VoiceIdentity            // replaces hooks/lib/identity.ts

// Phase tracking API (used by AlgorithmTracker)
export function writePhaseState(phase: AlgorithmPhase): void
export function readPhaseState(): PhaseState | null

// Subagent gate
export function isSubagentContext(): boolean            // reads stdin JSON agent_id
```

**`VoiceNotifyOptions`:**

```typescript
interface VoiceNotifyOptions {
  message: string;
  agent?: string;              // "engineer", "intern", etc. — looks up agents[key]
  voiceId?: string;            // explicit ElevenLabs voice_id override
  voiceSettings?: VoiceSettings; // explicit override (rare)
  title?: string;
  source: 'claude-code' | 'opencode' | 'pi' | 'cli';  // REQUIRED — for per-source rate limiting
  phase?: AlgorithmPhase;      // if set, voiceNotify() also writes phase state
  emotion?: string;            // [💡 insight] style marker
  volume?: number;             // 0.0–1.0, overrides config default
}
```

**Behavior:**

1. Reads `config.json` (cached, watches for changes)
2. Resolves voice from `agent` name OR `voiceId` OR default identity
3. Builds POST body with correct snake_case fields
4. POSTs to `config.server.url + "/notify"` with `source` header
5. 12s timeout, graceful error handling, returns `VoiceResult`
6. If `phase` set, writes phase state file for AlgorithmTracker
7. If `isSubagentContext()` returns true → no-op (VoiceGate replacement)

**Every consumer becomes ~3-5 lines:**

```typescript
// Before (VoiceNotification.ts, ~50 lines of fetch + error handling)
const resp = await fetch('http://localhost:8888/notify', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify(body),
  signal: controller.signal,
});
// ...

// After (one import, one call)
import { voiceNotify } from '@atlas/voice-client';
await voiceNotify({
  message: voiceCompletion,
  source: 'claude-code',
  title: `${identity.name} says`,
});
```

### 3.4 Phase Tracking — No More Bash Sniffing

**Current approach** (`AlgorithmTracker.hook.ts:47`):
```typescript
// Only fires if the AI's Bash command contains localhost:8888/notify
if (!command.includes('localhost:8888') || !command.includes('/notify')) {
  return { phase: null, isAlgorithmEntry: false };
}
```

**Problem:** Invisible to non-Bash voice calls. The hook's own `fetch()` at lines 191-198 doesn't trigger phase tracking. The OpenCode plugin's voice calls don't trigger phase tracking. Any future client that speaks without shelling out is invisible.

**New approach — Phase State File:**

```
~/.local/share/atlas-voice/state/phase.json

{
  "current_phase": "BUILD",
  "entered_at": "2026-04-10T12:30:00Z",
  "cycle_id": "cycle-42",
  "history": [
    { "phase": "OBSERVE", "entered_at": "...", "duration_ms": 1250 },
    { "phase": "THINK", "entered_at": "...", "duration_ms": 3400 },
    ...
  ]
}
```

**Flow:**

1. When the AI calls `voiceNotify({ ..., phase: "BUILD" })`, the client library writes the phase to `phase.json`.
2. The client detects transitions (OBSERVE → COMPLETE → OBSERVE = rework cycle) and fires a notification automatically.
3. `AlgorithmTracker.hook.ts` reads `phase.json` at specific lifecycle points instead of parsing Bash commands.
4. Voice calls from ANY source (hook, opencode plugin, CLI, curl) contribute to phase tracking.

**Transitional compatibility:** During rollout, AlgorithmTracker reads BOTH the phase state file AND the old Bash pattern. Once all consumers migrate to the client, the Bash sniffing code is deleted.

---

## 4. Implementation Phases

Every phase happens in a worktree. Final merge is atomic.

### Phase 0 — Worktree + Planning Scaffolding

| # | Task | Status |
|---|------|--------|
| 0.1 | `EnterWorktree` to create `feat/voice-overhaul` branch | PENDING |
| 0.2 | Create `voice/` package skeleton with `.stow-local-ignore` | PENDING |
| 0.3 | Add `voice` to `pkgs` in `justfile` | PENDING |
| 0.4 | Create `voice/README.md` stub with target architecture diagram | PENDING |

### Phase 1 — Config Schema + Migration Script

| # | Task | Status |
|---|------|--------|
| 1.1 | Write `config.json` schema definition in `voice/client/types.ts` | PENDING |
| 1.2 | Create `config.example.json` template | PENDING |
| 1.3 | Write `migrate-voice-config.ts` — reads settings.json + voices.json, merges, writes config.json | PENDING |
| 1.4 | Dry-run migration against current config, verify output matches expected merge | PENDING |
| 1.5 | Document migration in README with rollback instructions | PENDING |

### Phase 2 — Move VoiceServer to New Location

| # | Task | Status |
|---|------|--------|
| 2.1 | `git mv claudecode/.claude/VoiceServer/server.ts voice/.local/share/atlas-voice/server.ts` | PENDING |
| 2.2 | Move pronunciations.json to `voice/.config/atlas-voice/pronunciations.json` | PENDING |
| 2.3 | Move all shell scripts to `voice/.local/share/atlas-voice/` | PENDING |
| 2.4 | Move menubar/ directory | PENDING |
| 2.5 | Update server.ts to read `config.json` (not voices.json hardcoded) | PENDING |
| 2.6 | Fix line 144: `~/.claude/.env` → `~/.config/atlas-voice/.env` | PENDING |
| 2.7 | Change python path from hardcoded `/opt/homebrew/bin/python3` to `which python3` lookup | PENDING |
| 2.8 | Rename LaunchAgent: `com.pai.voice-server` → `com.atlas.voice` in all scripts and plist | PENDING |
| 2.9 | Update log paths: `pai-voice-server.log` → `atlas-voice.log` | PENDING |
| 2.10 | Delete `/notify/personality` endpoint from server.ts | PENDING |
| 2.11 | Rename `menubar/pai-voice.5s.sh` → `menubar/atlas-voice.5s.sh` | PENDING |

### Phase 3 — Build Shared Client Library

| # | Task | Status |
|---|------|--------|
| 3.1 | Create `voice/.local/share/atlas-voice/client/types.ts` — all interfaces | PENDING |
| 3.2 | Create `client/config.ts` — loads + caches + watches config.json | PENDING |
| 3.3 | Create `client/identity.ts` — replaces hooks/lib/identity.ts for voice concerns | PENDING |
| 3.4 | Create `client/index.ts` — `voiceNotify()`, `voiceHealth()`, `getVoiceConfig()` | PENDING |
| 3.5 | Create `client/phase.ts` — `writePhaseState()`, `readPhaseState()` | PENDING |
| 3.6 | Create `client/subagent.ts` — `isSubagentContext()` (VoiceGate logic) | PENDING |
| 3.7 | Write client unit tests — mock fetch, verify payload shape | PENDING |
| 3.8 | Document client API in `voice/client/README.md` | PENDING |

### Phase 4 — Server Enhancements

| # | Task | Status |
|---|------|--------|
| 4.1 | Add `GET /stats` endpoint — uptime, request counts, provider usage, circuit breaker state | PENDING |
| 4.2 | Add JSONL request logging to `~/.local/share/atlas-voice/logs/requests.jsonl` | PENDING |
| 4.3 | Add log rotation on startup (if > 1MB → `.prev`) | PENDING |
| 4.4 | Add config hot-reload (watch `config.json`, reload on change) | PENDING |
| 4.5 | Update rate limiter to key on `source` field from POST body (per-consumer buckets) | PENDING |
| 4.6 | Honor `volume` field in voice_settings (pass to afplay `-v`) | PENDING |
| 4.7 | Add `GET /version` endpoint returning `{name, version, api_version}` | PENDING |

### Phase 5 — Refactor Claude Code Consumers

| # | Task | Status |
|---|------|--------|
| 5.1 | Refactor `VoiceNotification.ts` — use `voiceNotify()` from client | PENDING |
| 5.2 | Refactor `VoiceGreeting.hook.ts` — two paths (agent + Atlas), use client | PENDING |
| 5.3 | Refactor `UpdateTabTitle.hook.ts` | PENDING |
| 5.4 | Refactor `AlgorithmTracker.hook.ts` — read phase state file, keep Bash fallback during transition | PENDING |
| 5.5 | Refactor `VoiceGate.hook.ts` — detect client library usage, not just curl | PENDING |
| 5.6 | Refactor `skills/PAI/Tools/pai.ts` `notifyVoice()` | PENDING |
| 5.7 | Refactor `PAI-Install/engine/actions.ts` health check, shutdown, test notification | PENDING |
| 5.8 | Update `hooks/lib/identity.ts` — delegate voice identity to client, keep non-voice identity | PENDING |

### Phase 6 — Refactor OpenCode Consumers

| # | Task | Status |
|---|------|--------|
| 6.1 | Refactor `plugins-source/voice-notification.ts` — use client, remove `VOICE_SERVER_DIR` auto-start | PENDING |
| 6.2 | Refactor `plugins-source/algorithm-tracker.ts` — read phase state file | PENDING |
| 6.3 | Refactor `skills/PAI/Tools/pai.ts` (opencode copy) | PENDING |
| 6.4 | Refactor `skills/PAI/Tools/algorithm.ts` (opencode) — line 50 `VOICE_URL` | PENDING |

### Phase 7 — Shell Scripts + Converters + Menubar

| # | Task | Status |
|---|------|--------|
| 7.1 | Update `install.sh` + `uninstall.sh` + start/stop/restart/status.sh for new paths | PENDING |
| 7.2 | Update `start.sh:39` health check curl to read server URL from config | PENDING |
| 7.3 | Update `atlas-voice.5s.sh` menubar to read config, show all 4 providers correctly | PENDING |
| 7.4 | Update `scripts/lib/agent-converter.ts` regex — match client imports, not raw curls | PENDING |
| 7.5 | Update `scripts/lib/skill-converter.ts` regex | PENDING |
| 7.6 | Update `scripts/convert-pai-to-oc.ts` — 3 replace() calls updated for client pattern | PENDING |

### Phase 8 — voice-doctor CLI

| # | Task | Status |
|---|------|--------|
| 8.1 | Create `voice/.local/share/atlas-voice/voice-doctor.ts` | PENDING |
| 8.2 | Checks: server reachable, all providers healthy, config file valid, circuit breakers, dedup flags, log sizes, recent events | PENDING |
| 8.3 | Exit codes: 0 healthy, 1 degraded, 2 broken | PENDING |
| 8.4 | Pretty output with Eldritch color scheme | PENDING |
| 8.5 | Add `voice/.local/bin/voice-doctor` symlink for global CLI | PENDING |

### Phase 9 — Documentation

| # | Task | Status |
|---|------|--------|
| 9.1 | Rewrite `voice/README.md` — 4-provider chain, new architecture, correct line counts | PENDING |
| 9.2 | Update `atlas-config/README.md` — reference new voice package | PENDING |
| 9.3 | Update `atlas-config/CLAUDE.md` — new paths | PENDING |
| 9.4 | Update `claudecode/.claude/hooks/README.md` — new voice flow diagrams | PENDING |
| 9.5 | Update `docs/claude-code.md` + `docs/opencode.md` — new paths | PENDING |
| 9.6 | Update `docs/pai-update-guide.md` — VoiceGreeting/VoiceGate no longer protected (they're now thin wrappers) | PENDING |
| 9.7 | Write migration guide `voice/docs/MIGRATION.md` — for anyone restoring from atlas-config backup | PENDING |

### Phase 10 — Verification + Merge

| # | Task | Status |
|---|------|--------|
| 10.1 | Run migration script against live config (in worktree isolation) | PENDING |
| 10.2 | `just status` dry-run — verify stow plan for `voice` package | PENDING |
| 10.3 | `just link` in worktree → `/tmp/atlas-test-home` — verify symlinks | PENDING |
| 10.4 | Unit test: client library against mocked server | PENDING |
| 10.5 | Integration test: start server, POST from client, verify response + JSONL log entry | PENDING |
| 10.6 | Full grep sanity check: `rg 'localhost:8888' voice/ claudecode/ opencode/ --glob '*.ts' --glob '*.js' --glob '*.sh'` returns only `config.json` defaults and `docs/` | PENDING |
| 10.7 | Start real Claude Code session → verify greeting, completion voice, phase tracking | PENDING |
| 10.8 | Start OpenCode session → verify session voice notifications | PENDING |
| 10.9 | Run `voice-doctor` — all checks green | PENDING |
| 10.10 | Test agent voice (spawn Engineer agent → hear Marcus Webb voice) | PENDING |
| 10.11 | Test config hot-reload — edit config.json, verify server picks up change | PENDING |
| 10.12 | Test rate limiter per-source — flood from two sources, verify independent buckets | PENDING |
| 10.13 | Merge worktree → main | PENDING |
| 10.14 | `just unlink && just link` on main — restore from main tree | PENDING |
| 10.15 | 24-hour soak — use normally, check JSONL log for errors | PENDING |

---

## 5. File-by-File Migration Map

### 5.1 Files Moving

| Source | Destination |
|--------|-------------|
| `claudecode/.claude/VoiceServer/server.ts` | `voice/.local/share/atlas-voice/server.ts` |
| `claudecode/.claude/VoiceServer/pronunciations.json` | `voice/.config/atlas-voice/pronunciations.json` |
| `claudecode/.claude/VoiceServer/install.sh` | `voice/.local/share/atlas-voice/install.sh` (rewritten) |
| `claudecode/.claude/VoiceServer/uninstall.sh` | `voice/.local/share/atlas-voice/uninstall.sh` (rewritten) |
| `claudecode/.claude/VoiceServer/start.sh` | `voice/.local/share/atlas-voice/start.sh` (rewritten) |
| `claudecode/.claude/VoiceServer/stop.sh` | `voice/.local/share/atlas-voice/stop.sh` (rewritten) |
| `claudecode/.claude/VoiceServer/restart.sh` | `voice/.local/share/atlas-voice/restart.sh` (rewritten) |
| `claudecode/.claude/VoiceServer/status.sh` | `voice/.local/share/atlas-voice/status.sh` (rewritten) |
| `claudecode/.claude/VoiceServer/menubar/pai-voice.5s.sh` | `voice/.local/share/atlas-voice/menubar/atlas-voice.5s.sh` |
| `claudecode/.claude/VoiceServer/menubar/install-menubar.sh` | `voice/.local/share/atlas-voice/menubar/install-menubar.sh` (rewritten) |

### 5.2 Files Refactored (Stay in Place)

| File | Change |
|------|--------|
| `claudecode/.claude/hooks/VoiceGreeting.hook.ts` | Replace direct settings.json reads with `getIdentity()` from voice client. Replace fetch calls with `voiceNotify()`. |
| `claudecode/.claude/hooks/VoiceGate.hook.ts` | Replace `command.includes('localhost:8888')` with `isVoiceClientInvocation()` from client library. |
| `claudecode/.claude/hooks/UpdateTabTitle.hook.ts` | Replace fetch with `voiceNotify({ source: 'claude-code' })`. |
| `claudecode/.claude/hooks/AlgorithmTracker.hook.ts` | Replace `detectPhaseFromBash()` with `readPhaseState()`. Keep Bash fallback during transition. |
| `claudecode/.claude/hooks/handlers/VoiceNotification.ts` | Replace ~100 lines with `voiceNotify()` call. Keep catchphrase dedup logic. |
| `claudecode/.claude/hooks/lib/identity.ts` | Remove voice-specific fields. Delegate `getVoiceId()`, `getVoiceProsody()`, `getVoicePersonality()` to voice client. |
| `claudecode/.claude/skills/PAI/Tools/pai.ts` | Replace `notifyVoice()` with client call. |
| `opencode/.config/opencode/skills/PAI/Tools/pai.ts` | Same refactor (mirror file). |
| `opencode/.config/opencode/skills/PAI/Tools/algorithm.ts` | Replace `VOICE_URL` hardcoded constant with client call. |
| `opencode/.config/opencode/plugins-source/voice-notification.ts` | Remove `VOICE_SERVER_URL`, `VOICE_SERVER_DIR`, `isServerRunning()`, `startServer()`. Use client. |
| `opencode/.config/opencode/plugins-source/algorithm-tracker.ts` | Replace `VOICE_CURL_PATTERN` regex with `readPhaseState()`. |
| `claudecode/.claude/PAI-Install/engine/actions.ts` | Three hardcoded URLs replaced with client calls. |
| `claudecode/scripts/lib/agent-converter.ts` | Update curl-stripping regex to match client-import pattern (OR remove if no longer needed). |
| `claudecode/scripts/lib/skill-converter.ts` | Same. |
| `claudecode/scripts/convert-pai-to-oc.ts` | Update 3 replace() calls. |

### 5.3 Files Deleted

| File | Why |
|------|-----|
| `claudecode/.claude/VoiceServer/` (entire directory) | Moved to `voice/` |
| `server.ts` lines 1151-1181 (/notify/personality handler) | Dead shim, no real consumers after refactor |
| `claudecode/.claude/hooks/lib/identity.ts` voice-related exports | Moved to voice client library |

### 5.4 Files Modified (settings.json)

| File | Before | After |
|------|--------|-------|
| `claudecode/.claude/settings.json` | `daidentity.voices.*`, `daidentity.personality`, `daidentity.startupCatchphrase(s)` | All moved to `~/.config/atlas-voice/config.json`. `settings.json` keeps only `name`, `displayName`, `fullName`, `color`. |

---

## 6. Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|-----------|
| **Migration script corrupts config** | Low | High | Dry-run in worktree first. Full backup of settings.json + voices.json before migration. Rollback instructions in `voice/docs/MIGRATION.md`. |
| **Hook breakage during transition** | Medium | Medium | All phase 5/6 refactors happen in worktree. Full smoke test before merge. Old code path preserved until all consumers refactored. |
| **LaunchAgent migration loses voice** | Medium | High | Phase 2 tests new LaunchAgent in worktree before removing old. Rollback by re-loading old plist. |
| **Python path auto-detection fails** | Low | High | Fall back to `/opt/homebrew/bin/python3` if `which python3` fails. Document override in config.json. |
| **AlgorithmTracker loses phase tracking during transition** | Medium | Low | Keep Bash sniffing as fallback during phase 5. Remove only after all consumers migrate. |
| **Stow conflicts with existing `~/.local/share/`** | Low | Medium | `just status` dry-run in phase 10. Detect and resolve conflicts before `just link`. |
| **OpenCode auto-start broken (no more VOICE_SERVER_DIR)** | Medium | Low | Auto-start was fragile anyway. Replaced with explicit `voice-doctor --start` instruction in OpenCode README. |
| **Menubar widget cosmetic break** | Low | Low | Cosmetic only. Not on critical path. |
| **Hot-reload triggers race conditions** | Low | Medium | Use atomic rename (write to temp + mv). Debounce file watcher to 500ms. |
| **Converter regex no longer matches** | Low | Low | Update + test. Worst case: some agent conversions need manual cleanup. |

### Critical Path Items

1. **Config migration must be idempotent and reversible.** Script writes new config, leaves old files in place. Rollback = restore old files.
2. **Phase 5 cannot merge until phase 3 (client library) is complete and tested.**
3. **LaunchAgent rename must not leave orphaned `com.pai.voice-server` running.** Explicit unload in install.sh before loading new plist.
4. **JSONL log path must exist before server startup** — create directory in install.sh.
5. **Phase 10.6 grep check is the authoritative "done" signal.** Zero `localhost:8888` in code files.

---

## 7. Out of Scope

Explicitly NOT in this plan:

1. **Extract to separate git repo** (Plan A). Deferred until voice actually has consumers outside atlas-config.
2. **CLI binary replacement** (Plan B). Rejected — the `--serve` shim defeats the point, circuit breakers are valuable, flock is new unbattle-tested code.
3. **VoiceLayer convergence.** VoiceLayer handles bidirectional converse (STT+TTS); this plan is about push notifications. Convergence is a future consideration.
4. **New features** beyond observability + hot-reload + volume fix. No audio queue redesign. No streaming TTS. No web dashboard.
5. **Agent voice personality expansion.** 14 agents stay. New agents can be added post-overhaul.
6. **Changing TTS providers.** EdgeTTS remains primary. ElevenLabs remains opt-in. Kokoro remains optional.
7. **Pi.dev adapter.** This plan makes voice agent-agnostic within atlas-config. Pi.dev integration belongs in the `2026-03-02-multi-agent-compatibility-restructure` plan.

---

## 8. Success Criteria (ISC — tracked via TaskList)

All 15 ISC criteria must pass before merge:

1. **ISC-1**: VoiceServer lives in dedicated `voice/` stow package
2. **ISC-2**: Single source-of-truth config at `~/.config/atlas-voice/config.json`
3. **ISC-3**: Server.ts has zero atlas-config coupling (no `~/.claude` references)
4. **ISC-4**: Shared TypeScript voice client library exists and is importable
5. **ISC-5**: All 17 voice consumers use the client library (zero `localhost:8888` in code)
6. **ISC-6**: camelCase/snake_case field mismatch fixed
7. **ISC-7**: AlgorithmTracker no longer sniffs Bash commands
8. **ISC-8**: Dead `/notify/personality` endpoint removed
9. **ISC-9**: Dropped `volume` field is honored
10. **ISC-10**: Observability — JSONL log + `/stats` endpoint + log rotation
11. **ISC-11**: `voice-doctor` CLI exists and reports subsystem health
12. **ISC-12**: LaunchAgent renamed `com.pai.voice-server` → `com.atlas.voice`
13. **ISC-13**: README and docs match actual architecture
14. **ISC-14**: Rate limiter uses caller-source identification
15. **ISC-15**: Config hot-reload — watch `config.json` for changes

---

## 9. Open Questions for Ed

Before implementation begins, these need decisions:

1. **Config location**: `~/.config/atlas-voice/config.json` (XDG, recommended) vs `~/.atlas-voice/config.json` (like ~/.claude) — which do you prefer?
2. **Runtime location**: `~/.local/share/atlas-voice/` (XDG, recommended) vs `~/Library/Application Support/atlas-voice/` (macOS-native)?
3. **LaunchAgent name**: `com.atlas.voice` or `com.ed.atlas-voice` or something else?
4. **Settings migration**: Should `daidentity.voices.*` be fully removed from `settings.json`, or kept as a stale-but-ignored backup?
5. **EdgeTTS agent voices**: Should we populate `edgetts` voice config for all 14 agents as part of this plan, or leave it as `en-US-AvaNeural` fallback and add agent voices later?
6. **Phase tracking transition**: Keep Bash sniffing as fallback forever, or delete after N days of successful phase-state-file usage?
7. **voice-doctor as binary**: Ship as compiled Bun binary or as a `.ts` file invoked via `bun run`?
8. **Agent conversion regex**: Update the converters to strip client imports, or remove the conversion entirely (agents no longer need to mention voice at all)?

---

## 10. Red Team Validation Results

The following claims from exploration were validated by a red team agent:

| # | Claim | Verdict | Notes |
|---|-------|---------|-------|
| 1 | server.ts zero coupling | **PARTIALLY REFUTED** | Line 144 hardcodes `~/.claude/.env`. Plan fixes this in ISC-3. |
| 2 | Complete consumer list | **PARTIALLY REFUTED** | 5 files missed: opencode algorithm-tracker, opencode algorithm.ts, skill-converter, convert-pai-to-oc, VoiceServer/start.sh. All added to plan. |
| 3 | camelCase/snake_case mismatch | **CONFIRMED** | settings.json:976 `similarityBoost`, identity.ts:32 `similarity_boost`. ISC-6. |
| 4 | No agent edgetts voices | **CONFIRMED** | voices.json has no `edgetts` block in any agent. ISC-2 schema supports it. |
| 5 | VoiceGreeting bypasses identity.ts | **CONFIRMED** | Lines 178, 238 direct Bun.file() reads. ISC-5 refactor. |
| 6 | /notify/personality is dead shim | **CONFIRMED** | server.ts:1161 `sendNotification(..., null)`. ISC-8 removes it. |
| 7 | AlgorithmTracker Bash dependency | **CONFIRMED** | Both claudecode and opencode versions. ISC-7 replaces with state file. |
| 8 | OpenCode hardcoded VOICE_SERVER_DIR | **CONFIRMED** | Line 23. Phase 6.1 removes. |
| 9 | Volume parsed but dropped | **CONFIRMED** | VoiceGreeting.hook.ts:123,208-213. ISC-9 honors it. |
| 10 | Single rate-limit bucket | **CONFIRMED** | server.ts:1086 `'localhost'` string literal. ISC-14 fixes. |

---

## 11. What This Plan Is NOT

- **NOT** a CLI binary rewrite (rejected — spread complexity, `--serve` shim defeats the point)
- **NOT** a separate git repo extraction (premature — no consumers outside atlas-config)
- **NOT** a greenfield rewrite (server.ts works, just needs coupling surgery)
- **NOT** a feature addition (observability + hot-reload are maintenance, not new capability)
- **NOT** an incremental rollout (too many interdependencies — must merge atomically)
- **NOT** over-engineered (no microservices, no etcd, no event bus, no new paradigm)
- **NOT** too simple (actually fixes all 10 validated bugs, not just moving files)

---

*Plan by Atlas for Ed, 2026-04-10. Red-team validated. Awaiting approval.*
