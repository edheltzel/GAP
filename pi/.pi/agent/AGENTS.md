<!-- BEGIN COMPOUND PI TOOL MAP -->
## Compound Engineering (Pi compatibility)

This block is managed by compound-plugin.

Pi extensions used by this plugin:
- Required: `pi-subagents` (by nicobailon) provides the `subagent` tool used by skills that dispatch parallel agents
- Recommended: `pi-ask-user` (by edlsh) provides the `ask_user` tool; skills fall back to numbered options in chat when it is missing

Install with:
  pi install npm:pi-subagents
  pi install npm:pi-ask-user
<!-- END COMPOUND PI TOOL MAP -->

## Atlas Flow safety note

Before changing Pi's global package set or implementing `/atlas-flow`, read `/Users/ed/Developer/atlas-config/.atlas/plans/2026-05-16-atlas-flow-research-digest.md` for the minimal-global baseline, flow manifest schema direction, safety rules, and GitHub issue roadmap.

## MEMORY

You have persistent memory via Recall. **Read the full guide:** ~/.pi/agent/Recall_GUIDE.md

Core rules:
1. Before asking user to repeat anything → search first with `recall-memory_memory_search`
2. When decisions are made → record with `recall-memory_memory_add`
3. End of session when user says `/dump` → call `recall-memory_memory_dump({ title: "Descriptive Title" })`
4. Dumped sessions are immediately searchable from new sessions via `recall-memory_memory_search`

Tool syntax:
- `recall-memory_memory_search({ query: "search terms" })`
- `recall-memory_memory_add({ type: "decision", content: "what", detail: "why" })`
- `recall-memory_memory_dump({ title: "Session title", skip_fabric: true })`
- `recall-memory_context_for_agent({ task_description: "what the agent will do" })`
