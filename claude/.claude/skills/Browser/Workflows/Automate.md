# Automate Workflow

## Voice notification

```bash
curl -s -X POST http://localhost:8888/notify \
  -H "Content-Type: application/json" \
  -d '{"message": "Running the Automate workflow in the Browser skill to execute a recipe template"}' \
  > /dev/null 2>&1 &
```

Running **Automate** in **Browser**...

Load a parameterized recipe, resolve its inputs, and execute it through the current browser contract.

## Steps

1. Discover recipe files under `skills/Browser/Recipes/*.md`, excluding `README.md`.
2. Match the requested recipe name. If ambiguous, list matches and ask which one.
3. Parse `name`, `description`, `tool`, and `defaults` from frontmatter.
4. Resolve explicit user overrides, defaults, `{PROMPT}`, and `{URL}` in that order.
5. Use `chrome-devtools-axi` when the recipe is deterministic. Use a bounded general-purpose agent for `ai-agent` recipes.
6. For browser execution, load the manifest skill contract and call `/opt/homebrew/bin/chrome-devtools-axi` sequentially. Preserve generation-prefixed refs and take a fresh snapshot after every state-changing action.
7. Return command results, screenshot paths, verification snapshots, and errors.
8. Stop the browser session when finished.

If `tool` is missing, default to `chrome-devtools-axi`. Do not translate a recipe into commands for an executable that is not installed.
