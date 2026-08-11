# Analyze.ts

Validates edit classifications supplied as JSON, adds deterministic long-pause cuts, and writes the list consumed by `Edit.ts`. It does not call a model.

## Usage

```bash
bun ~/.claude/skills/AudioEditor/Tools/Analyze.ts \
  <transcript.json> \
  --model-edits <inline-model-edits.json> \
  [--output <path>] [--aggressive]

# Deterministic pause detection only
bun ~/.claude/skills/AudioEditor/Tools/Analyze.ts <transcript.json> --pauses-only
```

## Options

| Flag | Description |
|------|-------------|
| `--model-edits <path>` | Required classification JSON produced inline by the active agent |
| `--pauses-only` | Skip classifications and emit only deterministic long-pause cuts |
| `--output <path>` | Output JSON path, defaulting beside the transcript |
| `--aggressive` | Treat pauses longer than 3 seconds as candidates instead of 5 seconds |

The model-edits input must be a JSON array. Each object needs `type`, numeric `start` and `end`, `reason`, `context`, and `confidence` from 0 to 1. Entries below 0.6 confidence are excluded.
