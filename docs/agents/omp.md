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

`omp/.omp/agent/extensions/atlas-status-line.ts` owns the canonical Atlas footer in OMP. `omp/.omp/agent/config.yml` lists it explicitly because stow folds the authored `extensions/` directory into a symlink and native glob discovery does not traverse that linked root. Install both with the scoped `just omp` workflow; focused contract tests live outside the stow package at `tests/omp/atlas-status-line.test.ts`.

- Render `▲ Atlas v{LIFEOS_VERSION} · {provider}/{model} ⚡︎ {thinking} · π {OMP_VERSION} · …/{parent}/{current} ·  {branch} · {context meter} {context}%`, followed by ` {cache}%` only after a successful assistant request reports real cache read/write usage.
- Read the Atlas version from `~/.claude/LIFEOS/VERSION`; derive model, provider, thinking, OMP version, active branch, compaction-aware context, and latest successful cache usage from live OMP state.
- Keep filesystem and provider labels control-safe; preserve trusted ANSI styling published through `ui.setStatus`. Measure and truncate with OMP's ANSI-aware TUI helpers.
- Use the canonical Eldritch dark and Rosé Pine Dawn light palettes. The green → yellow → orange → red context gradient uses dim empty `⛁` cells and responsive widths of 6 (`<35` columns), 8 (`35–54`), 16 (`55–79`), or 24 (`80+`).
- The downstream `atlas/fixes` branch must retain interactive `ctx.ui.setFooter` support and its read-only footer data provider. Published OMP currently types this API but leaves it inert in interactive mode; the extension cannot replace the built-in footer or preserve extension statuses without that patch.

## Patched source installation

The active OMP executable is source-linked through `~/Developer/AI/Harness/omp/.agents/worktrees/atlas-fixes`. That canonical path resolves to the verified worktree carrying the persistent `atlas/fixes` branch; both are local runtime state and are not tracked by atlas-config.

- `just omp-patched-status` verifies that the active executable resolves to the expected source wrapper, then reports the patch branch, worktree, version, and cleanliness. It exits nonzero when the source link is inactive.
- `just omp-patched-update` requires an active source link and a clean `atlas/fixes` worktree, fetches stable tags from the configured upstream into an isolated ref namespace, and creates a disposable candidate worktree. It rebases and verifies the candidate with a frozen dependency install, native build, streamed-response regressions, and OMP smoke check. Only then are the canonical worktree and executable links atomically switched and verified; failed candidates roll back to the previous branch and links, and the old worktree is removed only after promotion commits.
- Do not run `omp update` while the patched source installation is active. OMP's Bun updater replaces the source link with the published package.
- `OMP_SOURCE_DIR`, `OMP_PATCH_WORKTREE`, and `OMP_UPSTREAM_REMOTE` override the default checkout, worktree, and remote when needed.

The patch stack currently includes interactive custom-footer wiring required by the Atlas status line and backports upstream PR [#5149](https://github.com/can1357/oh-my-pi/pull/5149), which preserves streamed Responses text when a terminal `output_item.done` message has empty content. When either fix lands upstream, the next rebase should drop only the redundant patch; verify its regression contract before removing the downstream commit deliberately.

## Current tracked baseline

Tracked OMP-authored config and packages currently include:

- `omp/.omp/agent/RULES.md`
- `omp/.omp/agent/config.yml`
- `omp/.omp/agent/mcp.json`
- `omp/.omp/agent/extensions/codegraph.ts`
- `omp/.omp/agent/extensions/atlas-status-line.ts`
- `packages/omp-themis/` — installable OMP Themis extension package.
- `.claude-plugin/marketplace.json` — local marketplace catalog exposing `themis`.

`omp/.omp/agent/extensions/herdr-omp-agent-state.ts` is currently tracked but remains herdr-managed and may be overwritten by integration updates. Do not treat it as authored baseline or change its ownership incidentally; reconcile tracking separately.
