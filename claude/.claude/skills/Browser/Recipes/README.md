# Browser Recipes

Recipes are Markdown templates with parameter placeholders for reusable browser tasks.

## Format

```markdown
---
name: Recipe Name
description: What this recipe does
tool: chrome-devtools-axi | ai-agent
defaults:
  param1: default_value
---

# Recipe Name

1. Step using {param1}

{PROMPT}
```

## Parameters

- **`{PROMPT}`:** the user's full request
- **`{URL}`:** the target URL
- **`{param}`:** a custom parameter defined in `defaults`

Unresolved parameters remain visible so the executing agent can ask for them.

## Tool selection

| Tool | When to use |
| --- | --- |
| `chrome-devtools-axi` | Deterministic browser steps through `/opt/homebrew/bin/chrome-devtools-axi` |
| `ai-agent` | Page-dependent reasoning by a bounded general-purpose agent that loads the browser skill contract |

Recipes are discovered dynamically from `Recipes/*.md` and executed by `Workflows/Automate.md`.
