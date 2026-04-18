# Atlas — Global Agent Rules

The rules below are MANDATORY and apply to every session, every project, every agent.

## `.atlas/` Directory (MANDATORY)

All project-local Atlas artifacts MUST live under `.atlas/` at the project root:

| Path                          | Purpose                                    |
|-------------------------------|--------------------------------------------|
| `.atlas/plans/`               | Active plan/design/architecture docs       |
| `.atlas/plans/archive/`       | Completed plans                            |
| `.atlas/handoffs/`            | Session handoff docs                       |
| `.atlas/handoffs/archive/`    | Consumed handoffs                          |

- Always use `.atlas/plans/` — never `docs/plans/`, never `.atlas-plans/` (legacy), never inline in other directories
- Create `.atlas/plans/` (or `.atlas/handoffs/`) if it doesn't exist
- Use `.atlas/plans/archive/` for completed plans (singular `archive/`, not `archived/`)
- File naming: `YYYY-MM-DD-description.md` (kebab-case)
- Legacy `.atlas-plans/` layouts can be migrated with `/MigrateAtlasDir` (dry-run by default)
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
