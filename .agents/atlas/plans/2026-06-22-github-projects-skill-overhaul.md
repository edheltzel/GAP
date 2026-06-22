# GitHub Projects Skill Overhaul — Consistent, Interview-Driven, Repeatable Boards

**Date:** 2026-06-22 (rev 3 — answers locked, Phase 0 approved & started)
**Status:** APPROVED — Phase 0 in progress
**Owner:** Ed — org **`Detego-Research-Group`** already created (free plan); repos to be transferred from personal `edheltzel`
**Scope:** Redesign the `github-projects` skill into a consistent, opinionated, interview-driven board scaffolding system with reusable templates, a `:`-namespaced label taxonomy, native issue types (org), an `atlas` agent identity, and a $0 GitHub feedback agent.

## Decisions locked (rev 2)

| # | Decision | Resolution |
|---|----------|-----------|
| D1 | Residual UI gap | **Interceptor (chosen).** `gh project copy` covers ~95% (fields + **all views** + most workflows). Only irreducible gap = re-enabling **auto-add workflow** (one toggle) → a tiny **Interceptor** script (rides the logged-in Chrome session; no auth scripting). |
| D2 | Org | **DONE — org `Detego-Research-Group` exists** (free, Organization). Unlocks native issue types, org default labels, project templates. Remaining: transfer repos into it. |
| D3 | Agent identity | Create a dedicated **`atlas` machine account** (ToS-allowed, $0 on free org) as the human-facing persona + an **org-owned GitHub App** for heavy board automation. |
| D4 | Labels | **Canon = the live `edheltzel/Recall` taxonomy, fully colon-migrated** (`type:*`, `needs:*`, `risk:*`, `agent:*`, `wontfix`). `risk:` stays a **label** (Ed uses it). Migrate the two stragglers `needs-triage`→`needs:triage`, `needs-info`→`needs:info`. Update `docs/agents/triage-labels.md`. See §4.2. |
| D5 | Phase model | Recall Phase 0 (#15) is the liked shape: **lean** — default Status + sub-issues. Default archetype biases lean. Consolidate the "one-project-per-phase" pattern toward **one board + Phase (milestone/field)** — see §5. |
| D6 | Feedback depth | **Minimal first** — Issue Forms + built-in auto-add. GitHub App + AI triage are a later overlay (Phase 5). |

---

## 1. Why this exists (the problem)

The current skill is a solid `gh` command cheat-sheet plus thin PM recipes, but it does **not** deliver consistency. Concrete evidence found in the live system:

1. **The skill is forked and has drifted.** Two copies exist — `~/.agents/skills/github-projects` and `~/.agents/skills/githubprojects` — and they already disagree:
   - Status options: `Todo/In Progress/In Review/Done` **vs** `Backlog/Ready/In Progress/Done`
   - Triage labels: `needs:triage` (colon) **vs** `needs-triage` (hyphen) — only the hyphen form matches `docs/agents/triage-labels.md`
   - View filters differ accordingly
   - Frontmatter `name:` differs (`github-projects` vs `GithubProjects`)
   This is the inconsistency problem *in miniature*: two sources of truth, silently diverging.

2. **No consistency mechanism.** Defaults live in prose tables that a human copy-pastes. Nothing enforces that board N looks like board N+1.

3. **Phases are modeled inconsistently in the wild.** The live `edheltzel` account has a **"Recall: Phase 0/1/2/3/4" series — one project per phase** (projects #15–#19), which contradicts the skill's own guidance that *Phase is a board field on one board*. Both patterns exist; neither is enforced.

4. **No interview.** Setup is a manual command sequence; the human decides everything ad hoc.

5. **No scaffolding automation.** Field/option ID capture is manual, error-prone, and goes stale (the skill even documents this as a "gotcha").

6. **No feedback agent.** Feature #3 (a GitHub-side agent for human feedback) does not exist at all.

7. **No reusable templates.** "Predictable, repeatable, reusable" is the explicit goal and there is no template primitive today.

**Goal:** every board Ed (or an agent) creates is consistent by construction — same status vocabulary, same label taxonomy, same field set, same views per archetype — chosen via a short interview, applied idempotently, and reusable.

---

## 2. Verified API reality (the constraints that shape the whole design)

Researched and verified against the **live GraphQL schema, `gh` v2.95.0 `--help`, and docs.github.com (June 2026)**. These are load-bearing facts:

| Capability | Verdict | Mechanism | Design impact |
|---|---|---|---|
| Create project | ✅ Full | `gh project create` / `createProjectV2` | Scriptable |
| Create fields (TEXT/NUMBER/DATE/SINGLE_SELECT) | ✅ CLI | `gh project field-create` | Scriptable |
| Create ITERATION field | ✅ GraphQL only | `createProjectV2Field` + `iterationConfiguration` | CLI can't; use `gh api graphql` |
| **Edit single-select options** (add/rename/reorder) | ✅ GraphQL, **full-replace** | `updateProjectV2Field { singleSelectOptions: [...] }` | **Old skill's "immutable" claim is WRONG.** But it's a whole-list replace — **the #1 footgun** (see §6) |
| Option colors + descriptions | ✅ GraphQL only | option input has `color` (8-enum) + `description` | CLI sets names only |
| Customize Status options | ✅ GraphQL | same as any single-select; default = `Todo/In Progress/Done` | Status is not special |
| **Create/edit VIEWS** (layout/filter/group/sort) | ❌ **No API at all** | none — no `createProjectV2View` mutation exists | **Views are web-UI-only.** Hard constraint. |
| **Create/edit built-in workflows** (auto-add, close→Done, auto-archive) | ❌ Delete-only | only `deleteProjectV2Workflow` | **Automations are web-UI-only.** |
| **Copy a project** | ✅ Full | `gh project copy` / `copyProjectV2` | Copies **views + fields + workflows (except auto-add)**. This is the **only** programmatic path to views/workflows. |
| Mark project as template | ✅ but **ORG-ONLY** | `gh project mark-template` | Not usable on Ed's personal account |
| Sub-issues | ✅ Native CLI (gh ≥2.94) | `gh issue create --parent`, `gh issue edit --add-sub-issue` | Drop the GraphQL `addSubIssue` workaround. 8 levels, 100/parent. |
| Native issue **types** (Epic/Feature/Task) | ❌ **ORG-ONLY** | `createIssueType` / `gh issue create --type` | **Not available on personal repos** → must use a `type:` label instead |
| Issue dependencies (blocked-by/blocking) | ✅ Native | `gh issue edit --blocked-by/--blocking` | Replace any `blocked` label |
| Cross-repo label sync | ⚠️ Script it | `gh label create --force`, `gh label clone` | No native personal-repo sync; manifest + loop |

**Two architectural consequences fall directly out of this table:**

- **A. Fields, options, labels, items, hierarchy = fully reconcilable via API.** Build a converge engine for these.
- **B. Views + workflows = NOT reconcilable.** They can only be (i) created by hand in the UI once, then (ii) reproduced by `gh project copy` from a "golden" source project. So **templates must be delivered as copyable golden projects, not just as code.**

This hybrid — *code-reconcile for the data model, copy-from-golden for views/workflows* — is the spine of the design.

---

## 3. Target architecture

```
┌─────────────────────────────────────────────────────────────────┐
│  TEMPLATES (config-as-code, source of truth)                      │
│  skills/GithubProjects/templates/*.ts  (composable archetypes)    │
│   base archetype + overlay modules (iterations, estimation, …)    │
└───────────────┬───────────────────────────────────┬──────────────┘
                │                                     │
        ┌───────▼────────┐                   ┌────────▼─────────┐
        │ INTERVIEW       │  AskUserQuestion  │ GOLDEN PROJECTS  │
        │ 6–8 questions → │  → resolved config│ one hand-built   │
        │ {archetype,     │                   │ board per        │
        │  overlays, ...} │                   │ archetype (views │
        └───────┬─────────┘                   │ + workflows set  │
                │                             │ once in UI)      │
        ┌───────▼──────────────────────┐     └────────┬─────────┘
        │ RECONCILE ENGINE (bun CLI)    │              │
        │ Tools/Reconcile.ts            │   gh project copy (views,
        │  plan │ --apply │ --prune     │◄──workflows) then reconcile
        │  • fields / options / labels  │   fields+options+labels on top
        │  • names=key, IDs resolved    │
        │    fresh, cached in lockfile  │
        └───────┬──────────────────────┘
                │
        ┌───────▼──────────────────────┐     ┌──────────────────────┐
        │ HIERARCHY (native gh ≥2.94)   │     │ FEEDBACK AGENT ($0)   │
        │  type: label (personal) /     │     │ .github/ scaffold:    │
        │  issue type (org-detect)      │     │ Issue Forms + GitHub  │
        │  sub-issues, dependencies     │     │ App + router workflow │
        └───────────────────────────────┘     │ + slash cmds + Models │
                                               └──────────────────────┘
```

### Components

1. **Templates (config-as-code).** Composable TypeScript modules — a typed `Archetype` interface, one base archetype per work-kind, and additive overlay modules. Source of truth for status vocabulary, fields, labels, and the *intended* views (the latter emitted as a checklist / encoded in the golden project). TypeScript over YAML because the repo standard is bun/TS, it gives compile-time validation of field types/option shapes, and overlays compose as functions.

2. **Interview.** `AskUserQuestion`-driven, 6–8 questions, maps answers → `{ archetype, overlays[], owner, visibility }`. Q1 picks the base archetype; the rest toggle overlay modules.

3. **Golden projects + copy.** For each archetype, one hand-configured board (views + automations set once in the UI) lives on the account as the canonical source. New boards are born via `gh project copy --source <golden>` to inherit views/workflows, then the reconcile engine layers the exact field/option/label set on top. (Falls back to "manual view checklist" if no golden exists yet — see §6 decision.)

4. **Reconcile engine (`Tools/Reconcile.ts`, bun).** Converges a live board's *fields, single-select options, and repo labels* to the resolved archetype. Three op-classes: create-if-missing (auto), update-if-drift (auto for labels, **confirm-gated** for option removals/renames), never-destroy-without-`--prune`. Default mode is **plan** (dry-run diff, like `terraform plan`); `--apply` executes; `--prune --confirm` allows destructive ops. Names are the stable key; node IDs are resolved fresh every run and cached in a gitignored lockfile.

5. **Hierarchy.** Native `gh` ≥2.94: `--parent` / `--add-sub-issue` / `--blocked-by`. Detect `owner.type`: **org** → native issue types for kind; **personal** (Ed's case) → `type:epic|feature|task` label. Everything else identical.

6. **Agent identity (org).** Two identities, clean split:
   - **`atlas` machine account** — the human-facing persona. ToS explicitly permits one machine account per human alongside `edheltzel`; $0 on a free org (members/collaborators don't consume seats). This is who humans @mention, assign, and give feedback to. Writes that should *read as "Atlas the teammate"* are authored with the machine account's fine-grained PAT.
   - **Org-owned GitHub App** — the heavy automation. Fine-grained, short-lived (~1h) installation tokens minted in Actions; **can write to org Projects v2 where `GITHUB_TOKEN` cannot**; doesn't consume a seat. This is who moves cards, applies labels, and runs triage at scale.

7. **Feedback agent ($0).** A `.github/` scaffold the skill drops into any repo: Issue Forms for structured capture, the GitHub App for cross-repo + Projects-v2 auth, a router workflow that adds feedback to the board + acks the reporter (as `atlas`), slash-command handlers via `actions/github-script` on the free `GITHUB_TOKEN`, and optional AI triage via GitHub Models' free tier (~50 high-model req/day).

---

## 4. The four requested features → concrete designs

### Feature 1 — Scaffold projects consistently from an interview

**Interview (AskUserQuestion), Q1 selects archetype, Q2–Q8 toggle overlays:**

| # | Question | Drives |
|---|----------|--------|
| 1 | What kind of work? (software / writing / research / personal / open-source) | **Base archetype** |
| 2 | Public or private? | visibility; contributor-facing labels |
| 3 | Solo, or external contributors? | adds `Triage` status + `help wanted`/`good first issue` |
| 4 | Continuous flow, or time-boxed (sprints/milestones)? | `iterations` overlay + Milestone usage |
| 5 | Need effort estimation (points/sizes)? | `estimation` overlay (Estimate number / Size select) |
| 6 | Track phases or dated roadmap? | `roadmap` overlay (Start/Target date fields + Roadmap view) |
| 7 | Need external feedback intake? | installs the **Feedback agent** scaffold (Feature 3) |
| 8 | Priority scheme? (P0–P3 / High-Med-Low / none) | `Priority` option set |

Output: `{ archetype: "solo-software", overlays: ["estimation","roadmap"], visibility: "private" }` → reconcile resolves it to a merged `Archetype` and converges the board.

### Feature 2 — Opinionated defaults (views, status, labels, fields)

**Canonical defaults (single source of truth, ends the drift):**

- **Status vocabulary (default archetype):** `Backlog · Ready · In Progress · In Review · Done`. One canonical set per archetype; no more two-copies disagreement.
- **Board fields:** `Status` (built-in), `Priority` (P0–P3), `Phase` (single-select or Milestone, qualitative/dated stages) — Phase is a **field/milestone on one board**, *not* a separate project per phase (fixes the "Recall: Phase N" anti-pattern) and *not* a label.
- **Canonical label taxonomy = the live `edheltzel/Recall` set, fully colon-migrated.** This is the start set synced across every repo (via the Phase-2 label-sync target). Colors preserved from Recall.

  | Namespace | Labels | Role |
  |---|---|---|
  | `type:` | `type:bug` `type:feature` `type:docs` `type:test` `type:refactor` `type:chore` | work kind (color `#d4c5f9`) |
  | `needs:` | `needs:triage` `needs:info` `needs:human` | what the issue is waiting on / who acts next |
  | `risk:` | `risk:low` `risk:medium` `risk:high` | ambiguity/judgement level — **a label in Ed's usage**, not a board field |
  | `agent:` | `agent:ready` `agent:blocked` `agent:complete` | AFK-agent lifecycle signal (color `#bfd4f2`) |
  | — | `wontfix` | GitHub default; keep bare |

  - **Two stragglers to migrate** to match the colon canon: `needs-triage` → `needs:triage`, `needs-info` → `needs:info` (the rest are already colon in Recall).
  - **Work-kind:** keep `type:*` labels as the cross-repo taxonomy (Ed uses them today). Native org **issue types** are available now too — treat them as an *optional* later enhancement that could supersede `type:*`, not a forced swap. **Open: keep `type:` labels or adopt issue types?** (added to §8 open items).
  - `area:*` / archetype-specific (`channel:*`, `confidence:*`) added per project on top of the canon.
  - **Status** stays the board field; **Phase** a field/milestone. (Note: `risk:` is the one "priority-like" axis Ed keeps as a label — do not duplicate it as a board field.)
- **Views (per archetype, delivered via golden-copy or manual checklist):** Board-by-Status; a filtered Ready/active table sorted by Priority; archetype-specific extras (Roadmap on dates, "Good first issues", etc.).

### Feature 3 — A GitHub-side agent for human feedback, at $0

**Verified cost model:** Actions are **free & unlimited on public repos**; private repos get **2,000 min/mo (Free) / 3,000 (Pro)** — bot runs are seconds, so effectively $0. The catches: `GITHUB_TOKEN` **cannot write to Projects v2**, and `actions/add-to-project` needs a non-default token. Copilot coding agent **costs money** and is out; **GitHub Models** has a genuine $0 tier (≈50 high-model requests/day).

**Recommended $0 architecture:**

1. **Capture** — Issue Forms (`.github/ISSUE_TEMPLATE/feedback.yml`, `bug.yml`) with required fields, dropdowns, auto-applied `labels:` and `projects:` keys. Zero compute; auto-adds to the board on submit for same-account boards.
2. **Identity** — **one private GitHub App** ("atlas-feedback-bot"), free, installed on all repos, perms `issues:write` + `projects:write`; mints 1-hour tokens per run via `actions/create-github-app-token`. Chosen over a PAT (long-lived, account-tied) and over `GITHUB_TOKEN` (no Projects v2 scope).
3. **Route** — per-repo `feedback-router.yml` (trigger `issues: [opened,labeled]`) uses the App token + `actions/add-to-project` to push feedback onto the central board, then `create-or-update-comment` to acknowledge the reporter.
4. **Slash commands** — `issue_comment`-triggered `actions/github-script` using the free `GITHUB_TOKEN` for same-repo label/comment/react (`/triage`, `/needs-info`, `/ready`).
5. **Optional AI triage** — `issues: opened` step calling **GitHub Models free tier** to summarize/classify incoming feedback and suggest a triage label, staying under the daily cap; degrades gracefully when the cap is hit.

This is shipped as a **scaffold module** the skill writes into a target repo (gated behind interview Q7), so the "agent" is reproducible per repo at $0.

### Feature 4 — Predictable, repeatable, reusable templates

- **Reusable:** archetypes are versioned TypeScript in the repo; overlays compose; the same interview answers always yield the same board.
- **Repeatable:** `gh project copy` from golden projects reproduces views + workflows (the API can't); reconcile reproduces fields/options/labels deterministically.
- **Predictable:** default **plan** mode shows the exact diff before any write; lockfile records the last-applied spec hash + resolved IDs for audit.
- No existing tool does Projects-v2-as-code (verified — Terraform's project resources are dead post classic-Projects sunset; `github-label-sync` covers labels only). This is a genuine, worth-building gap; we assemble it from `gh project` + `gh api graphql` + a `github-label-sync`-style diff model.

---

## 5. Opinionated archetypes (starter set)

All share base fields `Status` + `Priority`. Views are the golden/checklist payload.

| Archetype | Status vocabulary | Extra fields | Distinct labels |
|---|---|---|---|
| **Solo Software** (default) | Backlog · Ready · In Progress · In Review · Done | Phase, Estimate(#), Area | `type:*`, `blocked`→(use native deps) |
| **Content Pipeline** | Idea · Drafting · Editing · Review · Scheduled · Published | Channel, Publish Date | `channel:blog|newsletter|social`, `needs-art` |
| **Research Tracker** | Open Question · Investigating · Evidence Gathered · Concluded · Parked | Confidence, Source(text) | `hypothesis`, `finding`, `dead-end`, `needs-verification` |
| **Life-OS Tasks** | Inbox · Next · In Progress · Waiting · Done | Area, Due(date), Energy | `quick-win`, `recurring`, `someday` |
| **Open-Source Repo** | Triage · Accepted · In Progress · In Review · Done | Type, Difficulty | `good first issue`, `help wanted`, `needs-repro`, `type:*` |

**Overlay modules** (additive, toggled by interview): `iterations` (Iteration field + sprint view), `estimation` (Estimate/Size), `roadmap` (Start/Target dates + Roadmap view), `triage` (Triage status + intake labels), `feedback-agent` (the §4.3 scaffold).

---

## 6. Highest-risk code path (must get right)

**Single-select option editing is destructive by API design.** `updateProjectV2Field` is a **full-list replace**; editing options **churns their node IDs and orphans existing item assignments** that referenced the old IDs. Mitigations baked into the reconcile engine:

- Always **read live options first** and send the **merged union** (existing ∪ desired) — never send only the delta (a subset wipes the rest).
- **Adding** options: safe-ish, auto.
- **Renaming/removing** options: **destructive → require `--confirm`**, warn "this resets N item assignments", and **re-resolve all affected item field values** after the mutation.
- **Fields and projects are never auto-deleted** — only flagged, pruned behind `--prune --confirm`.
- **Lockfile** `.agents/atlas/artifacts/<board-slug>.lock.json` (gitignored) caches names→IDs + last spec hash, but **every run re-queries live state** and rebuilds the map from names. Names are ground truth; IDs are ephemeral.

---

## 7. Implementation phases (each with a verify gate)

> Tracked on a board once approved (eat our own dog food — a "Solo Software" archetype board for this very skill).

**Phase 0 — Consolidate & de-drift** → *verify: one skill dir, both invocation names resolve, validators pass, no contradictions*
- **Keep `github-projects`; delete `githubprojects`.** Rationale: `github-projects` already matches the migration target on every axis — correct kebab `name:`, colon triage labels (`needs:triage`/`agent:ready`/`needs:human`), and lean `Todo/In Progress/In Review/Done` Status (= Recall Phase 0, the liked shape). `githubprojects` has the old hyphen labels, `Backlog/Ready/In Progress/Done` Status, and a wrong `GithubProjects` name.
  - Remove `~/.agents/skills/githubprojects/` from the store and its `~/.claude/skills/githubprojects` symlink; run `just skills-sync` to commit the removal.
  - Repoint the `/Continue`, `/Execute`, `/Status`, `/ProjectSetup` wrapper commands' recipe references from `githubprojects` → `github-projects` (descriptions + any recipe paths).
  - (Optional) keep a `githubprojects` → `github-projects` alias only if muscle-memory needs it; default is a clean single name.
- Reconcile the kept skill's remaining contradictions to the §4.2 canon (colon labels everywhere, issue-types note, lean Status).
- Pin `gh >= 2.94` as a prerequisite; replace GraphQL `addSubIssue` recipes with native `gh issue --parent/--add-sub-issue`; replace `blocked` label with native dependencies.

**Phase 0.5 — Org migration & identity** (see §8) → *verify: org exists, a test repo transferred with working redirects, `atlas` account + GitHub App can write a board, issue types visible*
- Create org; transfer repos; create `atlas` machine account; register the GitHub App; define issue types + org default labels.

**Phase 1 — Template layer** → *verify: `bun` typechecks; resolving an archetype+overlays yields the expected merged spec*
- `templates/types.ts` (`Archetype`, `Field`, `View`, `Label` interfaces; `FieldType` limited to API-supported types).
- The 5 base archetypes + overlay modules from §5.

**Phase 2 — Reconcile engine** → *verify: plan-mode diff on a throwaway board is correct; apply converges; re-apply is a no-op (idempotent); option-rename triggers confirm + preserves assignments*
- `Tools/Reconcile.ts`: snapshot → diff (3-bucket, github-label-sync model) → plan/apply/prune; the §6 option-merge safety; lockfile.
- Label sync across repos from a manifest (`gh label create --force`).

**Phase 3 — Interview + scaffold command** → *verify: interview output drives a full board bootstrap end-to-end on a test repo*
- `AskUserQuestion` flow → resolved config → `gh project copy` from golden (or create + manual-view checklist) → reconcile fields/options/labels → link repo → print the residual manual steps (auto-add workflow re-enable; views if no golden).

**Phase 4 — Golden projects** → *verify: a copied board has the intended views + workflows*
- Hand-build one golden board per archetype (views + automations in UI once); record their project numbers in the skill so copy can reference them.

**Phase 5 — Feedback agent scaffold** → *verify: on a test repo, an Issue-Form submission lands on the board with an ack comment, all at $0*
- `.github/` templates (Issue Forms, router workflow, slash-command workflow, optional Models triage); GitHub App setup runbook; wiring docs.

**Phase 6 — Recipe refresh + docs** → *verify: Continue/Execute/Status/ProjectSetup wrappers reference the new engine; AGENTS.md updated*
- Update the PM recipes and the `/Continue`, `/Execute`, `/Status`, `/ProjectSetup` command wrappers to the new model; document the whole thing.

---

## 8. Org migration & identity setup (new Phase 0.5)

This precedes the engine work because issue types, default labels, and templates only exist in an org.

1. **Org exists:** `Detego-Research-Group` (free, Organization). ✓
2. **Transfer repos** from `edheltzel` → `Detego-Research-Group`. Preserved: history, issues, PRs, stars, watchers, forks, **auto-redirects** for clones/links. **Watch:** Pages custom domains are NOT auto-redirected (update DNS first to avoid takeover); issue assignments to non-org-members are cleared; package↔repo links may break. Migrate low-risk repos first; verify redirects.
3. **Create the `atlas` machine account** (own email), add to the org as a member. $0 on free org.
4. **Register an org-owned GitHub App** ("atlas-bot") with `issues:write`, `pull-requests:write`, org `projects:write`. Store App ID + private key as org/repo secrets. Mint tokens in Actions via `actions/create-github-app-token`.
5. **Define org issue types**: enable defaults (`task`/`bug`/`feature`), add `Epic` and (optionally) `Spike` via `gh api graphql` + `createIssueType` (`GraphQL-Features: issue_types` header). *Verify in Settings → Planning → Issue types that this is free on your org plan — docs are GA but silent on the plan gate.*
6. **Set org default labels** to the `:`-namespaced taxonomy (applies to new repos only; backfill existing repos via the label-sync target in Phase 2).

**Caveat that the org does NOT fix:** the Actions `GITHUB_TOKEN` still can't write org Projects v2 — that's exactly why the GitHub App (step 4) exists.

## 9. Junior-dev explainer — "why GitHub Projects, not Linear?" (D5 / your Q5)

*Written plain, as requested. This is the rationale layer for the whole plan.*

**What GitHub's own docs recommend.** GitHub's "Best practices for Projects" page says, in plain terms: treat a project like a *database view over your issues*, not a sticky-note wall.
- Use **custom single-select fields** (Status, Priority, Phase) for structured data — **not labels** — because field values *sort and group* and labels don't. (This is the exact wall the cotera team hit: they used `P0`–`P3` labels and discovered "labels don't sort.")
- Keep **one project with many saved views** (backlog table, current iteration, roadmap), not many thin projects. ← *This is why we're consolidating your "Phase 0–4 = 5 projects" into one board with a Phase field/milestone.*
- **Automate status** (close issue → Done, auto-add new issues, auto-archive) so the board stays current without manual upkeep.
- **Decompose with sub-issues** (now GA, up to 50,000 items/project) for hierarchy.

**How others do it — and do startups pick Linear?** Yes, commonly — but it's a *scale event*, not a day-one default. The clearest real story (cotera, 10 engineers, ~500 tickets) switched to **Linear** for: a real sortable priority field, fast keyboard triage (30 issues in 5 min), and **Cycles** (sprints with auto carry-over, velocity, burndown) that GitHub Milestones lack. Linear's wins are **speed + opinionated defaults** — you get a working process with near-zero setup.

But teams **stay on GitHub Projects** for: **$0 cost** (Linear is $10–16/user/mo; its free tier hard-caps at **250 issues and then blocks new issue creation** — fatal for an agent that files issues), code lives in the same place (`Fixes #142` auto-closes on merge — cotera admitted GitHub's PR integration "remained superior"), and **no second system to keep in sync**. The documented "outgrow" signal is ~300–500 active issues across many repos with a 15+ person team running real sprints.

**The call for you (solo dev + AI agents already on GitHub): stay on GitHub Projects.**
1. The two reasons to switch (big-backlog triage, multi-person sprint analytics) **don't apply** to a solo operator.
2. **$0** vs a per-seat tax you'd pay even solo — and the 250-issue free cap would block your agents.
3. **Agent/CLI ergonomics are now excellent**: as of `gh` v2.94 (June 2026) the CLI manages sub-issues, issue types, and dependencies and returns them as **JSON** — your agents read/mutate the whole work structure from the terminal, no browser, no sync layer. (Linear has a strong Agent API too, but its best agent features sit behind the **Business** plan.)
4. **Gravity**: your code, PRs, CI already live on GitHub. One source of truth is exactly what GitHub's best-practices doc tells you to optimize for.

**The trend that seals it:** through 2025–2026 GitHub shipped sub-issues, issue types, dependencies, and full CLI/JSON support — closing most of the structural gap that used to force teams to Linear. The remaining Linear edge is *process polish and reporting*, which a solo dev doesn't need yet. **Revisit only if** you grow a team, run sprints, and your backlog crosses a few hundred active issues.

---

## 10. Risks & mitigations

- **Views/workflows API gap** → golden-copy + explicit residual-steps output; never pretend views were set when they weren't (log what was skipped).
- **Option-edit data loss** → §6 safety; default plan-mode; destructive ops confirm-gated.
- **Skill re-forking** → single canonical store + a validator that fails if a second copy reappears or if defaults contradict `docs/agents/triage-labels.md`.
- **`gh` version drift** → hard prerequisite check (`gh >= 2.94`) before any hierarchy op.
- **GitHub Models cap** (≈50/day) → AI triage degrades gracefully to no-AI routing.
- **Private-repo Actions minutes** → bot runs are seconds; document the 2,000-min ceiling so heavy CI isn't surprised.
- **Org migration breakage** (Pages domains not auto-redirected; non-member issue assignments cleared; package links) → migrate low-risk repos first, fix DNS before transfer, verify redirects per repo.
- **Issue-types plan gate uncertainty** → docs are GA but silent on free-org gating; verify in org Settings → Planning before committing the issue-type model; `type:` label remains the fallback.
- **Auto-add residual toggle** → keep it a documented one-step manual checklist by default; promote to an Interceptor script only if provisioning frequency justifies it. Never claim auto-add is on without verifying.

---

## 11. Sources (verified June 2026)

- Live GraphQL schema introspection (`gh api graphql`), `gh` v2.95.0 `--help`, REST `2026-03-10`.
- Projects API & limits: docs.github.com/en/issues/planning-and-tracking-with-projects (automating, copying, templates, best-practices).
- Sub-issues/types/deps GA + CLI: github.blog/changelog/2026-06-10, community #154148, #175785 (issue types personal-repo gap, open 2026-06).
- $0 agent: Actions billing docs, `GITHUB_TOKEN` auth, issue-forms syntax, `actions/add-to-project`, `actions/create-github-app-token`, GitHub Models rate limits.
- Declarative precedent: Financial-Times/github-label-sync (diff model), terraform-provider-github (classic projects dead), keycloak-config-cli (converge analogy).
```
