# VoiceServer Standalone Extraction — Deep Dive & Implementation Plan

> Extract VoiceServer from atlas-config into a standalone, agent-agnostic application that gives any AI coding agent a voice.

**Status:** Plan complete, awaiting approval
**Date:** 2026-04-06
**Supersedes:** `2026-03-11-atlas-voice-standalone-design.md` (pre-VoiceLayer, pre-Edge TTS primary)
**Target repo:** `~/Developer/AI/atlas-voice/`
**Runtime:** `~/.local/share/atlas-voice/` (symlinked from source)

---

## Table of Contents

1. [Current State Analysis](#1-current-state-analysis)
2. [What VoiceServer Actually Does](#2-what-voiceserver-actually-does)
3. [How Claude Code Integration Works Today](#3-how-claude-code-integration-works-today)
4. [VoiceServer vs VoiceLayer](#4-voiceserver-vs-voicelayer)
5. [Extraction Architecture](#5-extraction-architecture)
6. [Agent Adapter System](#6-agent-adapter-system)
7. [Implementation Phases](#7-implementation-phases)
8. [File-by-File Migration Map](#8-file-by-file-migration-map)
9. [What Changes in atlas-config](#9-what-changes-in-atlas-config)
10. [Risk Assessment](#10-risk-assessment)
11. [Open Questions](#11-open-questions)

---

## 1. Current State Analysis

### 1.1 What Exists Today

VoiceServer is a **Bun HTTP server** (`server.ts`, ~1,276 lines) running on port 8888. It lives at:

```
claudecode/.claude/VoiceServer/
├── server.ts              ← Main server (all-in-one: providers, routing, pipeline)
├── voices.json            ← Voice + provider configuration (6.4 KB)
├── pronunciations.json    ← TTS word-boundary replacements (629 B)
├── README.md              ← Documentation (14.7 KB)
├── install.sh             ← macOS LaunchAgent installer
├── uninstall.sh           ← Remove LaunchAgent
├── start.sh               ← launchctl load wrapper
├── stop.sh                ← launchctl unload + kill port
├── restart.sh             ← stop then start
├── status.sh              ← Health check + port check + log tail
├── logs/
│   ├── voice-server.log       ← stdout (25.6 MB)
│   └── voice-server-error.log ← stderr (32 MB)
└── menubar/
    ├── pai-voice.5s.sh        ← SwiftBar/BitBar health widget
    └── install-menubar.sh     ← Menubar plugin installer
```

### 1.2 Runtime Characteristics

| Property | Value |
|----------|-------|
| Runtime | Bun (TypeScript, no build step) |
| Port | 8888 (configurable via `PORT` env) |
| NPM dependencies | **Zero** — stdlib only |
| System dependencies | python3 + edge-tts, afplay (macOS), say (macOS) |
| Process management | macOS LaunchAgent (`com.pai.voice-server`) |
| Binding | localhost only |
| Rate limit | 10 req/60s per IP |
| Input limit | 500 chars per message |

### 1.3 Coupling Points to atlas-config

Despite the README claiming agent-agnosticism, these coupling points exist:

| Coupling | Where | Severity |
|----------|-------|----------|
| **Physical location** | Lives inside `claudecode/.claude/VoiceServer/` | High — Stow deploys it as part of Claude Code config |
| **LaunchAgent name** | `com.pai.voice-server` — PAI branding | Low — cosmetic |
| **Voice identity** | `voices.json` references Atlas-specific agent names | Medium — config, not code |
| **Hook consumers** | 5+ hooks POST to `localhost:8888` | High — but they're consumers, not dependencies |
| **Identity resolution** | Hooks use `hooks/lib/identity.ts` which reads `settings.json` | High — hook-side coupling, not server-side |
| **Log paths** | `MEMORY/VOICE/voice-events.jsonl` (hook-side logging) | Medium — hook concern |
| **Pronunciation data** | PAI/ISC-specific terms | Low — config file |

**Key insight:** The server code itself (`server.ts`) has **zero** coupling to Claude Code or atlas-config. All coupling is in the *consumers* (hooks) and *deployment* (Stow, LaunchAgent). The extraction is primarily a deployment and packaging concern, not a code refactor.

---

## 2. What VoiceServer Actually Does

### 2.1 Request Pipeline

```
POST /notify (JSON body)
     │
     ▼
┌─────────────────────────────────────────────────┐
│ 1. RATE LIMIT — 10 req/60s per IP               │
│ 2. PARSE — Extract message, voice_id, settings   │
│ 3. SANITIZE — Strip markdown, shell chars, XSS    │
│ 4. PRONOUNCE — Apply word-boundary replacements   │
│ 5. EMOTION — Extract [💡 insight] markers → tone  │
│ 6. VOICE RESOLVE — 3-tier settings resolution     │
│    a) Caller pass-through voice_settings           │
│    b) voices.json lookup by voice_id               │
│    c) Defaults (stability:0.5, boost:0.75)         │
│ 7. SPEAK — Provider chain with circuit breakers    │
│    EdgeTTS → ElevenLabs → Kokoro → macOS say       │
│ 8. PLAY — afplay (macOS) or mpv (Linux)            │
│ 9. NOTIFY — osascript macOS notification           │
└─────────────────────────────────────────────────┘
```

### 2.2 TTS Provider Chain (4 Providers)

| Priority | Provider | Engine | Quality | Latency | Requirements | Current Status |
|----------|----------|--------|---------|---------|-------------|----------------|
| 1 | **Edge TTS** | Microsoft Neural | High | ~2s | python3 + edge-tts pip | **Enabled** (primary) |
| 2 | **ElevenLabs** | Cloud API | Premium | ~1.5s | ELEVENLABS_API_KEY | Disabled (no key) |
| 3 | **Kokoro** | Local 82M model | Good | ~3s | Kokoro server on :8880 | Disabled (not running) |
| 4 | **macOS say** | System TTS | Basic | ~0.5s | macOS | **Enabled** (always-on fallback) |

Each cloud/local provider has a **circuit breaker**: 1 failure → open → 60s cooldown → half-open → retry.

### 2.3 API Surface

| Method | Path | Purpose | Auth |
|--------|------|---------|------|
| POST | `/notify` | Primary TTS with full voice_id/settings control | None (localhost) |
| POST | `/notify/personality` | Compat shim — message only, identity voice | None |
| POST | `/pai` | Simple alias — title + message, identity voice | None |
| GET | `/health` | Provider status, circuit breakers, config counts | None |
| POST | `/shutdown` | Graceful shutdown (used by installer) | None |

### 2.4 Configuration System

**`voices.json`** — Single source of truth, loaded once at startup:
- `providers{}` — Enable/disable + config per provider
- `defaultProvider` — Which provider to try first (`"edgetts"`)
- `fallbackOrder[]` — Provider chain sequence
- `identity{}` — Default voice (when no voice_id specified)
- `agents{}` — 14 named agent voice mappings with per-provider settings
- `default_rate` — macOS say WPM (175)
- `default_volume` — afplay volume 0.0-1.0 (0.8)

**`pronunciations.json`** — Word-boundary regex replacements:
- Compiled once at startup into `RegExp` array
- Applied before any TTS call
- Currently 4 rules: Kai→Kye, PAI→pie, ISC→"I S C", principal name→phonetic

### 2.5 Emotional Presets (13 markers)

Messages can include emoji markers like `[💡 insight]` that adjust voice tone via stability/similarity_boost overlays. The marker is stripped before speech. This is a unique differentiator — no other TTS server does emotional modulation from inline emoji markers.

---

## 3. How Claude Code Integration Works Today

### 3.1 Hook-to-VoiceServer Call Flow

```
Claude Code Session
     │
     ├── SessionStart
     │   └── VoiceGreeting.hook.ts (async)
     │         ├── Layered subagent suppression (5 guards)
     │         ├── Main session → POST /notify {catchphrase}
     │         └── Named agent → POST /notify {agent greeting, agent voice_id}
     │
     ├── UserPromptSubmit
     │   └── UpdateTabTitle.hook.ts
     │         └── LLM inference → title → POST /notify {title, identity voice}
     │
     ├── PreToolUse (Bash)
     │   └── VoiceGate.hook.ts
     │         └── Blocks subagent Bash commands containing localhost:8888
     │
     ├── PostToolUse (Bash, TaskCreate, TaskUpdate)
     │   └── AlgorithmTracker.hook.ts
     │         ├── Detects phase transitions from voice curl patterns
     │         └── POST /notify on rework cycles
     │
     └── Stop
         └── StopOrchestrator.hook.ts
               ├── VoiceNotification.ts handler
               │   ├── Extracts 🗣️ line from response
               │   ├── Resolves voice from identity.ts → settings.json
               │   └── POST /notify {completion message, identity voice}
               └── DocCrossRefIntegrity.ts handler
                   └── POST /notify on doc drift (3s timeout)
```

### 3.2 Voice Identity Resolution Chain

```
settings.json
  └── daidentity.voices.main.voiceId → ElevenLabs ID "21m00Tcm4TlvDq8ikWAM"
  └── daidentity.voices.main.* → stability, similarityBoost, style, speed

hooks/lib/identity.ts
  └── getIdentity() → cached DA_IDENTITY object
  └── getVoiceId() → mainDAVoiceID string

Agent .md frontmatter (e.g., Engineer.md)
  └── voiceId: "Fahco4VZzobUeiPqni1S"
  └── voice: {stability, similarity_boost, style, speed, use_speaker_boost, volume}
  └── persona: {name: "Marcus Webb"}
```

### 3.3 Subagent Protection (Layered)

This is the most complex integration — preventing agent swarms from flooding the server:

1. **VoiceGate (PreToolUse)** — Blocks any Bash command containing `localhost:8888` when `agent_id` is present in stdin (subagent context)
2. **VoiceGreeting guards** — 5 layers checking env vars and context to skip greetings in subagent processes
3. **StopOrchestrator gate** — `TERM_PROGRAM` env var present only in real terminal sessions
4. **PAI_SUPPRESS_VOICE** — Env var set by SessionExtract.ts when spawning `claude -p` child processes

### 3.4 Other Voice Consumers in atlas-config

| Consumer | File | Pattern |
|----------|------|---------|
| PAI CLI | `skills/PAI/Tools/pai.ts` | `/notify/personality` or `/notify` |
| Algorithm runner | `skills/PAI/Tools/algorithm.ts` | `/notify` with voice_id from settings |
| Integrity scanner | `skills/PAI/Tools/IntegrityMaintenance.ts` | `/notify` or `/notify/personality` |
| Browser tool | `skills/Browser/Tools/Browse.ts` | `/notify/personality` |
| Agent composer | `skills/Agents/Tools/ComposeAgent.ts` | Injects curl snippets into agent prompts |
| PAI installer | `PAI-Install/engine/actions.ts` | `/health`, `/shutdown`, `/notify` |
| Notification system doc | `skills/PAI/THENOTIFICATIONSYSTEM.md` | Documents curl patterns for AI to follow |

---

## 4. VoiceServer vs VoiceLayer

These are **two parallel voice systems** that must both be understood for a complete extraction:

| Dimension | VoiceServer (this extraction) | VoiceLayer (stays as MCP) |
|-----------|-------------------------------|---------------------------|
| **Role** | Push notifications — hooks fire-and-forget TTS | Pull I/O — Claude calls tools for interactive voice |
| **Direction** | One-way (TTS only) | Bidirectional (TTS + STT) |
| **Interface** | HTTP REST (`POST /notify`) | MCP tools (stdio via socat → Unix socket) |
| **Callers** | Hooks, scripts, curl | Claude Code tool system |
| **TTS** | Edge TTS / ElevenLabs / Kokoro / say | Edge TTS only |
| **STT** | None | whisper-cpp (local) or Wispr Flow (cloud) |
| **Session mgmt** | Rate limiter | Lockfile-based mic session booking |
| **Lifecycle** | macOS LaunchAgent (always-on daemon) | Singleton daemon on Unix socket |
| **Scope of extraction** | **This plan** — becomes standalone app | **Out of scope** — stays as MCP server |

**Relationship:** They share `edge_tts` Python CLI as a TTS engine but have zero shared state. VoiceServer extraction does not affect VoiceLayer.

**Future convergence opportunity:** VoiceLayer could become a consumer of VoiceServer's TTS endpoint instead of calling `edge_tts` directly. This would unify TTS through a single provider chain with circuit breakers and fallbacks. But this is P2, not P0.

---

## 5. Extraction Architecture

### 5.1 Target Repo Structure

```
~/Developer/AI/atlas-voice/
├── server.ts                    ← Voice server (from VoiceServer/server.ts)
├── voices.json                  ← Voice + provider config
├── pronunciations.json          ← TTS pronunciation overrides
├── README.md                    ← Standalone docs (rewritten)
├── LICENSE                      ← MIT or similar
├── package.json                 ← Bun project metadata (no deps)
├── tsconfig.json                ← TypeScript config
│
├── install.sh                   ← Universal installer (agent detection)
├── uninstall.sh                 ← Clean removal
├── start.sh                     ← Start daemon
├── stop.sh                      ← Stop daemon
├── restart.sh                   ← Restart daemon
├── status.sh                    ← Health check
│
├── cli/                         ← CLI companion (P1)
│   └── voice.ts                 ← `voice say "hello"`, `voice mute`, etc.
│
├── adapters/                    ← Per-agent integration packages
│   ├── claude-code/
│   │   ├── README.md            ← Claude Code integration guide
│   │   ├── adapter.json         ← Server URL, identity config
│   │   ├── hooks/
│   │   │   ├── VoiceGreeting.hook.ts
│   │   │   ├── VoiceGate.hook.ts
│   │   │   └── VoiceNotification.ts
│   │   └── install-adapter.sh   ← Deploys hooks + config to ~/.claude/
│   │
│   ├── opencode/                ← Future
│   │   ├── README.md
│   │   └── voice-handler.ts     ← OpenCode event handler
│   │
│   ├── gemini-cli/              ← Future
│   ├── codex/                   ← Future
│   └── generic/                 ← Shell script adapter for any agent
│       ├── README.md
│       ├── voice-notify.sh      ← Standalone curl wrapper
│       └── voice-hooks.sh       ← Git hooks integration
│
├── menubar/                     ← macOS menu bar widget
│   ├── pai-voice.5s.sh
│   └── install-menubar.sh
│
├── docs/                        ← Extended documentation
│   ├── providers.md             ← TTS provider deep dive
│   ├── emotional-presets.md     ← Emoji marker reference
│   └── adapter-development.md   ← How to write a new adapter
│
└── tests/                       ← Test suite (P1)
    ├── server.test.ts
    ├── providers.test.ts
    └── pipeline.test.ts
```

### 5.2 Design Principles

1. **One server, many consumers** — Single daemon on port 8888, adapters teach each agent how to POST to it
2. **Zero coupling in core** — `server.ts` has no agent-specific code (already true)
3. **Config over code** — Agent identities, voices, providers all configured via JSON
4. **Adapter isolation** — Each adapter is self-contained; installing/removing one doesn't affect others
5. **Symlink from source** — `install.sh` symlinks to `~/.local/share/atlas-voice/` for live development
6. **Progressive enhancement** — Works with just `curl` → better with adapter → best with full hook integration

### 5.3 Server Changes (Minimal)

The server itself needs almost no changes. Key modifications:

| Change | Reason |
|--------|--------|
| **Config path resolution** | Load `voices.json` and `pronunciations.json` from `__dirname` (already does this) |
| **Rename LaunchAgent** | `com.pai.voice-server` → `com.atlas-voice.server` |
| **Add `/version` endpoint** | Return server version for adapter compatibility checks |
| **Add `source` field logging** | Already accepts `source` in POST body, improve structured logging |
| **Add Linux audio support** | `mpv` playback path already exists but needs testing |
| **Config hot-reload** (P1) | Watch `voices.json` for changes, reload without restart |

### 5.4 New `/version` Endpoint

```json
GET /version
{
  "name": "atlas-voice",
  "version": "1.0.0",
  "api_version": "v1",
  "providers": ["edgetts", "elevenlabs", "kokoro", "say"],
  "features": ["emotional-presets", "pronunciations", "circuit-breakers"]
}
```

Adapters use this to verify compatibility before deploying.

---

## 6. Agent Adapter System

### 6.1 Adapter Interface

Every adapter must provide:

```
adapters/{agent-name}/
├── adapter.json            ← Adapter metadata + config template
├── install-adapter.sh      ← Deploy to agent's config directory
├── uninstall-adapter.sh    ← Clean removal
└── README.md               ← Integration documentation
```

**adapter.json** structure:

```json
{
  "agent": "claude-code",
  "version": "1.0.0",
  "min_server_version": "1.0.0",
  "detect_paths": ["~/.claude/settings.json"],
  "target_dir": "~/.claude/",
  "config": {
    "server": "http://localhost:8888",
    "identity": {
      "name": "Atlas",
      "voice_id": "21m00Tcm4TlvDq8ikWAM",
      "catchphrase": "Atlas, standing by"
    },
    "suppress_env": "PAI_SUPPRESS_VOICE"
  }
}
```

### 6.2 Claude Code Adapter (P0 — Ship First)

This is the existing integration, extracted and packaged:

**What it deploys:**
- `VoiceGreeting.hook.ts` → `~/.claude/hooks/`
- `VoiceGate.hook.ts` → `~/.claude/hooks/`
- `VoiceNotification.ts` → `~/.claude/hooks/handlers/`
- `voice-adapter.json` → `~/.claude/` (server URL + identity)

**What it changes in hooks:**
- Replace hardcoded `http://localhost:8888` with `require('./voice-adapter.json').server`
- Replace `getIdentity()` / `getVoiceId()` with adapter config for voice identity
- Keep subagent protection logic (VoiceGate, PAI_SUPPRESS_VOICE) — this is Claude Code-specific and belongs in the adapter

**Install flow:**
```bash
bash adapters/claude-code/install-adapter.sh
# 1. Check atlas-voice server is running
# 2. Copy hooks to ~/.claude/hooks/
# 3. Write voice-adapter.json to ~/.claude/
# 4. Merge hook entries into ~/.claude/settings.json (if not present)
# 5. Verify: curl localhost:8888/health
```

### 6.3 OpenCode Adapter (P2)

OpenCode has an event system (not hooks). The adapter would:
- Register voice event handlers for session lifecycle events
- Map OpenCode's agent spawn model to voice identity resolution
- Use the same `POST /notify` API

### 6.4 Generic Adapter (P1)

For agents without a formal hook system:
- `voice-notify.sh` — Standalone curl wrapper with nice defaults
- `voice-hooks.sh` — Git pre-commit/post-commit hooks that announce
- Works with any agent that can run shell scripts

### 6.5 Installer Agent Detection

```bash
# install.sh detects installed agents
detect_agents() {
    [[ -f ~/.claude/settings.json ]] && echo "claude-code"
    [[ -d ~/.config/opencode ]] && echo "opencode"
    [[ -d ~/.gemini ]] && echo "gemini-cli"
    [[ -d ~/.codex ]] && echo "codex"
}

# For each detected agent with a supported adapter:
# - Show: "Found Claude Code. Install voice adapter? [Y/n]"
# - Deploy adapter files
# - Verify integration
```

---

## 7. Implementation Phases

### Phase 0: Repository Setup (Day 1)

- [ ] Create `~/Developer/AI/atlas-voice/` git repo
- [ ] Copy core files from `claudecode/.claude/VoiceServer/`:
  - `server.ts`, `voices.json`, `pronunciations.json`, `README.md`
  - All lifecycle scripts (`install.sh`, `start.sh`, `stop.sh`, etc.)
  - `menubar/` directory
- [ ] Create `package.json` (metadata only, no deps)
- [ ] Create `tsconfig.json`
- [ ] Add `/version` endpoint to server.ts
- [ ] Rename LaunchAgent: `com.pai.voice-server` → `com.atlas-voice.server`
- [ ] Update `install.sh` to use new LaunchAgent name and `~/.local/share/atlas-voice/` path
- [ ] Verify server runs standalone from new repo: `bun run server.ts`
- [ ] Test all endpoints: `/notify`, `/health`, `/version`

### Phase 1: Claude Code Adapter (Day 1-2)

- [ ] Create `adapters/claude-code/` structure
- [ ] Extract voice hooks from atlas-config hooks into adapter:
  - `VoiceGreeting.hook.ts` — parameterize identity from `adapter.json`
  - `VoiceGate.hook.ts` — keep as-is (Claude Code-specific subagent logic)
  - `VoiceNotification.ts` — parameterize server URL and voice identity
- [ ] Create `adapter.json` with Atlas identity defaults
- [ ] Create `install-adapter.sh` that:
  - Checks server is running
  - Deploys hooks to `~/.claude/hooks/`
  - Writes `voice-adapter.json`
  - Adds hook entries to `settings.json` (idempotent merge)
- [ ] Create `uninstall-adapter.sh` (clean removal)
- [ ] Test full lifecycle: install adapter → session start → greeting → prompt → completion voice → uninstall

### Phase 2: atlas-config Decoupling (Day 2-3)

- [ ] Remove `claudecode/.claude/VoiceServer/` directory from atlas-config
- [ ] Update `.stow-local-ignore` for claudecode package
- [ ] Update atlas-config hooks to use `voice-adapter.json` for server URL:
  - `hooks/handlers/VoiceNotification.ts`
  - `hooks/VoiceGreeting.hook.ts`
  - `hooks/VoiceGate.hook.ts`
  - `hooks/UpdateTabTitle.hook.ts`
  - `hooks/AlgorithmTracker.hook.ts`
  - `hooks/handlers/DocCrossRefIntegrity.ts`
- [ ] Update other voice consumers:
  - `skills/PAI/Tools/pai.ts`
  - `skills/PAI/Tools/algorithm.ts`
  - `skills/PAI/Tools/IntegrityMaintenance.ts`
  - `skills/Browser/Tools/Browse.ts`
  - `skills/Agents/Tools/ComposeAgent.ts`
  - `skills/PAI/THENOTIFICATIONSYSTEM.md`
- [ ] Update `README.md`, `CLAUDE.md`, hooks `README.md` references
- [ ] Update `install.sh` in PAI-Install to reference atlas-voice
- [ ] Run `just update` (restow) — verify no breakage
- [ ] Test end-to-end: fresh session with voice greeting, prompt, completion

### Phase 3: Generic Adapter + CLI (Day 3-4)

- [ ] Create `adapters/generic/`:
  - `voice-notify.sh` — `voice-notify "Task complete"` wrapper
  - `voice-hooks.sh` — Git hook integration
  - `README.md` — Integration guide for any agent
- [ ] Create `cli/voice.ts`:
  - `voice say "hello"` — speak a message
  - `voice status` — show server health
  - `voice mute [duration]` — suppress audio (with optional auto-unmute)
  - `voice unmute` — resume audio
  - `voice logs [--tail]` — view request log
  - `voice test` — speak a test message from each provider
- [ ] Add `POST /mute` and `POST /unmute` endpoints to server
- [ ] Install CLI: `bun link` → `atlas-voice` global command

### Phase 4: Quality & Docs (Day 4-5)

- [ ] Write standalone README.md (not PAI-specific)
- [ ] Write `docs/providers.md` — TTS provider comparison and setup
- [ ] Write `docs/emotional-presets.md` — Complete marker reference
- [ ] Write `docs/adapter-development.md` — How to build a new adapter
- [ ] Add basic test suite:
  - `tests/server.test.ts` — endpoint tests
  - `tests/pipeline.test.ts` — sanitization, pronunciation, emotion extraction
  - `tests/providers.test.ts` — provider interface tests
- [ ] Config hot-reload: watch `voices.json`, reload without restart
- [ ] Structured JSONL request log (every notification: timestamp, agent, provider, latency)

### Phase 5: Future Adapters (Rolling)

- [ ] OpenCode adapter (when OpenCode event system is mature)
- [ ] Codex CLI adapter
- [ ] Gemini CLI adapter
- [ ] Cursor adapter (if extension API supports it)
- [ ] Windsurf adapter
- [ ] VoiceLayer convergence — make VoiceLayer a consumer of atlas-voice TTS

---

## 8. File-by-File Migration Map

### Files Moving to atlas-voice

| Source (atlas-config) | Destination (atlas-voice) | Changes |
|----------------------|--------------------------|---------|
| `VoiceServer/server.ts` | `server.ts` | +`/version` endpoint, rename LaunchAgent |
| `VoiceServer/voices.json` | `voices.json` | Unchanged |
| `VoiceServer/pronunciations.json` | `pronunciations.json` | Unchanged |
| `VoiceServer/README.md` | `README.md` | Rewritten for standalone context |
| `VoiceServer/install.sh` | `install.sh` | +agent detection, +adapter deployment |
| `VoiceServer/uninstall.sh` | `uninstall.sh` | Updated paths |
| `VoiceServer/start.sh` | `start.sh` | Updated LaunchAgent name |
| `VoiceServer/stop.sh` | `stop.sh` | Updated LaunchAgent name |
| `VoiceServer/restart.sh` | `restart.sh` | Unchanged |
| `VoiceServer/status.sh` | `status.sh` | Updated paths |
| `VoiceServer/menubar/*` | `menubar/*` | Unchanged |

### Files That Become Adapter (claude-code)

| Source (atlas-config) | Destination (atlas-voice adapter) | Changes |
|----------------------|----------------------------------|---------|
| `hooks/VoiceGreeting.hook.ts` | `adapters/claude-code/hooks/VoiceGreeting.hook.ts` | Read identity from `adapter.json` not `settings.json` |
| `hooks/VoiceGate.hook.ts` | `adapters/claude-code/hooks/VoiceGate.hook.ts` | Minimal — already agent-specific |
| `hooks/handlers/VoiceNotification.ts` | `adapters/claude-code/hooks/VoiceNotification.ts` | Read server URL from `adapter.json` |

### Files Staying in atlas-config (Updated)

These files call VoiceServer but stay in atlas-config — they just read the server URL from a config instead of hardcoding it:

| File | Change |
|------|--------|
| `hooks/UpdateTabTitle.hook.ts` | Read server URL from `voice-adapter.json` |
| `hooks/AlgorithmTracker.hook.ts` | Read server URL from `voice-adapter.json` |
| `hooks/handlers/DocCrossRefIntegrity.ts` | Read server URL from `voice-adapter.json` |
| `hooks/StopOrchestrator.hook.ts` | No change (delegates to VoiceNotification handler) |
| `skills/PAI/Tools/pai.ts` | Read server URL from env or config |
| `skills/PAI/Tools/algorithm.ts` | Read server URL from env or config |
| `skills/PAI/Tools/IntegrityMaintenance.ts` | Read server URL from env or config |
| `skills/Browser/Tools/Browse.ts` | Read server URL from env or config |
| `skills/Agents/Tools/ComposeAgent.ts` | Update curl template to use configured URL |
| `skills/PAI/THENOTIFICATIONSYSTEM.md` | Update documentation |

---

## 9. What Changes in atlas-config

### Before Extraction

```
atlas-config/
  claudecode/.claude/
    VoiceServer/           ← Full server lives here
    hooks/
      VoiceGreeting.hook.ts   ← Reads settings.json, POSTs localhost:8888
      VoiceGate.hook.ts       ← Blocks subagent voice curls
      handlers/
        VoiceNotification.ts  ← Reads identity.ts, POSTs localhost:8888
```

### After Extraction

```
atlas-config/
  claudecode/.claude/
    voice-adapter.json     ← Points to atlas-voice server (deployed by adapter installer)
    hooks/
      VoiceGreeting.hook.ts   ← Reads voice-adapter.json (deployed by adapter, or updated in-place)
      VoiceGate.hook.ts       ← Blocks subagent voice curls (deployed by adapter, or updated in-place)
      handlers/
        VoiceNotification.ts  ← Reads voice-adapter.json (deployed by adapter, or updated in-place)
    # VoiceServer/ REMOVED — lives in atlas-voice repo now

atlas-voice/
  server.ts                ← Voice server (standalone)
  voices.json              ← All voice config
  adapters/
    claude-code/           ← Hooks + config for Claude Code
```

### Stow Implications

The `claudecode` Stow package currently includes `VoiceServer/` in the stow tree. After extraction:
- Remove `VoiceServer/` from `claudecode/.claude/`
- Add `VoiceServer` to `claudecode/.stow-local-ignore`
- The adapter installer (from atlas-voice) deploys hooks to `~/.claude/hooks/` and `voice-adapter.json` to `~/.claude/`
- These hook files could either:
  - **Option A:** Be adapter-deployed files (not in atlas-config at all) — cleaner separation
  - **Option B:** Stay in atlas-config but be parameterized to read `voice-adapter.json` — simpler migration

**Recommended: Option B for Phase 2, migrate to Option A later.** This minimizes the blast radius of the initial extraction.

---

## 10. Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|-----------|
| **LaunchAgent regression** | Medium | High — no voice at all | Test install/start/stop/restart cycle before removing from atlas-config |
| **Hook breakage during transition** | Medium | Medium — degraded UX | Phase 2 is incremental; keep old hooks working while new config is tested |
| **Stow conflicts** | Low | Medium — deploy errors | `just status` dry run before and after removal |
| **Simultaneous two-server** | Low | Low — port conflict | install.sh checks for existing process on :8888 before starting |
| **Voice identity mismatch** | Low | Low — wrong voice | adapter.json explicitly declares identity, no implicit resolution |
| **Edge TTS CLI path** | Medium | High — silence | Server hardcodes `/opt/homebrew/bin/python3`; standalone needs `which python3` |

### Critical Path Items

1. **Python path portability** — Server hardcodes `/opt/homebrew/bin/python3`. Standalone version must use `which python3` or configurable path.
2. **Log path migration** — Server logs to `VoiceServer/logs/` (relative). Standalone should log to `~/.local/share/atlas-voice/logs/` or `~/Library/Logs/`.
3. **Config path in hooks** — Hooks currently import from relative paths (`../lib/identity.ts`). Adapter hooks need self-contained config loading.

---

## 11. Open Questions

### For Ed to decide:

1. **Repo location** — `~/Developer/AI/atlas-voice/` as planned? Or different org/path?
2. **LaunchAgent name** — `com.atlas-voice.server` or keep `com.pai.voice-server`?
3. **Hook strategy** — Option A (adapter-deployed hooks replace atlas-config hooks) or Option B (atlas-config hooks parameterized)?
4. **voices.json ownership** — Lives in atlas-voice repo? Or stays in atlas-config with symlink? (Recommendation: atlas-voice owns it, adapter can create symlink)
5. **ElevenLabs voice IDs** — Should `voices.json` in the public atlas-voice repo include ElevenLabs voice IDs? Or should those be in a separate private config overlay?
6. **VoiceLayer convergence timeline** — Should VoiceLayer eventually use atlas-voice as its TTS backend? Or keep them independent?
7. **Public vs private repo** — Should atlas-voice be open source? The server has no secrets; voices.json has ElevenLabs IDs (non-sensitive, but personal).

---

*Plan by Atlas for Ed, 2026-04-06. Supersedes 2026-03-11 standalone design.*
