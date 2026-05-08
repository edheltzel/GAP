---
title: Three-Agent Evaluation & Simplification
date: 2026-04-22
last_updated: 2026-04-22
status: draft
owner: Ed
type: evaluation/architecture
phases: 3
agents_reviewed: [pi, opencode, claudecode]
deliverables:
  - per-agent audit reports (3)
  - unified lessons-learned synthesis
  - extension manifest design + install utility spec
  - PAI reliance assessment + simplification roadmap
---

# Three-Agent Evaluation & Simplification

## Context

`atlas-config` hosts three coding-agent configurations via multi-package GNU Stow:
`claudecode/` → `~/.claude/`, `opencode/` → `~/.config/opencode/`, `pi-agent/` → `~/.pi/agent/`.

Over time, the PAI ecosystem inside `claudecode/` has become dense, and there is
suspected duplication between OpenCode and Claude Code (since OpenCode can read
`~/.claude/` skills and settings). Vendored agent extensions/skills sometimes
arrive with their own `.git/` directories, which git treats as submodule candidates.

This plan defines a **three-phase / three-milestone** evaluation that audits each
agent in depth, removes duplication, captures lessons learned, and produces a
**manifest-based install pattern** so extensions never need to be checked in with
their own git history.

## Goals

1. Produce one in-depth audit report per agent (Pi, OpenCode, Claude Code).
2. Eliminate the `.git/`-inside-stow problem via a portable install manifest + utility.
3. Quantify PAI reliance — how deep do our workflows actually depend on it?
4. Surface cross-agent duplication (especially OpenCode ↔ Claude Code).
5. Deliver a prioritized simplification roadmap with lessons learned.

## Non-Goals

- Implementing code changes in this plan's execution. The plan produces **recommendations + specs**.
  Implementation is scoped into follow-up plans after approval.
- Evaluating Pi Mono / PI Extension features that aren't currently wired up locally.
- Re-benchmarking agent *model* performance. This is a configuration/structure review.

## Success Criteria (Ideal State)

- [ ] Each of the 3 agents has a written audit under `.atlas/plans/archive/evaluations/`.
- [ ] Extension install manifest (text file format) + install utility (CLI) spec committed.
- [ ] Zero vendored `.git/` directories remain in the repo after implementation phase.
- [ ] PAI reliance scored on a 0–10 scale with concrete dependency evidence.
- [ ] OpenCode-Claude Code duplication inventory complete, with delete/symlink/keep decisions per item.
- [ ] Lessons-learned doc captures 10+ actionable insights from across the three reviews.
- [ ] Final consolidator produces one prioritized simplification backlog (P0/P1/P2).

---

## Agent Gauntlet (Full Fanout)

| # | Role                          | Subagent type        | Scope                                                                 |
|---|-------------------------------|----------------------|-----------------------------------------------------------------------|
| 1 | Architect                     | `Architect`          | Project structure, stow layout, extension-manifest design             |
| 2 | Engineer                      | `Engineer`           | Install-utility implementation spec (not build)                       |
| 3 | Pi Reviewer                   | `Explore` + `general`| Deep dive on `pi-agent/` — extensions, skills, utilities gaps         |
| 4 | OpenCode Reviewer             | `Explore` + `general`| Deep dive on `opencode/` — agents, plugins, commands, skills          |
| 5 | Claude Code Reviewer          | `Explore` + `general`| Deep dive on `claudecode/` — PAI ecosystem, hooks, skills, commands   |
| 6 | OpenCode Docs Researcher      | `PerplexityResearcher` | Fetch latest OpenCode docs (capability matrix, config schema, plugin API) |
| 7 | Claude Code Docs Researcher   | Context7 MCP via main | Pull fresh Claude Code docs (hooks, skills, MCP, settings schema)     |
| 8 | Red-Team "Overkill" Auditor   | via `RedTeam` skill  | Challenge every PAI subsystem — is it earning its weight?             |
| 9 | Lessons-Learned Synthesizer   | `general-purpose`    | Aggregate findings from #1–#8 into thematic insights                  |
| 10| Final Consolidator            | `Architect`          | Merge everything into the prioritized simplification backlog          |

