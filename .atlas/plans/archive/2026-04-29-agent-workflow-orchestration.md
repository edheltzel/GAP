---
title: Agent Workflow Orchestration — Plans, Phases, Tasks, "What's Next"
date: 2026-04-29
last_updated: 2026-04-29
status: superseded-round-1
percent_complete: 25
phase: Round 1 research rejected by Ed (too spec-driven/orchestration-focused) — Round 2 GitHub-native research in flight
owner: Ed
type: architecture/workflow
blockers:
  - Q1 state-layer choice (Backlog.md / Linear / Hybrid)
  - Q2 session-switcher choice (Claude Squad / CCManager)
  - Q3 atlas-next-resolver placement (Justfile / Bun CLI / Standup enhancement)
next_action: Ed answers Q1-Q3 → flip to Phase 1 implementation
deliverables:
  - Layered architecture spec (this document)
  - Visual ranking + recommendation (assets/2026-04-29-agent-workflow-ranking.html)
  - Migration plan with phase gates
research_agents:
  - PerplexityResearcher (landscape coverage, 30+ tools)
  - GeminiResearcher (adoption-weighted ranking, 6-axis decomposition)
  - ClaudeResearcher (architecture patterns, hybrid recommendation)
---

# Agent Workflow Orchestration

> ⚠️ **REDIRECTED 2026-04-29** — Ed rejected this round-1 recommendation. Reasons: top-8 was too focused on spec-driven dev (BMAD, Spec Kit, Kiro) and worktree-orchestration tools (Beads, Conductor), which are agent-creation methodologies, not project tracking. Ed wants to **keep GitHub Issues + Milestones + Projects as the spine** and find: (1) a "medium" layer that converts plans → issues, (2) a vibe-kanban-style cockpit ON TOP of GH state, (3) standup automation that knows yesterday/today/in-progress. Round-2 research with 5 parallel agents in flight. New plan will replace this one. Keep file for decision log.

> **TL;DR (round 1, superseded)** — The data shows a 5-layer architecture is the right answer, not a single tool. Recommendation: keep `.atlas/plans/` (Why), add **AGENTS.md** (conventions), add **Backlog.md + MCP** (queryable state), add **Claude Squad or CCManager** (cockpit), demote **GitHub Issues** to external-collaboration only. Build a small `atlas next` resolver to query state across all projects in one call.

---

## Problem Statement

Working across **5+ projects** (atlas-recall, atlas-config, alianza, mychron, peptideware, srcd, rainy-day-media) with **4+ agents** (Claude Code, OpenCode, Pi, Codex), the following keeps breaking:

1. **Plans accumulate, progress within them rots.** `.atlas/plans/` has the right frontmatter (`phase`, `percent_complete`, `next_action`) but nothing **queries** it. Each session begins with re-reading prose to reconstruct state.
2. **Phases inside plans aren't tracked.** Action items embedded as prose bullets aren't machine-checkable. Agents finish a phase without anything updating.
3. **Switching projects is expensive.** Pivoting from atlas-recall → mychron requires re-orienting context manually. No "what's next?" command exists.
4. **GitHub Issues + Milestones is too coarse.** Milestones are flat; no nested phase hierarchy. PR review via `gh dash` is great but says nothing about agent task queues.
5. **Vibe-kanban-class tools are insufficient** (your own observation, validated by research): they execute tasks but don't plan, don't track phase progress, and don't cross projects.

The failure mode is **write-only plans**. Capturing intent works; resolving "what now" doesn't.

---

## Research Methodology

Three parallel research agents (2026-04-29):

| Agent | Focus | Key finding |
|---|---|---|
| Perplexity | Landscape coverage (30+ tools) | Linear MCP is most polished but paid; Plane is OSS alternative; Backlog.md is best git-native |
| Gemini | Adoption-weighted rankings | BMAD-METHOD #1 by stars (~43k), but it's a *planning framework*, not a state tracker. Backlog.md is "underrated for solo CIO/CTO" |
| Claude | Architecture patterns | All viable answers converge on a 5-layer hybrid. Spec-driven (Spec Kit / Kiro / BMAD) ≠ task tracking ≠ session switching |

