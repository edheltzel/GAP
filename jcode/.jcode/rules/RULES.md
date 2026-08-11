# Jcode Harness Rules

## Purpose

Own portable Jcode preferences, runtime boundaries, browser setup, and Jcode-specific configuration behavior.

## Local Contracts

- Keep universal cross-harness behavior in `~/AGENTS.md`, not here.
- Track only curated files that Jcode reads from `~/.jcode/`.
- Keep credentials, sessions, memory, caches, telemetry, generated hotkeys, model data, logs, builds, browser assets, and runtime state machine-local.
- `config.toml` is writable because Jcode normalizes it in place.
- The `jcode-config` Git clean filter removes theme selection and generated `launch_hotkeys` state before indexing.
- Keep `provider.default_provider` and `provider.default_model` aligned when changing the preferred model.
- Stow the Jcode package with `--no-folding` so `~/.jcode/` remains a real runtime directory.

## Work Guidance

- Add settings only when they are durable preferences worth syncing.
- Prefer environment-variable references over credentials or machine-specific values.
- Do not track `mcp.json` until generated paths, versions, hashes, and home-directory values are made portable.
- Install or repair Firefox Agent Bridge with `jcode browser setup`; do not track `~/.jcode/browser/`.

## Verification

- Run `just jcode` and confirm `~/.jcode/config.toml` links to this package.
- Confirm local `display.theme` changes do not appear in `git diff -- jcode/.jcode/config.toml`.
- Run `jcode browser status` when browser automation is expected.
- Inspect `git diff -- jcode/` for secrets, absolute paths, generated state, and theme changes.

## Child Rules Index

- None.
