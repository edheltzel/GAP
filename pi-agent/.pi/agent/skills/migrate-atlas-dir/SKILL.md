---
name: migrate-atlas-dir
description: Migrate a project's legacy .atlas-plans/ layout to the new .atlas/ namespace (plans + handoffs). Dry-run by default. Invoke directly with /skill:migrate-atlas-dir [execute].
allowed-tools: Bash(git:*), Bash(mkdir:*), Bash(mv:*), Bash(rmdir:*), Bash(ls:*), Bash(find:*), Bash(eza:*), Bash(fd:*)
---

# Migrate Atlas Dir

Migrate a project's legacy `.atlas-plans/` directory to the new `.atlas/` namespace:

| Old path                                 | New path                         |
|------------------------------------------|----------------------------------|
| `.atlas-plans/*.md` (top-level)          | `.atlas/plans/*.md`              |
| `.atlas-plans/archive/*`                 | `.atlas/plans/archive/*`         |
| `.atlas-plans/handoffs/*`                | `.atlas/handoffs/*`              |
| `.atlas-plans/handoffs/archive/*`        | `.atlas/handoffs/archive/*`      |

Runs in the current working directory. Dry-run by default — only mutates the filesystem when user args contain `execute`, `yes`, or `force`.

## Behavior

1. **Sanity checks.**
   - If `.atlas-plans/` does not exist in cwd, report "No `.atlas-plans/` directory found — nothing to migrate" and stop.
   - If `.atlas/` already exists, report its current contents. Migration will merge into it — never clobber existing files. Any collision (same relative path present in both old and new) is a hard stop; list the collisions and exit without moving anything.

2. **Decide mode.** If user args are empty or do not contain `execute`/`yes`/`force`, treat as a dry run.

3. **Detect git context.**
   ```bash
   git rev-parse --is-inside-work-tree >/dev/null 2>&1 && IN_GIT=true || IN_GIT=false
   ```
   Use `git mv` when in a repo to preserve history; plain `mv` otherwise.

4. **Plan the moves** (order matters):
   a. `.atlas-plans/handoffs/archive/*` → `.atlas/handoffs/archive/`
   b. `.atlas-plans/handoffs/*` (remaining files) → `.atlas/handoffs/`
   c. `.atlas-plans/archive/*` → `.atlas/plans/archive/`
   d. `.atlas-plans/*.md` and any other top-level non-directory files → `.atlas/plans/`

5. **Dry-run output:** print a source→destination table and a summary line.

6. **Execute (only if confirmed):**
   - `mkdir -p` destination dirs.
   - `git mv` or `mv` each file.
   - `rmdir` the now-empty `.atlas-plans/` subtree; leave behind any non-empty dir and report it.
   - Do not commit. Leave the rename staged.

7. **Report:** total files moved, any leftover non-empty dirs, and either the execute command (dry-run) or a reminder to review `git status` and commit (executed).

## Notes

- Destinations are never overwritten; a conflict aborts the migration entirely.
- Running this skill twice is safe: the second run finds no `.atlas-plans/`.
