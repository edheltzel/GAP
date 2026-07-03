# ProjectSetup — bootstrap a board from an archetype

One-time board bootstrap for a repo, driven by the template layer + reconcile engine (paths relative to the skill root):

1. Pick an **archetype** (+ overlays) — `templates/archetypes.ts` + `templates/overlays.ts`, resolved to one merged spec by `templates/resolve.ts`.
2. Create the empty board and link it to the repo.
3. Converge fields / options / labels with `Tools/Reconcile.ts` (plan → apply).
4. Finish the manual-only steps (views, automations) from the checklist.

> The interview flow that picks archetype + overlays from answers is Phase 3 (#25, unbuilt) — until it ships, choose them by hand from the tables below.

## 0. Prerequisites

```bash
gh auth status                 # must be authenticated
gh auth refresh -s project     # the `project` scope is required for board ops
gh --version                   # >= 2.94 (skill prerequisite)
git rev-parse --is-inside-work-tree && git remote get-url origin
```

## 1. Pick an archetype + overlays

Archetypes (each declares its Status vocabulary, fields, labels, and views; all share the Priority field P0–P3 and the canonical label taxonomy — see `Labels.md`):

| Key | Status vocabulary | Extra fields |
|-----|-------------------|--------------|
| `solo-software` (default) | Backlog · Ready · In Progress · In Review · Done | Phase, Area |
| `content-pipeline` | Idea · Drafting · Editing · Review · Scheduled · Published | Channel, Publish Date |
| `research-tracker` | Open Question · Investigating · Evidence Gathered · Concluded · Parked | Confidence, Source |
| `life-os-tasks` | Inbox · Next · In Progress · Waiting · Done | Area, Due, Energy |
| `open-source-repo` | Triage · Accepted · In Progress · In Review · Done | Type, Difficulty |

Overlays (additive, composable — applied in the order given):

| Key | Adds |
|-----|------|
| `iterations` | Iteration field + "Current Sprint" view |
| `estimation` | Estimate (number) + Size (XS–XL) fields |
| `roadmap` | Start/Target Date fields + Roadmap view |
| `triage` | Leading "Triage" status column + intake labels |
| `feedback-agent` | "Feedback Intake" view + the repo-side `.github/` scaffold (`FEEDBACK_AGENT_SCAFFOLD` install manifest) — see `FeedbackAgent.md` |

## 2. Create the board and link it to the repo

The engine converges an **existing** board; it does not create one:

```bash
gh project create --owner OWNER --title "TITLE"
# note the returned project number, then:
gh project link PROJECT_NUM --owner OWNER --repo REPO
```

## 3. Reconcile fields, options, and labels (the engine)

`Tools/Reconcile.ts` snapshots live board + label state, diffs it against the resolved archetype spec into a 3-bucket plan (create / update / delete), prints a terraform-style plan, and only writes on `--apply`:

```bash
# dry-run plan (default — always run this first; no writes)
bun Tools/Reconcile.ts --project PROJECT_NUM --owner OWNER \
  --archetype solo-software --overlay estimation --repo OWNER/REPO

# apply SAFE ops only: field/option/label creates + colour/description drift
bun Tools/Reconcile.ts --project PROJECT_NUM --owner OWNER \
  --archetype solo-software --overlay estimation --repo OWNER/REPO --apply
```

Safety model (the plan's §6 invariants, enforced by the engine):

- **Option edits are a full-list replace** in the API — the engine always sends the merged union of live ∪ desired options with each existing option's node id carried forward, so a write never drops options it didn't mean to and never resets item assignments it keeps.
- **Removals are destructive** (they orphan item assignments) — included only under `--prune`, executed only with `--apply --prune --confirm`, preceded by a "resets N item assignments" warning and a recovery snapshot of the affected item→option assignments.
- **Renames are never inferred** from the diff — an explicit `--rename "Field:Old=New"` (repeatable, needs `--apply`) renames the option in place, id preserved, assignments kept.
- **Extra live fields are never auto-deleted** — flagged as warnings; removed only behind `--prune --confirm`.
- **Re-apply is a no-op** — a converged board plans "No changes" (idempotent). Note: extra live options are **silently preserved** in the union, so "No changes" means nothing is orphaned or lost — it does **not** mean the vocabulary matches the spec exactly.
- **A fresh board starts with GitHub's default Status** (`Todo / In Progress / Done`). `In Progress` and `Done` match the default archetype; the stray `Todo` is preserved as an extra. Fold it into the canon vocabulary with an assignment-preserving rename — `--rename "Status:Todo=Backlog" --apply` — or remove it with `--apply --prune --confirm`.
- **ITERATION fields can't be created via the CLI** — the engine skips them with a warning; create them in the web UI, then re-run.

After a **board-mode** `--apply` (label-sync-only runs write no lockfile), the engine writes a **lockfile** — `.agents/atlas/artifacts/<owner>-project-<n>.lock.json` (gitignored; `--lockdir` overrides) — caching names→IDs plus the spec hash. Live state stays ground truth (every run re-queries); the lockfile is the audit trail **and where the field/option IDs for the GraphQL snippets in `Hierarchy.md` / `Execute.md` live**. To capture IDs without an apply:

```bash
gh project field-list PROJECT_NUM --owner OWNER --format json \
  | jq '.fields[] | {name, id, options: (.options // [] | map({name, id}))}'
```

The GraphQL snippets also need the project's own node id — capture it any time (no apply required):

```bash
gh project view PROJECT_NUM --owner OWNER --format json | jq '{number, id, title}'
```

Label sync alone (no board) works across repos from flags or a manifest:

```bash
bun Tools/Reconcile.ts --archetype solo-software --repo OWNER/A --repo OWNER/B
bun Tools/Reconcile.ts --archetype solo-software --manifest repos.json
# manifest: { "repos": ["owner/a", ...], "archetype"?: "<key>", "overlays"?: ["<k>"] }
```

## 4. Manual-only checklist (the API can't create these)

Views and automations exist only in the web UI — `gh project view PROJECT_NUM --owner OWNER --web`:

- [ ] **Views** — create each view the resolved spec declares (each archetype's `views` array in `templates/archetypes.ts`, plus the overlay table in §1; e.g. `solo-software`: "Board by Status" board, "Ready" table filtered `status:Ready,"In Progress"`), plus any extras you want (an Epics table with hierarchy on, a `label:type:bug` board). The reconcile plan does not list views — the engine doesn't reconcile them (no API).
- [ ] **Automations** — Auto-add (new issue in repo → add to board), Item closed (→ Status Done), Auto-archive (Done 14+ days).
- [ ] **Iteration field** — only if an overlay declared one and the engine warned it was skipped.
- [ ] **feedback-agent overlay** — install the `.github/` scaffold + secrets per `FeedbackAgent.md`.

## 5. Verify

Re-run the engine in plan mode: a bootstrapped board reports **"No changes. Board matches the resolved spec (idempotent no-op)."**
