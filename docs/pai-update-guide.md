# PAI Update Guide

Reference for safely applying upstream PAI updates to Atlas-config.

## Local Modifications (PAIUpdate Protection)

The following files contain **local customizations** that must NOT be overwritten during a PAI update (`/pai-upgrade`). When pulling upstream PAI changes, preserve these modifications:

| File                                    | What Was Customized                                                                                                                                                                                                                                                                                                                                   |
| --------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `claudecode/.claude/skills/PAI/Tools/Banner.ts`    | Fully rewritten with Eldritch theme. WezTerm-native width detection (`wezterm cli list`). Animated asciigen.art planet frames — **high quality**, downsampled to 1/3 size (14 rows × ~39 cols, ROW_STEP=3 COL_STEP=3), single gray, 50 consecutive frames starting at frame 90 (North America). Falls back to raycast sphere when asciify dir absent. |
| `claudecode/.claude/skills/PAI/Tools/pai.ts`       | Added `--dangerously-skip-permissions` flag to `cmdLaunch()`. Removed hardcoded voice notification (handled by async hook instead).                                                                                                                                                                                                                   |
| `claudecode/.claude/hooks/StartupGreeting.hook.ts` | Removed banner display from hook (banner only shows via `pai.ts` CLI path). Kitty env persistence code is inactive (Ed uses WezTerm).                                                                                                                                                                                                                 |
| `claudecode/.claude/hooks/VoiceGreeting.hook.ts`   | **New file** — async SessionStart hook (matcher: `startup`) that speaks the startup catchphrase for ALL launch paths (`pai`, `atlas`, `claude`). Only fires on new sessions, not on context compaction/resume/clear.                                                                                                                                   |
| `claudecode/.claude/settings.json`                 | Registered `VoiceGreeting.hook.ts` as an async SessionStart hook with `"matcher": "startup"`. `statusLine.command` points to `atlas-statusline.sh` (not upstream `statusline-command.sh`). `plansDirectory` set to `./.atlas-plans` (project-local, Atlas-branded; avoids tilde-expansion bug). |
| `claudecode/.claude/hooks/RatingCapture.hook.ts`   | `ALGORITHM_REMINDER` banner (line ~60): removed `github.com/danielmiessler/TheAlgorithm` URL from the hardcoded banner string. Keeps version `(v0.4.9)` but strips upstream branding. If upstream changes the banner format, merge carefully — preserve the URL removal. |
| `claudecode/.claude/atlas-statusline.sh`           | **User-owned statusline** — full copy of upstream `statusline-command.sh` with color palette remapped to Eldritch theme (`#04d1f9` cyan, `#37f499` green, `#a48cf2` purple). Do NOT overwrite with upstream during PAIUpdate; instead diff and merge the logic sections while keeping the COLOR PALETTE block.                                        |
| `~/.zshrc`                              | Added `atlas` alias alongside `pai` alias.                                                                                                                                                                                                                                                                                                            |

**During PAIUpdate:** If upstream changes touch any of these files, merge carefully — do not blindly replace. The voice greeting architecture depends on the hook being async and `pai.ts` NOT having its own voice call. For `atlas-statusline.sh`: only merge changes to sections OTHER than `# COLOR PALETTE` — that block is intentionally Eldritch-themed.

## Upstream Sync Workflow

The PAIUpgrade skill now includes a **SyncUpstream** workflow for syncing atlas-config with the upstream PAI repo. Triggers: `"sync upstream"`, `"check PAI repo"`, `"sync PAI"`, `"merge upstream"`.

- **State:** `claudecode/.claude/skills/PAIUpgrade/State/upstream-sync.json` — tracks last sync ref, tag, and date
- **Protected files:** Algorithm fork, Banner.ts, settings.json, VoiceGreeting/VoiceGate hooks
- **New Release Mode:** Auto-detects new release tags and new release directories

Use this workflow instead of manual diffing when upstream PAI updates arrive.

## v4.0.3 Integration Decisions (2026-03-02)

### Algorithm: v3.7.0-atlas (upgraded 2026-03-16 from v3.5.0-atlas)

