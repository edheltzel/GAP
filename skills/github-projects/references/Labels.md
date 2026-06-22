# Labels — the canonical `:`-namespaced taxonomy

Policy for labels on a board-tracked repo. There is now a **canonical start taxonomy** (below), synced across every repo, derived from the live `edheltzel/Recall` label set. Apply it on bootstrap; extend per-repo only with `area:*` / archetype-specific labels.

## Canonical taxonomy (start set)

Every repo gets these on bootstrap. Colors match Recall.

| Namespace | Labels | Role | Color |
|-----------|--------|------|-------|
| `type:` | `type:bug` `type:feature` `type:docs` `type:test` `type:refactor` `type:chore` | work kind | `#d4c5f9` |
| `needs:` | `needs:triage` `needs:info` `needs:human` | what the issue is waiting on / who acts next | `#FBCA04` / `#c73aa4` / `#b60205` |
| `risk:` | `risk:low` `risk:medium` `risk:high` | ambiguity / judgement level (the priority-like axis) | `#edeedd` / `#fef2c0` / `#f9d0c4` |
| `agent:` | `agent:ready` `agent:blocked` `agent:complete` | AFK-agent lifecycle signal | `#bfd4f2` |
| — | `wontfix` | GitHub default; keep bare | `#ededed` |

Triage role meanings (the `who-acts-next` signals) — see `docs/agents/triage-labels.md`:

- `needs:triage` — maintainer needs to evaluate this issue
- `needs:info` — waiting on reporter for more information
- `agent:ready` — fully specified, ready for an AFK agent (`risk:low` work)
- `needs:human` — requires human implementation
- `wontfix` — will not be actioned

## Principles

- **`:`-namespaced, color-grouped.** Every label carries a `prefix:` (except the GitHub default `wontfix`). Prefixes group visually and prevent collisions.
- **`risk:` is a LABEL, not a board field.** It is the one priority-like axis kept as a label (Ed's convention). Do **not** duplicate it as a board `Priority` field.
- **Work-kind = `type:` labels.** On an **org** repo, native **issue types** are also available and may later supersede `type:*`; until that migration is decided, `type:*` labels are canonical.
- **Status is the board field; Phase is a field/milestone.** Never labels.

## Before applying any label

Read the live taxonomy first — never assume — then reconcile toward the canon:

```bash
gh label list --repo OWNER/REPO --limit 200 --json name,color,description
```

The Phase-2 label-sync target converges any repo to the canon above (`gh label create --force` from a manifest), preserving existing `area:*` and archetype labels.

## Migration note

Older repos may still carry the pre-colon **hyphen** spellings of `needs:triage` and `needs:info` — rename those two to the colon canon:

```bash
gh label edit "OLD_HYPHEN_NAME" --repo OWNER/REPO --name "needs:triage"
```

The rest of the set (`needs:human`, `agent:*`, `risk:*`, `type:*`) is already colon in Recall.

## Adding a new label

Only when a repo genuinely needs a category the canon lacks, matching the `prefix:value` style:

```bash
gh label create "area:dashboard" --repo OWNER/REPO --color BFD4F2 --description "Dashboard subsystem"
```
