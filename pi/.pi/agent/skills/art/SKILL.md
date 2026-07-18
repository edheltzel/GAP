---
name: art
description: Generate static visual content across 20+ formats — blog headers, technical and architecture diagrams, mermaid, frameworks, taxonomies, timelines, comparisons, stat cards, comics, icons, wallpapers, and D3 charts — using Flux, Nano Banana Pro (Gemini 3 Pro), and GPT-Image-2. Use when the user asks for art, an illustration, a diagram, flowchart, infographic, header image, blog or social thumbnail, chart, icon, wallpaper, mermaid, background removal, or to "visualize" / "generate an image". Every request routes through a named workflow that encodes the technique, palette, and composition rules; output stages to ~/Downloads first for review. Not for video or animation, or integrated web-UI / frontend layout.
compatibility: Requires bun, ImageMagick (magick), and rembg (~/.local/bin/rembg or $REMBG_BIN). Image-model API keys load from ~/.env — GOOGLE_API_KEY/GEMINI_API_KEY (nano-banana-pro, default), OPENAI_API_KEY (gpt-image-2), REPLICATE_API_TOKEN (flux). Run `bun install` once beside the real Tools files (see Pi conversion notes — under stow that is the atlas-config repo path, not ~/.pi).
metadata:
  source: converted from Claude skill ~/.claude/skills/Art v1.5.8
  engine: Tools/Generate.ts (bun)
---

# Art

Generate static visual content — illustrations, diagrams, headers, icons, charts — through named workflows that hold the composition rules the bare image model ignores. Pick a workflow first, follow its template, then call `Generate.ts --workflow=<name>`.

## Pi conversion notes

Converted from the Claude skill `~/.claude/skills/Art` for Pi. Differences from the source:

- Paths point at the Pi install: `~/.pi/agent/skills/art/…`. The bundled `Tools/` are self-contained bun scripts — **no dependency on the Claude install**.
- API keys resolve from `~/.env` (then `~/.pi/agent/.env`, then `~/.claude/.env`) — `Generate.ts` searches all three, so keys need not live in the Claude tree.
- The Claude-Code-only `ArtWorkflowGuard.hook.ts` PreToolUse guard is **not** ported. The workflow-first discipline is still enforced in code: `Generate.ts` itself refuses to run without `--workflow=<name>` (or `--freeform-confirmed`). That refusal is harness-agnostic and is the guard under Pi.
- Notifications target Echo (`localhost:8888`), not Pulse `:31337`, and are best-effort — never fail a workflow if the notifier is absent.
- First run — install deps **beside the real tool files**. Stow symlinks the `.ts` files into `~/.pi`, and bun resolves imports from each file's realpath, so `node_modules` must live next to the real files (in the atlas-config repo), not in `~/.pi`:
  `cd ~/Developer/Atlas/Config/pi/.pi/agent/skills/art/Tools && bun install`
  (`node_modules` there is git- and stow-ignored. On a non-stowed install where the Tools files are real, run `bun install` in the Tools dir directly.)

## Customization

Before executing, check for user customizations in this order (Pi first, legacy Claude fallback):

1. `~/.pi/agent/SKILLCUSTOMIZATIONS/art/`
2. `~/.claude/LIFEOS/USER/CUSTOMIZATIONS/SKILLS/Art/` (legacy source)

If present, load and apply `PREFERENCES.md` (aesthetic, default model, output location), `CharacterSpecs.md`, and `SceneConstruction.md`. Pi customizations override legacy. The thumbnail tools also read brand assets and headshots from `~/.pi/agent/SKILLCUSTOMIZATIONS/art/` (`brand/`, `HeadshotExamples/`).

## Best-effort notification

On invocation you may notify Echo (best-effort — never block on it):

```bash
curl -s -X POST http://localhost:8888/notify \
  -H "Content-Type: application/json" \
  -d '{"message": "Running the WORKFLOWNAME workflow in the Art skill to ACTION"}' \
  > /dev/null 2>&1 &
```

Also print: `Running the **WorkflowName** workflow in the **Art** skill to ACTION...`

## 🛑 Workflow-first is mandatory (enforced in code)

`Generate.ts` refuses to run without `--workflow=<name>`. Pick a workflow **before writing any prompt**, read that file, and build the prompt from its template — the templates encode the technique, palette, composition, and validation gate the bare model fails on their own. The `--workflow=` flag is your assertion that you read and followed it.

```bash
bun ~/.pi/agent/skills/art/Tools/Generate.ts \
  --workflow=<WorkflowName> \
  --model nano-banana-pro \
  --prompt "..." \
  --size 2K \
  --aspect-ratio 16:9 \
  --output ~/Downloads/<filename>.png
```

`<WorkflowName>` must match a file under `Workflows/` (without `.md`). The only exception is an explicit user "freeform / skip the workflow" instruction → pass `--freeform-confirmed` (logged to stderr). If no workflow fits, **stop and surface to the user** — propose the closest workflow, `Visualize` as the generic catch-all, or authoring a new workflow. Do not improvise.

### Routing — pick a workflow FIRST

