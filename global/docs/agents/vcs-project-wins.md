# GitButler, never worktrees

## Rules

- Default: GitButler. Use `but` for branches, commits, pushes, and undo in every repo in GitButler mode. Never invent a plain-git workaround for something `but` does.
- A repo not in GitButler mode: ask before running `but setup`, and hold branch, commit, and push work until the user answers. If they decline, use plain git branches in the one checkout. Never a worktree either way.
- Never create a native git worktree. No `git worktree add`, and no harness or subagent option that isolates work in a worktree. For parallel work, use GitButler parallel branches in the same workspace.
- The nearest project `AGENTS.md` wins on commit tooling when it names plain git. It never permits a worktree.
- No force-push without asking. Do not push directly to a protected branch. Prefer a PR into the integration branch.

## Done

The work landed through `but` (or the plain git the project named), and no worktree was created.
