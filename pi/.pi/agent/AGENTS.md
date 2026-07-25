# Pi Agent Configuration

## Global Rules

These rules are MANDATORY with NO EXCEPTIONS for operating is loaded from `~/.pi/agent/rules/global-agent-instructions.md`. Do not duplicate it here.

## MEMORY

Read and follow the canonical Recall guide at `~/.pi/agent/Recall_GUIDE.md`. Use the live `recall-memory` MCP schemas as the source of truth for tool call shapes; do not duplicate them here.

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
- Keep `npm:@tintinweb/pi-subagents` and the pinned
  `git:github.com/edheltzel/pi-ask-user@v0.13.0-herdr.1` fork installed as
  workflow companions.
- The fork preserves `pi-ask-user`'s full UI, schema, and decision-gate skill
  while adding Herdr blocked-state and metadata hooks. Do not also install
  `npm:pi-ask-user` or `npm:pi-ask-herdr`; either would duplicate `ask_user`.

## CURATED SKILLS

- `skills/RedTeam/` and `skills/art/` are deliberate Pi-native conversions of their authoritative Claude skills; compare every source file before syncing.
- Preserve RedTeam concepts and trigger coverage while adapting Claude-only paths, commands, agent syntax, and lifecycle rules to functional Pi equivalents.
- For PAI-to-Pi conversions, use a lowercase-hyphen directory and matching `name:`; use Pi frontmatter without Claude-only `version` or `effort` fields.
- Rewrite runtime paths to `~/.pi/agent/skills/<name>/`, keep each skill self-contained, and replace Claude-only hooks with harness-neutral enforcement where needed.
- Send best-effort notifications to Echo at `localhost:8888`, never disabled Pulse port `31337`.
- Pi skill customizations live under `~/.pi/agent/SKILLCUSTOMIZATIONS/<name>/`; execution logs live under `~/.pi/agent/memory/skills/`.
- Keep `SKILL.md` valid under Pi's Agent Skills rules and make every local workflow reachable through relative progressive-disclosure links.
- Bundled Bun tools must install dependencies beside the real repository files, not through the stowed `~/.pi` symlinks, because Bun resolves imports from realpaths; keep `node_modules` and `.env` ignored.
- Validate tracked sources directly in disposable worktrees. Restow with `just pi` only from the canonical clone after merge.

## TOOL RENDERING

- `npm:@mobrienv/pi-tidy-tools@0.3.0` is the pinned global renderer for Pi's built-in tools. It adds `/tidy`, `/diff`, and `ctrl+shift+o`.
- Keep the package version-pinned in `settings.json`; review source and release notes before updating it.
- Runtime preferences belong in machine-local `~/.pi/agent/pi-tidy-tools.json`. Do not run optional `/tidy pi-fff setup` without explicit user confirmation and a review of its settings changes.

## THEME

- `themes/atlas-eldritch.json` is Pi's curated Eldritch theme; `settings.json`
  selects it by name.
- Keep its reusable palette aligned with the Claude and OMP Eldritch themes while
  mapping only Pi-supported semantic tokens.
- Validate theme changes against Pi's installed schema and loader. Apply them with
  `just pi` only from the canonical clone, never from a disposable worktree.

## STATUS LINE

- `extensions/atlas-status-line.ts` owns Pi's global custom footer; `claude/.claude/Atlas/atlas-statusline.sh` is its canonical visual and information-design reference.
- Keep the display order Atlas/version → provider/model/thinking → Pi/version → directory/branch → context meter/cache.
- Render the working directory as `…/<parent>/<current>`; do not expose the full path.
- Match the Claude line's 24-cell `⛁` context meter at normal widths, responsive smaller meters at narrow widths, and whole-number context percentage from `ctx.getContextUsage()`.
- Cache percentage is the latest successful assistant prompt's `cacheRead / (input + cacheRead + cacheWrite)`, rounded to a whole percent.
- Preserve the canonical text symbols and Nerd Font branch/cache glyphs without emoji presentation, and preserve ANSI-styled status text published by other extensions.
- After changes, run `just pi` and reload Pi with `/reload`.

## Child DOX Index

- `extensions/pi-better-btw/AGENTS.md` — `/btw` overlay behavior, UI contracts, and verification.

## Maintaining this file

Keep this file for knowledge useful to almost every future agent session in this project.
Do not repeat what the codebase already shows; point to the authoritative file or command instead.
Prefer rewriting or pruning existing entries over appending new ones.
When updating this file, preserve this bar for all agents and keep entries concise.
