---
name: Browser
description: "Browser automation through the installed chrome-devtools-axi CLI. Opens pages, reads accessibility snapshots, clicks and fills by generation-scoped refs, captures screenshots, evaluates JavaScript, and inspects console or network activity. Workflows: ReviewStories, Automate, and Update. USE WHEN browser automation, screenshot, dev server test, form fill, extract rendered data, review stories, automate recipe, console debugging, or network inspection. NOT FOR simple static URL fetching (use fetch_content), residential-proxy crawling (use BrightData), or social actor scraping (use Apify)."
version: 10.1.0
effort: medium
---

## MANDATORY: Voice Notification (REQUIRED BEFORE ANY ACTION)

Send the notification before running a workflow:

```bash
curl -s -X POST http://localhost:8888/notify \
  -H "Content-Type: application/json" \
  -d '{"message": "Running the WORKFLOWNAME workflow in the Browser skill to ACTION"}' \
  > /dev/null 2>&1 &
```

Then output:

```
Running the **WorkflowName** workflow in the **Browser** skill to ACTION...
```

# Browser

Use the installed `/opt/homebrew/bin/chrome-devtools-axi` executable and follow the `chrome-devtools-axi` manifest skill contract. Do not publish commands for a direct `agent-browser` executable; that executable is not installed on this machine.

## Core workflow

```bash
BROWSER_CLI=/opt/homebrew/bin/chrome-devtools-axi
"$BROWSER_CLI" open https://example.com
"$BROWSER_CLI" snapshot
"$BROWSER_CLI" click @g1:button-1
"$BROWSER_CLI" snapshot
"$BROWSER_CLI" screenshot /tmp/example.png
"$BROWSER_CLI" stop
```

1. `open <url>` starts or reuses the persistent Chrome bridge and returns an accessibility snapshot.
2. Interactive refs include a generation prefix such as `g1:`. Pass refs back exactly as printed.
3. A rerender can invalidate refs. On `STALE_REF`, run a fresh `snapshot` and retry with the new ref.
4. After every state-changing action, confirm the outcome with a fresh `snapshot`, `eval`, or screenshot.
5. Use `console` and `network` for diagnosis. Use `lighthouse` or `perf-start` and `perf-stop` for audits.
6. Run `stop` when the browser session is no longer needed.

## Common commands

```bash
BROWSER_CLI=/opt/homebrew/bin/chrome-devtools-axi
"$BROWSER_CLI" open https://example.com
"$BROWSER_CLI" snapshot --full
"$BROWSER_CLI" fill @g1:textbox-1 "hello"
"$BROWSER_CLI" fillform @g2:textbox-1="Ada" @g2:textbox-2="ada@example.com"
"$BROWSER_CLI" click @g2:button-1
"$BROWSER_CLI" snapshot
"$BROWSER_CLI" eval 'document.title'
"$BROWSER_CLI" console
"$BROWSER_CLI" network
"$BROWSER_CLI" resize 390 844
"$BROWSER_CLI" screenshot /tmp/mobile.png
"$BROWSER_CLI" stop
```

For authenticated work, use the Chrome session managed by the CLI. If login requires a password, hardware approval, or another human-only step, drive the browser to that exact boundary and request only the missing action. Never claim authentication succeeded without a confirming snapshot.

## Delegating browser work

A general-purpose agent can load the `chrome-devtools-axi` skill and use the installed CLI. Give each agent a bounded URL/task packet and require it to return screenshot paths plus fresh-snapshot evidence after mutations. Do not invent session flags or profile commands that are outside the current CLI contract.

## Workflow routing

| Workflow | Trigger | File |
| --- | --- | --- |
| ReviewStories | review stories, run stories, UI review, validate stories | `Workflows/ReviewStories.md` |
| Automate | automate, recipe, template, or a recipe name | `Workflows/Automate.md` |
| Update | update, check version, verify browser tooling | `Workflows/Update.md` |

## Stories

Story files live in `Stories/`. Each action target should be human-readable, and each assertion should produce a clear pass or fail. Reviewers must refresh the snapshot after each action before evaluating assertions.

## Recipes

Recipes live in `Recipes/` and use either `chrome-devtools-axi` for deterministic browser steps or `ai-agent` for page-dependent reasoning. The Automate workflow resolves placeholders and executes the resulting steps through the installed CLI.

## Gotchas

- Refs are generation-scoped. Preserve the complete ref, including `g<N>:`.
- A successful click command does not prove the page changed. Verify with a fresh snapshot.
- Relative screenshot and trace paths resolve from the command's working directory.
- Large network bodies should be saved with `network-get --response-file` rather than printed into the transcript.
- Use `fetch_content` instead of launching Chrome when the page is public and static.
- If the installed Chrome path cannot access a target because of CAPTCHA or residential-IP controls, route to BrightData's optional credentialed escalation rather than an absent fallback executable.
