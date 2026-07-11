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

## Current tracked baseline

Tracked OMP-authored config and packages currently include:

- `omp/.omp/agent/RULES.md`
- `omp/.omp/agent/config.yml`
- `omp/.omp/agent/mcp.json`
- `omp/.omp/agent/extensions/codegraph.ts`
- `packages/omp-themis/` — installable OMP Themis extension package.
- `.claude-plugin/marketplace.json` — local marketplace catalog exposing `themis`.

`omp/.omp/agent/extensions/herdr-omp-agent-state.ts` is currently tracked but remains herdr-managed and may be overwritten by integration updates. Do not treat it as authored baseline or change its ownership incidentally; reconcile tracking separately.
