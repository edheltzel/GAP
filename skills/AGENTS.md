# AGENTS.md

## Purpose

`skills/` is the source for Gap-native skills whose `global/skills.json` entry uses `"source": "."`. The 66 Do skills are not copied here. They come from [edheltzel/Do-Skills](https://github.com/edheltzel/Do-Skills) at the commit in `gap.lock`.

## Ownership

- One Gap-native skill is `skills/<scope>/<group>/<skill>/SKILL.md`. Scope is `global` or `project`.
- Do skills are edited in Do-Skills, not here. Bump `do-skills.commit` (and `tree`) in `gap.lock` after a Do-Skills merge, then `gap sync`.
- `global/skills.json` lists what `gap sync` installs and checks. `skills.sh.json` owns catalog group blurbs.
- `scripts/update-readme.sh` owns the generated catalog in the root `README.md` block between `<!-- skills-start -->` and `<!-- skills-end -->`.

## Local Contracts

- A name `gap` installs from this checkout must match exactly one `skills/*/*/<name>/SKILL.md`.
- A name `gap` installs from Do-Skills must match exactly one skill in that pin's `skills/*/*/<name>/SKILL.md`.
- Bundled copies under a skill's `references/` are not separate skills.
- After adding, moving, or renaming a Gap-native skill, update `global/skills.json` when sync should manage it, then regenerate the catalog. After Do-Skills changes, bump the pin and regenerate. Do not hand-edit the marked README block.

## Work Guidance

- Group blurbs come from `skills.sh.json` (`"<Scope>: <Group>"`).
- `gap sync` clones the pin into `.vendor/do-skills` (gitignored). Do not commit that checkout.

## Verification

- After a catalog change, `bash scripts/update-readme.sh` leaves no diff on a second run.

## Child DOX Index

None.