| Request shape | Workflow |
|---|---|
| Blog header / editorial essay illustration | `Workflows/Essay.md` (Steps 1–8, in order) |
| Mermaid diagram | `Workflows/Mermaid.md` |
| Technical / architecture diagram | `Workflows/TechnicalDiagrams.md` |
| Framework / 2×2 / matrix | `Workflows/Frameworks.md` |
| D3 dashboard / chart | `Workflows/D3Dashboards.md` |
| Taxonomy / hierarchy | `Workflows/Taxonomies.md` |
| Timeline | `Workflows/Timelines.md` |
| Comparison / X vs Y | `Workflows/Comparisons.md` |
| Stat card / big number | `Workflows/Stats.md` |
| Aphorism / quote card | `Workflows/Aphorisms.md` |
| Comic panel | `Workflows/Comics.md` |
| Recipe card | `Workflows/RecipeCards.md` |
| Conceptual map | `Workflows/Maps.md` |
| Annotated screenshot | `Workflows/AnnotatedScreenshots.md` |
| Background removal only | `Workflows/RemoveBackground.md` |
| Logo wallpaper | `Workflows/LogoWallpaper.md` / `Workflows/EmbossedLogoWallpaper.md` |
| LifeOS pack icon | `Workflows/CreateLifeosPackIcon.md` |
| YouTube thumbnail | `Workflows/StyleMatchedThumbnail.md` · `Workflows/AdHocYouTubeThumbnail.md` · `Workflows/YouTubeThumbnailChecklist.md` |
| Anything else visual | `Workflows/Visualize.md` (generic catch-all) |

## 🚨 Output to ~/Downloads/ FIRST — always

Every workflow stages to `~/Downloads/` for preview. **Never** write directly to a project directory. The user reviews in Preview/Finder; only after approval do you copy to the destination.

## 🚨 Transparency rules for blog headers

- Inline (body) image → **transparent PNG** (composites over the sepia `#EAE9DF` page).
- Social thumbnail (frontmatter) → **opaque sepia `#EAE9DF`** (platforms ignore alpha).
- Every blog header runs `--thumbnail`, which implies `--remove-bg` and produces **both** `output.png` (transparent) and `output-thumb.png` (sepia). Background removal is local via `rembg`.
- Verify before declaring done: `file ~/Downloads/[name].png` must report `PNG image data, … RGBA`; a `.jpg` labeled "transparent" is not transparent.

## Models

| Model | `--size` | Notes |
|---|---|---|
| `nano-banana-pro` (default) | `1K`/`2K`/`4K` + separate `--aspect-ratio` | Best editorial composition; needs GOOGLE_API_KEY/GEMINI_API_KEY |
| `gpt-image-2` | `1024x1024`, `1536x1024`, `1024x1536`, `2048x2048`, `auto` (+ `--quality`) | Strongest text rendering — stat cards, frameworks, taxonomies; needs OPENAI_API_KEY |
| `flux` / `nano-banana` | aspect ratio (`1:1`, `16:9`, …) | Alternate register; `flux` crisper for technical; needs REPLICATE_API_TOKEN |
| `compare` | — | Runs gpt-image-2 + nano-banana-pro in parallel; pick the winner (default for editorial headers) |

`gpt-image-1` is deprecated and rejected by `Generate.ts`. `nano-banana-pro` uses `--size` for resolution and a **separate** `--aspect-ratio`.

## Reference images

Up to 5 human / 6 object / 14 total per request (Gemini limit). Repeat `--reference-image <path>` per image. For likeness, roll until one output is correct, then feed **that** output back as the reference for remaining variations — editing the reference beats describing the edit.

## Gotchas

- After generating, **Read the image** to confirm it before reporting success. "Generated successfully" means nothing unlooked-at.
- `--remove-bg` is unsafe for thin-linework diagrams (rembg strips thin ink as background) — prompt for thick saturated linework, or skip removal when the destination background already matches.
- Prompt large flat surfaces (desks, panels, paper) as "warm cream paper tone", never white — baked-white interiors survive rembg and render as white rectangles on the cream page.
- `nano-banana-pro` "4K 16:9" is natively 5504×3072 (~43:24) and saved `.jpg` even when `--output` says `.png` — probe the real filename before Read/delivery.
- Use the exact asset the user names; don't substitute similar images.
- Standalone background removal: `bun ~/.pi/agent/skills/art/Tools/RemoveBg.ts <in> [out]` (bundled; wraps local rembg, handles `.jpg → .png`).

## Context files

- `Workflows/*.md` — one file per format; the prompt template + validation gate.
- `Tools/Generate.ts` — the image-generation CLI (workflow gate, models, `--thumbnail`, rembg).
- `Tools/RemoveBg.ts`, `Tools/ThumbnailText.ts`, `Tools/PickExpression.ts`, `Tools/ComposeThumbnail.ts`, `Tools/FillFrame.ts`, `Tools/GeneratePrompt.ts`, `Tools/GenerateMidjourneyImage.ts` — supporting tools.
- `Tools/AGENTS.md` — bun conventions for the tools.

## Execution log

After completing a workflow, append one JSONL entry when practical:

```bash
mkdir -p ~/.pi/agent/memory/skills
echo '{"ts":"'$(date -u +%Y-%m-%dT%H:%M:%SZ)'","skill":"art","workflow":"WORKFLOW_USED","input":"8_WORD_SUMMARY","status":"ok|error","duration_s":SECONDS}' >> ~/.pi/agent/memory/skills/execution.jsonl
```
