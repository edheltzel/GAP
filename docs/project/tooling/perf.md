# Perf

Quickstart:

```bash
npx skills add edheltzel/Do-Skills --skill=do-perf
```

```bash
npx skills update do-perf
```

[Source](https://github.com/edheltzel/Do-Skills/tree/master/skills/project/tooling/do-perf)

## What it does

`perf` turns "make the app fast" into a number you can defend. It runs the
**full** loop: measure a production Electron or web build with a Playwright/CDP
A/B harness, find the cause (Blink style-invalidation trace, React commit
probe), fix what the trace named, remeasure the same way, and generate an HTML
report whose every figure comes from harness JSON.

The defining constraint is that the report cannot get ahead of the data: no
typed numbers, no skipped steps, no flame-graph guess in place of a named
selector. A lightweight render-cost note such as `lite-render-perf` is a
different, thinner thing — this skill is the complete audit with scripts.

## When to reach for it

Type `/do-perf`, or the agent reaches for it when you ask to profile, speed up,
slim down, find memory leaks, benchmark before and after, or produce a
performance report.

Reach for it when the claim has to survive review. For CSS technique choice
while fixing a named rule, use [modern-css](../frontend/modern-css.md). For component
tokens after the measurement, use [design-system](../frontend/design-system.md) — it is
not a required step of this loop.

## Prerequisites

The scripts import `playwright-core`, so install it in the folder you run them
from (`npm i -D playwright-core`). A web target also needs a browser binary
(`npx playwright-core install chromium`); an Electron target launches the
app's own Electron build. You also need a production build of the app to
point the harness at.

## The loop

1. **Measure.** Copy `scripts/profile.example.mjs`, name the phases people
   actually perform, and run `perf-ab.mjs` against the real production build.
2. **Find.** Run `style-trace.mjs` on the worst phase. Fix the selector or
   component the trace names, not a nearby tidy-up.
3. **Fix, then remeasure.** Rebuild from a purged output directory. Run the
   same harness with `--label=after`.
4. **Report.** `perf-report.mjs` interpolates both JSON files into one
   self-contained HTML page. Regressions print as loudly as wins.

Read `reference/traps.md` before the first run. Bundle size is walked from the
module graph, not summed from the output directory. The leak cycle must return
the app to the state it started in.

## It's working if

- Before and after JSON share the same harness, `platform`, and `repeats`.
- Every figure in the HTML is interpolated from those files.
- The leak series for DOM nodes is flat; if it is not, the heap number is
  ignored.
- Changes that did not pay (including reverted bundler grouping) are listed at
  `file:line`.

## Where it fits

A reach-for-it-anytime performance measurement tool. It sits beside
[review-frontend](../frontend/review-frontend.md) (pre-PR review) and
[cleanup-web](../frontend/cleanup-web.md) (end-of-session polish); those do not produce
an A/B harness. Adapted from
[proxysoul/SoulStack](https://github.com/proxysoul/SoulStack) `skills/enliven`
under MIT.
