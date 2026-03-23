# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [4.0.32] - 2026-03-23

### Added
- `archive-plan` and `list-plans` slash commands for Claude Code
- Plugins: context7, figma, php-lsp, lua-lsp
- `prompt-optimizer` skill for PI Agent

### Changed
- Statusline: native rate_limits JSON replaces Anthropic API call, adds worktree/agent/style indicators, exceeds_200k warning
- Statusline: removed cost/duration tracking (Max plan)
- PI Agent: version bump 0.58.1 → 0.58.4

### Removed
- `statusline-command.sh` (1430 lines of dead code)
- `ui-ux-pro-max` plugin and `extraKnownMarketplaces` config
- PI Agent `seo` skill symlink

## [4.0.31] - 2026-03-17

### Added
- **Mandatory `.atlas-plans/` directory** for all plans, handoffs, and design documents across all agents (Claude Code, OpenCode, PI Agent)
- Plans Directory section in global CLAUDE.md (Claude Code)
- Plans Directory section in AGENTS.md (PI Agent)
- Plans Directory section in OpenCode README.md
- Atlas statusline configuration

### Changed
- OpenCode plan commands (`x-plan-create`, `x-plan-implement`, `list-plans`, `archive-plan`): `docs/plans/` → `.atlas-plans/`
- Moved all existing plans from `docs/plans/` to `.atlas-plans/`
- `docs/` restructured as project documentation (agent guides, PAI update reference)
- README repo structure updated to reflect `.atlas-plans/` and `docs/` reorganization
- Settings.json version: 4.0.3 → 4.0.31

## [Unreleased]

### Added
- Algorithm v3.7.0-atlas: upgraded from v3.5.0-atlas with upstream v3.7.0 features (PRD stub, Platform Capabilities table, ISC granularity example, REFLECTIONS JSONL, "ALL WORK INSIDE" rule)
- SyncUpstream workflow in PAIUpgrade skill for upstream PAI repo syncing (diff, categorize, merge plan)
- `upstream-sync.json` state file tracking last sync point with upstream PAI repo
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

