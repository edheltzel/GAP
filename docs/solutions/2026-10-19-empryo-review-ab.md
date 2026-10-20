---
title: Empryo /review against do-adversarial-review, seeded A/B
date: 2026-10-19
module: do-adversarial-review
problem_type: tool_choice
tags:
  - empryo
  - review
  - ab-test
---

# Empryo /review against do-adversarial-review

Decision: inside Empryo, `do-adversarial-review` runs its hunt on `empryo --headless --review` ([empryo-review.md](https://github.com/edheltzel/Do-Skills/blob/78e2c15bfbc484d74a2d38ef3fa727a99272bf25/skills/global/core/do-adversarial-review/references/empryo-review.md)). Outside Empryo nothing changed.

## Setup

Scratch repos under `/tmp/review-ab` (outside GAP), each with a base commit and one uncommitted diff:

| Diff | Seeded bug | Decoy (looks wrong, is guarded) |
|---|---|---|
| d1 | `users.js:8` `users.get(id).name` with no guard; `greet(2)` throws | `config.js:12` `config.timeout.ms`, guarded by `loadConfig`'s check |
| d2 | `page.js:10` `i <= end` where `end` is exclusive | `page.js:18` `slice(0, lastIndex + 1)`, inclusive by contract |
| d3 | `store.js:10-13` `save` logs and still returns `{ ok: true }` | `store.js:16-22` `cleanup` ignores only `ENOENT`, rethrows the rest |

Both arms ran headless in a fresh context, so neither saw the seeds. The in-session run would have been graded by the agent that wrote them.

- Skill: `empryo --headless --skill do-adversarial-review --json --cwd <repo> "Run the do-adversarial-review skill on the uncommitted git diff in this repo. Report only, do not edit."` Model `subscriptions/claude-opus-5-5`.
- Empryo: `empryo --headless --review "<brief>" --json --cwd <repo>`, brief as in `references/empryo-review.md`. Model `subscriptions/gpt-6.1-sol` (`goalReview` fallback).

All six ran in parallel from inside a running Empryo tab, so the nested headless case completes.

## Results

| Diff | Arm | Seeded bug | Decoy flagged | Other findings | Wall (s) | Cost (USD) |
|---|---|---|---|---|---|---|
| d1 | skill | yes, P1 | no | 2: `loadConfig("null")` throws TypeError (real edge); `loadConfig("{}")` now throws (intended, noise) | 30 | 0.645 |
| d1 | review | yes, P2 | no | 0 | 20 | 0.035 |
| d2 | skill | yes, P0 | no | 0 | 18 | 0.611 |
| d2 | review | yes, P2 | no | 0 | 20 | 0.019 |
| d3 | skill | yes, P1 | no | 0 | 17 | 0.608 |
| d3 | review | yes, P1 | no | 0 | 17 | 0.018 |

Totals: skill 3/3 seeded, 0 decoys, 1 noise finding, $1.86. Review 3/3 seeded, 0 decoys, 0 noise, $0.072. Wall is close; both run startup.

Rule from the plan: Empryo wins if it finds at least as many true bugs with no more false positives. It did, so the skill swapped.

## Follow-up checks

- d1 again through the skill's own engine command: FAIL, exit 3, 18 s, $0.014. It found `users.js:8` and this time also the `loadConfig("null")` edge. Run-to-run variance is real, so the skill keeps its self-verification and re-grades severity.
- A clean repo: exit 0, `verdict: null`, "Nothing to review", $0. The skill treats that as not reviewed and hunts in-skill. This is also what a GitButler lane with committed changes returns.

## Caveats

- The reviewer grades severity low (null deref and off-by-one both P2). The skill re-grades with its own P0-P3 definitions.
- The skill arm executed the code with `node` to confirm triggers; the reviewer stayed static. Neither ran tests (none existed).
- Three diffs is a small sample. Re-run this A/B if the `review` router slot changes model.
