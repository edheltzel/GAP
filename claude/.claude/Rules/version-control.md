# Version Control (MANDATORY)

Covers worktrees and git. Worktree usage is always required. **Git is the VCS for all repos.**

## Worktrees

Always use git worktrees for implementation work unless Ed explicitly says otherwise.

- Use the built-in `EnterWorktree` tool to create an isolated worktree before starting feature work. The tool creates a worktree under `.atlas/worktrees/` with a new branch based on HEAD.
- Use `ExitWorktree` when done (keep or remove).
- This keeps the main working tree clean and enables parallel development.

## Git

Use standard git for all repos. The usual safety guardrails apply: no force-push without asking, investigate before destructive ops, never push directly to a protected branch, and prefer a PR into the integration branch over committing straight to it.
