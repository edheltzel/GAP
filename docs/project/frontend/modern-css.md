# Modern CSS

Quickstart:

```bash
npx skills add edheltzel/Do-Skills --skill=do-modern-css
```

```bash
npx skills update do-modern-css
```

[Source](https://github.com/edheltzel/Do-Skills/tree/master/skills/project/frontend/do-modern-css)

## What it does

`modern-css` teaches an agent to write CSS with the native features the platform
now ships — `place-items`, `aspect-ratio`, `:has()`, container queries, `oklch()`,
`<dialog>`, scroll snap — instead of the legacy hacks and JavaScript they used to
require. It carries 64 old-vs-modern comparisons sourced from
[modern-css.com](https://modern-css.com/), each showing the hack it retires. The
defining constraint is browser support: every technique is filed under a support
tier, and the agent's freedom to use it depends on that tier, not on whether it's
newer or cleaner.

## When to reach for it

Type `/do-modern-css`, or the agent reaches for it automatically when writing or
reviewing CSS.

Reach for it whenever you're authoring stylesheets and want the current idiom
rather than a 2015 one — and especially when you spot a JS scroll listener, a
padding-top aspect-ratio, or a transform-centering trick that a single modern
property would replace. For assembling those techniques into accessible,
themeable components, use [design-system](./design-system.md); for a
whole-session tidy that includes a CSS lens, use [cleanup-web](./cleanup-web.md).

## The three support tiers

The skill's whole behaviour turns on where a feature sits:

- **Widely Available (90%+).** Use without asking. If you find the legacy pattern
  in existing code, refactor it to the modern equivalent.
- **Newly Available (80–90%).** Suggest it, name the support level, offer a
  fallback. The user decides.
- **Limited (<80%).** Ask first. Warn about support and propose a
  progressive-enhancement path or `@supports` gate.

Legacy-browser requirements override all of it — if the project must support old
Safari or locked-down corporate browsers, the agent asks before using anything,
even a Widely Available feature, and reaches for `@supports` fallbacks.

## Guiding principles

Underneath the catalogue sit a handful of defaults: native CSS over JavaScript,
logical properties over physical `left`/`right`, intrinsic sizing (`clamp()`,
container queries) over fixed breakpoints, custom properties over preprocessor
variables, and `@layer` over `!important` specificity wars.

## Where it fits

A reach-for-it-anytime standalone for day-to-day styling, and the CSS foundation
the UI skills build on — [design-system](./design-system.md) leans on it for
tokens, theming, and `DESIGN.md`, and [cleanup-web](./cleanup-web.md) runs it as
one review lens on a session's changes.
