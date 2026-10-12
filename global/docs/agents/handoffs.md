# Handoffs

## Rules

- Resolve project root, then follow the nearest AGENTS/DOX handoff location.
- If the repo defines none, default to `$PROJECT_ROOT/docs/plans/handoffs/` (or the project's documented equivalent) and create it before writing.
- Never write handoffs only to `/tmp`, OS temp, home scratch, or harness install/config dirs.
- Use a timestamped descriptive Markdown name.
- End the handoff with a `## Todo` section of todo.txt lines for the open work, and add them to `~/todo.txt` with tuxedo. See [todo-txt.md](todo-txt.md).
- Report the final path.

## Done

The next agent can open the handoff from the path you reported.
