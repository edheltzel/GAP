# Plan: voice-notify CLI — Universal TTS Notifications for All Coding Agents

> **Scope note:** This plan covers the TTS notification layer only — the foundation of a broader audio feedback system Ed plans to build. voicemode (already installed) and agent-cli were evaluated and confirmed as complementary tools for different use cases (conversational voice and productivity toolkit respectively), not replacements for this notification system.

## Architecture (Read This First)

**There is ONE voice server: the `voice-notify` CLI binary.** It is installed ONCE at `~/.local/bin/voice-notify`, configured ONCE at `~/.config/voice-notify/`, and works for ANY coding agent that can run a shell command. Install it and forget it.

```
┌─────────────────────────────────────────────────────────────────┐
│                    voice-notify CLI (Bun binary)                │
│                 ~/.local/bin/voice-notify                       │
│                                                                 │
│  ONE binary. ONE config. THREE agents use it identically.       │
│                                                                 │
│  Config: ~/.config/voice-notify/voices.json                     │
│          ~/.config/voice-notify/pronunciations.json              │
│                                                                 │
│  TTS:    Kokoro (local) → ElevenLabs (cloud) → macOS say        │
│  Audio:  afplay with flock mutex (no overlap)                   │
│  Notify: macOS osascript notification                           │
└──────────────┬──────────────┬──────────────┬────────────────────┘
               │              │              │
     ┌─────────┴───┐  ┌──────┴──────┐  ┌────┴────────┐
     │ Claude Code  │  │  OpenCode   │  │  Pi Agent   │
     │              │  │             │  │             │
     │ hooks call:  │  │ plugin      │  │ extension   │
     │ voice-notify │  │ calls:      │  │ calls:      │
     │ "message"    │  │ voice-notify│  │ voice-notify│
     └──────────────┘  └─────────────┘  └─────────────┘
          (Team 2)         (Team 3)        (Team 4)
```

**What the teams do:**
- **Team 1** builds the ONE universal CLI binary (the voice server itself)
- **Teams 2, 3, 4** each update their agent's thin wiring to call the SAME binary

The voice server is agent-agnostic. It doesn't know or care which coding agent called it. It just speaks text.

---

## CLI Interface Contract

This is the single universal interface. Every coding agent calls the exact same binary with the exact same flags:

```
voice-notify [MESSAGE]

FLAGS:
  --message, -m TEXT      Message to speak (alt to positional arg)
  --voice-name NAME       Agent name lookup in voices.json (atlas, engineer, etc.)
  --voice-id ID           Direct ElevenLabs voice ID
  --emotion PRESET        Emotional preset (excited, celebration, insight, etc.)
  --stability F           Pass-through voice setting 0.0–1.0
  --similarity-boost F    Pass-through voice setting
  --style F               Pass-through voice setting
  --speed F               Pass-through voice setting
  --title TEXT             macOS notification title (default: "PAI")
  --no-voice              Skip TTS, notification only
  --health                Print provider status JSON, exit 0/1
  --serve                 Start HTTP wrapper on port 8888 (backwards compat)
  --config PATH           Override config path

EXIT CODES:
  0 = spoken successfully
  1 = all providers failed
  2 = bad arguments
```

**Key design decisions:**
- Circuit breakers dropped — each CLI invocation is atomic, tries full provider chain fresh
- Emoji emotion markers extracted from message text (same regex as server.ts)
- `--serve` wraps the CLI in an HTTP server for gradual migration
- Config fallback: `~/.config/voice-notify/voices.json` → `~/.claude/VoiceServer/voices.json`

---

## Stow Package Layout

New `voice-notify/` package at repo root:

```
voice-notify/
  src/                          ← NOT stowed (excluded by .stow-local-ignore)
    cli.ts                      ← main CLI entrypoint
    providers/
      kokoro.ts
      elevenlabs.ts
      say.ts
    lib/
      config.ts                 ← loadConfig(), getVoiceMapping()
      pronunciation.ts          ← loadPronunciations(), applyPronunciations()
      emotion.ts                ← EMOTIONAL_PRESETS, extractEmotionalMarker()
      sanitize.ts               ← sanitizeForSpeech(), validateInput()
      audio.ts                  ← playAudio() via afplay, flock mutex
      notify.ts                 ← macOS osascript notification
  .local/
    bin/
      voice-notify              ← compiled binary (bun build --compile), gitignored
  .config/
    voice-notify/
      voices.json               ← copied from VoiceServer/voices.json
      pronunciations.json       ← copied from VoiceServer/pronunciations.json
  .stow-local-ignore            ← excludes: src/, node_modules/, .git, .gitignore, README
```

