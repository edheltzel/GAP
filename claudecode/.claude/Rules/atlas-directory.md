# `.atlas/` Directory (MANDATORY)

All project-local Atlas artifacts MUST live under `.atlas/` at the project root:

| Path                       | Purpose                              |
| -------------------------- | ------------------------------------ |
| `.atlas/plans/`            | Active plan/design/architecture docs |
| `.atlas/plans/archive/`    | Completed plans                      |
| `.atlas/handoffs/`         | Session handoff docs                 |
| `.atlas/handoffs/archive/` | Consumed handoffs (archived on read) |
| `.atlas/worktrees/`        | Git Worktrees                        |

**Rules:**

- Always use `.atlas/plans/` — never `docs/plans/`, never `.atlas-plans/` (legacy), never inline in other directories
- Create `.atlas/plans/` (or `.atlas/handoffs/`) if it doesn't exist
- Use `.atlas/plans/archive/` for completed plans, `.atlas/handoffs/archive/` for consumed handoffs (singular `archive/`, not `archived/`)
- File naming: `YYYY-MM-DD-description.md` (kebab-case) — use today's date and a clear rundown of the plan title (e.g., `2026-04-05-migrate-auth-to-jwt.md`), never arbitrary or cryptic names
- This applies to ALL projects, ALL agents, no exceptions
