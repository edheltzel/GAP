# Atlas Voice — Implementation Plan

> Step-by-step plan for migrating VoiceServer to a standalone repo and implementing P0-P1 features.

**Design spec:** `2026-03-11-atlas-voice-standalone-design.md`
**Estimated phases:** 4 (P0 migration → P1 features → P2 observability → P3 agent-cli)

---

## Phase 0: Repo Setup & Migration

### Step 1: Create the atlas-voice repo

```bash
mkdir -p ~/Developer/AI/atlas-voice
cd ~/Developer/AI/atlas-voice
git init
```

**Files to move from `atlas-config/claudecode/.claude/VoiceServer/`:**
- `server.ts`
- `voices.json`
- `pronunciations.json`
- `install.sh` (will be rewritten)
- `uninstall.sh`
- `start.sh`
- `stop.sh`
- `restart.sh`
- `status.sh`
- `menubar/pai-voice.5s.sh`
- `menubar/install-menubar.sh`
- `logs/` (directory structure only)

**New files to create:**
- `README.md` (adapt from VoiceServer/README.md already written)
- `CLAUDE.md` (agent instructions for the new repo)
- `adapters/claude-code/` (adapter package)
- `cli/` (CLI companion stub)

**Verification:** `bun run server.ts` starts cleanly from the new location.

### Step 2: Create the Claude Code adapter

Create `adapters/claude-code/` with:

```
adapters/claude-code/
├── README.md              ← Adapter-specific docs
├── install.sh             ← Deploys hooks + config into ~/.claude/
├── uninstall.sh           ← Removes adapter from ~/.claude/
├── hooks/
│   ├── VoiceGreeting.hook.ts
│   ├── VoiceGate.hook.ts
│   └── handlers/
│       └── VoiceNotification.ts
└── voice-adapter.json     ← Template config
```

**Key change:** Hooks read server URL from `~/.claude/voice-adapter.json` instead of hardcoding `localhost:8888`.

```typescript
// Before (hardcoded)
const resp = await fetch('http://localhost:8888/notify', { ... });

// After (configurable)
import { join } from 'path';
const adapterConfig = await Bun.file(join(process.env.HOME!, '.claude', 'voice-adapter.json')).json();
const resp = await fetch(`${adapterConfig.server}/notify`, { ... });
```

**Verification:** Adapter installs cleanly, hooks fire correctly, voice works.

### Step 3: Rewrite install.sh

The new installer:

1. Checks prerequisites (Bun)
2. Creates symlink: `~/.local/share/atlas-voice/ → ~/Developer/AI/atlas-voice/`
3. Detects installed agents (check paths from agent registry)
4. For each supported agent found:
   - Run adapter's `install.sh`
   - Report success/failure
5. Install LaunchAgent (`com.atlas-voice.server`)
6. Start server
7. Print summary

**Verification:** `bash install.sh` on a clean system with Claude Code installed results in working voice.

### Step 4: Update LaunchAgent

New plist: `com.atlas-voice.server` (replaces `com.pai.voice-server`)

- Points to `~/.local/share/atlas-voice/server.ts`
- Logs to `~/Library/Logs/atlas-voice.log`
- Same KeepAlive/RunAtLoad behavior

**Verification:** `launchctl list | grep atlas-voice` shows loaded.

### Step 5: Update atlas-config

- Remove `claudecode/.claude/VoiceServer/` from the stow package
- Update `claudecode/.claude/hooks/` to use adapter pattern (read voice-adapter.json)
- Update `CLAUDE.md`, `README.md`, `hooks/README.md` to reference atlas-voice
- Add atlas-voice as a documented dependency

**Verification:** `just up` (restow) works. Voice still functions. No broken references.

### Step 6: Commit & validate

- Commit atlas-voice repo with initial files
- Commit atlas-config changes
- Full end-to-end test: start fresh session, verify greeting, verify completion voice, verify no duplication

---

## Phase 1: Core Improvements

### Step 7: Audio queue with priority

Add to `server.ts`:

```typescript
interface QueuedNotification {
  text: string;
  voiceId?: string;
  settings?: VoiceSettings;
  emotion?: string;
  priority: 'normal' | 'urgent';
  timestamp: number;
}

const audioQueue: QueuedNotification[] = [];
let isPlaying = false;

// Urgent messages jump to front of queue
// Process queue serially — only one audio plays at a time
```

