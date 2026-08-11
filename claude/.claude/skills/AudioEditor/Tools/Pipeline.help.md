# Pipeline.ts

Audio editing pipeline that chains transcription, explicit classification validation, editing, and optional polish.

## Usage

```bash
bun ~/.claude/skills/AudioEditor/Tools/Pipeline.ts \
  <audio-file> \
  --model-edits <inline-model-edits.json> \
  [options]
```

## Options

| Flag | Description |
|------|-------------|
| `--model-edits <path>` | Required classification JSON produced inline by the active agent |
| `--polish` | Apply Cleanvoice cloud polish after editing |
| `--aggressive` | Use the tighter deterministic pause threshold |
| `--preview` | Show validated edits without executing them |
| `--output <path>` | Specify output file path |

The pipeline does not call a model. Produce the edit classifications from the transcript in the active agent context, save them as JSON, then supply that file explicitly.
