# Pi Agent Configuration

## HANDOFFS

When using the `/handoff` skill or creating a session handoff, follow the nearest repository
`AGENTS.md`/DOX contract for the canonical handoff location.

- Resolve the project root with `PROJECT_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"`.
- If the repository defines no handoff location, default to
  `$PROJECT_ROOT/docs/plans/handoffs/` and create it before writing.
- Never let the global fallback override a committed repository location such as
  `docs/handoffs/`.
- Use a timestamped, descriptive Markdown filename, e.g.
  `YYYY-MM-DD-HHMMSS-handoff-<slug>.md`.
- Never write handoff docs to `/tmp`, OS temp folders, home-level scratch dirs, or the agent
  install/config directory.
- Report the final handoff path.

## COMPOUND ENGINEERING

- Compound Engineering is an intentional global package exception, loaded directly from the official `git:github.com/EveryInc/compound-engineering-plugin` Pi package in `settings.json`.
- Do not restore converter-managed `ce-*` skills or agents under `~/.pi/agent/`; the native package owns skill discovery from its Git checkout.
- Keep `npm:@tintinweb/pi-subagents` and `npm:pi-ask-user` installed as workflow companions.

## TOOL RENDERING

- `npm:@mobrienv/pi-tidy-tools@0.3.0` is the pinned global renderer for Pi's built-in tools. It adds `/tidy`, `/diff`, and `ctrl+shift+o`.
- Keep the package version-pinned in `settings.json`; review source and release notes before updating it.
- Runtime preferences belong in machine-local `~/.pi/agent/pi-tidy-tools.json`. Do not run optional `/tidy pi-fff setup` without explicit user confirmation and a review of its settings changes.

## STATUS LINE

- `extensions/atlas-status-line.ts` owns Pi's global custom footer; `claude/.claude/Atlas/atlas-statusline.sh` is its canonical visual and information-design reference.
- Keep the display order Atlas/version → provider/model/thinking → Pi/version → directory/branch → context meter/cache.
- Render the working directory as `…/<parent>/<current>`; do not expose the full path.
- Match the Claude line's 24-cell `⛁` context meter at normal widths, responsive smaller meters at narrow widths, and whole-number context percentage from `ctx.getContextUsage()`.
- Cache percentage is the latest successful assistant prompt's `cacheRead / (input + cacheRead + cacheWrite)`, rounded to a whole percent.
- Preserve the canonical text symbols and Nerd Font branch/cache glyphs without emoji presentation, and preserve ANSI-styled status text published by other extensions.
- After changes, run `just pi` and reload Pi with `/reload`.

## MEMORY

Read and follow the canonical Recall guide at `~/.pi/agent/Recall_GUIDE.md`. Use the live `recall-memory` MCP schemas as the source of truth for tool call shapes; do not duplicate them here.
