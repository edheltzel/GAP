# PRD: PAI Multi-Agent Compatibility Restructure

> Migrate PAI/Atlas from Claude Code-only to work across Claude Code, OpenCode, and pi.dev — with pi.dev as the primary migration target.

---

## Status

| What | State |
|------|-------|
| Overall | `PLANNED` — awaiting approval |
| Features | 5 child PRDs |
| Primary focus | PAI → pi.dev migration |
| Constraint | All work in git worktree — live Atlas untouched until merge |

---

## Context

PAI is currently deeply coupled to Claude Code. Ed wants PAI to work across **3 coding agents** — Claude Code, OpenCode, and pi.dev — with **pi.dev as the primary migration target**.

**Critical constraint:** All implementation MUST use git worktrees. Modifying Atlas directly would break Claude Code.

### Research Sources

| Source | What We Learned |
|--------|----------------|
| [badlogic/pi-mono](https://github.com/badlogic/pi-mono) | Pi.dev source — tools, skills system, extensions, config format |
| [badlogic/pi-skills](https://github.com/badlogic/pi-skills) | **Official cross-agent skills repo** — shows exact pattern for sharing skills across pi.dev, CC, Codex, Amp |
| [can1357/oh-my-pi](https://github.com/can1357/oh-my-pi) | Unified config discovery across 8 agent directories |
| [disler/pi-vs-claude-code](https://github.com/disler/pi-vs-claude-code) | Feature comparison, extension examples |
| [code-yeongyu/oh-my-opencode](https://github.com/code-yeongyu/oh-my-opencode) | Bridges CC ecosystem into OpenCode (35k stars) |
| [agents.md spec](https://agents.md/) | Linux Foundation AAIF standard, adopted by 60k+ projects |
| [pi-mono skills docs](https://github.com/badlogic/pi-mono/blob/main/packages/coding-agent/docs/skills.md) | Skill discovery paths, SKILL.md format, settings.json skills array |
| [pi-mono extensions docs](https://github.com/badlogic/pi-mono/blob/main/packages/coding-agent/docs/extensions.md) | Extension API, tool registration, lifecycle hooks |

### Validated Compatibility Matrix

| Feature | Claude Code | OpenCode + OmO | pi.dev |
|---------|------------|-----------------|--------|
| **Instruction file** | CLAUDE.md | AGENTS.md (+ CLAUDE.md fallback) | AGENTS.md (+ CLAUDE.md fallback) — one per dir, AGENTS.md wins |
| **Global config** | `~/.claude/` | `~/.config/opencode/` | `~/.pi/agent/` (overridable via `PI_CODING_AGENT_DIR`) |
| **Skills path** | `~/.claude/skills/` | OmO: `~/.claude/skills/` + native paths | `~/.pi/agent/skills/` + custom paths via `settings.json` |
| **Custom skill paths** | No | No | **Yes** — `"skills": ["~/.claude/skills"]` in settings.json |
| **MCP support** | Yes | Yes | **No** — Extensions replace MCP |
| **Task/Todo tools** | TaskCreate, TaskList, TaskUpdate, TaskGet | todowrite, todoread | **None** — no task management tools |
| **Tools** | Read, Write, Edit, Bash, Grep, Glob | read, write, edit, bash, grep, glob (via OmO) | read, write, edit, bash, grep, find, ls |
| **Hook/Event system** | Shell scripts (5+ events) | OmO plugin hooks (46 hooks) | Extensions API (TypeScript, 20+ events) |
| **SKILL.md format** | Native | Via OmO | **Native** — same YAML frontmatter format |

### Critical Findings (Claim Validation)

| Claim | Verdict | Evidence |
|-------|---------|----------|
| "Pi.dev reads AGENTS.md and CLAUDE.md" | **CONFIRMED** | `resource-loader.ts` — candidates: `["AGENTS.md", "CLAUDE.md"]`. Only ONE per directory. |
| "Config at ~/.pi/agent/" | **CONFIRMED** | `config.ts` — `getAgentDir()` returns `~/.pi/agent/`. JSON format. |
| "Pi.dev reads ~/.agents/skills/" | **REFUTED** | No such path in codebase. Actual: `~/.pi/agent/skills/`, `.pi/skills/`, custom paths in settings.json |
| "Pi.dev has no MCP" | **CONFIRMED** | Zero MCP references. Extensions (TypeScript modules) are the replacement. |
| "Tools: read, bash, edit, write, grep, find, ls" | **CONFIRMED** | Exactly 7 tools in `allTools`. No task management tools. |
| "PAI works in OpenCode with zero changes" | **FALSE** | 40+ tool name mismatches (TaskCreate vs todowrite) |
| "Generated AGENTS.md has no size limits" | **CONFIRMED** | No size guard in code. Content injected verbatim into system prompt. |
| "OmO is stable" | **ACCEPTABLE RISK** | 35k stars, CC compat is a core selling point |

### The Fundamental Gap: No Task Tools in Pi.dev

**This is the most important finding.** Pi.dev has **no TaskCreate, TaskList, TaskUpdate, or any equivalent**. The PAI algorithm's entire ISC (Ideal State Criteria) system depends on task tools for:
- Creating criteria in OBSERVE phase
- Tracking/updating criteria in BUILD/EXECUTE
- Verifying criteria in VERIFY phase
- Quality Gate checks

This is NOT just a tool name mapping — it's a **missing capability**. Options:
1. **Extension-based task system** — Write a pi.dev extension that provides `taskCreate`, `taskList`, `taskUpdate` tools via `pi.registerTool()`
2. **File-based ISC tracking** — PAI algorithm uses file I/O (read/write to a PRD markdown file) instead of task tools
3. **Algorithm adaptation** — The generated AGENTS.md instructs the LLM to track ISC in the PRD file directly (the persistent memory track) without working memory task tools

**Recommended: Option 2+3 combined.** The PRD file IS the persistent ISC store. In Claude Code, working memory (TaskCreate) and persistent memory (PRD file) are dual-tracked. In pi.dev, the PRD file becomes the ONLY track. The generated AGENTS.md instructs: "Create ISC as markdown checkboxes in the PRD file. Update them in-place. Use `grep` to check status."

### Cross-Agent Skills Pattern (from badlogic/pi-skills)

The **official pi-skills repo** (by pi.dev's author) shows the canonical cross-agent approach:

| Agent | Install Location |
|-------|-----------------|
| pi.dev | `~/.pi/agent/skills/{skill-name}/` |
| Claude Code | Symlink each skill subdir to `~/.claude/skills/{skill-name}/` (CC only looks one level deep) |
| Codex CLI | `~/.codex/skills/{skill-name}/` |
| Amp | `~/.config/amp/tools/{skill-name}/` |

**Key constraint:** Claude Code only looks one level deep in `~/.claude/skills/`. You cannot symlink a parent directory — must symlink each skill individually.

**Pi.dev alternative:** Add `"skills": ["~/.claude/skills"]` to pi.dev's `settings.json` — it discovers skills from custom paths natively. No symlinks needed.

---

## Architecture

### SKILL.md as Source of Truth + Agent Adapters

```
Source of Truth (unchanged — Claude Code native):
  claudecode/.claude/skills/PAI/SKILL.md    ← THE canonical PAI (CC-native tools)
  claudecode/.claude/skills/PAI/Components/ ← Identity, steering rules, etc.
  claudecode/.claude/skills/*/              ← All PAI skills
  claudecode/.claude/agents/                ← Agent definitions
  claudecode/.claude/hooks/                 ← Hook scripts

Pi.dev (primary migration target):
  pidotdev/.pi/agent/settings.json          ← Points skills to ~/.claude/skills + custom config
  pidotdev/.pi/agent/AGENTS.md              ← GENERATED: PAI algorithm adapted for pi.dev
  pidotdev/.pi/agent/tools/                 ← Extensions (task system, voice, etc.)

OpenCode (OmO-bridged):
  opencode/.config/opencode/AGENTS.md       ← STATIC tool-mapping overlay
  opencode/.config/opencode/opencode.json   ← Existing config + OmO pinned
```

### What Each Agent Gets

**Claude Code** — No changes. Primary platform. Best experience.

**Pi.dev** (primary migration) — Three integration layers:
1. **Skill discovery** — `settings.json` with `"skills": ["~/.claude/skills"]` so pi.dev discovers all PAI skills natively. No symlinks needed.
2. **AGENTS.md** — Generated from SKILL.md, adapted for pi.dev's tool set. Replaces TaskCreate/TaskList references with file-based ISC tracking via PRD files.
3. **Extensions** (future) — TypeScript extensions for task management, voice notifications, and other CC-specific capabilities.

**OpenCode + OmO** — OmO loads PAI skill files from `~/.claude/skills/`. Static AGENTS.md tool-mapping overlay bridges 40+ tool name differences.

### Key Design Decisions

| ID | Decision | Rationale |
|----|----------|-----------|
| D1 | Pi.dev is the primary migration target | Ed's explicit priority. Most architectural differences to solve. |
| D2 | SKILL.md stays CC-native | No compromises to primary platform. Adapters bridge gaps. |
| D3 | Pi.dev discovers CC skills via settings.json | `"skills": ["~/.claude/skills"]` — no symlinks, no duplication |
| D4 | ISC tracking via PRD files in pi.dev | No task tools → PRD file becomes sole ISC store. Checkbox-based tracking. |
| D5 | Extensions for future CC-equivalent features | Task system, voice, plan mode — all buildable via pi.dev extensions |
| D6 | OmO required for OpenCode | Bridges CC skills/agents/hooks/MCPs. Pin version. |
| D7 | Stow target isolation for testing | `STOW_TARGET=/tmp/atlas-test-home`, never live `~/` from worktree |
| D8 | AGENTS.md over CLAUDE.md in shared contexts | Linux Foundation standard. Pi.dev prioritizes it. |

---

## Feature PRDs

All work happens in a git worktree:
```bash
cd ~/Developer/atlas-config
git worktree add .claude/worktrees/multi-agent-compat -b feat/multi-agent-compat
```

---

### PRD-01: Migrate to `just`

**File:** `.atlas-plans/PRD-01-justfile-migration.md`
**Status:** `COMPLETE`
**Completed:** 2026-03-02

Replace `Makefile` with `justfile`. `just` is better suited — recipes are actions not file targets, no `.PHONY`, no tab footguns, built-in `--list`.

#### Epic 1: Port Existing Recipes

| # | Task | Status |
|---|------|--------|
| 1.1 | Create `justfile` with variables: `stow_dir`, `stow_target`, `stow_cmd`, `pkgs` | `DONE` |
| 1.2 | Port `help` recipe (use `just --list` with comments) | `DONE` |
| 1.3 | Port `update` recipe (restow all packages) | `DONE` |
| 1.4 | Port `link` recipe (first-time setup) | `DONE` |
| 1.5 | Port `unlink` recipe (remove symlinks) | `DONE` |
| 1.6 | Port `status` recipe (dry-run stow) | `DONE` |
| 1.7 | Port `push` recipe (commit and push) | `DONE` |
| 1.8 | Preserve Eldritch color scheme (YELLOW, GREEN, WHITE, CYAN, CLR) | `DONE` |

#### Epic 2: Validate and Clean Up

| # | Task | Status |
|---|------|--------|
| 2.1 | Run `just --list` and verify all recipes appear with descriptions | `DONE` |
| 2.2 | Run `just update` and verify identical behavior to `make update` | `DONE` |
| 2.3 | Run `just status` and verify identical behavior to `make status` | `DONE` |
| 2.4 | Delete `Makefile` | `DONE` |

---

### PRD-02: Pi.dev PAI Migration (PRIMARY)

**File:** `.atlas-plans/PRD-02-pidotdev-migration.md`
**Status:** `PENDING`
**Depends on:** PRD-01 (justfile exists)

Create `pidotdev/` stow package. Generate AGENTS.md adapted for pi.dev's tool set. Configure skill discovery to find PAI skills.

#### Epic 1: Stow Package Structure

| # | Task | Status |
|---|------|--------|
| 1.1 | Create `pidotdev/.pi/agent/` directory tree | `PENDING` |
| 1.2 | Create `pidotdev/.stow-local-ignore` | `PENDING` |
| 1.3 | Add `pidotdev` to `pkgs` in justfile | `PENDING` |

#### Epic 2: Pi.dev Settings

| # | Task | Status |
|---|------|--------|
| 2.1 | Create `pidotdev/.pi/agent/settings.json` — model defaults, theme, skill paths | `PENDING` |
| 2.2 | Add `"skills": ["~/.claude/skills"]` to point pi.dev at PAI skills | `PENDING` |
| 2.3 | Configure `defaultProvider`, `defaultModel`, `defaultThinkingLevel` | `PENDING` |
| 2.4 | Verify pi.dev discovers PAI skills via custom skill path | `PENDING` |

#### Epic 3: AGENTS.md Generation (Build Script)

| # | Task | Status |
|---|------|--------|
| 3.1 | Audit SKILL.md for all Claude Code-specific references | `PENDING` |
| 3.2 | Build tool-reference mapping table: CC tools → pi.dev equivalents | `PENDING` |
| 3.3 | Design ISC tracking adaptation: TaskCreate/List/Update → PRD file checkbox tracking | `PENDING` |
| 3.4 | Write `scripts/generate-agents-md.ts` — reads SKILL.md + Components | `PENDING` |
| 3.5 | Replace CC tool references with pi.dev equivalents (read, write, edit, bash, grep, find, ls) | `PENDING` |
| 3.6 | Replace TaskCreate/TaskList/TaskUpdate instructions with file-based ISC tracking | `PENDING` |
| 3.7 | Append pi.dev tool appendix documenting all 7 tools | `PENDING` |
| 3.8 | Output to `pidotdev/.pi/agent/AGENTS.md` | `PENDING` |
| 3.9 | Add `build` recipe to justfile | `PENDING` |
| 3.10 | Add `pidotdev/.pi/agent/AGENTS.md` to `.gitignore` as generated | `PENDING` |

#### Epic 4: Verify Generated Output

| # | Task | Status |
|---|------|--------|
| 4.1 | Run `just build` — no errors | `PENDING` |
| 4.2 | Verify generated AGENTS.md preserves all 7 algorithm phases (OBSERVE → LEARN) | `PENDING` |
| 4.3 | Verify no CC-specific tool names remain (TaskCreate, TaskList, TaskUpdate, TaskGet, Glob) | `PENDING` |
| 4.4 | Verify ISC tracking instructions use file-based approach (PRD checkboxes) | `PENDING` |
| 4.5 | Verify pi.dev tool appendix matches actual pi.dev tool set | `PENDING` |
| 4.6 | Verify skill discovery works: launch pi.dev, check PAI skills visible | `PENDING` |

---

### PRD-03: OpenCode Compatibility

**File:** `.atlas-plans/PRD-03-opencode-compatibility.md`
**Status:** `PENDING`
**Depends on:** PRD-01

Add tool-mapping overlay so PAI algorithm works in OpenCode+OmO.

#### Epic 1: Tool-Mapping Overlay

| # | Task | Status |
|---|------|--------|
| 1.1 | Audit SKILL.md for all CC tool references | `PENDING` |
| 1.2 | Map CC tools to OpenCode equivalents (TaskCreate→todowrite, TaskList→todoread, etc.) | `PENDING` |
| 1.3 | Create `opencode/.config/opencode/AGENTS.md` with mapping overlay | `PENDING` |
| 1.4 | Document tools with no OpenCode equivalent (EnterPlanMode, ExitPlanMode → skip) | `PENDING` |

#### Epic 2: OmO Version Pinning

| # | Task | Status |
|---|------|--------|
| 2.1 | Check current OmO version in opencode.json plugins | `PENDING` |
| 2.2 | Pin to specific tested version (not `@latest`) | `PENDING` |
| 2.3 | Document fallback: AGENTS.md works without OmO | `PENDING` |

---

### PRD-04: Skills Cross-Agent Sharing

**File:** `.atlas-plans/PRD-04-skills-sharing.md`
**Status:** `PENDING`
**Depends on:** PRD-02

Set up cross-agent skill discovery following the pattern from [badlogic/pi-skills](https://github.com/badlogic/pi-skills).

#### Epic 1: Identify Shareable Skills

| # | Task | Status |
|---|------|--------|
| 1.1 | Audit `~/.claude/skills/` — categorize as CC-only vs portable | `PENDING` |
| 1.2 | CC-only: skills that depend on CC hooks, CC-specific tools, or voice mode | `PENDING` |
| 1.3 | Portable: methodology/content skills, analysis skills, writing skills | `PENDING` |

#### Epic 2: Pi.dev Skill Access

| # | Task | Status |
|---|------|--------|
| 2.1 | Verify pi.dev's `settings.json` `"skills"` array discovers CC skills | `PENDING` |
| 2.2 | Test that PAI SKILL.md is discoverable and loadable by pi.dev | `PENDING` |
| 2.3 | Test that skill YAML frontmatter (name, description) parses correctly | `PENDING` |
| 2.4 | Add `agents-skills` recipe to justfile for documenting/managing cross-agent access | `PENDING` |

---

### PRD-05: Integration Testing + Merge

**File:** `.atlas-plans/PRD-05-integration-testing.md`
**Status:** `PENDING`
**Depends on:** PRD-01, PRD-02, PRD-03, PRD-04

Validate everything in worktree isolation, then merge to main and smoke test.

#### Epic 1: Worktree Validation (STOW TARGET ISOLATION)

| # | Task | Status |
|---|------|--------|
| 1.1 | `just build` — generates pi.dev AGENTS.md without error | `PENDING` |
| 1.2 | `just test-stow` — dry-run to `/tmp/atlas-test-home`, no conflicts | `PENDING` |
| 1.3 | Inspect `/tmp/atlas-test-home/` symlink targets for all 3 packages | `PENDING` |
| 1.4 | Verify pi.dev AGENTS.md is semantically correct | `PENDING` |
| 1.5 | Verify OpenCode AGENTS.md tool-mapping is accurate | `PENDING` |

#### Epic 2: Merge to Main

| # | Task | Status |
|---|------|--------|
| 2.1 | Review all changes in worktree branch | `PENDING` |
| 2.2 | Merge worktree branch to master | `PENDING` |
| 2.3 | `just update` to restow all packages from main tree | `PENDING` |

#### Epic 3: Smoke Tests

| # | Task | Status |
|---|------|--------|
| 3.1 | **Claude Code** — launch session, verify PAI loads unchanged, ISC works, voice curls fire | `PENDING` |
| 3.2 | **Pi.dev** — launch session, verify AGENTS.md loaded, skills discovered, algorithm phases execute | `PENDING` |
| 3.3 | **OpenCode+OmO** — launch session, verify skill loaded + tool mapping visible | `PENDING` |
| 3.4 | Clean up worktree | `PENDING` |

#### Rollback Plan

- Worktree isolation = main tree untouched until merge
- If merge breaks: `git revert HEAD` + `just update`
- `just unlink && just link` restores from repo state

---

## Files Summary

### New Files

| File | PRD | Purpose |
|------|-----|---------|
| `justfile` | PRD-01 | Replaces Makefile |
| `pidotdev/.pi/agent/settings.json` | PRD-02 | Pi.dev config: skill paths, model defaults |
| `pidotdev/.pi/agent/AGENTS.md` | PRD-02 | GENERATED: PAI algorithm adapted for pi.dev (7 tools, file-based ISC) |
| `pidotdev/.stow-local-ignore` | PRD-02 | Stow ignore rules |
| `scripts/generate-agents-md.ts` | PRD-02 | Build script: SKILL.md → pi.dev-adapted AGENTS.md |
| `opencode/.config/opencode/AGENTS.md` | PRD-03 | Static tool-mapping overlay for OpenCode |

### Deleted Files

| File | PRD | Why |
|------|-----|-----|
| `Makefile` | PRD-01 | Replaced by justfile |

### Modified Files

| File | PRD | Change |
|------|-----|--------|
| `opencode/.config/opencode/opencode.json` | PRD-03 | Pin OmO version |
| `.gitignore` | PRD-02 | Mark generated AGENTS.md |

### Protected (no changes)

`claudecode/.claude/skills/PAI/SKILL.md`, `claudecode/.claude/settings.json`, `claudecode/.claude/hooks/*`, `claudecode/.claude/agents/*`, `claudecode/.claude/MEMORY/*`

---

## Red Team Summary

### Validated Claims

| # | Claim | Status | Evidence |
|---|-------|--------|----------|
| 1 | Pi.dev reads AGENTS.md | CONFIRMED | `resource-loader.ts` — candidates array |
| 2 | Config at ~/.pi/agent/ | CONFIRMED | `config.ts` — `getAgentDir()` |
| 3 | Pi reads ~/.agents/skills/ | **REFUTED** | Not in codebase. Use `~/.pi/agent/skills/` or settings.json `skills` array |
| 4 | No MCP in pi.dev | CONFIRMED | Zero MCP references. Extensions are replacement. |
| 5 | Pi.dev tools: read, bash, edit, write, grep, find, ls | CONFIRMED | `tools/index.ts` — exactly 7 tools |
| 6 | No task management tools | CONFIRMED | No TaskCreate/TodoWrite equivalent |
| 7 | AGENTS.md no size limit | CONFIRMED | Injected verbatim, no guard |
| 8 | OmO bridges CC to OpenCode | CONFIRMED | 35k stars, reads `~/.claude/skills/` |
| 9 | CC tool name mismatch in OpenCode | CONFIRMED | 40+ references to TaskCreate/TaskList/TaskUpdate |
| 10 | Stow 3-package safety | CONFIRMED | Non-overlapping target dirs |

### Risk Register

| Risk | Severity | Mitigation |
|------|----------|------------|
| Pi.dev LLM ignores 2000-line AGENTS.md | Medium | Algorithm value is methodology; test with multiple models; file-based ISC doesn't require tool compliance |
| OmO breaks or deprecates CC compat | Low | Pin version. AGENTS.md fallback works without OmO. |
| Pi.dev extensions API changes | Low | Extensions are first-class, versioned TypeScript API |
| Generated AGENTS.md drift from SKILL.md | Medium | `just build` in CI/pre-commit; diff check in PRD-05 |

---

## Future Enhancements (not in scope)

| Enhancement | Source | Description |
|-------------|--------|-------------|
| Pi.dev task extension | D5 | TypeScript extension providing TaskCreate/TaskList/TaskUpdate tools |
| Pi.dev voice extension | D5 | Port voice mode notifications to pi.dev extension |
| File-based inter-agent comms | oh-my-claudecode | `~/.claude/mailboxes/` with JSONL for cross-agent work |
| Agent routing config | myclaude | Role-to-agent-to-model mapping |
| oh-my-pi integration | can1357/oh-my-pi | Unified config discovery across 8 agent directories |
