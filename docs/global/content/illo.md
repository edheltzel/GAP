# Illo

Quickstart:

```bash
npx skills add edheltzel/Do-Skills --skill=do-illo
```

```bash
npx skills update do-illo
```

[Source](https://github.com/edheltzel/Do-Skills/tree/master/skills/global/content/do-illo)

Upstream: [tmchow/illo-skill](https://github.com/tmchow/illo-skill)

## What it does

`do-illo` turns an idea or article into a print-style editorial illustration. A recurring mascot performs the idea in one scene, a mini-comic, an explainer diagram, or a transparent cutout.

It is a house style, not a generic image generator. It fires only when you invoke it or ask for "illo", not on ordinary illustrate or draw requests.

## When to reach for it

Type `/do-illo` or say "illo". Use it for article illustrations, one-concept scenes, surprise or random art, mini-comics, explainer diagrams, or character cutouts.

For general static visuals that are not this print-mascot style, use [art](./art.md).

## Prerequisites

Needs `python3` and one image backend: Codex CLI, Grok CLI, Grok Bot's image tool, or an OpenRouter key via `scripts/illo.py init`. Config lives in `~/.config/illo/`.

## Where it fits

`do-illo` sits next to `do-art` in Content. `do-art` covers many formats and models. `do-illo` is Trevin Chow's editorial mascot pipeline, with its own engine and character packs.
