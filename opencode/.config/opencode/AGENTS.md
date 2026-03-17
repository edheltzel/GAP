# Atlas — Global Agent Rules

The rules below are MANDATORY and apply to every session, every project, every agent.

## Plans Directory (MANDATORY)

All plans, handoff documents, and design documents MUST be stored in `.atlas-plans/` at the project root.
- Always use `.atlas-plans/` — never `docs/plans/`, never inline in other directories
- Create `.atlas-plans/` if it doesn't exist
- Use `.atlas-plans/archive/` for completed plans
- File naming: `YYYY-MM-DD-description.md` (kebab-case)
- This applies to ALL projects, no exceptions

## Git Worktrees (MANDATORY)

Always use git worktrees for implementation work unless Ed explicitly says otherwise. Before starting feature work, invoke the `superpowers:using-git-worktrees` skill — it handles worktree creation, directory selection, and safety checks.

## Git

- Conventional Commits
- Feature branches, squash merge to main
- PR descriptions explain WHY, not just WHAT

## MEMORY

You have persistent memory via Recall.

Core rules:
1. Before asking user to repeat anything → search first with `memory_search`
2. Before spawning agents (Task tool) → call `context_for_agent`
3. When decisions are made → record with `memory_add`

Tool syntax:
- `memory_search({ query: "search terms" })`
- `memory_add({ type: "decision", content: "what", detail: "why" })`
- `context_for_agent({ task_description: "what the agent will do" })`
