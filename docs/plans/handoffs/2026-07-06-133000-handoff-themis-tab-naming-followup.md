# Handoff — Themis tab naming (shipped) → PAI v6 install tracking (next)

**Date:** 2026-07-06
**Repo:** `edheltzel/atlas-config`

## Current state

The Themis `GH#<issue>-<role>` herdr tab-naming work is **fully shipped and merged** — no follow-up needed on it. Details are in git/GitHub, not duplicated here:

- PR: https://github.com/edheltzel/atlas-config/pull/35 (merged as `667553f`)
- Prior handoff with full context: `docs/plans/handoffs/2026-07-06-132313-handoff-themis-tab-naming.md`

A tracking issue for the next piece of work was filed this session:

- Issue: https://github.com/edheltzel/atlas-config/issues/36 — "Install PAI/LifeOS v6" (placeholder scope; no v6 release exists yet)

## Notable repo-state fact for the next session

`atlas-config/AGENTS.md` moved project-local Atlas artifacts from `.agents/atlas/` to `docs/` (`plans/`, `handoffs/`, `artifacts/`, `worktrees/`), gitignored as of 2026-06-30. The global user-level rule at `~/.claude/PAI/../Rules/handoff.md` (or wherever the atlas-directory/handoff rules live) still says handoffs belong under `.agents/atlas/handoffs/` — that rule is now **stale against this repo** and should be corrected next time it's edited. Until then, keep honoring the project's `docs/plans/handoffs/` convention here, as instructed by Ed.

## Next steps

1. When Ed brings a v6 release/migration guide, open issue #36 and turn it into a real plan — no upgrade work should start before that guide exists.
2. Consider filing (or asking Ed about) a small doc-fix issue/PR to reconcile the stale `.agents/atlas/` global rule with the repo's actual `docs/` convention, so future sessions don't need to re-derive this from AGENTS.md diffs.

## Suggested skills for the next session

- `pm-tools` / `Continue` — if picking the next board item rather than working from a named issue directly.
- `Themis` (this repo's persona) — if the next session wants delegated coordination rather than direct implementation, though for v6 there's nothing to delegate until a real migration guide exists.
- `ISA` (Scaffold workflow) — once issue #36 has real scope, scaffold an ISA rather than working ad hoc, since a version migration is exactly the kind of multi-file, cross-cutting work the Algorithm/ISA system is for.

## Redaction note

No credentials, tokens, or PII were referenced with actual values in this session — only environment-variable *names* (e.g. `ATLAS_KEY_PAT`) and public GitHub identities. Nothing to redact.
