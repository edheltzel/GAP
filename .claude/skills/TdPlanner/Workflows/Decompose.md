# Decompose Workflow

Reads a markdown plan/PRD/spec and decomposes it into TD task hierarchy with phases, epics, tasks, dependencies, and priorities.

## Prerequisites Check

**Before doing anything else:**

1. Verify `.todos/` directory exists in the current project:
   ```bash
   ls .todos/db.sqlite
   ```
   If missing, tell the user: "TD isn't initialized in this project. Start Sidecar in this directory first, or run `td init`."
   **Do NOT proceed without TD initialized.**

2. Check current TD state so we don't create duplicates:
   ```bash
   td list 2>/dev/null
   ```

## Input Handling

Accept input in these forms (detect automatically):

1. **File path** — User provides a path to a markdown file. Read it.
2. **Pasted content** — User pastes markdown directly in the conversation. Use it.
3. **Reference to current work** — User says "break down the current plan" — look for active PRD or plan files in the project.

## Decomposition Process

### Step 1: Read and Understand the Document

Read the entire document. Identify:
- The overall goal/project description
- Major sections (these become phases/epics)
- Sub-sections (these become epics or task groups)
- Action items, requirements, bullet points (these become individual tasks)
- Any explicit ordering, dependencies, or blocking language
- Priority signals in the language

### Step 2: Map to TD Hierarchy

Apply these mapping rules:

**Document Structure → TD Concepts:**

| Document Element | TD Concept | Command |
|---|---|---|
| Top-level sections (`#` or `##`) | Phase (epic) | `td epic create "Phase: [title]" --priority [P0-P1]` |
| Sub-sections (`###` or `####`) | Epic under phase | `td epic create "[title]"` + `td tree add-child` |
| Bullet points / action items | Individual tasks | `td create "[title]" --type [type] --priority [priority]` |
| Numbered/sequential items | Tasks with dependencies | `td create` + `td dep add` for ordering |

**Type Inference:**
- `feature` — New functionality, capabilities, features (default)
- `bug` — Language like "fix", "broken", "incorrect", "doesn't work"
- `chore` — Language like "clean up", "remove", "migrate", "update dependency", "refactor"
- `docs` — Documentation tasks
- `test` — Test-related tasks

**Priority Inference:**
- **P0** — "critical", "blocker", "security vulnerability", "data loss", "must ship"
- **P1** — "important", "core", "required", "must have", "high priority"
- **P2** — Default for most tasks. Normal feature work.
- **P3** — "nice to have", "optional", "future", "low priority", "stretch goal"

**Dependency Inference:**
- Numbered lists → sequential dependencies (item 2 depends on item 1)
- Explicit language: "requires", "after", "depends on", "blocked by", "needs X first" → `td dep add`
- Phase ordering → later phases depend on earlier phases completing
- Nested items → child depends on parent epic existing

### Step 3: Generate TD Commands

Produce a complete, ordered list of `td` commands. Format:

```bash
# ═══════════════════════════════════════
# Phase 1: [Phase Name]
# ═══════════════════════════════════════

td epic create "Phase 1: [Phase Name]" --priority P0
# Expected ID: td-XXXX (use actual ID from output in execute mode)

# --- Epic: [Epic Name] ---
td epic create "[Epic Name]"
td tree add-child <phase-id> <epic-id>

# Tasks
td create "[Task description]" --type feature --priority P1
td tree add-child <epic-id> <task-id>

td create "[Another task]" --type feature --priority P2
td tree add-child <epic-id> <task-id>
td dep add <this-task> <previous-task>  # sequential dependency

# ═══════════════════════════════════════
# Phase 2: [Phase Name]
# ═══════════════════════════════════════
# Phase 2 depends on Phase 1
td epic create "Phase 2: [Phase Name]" --priority P1
td dep add <phase2-id> <phase1-id>
```

**Command comments must explain:**
- Why this priority was chosen (if not P2 default)
- Why this dependency exists
- Any assumptions made about grouping

### Step 4: Present for Review or Execute

**Review Mode (default):**
- Output the full command list with comments
- Include a summary at the top: "X phases, Y epics, Z tasks, W dependencies"
- Ask: "Review the commands above. Say 'execute' to run them, or point out any changes."

**Execute Mode (user says "execute" or invoked with --execute):**
- Ask for single confirmation: "About to create X phases, Y epics, Z tasks with W dependencies. Proceed?"
- On approval, run commands sequentially
- Capture each `td create` output to get the actual issue IDs
- Use actual IDs for `td tree add-child` and `td dep add` commands
- Report results: "Created X issues. Run `td list` or open Sidecar to see them."

**For large plans (50+ tasks):**
- Break into batches by phase
- In execute mode, confirm per-phase before running
- Show running total: "Phase 1 complete (12 tasks). Proceed to Phase 2 (8 tasks)?"

## Edge Cases

- **Flat document (no headers):** Treat entire content as one phase. Create tasks from all actionable items. Flag: "Document had no section headers — created a flat task list. Consider organizing into phases."
- **Very short document:** If fewer than 3 actionable items found, warn: "Only found N tasks. Is this the complete plan?"
- **Existing TD tasks:** If `td list` shows existing tasks with similar titles, warn before creating potential duplicates.
- **Non-actionable content:** Skip background/context paragraphs that don't contain tasks. Only create TD issues from actionable items.
