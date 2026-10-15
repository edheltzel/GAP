---
name: do-skill-craft
description: >-
  Authors Agent Skills: the vocabulary and principles that make a skill predictable, distillation of a blog post, article, documentation, GitHub repo, video transcript, book, or paper into a reusable skill, and review of PRs that add or modify Agent Skills for structural validity, design quality, and marketplace consistency. Use when you want to write/edit a skill, tune a skill description, make a skill from this, distill into a skill, distill this into a skill, turn this repo into a skill, turn this repo/article into a skill, create a skill from this article, extract patterns from, convert to a skill, review skill PR, audit SKILL.md, review skill file changes, or run an automated skill PR review.
---

# Skill Craft

Author Agent Skills. Default mode is `write`. Review mode runs only on explicit request. Do not start a review because a skill file is open or a diff looks skill-related.

Bold terms in the writing principles are defined in [references/GLOSSARY.md](references/GLOSSARY.md). Load it when a term needs the full meaning.

## write (default)

Writing or editing a skill, including the skill description: invocation, information hierarchy, when to split, pruning, leading words, failure modes.

Load [references/writing-principles.md](references/writing-principles.md).

## distill

A URL, article, repo, transcript, book, paper, or other source should become a skill ("make a skill from this", "distill into a skill", "turn this repo/article into a skill").

Load [references/distill.md](references/distill.md). It applies the writing principles. Concrete before/after examples: [references/distill-examples.md](references/distill-examples.md). Packaging mechanics: `skill-creator` if installed.

## review

Only when the user runs `/do-skill-craft review [output-path] [--base <branch>]`.

Load [references/review.md](references/review.md) and the checks it names:

- [references/structural-checks.md](references/structural-checks.md)
- [references/design-checks.md](references/design-checks.md)
- [references/marketplace-checks.md](references/marketplace-checks.md)
- [references/review-verification-protocol.md](references/review-verification-protocol.md)

Apply [references/writing-principles.md](references/writing-principles.md) when judging design.

Provenance: write-mode principles and glossary are MIT (Matt Pocock, LICENSE-mattpocock), adapted from do-writing-great-skills; review-mode checks are Apache-2.0 (Beagle, LICENSE), adapted from do-review-skill.
