---
name: do-art
version: 1.5.8
description: "Static visual content across 20+ formats - diagrams, mermaid, infographics, D3 dashboards, comics, icons, wallpaper - via Grok Imagine (xAI), GPT-Image-2 (OpenAI), and Nano Banana / Nano Banana Pro (Gemini). USE WHEN art, illustration, diagram, flowchart, infographic, header image, blog social thumbnail, visualize, generate image, mermaid, architecture diagram, comic, icon, blog art, framework diagram, D3 chart, remove background, wallpaper. NOT FOR locked house-style YouTube/channel/video thumbnails, video or animation (use Hyperframes or Remotion), or web UI design and integrated frontend layout."
effort: medium
---

# Art Skill

`<skill-dir>` below is the folder containing this SKILL.md.

`<preview>` is `$ART_OUTPUT_DIR` if set, otherwise `~/Downloads` when that directory exists, otherwise `./art-output`. Generate there, review, then copy into the project. Do not write straight into a project's image tree.

## Before generating: check API keys

Run `bun <skill-dir>/Tools/Generate.ts --check-keys` once per session, before writing a prompt. It lists which keys are set: xAI (`XAI_API_KEY`), OpenAI (`OPENAI_API_KEY`), Google Gemini (`GEMINI_API_KEY` or `GOOGLE_API_KEY`), OpenRouter (`OPENROUTER_API_KEY`), and how each model will run (direct, via OpenRouter, or unavailable). An OpenRouter key alone covers all four models.

