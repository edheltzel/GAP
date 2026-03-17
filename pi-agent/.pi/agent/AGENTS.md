# Full-Stack Development

## Stack Awareness
- Detect and respect the project's tech stack
- Frontend: React/Vue/Svelte component patterns
- Backend: REST/GraphQL conventions, middleware patterns
- Database: Migration-first, never raw SQL in application code

## Quality
- Write tests for new features when asked
- Update docs when changing public APIs
- Consider accessibility (a11y) for frontend
- Performance: measure before optimizing

## Architecture
- Separation of concerns — keep layers clean
- Error boundaries at service edges
- Configuration via environment variables
- Logging: structured, leveled, no sensitive data

## Git
- Conventional Commits
- Feature branches, squash merge to main
- PR descriptions explain WHY, not just WHAT

## Plans Directory (MANDATORY)
All plans, handoff documents, and design documents MUST be stored in `.atlas-plans/` at the project root.
- Always use `.atlas-plans/` — never `docs/plans/`, never inline in other directories
- Create `.atlas-plans/` if it doesn't exist
- Use `.atlas-plans/archive/` for completed plans
- File naming: `YYYY-MM-DD-description.md` (kebab-case)
- This applies to ALL projects, no exceptions

## Git Worktrees (MANDATORY)
Always use git worktrees for implementation work unless Ed explicitly says otherwise. Before starting feature work, invoke the `superpowers:using-git-worktrees` skill — it handles worktree creation, directory selection, and safety checks.

## Ant Colony Auto-Trigger
If the ant_colony tool is available, automatically use it when the task is complex:
- 3 or more files likely need changes
- 2 or more independent workstreams exist
- large refactor / migration / feature implementation

For simple one-file tasks, execute directly without colony overhead.
After launching ant_colony, use passive mode: wait for COLONY_SIGNAL updates and do not poll bg_colony_status unless the user explicitly asks for a manual snapshot.

---

# my-pi Extensions & Skills

Loaded from `~/.pi/agent/git/github.com/noahsaso/my-pi/`

## Custom Tools

### Todo Tracking (`todo_list`, `todo_add`, `todo_toggle`, `todo_remove`)
Track work items during complex tasks with dependency support.

### Subagents (`subagent`)
Delegate to specialized agents: `scout`, `planner`, `reviewer`, `worker`, `remover`

### Clipboard (`clipboard_read`, `clipboard_write`)
Read/write system clipboard.

### Notifications (`notify`, `ask_user`)
System notifications with chime sound.

### File Watcher (`watch_start`, `watch_stop`, `watch_list`, `watch_events`)
Watch files/directories for changes.

### Code AST (`ast_references`, `ast_rename`, `ast_symbols`)
TypeScript-aware code intelligence.

### Image Generation (`generate_image`)
Generate images via Google Antigravity.

### Persistent Memory (`memory_save`, `memory_search`, `memory_list`, `memory_remove`)
Memories across sessions, auto-injected into prompt.

### Remote Access (`/remote`)
Remote terminal via WebSocket with Tailscale integration.

### Side Conversations (`/btw`)
Separate conversation while main agent works.

## Skills

- **browser-tools** — Browser automation via Chrome DevTools Protocol
- **superpowers** — 14 workflow skills: brainstorming, writing-plans, test-driven-development, systematic-debugging, verification-before-completion, etc.
