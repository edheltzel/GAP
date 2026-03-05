---
name: UpdateDocs
description: Update project README.md and CLAUDE.md based on what changed since docs were last updated. USE WHEN update docs, update documentation, update readme, refresh docs, docs are stale, sync docs.
---

# UpdateDocs

Scan a project's git history to find what changed since documentation was last updated, then surgically edit README.md and CLAUDE.md to reflect the current state. Preserves manually-written sections. Auto-commits the result.

## Customization

**Before executing, check for user customizations at:**
`~/.claude/skills/PAI/USER/SKILLCUSTOMIZATIONS/UpdateDocs/`

If this directory exists, load and apply any PREFERENCES.md, configurations, or resources found there. These override default behavior. If the directory does not exist, proceed with skill defaults.

## Voice Notification

**When executing a workflow, do BOTH:**

1. **Send voice notification**:
   ```bash
   curl -s -X POST http://localhost:8888/notify \
     -H "Content-Type: application/json" \
     -d '{"message": "Running the UpdateDocs workflow to refresh project documentation"}' \
     > /dev/null 2>&1 &
   ```

2. **Output text notification**:
   ```
   Running the **UpdateDocs** workflow in the **UpdateDocs** skill to refresh project documentation...
   ```

## Workflow Routing

| Workflow | Trigger | File |
|----------|---------|------|
| **UpdateDocs** | "update docs", "refresh docs", "docs are stale", "sync documentation" | `Workflows/UpdateDocs.md` |

## Examples

**Example 1: After adding a new feature**
```
User: "/UpdateDocs"
-> Finds 12 commits since last docs update
-> README: adds new CLI command to usage section
-> CLAUDE.md: adds new module to project structure
-> Commits: "docs: update README and CLAUDE.md for new CLI command"
```

**Example 2: After a refactor**
```
User: "Update the docs"
-> Finds renamed directories and moved files
-> README: updates file paths in project structure
-> CLAUDE.md: updates import paths and module descriptions
-> Commits: "docs: reflect directory restructure in project docs"
```

**Example 3: New project with no docs**
```
User: "/UpdateDocs"
-> No README.md or CLAUDE.md found
-> Scans project structure, package.json, config files
-> Generates README.md from scratch
-> Generates CLAUDE.md with agent instructions
-> Commits: "docs: add README.md and CLAUDE.md"
```

## Scope

This skill updates two files:

| File | Purpose |
|------|---------|
| `README.md` | Human-facing project documentation |
| `CLAUDE.md` | AI agent instructions for working in the project |
