# PAI Hook System

> **Lifecycle event handlers that extend Claude Code with voice, memory, and security.**

This document is the authoritative reference for PAI's hook system. When modifying any hook, update both the hook's inline documentation AND this README.

---

## Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [Hook Lifecycle Events](#hook-lifecycle-events)
3. [Complete Event-to-Hook Mapping](#complete-event-to-hook-mapping)
4. [Hook Registry](#hook-registry)
5. [Non-Stowed Hooks](#non-stowed-hooks)
6. [External Hooks](#external-hooks)
7. [Inter-Hook Dependencies](#inter-hook-dependencies)
8. [Data Flow Diagrams](#data-flow-diagrams)
9. [Shared Libraries](#shared-libraries)
10. [Configuration](#configuration)
11. [Claude Code Hook Behavior](#claude-code-hook-behavior)
12. [Debugging Guide](#debugging-guide)
13. [Documentation Standards](#documentation-standards)
14. [Maintenance Checklist](#maintenance-checklist)

---

## Architecture Overview

Hooks are TypeScript scripts that execute at specific lifecycle events in Claude Code. They enable:

- **Voice Feedback**: Spoken announcements of tasks and completions
- **Memory Capture**: Session summaries, work tracking, learnings
- **Security Validation**: Command filtering, path protection, prompt injection defense
- **Context Injection**: Identity, preferences, format specifications

### Design Principles

1. **Non-blocking by default**: Hooks should not delay the user experience
2. **Fail gracefully**: Errors in one hook must not crash the session
3. **Single responsibility**: Each hook does one thing well
4. **Orchestration over duplication**: Use StopOrchestrator for shared data needs

### Execution Model

```
┌─────────────────────────────────────────────────────────────────────┐
│                        Claude Code Session                          │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  SessionStart ──┬──► StartupGreeting (banner + stats)               │
│                 ├──► LoadContext (PAI skill injection)               │
│                 ├──► CheckVersion (async, update notification)       │
│                 └──► VoiceGreeting (async, matcher: "startup")       │
│                                                                     │
│  InstructionsLoaded ──► InstructionsLoaded (context load confirm)    │
│                                                                     │
│  UserPromptSubmit ──┬──► RatingCapture (algo reminder + ratings)     │
│                     ├──► AutoWorkCreation (work directory setup)     │
│                     ├──► UpdateTabTitle (tab + voice announcement)   │
│                     └──► SessionAutoName (session naming)            │
│                                                                     │
│  PreToolUse ──┬──► SecurityValidator (Bash/Edit/Write/Read)         │
│               ├──► VoiceGate (voice curl gating, Bash)              │
│               ├──► ModelRouter (model selection, Task)               │
│               ├──► SetQuestionTab (AskUserQuestion)                  │
│               ├──► AgentExecutionGuard (Task)                        │
│               └──► SkillGuard (Skill)                                │
│                                                                     │
│  PostToolUse ──┬──► AlgorithmTracker (Bash/TaskCreate/TaskUpdate/Task)│
│                ├──► PRDSync (Write/Edit PRD.md — dashboard sync)     │
│                └──► QuestionAnswered (AskUserQuestion)               │
│                                                                     │
│  Notification ──► kanban-code/hook.sh (external board sync)          │
│                                                                     │
│  Stop ──┬──► StopOrchestrator ──┬──► VoiceNotification              │
│         │                       ├──► TabState                        │
│         │                       └──► RebuildSkill                    │
│         └──► SessionExtract (non-stowed, context extraction)         │
│                                                                     │
│  SessionEnd ──┬──► WorkCompletionLearning (insight extraction)      │
│               ├──► SessionSummary (work directory completion)       │
│               ├──► RelationshipMemory (relationship notes)          │
│               ├──► UpdateCounts (system counts + usage cache)       │
│               └──► IntegrityCheck (PAI + doc drift detection)       │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Hook Lifecycle Events

| Event | When It Fires | Source Values | Typical Use Cases |
|-------|---------------|---------------|-------------------|
| `SessionStart` | Session begins, compacts, resumes, clears | `startup`, `compact`, `resume`, `clear` | Context loading, banner, version check, voice |
| `InstructionsLoaded` | CLAUDE.md or rules files loaded | — | Logging which instruction files are active |
| `UserPromptSubmit` | User sends a message | — | Format injection, work tracking, sentiment analysis |
| `PreToolUse` | Before a tool executes | — | Security validation, gating, UI state changes |
| `PostToolUse` | After a tool executes | — | Phase tracking, PRD sync, tab state reset |
| `Notification` | Claude Code sends a notification | — | External board sync |
| `Stop` | Claude finishes a response | — | Voice feedback, tab updates, skill rebuild, extraction |
| `SessionEnd` | Session terminates | — | Summary, learning, counts, integrity checks |

### SessionStart Source Values

Claude Code fires `SessionStart` with a `source` field indicating why:

| Source | When | Hooks Should... |
|--------|------|-----------------|
| `startup` | Fresh session launch | Run fully (greet, load context, etc.) |
| `compact` | Context window compaction | Skip greetings, reload context |
| `resume` | Resuming a suspended session | Skip greetings, reload context |
| `clear` | User runs /clear | Skip greetings, reload context |

**Important**: Async hooks may not reliably read stdin (1-second timeout). If source parsing fails, source falls to `"unknown"`. Hooks that rely on source must have fallback dedup logic (e.g., PID-based dedup in VoiceGreeting).

### Matcher Semantics

The `matcher` field in `settings.json` filters when a hook fires:

- **SessionStart**: Matches against `source` value. `"startup"` = only on fresh launches. Empty = all sources.
- **PreToolUse/PostToolUse**: Matches against `tool_name`. `"Bash"` = only Bash calls. `"Edit"` = only Edit calls.
- **Stop/SessionEnd/Notification**: Matches against response content (if any). Empty = all.

### Sync vs Async Hooks

| Mode | Behavior | Use When |
|------|----------|----------|
| **Sync** (default) | Blocks until completion. stdout injected into context. | Context loading, security gating |
| **Async** (`"async": true`) | Runs in background. stdout NOT injected. | Voice notifications, version checks |

Sync hooks can influence the session by writing to stdout (context injection) or returning JSON decisions (`{ "decision": "block" }`). Async hooks are fire-and-forget.

---

## Complete Event-to-Hook Mapping

This is the exact mapping from `settings.json`, the single source of truth:

### SessionStart (3 entries)

| # | Matcher | Hook | Timeout | Async |
|---|---------|------|---------|-------|
| 1 | *(all)* | `StartupGreeting.hook.ts` | 30s | No |
| 2 | *(all)* | `LoadContext.hook.ts` | 30s | No |
| 3 | *(all)* | `kanban-code/hook.sh` | 5s | No |
| 4 | *(all)* | `CheckVersion.hook.ts` | 60s | **Yes** |
| 5 | `startup` | `VoiceGreeting.hook.ts` | 30s | **Yes** |

### InstructionsLoaded (1 entry)

| # | Matcher | Hook | Timeout | Async |
|---|---------|------|---------|-------|
| 1 | *(all)* | `InstructionsLoaded.hook.ts` | 5s | No |

### UserPromptSubmit (1 entry, 5 hooks)

| # | Matcher | Hook | Timeout | Async |
|---|---------|------|---------|-------|
| 1 | *(all)* | `RatingCapture.hook.ts` | 10s | No |
| 2 | *(all)* | `AutoWorkCreation.hook.ts` | 30s | No |
| 3 | *(all)* | `UpdateTabTitle.hook.ts` | 5s | No |
| 4 | *(all)* | `SessionAutoName.hook.ts` | 10s | No |
| 5 | *(all)* | `kanban-code/hook.sh` | 5s | No |

### PreToolUse (8 entries)

| # | Matcher | Hook | Timeout | Async |
|---|---------|------|---------|-------|
| 1 | `Bash` | `VoiceGate.hook.ts` | 5s | No |
| 2 | `Bash` | `SecurityValidator.hook.ts` | 10s | No |
| 3 | `Edit` | `SecurityValidator.hook.ts` | 10s | No |
| 4 | `Write` | `SecurityValidator.hook.ts` | 10s | No |
| 5 | `Read` | `SecurityValidator.hook.ts` | 10s | No |
| 6 | `AskUserQuestion` | `SetQuestionTab.hook.ts` | 5s | No |
| 7 | `Task` | `AgentExecutionGuard.hook.ts` | 10s | No |
| 8 | `Skill` | `SkillGuard.hook.ts` | 10s | No |
| 9 | `Task` | `ModelRouter.hook.ts` | 10s | No |

### PostToolUse (5 entries)

| # | Matcher | Hook | Timeout | Async |
|---|---------|------|---------|-------|
| 1 | `AskUserQuestion` | `QuestionAnswered.hook.ts` | 5s | No |
| 2 | `Bash` | `AlgorithmTracker.hook.ts` | 5s | No |
| 3 | `TaskCreate` | `AlgorithmTracker.hook.ts` | 5s | No |
| 4 | `TaskUpdate` | `AlgorithmTracker.hook.ts` | 5s | No |
| 5 | `Task` | `AlgorithmTracker.hook.ts` | 5s | No |
| 6 | `Write` | `PRDSync.hook.ts` | 5s | No |
| 7 | `Edit` | `PRDSync.hook.ts` | 5s | No |

### Notification (1 entry)

| # | Matcher | Hook | Timeout | Async |
|---|---------|------|---------|-------|
| 1 | *(all)* | `kanban-code/hook.sh` | 5s | No |

### Stop (2 entries)

| # | Matcher | Hook | Timeout | Async |
|---|---------|------|---------|-------|
| 1 | *(all)* | `StopOrchestrator.hook.ts` | 30s | No |
| 2 | *(all)* | `kanban-code/hook.sh` | 5s | No |
| 3 | *(all)* | `SessionExtract.ts` *(non-stowed)* | none | No |

### SessionEnd (1 entry, 6 hooks)

| # | Matcher | Hook | Timeout | Async |
|---|---------|------|---------|-------|
| 1 | *(all)* | `WorkCompletionLearning.hook.ts` | 90s | No |
| 2 | *(all)* | `SessionSummary.hook.ts` | 90s | No |
| 3 | *(all)* | `RelationshipMemory.hook.ts` | 90s | No |
| 4 | *(all)* | `UpdateCounts.hook.ts` | 15s | No |
| 5 | *(all)* | `IntegrityCheck.hook.ts` | 30s | No |
| 6 | *(all)* | `kanban-code/hook.sh` | 5s | No |

---

## Hook Registry

### SessionStart Hooks

| Hook | Purpose | Blocking | Dependencies |
|------|---------|----------|--------------|
| `StartupGreeting.hook.ts` | Display PAI banner with system stats | No | None |
| `LoadContext.hook.ts` | Inject PAI skill into context | Yes (stdout) | `skills/PAI/SKILL.md` |
| `CheckVersion.hook.ts` | Notify if CC update available | No (async) | npm registry |
| `VoiceGreeting.hook.ts` | Voice announcement on session start | No (async) | Voice Server (:8888) |

### InstructionsLoaded Hooks

| Hook | Purpose | Blocking | Dependencies |
|------|---------|----------|--------------|
| `InstructionsLoaded.hook.ts` | Log which instruction files were loaded | No | None |

### UserPromptSubmit Hooks

| Hook | Purpose | Blocking | Dependencies |
|------|---------|----------|--------------|
| `RatingCapture.hook.ts` | Algorithm format reminder + explicit/implicit rating capture | Yes (stdout) | Inference API, `ratings.jsonl` |
| `AutoWorkCreation.hook.ts` | Create/update work directories | No | `MEMORY/STATE/current-work.json` |
| `UpdateTabTitle.hook.ts` | Set tab title + voice announcement | No | Inference API, Voice Server |
| `SessionAutoName.hook.ts` | Name session on first prompt | No | Inference API, `session-names.json` |

### PreToolUse Hooks

| Hook | Purpose | Blocking | Dependencies |
|------|---------|----------|--------------|
| `SecurityValidator.hook.ts` | Validate Bash/Edit/Write/Read commands | Yes (decision) | `patterns.yaml`, `MEMORY/SECURITY/` |
| `VoiceGate.hook.ts` | Gate voice curls to primary agent only | Yes (decision) | None |
| `ModelRouter.hook.ts` | Route model selection per context | Yes (decision) | None |
| `SetQuestionTab.hook.ts` | Set teal tab for questions | No | Kitty terminal |
| `AgentExecutionGuard.hook.ts` | Guard agent spawning (Task tool) | Yes (decision) | None |
| `SkillGuard.hook.ts` | Prevent erroneous skill invocations | Yes (decision) | None |

### PostToolUse Hooks

| Hook | Purpose | Blocking | Dependencies |
|------|---------|----------|--------------|
| `AlgorithmTracker.hook.ts` | Real-time phase tracking + ISC criteria tracking | No | `lib/algorithm-state.ts` |
| `PRDSync.hook.ts` | Sync PRD frontmatter + criteria to work.json | No | `lib/prd-utils.ts` |
| `QuestionAnswered.hook.ts` | Reset tab state after question answered | No | Kitty terminal |

### Stop Hooks

| Hook | Purpose | Blocking | Dependencies |
|------|---------|----------|--------------|
| `StopOrchestrator.hook.ts` | Coordinate all Stop handlers | No | Voice Server, Kitty |

### SessionEnd Hooks

| Hook | Purpose | Blocking | Dependencies |
|------|---------|----------|--------------|
| `WorkCompletionLearning.hook.ts` | Extract learnings from work | No | Inference API, `MEMORY/LEARNING/` |
| `SessionSummary.hook.ts` | Mark work as completed | No | `MEMORY/WORK/`, `current-work.json` |
| `RelationshipMemory.hook.ts` | Capture relationship notes | No | `MEMORY/RELATIONSHIP/` |
| `UpdateCounts.hook.ts` | Update system counts + usage cache | No | `settings.json`, Anthropic API |
| `IntegrityCheck.hook.ts` | PAI change detection + doc drift detection | No | `MEMORY/STATE/integrity-state.json`, handlers/ |

---

## Non-Stowed Hooks

These hooks live directly at `~/.claude/hooks/` and are NOT managed by GNU Stow / atlas-config. They are not version-controlled in this repo.

### SessionExtract.ts (Stop hook)

- **Path**: `~/.claude/hooks/SessionExtract.ts` (1072 lines)
- **Event**: Stop
- **Purpose**: Extracts session context using `claude -p --model haiku` for persistent memory
- **Behavior**: Spawns a background child process that calls `claude -p` to summarize the session transcript
- **Key detail**: Sets `PAI_SUPPRESS_VOICE=true` and `CLAUDECODE=''` in spawned process env
- **Note**: `claude -p` (pipe mode) fires SessionStart events but Claude Code suppresses hook execution in pipe mode at the binary level — hooks do NOT run. The env var is belt-and-suspenders.

### BatchExtract.ts (standalone script)

- **Path**: `~/.claude/hooks/BatchExtract.ts` (298 lines)
- **Event**: None (run manually)
- **Purpose**: Batch extraction tool for processing multiple session transcripts
- **Usage**: `bun ~/.claude/hooks/BatchExtract.ts`

---

## External Hooks

### kanban-code/hook.sh

- **Path**: `~/.kanban-code/hook.sh`
- **Events**: SessionStart, UserPromptSubmit, Notification, Stop, SessionEnd (all events)
- **Purpose**: Syncs Claude Code session events to a Kanban board
- **Behavior**: Reads JSON from stdin, appends timestamped event line to `~/.kanban-code/hook-events.jsonl`
- **Timeout**: 5s on all events
- **Installed by**: `kanban-code` plugin (independent of PAI)

---

## Inter-Hook Dependencies

### Voice System Flow

```
SessionStart (source="startup")
    │
    ├──► VoiceGreeting ─── Layered dedup ──► "Atlas, standing by"
    │    Layer 0: PAI_SUPPRESS_VOICE env check
    │    Layer 1: CLAUDE_CODE_AGENT_TASK_ID check
    │    Layer 2: CLAUDE_AGENT_TYPE loop-worker check
    │    Layer 3: Agents/ path check
    │    Layer 4: Source + PID dedup
    │
Stop
    │
    ├──► StopOrchestrator ──► VoiceNotification ──► "🗣️ completion line"
    │    (has catchphrase dedup to avoid re-saying "standing by")
    │
PreToolUse (Bash)
    │
    └──► VoiceGate ─── Blocks voice curls from subagents
```

### Rating System Flow

```
User Message
    │
    ▼
RatingCapture ─── explicit "8 - great work"? ──► write + exit
    │ (no explicit match)
    ▼
    └── implicit sentiment (Haiku) ──────────► write
                                                │
                                                ▼
                                        ratings.jsonl
                                              │
                                              ▼
                                      Status Line Display
                                      (atlas-statusline.sh)
```

### Work Tracking Flow

```
SessionStart
    │
    ▼
UserPromptSubmit ─► AutoWorkCreation ─► Creates WORK/<date>/<session>/
    │                                          │
    │                                          ▼
    │                               current-work.json (state)
    │                                          │
    ▼                                          │
SessionEnd ─┬─► WorkCompletionLearning ────────┤
            │                                  │
            └─► SessionSummary ─► Marks as COMPLETED
```

### Security Validation Flow

```
PreToolUse (Bash/Edit/Write/Read)
    │
    ▼
SecurityValidator ─► patterns.yaml
    │
    ├─► {continue: true} ──────────────► Tool executes
    │
    ├─► {decision: "ask", message} ────► User prompted
    │
    └─► exit(2) ───────────────────────► Hard block

All events logged to: MEMORY/SECURITY/security-events.jsonl
```

### Voice + Tab State Flow

```
UserPromptSubmit
    │
    ▼
UpdateTabTitle
    ├─► Sets tab to PURPLE (#5B21B6) ─► "Processing..."
    │
    ├─► Inference summarizes prompt
    │
    ├─► Sets tab to ORANGE (#B35A00) ─► "Fixing auth..."
    │
    └─► Voice announces: "Fixing auth bug"

PreToolUse (AskUserQuestion)
    │
    ▼
SetQuestionTab ─► Sets tab to AMBER (#604800) ─► Shows question summary

Stop
    │
    ▼
StopOrchestrator
    ├─► Resets tab to DEFAULT (UL blue)
    └─► Voice announces completion
```

---

## Data Flow Diagrams

### Memory System Integration

```
┌──────────────────────────────────────────────────────────────────┐
│                         MEMORY/                                  │
├────────────────┬─────────────────┬───────────────────────────────┤
│    WORK/       │   LEARNING/     │   STATE/                      │
│                │                 │                               │
│ ┌────────────┐ │ ┌─────────────┐ │ ┌───────────────────────────┐ │
│ │ Session    │ │ │ SIGNALS/    │ │ │ current-work.json         │ │
│ │ Directories│ │ │ ratings.jsonl│ │ │ trending-cache.json       │ │
│ │            │ │ │             │ │ │ model-cache.txt           │ │
│ └─────▲──────┘ │ └──────▲──────┘ │ └───────────▲───────────────┘ │
│       │        │        │        │             │                 │
├───────┼────────┼────────┼────────┼─────────────┼─────────────────┤
│    HOOKS/      │   VOICE/        │  SECURITY/                    │
│ ┌────────────┐ │ ┌─────────────┐ │ ┌───────────────────────────┐ │
│ │hook-debug  │ │ │voice-events │ │ │security-events.jsonl      │ │
│ │  .jsonl    │ │ │  .jsonl     │ │ │                           │ │
│ └─────▲──────┘ │ └──────▲──────┘ │ └───────────▲───────────────┘ │
│       │        │        │        │             │                 │
└───────┼────────┴────────┼────────┴─────────────┼─────────────────┘
        │                 │                      │
┌───────┴─────────────────┴──────────────────────┴─────────────────┐
│                        HOOKS                                     │
│                                                                  │
│  hookLog() ──────────────────────────────► HOOKS/hook-debug.jsonl│
│  AutoWorkCreation ──────────────────────► current-work.json      │
│  RatingCapture ────────────────────────► ratings.jsonl            │
│  VoiceNotification ────────────────────► VOICE/voice-events.jsonl│
│  SecurityValidator ────────────────────► SECURITY/               │
│  WorkCompletionLearning ────────────────► LEARNING/              │
│  SessionSummary ────────────────────────► WORK/ + state          │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

---

## Shared Libraries

Located in `hooks/lib/`:

| Library | Purpose | Used By |
|---------|---------|---------|
| `hook-logger.ts` | Structured debug logging to JSONL | All hooks (opt-in) |
| `identity.ts` | Get DA name, principal from settings | Most hooks |
| `time.ts` | PST timestamps, ISO formatting | Rating hooks, work hooks |
| `paths.ts` | Canonical path construction (`paiPath()`) | Work hooks, security, logger |
| `notifications.ts` | ntfy push notifications | SessionEnd hooks, UpdateTabTitle |
| `output-validators.ts` | Tab title + voice output validation | UpdateTabTitle, TabState, VoiceNotification, SetQuestionTab |
| `learning-utils.ts` | Learning categorization | Rating hooks, WorkCompletion |
| `learning-readback.ts` | Learning signal digest for context | LoadContext |
| `change-detection.ts` | Detect file/code changes | IntegrityCheck |
| `algorithm-state.ts` | Algorithm phase tracking state | AlgorithmTracker |
| `prd-utils.ts` | PRD frontmatter/criteria parsing | PRDSync |
| `tab-constants.ts` | Tab title colors and states | tab-setter.ts |
| `tab-setter.ts` | Kitty tab title manipulation | Tab-related hooks |

### Handler Modules

Located in `hooks/handlers/` — used exclusively by StopOrchestrator:

| Handler | Purpose |
|---------|---------|
| `VoiceNotification.ts` | Extract 🗣️ line, send to voice server |
| `TabState.ts` | Reset tab to default state |
| `RebuildSkill.ts` | Rebuild SKILL.md if components changed |
| `AlgorithmEnrichment.ts` | Algorithm state enrichment |
| `SystemIntegrity.ts` | System integrity checks |
| `DocCrossRefIntegrity.ts` | Documentation cross-reference validation |
| `UpdateCounts.ts` | Usage count updates |

### Worker Modules

Located in `hooks/workers/`:

| Worker | Purpose |
|--------|---------|
| `SentimentWorker.ts` | Sentiment analysis for relationship memory |

---

## Configuration

Hooks are configured in `settings.json` under the `hooks` key. See [Complete Event-to-Hook Mapping](#complete-event-to-hook-mapping) for the full current configuration.

### Key Configuration Patterns

```jsonc
// Entry without matcher — fires for ALL events of this type
{
  "hooks": [{ "type": "command", "command": "..." }]
}

// Entry with matcher — fires only when matcher matches
{
  "matcher": "startup",  // SessionStart: matches source value
  "hooks": [{ "type": "command", "command": "..." }]
}

// Async hook — runs in background, stdout NOT injected
{
  "hooks": [{
    "type": "command",
    "command": "...",
    "timeout": 60000,
    "async": true
  }]
}
```

### Path Variables

`settings.json` supports `${PAI_DIR}` which expands to the PAI directory (set via env in settings.json: `"PAI_DIR": "/Users/ed/.claude"`). Most hooks use this for portability. Some hooks use absolute paths directly.

---

## Claude Code Hook Behavior

Critical runtime behaviors that affect hook design:

### Pipe Mode (`claude -p`) Does NOT Run Hooks

**Confirmed empirically**: When Claude Code runs in pipe mode (`claude -p`), the binary fires the `SessionStart` event internally but returns **0 hook matchers** — no hooks execute. This is binary-level suppression, not configurable. Debug logs show:

```
[DEBUG] Getting matching hook commands for SessionStart with query: startup
[DEBUG] Found 0 hook matchers in settings
[DEBUG] Matched 0 unique hooks for query "startup" (0 before deduplication)
```

Interactive sessions show `Found 6-12 hook matchers` and execute hooks normally.

**Implication**: Scripts that spawn `claude -p` (like SessionExtract.ts) do NOT trigger VoiceGreeting or any other hooks via their pipe-mode calls. The `PAI_SUPPRESS_VOICE` env var in SessionExtract is belt-and-suspenders for future-proofing.

### Subagent Hook Inheritance

| Spawn Method | Hooks Run? | Why |
|-------------|-----------|------|
| `claude -p` (pipe mode) | **No** | Binary suppresses hook matchers |
| Task tool (subagent) | **Yes** | Full Claude Code process, has `CLAUDE_CODE_AGENT_TASK_ID` |
| Agent Teams (`SendMessage`) | **Yes** | Full process, has team context |
| `claude --print` (plugin `double-shot-latte`) | **No** | Same pipe-mode suppression |

### Context Compaction Re-fires

When Claude Code compacts context (long conversations), it re-fires `SessionStart` with `source="compact"`. This means:
- LoadContext re-injects PAI context (correct behavior)
- VoiceGreeting must NOT re-greet (filtered by source check + PID dedup)
- StartupGreeting must NOT re-display banner (filtered by source check)

---

## Debugging Guide

### Structured Debug Log

All hooks that import `hookLog` write structured JSONL to:

```
~/.claude/MEMORY/HOOKS/hook-debug.jsonl
```

**Live tail** (most useful during troubleshooting):

```bash
# Watch all hook activity in real-time
tail -f ~/.claude/MEMORY/HOOKS/hook-debug.jsonl | jq .

# Filter to a specific hook
tail -f ~/.claude/MEMORY/HOOKS/hook-debug.jsonl | jq 'select(.hook == "VoiceGreeting")'

# Filter to a specific event type
tail -f ~/.claude/MEMORY/HOOKS/hook-debug.jsonl | jq 'select(.event == "SessionStart")'

# Show recent entries with timestamps
tail -20 ~/.claude/MEMORY/HOOKS/hook-debug.jsonl | jq '{ts: .ts, hook: .hook, msg: .msg}'
```

**Log entry format:**

```json
{
  "ts": "2026-03-10T09:05:00.123Z",
  "hook": "VoiceGreeting",
  "event": "SessionStart",
  "msg": "skip: PAI_SUPPRESS_VOICE=true",
  "meta": { "pid": 12345 }
}
```

**Adding logging to a hook:**

```typescript
import { hookLog } from './lib/hook-logger';

// Log a decision point
hookLog('MyHook', 'PreToolUse', 'blocked dangerous command', { tool: 'Bash', command: 'rm -rf /' });

// Log a skip reason
hookLog('MyHook', 'SessionStart', 'skip: subagent detected');
```

### Other Debug Sources

| Source | Path | What It Shows |
|--------|------|---------------|
| **Hook debug log** | `~/.claude/MEMORY/HOOKS/hook-debug.jsonl` | Structured hook decisions |
| **Voice server log** | `~/Library/Logs/pai-voice-server.log` | TTS requests, voice IDs, errors |
| **Voice events** | `~/.claude/MEMORY/VOICE/voice-events.jsonl` | Voice notification history |
| **Security events** | `~/.claude/MEMORY/SECURITY/security-events.jsonl` | Security validator decisions |
| **Claude Code debug** | `~/.claude/debug/*.txt` | Full Claude Code internal debug output |
| **Extract log** | `~/.claude/MEMORY/EXTRACT_LOG.txt` | SessionExtract activity |
| **Kanban events** | `~/.kanban-code/hook-events.jsonl` | All session events (external) |
| **Hook stderr** | Claude Code UI | Real-time stderr from hooks (visible in terminal) |

### Claude Code Debug Files

Claude Code writes debug files to `~/.claude/debug/`. These are invaluable for understanding hook execution:

```bash
# Find the most recent debug file
ls -lt ~/.claude/debug/*.txt | head -5

# Check if hooks matched for a session
rg "hook matchers" ~/.claude/debug/<session-id>.txt

# Check if a specific hook was called
rg "VoiceGreeting" ~/.claude/debug/<session-id>.txt
```

**Key patterns in debug files:**
- `Found N hook matchers in settings` — how many hooks matched this event
- `Matched N unique hooks` — after deduplication
- `Hook command completed` — hook finished executing
- Small files (~23-25KB) = pipe mode sessions. Large files (~150-1300KB) = interactive sessions.

### Common Debugging Scenarios

**"Why did a hook fire twice?"**
1. Check `hook-debug.jsonl` for the hook name — look at timestamps and PIDs
2. Check if context compaction happened (source="compact" with stdin timeout → "unknown")
3. Check Claude debug file for `SessionStart` event count

**"Why didn't a hook fire?"**
1. Verify hook is in `settings.json` under correct event
2. Check matcher — is it filtering the event out?
3. Check file is executable: `chmod +x hook.ts`
4. Check shebang: `#!/usr/bin/env bun`
5. Run manually: `echo '{"session_id":"test"}' | bun hooks/HookName.hook.ts`
6. If using `claude -p` — hooks are suppressed in pipe mode

**"Voice is firing unexpectedly"**
1. Tail the hook debug log: `tail -f ~/.claude/MEMORY/HOOKS/hook-debug.jsonl | jq 'select(.hook == "VoiceGreeting")'`
2. Tail voice server log: `tail -f ~/Library/Logs/pai-voice-server.log | rg "standing by"`
3. Check which voice ID is being used — `af_heart` = VoiceGreeting, identity mapping = VoiceNotification
4. Check PID dedup file: `cat ~/.claude/MEMORY/STATE/voice-greeted-pid`

---

## Documentation Standards

### Hook File Structure

Every hook MUST follow this documentation structure:

```typescript
#!/usr/bin/env bun
/**
 * HookName.hook.ts - [Brief Description] ([Event Type])
 *
 * PURPOSE:
 * [2-3 sentences explaining what this hook does and why it exists]
 *
 * TRIGGER: [Event type, e.g., UserPromptSubmit]
 *
 * INPUT:
 * - [Field]: [Description]
 *
 * OUTPUT:
 * - stdout: [What gets injected into context, if any]
 * - exit(0): [Normal completion]
 * - exit(2): [Hard block, for security hooks]
 *
 * SIDE EFFECTS:
 * - [File writes]
 * - [External calls]
 *
 * PERFORMANCE:
 * - [Blocking vs async]
 * - [Typical execution time]
 */
```

### Update Protocol

When modifying ANY hook:

1. Update the hook's header documentation
2. Update this README's Hook Registry section
3. Update Inter-Hook Dependencies if relationships change
4. Update Data Flow Diagrams if data paths change
5. Test the hook in isolation AND with related hooks

---

## Maintenance Checklist

### Adding a New Hook

- [ ] Create hook file with full documentation header
- [ ] Add to `settings.json` under appropriate event
- [ ] Add to Hook Registry table in this README
- [ ] Add to Complete Event-to-Hook Mapping table
- [ ] Document inter-hook dependencies
- [ ] Add `hookLog()` calls at decision points
- [ ] Test hook in isolation
- [ ] Test hook with related hooks

### Modifying an Existing Hook

- [ ] Update inline documentation
- [ ] Update this README if interface changes
- [ ] Update inter-hook docs if dependencies change
- [ ] Test modified hook
- [ ] Test hooks that depend on this hook

### Removing a Hook

- [ ] Remove from `settings.json`
- [ ] Remove from Hook Registry in this README
- [ ] Remove from Complete Event-to-Hook Mapping table
- [ ] Update inter-hook dependencies
- [ ] Check for orphaned shared state files
- [ ] Delete hook file

---

*Last updated: 2026-03-10*
*Stowed hooks: 23 | Non-stowed hooks: 2 | External hooks: 1 | Events: 8 | Shared libs: 13*
