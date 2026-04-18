---
description: "Migrate a project's .atlas-plans/ layout to the new .atlas/ namespace (plans + handoffs)"
---

# Migrate Atlas Dir

Migrate a project's legacy `.atlas-plans/` directory to the new `.atlas/` namespace:

| Old path                                 | New path                         |
|------------------------------------------|----------------------------------|
| `.atlas-plans/*.md` (top-level)          | `.atlas/plans/*.md`              |
| `.atlas-plans/archive/*`                 | `.atlas/plans/archive/*`         |
| `.atlas-plans/handoffs/*`                | `.atlas/handoffs/*`              |
| `.atlas-plans/handoffs/archive/*`        | `.atlas/handoffs/archive/*`      |

Runs in the current working directory. Dry-run by default — only mutates the filesystem when invoked with `--execute` (or `$ARGUMENTS` contains `execute`/`yes`/`force`).

**Arguments:** $ARGUMENTS

## Behavior

1. **Sanity checks.**
   - If `.atlas-plans/` does not exist in cwd, report "No `.atlas-plans/` directory found — nothing to migrate" and stop.
   - If `.atlas/` already exists, report its current contents. Migration will merge into it — never clobber existing files. Any collision (same relative path present in both old and new) is a hard stop; list the collisions and exit without moving anything.

2. **Decide mode.** If `$ARGUMENTS` is empty or does not contain `execute`, `yes`, or `force`, treat this as a dry run: print the planned moves but make no changes. Otherwise, execute.

3. **Detect git context.** Check whether cwd is inside a git repo:
   ```bash
   git rev-parse --is-inside-work-tree >/dev/null 2>&1 && IN_GIT=true || IN_GIT=false
   ```
   When `IN_GIT=true`, use `git mv` so history follows the rename. When false, use plain `mv`.

4. **Plan the moves.** Walk `.atlas-plans/` and build a move list in this exact order (the order matters — move children before their parent directory so intermediate `rmdir`s succeed):

   a. `.atlas-plans/handoffs/archive/*` → `.atlas/handoffs/archive/`
   b. `.atlas-plans/handoffs/*` (remaining files) → `.atlas/handoffs/`
   c. `.atlas-plans/archive/*` → `.atlas/plans/archive/`
   d. `.atlas-plans/*.md` and any other top-level non-directory files → `.atlas/plans/`

   For each mapped file, compute source and destination paths. If destination exists, abort and list the conflict (see step 1).

5. **Dry-run output.** Regardless of mode, print a table:
   ```
   .atlas-plans/2026-04-02-foo.md           →  .atlas/plans/2026-04-02-foo.md
   .atlas-plans/archive/old-plan.md         →  .atlas/plans/archive/old-plan.md
   .atlas-plans/handoffs/2026-04-18-x.md    →  .atlas/handoffs/2026-04-18-x.md
   ```
   Follow with a summary line: `N files would move` (dry-run) or `N files moved` (execute).

6. **Execute (only if confirmed).**
   - Create destination directories with `mkdir -p` (plain, not `git mkdir` — git only tracks files).
   - Move each file with `git mv` (if in git) or `mv` (if not).
   - After all files are moved, `rmdir` `.atlas-plans/handoffs/archive/`, `.atlas-plans/handoffs/`, `.atlas-plans/archive/`, and finally `.atlas-plans/` — each only if empty. If a directory isn't empty after the move, leave it (something unexpected is in there) and report it.
   - Do NOT commit. Leave the rename staged so Ed can review `git status` and commit as part of his normal flow.

7. **Report.** Print:
   - Total files moved (or would-move in dry-run)
   - Any directories that could not be removed (non-empty)
   - If dry-run: the exact command to execute — `/MigrateAtlasDir execute`
   - If executed: a reminder to run `git status` and commit the rename

## Notes

- The migration is strictly additive for destinations: existing files in `.atlas/` are never overwritten.
- `.atlas-plans/` and the new `.atlas/` layout are not tracked via `.gitignore` by default. If a project has custom gitignore entries for the old path, Ed should update them manually.
- Running this command twice is safe: the second run finds no `.atlas-plans/` and exits cleanly.

## Example

```
$ /MigrateAtlasDir

DRY RUN — no changes will be made.

Planned moves:
  .atlas-plans/2026-04-05-foo.md          →  .atlas/plans/2026-04-05-foo.md
  .atlas-plans/archive/old-plan.md        →  .atlas/plans/archive/old-plan.md
  .atlas-plans/handoffs/2026-04-18-x.md   →  .atlas/handoffs/2026-04-18-x.md

3 files would move.
Run /MigrateAtlasDir execute to perform the migration.
```

$@
