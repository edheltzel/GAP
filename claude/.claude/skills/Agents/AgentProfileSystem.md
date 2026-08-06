# Agent Profile System

This system loads one self-contained markdown role context and combines it with one complete task description.

## Current Contract

Role contexts are prompt material, not dispatchable agent types. The current files are:

- `ArchitectContext.md`
- `EngineerContext.md`
- `DesignerContext.md`
- `ArtistContext.md`
- `QATesterContext.md`

`Tools/LoadAgentContext.ts` reads a selected file from `~/.claude/skills/Agents/` and appends the task. It does not select a model or dispatch a worker.

Use a role context only when its focus adds value. The parent must still include repository context, constraints, expected evidence, output format, and the complete current task.

## Dispatch

Use the harness-exposed `Agent` tool after generating the enriched prompt. Never represent the tool as a JavaScript or TypeScript function.

In prose, provide:

- a short `description`
- the loader's complete output as `prompt`
- a `subagent_type` verified from the active tool schema
- optional background, isolation, or model fields only when the active schema supports and justifies them

Every dispatch creates a fresh agent. The context file and task prompt must therefore be sufficient without inherited conversation or external files.

For parallel work, issue one `Agent` tool call per independent worker in the same assistant message. In Pi, use completion notifications and `get_subagent_result` for a background result. In Claude Code, `/tasks` inspects active agents and `TaskStop` may cancel one.

## CLI

```bash
bun run ~/.claude/skills/Agents/Tools/LoadAgentContext.ts
bun run ~/.claude/skills/Agents/Tools/LoadAgentContext.ts Architect
bun run ~/.claude/skills/Agents/Tools/LoadAgentContext.ts Architect "Design a bounded plugin interface"
```

## Context File Format

```markdown
# Role Agent Context

**Role**: One-line purpose.

## Operating Principles

Self-contained guidance for this role.

## Required Evidence

What the worker must inspect, verify, and return.

## Output Format

A concise result contract.
```

Do not add static model aliases, removed system paths, or assumptions that another context is automatically loaded. References to optional external material must be existence-checked, read-only, untrusted, and unnecessary for success.

## Adding a Context

1. Add `[Role]Context.md` under `~/.claude/skills/Agents/`.
2. Make the role brief self-contained.
3. Define evidence and output expectations.
4. Verify that `LoadAgentContext.ts` lists and loads it.
5. Use a verified active worker type at dispatch time.

## Historical Note

The 2025 version treated role labels as profile types and recorded model preferences inside context files. That behavior is historical and non-operational. Current dispatch relies only on the active harness schema.