**Convergent recommendations across all three researchers:**
- AGENTS.md per repo (Linux Foundation standard since Dec 2025)
- Retain `.atlas/plans/` for human prose
- Add a **queryable state layer** with deterministic next-up resolution
- Add a **session switcher** for cross-project navigation
- Demote GitHub Issues to external/public-collab role

**Divergence — Layer 3 (state layer):**
- Claude → Beads (Steve Yegge, JSONL+SQLite) or claude-task-master (MCP)
- Perplexity → Linear MCP (best UX) or Plane (OSS)
- Gemini → Backlog.md (most appropriate for solo + multi-agent + git-native)

This is a values decision (vendor lock-in vs polish) — **Q1** below.

---

## Recommended Architecture — Five Layers

### Layer 1 — Conventions
**Tool:** `AGENTS.md` (per project repo) + existing `CLAUDE.md` (Claude Code-specific, coexists)

- Linux Foundation standard since Dec 2025; native support in Codex, Cursor, Aider, Zed, Warp, Roo, Jules
- Claude Code reads `CLAUDE.md` natively; AGENTS.md is the cross-agent contract
- **Keep under ~150 lines** — file size correlates inversely with agent performance (builder.io, augmentcode research)
- Contents: build/test commands, coding conventions, architectural constraints, where docs live

**Effort:** 1 day to author across all repos.

---

### Layer 2 — Plans (Why)
**Tool:** `.atlas/plans/` (existing, unchanged) + `.atlas/handoffs/` (existing)

- Long-form rationale, architecture, decisions
- Frontmatter schema **already correct**: `status`, `phase`, `percent_complete`, `blockers`, `next_action`
- The gap isn't the schema — it's the absence of a query mechanism (solved by Layer 3)
- This layer answers **"why are we doing this?"** — never try to make it machine-queryable for "what's next"

**Effort:** Zero — already in production.

---

### Layer 3 — State (What's next) ⚠ DECISION REQUIRED

**Path A — Backlog.md + MCP** (recommended)
- Tasks as markdown files in `backlog/` directory per project, with frontmatter (id, status, priority, deps)
- MCP server (`mcp-backlog-md`) exposes CRUD to Claude Code, Codex, Cursor, Gemini CLI, OpenCode, Pi
- Deterministic "what's next?" via frontmatter query
- **Free, git-native, offline-capable, zero vendor lock-in**
- ~3k stars but real HN/Reddit traction; right architecture for this profile

**Path B — Linear MCP** (alternative)
- Cloud-hosted, paid, most polished agent integration
- Cycles + Initiatives = the phase hierarchy you've been missing
- Triage automations + agent-created issues (25% of new issues at Linear are agent-created in 2026)
- Vendor lock-in; per-project setup overhead; network dependency

**Path C — Hybrid**
- Linear for **cross-business roadmap** (Alianza/MyChron/Peptideware/Gentle Art holdco level)
- Backlog.md per-project for granular task state
- Highest cognitive overhead but best matches the multi-business reality

**Why Path A is recommended:**
1. Zero vendor lock-in matches your existing `.atlas/` philosophy
2. Tasks travel with code in git → portable across machines, agents, sessions
3. Reversible: Linear has a Backlog.md import path if you decide to migrate later
4. Cost: $0 vs Linear's per-seat
5. Schema continuity: existing `.atlas/plans/` frontmatter maps cleanly to Backlog.md task fields

---

### Layer 4 — Session Switcher
**Tool:** `Claude Squad` or `CCManager` (TUI cockpit)

