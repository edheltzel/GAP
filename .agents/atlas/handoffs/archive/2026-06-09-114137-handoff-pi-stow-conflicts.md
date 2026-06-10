# Handoff: Pi stow conflicts

Date: 2026-06-09
Project: `~/Developer/atlas-config`

## Context

While adding the OMP `atlas-statusline` theme, `just up` was run from the repo root. The command restows packages in this order from `justfile`:

```sh
for pkg in claude codex omp; do stow -d "$repo" -t "$HOME" --restow "$pkg"; done
stow -d "$repo" -t "$HOME" --no-folding --restow pi
```

`claude`, `codex`, and `omp` completed far enough that the new OMP theme is live. The command then failed on the `pi` package.

## Current state observed

OMP change is live despite the overall `just up` failure:

- `~/.omp/agent/config.yml` resolves `theme.dark: atlas-statusline`
- `~/.omp/agent/themes/atlas-statusline.json` exists
- `omp config get theme.dark` returned `atlas-statusline`
- `omp gallery --tool read --state success --width 100` rendered successfully under dark appearance
- `omp gallery --tool bash --state error --width 100` rendered successfully under dark appearance

## Failure

`just up` exits 1 because `stow --no-folding --restow pi` finds many existing real files under `~/.pi/agent/` that are neither symlinks nor directories.

Representative error:

```text
WARNING! stowing pi would cause conflicts:
  * cannot stow Developer/atlas-config/pi/.pi/agent/AGENTS.md over existing target .pi/agent/AGENTS.md since neither a link nor a directory and --adopt not specified
  * cannot stow Developer/atlas-config/pi/.pi/agent/agents/ce-adversarial-document-reviewer.md over existing target .pi/agent/agents/ce-adversarial-document-reviewer.md since neither a link nor a directory and --adopt not specified
  * cannot stow Developer/atlas-config/pi/.pi/agent/compound-engineering/install-manifest.json over existing target .pi/agent/compound-engineering/install-manifest.json since neither a link nor a directory and --adopt not specified
  * cannot stow Developer/atlas-config/pi/.pi/agent/skills/ce-agent-native-architecture/SKILL.md over existing target .pi/agent/skills/ce-agent-native-architecture/SKILL.md since neither a link nor a directory and --adopt not specified
```

The full captured failure was available in-session as `artifact://7`.

## Likely cause

The live Pi agent directory already contains installed Compound Engineering agents/skills as real files. The repo also contains corresponding tracked files under `pi/.pi/agent/`. Because Pi is stowed with `--no-folding`, Stow wants to place symlinks for curated files into the real `~/.pi/agent` tree, but it refuses to overwrite existing real files unless `--adopt` is used or the live files are moved out of the way.

This matches the repo policy in `AGENTS.md`:

- Pi is stowed with `--no-folding` so runtime state stays local.
- Curated Pi config files are tracked.
- Runtime/package state should stay in the real `~/.pi/agent`, not in atlas-config.

## Decision needed

Decide whether these CE agents/skills are supposed to be curated repo-owned config or runtime/package-managed Pi state.

### Option A — repo owns them

Use a deliberate one-time adoption flow:

```sh
stow -d ~/Developer/atlas-config -t ~ --no-folding --adopt pi
```

Then inspect the repo diff carefully before keeping it. Adoption will move live file contents into the repo, which is only correct if the repo should own these Pi files.

### Option B — Pi runtime owns them

Add ignore patterns to `pi/.stow-local-ignore` for package-installed CE agents/skills/runtime manifests, then restow. This is likely correct if these files are installed/updated by Pi or Compound Engineering rather than hand-authored atlas-config.

Candidate patterns to evaluate, not blindly apply:

```text
\.pi/agent/agents/ce-.*\.md
\.pi/agent/skills/ce-.*/
\.pi/agent/compound-engineering/
```

Need verify against what `pi/.pi/agent/` intentionally tracks before committing this direction.

## Suggested next steps

1. Inspect `pi/.pi/agent/` tracked intent and `pi/.stow-local-ignore`.
2. Compare a few conflicting live files against repo files to determine whether live package state diverged from tracked config.
3. Pick repo-owned vs runtime-owned.
4. If repo-owned: run one-time `stow --adopt pi`, inspect every adopted change, keep only intended differences.
5. If runtime-owned: update `pi/.stow-local-ignore`, then run `just up` and confirm it exits 0.

## Commands already run

```sh
just up
omp config get theme.dark
COLORFGBG='15;0' omp gallery --tool read --state success --width 100
COLORFGBG='15;0' omp gallery --tool bash --state error --width 100
```

## Files changed before handoff

Unrelated to the Pi issue, but in the same working tree:

- `omp/.omp/agent/themes/atlas-statusline.json` added
- `omp/.omp/agent/config.yml` changed from `theme.dark: dark-tokyo-night` to `theme.dark: atlas-statusline`

## Resolution (archived 2026-06-09)

Resolved via Option B (Pi runtime owns the CE files). `pi/.stow-local-ignore` and `.gitignore` gained `ce-*` / `lfg` / `compound-engineering` ignore patterns; the 202 corresponding files were removed from the repo worktree so those skills are runtime-owned in `~/.pi/agent` only. Verified: `just status` reports 0 stow conflicts across all packages. The OMP theme work noted under "Files changed before handoff" was committed separately.
