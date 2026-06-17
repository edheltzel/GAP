# Hierarchy — Epic → Feature → Task issue model

How to structure work as a three-level GitHub issue hierarchy and create issues correctly on a Projects v2 board. Adapted from the `claude-workflow-template` (`workflow.md` §3, §6); GraphQL plumbing is the high-value part.

## The model

```
Phase Epic (parent issue)            e.g. #1 "Phase 1: Foundation"
  └─ Feature Issue (sub-issue)       e.g. #5 "Add user authentication"
       └─ Task Sub-issue (optional)  e.g. "Configure JWT refresh"
```

- **Phase Epic** — one per development phase. Holds the phase goal; progress bar auto-rolls-up from sub-issues.
- **Feature Issue** — one per distinct piece of work. Carries the user story, acceptance criteria (checkboxes), and a link to its plan in `.agents/atlas/plans/`.
- **Task Sub-issue** — optional, for larger features that want granular progress.

**Why sub-issues, not just labels:** sub-issues give automatic roll-up ("11/21 complete"), hierarchy in the board, and real parent/child structure. Labels only filter.

**Phase-less issues** are fine — bugs, polish, ad-hoc work need no Phase. Track them by Status alone.

## Issue creation checklist

Run every time an issue is created (by a command, an agent, or by hand).

### Pre-creation: dedup + parent lookup

```bash
# 1. Search for duplicates before creating
gh issue list --repo OWNER/REPO --state open --json number,title,labels --limit 200 \
  --jq '.[] | "#\(.number): \(.title)"'

# 2. If it belongs to an epic, confirm it isn't already a sub-issue there
gh api graphql -f query='query { repository(owner: "OWNER", name: "REPO") {
  issue(number: EPIC_NUM) { subIssues(first: 50) { nodes { number title } } } } }' \
  --jq '.data.repository.issue.subIssues.nodes[]'
```

### Always required

1. Create the issue (`gh issue create`).
2. Add it to the project board (`addProjectV2ItemById`).
3. Set **Status** (usually Backlog for new issues).
4. Set **Priority** — labels alone do NOT set board fields.

### When it belongs to a Phase/Epic

5. Set the **Phase** field.
6. Link it as a sub-issue of the epic (`addSubIssue`).

## Full GraphQL recipe

Field/option IDs come from `ProjectSetup.md` (captured at board creation). See `Labels.md` for which labels to apply (use the repo's existing taxonomy — do **not** invent `type:`/`priority:` labels).

```bash
# 1. Create issue
ISSUE_URL=$(gh issue create --repo OWNER/REPO \
  --title "Feature title" \
  --label "enhancement" \
  --body "User story + acceptance criteria")

# 2. Resolve its node ID
ISSUE_NUM=$(echo "$ISSUE_URL" | grep -o '[0-9]*$')
ISSUE_ID=$(gh api graphql -f query='query { repository(owner: "OWNER", name: "REPO") {
  issue(number: '"$ISSUE_NUM"') { id } } }' --jq '.data.repository.issue.id')

# 3. Add to the board
ITEM_ID=$(gh api graphql -f query='mutation {
  addProjectV2ItemById(input: { projectId: "PROJECT_ID", contentId: "'"$ISSUE_ID"'" }) {
    item { id } } }' --jq '.data.addProjectV2ItemById.item.id')

# 4. Set Phase + Priority in one mutation
gh api graphql -f query='mutation {
  phase: updateProjectV2ItemFieldValue(input: { projectId: "PROJECT_ID", itemId: "'"$ITEM_ID"'",
    fieldId: "PHASE_FIELD_ID", value: { singleSelectOptionId: "PHASE_OPTION_ID" } }) { projectV2Item { id } }
  priority: updateProjectV2ItemFieldValue(input: { projectId: "PROJECT_ID", itemId: "'"$ITEM_ID"'",
    fieldId: "PRIORITY_FIELD_ID", value: { singleSelectOptionId: "PRIORITY_OPTION_ID" } }) { projectV2Item { id } }
}'

# 5. Link as sub-issue of the Phase Epic (if applicable)
gh api graphql -f query='mutation {
  addSubIssue(input: { issueId: "EPIC_NODE_ID", subIssueId: "'"$ISSUE_ID"'" }) {
    issue { number } subIssue { number } } }'
```

## Notes

- `addSubIssue` is a GraphQL mutation; there is no `gh issue` flag for it yet.
- Plans live in `.agents/atlas/plans/` — link the feature issue to its plan file, not the other way around.
- Keep the epic body's checklist in sync only if you aren't using sub-issues; with sub-issues the roll-up is automatic.
