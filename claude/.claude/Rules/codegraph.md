# Codegraph (MANDATORY on coding projects)

When a project has a codegraph index (`mcp__codegraph__*` tools available or `.codegraph/` present), consult it BEFORE writing or editing code. Never assume structure, callers, or conventions that codegraph can answer.

## Rules

- **Explore before you write.** `codegraph_explore` is the primary call for "how does X work", architecture, traces, and where-is-X questions — usually one call replaces a grep+read loop.
- **Check blast radius before changing.** `codegraph_callers` / `codegraph_impact` on any symbol you modify; `codegraph_callees` to understand what it depends on.
- **Brownfield: validate DRY before writing.** Search for an existing definition (`codegraph_search`) and extend it rather than duplicating logic.
- **Greenfield: part of the external tooling loop** that keeps agents in sync with project standards.

## Delegation

Every spawned agent (workers, reviewers, subagents) MUST be instructed to use codegraph for codebase questions — "never assume" goes in the brief. An agent that guesses at structure when codegraph could answer is a defective dispatch.

## Staleness caveat

Codegraph indexes the **local working tree**, lagging writes by about a second — but it does NOT see unfetched remote state. If the local checkout is behind origin (e.g. PRs merged remotely without a local pull), codegraph answers describe the stale tree. For merged-state verification, ground truth is `git fetch origin` + `git show origin/main:<path>` and `gh` — codegraph is orientation only until the checkout catches up.
