# Recall (LMF3) Integration — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Install Recall as a standalone complement to PAI memory — zero PAI code changes, both systems run in parallel.

**Architecture:** Recall runs as an independent MCP server (`mem-mcp`) + Stop hook (`SessionExtract.ts`). Binaries installed globally via `bun install -g`. Hook symlinked from the AtlasMemory repo. MCP registered in `~/.claude/.mcp.json`. CLAUDE.md gets tool usage instructions appended.

**Tech Stack:** Bun, SQLite (FTS5 + WAL), Ollama (nomic-embed-text embeddings, optional), MCP protocol

---

### Task 1: Build Recall from Source

**Files:**
- Work in: `/Users/ed/Developer/AI/AtlasMemory`

**Step 1: Install dependencies**

Run: `cd /Users/ed/Developer/AI/AtlasMemory && bun install`
Expected: Dependencies installed, no errors.

**Step 2: Build the project**

Run: `cd /Users/ed/Developer/AI/AtlasMemory && bun run build`
Expected: `dist/index.js` and `dist/mcp-server.js` created. Both have `#!/usr/bin/env bun` shebang.

**Step 3: Verify build output**

Run: `ls -la /Users/ed/Developer/AI/AtlasMemory/dist/`
Expected: `index.js`, `mcp-server.js`, and their `.d.ts` type files exist.

---

### Task 2: Install Binaries Globally

**Step 1: Install globally via bun**

Run: `cd /Users/ed/Developer/AI/AtlasMemory && bun link`
Expected: `mem` and `mem-mcp` available on PATH.

**Step 2: Verify `mem` CLI works**

Run: `mem --help`
Expected: Shows commander help output with subcommands (search, add, stats, loa, dump, etc.).

**Step 3: Verify `mem-mcp` exists**

Run: `which mem-mcp`
Expected: Returns a valid path (e.g., `~/.bun/bin/mem-mcp`).

---

### Task 3: Initialize the Database

**Step 1: Run init**

Run: `mem init`
Expected: Creates `~/.claude/memory.db`. Output confirms database initialized with tables.

**Step 2: Verify database exists**

Run: `ls -la ~/.claude/memory.db`
Expected: File exists, size > 0.

**Step 3: Verify tables**

Run: `mem stats`
Expected: Shows record counts for all tables (all zeros — fresh database).

---

### Task 4: Symlink the Extraction Hook

**Step 1: Verify target directory exists**

