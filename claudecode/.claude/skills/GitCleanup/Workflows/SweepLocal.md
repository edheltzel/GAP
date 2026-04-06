# SweepLocal Workflow

Remove stale local worktrees and their associated branches.

## Voice Notification

```bash
curl -s -X POST http://localhost:8888/notify \
  -H "Content-Type: application/json" \
  -d '{"message": "Running the SweepLocal workflow in the GitCleanup skill to prune local worktrees and branches"}' \
  > /dev/null 2>&1 &
```

Running the **SweepLocal** workflow in the **GitCleanup** skill to prune local worktrees and branches...

---

## Step 1: Inventory Worktrees

```bash
git worktree list --porcelain
```

For each worktree, note:
- Path
- Branch (`branch refs/heads/<name>`)
- Whether it's bare or detached
- Whether the path still exists on disk

Also check the Claude-managed worktree directory:
```bash
ls -la .claude/worktrees/ 2>/dev/null || echo "(no .claude/worktrees/)"
```

## Step 2: Identify Stale Worktrees

A worktree is stale if ANY of:
- Its directory no longer exists (`git worktree prune --dry-run` will flag these)
- Its branch has been fully merged into `main`/`master`/`develop`
- Its branch has an upstream that is gone (`git branch -vv` shows `[origin/branch: gone]`)
- It was created by Claude Code (path contains `.claude/worktrees/`) and the branch is merged

Run the dry-run prune to see what git already considers prunable:
```bash
git worktree prune --dry-run
```

## Step 3: Check Branch Merge Status

For each worktree's branch, check if it's been merged:
```bash
# Against main
git branch --merged main 2>/dev/null || git branch --merged master

# Show branches with gone upstream
git branch -vv | grep '\[.*: gone\]'
```

## Step 4: Present Cleanup Candidates

Show the user a clear table of what will be removed:

| Worktree Path | Branch | Status | Action |
|---------------|--------|--------|--------|
| `.claude/worktrees/foo` | `worktree-foo` | merged into main | Remove worktree + delete branch |
| `.claude/worktrees/bar` | `worktree-bar` | upstream gone | Remove worktree + delete branch |
| `/path/to/wt` | `feature/xyz` | dirty (uncommitted changes) | ⚠️ Skip unless confirmed |

**Always warn before touching dirty worktrees.** Ask the user explicitly before force-removing any worktree with uncommitted changes.

## Step 5: Remove Prunable Worktrees

For each confirmed-clean worktree:
```bash
# Remove worktree
git worktree remove <path>

# If the path no longer exists (orphaned):
git worktree prune
```

For dirty worktrees (only if user explicitly confirms):
```bash
git worktree remove --force <path>
```

## Step 6: Delete Associated Branches

After removing worktrees, delete their branches:
```bash
# Safe delete (only if merged)
git branch -d <branch-name>

# Force delete (if user confirms for unmerged)
git branch -D <branch-name>
```

For branches with a gone upstream tracking ref, clean them up:
```bash
git fetch --prune
```

## Step 7: Final State

```bash
git worktree list
git branch -vv
```

Report what was removed and what remains.

---

## Safety Rules

- **Never** force-remove a dirty worktree without explicit user confirmation
- **Never** `git branch -D` without confirming the branch is either merged or explicitly unwanted
- If in doubt, show the diff first: `git log main..<branch> --oneline`
- `git fetch --prune` is safe to run — it only updates tracking refs, doesn't delete local branches
