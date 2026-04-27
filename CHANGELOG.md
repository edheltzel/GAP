# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]
### Changed
- **Consolidated plan/handoff commands into `/Standup`.** Replaces `/MigrateAtlasDir`, `/list-plans`, `/archive-plan`, and `/ReviewHandoff` across all three agent packages. `/Standup` (default) runs the daily review and auto-archives handoffs on pickup + plans on completion. `/Standup migrate` handles the legacy `.atlas-plans/` → `.atlas/` migration. `/Standup list-plans` prints a read-only listing. Manual archive command removed — archiving is automatic during flow or on request in conversation.
- **`/CreateHandoff` simplified.** Now interview-driven with frontmatter (`parent_plan`, `phase`, `status`, `percent_complete`, `next_action`, `blockers`) so handoffs integrate with the Standup board. Still runs `mem dump` on completion, now gracefully skipped when the Recall CLI (`mem`) isn't installed.
- **BREAKING convention:** project-local Atlas artifacts moved from `.atlas-plans/` to `.atlas/` namespace. Plans live at `.atlas/plans/` (archive at `.atlas/plans/archive/`), handoffs at `.atlas/handoffs/` (archive at `.atlas/handoffs/archive/`). Singular `archive/`, not `archived/`.
- Updated all rule surfaces: `CLAUDE.md` (root + `claudecode/.claude/`), `opencode/.config/opencode/AGENTS.md`, `pi-agent/.pi/agent/AGENTS.md`
- Updated all handoff + plan skills: `CreateHandoff`, `ReviewHandoff`, `list-plans`, `archive-plan` across claudecode, opencode, and pi-agent packages
- Updated `x-plan-create`, `x-plan-implement`, and `PAIUpgrade/Workflows/SyncUpstream.md`
- `claudecode/.claude/settings.json`: `plansDirectory` bumped from `./.atlas-plans` to `./.atlas/plans`
- README.md + `docs/{claude-code,opencode,pi-agent,pai-update-guide}.md` all updated
- Physical migration of atlas-config's own plans: `.atlas-plans/` → `.atlas/plans/` at repo root, and `pi-agent/.pi/agent/.atlas-plans/` → `pi-agent/.pi/agent/.atlas/plans/` (via `git mv` so history follows the rename)

### Added
- **`/meow`** anti-sycophancy correction trigger ([AgriciDaniel/meowmeow](https://github.com/AgriciDaniel/meowmeow)). One signal, four context-sensitive modes — `Continue` (resume mid-task), `Proceed` (pick and act), `Retry` (different angle), `Recheck` (re-evaluate a claim). Holds correct answers under bare skepticism; only revises on concrete evidence. Installed as a Claude Code skill at `claudecode/.claude/skills/meow/SKILL.md` (per repo's recommended path) and as an OpenCode slash command at `opencode/.config/opencode/command/meow.md`. `disable-model-invocation: true` keeps it user-only.
- **`/Standup`** unified command (Claude Code + OpenCode) and `/skill:standup` Pi skill. Default flow reviews active plans, drills into in-progress work, surfaces handoffs, and prompts for pickup (auto-archives on selection). Subcommands: `migrate`, `list-plans`.

### Removed
- `/MigrateAtlasDir`, `/list-plans`, `/archive-plan`, `/ReviewHandoff` slash commands from all three packages. Functionality merged into `/Standup` (archiving is now automatic, not an explicit step). Manual `/skill:migrate-atlas-dir` and `/skill:review-handoff` Pi skills removed as well.
- GitCleanup skill: prune stale worktrees, branches, PRs, and issues across local and remote
  - SweepLocal: remove stale worktrees and associated local branches
  - SweepRemote: delete merged remote branches via `gh` CLI (GitHub-first)
  - ClosePR: close or merge stale pull requests on GitHub/GitLab
  - ResolveConflicts: guided conflict resolution before cleanup operations
  - AuditRepo: full repo audit with ranked cleanup priority list
- Fabric pattern sync: `sanitize_broken_html_to_markdown`, `suggest_openclaw_pattern`, `suggest_gt_command`, `extract_ethical_framework`, `extract_insights_dm`, `create_design_system`, `audit_transparency`, `detect_silent_victims`, `detect_mind_virus`, `check_falsifiability`, plus `Workflows/UpdatePatterns.md`
- PI Agent skill symlinks: `karpathy-guidelines`, `seo-backlinks`, `seo-dataforseo`, `seo-google`, `seo-image-gen`, `seo-maps`
- New plan: `2026-04-10-voice-system-architectural-overhaul.md`

### Fixed
- `atlas-statusline.sh`: remove duplicate worktree tag when `.worktree.name` equals the cwd basename (the worktree folder *is* the cwd, so rendering both was redundant)
- `atlas-statusline.sh`: drop session title from header; Claude Code v2.1.114+ renders it natively as a chip above the prompt input, making the statusline copy duplicative
- `atlas-statusline.sh`: truncate pwd basename at 20 chars (tail + ellipsis) and git branch at 24 chars (middle-truncation preserves date suffixes on worktree branches like `worktree-pai-upgrade-2026-04-17`)
- `atlas-statusline.sh`: split identity (Atlas/model/cc) and location (pwd/branch/tags) onto separate lines at `micro` (35-54 cols) and `mini` (55-79 cols) modes to avoid ugly wrap when the terminal is resized below half-screen; `normal` (80+) stays inline with trailing dashes

### Changed (other)
- `.atlas-plans/` entries renamed from cryptic codenames to `YYYY-MM-DD-description.md` format per CLAUDE.md convention; 7 completed plans moved to `.atlas-plans/archive/` (superseded by the `.atlas/` namespace rename above)
- `visual-explainer` skill updated across claudecode, pi-agent, and opencode (SKILL.md + commands: share, plan-review, project-recap, generate-slides, generate-visual-plan, generate-web-diagram, diff-review, fact-check)
- `opencode` plan-review and project-recap commands refreshed

### Removed
- Obsolete `/Users/ed/.kanban-code/hook.sh` hook entries from `settings.json` (Notification, Stop, SessionStart lifecycle events)

## [4.0.34] - 2026-04-01

### Fixed
- StopOrchestrator voice gate: replaced hardcoded terminal allowlist with `!!TERM_PROGRAM` check — fixes silent voice summary failure on Ghostty and any future terminal

## [4.0.33] - 2026-03-23

### Added
- Extraction infrastructure libraries for hooks: lock, migration, quality, semaphore, tracker, pid-utils (671 lines)
- 14 SEO skill symlinks for PI Agent (seo, seo-audit, seo-competitor-pages, seo-content, seo-geo, seo-hreflang, seo-images, seo-local, seo-page, seo-plan, seo-programmatic, seo-schema, seo-sitemap, seo-technical)

### Changed
- PI Agent keybindings: namespaced with `tui.editor.` prefix, added trailing newline
- Session stats counters updated (signals, sessions, ratings)

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

