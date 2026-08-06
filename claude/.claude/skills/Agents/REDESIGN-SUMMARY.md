# Agent Profile System Redesign Summary

**Historical date:** 2025-12-18
**Current status:** Non-operational historical record updated for the current harness contract

## Historical Change

The 2025 redesign replaced elaborate YAML profiles with one markdown context file per role and a small loader. The useful design survived:

- one readable role brief per specialty
- no duplicated knowledge blocks
- a loader that concatenates role context and task text
- role labels used as prompt composition, not as guaranteed worker types

The old YAML profiles and loader are not required by the current Agents skill.

## Current Interpretation

Context files now contain self-sufficient role guidance. They do not assume removed system files, automatically loaded context, static model aliases, or persistent worker state.

`Tools/LoadAgentContext.ts` returns:

- the role label
- the role context text
- an enriched prompt when a task is supplied

It does not dispatch a worker or choose a model.

## Current Dispatch

Use the harness-exposed `Agent` tool. Pass the complete enriched prompt and select only a worker type exposed by the active schema. For independent workers, issue one tool call per worker in the same assistant message.

Fresh agents receive no implicit context. Include the full task, constraints, repository details, required evidence, and output contract in every prompt.

## Verification

```bash
bun run ~/.claude/skills/Agents/Tools/LoadAgentContext.ts
bun run ~/.claude/skills/Agents/Tools/LoadAgentContext.ts Architect
bun run ~/.claude/skills/Agents/Tools/LoadAgentContext.ts Engineer "Implement a focused change"
```

## Historical Boundary

Earlier documents described system-wide context injection, profile-level model selection, and pseudo-code dispatch. Those descriptions are preserved here only as this explicit historical note. They are not supported operating instructions.