- **Exit 2 (no key at all):** stop. Tell the user that image generation needs at least one of those keys, show the list the command printed (where to get each key, and that keys load from the environment, `./.env`, or `${XDG_CONFIG_HOME:-~/.config}/do-art/.env`), and wait. Do not compose prompts or try other tools.
- **Some keys missing:** proceed, and tell the user which providers are unavailable when it changes the result (for example, `compare` will skip them, or the workflow's preferred model will fall back).

`Generate.ts` enforces the same rules: a model whose own key is missing runs through OpenRouter when `OPENROUTER_API_KEY` is set, otherwise falls back to another model that can run; either way it prints a warning (relay it to the user). With no usable key it exits 2 and nothing is generated. Reference images go only to Gemini models (direct or via OpenRouter).

## What It Does

Generates static visual content across 20+ formats - blog headers, technical and architecture diagrams, frameworks, taxonomies, timelines, comparisons, stat cards, comics, icons, wallpapers, D3 charts, Mermaid diagrams - using Grok Imagine, GPT-Image-2, Nano Banana, and Nano Banana Pro. Every request routes through a named workflow that encodes the technique and palette, output stages to <preview> for review first, and blog headers ship both a transparent inline version and an opaque social thumbnail.

## The Problem

The bare image model produces inconsistent, off-style output when handed a freeform prompt - one session shipped 12 rejected diagrams because the prompt skipped the workflow that holds the composition rules. Different formats need different models (text-heavy cards want GPT-Image-2; editorial headers want Nano Banana Pro), different size formats, and different transparency handling. Without a fixed routing-and-staging discipline, you get wrong sizes, opaque headers that bleed over the page background, and images pushed straight to a repo before anyone looked at them. This skill makes the workflow, the model choice, and a preview-before-copy review mandatory in code, not just in markdown.

## How It Works

A complete visual content system for illustrations, diagrams, and other static visuals. Each request picks a matching workflow file first, follows its prompt template, then calls `Generate.ts` with `--workflow=<name>` plus model/size/output flags. `Generate.ts` itself enforces that the workflow was followed, output defaults to <preview> for review, and blog headers run with `--thumbnail` to produce both the transparent PNG and the sepia-backed social thumbnail.

## 🛑 STRUCTURAL ENFORCEMENT - `--workflow=<name>` IS REQUIRED

**This rule used to be markdown-only and was silently ignored, producing 12 rejected diagrams in one session (incident 2026-04-30). It now lives in code.**

**`Generate.ts` itself** refuses to run unless you pass `--workflow=<name>` (or the explicit `--freeform-confirmed` opt-out). It exits non-zero with the workflow lookup table.

**The flow that works:** read the matching workflow file → follow its prompt template → invoke `Generate.ts` with `--workflow=<that-workflow-name>` plus your model/prompt/size flags. The `--workflow=<name>` flag is your explicit assertion "I read the workflow and followed it."

**The flow that's blocked:** composing a freeform prompt and shipping it directly to `Generate.ts`. It will refuse.

### Most Common Failure Mode (don't repeat it)

Reading the workflow's caps-warning, mentally noting "do the workflow," then composing a Bash command with your own prompt anyway because it feels faster. **Stop.** The workflow templates encode the technique, palette, composition rules, and validation gate the bare model fails to honor. Skipping them produced - verbatim - "absolute fucking ass" diagrams. Read the workflow file FIRST. Compose the prompt FROM the template. Pass `--workflow=<name>` so the gate can see you did it.

### Workflow → command (copy-paste)

```bash
bun <skill-dir>/Tools/Generate.ts \
  --workflow=<WorkflowName> \
  --model nano-banana-pro \
  --prompt "..." \
  --size 2K \
  --aspect-ratio 16:9 \
  --output <preview>/<filename>.png
```

`<WorkflowName>` MUST match a file under `Workflows/` (without `.md`):

**Routing rules - pick a workflow FIRST, before writing any prompt:**

| Request shape | Required workflow |
| --------------- | ------------------- |
| Blog header / editorial essay illustration | **`Workflows/Essay.md`** - Steps 1-8 in order, no skipping |
| Mermaid diagram | `Workflows/Mermaid.md` |
| Technical / architecture diagram | `Workflows/TechnicalDiagrams.md` |
| Framework / 2x2 / matrix | `Workflows/Frameworks.md` |
| D3 dashboard / chart | `Workflows/D3Dashboards.md` |
| Taxonomy / hierarchy | `Workflows/Taxonomies.md` |
| Timeline | `Workflows/Timelines.md` |
| Comparison | `Workflows/Comparisons.md` |
| Stat card | `Workflows/Stats.md` |
| Aphorism / quote card | `Workflows/Aphorisms.md` |
| Comic panel | `Workflows/Comics.md` |
| YouTube thumbnail | **`Workflows/StyleMatchedThumbnail.md`** - deterministic text + real-photo face |
| YouTube thumbnail (legacy / validation) | `Workflows/AdHocYouTubeThumbnail.md` or `Workflows/YouTubeThumbnailChecklist.md` |
| brand-logo wallpaper | `Workflows/LogoWallpaper.md` |
| Recipe card | `Workflows/RecipeCards.md` |
| Map / conceptual map | `Workflows/Maps.md` |
| Annotated screenshot | `Workflows/AnnotatedScreenshots.md` |
| Background removal only | `Workflows/RemoveBackground.md` |
| Embossed logo wallpaper | `Workflows/EmbossedLogoWallpaper.md` |
| Generic visualization (none of the above fit) | `Workflows/Visualize.md` |

**The ONLY exception:** the user explicitly says "freeform" / "skip the workflow" / "just run Generate.ts directly with this prompt: ...". In that case, pass `--freeform-confirmed` to `Generate.ts` (which logs the explicit opt-out to stderr for audit). Without that explicit instruction from the user, ALWAYS pick the matching workflow and pass `--workflow=<name>` - `Generate.ts` will refuse the call otherwise.

If no workflow matches the request, **stop and surface to the user** before generating - propose either (a) the closest existing workflow, (b) using `Visualize.md` as the generic catch-all, or (c) creating a new workflow file first. Do not improvise.

---

## Preview before copying into a project

Generate into `<preview>`, not into a project's `public/` tree. Review the file, then copy the approved image. This applies to every workflow in this skill.

## 🚨🚨🚨 MANDATORY: Transparency Rules for Blog Headers 🚨🚨🚨

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
⚠️  INLINE (body) image → TRANSPARENT (PNG with alpha)           ⚠️
⚠️  SOCIAL THUMBNAIL (frontmatter) → SEPIA #EAE9DF (opaque)       ⚠️
⚠️  EVERY blog header MUST use --thumbnail (produces both)        ⚠️
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

The blog page background is sepia #EAE9DF. Inline images MUST be transparent PNG so they composite cleanly over the page. Social platforms (X, LinkedIn, RSS readers) do NOT honor transparency - they show white/black bleed-through - so the `thumbnail:` frontmatter MUST point to the sepia-backed version.

**Enforcement when calling `Generate.ts`:**

- `--thumbnail` is the ONLY correct flag for blog headers - it implicitly enables `--remove-bg` and produces BOTH `output.png` (transparent) AND `output-thumb.png` (#EAE9DF background).
- Background removal runs locally via `rembg` (no external API). If the model returns JPEG (Nano Banana Pro often does), `Generate.ts` automatically renames the output from `.jpg` → `.png` after rembg processing so the final transparent file is a real PNG with a real alpha channel. If you ever see a `.jpg` labeled "transparent", that is NOT transparent.
- If `rembg` is not on `PATH` (last resort: `~/.local/bin/rembg`), the tool fails loudly with install instructions rather than silently producing an opaque image. Install: `pipx install rembg`. Set `REMBG_BIN` to override the path.

**Verification step before declaring an image done (REQUIRED):**

1. `file <preview>/[name].png` must report `PNG image data, ... RGBA` (8-bit/color RGBA). If it says `JPEG` or `8-bit colormap` without alpha, transparency failed.
2. `file <preview>/[name]-thumb.png` must report `PNG image data`. The thumb is intentionally opaque with sepia background.
3. Only after both pass: copy to the project directory and wire into the post.

**Wiring into the blog post:**

- Body inline: `[![Alt](/images/blog/[slug]/header.webp)](/images/blog/[slug]/header.webp)` - use the transparent WebP converted from the `.png`.
- Frontmatter: `thumbnail: https://example.com/images/blog/[slug]/header-thumb.png` - always the `-thumb.png` (opaque sepia).

Never reuse the opaque thumbnail for the inline slot. Never reuse the transparent file for the social thumbnail. These are two distinct outputs from one `--thumbnail` run.

**Sanctioned exception (this section is the canonical home):** transparent inline is the DEFAULT for every blog header. The one exception is thin-linework/charcoal pieces where rembg strips the artwork itself (see Gotchas) - those may ship an opaque sepia `#EAE9DF` inline image, which composites seamlessly on the matching page background. Opaque inline is a documented fallback for that failure mode, never a second default.

## Workflow Routing

Route to the appropriate workflow based on the request.

| Workflow | Trigger | File |
| ---------- | --------- | ------ |
| Essay | Blog header or editorial illustration | `Workflows/Essay.md` |
| RemoveBackground | Remove background from image | `Workflows/RemoveBackground.md` |
| LogoWallpaper | brand-logo wallpaper with logo integration | `Workflows/LogoWallpaper.md` |
| EmbossedLogoWallpaper | Embossed logo wallpaper | `Workflows/EmbossedLogoWallpaper.md` |
| D3Dashboards | D3.js interactive chart or dashboard | `Workflows/D3Dashboards.md` |
| Visualize | Visualization or unsure which format | `Workflows/Visualize.md` |
| Mermaid | Mermaid flowchart or sequence diagram | `Workflows/Mermaid.md` |
| TechnicalDiagrams | Technical or architecture diagram | `Workflows/TechnicalDiagrams.md` |
| Taxonomies | Taxonomy or classification grid | `Workflows/Taxonomies.md` |
| Timelines | Timeline or chronological progression | `Workflows/Timelines.md` |
| Frameworks | Framework or 2x2 matrix | `Workflows/Frameworks.md` |
| Comparisons | Comparison or X vs Y | `Workflows/Comparisons.md` |
| AnnotatedScreenshots | Annotated screenshot | `Workflows/AnnotatedScreenshots.md` |
| RecipeCards | Recipe card or step-by-step | `Workflows/RecipeCards.md` |
| Aphorisms | Aphorism or quote card | `Workflows/Aphorisms.md` |
| Maps | Conceptual map or territory | `Workflows/Maps.md` |
| Stats | Stat card or big number visual | `Workflows/Stats.md` |
| Comics | Comic or sequential panels | `Workflows/Comics.md` |
| YouTubeThumbnailChecklist | YouTube thumbnail checklist; YouTube thumbnail (with existing assets) | `Workflows/YouTubeThumbnailChecklist.md` |
| AdHocYouTubeThumbnail | Ad-hoc YouTube thumbnail (generate from content) | `Workflows/AdHocYouTubeThumbnail.md` |

---

## Core Aesthetic

**Default:** Production-quality concept art style appropriate for editorial and technical content.

**User customization** defines specific aesthetic preferences including:

- Visual style and influences
- Line treatment and rendering approach
- Color palette and wash technique
- Character design specifications
- Scene composition rules

---

## Reference Images

**User customization** may include reference images for consistent style.

- Reference image locations
- Style examples by use case
- Character and scene reference guidance

**Usage:** Before generating images, load relevant user-provided references to match their preferred style.

---

## Image Generation

**Default model:** `auto` (set `ART_MODEL` to change it). `auto` uses the first model that can run, in this order: `grok`, `gpt-image-2`, `nano-banana-pro`. Workflows usually name a model explicitly.

### Models and keys

| `--model` | Provider and API model | Key | Strengths |
| --- | --- | --- | --- |
| `grok` | xAI `grok-imagine-image-2.0` | `XAI_API_KEY` | Fast, cheap, good default for illustration and comics. Max 2k. No 4:5 or 5:4 (uses its own `auto` ratio). |
| `gpt-image-2` | OpenAI `gpt-image-2` | `OPENAI_API_KEY` | Strongest text rendering: stat cards, frameworks, taxonomies, timelines. `--quality low/medium/high/auto`. |
| `nano-banana` | Google `gemini-nano-banana-2.1` | `GEMINI_API_KEY` or `GOOGLE_API_KEY` | Fast drafts, 1K-4K. Accepts `--reference-image`. |
| `nano-banana-pro` | Google `gemini-3-pro-image` | `GEMINI_API_KEY` or `GOOGLE_API_KEY` | Best composition fidelity for editorial work, 1K-4K. Best `--reference-image` model (up to 14). |
| `compare` | Every model that can run | any | Same brief on each (one flagship per provider); pick the winner. |
| `auto` | First model that can run | any | Default. |

**OpenRouter (`OPENROUTER_API_KEY`)** is a route, not a separate model: when a model's own key is missing, `Generate.ts` sends the same model to OpenRouter's image API (`x-ai/grok-imagine-image-2.0`, `openai/gpt-image-2`, `google/gemini-nano-banana-2.1`, `google/gemini-3-pro-image`). A direct key always wins over OpenRouter.

`gpt-image-1` is deprecated and rejected. `flux` and Midjourney were removed.

### Size flags (same for every model)

- `--aspect-ratio`: `1:1`, `16:9`, `3:2`, `2:3`, `3:4`, `4:3`, `4:5`, `5:4`, `9:16`, `21:9` (default `16:9`).
- `--size`: resolution `1K`/`2K`/`4K` (default `2K`). It also accepts an aspect ratio, or an exact gpt-image-2 size (`1024x1024`, `1536x1024`, `1024x1536`, `2048x2048`, `auto`) when you need one.
- Each provider maps these itself: Grok caps at 2k; gpt-image-2 picks the nearest pixel size (`1024x1024`/`2048x2048` square, `1536x1024` landscape, `1024x1536` portrait).

### Model Selection - when to pick which

PREFERENCES.md (if present) pins the user's default; in absence of a pin, pick by job, then let fallback handle a missing key:

| Job | Recommended model | Why |
| ----- | ------------------- | ----- |
| Editorial illustration / blog header | `nano-banana-pro` | Best composition fidelity for editorial aesthetics. |
| Text-heavy work - stat cards, framework diagrams, taxonomies, timelines, aphorism cards | `gpt-image-2` | Strongest text rendering. |
| Editorial / blog / essay header, competing head-to-head | `compare` | Each keyed provider renders the same brief; pick the winner. See `Workflows/Essay.md`. |
| Stylistic variety, comics, iteration speed, low cost | `grok` or `nano-banana` | Different aesthetic register, fast drafts. |
| Character or style consistency from reference photos | `nano-banana-pro` | Only Gemini models take `--reference-image`. |

### Preview folder, then the project

Generate into `<preview>` first. Do not write straight into a project's `public/images/` directory. Review the image, then copy.

1. Generate to `<preview>/[descriptive-name].png`
2. User reviews the file
3. If approved, copy to the final destination (for example `cms/public/images/`)
4. Create WebP and thumbnail versions at the final destination

```bash
bun <skill-dir>/Tools/Generate.ts \
  --workflow=Essay \
  --model nano-banana-pro \
  --prompt "[PROMPT]" \
  --size 2K \
  --aspect-ratio 1:1 \
  --thumbnail \
  --output <preview>/blog-header-concept.png

# After approval, copy to the project path the user gave
cp <preview>/blog-header-concept.png <project>/public/images/
cp <preview>/blog-header-concept-thumb.png <project>/public/images/
```

### Multiple Reference Images (Character/Style Consistency)

For improved character or style consistency, use multiple `--reference-image` flags (Gemini models only):

```bash
# Multiple reference images for better likeness
bun run <skill-dir>/Tools/Generate.ts \
  --workflow=<WorkflowName> \
  --model nano-banana-pro \
  --prompt "Person from references at a party..." \
  --reference-image face1.jpg \
  --reference-image face2.jpg \
  --reference-image face3.jpg \
  --size 2K \
  --aspect-ratio 16:9 \
  --output <preview>/character-scene.png
```

**API Limits (Gemini):**

- Up to 5 human reference images
- Up to 6 object reference images
- Maximum 14 total reference images per request

**API keys:** process environment first. Optional files, only if present: `.env` in the current working directory, then `${XDG_CONFIG_HOME:-~/.config}/do-art/.env`. Neither file is required. `Generate.ts --check-keys` shows what it found.

## Examples

**Example 1: Blog header image**

```
User: "create a header for my AI agents post"
→ Invokes ESSAY workflow
→ Generates charcoal sketch prompt
→ Creates image with architectural aesthetic
→ Saves to <preview> for review
→ After approval, copies to public/images/
```

**Example 2: Technical architecture diagram**

```
User: "make a diagram showing the SPQA pattern"
→ Invokes TECHNICALDIAGRAMS workflow
→ Creates structured architecture visual
→ Outputs PNG with consistent styling
```

**Example 3: Comparison visualization**

```
User: "visualize humans vs AI decision-making"
→ Invokes COMPARISONS workflow
→ Creates side-by-side visual
→ Charcoal sketch with labeled elements
```

## Gotchas

- **Write to <preview> first, not straight into a project directory.** Review before copying. Pushing an unseen image into a repo is how bad headers ship.
- **Verify image dimensions match target use case before claiming done.** Social media previews, blog headers, and thumbnails have different size requirements. A header that works on the blog may break OG/social previews.
- **Relay key warnings.** If `Generate.ts` prints a fallback warning or exits 2, tell the user which key is missing and which provider ran instead (or that nothing ran). Never report success from a run that fell back without saying so.
- **Reference images: max 5 human, 6 object, 14 total per request** (Gemini API limit).
- **After generating, use Read tool to visually confirm the image before reporting success.** "Generated successfully" means nothing if you haven't looked at it.
- **When asked to use a specific image URL or file, use EXACTLY that asset.** Don't substitute similar images. Past rating-1 failures from using wrong image assets.
- **`--remove-bg` may produce black backgrounds instead of transparency.** Always verify transparent PNG output visually before deploying.
- **`--remove-bg` is unsafe for thin-linework technical diagrams.** rembg classifies thin black ink on a light field as "background" and strips it, leaving a near-empty ghost. Documented 2026-05-11 on the free-will flowchart. Mitigations: (a) prompt for *thick* saturated linework first so rembg has a strong signal, or (b) skip `--remove-bg` entirely when the destination background matches the image's background (blog page is sepia #EAE9DF - opaque sepia diagram on sepia page composites with zero visible seam, no alpha needed).
- **Logo fidelity breaks in 3D/perspective scenes even with a reference image.** Documented 2026-06-11 on a wallpaper set: straight-on and macro scenes held the glyph topology in 7/7 rolls, but the isometric 3D scene closed the open mark into a loop and dropped its isolated dot. For any perspective/3D composition with a logo, add topology-locked negative language to the prompt ("do not close the shape into a loop", "do not omit the isolated dot", name every stroke and terminal) on top of `--reference-image`, and vision-verify the topology specifically.
- **nano-banana-pro "4K 16:9" is actually 5504×3072 (43:24, ~0.8% wider than 16:9), saved as .jpg even when `--output` says .png.** Disclose the native ratio when the spec says 16:9, and probe the real filename before Read/delivery.
- **White-box-on-cream bug (2026-06-20): flattening an OPAQUE jpeg on `#EAE9DF` is a no-op.** nano-banana-pro returns an opaque JPEG; `magick -background "#EAE9DF" -flatten` only fills *alpha*, so the model's baked near-white ground survives and paints a white rectangle on the cream blog page ("it has a fucking white background"). For inline blog headers, cut true alpha FIRST (`bun <skill-dir>/Tools/RemoveBg.ts`), then derive the WebP, and verify `identify -format "%[channels]" inline.webp` == `srgba`. Opaque-sepia inline is valid ONLY on an image that already has alpha. See Essay.md Step 7.0.5.
- **Essay headers: run the Step 5A best-image deliberation before prompting.** Subject-list prompts produce flat tableaus. A composition reasoned from the essay's argument - scene concepts compared, every element given a narrative role - lands harder. The deliberation is the step. Devices like cutaways are possible outcomes, not rules. See Essay.md Step 5A.
- **Interior-white ban (2026-07-09, "giant white space" incident):** prompt large flat surfaces (desks, panels, windows, paper) as "warm cream paper tone", never bright white or unstated - baked-white interiors survive rembg intact and render as giant white rectangles on the cream page. Inside-the-subject sibling of the 2026-06-20 white-box bug. Also trim white padding off any external screenshot before embedding (`magick -fuzz 4% -trim` + sepia border).
- **Reference-image edits: negative text loses to the reference (2026-07-09 studio-background session).** When nano-banana-pro keeps reproducing an unwanted object that exists in the reference photo (e.g. a second floor lamp), "do NOT add/duplicate" prompt language fails ~7/8 rolls - the model preserves what it sees over what you forbid. Fix: roll until ONE output has the corrected composition, then use THAT output as the new `--reference-image` for the remaining variations; compliance jumped to 7/7. Editing the reference beats describing the edit.
- **No signature unless `ART_SIGNATURE` is set.** When that env var is non-empty, `Generate.ts` stamps it bottom-right after generation (small, semi-transparent, tucked in). Never prompt the signature into the model (it garbles). `--no-signature` skips the stamp. When unset, the image has no added text. Optional font: `ART_SIGNATURE_FONT`. Otherwise ImageMagick picks.
