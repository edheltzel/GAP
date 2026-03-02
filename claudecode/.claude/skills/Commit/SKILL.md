---
name: Commit
description: Generate and create git commits with conventional commit messages and CHANGELOG updates. USE WHEN commit, save work, commit changes, commit message, git commit.
---

# Commit

Create well-structured git commits following the [Conventional Commits](https://www.conventionalcommits.org/) specification. Automatically updates CHANGELOG.md.

## Customization

**Before executing, check for user customizations at:**
`~/.claude/skills/PAI/USER/SKILLCUSTOMIZATIONS/Commit/`

If this directory exists, load and apply any PREFERENCES.md, configurations, or resources found there. These override default behavior. If the directory does not exist, proceed with skill defaults.

## Voice Notification

**When executing a workflow, do BOTH:**

1. **Send voice notification**:
   ```bash
   curl -s -X POST http://localhost:8888/notify \
     -H "Content-Type: application/json" \
     -d '{"message": "Running the Commit workflow in the Commit skill to create a git commit"}' \
     > /dev/null 2>&1 &
   ```

2. **Output text notification**:
   ```
   Running the **Commit** workflow in the **Commit** skill to create a git commit...
   ```

## Workflow Routing

| Workflow | Trigger | File |
|----------|---------|------|
| **Commit** | "commit", "save work", "commit changes", "commit message" | `Workflows/Commit.md` |

## Examples

**Example 1: Commit staged changes**
```
User: "Commit these changes"
-> Invokes Commit workflow
-> Checks git status and diff
-> Analyzes changes for commit type
-> Updates CHANGELOG.md
-> Creates conventional commit
```

**Example 2: Stage and commit everything**
```
User: "Save all my work"
-> Invokes Commit workflow
-> Runs git status, stages unstaged changes
-> Generates appropriate commit message
-> Updates CHANGELOG.md
-> Commits with conventional message
```

**Example 3: Commit with specific message guidance**
```
User: "Commit this as a bugfix for the auth timeout"
-> Invokes Commit workflow
-> Uses "fix" type based on user context
-> Crafts message around auth timeout fix
-> Updates CHANGELOG.md under ### Fixed
-> Commits
```

## Quick Reference: Conventional Commit Types

| Type | When |
|------|------|
| `feat` | New feature |
| `fix` | Bug fix |
| `refactor` | Code restructuring, no behavior change |
| `docs` | Documentation changes |
| `style` | Formatting, whitespace |
| `test` | Adding or updating tests |
| `chore` | Maintenance tasks, dependencies |
| `perf` | Performance improvement |
| `build` | Build system or dependency changes |
| `ci` | CI/CD configuration changes |
| `revert` | Reverts a previous commit |
