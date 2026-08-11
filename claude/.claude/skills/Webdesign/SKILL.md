---
name: Webdesign
version: 1.3.0
description: "Design and integrate web interfaces through DirectDesign, using the bundled frontend design philosophy and current Chrome verification. USE WHEN web design, UI design, landing page, dashboard design, frontend polish, design audit, redesign, prototype code, brutalist, editorial, retro, or maximalist UI. NOT FOR illustrations or logos (use Art), video (use Remotion), or unsupported external design-canvas sync commands."
license: Complete terms in LICENSE.txt
effort: medium
---

## Voice notification

```bash
curl -s -X POST http://localhost:8888/notify \
  -H "Content-Type: application/json" \
  -d '{"message": "Running the Webdesign skill", "voice_enabled": true}' > /dev/null
```

# Webdesign

DirectDesign is the only supported execution route. It writes production-grade interface code in the target project, follows the bundled aesthetic doctrine, respects the existing framework and tokens, and verifies rendered output with the installed Chrome CLI.

Historical canvas, bundle, and native-sync documents remain in this directory as reference material only. They are not routed because their required commands or runtime are unavailable. Do not invoke unsupported design slash commands or an absent browser fallback.

## Required workflow

1. Read `References/FrontendDesignPhilosophy.md`.
2. Inspect the target project's existing components, design tokens, typography, and framework conventions.
3. Run `Workflows/DirectDesign.md`.
4. Verify the rendered page with `/opt/homebrew/bin/chrome-devtools-axi` under the `chrome-devtools-axi` skill contract.
5. Confirm state changes with fresh snapshots and inspect screenshots at desktop and mobile widths.
6. Return changed file paths, the aesthetic declaration, verification evidence, and any limitation.

## Integration-aware operation

- Produce focused diffs against the existing app rather than an isolated demo unless the user explicitly asks for a standalone file.
- Preserve existing components and tokens unless replacement is requested.
- Match the existing build, test, lint, and accessibility conventions.
- Never overwrite unrelated UI while implementing a design request.
- For greenfield work, create the smallest runnable surface that satisfies the requested output contract.

## Workflow routing

| Workflow | Trigger | File |
| --- | --- | --- |
| **DirectDesign** | Any supported web or UI design request | `Workflows/DirectDesign.md` |

## Gotchas

- DirectDesign is self-contained. Its load-bearing aesthetic doctrine lives in `References/FrontendDesignPhilosophy.md`.
- Existing code and tokens are the source of truth. Do not invent a second design system beside them.
- A build passing is not visual verification. Open the rendered page, inspect a fresh accessibility snapshot, capture screenshots, and check console output.
- Preserve complete generation-prefixed refs from Chrome snapshots. On `STALE_REF`, take a fresh snapshot.
- If browser verification cannot run, report it as unavailable rather than claiming the page was visually verified.
