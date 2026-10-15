# StyleMatchedThumbnail Workflow

Build a YouTube thumbnail that matches a reference's text style, places a real expression-matched photo on one third, sits it over a topic background, and looks hand-made.

The legacy `AdHocYouTubeThumbnail` path generated a fresh face every run and composited text with broken geometry. On a channel whose audience knows the creator's face, a synthesized face is the loudest tell. This workflow uses a real photo selected by expression, and composites text with `ThumbnailText.ts`. The background is the only generated element.

`<skill-dir>` is the folder containing this skill's SKILL.md. `<preview>` is `$ART_OUTPUT_DIR` if set, otherwise `~/Downloads` when that directory exists, otherwise `./art-output`.

## Overlay standard

Invoke `ThumbnailText.ts --mode overlay`. Layout rules:

- **Background is the whole frame.** A topic-themed art image (`--bg`, cover-fit), full-bleed, dark, no text, logos, or people. Or leave it unset and let the tool fill navy `#1A2744`.
- **Face: tall, far right, trim first.** Trim the cutout to the subject before scaling. Width-cap to about 54% so the head is large and bleeds slightly off the right edge. Not centered (centered text lands on the face).
- **Text: large, left-aligned, vertically centered, `--text-style shadow`.** Per-letter shadow: a blurred dark copy of each line's glyphs, then a crisp fill. Not a box behind the word and not one block behind the stack. Titles of three words or fewer stack one word per line. No subtitle in this style.
- **Font:** `--font`, else `ART_THUMBNAIL_FONT`, else `DejaVu-Sans-Bold` (ImageMagick ships it). Do not assume a personal font file.
- **Border:** about 30px, semantic color (`--variant core` blue `#316AE9` / `sponsored` green `#306F1D`).
- **Logo:** opt-in. Pass `--logo <path>` or set `ART_LOGO`. No logo when unset. When set, it sits top-right, inset from the border.
- **Validate against the brief:** face big and clear of the title? title large and readable at 320x180? shadow per letter, not a muddy box? border present? logo only if one was provided?

Describe the target in those terms. Do not depend on a canonical file on one machine.

## Inputs

- `TITLE` - headline. Optional kicker and subtitle for solo/interview modes.
- `TOPIC` - drives the background concept and the expression pick.
- `REFERENCE` (optional) - a thumbnail the user supplies to match. If they do not supply one, use the palette in this file.
- `N` - variants (default 4).
- `SENTIMENT` (optional) - overrides the auto-derived expression.
- `ART_HEADSHOT_DIR` or `--dir` - folder of expression-labeled headshots for `PickExpression.ts`. Ask for it. There is no default path.

## Steps

### 1. Analyze the reference style

If a `REFERENCE` image is given, read it and capture background grade, accent color, font feel, and which third holds the face. Defaults when no reference is given: navy `#1A2744`, title periwinkle `#6B8DD6`, white kicker/subtitle, semantic border via `--variant`.

### 2. Pick a real expression-matched face

```bash
bun <skill-dir>/Tools/PickExpression.ts --dir "<ART_HEADSHOT_DIR>" --topic "<TOPIC>"
# or: --sentiment skeptical|neutral|positive|curious|thinking|disgust|shock|surprise|casual
```

Returns a path under the headshot directory. Read it and confirm the expression fits. Default to the real photo.

Optional face generation, only when the library has no fitting expression. Use real headshots as references:

```bash
bun <skill-dir>/Tools/Generate.ts --workflow=StyleMatchedThumbnail --model nano-banana-pro \
  --size 2K --aspect-ratio 1:1 --no-signature \
  --reference-image <headshot-a.png> --reference-image <headshot-b.png> \
  --prompt "Photorealistic head-and-shoulders portrait of the person in the references, <EXPRESSION>, looking at camera, plain near-black background, studio lighting, NO text." \
  --output <preview>/face-gen.png
```

Read it and reroll on likeness drift before using it.

### 3. Background

Default is a solid deep-navy field `#1A2744`. `ThumbnailText.ts` fills that itself. A generated cinematic scene reads more like AI, not less.

