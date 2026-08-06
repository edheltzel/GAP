---
name: HTML
version: 1.0.0
description: Renders the current session's output as a self-contained HTML file through a deterministic local renderer, then opens it for local review with lavish-axi when available. USE WHEN /HTML, HTML artifact, render this as HTML, make this an HTML page, or designed HTML output. NOT FOR deployed websites or web apps, web UI design systems, static images, or writing the underlying analysis itself.
---

# HTML

Turns the session's substantive output into one self-contained, designed HTML file. The model distills content into typed JSON and picks a design register; `Tools/Render.ts` owns layout, typography, and color. Delivery is local by default through the file path or `/opt/homebrew/bin/lavish-axi` review surface.

## Voice Notification

**When executing a workflow, do BOTH:**

1. **Send voice notification**:

   ```bash
   curl -s -X POST http://localhost:8888/notify \
     -H "Content-Type: application/json" \
     -d '{"message": "Running the Render workflow in the HTML skill to build a designed HTML artifact"}' \
     > /dev/null 2>&1 &
   ```

2. **Output text notification**:

   ```
   Running **Render** in **HTML**...
   ```

## Workflow Routing

| Workflow | Trigger | File |
|----------|---------|------|
| **Render** | `/HTML`, "HTML artifact", "render this as HTML" | `Workflows/Render.md` |

## Quick Reference

- Renderer: `bun ~/.claude/skills/HTML/Tools/Render.ts --json content.json --register dossier --out artifact.html`
- `--schema` prints the content JSON shape with an example; `--registers` lists registers.
- Registers: `dossier` (dark ink-green / orange, condensed display + typewriter - evidence files, red teams, investigations) and `ledger` (dark navy / gold, old-style serif - reports, finance, plans, comparisons). Alternate between them so consecutive outputs don't converge; add new registers to Render.ts rather than hand-styling one-offs.
- Output is portable and self-contained: inline CSS, fonts embedded as data URIs from local font files, and zero external requests.

## Gotchas

- **DOM-render screenshot pipelines drop CSS pseudo-element generated content** (counters, `::before` labels render in the browser but vanish from DOM-render captures, and they're invisible to text extraction). Render.ts therefore emits all numbering and labels as real DOM text. Never add CSS counters to a register. (Discovered 2026-07-11.)
- **Avoid external requests.** A Google Fonts link can fail offline. Embed fonts with data-URI `@font-face` or use deliberate system stacks.
- **The renderer emits a fragment-oriented document.** Keep its existing `<title>`, `<style>`, and body-content contract unless `Render.ts` itself is intentionally extended.
- **Local delivery is authoritative.** Hand over the generated path. Open it with lavish-axi only when an interactive local review is useful.
- **Numbered section markers must mean something.** Render.ts numbers sections (document order - legitimate). Don't number list items unless the content is a true sequence; use `list` blocks with bold lead-ins instead.

## Examples

**Example 1: After a research or red-team session**

```
User: "/HTML"
→ Render workflow: distill the session's findings into content JSON
→ bun Tools/Render.ts --json content.json --register dossier --out artifact.html
→ Open locally with `/opt/homebrew/bin/lavish-axi artifact.html` when review is useful
→ Hand over the local HTML path
```

**Example 2: Different subject, different register**

```
User: "render the quarterly cost analysis as HTML"
→ Same flow with --register ledger (tables, callouts)
→ Alternates the look from the last artifact so outputs don't converge
```
