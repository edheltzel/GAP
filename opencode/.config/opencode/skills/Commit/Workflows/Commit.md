# Commit Workflow

Generate and create a git commit with a conventional commit message.

## Voice Notification

```bash
  -H "Content-Type: application/json" \
  -d '{"message": "Running the Commit workflow in the Commit skill to create a git commit"}' \
  > /dev/null 2>&1 &
```

Running the **Commit** workflow in the **Commit** skill to create a git commit...

## Step 1: Check Current State

Run these commands **in parallel**:

```bash
git status                # Staged and unstaged changes
git diff --cached         # What's staged
git diff                  # Unstaged changes
git log --oneline -5      # Recent commit style
```

## Step 2: Stage Changes If Needed

If there are unstaged changes the user wants to commit, stage them with `git add`.

- Prefer adding specific files by name rather than `git add -A` or `git add .`
- Never stage files that likely contain secrets (`.env`, `credentials.json`, etc.)

## Step 3: Analyze the Changes

Review what's being committed:
- What files are affected?
- What's the nature of the change (fix, feature, refactor, docs, etc.)?
- What's the primary purpose?

## Step 4: Update CHANGELOG.md

Add a description of the changes under the `## [Unreleased]` section:

1. Read the top of `CHANGELOG.md` to see the current format
2. If there's no `## [Unreleased]` section, create one at the top (after the header)
3. Add entries under the appropriate category header:
   - `### Added` - New features or capabilities
   - `### Changed` - Changes to existing functionality
   - `### Fixed` - Bug fixes
   - `### Removed` - Removed features or code
   - `### Security` - Security-related changes
4. Each entry should be a bullet point with a clear description
5. Use sub-bullets for additional context when helpful
6. Stage the CHANGELOG.md changes with `git add CHANGELOG.md`

**Example format:**
```markdown
## [Unreleased]
### Added
- New phone agent limitation directives
  - Agent now clarifies it cannot make calls including 911
  - Redirects persistent requests to main clinic
```

**If no CHANGELOG.md exists:** Skip this step. Do not create one unless the user asks.

## Step 5: Generate Commit Message

Follow the [Conventional Commits](https://www.conventionalcommits.org/) format:

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

### Types

| Type | When |
|------|------|
| `feat` | New feature |
| `fix` | Bug fix |
| `refactor` | Code restructuring without behavior change |
| `docs` | Documentation changes |
| `style` | Formatting, whitespace |
| `test` | Adding or updating tests |
| `chore` | Maintenance tasks, dependencies |
| `perf` | Performance improvement |
| `build` | Build system or dependency changes |
| `ci` | CI/CD configuration changes |
| `revert` | Reverts a previous commit |

### Scopes (optional)

Add a scope in parentheses for additional context about what changed:

```
feat(auth): add two-factor authentication
fix(api): handle rate limit errors
refactor(db): optimize user query performance
```

Choose scopes that match the project's architecture — e.g. `auth`, `api`, `ui`, `db`, `config`.

### Description rules

- Use imperative mood: "add" not "added" or "adds"
- Don't capitalize the first letter
- No period at the end
- Keep under 72 characters (ideally under 50)
- Focus on WHY the change was made, not just WHAT changed
- Don't include file lists — git tracks that
- Match the style of recent commits in the repo when possible

### Body (optional)

Include a body when the change needs explanation:

- Separate from description with a blank line
- Use imperative mood
- Wrap at 72 characters
- Explain non-obvious design decisions, migration notes, or context

### Breaking changes

Mark breaking changes with `!` after the type/scope, or with a `BREAKING CHANGE:` footer:

```
feat(api)!: change authentication endpoint

BREAKING CHANGE: The /auth endpoint now requires a client_id parameter.
```

### Issue references (optional)

Link to issues in the footer:

```
Fixes #123
Closes #456
Related to #789
```

## Step 6: Create the Commit

Use a HEREDOC for proper formatting:

```bash
git commit -m "$(cat <<'EOF'
type(scope): short description

Optional longer description explaining why.

Fixes #123
EOF
)"
```

## Step 7: Verify Success

Run `git status` to confirm the commit was created.

## Safety Rules

- **Never** use `--force` or destructive commands
- **Never** skip hooks with `--no-verify` unless explicitly asked
- **Never** amend commits that have been pushed
- **Never** push to remote unless explicitly asked
- If there are no changes to commit, inform the user
- If pre-commit hooks fail, fix the issue and create a **new** commit (do not use `--amend`)
