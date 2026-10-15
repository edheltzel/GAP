# Technical Writing

Quickstart:

```bash
npx skills add edheltzel/Do-Skills --skill=do-tech-writing
```

```bash
npx skills update do-tech-writing
```

[Source](https://github.com/edheltzel/Do-Skills/tree/master/skills/global/workflow/do-tech-writing)

## What it does

`tech-writing` steers written output — commits, issues, PRDs, specs, PR
descriptions, comments — toward prose that is clean, terse, and reads the same to
a human and a machine. The defining constraint is that every sentence earns its
place or gets cut: it leads with the point, allows one idea per unit, and deletes
filler and hedging rather than softening it. It's a style, not a template
factory; the formats it ships are scaffolds you trim, not sections you dutifully
fill.

## When to reach for it

Type `/do-tech-writing`, or the agent reaches for it automatically when writing a
commit message, issue, PRD, spec, or PR description.

Reach for it whenever you're producing technical prose and want it short and
direct the first time. For the narrower question of what belongs in a code
comment, use [code-comments](../core/code-comments.md); for the specific files
[agent-docs](../core/agent-docs.md) and [architecture-md](../core/architecture-md.md)
govern, use those — this skill sets the writing posture they all share.

## The rules that do the work

- **Lead with the point.** First sentence is the takeaway; context comes after,
  if at all. Inverting the "wall of context" is the single highest-leverage move.
- **One idea per unit.** One point per sentence, one topic per paragraph, one
  concern per section.
- **Active voice, imperative mood.** "Add retry logic", not "Retry logic should
  be added" or "Added retry logic".
- **Cut filler and hedging.** Delete "basically", "simply", "just", "in order
  to". Say "this will break X", not "this could potentially impact X". If removing
  a word doesn't change the meaning, remove it.
- **Requirements are testable.** "Fast" is not a requirement; "loads in < 200ms at
  p95" is. Success criteria are measurable; open questions have owners.

## It's working if

- The takeaway is in the first sentence, not the fourth paragraph.
- Commit subjects are imperative, under 72 chars, and the body says *why* not
  *what*.
- There are no "Risks: None" filler sections and no weasel words ("some users",
  "significant impact") left to quantify.

## Where it fits

A reach-for-it-anytime standalone for any technical prose, and the shared voice
behind the documentation skills: [code-comments](../core/code-comments.md),
[agent-docs](../core/agent-docs.md), and
[architecture-md](../core/architecture-md.md) each format one artifact, while this
skill governs how the words in all of them should read.
