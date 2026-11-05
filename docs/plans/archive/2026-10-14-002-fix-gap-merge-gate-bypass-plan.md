---
title: Close two merge-gate bypasses in gap sync
type: fix
date: 2026-10-14
execution: code
---

# Close two merge-gate bypasses in gap sync

ATL-613 red-team proved two ways to publish policy that is not on `origin/HEAD`, both inherited from the Node script. The contract (`AGENTS.md`: "Sync refuses while `global/` differs from `origin/HEAD`") says neither should work.

## Decisions

- Remove `GAP_SOURCE` (rejected: document it as a debug hook). It was added in #56 with no doc or test that uses it, and it publishes bytes the merge gate never checked.
- Strip git's local env vars from every `git` call gap makes (rejected: refuse when `GIT_DIR` is set, which breaks runs from git hooks for no gain). The list is `git rev-parse --local-env-vars`.

## Units

1. `cmd/gap/main.go`: read only `<repo>/global/AGENTS.md`. In `git()`, run with `os.Environ()` minus the local env vars.
2. `cmd/gap/main_test.go`, `TestSyncEndToEnd`: with an unmerged `global/AGENTS.md`, `GAP_SOURCE=<file>` still refuses, and `GIT_DIR=<other repo>` still refuses. Both fail before the fix.

## Verification

- Both new assertions fail on master and pass after the fix.
- `go vet ./...`, `go test ./...` pass.

## todo.txt

- Global `ATL-613` closes when this PR merges.
