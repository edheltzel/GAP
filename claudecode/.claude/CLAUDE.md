# Atlas — Global Agent Rules

## PAI System
Read the PAI system for system understanding and initiation:
`read skills/PAI/SKILL.md`

The rules below are MANDATORY and apply to every session, every project, every agent.
`read skills/PAI/SKILL.md`

## Plans Directory (MANDATORY)

All plans, handoff documents, and design documents MUST be stored in `.atlas-plans/` at the project root. This applies to:
- Implementation plans
- Design documents
- Handoff documents
- Architecture decisions
- Any plan-like artifact

**Rules:**
- Always use `.atlas-plans/` — never `docs/plans/`, never inline in other directories
- Create `.atlas-plans/` if it doesn't exist
- Use `.atlas-plans/archive/` for completed plans
- File naming: `YYYY-MM-DD-description.md` (kebab-case)
- This applies to ALL projects, ALL agents, no exceptions

## Git Worktrees (MANDATORY)

Always use git worktrees for implementation work unless Ed explicitly says otherwise. Before starting feature work, invoke the `superpowers:using-git-worktrees` skill — it handles worktree creation, directory selection, and safety checks. This keeps the main working tree clean and enables parallel development.

## MEMORY

You have persistent memory via Recall. **Read the full guide:** /Users/ed/.claude/Recall_GUIDE.md

Core rules:
1. Before asking user to repeat anything → search first with `memory_search`
2. Before spawning agents (Task tool) → call `context_for_agent`
3. When decisions are made → record with `memory_add`
4. End of session when user says `/dump` → run `mem dump "Descriptive Title"`

Tool syntax:
- `memory_search({ query: "search terms" })`
- `memory_add({ type: "decision", content: "what", detail: "why" })`
- `context_for_agent({ task_description: "what the agent will do" })`
