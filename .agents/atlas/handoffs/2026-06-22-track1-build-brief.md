# Worker Brief — Track 1: Phase 0 finalize + Phase 1 template layer

**Dispatched by:** Themis (PM). **You are a Worker.** Report results back; do not over-reach scope.
**Repo:** `/Users/ed/Developer/atlas-config` **Plan:** `.agents/atlas/plans/2026-06-22-github-projects-skill-overhaul.md` (read §2, §4.2, §5, §6, §7 Phase 0 + Phase 1 before touching anything).

## Mandatory process (non-negotiable)
1. **Start with `/ce-worktree`** to create an isolated worktree, then **`/ce-work`** to execute. Do NOT work on the main checkout.
2. **`/KarpathyGuidelines`** govern every edit: think before coding, simplicity first, surgical changes, goal-driven (define verify criteria, loop until met).
3. **Codegraph note:** this repo has NO codegraph index (it's a markdown/config repo) — codegraph MCP/CLI is unavailable this session. Ground every claim in `Read`/`Grep`/`gh` instead. Never assume file structure — verify by reading.
4. The canonical skill directory is **`skills/github-projects/`** (kebab-case). The plan prose sometimes writes `GithubProjects` (TitleCase) — ignore that; kebab is canon. Do NOT create a TitleCase dir.

## Scope — exactly two units, in order. Stop after Phase 1. Do NOT start Phase 2 (reconcile engine).

### Unit A — Finalize Phase 0 de-drift (Task #1)
Current state (verified by Themis): commit `acae194` already migrated `skills/github-projects/references/Labels.md` to the colon canon and updated `docs/agents/triage-labels.md`. Remaining loose ends:

1. **Dangling tracked deletions.** Git still tracks the OLD `skills/GithubProjects/` (TitleCase) variant; its files show as deleted (`D`) in the working tree. **Before finalizing the deletion, verify content parity**: confirm the kept `skills/github-projects/` contains every reference the TitleCase copy had — the TitleCase copy had `SKILL.md` + references `Continue, Execute, Hierarchy, Labels, ProjectSetup, Status, fields.md, items.md`. If `fields.md` or `items.md` (or any reference) is MISSING from the kept kebab dir, recover it from git (`git show acae194~1:skills/GithubProjects/references/fields.md` etc. — find the right ref) into the kebab dir FIRST. Only once parity is confirmed, stage and commit the TitleCase deletion.
2. **Document the `gh >= 2.94` prerequisite** in `skills/github-projects/SKILL.md` (a prerequisites note). Installed version is 2.95.0 — fine — but the skill must state the floor since hierarchy/deps recipes depend on it.
3. **Native hierarchy/deps recipes.** In the kept skill (likely `references/Hierarchy.md`), replace any GraphQL `addSubIssue` workaround with native `gh issue create --parent <#>` / `gh issue edit <#> --add-sub-issue <#>`, and replace any `blocked` *label* usage with native `gh issue edit --blocked-by / --blocking`. Per plan §2.
4. **Reconcile remaining contradictions** to the §4.2 label canon (colon namespaces everywhere: `type:* needs:* risk:* agent:* wontfix`; lean Status `Backlog · Ready · In Progress · In Review · Done` per §4.2/§5). Grep the whole skill for hyphen labels (`needs-triage`, `needs-info`, `blocked`) and old status vocab; fix any stragglers.

**Verify gate (Unit A):** one skill dir only; `grep -rn 'needs-triage\|needs-info' skills/github-projects` returns nothing (all colon); SKILL.md states gh>=2.94; Hierarchy recipes are native; `git status` clean after commit. Report the exact grep outputs as proof.

### Unit B — Phase 1 template layer (Task #2)
Build the config-as-code template primitive per plan §3 (component 1), §4.2 (defaults), §5 (archetypes + overlays). Location: **`skills/github-projects/templates/`**.

1. `templates/types.ts` — typed interfaces: `Archetype`, `Field`, `View`, `Label`, plus a `FieldType` union **limited to API-supported types** (`TEXT | NUMBER | DATE | SINGLE_SELECT | ITERATION`). Single-select options carry `name`, `color` (the 8-value GraphQL color enum), optional `description`. Keep it minimal — only what §5 archetypes actually need. No speculative fields.
2. The **5 base archetypes** from §5 table (Solo Software [default], Content Pipeline, Research Tracker, Life-OS Tasks, Open-Source Repo) — each with its Status vocabulary, base fields (`Status`+`Priority` shared), extra fields, and distinct labels.
3. The **overlay modules** from §5 (`iterations`, `estimation`, `roadmap`, `triage`, `feedback-agent`) as additive composable functions that merge into an archetype.
4. A small resolve function: `(archetype, overlays[]) => mergedArchetype` — pure, deterministic. This is what the later interview/reconcile will call.
5. Use **bun + TypeScript** (repo standard — never npm/npx). Labels must match the §4.2 canon exactly (colors preserved from Recall where stated).

**Verify gate (Unit B):** `bun` typechecks the templates dir clean (e.g. `bunx tsc --noEmit` or `bun build`/whatever the repo uses — check for existing tsconfig first; if none, `bunx tsc --noEmit templates/*.ts` with a minimal config). Resolving `solo-software` + `["estimation","roadmap"]` yields the expected merged spec (Status = Backlog·Ready·In Progress·In Review·Done; fields include Phase, Estimate#, Area, Start/Target dates; labels include `type:*`). Write a tiny resolve smoke-check and show its output. Report typecheck output + a sample resolved spec as proof.

## Reporting back
When done (or blocked), summarize: branch/worktree name, commits made, files added/changed, the exact verify-gate evidence (grep + typecheck outputs), and anything you had to decide. Open a PR for the worktree per `/ce-work` norms. If you hit ANY ambiguity about scope or canon, STOP and write the question — Themis will answer. Do not guess.
