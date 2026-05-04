# Lean Voice Helper — `atlas-speak`

**Date:** 2026-05-03
**Status:** Plan
**Replaces:** Port of VoiceServer (rejected after research)

---

## Goal

Give Atlas a voice — greeting on session start, summary on session end — without porting the 1,278-line VoiceServer framework. Keep it harness-agnostic (Claude Code today; Pi, OpenCode, others later) and lean (~200 LOC total).

## What we're not building, and why

| Old VoiceServer feature | Drop / keep | Why |
|---|---|---|
| HTTP server on `:8888` | **Drop** | No live cross-harness callers exist. Hooks calling a CLI binary is simpler and just as agent-agnostic — anything that can `exec` can use it. |
| LaunchAgent | **Drop** | No daemon, no daemon-management surface. Two stale agents are currently thrashing; we'll unload them. |
| 4-tier provider chain (edgetts → elevenlabs → kokoro → say) | **Reduce to 2** | Edge TTS + macOS `say` fallback. ElevenLabs and Kokoro are gone. |
| Circuit breakers | **Drop** | Two providers, one fallback. A try/catch covers it. |
| 13 emoji emotion markers + stability/similarity overlay | **Drop entirely** | Emoji stripping was the only real effect against Edge TTS. AvaNeural's neural prosody handles inflection naturally. |
| 15 named agent voices in `voices.json` | **Defer** | Add as frontmatter lookup when subagents are ported. For now, one voice (en-US-AvaNeural). |
| `pronunciations.json` (regex pipeline) | **Replace with typed array** | 2 entries: Kai → Kye, PAI → pie. Acronyms auto-spell correctly. |
| Per-IP rate limiter | **Drop** | Lockfile mutex covers in-process dedup. |
| Menubar widget | **Drop** | Polling overhead for a service that should "just work." |
| AppleScript notification banner via `osascript` | **Replace** | Use OSC escape sequences (Ghostty/Kitty/Wezterm native), fallback to osascript. |

**LOC:** 1,278 → ~200 (–84%).

---

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│  HARNESS-AGNOSTIC LAYER (this plan)                         │
│                                                              │
│   atlas-speak  (Bun CLI · ~/.local/bin/atlas-speak)         │
│      ├── input pipeline                                     │
│      ├── pronunciation (typed map, 2 entries)               │
│      ├── edge-tts → /tmp/atlas-*.mp3 → afplay               │
│      ├── fallback: macOS say -v 'Daniel (Enhanced)'         │
│      ├── lockfile mutex: /tmp/atlas-speak.lock              │
│      └── notification: OSC → terminal-notifier → osascript  │
└─────────────────────────────────────────────────────────────┘
                          ▲          ▲          ▲
                          │          │          │
                  ┌───────┴───┐  ┌───┴────┐  ┌──┴────────┐
                  │ Claude    │  │  Pi    │  │ OpenCode  │
                  │ Code      │  │ (later)│  │ (later)   │
                  │ hooks     │  │        │  │           │
                  └───────────┘  └────────┘  └───────────┘
                  (PHASE 1)
```

### Why a CLI binary, not a function or HTTP server

- **Function** would couple to Bun/Node — harness-specific.
- **HTTP server** would resurrect the daemon-management problem that just bit us (broken LaunchAgents thrashing).
- **CLI binary on PATH** — any process that can `exec` can speak. Bash hooks, Python scripts, Pi's adapters, future harnesses. Zero IPC.

---

## File layout

### New stow package: `voice/`

```
atlas-config/
├── claudecode/                    # existing
├── pi/                            # existing
└── voice/                         # NEW package
    ├── .stow-local-ignore
    ├── README.md
    └── .local/
        └── bin/
            └── atlas-speak        # Bun script with shebang, executable
```

After `just up`:
```
~/.local/bin/atlas-speak           # symlink → atlas-config/voice/.local/bin/atlas-speak
```

`~/.local/bin/` is already on Ed's `$PATH` (standard on macOS for user binaries). If not, the README documents adding it.

### Claude Code hooks (Phase 1 callers)

```
claudecode/.claude/hooks/
├── VoiceGreeting.ts               # NEW · SessionStart
├── VoiceSummary.ts                # NEW · Stop
└── (existing hooks unchanged)
```

Each hook is ~25–35 LOC: parse the JSON input from stdin, build a message string, `exec` `atlas-speak`, exit. No imports from the CLI binary — true loose coupling.

---

## `atlas-speak` CLI specification

### Usage

```bash
atlas-speak [options] <message>

Options:
  --title <text>     Notification banner title (default: "Atlas")
  --voice <id>       Edge TTS voice (default: en-US-AvaNeural)
  --rate <pct>       Speech rate, e.g. +0%, +10%, -5% (default: +0%)
  --no-banner        Skip the notification banner
  --no-tts           Skip TTS (banner only)
  --quiet            Skip both (no-op, useful for testing)
  -h, --help

