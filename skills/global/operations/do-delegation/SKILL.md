---
name: do-delegation
version: 1.0.24
description: "Routes independent work through the harness's subagent tool: background execution, role briefs, worktree isolation, and coordinator-managed synthesis. USE WHEN parallel execution, agent team, swarm, spawn agents, fan out, divide and conquer, multi-agent, coordinate agents, custom agents."
effort: medium
---

# Delegation: Agent Orchestration and Parallelization

## What It Does

Delegates bounded work through the subagent tool your harness exposes (`Agent` in Claude Code and Pi; the equivalent tool elsewhere). It distinguishes direct work from delegated work, independent workers from coordinator-managed role cohorts, foreground results from background execution, and shared-checkout work from worktree-isolated edits.

## Core Rule

Delegate only when specialization, context isolation, or parallel execution saves more time than dispatch and verification cost.

Independent agents do not share context, memory, task state, or results. The parent conversation remains the coordinator. Every agent prompt must be complete, and the parent must validate each result before using it.

## Current Agent Contract

Use the subagent tool supplied by the active harness. Do not write source-code snippets that pretend to call the tool. If the harness has no subagent tool, do the work directly and say so.

Use only fields exposed by the current tool schema. The portable core, named as in Claude Code and Pi (map each to the active tool's equivalent field):

- `description`: short human-readable unit label
- `prompt`: complete task, context, constraints, permissions, and expected output
- `subagent_type`: an agent type confirmed by the active harness
- `run_in_background`: use when coordinator work can continue before the result is needed
- `max_turns`: optional bound for small one-shot work when supported
- `isolation`: request worktree isolation for editing work when supported

A new dispatch starts a fresh worker. Do not assume persistent peer messaging or a shared task list. For another round, launch a fresh agent and include the prior result plus the new question in its prompt.

### Harness Controls

- **Claude Code**: background completions notify the parent. Use `/tasks` to inspect running agents and `TaskStop` to cancel a specific task or agent.
- **Pi**: background completions notify the parent. Use the returned agent ID with `get_subagent_result` only when the result is needed and no completion result is already present. Use `steer_subagent` only for a supported running background agent that needs correction.
- **Other harnesses**: use their documented status and cancel controls.
- **All**: do not poll, sleep, tail output files, or duplicate work while a background agent is already handling it.

## Delegation Gate

Run this gate before dispatch:

1. **Direct-work check**: If the answer is already known or a few direct reads can settle it, use zero agents.
2. **Independence check**: Parallelize only units that do not depend on each other's unfinished results.
3. **Context check**: Include every fact the worker needs. Agents start without the parent's implicit working context.
4. **Write-conflict check**: Agents editing overlapping files must use supported worktree isolation or run serially.
5. **Verification reservation**: Keep enough time and context to inspect outputs, diffs, tests, and evidence.
6. **Fan-out bound**: Dispatch the smallest number of agents that covers the independent units.

Output the decision as: `RIGHT-SIZE: direct` or `RIGHT-SIZE: N agents, verification reserved`.

## Delegation Patterns

### 1. Direct Work

Use no agent when:

- the task is a single small file or lookup
- direct tools can answer quickly
- delegation setup would exceed the work
- the parent already has the needed context

### 2. Foreground Agent

Use one foreground subagent dispatch when the parent needs the result before proceeding.

Best for:

- bounded exploration that determines the next step
- an independent review before implementation continues
- a specialist answer needed by the current decision

The prompt must define the result contract. Verify the returned claim directly when it affects files or external state.

### 3. Parallel Background Agents

Use background agents for independent work while the parent can continue useful coordinator tasks.

Issue one subagent call per independent unit in the same assistant message, in background mode (`run_in_background: true` in Claude Code and Pi) when the harness supports it. Trust completion notifications and continue non-dependent work.

Best for:

- separate research questions
- analysis of separate files or modules
- independent review perspectives
- bounded data extraction across distinct inputs

Do not launch background agents when the next parent action immediately requires every result.

### 4. Worktree-Isolated Editing Agents

Use worktree isolation when parallel writers could touch the same checkout or when competing implementations must remain separate.

Requirements:

- confirm the current subagent tool supports worktree isolation
- give each worker a non-overlapping objective
- verify the worker's actual diff and test evidence
- integrate deliberately; do not assume changes appear in the parent checkout

If isolation is unavailable, serialize editing work.

### 5. Custom Role Briefs

When the user asks for custom, specialized, or contrasting agents, write one explicit role brief per perspective and dispatch a verified general-purpose agent type.

Each role brief needs:

- role and domain expertise
- stance or decision lens
- the complete shared task
- unique focus and exclusions
- evidence standard
- expected output

Distinct labels without distinct briefs do not produce meaningful diversity.

### 6. Coordinator-Managed Role Cohort

Use a role cohort when several complementary specialists contribute to one deliverable. This is the portable replacement for workflows that assumed persistent teams or shared task infrastructure.

The parent coordinator must:

1. Define the role catalog and select the minimum useful subset.
2. Give each fresh agent the full shared context and a role-specific result contract.
3. Dispatch independent first-round work in parallel.
4. Collect every required result.
5. Pass relevant prior results explicitly into any fresh follow-up or adjudication agent.
6. Verify claims, preserve material dissent, and synthesize one deliverable.

Use this pattern for cross-layer design, competing hypotheses, or review panels. Use simple parallel dispatch when every worker performs the same operation on a different input.

## Lightweight and Full Delegation

### Lightweight

Use for one-shot extraction, classification, summarization, or transformation where all input fits in the prompt and no tool use is expected.

- keep the prompt self-contained
- set a small `max_turns` only when the active schema supports it
- demand a compact output contract

### Full

Use when the worker must read files, search, browse, edit, test, or iterate.

- provide repository and scope context
- state allowed side effects
- define verification requirements
- do not impose a turn cap that prevents completion

Decision rule: if the worker needs tools or iteration, use full delegation.

## Parallel Dispatch Procedure

1. Enumerate independent units.
2. Remove units that are faster to handle directly.
3. Write one complete prompt per remaining unit.
4. Issue all independent `Agent` calls in one assistant message.
5. Continue only work that does not depend on pending results.
6. Collect completion results without polling.
7. Verify coverage, evidence, and any disk effects.
8. Retry a failed unit once with a corrected prompt or report the gap.
9. Synthesize the validated results.

## Verification Contract

An agent completion message is a claim, not proof.

For every writing agent:

- confirm the claimed file exists
- inspect the actual diff
- run the relevant checks
- confirm only allowed paths changed

For every research or analysis agent:

- check that the requested scope is complete
- inspect cited evidence when the claim matters
- compare conflicting results
- label unresolved uncertainty

Do not claim full delegation coverage when a required unit failed or returned an unverifiable result.

## Anti-Patterns

- delegating a lookup that direct tools can answer immediately
- spawning agents for one trivial file edit
- sending incomplete prompts that rely on parent-only context
- parallelizing dependent tasks
- allowing parallel writers to collide in one checkout
- assuming workers share state or can coordinate privately
- polling background agents or reading their output files
- trusting a worker's claimed disk changes without inspection
- launching more agents than the parent can verify
- naming specialized agent types that the active harness does not expose

## Examples

### Parallel research

Four independent topics can use four fresh agents with the same evidence and output contract. The parent validates and synthesizes their results.

### Cross-layer feature

API, UI, and test work may run in separate worktrees when interfaces are already defined. If interfaces are unresolved, design them first, then dispatch implementation.

### Competing hypotheses

Dispatch one agent per hypothesis with the same evidence standard. After results return, launch a fresh adjudicator agent with all prior results embedded in its prompt, then verify the adjudication directly.

## Gotchas

- Agent types differ by harness and installation. Confirm the available types before dispatch.
- Background execution is useful only when the parent has non-dependent work.
- Worktree changes require explicit inspection and integration.
- Fresh dispatches do not remember earlier workers. Include prior results in follow-up prompts.
- The parent owns final verification and user-facing synthesis.
