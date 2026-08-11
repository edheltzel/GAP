---
name: Summarize Page
description: Navigate to a URL and extract a summary of the page content
tool: ai-agent
defaults:
  format: markdown
---

# Summarize Page

1. Load the `chrome-devtools-axi` skill contract.
2. Run `/opt/homebrew/bin/chrome-devtools-axi open {URL}`.
3. Take a full snapshot and extract the main content area, excluding navigation, headers, and footers.
4. Summarize in {format} format.
5. Stop the browser session when finished.

**Output requirements:**

- Title of the page
- Main content summary (3-5 bullet points)
- Key links or calls to action found
- Word count estimate of main content

{PROMPT}
