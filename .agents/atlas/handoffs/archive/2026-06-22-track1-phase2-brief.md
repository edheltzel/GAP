# Worker Brief — Track 1 Phase 2: Reconcile engine (HIGHEST-RISK path)

**Dispatched by:** Themis (PM). **You are a Worker.** This is the most dangerous unit in the plan — §6 "highest-risk code path." Precision over speed. When unsure, STOP and ask Themis; never guess.
**Repo:** `/Users/ed/Developer/atlas-config` **Plan:** `.agents/atlas/plans/2026-06-22-github-projects-skill-overhaul.md` — read **§2** (API reality), **§4.4**, **§6** (the destructive-option footgun — READ TWICE), **§7 Phase 2 gate** before any code.

## Mandatory process
1. **`/ce-worktree`** first — branch from **`origin/master`** (NOT local HEAD; the main checkout is dirty with unrelated WIP, and origin/master already carries the merged Phase 1 templates). Then **`/ce-work`**.
2. **`/KarpathyGuidelines`** — minimum code that solves it; no speculative config; surgical.
3. **Codegraph unavailable** (markdown/config repo) — ground in Read/Grep/gh. Never assume.
4. **bun + TypeScript only** (never npm/npx). Build ON TOP of the existing `skills/github-projects/templates/` (types.ts/archetypes.ts/overlays.ts/resolve.ts/merge.ts — already merged). Reuse the `Archetype`/`Field`/`Label` types; do not redefine them.

## Scope — `skills/github-projects/Tools/Reconcile.ts` (bun CLI)
Converge a live board's **fields, single-select options, and repo labels** to a resolved archetype spec. Three modes: **plan** (default, dry-run diff like `terraform plan`), **`--apply`** (execute), **`--prune --confirm`** (destructive).

### Architecture (per §4.4 + §6)
1. **Snapshot** live state: project fields + single-select options + repo labels via `gh`/`gh api graphql`. Names are the stable key; node IDs resolved fresh each run.
2. **Diff** — 3-bucket model (github-label-sync style): create-if-missing / update-if-drift / extra (delete only under prune). Compute for fields, options, labels.
3. **Plan/apply/prune** with the §6 safety baked in (NON-NEGOTIABLE):
   - **`updateProjectV2Field` is a FULL-LIST REPLACE.** Always read live options first and send the **merged union (existing ∪ desired)** — never a subset (a subset wipes the rest).
   - **Adding** options → auto-safe.
   - **Renaming/removing** options → destructive (churns node IDs, orphans item assignments) → **require `--confirm`**, warn "resets N item assignments," and **re-resolve affected item field values** after the mutation.
   - **Fields/projects never auto-deleted** — flagged only; removed behind `--prune --confirm`.
   - **Lockfile** `.agents/atlas/artifacts/<board-slug>.lock.json` (must be gitignored — verify/add the ignore) caches names→IDs + last spec hash, but **every run re-queries live state** and rebuilds from names.
4. **Label sync across repos** from a manifest using `gh label create --force` (the §4.4 / Phase 2 second deliverable).

### Decompose the risk: pure core + thin IO shell
Put ALL diff/merge/option-union/op-classification logic in **pure functions** (input: snapshot + desired spec → output: typed plan of ops). The `gh`/graphql calls are a thin shell around them. This makes the dangerous logic unit-testable WITHOUT touching live GitHub.

## Verify gate (§7 Phase 2) — show evidence for each
1. **Pure-logic tests** (no API): option-union never drops existing; rename/remove classified destructive + confirm-gated; add classified safe; re-apply of an already-converged spec yields an empty plan (idempotent). Write these as a runnable bun test/smoke and show output.
2. **Live integration** on a **throwaway board you create AND delete**: create a clearly-named disposable project (e.g. `ZZ-reconcile-test-<n>`) on the `edheltzel` account, run plan→apply→re-apply (must be no-op), then an option-rename to prove the confirm-gate + assignment-preservation, then **delete the test project and clean any test labels**. Document every command. If any cleanup step fails, STOP and report — do not leave litter on the account.
3. Typecheck clean (`bunx tsc --noEmit`).

## Reporting back
Open a PR. Report: worktree/branch/PR, files added, the pure-logic test output, the live-integration transcript (incl. proof the test board was deleted), typecheck result, and every design decision you made. Flag anything in §6 you found ambiguous. This unit gets an adversarial RedTeam + reviewer pass before merge — write it to survive that.
