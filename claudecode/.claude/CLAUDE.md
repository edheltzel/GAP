This file does nothing.

# Read the PAI system for system understanding and initiation
`read skills/PAI/SKILL.md`

## MEMORY - SQLite + FTS5 (Recall)

**CRITICAL: Before asking the user to repeat ANYTHING, search first.**

Your entire work history is in `~/.claude/memory.db`.

### MCP Server Tools

> **Note:** These tools require the `lmf-memory` MCP server to be loaded.
> If unavailable, use `mem` CLI commands below as fallback.

| Tool | Usage |
|------|-------|
| `context_for_agent` | Hybrid search for agent context |
| `memory_search` | Keyword search (FTS5) - USE before asking user to repeat |
| `memory_hybrid_search` | Natural language search (FTS5 + semantic + RRF fusion) |
| `memory_recall` | Recent LoA entries, decisions, breadcrumbs |
| `memory_add` | Add decision/learning/breadcrumb during sessions |
| `loa_show` | View full LoA entry |
| `memory_stats` | Database statistics |

### Recommended: Agent Context

**Before spawning any agent (Task tool), search for relevant context using `context_for_agent` (if available) or `mem` CLI.**

If MCP tools aren't loaded, use:
```bash
mem "relevant query" -k
```
and include results in the agent prompt.

### CLI Commands

```bash
# Search (default: hybrid FTS5 + semantic)
mem "natural language query"      # hybrid search (best results)
mem "query" -k                    # keyword only (FTS5)
mem "query" -v                    # vector only (semantic)

# Capture session wisdom
mem loa write "session title"

# Shortcut for session end
mem dump "session title"          # imports + LoA in one command

# Add structured records
mem add decision "chose X" --why "because Y"
mem add learning "problem" "solution"
mem add breadcrumb "context note"

# Stats
mem stats
```

### Tables

| Table | Contents |
|-------|----------|
| `messages` | Every conversation turn |
| `loa_entries` | Library of Alexandria (Fabric-extracted wisdom) |
| `decisions` | Architectural decisions with reasoning |
| `learnings` | Problems solved, patterns discovered |
| `breadcrumbs` | Context notes |
| `telos` | Purpose framework (optional) |
| `documents` | Standalone knowledge files (optional) |
