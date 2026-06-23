# Explorer Brief — Track 2: Phase 0.5 org runbook verification (READ-ONLY)

**Dispatched by:** Themis (PM). **You are an Explorer.** You produce a report. You make **NO** irreversible or outward-facing changes.
**Repo:** `/Users/ed/Developer/atlas-config` **Plan:** `.agents/atlas/plans/2026-06-22-github-projects-skill-overhaul.md` — read §8 (the Phase 0.5 RUNBOOK) in full plus §10 risks.

## Mandatory process
1. **`/KarpathyGuidelines`** — simplicity, no speculation, verify every claim.
2. **Codegraph note:** no codegraph index here (markdown/config repo). Ground in `Read`/`Grep`/`gh`. Never assume.
3. **HARD CONSTRAINT — read-only / no side effects.** You may run only **non-mutating** `gh`/`gh api` queries (GET, introspection, `--jq` reads). You must NOT: create accounts, transfer repos, create issue types, create/install GitHub Apps, create labels, mint tokens, or push anything. Those are Ed's hands or gated behind his explicit go-ahead. If a verification *requires* a mutation, do NOT do it — note it as "requires Ed" instead.

## Your deliverable
A single report file: `.agents/atlas/artifacts/2026-06-22-phase0.5-org-readiness.md`. It must let Ed (and a later automation agent) execute Phase 0.5 with zero guesswork. Cover:

1. **Org reachability.** Confirm `Detego-Research-Group` exists and Ed's `gh` auth can see it: `gh api orgs/Detego-Research-Group --jq '.login,.plan.name'` (note plan tier). Report exact output. State whether current `gh auth status` identity is `edheltzel` and has org access.
2. **Issue-types availability (the key open risk, §8.5 + §10).** Determine whether native issue types are actually usable on this **free** org. Use READ-ONLY introspection only: query existing issue types via `gh api graphql -H "GraphQL-Features: issue_types" -f query='query{organization(login:"Detego-Research-Group"){issueTypes(first:20){nodes{name isEnabled}}}}'`. Report whether the field resolves, what defaults exist, and whether enabling/creating would be gated. **Do NOT create any issue type.** Give a clear verdict: native issue types available on free org? yes / no / unconfirmed — with the evidence.
3. **Transfer-candidate inventory.** Enumerate Ed's repos (`gh repo list edheltzel --limit 200 --json name,visibility,isArchived,isFork`) and map them against the §8.1 tiers (Core / Your call / Test-first `drg-temp` / Keep / Never). Flag which Core repos still exist, whether `drg-temp` exists for the test transfer, and any Pages/DNS risk. Output a clean table.
4. **Exact, ordered runbook** — copy-pasteable, marking each step's owner per §8.8: 🧑 Ed (atlas account creation, repo transfers, App private key + secrets), 🌐 Atlas-via-Interceptor (browser account/App registration when triggered), 🤖 Atlas-via-gh/API (issue types, label sync, board scaffold). Make the 🧑 Ed steps a crisp checklist he can act on directly.
5. **`atlas-config` transfer impact.** IF `atlas-config` were transferred (it's in the "Your call" tier), grep the repo for `edheltzel/atlas-config` references that would need updating (AGENTS.md, `docs/agents/issue-tracker.md`, skill `Labels.md` examples, memory). List exact file:line hits so the rename is mechanical. (Do not change them — just inventory.)
6. **Blocking dependencies.** Note that label sync (§8.6) depends on the Phase 2 reconcile engine/manifest (not built yet), and that the Actions `GITHUB_TOKEN` cannot write org Projects v2 (why atlas-bot exists). State what truly blocks vs. what Ed can do today.

## Reporting back
When the report file is written, summarize to Themis: the issue-types verdict (the big unknown), the Core-repo transfer list, and the precise set of 🧑 Ed-only steps that gate the rest. If any read-only query fails on auth/permissions, report that as a finding (it tells us what Ed must grant) — do not attempt to escalate or mutate.
