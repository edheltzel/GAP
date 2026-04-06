# SweepRemote Workflow

Delete merged or stale branches on GitHub (primary), GitLab, or Bitbucket.

## Voice Notification

```bash
curl -s -X POST http://localhost:8888/notify \
  -H "Content-Type: application/json" \
  -d '{"message": "Running the SweepRemote workflow in the GitCleanup skill to prune remote branches"}' \
  > /dev/null 2>&1 &
```

Running the **SweepRemote** workflow in the **GitCleanup** skill to prune remote branches...

---

## Step 1: Detect Remote Platform

Check which CLI tools are available and which remote is configured:

```bash
# Check remote URL
git remote get-url origin

# Check available CLI tools
which gh 2>/dev/null && echo "gh available"
which glab 2>/dev/null && echo "glab available"
```

**Platform detection:**
- `github.com` in remote URL → use `gh` (GitHub CLI)
- `gitlab.com` or self-hosted GitLab → use `glab`
- `bitbucket.org` → use git push delete (no official CLI)

**Preference:** GitHub (`gh`) is primary. If `gh` is not authenticated, prompt:
```bash
gh auth status
```

## Step 2: List Remote Branches with Merge Status

### GitHub (gh CLI)
```bash
# List all remote branches
gh api repos/{owner}/{repo}/branches --paginate \
  --jq '.[] | {name: .name, protected: .protected}'

# Check which are merged into default branch
gh api repos/{owner}/{repo}/branches \
  --jq '.[] | select(.name != "main" and .name != "master" and .name != "develop") | .name' \
  | while read branch; do
    merged=$(gh api repos/{owner}/{repo}/compare/main...${branch} --jq '.ahead_by == 0' 2>/dev/null)
    echo "$branch: merged=$merged"
  done
```

**Simpler approach using git (works everywhere):**
```bash
# Fetch latest remote state
git fetch --prune origin

# List branches merged into main on remote
git branch -r --merged origin/main \
  | grep -v 'HEAD\|main\|master\|develop\|release\|hotfix' \
  | sed 's|origin/||'
```

### GitLab (glab CLI)
```bash
glab api projects/:id/repository/branches \
  --jq '.[] | select(.merged == true) | .name'
```

## Step 3: Identify Protected Branches

Never delete these branches:
- `main`, `master`
- `develop`, `development`
- `release/*`, `hotfix/*`
- Any branch marked as "protected" in the remote

```bash
# GitHub: list protected branches
gh api repos/{owner}/{repo}/branches --jq '.[] | select(.protected) | .name'
```

## Step 4: Present Cleanup Candidates

Show the user what will be deleted before acting:

```
Remote branches to delete (merged into main):

  feature/add-auth         [merged 3 days ago]
  fix/login-redirect       [merged 1 week ago]
  worktree-old-feature     [merged 2 weeks ago]
  chore/update-deps        [merged 1 month ago]

Protected (will NOT be deleted):
  main, develop

Proceed? [y/N]
```

Always ask for confirmation before deleting.

## Step 5: Delete Remote Branches

### GitHub
```bash
# Using gh CLI
gh api -X DELETE repos/{owner}/{repo}/git/refs/heads/{branch}

# Or using git push
git push origin --delete <branch-name>
```

### GitLab
```bash
glab api -X DELETE projects/:id/repository/branches/:branch
# Or:
git push origin --delete <branch-name>
```

### Bitbucket / Generic
```bash
git push origin --delete <branch-name>
```

## Step 6: Clean Up Local Tracking Refs

After deleting remote branches, prune stale tracking refs locally:
```bash
git fetch --prune
git remote prune origin
```

## Step 7: Report

List what was deleted and confirm current remote branch state:
```bash
git branch -r | grep -v HEAD
```

Report total deleted count.

---

## Safety Rules

- **Never delete protected branches** (main, master, develop, release/*, hotfix/*)
- **Always confirm** before executing bulk deletes
- **Dry-run first** if user says "show me what would be deleted"
- If `gh` auth fails, fall back to `git push origin --delete` with explicit branch names
- Prefer `gh` over raw git for GitHub repos — gives richer metadata (merge date, PR number)
