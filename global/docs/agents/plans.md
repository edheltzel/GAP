# Plans

## Two versions

- Every plan, spec, and regroup produces two files with the same stem.
- `<name>.md` is for agents: full detail, plus the todo.txt section. See [todo-txt.md](todo-txt.md).
- `<name>.human.md` is for the human: the same plan, short and visual.
- When a request names the overview, resolve it to its full sibling (drop the `.human` before the extension) and hand that path to the skill that builds or executes the plan.
- Project location rules still win. Write both files where the project keeps plans.
- Revise the agent version, then refresh the human version in the same change. A stale human version fails the test below.
- When a skill writes only the agent version, write the human version yourself right after it.
- The paired-file rule overrides a skill's single-output limit, but only for the companion files (the `.human.md` and the optional `.human.html`); never overwrite or delete an older regroup file; regroup never edits todo.txt or done.txt, and any open follow-up tasks go only to the project's todo.txt, through tuxedo, never to done.txt.
- Compare against the previous agent version, never the human version. Regroup diffs need the raw task lines.

## Human version

Put a diagram or table first. Then goal, now, next, and risks. It fits one screen.

Human plan test. All must pass before you hand it over:

- Visual first.
- Goal, now, and next read in about a minute.
- Small diagrams: 8 nodes or fewer, a flowchart or a table.
- No code, diffs, or long file lists.
- Matches the agent version: same steps, same status.

Phase flowchart:

```mermaid
flowchart LR
  A[Design: done] --> B[Build: active] --> C[Ship: next]
  classDef done fill:#cfc
  classDef active fill:#ffc
  class A done
  class B active
```

Status table:

```markdown
| Step   | Status | Note            |
| ------ | ------ | --------------- |
| Design | done   | approved        |
| Build  | active | 2 of 3 parts    |
| Ship   | next   | after the tests |
```

## Open visually

This previews the human version. Only when the human asks: "open this visually", "show me".

- A request to review or approve the plan in Plannotator is not a preview. Open the agent version in Plannotator and follow the plan skill's approval gate.
- Check the harness per [harness-detection.md](harness-detection.md).
- In Empryo, use the native `show` tool.
- Else, write a self-contained `<name>.human.html` from the human version, beside the human version, with the mermaid rendered, and open it with `open` (macOS) or `xdg-open`. `plannotator annotate` always blocks the turn until the review session ends, and in remote mode (`--tailscale` or `--cloudflare-access`) it prints a share URL that contains the overview, so a preview skips Plannotator and uses this HTML fallback; reserve Plannotator for a review the human is present for.
- An uncertain harness falls back to the portable path.

## Done

The human reads the plan in about a minute, and the agent version carries the full detail and the todo section.
