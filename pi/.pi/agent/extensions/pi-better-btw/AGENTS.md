# Pi Better BTW

## Purpose

Provide `/btw`, a temporary side-conversation overlay inside Pi's interactive TUI.

## Ownership

- `index.ts` owns the command, in-memory agent session, overlay lifecycle, and UI.
- `index.test.ts` owns frame geometry, opacity, control-placement, and keybinding regressions.
- `README.md` owns user-facing installation, usage, and controls.

## Local Contracts

- Keep the side conversation in memory only; closing it must dispose the session.
- Preserve the current Pi model when creating the side conversation; show it and the
  current thinking level on the title row.
- Every rendered overlay row must match the width Pi supplies.
- Every rendered overlay cell must carry an explicit background. Cells left on the
  terminal's default background are transparent under terminal blur, which is what
  makes the screen behind show through the overlay body.
- The rendered row count must equal `resolveOverlayHeight`, which never exceeds
  `resolveOverlayMaxHeight`. Pi slices rows past its reserved box, taking the bottom
  border with them.
- Keep the title area quiet and the shortcut menu pinned to the bottom.
- Preserve keyboard behavior for scrolling, hiding, closing, and submitting.
- The pipe character is reserved for frame edges; inline dividers use the bullet.

## Work Guidance

- Use Pi's native TUI components and callback-provided theme.
- ANSI-aware truncation must happen before frame padding.
- Overlay geometry has one source of truth: the constants feeding both `overlayOptions`
  and `resolveOverlayHeight`. Never size the frame independently of what Pi reserves.
- Re-assert the background after any SGR that resets it. Nested content ends with
  `ESC[49m` or `ESC[0m` and would otherwise reopen a transparent gap mid-row.
- Terminals below `OVERLAY_MIN_HEIGHT` rows of reserved box cannot show the frame
  intact; that is a known limit, not a regression.

## Verification

Run from this directory:

```bash
npm test
npm run typecheck
```

After source changes, run `just pi` from the Atlas Config root and `/reload` in Pi.

## Child DOX Index

None.