Exit codes:
  0  spoke + notified
  1  argument error
  2  edge-tts failed AND say fallback failed
  3  lock contention timeout (another atlas-speak still running)
```

### Pipeline

```
input
  ├─ sanitize (markdown strip, 500-char cap, shell-meta strip)
  ├─ apply pronunciations (typed map, ~5 LOC)
  ├─ acquire /tmp/atlas-speak.lock (or wait 2s, then drop with code 3)
  ├─ TRY: python3 -m edge_tts --text <text> --voice <voice> --rate <rate> --write-media /tmp/atlas-XXXX.mp3
  │       └─ afplay /tmp/atlas-XXXX.mp3
  │       └─ unlinkSync /tmp/atlas-XXXX.mp3   ← (fixes the latent bug from old code)
  ├─ CATCH: say -v 'Daniel (Enhanced)' <text>
  ├─ release lock
  └─ notification banner (OSC → terminal-notifier → osascript)
```

### Pronunciation map (in code, not JSON)

```typescript
// voice/.local/bin/atlas-speak — embedded in the script
const PRONUNCIATIONS: Array<[RegExp, string]> = [
  [/\bKai\b/g, "Kye"],   // would say "Kay" otherwise
  [/\bPAI\b/g, "pie"],   // ambiguous: "pie" / "pay"
  // ISC, API, CLI are correctly spelled out by AvaNeural — no entry needed
];
```

Future addition path: when subagents are ported, each subagent's frontmatter can declare a `voiceId`:

```yaml
---
name: pentester
voiceId: en-GB-RyanNeural
voiceRate: "+5%"
---
```

A frontmatter loader (separate plan) reads this and passes `--voice` to `atlas-speak`. The CLI itself stays one voice today.

### Notification banner (cascading detection)

Detection priority (env vars set by terminal):
1. **Ghostty** — `$GHOSTTY_RESOURCES_DIR` → `OSC 9` (single string, merge title + body)
2. **Kitty** — `$KITTY_PID` → `OSC 99` (two-write protocol with title + body split)
3. **Wezterm** — `$WEZTERM_PANE` → `OSC 777` (title + body in one write)
4. **Fallback to `terminal-notifier`** if on PATH
5. **Fallback to `/usr/bin/osascript`**

Critical: write OSC bytes to `/dev/tty`, not stdout — hook stdout is often piped/captured and would swallow the sequence.

### Lockfile mutex

`/tmp/atlas-speak.lock` — created via `fs.openSync(path, 'wx')`. If file exists, wait up to 2s polling at 100ms, then exit with code 3. Released in a `finally` block. `process.on('SIGINT'|'SIGTERM'|'exit')` ensures cleanup on abnormal termination.

This replaces the old `VoiceGate` hook + per-IP rate limiter — purpose was always to prevent overlapping speech when subagents finish in parallel.

---

## Claude Code hooks (Phase 1)

### `claudecode/.claude/hooks/VoiceGreeting.ts`

```typescript
#!/usr/bin/env bun
// SessionStart — speak a greeting
import { spawn } from 'child_process';
import { readFileSync } from 'fs';

const input = JSON.parse(readFileSync(0, 'utf-8'));
const greeting = pickGreeting(input.session_id);

spawn('atlas-speak', ['--title', 'Atlas', greeting], {
  detached: true,
  stdio: 'ignore',
}).unref();

function pickGreeting(seed: string): string {
  const lines = [
    "Atlas, standing by.",
    "Ready when you are, Ed.",
    "Atlas online.",
  ];
  // Stable per-session pick (deterministic shuffle by session_id)
  const idx = seed ? [...seed].reduce((a, c) => a + c.charCodeAt(0), 0) % lines.length : 0;
  return lines[idx];
}
```

### `claudecode/.claude/hooks/VoiceSummary.ts`

```typescript
#!/usr/bin/env bun
// Stop — speak the last assistant turn's summary
import { spawn } from 'child_process';
import { readFileSync } from 'fs';

const input = JSON.parse(readFileSync(0, 'utf-8'));

// Extract a 1-line summary. Strategy: look for a 🗣️ marker first, else
// fall back to the first sentence of the last assistant message.
const summary = extractSummary(input);
if (!summary) process.exit(0);

spawn('atlas-speak', ['--title', 'Atlas', summary], {
  detached: true,
  stdio: 'ignore',
}).unref();

