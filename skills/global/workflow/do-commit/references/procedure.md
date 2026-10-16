# Procedure

## GitButler

GitButler is in use when `command -v but` succeeds **and** `but status` exits 0. Do not run `but setup`. If either check fails, use the git commands in this skill.

When GitButler is in use:
- Inspect with `but diff`. `git log --oneline` is fine (read-only).
- Commit with `but commit -b <feature-branch> -m "<msg>"` (omit IDs to take all uncommitted changes; pass `but diff` IDs for a subset). Take `<feature-branch>` from `but status`, never `gitbutler/workspace`.
- Never `git add`, `git commit`, `git stash`, or `git checkout`.
- **Push path only:** push with `but push <feature-branch>` unless No Mistakes is in use. Never `git push`.

## No Mistakes

Push path only. Skip this section on the commit-only path.

No Mistakes is in use when `command -v no-mistakes` succeeds **and** `git remote` lists `no-mistakes` (or `no-mistakes axi` shows this repo). Do not run `no-mistakes init` unless the user asked.

When No Mistakes is in use:
- GitButler still owns inspect and commit. Raw git still owns `git add` / `git commit` when GitButler is not in use.
- After the commit, publish with `no-mistakes axi run --yes --intent "<what the user set out to accomplish>"`. Do not `but push`, `but pr new`, `git push`, or `git push origin`.
- The installed `/no-mistakes` skill owns the axi loop. Drive it; do not copy it here.
- If `no-mistakes axi` reports `current_branch: gitbutler/workspace` or the default branch, stop. Tell the user the gate cannot see a real feature branch. Do not invent a push workaround.

## Gates

Complete **in order**. Do not run the next action until the **Pass** condition is satisfied (use command output as evidence, not memory).

1. **Diff understood** — **Pass when:** `git status`/`git diff`/`git diff --cached`, **or** `but diff` if GitButler is in use, match your one-sentence description of what changed (or you recorded that there is nothing to commit).
2. **Commit line chosen** — **Pass when:** You have a draft first line `type(scope): description` (or `type: description` if omitting scope) that matches the change set you intend to commit (on the push path, the set you intend to ship).
3. **Contents match intent** — **Pass when:** After `git add`, `git diff --cached --stat` shows only the paths you meant; **or** if GitButler, the IDs you will pass to `but commit` (or all uncommitted, if omitting IDs) match that set.
4. **Push target confirmed** — **Push path only.** **Pass when:** Current feature branch and remote are the ones you intend (`git branch -vv` / `git remote -v`, **or** `but status` if GitButler); then push. **If No Mistakes:** skip origin/push confirmation; pass when you have started `no-mistakes axi run --yes --intent "..."` on a real feature branch (not `gitbutler/workspace` or the default branch).
5. **Remote caught up** — **Push path only.** **Pass when:** `git status -sb` shows the branch up to date with upstream, **or** `but status` shows the branch pushed, with no unexpected leftover commits for this task. **If No Mistakes:** pass only on axi `outcome: checks-passed` or `outcome: passed`. `passed-with-skips` is not publication proof — report the skip. A `gate:` (including `ask-user`) is blocked/pending, not a pass. Do not treat origin sync as the proof.

## Step 1: Gather Context

Run these commands in parallel to understand the changes:

```bash
git status
git diff
git diff --cached
git log --oneline -10
```

If GitButler is in use, use `but diff` instead of the `git status`/`git diff`/`git diff --cached` trio. Keep `git log --oneline -10`.

## Step 2: Analyze Changes

Review the changes and determine:
- **Type**: What kind of change is this?
  - `feat` - New feature or capability
  - `fix` - Bug fix
  - `docs` - Documentation only
  - `refactor` - Code restructure without behavior change
  - `test` - Adding or updating tests
  - `chore` - Maintenance, dependency updates
  - `perf` - Performance improvement
  - `ci` - CI/CD changes

