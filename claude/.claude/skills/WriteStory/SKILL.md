---
name: WriteStory
version: 1.2.0
description: "Scaffolding that helps a writer build a story they already want to tell. It derives structure, character change, theme, and prose choices from the writer's own material across seven narrative layers. USE WHEN write story, fiction, novel, chapter, story bible, character arc, plot outline, creative writing, worldbuilding, draft story, help me write my story, develop my novel, layer my story, build out my book, I have notes for a story. NOT FOR narrative summaries of real content."
effort: high
---

# WriteStory

## What It Does

WriteStory helps a writer develop fiction without taking ownership of the story. It reads the writer's material first, derives proposals from that material, labels new options clearly, and lets the writer ratify every structural choice.

## Project Path Contract

Continuity lives only in a **user-selected project directory**. The two project-local artifacts are:

- `interview.md`: confirmed interview results, provenance, open questions, and the rough spine.
- `story-bible.md`: the living story plan, beat map, continuity notes, and normal Markdown checklists.

Never guess or create a project directory. Before reading or writing project state:

1. Ask the user to identify or confirm the project directory.
2. Confirm whether the requested input files exist.
3. Read every existing target file in full before proposing an update.
4. Preserve writer-authored material and distinguish it from new proposals.
5. Stop when the project path or required input is missing.

Do not create either artifact until the user confirms the path and asks for the corresponding workflow. Existing retained user material outside the selected project is optional and read-only. Never relocate, rewrite, or depend on it.

## The Seven Layers

1. **Meaning:** Theme, argument, and the question the story asks.
2. **Character change:** Flaw or misbelief, pressure, crisis, and transformation.
3. **Plot:** The causal chain of events.
4. **Mystery:** What the reader knows, when, and why.
5. **World:** Setting and rules that pressure the characters.
6. **Relationships:** How key bonds evolve and constrain choices.
7. **Prose:** Voice, aesthetic, rhetoric, and sentence-level craft.

## Workflow Routing

| Workflow | Trigger | File |
| ---------- | --------- | ------ |
| Interview | "interview me", "I have notes", "help me plan a story" | `Workflows/Interview.md` |
| BuildBible | "build story bible", "create story plan", "map the story" | `Workflows/BuildBible.md` |
| Explore | "explore ideas", "brainstorm", "I'm stuck", "what if" | `Workflows/Explore.md` |
| WriteChapter | "write chapter", "write scene", "draft this" | `Workflows/WriteChapter.md` |
| Revise | "revise", "edit", "improve", "polish" | `Workflows/Revise.md` |

The usual path is Interview, BuildBible, WriteChapter, then Revise. Explore is available when a layer is genuinely open.

## The Stance

- Read the writer's material before asking questions.
- Never invent the spine. Derive from the writer's seed.
- Label every proposal as inferred from supplied material or newly offered.
- Offer options, not a single answer disguised as a choice.
- Preserve the writer's voice, words, and confirmed decisions.
- Only WriteChapter emits prose, and only from confirmed inputs.
- Read `story-bible.md` before continuity-sensitive writing or revision.
- Never overwrite an existing artifact wholesale. Update only the agreed sections.

## References

| Reference | File |
| ----------- | ------ |
| Layer architecture | `StoryLayers.md` |
| Character construction | `StorrFramework.md` |
| Concept and spine | `PressfieldFramework.md` |
| Derivation lenses | `DerivationLenses.md` |
| Structure and beats | `PhasesAndEvents.md`, `StoryStructures.md` |
| Prose craft | `RhetoricalFigures.md`, `AestheticProfiles.md` |
| Freshness checks | `AntiCliche.md` |
| Review profiles | `Critics.md` |
