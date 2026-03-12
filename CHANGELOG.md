# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- VoiceServer README documentation (API, providers, config, agent integration)
- Conversion skills: `converting-claude-subagents`, `converting-slash-commands` for PAI→OpenCode porting
- `claudecode/scripts/` with `convert-pai-to-oc.ts` converter tooling
- OpenCode agent definitions (15 agents: Algorithm, Architect, Artist, BrowserAgent, etc.)
- OpenCode skills port (43 skills from Claude Code → OpenCode format)
- `pi-agent` stow package (replaces deleted `piagent`)
- Voice system section in root CLAUDE.md
- Design specs: atlas-voice-standalone, zellij-keymaps-statusbar plans

### Changed
- VoiceGreeting hook: removed PID dedup, simplified to strict startup-only greeting (-34 lines)
- Hook docs: corrected `claude -p` behavior — DOES run hooks/MCP servers (was wrongly documented as suppressed)
- Settings.json: consolidated plugins (removed 8 superpowers-marketplace entries, added superpowers@claude-plugins-official)
- OpenCode: switched build model to kimi-k2.5, added read permission, fixed find-skills symlink to absolute path
- PAI-Install README: updated hook count to 26 (23 stowed + 2 non-stowed + 1 external)
- `.claude/README.md`: updated to PAI 4.0.3, accurate hook/skill counts, added doc links
- README.md: updated voice section to reference VoiceServer with full feature list
- Justfile: added `pi-agent` to stow packages

### Removed
- Broken skill symlinks: `agent-browser`, `prompt-optimization`, `seo` (pointed to nonexistent `.agents/skills/`)
- `piagent/.pi/agent/settings.json` (replaced by `pi-agent` package)

