# PAI Config

Personal AI Infrastructure configuration managed with [GNU Stow](https://www.gnu.org/software/stow/).

PAI files live here and get symlinked into `~/.claude/` — keeping PAI config version-controlled and separate from Claude Code's runtime files.

## Structure

```
pai-config/
├── .claude/           ← PAI config (symlinked into ~/.claude/)
│   ├── settings.json  ← hooks, identity, statusline config
│   ├── CLAUDE.md      ← PAI context injection
│   ├── skills/        ← 37 PAI skills
│   ├── hooks/         ← 20 lifecycle hooks
│   ├── agents/        ← 13 agent personas
│   ├── MEMORY/        ← persistent memory (WORK, STATE, LEARNING)
│   ├── VoiceServer/   ← TTS service
│   ├── lib/           ← shared utilities
│   └── PAI-Install/   ← installer
└── makefile           ← stow commands
```

Claude Code runtime files (`debug/`, `todos/`, `projects/`, `history.jsonl`, etc.) stay as real files in `~/.claude/` and are **not** tracked here.

## Usage

```sh
make help      # show all commands
make link      # first-time setup — create symlinks
make up        # restow after adding/removing files
make status    # dry run + show current symlinks
make unlink    # remove all symlinks
make push      # commit and push changes
```

## Requirements

- [GNU Stow](https://www.gnu.org/software/stow/) (`brew install stow`)
- [PAI](https://github.com/danielmiessler/PAI) v3.0+
