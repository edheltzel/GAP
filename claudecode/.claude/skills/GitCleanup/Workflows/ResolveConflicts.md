# ResolveConflicts Workflow

Guide through merge conflict resolution before cleanup operations.

## Voice Notification

```bash
curl -s -X POST http://localhost:8888/notify \
  -H "Content-Type: application/json" \
  -d '{"message": "Running the ResolveConflicts workflow in the GitCleanup skill to resolve merge conflicts"}' \
  > /dev/null 2>&1 &
```

Running the **ResolveConflicts** workflow in the **GitCleanup** skill to resolve merge conflicts...

---

## Step 1: Identify Conflict State

Check what kind of conflict situation we're in:

```bash
# Check current merge/rebase/cherry-pick state
git status

# What operation is in progress?
ls .git/MERGE_HEAD 2>/dev/null && echo "MERGE in progress"
ls .git/rebase-merge 2>/dev/null && echo "REBASE in progress"
ls .git/CHERRY_PICK_HEAD 2>/dev/null && echo "CHERRY-PICK in progress"
```

```bash
# List all conflicted files
git diff --name-only --diff-filter=U
```

## Step 2: Show Conflict Summary

For each conflicted file, show a brief summary:

```bash
# Show just the conflict markers with filenames
grep -rn "<<<<<<< " --include="*.ts" --include="*.js" --include="*.py" --include="*.go" . 2>/dev/null \
  | awk -F: '{print $1}' | sort -u
```

Present the user with:
```
Conflicted files (N total):

  src/auth/login.ts      — 2 conflicts
  src/api/routes.ts      — 1 conflict
  package.json           — 1 conflict (likely version bump)

Approach: [a]utomatic (ours/theirs), [m]anual (file by file), [t]ool (open mergetool)
```

## Step 3: Determine Resolution Strategy

Ask the user which strategy to use:

| Strategy | When to use |
|----------|-------------|
| **Ours** (`--strategy-option=ours`) | Keep current branch version entirely |
| **Theirs** (`--strategy-option=theirs`) | Take incoming changes entirely |
| **Manual** | File-by-file review (recommended for logic conflicts) |
| **Mergetool** | Open configured git mergetool |

### Automatic resolution (all files, one side)

```bash
# Accept all "ours" (current branch)
git checkout --ours -- <file>
git add <file>

# Accept all "theirs" (incoming branch)
git checkout --theirs -- <file>
git add <file>
```

### Manual resolution

For each conflicted file, read it and show the conflict sections:

```bash
# Show conflict diff
git diff <file>
```

Then edit the file to resolve (remove conflict markers `<<<<<<<`, `=======`, `>>>>>>>`), then:

```bash
git add <file>
```

### Open mergetool

```bash
git mergetool
```

## Step 4: Resolve Package.json Conflicts (Special Case)

Package.json version conflicts are common. Handle carefully:

```bash
git diff package.json
```

- **version field:** Usually take the higher version, or the one on the target branch
- **dependencies:** If both sides added different deps, keep both
- **lockfile conflicts:** After resolving package.json, regenerate:

```bash
bun install  # or npm install / yarn
git add bun.lockb  # or package-lock.json / yarn.lock
```

## Step 5: Complete the Operation

After resolving all conflicts:

```bash
# Verify no conflicts remain
git diff --name-only --diff-filter=U
# Should return empty

# Complete the merge/rebase
git merge --continue   # if merge was in progress
git rebase --continue  # if rebase was in progress
git cherry-pick --continue  # if cherry-pick was in progress
```

Or to abort entirely if things are too tangled:

```bash
git merge --abort
git rebase --abort
git cherry-pick --abort
```

## Step 6: Verify Clean State

```bash
git status
git log --oneline -5
```

Confirm the working tree is clean before proceeding with any cleanup operations.

---

## Safety Rules

- **Never blindly accept --ours or --theirs for all files** — ask per-file or per-category
- **Always check package.json and lockfiles manually** — version bump conflicts are tricky
- **Show the conflict diff** before asking what to do — never resolve blind
- **Offer to abort** at any point if the user wants to step back
- If the conflict is in generated files (e.g. `*.lock`, `dist/`), prefer regenerating over manual resolution
