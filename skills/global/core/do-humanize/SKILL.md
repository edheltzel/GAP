---
name: do-humanize
description: "Rewrite developer text so a sentence states a fact about this project, not a feeling. Use when docs, docstrings, commits, PRs, or comments are inflated, filler, or hard to act on. NOT FOR a README that drifted from the code (`do-readme-update`, if installed)."
disable-model-invocation: true
user-invocable: true
---

# Humanize

Two modes. Default is **review**: scan developer text and write `ai-writing-review.json`. **fix** consumes that file and applies classified edits. Judge the text. Do not say who wrote it. Builds on [docs-style](references/docs-style.md). The compass copy in [diataxis-compass](references/diataxis-compass.md) stays in this skill.

## Usage

- Review (default): `/do-humanize [--all] [--category <name>] [path]`
- Fix: `/do-humanize fix [--dry-run] [--all] [--category <name>]`

**Flags:**

- `--all` - Scan or fix the whole codebase (review default: changed files from main)
- `--category <name>` - `content|vocabulary|formatting|communication|filler|code_docs`
- `--dry-run` - Fix mode only. Preview without changing files
- Path - Review mode target directory (default: current working directory)

If fix mode has no review file and `--all` is set, run review mode in this skill first. Do not look for a separate review skill.

## Which file to load

| Mode | Load |
|------|------|
| review (default, no `fix` argument) | [references/review.md](references/review.md), then the pattern files it names |
| fix (`fix` is the first argument) | [references/fix.md](references/fix.md), then the strategy files it names |

Read [review-verification-protocol](references/review-verification-protocol.md) before reporting findings.

## Pattern references

Load only the categories in scope:

- [content-patterns](references/content-patterns.md)
- [vocabulary-patterns](references/vocabulary-patterns.md)
- [formatting-patterns](references/formatting-patterns.md)
- [communication-patterns](references/communication-patterns.md)
- [filler-patterns](references/filler-patterns.md)
- [code-docs-patterns](references/code-docs-patterns.md)
- [vocabulary-swaps](references/vocabulary-swaps.md)
- [fix-strategies](references/fix-strategies.md)
- [developer-voice](references/developer-voice.md)
- [diataxis-compass](references/diataxis-compass.md)

## Rules

- Default invocation is review. `fix` is the first argument.
- Review writes `ai-writing-review.json` and does not rewrite files.
- Fix does not edit until that JSON exists and passes the gates in [fix.md](references/fix.md).

## Plain prose

Apply these on every sentence, in review and in fix. The pattern files already cover density (a cluster of tells, not one word) and promotional wording. These tests are the ones those files do not state. A finding names the failed test. It does not name an author.

- **Portability.** Could the sentence move to another project unchanged? It carries no fact about this one. Put a fact in, or cut it.
- **Actor.** Who does the verb? Name them when the source says who. Keep the passive when the actor is unknown or naming them adds nothing.
- **One read.** Would the reader have to reread the sentence? Split it. A sentence that lands on the first pass stays, however long.
- **Decision first.** The first sentence carries the outcome the reader needs.
- **Reader.** Can someone who has not opened the code act on this? Gloss the identifier, or name the consequence.
- **Mechanism.** Does the sentence say what the thing does, or how it feels? Put the fact back, or cut the sentence.

Keep every fact, number, name, quote, and citation. Do not add a fact the source did not state. Leave code, quotes, frontmatter, link targets, and identifiers alone unless the user named that text.

Review records these as category `content`, types `portability`, `actor`, `one_read`, `decision_first`, `reader`, `mechanism`, fix safety Needs review. Fix keeps the source facts while applying them.