Stows to:
- `~/.local/bin/voice-notify` (already in PATH)
- `~/.config/voice-notify/voices.json`
- `~/.config/voice-notify/pronunciations.json`

---

## Agent Teams (4 parallel worktrees)

> **Reminder:** Team 1 builds the ONE universal voice server. Teams 2–4 are just rewiring each agent to call it. The voice server itself is completely agent-agnostic — it doesn't import or reference anything from Claude Code, OpenCode, or Pi.

### Team 1: Build THE Universal voice-notify CLI
**Branch:** `feature/voice-notify-cli`
**ISC:** 1–14, 30

This is the actual voice server. One binary, installed once, works for all agents forever.

**Files created (all new — ZERO overlap):**
- `voice-notify/src/cli.ts` — main entrypoint, arg parsing, provider chain
- `voice-notify/src/providers/kokoro.ts` — port from server.ts lines 631–713
- `voice-notify/src/providers/elevenlabs.ts` — port from server.ts lines 542–628
- `voice-notify/src/providers/say.ts` — port from server.ts lines 503–540
- `voice-notify/src/lib/config.ts` — port from server.ts lines 139–189, 375–398
- `voice-notify/src/lib/pronunciation.ts` — port from server.ts lines 223–253
- `voice-notify/src/lib/emotion.ts` — port from server.ts lines 267–321
- `voice-notify/src/lib/sanitize.ts` — port from server.ts lines 412–443
- `voice-notify/src/lib/audio.ts` — port from server.ts lines 453–496 + flock
- `voice-notify/src/lib/notify.ts` — port from server.ts lines 903–911 (osascript)
- `voice-notify/.config/voice-notify/voices.json` — copy of VoiceServer/voices.json
- `voice-notify/.config/voice-notify/pronunciations.json` — copy
- `voice-notify/.stow-local-ignore`
- `voice-notify/.gitignore` — ignore `.local/bin/voice-notify` (compiled)

**Porting map from server.ts (~1125 lines):**

| server.ts | CLI destination | Lines |
|-----------|----------------|-------|
| Types & interfaces (30–91) | `lib/config.ts` types | 60 |
| Config loading (96–189) | `lib/config.ts` | 90 |
| Pronunciation (198–253) | `lib/pronunciation.ts` | 55 |
| Emotional presets (260–321) | `lib/emotion.ts` | 60 |
| Voice lookup (375–398) | `lib/config.ts` | 25 |
| Sanitize + validate (400–443) | `lib/sanitize.ts` | 45 |
| Audio playback (453–496) | `lib/audio.ts` + flock | 45 |
| MacOS say provider (503–540) | `providers/say.ts` | 40 |
| ElevenLabs provider (542–628) | `providers/elevenlabs.ts` | 85 |
| Kokoro provider (631–713) | `providers/kokoro.ts` | 80 |
| speakWithFallback (763–846) | `cli.ts` main flow | 80 |
| sendNotification (852–911) | `cli.ts` + `lib/notify.ts` | 60 |
| HTTP server (942–1101) | DROPPED (or `--serve` mode) | — |

**flock mutex implementation:**
```typescript
// In lib/audio.ts — prevent concurrent afplay
const lockPath = '/tmp/voice-notify.lock';
const lockFd = fs.openSync(lockPath, 'w');
fs.flockSync(lockFd, 'ex'); // blocks until lock acquired
try {
  await playAudio(buffer);
} finally {
  fs.flockSync(lockFd, 'un');
  fs.closeSync(lockFd);
}
```

---

### Team 2: Claude Code Wiring (point hooks at the universal CLI)
**Branch:** `feature/voice-notify-hooks`
**ISC:** 17–24

This team doesn't build voice functionality — it just changes Claude Code's hooks from `fetch('http://localhost:8888/...')` to `spawnSync('voice-notify', [...])`. Same universal CLI, just a different calling mechanism.

**Files modified (claudecode/ only — ZERO overlap with Teams 1, 3, 4):**

| File | Change |
|------|--------|
| `claudecode/.claude/hooks/VoiceGreeting.hook.ts` | Replace 2x `fetch()` → `spawnSync('voice-notify', [...])` |
| `claudecode/.claude/hooks/handlers/VoiceNotification.ts` | Replace `fetch()` → `spawnSync('voice-notify', [...])` |
| `claudecode/.claude/hooks/UpdateTabTitle.hook.ts` | Replace `fetch()` → backgrounded `spawn('voice-notify', [...])` |
| `claudecode/.claude/hooks/AlgorithmTracker.hook.ts` | Update `detectPhaseFromBash()` — add `voice-notify` command detection |
| `claudecode/.claude/hooks/VoiceGate.hook.ts` | Add `voice-notify` to detection alongside `localhost:8888` |
| `claudecode/.claude/skills/PAI/Tools/pai.ts` | Replace `notifyVoice()` fetch → `spawnSync('voice-notify', [...])` |

