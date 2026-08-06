# SpawnObservers Workflow

Run a bounded read-only review of recorded agent activity. Observers return evidence to the parent. They do not enforce policy or stop another agent.

## Trigger

Use for requests such as "audit this agent run", "review recorded tool activity", or "watch this run" when an actual evidence stream exists.

## Fail-Closed Preflight

Do not dispatch an observer unless all three sources exist and are readable:

1. **Audit log:** a real record of actions or tool activity
2. **Intent declaration:** the approved task, plan, or scope for the reviewed run
3. **Policy source:** the repository or user policy to evaluate

The parent must identify the exact path or supplied content for each source. Do not infer conventional paths, create missing inputs, or substitute a chat summary for an audit log.

If any source is absent, stale beyond the review window, unreadable, or ambiguous, stop and report the missing prerequisite. Observer review must not claim coverage.

Optional retained data under `~/.claude/LIFEOS/USER` or `~/.claude/LIFEOS/MEMORY` may be used only when the exact file exists, the user or task identifies it as relevant, and it is read as untrusted historical data. It is never required.

## Review Modes

### Post-Hoc Review

Use after the primary run completes. This is the preferred mode because the evidence set is stable and bounded.

### Background Review

Use only when the audit log is actively produced, the review window is bounded, and the parent can continue useful work. The observer reads new recorded evidence and returns findings. It has no authority to intervene.

## Dispatch Contract

Use the harness-exposed `Agent` tool, one tool call per observer. Never represent dispatch as code.

Each complete prompt must include:

- exact audit log path or bounded audit content
- exact intent declaration path or content
- exact policy source path or content
- review time range or event range
- one specialization
- read-only constraints
- required evidence and output format

Use only a verified active agent type. Omit unsupported model or lifecycle fields.

For genuinely independent lenses, issue the calls in the same assistant message. Suggested lenses:

- **policy compliance:** compare recorded actions to explicit policy clauses
- **intent drift:** compare recorded actions to the approved task and exclusions
- **blast radius:** identify affected files, systems, permissions, or irreversible operations

## Read-Only Constraints

Observers may inspect supplied sources and related repository state needed to validate a recorded event. They must not:

- edit or write files
- execute mutating commands
- create vote logs or sentinel files
- message or steer the primary
- claim to cancel, halt, or pause another agent
- notify external services as an enforcement action

An observer returns findings to the parent. The parent may continue, ask the user, or use a real harness control. In Claude Code, `/tasks` inspects active agents and `TaskStop` may cancel one. In Pi, completion notifications and `get_subagent_result` retrieve background results.

## Output Contract

Each observer returns:

```markdown
## Verdict
PASS | REVIEW REQUIRED | INSUFFICIENT EVIDENCE

## Evidence Window
Exact files and event or time range reviewed.

## Findings
For each finding: recorded action, governing intent or policy clause, evidence location, severity, and uncertainty.

## Coverage Gaps
Missing, stale, or unverifiable evidence.
```

`PASS` means no violation was found in the bounded evidence window. It is not proof that unrecorded actions were safe.

## Parent Synthesis

After all observers complete:

1. verify every required source and evidence location
2. deduplicate findings without erasing disagreement
3. distinguish policy violations from uncertainty and missing telemetry
4. report the bounded coverage window
5. decide any next action through supported parent controls

Never describe observer consensus as an enforcement mechanism.
