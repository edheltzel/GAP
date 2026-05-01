# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

### Changed — 2026-05-01: Soft reset to minimal config

- **Wiped pre-built Atlas/PAI build.** Removed the entire `claudecode/.claude/` tree (skills, hooks, VoiceServer, Observability, MEMORY, agents, lib, statusline, etc.) along with the `opencode/` and `pi-agent/` packages. Going forward this repo cherry-picks individual pieces of config rather than shipping a framework.
- **Stow packages reduced to two:** `claudecode` and `pi`. The `pi` package is currently an empty placeholder that will be filled in as the Pi agent is reconfigured. `opencode` removed (Ed no longer uses OpenCode).
- **`justfile` updated:** `pkgs := "claudecode pi"`.
- **`.gitignore` simplified:** dropped paths for now-deleted directories (Observability, pai_updates/backups, VisualExplainer, opencode superpowers, etc.).
- **Top-level docs rewritten:** `README.md` and `CLAUDE.md` are now short, minimal, and accurate. `docs/` directory removed (claude-code.md, opencode.md, pi-agent.md, pai-update-guide.md were all stale).
- **Plans archived:** every active plan in `.atlas/plans/` referenced systems that were deleted in this reset (voice, PAI, multi-agent, three-agent eval, agent-workflow orchestration). All moved to `.atlas/plans/archive/` for reference. The active handoff was archived to `.atlas/handoffs/archive/`.

### Kept

- `.atlas/plans/archive/` history (research, decisions, rationale).
- `.atlas/plans/assets/` (visual outputs from prior planning).
- `prompts/design-system-bootstrap.md` (reusable, framework-agnostic).
- `eldritch-maestro-theme.json` (theme reference).
- Git history (the deletion is a single explicit commit, not a force-push).
