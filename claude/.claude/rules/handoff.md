# Handoff Storage Rule

When using the `docs/plans/handoff` skill or creating a session handoff, always save the handoff markdown under the current project's `docs/plans/handoff` directory.

- Resolve the project root with `PROJECT_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"`.
- Create the directory before writing: `mkdir -p "$PROJECT_ROOT/docs/plans/handoff"`.
- Use a timestamped, descriptive Markdown filename, e.g. `YYYY-MM-DD-HHMMSS-handoff-<slug>.md`.
- NEVER write handoff docs to `/tmp`, OS temp folders, home-level scratch dirs, or the Claude/user config directory.
- Report the final handoff path.
