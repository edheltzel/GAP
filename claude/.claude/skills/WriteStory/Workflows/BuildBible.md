# BuildBible Workflow

Construct or update `story-bible.md`, the project-local continuity source for the story.

## Preflight

1. Require a user-confirmed project directory.
2. Require `<project>/interview.md` or equivalent writer-supplied planning input.
3. Stop if the path or required input is missing. Do not invent a location or story state.
4. Read `interview.md` in full.
5. If `story-bible.md` exists, read it in full before changing anything.
6. Treat other retained user material as optional and read-only.

## Procedure

### 1. Confirm Story Parameters

Confirm scope, genre, aesthetic profile, point of view, tense, concept, theme question, hook, opposition, and intended reader experience. Label any unresolved element as open.

### 2. Build Character Architecture

For each major character, record flaw or misbelief, origin wound, external want, internal need, thematic purpose, crisis choice, arc direction, status dynamics, and mystery hooks.

### 3. Map the Plot

Use `PhasesAndEvents.md` and `StoryStructures.md` as optional scaffolding. For every major beat, record the event, its cause, its consequence, the character choice that drives it, and the layers it advances.

### 4. Design Mystery and Information Flow

Record the primary question, clue plants, red herrings, reveal sequence, resolution, and the reader's expected belief at each stage.

### 5. Build World and Relationship Constraints

Include only world details that affect character behavior, conflict, plot, or meaning. Map each key relationship from initial state through pressure, crisis, and resolution.

### 6. Define Prose Strategy

Record voice, sentence patterns, dialogue distinctions, description density, rhetorical figure preferences, and explicit anti-cliche constraints.

### 7. Build the Beat Map

Map all seven layers for every major beat: meaning, character change, plot, mystery, world, relationships, and prose.

### 8. Write or Update the Artifact

Write to `<project>/story-bible.md`. Read the existing file in full before updating it. Preserve writer material verbatim unless the writer approves a revision. Add proposals in clearly labeled blocks rather than blending them into confirmed material.

Use normal Markdown sections and checklists:

```markdown
# Story Bible: [Title]

## Continuity status
- Current phase: ...
- Last confirmed change: ...
- Next agreed action: ...

## Confirmed spine
- ...

## Characters
- ...

## Beat map
### Beat 1: ...
- Meaning: ...
- Character change: ...
- Plot: ...
- Mystery: ...
- World: ...
- Relationships: ...
- Prose: ...

## Mystery architecture
- ...

## World framework
- ...

## Relationship arcs
- ...

## Prose strategy
- ...

## Story checklist
- [ ] C-01: [Observable story requirement]
- [ ] C-02: [Observable continuity requirement]

## Open questions
- [ ] ...

## Change log
- [Date]: [Writer-approved change and source]
```

Checklist items are ordinary project-local Markdown. Keep IDs stable when wording changes and note the reason in the change log.

## Series Scale

For a series, keep one `story-bible.md` in the confirmed series directory with a series section and separate book sections. Do not create child directories or additional files without explicit path confirmation.

## Output

Present a summary of mapped sections, preserved material, new proposals, remaining gaps, and the next writer decision. Do not claim completion for open or unratified material.
