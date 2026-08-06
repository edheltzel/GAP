# Explore Workflow

Generate fresh options for an open story layer while preserving confirmed continuity and writer ownership.

## Preflight

1. Require the writer to confirm the project directory.
2. Stop if the path is missing or ambiguous.
3. If `story-bible.md` exists, read it in full before proposing ideas.
4. Read any writer-supplied source material named for this exploration.
5. Treat retained material outside the confirmed project as optional and read-only.
6. Do not update project files until the writer chooses a direction.

## Define the Exploration Target

Confirm:

- the open layer: meaning, character change, plot, mystery, world, relationships, or prose;
- fixed constraints from `story-bible.md`;
- the desired range from conservative to radical;
- references the writer wants used or avoided.

## Generate Options

Use at least three distinct approaches when useful:

1. **Combinatorial:** Combine compatible elements from the writer's references.
2. **Constraint reversal:** Invert the expected choice while preserving fixed facts.
3. **Cross-genre:** Borrow a technique from another genre or medium.
4. **Wild card:** Offer one higher-risk option that still serves the confirmed story.

Every option must state whether it is inferred from supplied material or newly offered.

## Freshness Filter

Read `AntiCliche.md`. Reject or revise an option when it is only a familiar trope, contradicts confirmed continuity, or ignores the writer's stated aesthetic.

## Output

Return each option with:

```markdown
### Option [N]: [Name]
- Source: inferred from supplied material | new option
- Idea: ...
- Why it fits: ...
- Continuity impact: ...
- Risk: ...
- Freshness: ...
```

Include a short comparison and ask the writer which direction, if any, they want to adopt.

## Integrate a Chosen Direction

After explicit selection:

1. Re-read `story-bible.md` before editing.
2. Update only the affected sections.
3. Preserve all unrelated writer material.
4. Update ordinary checklist items and ripple effects where needed.
5. Add a change-log entry naming the writer-approved decision.

If the writer selects nothing, make no file changes.