Run: `ls -d ~/.claude/hooks/`
Expected: Directory exists (PAI's hook directory, stowed from atlas-config).

**Step 2: Create symlink**

Run: `ln -s /Users/ed/Developer/AI/AtlasMemory/hooks/SessionExtract.ts ~/.claude/hooks/SessionExtract.ts`
Expected: Symlink created with no errors.

**Step 3: Verify symlink**

Run: `ls -la ~/.claude/hooks/SessionExtract.ts`
Expected: Shows symlink pointing to `/Users/ed/Developer/AI/AtlasMemory/hooks/SessionExtract.ts`.

---

### Task 5: Register the Stop Hook in settings.json

**Files:**
- Modify: `claudecode/.claude/settings.json` — the `hooks.Stop` array

The existing `Stop` section has only `StopOrchestrator.hook.ts`. Add a second entry for `SessionExtract.ts`.

**Step 1: Add the hook entry**

In `claudecode/.claude/settings.json`, change the `"Stop"` section from:

```json
"Stop": [
  {
    "hooks": [
      {
        "type": "command",
        "command": "${PAI_DIR}/hooks/StopOrchestrator.hook.ts",
        "timeout": 30000
      }
    ]
  }
]
```

To:

```json
"Stop": [
  {
    "hooks": [
      {
        "type": "command",
        "command": "${PAI_DIR}/hooks/StopOrchestrator.hook.ts",
        "timeout": 30000
      }
    ]
  },
  {
    "hooks": [
      {
        "type": "command",
        "command": "${PAI_DIR}/hooks/SessionExtract.ts",
        "timeout": 30000
      }
    ]
  }
]
```

Note: Each entry in the Stop array is a separate hook group. The symlink at `~/.claude/hooks/SessionExtract.ts` means `${PAI_DIR}/hooks/SessionExtract.ts` resolves correctly since `PAI_DIR=/Users/ed/.claude`.

**Step 2: Validate JSON**

Run: `bun -e "JSON.parse(require('fs').readFileSync('claudecode/.claude/settings.json', 'utf-8')); console.log('Valid JSON')"`
Expected: `Valid JSON`

**Step 3: Commit**

```bash
git add claudecode/.claude/settings.json
git commit -m "feat: register Recall SessionExtract.ts as Stop hook"
```

---

### Task 6: Create MCP Server Configuration

**Files:**
- Create: `~/.claude/.mcp.json`

**Step 1: Find the mem-mcp binary path**

Run: `which mem-mcp`
Expected: Returns path like `/Users/ed/.bun/bin/mem-mcp`.

**Step 2: Create `.mcp.json`**

Write to `~/.claude/.mcp.json`:

```json
{
  "mcpServers": {
    "lmf-memory": {
      "command": "mem-mcp",
      "args": []
    }
  }
}
```

Note: This file is NOT managed by stow (it lives at `~/.claude/.mcp.json` directly). It is gitignored by Claude Code and machine-specific. Do not add it to the atlas-config repo.

**Step 3: Verify file**

Run: `cat ~/.claude/.mcp.json | bun -e "JSON.parse(await Bun.stdin.text()); console.log('Valid')"`
Expected: `Valid`

---

### Task 7: Append CLAUDE.md Instructions

**Files:**
- Modify: `claudecode/.claude/CLAUDE.md` (stowed to `~/.claude/CLAUDE.md`)

**Step 1: Read the current CLAUDE.md**

Current content is minimal:
```
This file does nothing.

# Read the PAI system for system understanding and initiation
`read skills/PAI/SKILL.md`
```

**Step 2: Append Recall's template**

Append the memory instructions section from `/Users/ed/Developer/AI/AtlasMemory/templates/CLAUDE.md.template` to `claudecode/.claude/CLAUDE.md`. Only append the `## MEMORY - SQLite + FTS5 (Recall)` section through the tables section. Skip the "Template Variables", "Optional: Permission to Fail", and "Optional: Stack Preferences" sections (PAI already handles these concerns).

The file should end up as:

```markdown
This file does nothing.

# Read the PAI system for system understanding and initiation
`read skills/PAI/SKILL.md`

## MEMORY - SQLite + FTS5 (Recall)

**CRITICAL: Before asking the user to repeat ANYTHING, search first.**

Your entire work history is in `~/.claude/memory.db`.

### MCP Server Tools

When recall-memory MCP is configured, you have access to:

| Tool | Usage |
|------|-------|
| `context_for_agent` | **MANDATORY before Task tool.** Hybrid search for agent context |
| `memory_search` | Keyword search (FTS5) - USE before asking user to repeat |
| `memory_hybrid_search` | Natural language search (FTS5 + semantic + RRF fusion) |
| `memory_recall` | Recent LoA entries, decisions, breadcrumbs |
| `memory_add` | Add decision/learning/breadcrumb during sessions |
| `loa_show` | View full LoA entry |
| `memory_stats` | Database statistics |

### LAW: Agent Context

**Before spawning any agent (Task tool), MUST call `context_for_agent` first.**

If it recommends Brave search, call `mcp__brave-search__brave_web_search` and include results in agent prompt.

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
```

**Step 3: Commit**

```bash
git add claudecode/.claude/CLAUDE.md
git commit -m "feat: append Recall memory tool instructions to CLAUDE.md"
```

---

### Task 8: Import Historical Sessions (Optional)

**Step 1: Check what's available**

Run: `ls ~/.claude/projects/ | head -20`
Expected: Lists encoded project directories with JSONL files inside.

**Step 2: Import**

Run: `mem import`
Expected: Scans JSONL files, imports sessions and messages into SQLite. May take a few minutes depending on history size.

**Step 3: Verify import**

Run: `mem stats`
Expected: Shows non-zero counts for `sessions` and `messages` tables.

---

### Task 9: Verify End-to-End Integration

**Step 1: Restow to pick up settings.json and CLAUDE.md changes**

Run: `cd /Users/ed/Developer/atlas-config && just up` (or `stow -R claudecode`)
Expected: Stow restows the claudecode package, symlinks updated.

**Step 2: Test MCP server starts**

Run: `echo '{"jsonrpc":"2.0","method":"initialize","id":1,"params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}' | mem-mcp`
Expected: Returns JSON-RPC response with server capabilities and tool list.

**Step 3: Test mem search**

Run: `mem search "test query"`
Expected: Returns results (or empty results if no data imported). No errors.

**Step 4: Commit all changes**

```bash
git add -A
git commit -m "feat: complete Recall (LMF3) integration — standalone complement to PAI memory"
```

---

### Task 10: Update CLAUDE.md Local Modifications Table

**Files:**
- Modify: `/Users/ed/Developer/atlas-config/CLAUDE.md` (repo root)

**Step 1: Add Recall entries to the Local Modifications table**

Add two rows to the "Local Modifications (PAIUpdate Protection)" table:

| File | What Was Customized |
|------|---------------------|
| `claudecode/.claude/CLAUDE.md` | Appended Recall (LMF3) memory tool instructions (MCP tools table, CLI commands, LAW: Agent Context rule). |
| `claudecode/.claude/settings.json` | Added `SessionExtract.ts` Stop hook for Recall session extraction. |

**Step 2: Commit**

```bash
git add CLAUDE.md
git commit -m "docs: document Recall integration in local modifications table"
```

---

### Post-Integration Notes

**What to watch for after first real session:**
- Check `~/.claude/MEMORY/EXTRACT_LOG.txt` for extraction success/failure
- Check `~/.claude/MEMORY/HOT_RECALL.md` for session summaries appearing
- Run `mem stats` to see if records are accumulating
- Verify MCP tools appear in Claude Code's tool list (they should auto-discover from `.mcp.json`)

**Future evolution to Approach B (PAI-Aware):**
- Add `"MEMORY/HOT_RECALL.md"` to `contextFiles` in settings.json
- Add Recall search as a capability recommendation in the Algorithm's OBSERVE phase
- No architectural changes needed — just configuration
