# Reviewer Brief — PR #19 (github-projects Phase 0 finalize + Phase 1 templates)

**Dispatched by:** Themis (PM). **You are a Reviewer.** Your job is to try to find what's wrong before this merges. Be adversarial, not agreeable.
**Repo:** `/Users/ed/Developer/atlas-config` · **PR:** #19 · branch `feat/gh-projects-phase0-1` · base `master`.
**Plan:** `.agents/atlas/plans/2026-06-22-github-projects-skill-overhaul.md` (read §2 API reality, §4.2 label/status canon, §5 archetypes/overlays, §6 highest-risk path, §7 Phase 0/1 gates).
**Worker's own report** is in the worktree session; treat its claims as *unverified* — re-derive, don't trust.

## Mandatory process
1. Run **`/pr-review-toolkit:review-pr`** on PR #19 and **`/ce-code-review`**, calibrated to this repo's standards (bun/TS only, no npm/npx; markdown-zealot; surgical changes per `/KarpathyGuidelines`).
2. **Engage the `ce-testing-reviewer`** agent for test/verify-coverage review of `templates/smoke.ts` — is it a real assertion harness or a rubber stamp? Does it cover all 5 archetypes × overlays, the triage status-prepend, and the unknown-key throw?
3. **`/KarpathyGuidelines`** as the quality bar.
4. **Codegraph unavailable** (no index, markdown/config repo) — ground in Read/Grep/gh, never assume.

## Adversarial focus (the things most likely to be wrong)
1. **De-dup safety.** The PR deletes `skills/GithubProjects/` (TitleCase, 9 files) and keeps `skills/github-projects/` (kebab). Worker claims file-by-file parity. **Verify independently**: diff the deleted TitleCase files (from git history, e.g. `git show <base>:skills/GithubProjects/references/fields.md`) against the kept kebab equivalents. Confirm NO content (recipes, gotchas, examples) was silently dropped in the rename/migration — only the intended colon-canon label changes. Flag any lost prose.
2. **Canon conformance.** Grep the kept skill for any surviving hyphen labels, the old `Backlog/Ready/In Progress/Done` vs the §4.2 lean `Backlog · Ready · In Progress · In Review · Done` status drift, and stale `blocked`-label or GraphQL `addSubIssue` recipes. The worker kept two `Todo` strings intentionally (ProjectSetup.md as GitHub's factual default, fields.md as an API-response example) — judge whether those are genuinely non-prescriptive or are canon leaks.
3. **Template merge correctness (`templates/`).** Read `types.ts`, `archetypes.ts`, `overlays.ts`, `merge.ts`, `resolve.ts`. Verify: (a) `FieldType` is limited to API-supported types only (TEXT/NUMBER/DATE/SINGLE_SELECT/ITERATION) per §2; (b) the resolve/merge is pure & deterministic (no order-dependence, no mutation of shared archetype objects across calls); (c) label sets match the §4.2 canon exactly incl. colors; (d) overlays compose additively without clobbering base fields; (e) single-select option color enum matches GitHub's 8-value set.
4. **The worker's 5 flagged decisions** (Estimate→estimation-overlay vs base; status-field-vs-fields[] modeling; §5 field-vs-label duplications like Channel field + channel:* labels; Labels.md migration-note rewording; the two retained Todo strings). For each: is it a sound resolution of a genuine plan ambiguity, or does it bake in a defect? Give a verdict per decision.

## Output
Write findings to `.agents/atlas/artifacts/2026-06-22-pr19-review.md`: per-finding severity (blocker / should-fix / nit), file:line, evidence, and a clear **MERGE / FIX-FIRST / REJECT** recommendation. Do NOT modify code or merge — you review only. Report the recommendation + blocker count back to Themis.
