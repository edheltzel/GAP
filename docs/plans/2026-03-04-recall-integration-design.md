# Recall (LMF3) Integration Design

**Date:** 2026-03-04
**Approach:** A — Standalone Complement
**Status:** Approved

## Summary

Integrate Recall (formerly LMF3) alongside the existing PAI memory system as a complementary, independent memory layer. Zero changes to PAI code. Both systems run in parallel, accumulating data independently.

## Context

### Current PAI Memory (5 Layers)

1. **Claude Code Transcripts** — `~/.claude/projects/*.jsonl`, 30-day TTL, raw session data
2. **Episodic Memory Plugin** — SQLite + vector embeddings via Transformers.js, semantic search over archived conversations, no expiry
3. **MEMORY/ Filesystem** — Structured directories: WORK/, LEARNING/, RELATIONSHIP/, STATE/, SECURITY/. Hook-driven writes.
4. **LoadContext.hook.ts** — Synchronous SessionStart injection of SKILL.md, contextFiles, relationship notes, learnings, active work
5. **Auto Memory** — `MEMORY.md` at `~/.claude/projects/{hash}/memory/`, Claude-managed project notes, first 200 lines injected per session

### Recall (LMF3)

Source: `/Users/ed/Developer/AI/AtlasMemory` (fork of `nixfred/Recall`)

**Components:**
- **MCP Server** (`mem-mcp`) — stdio transport, 7 tools: `memory_search`, `memory_hybrid_search`, `memory_recall`, `memory_add`, `memory_stats`, `loa_show`, `context_for_agent`
- **CLI** (`mem`) — commander-based human interface for search, capture, import, stats, LoA management
- **SQLite Database** (`~/.claude/memory.db`) — WAL mode, 9 tables (sessions, messages, loa_entries, decisions, learnings, breadcrumbs, telos, documents, embeddings), 7 FTS5 virtual tables, 21 sync triggers
- **Session Extraction Hook** (`SessionExtract.ts`) — Stop hook, self-spawns detached background process, reads JSONL, pipes through `claude -p --model haiku` for structured 8-section extraction
- **Flat Files** — `MEMORY/DISTILLED.md` (full archive), `MEMORY/HOT_RECALL.md` (last 10 sessions), `MEMORY/SESSION_INDEX.json`, `MEMORY/DECISIONS.log`, `MEMORY/REJECTIONS.log`, `MEMORY/ERROR_PATTERNS.json`
- **Embeddings** — 768-dim float32 vectors via Ollama `nomic-embed-text`, cosine similarity in TypeScript

## Design Decisions

### 1. Complement, not replace
Both systems coexist and accumulate data independently. No PAI code changes required.

### 2. MEMORY/ root files coexist with PAI subdirectories
Recall writes flat files (DISTILLED.md, HOT_RECALL.md, etc.) at `~/.claude/MEMORY/` root level. PAI uses subdirectories (WORK/, LEARNING/, etc.). No filename collisions.

### 3. Keep both search systems
- **Episodic Memory** — searches raw conversations (full context, vector similarity via Transformers.js)
- **Recall** — searches structured extractions (decisions, learnings, LoA via FTS5 + Ollama embeddings)
- Different data sources = complementary results

### 4. Hooks run in parallel
Recall's `SessionExtract.ts` self-spawns as a detached background process on Stop. No blocking, no ordering dependency with PAI's session-end hooks (SessionSummary, WorkCompletionLearning, RelationshipMemory).

### 5. Symlink approach — repos stay independent
- Recall's hook symlinked from its repo into `~/.claude/hooks/`
- Binaries installed globally via `bun install -g`
- atlas-config repo modified only in: `settings.json` (hook registration) and `~/.claude/CLAUDE.md` (tool instructions)

### 6. CLAUDE.md for tool instructions
Recall's usage template appended to `~/.claude/CLAUDE.md`. Claude Code loads this file natively — no additional contextFile needed.

## Coexistence Matrix

| Concern | PAI | Recall | Conflict? |
|---------|-----|--------|-----------|
| Session-end hooks | SessionSummary, WorkCompletionLearning, RelationshipMemory | SessionExtract.ts (self-spawns, non-blocking) | None — parallel |
| MEMORY/ writes | WORK/, LEARNING/, STATE/, RELATIONSHIP/, SECURITY/ (subdirs) | DISTILLED.md, HOT_RECALL.md, SESSION_INDEX.json, etc. (root files) | None — no overlap |
| Search | Episodic Memory (raw conversation vectors) | FTS5 + Ollama embeddings (structured extractions) | None — different data |
| Decision tracking | PRD ## Decisions section | decisions table in SQLite | None — independent |
| Learning capture | LEARNING/ markdown files | learnings table in SQLite | None — independent |
| Context injection | LoadContext.hook.ts → system-reminder | CLAUDE.md template append | None — different paths |
| MCP registration | episodic-memory (plugin) | lmf-memory in .mcp.json | None — separate servers |

## Installation Steps

1. **Install binaries globally** — `cd /Users/ed/Developer/AI/AtlasMemory && bun install -g`
2. **Initialize the database** — `mem init` (creates `~/.claude/memory.db`)
3. **Symlink the extraction hook** — `ln -s /Users/ed/Developer/AI/AtlasMemory/hooks/SessionExtract.ts ~/.claude/hooks/SessionExtract.ts`
4. **Register MCP server** — Add `lmf-memory` entry to `~/.claude/.mcp.json`
5. **Register Stop hook** — Add Stop hook entry in `settings.json`
6. **Append CLAUDE.md instructions** — `cat /Users/ed/Developer/AI/AtlasMemory/templates/CLAUDE.md.template >> ~/.claude/CLAUDE.md`
7. **Import historical sessions (optional)** — `mem import`

## Future Evolution

Once Recall is validated, evolve to **Approach B (PAI-Aware)** by:
- Adding `HOT_RECALL.md` to `contextFiles` in settings.json
- Referencing Recall tools in the Algorithm's OBSERVE phase for context recovery
- Adding Recall search as a recommended capability in the Algorithm

No architectural changes needed — just configuration additions.

## Visual Design

Full architecture diagram: `~/.agent/diagrams/recall-integration-design.html`
