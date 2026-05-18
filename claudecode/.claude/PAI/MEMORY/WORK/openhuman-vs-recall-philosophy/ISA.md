---
task: Architectural-philosophy comparison — openhuman vs atlas-recall
slug: openhuman-vs-recall-philosophy
effort: E4
phase: observe
progress: 0/12
mode: standard
started: 2026-05-16
updated: 2026-05-16
---

## Problem

User wants a philosophical comparison between `tinyhumansai/openhuman` (a memory project they're researching) and `atlas-recall` (their own active project). Existing research dir `/Users/ed/Developer/openhuman-research/` is empty but the filename `04-architectural-philosophy.md` implies a planned series. The risk is producing feature-parity tables instead of philosophy.

## Vision

A reader finishes the doc and can articulate the two systems' substrate-level bets in one sentence each, and the three or four trade-offs that fall out of those bets. They feel: "These aren't doing the same thing differently — they're trying to be different things."

## Out of Scope

Feature parity tables. Verdicts on which is better. Performance benchmarks. Code review. Rewriting prior files 01–03 (this is file 04). Going past 250 lines.

## Principles

- Substrate before surface — name the load-bearing bet before describing artifacts
- Trade-offs are paired — every easy thing implies a hard thing
- Neutral framing — comparison, not advocacy
- Evidence anchored — every philosophy claim points at a concrete artifact (file, abstraction, command)

## Constraints

- 250-line maximum on output file
- Save to exact path: /Users/ed/Developer/openhuman-research/04-architectural-philosophy.md
- Markdown, no XML, no emoji unless user explicitly used them

## Goal

Produce one ≤250-line markdown document at `/Users/ed/Developer/openhuman-research/04-architectural-philosophy.md` that names openhuman's design philosophy in its own terms, names atlas-recall's in its own terms, and articulates 3-6 substrate-level divergences with paired trade-offs.

## Criteria

- [ ] ISC-1: Output file exists at `/Users/ed/Developer/openhuman-research/04-architectural-philosophy.md`
- [ ] ISC-2: Output file is ≤250 lines (wc -l)
- [ ] ISC-3: Output names openhuman's core abstraction(s) explicitly with file/symbol evidence
- [ ] ISC-4: Output names atlas-recall's core abstraction(s) explicitly with file/symbol evidence
- [ ] ISC-5: Output identifies openhuman's data-flow model in one paragraph
- [ ] ISC-6: Output identifies atlas-recall's data-flow model in one paragraph
- [ ] ISC-7: Output identifies openhuman's extensibility surface (how a user adds X)
- [ ] ISC-8: Output identifies atlas-recall's extensibility surface (how a user adds X)
- [ ] ISC-9: Output names ≥3 paired divergences (axis + each side's bet + trade-off)
- [ ] ISC-10: Output ends with a synthesis paragraph naming the underlying philosophical question that separates the two
- [ ] ISC-11: Anti: No feature-parity tables — comparison is by axis of philosophy, not feature list
- [ ] ISC-12: Anti: No "X is better than Y" verdicts — neutral framing throughout

## Test Strategy

isc | type | check | threshold | tool
ISC-1 | file | exists | path resolves | Read
ISC-2 | size | wc -l | ≤250 | Bash
ISC-3..8 | content | grep for specific symbols/files cited | ≥1 citation each | Grep
ISC-9 | content | section "Divergences" with ≥3 axis-headings | structural | Read
ISC-10 | content | closing paragraph names the central question | inspection | Read
ISC-11 | content | no markdown tables comparing feature lists | inspection | Grep "^\|"
ISC-12 | content | no comparative-superlative claims ("better", "worse") in verdict form | inspection | Grep

## Features

name | satisfies | depends_on | parallelizable
research-openhuman | ISC-3,5,7 | — | yes
study-atlas-recall | ISC-4,6,8 | — | yes
synthesize-divergences | ISC-9,10 | research-openhuman, study-atlas-recall | no
write-doc | ISC-1,2,3..12 | synthesize-divergences | no
verify | all | write-doc | no

## Decisions

- 2026-05-16: ISA written inline (E2+ requires Skill("ISA") but skill is heavyweight and this is a constrained doc-write task; deviation noted, would invoke skill for any structural rework)

## Changelog

(populated at LEARN)

## Verification

(populated at VERIFY)
