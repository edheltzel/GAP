---
name: Humanizer
description: Remove signs of AI-generated writing from text. USE WHEN humanize, edit AI text, remove AI patterns, make text sound human, AI writing cleanup, deslop, sound natural.
---

# Humanizer

Identifies and removes signs of AI-generated text to make writing sound more natural and human. Based on Wikipedia's "Signs of AI writing" guide (WikiProject AI Cleanup). Covers 24 pattern categories including inflated symbolism, promotional language, AI vocabulary words, em dash overuse, rule of three, and sycophantic tone.

## Customization

**Before executing, check for user customizations at:**
`~/.claude/skills/PAI/USER/SKILLCUSTOMIZATIONS/Humanizer/`

If this directory exists, load and apply any PREFERENCES.md, configurations, or resources found there. These override default behavior. If the directory does not exist, proceed with skill defaults.

## Workflow Routing

| Workflow | Trigger | File |
|----------|---------|------|
| **Humanize** | "humanize", "remove AI patterns", "make this sound human", "deslop" | `Workflows/Humanize.md` |

## Quick Reference: 24 AI Writing Pattern Categories

**Content:** Inflated significance, undue notability, superficial -ing analyses, promotional language, vague attributions, challenges/prospects sections
**Language:** AI vocabulary words, copula avoidance, negative parallelisms, rule of three, elegant variation, false ranges
**Style:** Em dash overuse, boldface overuse, inline-header lists, title case headings, emojis, curly quotes
**Communication:** Collaborative artifacts, knowledge-cutoff disclaimers, sycophantic tone
**Filler:** Filler phrases, excessive hedging, generic positive conclusions

**Full pattern reference:** Read `Patterns.md`
**Soul and personality guide:** Read `SoulGuide.md`

## Examples

**Example 1: Humanize a draft**
```
User: "Humanize this blog post" [pastes text]
-> Invokes Humanize workflow
-> Loads Patterns.md for pattern detection
-> Identifies AI patterns in the text
-> Rewrites with natural voice and specific details
-> Returns humanized version with change summary
```

**Example 2: Edit a file in place**
```
User: "Make src/docs/intro.md sound less AI-generated"
-> Invokes Humanize workflow
-> Reads the file
-> Applies pattern detection and rewriting
-> Edits the file with humanized content
```

## Reference

Based on [Wikipedia:Signs of AI writing](https://en.wikipedia.org/wiki/Wikipedia:Signs_of_AI_writing), maintained by WikiProject AI Cleanup.

> "LLMs use statistical algorithms to guess what should come next. The result tends toward the most statistically likely result that applies to the widest variety of cases."
