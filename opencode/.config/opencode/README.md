# OpenCode Configuration

Personal OpenCode configuration with custom commands, plugins, and themes.

> For Claude Code configuration, see [`~/.claude/README.md`](../../.claude/README.md)

## Gotchas

### Custom Tools/Plugins: Missing `@opencode-ai/plugin`

If you see this error:

```
error: Cannot find module '@opencode-ai/plugin' from '.../tool/your_tool.ts'
```

You need to install the plugin package in the opencode config directory:

```bash
cd ~/.dotfiles/atlas/.config/opencode && bun add @opencode-ai/plugin
```

This applies to both `tool/` and `plugin/` directories when importing from `@opencode-ai/plugin`.

Since we use Stow, `package.json` and `bun.lock` exist in this directory (ignored by stow).

---

## Hey Future Ed

Just in case you forget, plugins in the `plugin/` directory are **automatically loaded** by OpenCode.
There is no need to register them in `opencode.jsonc`.

See: https://opencode.ai/docs/plugins#location

## Voice Notifications

OpenCode integrates with the shared VoiceServer (same server Claude Code uses) for TTS notifications.

### Features

- **Session start**: Speaks startup catchphrase when OpenCode launches
- **Task completion**: Announces when agent goes idle
- **Permission requests**: Alerts when approval is needed
- **Test results**: Announces pass/fail for test commands

### Configuration

Voice settings are read from `~/.claude/settings.json` (shared with Claude Code):

```json
{
  "daidentity": {
    "voiceId": "your-elevenlabs-voice-id",
    "startupCatchphrase": "Hello, ready to help.",
    "voice": {
      "stability": 0.35,
      "similarity_boost": 0.8
    }
  }
}
```

### Manual Voice Command

```bash
/voice Hello world  # Speak arbitrary text
```

### Troubleshooting

```bash
# Check if VoiceServer is running
curl http://localhost:8888/health

# Start VoiceServer manually
~/.claude/VoiceServer/start-server.sh

# View server logs
tail -f /tmp/voice-server.log
```

### Files

- `plugins/voice-notification.ts` - Main voice plugin (auto-loaded)
- `command/voice.md` - Manual /voice command
- `~/.claude/VoiceServer/` - Shared voice server (ElevenLabs + macOS fallback)

---

## Structure

- `opencode.jsonc` - Main configuration (model, MCP servers, theme)
- `agent/` - Custom sub-agents for specialized tasks
- `command/` - Custom slash commands (`/commit`, `/commit-push`, `/voice`, etc.)
- `plugins/` - Autoloaded plugins (voice notifications, etc.)
- `skill/` - Loadable skills (frontend-design, etc.)
- `themes/` - Custom color themes (Eldritch variants)
- `tool/` - Custom tools

## Setup

### Per-Project

```bash
curl -o opencode.json https://raw.githubusercontent.com/edheltzel/dotfiles/main/ai/.config/opencode/opencode.json
```

### Global

```bash
mkdir -p ~/.config/opencode
curl -o ~/.config/opencode/opencode.json https://raw.githubusercontent.com/edheltzel/dotfiles/main/ai/.config/opencode/opencode.json
```

## Permissions Format

OpenCode uses a permission object with `"allow"`, `"ask"`, or `"deny"` values:

```json
{
  "permission": {
    "bash": {
      "git *": "allow",
      "rm -rf /": "deny",
      "*": "ask"
    }
  }
}
```

| Goal                   | Example                 |
| ---------------------- | ----------------------- |
| Allow all Git commands | `"git *": "allow"`      |
| Block specific command | `"rm -rf /": "deny"`    |
| Prompt for approval    | `"docker run *": "ask"` |

## What's Allowed and Blocked

### Allowed

The config includes ~250 command patterns covering tools most devs actually use:

- **Languages & Package Managers:** Node (npm, yarn, pnpm, bun), Python (pip, poetry), Rust (cargo), Go, Java, Ruby, and PHP.
- **Infrastructure & Cloud:** Docker, Kubernetes (kubectl, helm), AWS, Google Cloud, Azure, Vercel, and Netlify.
- **Git & GitHub:** Full git operations and GitHub CLI (`gh`).
- **Database Tools:** Postgres, MySQL, MongoDB, Redis, and SQLite.
- **System Tools:** File manipulation, text processing (grep, sed, jq), SSH, curl, wget.

### Blocked

These are blocked to prevent you from accidentally nuking something:

- **Destructive Acts:** Recursive root deletion (`rm -rf /`), disk formatting, kernel mods.
- **Docker Risks:** Privileged containers, host socket mounting.
- **Malicious Activity:** Reverse shells, crypto mining, credential theft.
- **User Admin:** Password changes, user account modification.

## Customizing

Edit `opencode.jsonc` to add or modify patterns.

**Tip:** Put private servers and domains in local config files so they don't end up in version control.

## How Settings are Applied

Settings merge using deep merge strategy:

1. **Environment Variable:** `OPENCODE_CONFIG` path
2. **Custom Directory:** `OPENCODE_CONFIG_DIR`
3. **Project:** `opencode.json` in project root
4. **Global:** `~/.config/opencode/opencode.json`

## Security Notes

1. **Sudo:** Only allowed for package managers (`sudo apt install`, etc). Remove those patterns if you don't need them.
2. **Permissiveness:** These settings lean loose so you're not approving every command. Tighten them up if your environment requires it.

## Links

- [Configuration Docs](https://opencode.ai/docs/config)
- [Permissions Docs](https://opencode.ai/docs/permissions)
