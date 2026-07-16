# OMP config

OMP-native config is stowed from `omp/.omp/agent/` to `~/.omp/agent/`.

## Track

Track authored, portable files only:

- `config.yml` — global OMP settings.
- `models.yml` — custom model/provider definitions. Use environment-variable names for keys, never literal tokens.
- `mcp.json` — OMP-owned MCP server config and discovered-server overrides. Use `disabledServers` to suppress foreign MCP entries that OMP must not launch; prefer `${ENV}` placeholders or env-name indirection for configured servers.
- `keybindings.yml` — user key remaps.
- `APPEND_SYSTEM.md` — preferred global prompt additions; keeps OMP defaults, generated skills/rules/tool guidance, and project context.
- `SYSTEM.md` — rare; replaces OMP's stable default instruction block, so do not add unless that is intentional.
- `skills/*/SKILL.md` — one skill directory per skill, with explicit `name` and `description` frontmatter.
- `commands/*.md`, `rules/*.{md,mdc}`, `prompts/*.md`, `instructions/*.md`.
- `tools/*.ts`, `tools/<name>/index.ts` — authored custom tools.
- `extensions/*.ts`, `extensions/<name>/index.ts` — authored extensions. Do not track files marked as installed or managed by another app.
- `themes/*.json` — custom OMP themes.

## Never track

Runtime state, credentials, generated caches, and local identity stay out of git:

```text
.omp/install-id
.omp/.env
.omp/agent/.env
.omp/agent/secrets.yml
.omp/agent/agent.db*
.omp/agent/history.db*
.omp/agent/models.db*
.omp/agent/sessions/
.omp/agent/blobs/
.omp/agent/terminal-sessions/
.omp/agent/memories/
.omp/agent/cache/
.omp/cache/
.omp/auth-broker.token
.omp/auth-gateway.token
```

`secrets.yml` stores actual secret values or regexes and is never repo material.

## Adding OMP files

1. Stowed global config goes under `omp/.omp/agent/`.
2. Installable OMP extension packages go under `packages/<name>/` with `package.json#omp.extensions`.
3. Marketplace entries go in `.claude-plugin/marketplace.json`.
4. If the matching live stowed file already exists under `~/.omp/agent/`, use `stow --adopt` once or move the file into the repo before restowing.
5. Run `just up` after changing stowed global config. Package-only changes do not need stow.
6. If stow reports a conflict, resolve the live file deliberately; do not delete runtime state directories to make stow pass.

## Atlas status line

`omp/.omp/agent/extensions/atlas-status-line.ts` owns the canonical Atlas footer in OMP. `omp/.omp/agent/config.yml` lists it explicitly because stow folds the authored `extensions/` directory into a symlink and native glob discovery does not traverse that linked root. Run `just up` when adding an authored OMP file and the scoped `just omp` workflow for later package-only restows; focused contract tests live outside the stow package at `tests/omp/atlas-status-line.test.ts`.

- Render the authored Atlas fields in OMP's native status-line shape: `▲ Atlas v{LIFEOS_VERSION}  {provider}/{model} ⚡︎ {thinking}  π {OMP_VERSION}  …/{parent}/{current}   {branch}  {context meter} {context}%   {cache}%`, followed by the native horizontal divider and a right-aligned session title when one exists. Cache remains hidden until cumulative session usage reports real cache read/write activity.
- Register a below-editor widget as the extension's TUI lifecycle anchor, then install the Atlas primary row through the editor's `setTopBorderProvider` so it interrupts the input's top border instead of occupying a separate footer row. Match OMP's default top-border composition: grouped content uses thin powerline separators, the divider fills the gap before the session title, and the editor supplies any remaining trailing rule and right corner. Restore OMP's native `getTopBorder` provider and leave `borderColor` untouched when the widget is disposed or the editor instance changes.
- Keep the primary status line to one row with native group insets. At narrow widths, drop path, OMP version, branch, Atlas identity, then model metadata; preserve the responsive context meter and cache, and ANSI-truncate the session title into the right group when space remains.
- Read the Atlas version from `~/.claude/LIFEOS/VERSION`; derive model, provider, thinking, OMP version, compaction-aware context, session title, and cumulative cache usage from live OMP state. Re-read the session working directory and active branch during rendering so `/move` updates the existing footer without a restart.
- Capture `ui.setStatus` publications and render them through the widget as inset additional rows beneath the editor, preserving trusted ANSI SGR styling while removing other controls. Terminate every published status with a full SGR reset and reapply the opaque row background so styles cannot bleed between publishers or beyond the row. Restore OMP's original status publisher on session switch or shutdown; measure and truncate with OMP's ANSI-aware TUI helpers.
- Derive footer text, separator, semantic percentage, icon, path, branch, divider, and background colors from the active OMP theme. `omp/.omp/agent/themes/atlas-default.json` is an intentional snapshot of OMP 17.0.1's built-in dark theme with only its name changed; `config.yml` selects it as `theme.dark`. Other themes remain compatible because the footer consumes semantic theme tokens.
- Use the theme's `statusLineBg` behind the primary groups and full-width auxiliary status rows unless `statusLine.transparent: true`, which inherits the terminal background. The context bar alone retains the canonical Atlas dark or Rosé Pine Dawn gradient and dim empty `⛁` cells, selected by light/dark appearance, with widths of 6 (`<35` columns), 8 (`35–54`), 16 (`55–79`), or 24 (`80+`). Its percentage uses the active theme's semantic colors.
- Published OMP 17 still types `ctx.ui.setFooter` but leaves it inert in interactive mode. Do not call it or restore downstream core wiring; the extension-only bridge uses the widget factory's TUI handle, public component children, the native status component's `getTopBorder`, and the editor's public `setTopBorderProvider` and `borderColor` surfaces.

## OMP updates

The published OMP package installed by the official updater is canonical. OMP 17 incorporated the streamed-response fixes that previously required local patches; the Atlas footer now uses the release's supported widget API instead of patching OMP core.

- Run `omp update` to install the latest published release.
- Run `omp update --check` for a non-installing availability check.
- Run `just omp` after changing authored files under `omp/.omp/agent/`; stow manages configuration, not the OMP binary.
- Never relink `~/.bun/bin/omp` to a local source wrapper, revive a retired patch branch, or pin an older release.

## Current tracked baseline

Tracked OMP-authored config and packages currently include:

- `omp/.omp/agent/RULES.md`
- `omp/.omp/agent/config.yml`
- `omp/.omp/agent/mcp.json`
- `omp/.omp/agent/extensions/codegraph.ts`
- `omp/.omp/agent/extensions/atlas-status-line.ts`
- `omp/.omp/agent/themes/atlas-eldritch.json`
- `omp/.omp/agent/themes/atlas-default.json`
- `packages/omp-themis/` — installable OMP Themis extension package.
- `.claude-plugin/marketplace.json` — local marketplace catalog exposing `themis`.

`omp/.omp/agent/extensions/herdr-omp-agent-state.ts` is currently tracked but remains herdr-managed and may be overwritten by integration updates. Do not treat it as authored baseline or change its ownership incidentally; reconcile tracking separately.