function extractSummary(input: any): string | null {
  // Implementation reads input.transcript_path or input.last_message
  // depending on what Claude Code's Stop hook payload provides.
  // Cap at ~120 chars for natural-sounding speech.
  // ...
}
```

Both hooks are **fire-and-forget** (`detached + unref`) — they never block Claude Code's session lifecycle. If `atlas-speak` is missing or fails, the hook exits 0 and nothing visible happens.

### Wiring in `settings.json`

Add to `claudecode/.claude/settings.json` under `hooks`:

```json
{
  "hooks": {
    "SessionStart": [
      { "matcher": "*", "hooks": [{ "type": "command", "command": "bun run ~/.claude/hooks/VoiceGreeting.ts" }] }
    ],
    "Stop": [
      { "matcher": "*", "hooks": [{ "type": "command", "command": "bun run ~/.claude/hooks/VoiceSummary.ts" }] }
    ]
  }
}
```

---

## Migration steps

1. **Cleanup (today, independent)**: Unload the two thrashing LaunchAgents.
   ```bash
   launchctl unload ~/Library/LaunchAgents/com.pai.voice-server.plist 2>/dev/null
   launchctl unload ~/Library/LaunchAgents/com.paivoice.server.plist 2>/dev/null
   rm -f ~/Library/LaunchAgents/com.pai.voice-server.plist ~/Library/LaunchAgents/com.paivoice.server.plist
   ```

2. **Pre-flight checks**:
   ```bash
   command -v bun                                      # Bun 1.3.x present
   /opt/homebrew/bin/python3 -m edge_tts --help        # edge-tts installed
   say -v '?' | rg 'Daniel'                            # Daniel (Enhanced) installed
   echo $PATH | tr ':' '\n' | rg "$HOME/.local/bin"    # ~/.local/bin on PATH
   ```

3. **Worktree the implementation**: `EnterWorktree` named `voice-helper-2026-05-03`.

4. **Build the package**:
   - `mkdir -p voice/.local/bin`
   - Write `voice/.local/bin/atlas-speak` (single Bun script, ~150 LOC)
   - `chmod +x voice/.local/bin/atlas-speak`
   - Copy `claudecode/.stow-local-ignore` → `voice/.stow-local-ignore`
   - Write `voice/README.md`

5. **Build the hooks**: `claudecode/.claude/hooks/VoiceGreeting.ts` and `VoiceSummary.ts`.

6. **Wire `settings.json`**: add SessionStart + Stop hook entries.

7. **Update `justfile`**: confirm `just up` picks up the new package automatically.

8. **Stow + test**:
   ```bash
   just up
   atlas-speak "Atlas online"
   curl -s -X POST  # ← N/A, no server now
   ```

9. **Smoke test from a real Claude Code session**: open a fresh session, confirm the greeting fires; submit a prompt, end session, confirm the summary fires.

10. **Document in `MEMORY.md`**: archive this plan to `.atlas/plans/archive/` once verified.

---

## Risks / open items

| Risk | Mitigation |
|---|---|
| Stop hook payload schema not what we expect | Build VoiceSummary.ts after confirming the actual `input` shape from a logged real Stop event. |
| Lock contention surfaces on legitimate parallel work | 2s wait is forgiving. If real parallelism causes drops, increase wait or queue. |
| Background spawn → notification banner fails because no TTY | Documented fallback chain (osascript). The banner is best-effort, not load-bearing. |
| Frontmatter voice lookup gets requested before subagents land | Documented as deferred; `atlas-speak` accepts `--voice` already, so the wiring is just a frontmatter parser. |
| Edge TTS service goes down | Circuit-of-one: edge-tts fails → say fallback. Same UX, less voice. |

## Future work (deferred, not in this plan)

- **Subagent voice lookup** — frontmatter `voiceId` field, parser util, hook integration. Triggered when the subagents directory is restored.
- **Pi / OpenCode adapters** — one shell line each (`atlas-speak "$message"`); zero work on the helper itself.
- **Volume / rate / pitch tuning** — `atlas-speak` exposes flags; tune in hooks if needed.
- **Premium voice (ElevenLabs/Azure)** — add as 3rd provider in the cascade if AvaNeural starts to feel limiting.

## File diff summary

| Path | Change |
|---|---|
| `voice/.local/bin/atlas-speak` | NEW — ~150 LOC Bun CLI |
| `voice/.stow-local-ignore` | NEW — copy from claudecode |
| `voice/README.md` | NEW — usage + integration notes |
| `claudecode/.claude/hooks/VoiceGreeting.ts` | NEW — ~25 LOC |
| `claudecode/.claude/hooks/VoiceSummary.ts` | NEW — ~35 LOC |
| `claudecode/.claude/settings.json` | EDIT — add SessionStart + Stop entries |
| `~/Library/LaunchAgents/com.pai.voice-server.plist` | DELETE (today) |
| `~/Library/LaunchAgents/com.paivoice.server.plist` | DELETE (today) |
| `.gitignore` (repo root) | NO CHANGE — no logs to gitignore |

Total NEW source: ~210 LOC. Total EDITED: ~10 lines in settings.json.
