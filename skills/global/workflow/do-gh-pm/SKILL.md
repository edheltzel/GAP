---
name: do-gh-pm
description: GitHub Projects management via gh CLI — creating projects, managing items/fields, plus opinionated PM recipes — board bootstrap, Epic→Feature→Task issue hierarchy with sub-issue linking, label policy, running a plan against the board, picking next work, and status reporting. Use when working with GitHub Projects (v2), adding issues/PRs to projects, creating custom fields, bootstrapping a board, decomposing work into epics/features/tasks, executing a plan with board updates, or reporting board progress. Triggers on gh project, project board, kanban, GitHub project, project items, epic, sub-issue, board setup, next work, project status.
alwaysAllow: ["Bash"]
---

# do-gh-pm — GitHub Projects CLI

GitHub Projects (v2) management via `gh project` commands. Requires the `project` scope which can be added with `gh auth refresh -s project`.

## Prerequisites

Requires **`gh` >= 2.94** — the native sub-issue, parent, and dependency flags
(`gh issue create --parent`, `gh issue edit --add-sub-issue/--add-blocked-by/--add-blocking`)
used by the hierarchy recipes landed in that release. Check with `gh --version`.

Verify authentication includes project scope:

```bash
gh auth status  # Check current scopes
gh auth refresh -s project  # Add project scope if missing
```

## Quick Reference

### List & View Projects

```bash
# List your projects
gh project list

# List org projects (including closed)
gh project list --owner ORG_NAME --closed

# View project details
gh project view PROJECT_NUM --owner OWNER

# Open in browser
gh project view PROJECT_NUM --owner OWNER --web

# JSON output with jq filtering
gh project list --format json | jq '.projects[] | {number, title}'
```

### Create & Edit Projects

```bash
# Create project
gh project create --owner OWNER --title "Project Title"

# Edit project
gh project edit PROJECT_NUM --owner OWNER --title "New Title"
gh project edit PROJECT_NUM --owner OWNER --description "New description"
gh project edit PROJECT_NUM --owner OWNER --visibility PUBLIC

# Close/reopen project
gh project close PROJECT_NUM --owner OWNER
gh project close PROJECT_NUM --owner OWNER --undo  # Reopen
```

### Link Projects to Repos

```bash
# Link to repo
gh project link PROJECT_NUM --owner OWNER --repo REPO_NAME

# Link to team
gh project link PROJECT_NUM --owner ORG --team TEAM_NAME

# Unlink
gh project unlink PROJECT_NUM --owner OWNER --repo REPO_NAME
```

## Project Items

### Add Existing Issues/PRs

```bash
# Add issue to project
gh project item-add PROJECT_NUM --owner OWNER --url https://github.com/OWNER/REPO/issues/123

# Add PR to project
gh project item-add PROJECT_NUM --owner OWNER --url https://github.com/OWNER/REPO/pull/456
```

### Create Draft Items

```bash
gh project item-create PROJECT_NUM --owner OWNER --title "Draft item" --body "Description"
```

### List Items

```bash
# List items (default 30)
gh project item-list PROJECT_NUM --owner OWNER

# List more items
gh project item-list PROJECT_NUM --owner OWNER --limit 100

# JSON output
gh project item-list PROJECT_NUM --owner OWNER --format json
```

### Edit Items

Items are edited by their ID (obtained from `item-list --format json`).

```bash
# Edit draft issue title/body
gh project item-edit --id ITEM_ID --title "New Title" --body "New body"

# Update field value (requires field-id and project-id)
gh project item-edit --id ITEM_ID --project-id PROJECT_ID --field-id FIELD_ID --text "value"
gh project item-edit --id ITEM_ID --project-id PROJECT_ID --field-id FIELD_ID --number 42
gh project item-edit --id ITEM_ID --project-id PROJECT_ID --field-id FIELD_ID --date "2024-12-31"
gh project item-edit --id ITEM_ID --project-id PROJECT_ID --field-id FIELD_ID --single-select-option-id OPTION_ID
gh project item-edit --id ITEM_ID --project-id PROJECT_ID --field-id FIELD_ID --iteration-id ITER_ID

# Clear field value
gh project item-edit --id ITEM_ID --project-id PROJECT_ID --field-id FIELD_ID --clear
```

### Archive/Delete Items

```bash
gh project item-archive PROJECT_NUM --owner OWNER --id ITEM_ID
gh project item-delete PROJECT_NUM --owner OWNER --id ITEM_ID
```

## Project Fields

### List Fields

```bash
gh project field-list PROJECT_NUM --owner OWNER
gh project field-list PROJECT_NUM --owner OWNER --format json
```

### Create Fields