- Normal messages: FIFO queue
- Urgent messages (from `[🚨 urgent]` or `priority: "urgent"` in POST body): interrupt queue, play immediately after current finishes
- Queue overflow: drop oldest non-urgent if queue > 10

**Verification:** Send 5 rapid notifications — they play sequentially without overlap.

### Step 8: CLI companion

Create `cli/voice.ts`:

```bash
voice say "hello world"        # Speak text
voice say -v kai "hello"       # Speak with agent voice
voice mute                     # Silence notifications
voice mute 30m                 # Mute for 30 minutes
voice unmute                   # Resume notifications
voice status                   # Show server health, queue, mute state
voice logs                     # Tail server log
voice logs -f                  # Follow mode
```

Install as: `bun link` → `voice` command available globally.

**Verification:** `voice status` returns clean output. `voice say "test"` plays audio. `voice mute && voice say "test"` is silent.

### Step 9: Mute endpoint

Add to `server.ts`:

```
POST /mute              → mute indefinitely
POST /mute?duration=30m → mute for 30 minutes
POST /unmute            → unmute
GET  /health            → includes mute state
```

When muted: notifications are logged but not spoken. macOS notifications still show.

**Verification:** `curl -X POST localhost:8888/mute` → next notification is silent.

### Step 10: Hot-reload config

Watch `voices.json` for changes using `Bun.file().watch()` or `fs.watch()`. On change:
- Reload provider config
- Reload agent voices
- Reload pronunciation rules
- Log: "Config reloaded (N agents, M pronunciations)"

**Verification:** Edit voices.json while server is running → new settings take effect without restart.

---

## Phase 2: Observability

### Step 11: JSONL request log

Log every notification to `~/.local/share/atlas-voice/logs/requests.jsonl`:

```json
{"ts":"2026-03-11T10:00:00Z","agent":"claude-code","message":"Task complete","provider":"kokoro","latency_ms":342,"queued_ms":0}
```

Auto-rotate: on startup, if file > 10MB, rename to `requests.YYYY-MM-DD.jsonl`.

### Step 12: /stats endpoint

```
GET /stats → {
  "uptime_hours": 4.2,
  "requests_today": 47,
  "providers": { "kokoro": 42, "say": 5 },
  "avg_latency_ms": 380,
  "queue_depth": 0,
  "muted": false
}
```

### Step 13: Web dashboard

`GET /dashboard` serves a self-contained HTML page:
- Real-time queue visualization
- Provider health indicators
- Recent notifications (last 20)
- Mute toggle button

---

## Phase 3: agent-cli Integration

### Step 14: Install agent-cli alongside

Add to `install.sh`:

```bash
# Optional: install agent-cli for enhanced voice features
read -p "Install agent-cli for system-wide hotkeys and voice editing? [y/N] " -n 1
if [[ $REPLY =~ ^[Yy]$ ]]; then
  uv tool install agent-cli -p 3.13
  agent-cli install-services
  agent-cli install-hotkeys
fi
```

### Step 15: Swap voicemode → agent-cli

- Remove `voicemode` MCP server from Claude Code settings
- agent-cli's local TTS server replaces Kokoro standalone (same API)
- agent-cli's `transcribe` replaces voicemode's STT
- agent-cli's `assistant` or `chat` replaces `/voicemode:converse`

### Step 16: Voice-to-text integration

Add `POST /listen` endpoint that triggers agent-cli's transcription:

```bash
# Start recording → returns transcript
curl -X POST localhost:8888/listen
# → {"transcript": "Fix the authentication bug in the login component"}
```

Uses agent-cli's Whisper service under the hood.

---

## Rollout Strategy

| Phase | Scope | When |
|-------|-------|------|
| **P0** | Repo setup, migration, adapter | First |
| **P1** | Audio queue, CLI, mute, hot-reload | After P0 stable |
| **P2** | Logging, stats, dashboard | After P1 stable |
| **P3** | agent-cli integration | After P2 stable |

Each phase is a clean commit boundary. P0 must be fully validated before P1 begins.

---

*Implementation plan by Atlas + Ed, 2026-03-11*