| Tool | Stars | Strengths | Multi-agent breadth |
|---|---|---|---|
| Claude Squad | ~5,800 | Most-discussed, tmux + worktree per agent | CC, Codex, Gemini, Aider, OpenCode, Amp |
| CCManager | ~2,500 | vi-style search, multi-project discovery, context transfer | CC, Gemini CLI, Codex, Cursor Agent, OpenCode |

Both fill the **"open a TUI, see what I'm working on across projects, jump in"** role you're missing today.

**Companion: `atlas next` resolver** — a small Atlas-native command that aggregates across all `.atlas/plans/*.md` + `backlog/*` per project, returning one ranked answer. This is the function the data shows is missing from every off-the-shelf tool. Implementation: see Q3.

---

### Layer 5 — External / GitHub
**Tool:** `GitHub Issues + Milestones` + `gh-dash`

- **Demoted role:** PR reviews, public bug reports, OSS contributions, milestone-level external coordination
- Stop using GitHub Issues for **private** per-project task tracking — that's Layer 3's job
- gh-dash remains the cockpit for the GitHub-side view
- GitHub Coding Agent / Copilot integration for issue→PR loops (when public-facing)

---

## Recommended Stack — At a Glance

```
┌─────────────────────────────────────────────────────────────────┐
│  Layer 5  External    │ GitHub Issues + gh-dash (PRs, public)  │
├─────────────────────────────────────────────────────────────────┤
│  Layer 4  Session     │ Claude Squad/CCManager + `atlas next`  │
├─────────────────────────────────────────────────────────────────┤
│  Layer 3  State       │ Backlog.md + MCP  (queryable, git)     │ ← decision
├─────────────────────────────────────────────────────────────────┤
│  Layer 2  Plans       │ .atlas/plans/ + .atlas/handoffs/       │ ← already in place
├─────────────────────────────────────────────────────────────────┤
│  Layer 1  Conventions │ AGENTS.md + CLAUDE.md                  │
└─────────────────────────────────────────────────────────────────┘
```

The "what's next?" function becomes:
1. Open Claude Squad / CCManager → see all projects with active sessions
2. Pick a project → `atlas next` → returns the next task aggregated from Backlog.md + plan frontmatter
3. Agent reads AGENTS.md (per-project conventions) + relevant `.atlas/plans/` doc (the why)
4. Work, commit, update Backlog.md task status (via MCP)
5. On stop: handoff written if context is full; otherwise next session resolves cleanly

---

## What Was Considered & Rejected

| Tool | Rejected because |
|---|---|
| **vibe-kanban** | BloopAI sunsetting commercial late April 2026; community-maintained survival uncertain |
| **GitHub Spec Kit** | VS Code/Copilot-centric; CC/Pi integration is DIY |
| **Kiro** | IDE lock-in, doesn't fit multi-agent CLI workflow |
| **BMAD-METHOD** | Powerful planning framework, but heavy upfront ceremony for solo iteration loops |
| **Notion 3.0 Agents** | State decoupled from code repo; per-project setup expensive |
| **Plane (self-hosted)** | More ops than Backlog.md for similar capability; UI less polished than Linear |
| **Beads** | Architecturally elegant (`bd next` deterministic) but very young (v0.59, late 2025) — watch list |
| **claude-task-master** | Strong MCP fit, but ties workflow to Claude branding/ecosystem |
| **GitHub Issues alone** | Milestones are flat (no nested phases); weak deterministic "next" without custom-field automation |

---

## Migration Plan (assumes Path A)

