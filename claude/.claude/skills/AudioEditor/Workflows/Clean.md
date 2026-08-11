# Clean Workflow

Clean, edit, and polish audio files by removing filler words, stutters, false starts, dead air, and edit markers.

## Voice Notification

```bash
curl -s -X POST http://localhost:8888/notify \
  -H "Content-Type: application/json" \
  -d '{"message": "Running the Clean workflow in the AudioEditor skill to clean audio"}' \
  > /dev/null 2>&1 &
```

Running the **Clean** workflow in the **AudioEditor** skill to clean audio...

## Step 1: Locate the Audio File

Identify the audio file from the user's request. Check common locations:
- Explicit path provided by user
- `~/Downloads/` for recently downloaded files
- Use `fd` to search if needed: `fd -e mp3 -e wav -e m4a -e flac '<keyword>' ~/Downloads`

If multiple matches exist, ask the user which file to use.

## Step 2: Determine Flags from Intent

Map the user's request to Pipeline.ts flags:

| User Says | Flag | Effect |
|-----------|------|--------|
| "preview", "show edits", "what would you cut" | `--preview` | Show proposed edits without executing |
| "aggressive", "tight", "heavy edit" | `--aggressive` | Tighter silence/filler thresholds |
| "polish", "cleanvoice", "final pass" | `--polish` | Cleanvoice API cloud polish (requires CLEANVOICE_API_KEY) |
| (default) | (none) | Standard cleaning with conservative thresholds |

## Step 3: Transcribe

Run the transcription tool. It checks for `insanely-fast-whisper` and `whisper` before use and fails with installation guidance when neither is installed.

```bash
bun ~/.claude/skills/AudioEditor/Tools/Transcribe.ts \
  "<audio-file-path>" \
  --output "<transcript-path>"
```

## Step 4: Classify Inline

Read the transcript and classify candidate cuts in the active agent context. Save a JSON array with this exact shape:

```json
[
  {
    "type": "CUT_FILLER",
    "start": 12.1,
    "end": 12.5,
    "reason": "Standalone filler",
    "context": "um, the next point",
    "confidence": 0.9
  }
]
```

Allowed types are `CUT_FILLER`, `CUT_FALSE_START`, `CUT_EDIT_MARKER`, `CUT_STUTTER`, `CUT_SELF_CORRECTION`, `CUT_REDO_TAKE`, and `CUT_DEAD_AIR`. Do not claim an external model call occurred. The classification is an explicit workflow artifact.

## Step 5: Validate and Apply

Pass the inline classification to the pipeline. Existing transcripts are reused, so this does not repeat transcription.

```bash
bun ~/.claude/skills/AudioEditor/Tools/Pipeline.ts \
  "<audio-file-path>" \
  --model-edits "<inline-model-edits.json>" \
  [FLAGS_FROM_INTENT_MAPPING] \
  --output "<output-path>"
```

For deterministic silence removal without any model classification, run `Analyze.ts` directly with `--pauses-only`, review the result, then pass it to `Edit.ts`.

**Output naming convention:** `<original-name>_edited.<ext>` in the same directory as the input file.

**Timeout:** Set a 10-minute timeout. Transcription of long files can take several minutes.

## Step 6: Report Results

After the pipeline completes, report the number of edits, total time removed, original and edited durations, output path, transcript path, and edit JSON path. If `--preview` was used, display the edit list and ask if the user wants to apply it.

## Individual Tool Usage

For debugging or partial workflows, individual tools can be run standalone:

```bash
# Transcription only
bun ~/.claude/skills/AudioEditor/Tools/Transcribe.ts <file>

# Validate inline classifications and add deterministic pause cuts
bun ~/.claude/skills/AudioEditor/Tools/Analyze.ts <transcript.json> --model-edits <inline-model-edits.json>

# Edit only (requires audio + edits)
bun ~/.claude/skills/AudioEditor/Tools/Edit.ts <file> <edits.json>

# Polish only (requires CLEANVOICE_API_KEY)
bun ~/.claude/skills/AudioEditor/Tools/Polish.ts <file>
```
