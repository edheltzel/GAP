# Atlas — Global Agent Rules

## PAI System
Read the PAI system for system understanding and initiation:
`read skills/PAI/SKILL.md`

The rules below are MANDATORY and apply to every session, every project, every agent.

## `.atlas/` Directory (MANDATORY)

All project-local Atlas artifacts MUST live under `.atlas/` at the project root:

| Path                          | Purpose                                    |
|-------------------------------|--------------------------------------------|
| `.atlas/plans/`               | Active plan/design/architecture docs       |
| `.atlas/plans/archive/`       | Completed plans                            |
| `.atlas/handoffs/`            | Session handoff docs                       |
| `.atlas/handoffs/archive/`    | Consumed handoffs (archived on read)       |

**Rules:**
- Always use `.atlas/plans/` — never `docs/plans/`, never `.atlas-plans/` (legacy), never inline in other directories
- Create `.atlas/plans/` (or `.atlas/handoffs/`) if it doesn't exist
- Use `.atlas/plans/archive/` for completed plans, `.atlas/handoffs/archive/` for consumed handoffs (singular `archive/`, not `archived/`)
- File naming: `YYYY-MM-DD-description.md` (kebab-case) — use today's date and a clear rundown of the plan title (e.g., `2026-04-05-migrate-auth-to-jwt.md`), never arbitrary or cryptic names
- This applies to ALL projects, ALL agents, no exceptions

## Git Worktrees (MANDATORY)

Always use git worktrees for implementation work unless Ed explicitly says otherwise. Use the built-in `EnterWorktree` tool to create an isolated worktree before starting feature work. This creates a worktree under `.claude/worktrees/` with a new branch based on HEAD. Use `ExitWorktree` when done (keep or remove). This keeps the main working tree clean and enables parallel development.

## MEMORY

You have persistent memory via Recall. **Read the full guide:** /Users/ed/.claude/Recall_GUIDE.md

Core rules:
1. **MANDATORY: Always use `memory_search` FIRST** — before git, file reads, shell commands, or any other source — when answering any question about past events, decisions, installations, or prior work. No exceptions.
2. Before spawning agents (Task tool) → call `context_for_agent`
3. When decisions are made → record with `memory_add`
4. End of session when user says `/dump` → run `mem dump "Descriptive Title"`

Tool syntax:
- `memory_search({ query: "search terms" })`
- `memory_add({ type: "decision", content: "what", detail: "why" })`
- `context_for_agent({ task_description: "what the agent will do" })`

## karpathy-guidelines & PAI ISC

The `karpathy-guidelines` skill (installed via Claude Code plugin marketplace from `forrestchang/andrej-karpathy-skills`) is **supplemental** to PAI ISC, not parallel. PAI's "Nothing escapes the Algorithm" rule means ISC is always active. Where karpathy Section 4 (Goal-Driven Execution) overlaps with ISC, **ISC governs**. Sections 1-3 (assumption surfacing, simplicity-first, surgical changes) are additive discipline that reinforce ISC habits without competing with them.
