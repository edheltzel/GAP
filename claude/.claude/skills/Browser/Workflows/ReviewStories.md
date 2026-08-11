# ReviewStories Workflow - Parallel User Story Validation

## Voice Notification

```bash
curl -s -X POST http://localhost:8888/notify \
  -H "Content-Type: application/json" \
  -d '{"message": "Running the ReviewStories workflow in the Browser skill to validate user stories"}' \
  > /dev/null 2>&1 &
```

Running **ReviewStories** in **Browser**...

---

Fan out YAML user stories to parallel general-purpose reviewer agents and aggregate results.

## When to Use

- Validating that a web app meets user story requirements
- Running regression checks across multiple pages/flows
- Batch UI validation after deployment

## Trigger Words

"review stories", "run stories", "ui review", "validate stories", or referencing a `.yaml` story file

## Input

Either:

- **Specific file:** `Stories/HackerNews.yaml` (or full path)
- **All stories:** "all" - globs `Stories/*.yaml`

## Steps

### 1. Discover Stories

```
# Specific file
Read the specified .yaml file from skills/Browser/Stories/

# All stories
Glob: skills/Browser/Stories/*.yaml
```

### 2. Parse YAML

For each `.yaml` file:

- Extract `name`, `url`, and `stories[]` array
- Each story in the array becomes one reviewer-agent dispatch

### 3. Fan Out to Parallel parallel reviewer agents

For each individual story, spawn one general-purpose reviewer agent via the Task tool. **All Task calls go in a single message** for true parallelism.

**Maximum 8 parallel reviewer agents per invocation.** If more than 8 stories, batch into groups of 8.

**Prompt template per reviewer agent:**

```
You are validating a user story. Execute it and report results.

Story file: {file_name}
Base URL: {url}

story:
  name: "{story.name}"
  url: "{url}"
  steps:
{formatted_steps}
  assertions:
{formatted_assertions}

Execute this story. Follow your 5-phase workflow. Return the JSON report AND the RESULT: sentinel line.
```

### 4. Collect Results

After all parallel reviewer agents complete, parse each agent's output for the `RESULT:` sentinel line:

```
RESULT: PASS | Steps: N/M | Assertions: X/Y | Duration: Zs
RESULT: FAIL | Steps: N/M | Assertions: X/Y | Failed: "reason" | Duration: Zs
```

### 5. Aggregate Report

Produce a summary table:

```
## Story Review Results

| Story | File | Result | Steps | Assertions | Duration |
|-------|------|--------|-------|------------|----------|
| Front page loads | HackerNews.yaml | PASS | 1/1 | 2/2 | 8s |
| First story clickable | HackerNews.yaml | PASS | 2/2 | 1/1 | 12s |
| Login flow | ExampleApp.yaml | FAIL | 3/4 | 1/2 | 15s |

**Summary: 2/3 PASS | 1/3 FAIL**
```

Include screenshot paths from each reviewer's report for failed stories.

## Design Decisions

- **Independent Agent dispatch.** Issue one current `Agent` tool call per story in the same assistant message, with a complete self-contained reviewer prompt.
- **Stories as YAML text in prompt.** This avoids extra file discovery and keeps each reviewer scoped to one story.
- **RESULT sentinel parsing.** Parse the final sentinel instead of extracting JSON from freeform reviewer output.
- **Bounded concurrency.** Cap one batch at eight reviewers, or lower when the active harness reports a smaller safe limit.

## Error Handling

- If a YAML file fails to parse → report the parse error, skip that file
- If a reviewer agent times out → mark that story as TIMEOUT in the summary
- If no RESULT sentinel found → mark as UNKNOWN and include raw agent output for debugging
