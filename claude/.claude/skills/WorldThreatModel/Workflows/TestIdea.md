---
workflow: TestIdea
mode: single-run
---

# Test Idea Against World Threat Models

Test an idea, strategy, investment, brand, or concept against 11 user-owned world model files spanning six months to 50 years.

## When to Use

- The user says "test this idea," "how will this hold up," "test my strategy," or "stress test this."
- The user wants temporal viability analysis across multiple horizons.
- The user understands that these are scenarios, not predictions.

## Required Runtime Data

This restored skill does not ship the world model corpus. It must not invent, approximate, or silently replace missing models.

The required directory is:

`~/.claude/LIFEOS/MEMORY/RESEARCH/WorldModels/`

The required files are:

1. `6-month.md`
2. `1-year.md`
3. `2-year.md`
4. `3-year.md`
5. `5-year.md`
6. `7-year.md`
7. `10-year.md`
8. `15-year.md`
9. `20-year.md`
10. `30-year.md`
11. `50-year.md`

`INDEX.md` is useful for freshness metadata but does not replace any horizon file.

## Required Frameworks

Framework availability depends on the active installation. Resolve each required skill in the current harness before analysis begins.

| Tier     | Required skills                              |
|----------|----------------------------------------------|
| Fast     | None beyond this skill                       |
| Standard | FirstPrinciples, RedTeam                     |
| Deep     | FirstPrinciples, RedTeam, Council            |

Deep also requires the active harness to expose `web_search`, `source_check`, and
`fetch_content` or equivalent interfaces that return cited current sources. The output
template at `~/.claude/skills/WorldThreatModel/OutputFormat.md` is required for every tier.

If a required skill, research interface, or the output template is unavailable, STOP. Do
not substitute an informal approximation and do not downgrade the tier unless the user
explicitly chooses a lower tier whose prerequisites pass.

## Tier Detection

- "fast" or "quick" selects Fast.
- "deep," "thorough," or "comprehensive" selects Deep.
- No modifier selects Standard.

State the selected tier before dispatch.

## Workflow

### Step 0: Fail-Closed Preflight

Run the model-file check before reading content, invoking frameworks, or launching agents:

```bash
model_root="$HOME/.claude/LIFEOS/MEMORY/RESEARCH/WorldModels"
required_models=(
  6-month.md
  1-year.md
  2-year.md
  3-year.md
  5-year.md
  7-year.md
  10-year.md
  15-year.md
  20-year.md
  30-year.md
  50-year.md
)
missing=()
for model in "${required_models[@]}"; do
  [ -f "$model_root/$model" ] || missing+=("$model")
done
if [ "${#missing[@]}" -ne 0 ]; then
  printf 'World models incomplete at %s\n' "$model_root" >&2
  printf 'Missing: %s\n' "${missing[*]}" >&2
  exit 1
fi
```

Then verify `OutputFormat.md` exists and resolve every framework required by the selected tier through the active skill interface.

On any preflight failure, report the exact missing files or frameworks, then STOP. Do not run any analysis. The user may populate the model corpus through a separately verified process and retry.

### Step 1: Check Freshness

Read the frontmatter or update date in each required model. If any model is older than 30 days, warn before continuing. If freshness cannot be determined, label it unknown. Staleness is a warning after completeness passes, not permission to fabricate an update.

### Step 2: Notify

```bash
curl -s -X POST http://localhost:8888/notify \
  -H "Content-Type: application/json" \
  -d '{"message": "Testing your idea against eleven world models"}' \
  > /dev/null 2>&1 &
```

### Step 3: Decompose the Idea

Write:

1. A one or two sentence statement of the idea
2. Core assumptions about markets, technology, culture, regulation, competition, and timing
3. Success dependencies that must remain true

For Standard and Deep, invoke the verified FirstPrinciples skill and require it to classify:

- hard constraints
- soft constraints
- unvalidated assumptions

If the required framework invocation fails, STOP and report the failure. Do not continue with an improvised substitute.

### Step 4: Analyze the Horizons

Read only the 11 validated files from the required runtime directory.

#### Fast Tier

The parent analyzes all 11 models directly. For each horizon, return:

- verdict
- confidence
- two or three key factors
- assumptions that fail or strengthen

Then produce the abbreviated format from `OutputFormat.md`.

#### Standard Tier

Use the current `Agent` tool exposed by the active Pi or Claude harness.

In one assistant message, issue one `Agent` call per horizon. Each call must include:

- a horizon-specific `description`
- a complete `prompt` containing the idea, decomposed assumptions, the exact model file path, evidence requirements, and output contract
- `subagent_type` set to a verified general-purpose agent type
- `run_in_background` set to `true` when coordinator work can continue while analyses run

Each horizon prompt must require:

- verdict and confidence
- key conditions from that model
- assumption-by-assumption testing
- evidence quoted or referenced from the assigned model
- risks, opportunities, and uncertainty

Trust harness completion notifications. Do not poll or read agent output files. In Claude Code, use `/tasks` to inspect active work and `TaskStop` to cancel a failed run. In Pi, use the returned agent ID with `get_subagent_result` only when the result is needed and no completion result is already present.

After all 11 analyses complete, invoke the verified RedTeam skill with the idea, assumptions, and all horizon results. If any required horizon result or RedTeam result is missing, STOP and report the incomplete coverage.

Synthesize using the full format in `OutputFormat.md`.

#### Deep Tier

Deep runs the Standard procedure plus:

1. Use `web_search` for current developments relevant to each horizon, `source_check` for load-bearing claims, and `fetch_content` for primary-source passages. Keep current research separate from the stored model text and cite it.
2. Require deeper horizon prompts with counterfactuals, leading indicators, and mitigation triggers.
3. Invoke the verified RedTeam skill for cross-horizon adversarial analysis.
4. Invoke the verified Council skill with all validated horizon and adversarial results.
5. Synthesize the complete format from `OutputFormat.md`.

If current-source research, RedTeam, Council, or any horizon analysis fails, STOP. Do not silently omit the section or claim Deep coverage.

### Step 5: Validate the Synthesis

Before delivery, confirm:

- all 11 horizons appear exactly once
- every conclusion traces to a stored model, cited research, or named framework result
- stored-model evidence and new research are clearly distinguished
- confidence reflects both model freshness and analytical uncertainty
- disagreement between horizons or frameworks is preserved
- no missing framework or horizon is represented as completed

### Step 6: Deliver and Notify

Use `OutputFormat.md`. Present scenarios and probability ranges, not predictions.

```bash
curl -s -X POST http://localhost:8888/notify \
  -H "Content-Type: application/json" \
  -d '{"message": "World model analysis complete"}' \
  > /dev/null 2>&1 &
```

## Failure Rules

- Missing model file: fail preflight and stop.
- Missing required framework: fail preflight and stop.
- Missing output template: fail preflight and stop.
- Failed required agent: retry once with a corrected complete prompt, then stop if still incomplete.
- Stale model: warn and label confidence; do not fabricate a refresh.
- User requests downgrade after failure: rerun preflight for the newly selected tier before analysis.
