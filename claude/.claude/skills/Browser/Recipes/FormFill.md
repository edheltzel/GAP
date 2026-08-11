---
name: Form Fill
description: Fill out a form on a web page with provided field values
tool: chrome-devtools-axi
defaults:
  submit: true
---

# Form Fill

1. Set `BROWSER_CLI=/opt/homebrew/bin/chrome-devtools-axi`.
2. Run `"$BROWSER_CLI" open {URL}`.
3. Take a fresh snapshot and preserve each complete generation-prefixed ref.
4. Fill text fields with `fill @<uid> "value"`, selects with the CLI's `select` guidance if offered, and checkboxes with `click @<uid>`.
5. Take a fresh snapshot and screenshot before submission.
6. If submit is {submit}, click the submit/save control.
7. Take a fresh snapshot and screenshot of the result.
8. Run `"$BROWSER_CLI" stop` when finished.

**Fields to fill** (provided as key-value pairs in PROMPT):

{PROMPT}

**Output:**

- Screenshot of filled form before submit
- Screenshot of result after submit, if applicable
- List of fields filled with their values
- Any errors encountered