- Supporting art: `--art <image.png>` (diagram, screenshot, product mark). The tool darkens it and places it behind the text.
- Custom plate: `--bg <plate.png>`.
- Generated plate, only when a rendered scene is actually wanted: `Generate.ts --workflow=StyleMatchedThumbnail --model nano-banana-pro --no-signature` (never `--thumbnail`), then pass it as `--bg`. The model may save JPEG. Glob both extensions.

### 4. Compose

Solo:

```bash
bun <skill-dir>/Tools/ThumbnailText.ts \
  --face "<headshot from step 2>" --art "<diagram.png, optional>" \
  --kicker "A LOOK AT" --title "HOW AGENTS WORK" \
  --variant core --face-side right \
  --output <preview>/final-1.png
```

Interview:

```bash
bun <skill-dir>/Tools/ThumbnailText.ts --mode interview \
  --kicker "A CONVERSATION WITH" --title "THE GUEST" --subtitle "ON THE TOPIC" \
  --face host.png --face2 guest.png --name1 "Host" --name2 "Guest" \
  --accent "#F5A623" --variant sponsored \
  --output <preview>/final-1.png
```

Add `--logo <path>` only when the user supplied a mark.

- Semantic border is on by default. `--variant core` blue, `sponsored` green.
- Type hierarchy: `--kicker` / `--title` / `--subtitle` / `--tag`.
- Font default is `DejaVu-Sans-Bold`. Override with `--font` or `ART_THUMBNAIL_FONT`.
- Face cutout is automatic (rembg or floodfill from the source background).
- Logo is omitted unless `--logo` or `ART_LOGO` is set. `--no-logo` forces it off.

Writes 1280x720 plus a 320x180 proof and prints JSON (`titlePt`, `contrastRatio`, `overflowed`). Non-zero exit means the title does not fit. Split it.

Flags: `--mode solo|interview|overlay`, `--variant core|sponsored`, `--accent #hex`, `--subtitle-color #hex`, `--tag`, `--art`, `--bg`, `--logo`, `--cut auto|floodfill|rembg|none`, `--face-side left`, `--font`, `--border "30,#hex"`, `--no-border`, `--no-logo`, `--no-rule`, `--text-style shadow|boxed|accent`.

### 5. Quality gates

- `magick identify -format '%wx%h %m' final-1.png` must be `1280x720 PNG`.
- Read the 320x180 proof. The title must be legible.
- Read full-res. Title spelling is guaranteed (composited, not generated). Face is a real photo. Text is grounded by shadow.
- JSON: `overflowed:false` and `contrastRatio` at least 3.0.

### 6. Present

```bash
magick montage <preview>/final-*.png -tile 2x2 -geometry 640x360+6+6 <preview>/CONTACT.png
```

Read the sheet and let the user pick.

## Gotchas

- Validate against the brief (and against a user-supplied reference if they gave one), element by element. Reading your own output alone is not validation. Check layout, font, face size, logo only if one was passed, and border.
- Overlay text is left, large, and vertically centered, with a per-letter shadow. Solo mode can use a full-width title band plus a big bottom-anchored face. Do not pin text to the top corner and leave the rest empty.
- Subtitle scale in solo mode is 0.58 so a long subtitle does not crush the headline. If `titlePt` is 60 or below, shorten the subtitle.
- Never pass `--thumbnail` to `Generate.ts` here. That flag is blog-header mode (transparent plus sepia thumb). Pass `--no-signature` unless the user set `ART_SIGNATURE` and wants it.
- The face must be a real photo by default. Generating a face is opt-in and only for a missing expression.
- Plate prompts must forbid text, logos, and people. The compositor owns the text.
- The model sometimes writes JPEG when you asked for PNG. Glob both.
- `ComposeThumbnail.ts` is the legacy path (trim-then-guess text, border-then-squash). This workflow uses `ThumbnailText.ts`.
- Default field is solid navy plus optional real supporting art, not a generated scene.
- The colored outline is semantic: blue core, green sponsored. Set `--variant`.
- Interview layout is two faces, centered text, and the sponsored border. It is not a green-bordered solo layout.
