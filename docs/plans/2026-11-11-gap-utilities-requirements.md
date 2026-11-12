# gap-utilities

## Goal

Anyone can make a GitHub repo installable, updatable, checkable, and removable through GAP by adding one manifest to that repo, the way a lazy.nvim plugin works. A utility is a repo that ships harness skills, commands, or its own tooling.

## In scope

- A manifest contract that a utility ships in its own repo. It declares skills, commands, and optional `install`, `update`, `uninstall`, and `doctor` hooks. GAP runs hooks from the pinned checkout.
- `gap add owner/repo`: reads the manifest, writes the spec to `~/.config/gap/gap.toml`, and pins it in a user lock beside that file (the `lazy-lock.json` equivalent).
- `gap update [id]`: moves the user pin to the newest commit on the spec's ref, then installs it.
- `gap sync`: installs exactly what the distro lock and the user lock say. Sync never moves a pin.
- `gap remove id`: runs the `uninstall` hook, deletes the skills and commands GAP copied for that utility, backs up locally edited copies under `~/.gap/backups/` first, and drops the spec and lock entry. When the utility has an `install` hook but no `uninstall` hook, it warns that GAP cannot undo the hook.
- `gap list`: shows each utility with its source (distro or user), pin, and trust state.
- Hook approval follows the [Hooks and trust](../../README.md#hooks-and-trust) contract.
- A trust setting in the user config, per `owner/repo` or per owner (`owner/*`). A trusted utility installs and updates with no hook prompts or warnings, the way a trusted Homebrew tap does.
- Do-Skills, Echo, Recall, and Forge each ship their own manifest. The distro `gap.toml` shrinks to the repos it adds plus its selections (such as `do-astro`'s `load = "project"`). Distro and user utilities share one code path.
- User docs in `README.md` (published at `/docs/`): an "Add a utility" section covering `gap add`, `update`, `remove`, `list`, the user lock, hook approval, and trust. Rewrite Components and Manage skills for the manifest model, and update the Commands list.
- Author docs in `README.md`: a "Publish a utility" section with the full manifest reference (every field, hooks and their exit codes, what `uninstall` must undo) and one minimal example manifest that an author can copy.
- `CONCEPTS.md` (published at `/docs/concepts/`): Utility, Manifest, Hook, User lock, and Trust terms. Update Managed skill to match.
- Contributor docs: `cmd/gap/AGENTS.md` (sync steps, lock, store, remove), root `AGENTS.md` Publish and Edit where they name `gap.toml` components, and `global/AGENTS.md` only if a shared rule changes.
- Website (`site/`): the landing page says GAP can add any utility from GitHub and shows a `gap add owner/repo` example. Wording lives in `site/src/site.js` and `src/components/landing/`. The landing and docs links point at the new README anchors. Any README heading that is renamed or removed gets a `public/_redirects` entry.

## Out of scope

- A registry, index, catalog, or search command. GitHub is the only source.
- A project-level add (`.gap/gap.toml`) for now.
- Installing harnesses.
- New support for repos without a manifest, beyond today's `global/skills.json` path.

## Success

- A fixture repo with a manifest goes through `gap add`, `gap sync`, `gap update`, and `gap remove` end to end. After remove, no file GAP copied for it remains except backups, and its spec and lock entry are gone.
- A second `gap sync` with no changes writes nothing.
- An untrusted utility whose hook changed between pins prompts on a terminal, and without a terminal skips the hook and exits `1`.
- A utility under a trusted `owner/*` never prompts or warns during add, update, sync, or remove.
- `gap remove` on a utility with a locally edited skill leaves a backup of that edit under `~/.gap/backups/`.
- `gap remove` on a utility with `install` and no `uninstall` prints the warning.
- Following only the README "Publish a utility" section, someone new to GAP can write a manifest for a fixture repo that `gap add` accepts on the first try.
- `/docs/` and `/docs/concepts/` on the built site show the new sections and terms. The landing page shows the `gap add` example. From `site/`, `npm run build`, `npm run typecheck`, and `npm run links` pass, and no old anchor breaks.
- The distro `gap.toml` has no hand-written `install`, `doctor`, or `names` for Do-Skills, Echo, Recall, or Forge, and `gap sync` and `gap doctor` give the same result as before the migration.

## Constraints

- `gap.toml`, `gap.lock`, and any new TOML go through the TOML subset parser in `cmd/gap/toml.go`.
- HIL in `CONCEPTS.md`: silence is not agreement. Only an explicit trust setting replaces the prompt.
- The distro `gap.toml` and `gap.lock` keep passing the sync merge gate. User files under `~/.config/gap/` do not.
- GAP writes regular files, never symlinks.
- Deletes use `trash`, not `rm`.
- The site publishes only `README.md` and `CONCEPTS.md` (`repoPages` in `site/src/lib/repo-links.ts`). User and author docs go in those files, not in a new page under `site/` (`site/AGENTS.md`).
- Docs land in the same change as the behavior they describe, so the site never documents a command that does not exist yet.
- The migration needs a manifest change in each of the four component repos. Skill source stays in those repos (root `AGENTS.md` Edit).

## Decisions

- A utility describes itself in its own repo. No registry and no index (user, 2026-11-11).
- `gap add` records to the user config and a user lock, with no project layer yet (assumed lazy.nvim layout, 2026-11-11).
- The manifest declares files plus optional hooks (user, 2026-11-11).
- Hooks need approval by default. A per-repo or per-owner trust setting skips prompts and warnings, Homebrew style (user, 2026-11-11).
- All four distro components move to manifests. There is one code path and no permanent two-tier model (user, 2026-11-11).
- The lifecycle is add, update, sync, remove, and list, following lazy.nvim (assumed, 2026-11-11).
- Remove cleans up fully and backs up edited copies without asking (user, 2026-11-11).
- User docs, author docs, and the website are in scope (user, 2026-11-11).

## Open questions

- Manifest filename and format: a `[package]` section in a repo-root `gap.toml`, or a separate file.
- Where the trust setting lives: a `[trust]` table in `~/.config/gap/gap.toml` or a separate file.
- Whether `global/skills.json` folds into this model.
- How a user overrides a distro utility's spec (ref, selections) once the distro also uses manifests.
- Whether the author guide outgrows the README and needs its own published page, which would add an entry to `repoPages`.

## todo.txt
- Plan GAP utilities from the requirements with do-plan +gap
- Ship a GAP manifest in Do-Skills, Echo, Recall, and Forge for the utilities migration +gap
- Update README, CONCEPTS, and the landing page for GAP utilities +gap
