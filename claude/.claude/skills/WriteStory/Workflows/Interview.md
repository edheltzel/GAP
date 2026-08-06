# Interview Workflow: The Layering Interview

Build a confirmed story spine from the writer's existing material. This workflow produces planning state, not prose.

## Preflight

1. Require the writer to identify a user-selected project directory.
2. Stop if the path is missing or ambiguous. Do not guess or create a location.
3. Ask which supplied notes, drafts, or reference files may be read.
4. If `interview.md` or `story-bible.md` already exists, read it in full before asking new questions.
5. Treat retained material outside the confirmed project as optional and read-only.

## Stance

- Read the writer's material before asking anything.
- Derive proposals from their seed and quote the source.
- Label each proposal as `inferred from your material` or `new option`.
- Offer two or three contrasting options when a layer is open.
- Treat the writer's redirect as authoritative.
- Do not silently fill a blank layer.

## Seed Classes

Classify the starting material as character, premise, plot, theme, world, or vibe/image. Select a derivation lens from `DerivationLenses.md` and explain why it fits.

## Interview Sequence

1. Reflect back what the supplied material already establishes.
2. Ask what most excites the writer and what must remain unchanged.
3. Establish the reader experience target and favorite reference works.
4. Derive a rough spine: theme question, dramatic question, hook, central pressure, climax, and three-act movement.
5. Review all seven layers and mark each `confirmed`, `partial`, or `open`.
6. Play the complete spine back for ratification.

## Continuity Artifact

With the confirmed project path, create or update `<project>/interview.md` using normal Markdown:

```markdown
# Story Interview: [Title]

## Source material read
- ...

## Seed and derivation lens
- Seed class: ...
- Lens: ...
- Why: ...

## Confirmed spine
- Theme question: ...
- Dramatic question: ...
- Hook: ...
- Central pressure: ...
- Climax: ...
- Act movement: ...

## Layer state
- Meaning: confirmed | partial | open
- Character change: confirmed | partial | open
- Plot: confirmed | partial | open
- Mystery: confirmed | partial | open
- World: confirmed | partial | open
- Relationships: confirmed | partial | open
- Prose: confirmed | partial | open

## Provenance and decisions
- ...

## Open questions
- [ ] ...
```

Read the existing file before updates. Preserve writer-authored text and edit only the sections the writer approved.

## Handoff

Offer these next steps:

- Deepen an open layer with Explore.
- Build or update `story-bible.md` with BuildBible.
- Draft a confirmed scene with WriteChapter.