```bash
# Text field
gh project field-create PROJECT_NUM --owner OWNER --name "Notes" --data-type TEXT

# Number field
gh project field-create PROJECT_NUM --owner OWNER --name "Points" --data-type NUMBER

# Date field
gh project field-create PROJECT_NUM --owner OWNER --name "Due Date" --data-type DATE

# Single select with options
gh project field-create PROJECT_NUM --owner OWNER --name "Priority" \
  --data-type SINGLE_SELECT \
  --single-select-options "P0,P1,P2,P3"
```

### Delete Fields

```bash
gh project field-delete --id FIELD_ID
```

## Common Workflows

### Add Issue and Set Status

```bash
# 1. Add issue to project
gh project item-add 1 --owner "@me" --url https://github.com/owner/repo/issues/123

# 2. Get item ID and field IDs
gh project item-list 1 --owner "@me" --format json | jq '.items[-1]'
gh project field-list 1 --owner "@me" --format json

# 3. Update status field
gh project item-edit --id ITEM_ID --project-id PROJECT_ID \
  --field-id STATUS_FIELD_ID --single-select-option-id OPTION_ID
```

### Bulk Add Issues

```bash
# Add all open issues from a repo
gh issue list --repo owner/repo --state open --json url -q '.[].url' | \
  xargs -I {} gh project item-add 1 --owner "@me" --url {}
```

## Template layer (`templates/`)

The board's intended shape is config-as-code: five **archetypes** (`templates/archetypes.ts` — `solo-software` (default), `content-pipeline`, `research-tracker`, `life-os-tasks`, `open-source-repo`), each declaring its Status vocabulary, fields, labels, and views, plus the canonical `:`-namespaced label taxonomy (`CANON_LABELS`). Five additive **overlays** (`templates/overlays.ts` — `iterations`, `estimation`, `roadmap`, `triage`, `feedback-agent`) compose on top; `feedback-agent` also carries the `FEEDBACK_AGENT_SCAFFOLD` install manifest for the repo-side `.github/` files.

`templates/resolve.ts` resolves `(archetype key, overlay keys, priority scheme)` to one merged spec — pure and deterministic, returning a detached clone. Priority is selected as `p0-p3`, `high-medium-low`, or `none`. Merge semantics (`templates/merge.ts`): dedupe by name, first occurrence wins; overlay status options prepend, while fields, labels, and views append. Self-test: `bun templates/smoke.ts`.

## Interview scaffold (`Tools/Scaffold.ts`)

`ProjectSetup` conducts eight explicit `AskUserQuestion` choices, and `Tools/scaffold-core.ts` deterministically maps them to an archetype, ordered overlays, visibility, and Priority scheme. `Tools/Scaffold.ts` is dry-run by default: it prints the create-or-copy, visibility, repository-link, reconcile-plan, and safe-apply operations without executing them.

After the user reviews that plan, `--apply` verifies `gh >= 2.94`, preflights feedback-file conflicts, copies a supplied golden project or creates an empty board, links the repository, runs `Tools/Reconcile.ts` in the target checkout (so the lockfile is not written into the installed skill), installs the optional feedback scaffold without overwriting differing files, and prints the views/workflows that remain manual. It never prunes live board data. See [ProjectSetup.md](references/ProjectSetup.md).

Run from the target checkout. `$SKILL` is the installed skill directory.

```bash
bun "$SKILL/Tools/Scaffold.ts" --owner OWNER --repo OWNER/REPO --title "TITLE" \
  --work-kind software --visibility private --contributors solo \
  --cadence continuous --estimation no --roadmap no --feedback no \
  --priority p0-p3 --repo-path "$PWD"
# review, then rerun the identical command with --apply
```

## Reconcile engine (`Tools/Reconcile.ts`)

Converges a live board + repo labels toward a resolved spec, terraform-style: **snapshot** live state (GraphQL fields/options, `gh label list`) → **3-bucket diff** (create / update / delete, each op classified safe or destructive) → **plan** (default, dry-run) / **apply** (safe ops only) / **prune** (`--apply --prune --confirm` for destructive removals). Re-apply on a converged board is a no-op (idempotent). Diff logic lives in the pure, unit-tested `Tools/reconcile-core.ts`; the CLI shell only does I/O.

Safety (plan §6): single-select option editing is a **full-list replace** in the API, so the engine always sends the merged union of live ∪ desired options with existing node ids carried forward — never a subset, never id churn. Option **removals** orphan item assignments → destructive, gated behind `--prune --confirm`, preceded by an exact "resets N item assignments" warning and a recovery snapshot. **Renames are never inferred** — `--rename "Field:Old=New"` does an explicit in-place, assignment-preserving rename. Extra live fields are flagged, never auto-deleted. After a board-mode apply (label-sync-only runs write no lockfile), a gitignored **lockfile** (`.agents/atlas/artifacts/<owner>-project-<n>.lock.json`) caches names→IDs + the spec hash for audit; live state stays ground truth.

