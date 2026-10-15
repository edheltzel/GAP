# Astro

Quickstart:

```bash
npx skills add edheltzel/Do-Skills --skill=do-astro
```

```bash
npx skills update do-astro
```

[Source](https://github.com/edheltzel/Do-Skills/tree/master/skills/project/frontend/do-astro)

## What it does

`astro` steers work in an Astro project — pages, components, CLI, adapters, and
project layout. The defining constraint is that [docs.astro.build](https://docs.astro.build)
is the API source of truth: this skill is a map of file locations, commands, and
workflows, not a copy of the docs.

## When to reach for it

Type `/do-astro`, or the agent reaches for it when you mention `.astro` files,
Astro config, SSG, islands, content collections, or deploying an Astro site.

Reach for it while building or deploying with Astro. For CSS techniques inside
those pages, use [modern-css](./modern-css.md). For reusable UI components and
tokens, use [design-system](./design-system.md). For a frontend review pass, use
[review-frontend](./review-frontend.md).

## The map it keeps

- **Where files go.** `src/pages` is required and is the router. Components,
  layouts, and styles are convention. `public/` is copied as-is.
- **CLI.** `dev`, `build`, `check`, `add`, `sync` — re-run after plugin changes.
- **Adapters.** `astro add` for Node, Cloudflare, Netlify, or Vercel, then
  `check` and `build` before deploy.

## It's working if

- New pages land under `src/pages/` and new components under `src/components/`.
- Config and API usage match current docs.astro.build, not remembered APIs.
- A deploy path runs `astro add`, `astro check`, and `astro build` in that order.

## Where it fits

A reach-for-it-anytime standalone for Astro apps, sitting with the other
frontend craft skills. Adapted from
[astrolicious/agent-skills](https://github.com/astrolicious/agent-skills) under MIT.
