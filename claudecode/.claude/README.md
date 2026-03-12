# PAI — Personal AI Infrastructure

> **PAI 4.0.3** — Algorithm v3.5.0-atlas

## Install

```bash
bash ~/.claude/install.sh
```

The script installs Bun (if needed), then walks you through setup:

- Your name and AI assistant name
- ElevenLabs voice configuration (optional)
- File permissions and directory structure
- Voice server startup

After the wizard finishes, activate the `pai` command and launch:

```bash
source ~/.zshrc
pai
```

## What's Inside

| Directory | Purpose |
|-----------|---------|
| `skills/` | Skill modules (PAI core + 43 domain skills) |
| `hooks/` | Lifecycle event handlers (23 stowed + 2 non-stowed + 1 external) |
| `agents/` | Named agent configurations |
| `VoiceServer/` | Multi-provider TTS server ([docs](VoiceServer/README.md)) |
| `MEMORY/` | Session history, learnings, and state |

## Documentation

- **[VoiceServer/README.md](VoiceServer/README.md)** — Voice server architecture, API, and integration guide
- **[hooks/README.md](hooks/README.md)** — Hook system reference (23 stowed hooks, 8 lifecycle events)
- **[skills/PAI/SKILL.md](skills/PAI/SKILL.md)** — Full system documentation
- **[github.com/danielmiessler/PAI](https://github.com/danielmiessler/PAI)** — Source repository
