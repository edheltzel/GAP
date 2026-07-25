# Pi Better BTW

## Purpose

Provide `/btw`, an ephemeral side-conversation overlay inside Pi's interactive TUI.

## Ownership

- `index.ts` owns the command, in-memory agent session, overlay lifecycle, and UI.
- `index.test.ts` owns frame-width and control-placement regressions.
- `README.md` owns user-facing installation, usage, and controls.

## Local Contracts

- Keep the side conversation in memory only; closing it must dispose the session.
- Preserve the current Pi model when creating the side conversation.
- Every rendered overlay row must match the width Pi supplies.
- Keep the title area quiet and the shortcut menu pinned to the bottom.
- Preserve keyboard behavior for scrolling, hiding, closing, and submitting.

## Work Guidance

- Use Pi's native TUI components and callback-provided theme.
- ANSI-aware truncation must happen before frame padding.
- Keep viewport-height calculations synchronized with fixed chrome rows.

## Verification

Run from this directory:

```bash
npm test
npm run typecheck
```

After source changes, run `just pi` from the Atlas Config root and `/reload` in Pi.

## Child DOX Index

None.