**Migration patterns:**

VoiceGreeting agent path (line 241):
```typescript
// BEFORE:
const resp = await fetch('http://localhost:8888/notify', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify(body),
});

// AFTER:
const args = ['--message', message];
if (voiceId) args.push('--voice-id', voiceId);
if (voice_settings) {
  if (voice_settings.stability) args.push('--stability', String(voice_settings.stability));
  if (voice_settings.similarity_boost) args.push('--similarity-boost', String(voice_settings.similarity_boost));
  if (voice_settings.style) args.push('--style', String(voice_settings.style));
  if (voice_settings.speed) args.push('--speed', String(voice_settings.speed));
}
spawnSync('voice-notify', args);
```

VoiceGreeting Atlas path (line 293) — the `/notify/personality` endpoint was a no-op shim (discarded personality, called sendNotification with null voiceId). Correct replacement:
```typescript
// AFTER:
spawnSync('voice-notify', ['--message', catchphrase, '--voice-name', 'atlas']);
```

AlgorithmTracker `detectPhaseFromBash()` (line 45–69) — **HIGHEST RISK CHANGE:**
```typescript
// BEFORE: only detected curl to localhost:8888
if (!command.includes('localhost:8888') || !command.includes('/notify')) {
  return { phase: null, isAlgorithmEntry: false };
}
const messageMatch = command.match(/"message"\s*:\s*"([^"]+)"/);

// AFTER: detect both curl (transition period) and voice-notify CLI
const isCurlVoice = command.includes('localhost:8888') && command.includes('/notify');
const isCliVoice = command.includes('voice-notify');
if (!isCurlVoice && !isCliVoice) {
  return { phase: null, isAlgorithmEntry: false };
}

let message: string | null = null;
if (isCurlVoice) {
  const m = command.match(/"message"\s*:\s*"([^"]+)"/);
  if (m) message = m[1];
} else {
  // Match: voice-notify --message "Entering the Think phase."
  // OR:    voice-notify "Entering the Think phase."
  const m = command.match(/voice-notify\s+(?:--message\s+)?["']([^"']+)["']/);
  if (m) message = m[1];
}
if (!message) return { phase: null, isAlgorithmEntry: false };
```

VoiceGate (line 67):
```typescript
// BEFORE:
if (!command.includes('localhost:8888')) { /* pass */ }

// AFTER:
const isVoiceCommand = command.includes('localhost:8888') || command.includes('voice-notify');
if (!isVoiceCommand) { /* pass */ }
```

