# Render Workflow

Turn the current session's output into a locally rendered and reviewable HTML file.

## Voice Notification

```bash
curl -s -X POST http://localhost:8888/notify \
  -H "Content-Type: application/json" \
  -d '{"message": "Running the Render workflow in the HTML skill to build a designed HTML artifact"}' \
  > /dev/null 2>&1 &
```

Running **Render** in **HTML**...

## Step 0 - Sufficiency Check

If the session contains no substantive output to render (nothing analyzed, researched, or produced yet), say so and ask what to render - don't invent content. If multiple candidate outputs exist, flag which one you picked (`⚠️ Rendering X, not Y; redirect if wrong.`) and proceed.

## Ideal State

- One self-contained HTML file rendering the session's actual output - real content, verbatim quotes kept verbatim, sources preserved, verification flags preserved.
- Rendered by `Tools/Render.ts` from a content JSON. No hand-authored one-off CSS. If the content genuinely doesn't fit the block types, extend Render.ts (new block type or register) so the next run benefits.
- Delivered as a local HTML file and optionally opened in the local lavish-axi review surface.

## Tool Contract

```bash
bun ~/.claude/skills/HTML/Tools/Render.ts --schema      # content JSON shape + example
bun ~/.claude/skills/HTML/Tools/Render.ts --registers   # available registers
bun ~/.claude/skills/HTML/Tools/Render.ts \
  --json <content.json> \
  --register <dossier|ledger> \
  --out <artifact.html>
```

Block types: `prose`, `callout`, `quote` (id + badge + quote/text + note + source), `list` (bold lead-ins), `cut` (strikethrough + stamp - for disclosed rejections), `table`, `group` (era/category separators). Badges listed in the register's `badgeSolid` render filled; others outlined.

### Register choice

| Content | Register |
| --------- | ---------- |
| Evidence file, red team, investigation, claim testing | `dossier` |
| Report, plan, comparison, metrics, finance | `ledger` |
| Same register as the previous artifact this week | pick the other one |

## Deliver + Verify (output contract)

1. Render to a local HTML file with `Tools/Render.ts` and confirm the file exists and is non-empty.
2. For interactive local review, run:

   ```bash
   /opt/homebrew/bin/lavish-axi <artifact.html>
   ```

   Follow the installed lavish skill contract for polling and ending a review session. Do not background an untracked poll.
3. If lavish-axi is unavailable or review was not requested, deliver the local HTML file path directly. The file itself is the durable output.
4. Report the local path and a one-line description. Do not claim a public URL or remote publication unless a separate, explicitly requested publishing workflow actually produced one.
