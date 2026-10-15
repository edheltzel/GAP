# Design Systems

Quickstart:

```bash
npx skills add edheltzel/Do-Skills --skill=do-design-system
```

```bash
npx skills update do-design-system
```

[Source](https://github.com/edheltzel/Do-Skills/tree/master/skills/project/frontend/do-design-system)

## What it does

`design-system` covers two jobs in one skill. Day to day it steers UI
components toward being accessible, themeable, composable, and visually
consistent — component APIs, two-layer tokens, WCAG-grade accessibility,
theming, and interaction patterns. When asked, it also captures the project's
existing brand into a portable `DESIGN.md` plus a live style-guide page.
Accessibility and the token scale are baselines, not features: an interactive
component that isn't keyboard-operable, or a value that isn't on the scale, is
wrong. The spec loop discovers tokens; it does not invent missing ones.

## When to reach for it

Type `/do-design-system`, or the agent reaches for it when building buttons,
dialogs, forms, cards, or any reusable component (it triggers on `.tsx`,
`.jsx`, `.css`, and `.scss`).

Reach for it to build or review components — props, theming, keyboard and
screen-reader behavior. Pass `route:`/`mood:`/`name:`/`framework:` or ask for
`DESIGN.md` / a style guide to run the spec loop instead. That loop does not
run just because a `.tsx` file is open. For raw CSS techniques, use
[modern-css](./modern-css.md).

## The patterns it enforces

- **Headless + styled layers.** Behaviour from accessible primitives (Radix, Ark,
  Headless UI); appearance in your own styling layer on top.
- **Compound components over prop soup.** `<Dialog.Root><Dialog.Trigger>…` with
  named parts, not a monolith configured by a dozen booleans. Variants are
  discriminated unions, never boolean flags.
- **Two token layers.** Primitive tokens hold raw values; semantic tokens assign
  meaning and are what components consume. OKLCH over HSL for perceptual
  uniformity and P3 gamut. No third "component token" layer unless white-labeling.
- **Accessibility baseline.** `:focus-visible` rings, focus trap and restore on
  overlays, semantic HTML before ARIA, WCAG AA contrast, `prefers-reduced-motion`.

## It's working if

- Components expose composable named parts and union-typed variants, not booleans.
- Every color, space, and radius comes from the token scale — no arbitrary values.
- Interactive elements are `<button>`/`<a>`, operable by keyboard, with visible
  focus and correct focus handling on open/close.
- A spec run writes `DESIGN.md` from discovered tokens only, plus a style-guide
  page that dogfoods project primitives.

## Where it fits

A reach-for-it-anytime standalone for component work that sits above
[modern-css](./modern-css.md) and pairs with
[write-typescript](../typescript/write-typescript.md) for prop types like
discriminated unions. The spec loop is the same skill, not a second install.