- **Component file:** `claudecode/.claude/skills/PAI/Components/Algorithm/v3.7.0-atlas.md`
- **LATEST pointer:** Updated to `v3.7.0-atlas`
- Based on upstream v3.7.0, with all Atlas customizations preserved
- v3.5.0-atlas file retained as archive (no longer active)

**Atlas customizations preserved in v3.7.0-atlas:**
- `YOUR_VOICE_ID_HERE` placeholder (not Daniel's hardcoded voice_id)
- `~/.claude/MEMORY/WORK/` absolute paths (upstream still uses bare relative paths)
- `~/.claude/skills/PAI/PRDFORMAT.md` path (Atlas skill layout)
- Atlas Extensions section (Loop Mode, Agent Teams)
- VERIFY LOAD-BEARING ASSUMPTIONS + DISAMBIGUATION in OBSERVE
- "Test each change individually" in EXECUTE
- Entry banner in Algorithm file (upstream moved it to CLAUDE.md)

**New from upstream v3.7.0:**
- PRD stub at Algorithm entry (immediate dashboard visibility)
- Platform Capabilities table (/simplify, /batch, Agent Teams, etc.)
- ISC granularity example (blog publishing decomposition)
- "ALL WORK INSIDE THE ALGORITHM" explicit rule
- REFLECTIONS JSONL in LEARN phase
- RPG scenario examples for capability selection
- RLM paper citation on context compaction
- ISC Count Gate data point (50 PRDs analysis)

### BuildCLAUDE.ts: DEFERRED

- v4.0.3 generates `CLAUDE.md` dynamically from `CLAUDE.md.template` + `settings.json` variables
- Atlas uses `contextFiles` array in settings.json + `SKILL.md` (built by RebuildPAI.ts) — functionally equivalent
- Adopting BuildCLAUDE.ts would require creating an Atlas-specific template, migrating from contextFiles, and maintaining two template systems (RebuildPAI + BuildCLAUDE)
- **Decision:** Keep current RebuildPAI.ts + contextFiles approach. Revisit if upstream PAI makes BuildCLAUDE.ts the only supported path

### NATIVE Mode: ADOPTED (via Algorithm v3.5.0-atlas)

- v4.0.3's NATIVE mode for simple tasks (<2min) is included in the v3.5.0-atlas Algorithm component
- Replaces the old Instant/Fast effort level tiers with a cleaner mode-selection model
- Mode routing (ALGORITHM vs NATIVE vs MINIMAL) handled within the Algorithm text, not in CLAUDE.md template

### PRDSync.hook.ts: ADOPTED

- Read-only PostToolUse hook that syncs PRD frontmatter to `work.json` for dashboard visibility
- Triggers on Write/Edit of `PRD.md` files in `MEMORY/WORK/`
- Registered in settings.json for both Write and Edit matchers

### New lib files added from v4.0.3

- `hooks/lib/prd-utils.ts` — PRD CRUD utilities (frontmatter parsing, work.json sync)
- `hooks/lib/learning-readback.ts` — Algorithm reflection readback for session context

*Note: `hook-io.ts` and `prd-template.ts` were added but removed in the 2026-03-03 codebase audit — they had zero consumers.*

## Codebase Audit (2026-03-03)

A comprehensive multi-agent audit removed 55 orphaned files (~12,900 lines deleted). Two red team validations confirmed zero false positives. All removed files are backed up in `.audit-backup/` at repo root (gitignored) for rollback.

**Key removals:** `skills/CORE/` (dead directory), 18 orphaned PAI tools/banner prototypes, `lib/migration/` (orphan tree), 3 unwired hook libs, misc dead files.

**Key fixes:** Case-sensitive SKILL.md paths (Aphorisms, BrightData, Sales, USMetrics), `LoadAgentContext.ts` dir casing, settings.json versions, hooks/README.md accuracy.

**Kept (live dependencies):** `IntegrityMaintenance.ts` (subprocess from SystemIntegrity handler), `LoadSkillConfig.ts` (PAIUpgrade workflow).

**Full details:** `docs/plans/2026-03-03-codebase-audit-cleanup.md`, audit reports at `~/.claude/MEMORY/WORK/20260303-042800_codebase-orphan-audit/`