Agents 3–7 run **in parallel** in Phase 1 kickoff. Agents 8–10 run **sequentially** after
the per-agent audits are in hand.

---

## Phase 1 — Pi Coding Agent (Milestone 1)

**Goal:** Understand Pi's current capabilities, gaps, and how we've been feeding extensions into it.
This phase also designs the extension-manifest pattern that Phases 2 & 3 will consume.

### Epic 1.1 — Pi Inventory & Gap Analysis
- [ ] **Task 1.1.1:** Run Pi Reviewer agent over `pi-agent/.pi/agent/` — catalog every dir, SKILL.md, command, extension, hook, plugin.
- [ ] **Task 1.1.2:** Map `extensions/` subdirs (ant-colony, brief, code-ast, etc.) to their upstream sources — note which are vendored, which have `.git/` inside, which are live-updated.
- [ ] **Task 1.1.3:** Map `pi-agent/.pi/agent/git/github.com/*` — is this a parallel vendor tree? Duplicate of `extensions/`?
- [ ] **Task 1.1.4:** List utilities Pi **has** vs **needs** (Ed's working preferences: standup, handoff, recall, memory, voice). Flag gaps.
- [ ] **Task 1.1.5:** Produce `pi-audit.md` at `.atlas/plans/archive/evaluations/2026-04-22-pi-audit.md`.

### Epic 1.2 — Extension Manifest & Install Utility
- [ ] **Task 1.2.1:** Architect designs manifest schema. Decision point: TOML vs simple line-per-entry (`owner/repo@ref  dest/path  install_cmd`). Prefer the simpler one.
- [ ] **Task 1.2.2:** Define ignore rules for `.gitignore` to exclude all vendored extension trees (catch-all globs, not per-extension).
- [ ] **Task 1.2.3:** Engineer specs the install utility as a top-level `just` task (e.g., `just install-extensions [--agent pi|opencode|claudecode|all]`). Reads per-agent `<agent>/extensions.manifest`, clones shallow, strips `.git/`, runs install_cmd, symlinks or copies to dest. Bash-only fallback script for pre-`just` bootstrap.
- [ ] **Task 1.2.4:** Design the uninstall/update paths — can an upstream update be pulled without re-cloning the world?
- [ ] **Task 1.2.5:** Write migration plan: convert existing `pi-agent/extensions/*` and `opencode/plugins-source/*` into manifest entries, back up, test the install utility, remove from git.

### Epic 1.3 — Pi Lessons Learned
- [ ] **Task 1.3.1:** Captures "what we'd do differently if rebuilding Pi config from scratch."
- [ ] **Task 1.3.2:** Identify any Pi skill/command that duplicates a Claude Code or OpenCode equivalent (defer consolidation decision to Phase 3 synthesis).

**Phase 1 Exit Criteria:** Pi audit written, manifest schema approved by Ed, install-utility spec ready to build (implementation is post-plan).

---

## Phase 2 — OpenCode (Milestone 2)

**Goal:** Determine exactly what OpenCode needs to own vs what it can inherit from `~/.claude/`.
Kill duplication. Research current OpenCode docs (we're likely on an older mental model).

### Epic 2.1 — OpenCode Docs Refresh
- [ ] **Task 2.1.1:** PerplexityResearcher pulls current OpenCode docs — config schema, plugin API, agent format, command format, which Claude Code artifacts it can read natively.
- [ ] **Task 2.1.2:** Document OpenCode's auto-discovery behavior: does it read `~/.claude/skills/`, `~/.claude/commands/`, `~/.claude/agents/`? What about hooks? MCP config?
- [ ] **Task 2.1.3:** Cross-reference `converting-claude-subagents` and `converting-slash-commands` skills — do we still need manual conversions, or does OpenCode now handle Claude Code formats directly?

### Epic 2.2 — Duplication Audit
- [ ] **Task 2.2.1:** OpenCode Reviewer enumerates every file in `opencode/.config/opencode/`. Tag each as: (a) OpenCode-only required, (b) duplicates Claude Code, (c) could symlink/delete.
- [ ] **Task 2.2.2:** Specific targets: `agents/` vs `agent/`, `skills/` vs `skill/`, `commands/` vs `command/`, `plugins/` vs `plugins-source/`, `tool/`, `themes/`. Why do we have both singular and plural dirs?
- [ ] **Task 2.2.3:** `superpowers/` — still used? (CLAUDE.md note says Superpowers plugin removed 2026-04-05 from Claude Code. Status in OpenCode?)
- [ ] **Task 2.2.4:** Propose a "thin OpenCode" — only files that OpenCode uniquely needs; everything else deleted or pointed at `~/.claude/`.

### Epic 2.3 — OpenCode Lessons Learned
- [ ] **Task 2.3.1:** Capture workflow friction points from using OpenCode alongside Claude Code.
- [ ] **Task 2.3.2:** Identify any OpenCode feature Claude Code should adopt (or vice versa).

**Phase 2 Exit Criteria:** OpenCode audit written, duplication inventory with decisions, manifest entries drafted for any remaining OpenCode vendored extensions.

---

## Phase 3 — Claude Code + PAI Reliance (Milestone 3)

**Goal:** Measure how heavily we depend on PAI, identify simplification targets, and produce the
final consolidated roadmap. This is the longest phase — PAI is the densest surface area.

### Epic 3.1 — Claude Code Docs Refresh
- [ ] **Task 3.1.1:** Main agent queries Context7 MCP for current Claude Code docs (hooks lifecycle, skill format, MCP config, settings schema, permission system, plan mode).
- [ ] **Task 3.1.2:** Diff what's documented now vs what we implemented in `claudecode/.claude/`. Flag anything using deprecated patterns or missing new features we'd want.

### Epic 3.2 — PAI Reliance Assessment
- [ ] **Task 3.2.1:** Claude Code Reviewer maps every PAI subsystem:
  - `skills/PAI/` (Algorithm, Components, Tools)
  - Hooks in `claudecode/.claude/hooks/` (23 stowed + 2 non-stowed + 1 external)
  - `MEMORY/` subsystem (SessionExtract, WorkCompletion, etc.)
  - VoiceServer + VoiceLayer integration
  - `Observability/` dashboard
  - Agent system (`agents/`, `Agents` skill)
  - Commands (`/Standup`, `/CreateHandoff`, etc.)
- [ ] **Task 3.2.2:** For each subsystem, answer: (a) how often triggered, (b) what breaks if removed, (c) cognitive load (0–10), (d) value delivered (0–10).
- [ ] **Task 3.2.3:** Score overall PAI reliance. Ed's prior: "pretty heavy" — verify or refute with evidence.

### Epic 3.3 — Red-Team Overkill Audit
- [ ] **Task 3.3.1:** Invoke `/RedTeam` or similar adversarial skill. For each PAI subsystem, construct the strongest argument for removal.
- [ ] **Task 3.3.2:** Identify subsystems where the argument-for-removal is stronger than the argument-for-keeping. Flag as simplification candidates.
- [ ] **Task 3.3.3:** Sanity-check: which recently-added features (last 90 days per git log) have actually been used? Unused-but-installed is overkill.

### Epic 3.4 — Simplification Recommendations
- [ ] **Task 3.4.1:** For each flagged candidate from Epic 3.3, propose one of: `DELETE`, `CONSOLIDATE`, `KEEP-AS-IS`, `REFACTOR`. Include reasoning.
- [ ] **Task 3.4.2:** Draft Claude Code audit at `.atlas/plans/archive/evaluations/2026-04-22-claudecode-audit.md`.

### Epic 3.5 — Final Synthesis
- [ ] **Task 3.5.1:** Lessons-Learned Synthesizer reads all 3 audits + red-team report. Produces 10+ themed insights covering: duplication patterns, cognitive-load hotspots, install/config pain, cross-agent mismatches, "should have been built differently" moments.
- [ ] **Task 3.5.2:** Final Consolidator produces a prioritized simplification backlog:
  - **P0:** Remove vendored `.git/`, implement manifest install utility.
  - **P1:** Delete/symlink OpenCode-Claude Code duplication.
  - **P2:** PAI subsystem cuts (from Epic 3.4 decisions).
  - **P3:** Per-agent quality-of-life fixes (from Epics 1.3 / 2.3).
- [ ] **Task 3.5.3:** Write consolidated roadmap at `.atlas/plans/2026-04-22-simplification-roadmap.md`. Each P0/P1 item becomes its own follow-up plan file.

**Phase 3 Exit Criteria:** Reliance scored, red-team audit complete, simplification roadmap merged, lessons-learned doc published.

---

## Deliverables (Final)

| # | Artifact                                                                 | Location                                                |
|---|--------------------------------------------------------------------------|---------------------------------------------------------|
| 1 | Pi audit                                                                 | `.atlas/plans/archive/evaluations/2026-04-22-pi-audit.md`       |
| 2 | OpenCode audit                                                           | `.atlas/plans/archive/evaluations/2026-04-22-opencode-audit.md` |
| 3 | Claude Code / PAI audit                                                  | `.atlas/plans/archive/evaluations/2026-04-22-claudecode-audit.md` |
| 4 | Extension manifest schema + install utility spec                         | `.atlas/plans/2026-04-22-extension-manifest-spec.md`            |
| 5 | PAI reliance assessment (scorecard)                                      | embedded in Claude Code audit                                   |
| 6 | Red-team overkill report                                                 | `.atlas/plans/archive/evaluations/2026-04-22-redteam-overkill.md` |
| 7 | Lessons-learned synthesis                                                | `.atlas/plans/2026-04-22-lessons-learned.md`                    |
| 8 | **Simplification roadmap** (the consolidated output)                     | `.atlas/plans/2026-04-22-simplification-roadmap.md`             |

## Risks & Mitigations

| Risk                                                              | Mitigation                                                       |
|-------------------------------------------------------------------|-------------------------------------------------------------------|
| Red-team over-prunes subsystems Ed actually uses                  | Red-team proposes, Ed approves. No deletions until P0/P1/P2 plans are individually greenlit. |
| Manifest install utility fails on new-machine bootstrap           | Engineer includes a dry-run mode + bash-only fallback            |
| OpenCode docs moved / changed schema since we last integrated     | Researcher dated snapshot + delta vs current config               |
| Ed disagrees with some agent findings                             | All agent reports are drafts; final consolidator waits for Ed's pass |
| Context bleed between parallel agents producing contradictions    | Final consolidator explicitly reconciles disagreements in writing |

## Execution Order

1. **Ed approves this plan.**
2. Phase 1 kicks off: spawn Architect + Engineer + Pi Reviewer + OpenCode Docs Researcher + (later) Claude Code Docs fetch in parallel.
3. Phase 1 exit review with Ed (~1 working session).
4. Phase 2 kicks off: OpenCode Reviewer + completion of docs research.
5. Phase 2 exit review with Ed.
6. Phase 3 kicks off: Claude Code Reviewer + Red-Team + Synthesizer + Consolidator.
7. Final review: simplification roadmap presented; follow-up P0/P1/P2 plans created on approval.

## Decisions (locked 2026-04-22)

- **Install utility:** top-level — implemented as a `just` task in atlas-config's `justfile` (e.g., `just install-extensions`). Not a Pi-internal command. This keeps bootstrap agent-agnostic and usable before any single agent is fully configured.
- **Manifest location:** per-agent. Pi first — `pi-agent/extensions.manifest`. OpenCode and Claude Code get their own manifests in Phases 2 & 3 if/when they vendor extensions (`opencode/extensions.manifest`, `claudecode/extensions.manifest`).
- **Phase-exit reviews:** via VoiceLayer (converse mode). Atlas summarizes findings, Ed responds by voice, decisions captured back into the relevant audit/roadmap doc.
