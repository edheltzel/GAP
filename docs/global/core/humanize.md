# Humanize

Quickstart:

```bash
npx skills add edheltzel/Do-Skills --skill=do-humanize
```

```bash
npx skills update do-humanize
```

[Source](https://github.com/edheltzel/Do-Skills/tree/master/skills/global/core/do-humanize)

## What it does

`do-humanize` scans developer text and can rewrite the sentences that fail a plain-prose test. Default mode writes `ai-writing-review.json` and does not rewrite files. `fix` mode consumes that report: safe findings are applied automatically, needs-review findings wait for confirmation.

It judges the text, not who wrote it. A finding names the failed test. Evidence comes before a flag: every finding cites file:line and passes the bundled verification protocol. `--dry-run` previews edits. The skill will not stash. If `git status --porcelain` shows anything other than untracked `ai-writing-review.json`, fix mode stops.

## When to reach for it

You invoke this by typing `/do-humanize`. The agent won't reach for it on its own.

Reach for review when prose is inflated, filler, or hard to act on, including when it sounds like ChatGPT: `/do-humanize [--all] [--category <name>] [path]`. Reach for fix after that JSON exists: `/do-humanize fix [--dry-run]`. The pass names the failed test. It does not say who wrote the text.

## Prerequisites

Fix mode needs `ai-writing-review.json` from a review run in this skill, or `--all` so fix mode runs review itself first. A dirty worktree other than that untracked report stops the edit pass.

## Review, then fix

The skill partitions files into prose, code docs, and git, then checks six pattern categories (content, vocabulary, formatting, communication, filler, code docs) plus the plain-prose tests in the skill: portability, actor, one read, decision first, reader, and mechanism. Safe vs needs-review is classified up front so fix mode can auto-apply only the mechanical catalog fixes.

**Safe** means mechanical: delete a chat leak, swap a listed filler phrase, drop an emoji. **Needs review** means a rewrite that could change meaning, including every plain-prose miss. Those rewrites keep every fact, number, name, quote, and citation the source stated. Git artifacts (commits, PRs) are never auto-fixed.

## It's working if

- Review: `ai-writing-review.json` parses and `git_head` matches `HEAD`, or the run exited with no files to scan. Each finding has file:line, a category, and a fix-safety label. False positives listed in the skill (intentional formality, licenses, generated code) are not flagged.
- Fix: `--dry-run` lists the same partitions and does not stash. A normal review-then-fix run leaves `ai-writing-review.json` readable after preflight (G1 allows that untracked file and nothing else). Applied files pass the skill's per-type validation. Failures are checked out and not listed as OK. Success deletes `ai-writing-review.json`. A failed validation keeps that file.

## Where it fits

A user-invoked detection and rewrite pass. Review feeds fix inside the same skill. [tech-writing](../workflow/tech-writing.md) is the voice to write with in the first place. A README that drifted from the code belongs to [readme-update](../workflow/readme-update.md). This skill audits and cleans text that already landed.
