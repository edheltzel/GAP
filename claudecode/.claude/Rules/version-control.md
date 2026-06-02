# Version Control (MANDATORY)

Covers worktrees, git, and jj (Jujutsu). Worktree usage is always required. **Git is the default VCS for all repos.** jj rules apply **only when Ed explicitly asks** — never auto-activate jj based on the presence of a `.jj/` directory.

## Worktrees

Always use git worktrees for implementation work unless Ed explicitly says otherwise.

- Use the built-in `EnterWorktree` tool to create an isolated worktree before starting feature work. The tool creates a worktree under `.atlas/worktrees/` with a new branch based on HEAD.
- Use `ExitWorktree` when done (keep or remove).
- This keeps the main working tree clean and enables parallel development.
- If Ed has asked for jj on a jj-colocated repo, `EnterWorktree` still applies — the git worktree it creates is visible to jj via colocation, so `jj` commands work inside it normally. Do **not** substitute `jj workspace add` for `EnterWorktree`.

## Git (default)

Use standard git for **all** repos by default — including jj-colocated ones — unless Ed has explicitly asked you to use jj for this work. The presence of a `.jj/` directory does **not** switch you into jj mode on its own. No special rules beyond the worktree mandate above and the usual safety guardrails (no force-push without asking, investigate before destructive ops, etc.).

## jj — Conditional Rules (only when Ed explicitly asks)

Activation: use jj **only when Ed explicitly asks for it** on this task. A `.jj/` directory at the repo root is *not* a trigger — it just means jj is available if Ed requests it. Once Ed asks, confirm `.jj/` exists at the repo root before running any jj command; if it's absent, do not initialize jj without Ed's explicit approval. When Ed has not asked, default to git per the section above.

### Hard restrictions

Inside a jj-colocated repo, **never** use:

- `git add` — there is no staging area; the working copy *is* the current change
- `git commit` — use `jj commit` or `jj describe` + `jj new`
- `git stash` — use `jj new` to start fresh; previous work is preserved automatically
- `git checkout` — use `jj edit <change>` or `jj new <change>`

`git log`, `git diff`, `git status`, `git push`, `git fetch` are fine for inspection, but prefer their `jj` equivalents.

### AI-agent rules

- **Always pass `-m "message"`** to `jj new`, `jj describe`, `jj commit`, `jj squash`. Without `-m`, jj opens an editor and the agent will hang.
- **Avoid interactive commands** (`jj split`, `jj diffedit`) unless Ed explicitly asks for them.

### Common commands

| Purpose | Command |
| --- | --- |
| Show change graph | `jj log` |
| Show current diff | `jj diff` |
| Set description | `jj describe -m "msg"` |
| Finish change, start next | `jj new -m "msg"` |
| Commit working copy | `jj commit -m "msg"` |
| Jump to a change | `jj edit <change-id>` |
| Discard current change | `jj abandon` |
| Undo last operation | `jj undo` |
| Operation history | `jj op log` |
| Restore past state | `jj op restore <op-id>` |
| Distribute hunks to ancestors | `jj absorb` |

### Bookmarks and pushing

- Bookmarks do **not** auto-advance. Set explicitly before pushing:
  ```
  jj bookmark set <name> -r @-
  jj git push
  ```
- Push a specific bookmark: `jj git push --bookmark <name>`.
- In colocated repos, git HEAD is detached. For `gh pr create`, pass `--head <bookmark>` explicitly.
- **Never rewrite history on a reviewed PR** — review comments anchor to commit SHAs and will detach. Add new commits instead; squash only after approval.

### Recovery

`jj op log` records every mutation. `jj undo` reverses the last operation; `jj op restore <id>` jumps back to any prior state. Use these before reaching for destructive git commands.

### Immutable heads

To rewrite protected commits, temporarily relax the immutable-heads revset — but **ask Ed first**, since this enables rewriting shared history:

```
jj config set --repo 'revset-aliases."immutable_heads()"' 'none()'
```

Restore immediately after:

```
jj config set --repo 'revset-aliases."immutable_heads()"' 'builtin_immutable_heads() | remote_bookmarks()'
```
