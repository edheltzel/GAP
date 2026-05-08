# Atlas Voice — Standalone Agent-Agnostic Voice Server

> Design spec for migrating VoiceServer from a Claude-Code-specific component to a standalone, agent-agnostic application.

**Status:** Design complete, awaiting implementation
**Date:** 2026-03-11
**Source repo:** `~/Developer/AI/atlas-voice/` (to be created)
**Runtime location:** `~/.local/share/atlas-voice/` (symlinked from source)

---

## 1. Problem Statement

The VoiceServer currently lives inside `~/.claude/VoiceServer/` and is deployed via GNU Stow as part of the atlas-config repo. While the server's HTTP API is already agent-agnostic (any process can POST JSON), the deployment, lifecycle, and integration are coupled to the Claude Code ecosystem. As AI coding agents proliferate (Claude Code, OpenCode, Gemini CLI, Codex, Cursor, Windsurf, Factory.ai Droid), the voice server should be a standalone application that any agent can use with minimal setup.

Additionally, the current voice conversation system uses `voicemode` (MCP server) for bidirectional voice. `agent-cli` provides a superset of voice features with better DX (system-wide hotkeys, service management, transcription, voice editing, wake word). The migration should support swapping voicemode for agent-cli.

## 2. Architecture

### 2.1 Deployment Model

```
~/Developer/AI/atlas-voice/          ← Source repo (git, development)
    ├── server.ts                    ← Voice server (Bun)
    ├── voices.json                  ← Voice + provider config
    ├── pronunciations.json          ← TTS pronunciation overrides
    ├── install.sh                   ← Agent-aware installer
    ├── adapters/                    ← Per-agent integration packages
    │   ├── claude-code/             ← Hooks + config for Claude Code
    │   ├── opencode/                ← Event handlers for OpenCode (future)
    │   └── generic/                 ← Shell script fallback (future)
    ├── cli/                         ← CLI companion (voice say/mute/status)
    └── docs/                        ← Standalone documentation
         │
         │  install.sh
         │  detects agents → deploys adapters
         ▼
    ~/.local/share/atlas-voice/      ← Runtime (symlinked from source)
         │
         │  per-agent adapter install
         ▼
    ~/.claude/voice-adapter.json     ← Claude Code adapter config
    ~/.claude/hooks/Voice*.hook.ts   ← Claude Code hooks (from adapters/)
    ~/.config/opencode/voice.json    ← OpenCode adapter config (future)
```

### 2.2 Key Principles

1. **One server, many consumers** — Single process on port 8888, adapters teach each agent how to talk to it
2. **Symlink from source** — `install.sh` symlinks the repo to `~/.local/share/atlas-voice/` for live development
3. **Detect, don't assume** — Installer probes for known agent config directories to determine what's installed
4. **Tested adapters only** — Ship Claude Code adapter first. Add others as built and verified. No generic half-working fallbacks.
5. **Full deployment** — Each adapter includes everything needed: hooks, config, and integration scripts. Working out of the box.

### 2.3 Server Architecture (unchanged)

The server itself (`server.ts`) requires no changes — it's already agent-agnostic. The HTTP API stays the same:

| Method | Path | Purpose |
|--------|------|---------|
| POST | `/notify` | Primary TTS notification |
| POST | `/notify/personality` | Compatibility shim |
| POST | `/pai` | Simple notification alias |
| GET | `/health` | Provider status + circuit breakers |

## 3. Agent Detection & Adapter Deployment

### 3.1 Agent Registry

```typescript
interface AgentDefinition {
  name: string;
  detectPaths: string[];      // Check these paths to confirm agent is installed
  adapterDir: string;         // Source adapter in adapters/{name}/
  targetDir: string;          // Where to deploy config/hooks
  supported: boolean;         // false = detected but no adapter yet
}

const AGENTS: AgentDefinition[] = [
  {
    name: 'claude-code',
    detectPaths: ['~/.claude/settings.json'],
    adapterDir: 'adapters/claude-code/',
    targetDir: '~/.claude/',
    supported: true,
  },
  {
    name: 'opencode',
    detectPaths: ['~/.config/opencode/'],
    adapterDir: 'adapters/opencode/',
    targetDir: '~/.config/opencode/',
    supported: false,  // roadmap
  },
  {
    name: 'gemini-cli',
    detectPaths: ['~/.gemini/'],
    adapterDir: 'adapters/gemini-cli/',
    targetDir: '~/.gemini/',
    supported: false,  // roadmap
  },
  {
    name: 'codex',
    detectPaths: ['~/.codex/'],
    adapterDir: 'adapters/codex/',
    targetDir: '~/.codex/',
    supported: false,  // roadmap
  },
];
```

