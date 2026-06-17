# ProjectSetup — bootstrap a board with fields, views, and captured IDs

One-time board bootstrap for a repo: create the Projects v2 board, add the Status / Phase / Priority fields, create the working views, and **capture every field/option ID** so the GraphQL snippets in `Hierarchy.md`, `Execute.md`, etc. have something to reference. Adapted from `init-project.md` — board logic only. **No `CLAUDE.md` generator, no Husky/npm.**

## 0. Prerequisites

```bash
gh auth status                 # must be authenticated
gh auth refresh -s project     # the `project` scope is required for board ops
git rev-parse --is-inside-work-tree && git remote get-url origin
```

## 1. Create the board and link it to the repo

```bash
gh project create --owner OWNER --title "PROJECT Roadmap"
# note the returned project number, then:
gh project link PROJECT_NUM --owner OWNER --repo REPO
```

## 2. Create the fields

Status usually exists by default (Backlog/Ready/In Progress/Done — adjust to match). Add Phase and Priority:

```bash
gh project field-create PROJECT_NUM --owner OWNER --name "Phase" \
  --data-type SINGLE_SELECT --single-select-options "Phase 1,Phase 2,Phase 3"
gh project field-create PROJECT_NUM --owner OWNER --name "Priority" \
  --data-type SINGLE_SELECT --single-select-options "Low,Medium,High,Critical"
```

## 3. Capture field + option IDs (critical)

The GraphQL mutations elsewhere need these IDs. Capture them once and store them where the project's recipes can find them (a project-local notes block, e.g. the repo's `AGENTS.md` or a `.agents/atlas/` note):

```bash
gh project field-list PROJECT_NUM --owner OWNER --format json \
  | jq '.fields[] | {name, id, options: (.options // [] | map({name, id}))}'

# Project ID itself:
gh project list --owner OWNER --format json \
  | jq '.projects[] | select(.number == PROJECT_NUM) | {number, id, title}'
```

Record: `PROJECT_ID`, `STATUS_FIELD_ID` + each status option ID, `PHASE_FIELD_ID` + option IDs, `PRIORITY_FIELD_ID` + option IDs.

## 4. Views (one board, multiple views)

Views give visual separation without multiple boards. Create them in the web UI (`gh project view PROJECT_NUM --owner OWNER --web`) or accept the defaults:

| View | Layout | Filter | Purpose |
|------|--------|--------|---------|
| Epics | Table (hierarchy on) | `has:phase` | Phase overview with expandable sub-issues |
| Active Work | Board by Status | `status:Ready,"In Progress"` | Daily working view |
| Backlog | Board | `status:Backlog` | Not yet started |
| Bugs | Board | `label:bug` | All bugs by status |

## 5. Automations (web UI)

| Automation | Trigger | Action |
|------------|---------|--------|
| Auto-add | New issue in repo | Add to board |
| Item closed | Issue closed | Status → Done |
| Auto-archive | Done for 14+ days | Archive |

## Labels

Label setup is a separate concern — see `Labels.md`. Use the repo's existing labels; do **not** create the template's `type:`/`priority:`/`source:` taxonomy. Phase and Priority are **board fields**, not labels.

## Gotcha — field-ID brittleness

If you later rename/recreate a field or option in the web UI, its ID changes and the captured IDs go stale (mutations fail silently or error). Re-run step 3 and update the stored IDs whenever the board's field structure changes.
