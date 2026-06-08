# Global OMP Rules

## Handoff Storage

When using the `/handoff` skill or creating a session handoff, always save the handoff markdown under the current project's `.agents/atlas/handoffs/` directory.

- Resolve the project root with `PROJECT_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"`.
- Create the directory before writing: `mkdir -p "$PROJECT_ROOT/.agents/atlas/handoffs"`.
- Use a timestamped, descriptive Markdown filename, e.g. `YYYY-MM-DD-HHMMSS-handoff-<slug>.md`.
- Never write handoff docs to `/tmp`, OS temp folders, home-level scratch dirs, or the OMP/user config directory.
- Report the final handoff path.
