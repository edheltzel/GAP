---
name: Screenshot Compare
description: Take before and after screenshots of a URL for visual comparison
tool: chrome-devtools-axi
defaults:
  viewport: 1440x900
  wait: 2000
---

# Screenshot Compare

1. Set `BROWSER_CLI=/opt/homebrew/bin/chrome-devtools-axi`.
2. Resize to {viewport} using the CLI's `resize <w> <h>` command.
3. Open {URL}, wait {wait}ms, take a fresh snapshot, and save `/tmp/pai-browser/compare/before.png`.
4. Make the change described in PROMPT.
5. Open or reload {URL}, wait {wait}ms, take a fresh snapshot, and save `/tmp/pai-browser/compare/after.png`.
6. Confirm both screenshots show the intended page, then run `"$BROWSER_CLI" stop`.
7. Present both screenshots for comparison.

**Output:**

- Before screenshot path
- After screenshot path
- Summary of visible differences

{PROMPT}