### 3.2 Installer Flow

```
install.sh
  │
  ├── 1. Check prerequisites (Bun installed?)
  │
  ├── 2. Symlink repo → ~/.local/share/atlas-voice/
  │
  ├── 3. Detect installed agents
  │     ├── Found: claude-code (supported) ✅
  │     ├── Found: opencode (no adapter yet) ⚠️
  │     └── Not found: gemini-cli, codex
  │
  ├── 4. Deploy adapters for supported agents
  │     └── Claude Code:
  │           ├── Copy hooks → ~/.claude/hooks/
  │           ├── Write voice-adapter.json → ~/.claude/
  │           ├── Merge hook entries into settings.json
  │           └── Verify: curl localhost:8888/health
  │
  ├── 5. Install LaunchAgent (com.atlas-voice.server)
  │     └── Points to ~/.local/share/atlas-voice/server.ts
  │
  ├── 6. Start server
  │
  └── 7. Summary: what was installed, what's pending
```

### 3.3 Claude Code Adapter

The Claude Code adapter deploys these files:

| File | Target | Purpose |
|------|--------|---------|
| `VoiceGreeting.hook.ts` | `~/.claude/hooks/` | Startup catchphrase |
| `VoiceGate.hook.ts` | `~/.claude/hooks/` | Subagent voice blocking |
| `VoiceNotification.ts` | `~/.claude/hooks/handlers/` | Completion speech |
| `voice-adapter.json` | `~/.claude/` | Server URL, default voice, agent identity |

The adapter reads `voice-adapter.json` for server location instead of hardcoding `localhost:8888`:

```json
{
  "server": "http://localhost:8888",
  "identity": {
    "name": "Atlas",
    "voice_id": "JBFqnCBsd6RMkjVDRZzb",
    "catchphrase": "Atlas, standing by"
  },
  "suppress_env": "PAI_SUPPRESS_VOICE"
}
```

## 4. Voice Conversation Provider Swap (voicemode → agent-cli)

### 4.1 Current State (voicemode)

- MCP server providing `converse` and `service` tools
- Installed as Claude Code plugin or via `uvx voice-mode`
- Uses Whisper (local or OpenAI) for STT, Kokoro/OpenAI for TTS
- Tight coupling: MCP protocol, Claude Code-specific

### 4.2 Target State (agent-cli)

`agent-cli` provides everything voicemode does plus:

| Feature | voicemode | agent-cli |
|---------|-----------|-----------|
| TTS | Kokoro/OpenAI | Kokoro/Piper + local server with OpenAI-compatible API |
| STT | Whisper | Whisper (MLX on Apple Silicon, faster-whisper on Linux) |
| Hotkeys | None | System-wide Cmd+Shift+R (transcribe), Cmd+Shift+A (autocorrect), Cmd+Shift+V (voice edit) |
| Service mgmt | Manual | `agent-cli install-services`, `agent-cli start-services` |
| Voice editing | No | `agent-cli voice-edit` — edit clipboard text with voice commands |
| Transcription | Live in conversation only | `transcribe` (one-shot), `transcribe-live` (continuous), clipboard integration |
| Wake word | No | `agent-cli assistant` — wake word triggers listening |
| Memory | No | Built-in memory system with RAG proxy |
| Dev workflow | No | `agent-cli dev` — parallel AI agents in git worktrees |

### 4.3 Migration Path

**Phase 1: Coexistence** — Install agent-cli alongside voicemode. Use agent-cli for hotkeys and transcription. VoiceServer continues using Kokoro directly for TTS.

**Phase 2: Provider swap** — Point VoiceServer's Kokoro provider at agent-cli's local TTS server (same OpenAI-compatible API, potentially on the same port 8880 or configurable). Agent-cli's `server` command manages Whisper + TTS lifecycle.

**Phase 3: Full migration** — Remove voicemode MCP server from Claude Code settings. agent-cli's `assistant` or `chat` replaces `/voicemode:converse`. Update atlas-voice `install.sh` to offer agent-cli setup as part of the install flow.