**Blocking vs fire-and-forget:**
- VoiceGreeting: `spawnSync` is fine (async hook, won't block main session)
- VoiceNotification: `spawnSync` is fine (Stop event, sequential is correct)
- UpdateTabTitle: Use `spawn` (non-blocking) — don't stall prompt handling for voice
- AlgorithmTracker: `spawn` (fire-and-forget, `.catch(() => {})` pattern)

**Files NOT touched by Team 2 (verified no modification needed):**
- `lib/identity.ts` — read-only, no schema changes
- `lib/paths.ts` — no change
- `StopOrchestrator.hook.ts` — delegates to VoiceNotification handler, no direct voice calls

---

### Team 3: OpenCode Wiring (point plugin at the universal CLI)
**Branch:** `feature/voice-notify-opencode`
**ISC:** 25–26

Same pattern as Team 2 but for OpenCode. Replace HTTP POST with CLI spawn. The plugin calls the exact same `voice-notify` binary.

**Files modified (opencode/ only — ZERO overlap):**

| File | Change |
|------|--------|
| `opencode/.config/opencode/plugins/voice-notification.ts` | Replace all `fetch()` → `spawnSync('voice-notify')` |
| `opencode/.config/opencode/command/voice.md` | Replace curl → `voice-notify --message "$ARGUMENTS"` |

**Plugin changes:**
- Remove: `VOICE_SERVER_URL`, `VOICE_SERVER_DIR`, `isServerRunning()`, `startServer()`, `ensureServerRunning()`, health check fetch
- Replace `notify()` function body with `spawnSync('voice-notify', ['--message', message, ...opts])`
- All event handlers and `tool.after` callback remain structurally identical

---

### Team 4: Infrastructure + Agent Skill (make CLI discoverable + update docs)
**Branch:** `feature/voice-notify-infra`
**ISC:** 15–16, 27–29, 31

Creates the shared Agent Skill at `~/.agents/skills/voice-notify/SKILL.md` so ALL coding agents (including Pi) can auto-discover how to use the universal CLI. Also updates docs and build recipes.

**Files modified (no overlap with Teams 1–3):**

| File | Change |
|------|--------|
| `justfile` | Add `voice-notify` to `pkgs`, add `build-voice` recipe |
| `claudecode/.claude/VoiceServer/menubar/pai-voice.5s.sh` | Replace curl health → `voice-notify --health` |
| `claudecode/.claude/skills/PAI/Components/Algorithm/v3.5.0-atlas.md` | Replace curl → `voice-notify` CLI |
| `claudecode/.claude/skills/PAI/THENOTIFICATIONSYSTEM.md` | Replace curl templates → `voice-notify` |

**Files created:**
- `~/.agents/skills/voice-notify/SKILL.md` — cross-agent skill (NOT in repo, global install)
- Symlink: `~/.claude/skills/voice-notify` → `~/.agents/skills/voice-notify`

**Algorithm v3.5.0-atlas.md change (lines ~26, ~78):**
```markdown
<!-- BEFORE: -->
curl -s -X POST http://localhost:8888/notify \
  -H "Content-Type: application/json" \
  -d '{"message": "MESSAGE", "voice_id": "YOUR_VOICE_ID_HERE"}'

<!-- AFTER: -->
voice-notify "MESSAGE"
```

**justfile additions:**
```just
pkgs := "claudecode opencode voice-notify"

# Build voice-notify CLI binary
[group('voice')]
build-voice:
    bun build --compile --target=bun-darwin-arm64 \
      voice-notify/src/cli.ts \
      --outfile voice-notify/.local/bin/voice-notify
    echo -e "{{ green }}voice-notify built{{ clr }}"
```

**SKILL.md at `~/.agents/skills/voice-notify/SKILL.md`:**
```markdown
---
name: voice-notify
description: Speak text aloud via TTS. Use when completing tasks, greeting users, or announcing status changes.
---
# voice-notify
Invoke from Bash: `voice-notify "Your message here"`
Flags: --voice-name (atlas, engineer, etc.), --emotion (excited, celebration, focused), --title "Custom Title"
```

**After Algorithm markdown changes:** Run `bun ~/.claude/skills/PAI/Tools/RebuildPAI.ts` to rebuild SKILL.md (ISC-24).

---

## Merge Order

```
1. Team 1 (CLI)    → merge first, run `just build-voice && just update`
2. Team 4 (infra)  → merge second (justfile, skill docs, algorithm markdown)
3. Team 2 (hooks)  → merge third (claude code consumers)
4. Team 3 (opencode) → merge fourth (opencode consumers)
```

Teams 2, 3, 4 can merge in any order after Team 1 — listed sequentially for clarity but they have zero file overlap.

---

## Verification

**Per-team smoke tests (before merge):**
- Team 1: `voice-notify "Hello" && voice-notify --health && voice-notify --voice-name engineer "Test"`
- Team 2: Launch Atlas session, verify voice greeting + VoiceGate blocks subagents
- Team 3: Launch OpenCode, verify session.created speaks + /voice command works
- Team 4: `just build-voice`, verify menubar, verify Algorithm emits `voice-notify` commands

**Integration test (after all merges):**
```bash
# No more localhost:8888 references in active code
rg 'localhost:8888' claudecode/.claude/hooks/ opencode/ --glob '*.ts' --glob '*.md'
# Should return 0 matches

# Full session test
voice-notify "Integration test complete" --voice-name atlas
```

**Regression:** Verify identical audio output by comparing voice quality before/after for Atlas, Engineer, and macOS say fallback.

---

## Critical Files Reference

| Purpose | Path |
|---------|------|
| Source of all TTS logic to port | `claudecode/.claude/VoiceServer/server.ts` |
| Voice config (copy to new location) | `claudecode/.claude/VoiceServer/voices.json` |
| Pronunciation rules (copy) | `claudecode/.claude/VoiceServer/pronunciations.json` |
| Most complex consumer (agent voice branching) | `claudecode/.claude/hooks/VoiceGreeting.hook.ts` |
| Highest-risk migration (phase detection regex) | `claudecode/.claude/hooks/AlgorithmTracker.hook.ts` |
| Subagent gate (must detect new command) | `claudecode/.claude/hooks/VoiceGate.hook.ts` |
| OpenCode plugin (largest single rewrite) | `opencode/.config/opencode/plugins/voice-notification.ts` |
| Algorithm voice instructions | `claudecode/.claude/skills/PAI/Components/Algorithm/v3.5.0-atlas.md` |
| Stow recipe | `justfile` |