- **Scope**: Which component is affected?
  - Examine the changed files and determine the appropriate scope
  - Use consistent scope names within the project (check `git log` for patterns)
  - *(omit scope for cross-cutting changes)*

- **Breaking**: Does this break backward compatibility? If yes, add **!** after scope.

## Step 3: Write Commit Message

Format:
```
type(scope): description

[optional body explaining why, not what]

[optional footer with issue references]
```

Rules:
- Use imperative mood: "add feature" not "added feature"
- Keep first line under 72 characters
- Focus on *why* in the body, the diff shows *what*
- Reference issues: `Closes #123` or `Fixes #456`

## Step 4: Stage and Commit

Satisfy **Gates** 1–3 before committing. On the push path, satisfy **Gate** 4 before pushing and **Gate** 5 after push. On the commit-only path, the commit is the last git action; continue to Next in SKILL.md.

If GitButler is in use:

```bash
but commit -b <feature-branch> -m "$(cat <<'EOF'
type(scope): description

Optional body explaining the motivation.

Closes #123
EOF
)"
```

Otherwise:

```bash
# Stage all changes (or selectively stage)
git add -A

# Gate 3: confirm staged set before committing
git diff --cached --stat

# Commit with message (use HEREDOC for multi-line)
git commit -m "$(cat <<'EOF'
type(scope): description

Optional body explaining the motivation.

Closes #123
EOF
)"
```

Then publish (push path only):

- **No Mistakes:** `no-mistakes axi run --yes --intent "<what the user set out to accomplish>"`. Drive gates with `/no-mistakes`. Do not `but push` or `git push`.
- **GitButler, no gate:** `but push <feature-branch>`
- **Otherwise:** `git push`

## Examples

```bash
# Simple feature
git commit -m "feat(api): add pagination support to list endpoints"

# Bug fix with body
git commit -m "$(cat <<'EOF'
fix(auth): handle token expiration during long requests

The previous implementation did not account for tokens expiring
during the processing of long-running requests.

Fixes #42
EOF
)"

# Breaking change
git commit -m "$(cat <<'EOF'
feat!(api): change response format for user endpoints

BREAKING CHANGE: The `status` field is now an object with `state` and
`message` properties instead of a plain string.
EOF
)"
```

Optionally append a co-author or footer trailer per project convention (e.g. a `Co-Authored-By:` line or a tool-attribution footer). Omit it when the project has no such convention.

## Step 5: Verify

Push path only. Skip on the commit-only path.

After pushing, satisfy **Gate 5**: `git status` and `git status -sb`, **or** `but status` if GitButler is in use. Confirm a clean tree and upstream sync (or an expected ahead/behind you can explain). **If No Mistakes:** the axi outcome is Gate 5. Do not `git fetch` origin to prove the push.

## Step 6: PR and Land

Modes `cpp` and `land` only. Run after Gate 5 passes. If No Mistakes already opened the PR, reuse it; do not open a second one.

**Open the PR** (`cpp`, `land`):
- GitButler: `but pr new <feature-branch> -t` (title and body from the commits). Pass `-m` for a custom title and body.
- Otherwise: `gh pr create --fill`.

For `cpp`, report the PR URL, then continue to Next in SKILL.md. Do not land.

**Land** (`land`): with GitButler, `but land <feature-branch> --yes`. It moves the target to the branch tip, so GitHub marks the PR merged. If it refuses because of a stacked segment below, stop and report; do not retry with `--whole-stack` unless the user asks.

Fallback: if GitButler is not in use, or `but land` is rejected (protected branch, required checks), merge through the forge with `gh pr merge <number> --squash --delete-branch` (or the method the repo allows). If required checks are still pending, add `--auto` and report it as pending.

Verify: `gh pr view <number> --json state,mergedAt` shows `MERGED` (or auto-merge enabled and pending checks, which you report as pending, not merged).
