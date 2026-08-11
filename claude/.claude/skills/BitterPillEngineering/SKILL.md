---
name: BitterPillEngineering
version: 1.1.0
description: "Audits AI instruction sets for over-prompting. Applies six tests to each rule, then classifies it as CUT, RESOLVE, MERGE, EVALUATE, SHARPEN, MOVE, or KEEP. USE WHEN BPE, bitter pill, audit setup, trim instructions, simplify setup, or clean up CLAUDE.md. NOT FOR attacking an idea's logic (use RedTeam)."
effort: medium
---

# BitterPillEngineering

## Purpose

Audit an AI instruction set for rules that restate defaults, conflict, duplicate another source, preserve a one-off workaround, remain vague, or prescribe reasoning choreography without improving the required outcome.

The core test is: **Would a more capable model make this rule unnecessary?** If yes, the rule is probably scaffolding rather than a durable contract.

## Workflow routing

| Workflow | Trigger | File |
| --- | --- | --- |
| Audit | full setup, repository, or instruction hierarchy | `Workflows/Audit.md` |
| QuickCheck | one file or supplied rule block | `Workflows/QuickCheck.md` |

## Instruction discovery

Discover instructions from current, observable sources instead of assuming harness-specific settings keys:

1. Start with the active system, developer, and user instructions supplied by the harness.
2. Walk from the repository root to the target and read applicable `AGENTS.md`, `CLAUDE.md`, or equivalent instruction files.
3. Inspect harness configuration only when present, and report only keys that actually exist.
4. Include skill manifests, routing tables, hooks, or generated context only when the current harness demonstrably loads them.
5. Record the source and scope of every instruction reviewed.

Do not claim that a file is force-loaded merely because it exists.

## Six tests

For every rule, ask:

1. **Default behavior?** Does the current model or harness already enforce it?
2. **Contradiction?** Does it conflict with another applicable instruction?
3. **Redundancy?** Is the same contract already stated in the same scope?
4. **One-off fix?** Is it a workaround for an isolated failure without recurrence evidence?
5. **Vague?** Can compliance be verified consistently?
6. **HOW versus WHAT?** Does it script reasoning steps where an outcome, constraint, tool contract, or acceptance criterion would suffice?

Procedural instructions remain justified when they are a safety gate, verified gotcha, exact tool contract, or required output format.

## Classification

| Category | Action |
| --- | --- |
| Restates enforced default behavior | **CUT** |
| Conflicts with another applicable rule | **RESOLVE** |
| Duplicates another rule | **MERGE** |
| Preserves an unverified one-off workaround | **EVALUATE** |
| Cannot be tested consistently | **SHARPEN** or **CUT** |
| Rarely relevant in an always-loaded source | **MOVE** to on-demand guidance |
| Specific, current, actionable, and non-default | **KEEP** |

## Output

```text
## BitterPillEngineering Audit

Scope: [audited scope]
Instruction sources: [files and active context]
Rules evaluated: [count]

### CUT
- [rule]: [reason]

### RESOLVE
- [rule A] vs [rule B]: [recommended resolution]

### MERGE
- [locations]: [canonical location]

### EVALUATE
- [rule]: [evidence needed]

### SHARPEN or CUT
- [rule]: [testable replacement or reason to remove]

### MOVE
- [content]: [on-demand location]

### KEEP
- [rule]: [load-bearing purpose]

Estimated savings: [lines and approximate tokens]
Uncertainties: [unverified defaults or loading behavior]
```

## Guardrails

- Built-in behavior changes across model and harness versions. Test or mark uncertain rather than assuming.
- Check failure history before removing a rule that appears redundant.
- Preserve repository-specific safety, verification, and tool contracts even when they resemble general best practices.
- Keep the audit read-only unless the user explicitly requests edits.
