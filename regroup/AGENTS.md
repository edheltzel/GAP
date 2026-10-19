# AGENTS.md

## Purpose

Dated status snapshots. The chat reply never replaces the file.

## Ownership

- One project: `regroup/YYYY-MM-DD-regroup-<project-slug>.md`.
- All projects: `regroup/YYYY-MM-DD-regroup-status-check.md`.
- The date is the request date. Do not substitute a later "today" once that date is known.

## Local Contracts

- Do not overwrite or delete an older regroup file. If today's name exists, keep it and add a numeric suffix (`-2`, then `-3`).
- Read the previous file of the same mode, and for one project the same project, when one exists.
- Keep each todo.txt task's raw line so the next file can tell what changed.
- Regroup reads `todo.txt` and `done.txt`. It does not edit them, complete a task, or archive one.
- The only file a regroup run writes is the new snapshot.

## Work Guidance

- Follow the `do-regroup` skill for sources, status labels, and section order.

## Verification

## Child DOX Index

None.
