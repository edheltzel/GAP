---
title: Name the product Global Agent Policies (GAP) in the merged repo
type: docs
date: 2026-10-14
execution: code
---

# Name the product Global Agent Policies (GAP)

ATL-623 named the product Global Agent Policies (GAP) in `edheltzel/GAP#2` (merged 2026-09-30). PR #74 folded that repo into this one and its README does not carry the name; it says "a global agent policy". Restore the name where a reader first meets the product.

## Decisions

- Product name is **Global Agent Policies (GAP)** (user-directed, ATL-623; rejected: leaving "global agent policy" as the only expansion). Repo and CLI stay `gap`.

## Units

1. `README.md` title and lede: `# GAP` becomes `# Global Agent Policies (GAP)`; the lede says GAP, short for Global Agent Policies, is one set of rules every coding agent on the machine follows. The rest of the README is unchanged.
2. `CONCEPTS.md` `### GAP` entry: first sentence names it Global Agent Policies; the definition otherwise stays.

Out of scope: `global/AGENTS.md` (published to every harness; its wording is a separate policy edit), the GitHub repo description and rename (todo task 8).

## Verification

- `grep -n "Global Agent Policies" README.md CONCEPTS.md` shows both.
- README links still resolve; no em dashes added (`global/AGENTS.md` style rule).
- `go test ./...` unaffected (docs only).

## todo.txt

- Existing global task `ATL-623` closes when this PR merges.