### 4.4 Configuration

`voices.json` provider config supports this swap without code changes:

```jsonc
{
  "providers": {
    "kokoro": {
      "enabled": true,
      // During coexistence: point at Kokoro's native server
      "endpoint": "http://127.0.0.1:8880/v1",
      // After swap: point at agent-cli's TTS server (same API)
      // "endpoint": "http://127.0.0.1:8880/v1",  // agent-cli uses same port
    }
  }
}
```

The swap is transparent because both Kokoro (standalone) and agent-cli's TTS server speak the same OpenAI-compatible `/v1/audio/speech` API.

## 5. Feature Roadmap

### P0: Migration Foundation (this spec)

- [ ] Create `~/Developer/AI/atlas-voice/` git repo
- [ ] Move server.ts, voices.json, pronunciations.json, lifecycle scripts
- [ ] Create installer with agent detection
- [ ] Create Claude Code adapter
- [ ] Symlink deployment to `~/.local/share/atlas-voice/`
- [ ] Update LaunchAgent to use new path
- [ ] Update atlas-config to reference atlas-voice instead of bundling VoiceServer

### P1: Core Improvements

- [ ] **Audio queue with priority** — FIFO queue with urgent message interruption. Prevents audio overlap when multiple agents/hooks fire simultaneously.
- [ ] **CLI companion** — `voice say "hello"`, `voice mute`, `voice unmute`, `voice status`, `voice logs`. Makes VoiceServer a daily-use tool.
- [ ] **Do Not Disturb / mute** — `POST /mute` and `POST /unmute` endpoints. Time-based auto-unmute. For meetings and focus time.
- [ ] **Hot-reload config** — Watch `voices.json` for changes, reload without restart.

### P2: Observability

- [ ] **JSONL request log** — Every notification logged with timestamp, agent, message, provider, latency.
- [ ] **`GET /stats` endpoint** — Aggregated metrics: requests today, provider breakdown, avg TTS latency.
- [ ] **Web dashboard** — `GET /dashboard` serves local HTML: active queue, recent notifications, provider health.

### P3: agent-cli Integration

- [ ] **Install agent-cli** as part of atlas-voice setup (optional)
- [ ] **System-wide hotkeys** via agent-cli: Cmd+Shift+R for transcription
- [ ] **Swap voicemode → agent-cli** for voice conversations
- [ ] **Voice-to-text input** — `POST /listen` starts recording via agent-cli's Whisper, returns transcript

### P4: Advanced Features

- [ ] **Streaming TTS** — Chunked audio playback for long messages (reduced perceived latency)
- [ ] **Ambient status sounds** — Subtle audio cues for test pass/fail, build complete, agent spawn
- [ ] **Per-project voice profiles** — Different voice tones per project context
- [ ] **Custom emotional presets** — User-defined emoji→tone mappings
- [ ] **Pronunciation editor** — Web UI to add/test pronunciation rules

### Future Agent Adapters (rolling releases)

- [ ] OpenCode adapter
- [ ] Codex CLI adapter
- [ ] Gemini CLI adapter
- [ ] Factory.ai Droid adapter
- [ ] Cursor adapter
- [ ] Windsurf adapter

## 6. What Changes in atlas-config

After migration:

| Before | After |
|--------|-------|
| `claudecode/.claude/VoiceServer/` contains server code | Directory replaced with symlink or removed; server lives in atlas-voice repo |
| `VoiceServer/voices.json` is stow-managed | `voices.json` lives in atlas-voice, symlinked to runtime |
| Voice hooks hardcode `localhost:8888` | Hooks read `voice-adapter.json` for server URL |
| `install.sh` bundles voice server setup | atlas-voice has its own `install.sh` |
| LaunchAgent plist in VoiceServer/ | LaunchAgent in atlas-voice |

atlas-config's `claudecode/.claude/VoiceServer/` becomes either:
- A symlink to `~/.local/share/atlas-voice/` (if you want Stow to manage it)
- Removed entirely (if atlas-voice manages its own deployment)

Recommended: **remove from atlas-config**, let atlas-voice own its lifecycle. atlas-config's hooks continue to work because the adapter deploys them.

---

*Design by Atlas + Ed, 2026-03-11*
