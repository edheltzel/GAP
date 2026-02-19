---
name: TdPlanner
description: Decompose plans, PRDs, and specs into TD task hierarchies. USE WHEN break down plan into td tasks, convert PRD to td, td planner, create td tasks from plan, decompose into td, check td docs.
---

## Customization

**Before executing, check for user customizations at:**
`~/.claude/skills/PAI/USER/SKILLCUSTOMIZATIONS/TdPlanner/`

If this directory exists, load and apply any PREFERENCES.md, configurations, or resources found there. These override default behavior. If the directory does not exist, proceed with skill defaults.

## 🚨 MANDATORY: Voice Notification (REQUIRED BEFORE ANY ACTION)

**You MUST send this notification BEFORE doing anything else when this skill is invoked.**

1. **Send voice notification**:
   ```bash
   curl -s -X POST http://localhost:8888/notify \
     -H "Content-Type: application/json" \
     -d '{"message": "Running the WORKFLOWNAME workflow in the TdPlanner skill to ACTION"}' \
     > /dev/null 2>&1 &
   ```

2. **Output text notification**:
   ```
   Running the **WorkflowName** workflow in the **TdPlanner** skill to ACTION...
   ```

**This is not optional. Execute this curl command immediately upon skill invocation.**

# TdPlanner

Reads any structured markdown document (plan, PRD, spec, design doc) and decomposes it into a proper TD task hierarchy: phases, milestones, epics, and tasks with auto-detected dependencies and priorities.

**Prerequisites:** TD must be initialized in the project (via Sidecar or `td init`). The skill checks for `.todos/` directory before proceeding.

## TD Documentation References

Keep these current. Use the **CheckDocs** workflow periodically to verify accuracy.

- **TD Core Workflow:** https://td.haplab.com/docs/core-workflow/
- **TD GitHub:** https://github.com/marcus/td
- **Sidecar Docs:** https://sidecar.haplab.com/docs/intro
- **Sidecar TD Integration:** https://sidecar.haplab.com/docs/td

## Workflow Routing

| Workflow | Trigger | File |
|----------|---------|------|
| **Decompose** | "break down plan", "convert to td", "decompose into td", "td planner" | `Workflows/Decompose.md` |
| **CheckDocs** | "check td docs", "update td knowledge" | `Workflows/CheckDocs.md` |

## Examples

**Example 1: Break down a PRD**
```
User: "Break down this PRD into td tasks: docs/plans/auth-system-design.md"
-> Invokes Decompose workflow
-> Reads the markdown file
-> Outputs td commands organized by phase/epic/task hierarchy
-> User reviews and approves, then commands execute
```

**Example 2: Convert inline plan**
```
User: "Convert this plan to td tasks: [pastes markdown]"
-> Invokes Decompose workflow
-> Parses the pasted content
-> Outputs td command hierarchy
```

**Example 3: Check for TD updates**
```
User: "Check td docs for any new features"
-> Invokes CheckDocs workflow
-> Fetches latest TD documentation
-> Compares against stored reference
-> Reports new commands or features
```

## Quick Reference: TD Command Vocabulary

The skill uses these TD commands (see Workflows/Decompose.md for full mapping rules):

```bash
# Structure
td epic create "title" --priority P0        # Phase/milestone container
td create "title" --type feature --priority P1  # Individual task
td tree add-child <parent> <child>           # Nest task under epic

# Dependencies
td dep add <task> <depends-on>               # Task blocks on another

# Execution (when in execute mode)
td start <id>                                # Begin work
td log "message"                             # Track progress
td handoff <id> --done "..." --remaining "..." --decision "..." --uncertain "..."
```