```bash
bun "$SKILL/Tools/Reconcile.ts" --project N --owner OWNER --archetype solo-software   # plan, from the target checkout
bun "$SKILL/Tools/Reconcile.ts" ... --apply                                          # safe ops
bun "$SKILL/Tools/Reconcile.ts" --archetype KEY --repo OWNER/REPO                    # label sync only
```

Full bootstrap walk-through: [ProjectSetup.md](references/ProjectSetup.md). Tests: `cd Tools && bun test`.

## Feedback Agent (overlay)

The `feedback-agent` overlay (interview Q7) adds a "Feedback Intake" board view and drops a `.github/` scaffold into the target repo — Issue Forms plus two workflows, all at $0. Templates live in `templates/feedback-agent/`; the install manifest is `FEEDBACK_AGENT_SCAFFOLD` in `templates/overlays.ts`.

| Installed file | Purpose |
|---|---|
| `.github/ISSUE_TEMPLATE/feedback.yml` | Feedback Issue Form — auto-applies `needs:triage`; best-effort project auto-add; router is the reliable path |
| `.github/ISSUE_TEMPLATE/bug.yml` | Bug Issue Form — `type:bug` + `needs:triage` |
| `.github/workflows/feedback-router.yml` | Routes intake onto the central board (App token), acks the reporter as Atlas, optional GitHub Models triage |
| `.github/workflows/pm-commands.yml` | `/pm:triage` `/pm:needs-info` `/pm:ready` maintainer slash commands on the free `GITHUB_TOKEN` |

**GitHub App setup (once):** use a GitHub App (`<APP_ID>`, installation `<INSTALLATION_ID>`). Set `ATLAS_APP_ID` / `ATLAS_APP_PRIVATE_KEY` (App token mints the Projects v2 write — `GITHUB_TOKEN` can't) and `ATLAS_KEY_PAT` (ack-comment author) as repo/org secrets, then replace the `ORG/PROJECT_NUMBER` placeholders in the copied files. Full install steps and the $0 e2e verification runbook: [FeedbackAgent.md](references/FeedbackAgent.md).

## JSON Output & jq Patterns

```bash
# Get project IDs
gh project list --format json | jq '.projects[] | {number, id, title}'

# Get field IDs and options
gh project field-list 1 --owner "@me" --format json | jq '.fields[] | {id, name, options}'

# Get item IDs with field values
gh project item-list 1 --owner "@me" --format json | jq '.items[] | {id, title, fieldValues}'

# Filter items by status
gh project item-list 1 --owner "@me" --format json | \
  jq '.items[] | select(.status == "In Progress")'
```

## Reference Files

### gh primitives

- **[items.md](references/items.md)**: Item management, editing field values, bulk operations
- **[fields.md](references/fields.md)**: Field types, creating custom fields, option management

### PM recipes

- **[Hierarchy.md](references/Hierarchy.md)**: Epic→Feature→Task issue model + issue-creation checklist using native sub-issues/dependencies (`gh issue create --parent`, `gh issue edit --add-sub-issue/--add-blocked-by`); GraphQL only for board-item field values
- **[ProjectSetup.md](references/ProjectSetup.md)**: Eight-question board interview → deterministic archetype/overlays → reviewed create-or-copy plan → linked and safely reconciled board → manual views/automations checklist
- **[Labels.md](references/Labels.md)**: Label policy — the canonical `:`-namespaced taxonomy (`type:` `needs:` `risk:` `agent:` `wontfix`); Status/Phase/Priority are board fields, not labels
- **[FeedbackAgent.md](references/FeedbackAgent.md)**: The `feedback-agent` overlay's `.github/` scaffold — Issue Forms, feedback router, `/pm:*` slash commands, GitHub App setup, $0 e2e verification runbook
- **[Execute.md](references/Execute.md)**: Run an existing plan against the board — pre-flight (bun), In Progress, 4-tier deviation rules, AC/close updates
- **[Continue.md](references/Continue.md)**: Pick the next task from board state (status → priority → deps)
- **[Status.md](references/Status.md)**: Board progress report — phase progress, status counts, recently-closed, up-next

### Related skills

- Personal tasks in a todo.txt file, not a board: `do-todo`, if installed.
- Status across several projects or sources, not one board: `do-regroup`, if installed.
- Writing the issue body itself: `do-tech-writing`, if installed.

## Command Summary

| Command | Purpose |
|---------|---------|
| `project list` | List projects |
| `project view` | View project details |
| `project create` | Create new project |
| `project edit` | Modify project settings |
| `project close` | Close/reopen project |
| `project link/unlink` | Connect to repo/team |
| `project item-add` | Add existing issue/PR |
| `project item-create` | Create draft item |
| `project item-list` | List project items |
| `project item-edit` | Update item fields |
| `project item-archive` | Archive item |
| `project item-delete` | Remove item |
| `project field-list` | List project fields |
| `project field-create` | Add custom field |
| `project field-delete` | Remove field |
