# Checkout-Review Brief — PR #20 (fresh hands-on verification + post to PR)

**Dispatched by:** Themis. PR #20 has already had a static review + delta-review (both at `.agents/atlas/artifacts/2026-06-22-pr20-review.md`, verdict CLEAR-TO-MERGE). Your job is the **fresh hands-on confirmation Ed asked for**: actually check the PR out, build and run it in a clean checkout, and **post the consolidated verdict onto the PR** so it's actionable for merge. This is a confirmation pass — do not re-derive the whole review from scratch; verify it holds when checked out fresh.

## Steps
1. `gh pr checkout 20` (into a worktree if cleaner — or check out the branch). Confirm head is `0ce4897`.
2. In `skills/github-projects`:
   - `bunx tsc --noEmit -p Tools/tsconfig.json` and `-p templates/tsconfig.json` → expect clean.
   - `bun test Tools/` → expect 42 pass / 0 fail.
   - Spot-read the F1 blocker fix (id-preserving `--rename` in `reconcile-core.ts` + `Reconcile.ts`) and confirm the rename test asserts assignment preservation (option id carried, no destructive removal). Confirm `captureAssignments` never returns a confident false `0`.
3. `/KarpathyGuidelines`. **Codegraph unavailable** — Read/Grep/git. Do NOT run live-API mutations (no real GitHub board writes).
4. **Post the verdict to the PR**: `gh pr review 20 --approve --body "<concise summary>"` if it holds (or `--comment` with findings if not). The body should summarize: blocker + 4 should-fixes resolved, 42/42 tests, §6 core safe, decision #3 settled (report-only + explicit id-preserving rename), and note F6 + nits as tracked follow-ups. Keep it tight and reference the artifact.

## Output
Report to Themis: head SHA confirmed, typecheck + test results from your fresh checkout, whether the verdict holds (CLEAR-TO-MERGE or not), and the URL/confirmation that the PR review was posted. Do NOT merge — that's Ed's call.
