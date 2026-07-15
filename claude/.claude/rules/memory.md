# MEMORY

Recall provides persistent, searchable memory. **Read the canonical installed guide:** `/Users/ed/.claude/Recall_GUIDE.md`

The guide and the live `recall-memory` MCP schemas are authoritative. This rule keeps only the session-critical contract.

## Session start

- Review the injected `## Recall — Session Memory (tiered)` context before the first response.
- Treat **L0 Identity** and **L1 Top Memory** as authoritative baseline context. Fetch L2/L3 only when needed.
- If L0 Identity is absent or empty, suggest `recall onboard` once during the session.

## Core rules

1. **Memory first:** Search Recall before `git log`, `git show`, or commit history.
2. **Search before asking:** Before asking the user to repeat information, call `memory_search` or `memory_hybrid_search`.
3. **Context for agents:** Before spawning any Task agent, call `context_for_agent` and include the returned context in the agent prompt.
4. **Record durable state:** Use `memory_add` for architectural decisions and immediately record user corrections as high-confidence learnings. Use `decision_update` when an active decision is superseded or reverted.
5. **Capture on request:** When the user says `/dump` or `/recall-dump`, call `memory_dump` immediately. It also works mid-session.
6. **Never persist secrets:** Redact keys, tokens, passwords, and credential-bearing snippets before `memory_add`. Before dumping a credential-bearing session, warn the user and obtain confirmation.

## Context resolution order

1. Injected SessionStart context
2. `memory_hybrid_search` for relevant history
3. `memory_recall` for recent LoA entries, decisions, and breadcrumbs
4. Native Claude memory and conversation context
5. `git log` or `git show` only for commit-level detail

## Current MCP call shapes

- `memory_search({ query: "search terms" })`
- `memory_hybrid_search({ query: "natural-language question" })`
- `memory_recall({ limit: 5 })`
- `memory_add({ type: "decision", content: "what", detail: "why" })`
- `memory_add({ type: "learning", content: "wrong -> right", confidence: "high", importance: 7 })`
- `decision_update({ id: 5, action: "supersede" })`
- `memory_dump({ title: "Session title", skip_fabric: true })`
- `context_for_agent({ agent_task: "what the agent will do", project: "project-name" })`
