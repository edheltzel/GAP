---
name: GitCleanup
description: Git repository cleanup and hygiene. USE WHEN user wants to clean up branches, delete stale or merged branches, remove worktrees, prune remote refs, close or merge stale PRs or MRs, resolve merge conflicts before cleanup, or audit the state of a git repo including branches, worktrees, open pull requests, and issues.
---

# GitCleanup

Git repository cleanup — prune stale branches, worktrees, PRs, and issues across local and remote (GitHub-first, GitLab/Bitbucket supported).

## Customization

**Before executing, check for user customizations at:**
`~/.claude/skills/PAI/USER/SKILLCUSTOMIZATIONS/GitCleanup/`

If this directory exists, load and apply any PREFERENCES.md or configurations found there. If not, proceed with defaults.

## Voice Notification

**When executing a workflow, do BOTH:**

1. **Send voice notification**:
   ```bash
   curl -s -X POST http://localhost:8888/notify \
     -H "Content-Type: application/json" \
     -d '{"message": "Running the WORKFLOWNAME workflow in the GitCleanup skill to ACTION"}' \
     > /dev/null 2>&1 &
   ```

2. **Output text notification**:
   ```
   Running the **WorkflowName** workflow in the **GitCleanup** skill to ACTION...
   ```

## Workflow Routing

| Workflow | Trigger | File |
|----------|---------|------|
| **SweepLocal** | "clean up worktrees", "remove stale branches", "prune local" | `Workflows/SweepLocal.md` |
| **SweepRemote** | "delete remote branches", "clean remote", "prune remote refs" | `Workflows/SweepRemote.md` |
| **ClosePR** | "close PRs", "merge stale pull requests", "clean up MRs" | `Workflows/ClosePR.md` |
| **ResolveConflicts** | "resolve conflicts", "fix merge conflicts" | `Workflows/ResolveConflicts.md` |
| **AuditRepo** | "audit the repo", "show branch status", "what needs cleanup" | `Workflows/AuditRepo.md` |

## Examples

**Example 1: Full local cleanup**
```
User: "Clean up my stale worktrees and merged branches"
→ Invokes SweepLocal workflow
→ Lists all worktrees + their branch state
→ Removes prunable worktrees (prompts before force-removing dirty ones)
→ Deletes associated local branches that are merged or gone
```

**Example 2: Remote branch hygiene**
```
User: "Delete all the merged branches on GitHub"
→ Invokes SweepRemote workflow
→ Uses gh CLI to list branches with merge status
→ Shows candidates, confirms, then deletes merged remote branches
```

**Example 3: Full repo audit**
```
User: "Audit the repo — show me what needs cleanup"
→ Invokes AuditRepo workflow
→ Reports on: local branches, remote branches, worktrees, open PRs, stale issues
→ Outputs a ranked cleanup priority list
```
