# Reviewer + RedTeam Brief — PR #20 (Phase 2 reconcile engine, HIGHEST-RISK)

**Dispatched by:** Themis (PM). **Adversarial mandate.** This is the plan's §6 "highest-risk code path." Your job is to break it — find the data-loss bug before it ships. Assume the worker's tests are insufficient until you prove otherwise.
**Repo:** `/Users/ed/Developer/atlas-config` · **PR:** #20 · branch `worktree-feat+gh-projects-phase2-reconcile` · worktree `.claude/worktrees/feat+gh-projects-phase2-reconcile`.
**Plan:** `.agents/atlas/plans/2026-06-22-github-projects-skill-overhaul.md` — **§6 is the spec; read it twice.** Also §2 (API reality), §4.4.
**Code:** `skills/github-projects/Tools/{reconcile-core.ts, reconcile-core.test.ts, Reconcile.ts}`.

## Mandatory process
1. `/pr-review-toolkit:review-pr` + `/ce-code-review` on PR #20, calibrated to this repo (bun/TS only).
2. **`ce-testing-reviewer`** on `reconcile-core.test.ts` — are the 19 tests real adversarial coverage of the destructive paths, or do they dodge the hard cases? What ISN'T tested?
3. `/KarpathyGuidelines`. **Codegraph unavailable** — ground in Read/Grep/git; never assume.
4. Themis already independently confirmed: typecheck exit 0, 19/19 tests pass, live board #21 deleted. Do NOT re-run live-API mutations — the engine has been exercised live once already; your job is static/logic adversarial review, not more live writes.

## Attack surface (where data-loss hides — §6)
1. **The full-list-replace footgun.** `updateProjectV2Field` replaces the WHOLE option list. Prove the code ALWAYS sends `existing ∪ desired`, never a subset, on EVERY mutation path (add, drift-update, prune). Find any code path that could send a partial list and wipe options. Check ordering-independence.
2. **Destructive gating.** Verify rename/remove of options and field/project deletion are genuinely `--confirm`/`--prune`-gated with NO bypass (flag parsing, default-mode leakage, an apply path that reaches a destructive mutation without the gate). Try to construct an invocation that destroys without confirm.
3. **Decision #3 (the key §6 semantic — flag your verdict).** The worker implemented "re-resolve affected item field values" as **report-only** (re-snapshot + accurate reset-count warning + lockfile rewrite), NOT auto-reassignment, arguing auto-reassign would be guessing. Is report-only a faithful, safe reading of §6, or does §6 demand active preservation? Give a clear verdict + reasoning — Themis is escalating this to Ed and wants your independent read.
4. **Partial-failure safety.** If a mutation sequence fails midway (network error after the destructive replace but before reassignment), what state is the board left in? Is it recoverable on re-run (idempotent), or corrupted?
5. **Lockfile hygiene.** Confirm the lockfile path is actually gitignored (the brief required it) and that every run re-queries live state rather than trusting cached IDs. The worker noted a CWD-relative lockdir bug it fixed — verify the fix is correct (anchored to repo root) and has no path-traversal/edge issue.
6. **Idempotency.** Static-trace: does a converged spec truly yield an empty plan, or are there fields (e.g. option order, colors, descriptions) that would show spurious drift and trigger needless mutations?

## Output
Write findings to `.agents/atlas/artifacts/2026-06-22-pr20-review.md`: per-finding severity (blocker / should-fix / nit), file:line, evidence, and a **MERGE / FIX-FIRST / REJECT** verdict + your ruling on decision #3. Do NOT modify code or merge. Report verdict + blocker count + decision-#3 ruling back to Themis.
