# Art

Quickstart:

```bash
npx skills add edheltzel/Do-Skills --skill=do-art
```

```bash
npx skills update do-art
```

[Source](https://github.com/edheltzel/Do-Skills/tree/master/skills/global/content/do-art)

## What it does

`do-art` turns an audience-facing idea into a static visual: an editorial header,
diagram, infographic, chart, comic, icon, wallpaper, or another presentation
asset. Content here means media made for people to see and understand, rather
than the interface components people use to operate software.

It begins with a named workflow, not a freeform image prompt. That workflow
selects the composition and generation approach, and its output stages in a
preview folder (`$ART_OUTPUT_DIR`, else `~/Downloads` when it exists, else
`./art-output`) for review before it is used in a project.

## When to reach for it

Type `/do-art`, or the agent reaches for it automatically when a request calls for
an illustration, diagram, flowchart, infographic, social image, or other static
visual.

Reach for it when the deliverable is an audience-facing image or visual
explanation. For reusable product UI, component tokens, interaction states, and
accessibility, use [design-system](../../project/frontend/design-system.md) instead.

## Prerequisites

At least one image API key. `do-art` generates with three providers, plus
OpenRouter as a gateway to all of them, and uses whichever have a key:

| Provider | Models | Key |
| --- | --- | --- |
| xAI | `grok` (Grok Imagine) | `XAI_API_KEY` |
| OpenAI | `gpt-image-2` | `OPENAI_API_KEY` |
| Google Gemini | `nano-banana`, `nano-banana-pro` | `GEMINI_API_KEY` or `GOOGLE_API_KEY` |
| OpenRouter | any of the above, when its own key is missing | `OPENROUTER_API_KEY` |

Keys load from the environment, then `./.env`, then
`${XDG_CONFIG_HOME:-~/.config}/do-art/.env`. Run
`bun <skill-dir>/Tools/Generate.ts --check-keys` to see what it found and how
each model will run. A model without its own key runs through OpenRouter if
that key is set, otherwise another model runs instead, and it says so. With no
key at all it stops and tells you which keys to set. ImageMagick and, for
transparent headers, `rembg` are also used.

## Workflow before prompt

Choose the workflow that matches the request before writing the prompt: essay
headers, technical diagrams, Mermaid diagrams, comparisons, timelines, and
other formats each carry their own visual rules. `Generate.ts` requires that
workflow choice unless the user explicitly asks to skip it.

For blog headers, the generation produces a transparent inline image and an
opaque social thumbnail as separate outputs. Review generated files in the
preview folder before copying an approved asset into a project. A signature,
logo, or custom font is added only when you set `ART_SIGNATURE`, `ART_LOGO`,
or `ART_THUMBNAIL_FONT`.

## Where it fits

`do-art` is the Content group's static-media skill: it makes material that
explains, frames, or promotes work to an audience. It complements engineering
design-system work rather than replacing it; the latter shapes product UI, while
this skill creates the surrounding visual media.
