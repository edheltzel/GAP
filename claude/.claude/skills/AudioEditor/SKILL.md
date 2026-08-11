---
name: AudioEditor
version: 1.0.22
description: "Audio editing pipeline: Whisper word-level transcription, inline agent classification supplied as JSON, ffmpeg cuts with 40ms qsin crossfades and room-tone fill, and optional Cleanvoice cloud polish. Modes: --preview, --aggressive, --polish. Workflow: Clean. USE WHEN clean audio, edit audio, remove filler words, clean podcast, remove ums, cut dead air, polish audio, trim recording, cut stutters. NOT FOR video composition (use Remotion)."
effort: medium
---

# AudioEditor

## Voice Notification

**You MUST send this notification BEFORE doing anything else when this skill is invoked.**

1. **Send voice notification**:
   ```bash
   curl -s -X POST http://localhost:8888/notify \
     -H "Content-Type: application/json" \
     -d '{"message": "Running the WORKFLOWNAME workflow in the AudioEditor skill to ACTION"}' \
     > /dev/null 2>&1 &
   ```

2. **Output text notification**:
   ```
   Running the **WorkflowName** workflow in the **AudioEditor** skill to ACTION...
   ```

**This is not optional. Execute this curl command immediately upon skill invocation.**

## What It Does

Cleans recorded audio by combining word-level transcription with an explicit edit list produced inline by the active agent. `Analyze.ts` validates that JSON, adds deterministic long-pause cuts, and passes the result to `Edit.ts`. It does not import or call a hidden model adapter. An optional Cleanvoice pass adds final polish.

## The Problem

Cleaning a recording by hand means scrubbing a waveform for every "um," half-started sentence, and three-second silence, then crossfading each cut so it does not click. A blunt auto-tool can remove rhetorical pauses or leave audible seams. This workflow keeps classification visible for review and uses deterministic validation before ffmpeg changes the recording.

## How It Works

Whisper produces word-level timestamps. The active agent reads the transcript, classifies candidate cuts inline, and writes the documented edit JSON shape. `Analyze.ts` validates those decisions and adds long-pause cuts. ffmpeg then executes the cuts with 40ms qsin crossfades and room-tone gap fill. An optional Cleanvoice API pass handles mouth-sound removal, residual filler, and loudness normalization.

### Pipeline

```
Audio Input
    |
[Transcribe] Whisper word-level timestamps (insanely-fast-whisper on MPS)
    |
[Classify] Active agent writes explicit model-edits JSON:
    |   KEEP / CUT_FILLER / CUT_FALSE_START / CUT_EDIT_MARKER / CUT_STUTTER / CUT_DEAD_AIR
    |   Distinguishes rhetorical emphasis from accidental repetition
    |
[Edit] ffmpeg executes cuts:
    |   - 40ms qsin crossfades at every edit point
    |   - Room tone extraction and gap filling
    |   - Breath attenuation (50% volume, not removal)
    |
[Polish] (optional) Cleanvoice API final pass:
        - Mouth sound removal
        - Remaining filler detection
        - Loudness normalization

Output: cleaned MP3/WAV
```

## Workflow Routing

| Workflow | Trigger | File |
|----------|---------|------|
| **Clean** | "clean audio", "edit audio", "remove filler words", "clean podcast", "remove ums", "cut dead air", "polish audio" | `Workflows/Clean.md` |

## Tools

| Tool | Command | Purpose |
|------|---------|---------|
| **Transcribe** | `bun ~/.claude/skills/AudioEditor/Tools/Transcribe.ts <file>` | Word-level transcription via an installed Whisper CLI |
| **Analyze** | `bun ~/.claude/skills/AudioEditor/Tools/Analyze.ts <transcript.json> --model-edits <json>` | Validate explicit classifications and add long pauses |
| **Edit** | `bun ~/.claude/skills/AudioEditor/Tools/Edit.ts <file> <edits.json>` | Execute cuts with crossfades and room tone |
| **Polish** | `bun ~/.claude/skills/AudioEditor/Tools/Polish.ts <file>` | Cleanvoice API cloud polish |
| **Pipeline** | `bun ~/.claude/skills/AudioEditor/Tools/Pipeline.ts <file> --model-edits <json> [--polish]` | Run the validated pipeline |

## API Keys Required

| Service | Env Var | Where to Get |
|---------|---------|-------------|
| Cleanvoice (for polish step, optional) | `CLEANVOICE_API_KEY` | cleanvoice.ai Dashboard Settings API Key |

## Examples

**Example 1: Clean a podcast recording**
```
User: "clean up the audio on this podcast file"
-> Invokes Clean workflow
-> Runs full pipeline: transcribe -> analyze -> edit
-> Outputs cleaned MP3 with filler words, stutters, and dead air removed
```

**Example 2: Preview edits before applying**
```
User: "show me what edits you'd make to this recording"
-> Invokes Clean workflow with --preview flag
-> Transcribes and analyzes, shows proposed edits without modifying audio
-> User reviews edit list, then runs again to apply
```

**Example 3: Aggressive clean with cloud polish**
```
User: "aggressively clean this audio and polish it"
-> Invokes Clean workflow with --aggressive --polish flags
-> Tighter thresholds for filler detection
-> Cleanvoice API pass for mouth sounds and normalization
```

## Gotchas

- **Transcription accuracy varies with audio quality.** Background noise, multiple speakers, and accents reduce accuracy.
- **Cut detection requires judgment.** Always review the supplied JSON and preview edits before applying them.
- **Cloud polish uploads audio to external service.** Confirm the user is okay with cloud processing for sensitive content.

Replace `WORKFLOW_USED` with the workflow executed, `8_WORD_SUMMARY` with a brief input description, and `SECONDS` with approximate wall-clock time. Log `status: "error"` if the workflow failed.
