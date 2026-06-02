# `.agents/atlas/` Directory (MANDATORY)

All project-local Atlas artifacts MUST live under `.agents/atlas/` at the project root:

| Path                              | Purpose                              |
| --------------------------------- | ------------------------------------ |
| `.agents/atlas/plans/`            | Active plan/design/architecture docs |
| `.agents/atlas/plans/archive/`    | Completed plans                      |
| `.agents/atlas/handoffs/`         | Session handoff docs                 |
| `.agents/atlas/handoffs/archive/` | Consumed handoffs (archived on read) |
| `.agents/atlas/worktrees/`        | Git Worktrees                        |

**Rules:**

- Always use `.agents/atlas/plans/` — never `docs/plans/`, never `Plans/`, never `.atlas/plans/` (legacy), never inline in other directories
- Create `.agents/atlas/plans/` (or `.agents/atlas/handoffs/`) if it doesn't exist
- Use `.agents/atlas/plans/archive/` for completed plans, `.agents/atlas/handoffs/archive/` for consumed handoffs (singular `archive/`, not `archived/`)
- File naming: `YYYY-MM-DD-description.md` (kebab-case) — use today's date and a clear rundown of the plan title (e.g., `2026-04-05-migrate-auth-to-jwt.md`), never arbitrary or cryptic names
- This applies to ALL projects, ALL agents, no exceptions
