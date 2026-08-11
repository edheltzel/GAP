# Remove Background Workflow

**Remove backgrounds from existing images using local rembg (no external API).**

## Voice Notification

```bash
curl -s -X POST http://localhost:8888/notify \
  -H "Content-Type: application/json" \
  -d '{"message": "Running the RemoveBackground workflow in the Art skill to remove image backgrounds"}' \
  > /dev/null 2>&1 &
```

Running **RemoveBackground** in **Art**...

---

## Purpose

Remove backgrounds from existing images to create transparent PNGs. Useful for:

- Converting diagrams to transparent backgrounds
- Preparing images for web display (composites cleanly over the cream blog background)
- Creating icons with transparent backgrounds
- Cleaning up screenshots

---

## Tooling

Local `rembg` (Python, ONNX-based, runs offline). No external API, no rate limits, no API keys.

**Availability gate:** `RemoveBg.ts` first honors `REMBG_BIN`, then searches `PATH`, then checks `~/.local/bin/rembg`. It exits before reading or modifying an image when no executable is available.

**Install if missing:**

```bash
pipx install rembg          # preferred
# or
uv tool install rembg       # if you use uv
```

---

## Workflow Steps

### Step 1: Verify Input File

Confirm the image file exists and note its current size.

```bash
ls -lh /path/to/image.png
```

### Step 2: Remove Background

Use the native Art `RemoveBg.ts` wrapper, which checks availability, calls local `rembg`, and handles the `.jpg to .png` rename automatically because rembg always emits PNG:

```bash
# Single file (overwrites; renames .jpg→.png)
bun ~/.claude/skills/Art/Tools/RemoveBg.ts input-image.png

# Single file with explicit output path
bun ~/.claude/skills/Art/Tools/RemoveBg.ts input-image.jpg output-image.png

# Batch (overwrites each in place)
bun ~/.claude/skills/Art/Tools/RemoveBg.ts img1.png img2.png img3.png
```

If you need to call `rembg` directly, first verify the maintained command is available:

```bash
command -v rembg >/dev/null || { echo "Install with: pipx install rembg" >&2; exit 1; }
rembg i input-image.png output-image.png
```

### Step 3: Verify Transparency

Confirm the output is real PNG with an alpha channel:

```bash
# MUST report "PNG image data, ... RGBA"
file output-image.png

# Sanity-check alpha via ImageMagick
magick identify -format "%[channels]" output-image.png
# → "srgba" (or contains "a") = alpha present
# → "srgb" without "a" = NO alpha - transparency failed
```

### Step 4: Replace or Copy to Destination

Either replace the original or copy to the intended destination:

```bash
# Replace original (after verification)
mv output-image.png input-image.png

# Or copy to specific destination
cp output-image.png /destination/path/transparent-image.png
```

---

## Examples

### Example 1: Remove background from a diagram

```bash
bun ~/.claude/skills/Art/Tools/RemoveBg.ts ~/Downloads/TheAlgorithm.png
```

### Example 2: Remove background and save with new name

```bash
bun ~/.claude/skills/Art/Tools/RemoveBg.ts \
  ~/your-site/public/images/logo-with-bg.png \
  ~/your-site/public/images/logo-transparent.png
```

### Example 3: Process multiple images

```bash
cd ~/Downloads
bun ~/.claude/skills/Art/Tools/RemoveBg.ts diagram-*.png
```

---

## Troubleshooting

**Problem:** `rembg not found or not executable`
**Solution:** `pipx install rembg`, put it on `PATH`, or set `REMBG_BIN` to an executable path.

**Problem:** First run is slow (downloads ONNX model)
**Solution:** Expected. The default `u2net` model (~176MB) is fetched once into `~/.u2net/`, then cached forever. Subsequent runs are fast.

**Problem:** Output file looks identical to input
**Solution:** rembg failed to detect a clear subject. Try a model better suited to the content:

```bash
rembg i -m u2netp input.png output.png             # smaller/faster
rembg i -m isnet-general-use input.png output.png   # general-purpose, often better edges
rembg i -m birefnet-general input.png output.png    # higher quality, slower
```

**Problem:** Edges are jagged or hair/fine detail is lost
**Solution:** Use `birefnet-general` (or `birefnet-portrait` for people) - both produce noticeably better edges than the default `u2net`.

---

## Related Workflows

- `Workflows/Essay.md` - `--thumbnail` flag in Generate.ts implicitly removes background via local rembg

---

**Last Updated:** 2026-04-27 - switched from poof.bg to local rembg
