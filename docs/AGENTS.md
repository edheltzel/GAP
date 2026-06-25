# docs — DOX Instructions

## Purpose

Durable project documentation for agent operating rules, issue tracking, domain docs, and repo-specific procedures.

## Ownership

- `docs/agents/` owns operational guidance for agents working in atlas-config.
- Root AGENTS.md owns project-wide rules; docs files should explain procedures without contradicting it.

## Local Contracts

- Keep docs operational and current; remove stale instructions instead of adding historical caveats.
- Prefer concise bullets, exact paths, and command names over broad prose.
- When documenting tracked versus runtime files, preserve the repo's safety boundary around secrets and generated state.

## Work Guidance

- Update docs when file layout, install workflow, package ownership, or agent behavior changes.
- If a doc references a generated/live file, state whether it is tracked, stowed, installed, or runtime-only.

## Verification

No docs-specific automated check exists yet. Verify by reading the changed section and, when commands are documented, running the narrow command if it is safe and relevant.

## Child DOX Index

No child AGENTS.md files yet.
