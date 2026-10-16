# Labels — use the repo's existing taxonomy

Policy for labels on a board-tracked repo. **Keep the repo's existing labels; adjust the workflow to them.** Do **not** import the `claude-workflow-template`'s `type:`/`priority:`/`source:` taxonomy — it duplicates what the repo and the board already express.

## Principle

- **Labels = categorization the repo already uses.** Whatever `gh label list` returns is the taxonomy. Extend it on the repo's own terms, not by copying a foreign scheme.
- **Workflow state = triage labels.** If the repo defines triage labels (for example `needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, `wontfix`), use them to signal *who acts next*.
- **Phase and Priority are board FIELDS, not labels** (see `ProjectSetup.md`). This avoids the `phase:N` / `priority:N` label-vs-field duplication the template suffers from.

## Before applying any label

Always read the live taxonomy first — never assume:

```bash
gh label list --repo OWNER/REPO --limit 200
```

A typical repo has GitHub's defaults (`bug`, `documentation`, `enhancement`, `duplicate`, `wontfix`, …) plus its own labels (for example `area:api`, `safety`, `testing`). The recipes apply labels **from the live set** — they do not invent new ones.

## What changed vs the template

| Template did | We do instead |
|--------------|---------------|
| `type:feature\|bug\|infra\|polish\|dx\|tech-debt\|docs` | Use existing labels (`enhancement`, `bug`, `documentation`, …) |
| `priority:critical\|high\|medium\|low` labels | **Priority is a board field** |
| `phase:1`…`phase:N` labels | **Phase is a board field** |
| `source:research\|user-report\|internal` | Skipped (not adopted) |
| Triage via ad-hoc labels | The repo's own triage labels, if it has them |

## Adding a new label

Only when the repo genuinely lacks a needed category, and matching the repo's existing naming style:

```bash
gh label create "area:dashboard" --repo OWNER/REPO --color BFD4F2 --description "Dashboard subsystem"
```

If the repo uses a `prefix:value` style (`area:api`), follow it. If it uses bare GitHub-default names, follow that. Consistency with the repo beats consistency with this template.
