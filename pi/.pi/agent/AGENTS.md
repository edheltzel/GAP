<!-- BEGIN COMPOUND PI TOOL MAP -->
## Compound Engineering (Pi compatibility)

This block is managed by compound-plugin.

Pi extensions used by this plugin:
- Required: `pi-subagents` (by nicobailon) provides the `subagent` tool used by skills that dispatch parallel agents
- Recommended: `pi-ask-user` (by edlsh) provides the `ask_user` tool; skills fall back to numbered options in chat when it is missing

Install with:
  pi install npm:pi-subagents
  pi install npm:pi-ask-user
<!-- END COMPOUND PI TOOL MAP -->

## HANDOFFS

When using the `/handoff` skill or creating a session handoff, always save the handoff markdown under the current project's `.agents/atlas/handoffs/` directory.

- Resolve the project root with `PROJECT_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"`.
- Create the directory before writing: `mkdir -p "$PROJECT_ROOT/.agents/atlas/handoffs"`.
- Use a timestamped, descriptive Markdown filename, e.g. `YYYY-MM-DD-HHMMSS-handoff-<slug>.md`.
- Never write handoff docs to `/tmp`, OS temp folders, home-level scratch dirs, or the agent install/config directory.
- Report the final handoff path.

## TOOL RENDERING

- `npm:@mobrienv/pi-tidy-tools@0.3.0` is the pinned global renderer for Pi's built-in tools. It adds `/tidy`, `/diff`, and `ctrl+shift+o`.
- Keep the package version-pinned in `settings.json`; review source and release notes before updating it.
- Runtime preferences belong in machine-local `~/.pi/agent/pi-tidy-tools.json`. Do not run optional `/tidy pi-fff setup` without explicit user confirmation and a review of its settings changes.

## MEMORY

Read and follow the canonical Recall guide at `~/.pi/agent/Recall_GUIDE.md`. Use the live `recall-memory` MCP schemas as the source of truth for tool call shapes; do not duplicate them here.




