# Labels — the canonical `:`-namespaced taxonomy

Policy for labels on a board-tracked repo. There is a **canonical start taxonomy** (below), synced across every repo, derived from the live `edheltzel/Recall` label set. Apply it on bootstrap; extend per-repo only with `area:*` and archetype-specific labels. The strings and colors match `CANON_LABELS` in `templates/archetypes.ts`.

## Canonical taxonomy (start set)

Every repo gets these on bootstrap. Colors match Recall.

| Namespace | Labels | Role | Color |
|-----------|--------|------|-------|
| `type:` | `type:bug` `type:feature` `type:docs` `type:test` `type:refactor` `type:chore` | work kind | `#d4c5f9` |
| `needs:` | `needs:triage` `needs:info` `needs:human` | what the issue is waiting on / who acts next | `#FBCA04` / `#c73aa4` / `#b60205` |
| `risk:` | `risk:low` `risk:medium` `risk:high` | ambiguity / judgement level | `#edeedd` / `#fef2c0` / `#f9d0c4` |
| `agent:` | `agent:ready` `agent:blocked` `agent:complete` | AFK-agent lifecycle signal | `#bfd4f2` |
| — | `wontfix` | GitHub default; keep bare | `#ededed` |

Triage role meanings (who acts next):

- `needs:triage` — maintainer needs to evaluate this issue
- `needs:info` — waiting on reporter for more information
- `agent:ready` — fully specified, ready for an AFK agent (`risk:low` work)
- `needs:human` — requires human implementation
- `wontfix` — will not be actioned

Slash-command transitions among `needs:triage`, `needs:info`, and `agent:ready` are defined in [FeedbackAgent.md](FeedbackAgent.md).

## Principles

- **`:`-namespaced, color-grouped.** Every label carries a `prefix:` (except the GitHub default `wontfix`). Prefixes group visually and prevent collisions.
- **`risk:` is a label, not the Priority field.** Priority is a board field (`P0–P3`, `High/Medium/Low`, or omitted), chosen in the ProjectSetup interview. Do not encode risk as that field, and do not add `priority:` labels.
- **Work-kind = `type:` labels.** On an org repo, native issue types may later supersede `type:*`; until that migration is decided, `type:*` labels are canonical.
- **Status and Phase are board fields, not labels.** Blocked/blocking relationships are native issue dependencies, not a `blocked` label.

## Before applying any label

Read the live taxonomy first — never assume — then reconcile toward the canon:

```bash
gh label list --repo OWNER/REPO --limit 200 --json name,color,description
```

The reconcile engine's label-sync mode converges any repo to the canon above (`bun Tools/Reconcile.ts --archetype <key> --repo OWNER/REPO`; idempotent via `gh label create --force`), preserving existing `area:*` and archetype labels. Deletes only behind `--prune --confirm`.

## Migration note

Older repos may still carry the pre-colon hyphen spellings of `needs:triage` and `needs:info`. Rename those two to the colon canon:

```bash
gh label edit "OLD_HYPHEN_NAME" --repo OWNER/REPO --name "needs:triage"
```

The rest of the set (`needs:human`, `agent:*`, `risk:*`, `type:*`) is already colon in Recall.

## Adding a new label

Only when a repo genuinely needs a category the canon lacks, matching the `prefix:value` style:

```bash
gh label create "area:dashboard" --repo OWNER/REPO --color BFD4F2 --description "Dashboard subsystem"
```