| Phase | Effort | Deliverable | Gate to next phase |
|---|---|---|---|
| **0. Decisions** | ½ day | Ed answers Q1-Q3 below | All three answered |
| **1. Conventions** | 1 day | `AGENTS.md` authored for atlas-config, atlas-recall, alianza, mychron, peptideware, srcd, rainy-day-media | All repos have AGENTS.md ≤150 lines |
| **2. State pilot** | ½ day | Install Backlog.md + MCP in atlas-config + atlas-recall; migrate one in-progress plan's action items to tasks | Both agents (CC + OpenCode) successfully read/write Backlog.md via MCP |
| **3. Cockpit + resolver** | 1 day | Install chosen session switcher; build `atlas next` (Q3 placement); wire into Standup skill | `atlas next` returns deterministic answer in <500ms across pilot projects |
| **4. One-week pilot** | 1 week | Use Backlog.md exclusively for atlas-recall private tasks for a full week; track friction | Friction journal shows ≤3 issues |
| **5. Rollout** | 2 days | Extend to remaining projects; archive private GitHub issues into Backlog.md | All projects on stack |
| **6. Hardening** | ongoing | Standup skill enhancements; cross-project queries; dashboard tweaks | — |

**Reversibility:** Each phase commits independently. Linear migration remains an option after Phase 5 (Linear has Backlog.md import); CCManager↔Claude Squad swap is one config change.

---

## Open Questions for Ed

### Q1 — State layer (must answer before Phase 2)
- [ ] **A: Backlog.md** ← recommended
- [ ] **B: Linear MCP**
- [ ] **C: Hybrid** (Linear for cross-business, Backlog.md per-project)

### Q2 — Session switcher
- [ ] **A: Claude Squad** (more stars, larger community)
- [ ] **B: CCManager** (broader multi-agent breadth — includes OpenCode + Cursor Agent + Gemini CLI explicitly)

### Q3 — `atlas next` resolver placement
- [ ] **A: Justfile recipe** (`just next`) — simplest, no new code
- [ ] **B: Standalone Bun CLI** under `claudecode/.claude/skills/AtlasNext/`
- [ ] **C: Standup skill enhancement** (add `next` subcommand to existing `/Standup`)

### Q4 — Optional: BMAD-METHOD adoption
For projects with **greenfield architecture work** (e.g., a new srcd product), is BMAD's structured persona-driven planning worth the ceremony? It's separate from the 5-layer stack — a planning framework that *produces artifacts* the stack consumes.
- [ ] **A:** Adopt BMAD selectively for new greenfield work
- [ ] **B:** Skip — current `.atlas/plans/` discipline is enough
- [ ] **C:** Defer this decision (revisit after stack is in place)

---

## Out of Scope (this plan)

- Migrating existing GitHub issues to Backlog.md (deferred to Phase 5)
- Building a custom MCP server (use upstream `mcp-backlog-md`)
- Replacing Recall memory database (orthogonal — Recall stays as cross-session memory; Backlog.md handles task state)
- Building a web UI (Backlog.md ships one; Claude Squad/CCManager are TUI; combined coverage is enough)

---

## References

- Visual ranking: [`assets/2026-04-29-agent-workflow-ranking.html`](assets/2026-04-29-agent-workflow-ranking.html)
- Backlog.md: https://github.com/MrLesk/Backlog.md
- AGENTS.md standard: https://agents.md
- Claude Squad: https://github.com/smtg-ai/claude-squad
- CCManager: https://github.com/kbwo/ccmanager
- Linear for Agents: https://linear.app/agents
- Beads (watch list): https://github.com/steveyegge/beads
- BMAD-METHOD: https://github.com/bmad-code-org/BMAD-METHOD
- Plane (OSS Linear alt): https://plane.so

---

## Decision Log

| Date | Decision | Rationale |
|---|---|---|
| 2026-04-29 | Adopt 5-layer architecture (vs single-tool) | All 3 researchers converged here; no single tool covers all needed roles |
| 2026-04-29 | Recommend Path A (Backlog.md) over Linear | Vendor lock-in + cost + offline incompatibility outweigh polish for solo multi-business operator |
| 2026-04-29 | Demote GitHub Issues to external-only | Milestones too flat; private state belongs in repo (Backlog.md), not in cloud-platform issue tracker |
| 2026-04-29 | Defer Beads despite architectural elegance | v0.59 is too young to bet workflow on; revisit Q3 2026 |
