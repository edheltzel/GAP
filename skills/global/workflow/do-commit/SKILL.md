---
name: do-commit
description: >-
  Commit local changes as a Conventional Commit. Use when the user says
  commit, push, ship, or open a PR, or when a do-* hand-off names do-commit
  with a mode (commit, cp, cpp, land). Never pushes unless the mode or the
  user asks. NOT FOR sync, conflicts, or undo (do-git-safe-pr-workflow),
  stacked PRs (do-gh-stack), or review-comment triage (do-git-pr-review-triage).
---

# Commit

Commit all local changes in Conventional Commits form. Never push unless the mode or the user asks.

Read [references/procedure.md](references/procedure.md) and follow it. A "stop" after the commit or after `cpp` means that action is done; continue to Next. `land` ends after its verify.

## Modes

Pick the mode from the argument (`/do-commit cpp`, or a caller passing `cpp`) or the user's words. Shorthands are case-insensitive. A passed mode wins over a bare "commit" reading of the same turn.

| Mode | Shorthand | Does |
|------|-----------|------|
| Commit (default) | none, `commit` | Commit. Then Next. |
| Push | `push`, `cp` | Commit, push. Then Next. |
| Commit, push, PR | `cpp` | Commit, push, open a PR. Then Next. |
| Land | `land` | Commit, push, open a PR, land it. No Next. |

- **Commit (default).** Any request that does not ask to push and passes no push mode. Do not push, do not run No Mistakes, and skip gates 4-5 and Step 5.
- **Every other mode** runs the push path: No Mistakes, gates 4-5, publish, and Step 5. `cpp` and `land` then run Step 6. If there is nothing uncommitted, skip the commit and publish existing commits.
- **`land` only on the user's explicit go.** They typed `land` or picked it. A caller that passes `land` with `mode:return-to-caller` is not that go: return `status: needs-human` and do not merge.

## Caller

`do-it`, `do-work`, and `do-debug` may call this with a mode. `mode:return-to-caller` (used by `do-it` and `do-work`) skips Next. Return `status: done | blocked | needs-human` plus the artifact: the PR URL if this run opened or reused one, otherwise the commit SHA.

- `done` - the mode finished (push path: Gate 5 passed).
- `needs-human` - No Mistakes `ask-user`, `land` without an explicit go, or `but land` refused.
- `blocked` - no feature branch, or push/PR failed.

## Related skills

- Syncing, conflicts, undo, or merging after the commit: `do-git-safe-pr-workflow`, if installed.
- Work that should land as dependent PRs: `do-gh-stack`, if installed.

## Next

End with this block. Run the next skill only when the user picks it. When a caller started this run (`mode:return-to-caller`, used by `do-it` and `do-work`), skip the block and return `status: done | blocked | needs-human` plus the artifact path.

Offer the block for the mode that just finished, and only after that mode's gates passed. A blocked or needs-human push does not get the push-path block.

**Commit:**

Next: `do-commit cp` - push this commit, passing `<commit SHA>`.

Other moves:

1. `do-commit cpp` - also open a PR.
2. Stop - the commit is already saved.

**cp or cpp** (after no-mistakes `checks-passed` or `passed`, or the push verified):

Next: Stop - the PR is ready.

On `cp` with no PR, say the push is verified instead.

Other moves:

1. `do-git-pr-review-triage` - when review comments land, passing `<PR URL>`.
2. `do-commit land` - only when the user says go.

**land:** no block.

A pick of `do-commit` with a mode continues in this skill with that mode. `land` still needs the user's go in that pick.

Reply with a number, the label, or what you want instead.

Ask with the harness's blocking question tool when it has one, and drop the last line. Otherwise print the block and wait. A pick loads that skill now, in this session.
