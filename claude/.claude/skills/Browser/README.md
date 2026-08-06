# Browser Automation

**Executable:** `/opt/homebrew/bin/chrome-devtools-axi`

The Browser skill follows the installed `chrome-devtools-axi` contract for Chrome navigation, accessibility snapshots, ref-based interaction, screenshots, JavaScript evaluation, console inspection, and network inspection.

## Quick start

```bash
BROWSER_CLI=/opt/homebrew/bin/chrome-devtools-axi
"$BROWSER_CLI" open https://example.com
"$BROWSER_CLI" snapshot
"$BROWSER_CLI" click @g1:link-1
"$BROWSER_CLI" snapshot
"$BROWSER_CLI" screenshot /tmp/example.png
"$BROWSER_CLI" stop
```

Keep the complete generation-prefixed ref from each snapshot. If an action returns `STALE_REF`, take a fresh snapshot and use the replacement ref. Confirm every state change with another snapshot, `eval`, or screenshot.

## Files

| File | Purpose |
| --- | --- |
| `SKILL.md` | Current executable contract and workflow routing |
| `Stories/` | YAML user story definitions |
| `Recipes/` | Parameterized workflow templates |
| `Workflows/ReviewStories.md` | Fan out story checks to bounded reviewer agents |
| `Workflows/Automate.md` | Resolve and execute recipe templates |
| `Workflows/Update.md` | Verify installed browser tooling |
