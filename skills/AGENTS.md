# AGENTS.md

## Purpose

`skills/` holds the vendored skill sources and catalogs. Installed Do-Skills skills come from the pinned component; see [Manage skills](../README.md#manage-skills).

## Ownership

- One skill is `skills/<scope>/<group>/<skill>/SKILL.md`. Scope is `global` or `project`.
- `skills/global/` and `skills/project/` are scopes, not separate workflows. Do not add an AGENTS.md per skill.
- Installation selection is in [Manage skills](../README.md#manage-skills). `skills.sh.json` owns catalog group blurbs.
- `scripts/update-readme.sh` owns the generated catalog: scope and group READMEs under `skills/`, and the root `README.md` block between `<!-- skills-start -->` and `<!-- skills-end -->`.

## Local Contracts

- A name `gap` installs from this repo must match exactly one `skills/*/*/<name>/SKILL.md`.
- Bundled copies under a skill's `references/` are not separate skills. The catalog generator ignores them.
- After adding, moving, or renaming a skill, follow [Manage skills](../README.md#manage-skills) for installation selection, then regenerate the catalog. Do not hand-edit the generated READMEs or the marked README block.
- Human pages linked from promoted group READMEs live in `docs/global/` and `docs/project/`. They are not skill source.

## Work Guidance

- Group blurbs come from `skills.sh.json` (`"<Scope>: <Group>"`). Unlisted groups on disk are appended by the generator.

## Verification

- After a catalog change, `bash scripts/update-readme.sh` leaves no diff on a second run.

## Child DOX Index

None.
